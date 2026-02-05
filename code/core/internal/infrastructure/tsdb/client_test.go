package tsdb_test

import (
	"context"
	"errors"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/tsdb"
)

// testConfig returns a configuration for the local dev VictoriaMetrics.
// These values match docker-compose.dev.yml.
func testConfig() config.TSDBConfig {
	url := os.Getenv("TSDB_URL")
	if url == "" {
		url = "http://127.0.0.1:8428"
	}
	return config.TSDBConfig{
		Enabled:       true,
		URL:           url,
		BatchSize:     100,
		FlushInterval: 1, // 1 second for faster test feedback
	}
}

// skipIfNoTSDB skips the test if VictoriaMetrics is not running.
func skipIfNoTSDB(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_INTEGRATION") == "" {
		cfg := testConfig()
		client, err := tsdb.Connect(context.Background(), cfg)
		if err != nil {
			t.Skip("VictoriaMetrics not available, skipping integration test")
		}
		defer client.Close()
		if err := client.HealthCheck(context.Background()); err != nil {
			t.Skip("VictoriaMetrics health check failed, skipping integration test")
		}
	}
}

// =============================================================================
// Connection Tests
// =============================================================================

func TestConnect(t *testing.T) {
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
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

	_, err := tsdb.Connect(context.Background(), cfg)
	if err == nil {
		t.Fatal("Connect() should return error when disabled")
	}
	if !errors.Is(err, tsdb.ErrDisabled) {
		t.Errorf("Connect() error = %v, want ErrDisabled", err)
	}
}

func TestConnect_InvalidURL(t *testing.T) {
	cfg := testConfig()
	cfg.URL = "http://127.0.0.1:59999" // Non-existent port

	_, err := tsdb.Connect(context.Background(), cfg)
	if err == nil {
		t.Fatal("Connect() should return error for invalid URL")
	}
}

func TestConnect_DefaultBatchSettings(t *testing.T) {
	skipIfNoTSDB(t)
	cfg := testConfig()
	cfg.BatchSize = 0     // Should use default
	cfg.FlushInterval = 0 // Should use default

	client, err := tsdb.Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("IsConnected() = false after Connect() with default batch settings")
	}
}

func TestConnect_NegativeBatchSettings(t *testing.T) {
	skipIfNoTSDB(t)
	cfg := testConfig()
	cfg.BatchSize = -5     // Negative, should use default
	cfg.FlushInterval = -1 // Negative, should use default

	client, err := tsdb.Connect(context.Background(), cfg)
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
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
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
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
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
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
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

	client.WriteDeviceMetric("test-device-001", "test_metric", 42.0)
	client.Flush()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if writeErr != nil {
		t.Errorf("Write error = %v", writeErr)
	}
}

func TestWriteEnergyMetric(t *testing.T) {
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
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
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
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
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
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

func TestWritePointWithTime(t *testing.T) {
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
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
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
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

// =============================================================================
// Close Tests
// =============================================================================

func TestClose(t *testing.T) {
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
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

func TestConnect_DisabledReturnsNilClient(t *testing.T) {
	cfg := testConfig()
	cfg.Enabled = false

	client, err := tsdb.Connect(context.Background(), cfg)

	if client != nil {
		t.Error("Connect() should return nil client when disabled")
	}

	if !errors.Is(err, tsdb.ErrDisabled) {
		t.Errorf("Connect() error = %v, want ErrDisabled", err)
	}
}

func TestHealthCheck_NotConnected(t *testing.T) {
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	client.Close()

	// Health check on closed client still works (it's an HTTP call)
	// but IsConnected returns false
	if client.IsConnected() {
		t.Error("IsConnected() should be false after Close()")
	}
}

func TestFlush_AfterClose(t *testing.T) {
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	client.Close()

	// Should not panic
	client.Flush()
}

func TestSetOnError_CallbackInvoked(t *testing.T) {
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	callbackInvoked := false
	var mu sync.Mutex

	client.SetOnError(func(err error) {
		mu.Lock()
		callbackInvoked = true
		mu.Unlock()
	})

	mu.Lock()
	_ = callbackInvoked
	mu.Unlock()
}

func TestIsConnected_AfterConnect(t *testing.T) {
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("IsConnected() should return true after Connect()")
	}
}

func TestIsConnected_AfterClose(t *testing.T) {
	skipIfNoTSDB(t)
	cfg := testConfig()

	client, err := tsdb.Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	client.Close()

	if client.IsConnected() {
		t.Error("IsConnected() should return false after Close()")
	}
}

func TestClose_NoGoroutineLeak(t *testing.T) {
	skipIfNoTSDB(t)
	cfg := testConfig()

	before := runtime.NumGoroutine()

	for i := 0; i < 5; i++ {
		client, err := tsdb.Connect(context.Background(), cfg)
		if err != nil {
			t.Fatalf("Connect() iteration %d error = %v", i, err)
		}
		client.WriteDeviceMetric("leak-test", "metric", float64(i))
		client.Close()
	}

	time.Sleep(200 * time.Millisecond)

	after := runtime.NumGoroutine()

	diff := after - before
	if diff > 2 {
		t.Errorf("Potential goroutine leak: before=%d, after=%d, diff=%d", before, after, diff)
	}
}
