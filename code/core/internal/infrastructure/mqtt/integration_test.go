//go:build integration

package mqtt

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
)

// Integration tests for MQTT reconnection behaviour.
// These tests require a running MQTT broker at 127.0.0.1:1883.
//
// Run with:
//   go test -tags=integration -v ./internal/infrastructure/mqtt/...
//
// Note: Some tests may be flaky in CI due to timing dependencies.
// Consider running with: go test -tags=integration -count=1 -v ...

func integrationConfig() config.MQTTConfig {
	return config.MQTTConfig{
		Broker: config.MQTTBrokerConfig{
			Host:     "127.0.0.1",
			Port:     1883,
			ClientID: "graylogic-integration-test",
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

// TestIntegration_SubscriptionTracking verifies subscriptions are tracked.
//
// This test doesn't actually disconnect the broker (which would require
// external control), but verifies the subscription tracking mechanism
// that would be used during reconnection.
func TestIntegration_SubscriptionTracking(t *testing.T) {
	cfg := integrationConfig()
	cfg.Broker.ClientID = "graylogic-int-sub-track"

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	topics := []string{
		"graylogic/int/test/topic1",
		"graylogic/int/test/topic2",
		"graylogic/int/test/topic3",
	}

	handler := func(topic string, payload []byte) error {
		return nil
	}

	for _, topic := range topics {
		if err := client.Subscribe(topic, 1, handler); err != nil {
			t.Fatalf("Subscribe(%s) error = %v", topic, err)
		}
	}

	if client.SubscriptionCount() != len(topics) {
		t.Errorf("SubscriptionCount() = %d, want %d", client.SubscriptionCount(), len(topics))
	}

	for _, topic := range topics {
		if !client.HasSubscription(topic) {
			t.Errorf("HasSubscription(%s) = false, want true", topic)
		}
	}

	if err := client.Unsubscribe(topics[0]); err != nil {
		t.Fatalf("Unsubscribe() error = %v", err)
	}

	if client.SubscriptionCount() != len(topics)-1 {
		t.Errorf("SubscriptionCount() after unsubscribe = %d, want %d", client.SubscriptionCount(), len(topics)-1)
	}

	if client.HasSubscription(topics[0]) {
		t.Errorf("HasSubscription(%s) = true after unsubscribe", topics[0])
	}
}

// TestIntegration_CallbacksRegistered verifies callbacks can be set and cleared.
func TestIntegration_CallbacksRegistered(t *testing.T) {
	cfg := integrationConfig()
	cfg.Broker.ClientID = "graylogic-int-callbacks"

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	var connectCount int32
	var disconnectCount int32

	client.SetOnConnect(func() {
		atomic.AddInt32(&connectCount, 1)
	})

	client.SetOnDisconnect(func(err error) {
		atomic.AddInt32(&disconnectCount, 1)
	})

	client.SetOnConnect(nil)
	client.SetOnDisconnect(nil)
}

// TestIntegration_MessageRoundtrip verifies pub/sub works end-to-end.
func TestIntegration_MessageRoundtrip(t *testing.T) {
	cfg := integrationConfig()

	cfg.Broker.ClientID = "graylogic-int-pub"
	pubClient, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() publisher error = %v", err)
	}
	defer pubClient.Close()

	cfg.Broker.ClientID = "graylogic-int-sub"
	subClient, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() subscriber error = %v", err)
	}
	defer subClient.Close()

	topic := "graylogic/int/roundtrip"
	expected := "test-message-12345"

	received := make(chan string, 1)
	var once sync.Once

	err = subClient.Subscribe(topic, 1, func(t string, p []byte) error {
		once.Do(func() {
			received <- string(p)
		})
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	err = pubClient.PublishString(topic, expected, 1, false)
	if err != nil {
		t.Fatalf("PublishString() error = %v", err)
	}

	select {
	case msg := <-received:
		if msg != expected {
			t.Errorf("Received = %q, want %q", msg, expected)
		}
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

// TestIntegration_LoggerSet verifies logger can be set.
func TestIntegration_LoggerSet(t *testing.T) {
	cfg := integrationConfig()
	cfg.Broker.ClientID = "graylogic-int-logger"

	client, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	logger := &mockLogger{}
	client.SetLogger(logger)

	got := client.getLogger()
	if got == nil {
		t.Error("getLogger() = nil after SetLogger()")
	}

	client.SetLogger(nil)

	got = client.getLogger()
	if got != nil {
		t.Error("getLogger() should be nil after SetLogger(nil)")
	}
}

// mockLogger implements Logger interface for testing.
type mockLogger struct {
	errors []string
	warns  []string
	mu     sync.Mutex
}

func (l *mockLogger) Error(msg string, args ...any) {
	l.mu.Lock()
	l.errors = append(l.errors, msg)
	l.mu.Unlock()
}

func (l *mockLogger) Warn(msg string, args ...any) {
	l.mu.Lock()
	l.warns = append(l.warns, msg)
	l.mu.Unlock()
}
