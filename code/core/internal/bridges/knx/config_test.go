//nolint:goconst // Test files use repeated literals for clarity
package knx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
bridge:
  id: "test-knx-bridge"
  health_interval: 30

knxd:
  connection: "tcp://localhost:6720"
  connect_timeout: 10
  read_timeout: 30
  reconnect_interval: 5

mqtt:
  broker: "tcp://localhost:1883"
  client_id: "test-knx-mqtt"
  qos: 1
  keep_alive: 60

logging:
  level: "info"
  format: "json"

devices:
  - device_id: "light-living-main"
    type: "light_dimmer"
    addresses:
      switch:
        ga: "1/0/1"
        dpt: "1.001"
        flags: ["write"]
      brightness:
        ga: "1/0/2"
        dpt: "5.001"
        flags: ["write"]
      brightness_status:
        ga: "6/0/2"
        dpt: "5.001"
        flags: ["read", "transmit"]
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify bridge settings
	if cfg.Bridge.ID != "test-knx-bridge" {
		t.Errorf("Bridge.ID = %q, want %q", cfg.Bridge.ID, "test-knx-bridge")
	}
	if cfg.Bridge.HealthInterval != 30 {
		t.Errorf("Bridge.HealthInterval = %d, want 30", cfg.Bridge.HealthInterval)
	}

	// Verify KNXD settings
	if cfg.KNXD.Connection != "tcp://localhost:6720" {
		t.Errorf("KNXD.Connection = %q, want tcp://localhost:6720", cfg.KNXD.Connection)
	}

	// Verify MQTT settings
	if cfg.MQTT.Broker != "tcp://localhost:1883" {
		t.Errorf("MQTT.Broker = %q, want tcp://localhost:1883", cfg.MQTT.Broker)
	}
	if cfg.MQTT.ClientID != "test-knx-mqtt" {
		t.Errorf("MQTT.ClientID = %q, want test-knx-mqtt", cfg.MQTT.ClientID)
	}

}

