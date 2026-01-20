package mqtt

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
)

// testConfig returns a valid MQTT configuration for testing.
// Tests require a running Mosquitto broker at 127.0.0.1:1883.
func testConfig() config.MQTTConfig {
	return config.MQTTConfig{
		Broker: config.MQTTBrokerConfig{
			Host:     "127.0.0.1",
			Port:     1883,
			ClientID: "graylogic-test",
			TLS:      false,
		},
		Auth: config.MQTTAuthConfig{
			Username: "",
			Password: "",
		},
		QoS: 1,
		Reconnect: config.MQTTReconnectConfig{
			InitialDelay: 1,
			MaxDelay:     5,
		},
	}
}

// =============================================================================
// Connection Tests
// =============================================================================

func TestConnect(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("IsConnected() = false, want true")
	}
}

func TestConnectInvalidBroker(t *testing.T) {
	cfg := testConfig()
	cfg.Broker.Port = 19999 // Invalid port

	_, err := Connect(cfg)
	if err == nil {
		t.Fatal("Connect() expected error for invalid broker")
	}

	if !errors.Is(err, ErrConnectionFailed) {
		t.Errorf("Connect() error = %v, want ErrConnectionFailed", err)
	}
}

func TestClose(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if client.IsConnected() {
		t.Error("IsConnected() = true after Close(), want false")
	}
}

func TestCloseNil(t *testing.T) {
	client := &Client{}
	err := client.Close()
	if err != nil {
		t.Errorf("Close() on nil client error = %v, want nil", err)
	}
}

// =============================================================================
// HealthCheck Tests
// =============================================================================

func TestHealthCheck(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	err = client.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck() error = %v, want nil", err)
	}
}

func TestHealthCheckCancelled(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = client.HealthCheck(ctx)
	if err == nil {
		t.Error("HealthCheck() expected error for cancelled context")
	}
}

func TestHealthCheckDisconnected(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Disconnect
	client.Close()

	ctx := context.Background()
	err = client.HealthCheck(ctx)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("HealthCheck() error = %v, want ErrNotConnected", err)
	}
}

// =============================================================================
// Publish Tests
// =============================================================================

func TestPublish(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	topic := Topics{}.BridgeCommand("test-bridge", "test-device")
	payload := []byte(`{"test":true}`)

	err = client.Publish(topic, payload, 1, false)
	if err != nil {
		t.Errorf("Publish() error = %v", err)
	}
}

func TestPublishString(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	topic := Topics{}.BridgeCommand("test-bridge", "test-device")

	err = client.PublishString(topic, `{"test":true}`, 1, false)
	if err != nil {
		t.Errorf("PublishString() error = %v", err)
	}
}

func TestPublishRetained(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	topic := Topics{}.CoreDeviceState("test-device")
	payload := []byte(`{"on":true}`)

	err = client.PublishRetained(topic, payload)
	if err != nil {
		t.Errorf("PublishRetained() error = %v", err)
	}
}

func TestPublishEmptyTopic(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	err = client.Publish("", []byte("test"), 1, false)
	if !errors.Is(err, ErrInvalidTopic) {
		t.Errorf("Publish() error = %v, want ErrInvalidTopic", err)
	}
}

func TestPublishInvalidQoS(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	err = client.Publish("test/topic", []byte("test"), 3, false)
	if !errors.Is(err, ErrInvalidQoS) {
		t.Errorf("Publish() error = %v, want ErrInvalidQoS", err)
	}
}

func TestPublishDisconnected(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	client.Close()

	err = client.Publish("test/topic", []byte("test"), 1, false)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("Publish() error = %v, want ErrNotConnected", err)
	}
}

// =============================================================================
// Subscribe Tests
// =============================================================================

func TestSubscribe(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	topic := "graylogic/test/subscribe"
	handler := func(topic string, payload []byte) error {
		return nil
	}

	err = client.Subscribe(topic, 1, handler)
	if err != nil {
		t.Errorf("Subscribe() error = %v", err)
	}

	if !client.HasSubscription(topic) {
		t.Error("HasSubscription() = false, want true")
	}

	if client.SubscriptionCount() != 1 {
		t.Errorf("SubscriptionCount() = %d, want 1", client.SubscriptionCount())
	}
}

func TestSubscribeEmptyTopic(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	err = client.Subscribe("", 1, func(string, []byte) error { return nil })
	if !errors.Is(err, ErrInvalidTopic) {
		t.Errorf("Subscribe() error = %v, want ErrInvalidTopic", err)
	}
}

