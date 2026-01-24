package knx

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Bridge operation constants.
const (
	// minTopicParts is the minimum number of parts in a valid MQTT topic.
	minTopicParts = 3

	// commandTimeout is the timeout for sending commands to devices.
	commandTimeout = 5 * time.Second

	// readAllTimeout is the timeout for reading all device states.
	readAllTimeout = 30 * time.Second

	// interReadDelay is the delay between read requests to avoid bus flooding.
	interReadDelay = 50 * time.Millisecond
)

// Bridge orchestrates bidirectional translation between KNX and MQTT.
// It handles:
//   - Receiving commands from Core via MQTT and translating to KNX telegrams
//   - Receiving KNX telegrams and publishing state updates to MQTT
//   - Health reporting and graceful shutdown
//
// Thread Safety: All methods are safe for concurrent use.
type Bridge struct {
	cfg      *Config
	mqtt     MQTTClient
	knxd     Connector
	health   *HealthReporter
	registry DeviceRegistry // Optional device registry for state/health persistence

	// Device mappings (built from config)
	gaToDevice  map[string]GAMapping
	deviceToGAs map[string]map[string]AddressConfig
	mappingMu   sync.RWMutex

	// State cache for change detection
	stateCache   map[string]map[string]any
	stateCacheMu sync.RWMutex

	// Shutdown coordination
	done      chan struct{}
	wg        sync.WaitGroup
	stopOnce  sync.Once
	ctx       context.Context    // Bridge-level context, cancelled on Stop()
	ctxCancel context.CancelFunc // Cancel function for ctx

	// Logger
	logger   Logger
	loggerMu sync.RWMutex
}

// MQTTClient is the interface for MQTT operations.
// This allows mocking in tests and flexibility in implementation.
type MQTTClient interface {
	// Publish sends a message to a topic.
	Publish(topic string, payload []byte, qos byte, retained bool) error

	// Subscribe registers a handler for a topic pattern.
	Subscribe(topic string, qos byte, handler func(topic string, payload []byte)) error

	// IsConnected returns true if connected to the broker.
	IsConnected() bool

	// Disconnect closes the connection gracefully.
	Disconnect(quiesce uint)
}

// DeviceRegistry provides device state and health persistence.
// This interface is satisfied by *device.Registry (via adapter in main.go).
// It is optional - if nil, the bridge operates without registry integration.
type DeviceRegistry interface {
	// SetDeviceState updates the state of a device.
	SetDeviceState(ctx context.Context, id string, state map[string]any) error

	// SetDeviceHealth updates the health status of a device.
	SetDeviceHealth(ctx context.Context, id string, status string) error

	// CreateDeviceIfNotExists seeds a device record from bridge config.
	// No-op if the device already exists (preserves user modifications).
	CreateDeviceIfNotExists(ctx context.Context, seed DeviceSeed) error
}

// DeviceSeed holds device fields derivable from bridge config.
// Used to auto-populate the device registry on startup.
type DeviceSeed struct {
	ID           string
	Name         string
	Type         string
	Domain       string
	Protocol     string
	GatewayID    string
	Capabilities []string
	Address      map[string]string
}

// BridgeOptions holds configuration for creating a bridge.
type BridgeOptions struct {
	// Config is the loaded bridge configuration.
	Config *Config

	// MQTTClient is the MQTT client implementation.
	MQTTClient MQTTClient

	// KNXDClient is the knxd connection.
	KNXDClient Connector

	// Logger is optional structured logger.
	Logger Logger

	// Registry is optional device registry for state/health persistence.
	// If nil, the bridge operates without registry integration.
	Registry DeviceRegistry
}

