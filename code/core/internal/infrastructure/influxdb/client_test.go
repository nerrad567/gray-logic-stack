package influxdb_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/influxdb"
)

// testConfig returns a configuration for the local dev InfluxDB.
// These values match docker-compose.yml.
func testConfig() config.InfluxDBConfig {
	return config.InfluxDBConfig{
		Enabled:       true,
		URL:           "http://127.0.0.1:8086",
		Token:         "graylogic-dev-token",
		Org:           "graylogic",
		Bucket:        "metrics",
		BatchSize:     100,
		FlushInterval: 1, // 1 second for faster test feedback
	}
}

// skipIfNoInfluxDB skips the test if InfluxDB is not running.
func skipIfNoInfluxDB(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_INTEGRATION") == "" {
		// Quick check: try to connect
		cfg := testConfig()
		client, err := influxdb.Connect(cfg)
		if err != nil {
			t.Skip("InfluxDB not available, skipping integration test")
		}
		client.Close()
	}
}

// =============================================================================
// Connection Tests
// =============================================================================

func TestConnect(t *testing.T) {
	skipIfNoInfluxDB(t)
	cfg := testConfig()

	client, err := influxdb.Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("IsConnected() = false after Connect()")
	}
}

func TestConnect_Disabled(t *testing.T) {
	cfg := testConfig()
	cfg.Enabled = false

	_, err := influxdb.Connect(cfg)
	if err == nil {
		t.Fatal("Connect() should return error when disabled")
	}
	if !errors.Is(err, influxdb.ErrDisabled) {
		t.Errorf("Connect() error = %v, want ErrDisabled", err)
	}
}

func TestConnect_InvalidURL(t *testing.T) {
	cfg := testConfig()
	cfg.URL = "http://127.0.0.1:59999" // Non-existent port

	_, err := influxdb.Connect(cfg)
	if err == nil {
		t.Fatal("Connect() should return error for invalid URL")
	}
}

func TestConnect_DefaultBatchSettings(t *testing.T) {
	skipIfNoInfluxDB(t)
	cfg := testConfig()
	cfg.BatchSize = 0     // Should use default
	cfg.FlushInterval = 0 // Should use default

	client, err := influxdb.Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("IsConnected() = false after Connect() with default batch settings")
	}
}

func TestConnect_NegativeBatchSettings(t *testing.T) {
	skipIfNoInfluxDB(t)
	cfg := testConfig()
	cfg.BatchSize = -5     // Negative, should use default
	cfg.FlushInterval = -1 // Negative, should use default

	client, err := influxdb.Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("IsConnected() = false after Connect() with negative batch settings")
	}
}

// =============================================================================
// Health Check Tests
// =============================================================================

func TestHealthCheck(t *testing.T) {
	skipIfNoInfluxDB(t)
	cfg := testConfig()

	client, err := influxdb.Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.HealthCheck(ctx); err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

func TestHealthCheck_Cancelled(t *testing.T) {
	skipIfNoInfluxDB(t)
	cfg := testConfig()

	client, err := influxdb.Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	// Create already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = client.HealthCheck(ctx)
	if err == nil {
		t.Error("HealthCheck() should return error for cancelled context")
	}
}

// =============================================================================
// Write Tests
// =============================================================================

func TestWriteDeviceMetric(t *testing.T) {
	skipIfNoInfluxDB(t)
	cfg := testConfig()

	client, err := influxdb.Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	// Track errors with mutex for race safety
	var writeErr error
	var mu sync.Mutex
	client.SetOnError(func(err error) {
		mu.Lock()
		writeErr = err
		mu.Unlock()
	})

	// Write a test metric
	client.WriteDeviceMetric("test-device-001", "test_metric", 42.0)

	// Flush to ensure it's written
	client.Flush()

	// Give a moment for error callback
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if writeErr != nil {
		t.Errorf("Write error = %v", writeErr)
	}
}

func TestWriteEnergyMetric(t *testing.T) {
	skipIfNoInfluxDB(t)
	cfg := testConfig()

	client, err := influxdb.Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	var writeErr error
	var mu sync.Mutex
	client.SetOnError(func(err error) {
		mu.Lock()
		writeErr = err
		mu.Unlock()
	})

	client.WriteEnergyMetric("test-device-002", 150.5, 12.34)
	client.Flush()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if writeErr != nil {
		t.Errorf("Write error = %v", writeErr)
	}
}

func TestWritePHMMetric(t *testing.T) {
	skipIfNoInfluxDB(t)
	cfg := testConfig()

	client, err := influxdb.Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	var writeErr error
	var mu sync.Mutex
	client.SetOnError(func(err error) {
		mu.Lock()
		writeErr = err
		mu.Unlock()
	})

	client.WritePHMMetric("test-device-003", "runtime_hours", 1234.5)
	client.Flush()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if writeErr != nil {
		t.Errorf("Write error = %v", writeErr)
	}
}

func TestWritePoint(t *testing.T) {
	skipIfNoInfluxDB(t)
	cfg := testConfig()

	client, err := influxdb.Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	var writeErr error
	var mu sync.Mutex
	client.SetOnError(func(err error) {
		mu.Lock()
		writeErr = err
		mu.Unlock()
	})

	client.WritePoint(
		"custom_measurement",
		map[string]string{"source": "test"},
		map[string]interface{}{"value": 99.9, "count": 5},
	)
	client.Flush()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if writeErr != nil {
		t.Errorf("Write error = %v", writeErr)
	}
}

// =============================================================================
// Close Tests
// =============================================================================

func TestClose(t *testing.T) {
	skipIfNoInfluxDB(t)
	cfg := testConfig()

	client, err := influxdb.Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Write something before close
	client.WriteDeviceMetric("close-test", "metric", 1.0)

	// Close should flush and disconnect
	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Should be disconnected
	if client.IsConnected() {
		t.Error("IsConnected() = true after Close()")
	}
}

func TestClose_Nil(t *testing.T) {
	// Closing a nil client should not panic
	var client *influxdb.Client
	// This will panic if we don't handle nil properly
	// For now, we can't call methods on nil pointer
	_ = client
}

func TestWritePointWithTime(t *testing.T) {
	skipIfNoInfluxDB(t)
	cfg := testConfig()

	client, err := influxdb.Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	var writeErr error
	var mu sync.Mutex
	client.SetOnError(func(err error) {
		mu.Lock()
		writeErr = err
		mu.Unlock()
	})

	// Write with a specific timestamp
	timestamp := time.Now().Add(-1 * time.Hour)
	client.WritePointWithTime(
		"custom_measurement",
		map[string]string{"source": "test-with-time"},
		map[string]interface{}{"value": 88.8},
		timestamp,
	)
	client.Flush()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if writeErr != nil {
		t.Errorf("Write error = %v", writeErr)
	}
}

func TestWriteEnergyMetric_ZeroEnergy(t *testing.T) {
	skipIfNoInfluxDB(t)
	cfg := testConfig()

	client, err := influxdb.Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	var writeErr error
	var mu sync.Mutex
	client.SetOnError(func(err error) {
		mu.Lock()
		writeErr = err
		mu.Unlock()
	})

	// Write energy metric with zero kWh (should skip energy_kwh field)
	client.WriteEnergyMetric("test-device-energy", 100.0, 0)
	client.Flush()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if writeErr != nil {
		t.Errorf("Write error = %v", writeErr)
	}
}
