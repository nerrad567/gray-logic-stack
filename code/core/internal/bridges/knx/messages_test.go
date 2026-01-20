package knx

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCommandMessageJSON(t *testing.T) {
	cmd := CommandMessage{
		ID:        "cmd-123",
		Timestamp: time.Date(2026, 1, 20, 10, 30, 0, 0, time.UTC),
		DeviceID:  "light-living-1",
		Command:   "dim",
		Parameters: map[string]any{
			"level":         50,
			"transition_ms": 1000,
		},
		Source: "api",
		UserID: "user-darren",
	}

	// Marshal to JSON
	data, err := json.Marshal(&cmd)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify timestamp format
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}
	ts, ok := raw["timestamp"].(string)
	if !ok {
		t.Fatal("timestamp should be a string")
	}
	if ts != "2026-01-20T10:30:00Z" {
		t.Errorf("timestamp = %q, want 2026-01-20T10:30:00Z", ts)
	}

	// Unmarshal back
	var decoded CommandMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.ID != cmd.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, cmd.ID)
	}
	if decoded.DeviceID != cmd.DeviceID {
		t.Errorf("DeviceID = %q, want %q", decoded.DeviceID, cmd.DeviceID)
	}
	if decoded.Command != cmd.Command {
		t.Errorf("Command = %q, want %q", decoded.Command, cmd.Command)
	}
	if !decoded.Timestamp.Equal(cmd.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", decoded.Timestamp, cmd.Timestamp)
	}
}

func TestNewAckMessage(t *testing.T) {
	cmd := CommandMessage{
		ID:        "cmd-456",
		Timestamp: time.Now().UTC(),
		DeviceID:  "light-bedroom-1",
		Command:   "on",
		Source:    "automation",
	}

	ack := NewAckMessage(cmd, AckAccepted, "1/0/1")

	if ack.CommandID != cmd.ID {
		t.Errorf("CommandID = %q, want %q", ack.CommandID, cmd.ID)
	}
	if ack.DeviceID != cmd.DeviceID {
		t.Errorf("DeviceID = %q, want %q", ack.DeviceID, cmd.DeviceID)
	}
	if ack.Status != AckAccepted {
		t.Errorf("Status = %q, want %q", ack.Status, AckAccepted)
	}
	if ack.Protocol != "knx" {
		t.Errorf("Protocol = %q, want knx", ack.Protocol)
	}
	if ack.Address != "1/0/1" {
		t.Errorf("Address = %q, want 1/0/1", ack.Address)
	}
	if ack.Error != nil {
		t.Error("Error should be nil for accepted status")
	}
}

func TestNewAckError(t *testing.T) {
	cmd := CommandMessage{
		ID:       "cmd-789",
		DeviceID: "sensor-temp-1",
	}

	ack := NewAckError(cmd, "1/2/3", ErrCodeDeviceUnreachable, "Device did not respond", 3)

	if ack.Status != AckFailed {
		t.Errorf("Status = %q, want %q", ack.Status, AckFailed)
	}
	if ack.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if ack.Error.Code != ErrCodeDeviceUnreachable {
		t.Errorf("Error.Code = %q, want %q", ack.Error.Code, ErrCodeDeviceUnreachable)
	}
	if ack.Error.Message != "Device did not respond" {
		t.Errorf("Error.Message = %q, want 'Device did not respond'", ack.Error.Message)
	}
	if ack.Error.Retries != 3 {
		t.Errorf("Error.Retries = %d, want 3", ack.Error.Retries)
	}

	// Test timeout status
	ackTimeout := NewAckError(cmd, "1/2/3", ErrCodeTimeout, "Timeout", 2)
	if ackTimeout.Status != AckTimeout {
		t.Errorf("Timeout status = %q, want %q", ackTimeout.Status, AckTimeout)
	}
}