// NewBridge creates a new bridge instance.
// Call Start() to begin operation.
func NewBridge(opts BridgeOptions) (*Bridge, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if opts.MQTTClient == nil {
		return nil, fmt.Errorf("MQTT client is required")
	}
	if opts.KNXDClient == nil {
		return nil, fmt.Errorf("knxd client is required")
	}

	// Build device index from config
	gaToDevice, deviceToGAs := opts.Config.BuildDeviceIndex()

	// Create bridge-level context for command cancellation on shutdown
	ctx, ctxCancel := context.WithCancel(context.Background())

	b := &Bridge{
		cfg:         opts.Config,
		mqtt:        opts.MQTTClient,
		knxd:        opts.KNXDClient,
		registry:    opts.Registry, // May be nil (optional)
		gaToDevice:  gaToDevice,
		deviceToGAs: deviceToGAs,
		stateCache:  make(map[string]map[string]any),
		done:        make(chan struct{}),
		ctx:         ctx,
		ctxCancel:   ctxCancel,
		logger:      opts.Logger,
	}

	// Create health reporter
	b.health = NewHealthReporter(HealthReporterConfig{
		BridgeID:   opts.Config.Bridge.ID,
		Version:    "1.0.0", // TODO: inject from build
		Interval:   opts.Config.GetHealthInterval(),
		Publisher:  opts.MQTTClient,
		KNXDClient: opts.KNXDClient,
	})
	b.health.SetDeviceCount(len(opts.Config.Devices))
	if opts.Logger != nil {
		b.health.SetLogger(opts.Logger)
	}

	return b, nil
}

// Start begins bridge operation.
// This subscribes to MQTT topics, sets up the KNX telegram handler,
// and starts health reporting.
func (b *Bridge) Start(ctx context.Context) error {
	// Seed device registry from bridge config (idempotent)
	b.seedDeviceRegistry(ctx)

	// Publish starting status
	if err := b.health.PublishStarting(); err != nil {
		b.logError("failed to publish starting status", err)
	}

	// Set up KNX telegram handler
	b.knxd.SetOnTelegram(b.handleKNXTelegram)

	// Subscribe to command topics
	commandTopic := CommandSubscribeTopic()
	if err := b.mqtt.Subscribe(commandTopic, 1, b.handleMQTTMessage); err != nil {
		return fmt.Errorf("subscribe to commands: %w", err)
	}
	b.logInfo("subscribed to commands", "topic", commandTopic)

	// Subscribe to request topics
	requestTopic := RequestSubscribeTopic()
	if err := b.mqtt.Subscribe(requestTopic, 1, b.handleMQTTMessage); err != nil {
		return fmt.Errorf("subscribe to requests: %w", err)
	}
	b.logInfo("subscribed to requests", "topic", requestTopic)

	// Start health reporting
	b.health.Start(ctx)

	// Publish initial healthy status
	if err := b.health.PublishNow(); err != nil {
		b.logError("failed to publish healthy status", err)
	}

	b.logInfo("bridge started",
		"bridge_id", b.cfg.Bridge.ID,
		"devices", len(b.cfg.Devices))

	return nil
}

// Stop gracefully shuts down the bridge.
func (b *Bridge) Stop() {
	b.stopOnce.Do(func() {
		close(b.done)

		// Cancel bridge context to abort in-flight commands
		b.ctxCancel()

		// Stop health reporting (publishes "stopping" status)
		b.health.Stop()

		// Wait for pending operations
		b.wg.Wait()

		b.logInfo("bridge stopped")
	})
}

// seedDeviceRegistry ensures all devices in the bridge config exist in the
// device registry. This makes knx-bridge.yaml the single source of truth for
// KNX device definitions — no manual device creation required.
// Idempotent: existing devices are not modified (preserves user enrichments).
func (b *Bridge) seedDeviceRegistry(ctx context.Context) {
	if b.registry == nil {
		return
	}
	for _, devCfg := range b.cfg.Devices {
		seed := buildDeviceSeed(devCfg, b.cfg.Bridge.ID)
		if err := b.registry.CreateDeviceIfNotExists(ctx, seed); err != nil {
			b.logInfo("failed to seed device", "device", devCfg.DeviceID, "error", err.Error())
		}
	}
}

// buildDeviceSeed derives a DeviceSeed from bridge config.
func buildDeviceSeed(cfg DeviceConfig, bridgeID string) DeviceSeed {
	deviceType := deriveDeviceType(cfg.Type, cfg.Addresses)
	return DeviceSeed{
		ID:           cfg.DeviceID,
		Name:         idToName(cfg.DeviceID),
		Type:         deviceType,
		Domain:       deriveDomain(cfg.Type),
		Protocol:     "knx",
		GatewayID:    bridgeID,
		Capabilities: deriveCapabilities(cfg.Type, cfg.Addresses),
		Address:      buildRegistryAddress(cfg.Addresses),
	}
}

