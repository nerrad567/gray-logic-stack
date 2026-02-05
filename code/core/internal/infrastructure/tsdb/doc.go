// Package tsdb provides time-series database connectivity for Gray Logic Core.
//
// It writes to VictoriaMetrics using InfluxDB line protocol over HTTP and
// queries using PromQL. Zero external dependencies — uses only net/http.
//
// # Purpose
//
// This package handles time-series data storage for:
//   - Device state telemetry (temperature, brightness, valve position, etc.)
//   - Predictive Health Monitoring (PHM) metrics
//   - Energy consumption tracking
//
// # Usage
//
//	cfg := config.TSDBConfig{
//	    Enabled:       true,
//	    URL:           "http://localhost:8428",
//	    BatchSize:     1000,
//	    FlushInterval: 1,
//	}
//
//	client, err := tsdb.Connect(ctx, cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Write device metrics
//	client.WriteDeviceMetric("light-living", "power_watts", 12.5)
//
// # Thread Safety
//
// All methods are safe for concurrent use from multiple goroutines.
// Writes are batched internally and flushed on size threshold or timer.
//
// # Error Handling
//
// Write operations are non-blocking and batch errors are reported via a callback.
// Connection and health check errors are returned directly.
//
// # Performance
//
// Writes are batched according to config.yaml settings (batch_size, flush_interval).
// Batch flush is a single HTTP POST with newline-delimited line protocol.
// VictoriaMetrics processes these with minimal overhead.
package tsdb