func TestNewStateMessage(t *testing.T) {
	state := map[string]any{
		"on":    true,
		"level": 75,
	}

	msg := NewStateMessage("light-kitchen-1", "1/0/5", state)

	if msg.DeviceID != "light-kitchen-1" {
		t.Errorf("DeviceID = %q, want light-kitchen-1", msg.DeviceID)
	}
	if msg.Protocol != "knx" {
		t.Errorf("Protocol = %q, want knx", msg.Protocol)
	}
	if msg.Address != "1/0/5" {
		t.Errorf("Address = %q, want 1/0/5", msg.Address)
	}
	if msg.State["on"] != true {
		t.Errorf("State[on] = %v, want true", msg.State["on"])
	}
	if msg.State["level"] != 75 {
		t.Errorf("State[level] = %v, want 75", msg.State["level"])
	}
}

func TestNewHealthMessage(t *testing.T) {
	stats := KNXDStats{
		TelegramsTx:  100,
		TelegramsRx:  500,
		ErrorsTotal:  2,
		LastActivity: time.Now().UTC(),
		Connected:    true,
	}
	startTime := time.Now().Add(-1 * time.Hour)

	msg := NewHealthMessage("knx-bridge-01", "1.0.0", HealthHealthy, stats, 42, startTime)

	if msg.Bridge != "knx-bridge-01" {
		t.Errorf("Bridge = %q, want knx-bridge-01", msg.Bridge)
	}
	if msg.Status != HealthHealthy {
		t.Errorf("Status = %q, want %q", msg.Status, HealthHealthy)
	}
	if msg.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", msg.Version)
	}
	if msg.DevicesManaged != 42 {
		t.Errorf("DevicesManaged = %d, want 42", msg.DevicesManaged)
	}
	if msg.UptimeSeconds < 3500 || msg.UptimeSeconds > 3700 {
		t.Errorf("UptimeSeconds = %d, want ~3600", msg.UptimeSeconds)
	}
	if msg.Connection == nil {
		t.Fatal("Connection should not be nil")
	}
	if msg.Connection.Status != "connected" {
		t.Errorf("Connection.Status = %q, want connected", msg.Connection.Status)
	}
	if msg.Statistics == nil {
		t.Fatal("Statistics should not be nil")
	}
	if msg.Statistics.MessagesSent != 100 {
		t.Errorf("Statistics.MessagesSent = %d, want 100", msg.Statistics.MessagesSent)
	}
	if msg.Statistics.MessagesReceived != 500 {
		t.Errorf("Statistics.MessagesReceived = %d, want 500", msg.Statistics.MessagesReceived)
	}
}

func TestNewLWTMessage(t *testing.T) {
	msg := NewLWTMessage("knx-bridge-01")

	if msg.Bridge != "knx-bridge-01" {
		t.Errorf("Bridge = %q, want knx-bridge-01", msg.Bridge)
	}
	if msg.Status != HealthOffline {
		t.Errorf("Status = %q, want %q", msg.Status, HealthOffline)
	}
	if msg.Reason != "unexpected_disconnect" {
		t.Errorf("Reason = %q, want unexpected_disconnect", msg.Reason)
	}
}

func TestTopicHelpers(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"CommandTopic", CommandTopic("1/0/1"), "graylogic/command/knx/1%2F0%2F1"},
		{"AckTopic", AckTopic("1/2/3"), "graylogic/ack/knx/1%2F2%2F3"},
		{"StateTopic", StateTopic("6/0/10"), "graylogic/state/knx/6%2F0%2F10"},
		{"HealthTopic", HealthTopic(), "graylogic/health/knx"},
		{"RequestTopic", RequestTopic("req-123"), "graylogic/request/knx/req-123"},
		{"ResponseTopic", ResponseTopic("req-123"), "graylogic/response/knx/req-123"},
		{"DiscoveryTopic", DiscoveryTopic(), "graylogic/discovery/knx"},
		{"CommandSubscribeTopic", CommandSubscribeTopic(), "graylogic/command/knx/#"},
		{"RequestSubscribeTopic", RequestSubscribeTopic(), "graylogic/request/knx/#"},
		{"ConfigTopic", ConfigTopic(), "graylogic/config/knx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestEncodeDecodeTopicAddress(t *testing.T) {
	tests := []struct {
		decoded string
		encoded string
	}{
		{"1/0/1", "1%2F0%2F1"},
		{"1/2/3", "1%2F2%2F3"},
		{"31/7/255", "31%2F7%2F255"},
		{"0/0/0", "0%2F0%2F0"},
	}

	for _, tt := range tests {
		t.Run(tt.decoded, func(t *testing.T) {
			// Test encoding
			encoded := EncodeTopicAddress(tt.decoded)
			if encoded != tt.encoded {
				t.Errorf("EncodeTopicAddress(%q) = %q, want %q", tt.decoded, encoded, tt.encoded)
			}

			// Test decoding
			decoded := DecodeTopicAddress(tt.encoded)
			if decoded != tt.decoded {
				t.Errorf("DecodeTopicAddress(%q) = %q, want %q", tt.encoded, decoded, tt.decoded)
			}
		})
	}
}