// buildRegistryAddress creates a device.Address map that satisfies the
// KNX address validator (requires "group_address" key) while including
// all GA mappings for reference.
func buildRegistryAddress(addresses map[string]AddressConfig) map[string]string {
	m := make(map[string]string, len(addresses)+1)
	var primaryGA string
	for fn, addr := range addresses {
		m[fn] = addr.GA
		// Prefer a write-flagged GA as the primary address
		if primaryGA == "" {
			primaryGA = addr.GA
		}
		for _, flag := range addr.Flags {
			if flag == "write" {
				primaryGA = addr.GA
			}
		}
	}
	m["group_address"] = primaryGA
	return m
}

// idToName converts a device ID to a human-readable name.
// "living-room-ceiling-light" → "Living Room Ceiling Light"
func idToName(id string) string {
	words := strings.Split(id, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// deriveDeviceType maps bridge config type to registry DeviceType,
// refining generic types (e.g. "sensor") based on configured addresses.
func deriveDeviceType(bridgeType string, addresses map[string]AddressConfig) string {
	switch bridgeType {
	case "light_switch":
		return "light_switch"
	case "light_dimmer":
		return "light_dimmer"
	case "blind":
		return "blind_position"
	case "sensor":
		return deriveSensorType(addresses)
	case "scene":
		return "relay_channel"
	default:
		return bridgeType
	}
}

// deriveSensorType determines the specific sensor type from address functions.
func deriveSensorType(addresses map[string]AddressConfig) string {
	if _, ok := addresses["presence"]; ok {
		return "presence_sensor"
	}
	if _, ok := addresses["humidity"]; ok {
		return "humidity_sensor"
	}
	if _, ok := addresses["lux"]; ok {
		return "light_sensor"
	}
	return "temperature_sensor"
}

// deriveDomain maps bridge config type to the device domain.
func deriveDomain(bridgeType string) string {
	switch bridgeType {
	case "light_switch", "light_dimmer", "scene":
		return "lighting"
	case "blind":
		return "blinds"
	case "sensor":
		return "sensor"
	default:
		return "sensor"
	}
}

// deriveCapabilities infers device capabilities from type and address functions.
func deriveCapabilities(bridgeType string, addresses map[string]AddressConfig) []string {
	switch bridgeType {
	case "light_switch":
		return []string{"on_off"}
	case "light_dimmer":
		return []string{"on_off", "dim"}
	case "blind":
		caps := []string{"position"}
		if _, ok := addresses["slat"]; ok {
			caps = append(caps, "tilt")
		}
		return caps
	case "sensor":
		return deriveSensorCapabilities(addresses)
	default:
		return nil
	}
}

// deriveSensorCapabilities determines capabilities from sensor address functions.
func deriveSensorCapabilities(addresses map[string]AddressConfig) []string {
	var caps []string
	if _, ok := addresses["temperature"]; ok {
		caps = append(caps, "temperature_read")
	}
	if _, ok := addresses["humidity"]; ok {
		caps = append(caps, "humidity_read")
	}
	if _, ok := addresses["lux"]; ok {
		caps = append(caps, "light_level_read")
	}
	if _, ok := addresses["presence"]; ok {
		caps = append(caps, "presence_detect")
	}
	if len(caps) == 0 {
		caps = []string{"temperature_read"}
	}
	return caps
}

// handleMQTTMessage routes incoming MQTT messages to appropriate handlers.
func (b *Bridge) handleMQTTMessage(topic string, payload []byte) {
	// Parse topic to determine message type
	parts := strings.Split(topic, "/")
	if len(parts) < minTopicParts {
		b.logError("invalid topic format", fmt.Errorf("topic: %s", topic))
		return
	}

	messageType := parts[1] // command, request, etc.

	switch messageType {
	case "command":
		b.handleCommand(payload)
	case "request":
		b.handleRequest(payload)
	default:
		b.logError("unknown message type", fmt.Errorf("type: %s", messageType))
	}
}

// handleCommand processes a command message from Core.
func (b *Bridge) handleCommand(payload []byte) {
	// Parse command message
	var cmd CommandMessage
	if err := json.Unmarshal(payload, &cmd); err != nil {
		b.logError("failed to parse command", err)
		return
	}

	b.logInfo("received command",
		"command_id", cmd.ID,
		"device_id", cmd.DeviceID,
		"command", cmd.Command)

	// Look up device configuration
	b.mappingMu.RLock()
	deviceGAs, ok := b.deviceToGAs[cmd.DeviceID]
	b.mappingMu.RUnlock()

	if !ok {
		b.publishAckError(cmd, "", ErrCodeNotConfigured,
			fmt.Sprintf("device %s not configured", cmd.DeviceID), 0)
		return
	}

	// Execute command based on type
	err := b.executeCommand(cmd, deviceGAs)

	if err != nil {
		b.logError("command execution failed", err)
		// Error ack already sent by executeCommand
		return
	}

	// Success - ack already sent by executeCommand
}

// executeCommand translates and sends a command to the KNX bus.
func (b *Bridge) executeCommand(cmd CommandMessage, deviceGAs map[string]AddressConfig) error {
	// Derive timeout from bridge context so commands are cancelled on shutdown
	ctx, cancel := context.WithTimeout(b.ctx, commandTimeout)
	defer cancel()

	switch cmd.Command {
	case "on":
		return b.executeOnOff(ctx, cmd, deviceGAs, true)
	case "off":
		return b.executeOnOff(ctx, cmd, deviceGAs, false)
	case "dim":
		return b.executeDim(ctx, cmd, deviceGAs)
	case "set_position":
		return b.executeSetPosition(ctx, cmd, deviceGAs)
	case "stop":
		return b.executeStop(ctx, cmd, deviceGAs)
	default:
		b.publishAckError(cmd, "", ErrCodeInvalidCommand,
			fmt.Sprintf("unknown command: %s", cmd.Command), 0)
		return fmt.Errorf("unknown command: %s", cmd.Command)
	}
}

// executeOnOff sends an on/off command (DPT 1.001).
func (b *Bridge) executeOnOff(ctx context.Context, cmd CommandMessage, deviceGAs map[string]AddressConfig, on bool) error {
	// Find the switch address
	addr, ok := deviceGAs["switch"]
	if !ok {
		b.publishAckError(cmd, "", ErrCodeNotConfigured,
			"device has no switch address", 0)
		return fmt.Errorf("no switch address")
	}

	ga, err := ParseGroupAddress(addr.GA)
	if err != nil {
		b.publishAckError(cmd, addr.GA, ErrCodeProtocolError,
			fmt.Sprintf("invalid GA: %v", err), 0)
		return err
	}

	// Encode DPT 1.001
	data := EncodeDPT1(on)

	// Publish accepted ack before sending
	b.publishAck(cmd, addr.GA, AckAccepted)

	// Send to KNX bus
	if err := b.knxd.Send(ctx, ga, data); err != nil {
		b.publishAckError(cmd, addr.GA, ErrCodeDeviceUnreachable,
			fmt.Sprintf("send failed: %v", err), 0)
		return err
	}

	return nil
}

// executeDim sends a dim command (DPT 5.001).
func (b *Bridge) executeDim(ctx context.Context, cmd CommandMessage, deviceGAs map[string]AddressConfig) error {
	// Get level from parameters
	levelAny, ok := cmd.Parameters["level"]
	if !ok {
		b.publishAckError(cmd, "", ErrCodeInvalidParameters,
			"missing 'level' parameter", 0)
		return fmt.Errorf("missing level parameter")
	}

	level, ok := levelAny.(float64)
	if !ok {
		b.publishAckError(cmd, "", ErrCodeInvalidParameters,
			"'level' must be a number", 0)
		return fmt.Errorf("level must be a number")
	}

	// Validate range (0-100%)
	if level < 0 || level > 100 {
		b.publishAckError(cmd, "", ErrCodeInvalidParameters,
			fmt.Sprintf("'level' must be 0-100, got %.2f", level), 0)
		return fmt.Errorf("level out of range: %.2f", level)
	}

	// Find the brightness address
	addr, ok := deviceGAs["brightness"]
	if !ok {
		// Fall back to switch address for basic dimmers
		addr, ok = deviceGAs["switch"]
		if !ok {
			b.publishAckError(cmd, "", ErrCodeNotConfigured,
				"device has no brightness address", 0)
			return fmt.Errorf("no brightness address")
		}
	}

	ga, err := ParseGroupAddress(addr.GA)
	if err != nil {
		b.publishAckError(cmd, addr.GA, ErrCodeProtocolError,
			fmt.Sprintf("invalid GA: %v", err), 0)
		return err
	}

	// Encode DPT 5.001 (0-100% → 0-255)
	data := EncodeDPT5(level)

	// Publish accepted ack before sending
	b.publishAck(cmd, addr.GA, AckAccepted)

	// Send to KNX bus
	if err := b.knxd.Send(ctx, ga, data); err != nil {
		b.publishAckError(cmd, addr.GA, ErrCodeDeviceUnreachable,
			fmt.Sprintf("send failed: %v", err), 0)
		return err
	}

	return nil
}

// executeSetPosition sends a position command for blinds (DPT 5.001).
func (b *Bridge) executeSetPosition(ctx context.Context, cmd CommandMessage, deviceGAs map[string]AddressConfig) error {
	// Get position from parameters
	posAny, ok := cmd.Parameters["position"]
	if !ok {
		b.publishAckError(cmd, "", ErrCodeInvalidParameters,
			"missing 'position' parameter", 0)
		return fmt.Errorf("missing position parameter")
	}

	position, ok := posAny.(float64)
	if !ok {
		b.publishAckError(cmd, "", ErrCodeInvalidParameters,
			"'position' must be a number", 0)
		return fmt.Errorf("position must be a number")
	}

	// Validate range (0-100%)
	if position < 0 || position > 100 {
		b.publishAckError(cmd, "", ErrCodeInvalidParameters,
			fmt.Sprintf("'position' must be 0-100, got %.2f", position), 0)
		return fmt.Errorf("position out of range: %.2f", position)
	}

	// Find the position address
	addr, ok := deviceGAs["position"]
	if !ok {
		b.publishAckError(cmd, "", ErrCodeNotConfigured,
			"device has no position address", 0)
		return fmt.Errorf("no position address")
	}

	ga, err := ParseGroupAddress(addr.GA)
	if err != nil {
		b.publishAckError(cmd, addr.GA, ErrCodeProtocolError,
			fmt.Sprintf("invalid GA: %v", err), 0)
		return err
	}

	// Encode DPT 5.001 (0-100%)
	data := EncodeDPT5(position)

	// Publish accepted ack before sending
	b.publishAck(cmd, addr.GA, AckAccepted)

	// Send to KNX bus
	if err := b.knxd.Send(ctx, ga, data); err != nil {
		b.publishAckError(cmd, addr.GA, ErrCodeDeviceUnreachable,
			fmt.Sprintf("send failed: %v", err), 0)
		return err
	}

	return nil
}

// executeStop sends a stop command for blinds (DPT 1.007).
func (b *Bridge) executeStop(ctx context.Context, cmd CommandMessage, deviceGAs map[string]AddressConfig) error {
	// Find the stop address (or use move address with stop value)
	addr, ok := deviceGAs["stop"]
	if !ok {
		addr, ok = deviceGAs["move"]
		if !ok {
			b.publishAckError(cmd, "", ErrCodeNotConfigured,
				"device has no stop/move address", 0)
			return fmt.Errorf("no stop address")
		}
	}

	ga, err := ParseGroupAddress(addr.GA)
	if err != nil {
		b.publishAckError(cmd, addr.GA, ErrCodeProtocolError,
			fmt.Sprintf("invalid GA: %v", err), 0)
		return err
	}

	// Encode stop command (DPT 1.007: 1 = stop)
	data := EncodeDPT1(true)

	// Publish accepted ack before sending
	b.publishAck(cmd, addr.GA, AckAccepted)

	// Send to KNX bus
	if err := b.knxd.Send(ctx, ga, data); err != nil {
		b.publishAckError(cmd, addr.GA, ErrCodeDeviceUnreachable,
			fmt.Sprintf("send failed: %v", err), 0)
		return err
	}

	return nil
}

// publishAck publishes a command acknowledgment.
//
//nolint:unparam // status parameter will be used for AckQueued when queue support is added
func (b *Bridge) publishAck(cmd CommandMessage, address string, status AckStatus) {
	ack := NewAckMessage(cmd, status, address)

	payload, err := json.Marshal(ack)
	if err != nil {
		b.logError("failed to marshal ack", err)
		return
	}

	topic := AckTopic(address)
	if err := b.mqtt.Publish(topic, payload, 1, false); err != nil {
		b.logError("failed to publish ack", err)
	}
}

// publishAckError publishes a failed command acknowledgment.
//
//nolint:unparam // retries parameter will be used when retry logic is implemented
func (b *Bridge) publishAckError(cmd CommandMessage, address, code, message string, retries int) {
	ack := NewAckError(cmd, address, code, message, retries)

	payload, err := json.Marshal(ack)
	if err != nil {
		b.logError("failed to marshal ack error", err)
		return
	}

	topic := AckTopic(address)
	if err := b.mqtt.Publish(topic, payload, 1, false); err != nil {
		b.logError("failed to publish ack error", err)
	}

	b.logError("command failed",
		fmt.Errorf("code=%s message=%s", code, message))
}

// handleRequest processes a request message from Core.
func (b *Bridge) handleRequest(payload []byte) {
	var req RequestMessage
	if err := json.Unmarshal(payload, &req); err != nil {
		b.logError("failed to parse request", err)
		return
	}

	b.logInfo("received request",
		"request_id", req.RequestID,
		"action", req.Action)

	var resp ResponseMessage

	switch req.Action {
	case "read_state":
		resp = b.handleReadState(req)
	case "read_all":
		resp = b.handleReadAll(req)
	default:
		resp = ResponseMessage{
			RequestID: req.RequestID,
			Timestamp: time.Now().UTC(),
			Success:   false,
			Error: &ResponseError{
				Code:    ErrCodeInvalidCommand,
				Message: fmt.Sprintf("unknown action: %s", req.Action),
			},
		}
	}

	// Publish response
	respPayload, err := json.Marshal(resp)
	if err != nil {
		b.logError("failed to marshal response", err)
		return
	}

	respTopic := ResponseTopic(req.RequestID)
	if err := b.mqtt.Publish(respTopic, respPayload, 1, false); err != nil {
		b.logError("failed to publish response", err)
	}
}

// handleReadState handles a read_state request.
func (b *Bridge) handleReadState(req RequestMessage) ResponseMessage {
	if req.DeviceID == "" {
		return ResponseMessage{
			RequestID: req.RequestID,
			Timestamp: time.Now().UTC(),
			Success:   false,
			Error: &ResponseError{
				Code:    ErrCodeInvalidParameters,
				Message: "device_id is required",
			},
		}
	}

	// Look up device
	b.mappingMu.RLock()
	deviceGAs, ok := b.deviceToGAs[req.DeviceID]
	b.mappingMu.RUnlock()

	if !ok {
		return ResponseMessage{
			RequestID: req.RequestID,
			Timestamp: time.Now().UTC(),
			Success:   false,
			Error: &ResponseError{
				Code:    ErrCodeNotConfigured,
				Message: fmt.Sprintf("device %s not configured", req.DeviceID),
			},
		}
	}

	// Send read requests for readable addresses
	// Derive timeout from bridge context so reads are cancelled on shutdown
	ctx, cancel := context.WithTimeout(b.ctx, commandTimeout)
	defer cancel()

	for funcName, addr := range deviceGAs {
		if !addr.HasFlag("read") {
			continue
		}

		ga, err := ParseGroupAddress(addr.GA)
		if err != nil {
			continue
		}

		if err := b.knxd.SendRead(ctx, ga); err != nil {
			b.logError("read request failed",
				fmt.Errorf("device=%s func=%s: %w", req.DeviceID, funcName, err))
		}
	}

	// Return success - actual state will come via telegram callback
	return ResponseMessage{
		RequestID: req.RequestID,
		Timestamp: time.Now().UTC(),
		Success:   true,
		Data: map[string]any{
			"message": "read requests sent, state updates will follow",
		},
	}
}

// handleReadAll handles a read_all request.
func (b *Bridge) handleReadAll(req RequestMessage) ResponseMessage {
	// Derive timeout from bridge context so reads are cancelled on shutdown
	ctx, cancel := context.WithTimeout(b.ctx, readAllTimeout)
	defer cancel()

	b.mappingMu.RLock()
	devices := b.deviceToGAs
	b.mappingMu.RUnlock()

	readCount := 0
	for deviceID, deviceGAs := range devices {
		for funcName, addr := range deviceGAs {
			if !addr.HasFlag("read") {
				continue
			}

			ga, err := ParseGroupAddress(addr.GA)
			if err != nil {
				continue
			}

			if err := b.knxd.SendRead(ctx, ga); err != nil {
				b.logError("read request failed",
					fmt.Errorf("device=%s func=%s: %w", deviceID, funcName, err))
				continue
			}

			readCount++

			// Small delay between reads to avoid flooding the bus
			select {
			case <-ctx.Done():
				return ResponseMessage{
					RequestID: req.RequestID,
					Timestamp: time.Now().UTC(),
					Success:   false,
					Error: &ResponseError{
						Code:    ErrCodeTimeout,
						Message: "read_all timed out",
					},
				}
			case <-time.After(interReadDelay):
			}
		}
	}

	return ResponseMessage{
		RequestID: req.RequestID,
		Timestamp: time.Now().UTC(),
		Success:   true,
		Data: map[string]any{
			"reads_sent": readCount,
			"message":    "read requests sent, state updates will follow",
		},
	}
}

// handleKNXTelegram processes an incoming telegram from the KNX bus.
func (b *Bridge) handleKNXTelegram(t Telegram) {
	// Convert GA to string for lookup
	gaStr := t.Destination.String()

	// Look up device mapping
	b.mappingMu.RLock()
	mapping, ok := b.gaToDevice[gaStr]
	b.mappingMu.RUnlock()

	if !ok {
		// Unknown GA, ignore (might be traffic we don't care about)
		return
	}

	// Decode the value based on DPT
	value, err := b.decodeTelegramValue(t, mapping.DPT)
	if err != nil {
		b.logError("failed to decode telegram",
			fmt.Errorf("ga=%s dpt=%s: %w", gaStr, mapping.DPT, err))
		return
	}

	// Build state update
	state := b.buildStateUpdate(mapping, value)

	// Check if state changed (for change detection)
	if b.stateUnchanged(mapping.DeviceID, mapping.Function, value) {
		return // No change, skip publish
	}

	// Publish state message
	msg := NewStateMessage(mapping.DeviceID, gaStr, state)

	payload, err := json.Marshal(msg)
	if err != nil {
		b.logError("failed to marshal state", err)
		return
	}

	topic := StateTopic(gaStr)
	if err := b.mqtt.Publish(topic, payload, 1, true); err != nil {
		b.logError("failed to publish state", err)
		return
	}

	// Update device registry (if configured)
	if b.registry != nil {
		if err := b.registry.SetDeviceState(b.ctx, mapping.DeviceID, state); err != nil {
			// Log but don't fail - device may not be in registry yet
			b.logDebug("registry state update skipped",
				"device", mapping.DeviceID,
				"reason", err.Error())
		} else {
			// Mark device as online since we received traffic
			if healthErr := b.registry.SetDeviceHealth(b.ctx, mapping.DeviceID, "online"); healthErr != nil {
				b.logDebug("registry health update skipped",
					"device", mapping.DeviceID,
					"reason", healthErr.Error())
			}
		}
	}

	b.logInfo("published state",
		"device_id", mapping.DeviceID,
		"function", mapping.Function,
		"value", value)
}

// decodeTelegramValue decodes the telegram data based on DPT.
func (b *Bridge) decodeTelegramValue(t Telegram, dpt string) (any, error) {
	switch {
	case strings.HasPrefix(dpt, "1."):
		return DecodeDPT1(t.Data)
	case strings.HasPrefix(dpt, "5."):
		return DecodeDPT5(t.Data)
	case strings.HasPrefix(dpt, "9."):
		return DecodeDPT9(t.Data)
	default:
		// Return raw bytes for unknown DPT
		return t.Data, nil
	}
}

// functionToStateKey maps KNX function names to normalised state keys.
var functionToStateKey = map[string]string{
	"switch":            "on",
	"switch_status":     "on",
	"brightness":        "level",
	"brightness_status": "level",
	"position":          "position",
	"position_status":   "position",
	"tilt":              "tilt",
	"tilt_status":       "tilt",
	"temperature":       "temperature",
	"humidity":          "humidity",
	"motion":            "motion",
}

// buildStateUpdate builds a state object from the decoded value.
func (b *Bridge) buildStateUpdate(mapping GAMapping, value any) map[string]any {
	state := make(map[string]any)

	// Look up the state key from the function name
	stateKey, known := functionToStateKey[mapping.Function]
	if !known {
		// Generic: use function name as key
		state[mapping.Function] = value
		return state
	}

	// Set the state with the normalised key
	state[stateKey] = value

	// Special handling for motion: add last_motion timestamp
	if stateKey == "motion" {
		if v, ok := value.(bool); ok && v {
			state["last_motion"] = time.Now().UTC().Format(time.RFC3339)
		}
	}

	return state
}

// stateUnchanged checks if the new value matches the cached state.
// Returns true if unchanged (should skip publish).
func (b *Bridge) stateUnchanged(deviceID, function string, value any) bool {
	b.stateCacheMu.Lock()
	defer b.stateCacheMu.Unlock()

	if b.stateCache[deviceID] == nil {
		b.stateCache[deviceID] = make(map[string]any)
	}

	cached := b.stateCache[deviceID][function]
	if cached == value {
		return true // Unchanged
	}

	// Update cache
	b.stateCache[deviceID][function] = value
	return false
}

// ClearStateCache removes all entries from the state cache.
// Call this when configuration is reloaded to prevent unbounded memory growth
// from stale device IDs accumulating over multi-decade deployments.
func (b *Bridge) ClearStateCache() {
	b.stateCacheMu.Lock()
	defer b.stateCacheMu.Unlock()

	// Replace with fresh map to allow GC of old entries
	b.stateCache = make(map[string]map[string]any)
}

// PruneStateCache removes cache entries for devices not in the current config.
// This is a less disruptive alternative to ClearStateCache that preserves
// state for active devices while removing orphaned entries.
func (b *Bridge) PruneStateCache() {
	b.stateCacheMu.Lock()
	defer b.stateCacheMu.Unlock()

	// Deep copy valid device IDs to avoid data race with concurrent config reload
	b.mappingMu.RLock()
	validIDs := make(map[string]struct{}, len(b.deviceToGAs))
	for id := range b.deviceToGAs {
		validIDs[id] = struct{}{}
	}
	b.mappingMu.RUnlock()

	// Remove entries for devices not in current config
	for deviceID := range b.stateCache {
		if _, exists := validIDs[deviceID]; !exists {
			delete(b.stateCache, deviceID)
		}
	}
}

// SetLogger sets the logger for the bridge.
func (b *Bridge) SetLogger(logger Logger) {
	b.loggerMu.Lock()
	b.logger = logger
	b.loggerMu.Unlock()

	if b.health != nil {
		b.health.SetLogger(logger)
	}
}

// logInfo logs an info message if logger is set.
func (b *Bridge) logInfo(msg string, keysAndValues ...any) {
	b.loggerMu.RLock()
	logger := b.logger
	b.loggerMu.RUnlock()

	if logger != nil {
		logger.Info(msg, keysAndValues...)
	}
}

// logError logs an error message if logger is set.
func (b *Bridge) logError(msg string, err error) {
	b.loggerMu.RLock()
	logger := b.logger
	b.loggerMu.RUnlock()

	if logger != nil {
		logger.Error(msg, "error", err)
	}
}

// logDebug logs a debug message if logger is set.
func (b *Bridge) logDebug(msg string, keysAndValues ...any) {
	b.loggerMu.RLock()
	logger := b.logger
	b.loggerMu.RUnlock()

	if logger != nil {
		logger.Debug(msg, keysAndValues...)
	}
}