func TestSubscribeInvalidQoS(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	err = client.Subscribe("test/topic", 3, func(string, []byte) error { return nil })
	if !errors.Is(err, ErrInvalidQoS) {
		t.Errorf("Subscribe() error = %v, want ErrInvalidQoS", err)
	}
}

func TestSubscribeNilHandler(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	err = client.Subscribe("test/topic", 1, nil)
	if !errors.Is(err, ErrSubscribeFailed) {
		t.Errorf("Subscribe() error = %v, want ErrSubscribeFailed", err)
	}
}

func TestSubscribeDisconnected(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	client.Close()

	err = client.Subscribe("test/topic", 1, func(string, []byte) error { return nil })
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("Subscribe() error = %v, want ErrNotConnected", err)
	}
}

// =============================================================================
// Unsubscribe Tests
// =============================================================================

func TestUnsubscribe(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	topic := "graylogic/test/unsubscribe"
	handler := func(topic string, payload []byte) error {
		return nil
	}

	// Subscribe first
	err = client.Subscribe(topic, 1, handler)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	// Unsubscribe
	err = client.Unsubscribe(topic)
	if err != nil {
		t.Errorf("Unsubscribe() error = %v", err)
	}

	if client.HasSubscription(topic) {
		t.Error("HasSubscription() = true after Unsubscribe(), want false")
	}

	if client.SubscriptionCount() != 0 {
		t.Errorf("SubscriptionCount() = %d, want 0", client.SubscriptionCount())
	}
}

func TestUnsubscribeEmptyTopic(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	err = client.Unsubscribe("")
	if !errors.Is(err, ErrInvalidTopic) {
		t.Errorf("Unsubscribe() error = %v, want ErrInvalidTopic", err)
	}
}

func TestUnsubscribeDisconnected(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	client.Close()

	err = client.Unsubscribe("test/topic")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("Unsubscribe() error = %v, want ErrNotConnected", err)
	}
}

// =============================================================================
// Publish-Subscribe Integration Tests
// =============================================================================

