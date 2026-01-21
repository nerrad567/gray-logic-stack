package knx

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// closeOnce wraps a channel with sync.Once to prevent double-close panics.
type closeOnce struct {
	ch   chan struct{}
	once sync.Once
}

func newCloseOnce() *closeOnce {
	return &closeOnce{ch: make(chan struct{})}
}

func (c *closeOnce) Close() {
	c.once.Do(func() { close(c.ch) })
}

func (c *closeOnce) Done() <-chan struct{} {
	return c.ch
}

// Default timeouts and intervals for knxd communication.
const (
	// defaultConnectTimeout is the maximum time to wait for initial connection.
	defaultConnectTimeout = 10 * time.Second

	// defaultReadTimeout is the timeout for individual read operations.
	defaultReadTimeout = 30 * time.Second

	// defaultWriteTimeout is the timeout for write operations.
	defaultWriteTimeout = 5 * time.Second

	// defaultReconnectInterval is the initial delay between reconnection attempts.
	defaultReconnectInterval = 5 * time.Second

	// readBufferSize is the size of the read buffer for incoming messages.
	readBufferSize = 256

	// callbackQueueSize is the buffer size for the telegram callback queue.
	callbackQueueSize = 100

	// callbackWorkerCount is the number of concurrent callback workers.
	callbackWorkerCount = 4
)

// KNXDConfig holds knxd connection configuration.
//
//nolint:revive // KNXDConfig is clearer than DConfig for external use
type KNXDConfig struct {
	// Connection is the knxd connection URL.
	// Supported formats:
	//   - "unix:///run/knxd" (Unix socket)
	//   - "tcp://localhost:6720" (TCP)
	Connection string

	// ConnectTimeout is the maximum time to wait for connection.
	// Default: 10 seconds.
	ConnectTimeout time.Duration

	// ReadTimeout is the timeout for read operations.
	// Default: 30 seconds.
	ReadTimeout time.Duration

	// ReconnectInterval is the initial delay between reconnection attempts.
	// Default: 5 seconds.
	ReconnectInterval time.Duration
}

// KNXDStats holds operational statistics.
//
//nolint:revive // KNXDStats is clearer than DStats for external use
type KNXDStats struct {
	TelegramsTx      uint64
	TelegramsRx      uint64
	TelegramsDropped uint64 // Telegrams dropped due to full callback queue
	ErrorsTotal      uint64
	LastActivity     time.Time
	Connected        bool
}

// Logger interface for optional logging.
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
}

// Connector interface for testability.
// This allows mocking the knxd client in tests.
type Connector interface {
	Send(ctx context.Context, ga GroupAddress, data []byte) error
	SendRead(ctx context.Context, ga GroupAddress) error
	SetOnTelegram(callback func(Telegram))
	IsConnected() bool
	Stats() KNXDStats
	Close() error
}

// Ensure KNXDClient implements Connector.
var _ Connector = (*KNXDClient)(nil)

// KNXDClient provides connection to the knxd daemon.
//
// Thread Safety:
//   - All methods are safe for concurrent use.
//   - Telegram callbacks are invoked in a dedicated goroutine.
//
//nolint:revive // KNXDClient is clearer than DClient for external use
type KNXDClient struct {
	cfg  KNXDConfig
	conn net.Conn

	// Connection state
	connMu    sync.RWMutex
	connected bool

	// Telegram handler callback
	onTelegram func(Telegram)
	callbackMu sync.RWMutex

	// Callback worker pool (bounded goroutine spawning)
	callbackQueue chan Telegram

	// Shutdown coordination (closeOnce prevents double-close panics)
	done *closeOnce
	wg   sync.WaitGroup

	// Logger (optional)
	logger   Logger
	loggerMu sync.RWMutex

	// Statistics (atomic for performance)
	telegramsTx      atomic.Uint64
	telegramsRx      atomic.Uint64
	telegramsDropped atomic.Uint64 // Telegrams dropped due to full queue
	errorsTotal      atomic.Uint64
	lastActivity     atomic.Int64 // Unix timestamp
}

