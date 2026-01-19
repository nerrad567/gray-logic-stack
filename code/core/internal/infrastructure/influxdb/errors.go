package influxdb

import "errors"

// Sentinel errors for InfluxDB operations.
//
// These errors can be checked using errors.Is() for specific handling:
//
//	if errors.Is(err, influxdb.ErrNotConnected) {
//	    // Handle disconnected state
//	}
var (
	// ErrNotConnected indicates the client is not connected to InfluxDB.
	ErrNotConnected = errors.New("influxdb: not connected")

	// ErrConnectionFailed indicates the initial connection attempt failed.
	ErrConnectionFailed = errors.New("influxdb: connection failed")

	// ErrWriteFailed indicates a write operation failed.
	// Note: Most write errors are handled asynchronously via the error callback.
	ErrWriteFailed = errors.New("influxdb: write failed")

	// ErrDisabled indicates InfluxDB integration is disabled in config.
	ErrDisabled = errors.New("influxdb: disabled in configuration")
)