func TestLoadConfigDefaults(t *testing.T) {
	// Create minimal config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Minimal config with just required fields
	configContent := `
bridge:
  id: "minimal-bridge"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify defaults are applied
	if cfg.Bridge.HealthInterval != 30 {
		t.Errorf("Default HealthInterval = %d, want 30", cfg.Bridge.HealthInterval)
	}
	if cfg.KNXD.Connection != "tcp://localhost:6720" {
		t.Errorf("Default KNXD.Connection = %q, want tcp://localhost:6720", cfg.KNXD.Connection)
	}
	if cfg.MQTT.Broker != "tcp://localhost:1883" {
		t.Errorf("Default MQTT.Broker = %q, want tcp://localhost:1883", cfg.MQTT.Broker)
	}
	if cfg.MQTT.QoS != 1 {
		t.Errorf("Default MQTT.QoS = %d, want 1", cfg.MQTT.QoS)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Default Logging.Level = %q, want info", cfg.Logging.Level)
	}
}

func TestLoadConfigEnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
bridge:
  id: "env-test-bridge"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set environment variables
	t.Setenv("KNX_BRIDGE_ID", "override-bridge-id")
	t.Setenv("KNX_BRIDGE_KNXD_CONNECTION", "tcp://knxd.local:6720")
	t.Setenv("KNX_BRIDGE_MQTT_BROKER", "tcp://mqtt.local:1883")
	t.Setenv("KNX_BRIDGE_MQTT_USERNAME", "test-user")
	t.Setenv("KNX_BRIDGE_MQTT_PASSWORD", "test-pass")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Bridge.ID != "override-bridge-id" {
		t.Errorf("Bridge.ID = %q, want override-bridge-id", cfg.Bridge.ID)
	}
	if cfg.KNXD.Connection != "tcp://knxd.local:6720" {
		t.Errorf("KNXD.Connection = %q, want tcp://knxd.local:6720", cfg.KNXD.Connection)
	}
	if cfg.MQTT.Broker != "tcp://mqtt.local:1883" {
		t.Errorf("MQTT.Broker = %q, want tcp://mqtt.local:1883", cfg.MQTT.Broker)
	}
	if cfg.MQTT.Username != "test-user" {
		t.Errorf("MQTT.Username = %q, want test-user", cfg.MQTT.Username)
	}
	if cfg.MQTT.Password != "test-pass" {
		t.Errorf("MQTT.Password = %q, want test-pass", cfg.MQTT.Password)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError string
	}{
		{
			name: "missing bridge ID",
			config: Config{
				Bridge:  BridgeConfig{ID: "", HealthInterval: 30},
				KNXD:    KNXDSettings{Connection: "tcp://localhost:6720", ConnectTimeout: 10, ReadTimeout: 30},
				MQTT:    MQTTSettings{Broker: "tcp://localhost:1883", QoS: 1},
				Logging: LoggingConfig{Level: "info", Format: "json"},
			},
			wantError: "bridge.id is required",
		},
		{
			name: "invalid health interval",
			config: Config{
				Bridge:  BridgeConfig{ID: "test", HealthInterval: 0},
				KNXD:    KNXDSettings{Connection: "tcp://localhost:6720", ConnectTimeout: 10, ReadTimeout: 30},
				MQTT:    MQTTSettings{Broker: "tcp://localhost:1883", QoS: 1},
				Logging: LoggingConfig{Level: "info", Format: "json"},
			},
			wantError: "health_interval must be at least 1",
		},
		{
			name: "invalid QoS",
			config: Config{
				Bridge:  BridgeConfig{ID: "test", HealthInterval: 30},
				KNXD:    KNXDSettings{Connection: "tcp://localhost:6720", ConnectTimeout: 10, ReadTimeout: 30},
				MQTT:    MQTTSettings{Broker: "tcp://localhost:1883", QoS: 3},
				Logging: LoggingConfig{Level: "info", Format: "json"},
			},
			wantError: "mqtt.qos must be 0, 1, or 2",
		},
		{
			name: "invalid log level",
			config: Config{
				Bridge:  BridgeConfig{ID: "test", HealthInterval: 30},
				KNXD:    KNXDSettings{Connection: "tcp://localhost:6720", ConnectTimeout: 10, ReadTimeout: 30},
				MQTT:    MQTTSettings{Broker: "tcp://localhost:1883", QoS: 1},
				Logging: LoggingConfig{Level: "verbose", Format: "json"},
			},
			wantError: "logging.level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err == nil {
				t.Fatal("Validate() should have returned an error")
			}
			if !containsString(err.Error(), tt.wantError) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.wantError)
			}
		})
	}
}

func TestConfigValidationSuccess(t *testing.T) {
	cfg := Config{
		Bridge:  BridgeConfig{ID: "test-bridge", HealthInterval: 30},
		KNXD:    KNXDSettings{Connection: "tcp://localhost:6720", ConnectTimeout: 10, ReadTimeout: 30, ReconnectInterval: 5},
		MQTT:    MQTTSettings{Broker: "tcp://localhost:1883", QoS: 1},
		Logging: LoggingConfig{Level: "info", Format: "json"},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() returned unexpected error: %v", err)
	}
}

func TestToKNXDConfig(t *testing.T) {
	cfg := Config{
		KNXD: KNXDSettings{
			Connection:        "tcp://knxd.local:6720",
			ConnectTimeout:    15,
			ReadTimeout:       45,
			ReconnectInterval: 10,
		},
	}

	knxdCfg := cfg.ToKNXDConfig()

	if knxdCfg.Connection != "tcp://knxd.local:6720" {
		t.Errorf("Connection = %q, want tcp://knxd.local:6720", knxdCfg.Connection)
	}
	if knxdCfg.ConnectTimeout.Seconds() != 15 {
		t.Errorf("ConnectTimeout = %v, want 15s", knxdCfg.ConnectTimeout)
	}
	if knxdCfg.ReadTimeout.Seconds() != 45 {
		t.Errorf("ReadTimeout = %v, want 45s", knxdCfg.ReadTimeout)
	}
	if knxdCfg.ReconnectInterval.Seconds() != 10 {
		t.Errorf("ReconnectInterval = %v, want 10s", knxdCfg.ReconnectInterval)
	}
}

func TestGetMQTTClientID(t *testing.T) {
	// With explicit client ID
	cfg := Config{
		Bridge: BridgeConfig{ID: "knx-01"},
		MQTT:   MQTTSettings{ClientID: "custom-client-id"},
	}
	if got := cfg.GetMQTTClientID(); got != "custom-client-id" {
		t.Errorf("GetMQTTClientID() = %q, want custom-client-id", got)
	}

	// Without explicit client ID (should use bridge ID)
	cfg.MQTT.ClientID = ""
	if got := cfg.GetMQTTClientID(); got != "knx-01-mqtt" {
		t.Errorf("GetMQTTClientID() = %q, want knx-01-mqtt", got)
	}
}

func TestBuildDeviceIndex(t *testing.T) {
	devices := []DeviceConfig{
		{
			DeviceID: "light-living",
			Type:     "light_dimmer",
			Addresses: map[string]AddressConfig{
				"switch":            {GA: "1/0/1", DPT: "1.001", Flags: []string{"write"}},
				"brightness":        {GA: "1/0/2", DPT: "5.001", Flags: []string{"write"}},
				"switch_status":     {GA: "6/0/1", DPT: "1.001", Flags: []string{"transmit"}},
				"brightness_status": {GA: "6/0/2", DPT: "5.001", Flags: []string{"read", "transmit"}},
			},
		},
		{
			DeviceID: "blind-bedroom",
			Type:     "blind",
			Addresses: map[string]AddressConfig{
				"position":        {GA: "2/0/1", DPT: "5.001", Flags: []string{"write"}},
				"position_status": {GA: "7/0/1", DPT: "5.001", Flags: []string{"transmit"}},
			},
		},
	}

	gaToDevice, deviceToGAs := BuildDeviceIndex(devices)

	// Check gaToDevice (only transmit GAs should be indexed)
	if len(gaToDevice) != 3 {
		t.Errorf("len(gaToDevice) = %d, want 3 (only transmit GAs)", len(gaToDevice))
	}

	// Check specific GA mapping
	mappings, ok := gaToDevice["6/0/2"]
	if !ok || len(mappings) == 0 {
		t.Fatal("gaToDevice[6/0/2] not found")
	}
	mapping := mappings[0]
	if mapping.DeviceID != "light-living" {
		t.Errorf("mapping.DeviceID = %q, want light-living", mapping.DeviceID)
	}
	if mapping.Function != "brightness_status" {
		t.Errorf("mapping.Function = %q, want brightness_status", mapping.Function)
	}
	if mapping.DPT != "5.001" {
		t.Errorf("mapping.DPT = %q, want 5.001", mapping.DPT)
	}

	// Check deviceToGAs
	if len(deviceToGAs) != 2 {
		t.Errorf("len(deviceToGAs) = %d, want 2", len(deviceToGAs))
	}

	lightAddrs := deviceToGAs["light-living"]
	if len(lightAddrs) != 4 {
		t.Errorf("len(deviceToGAs[light-living]) = %d, want 4", len(lightAddrs))
	}

	switchAddr := lightAddrs["switch"]
	if switchAddr.GA != "1/0/1" {
		t.Errorf("switch.GA = %q, want 1/0/1", switchAddr.GA)
	}
}

func TestAddressConfigHasFlag(t *testing.T) {
	addr := AddressConfig{
		GA:    "1/0/1",
		DPT:   "1.001",
		Flags: []string{"read", "write"},
	}

	if !addr.HasFlag("read") {
		t.Error("HasFlag(read) = false, want true")
	}
	if !addr.HasFlag("write") {
		t.Error("HasFlag(write) = false, want true")
	}
	if addr.HasFlag("transmit") {
		t.Error("HasFlag(transmit) = true, want false")
	}
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
