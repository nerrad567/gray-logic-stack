package knx

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// mockPublisher implements HealthPublisher for testing.
type mockPublisher struct {
	mu        sync.Mutex
	connected bool
	messages  []publishedMessage
}

type publishedMessage struct {
	topic    string
	payload  []byte
	qos      byte
	retained bool
}

func newMockPublisher(connected bool) *mockPublisher {
	return &mockPublisher{connected: connected}
}

func (m *mockPublisher) Publish(topic string, payload []byte, qos byte, retained bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, publishedMessage{
		topic:    topic,
		payload:  payload,
		qos:      qos,
		retained: retained,
	})
	return nil
}

func (m *mockPublisher) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *mockPublisher) getMessages() []publishedMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]publishedMessage, len(m.messages))
	copy(result, m.messages)
	return result
}

// mockConnector implements Connector for testing.
type mockConnector struct {
	mu        sync.Mutex
	connected bool
	stats     KNXDStats
}

func newMockConnector(connected bool) *mockConnector {
	return &mockConnector{
		connected: connected,
		stats: KNXDStats{
			TelegramsTx:  100,
			TelegramsRx:  500,
			ErrorsTotal:  2,
			LastActivity: time.Now(),
			Connected:    connected,
		},
	}
}

func (m *mockConnector) Send(_ context.Context, _ GroupAddress, _ []byte) error {
	return nil
}

func (m *mockConnector) SendRead(_ context.Context, _ GroupAddress) error {
	return nil
}

func (m *mockConnector) SetOnTelegram(_ func(Telegram)) {}

func (m *mockConnector) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *mockConnector) Stats() KNXDStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stats
}

func (m *mockConnector) Close() error {
	return nil
}

func TestNewHealthReporter(t *testing.T) {
	pub := newMockPublisher(true)
	knxd := newMockConnector(true)

	cfg := HealthReporterConfig{
		BridgeID:   "test-bridge",
		Version:    "1.0.0",
		Interval:   5 * time.Second,
		Publisher:  pub,
		KNXDClient: knxd,
	}

	hr := NewHealthReporter(cfg)

	if hr.bridgeID != "test-bridge" {
		t.Errorf("bridgeID = %q, want test-bridge", hr.bridgeID)
	}
	if hr.version != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", hr.version)
	}
	if hr.interval != 5*time.Second {
		t.Errorf("interval = %v, want 5s", hr.interval)
	}
}

func TestHealthReporterDefaultInterval(t *testing.T) {
	cfg := HealthReporterConfig{
		BridgeID: "test-bridge",
		// Interval not set, should default to 30 seconds
	}

	hr := NewHealthReporter(cfg)

	if hr.interval != 30*time.Second {
		t.Errorf("default interval = %v, want 30s", hr.interval)
	}
}

func TestHealthReporterPublishNow(t *testing.T) {
	pub := newMockPublisher(true)
	knxd := newMockConnector(true)

	cfg := HealthReporterConfig{
		BridgeID:   "health-test",
		Version:    "2.0.0",
		Publisher:  pub,
		KNXDClient: knxd,
	}

	hr := NewHealthReporter(cfg)
	hr.SetDeviceCount(25)

	if err := hr.PublishNow(); err != nil {
		t.Fatalf("PublishNow failed: %v", err)
	}

	messages := pub.getMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	msg := messages[0]
	if msg.topic != "graylogic/health/knx" {
		t.Errorf("topic = %q, want graylogic/health/knx", msg.topic)
	}
	if msg.qos != 1 {
		t.Errorf("qos = %d, want 1", msg.qos)
	}
	if !msg.retained {
		t.Error("message should be retained")
	}

	// Parse and verify content
	var health HealthMessage
	if err := json.Unmarshal(msg.payload, &health); err != nil {
		t.Fatalf("failed to unmarshal health message: %v", err)
	}

	if health.Bridge != "health-test" {
		t.Errorf("Bridge = %q, want health-test", health.Bridge)
	}
	if health.Status != HealthHealthy {
		t.Errorf("Status = %q, want %q", health.Status, HealthHealthy)
	}
	if health.Version != "2.0.0" {
		t.Errorf("Version = %q, want 2.0.0", health.Version)
	}
	if health.DevicesManaged != 25 {
		t.Errorf("DevicesManaged = %d, want 25", health.DevicesManaged)
	}
}

func TestHealthReporterDegradedWhenKNXDDisconnected(t *testing.T) {
	pub := newMockPublisher(true)
	knxd := newMockConnector(false) // Disconnected

	cfg := HealthReporterConfig{
		BridgeID:   "test-bridge",
		Publisher:  pub,
		KNXDClient: knxd,
	}

	hr := NewHealthReporter(cfg)
	if err := hr.PublishNow(); err != nil {
		t.Fatalf("PublishNow failed: %v", err)
	}

	messages := pub.getMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	var health HealthMessage
	if err := json.Unmarshal(messages[0].payload, &health); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if health.Status != HealthDegraded {
		t.Errorf("Status = %q, want %q (knxd disconnected)", health.Status, HealthDegraded)
	}
	if health.Reason != "knxd disconnected" {
		t.Errorf("Reason = %q, want 'knxd disconnected'", health.Reason)
	}
}