// Connect establishes connection to knxd daemon.
//
// The connection URL determines the transport:
//   - "unix:///run/knxd" → Unix socket
//   - "tcp://localhost:6720" → TCP socket
//
// After connecting, it opens group communication mode and starts
// a goroutine to receive incoming telegrams.
//
// Parameters:
//   - ctx: Context for cancellation (used for initial connection)
//   - cfg: Connection configuration
//
// Returns:
//   - *KNXDClient: Connected client ready for use
//   - error: If connection or handshake fails
func Connect(ctx context.Context, cfg KNXDConfig) (*KNXDClient, error) {
	// Apply defaults
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = defaultConnectTimeout
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = defaultReadTimeout
	}
	if cfg.ReconnectInterval == 0 {
		cfg.ReconnectInterval = defaultReconnectInterval
	}

	// Parse connection URL
	network, address, err := parseConnectionURL(cfg.Connection)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConnectionFailed, err)
	}

	// Create connection with timeout
	connectCtx := ctx
	if connectCtx == nil {
		connectCtx = context.Background()
	}
	connectCtx, cancel := context.WithTimeout(connectCtx, cfg.ConnectTimeout)
	defer cancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(connectCtx, network, address)
	if err != nil {
		return nil, fmt.Errorf("%w: dial failed: %w", ErrConnectionFailed, err)
	}

	client := &KNXDClient{
		cfg:           cfg,
		conn:          conn,
		done:          newCloseOnce(),
		callbackQueue: make(chan Telegram, callbackQueueSize),
	}
	client.lastActivity.Store(time.Now().Unix())

	// Open group communication mode (respects context deadline)
	if err := client.openGroupCon(connectCtx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("%w: handshake failed: %w", ErrConnectionFailed, err)
	}

	// Mark as connected
	client.connMu.Lock()
	client.connected = true
	client.connMu.Unlock()

	// Start callback worker pool (bounded goroutine count)
	for range callbackWorkerCount {
		client.wg.Add(1)
		go client.callbackWorker()
	}

	// Start receive loop
	client.wg.Add(1)
	go client.receiveLoop()

	return client, nil
}

// parseConnectionURL parses a knxd connection URL into network and address.
func parseConnectionURL(connURL string) (network, address string, err error) {
	u, err := url.Parse(connURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	switch u.Scheme {
	case "unix":
		return "unix", u.Path, nil
	case "tcp":
		host := u.Host
		if host == "" {
			host = "localhost:6720"
		}
		return "tcp", host, nil
	default:
		return "", "", fmt.Errorf("unsupported scheme %q (use unix or tcp)", u.Scheme)
	}
}

// openGroupCon sends the EIB_OPEN_T_GROUP message to knxd.
// It respects the context deadline to ensure the overall connect timeout is honoured.
//
// The EIB_OPEN_T_GROUP (0x0022) message format:
//   - type: 0x0022
//   - group_addr: 2 bytes (0x0000 for all groups)
//   - flags: 1 byte (0xFF for read/write mode)
func (c *KNXDClient) openGroupCon(ctx context.Context) error {
	// EIB_OPEN_T_GROUP payload: group_addr(2) + flags(1)
	// Use 0x0000 for all groups, 0xFF for read/write mode
	payload := []byte{0x00, 0x00, 0xFF}
	msg := EncodeKNXDMessage(EIBOpenTGroup, payload)

	// Calculate deadline: use context deadline if set and sooner than default
	writeDeadline := time.Now().Add(defaultWriteTimeout)
	if deadline, ok := ctx.Deadline(); ok && deadline.Before(writeDeadline) {
		writeDeadline = deadline
	}

	if err := c.conn.SetWriteDeadline(writeDeadline); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}

	// Check context before write
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	if _, err := c.conn.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	// Read response - respect context deadline
	readDeadline := time.Now().Add(c.cfg.ReadTimeout)
	if deadline, ok := ctx.Deadline(); ok && deadline.Before(readDeadline) {
		readDeadline = deadline
	}

	if err := c.conn.SetReadDeadline(readDeadline); err != nil {
		return fmt.Errorf("set read deadline: %w", err)
	}

	// Check context before read
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	// Read response using proper message framing
	// First read 2-byte size field
	sizeBytes := make([]byte, 2)
	if _, err := io.ReadFull(c.conn, sizeBytes); err != nil {
		return fmt.Errorf("read response size: %w", err)
	}

	// Parse size (size = type(2) + payload, does NOT include size field)
	msgSize := binary.BigEndian.Uint16(sizeBytes)
	if msgSize < 2 {
		return fmt.Errorf("invalid response size: %d", msgSize)
	}

	// Read remaining bytes (type + payload)
	resp := make([]byte, 2+int(msgSize))
	copy(resp[:2], sizeBytes)
	if _, err := io.ReadFull(c.conn, resp[2:]); err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	msgType, _, err := ParseKNXDMessage(resp)
	if err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if msgType != EIBOpenTGroup {
		return fmt.Errorf("unexpected response type: 0x%04X", msgType)
	}

	return nil
}

