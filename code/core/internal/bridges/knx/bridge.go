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
	cfg        *Config
	mqtt       MQTTClient
	knxd       Connector
	health     *HealthReporter
	registry   DeviceRegistry      // Optional device registry for state/health persistence
	gaRecorder GARecorderInterface // Optional GA recorder for passive discovery

	// Device mappings (built from config)
	gaToDevice        map[string][]GAMapping
	deviceToGAs       map[string]map[string]AddressConfig
	infrastructureIDs map[string]bool // device IDs with domain "infrastructure"
	mappingMu         sync.RWMutex

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

	// GetKNXDevices returns all devices with protocol "knx".
	// Used to load device mappings from the registry on startup.
	GetKNXDevices(ctx context.Context) ([]RegistryDevice, error)
}

// GARecorderInterface records telegrams seen on the bus for passive discovery.
// This is optional - if nil, the bridge operates without recording.
type GARecorderInterface interface {
	// RecordTelegram records a telegram's source device and destination GA.
	// source is the sender's individual address (e.g., "1.1.5").
	// ga is the destination group address (e.g., "1/2/3").
	// isResponse indicates if this was a GroupValue_Response (APCI 0x40).
	RecordTelegram(source, ga string, isResponse bool)
}

// RegistryDevice represents a device loaded from the registry.
// This is a subset of device.Device fields needed for bridge operation.
type RegistryDevice struct {
	ID           string
	Name         string
	Type         string
	Domain       string
	Functions    map[string]FunctionMapping // function -> {GA, DPT, Flags}
	Capabilities []string
}

// FunctionMapping holds the GA, DPT, and flags for a single device function.
type FunctionMapping struct {
	GA    string
	DPT   string
	Flags []string
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

	// GARecorder is optional GA recorder for passive discovery.
	// If nil, the bridge operates without recording seen GAs.
	GARecorder GARecorderInterface
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

	// Create bridge-level context for command cancellation on shutdown
	ctx, ctxCancel := context.WithCancel(context.Background())

	b := &Bridge{
		cfg:               opts.Config,
		mqtt:              opts.MQTTClient,
		knxd:              opts.KNXDClient,
		registry:          opts.Registry,   // May be nil (optional)
		gaRecorder:        opts.GARecorder, // May be nil (optional)
		gaToDevice:        make(map[string][]GAMapping),
		deviceToGAs:       make(map[string]map[string]AddressConfig),
		infrastructureIDs: make(map[string]bool),
		stateCache:        make(map[string]map[string]any),
		done:              make(chan struct{}),
		ctx:               ctx,
		ctxCancel:         ctxCancel,
		logger:            opts.Logger,
	}

	// Create health reporter
	b.health = NewHealthReporter(HealthReporterConfig{
		BridgeID:   opts.Config.Bridge.ID,
		Version:    "1.0.0", // TODO: inject from build
		Interval:   opts.Config.GetHealthInterval(),
		Publisher:  opts.MQTTClient,
		KNXDClient: opts.KNXDClient,
	})
	b.health.SetDeviceCount(0) // Starts empty; updated after registry load
	if opts.Logger != nil {
		b.health.SetLogger(opts.Logger)
	}

	return b, nil
}

