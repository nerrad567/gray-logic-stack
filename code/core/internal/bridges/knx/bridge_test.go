package knx

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// MockMQTTClient implements MQTTClient for testing.
type MockMQTTClient struct {
	mu            sync.Mutex
	published     []mockPublish
	subscriptions []mockSubscription
	connected     bool
	handlers      map[string]func(topic string, payload []byte)
}

type mockPublish struct {
	Topic    string
	Payload  []byte
	QoS      byte
	Retained bool
}

type mockSubscription struct {
	Topic string
	QoS   byte
}

func NewMockMQTTClient() *MockMQTTClient {
	return &MockMQTTClient{
		connected: true,
		handlers:  make(map[string]func(topic string, payload []byte)),
	}
}

func (m *MockMQTTClient) Publish(topic string, payload []byte, qos byte, retained bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, mockPublish{
		Topic:    topic,
		Payload:  payload,
		QoS:      qos,
		Retained: retained,
	})
	return nil
}

func (m *MockMQTTClient) Subscribe(topic string, qos byte, handler func(topic string, payload []byte)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscriptions = append(m.subscriptions, mockSubscription{Topic: topic, QoS: qos})
	m.handlers[topic] = handler
	return nil
}

func (m *MockMQTTClient) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *MockMQTTClient) Disconnect(quiesce uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
}

func (m *MockMQTTClient) GetPublished() []mockPublish {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.published
}

func (m *MockMQTTClient) GetSubscriptions() []mockSubscription {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.subscriptions
}

func (m *MockMQTTClient) ClearPublished() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = nil
}

// SimulateMessage simulates receiving an MQTT message on a topic.
func (m *MockMQTTClient) SimulateMessage(topic string, payload []byte) {
	m.mu.Lock()
	handler, ok := m.handlers[topic]
	m.mu.Unlock()
	if ok {
		handler(topic, payload)
	}
}

// MockConnector implements Connector for testing.
type MockConnector struct {
	mu             sync.Mutex
	connected      bool
	stats          KNXDStats
	sentTelegrams  []sentTelegram
	readRequests   []GroupAddress
	onTelegramFunc func(Telegram)
	sendError      error
}

type sentTelegram struct {
	GA   GroupAddress
	Data []byte
}

func NewMockConnector() *MockConnector {
	return &MockConnector{
		connected: true,
	}
}

func (m *MockConnector) Send(ctx context.Context, ga GroupAddress, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendError != nil {
		return m.sendError
	}
	m.sentTelegrams = append(m.sentTelegrams, sentTelegram{GA: ga, Data: data})
	return nil
}

func (m *MockConnector) SendRead(ctx context.Context, ga GroupAddress) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readRequests = append(m.readRequests, ga)
	return nil
}

func (m *MockConnector) SetOnTelegram(callback func(Telegram)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onTelegramFunc = callback
}

func (m *MockConnector) Stats() KNXDStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stats
}

func (m *MockConnector) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *MockConnector) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockConnector) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

func (m *MockConnector) GetSentTelegrams() []sentTelegram {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sentTelegrams
}

func (m *MockConnector) GetReadRequests() []GroupAddress {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readRequests
}

func (m *MockConnector) ClearSent() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentTelegrams = nil
	m.readRequests = nil
}

// SimulateTelegram simulates receiving a KNX telegram.
func (m *MockConnector) SimulateTelegram(t Telegram) {
	m.mu.Lock()
	fn := m.onTelegramFunc
	m.mu.Unlock()
	if fn != nil {
		fn(t)
	}
}

func (m *MockConnector) SetSendError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendError = err
}

