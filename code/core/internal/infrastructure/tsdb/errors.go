package tsdb

import "errors"

// Sentinel errors for time-series database operations.
//
// These errors can be checked using errors.Is() for specific handling:
//
//	if errors.Is(err, tsdb.ErrNotConnected) {
//	    // Handle disconnected state
//	}
var (
	// ErrNotConnected indicates the client is not connected to the TSDB.
	ErrNotConnected = errors.New("tsdb: not connected")

	// ErrConnectionFailed indicates the initial connection attempt failed.
	ErrConnectionFailed = errors.New("tsdb: connection failed")

	// ErrWriteFailed indicates a write operation failed.
	ErrWriteFailed = errors.New("tsdb: write failed")

	// ErrDisabled indicates TSDB integration is disabled in config.
	ErrDisabled = errors.New("tsdb: disabled in configuration")
)