// Start begins bridge operation.
// This subscribes to MQTT topics, sets up the KNX telegram handler,
// and starts health reporting.
func (b *Bridge) Start(ctx context.Context) error {
	// Load devices from registry (sole source of device mappings)
	b.loadDevicesFromRegistry(ctx)

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

	b.mappingMu.RLock()
	deviceCount := len(b.deviceToGAs)
	b.mappingMu.RUnlock()

	b.logInfo("bridge started",
		"bridge_id", b.cfg.Bridge.ID,
		"devices", deviceCount)

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

// loadDevicesFromRegistry loads KNX devices from the device registry and
// builds the bridge's device mappings. The registry is the sole source of
// device→GA mappings — devices are created via ETS import or the admin panel.
func (b *Bridge) loadDevicesFromRegistry(ctx context.Context) {
	if b.registry == nil {
		return
	}

	devices, err := b.registry.GetKNXDevices(ctx)
	if err != nil {
		b.logError("failed to load devices from registry", err)
		return
	}

	if len(devices) == 0 {
		return
	}

	b.mappingMu.Lock()
	defer b.mappingMu.Unlock()

	loaded := 0
	for _, dev := range devices {
		// Convert registry device functions to address configs.
		// Uses stored DPT and flags from the registry, falling back to
		// inference only for legacy devices that lack structured data.
		addresses := make(map[string]AddressConfig)
		for fn, fm := range dev.Functions {
			if fn == "" || fm.GA == "" {
				continue
			}

			dpt := fm.DPT
			if dpt == "" {
				dpt = inferDPTFromFunction(fn) // backward compat fallback
			}
			flags := fm.Flags
			if len(flags) == 0 {
				flags = inferFlagsFromFunction(fn) // backward compat fallback
			}

			addresses[fn] = AddressConfig{
				GA:    fm.GA,
				DPT:   dpt,
				Flags: flags,
			}
		}

		if len(addresses) == 0 {
			continue
		}

		// Build GA mappings (one GA may map to multiple devices)
		b.deviceToGAs[dev.ID] = addresses
		for fn, addr := range addresses {
			b.gaToDevice[addr.GA] = append(b.gaToDevice[addr.GA], GAMapping{
				DeviceID: dev.ID,
				Function: fn,
				Type:     dev.Type,
				DPT:      addr.DPT,
			})
		}

		// Track infrastructure devices for periodic polling
		if dev.Domain == "infrastructure" {
			b.infrastructureIDs[dev.ID] = true
		}

		loaded++
	}

	if loaded > 0 {
		b.logInfo("loaded devices from registry", "count", loaded)
	}
	// Update health reporter with total device count from maps
	b.health.SetDeviceCount(len(b.deviceToGAs))
}

// ReloadDevices reloads device mappings from the registry.
// Call this after ETS import or other operations that create new KNX devices
// so the bridge can control them without requiring a restart.
// After reloading, it sends read requests for all readable GAs to populate
// initial state.
func (b *Bridge) ReloadDevices(ctx context.Context) {
	b.loadDevicesFromRegistry(ctx)

	// Send read requests in background using the bridge's own context
	// (not the caller's — it may be an HTTP request context that gets cancelled).
	go b.readAllDevices(b.ctx)
}

// readAllDevices sends KNX read requests for all readable GAs across all
// devices. Used after import/reload to populate initial state values.
func (b *Bridge) readAllDevices(ctx context.Context) {
	readCtx, cancel := context.WithTimeout(ctx, readAllTimeout)
	defer cancel()

	b.mappingMu.RLock()
	devices := make(map[string]map[string]AddressConfig, len(b.deviceToGAs))
	for id, gas := range b.deviceToGAs {
		devices[id] = gas
	}
	b.mappingMu.RUnlock()

	readCount := 0
	for devID, deviceGAs := range devices {
		// Skip infrastructure devices — their status GAs share the same
		// physical GAs as per-room devices. Reading them would duplicate
		// reads and may return stale values from simulators.
		if b.infrastructureIDs[devID] {
			continue
		}
		for _, addr := range deviceGAs {
			if !addr.HasFlag("read") {
				continue
			}
			ga, err := ParseGroupAddress(addr.GA)
			if err != nil {
				continue
			}
			if err := b.knxd.SendRead(readCtx, ga); err != nil {
				continue
			}
			readCount++
			select {
			case <-readCtx.Done():
				b.logInfo("read-all interrupted", "reads_sent", readCount)
				return
			case <-time.After(interReadDelay):
			}
		}
	}

	if readCount > 0 {
		b.logInfo("initial read-all complete", "reads_sent", readCount)
	}
}

// inferFlagsFromFunction returns appropriate flags based on the function name.
// Uses the canonical function registry first, falling back to heuristics.
func inferFlagsFromFunction(fn string) []string {
	// Try canonical registry (handles names, aliases, and channel prefixes)
	if flags := DefaultFlagsForFunction(fn); flags != nil {
		return flags
	}

	// Heuristic fallback for truly unknown functions
	fnLower := strings.ToLower(fn)

	if strings.Contains(fnLower, "status") || strings.Contains(fnLower, "feedback") {
		return []string{"read", "transmit"}
	}

	return []string{"write"}
}

// inferDPTFromFunction returns the appropriate DPT based on the function name.
// Uses the canonical function registry first, falling back to heuristics for
// truly unknown function names.
func inferDPTFromFunction(fn string) string {
	// Try canonical registry (handles names, aliases, and channel prefixes)
	if dpt := DefaultDPTForFunction(fn); dpt != "" {
		return dpt
	}

	// Heuristic fallback for truly unknown functions
	fnLower := strings.ToLower(fn)

	if strings.Contains(fnLower, "temperature") || strings.Contains(fnLower, "setpoint") {
		return "9.001"
	}
	if strings.Contains(fnLower, "humidity") {
		return "9.007"
	}
	if strings.Contains(fnLower, "lux") {
		return "9.004"
	}
	if strings.Contains(fnLower, "brightness") || strings.Contains(fnLower, "position") ||
		strings.Contains(fnLower, "valve") || strings.Contains(fnLower, "slat") {
		return "5.001"
	}
	if strings.Contains(fnLower, "switch") || strings.Contains(fnLower, "button") {
		return "1.001"
	}

	return ""
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

	// Record telegram for passive discovery (before any early returns)
	// This builds a database of all devices and GAs seen on the bus
	if b.gaRecorder != nil {
		isResponse := t.APCI == APCIResponse
		b.gaRecorder.RecordTelegram(t.Source, gaStr, isResponse)
	}

	// Look up device mappings (one GA may map to multiple devices)
	b.mappingMu.RLock()
	mappings, ok := b.gaToDevice[gaStr]
	b.mappingMu.RUnlock()

	if !ok || len(mappings) == 0 {
		// Unknown GA, ignore (might be traffic we don't care about)
		return
	}

	// Decode the value using the best available DPT from the mappings.
	// Pick the first non-empty DPT (order may vary due to map iteration).
	dpt := ""
	for _, m := range mappings {
		if m.DPT != "" {
			dpt = m.DPT
			break
		}
	}
	value, err := b.decodeTelegramValue(t, dpt)
	if err != nil {
		b.logError("failed to decode telegram",
			fmt.Errorf("ga=%s dpt=%s: %w", gaStr, dpt, err))
		return
	}

	// Update each mapped device
	for _, mapping := range mappings {
		state := b.buildStateUpdate(mapping, value)

		if b.stateUnchanged(mapping.DeviceID, mapping.Function, value) {
			continue // No change for this device, skip
		}

		// Publish state message
		msg := NewStateMessage(mapping.DeviceID, gaStr, state)

		payload, err := json.Marshal(msg)
		if err != nil {
			b.logError("failed to marshal state", err)
			continue
		}

		topic := StateTopic(gaStr)
		if err := b.mqtt.Publish(topic, payload, 1, true); err != nil {
			b.logError("failed to publish state", err)
			continue
		}

		// Update device registry (if configured)
		if b.registry != nil {
			if err := b.registry.SetDeviceState(b.ctx, mapping.DeviceID, state); err != nil {
				b.logDebug("registry state update skipped",
					"device", mapping.DeviceID,
					"reason", err.Error())
			} else {
				if healthErr := b.registry.SetDeviceHealth(b.ctx, mapping.DeviceID, "online"); healthErr != nil {
					b.logDebug("registry health update skipped",
						"device", mapping.DeviceID,
						"reason", healthErr.Error())
				}
			}
		}
	}
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

// buildStateUpdate builds a state object from the decoded value.
// Uses the canonical function registry (functions.go) for state key lookup.
func (b *Bridge) buildStateUpdate(mapping GAMapping, value any) map[string]any {
	state := make(map[string]any)

	if mapping.Function == "" {
		return state
	}

	// Look up the state key from the canonical function registry.
	// This handles canonical names, aliases, and channel-prefixed functions.
	stateKey := StateKeyForFunction(mapping.Function)
	state[stateKey] = value

	// Special handling for presence: add last_motion timestamp
	if stateKey == "presence" {
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
	if valuesEqual(cached, value) {
		return true // Unchanged
	}

	// Update cache
	b.stateCache[deviceID][function] = value
	return false
}

// valuesEqual compares two values for equality, handling []byte specially
// since Go's == operator cannot compare slices directly.
func valuesEqual(a, b any) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle []byte specially - cannot use == on slices
	aBytes, aIsBytes := a.([]byte)
	bBytes, bIsBytes := b.([]byte)
	if aIsBytes && bIsBytes {
		if len(aBytes) != len(bBytes) {
			return false
		}
		for i := range aBytes {
			if aBytes[i] != bBytes[i] {
				return false
			}
		}
		return true
	}

	// For all other types, use direct comparison
	// This is safe because decode functions return bool, float64, uint8, etc.
	return a == b
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

// BridgeMetrics contains metrics data for the API metrics endpoint.
type BridgeMetrics struct {
	Connected      bool
	Status         string
	TelegramsTx    uint64
	TelegramsRx    uint64
	DevicesManaged int
}

// GetMetrics returns current bridge metrics for the API metrics endpoint.
func (b *Bridge) GetMetrics() BridgeMetrics {
	b.mappingMu.RLock()
	deviceCount := len(b.deviceToGAs)
	b.mappingMu.RUnlock()

	connected := false
	var stats KNXDStats
	status := "disconnected"

	if b.knxd != nil {
		connected = b.knxd.IsConnected()
		stats = b.knxd.Stats()
		if connected {
			status = "healthy"
		}
	}

	return BridgeMetrics{
		Connected:      connected,
		Status:         status,
		TelegramsTx:    stats.TelegramsTx,
		TelegramsRx:    stats.TelegramsRx,
		DevicesManaged: deviceCount,
	}
}
