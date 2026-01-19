// Package influxdb provides InfluxDB connectivity for Gray Logic Core.
//
// It wraps the official influxdb-client-go v2 library with Gray Logic-specific
// patterns for connection management, metric writing, and health monitoring.
//
// # Purpose
//
// This package handles time-series data storage for:
//   - Predictive Health Monitoring (PHM) metrics
//   - Energy consumption tracking
//   - Device telemetry and sensor data
//
// # Usage
//
//	cfg := config.InfluxDBConfig{
//	    URL:    "http://localhost:8086",
//	    Token:  "your-token",
//	    Org:    "graylogic",
//	    Bucket: "metrics",
//	}
//
//	client, err := influxdb.Connect(cfg)
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
// The underlying write API uses non-blocking batched writes.
//
// # Error Handling
//
// Write operations are non-blocking and batch errors are logged via a callback.
// Connection and health check errors are returned directly.
//
// # Performance
//
// Writes are batched according to config.yaml settings (batch_size, flush_interval).
// This reduces network overhead for high-frequency telemetry data.
package influxdb