// receiveLoop continuously reads telegrams from knxd.
func (c *KNXDClient) receiveLoop() {
	defer c.wg.Done()

	buf := make([]byte, readBufferSize)

	for {
		select {
		case <-c.done.Done():
			return
		default:
		}

		msgType, payload, err := c.readMessage(buf)
		if err != nil {
			if c.handleReadError(err) {
				return // Fatal error, stop loop
			}
			continue // Recoverable error, retry
		}

		// Handle group packet
		if msgType == EIBGroupPacket && len(payload) >= 2 {
			c.handleGroupPacket(payload)
		}
	}
}

// readMessage reads a single knxd message from the connection.
// Returns the message type, payload, and any error.
// If the message is oversized, returns ErrProtocolDesync which is fatal.
func (c *KNXDClient) readMessage(buf []byte) (uint16, []byte, error) {
	// Set read deadline
	if err := c.conn.SetReadDeadline(time.Now().Add(c.cfg.ReadTimeout)); err != nil {
		c.logError("set read deadline failed", err)
		return 0, nil, fmt.Errorf("set deadline: %w", err)
	}

	// Read message size (2 bytes)
	if _, err := io.ReadFull(c.conn, buf[:2]); err != nil {
		return 0, nil, fmt.Errorf("read size: %w", err)
	}

	// Parse message size (size field = type(2) + payload, NOT including size field itself)
	msgSize := binary.BigEndian.Uint16(buf[:2])
	if msgSize < 2 {
		c.errorsTotal.Add(1)
		return 0, nil, fmt.Errorf("invalid message size: %d (minimum 2 for type field)",
			msgSize)
	}

	// Total message length = size field(2) + msgSize (type + payload)
	totalLen := 2 + int(msgSize)

	// Oversized message detection - FATAL error to prevent protocol desync.
	// We cannot safely skip the message because we'd need to read and discard
	// an unknown number of bytes, risking buffer overflow or incorrect framing.
	// Closing the connection forces a clean reconnect.
	if totalLen > len(buf) {
		c.errorsTotal.Add(1)
		c.logError("oversized message, closing connection to prevent desync",
			fmt.Errorf("size %d exceeds buffer %d", totalLen, len(buf)))
		return 0, nil, ErrProtocolDesync
	}

	// Read rest of message (type + payload = msgSize bytes)
	if _, err := io.ReadFull(c.conn, buf[2:totalLen]); err != nil {
		return 0, nil, fmt.Errorf("read message: %w", err)
	}

	// Parse message
	msgType, payload, err := ParseKNXDMessage(buf[:totalLen])
	if err != nil {
		c.logError("parse message failed", err)
		c.errorsTotal.Add(1)
		return 0, nil, nil // Recoverable
	}

	return msgType, payload, nil
}