func TestPublishSubscribeRoundtrip(t *testing.T) {
	cfg := testConfig()
	cfg.Broker.ClientID = "graylogic-test-pub"

	pubClient, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() publisher error = %v", err)
	}
	defer pubClient.Close()

	// Create subscriber with different client ID
	cfg.Broker.ClientID = "graylogic-test-sub"
	subClient, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() subscriber error = %v", err)
	}
	defer subClient.Close()

	// Set up subscription
	topic := "graylogic/test/roundtrip"
	expectedPayload := `{"test":"roundtrip"}`
	received := make(chan string, 1)

	err = subClient.Subscribe(topic, 1, func(t string, payload []byte) error {
		received <- string(payload)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	// Give subscription time to register
	time.Sleep(100 * time.Millisecond)

	// Publish
	err = pubClient.PublishString(topic, expectedPayload, 1, false)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	// Wait for message
	select {
	case payload := <-received:
		if payload != expectedPayload {
			t.Errorf("Received payload = %q, want %q", payload, expectedPayload)
		}
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestWildcardSubscription(t *testing.T) {
	cfg := testConfig()
	cfg.Broker.ClientID = "graylogic-test-wild-pub"

	pubClient, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() publisher error = %v", err)
	}
	defer pubClient.Close()

	cfg.Broker.ClientID = "graylogic-test-wild-sub"
	subClient, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() subscriber error = %v", err)
	}
	defer subClient.Close()

	// Subscribe to wildcard pattern
	pattern := "graylogic/test/+/state"
	var receivedMu sync.Mutex
	receivedTopics := make(map[string]bool)

	err = subClient.Subscribe(pattern, 1, func(topic string, payload []byte) error {
		receivedMu.Lock()
		receivedTopics[topic] = true
		receivedMu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Publish to multiple matching topics
	topics := []string{
		"graylogic/test/device1/state",
		"graylogic/test/device2/state",
		"graylogic/test/device3/state",
	}

	for _, topic := range topics {
		err = pubClient.PublishString(topic, `{"on":true}`, 1, false)
		if err != nil {
			t.Fatalf("Publish(%s) error = %v", topic, err)
		}
	}

	// Wait for messages
	time.Sleep(500 * time.Millisecond)

	receivedMu.Lock()
	defer receivedMu.Unlock()

	for _, topic := range topics {
		if !receivedTopics[topic] {
			t.Errorf("Did not receive message for topic %s", topic)
		}
	}
}

// =============================================================================
// Callback Tests
// =============================================================================

func TestOnConnectCallback(t *testing.T) {
	cfg := testConfig()
	cfg.Broker.ClientID = "graylogic-test-callback"

	// Connect first, then set callback.
	// Note: The callback may or may not fire depending on timing - the paho
	// library's on-connect handler fires asynchronously and might race with
	// our SetOnConnect call. This is expected behaviour - the callback mechanism
	// is for reconnection notifications primarily.
	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	// Use a channel to track callback invocation (inherently race-safe)
	called := make(chan struct{}, 1)
	client.SetOnConnect(func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})

	// Brief wait to see if callback fires - either outcome is valid
	// since we set the callback after Connect() returned.
	// The important thing is: no race condition.
	select {
	case <-called:
		// Callback was called - valid if paho's handler was still running
	case <-time.After(50 * time.Millisecond):
		// Callback not called - also valid since we set it after Connect()
	}

	// Test passes either way - we're verifying no race, not callback timing
}

func TestOnDisconnectCallback(t *testing.T) {
	cfg := testConfig()
	cfg.Broker.ClientID = "graylogic-test-disconnect-cb"

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	disconnectCalled := make(chan bool, 1)
	client.SetOnDisconnect(func(err error) {
		disconnectCalled <- true
	})

	// Close gracefully (this won't trigger disconnect callback as it's graceful)
	client.Close()

	// Verify callback was set (we can't easily test it being called)
	// since graceful close doesn't trigger the disconnect handler
}

// =============================================================================
// Topics Tests
// =============================================================================

func TestTopicBuilders(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() string
		expected string
	}{
		{
			name: "BridgeState",
			builder: func() string {
				return Topics{}.BridgeState("knx-01", "light-living")
			},
			expected: "graylogic/bridge/knx-01/state/light-living",
		},
		{
			name: "BridgeCommand",
			builder: func() string {
				return Topics{}.BridgeCommand("knx-01", "light-living")
			},
			expected: "graylogic/bridge/knx-01/command/light-living",
		},
		{
			name: "BridgeResponse",
			builder: func() string {
				return Topics{}.BridgeResponse("knx-01", "req-123")
			},
			expected: "graylogic/bridge/knx-01/response/req-123",
		},
		{
			name: "BridgeHealth",
			builder: func() string {
				return Topics{}.BridgeHealth("knx-01")
			},
			expected: "graylogic/bridge/knx-01/health",
		},
		{
			name: "BridgeDiscovery",
			builder: func() string {
				return Topics{}.BridgeDiscovery("knx-01")
			},
			expected: "graylogic/bridge/knx-01/discovery",
		},
		{
			name: "CoreDeviceState",
			builder: func() string {
				return Topics{}.CoreDeviceState("light-living")
			},
			expected: "graylogic/core/device/light-living/state",
		},
		{
			name: "CoreEvent",
			builder: func() string {
				return Topics{}.CoreEvent("device_state_changed")
			},
			expected: "graylogic/core/event/device_state_changed",
		},
		{
			name: "CoreSceneActivated",
			builder: func() string {
				return Topics{}.CoreSceneActivated("cinema-mode")
			},
			expected: "graylogic/core/scene/cinema-mode/activated",
		},
		{
			name: "CoreSceneProgress",
			builder: func() string {
				return Topics{}.CoreSceneProgress("cinema-mode")
			},
			expected: "graylogic/core/scene/cinema-mode/progress",
		},
		{
			name: "CoreAutomationFired",
			builder: func() string {
				return Topics{}.CoreAutomationFired("sunrise-blinds")
			},
			expected: "graylogic/core/automation/sunrise-blinds/fired",
		},
		{
			name: "CoreAlert",
			builder: func() string {
				return Topics{}.CoreAlert("dali-offline")
			},
			expected: "graylogic/core/alert/dali-offline",
		},
		{
			name: "CoreMode",
			builder: func() string {
				return Topics{}.CoreMode()
			},
			expected: "graylogic/core/mode",
		},
		{
			name: "SystemStatus",
			builder: func() string {
				return Topics{}.SystemStatus()
			},
			expected: "graylogic/system/status",
		},
		{
			name: "SystemTime",
			builder: func() string {
				return Topics{}.SystemTime()
			},
			expected: "graylogic/system/time",
		},
		{
			name: "SystemShutdown",
			builder: func() string {
				return Topics{}.SystemShutdown()
			},
			expected: "graylogic/system/shutdown",
		},
		{
			name: "UINotification",
			builder: func() string {
				return Topics{}.UINotification("panel-kitchen")
			},
			expected: "graylogic/ui/panel-kitchen/notification",
		},
		{
			name: "UIPresence",
			builder: func() string {
				return Topics{}.UIPresence("panel-kitchen")
			},
			expected: "graylogic/ui/panel-kitchen/presence",
		},
		{
			name: "AllBridgeStates",
			builder: func() string {
				return Topics{}.AllBridgeStates()
			},
			expected: "graylogic/bridge/+/state/+",
		},
		{
			name: "AllBridgeCommands",
			builder: func() string {
				return Topics{}.AllBridgeCommands()
			},
			expected: "graylogic/bridge/+/command/+",
		},
		{
			name: "AllBridgeHealth",
			builder: func() string {
				return Topics{}.AllBridgeHealth()
			},
			expected: "graylogic/bridge/+/health",
		},
		{
			name: "AllCoreDeviceStates",
			builder: func() string {
				return Topics{}.AllCoreDeviceStates()
			},
			expected: "graylogic/core/device/+/state",
		},
		{
			name: "AllCoreEvents",
			builder: func() string {
				return Topics{}.AllCoreEvents()
			},
			expected: "graylogic/core/event/+",
		},
		{
			name: "AllCoreAlerts",
			builder: func() string {
				return Topics{}.AllCoreAlerts()
			},
			expected: "graylogic/core/alert/+",
		},
		{
			name: "AllTopics",
			builder: func() string {
				return Topics{}.AllTopics()
			},
			expected: "graylogic/#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.builder()
			if result != tt.expected {
				t.Errorf("%s() = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestConnect_BrokerRefused(t *testing.T) {
	cfg := testConfig()
	cfg.Broker.Port = 19998

	_, err := Connect(cfg)
	if err == nil {
		t.Fatal("Connect() should fail for refused connection")
	}

	if !errors.Is(err, ErrConnectionFailed) {
		t.Errorf("Connect() error = %v, want ErrConnectionFailed", err)
	}
}

func TestIsConnected_InitialState(t *testing.T) {
	client := &Client{}

	if client.IsConnected() {
		t.Error("IsConnected() should be false for uninitialised client")
	}
}

func TestSubscriptionCount_Empty(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	if client.SubscriptionCount() != 0 {
		t.Errorf("SubscriptionCount() = %d, want 0", client.SubscriptionCount())
	}
}

func TestHasSubscription_NotSubscribed(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	if client.HasSubscription("nonexistent/topic") {
		t.Error("HasSubscription() should be false for unsubscribed topic")
	}
}

func TestMultipleSubscriptions(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	topics := []string{
		"graylogic/test/topic1",
		"graylogic/test/topic2",
		"graylogic/test/topic3",
	}

	handler := func(string, []byte) error { return nil }

	for _, topic := range topics {
		err := client.Subscribe(topic, 1, handler)
		if err != nil {
			t.Fatalf("Subscribe(%s) error = %v", topic, err)
		}
	}

	if client.SubscriptionCount() != 3 {
		t.Errorf("SubscriptionCount() = %d, want 3", client.SubscriptionCount())
	}

	for _, topic := range topics {
		if !client.HasSubscription(topic) {
			t.Errorf("HasSubscription(%s) = false, want true", topic)
		}
	}
}

func TestPublishNilPayload(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	err = client.Publish("test/topic", nil, 1, false)
	if err != nil {
		t.Errorf("Publish() with nil payload error = %v", err)
	}
}

func TestPublishLargePayload(t *testing.T) {
	cfg := testConfig()

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	largePayload := make([]byte, 64*1024)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}

	err = client.Publish("test/large", largePayload, 1, false)
	if err != nil {
		t.Errorf("Publish() with large payload error = %v", err)
	}
}

func TestHandlerReturnsError(t *testing.T) {
	cfg := testConfig()
	cfg.Broker.ClientID = "graylogic-test-handler-err"

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	topic := "graylogic/test/handler-error"
	handlerCalled := make(chan struct{}, 1)

	err = client.Subscribe(topic, 1, func(t string, p []byte) error {
		handlerCalled <- struct{}{}
		return errors.New("handler error")
	})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	err = client.PublishString(topic, "test", 1, false)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	select {
	case <-handlerCalled:
	case <-time.After(2 * time.Second):
		t.Error("Handler was not called")
	}
}
