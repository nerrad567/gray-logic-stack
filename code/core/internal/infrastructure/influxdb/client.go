package influxdb

import (
	"context"
	"fmt"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"

	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
)

// Default timeouts for InfluxDB operations.
const (
	defaultConnectTimeout = 10 * time.Second
	defaultPingTimeout    = 5 * time.Second

	// millisecondsPerSecond converts seconds to milliseconds for the InfluxDB API.
	millisecondsPerSecond = 1000
)

// Client wraps the InfluxDB v2 client with Gray Logic-specific functionality.
//
// It provides connection management, metric writing with batching,
// and health monitoring.
//
// Thread Safety:
//   - All methods are safe for concurrent use from multiple goroutines.
//   - Write operations are non-blocking and batched.
type Client struct {
	client   influxdb2.Client
	writeAPI api.WriteAPI
	cfg      config.InfluxDBConfig

	// connected tracks current connection state.
	connected bool
	mu        sync.RWMutex

	// onError is called when async write errors occur.
	onError func(err error)

	// done signals the error handler goroutine to stop.
	done chan struct{}
}

// Connect establishes a connection to the InfluxDB server.
//
// It performs the following setup:
//  1. Creates the client with token authentication
//  2. Verifies connectivity with a ping
//  3. Configures the non-blocking write API with batching
//  4. Sets up error callback for async write failures
//
// Parameters:
//   - ctx: Context for cancellation (used for ping verification)
//   - cfg: InfluxDB configuration from config.yaml
//
// Returns:
//   - *Client: Connected client ready for use
//   - error: If InfluxDB is disabled or connection fails
func Connect(ctx context.Context, cfg config.InfluxDBConfig) (*Client, error) {
	if !cfg.Enabled {
		return nil, ErrDisabled
	}

	// Validate and convert config values (ensure non-negative for uint conversion)
	// Upper bounds prevent integer overflow when multiplying flushInterval by 1000.
	const maxBatchSize = 100000
	const maxFlushIntervalSeconds = 3600 // 1 hour max

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 100 // Default
	} else if batchSize > maxBatchSize {
		return nil, fmt.Errorf("batch_size %d exceeds maximum %d", batchSize, maxBatchSize)
	}
	flushInterval := cfg.FlushInterval
	if flushInterval <= 0 {
		flushInterval = 10 // Default
	} else if flushInterval > maxFlushIntervalSeconds {
		return nil, fmt.Errorf("flush_interval %d exceeds maximum %d seconds", flushInterval, maxFlushIntervalSeconds)
	}

	// Create client with token auth
	// #nosec G115 -- values validated above to be positive
	client := influxdb2.NewClientWithOptions(
		cfg.URL,
		cfg.Token,
		influxdb2.DefaultOptions().
			SetBatchSize(uint(batchSize)).
			SetFlushInterval(uint(flushInterval)*millisecondsPerSecond), // Convert to milliseconds
	)

	// Verify connectivity with timeout
	// Always enforce a maximum timeout, even if caller provides a non-cancellable context.
	// This prevents indefinite hangs in a 20-year deployment system.
	pingCtx := ctx
	if pingCtx == nil {
		pingCtx = context.Background()
	}
	pingCtx, cancel := context.WithTimeout(pingCtx, defaultConnectTimeout)
	defer cancel()

	healthy, err := client.Ping(pingCtx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("%w: ping failed: %w", ErrConnectionFailed, err)
	}
	if !healthy {
		client.Close()
		return nil, fmt.Errorf("%w: server not healthy", ErrConnectionFailed)
	}

	// Create non-blocking write API
	writeAPI := client.WriteAPI(cfg.Org, cfg.Bucket)

	c := &Client{
		client:    client,
		writeAPI:  writeAPI,
		cfg:       cfg,
		connected: true,
		done:      make(chan struct{}),
	}

	// Set up error callback for async write failures
	errorsCh := writeAPI.Errors()
	go c.handleWriteErrors(errorsCh)

	return c, nil
}

// handleWriteErrors processes async write errors from the WriteAPI.
// Exits when the done channel is closed or the error channel is closed.
func (c *Client) handleWriteErrors(errorsCh <-chan error) {
	for {
		select {
		case <-c.done:
			return
		case err, ok := <-errorsCh:
			if !ok {
				return
			}
			c.mu.RLock()
			callback := c.onError
			c.mu.RUnlock()

			if callback != nil {
				callback(err)
			}
		}
	}
}

// Close gracefully shuts down the InfluxDB connection.
//
// It performs:
//  1. Marks client as disconnected
//  2. Flushes any pending writes (while error handler is still running)
//  3. Signals the error handler goroutine to stop
//  4. Closes the underlying client
//
// The flush happens BEFORE signalling the goroutine to stop, ensuring
// any errors during the final flush can still be processed by the
// error handler callback.
//
// Returns:
//   - error: nil (InfluxDB client Close doesn't return errors)
func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}

	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()

	// Flush pending writes FIRST (while error handler goroutine is still running)
	// This ensures any errors during flush are still delivered to the callback
	c.writeAPI.Flush()

	// THEN signal goroutine to stop
	if c.done != nil {
		close(c.done)
	}

	// Close the client
	c.client.Close()

	return nil
}

// HealthCheck verifies the InfluxDB connection is alive and functioning.
//
// Parameters:
//   - ctx: Context for timeout/cancellation
//
// Returns:
//   - error: nil if healthy, error describing the issue otherwise
func (c *Client) HealthCheck(ctx context.Context) error {
	if !c.IsConnected() {
		return ErrNotConnected
	}

	// Create a timeout context if none provided
	checkCtx, cancel := context.WithTimeout(ctx, defaultPingTimeout)
	defer cancel()

	healthy, err := c.client.Ping(checkCtx)
	if err != nil {
		return fmt.Errorf("influxdb health check failed: %w", err)
	}
	if !healthy {
		return fmt.Errorf("influxdb health check failed: server not healthy")
	}

	return nil
}

// IsConnected returns the current connection state.
//
// Note: This reflects the last known state. For reliability,
// use HealthCheck which performs an active ping.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// SetOnError sets a callback to be invoked when async write errors occur.
//
// Since writes are non-blocking, errors are delivered asynchronously.
// Use this callback to log or handle write failures.
func (c *Client) SetOnError(callback func(err error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onError = callback
}

// Flush forces all pending writes to be sent to InfluxDB.
//
// This blocks until all buffered points are written.
// Useful for testing or before graceful shutdown.
// Safe to call after Close() (no-op).
func (c *Client) Flush() {
	if c.writeAPI == nil {
		return
	}

	c.mu.RLock()
	connected := c.connected
	c.mu.RUnlock()

	if !connected {
		return
	}

	c.writeAPI.Flush()
}