// handleReadError processes a read error and returns true if the loop should stop.
func (c *KNXDClient) handleReadError(err error) bool {
	if err == nil {
		return false // No error, continue
	}

	if c.isClosed() {
		return true // Clean shutdown
	}

	// Protocol desync is always fatal - stream is corrupted
	// Must close socket immediately to stop corrupted data flow
	if errors.Is(err, ErrProtocolDesync) {
		c.logError("protocol desync detected, closing socket", err)
		if c.conn != nil {
			c.conn.Close() // Force immediate close to prevent further corruption
		}
		c.handleDisconnect()
		return true // Fatal, must reconnect
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return false // Timeout is normal, continue
	}

	c.logError("read failed", err)
	c.errorsTotal.Add(1)
	c.handleDisconnect()
	return true // Fatal error, stop
}

// handleGroupPacket processes a received group telegram.
func (c *KNXDClient) handleGroupPacket(payload []byte) {
	telegram, err := ParseTelegram(payload)
	if err != nil {
		c.logError("parse telegram failed", err)
		c.errorsTotal.Add(1)
		return
	}

	c.telegramsRx.Add(1)
	c.lastActivity.Store(time.Now().Unix())

	// Check if callback is set before queueing
	c.callbackMu.RLock()
	hasCallback := c.onTelegram != nil
	c.callbackMu.RUnlock()

	if hasCallback {
		// Queue telegram for bounded worker pool (non-blocking with drop on overflow)
		select {
		case c.callbackQueue <- telegram:
			// Queued successfully
		default:
			// Queue full, drop telegram to prevent memory exhaustion
			c.logError("callback queue full, dropping telegram", nil)
			c.telegramsDropped.Add(1)
			c.errorsTotal.Add(1)
		}
	}
}

// callbackWorker processes telegrams from the callback queue.
// Runs in a bounded worker pool to prevent goroutine explosion.
func (c *KNXDClient) callbackWorker() {
	defer c.wg.Done()

	for {
		select {
		case <-c.done.Done():
			// Drain any remaining items (best-effort, non-blocking)
			c.drainCallbackQueue()
			return
		case telegram := <-c.callbackQueue:
			c.callbackMu.RLock()
			callback := c.onTelegram
			c.callbackMu.RUnlock()

			if callback != nil {
				func() {
					defer func() {
						if r := recover(); r != nil {
							c.logError("telegram callback panic", fmt.Errorf("%v", r))
						}
					}()
					callback(telegram)
				}()
			}
		}
	}
}

// handleDisconnect handles connection loss.
func (c *KNXDClient) handleDisconnect() {
	c.connMu.Lock()
	c.connected = false
	c.connMu.Unlock()

	c.logInfo("connection lost")
}

// drainCallbackQueue removes and discards any remaining items from the callback queue.
// Called during shutdown to prevent goroutines from blocking on send.
func (c *KNXDClient) drainCallbackQueue() {
	for {
		select {
		case <-c.callbackQueue:
			// Discard item
		default:
			return // Queue is empty
		}
	}
}

// isClosed returns true if the client has been closed.
func (c *KNXDClient) isClosed() bool {
	select {
	case <-c.done.Done():
		return true
	default:
		return false
	}
}

// Close gracefully closes the connection.
//
// It signals the receive loop to stop and closes the underlying
// network connection. Safe to call multiple times (uses sync.Once).
//
// Returns:
//   - error: nil (closing is best-effort)
func (c *KNXDClient) Close() error {
	// Signal shutdown (safe to call multiple times via sync.Once)
	c.done.Close()

	// Mark disconnected
	c.connMu.Lock()
	c.connected = false
	c.connMu.Unlock()

	// Close connection (this will unblock any pending reads)
	if c.conn != nil {
		c.conn.Close()
	}

	// Wait for all goroutines to finish
	c.wg.Wait()

	c.logInfo("connection closed")
	return nil
}

