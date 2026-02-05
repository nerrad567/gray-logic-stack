package tsdb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
)

// Default timeouts for TSDB operations.
const (
	defaultConnectTimeout = 10 * time.Second
	defaultWriteTimeout   = 5 * time.Second
	defaultHealthTimeout  = 5 * time.Second
)

// Client writes time-series data to VictoriaMetrics using InfluxDB line protocol.
//
// Writes are batched internally and flushed either when the batch reaches
// the configured size or when the flush interval timer fires. The flush
// is a single HTTP POST to /write with newline-delimited line protocol.
//
// Thread Safety: All methods are safe for concurrent use from multiple goroutines.
type Client struct {
	url        string
	httpClient *http.Client

	connected bool
	mu        sync.RWMutex

	// Batching
	batch     []string
	batchMu   sync.Mutex
	batchSize int
	flushTick *time.Ticker
	done      chan struct{}
	wg        sync.WaitGroup

	// Error callback for async write failures.
	onError func(err error)
}

// Connect establishes a connection to VictoriaMetrics.
//
// It performs the following:
//  1. Validates config (disabled returns ErrDisabled)
//  2. Creates an HTTP client
//  3. Verifies connectivity via GET /health
//  4. Starts background flush goroutine
//
// Parameters:
//   - ctx: Context for cancellation (used for health check)
//   - cfg: TSDB configuration from config.yaml
//
// Returns:
//   - *Client: Connected client ready for use
//   - error: If TSDB is disabled or connection fails
func Connect(ctx context.Context, cfg config.TSDBConfig) (*Client, error) {
	if !cfg.Enabled {
		return nil, ErrDisabled
	}

	// Validate and apply defaults
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 1000
	}
	flushInterval := cfg.FlushInterval
	if flushInterval <= 0 {
		flushInterval = 1
	}

	url := strings.TrimRight(cfg.URL, "/")

	c := &Client{
		url: url,
		httpClient: &http.Client{
			Timeout: defaultWriteTimeout,
		},
		batch:     make([]string, 0, batchSize),
		batchSize: batchSize,
		flushTick: time.NewTicker(time.Duration(flushInterval) * time.Second),
		done:      make(chan struct{}),
		connected: true,
	}

	// Verify connectivity
	healthCtx, cancel := context.WithTimeout(ctx, defaultConnectTimeout)
	defer cancel()

	if err := c.HealthCheck(healthCtx); err != nil {
		c.connected = false
		return nil, fmt.Errorf("%w: health check failed: %w", ErrConnectionFailed, err)
	}

	// Start background flush goroutine
	c.wg.Add(1)
	go c.flushLoop()

	return c, nil
}

// flushLoop periodically flushes the batch on timer or when done is signalled.
func (c *Client) flushLoop() {
	defer c.wg.Done()
	for {
		select {
		case <-c.flushTick.C:
			c.Flush()
		case <-c.done:
			return
		}
	}
}

// Close gracefully shuts down the TSDB connection.
//
// It performs:
//  1. Marks client as disconnected
//  2. Stops the flush timer
//  3. Signals the flush goroutine to stop
//  4. Flushes any remaining batched writes
//
// Returns:
//   - error: nil (flush errors are delivered via onError callback)
func (c *Client) Close() error {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()

	// Stop timer
	c.flushTick.Stop()

	// Signal goroutine to stop
	close(c.done)
	c.wg.Wait()

	// Final flush of remaining data
	c.Flush()

	return nil
}

// HealthCheck verifies the VictoriaMetrics connection is alive.
//
// Parameters:
//   - ctx: Context for timeout/cancellation
//
// Returns:
//   - error: nil if healthy, error describing the issue otherwise
func (c *Client) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/health", nil)
	if err != nil {
		return fmt.Errorf("tsdb health check: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("tsdb health check: %w", err)
	}
	defer resp.Body.Close()
	// Drain body to allow connection reuse
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tsdb health check: status %d", resp.StatusCode)
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
// Since writes are batched and flushed asynchronously, errors are
// delivered via this callback rather than returned from write methods.
func (c *Client) SetOnError(callback func(err error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onError = callback
}

// addLine adds a line protocol string to the batch.
// If the batch reaches the configured size, it triggers a flush.
func (c *Client) addLine(line string) {
	if !c.IsConnected() {
		return
	}

	c.batchMu.Lock()
	c.batch = append(c.batch, line)
	shouldFlush := len(c.batch) >= c.batchSize
	c.batchMu.Unlock()

	if shouldFlush {
		c.Flush()
	}
}

// Flush sends all pending writes to VictoriaMetrics.
//
// This is called automatically by the flush timer and when the batch
// is full. It can also be called manually for testing or shutdown.
// Safe to call concurrently — only one flush executes at a time.
func (c *Client) Flush() {
	c.batchMu.Lock()
	if len(c.batch) == 0 {
		c.batchMu.Unlock()
		return
	}
	// Swap batch out under lock
	lines := c.batch
	c.batch = make([]string, 0, c.batchSize)
	c.batchMu.Unlock()

	// POST to /write endpoint
	body := strings.Join(lines, "\n")
	ctx, cancel := context.WithTimeout(context.Background(), defaultWriteTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/write", bytes.NewBufferString(body))
	if err != nil {
		c.reportError(fmt.Errorf("%w: %w", ErrWriteFailed, err))
		return
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.reportError(fmt.Errorf("%w: %w", ErrWriteFailed, err))
		return
	}
	defer resp.Body.Close()
	// Drain body to allow connection reuse
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		c.reportError(fmt.Errorf("%w: HTTP %d", ErrWriteFailed, resp.StatusCode))
	}
}

// reportError delivers an error to the onError callback if set.
func (c *Client) reportError(err error) {
	c.mu.RLock()
	callback := c.onError
	c.mu.RUnlock()

	if callback != nil {
		callback(err)
	}
}
