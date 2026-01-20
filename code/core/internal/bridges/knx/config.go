package knx

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultKNXDConnection is the default knxd connection address.
const DefaultKNXDConnection = "tcp://localhost:6720"

// Config is the root configuration for the KNX bridge.
// Loaded from YAML with environment variable overrides.
type Config struct {
	Bridge  BridgeConfig   `yaml:"bridge"`
	KNXD    KNXDSettings   `yaml:"knxd"`
	MQTT    MQTTSettings   `yaml:"mqtt"`
	Devices []DeviceConfig `yaml:"devices"`
	Logging LoggingConfig  `yaml:"logging"`
}

// BridgeConfig contains bridge identity and operational settings.
type BridgeConfig struct {
	// ID uniquely identifies this bridge instance.
	// Used in MQTT client ID and health reporting.
	ID string `yaml:"id"`

	// HealthInterval is how often to publish health status (seconds).
	// Default: 30 seconds.
	HealthInterval int `yaml:"health_interval"`
}

// KNXDSettings contains knxd daemon connection settings.
// These override the defaults in KNXDConfig.
//
//nolint:revive // KNXDSettings is clearer than DSettings for external use
type KNXDSettings struct {
	// Connection is the knxd connection URL.
	// Supported formats:
	//   - "unix:///run/knxd" (Unix socket)
	//   - "tcp://localhost:6720" (TCP)
	// Default: "tcp://localhost:6720"
	Connection string `yaml:"connection"`

	// ConnectTimeout is the maximum time to wait for connection (seconds).
	// Default: 10 seconds.
	ConnectTimeout int `yaml:"connect_timeout"`

	// ReadTimeout is the timeout for read operations (seconds).
	// Default: 30 seconds.
	ReadTimeout int `yaml:"read_timeout"`

	// ReconnectInterval is the delay between reconnection attempts (seconds).
	// Default: 5 seconds.
	ReconnectInterval int `yaml:"reconnect_interval"`
}

// MQTTSettings contains MQTT broker connection settings.
type MQTTSettings struct {
	// Broker is the MQTT broker URL.
	// Example: "tcp://localhost:1883"
	Broker string `yaml:"broker"`

	// ClientID is the MQTT client identifier.
	// Should be unique per bridge instance.
	// Default: bridge.id + "-mqtt"
	ClientID string `yaml:"client_id"`

	// Username for MQTT authentication (optional).
	Username string `yaml:"username"`

	// Password for MQTT authentication (optional).
	// WARNING: Never log this value. Use String() method for safe logging.
	Password string `yaml:"password"`

	// QoS is the MQTT quality of service level (0, 1, or 2).
	// Default: 1 (at least once delivery).
	QoS int `yaml:"qos"`

	// KeepAlive is the MQTT keep-alive interval (seconds).
	// Default: 60 seconds.
	KeepAlive int `yaml:"keep_alive"`
}

// String returns a string representation with password masked.
// Use this for logging to prevent credential exposure.
func (m MQTTSettings) String() string {
	password := ""
	if m.Password != "" {
		password = "[REDACTED]"
	}
	return fmt.Sprintf("MQTTSettings{Broker:%q, ClientID:%q, Username:%q, Password:%s, QoS:%d, KeepAlive:%d}",
		m.Broker, m.ClientID, m.Username, password, m.QoS, m.KeepAlive)
}

// MarshalJSON implements json.Marshaler to redact password in JSON output.
// This prevents accidental password exposure in logs or API responses.
func (m MQTTSettings) MarshalJSON() ([]byte, error) {
	// Create a copy with redacted password for serialisation
	type redacted MQTTSettings
	safe := redacted(m)
	if safe.Password != "" {
		safe.Password = "[REDACTED]"
	}
	return json.Marshal(safe)
}

// LoggingConfig contains logging settings.
type LoggingConfig struct {
	// Level is the minimum log level: debug, info, warn, error.
	// Default: info
	Level string `yaml:"level"`

	// Format is the log output format: json or text.
	// Default: json
	Format string `yaml:"format"`
}

// DeviceConfig defines a device and its KNX group address mappings.
type DeviceConfig struct {
	// DeviceID is the Gray Logic device identifier.
	// Must match the device_id in Core's device registry.
	DeviceID string `yaml:"device_id"`

	// Type is the device type: light_switch, light_dimmer, blind, sensor, etc.
	Type string `yaml:"type"`

	// Addresses maps function names to group address configurations.
	// Common functions: switch, brightness, position, status, temperature.
	Addresses map[string]AddressConfig `yaml:"addresses"`
}

