package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	// Create a temporary config file
	content := `
site:
  id: "test-site"
database:
  path: "/tmp/test.db"
  wal_mode: true
  busy_timeout: 5
mqtt:
  broker:
    host: "localhost"
    port: 1883
    client_id: "test-client"
  qos: 1
api:
  host: "0.0.0.0"
  port: 8080
security:
  jwt:
    secret: "test-secret-key-at-least-32-chars!"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Site.ID != "test-site" {
		t.Errorf("Site.ID = %q, want %q", cfg.Site.ID, "test-site")
	}

	if cfg.Database.Path != "/tmp/test.db" {
		t.Errorf("Database.Path = %q, want %q", cfg.Database.Path, "/tmp/test.db")
	}

	if cfg.MQTT.Broker.Host != "localhost" {
		t.Errorf("MQTT.Broker.Host = %q, want %q", cfg.MQTT.Broker.Host, "localhost")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Load() expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("invalid: [yaml: content"), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() expected error for invalid YAML, got nil")
	}
}

func TestLoad_ValidationFailure(t *testing.T) {
	content := `
site:
  id: ""
database:
  path: "/tmp/test.db"
api:
  port: 8080
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() expected validation error for empty site.id, got nil")
	}
}

func TestConfig_Validate(t *testing.T) {
	// validJWTSecret is a secret that meets the 32-character minimum requirement
	validJWTSecret := "test-secret-key-at-least-32-chars!"

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Site: SiteConfig{ID: "site-001"},
				Database: DatabaseConfig{
					Path: "/data/graylogic.db",
				},
				MQTT: MQTTConfig{
					QoS: 1,
				},
				API: APIConfig{
					Port: 8080,
				},
				Security: SecurityConfig{
					JWT: JWTConfig{Secret: validJWTSecret},
				},
			},
			wantErr: false,
		},
		{
			name: "missing site ID",
			config: &Config{
				Site:     SiteConfig{ID: ""},
				Database: DatabaseConfig{Path: "/data/graylogic.db"},
				API:      APIConfig{Port: 8080},
				Security: SecurityConfig{JWT: JWTConfig{Secret: validJWTSecret}},
			},
			wantErr: true,
		},
		{
			name: "missing database path",
			config: &Config{
				Site:     SiteConfig{ID: "site-001"},
				Database: DatabaseConfig{Path: ""},
				API:      APIConfig{Port: 8080},
				Security: SecurityConfig{JWT: JWTConfig{Secret: validJWTSecret}},
			},
			wantErr: true,
		},
		{
			name: "invalid QoS",
			config: &Config{
				Site:     SiteConfig{ID: "site-001"},
				Database: DatabaseConfig{Path: "/data/graylogic.db"},
				MQTT:     MQTTConfig{QoS: 3},
				API:      APIConfig{Port: 8080},
				Security: SecurityConfig{JWT: JWTConfig{Secret: validJWTSecret}},
			},
			wantErr: true,
		},
		{
			name: "invalid port low",
			config: &Config{
				Site:     SiteConfig{ID: "site-001"},
				Database: DatabaseConfig{Path: "/data/graylogic.db"},
				MQTT:     MQTTConfig{QoS: 1},
				API:      APIConfig{Port: 0},
				Security: SecurityConfig{JWT: JWTConfig{Secret: validJWTSecret}},
			},
			wantErr: true,
		},
		{
			name: "invalid port high",
			config: &Config{
				Site:     SiteConfig{ID: "site-001"},
				Database: DatabaseConfig{Path: "/data/graylogic.db"},
				MQTT:     MQTTConfig{QoS: 1},
				API:      APIConfig{Port: 70000},
				Security: SecurityConfig{JWT: JWTConfig{Secret: validJWTSecret}},
			},
			wantErr: true,
		},
		{
			name: "missing JWT secret",
			config: &Config{
				Site:     SiteConfig{ID: "site-001"},
				Database: DatabaseConfig{Path: "/data/graylogic.db"},
				MQTT:     MQTTConfig{QoS: 1},
				API:      APIConfig{Port: 8080},
				Security: SecurityConfig{JWT: JWTConfig{Secret: ""}},
			},
			wantErr: true,
		},
		{
			name: "JWT secret too short",
			config: &Config{
				Site:     SiteConfig{ID: "site-001"},
				Database: DatabaseConfig{Path: "/data/graylogic.db"},
				MQTT:     MQTTConfig{QoS: 1},
				API:      APIConfig{Port: 8080},
				Security: SecurityConfig{JWT: JWTConfig{Secret: "short"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_GetTimeouts(t *testing.T) {
	cfg := &Config{
		API: APIConfig{
			Timeouts: APITimeoutConfig{
				Read:  30,
				Write: 45,
				Idle:  60,
			},
		},
	}

	if got := cfg.GetReadTimeout().Seconds(); got != 30 {
		t.Errorf("GetReadTimeout() = %v, want 30", got)
	}

	if got := cfg.GetWriteTimeout().Seconds(); got != 45 {
		t.Errorf("GetWriteTimeout() = %v, want 45", got)
	}

	if got := cfg.GetIdleTimeout().Seconds(); got != 60 {
		t.Errorf("GetIdleTimeout() = %v, want 60", got)
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	cfg := defaultConfig()

	// Set environment variables
	t.Setenv("GRAYLOGIC_DATABASE_PATH", "/custom/path.db")
	t.Setenv("GRAYLOGIC_MQTT_HOST", "mqtt.example.com")
	t.Setenv("GRAYLOGIC_MQTT_USERNAME", "testuser")
	t.Setenv("GRAYLOGIC_MQTT_PASSWORD", "testpass")
	t.Setenv("GRAYLOGIC_API_HOST", "192.168.1.1")
	t.Setenv("GRAYLOGIC_INFLUXDB_TOKEN", "secret-token")
	t.Setenv("GRAYLOGIC_JWT_SECRET", "jwt-secret")

	applyEnvOverrides(cfg)

	if cfg.Database.Path != "/custom/path.db" {
		t.Errorf("Database.Path = %q, want %q", cfg.Database.Path, "/custom/path.db")
	}

	if cfg.MQTT.Broker.Host != "mqtt.example.com" {
		t.Errorf("MQTT.Broker.Host = %q, want %q", cfg.MQTT.Broker.Host, "mqtt.example.com")
	}

	if cfg.MQTT.Auth.Username != "testuser" {
		t.Errorf("MQTT.Auth.Username = %q, want %q", cfg.MQTT.Auth.Username, "testuser")
	}

	if cfg.MQTT.Auth.Password != "testpass" {
		t.Errorf("MQTT.Auth.Password = %q, want %q", cfg.MQTT.Auth.Password, "testpass")
	}

	if cfg.API.Host != "192.168.1.1" {
		t.Errorf("API.Host = %q, want %q", cfg.API.Host, "192.168.1.1")
	}

	if cfg.InfluxDB.Token != "secret-token" {
		t.Errorf("InfluxDB.Token = %q, want %q", cfg.InfluxDB.Token, "secret-token")
	}

	if cfg.Security.JWT.Secret != "jwt-secret" {
		t.Errorf("Security.JWT.Secret = %q, want %q", cfg.Security.JWT.Secret, "jwt-secret")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.Site.ID == "" {
		t.Error("defaultConfig should have non-empty Site.ID")
	}

	if cfg.Database.Path == "" {
		t.Error("defaultConfig should have non-empty Database.Path")
	}

	if cfg.MQTT.Broker.Port != 1883 {
		t.Errorf("defaultConfig MQTT.Broker.Port = %d, want 1883", cfg.MQTT.Broker.Port)
	}

	if cfg.API.Port != 8080 {
		t.Errorf("defaultConfig API.Port = %d, want 8080", cfg.API.Port)
	}
}
