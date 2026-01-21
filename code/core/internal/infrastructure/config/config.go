package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure for Gray Logic Core.
// All configuration is loaded from YAML and can be overridden by environment variables.
type Config struct {
	Site      SiteConfig      `yaml:"site"`
	Database  DatabaseConfig  `yaml:"database"`
	MQTT      MQTTConfig      `yaml:"mqtt"`
	API       APIConfig       `yaml:"api"`
	WebSocket WebSocketConfig `yaml:"websocket"`
	InfluxDB  InfluxDBConfig  `yaml:"influxdb"`
	Logging   LoggingConfig   `yaml:"logging"`
	Protocols ProtocolsConfig `yaml:"protocols"`
	Security  SecurityConfig  `yaml:"security"`
}

// SiteConfig contains site-specific information.
type SiteConfig struct {
	ID       string         `yaml:"id"`
	Name     string         `yaml:"name"`
	Timezone string         `yaml:"timezone"`
	Location LocationConfig `yaml:"location"`
}

// LocationConfig contains geographic coordinates for astronomical calculations.
type LocationConfig struct {
	Latitude  float64 `yaml:"latitude"`
	Longitude float64 `yaml:"longitude"`
}

// DatabaseConfig contains SQLite database settings.
type DatabaseConfig struct {
	Path        string `yaml:"path"`
	WALMode     bool   `yaml:"wal_mode"`
	BusyTimeout int    `yaml:"busy_timeout"`
}

// MQTTConfig contains MQTT broker connection settings.
type MQTTConfig struct {
	Broker    MQTTBrokerConfig    `yaml:"broker"`
	Auth      MQTTAuthConfig      `yaml:"auth"`
	QoS       int                 `yaml:"qos"`
	Reconnect MQTTReconnectConfig `yaml:"reconnect"`
}

// MQTTBrokerConfig contains MQTT broker connection details.
type MQTTBrokerConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	TLS      bool   `yaml:"tls"`
	ClientID string `yaml:"client_id"`
}

// MQTTAuthConfig contains MQTT authentication credentials.
type MQTTAuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// MQTTReconnectConfig contains MQTT reconnection settings.
type MQTTReconnectConfig struct {
	InitialDelay int `yaml:"initial_delay"`
	MaxDelay     int `yaml:"max_delay"`
	MaxAttempts  int `yaml:"max_attempts"`
}

// APIConfig contains HTTP API server settings.
type APIConfig struct {
	Host     string           `yaml:"host"`
	Port     int              `yaml:"port"`
	TLS      TLSConfig        `yaml:"tls"`
	Timeouts APITimeoutConfig `yaml:"timeouts"`
	CORS     CORSConfig       `yaml:"cors"`
}

// TLSConfig contains TLS certificate settings.
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// APITimeoutConfig contains HTTP timeout settings.
type APITimeoutConfig struct {
	Read  int `yaml:"read"`
	Write int `yaml:"write"`
	Idle  int `yaml:"idle"`
}

// CORSConfig contains Cross-Origin Resource Sharing settings.
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
}

// WebSocketConfig contains WebSocket server settings.
type WebSocketConfig struct {
	Path           string `yaml:"path"`
	MaxMessageSize int    `yaml:"max_message_size"`
	PingInterval   int    `yaml:"ping_interval"`
	PongTimeout    int    `yaml:"pong_timeout"`
}

// InfluxDBConfig contains InfluxDB connection settings.
type InfluxDBConfig struct {
	Enabled       bool   `yaml:"enabled"`
	URL           string `yaml:"url"`
	Token         string `yaml:"token"`
	Org           string `yaml:"org"`
	Bucket        string `yaml:"bucket"`
	BatchSize     int    `yaml:"batch_size"`
	FlushInterval int    `yaml:"flush_interval"`
}

// LoggingConfig contains logging settings.
type LoggingConfig struct {
	Level  string            `yaml:"level"`
	Format string            `yaml:"format"`
	Output string            `yaml:"output"`
	File   FileLoggingConfig `yaml:"file"`
}