// Send sends a group write telegram to the KNX bus.
//
// Parameters:
//   - ctx: Context for cancellation
//   - ga: Target group address
//   - data: DPT-encoded payload
//
// Returns:
//   - error: If sending fails or client is not connected
func (c *KNXDClient) Send(ctx context.Context, ga GroupAddress, data []byte) error {
	if !c.IsConnected() {
		return ErrNotConnected
	}

	telegram := NewWriteTelegram(ga, data)
	return c.sendTelegram(ctx, telegram)
}

// SendRead sends a group read request to the KNX bus.
//
// Parameters:
//   - ctx: Context for cancellation
//   - ga: Target group address to read
//
// Returns:
//   - error: If sending fails or client is not connected
func (c *KNXDClient) SendRead(ctx context.Context, ga GroupAddress) error {
	if !c.IsConnected() {
		return ErrNotConnected
	}

	telegram := NewReadTelegram(ga)
	return c.sendTelegram(ctx, telegram)
}

// sendTelegram sends a telegram to knxd.
func (c *KNXDClient) sendTelegram(ctx context.Context, t Telegram) error {
	// Check context
	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", ErrTelegramFailed, ctx.Err())
	default:
	}

	// Encode telegram
	payload := t.Encode()
	msg := EncodeKNXDMessage(EIBGroupPacket, payload)

	// Send with deadline
	deadline := time.Now().Add(defaultWriteTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return ErrNotConnected
	}

	if err := conn.SetWriteDeadline(deadline); err != nil {
		return fmt.Errorf("%w: set deadline: %w", ErrTelegramFailed, err)
	}

	// Check context again before write (might have been cancelled during encoding)
	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", ErrTelegramFailed, ctx.Err())
	default:
	}

	if _, err := conn.Write(msg); err != nil {
		c.errorsTotal.Add(1)
		return fmt.Errorf("%w: write: %w", ErrTelegramFailed, err)
	}

	c.telegramsTx.Add(1)
	c.lastActivity.Store(time.Now().Unix())

	return nil
}

// SetOnTelegram sets the callback for received telegrams.
//
// The callback is invoked in a separate goroutine for each telegram.
// Panics in the callback are recovered and logged.
//
// Parameters:
//   - callback: Function to call when a telegram is received
func (c *KNXDClient) SetOnTelegram(callback func(Telegram)) {
	c.callbackMu.Lock()
	c.onTelegram = callback
	c.callbackMu.Unlock()
}

// SetLogger sets the logger for this client.
func (c *KNXDClient) SetLogger(logger Logger) {
	c.loggerMu.Lock()
	c.logger = logger
	c.loggerMu.Unlock()
}

// IsConnected returns true if connected to knxd.
func (c *KNXDClient) IsConnected() bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.connected
}

// Stats returns current operational statistics.
func (c *KNXDClient) Stats() KNXDStats {
	return KNXDStats{
		TelegramsTx:      c.telegramsTx.Load(),
		TelegramsRx:      c.telegramsRx.Load(),
		TelegramsDropped: c.telegramsDropped.Load(),
		ErrorsTotal:      c.errorsTotal.Load(),
		LastActivity:     time.Unix(c.lastActivity.Load(), 0),
		Connected:        c.IsConnected(),
	}
}

// HealthCheck verifies the connection is alive.
//
// Note: This only checks connection state. For active verification,
// send a read request to a known GA and wait for response.
func (c *KNXDClient) HealthCheck(_ context.Context) error {
	if !c.IsConnected() {
		return ErrNotConnected
	}
	return nil
}

// logInfo logs an info message if logger is set.
func (c *KNXDClient) logInfo(msg string, keysAndValues ...any) {
	c.loggerMu.RLock()
	logger := c.logger
	c.loggerMu.RUnlock()

	if logger != nil {
		logger.Info(msg, keysAndValues...)
	}
}

// logError logs an error message if logger is set.
func (c *KNXDClient) logError(msg string, err error) {
	c.loggerMu.RLock()
	logger := c.logger
	c.loggerMu.RUnlock()

	if logger != nil {
		logger.Error(msg, "error", err)
	}
}
