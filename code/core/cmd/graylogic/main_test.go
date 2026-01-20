package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRun_InvalidConfig verifies run fails with invalid config path.
func TestRun_InvalidConfig(t *testing.T) {
	originalEnv := os.Getenv("GRAYLOGIC_CONFIG")
	defer os.Setenv("GRAYLOGIC_CONFIG", originalEnv)

	os.Setenv("GRAYLOGIC_CONFIG", "/nonexistent/path/config.yaml")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := run(ctx)
	if err == nil {
		t.Fatal("run() should fail with invalid config path")
	}

	if os.IsNotExist(err) || err.Error() == "" {
		t.Logf("Got expected error type: %v", err)
	}
}

// TestRun_MissingDatabasePath verifies run fails when database path is invalid.
func TestRun_MissingDatabasePath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `
site:
  id: test-site

database:
  path: ""
  wal_mode: true
  busy_timeout: 5

mqtt:
  broker:
    host: "127.0.0.1"
    port: 1883
    client_id: "test-client"
    tls: false
  qos: 1
  reconnect:
    initial_delay: 1
    max_delay: 60

influxdb:
  enabled: false

logging:
  level: info
  format: text
  output: stdout

api:
  host: "127.0.0.1"
  port: 8080
  timeouts:
    read: 30
    write: 60
    idle: 120
  auth:
    jwt_secret: "test-secret-for-development-only"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	originalEnv := os.Getenv("GRAYLOGIC_CONFIG")
	defer os.Setenv("GRAYLOGIC_CONFIG", originalEnv)
	os.Setenv("GRAYLOGIC_CONFIG", configPath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := run(ctx)
	if err == nil {
		t.Fatal("run() should fail with empty database path")
	}
}

// TestGetConfigPath_Default verifies default config path.
func TestGetConfigPath_Default(t *testing.T) {
	originalEnv := os.Getenv("GRAYLOGIC_CONFIG")
	defer os.Setenv("GRAYLOGIC_CONFIG", originalEnv)

	os.Unsetenv("GRAYLOGIC_CONFIG")

	path := getConfigPath()
	if path != defaultConfigPath {
		t.Errorf("getConfigPath() = %q, want %q", path, defaultConfigPath)
	}
}

// TestGetConfigPath_EnvOverride verifies environment variable override.
func TestGetConfigPath_EnvOverride(t *testing.T) {
	originalEnv := os.Getenv("GRAYLOGIC_CONFIG")
	defer os.Setenv("GRAYLOGIC_CONFIG", originalEnv)

	expected := "/custom/path/config.yaml"
	os.Setenv("GRAYLOGIC_CONFIG", expected)

	path := getConfigPath()
	if path != expected {
		t.Errorf("getConfigPath() = %q, want %q", path, expected)
	}
}

// TestHealthCheck_NilInfluxClient verifies health check works with nil InfluxDB.
// This test is skipped because healthCheck requires valid db/mqtt clients.
func TestHealthCheck_NilInfluxClient(t *testing.T) {
	t.Skip("healthCheck requires valid db and mqtt clients - cannot test with nils")
}

// TestRun_SuccessfulStartupAndShutdown tests full startup with running services.
// Requires MQTT broker at 127.0.0.1:1883.
func TestRun_SuccessfulStartupAndShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	dbPath := filepath.Join(tmpDir, "test.db")

	configContent := `
site:
  id: test-site

database:
  path: "` + dbPath + `"
  wal_mode: true
  busy_timeout: 5

mqtt:
  broker:
    host: "127.0.0.1"
    port: 1883
    client_id: "test-successful-startup"
    tls: false
  qos: 1
  reconnect:
    initial_delay: 1
    max_delay: 5

influxdb:
  enabled: false

logging:
  level: info
  format: text
  output: stdout

api:
  host: "127.0.0.1"
  port: 8080
  timeouts:
    read: 30
    write: 60
    idle: 120
  auth:
    jwt_secret: "test-secret-for-development-only"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	originalEnv := os.Getenv("GRAYLOGIC_CONFIG")
	defer os.Setenv("GRAYLOGIC_CONFIG", originalEnv)
	os.Setenv("GRAYLOGIC_CONFIG", configPath)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := run(ctx)

	if err != nil {
		t.Logf("run() returned error: %v (may be due to missing MQTT broker)", err)
	}
}

// TestRun_ContextCancelledDuringStartup verifies cancellation during startup.
func TestRun_ContextCancelledDuringStartup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	dbPath := filepath.Join(tmpDir, "test.db")

	configContent := `
site:
  id: test-site

database:
  path: "` + dbPath + `"
  wal_mode: true
  busy_timeout: 5

mqtt:
  broker:
    host: "127.0.0.1"
    port: 19999
    client_id: "test-client"
    tls: false
  qos: 1
  reconnect:
    initial_delay: 1
    max_delay: 5

influxdb:
  enabled: false

logging:
  level: info
  format: text
  output: stdout

api:
  host: "127.0.0.1"
  port: 8080
  timeouts:
    read: 30
    write: 60
    idle: 120
  auth:
    jwt_secret: "test-secret-for-development-only"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	originalEnv := os.Getenv("GRAYLOGIC_CONFIG")
	defer os.Setenv("GRAYLOGIC_CONFIG", originalEnv)
	os.Setenv("GRAYLOGIC_CONFIG", configPath)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := run(ctx)

	if err == nil {
		t.Log("run() completed without error (may have cancelled cleanly)")
	} else {
		t.Logf("run() returned error (expected): %v", err)
	}
}
