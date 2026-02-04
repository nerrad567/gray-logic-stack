//go:build integration

package knx

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"
)

// Integration tests for the KNX bridge.
// Run with: go test -tags=integration -v ./internal/bridges/knx/...
//
// These tests require either:
// 1. A running MQTT broker (set KNX_TEST_MQTT_BROKER env var)
// 2. Or they will use the mock implementations
//
// For full integration testing, ensure Docker services are running:
//   docker compose -f docker-compose.dev.yml up -d

// TestIntegrationBridgeFullCycle tests the complete command → KNX → state cycle.
func TestIntegrationBridgeFullCycle(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createIntegrationTestConfig()

	b, err := NewBridge(BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})
	if err != nil {
		t.Fatalf("NewBridge() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	// Verify subscriptions
	subs := mqtt.GetSubscriptions()
	if len(subs) < 2 {
		t.Fatalf("Expected at least 2 subscriptions (command + request), got %d", len(subs))
	}

	// Clear initial health publishes
	mqtt.ClearPublished()

	// Phase 1: Send ON command
	t.Run("phase1_on_command", func(t *testing.T) {
		cmd := CommandMessage{
			ID:        "int-cmd-001",
			DeviceID:  "int-light-main",
			Command:   "on",
			Timestamp: time.Now().UTC(),
		}
		cmdPayload, _ := json.Marshal(cmd)
		b.handleMQTTMessage("graylogic/command/knx/int-light-main", cmdPayload)

		// Verify telegram sent
		telegrams := knxd.GetSentTelegrams()
		if len(telegrams) != 1 {
			t.Fatalf("Expected 1 telegram, got %d", len(telegrams))
		}

		// Verify correct GA (1/0/1 = switch)
		expected := GroupAddress{Main: 1, Middle: 0, Sub: 1}
		if telegrams[0].GA != expected {
			t.Errorf("GA = %v, want %v", telegrams[0].GA, expected)
		}

		// Verify DPT 1.001 ON (0x01)
		if len(telegrams[0].Data) != 1 || telegrams[0].Data[0] != 0x01 {
			t.Errorf("Data = %v, want [0x01]", telegrams[0].Data)
		}

		// Verify ack published
		verifyAckPublished(t, mqtt, AckAccepted)
	})

	knxd.ClearSent()
	mqtt.ClearPublished()

	// Phase 2: Simulate KNX status response → State published
	t.Run("phase2_knx_state_response", func(t *testing.T) {
		// Simulate device responding with status
		telegram := Telegram{
			Destination: GroupAddress{Main: 1, Middle: 0, Sub: 2}, // switch_status
			APCI:        APCIWrite,
			Data:        []byte{0x01}, // ON
		}
		knxd.SimulateTelegram(telegram)

		time.Sleep(100 * time.Millisecond)

		// Verify state published
		published := mqtt.GetPublished()
		var stateMsg *StateMessage
		for _, p := range published {
			if p.Topic == StateTopic("1/0/2") {
				var s StateMessage
				if err := json.Unmarshal(p.Payload, &s); err == nil {
					stateMsg = &s
				}
				break
			}
		}

		if stateMsg == nil {
			t.Fatal("State message not published")
		}

		if stateMsg.DeviceID != "int-light-main" {
			t.Errorf("DeviceID = %s, want int-light-main", stateMsg.DeviceID)
		}

		if on, ok := stateMsg.State["on"]; !ok || on != true {
			t.Errorf("State[on] = %v, want true", stateMsg.State["on"])
		}
	})

	knxd.ClearSent()
	mqtt.ClearPublished()

	// Phase 3: Dim command
	t.Run("phase3_dim_command", func(t *testing.T) {
		cmd := CommandMessage{
			ID:         "int-cmd-002",
			DeviceID:   "int-light-main",
			Command:    "dim",
			Parameters: map[string]any{"level": 50.0},
			Timestamp:  time.Now().UTC(),
		}
		cmdPayload, _ := json.Marshal(cmd)
		b.handleMQTTMessage("graylogic/command/knx/int-light-main", cmdPayload)

		telegrams := knxd.GetSentTelegrams()
		if len(telegrams) != 1 {
			t.Fatalf("Expected 1 telegram, got %d", len(telegrams))
		}

		// Verify brightness GA (1/0/3)
		expected := GroupAddress{Main: 1, Middle: 0, Sub: 3}
		if telegrams[0].GA != expected {
			t.Errorf("GA = %v, want %v", telegrams[0].GA, expected)
		}

		// Verify DPT 5.001 50% ≈ 127/128 (0x7F or 0x80)
		if len(telegrams[0].Data) != 1 || (telegrams[0].Data[0] != 0x7F && telegrams[0].Data[0] != 0x80) {
			t.Errorf("Data = %v, want [0x7F] or [0x80]", telegrams[0].Data)
		}
	})

	knxd.ClearSent()
	mqtt.ClearPublished()

	// Phase 4: Read all request
	t.Run("phase4_read_all_request", func(t *testing.T) {
		req := RequestMessage{
			RequestID: "int-req-001",
			Action:    "read_all",
			Timestamp: time.Now().UTC(),
		}
		reqPayload, _ := json.Marshal(req)
		b.handleMQTTMessage("graylogic/request/knx/all", reqPayload)

		// Allow time for reads with inter-read delays
		time.Sleep(500 * time.Millisecond)

		// Verify read requests were sent
		reads := knxd.GetReadRequests()
		// We have 2 devices with "read" flag: light (switch) and sensor (temperature)
		if len(reads) != 2 {
			t.Errorf("Expected 2 read requests, got %d", len(reads))
		}

		// Verify response published
		published := mqtt.GetPublished()
		var resp *ResponseMessage
		for _, p := range published {
			if p.Topic == ResponseTopic("int-req-001") {
				var r ResponseMessage
				if err := json.Unmarshal(p.Payload, &r); err == nil {
					resp = &r
				}
				break
			}
		}

		if resp == nil {
			t.Fatal("Response message not published")
		}

		if !resp.Success {
			t.Error("Response.Success = false, want true")
		}
	})
}

// TestIntegrationHealthReporting tests health status lifecycle.
func TestIntegrationHealthReporting(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createIntegrationTestConfig()
	cfg.Bridge.HealthInterval = 1 // 1 second for fast testing

	b, err := NewBridge(BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})
	if err != nil {
		t.Fatalf("NewBridge() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Wait for initial health messages
	time.Sleep(100 * time.Millisecond)

	// Verify starting and healthy status were published
	published := mqtt.GetPublished()
	var statuses []HealthStatus
	for _, p := range published {
		if p.Topic == HealthTopic() {
			var h HealthMessage
			if err := json.Unmarshal(p.Payload, &h); err == nil {
				statuses = append(statuses, h.Status)
			}
		}
	}

	if len(statuses) < 2 {
		t.Errorf("Expected at least 2 health messages, got %d", len(statuses))
	}

	// First should be "starting", then "healthy"
	hasStarting := false
	hasHealthy := false
	for _, s := range statuses {
		if s == HealthStarting {
			hasStarting = true
		}
		if s == HealthHealthy {
			hasHealthy = true
		}
	}

	if !hasStarting {
		t.Error("Expected 'starting' health status")
	}
	if !hasHealthy {
		t.Error("Expected 'healthy' health status")
	}

	mqtt.ClearPublished()

	// Wait for periodic health update
	time.Sleep(1500 * time.Millisecond)

	published = mqtt.GetPublished()
	hasPeriodicHealth := false
	for _, p := range published {
		if p.Topic == HealthTopic() {
			hasPeriodicHealth = true
			break
		}
	}

	if !hasPeriodicHealth {
		t.Error("Expected periodic health message")
	}

	// Stop bridge and verify stopping status
	mqtt.ClearPublished()
	b.Stop()

	// Check for stopping status
	published = mqtt.GetPublished()
	hasStopping := false
	for _, p := range published {
		if p.Topic == HealthTopic() {
			var h HealthMessage
			if err := json.Unmarshal(p.Payload, &h); err == nil {
				if h.Status == HealthStopping {
					hasStopping = true
				}
			}
		}
	}

	if !hasStopping {
		t.Error("Expected 'stopping' health status on shutdown")
	}
}

// TestIntegrationDegradedHealth tests health degrades when connections drop.
func TestIntegrationDegradedHealth(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createIntegrationTestConfig()

	b, err := NewBridge(BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})
	if err != nil {
		t.Fatalf("NewBridge() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Simulate knxd disconnect
	knxd.mu.Lock()
	knxd.connected = false
	knxd.mu.Unlock()

	// Force health check
	if err := b.health.PublishNow(); err != nil {
		t.Fatalf("PublishNow() error: %v", err)
	}

	published := mqtt.GetPublished()
	var lastHealth *HealthMessage
	for _, p := range published {
		if p.Topic == HealthTopic() {
			var h HealthMessage
			if err := json.Unmarshal(p.Payload, &h); err == nil {
				lastHealth = &h
			}
		}
	}

	if lastHealth == nil {
		t.Fatal("No health message published")
	}

	if lastHealth.Status != HealthDegraded {
		t.Errorf("Status = %s, want %s", lastHealth.Status, HealthDegraded)
	}

	if lastHealth.Reason != "knxd disconnected" {
		t.Errorf("Reason = %q, want 'knxd disconnected'", lastHealth.Reason)
	}
}

// TestIntegrationConcurrentCommands tests handling multiple concurrent commands.
func TestIntegrationConcurrentCommands(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createIntegrationTestConfig()

	b, err := NewBridge(BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})
	if err != nil {
		t.Fatalf("NewBridge() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()
	knxd.ClearSent()

	// Send 10 concurrent commands
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cmd := CommandMessage{
				ID:        "concurrent-cmd-" + string(rune('0'+n)),
				DeviceID:  "int-light-main",
				Command:   "on",
				Timestamp: time.Now().UTC(),
			}
			cmdPayload, _ := json.Marshal(cmd)
			b.handleMQTTMessage("graylogic/command/knx/int-light-main", cmdPayload)
		}(i)
	}

	wg.Wait()

	// All 10 commands should have been sent
	telegrams := knxd.GetSentTelegrams()
	if len(telegrams) != 10 {
		t.Errorf("Expected 10 telegrams, got %d", len(telegrams))
	}

	// All 10 acks should have been published
	published := mqtt.GetPublished()
	ackCount := 0
	for _, p := range published {
		var ack AckMessage
		if err := json.Unmarshal(p.Payload, &ack); err == nil {
			if ack.Status == AckAccepted {
				ackCount++
			}
		}
	}

	if ackCount != 10 {
		t.Errorf("Expected 10 acks, got %d", ackCount)
	}
}

// TestIntegrationStateChangeCoalescing tests that rapid state changes are handled correctly.
func TestIntegrationStateChangeCoalescing(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createIntegrationTestConfig()

	b, err := NewBridge(BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})
	if err != nil {
		t.Fatalf("NewBridge() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	mqtt.ClearPublished()

	// Simulate rapid toggling: ON → OFF → ON → OFF → ON
	values := []byte{0x01, 0x00, 0x01, 0x00, 0x01}
	for _, v := range values {
		telegram := Telegram{
			Destination: GroupAddress{Main: 1, Middle: 0, Sub: 2}, // switch_status
			APCI:        APCIWrite,
			Data:        []byte{v},
		}
		knxd.SimulateTelegram(telegram)
		time.Sleep(10 * time.Millisecond) // Small delay between telegrams
	}

	time.Sleep(100 * time.Millisecond)

	// Should have 5 state publishes (each value is different from previous)
	published := mqtt.GetPublished()
	stateCount := 0
	for _, p := range published {
		if p.Topic == StateTopic("1/0/2") {
			stateCount++
		}
	}

	if stateCount != 5 {
		t.Errorf("Expected 5 state publishes, got %d", stateCount)
	}
}

// TestIntegrationGracefulShutdown tests shutdown doesn't lose messages.
func TestIntegrationGracefulShutdown(t *testing.T) {
	mqtt := NewMockMQTTClient()
	knxd := NewMockConnector()
	cfg := createIntegrationTestConfig()

	b, err := NewBridge(BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: knxd,
	})
	if err != nil {
		t.Fatalf("NewBridge() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Send a command
	cmd := CommandMessage{
		ID:        "shutdown-cmd",
		DeviceID:  "int-light-main",
		Command:   "off",
		Timestamp: time.Now().UTC(),
	}
	cmdPayload, _ := json.Marshal(cmd)
	b.handleMQTTMessage("graylogic/command/knx/int-light-main", cmdPayload)

	// Immediately stop
	b.Stop()

	// Verify command was still processed
	telegrams := knxd.GetSentTelegrams()
	if len(telegrams) == 0 {
		t.Error("Command was not processed before shutdown")
	}

	// Verify stopping status was published
	published := mqtt.GetPublished()
	hasStopping := false
	for _, p := range published {
		if p.Topic == HealthTopic() {
			var h HealthMessage
			if err := json.Unmarshal(p.Payload, &h); err == nil {
				if h.Status == HealthStopping {
					hasStopping = true
				}
			}
		}
	}

	if !hasStopping {
		t.Error("Stopping status not published")
	}
}

// createIntegrationTestConfig creates a config for integration tests.
// Devices are not configured here — they come from the device registry in production.
func createIntegrationTestConfig() *Config {
	return &Config{
		Bridge: BridgeConfig{
			ID:             "int-test-bridge",
			HealthInterval: 30,
		},
		KNXD: KNXDSettings{
			Connection:        "tcp://localhost:6720",
			ConnectTimeout:    10,
			ReadTimeout:       30,
			ReconnectInterval: 5,
		},
		MQTT: MQTTSettings{
			Broker:    getTestMQTTBroker(),
			QoS:       1,
			KeepAlive: 60,
		},
		Logging: LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
	}
}

// getTestMQTTBroker returns the MQTT broker URL for testing.
func getTestMQTTBroker() string {
	if broker := os.Getenv("KNX_TEST_MQTT_BROKER"); broker != "" {
		return broker
	}
	return "tcp://localhost:1883"
}

// verifyAckPublished checks that an ack with the given status was published.
func verifyAckPublished(t *testing.T, mqtt *MockMQTTClient, expectedStatus AckStatus) {
	t.Helper()
	published := mqtt.GetPublished()
	for _, p := range published {
		var ack AckMessage
		if err := json.Unmarshal(p.Payload, &ack); err == nil {
			if ack.Status == expectedStatus {
				return
			}
		}
	}
	t.Errorf("Expected ack with status %s not found", expectedStatus)
}