// createTestConfig creates a test configuration (no devices — those come from the registry).
func createTestConfig() *Config {
	return &Config{
		Bridge: BridgeConfig{
			ID:             "test-bridge",
			HealthInterval: 30,
		},
		KNXD: KNXDSettings{
			Connection:        "tcp://localhost:6720",
			ConnectTimeout:    10,
			ReadTimeout:       30,
			ReconnectInterval: 5,
		},
		MQTT: MQTTSettings{
			Broker:    "tcp://localhost:1883",
			QoS:       1,
			KeepAlive: 60,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// createTestDevices returns sample device configs for testing.
// In production, devices come from the device registry.
func createTestDevices() []DeviceConfig {
	return []DeviceConfig{
		{
			DeviceID: "light-living-main",
			Type:     "light_dimmer",
			Addresses: map[string]AddressConfig{
				"switch": {
					GA:    "1/2/3",
					DPT:   "1.001",
					Flags: []string{"write", "read"},
				},
				"switch_status": {
					GA:    "1/2/4",
					DPT:   "1.001",
					Flags: []string{"transmit"},
				},
				"brightness": {
					GA:    "1/2/5",
					DPT:   "5.001",
					Flags: []string{"write"},
				},
				"brightness_status": {
					GA:    "1/2/6",
					DPT:   "5.001",
					Flags: []string{"transmit"},
				},
			},
		},
		{
			DeviceID: "blind-living",
			Type:     "blind",
			Addresses: map[string]AddressConfig{
				"position": {
					GA:    "2/1/1",
					DPT:   "5.001",
					Flags: []string{"write"},
				},
				"position_status": {
					GA:    "2/1/2",
					DPT:   "5.001",
					Flags: []string{"transmit", "read"},
				},
				"stop": {
					GA:    "2/1/3",
					DPT:   "1.007",
					Flags: []string{"write"},
				},
			},
		},
		{
			DeviceID: "sensor-temp",
			Type:     "sensor",
			Addresses: map[string]AddressConfig{
				"temperature": {
					GA:    "6/0/1",
					DPT:   "9.001",
					Flags: []string{"transmit"},
				},
			},
		},
	}
}

// createTestBridge creates a bridge and pre-loads device maps.
// In production, device maps come from the registry; in tests we load them
// from createTestDevices() using BuildDeviceIndex for convenience.
func createTestBridge(t *testing.T, opts BridgeOptions) *Bridge {
	t.Helper()
	b, err := NewBridge(opts)
	if err != nil {
		t.Fatalf("NewBridge() error: %v", err)
	}
	// Pre-load device maps (simulates what the registry would provide)
	gaToDevice, deviceToGAs := BuildDeviceIndex(createTestDevices())
	b.mappingMu.Lock()
	b.gaToDevice = gaToDevice
	b.deviceToGAs = deviceToGAs
	b.mappingMu.Unlock()
	b.health.SetDeviceCount(len(deviceToGAs))
	return b
}

func TestNewBridge(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b, err := NewBridge(BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	if err != nil {
		t.Fatalf("NewBridge() error: %v", err)
	}

	if b == nil {
		t.Fatal("NewBridge() returned nil")
	}

	if b.health == nil {
		t.Error("NewBridge() did not create health reporter")
	}
}

func TestNewBridgeMissingConfig(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()

	_, err := NewBridge(BridgeOptions{
		Config:     nil,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	if err == nil {
		t.Error("NewBridge() expected error for nil config")
	}
}

func TestNewBridgeMissingMQTT(t *testing.T) {
	knxd := NewMockConnector()
	cfg := createTestConfig()

	_, err := NewBridge(BridgeOptions{
		Config:     cfg,
		MQTTClient: nil,
		KNXDClient: knxd,
	})

	if err == nil {
		t.Error("NewBridge() expected error for nil MQTT client")
	}
}

func TestNewBridgeMissingKNXD(t *testing.T) {
	mqtt := NewMockMQTTClient()
	cfg := createTestConfig()

	_, err := NewBridge(BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: nil,
	})

	if err == nil {
		t.Error("NewBridge() expected error for nil KNXD client")
	}
}

func TestBridgeStartStop(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Verify subscriptions were made
	subs := mqtt.GetSubscriptions()
	if len(subs) < 2 {
		t.Errorf("Expected at least 2 subscriptions, got %d", len(subs))
	}

	// Verify health message was published
	published := mqtt.GetPublished()
	hasHealth := false
	for _, p := range published {
		if p.Topic == HealthTopic() {
			hasHealth = true
			break
		}
	}
	if !hasHealth {
		t.Error("Expected health message to be published")
	}

	// Stop
	b.Stop()

	// Calling Stop again should be safe (sync.Once)
	b.Stop()
}

func TestBridgeOnCommand(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Send an "on" command
	cmd := CommandMessage{
		ID:        "cmd-001",
		DeviceID:  "light-living-main",
		Command:   "on",
		Timestamp: time.Now().UTC(),
	}
	cmdPayload, _ := json.Marshal(cmd)

	// Simulate receiving the command
	b.handleMQTTMessage("graylogic/command/knx/light-living-main", cmdPayload)

	// Check that a telegram was sent
	telegrams := knxd.GetSentTelegrams()
	if len(telegrams) != 1 {
		t.Fatalf("Expected 1 telegram sent, got %d", len(telegrams))
	}

	// Verify GA is 1/2/3 (switch address)
	expected := GroupAddress{Main: 1, Middle: 2, Sub: 3}
	if telegrams[0].GA != expected {
		t.Errorf("Telegram GA = %v, want %v", telegrams[0].GA, expected)
	}

	// Verify data is DPT 1.001 true (0x01)
	if len(telegrams[0].Data) != 1 || telegrams[0].Data[0] != 0x01 {
		t.Errorf("Telegram data = %v, want [0x01]", telegrams[0].Data)
	}

	// Check that an ack was published
	published := mqtt.GetPublished()
	hasAck := false
	for _, p := range published {
		if p.Topic == AckTopic("1/2/3") {
			hasAck = true
			var ack AckMessage
			if err := json.Unmarshal(p.Payload, &ack); err != nil {
				t.Errorf("Failed to unmarshal ack: %v", err)
			}
			if ack.Status != AckAccepted {
				t.Errorf("Ack status = %v, want %v", ack.Status, AckAccepted)
			}
			break
		}
	}
	if !hasAck {
		t.Error("Expected ack message to be published")
	}
}

func TestBridgeOffCommand(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Send an "off" command
	cmd := CommandMessage{
		ID:        "cmd-002",
		DeviceID:  "light-living-main",
		Command:   "off",
		Timestamp: time.Now().UTC(),
	}
	cmdPayload, _ := json.Marshal(cmd)

	b.handleMQTTMessage("graylogic/command/knx/light-living-main", cmdPayload)

	telegrams := knxd.GetSentTelegrams()
	if len(telegrams) != 1 {
		t.Fatalf("Expected 1 telegram sent, got %d", len(telegrams))
	}

	// Verify data is DPT 1.001 false (0x00)
	if len(telegrams[0].Data) != 1 || telegrams[0].Data[0] != 0x00 {
		t.Errorf("Telegram data = %v, want [0x00]", telegrams[0].Data)
	}
}

func TestBridgeDimCommand(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Send a "dim" command to 75%
	cmd := CommandMessage{
		ID:         "cmd-003",
		DeviceID:   "light-living-main",
		Command:    "dim",
		Parameters: map[string]any{"level": 75.0},
		Timestamp:  time.Now().UTC(),
	}
	cmdPayload, _ := json.Marshal(cmd)

	b.handleMQTTMessage("graylogic/command/knx/light-living-main", cmdPayload)

	telegrams := knxd.GetSentTelegrams()
	if len(telegrams) != 1 {
		t.Fatalf("Expected 1 telegram sent, got %d", len(telegrams))
	}

	// Verify GA is 1/2/5 (brightness address)
	expected := GroupAddress{Main: 1, Middle: 2, Sub: 5}
	if telegrams[0].GA != expected {
		t.Errorf("Telegram GA = %v, want %v", telegrams[0].GA, expected)
	}

	// Verify data is DPT 5.001 75% ≈ 191 (0xBF)
	if len(telegrams[0].Data) != 1 || telegrams[0].Data[0] != 0xBF {
		t.Errorf("Telegram data = %v, want [0xBF]", telegrams[0].Data)
	}
}

func TestBridgeSetPositionCommand(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Send a "set_position" command to 50%
	cmd := CommandMessage{
		ID:         "cmd-004",
		DeviceID:   "blind-living",
		Command:    "set_position",
		Parameters: map[string]any{"position": 50.0},
		Timestamp:  time.Now().UTC(),
	}
	cmdPayload, _ := json.Marshal(cmd)

	b.handleMQTTMessage("graylogic/command/knx/blind-living", cmdPayload)

	telegrams := knxd.GetSentTelegrams()
	if len(telegrams) != 1 {
		t.Fatalf("Expected 1 telegram sent, got %d", len(telegrams))
	}

	// Verify GA is 2/1/1 (position address)
	expected := GroupAddress{Main: 2, Middle: 1, Sub: 1}
	if telegrams[0].GA != expected {
		t.Errorf("Telegram GA = %v, want %v", telegrams[0].GA, expected)
	}

	// Verify data is DPT 5.001 50% ≈ 127/128 (0x7F or 0x80)
	if len(telegrams[0].Data) != 1 || (telegrams[0].Data[0] != 0x7F && telegrams[0].Data[0] != 0x80) {
		t.Errorf("Telegram data = %v, want [0x7F] or [0x80]", telegrams[0].Data)
	}
}

func TestBridgeStopCommand(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Send a "stop" command
	cmd := CommandMessage{
		ID:        "cmd-005",
		DeviceID:  "blind-living",
		Command:   "stop",
		Timestamp: time.Now().UTC(),
	}
	cmdPayload, _ := json.Marshal(cmd)

	b.handleMQTTMessage("graylogic/command/knx/blind-living", cmdPayload)

	telegrams := knxd.GetSentTelegrams()
	if len(telegrams) != 1 {
		t.Fatalf("Expected 1 telegram sent, got %d", len(telegrams))
	}

	// Verify GA is 2/1/3 (stop address)
	expected := GroupAddress{Main: 2, Middle: 1, Sub: 3}
	if telegrams[0].GA != expected {
		t.Errorf("Telegram GA = %v, want %v", telegrams[0].GA, expected)
	}
}

func TestBridgeUnknownDevice(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Send a command to unknown device
	cmd := CommandMessage{
		ID:        "cmd-006",
		DeviceID:  "unknown-device",
		Command:   "on",
		Timestamp: time.Now().UTC(),
	}
	cmdPayload, _ := json.Marshal(cmd)

	b.handleMQTTMessage("graylogic/command/knx/unknown-device", cmdPayload)

	// No telegrams should be sent
	telegrams := knxd.GetSentTelegrams()
	if len(telegrams) != 0 {
		t.Errorf("Expected 0 telegrams sent, got %d", len(telegrams))
	}

	// Error ack should be published
	published := mqtt.GetPublished()
	hasErrorAck := false
	for _, p := range published {
		var ack AckMessage
		if err := json.Unmarshal(p.Payload, &ack); err == nil {
			if ack.Status == AckFailed {
				hasErrorAck = true
				break
			}
		}
	}
	if !hasErrorAck {
		t.Error("Expected error ack to be published")
	}
}

func TestBridgeUnknownCommand(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Send an unknown command
	cmd := CommandMessage{
		ID:        "cmd-007",
		DeviceID:  "light-living-main",
		Command:   "explode", // Unknown
		Timestamp: time.Now().UTC(),
	}
	cmdPayload, _ := json.Marshal(cmd)

	b.handleMQTTMessage("graylogic/command/knx/light-living-main", cmdPayload)

	// No telegrams should be sent
	telegrams := knxd.GetSentTelegrams()
	if len(telegrams) != 0 {
		t.Errorf("Expected 0 telegrams sent, got %d", len(telegrams))
	}
}

func TestBridgeKNXTelegramToState(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Simulate receiving a KNX telegram for light switch status
	telegram := Telegram{
		Destination: GroupAddress{Main: 1, Middle: 2, Sub: 4}, // switch_status
		APCI:        APCIWrite,
		Data:        []byte{0x01}, // on
	}
	knxd.SimulateTelegram(telegram)

	// Give some time for processing
	time.Sleep(50 * time.Millisecond)

	// Check that a state message was published
	published := mqtt.GetPublished()
	hasState := false
	for _, p := range published {
		if p.Topic == StateTopic("1/2/4") {
			hasState = true
			var state StateMessage
			if err := json.Unmarshal(p.Payload, &state); err != nil {
				t.Errorf("Failed to unmarshal state: %v", err)
			}
			if state.DeviceID != "light-living-main" {
				t.Errorf("DeviceID = %s, want light-living-main", state.DeviceID)
			}
			// Check the state value
			if on, ok := state.State["on"]; !ok || on != true {
				t.Errorf("State[on] = %v, want true", state.State["on"])
			}
			break
		}
	}
	if !hasState {
		t.Error("Expected state message to be published")
	}
}

func TestBridgeKNXTelegramBrightness(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Simulate receiving a brightness status telegram (75%)
	telegram := Telegram{
		Destination: GroupAddress{Main: 1, Middle: 2, Sub: 6}, // brightness_status
		APCI:        APCIWrite,
		Data:        []byte{0xBF}, // 191/255 ≈ 75%
	}
	knxd.SimulateTelegram(telegram)

	time.Sleep(50 * time.Millisecond)

	published := mqtt.GetPublished()
	hasState := false
	for _, p := range published {
		if p.Topic == StateTopic("1/2/6") {
			hasState = true
			var state StateMessage
			if err := json.Unmarshal(p.Payload, &state); err != nil {
				t.Errorf("Failed to unmarshal state: %v", err)
			}
			if level, ok := state.State["level"].(float64); !ok || level < 74 || level > 76 {
				t.Errorf("State[level] = %v, want ~75", state.State["level"])
			}
			break
		}
	}
	if !hasState {
		t.Error("Expected state message to be published")
	}
}

func TestBridgeKNXTelegramTemperature(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Simulate receiving a temperature telegram (21.5°C)
	// DPT 9.001: 0x0C33 = 21.5°C
	// Encoding: E=1, M=1075 → (0.01 * 1075) * 2^1 = 21.5
	telegram := Telegram{
		Destination: GroupAddress{Main: 6, Middle: 0, Sub: 1}, // temperature
		APCI:        APCIWrite,
		Data:        []byte{0x0C, 0x33},
	}
	knxd.SimulateTelegram(telegram)

	time.Sleep(50 * time.Millisecond)

	published := mqtt.GetPublished()
	hasState := false
	for _, p := range published {
		if p.Topic == StateTopic("6/0/1") {
			hasState = true
			var state StateMessage
			if err := json.Unmarshal(p.Payload, &state); err != nil {
				t.Errorf("Failed to unmarshal state: %v", err)
			}
			if temp, ok := state.State["temperature"].(float64); !ok || temp < 21 || temp > 22 {
				t.Errorf("State[temperature] = %v, want ~21.5", state.State["temperature"])
			}
			break
		}
	}
	if !hasState {
		t.Error("Expected state message to be published")
	}
}

func TestBridgeStateChangeDetection(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// First telegram - should publish
	telegram := Telegram{
		Destination: GroupAddress{Main: 1, Middle: 2, Sub: 4},
		APCI:        APCIWrite,
		Data:        []byte{0x01},
	}
	knxd.SimulateTelegram(telegram)
	time.Sleep(50 * time.Millisecond)

	initialCount := len(mqtt.GetPublished())
	if initialCount == 0 {
		t.Fatal("Expected first telegram to publish state")
	}

	mqtt.ClearPublished()

	// Same value again - should NOT publish (cached)
	knxd.SimulateTelegram(telegram)
	time.Sleep(50 * time.Millisecond)

	if len(mqtt.GetPublished()) != 0 {
		t.Error("Expected no publish for unchanged state")
	}

	// Different value - should publish
	telegram.Data = []byte{0x00}
	knxd.SimulateTelegram(telegram)
	time.Sleep(50 * time.Millisecond)

	if len(mqtt.GetPublished()) == 0 {
		t.Error("Expected publish for changed state")
	}
}

func TestBridgeUnknownGA(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Telegram from unknown GA - should be ignored
	telegram := Telegram{
		Destination: GroupAddress{Main: 15, Middle: 7, Sub: 255}, // Not in config
		APCI:        APCIWrite,
		Data:        []byte{0x01},
	}
	knxd.SimulateTelegram(telegram)
	time.Sleep(50 * time.Millisecond)

	// Should not publish any state
	published := mqtt.GetPublished()
	for _, p := range published {
		if p.Topic == StateTopic("15/7/255") {
			t.Error("Should not publish state for unknown GA")
		}
	}
}

func TestBridgeReadStateRequest(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	knxd.ClearSent()
	mqtt.ClearPublished()

	// Send a read_state request
	req := RequestMessage{
		RequestID: "req-001",
		DeviceID:  "light-living-main",
		Action:    "read_state",
		Timestamp: time.Now().UTC(),
	}
	reqPayload, _ := json.Marshal(req)

	b.handleMQTTMessage("graylogic/request/knx/light-living-main", reqPayload)

	// Check that read requests were sent (only for addresses with "read" flag)
	reads := knxd.GetReadRequests()
	if len(reads) != 1 { // Only switch has "read" flag
		t.Errorf("Expected 1 read request, got %d", len(reads))
	}

	// Check response was published
	published := mqtt.GetPublished()
	hasResponse := false
	for _, p := range published {
		if p.Topic == ResponseTopic("req-001") {
			hasResponse = true
			var resp ResponseMessage
			if err := json.Unmarshal(p.Payload, &resp); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
			}
			if !resp.Success {
				t.Errorf("Response.Success = false, want true")
			}
			break
		}
	}
	if !hasResponse {
		t.Error("Expected response to be published")
	}
}

func TestBridgeReadAllRequest(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	knxd.ClearSent()
	mqtt.ClearPublished()

	// Send a read_all request
	req := RequestMessage{
		RequestID: "req-002",
		Action:    "read_all",
		Timestamp: time.Now().UTC(),
	}
	reqPayload, _ := json.Marshal(req)

	b.handleMQTTMessage("graylogic/request/knx/all", reqPayload)

	// Give time for read requests (with delays)
	time.Sleep(200 * time.Millisecond)

	// Check that read requests were sent
	reads := knxd.GetReadRequests()
	// We have 2 devices with "read" flags: light (switch) and blind (position_status)
	if len(reads) != 2 {
		t.Errorf("Expected 2 read requests, got %d", len(reads))
	}

	// Check response was published
	published := mqtt.GetPublished()
	hasResponse := false
	for _, p := range published {
		if p.Topic == ResponseTopic("req-002") {
			hasResponse = true
			var resp ResponseMessage
			if err := json.Unmarshal(p.Payload, &resp); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
			}
			if !resp.Success {
				t.Errorf("Response.Success = false, want true")
			}
			if reads, ok := resp.Data["reads_sent"].(float64); !ok || reads != 2 {
				t.Errorf("Response.Data[reads_sent] = %v, want 2", resp.Data["reads_sent"])
			}
			break
		}
	}
	if !hasResponse {
		t.Error("Expected response to be published")
	}
}

func TestBridgeRequestUnknownDevice(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Send request for unknown device
	req := RequestMessage{
		RequestID: "req-003",
		DeviceID:  "unknown-device",
		Action:    "read_state",
		Timestamp: time.Now().UTC(),
	}
	reqPayload, _ := json.Marshal(req)

	b.handleMQTTMessage("graylogic/request/knx/unknown-device", reqPayload)

	// Check error response was published
	published := mqtt.GetPublished()
	for _, p := range published {
		if p.Topic == ResponseTopic("req-003") {
			var resp ResponseMessage
			if err := json.Unmarshal(p.Payload, &resp); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
			}
			if resp.Success {
				t.Error("Response.Success = true, want false")
			}
			if resp.Error == nil {
				t.Error("Response.Error should not be nil")
			}
			return
		}
	}
	t.Error("Expected error response to be published")
}

func TestBridgeInvalidTopicFormat(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()
	knxd.ClearSent()

	// Send message with invalid topic (too few parts)
	b.handleMQTTMessage("invalid/topic", []byte("{}"))

	// No telegrams should be sent
	if len(knxd.GetSentTelegrams()) != 0 {
		t.Error("Expected no telegrams for invalid topic")
	}
}

func TestBridgeDimMissingLevel(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createTestConfig()

	b := createTestBridge(t, BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Dim command without level parameter
	cmd := CommandMessage{
		ID:        "cmd-008",
		DeviceID:  "light-living-main",
		Command:   "dim",
		Timestamp: time.Now().UTC(),
	}
	cmdPayload, _ := json.Marshal(cmd)

	b.handleMQTTMessage("graylogic/command/knx/light-living-main", cmdPayload)

	// No telegrams should be sent
	if len(knxd.GetSentTelegrams()) != 0 {
		t.Error("Expected no telegrams for dim without level")
	}

	// Error ack should be published
	published := mqtt.GetPublished()
	hasErrorAck := false
	for _, p := range published {
		var ack AckMessage
		if err := json.Unmarshal(p.Payload, &ack); err == nil {
			if ack.Status == AckFailed && ack.Error != nil {
				if ack.Error.Code == ErrCodeInvalidParameters {
					hasErrorAck = true
					break
				}
			}
		}
	}
	if !hasErrorAck {
		t.Error("Expected invalid parameters error ack")
	}
}

func TestResolveFunction(t *testing.T) {
	deviceGAs := map[string]AddressConfig{
		"switch":             {GA: "1/0/1", DPT: "1.001"},
		"switch_status":      {GA: "1/0/2", DPT: "1.001"},
		"ch_a_switch":        {GA: "2/0/1", DPT: "1.001"},
		"ch_a_switch_status": {GA: "2/0/2", DPT: "1.001"},
		"ch_b_switch":        {GA: "2/0/10", DPT: "1.001"},
		"ch_b_switch_status": {GA: "2/0/11", DPT: "1.001"},
		"brightness":         {GA: "1/0/3", DPT: "5.001"},
	}

	tests := []struct {
		name      string
		params    map[string]any
		fallbacks []string
		wantGA    string
		wantFn    string
		wantOK    bool
	}{
		{
			name:      "canonical fallback",
			params:    nil,
			fallbacks: []string{"switch"},
			wantGA:    "1/0/1",
			wantFn:    "switch",
			wantOK:    true,
		},
		{
			name:      "explicit function param overrides fallback",
			params:    map[string]any{"function": "ch_a_switch"},
			fallbacks: []string{"switch"},
			wantGA:    "2/0/1",
			wantFn:    "ch_a_switch",
			wantOK:    true,
		},
		{
			name:      "explicit function for channel B",
			params:    map[string]any{"function": "ch_b_switch"},
			fallbacks: []string{"switch"},
			wantGA:    "2/0/10",
			wantFn:    "ch_b_switch",
			wantOK:    true,
		},
		{
			name:      "explicit function not found falls through to fallback",
			params:    map[string]any{"function": "ch_z_switch"},
			fallbacks: []string{"switch"},
			wantGA:    "1/0/1",
			wantFn:    "switch",
			wantOK:    true,
		},
		{
			name:      "no match at all",
			params:    nil,
			fallbacks: []string{"position"},
			wantGA:    "",
			wantFn:    "",
			wantOK:    false,
		},
		{
			name:      "multiple fallbacks uses first match",
			params:    nil,
			fallbacks: []string{"brightness", "switch"},
			wantGA:    "1/0/3",
			wantFn:    "brightness",
			wantOK:    true,
		},
		{
			name:      "empty function param ignored",
			params:    map[string]any{"function": ""},
			fallbacks: []string{"switch"},
			wantGA:    "1/0/1",
			wantFn:    "switch",
			wantOK:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, fn, ok := resolveFunction(tt.params, deviceGAs, tt.fallbacks...)
			if ok != tt.wantOK {
				t.Errorf("resolveFunction() ok = %v, want %v", ok, tt.wantOK)
			}
			if fn != tt.wantFn {
				t.Errorf("resolveFunction() fn = %q, want %q", fn, tt.wantFn)
			}
			if ok && addr.GA != tt.wantGA {
				t.Errorf("resolveFunction() GA = %q, want %q", addr.GA, tt.wantGA)
			}
		})
	}
}

func TestBridge_StateKeyForFunction(t *testing.T) {
	tests := []struct {
		function string
		wantKey  string
	}{
		// Canonical names
		{"switch", "on"},
		{"switch_status", "on"},
		{"brightness", "level"},
		{"brightness_status", "level"},
		{"position", "position"},
		{"position_status", "position"},
		{"slat", "tilt"},
		{"slat_status", "tilt"},
		{"temperature", "temperature"},
		{"humidity", "humidity"},
		{"presence", "presence"},
		{"valve", "valve"},
		{"valve_status", "valve"},
		{"setpoint", "setpoint"},
		{"heating_output", "heating_output"},
		{"lux", "lux"},
		{"button_1", "button_1"},
		{"scene_number", "scene"},

		// Aliases
		{"motion", "presence"},
		{"actual_temperature", "temperature"},
		{"tilt", "tilt"},
		{"dim", "level"},

		// Channel prefixed
		{"ch_a_switch", "ch_a_on"},
		{"ch_b_valve_status", "ch_b_valve"},

		// Unknown fallback (returns function name as-is)
		{"unknown_function", "unknown_function"},
	}

	for _, tt := range tests {
		t.Run(tt.function, func(t *testing.T) {
			key := StateKeyForFunction(tt.function)
			if key != tt.wantKey {
				t.Errorf("StateKeyForFunction(%q) = %q, want %q", tt.function, key, tt.wantKey)
			}
		})
	}
}