// AddressConfig defines a single KNX group address mapping.
type AddressConfig struct {
	// GA is the KNX group address in 3-level format (e.g., "1/2/3").
	GA string `yaml:"ga"`

	// DPT is the KNX datapoint type (e.g., "1.001", "5.001", "9.001").
	DPT string `yaml:"dpt"`

	// Flags indicate how this address is used.
	// Valid flags: read, write, transmit
	//   - read: Bridge can send read requests to this GA
	//   - write: Bridge can send write commands to this GA
	//   - transmit: Device transmits state changes on this GA
	Flags []string `yaml:"flags"`
}

// LoadConfig reads configuration from a YAML file.
//
// The configuration loading order is:
//  1. Default values (hardcoded)
//  2. YAML file values (override defaults)
//  3. Environment variables (override file values)
//
// Environment variables follow the pattern: KNX_BRIDGE_SECTION_KEY
// For example: KNX_BRIDGE_KNXD_CONNECTION, KNX_BRIDGE_MQTT_BROKER
//
// Parameters:
//   - path: Path to the YAML configuration file
//
// Returns:
//   - *Config: Loaded and validated configuration
//   - error: If file cannot be read, parsed, or validation fails
func LoadConfig(path string) (*Config, error) {
	// Start with defaults
	cfg := defaultConfig()

	// Read and parse YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}

// defaultConfig returns a Config with sensible defaults.
func defaultConfig() *Config {
	return &Config{
		Bridge: BridgeConfig{
			ID:             "knx-bridge-01",
			HealthInterval: 30,
		},
		KNXD: KNXDSettings{
			Connection:        DefaultKNXDConnection,
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
		Devices: []DeviceConfig{},
	}
}

// applyEnvOverrides applies environment variable overrides to the configuration.
// Environment variables follow the pattern: KNX_BRIDGE_SECTION_KEY
func applyEnvOverrides(cfg *Config) {
	// Bridge
	if v := os.Getenv("KNX_BRIDGE_ID"); v != "" {
		cfg.Bridge.ID = v
	}

	// KNXD
	if v := os.Getenv("KNX_BRIDGE_KNXD_CONNECTION"); v != "" {
		cfg.KNXD.Connection = v
	}

	// MQTT
	if v := os.Getenv("KNX_BRIDGE_MQTT_BROKER"); v != "" {
		cfg.MQTT.Broker = v
	}
	if v := os.Getenv("KNX_BRIDGE_MQTT_USERNAME"); v != "" {
		cfg.MQTT.Username = v
	}
	if v := os.Getenv("KNX_BRIDGE_MQTT_PASSWORD"); v != "" {
		cfg.MQTT.Password = v
	}
}

// Validate checks the configuration for errors.
//
// Returns:
//   - error: Description of validation failure, or nil if valid
func (c *Config) Validate() error {
	var errs []string

	errs = append(errs, c.validateBridge()...)
	errs = append(errs, c.validateKNXD()...)
	errs = append(errs, c.validateMQTT()...)
	errs = append(errs, c.validateDevices()...)
	errs = append(errs, c.validateLogging()...)

	if len(errs) > 0 {
		return fmt.Errorf("configuration errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// validateBridge validates bridge settings.
func (c *Config) validateBridge() []string {
	var errs []string
	if c.Bridge.ID == "" {
		errs = append(errs, "bridge.id is required")
	}
	if c.Bridge.HealthInterval < 1 {
		errs = append(errs, "bridge.health_interval must be at least 1 second")
	}
	return errs
}

// validateKNXD validates knxd connection settings.
func (c *Config) validateKNXD() []string {
	var errs []string
	if c.KNXD.Connection == "" {
		errs = append(errs, "knxd.connection is required")
	}
	if c.KNXD.ConnectTimeout < 1 {
		errs = append(errs, "knxd.connect_timeout must be at least 1 second")
	}
	if c.KNXD.ReadTimeout < 1 {
		errs = append(errs, "knxd.read_timeout must be at least 1 second")
	}
	return errs
}

// validateMQTT validates MQTT broker settings.
func (c *Config) validateMQTT() []string {
	var errs []string
	if c.MQTT.Broker == "" {
		errs = append(errs, "mqtt.broker is required")
	}
	if c.MQTT.QoS < 0 || c.MQTT.QoS > 2 {
		errs = append(errs, "mqtt.qos must be 0, 1, or 2")
	}
	return errs
}

// validateDevices validates device configurations.
func (c *Config) validateDevices() []string {
	var errs []string
	deviceIDs := make(map[string]bool)

	for i, dev := range c.Devices {
		if dev.DeviceID == "" {
			errs = append(errs, fmt.Sprintf("devices[%d].device_id is required", i))
			continue
		}
		if deviceIDs[dev.DeviceID] {
			errs = append(errs, fmt.Sprintf("devices[%d].device_id %q is duplicate", i, dev.DeviceID))
		}
		deviceIDs[dev.DeviceID] = true

		if dev.Type == "" {
			errs = append(errs, fmt.Sprintf("devices[%d].type is required", i))
		}
		if len(dev.Addresses) == 0 {
			errs = append(errs, fmt.Sprintf("devices[%d].addresses must have at least one entry", i))
		}

		errs = append(errs, validateDeviceAddresses(i, dev.Addresses)...)
	}

	return errs
}

// validateDeviceAddresses validates address configurations for a single device.
func validateDeviceAddresses(deviceIdx int, addresses map[string]AddressConfig) []string {
	var errs []string

	for name, addr := range addresses {
		if addr.GA == "" {
			errs = append(errs, fmt.Sprintf("devices[%d].addresses.%s.ga is required", deviceIdx, name))
		} else if _, err := ParseGroupAddress(addr.GA); err != nil {
			errs = append(errs, fmt.Sprintf("devices[%d].addresses.%s.ga %q is invalid: %v", deviceIdx, name, addr.GA, err))
		}

		if addr.DPT == "" {
			errs = append(errs, fmt.Sprintf("devices[%d].addresses.%s.dpt is required", deviceIdx, name))
		}

		for _, flag := range addr.Flags {
			if flag != "read" && flag != "write" && flag != "transmit" {
				errs = append(errs, fmt.Sprintf("devices[%d].addresses.%s.flags contains invalid value %q", deviceIdx, name, flag))
			}
		}
	}

	return errs
}

// validateLogging validates logging settings.
func (c *Config) validateLogging() []string {
	var errs []string

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		errs = append(errs, fmt.Sprintf("logging.level %q is invalid (use debug, info, warn, or error)", c.Logging.Level))
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[c.Logging.Format] {
		errs = append(errs, fmt.Sprintf("logging.format %q is invalid (use json or text)", c.Logging.Format))
	}

	return errs
}

// ToKNXDConfig converts settings to a KNXDConfig for the client.
func (c *Config) ToKNXDConfig() KNXDConfig {
	return KNXDConfig{
		Connection:        c.KNXD.Connection,
		ConnectTimeout:    time.Duration(c.KNXD.ConnectTimeout) * time.Second,
		ReadTimeout:       time.Duration(c.KNXD.ReadTimeout) * time.Second,
		ReconnectInterval: time.Duration(c.KNXD.ReconnectInterval) * time.Second,
	}
}

// GetHealthInterval returns the health reporting interval as a Duration.
func (c *Config) GetHealthInterval() time.Duration {
	return time.Duration(c.Bridge.HealthInterval) * time.Second
}

// GetMQTTClientID returns the MQTT client ID, defaulting to bridge ID if not set.
func (c *Config) GetMQTTClientID() string {
	if c.MQTT.ClientID != "" {
		return c.MQTT.ClientID
	}
	return c.Bridge.ID + "-mqtt"
}

// BuildDeviceIndex creates lookup maps for efficient device/GA resolution.
// Returns:
//   - gaToDevice: Maps "ga" → (device_id, function_name)
//   - deviceToGAs: Maps device_id → function_name → AddressConfig
func (c *Config) BuildDeviceIndex() (gaToDevice map[string]GAMapping, deviceToGAs map[string]map[string]AddressConfig) {
	gaToDevice = make(map[string]GAMapping)
	deviceToGAs = make(map[string]map[string]AddressConfig)

	for _, dev := range c.Devices {
		deviceToGAs[dev.DeviceID] = make(map[string]AddressConfig)

		for funcName, addr := range dev.Addresses {
			deviceToGAs[dev.DeviceID][funcName] = addr

			// Only index GAs that transmit (so we can map incoming telegrams to devices)
			for _, flag := range addr.Flags {
				if flag == "transmit" {
					gaToDevice[addr.GA] = GAMapping{
						DeviceID: dev.DeviceID,
						Function: funcName,
						DPT:      addr.DPT,
						Type:     dev.Type,
					}
					break
				}
			}
		}
	}

	return gaToDevice, deviceToGAs
}

// GAMapping holds the device mapping for a single group address.
type GAMapping struct {
	DeviceID string // Gray Logic device ID
	Function string // Function name (e.g., "switch", "brightness_status")
	DPT      string // Datapoint type
	Type     string // Device type
}

// HasFlag checks if an AddressConfig has a specific flag.
func (a AddressConfig) HasFlag(flag string) bool {
	for _, f := range a.Flags {
		if f == flag {
			return true
		}
	}
	return false
}