func TestHealthReporterDegradedWhenMQTTDisconnected(t *testing.T) {
	pub := newMockPublisher(false) // MQTT disconnected
	knxd := newMockConnector(true)

	cfg := HealthReporterConfig{
		BridgeID:   "test-bridge",
		Publisher:  pub,
		KNXDClient: knxd,
	}

	hr := NewHealthReporter(cfg)

	// Determine status without publishing (since MQTT is down)
	status, reason := hr.determineStatus()

	if status != HealthDegraded {
		t.Errorf("Status = %q, want %q", status, HealthDegraded)
	}
	if reason != "MQTT disconnected" {
		t.Errorf("Reason = %q, want 'MQTT disconnected'", reason)
	}
}

func TestHealthReporterPublishStarting(t *testing.T) {
	pub := newMockPublisher(true)

	cfg := HealthReporterConfig{
		BridgeID:  "test-bridge",
		Publisher: pub,
	}

	hr := NewHealthReporter(cfg)
	if err := hr.PublishStarting(); err != nil {
		t.Fatalf("PublishStarting failed: %v", err)
	}

	messages := pub.getMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	var health HealthMessage
	if err := json.Unmarshal(messages[0].payload, &health); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if health.Status != HealthStarting {
		t.Errorf("Status = %q, want %q", health.Status, HealthStarting)
	}
}

func TestHealthReporterGetLWT(t *testing.T) {
	cfg := HealthReporterConfig{
		BridgeID: "lwt-test-bridge",
	}

	hr := NewHealthReporter(cfg)

	topic := hr.GetLWTTopic()
	if topic != "graylogic/health/knx" {
		t.Errorf("LWT topic = %q, want graylogic/health/knx", topic)
	}

	payload, err := hr.GetLWTPayload()
	if err != nil {
		t.Fatalf("GetLWTPayload failed: %v", err)
	}

	var health HealthMessage
	if err := json.Unmarshal(payload, &health); err != nil {
		t.Fatalf("failed to unmarshal LWT: %v", err)
	}

	if health.Bridge != "lwt-test-bridge" {
		t.Errorf("LWT Bridge = %q, want lwt-test-bridge", health.Bridge)
	}
	if health.Status != HealthOffline {
		t.Errorf("LWT Status = %q, want %q", health.Status, HealthOffline)
	}
	if health.Reason != "unexpected_disconnect" {
		t.Errorf("LWT Reason = %q, want unexpected_disconnect", health.Reason)
	}
}

func TestHealthReporterSetDeviceCount(t *testing.T) {
	pub := newMockPublisher(true)

	cfg := HealthReporterConfig{
		BridgeID:  "test-bridge",
		Publisher: pub,
	}

	hr := NewHealthReporter(cfg)

	hr.SetDeviceCount(10)
	hr.PublishNow()

	hr.SetDeviceCount(20)
	hr.PublishNow()

	messages := pub.getMessages()
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	var health1, health2 HealthMessage
	json.Unmarshal(messages[0].payload, &health1)
	json.Unmarshal(messages[1].payload, &health2)

	if health1.DevicesManaged != 10 {
		t.Errorf("first DevicesManaged = %d, want 10", health1.DevicesManaged)
	}
	if health2.DevicesManaged != 20 {
		t.Errorf("second DevicesManaged = %d, want 20", health2.DevicesManaged)
	}
}

func TestHealthReporterStartStop(t *testing.T) {
	pub := newMockPublisher(true)
	knxd := newMockConnector(true)

	cfg := HealthReporterConfig{
		BridgeID:   "lifecycle-test",
		Interval:   50 * time.Millisecond, // Short interval for testing
		Publisher:  pub,
		KNXDClient: knxd,
	}

	hr := NewHealthReporter(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hr.Start(ctx)

	// Wait for at least 2 health reports
	time.Sleep(150 * time.Millisecond)

	hr.Stop()

	messages := pub.getMessages()
	// Should have: initial + at least 2 periodic + stopping
	if len(messages) < 3 {
		t.Errorf("expected at least 3 messages, got %d", len(messages))
	}

	// Verify last message is stopping
	var lastHealth HealthMessage
	json.Unmarshal(messages[len(messages)-1].payload, &lastHealth)
	if lastHealth.Status != HealthStopping {
		t.Errorf("last Status = %q, want %q", lastHealth.Status, HealthStopping)
	}
}

func TestHealthReporterWithNoPublisher(t *testing.T) {
	cfg := HealthReporterConfig{
		BridgeID:  "no-publisher",
		Publisher: nil, // No publisher
	}

	hr := NewHealthReporter(cfg)

	// Should not panic or error
	if err := hr.PublishNow(); err != nil {
		t.Errorf("PublishNow with nil publisher should not error: %v", err)
	}
}

func TestHealthReporterUptimeCalculation(t *testing.T) {
	pub := newMockPublisher(true)

	cfg := HealthReporterConfig{
		BridgeID:  "uptime-test",
		Publisher: pub,
	}

	hr := NewHealthReporter(cfg)

	// Wait a bit to accumulate uptime
	time.Sleep(100 * time.Millisecond)

	hr.PublishNow()

	messages := pub.getMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	var health HealthMessage
	json.Unmarshal(messages[0].payload, &health)

	// Uptime should be at least 0 (could be 0 or 1 depending on timing)
	if health.UptimeSeconds < 0 {
		t.Errorf("UptimeSeconds = %d, should be >= 0", health.UptimeSeconds)
	}
}