func TestAckMessageJSON(t *testing.T) {
	ack := AckMessage{
		CommandID: "cmd-test",
		Timestamp: time.Date(2026, 1, 20, 11, 0, 0, 0, time.UTC),
		DeviceID:  "light-1",
		Status:    AckFailed,
		Protocol:  "knx",
		Address:   "1/0/1",
		Error: &AckError{
			Code:    ErrCodeDeviceUnreachable,
			Message: "No response from device",
			Retries: 3,
		},
	}

	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded AckMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.CommandID != ack.CommandID {
		t.Errorf("CommandID = %q, want %q", decoded.CommandID, ack.CommandID)
	}
	if decoded.Status != ack.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, ack.Status)
	}
	if decoded.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if decoded.Error.Code != ack.Error.Code {
		t.Errorf("Error.Code = %q, want %q", decoded.Error.Code, ack.Error.Code)
	}
}

func TestStateMessageJSON(t *testing.T) {
	msg := StateMessage{
		DeviceID:  "blind-bedroom",
		Timestamp: time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC),
		State: map[string]any{
			"position": 50,
			"tilt":     30,
			"moving":   false,
		},
		Protocol: "knx",
		Address:  "2/0/1",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded StateMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.DeviceID != msg.DeviceID {
		t.Errorf("DeviceID = %q, want %q", decoded.DeviceID, msg.DeviceID)
	}
	// Note: JSON numbers unmarshal as float64
	if decoded.State["position"].(float64) != 50 {
		t.Errorf("State[position] = %v, want 50", decoded.State["position"])
	}
}

func TestHealthMessageJSON(t *testing.T) {
	connTime := time.Date(2026, 1, 20, 8, 0, 0, 0, time.UTC)
	msg := HealthMessage{
		Bridge:         "knx-bridge-01",
		Timestamp:      time.Date(2026, 1, 20, 12, 30, 0, 0, time.UTC),
		Status:         HealthHealthy,
		Version:        "1.0.0",
		UptimeSeconds:  16200,
		DevicesManaged: 25,
		Connection: &ConnectionStatus{
			Status:         "connected",
			Address:        "tcp://localhost:6720",
			ConnectedSince: &connTime,
		},
		Statistics: &BridgeStatistics{
			MessagesReceived: 1234,
			MessagesSent:     567,
			Errors:           2,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded HealthMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Bridge != msg.Bridge {
		t.Errorf("Bridge = %q, want %q", decoded.Bridge, msg.Bridge)
	}
	if decoded.Status != msg.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, msg.Status)
	}
	if decoded.UptimeSeconds != msg.UptimeSeconds {
		t.Errorf("UptimeSeconds = %d, want %d", decoded.UptimeSeconds, msg.UptimeSeconds)
	}
	if decoded.Statistics.MessagesReceived != msg.Statistics.MessagesReceived {
		t.Errorf("Statistics.MessagesReceived = %d, want %d",
			decoded.Statistics.MessagesReceived, msg.Statistics.MessagesReceived)
	}
}