// FileLoggingConfig contains file-based logging settings.
type FileLoggingConfig struct {
	Path       string `yaml:"path"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

// ProtocolsConfig contains protocol bridge settings.
type ProtocolsConfig struct {
	KNX    KNXConfig    `yaml:"knx"`
	DALI   DALIConfig   `yaml:"dali"`
	Modbus ModbusConfig `yaml:"modbus"`
}

// KNXConfig contains KNX protocol bridge settings.
type KNXConfig struct {
	Enabled    bool   `yaml:"enabled"`
	ConfigFile string `yaml:"config_file"` // Path to KNX bridge config (devices, mappings)
	KNXDHost   string `yaml:"knxd_host"`
	KNXDPort   int    `yaml:"knxd_port"`
	// KNXD contains knxd daemon management settings
	KNXD KNXDConfig `yaml:"knxd"`
}

// KNXDConfig contains settings for managing the knxd daemon.
type KNXDConfig struct {
	// Managed indicates whether Gray Logic should manage knxd lifecycle.
	// If false, knxd is expected to be running externally (e.g., as a systemd service).
	Managed bool `yaml:"managed"`

	// Binary is the path to the knxd executable.
	// Default: "/usr/bin/knxd"
	Binary string `yaml:"binary"`

	// PhysicalAddress is knxd's own address on the KNX bus.
	// Format: "area.line.device" (e.g., "0.0.1")
	PhysicalAddress string `yaml:"physical_address"`

	// ClientAddresses is the range of addresses knxd can assign to clients.
	// Format: "area.line.device:count" (e.g., "0.0.2:8")
	ClientAddresses string `yaml:"client_addresses"`

	// Backend configures how knxd connects to the KNX bus.
	Backend KNXDBackendConfig `yaml:"backend"`

	// RestartOnFailure enables automatic restart if knxd crashes.
	// Default: true
	RestartOnFailure bool `yaml:"restart_on_failure"`

	// RestartDelaySeconds is the time to wait before restarting (in seconds).
	// Default: 5
	RestartDelaySeconds int `yaml:"restart_delay_seconds"`

	// MaxRestartAttempts limits restart attempts. 0 means unlimited.
	// Default: 10
	MaxRestartAttempts int `yaml:"max_restart_attempts"`

	// HealthCheckInterval is how often to run watchdog health checks.
	// Default: 30s
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`

	// HealthCheckDeviceAddress is an optional KNX group address to read during
	// health checks. This provides end-to-end verification that the entire
	// communication chain works (knxd → interface → bus → device).
	// Format: "main/middle/sub" (e.g., "1/7/0" for PSU status)
	// If empty, bus-level health checks are disabled.
	HealthCheckDeviceAddress string `yaml:"health_check_device_address,omitempty"`

	// HealthCheckDeviceTimeout is how long to wait for a response from the
	// health check device.
	// Default: 3s
	HealthCheckDeviceTimeout time.Duration `yaml:"health_check_device_timeout,omitempty"`

	// LogLevel sets knxd's verbosity (0-9).
	// Default: 0
	LogLevel int `yaml:"log_level"`
}

// KNXDBackendConfig configures how knxd connects to the KNX bus.
type KNXDBackendConfig struct {
	// Type is the backend connection type: "usb", "ipt", or "ip"
	Type string `yaml:"type"`

	// Host is the IP address for ipt (tunnelling) connections.
	Host string `yaml:"host,omitempty"`

	// Port is the port for ipt connections. Default: 3671
	Port int `yaml:"port,omitempty"`

	// MulticastAddress is the multicast group for ip (routing) connections.
	// Default: "224.0.23.12"
	MulticastAddress string `yaml:"multicast_address,omitempty"`
}

// DALIConfig contains DALI protocol bridge settings.
type DALIConfig struct {
	Enabled     bool   `yaml:"enabled"`
	GatewayType string `yaml:"gateway_type"`
	GatewayHost string `yaml:"gateway_host"`
	GatewayPort int    `yaml:"gateway_port"`
}

// ModbusConfig contains Modbus protocol bridge settings.
type ModbusConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Mode      string `yaml:"mode"`
	TCPHost   string `yaml:"tcp_host"`
	TCPPort   int    `yaml:"tcp_port"`
	RTUDevice string `yaml:"rtu_device"`
	RTUBaud   int    `yaml:"rtu_baud"`
}

// SecurityConfig contains security settings.
type SecurityConfig struct {
	JWT       JWTConfig       `yaml:"jwt"`
	APIKeys   APIKeyConfig    `yaml:"api_keys"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

// JWTConfig contains JWT token settings.
type JWTConfig struct {
	Secret          string `yaml:"secret"`
	AccessTokenTTL  int    `yaml:"access_token_ttl"`
	RefreshTokenTTL int    `yaml:"refresh_token_ttl"`
}

// APIKeyConfig contains API key settings.
type APIKeyConfig struct {
	Enabled bool `yaml:"enabled"`
}

// RateLimitConfig contains rate limiting settings.
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
}

// Load reads configuration from a YAML file and applies environment variable overrides.
//
// The configuration loading order is:
//  1. Default values (hardcoded)
//  2. YAML file values (override defaults)
//  3. Environment variables (override file values)
//
// Environment variables follow the pattern: GRAYLOGIC_SECTION_KEY
// For example: GRAYLOGIC_DATABASE_PATH, GRAYLOGIC_API_PORT
//
// Parameters:
//   - path: Path to the YAML configuration file
//
// Returns:
//   - *Config: Loaded and validated configuration
//   - error: If file cannot be read, parsed, or validation fails
func Load(path string) (*Config, error) {
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
		Site: SiteConfig{
			ID:       "site-001",
			Name:     "Gray Logic",
			Timezone: "UTC",
		},
		Database: DatabaseConfig{
			Path:        "./data/graylogic.db",
			WALMode:     true,
			BusyTimeout: 5,
		},
		MQTT: MQTTConfig{
			Broker: MQTTBrokerConfig{
				Host:     "localhost",
				Port:     1883,
				ClientID: "graylogic-core",
			},
			QoS: 1,
			Reconnect: MQTTReconnectConfig{
				InitialDelay: 1,
				MaxDelay:     60,
				MaxAttempts:  0,
			},
		},
		API: APIConfig{
			Host: "0.0.0.0",
			Port: 8080,
			Timeouts: APITimeoutConfig{
				Read:  30,
				Write: 30,
				Idle:  60,
			},
		},
		WebSocket: WebSocketConfig{
			Path:           "/ws",
			MaxMessageSize: 8192,
			PingInterval:   30,
			PongTimeout:    10,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Security: SecurityConfig{
			JWT: JWTConfig{
				AccessTokenTTL:  15,
				RefreshTokenTTL: 1440,
			},
			RateLimit: RateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 100,
			},
		},
	}
}

// applyEnvOverrides applies environment variable overrides to the configuration.
// Environment variables follow the pattern: GRAYLOGIC_SECTION_KEY
func applyEnvOverrides(cfg *Config) {
	// Database
	if v := os.Getenv("GRAYLOGIC_DATABASE_PATH"); v != "" {
		cfg.Database.Path = v
	}

	// MQTT
	if v := os.Getenv("GRAYLOGIC_MQTT_HOST"); v != "" {
		cfg.MQTT.Broker.Host = v
	}
	if v := os.Getenv("GRAYLOGIC_MQTT_USERNAME"); v != "" {
		cfg.MQTT.Auth.Username = v
	}
	if v := os.Getenv("GRAYLOGIC_MQTT_PASSWORD"); v != "" {
		cfg.MQTT.Auth.Password = v
	}

	// API
	if v := os.Getenv("GRAYLOGIC_API_HOST"); v != "" {
		cfg.API.Host = v
	}

	// InfluxDB
	if v := os.Getenv("GRAYLOGIC_INFLUXDB_TOKEN"); v != "" {
		cfg.InfluxDB.Token = v
	}

	// Security - JWT secret (IMPORTANT: always override in production)
	if v := os.Getenv("GRAYLOGIC_JWT_SECRET"); v != "" {
		cfg.Security.JWT.Secret = v
	}
}

// Validate checks the configuration for errors and security issues.
//
// Returns:
//   - error: Description of validation failure, or nil if valid
func (c *Config) Validate() error {
	var errs []string

	// Site validation
	if c.Site.ID == "" {
		errs = append(errs, "site.id is required")
	}

	// Database validation
	if c.Database.Path == "" {
		errs = append(errs, "database.path is required")
	}

	// MQTT validation
	if c.MQTT.QoS < 0 || c.MQTT.QoS > 2 {
		errs = append(errs, "mqtt.qos must be 0, 1, or 2")
	}

	// API validation
	if c.API.Port < 1 || c.API.Port > 65535 {
		errs = append(errs, "api.port must be between 1 and 65535")
	}

	// Security validation - JWT secret is REQUIRED
	// For building automation systems, authentication security is critical.
	// Empty or weak secrets could allow attackers to forge tokens and
	// gain unauthorised access to physical security devices.
	const minJWTSecretLength = 32
	if c.Security.JWT.Secret == "" {
		errs = append(errs, "security.jwt.secret is required (set GRAYLOGIC_JWT_SECRET environment variable)")
	} else if len(c.Security.JWT.Secret) < minJWTSecretLength {
		errs = append(errs, "security.jwt.secret must be at least 32 characters for adequate security")
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// GetReadTimeout returns the API read timeout as a Duration.
func (c *Config) GetReadTimeout() time.Duration {
	return time.Duration(c.API.Timeouts.Read) * time.Second
}

// GetWriteTimeout returns the API write timeout as a Duration.
func (c *Config) GetWriteTimeout() time.Duration {
	return time.Duration(c.API.Timeouts.Write) * time.Second
}

// GetIdleTimeout returns the API idle timeout as a Duration.
func (c *Config) GetIdleTimeout() time.Duration {
	return time.Duration(c.API.Timeouts.Idle) * time.Second
}
