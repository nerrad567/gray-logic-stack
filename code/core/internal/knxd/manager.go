package knxd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/nerrad567/gray-logic-core/internal/process"
)

// Timeouts and intervals for knxd management.
const (
	// readyTimeout is how long to wait for knxd to accept TCP connections after starting.
	readyTimeout = 30 * time.Second

	// readyPollInterval is how often to try connecting during readiness check.
	readyPollInterval = 100 * time.Millisecond

	// dialTimeout is the timeout for individual TCP connection attempts.
	dialTimeout = 500 * time.Millisecond

	// pidFilePath is the default location for the knxd PID file.
	// This prevents multiple instances from running simultaneously.
	pidFilePath = "/var/run/graylogic-knxd.pid"

	// pidFileMode is the permission mode for the PID file.
	pidFileMode = 0600

	// pidFileFallbackPath is used if we can't write to /var/run
	pidFileFallbackPath = "/tmp/graylogic-knxd.pid"
)

// HealthError represents a health check failure with recoverability information.
// This allows the process manager to decide whether restarting will help.
type HealthError struct {
	// Layer is which health check layer failed (0-4)
	Layer int
	// Recoverable indicates if restarting the process might fix the issue
	Recoverable bool
	// Err is the underlying error
	Err error
}

func (e *HealthError) Error() string {
	return fmt.Sprintf("health check layer %d failed: %v", e.Layer, e.Err)
}

func (e *HealthError) Unwrap() error {
	return e.Err
}

// IsRecoverable implements the process.RecoverableError interface.
func (e *HealthError) IsRecoverable() bool {
	return e.Recoverable
}

// newHealthError creates a health check error for a specific layer.
func newHealthError(layer int, recoverable bool, err error) *HealthError {
	return &HealthError{
		Layer:       layer,
		Recoverable: recoverable,
		Err:         err,
	}
}

// Logger defines the logging interface for the knxd manager.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// noopLogger is a logger that does nothing.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

// GroupAddressProvider supplies group addresses for Layer 3 health checks.
// This is typically implemented by the bus monitor which learns addresses passively.
type GroupAddressProvider interface {
	// GetHealthCheckGroupAddresses returns group addresses for health checks.
	// Returns up to 'limit' addresses, cycling through them to spread load
	// and discover read capabilities on more addresses over time.
	GetHealthCheckGroupAddresses(ctx context.Context, limit int) ([]string, error)

	// MarkHealthCheckUsed records that a GA was just used for a health check.
	// This enables cycling through addresses instead of always using the same one.
	MarkHealthCheckUsed(ctx context.Context, ga string) error
}

// Manager manages the knxd daemon process.
type Manager struct {
	config               Config
	process              *process.Manager
	logger               Logger
	groupAddressProvider GroupAddressProvider // For Layer 3 health checks

	// dStateCount tracks consecutive health checks where knxd is in D (uninterruptible sleep) state.
	// Reset to 0 when knxd returns to a healthy state.
	// Uses atomic.Int32 for thread-safe access from health check goroutine.
	dStateCount atomic.Int32

	// activePIDFilePath stores the path used when acquiring the PID file.
	// This ensures removePIDFile() removes the same file that was acquired,
	// even if /var/run permissions change at runtime.
	activePIDFilePath string
}

// NewManager creates a new knxd manager.
func NewManager(cfg Config) (*Manager, error) {
	// Apply defaults for zero values
	if cfg.Binary == "" {
		cfg.Binary = "/usr/bin/knxd"
	}
	if cfg.PhysicalAddress == "" {
		cfg.PhysicalAddress = "0.0.1"
	}
	if cfg.ClientAddresses == "" {
		cfg.ClientAddresses = "0.0.2:8"
	}
	if cfg.TCPPort == 0 {
		cfg.TCPPort = 6720
	}
	if cfg.UnixSocket == "" {
		cfg.UnixSocket = "/tmp/graylogic-knxd.sock"
	}
	if cfg.RestartDelay == 0 {
		cfg.RestartDelay = 5 * time.Second
	}
	if cfg.MaxRestartAttempts == 0 {
		cfg.MaxRestartAttempts = 10
	}
	if cfg.GracefulTimeout == 0 {
		cfg.GracefulTimeout = 10 * time.Second
	}
	if cfg.HealthCheckInterval == 0 {
		cfg.HealthCheckInterval = 30 * time.Second
	}
	if cfg.HealthCheckDeviceTimeout == 0 {
		cfg.HealthCheckDeviceTimeout = 3 * time.Second
	}
	if cfg.Backend.Type == "" {
		cfg.Backend.Type = BackendUSB
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid knxd config: %w", err)
	}

	m := &Manager{
		config: cfg,
		logger: noopLogger{},
	}

	return m, nil
}

// SetLogger sets the logger for the manager.
func (m *Manager) SetLogger(logger Logger) {
	m.logger = logger
}

// SetGroupAddressProvider sets the group address provider for Layer 3 health checks.
// The provider supplies group addresses learned from bus traffic.
func (m *Manager) SetGroupAddressProvider(provider GroupAddressProvider) {
	m.groupAddressProvider = provider
}

// Start launches the knxd daemon.
// It will block until knxd is ready to accept connections.
func (m *Manager) Start(ctx context.Context) error {
	if !m.config.Managed {
		m.logger.Info("knxd management disabled, expecting external knxd")
		return nil
	}

	args := m.config.BuildArgs()

	m.logger.Info("starting knxd",
		"binary", m.config.Binary,
		"args", args,
		"backend", m.config.Backend.Type,
	)

	// Create the process manager
	procConfig := process.Config{
		Name:               "knxd",
		Binary:             m.config.Binary,
		Args:               args,
		RestartOnFailure:   m.config.RestartOnFailure,
		RestartDelay:       m.config.RestartDelay,
		MaxRestartAttempts: m.config.MaxRestartAttempts,
		GracefulTimeout:    m.config.GracefulTimeout,
		OnStart: func() {
			m.logger.Info("knxd process started", "pid", m.process.PID())
		},
		OnStop: func(err error) {
			if err != nil {
				m.logger.Warn("knxd process stopped", "error", err)
			} else {
				m.logger.Info("knxd process stopped")
			}
		},
		OnRestart: func(attempt int) {
			m.logger.Info("knxd restarting", "attempt", attempt)
			// Reset USB device before restart if configured
			if m.config.Backend.USBResetOnRetry {
				if err := m.resetUSBDevice(); err != nil {
					m.logger.Warn("USB reset failed before restart", "error", err)
				}
			}
		},
		// Watchdog: periodic health check to detect hung knxd
		HealthCheckInterval: m.config.HealthCheckInterval,
		HealthCheckFunc: func(ctx context.Context) error {
			return m.HealthCheck(ctx)
		},
	}

	m.process = process.NewManager(procConfig)
	m.process.SetLogger(m.logger)

	// Start the process
	if err := m.process.Start(ctx); err != nil {
		return fmt.Errorf("starting knxd: %w", err)
	}

	// Wait for knxd to be ready (TCP port accepting connections)
	if m.config.ListenTCP {
		if err := m.waitForReady(ctx); err != nil {
			// Stop the process if it didn't become ready
			if stopErr := m.process.Stop(); stopErr != nil {
				m.logger.Warn("error stopping knxd after failed readiness check", "error", stopErr)
			}
			return fmt.Errorf("knxd failed to become ready: %w", err)
		}
	}

	// Atomically acquire PID file to prevent duplicate instances
	// This is done AFTER knxd starts so we have the real PID
	pid := m.process.PID()
	if pid > 0 {
		if err := m.acquirePIDFile(pid); err != nil {
			// Another instance started between our check - stop this one
			m.logger.Error("failed to acquire PID file, stopping duplicate instance", "error", err)
			_ = m.process.Stop() //nolint:errcheck // Error ignored - we're already handling a fatal error
			return fmt.Errorf("cannot start: %w", err)
		}
	}

	m.logger.Info("knxd ready",
		"connection_url", m.config.ConnectionURL(),
		"physical_address", m.config.PhysicalAddress,
	)

	return nil
}

// waitForReady waits for knxd to be ready to accept connections.
func (m *Manager) waitForReady(ctx context.Context) error {
	addr := fmt.Sprintf("localhost:%d", m.config.TCPPort)
	deadline := time.Now().Add(readyTimeout)

	m.logger.Debug("waiting for knxd to be ready", "address", addr)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for knxd: %w", ctx.Err())
		default:
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for knxd on %s after %v", addr, readyTimeout)
		}

		// Check if process is still running
		if !m.process.IsRunning() {
			lastErr := m.process.LastError()
			if lastErr != nil {
				return fmt.Errorf("knxd process exited: %w", lastErr)
			}
			return errors.New("knxd process exited unexpectedly")
		}

		// Try to connect
		conn, err := net.DialTimeout("tcp", addr, dialTimeout)
		if err == nil {
			conn.Close()
			return nil
		}

		time.Sleep(readyPollInterval)
	}
}

// Stop gracefully stops the knxd daemon.
func (m *Manager) Stop() error {
	if !m.config.Managed || m.process == nil {
		return nil
	}

	m.logger.Info("stopping knxd")

	// Stop the process first, then remove PID file.
	// This prevents a race where a new instance could start before the old one
	// has fully released resources (TCP port, USB device).
	err := m.process.Stop()

	// Remove PID file after process has stopped
	m.removePIDFile()

	return err
}

// IsRunning returns true if knxd is currently running.
func (m *Manager) IsRunning() bool {
	if !m.config.Managed {
		// If not managed, assume external knxd is running
		// Could add a health check here
		return true
	}
	if m.process == nil {
		return false
	}
	return m.process.IsRunning()
}

// IsManaged returns true if this manager is controlling knxd.
func (m *Manager) IsManaged() bool {
	return m.config.Managed
}

// ConnectionURL returns the URL for connecting to knxd.
func (m *Manager) ConnectionURL() string {
	return m.config.ConnectionURL()
}

// Stats returns current statistics for knxd.
func (m *Manager) Stats() Stats {
	stats := Stats{
		Managed:       m.config.Managed,
		Backend:       string(m.config.Backend.Type),
		ConnectionURL: m.config.ConnectionURL(),
	}

	if m.process != nil {
		procStats := m.process.Stats()
		stats.Status = string(procStats.Status)
		stats.PID = procStats.PID
		stats.Uptime = procStats.Uptime
		stats.RestartCount = procStats.RestartCount
		stats.LastError = procStats.LastError
	} else if !m.config.Managed {
		stats.Status = "external"
	} else {
		stats.Status = "stopped"
	}

	return stats
}

// Stats holds statistics about the knxd daemon.
type Stats struct {
	Managed       bool          `json:"managed"`
	Status        string        `json:"status"`
	Backend       string        `json:"backend"`
	ConnectionURL string        `json:"connection_url"`
	PID           int           `json:"pid,omitempty"`
	Uptime        time.Duration `json:"uptime,omitempty"`
	RestartCount  int           `json:"restart_count"`
	LastError     string        `json:"last_error,omitempty"`
}

// HealthCheck verifies knxd is healthy using a multi-layer approach:
//
// Layer 0: USB device presence check (USB backend only)
//   - Detects: USB device disconnection, hardware failure
//   - Speed: ~5ms (lsusb call)
//   - NOT RECOVERABLE: If hardware is missing, restarting won't help
//
// Layer 1: Process state check (/proc/PID/stat)
//   - Detects: SIGSTOP (T), zombie (Z), dead (X) states
//   - Speed: ~0.1ms
//
// Layer 3: Group Read check (GroupValue_Read) - NEW
//   - Detects: Interface failure, bus disconnection, knxd hang
//   - Speed: ~100-500ms
//   - Uses group addresses learned passively from bus monitor
//   - Works with ALL devices including simulators
//   - Fallback when Layer 4 fails
//
// Layer 4: Bus device check (A_DeviceDescriptor_Read) - preferred
//   - Detects: Interface failure, bus disconnection, knxd hang
//   - Speed: ~100-500ms
//
// HealthCheck verifies knxd and the KNX bus are healthy.
//
// Layers:
//   - Layer 0: USB device presence (USB backend only)
//   - Layer 1: Process state via /proc
//   - Layer 3: Bus health via GroupValue_Read
//
// Uses group addresses learned passively from bus traffic.
func (m *Manager) HealthCheck(ctx context.Context) error {
	if !m.config.Managed && !m.config.ListenTCP {
		return nil // Can't health check if no TCP
	}

	// Layer 0: USB device presence check (USB backend only)
	// This is the fastest check and catches hardware disconnection immediately
	// NOT RECOVERABLE: If hardware is missing, restarting knxd won't help
	if m.config.Backend.Type == BackendUSB {
		if err := m.checkUSBDevicePresent(ctx); err != nil {
			return newHealthError(0, false, err) // Layer 0, NOT recoverable
		}
	}

	// Layer 1: Verify process state via /proc (fast, catches SIGSTOP/zombie)
	// RECOVERABLE: Restarting will fix zombie/stopped states
	if m.process != nil {
		pid := m.process.PID()
		if pid > 0 {
			if err := m.checkProcessState(pid); err != nil {
				return newHealthError(1, true, err) // Layer 1, recoverable
			}
		}
	}

	// Layer 3: Bus health check using GroupValue_Read
	// This works with ALL devices including simulators
	// Uses group addresses learned passively from bus traffic
	if m.groupAddressProvider != nil {
		addresses, err := m.groupAddressProvider.GetHealthCheckGroupAddresses(ctx, 5)
		if err != nil {
			m.logger.Debug("failed to get health check group addresses", "error", err)
		} else if len(addresses) > 0 {
			usedAddr, err := m.checkBusHealthWithGroupAddresses(ctx, addresses)
			if err != nil {
				// Bus health check failed
				// For USB backend, attempt USB reset before reporting failure
				if m.config.Backend.Type == BackendUSB && m.config.Backend.USBResetOnBusFailure {
					m.logger.Warn("bus health check failed, attempting USB reset",
						"error", err,
						"device", fmt.Sprintf("%s:%s", m.config.Backend.USBVendorID, m.config.Backend.USBProductID),
					)
					if resetErr := m.resetUSBDeviceWithContext(ctx); resetErr != nil {
						m.logger.Warn("USB reset failed", "error", resetErr)
					}
				}
				return newHealthError(3, true, err)
			}
			// Mark the GA as used so we cycle to a different one next time
			if usedAddr != "" {
				if markErr := m.groupAddressProvider.MarkHealthCheckUsed(ctx, usedAddr); markErr != nil {
					m.logger.Debug("failed to mark health check GA as used", "address", usedAddr, "error", markErr)
				}
			}
			return nil // Layer 3 succeeded
		}
	}

	// No addresses discovered yet - system still in discovery mode
	return nil
}

// checkUSBDevicePresent verifies the USB KNX interface is physically connected.
// This is Layer 0 of the health check - the fastest possible check for USB backends.
// It uses lsusb to check if the device with the configured vendor:product ID exists.
// The parent context is respected to allow clean shutdown during health checks.
func (m *Manager) checkUSBDevicePresent(ctx context.Context) error {
	vendorID := m.config.Backend.USBVendorID
	productID := m.config.Backend.USBProductID

	// If USB IDs aren't configured, skip this check
	if vendorID == "" || productID == "" {
		return nil
	}

	// Use lsusb to check if device is present
	// Format: lsusb -d vendor:product
	// Apply timeout to prevent hanging if USB subsystem is unresponsive
	const usbCheckTimeout = 5 * time.Second
	checkCtx, cancel := context.WithTimeout(ctx, usbCheckTimeout)
	defer cancel()

	deviceID := fmt.Sprintf("%s:%s", vendorID, productID)
	cmd := exec.CommandContext(checkCtx, "lsusb", "-d", deviceID)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if checkCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("USB device check timed out after %v", usbCheckTimeout)
		}
		// Check if parent context was cancelled (shutdown in progress)
		if ctx.Err() != nil {
			return fmt.Errorf("USB device check cancelled: %w", ctx.Err())
		}
		// lsusb returns exit code 1 if device not found
		return fmt.Errorf("USB KNX interface not detected (device %s): %w", deviceID, err)
	}

	// Verify we actually got output (device found)
	if len(output) == 0 {
		return fmt.Errorf("USB KNX interface not detected (device %s): no lsusb output", deviceID)
	}

	m.logger.Debug("USB device present", "device", deviceID, "info", strings.TrimSpace(string(output)))
	return nil
}

// EIB protocol constants for health check handshake.
const (
	eibOpenTGroup uint16 = 0x0022 // EIB_OPEN_T_GROUP - opens group communication
)

// EIB protocol constants for bus-level health check.
const (
	apciRead     byte = 0x00 // APCI: group read request
	apciResponse byte = 0x40 // APCI: group read response
)

// checkBusHealth performs end-to-end verification by sending a group read request
// to the configured health check device and waiting for a response. This verifies:
//   - knxd is processing messages
//   - The KNX interface (USB/IP) is working
//   - The KNX bus is operational
//   - The target device is powered and responding
//
// This catches failures that protocol-level checks cannot detect, such as:
//   - USB interface disconnection
//   - IP gateway failure
//   - Bus power supply failure
//   - Interface configuration errors
func (m *Manager) checkBusHealth(ctx context.Context) error {
	// Parse the configured health check address
	ga, err := ParseGroupAddress(m.config.HealthCheckDeviceAddress)
	if err != nil {
		return fmt.Errorf("invalid health check device address: %w", err)
	}

	// Determine timeout
	timeout := m.config.HealthCheckDeviceTimeout
	if timeout == 0 {
		timeout = 3 * time.Second // Default
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := fmt.Sprintf("localhost:%d", m.config.TCPPort)

	// Connect to knxd
	var d net.Dialer
	conn, err := d.DialContext(checkCtx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("bus health check failed (connect): %w", err)
	}
	defer conn.Close()

	// Set deadline for the entire operation
	deadline := time.Now().Add(timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("bus health check failed (set deadline): %w", err)
	}

	// Step 1: Send EIB_OPEN_T_GROUP handshake for the specific target address
	handshake := []byte{
		0x00, 0x05, // size = 5 (type + payload)
		0x00, 0x22, // type = EIB_OPEN_T_GROUP (0x0022)
		byte(ga >> 8), byte(ga & 0xFF), // target group address (not 0/0/0!)
		0xFF, // flags (matches knxtool behaviour)
	}

	if _, err := conn.Write(handshake); err != nil {
		return fmt.Errorf("bus health check failed (write handshake): %w", err)
	}

	// Read handshake response
	hsResponse := make([]byte, 4)
	if _, err := io.ReadFull(conn, hsResponse); err != nil {
		return fmt.Errorf("bus health check failed (read handshake response): %w", err)
	}

	respType := uint16(hsResponse[2])<<8 | uint16(hsResponse[3])
	if respType != eibOpenTGroup {
		return fmt.Errorf("bus health check failed: unexpected handshake response 0x%04X", respType)
	}

	// Brief delay after T_Group handshake before sending traffic.
	// USB interfaces need time to process the connection setup.
	time.Sleep(200 * time.Millisecond)

	// Step 2: Send group read request via T_Group connection
	// With T_Group open, we use EIB_APDU_PACKET (dest already set in handshake)
	// Format: size(2) + type(2) + apci_data(2)
	// APCI for read: 0x00 0x00 (read request)
	readRequest := []byte{
		0x00, 0x04, // size = 4 (type + payload)
		0x00, 0x25, // type = EIB_APDU_PACKET (0x0025)
		0x00, apciRead, // APCI: read request
	}

	if _, err := conn.Write(readRequest); err != nil {
		return fmt.Errorf("bus health check failed (write read request): %w", err)
	}

	m.logger.Debug("bus health check: sent read request",
		"address", m.config.HealthCheckDeviceAddress,
		"ga_hex", fmt.Sprintf("0x%04X", ga),
	)

	// Step 3: Wait for response (EIB_GROUP_PACKET with response data)
	// We need to read packets until we get a response to our address, or timeout
	// Defence-in-depth: limit iterations in case of spurious packets on a busy bus
	const maxBusHealthIterations = 100
	iterations := 0
	for {
		iterations++
		if iterations > maxBusHealthIterations {
			return fmt.Errorf("bus health check failed: received %d packets without response from %s", iterations-1, m.config.HealthCheckDeviceAddress)
		}

		select {
		case <-checkCtx.Done():
			return fmt.Errorf("bus health check failed: timeout waiting for response from %s", m.config.HealthCheckDeviceAddress)
		default:
		}

		// Read packet header (size + type)
		header := make([]byte, 4)
		if _, err := io.ReadFull(conn, header); err != nil {
			if errors.Is(err, context.DeadlineExceeded) || isTimeoutError(err) {
				return fmt.Errorf("bus health check failed: no response from %s within %v", m.config.HealthCheckDeviceAddress, timeout)
			}
			return fmt.Errorf("bus health check failed (read response header): %w", err)
		}

		pktSize := uint16(header[0])<<8 | uint16(header[1])
		pktType := uint16(header[2])<<8 | uint16(header[3])

		// Read rest of packet
		// Defence-in-depth: limit packet size to prevent memory exhaustion from malformed data
		// KNX telegrams are max ~23 bytes; EIB messages similarly small; 1KB is very generous
		const maxPacketSize = 1024
		if pktSize < 2 {
			continue // Invalid packet, skip
		}
		if pktSize > maxPacketSize {
			return fmt.Errorf("bus health check failed: packet size %d exceeds maximum %d", pktSize, maxPacketSize)
		}
		payload := make([]byte, pktSize-2)
		if len(payload) > 0 {
			if _, err := io.ReadFull(conn, payload); err != nil {
				return fmt.Errorf("bus health check failed (read payload): %w", err)
			}
		}

		// With T_Group connection, responses come as EIB_APDU_PACKET (0x0025)
		// Payload format: src_addr(2) + apdu(2+)
		const eibApduPacket uint16 = 0x0025
		if pktType != eibApduPacket || len(payload) < 4 {
			continue // Not what we're looking for
		}

		// Parse: src_addr is payload[0:2], APDU is payload[2:]
		// APCI is in the first 2 bytes of APDU: payload[2] and payload[3]
		// For response: APCI low byte contains 0x40 (bits 6-7 of payload[3])
		apci := payload[3] & 0xC0 // High 2 bits of APDU byte 2 indicate response type

		// Check if this is a response (APCI = 0x40 for GroupValue_Response)
		if apci == apciResponse {
			m.logger.Debug("bus health check: received response",
				"address", m.config.HealthCheckDeviceAddress,
			)
			return nil // Success!
		}
		// Not a response, keep waiting (might be our own read request echo)
	}
}

// isTimeoutError checks if an error is a timeout error.
func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}

// checkBusHealthWithGroupAddresses verifies KNX bus health by sending GroupValue_Read
// to one of the passively-learned group addresses.
//
// Returns the address that responded successfully (for tracking), or an error if all failed.
//
// This check works with ALL KNX devices and simulators because group communication
// is the fundamental KNX messaging model - every device supports it.
func (m *Manager) checkBusHealthWithGroupAddresses(ctx context.Context, addresses []string) (string, error) {
	if len(addresses) == 0 {
		return "", nil // No addresses to check
	}

	// Try each address until one responds
	var lastErr error
	for _, addr := range addresses {
		err := m.checkGroupAddressRead(ctx, addr)
		if err == nil {
			return addr, nil // Success - return which address worked
		}
		lastErr = err
		m.logger.Debug("group address read check failed", "address", addr, "error", err)
	}

	return "", fmt.Errorf("layer 3 health check failed: all %d group addresses unresponsive: %w", len(addresses), lastErr)
}

// checkGroupAddressRead sends a GroupValue_Read to a group address and waits for response.
func (m *Manager) checkGroupAddressRead(ctx context.Context, groupAddr string) error {
	// Parse group address
	ga, err := ParseGroupAddress(groupAddr)
	if err != nil {
		return fmt.Errorf("invalid group address %q: %w", groupAddr, err)
	}

	// Determine timeout
	timeout := m.config.HealthCheckDeviceTimeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := fmt.Sprintf("localhost:%d", m.config.TCPPort)

	// Connect to knxd
	var d net.Dialer
	conn, err := d.DialContext(checkCtx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	defer conn.Close()

	// Set deadline for the entire operation
	deadline := time.Now().Add(timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set deadline failed: %w", err)
	}

	// Step 1: Open T_Group connection for the target address
	handshake := []byte{
		0x00, 0x05, // size = 5 (type + payload)
		0x00, 0x22, // type = EIB_OPEN_T_GROUP (0x0022)
		byte(ga >> 8), byte(ga & 0xFF), // target group address
		0xFF, // flags (matches knxtool behaviour)
	}

	if _, err := conn.Write(handshake); err != nil {
		return fmt.Errorf("write handshake failed: %w", err)
	}

	// Read handshake response
	hsResponse := make([]byte, 4)
	if _, err := io.ReadFull(conn, hsResponse); err != nil {
		return fmt.Errorf("read handshake response failed: %w", err)
	}

	respType := uint16(hsResponse[2])<<8 | uint16(hsResponse[3])
	if respType != eibOpenTGroup {
		return fmt.Errorf("unexpected handshake response: 0x%04X", respType)
	}

	// Brief delay after T_Group handshake before sending traffic.
	// USB interfaces need time to process the connection setup.
	time.Sleep(200 * time.Millisecond)

	// Step 2: Send GroupValue_Read request
	readRequest := []byte{
		0x00, 0x04, // size = 4 (type + payload)
		0x00, 0x25, // type = EIB_APDU_PACKET (0x0025)
		0x00, apciRead, // APCI: read request
	}

	if _, err := conn.Write(readRequest); err != nil {
		return fmt.Errorf("write read request failed: %w", err)
	}

	m.logger.Debug("layer 3 health check: sent group read", "address", groupAddr)

	// Step 3: Wait for GroupValue_Response
	const maxIterations = 50
	for i := 0; i < maxIterations; i++ {
		select {
		case <-checkCtx.Done():
			return fmt.Errorf("timeout waiting for response from %s", groupAddr)
		default:
		}

		// Read packet header
		header := make([]byte, 4)
		if _, err := io.ReadFull(conn, header); err != nil {
			if isTimeoutError(err) {
				return fmt.Errorf("timeout waiting for response from %s", groupAddr)
			}
			return fmt.Errorf("read response failed: %w", err)
		}

		pktSize := uint16(header[0])<<8 | uint16(header[1])
		pktType := uint16(header[2])<<8 | uint16(header[3])

		if pktSize < 2 || pktSize > 256 {
			continue // Invalid packet, skip
		}

		// Read payload
		payloadSize := int(pktSize) - 2
		if payloadSize > 0 {
			payload := make([]byte, payloadSize)
			if _, err := io.ReadFull(conn, payload); err != nil {
				continue
			}

			// Check for APDU packet with response
			// T_Group responses have format: src_addr(2) + apdu(2+)
			// So we need at least 4 bytes: 2 for source, 2 for APCI
			const eibApduPacket uint16 = 0x0025
			if pktType == eibApduPacket && len(payload) >= 4 {
				// Parse: src_addr is payload[0:2], APDU is payload[2:]
				// APCI is in the first 2 bytes of APDU: payload[2] and payload[3]
				// For response: APCI low byte contains 0x40 (bits 6-7 of payload[3])
				apci := payload[3] & 0xC0 // High 2 bits of APDU byte 2 indicate response type
				if apci == apciResponse {
					srcAddr := uint16(payload[0])<<8 | uint16(payload[1])
					m.logger.Debug("layer 3 health check: received response",
						"address", groupAddr,
						"from", FormatIndividualAddress(srcAddr),
					)
					return nil // Success!
				}
			}
		}
	}

	return fmt.Errorf("no response from %s after %d packets", groupAddr, maxIterations)
}

// checkProcessState reads /proc/PID/stat to verify the process is in a healthy state.
// Returns an error if the process is stopped (T), traced (t), zombie (Z), or dead (X/x).
func (m *Manager) checkProcessState(pid int) error {
	// Read /proc/PID/stat which contains process state as the 3rd field
	// Format: pid (comm) state ...
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		// Process might have just exited
		return fmt.Errorf("cannot read process state: %w", err)
	}

	// Parse the stat file. The state is the 3rd field, after "(comm)"
	// We need to find the closing ) of the comm field first
	statStr := string(data)
	closeParenIdx := strings.LastIndex(statStr, ")")
	if closeParenIdx == -1 || closeParenIdx+2 >= len(statStr) {
		return fmt.Errorf("invalid /proc/stat format")
	}

	// Fields after ) are space-separated, state is the first one
	fields := strings.Fields(statStr[closeParenIdx+2:])
	if len(fields) < 1 {
		return fmt.Errorf("invalid /proc/stat format: no state field")
	}

	state := fields[0]

	// Process states (from proc(5) man page):
	// R = running, S = sleeping, D = disk sleep (uninterruptible)
	// T = stopped (SIGSTOP), t = tracing stop
	// Z = zombie, X/x = dead
	// W = paging (not used since 2.6.xx), I = idle
	switch state {
	case "T", "t":
		return fmt.Errorf("knxd process is stopped (state=%s)", state)
	case "Z":
		return fmt.Errorf("knxd process is zombie (state=%s)", state)
	case "X", "x":
		return fmt.Errorf("knxd process is dead (state=%s)", state)
	case "D":
		// D (uninterruptible sleep) is usually temporary (disk/USB I/O).
		// However, if knxd is stuck in D-state for multiple health checks,
		// the USB interface is likely hung and needs recovery.
		count := m.dStateCount.Add(1)
		if count >= 3 {
			return fmt.Errorf("knxd process stuck in uninterruptible sleep (state=D, count=%d)", count)
		}
		m.logger.Debug("knxd process in uninterruptible sleep (state=D)", "count", count)
		return nil
	default:
		// R, S, I are all healthy states - reset D-state counter
		m.dStateCount.Store(0)
		return nil
	}
}

// resetUSBDeviceWithContext resets the USB KNX interface using the usbreset utility.
// This helps recover from LIBUSB_ERROR_BUSY conditions without requiring
// root privileges, as long as proper udev rules are in place.
//
// The usbreset utility uses the USBDEVFS_RESET ioctl, which only requires
// write access to the USB device (granted via udev rules).
//
// Required udev rule (e.g., /etc/udev/rules.d/90-knx-usb.rules):
//
//	SUBSYSTEM=="usb", ATTR{idVendor}=="0e77", ATTR{idProduct}=="0104", MODE="0666"
//
// The parent context is respected to allow clean shutdown during reset operations.
func (m *Manager) resetUSBDeviceWithContext(ctx context.Context) error {
	if m.config.Backend.Type != BackendUSB {
		return nil // Only applicable for USB backends
	}

	vendorID := m.config.Backend.USBVendorID
	productID := m.config.Backend.USBProductID

	if vendorID == "" || productID == "" {
		m.logger.Debug("USB reset skipped: vendor/product ID not configured")
		return nil
	}

	deviceID := fmt.Sprintf("%s:%s", vendorID, productID)
	m.logger.Info("resetting USB device", "device", deviceID)

	// Use usbreset utility (standard on most Linux systems with usbutils)
	// Format: usbreset <vendor>:<product>
	// Apply timeout to prevent hanging if USB hardware is unresponsive
	const usbResetTimeout = 10 * time.Second
	resetCtx, cancel := context.WithTimeout(ctx, usbResetTimeout)
	defer cancel()

	cmd := exec.CommandContext(resetCtx, "usbreset", deviceID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if resetCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("usbreset timed out after %v", usbResetTimeout)
		}
		// Check if parent context was cancelled (shutdown in progress)
		if ctx.Err() != nil {
			return fmt.Errorf("usbreset cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("usbreset failed: %w (output: %s)", err, string(output))
	}

	m.logger.Info("USB device reset successful", "device", deviceID)

	// Brief delay to allow the device to fully reinitialise
	time.Sleep(500 * time.Millisecond)

	return nil
}

// resetUSBDevice resets the USB device without a context (uses background context).
// Used by OnRestart callback which doesn't have access to a context.
func (m *Manager) resetUSBDevice() error {
	return m.resetUSBDeviceWithContext(context.Background())
}

// ResetUSBDevice is the public method to manually reset the USB device.
// This can be called externally when USB issues are detected.
func (m *Manager) ResetUSBDevice() error {
	return m.resetUSBDevice()
}

// getPIDFilePath returns the path for the PID file, preferring /var/run but
// falling back to /tmp if that's not writable.
func (m *Manager) getPIDFilePath() string {
	// Try /var/run first (standard location for daemon PID files)
	if f, err := os.OpenFile(pidFilePath, os.O_CREATE|os.O_WRONLY, pidFileMode); err == nil {
		f.Close()
		os.Remove(pidFilePath) // Remove the test file
		return pidFilePath
	}
	// Fall back to /tmp
	return pidFileFallbackPath
}

// acquirePIDFile atomically creates the PID file and writes our PID.
// This uses O_EXCL to ensure no race condition between checking for existing
// instances and claiming the PID file.
//
// Returns nil on success (PID file created with our PID).
// Returns an error if another instance is already running.
func (m *Manager) acquirePIDFile(pid int) error {
	return m.acquirePIDFileWithRetry(pid, 0)
}

// maxPIDFileRetries limits recursion depth for PID file acquisition.
const maxPIDFileRetries = 3

// acquirePIDFileWithRetry implements PID file acquisition with bounded retries.
func (m *Manager) acquirePIDFileWithRetry(pid int, attempt int) error {
	if attempt >= maxPIDFileRetries {
		return fmt.Errorf("failed to acquire PID file after %d attempts", maxPIDFileRetries)
	}

	// Determine path once on first attempt and store it.
	// This ensures removePIDFile() uses the same path even if /var/run permissions change.
	if attempt == 0 {
		m.activePIDFilePath = m.getPIDFilePath()
	}
	pidFile := m.activePIDFilePath
	content := fmt.Sprintf("%d\n", pid)

	// Try atomic exclusive create - fails if file already exists
	f, err := os.OpenFile(pidFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, pidFileMode)
	if err == nil {
		// Success - we got the lock, write our PID
		defer f.Close()
		if _, writeErr := f.WriteString(content); writeErr != nil {
			os.Remove(pidFile)
			return fmt.Errorf("writing PID file: %w", writeErr)
		}
		m.logger.Debug("acquired PID file", "path", pidFile, "pid", pid)
		return nil
	}

	// File exists - check if it's stale
	if !os.IsExist(err) {
		return fmt.Errorf("creating PID file %s: %w", pidFile, err)
	}

	// Read existing PID
	data, readErr := os.ReadFile(pidFile)
	if readErr != nil {
		// Can't read it, try to remove and retry
		os.Remove(pidFile)
		return m.acquirePIDFileWithRetry(pid, attempt+1)
	}

	pidStr := strings.TrimSpace(string(data))
	existingPID, parseErr := strconv.Atoi(pidStr)
	if parseErr != nil {
		// Invalid PID file, remove and retry
		m.logger.Warn("removing invalid PID file", "path", pidFile, "content", pidStr)
		os.Remove(pidFile)
		return m.acquirePIDFileWithRetry(pid, attempt+1)
	}

	// Check if the existing PID is still alive
	if !m.isKnxdProcessAlive(existingPID) {
		// Stale PID file, remove and retry
		m.logger.Info("removing stale PID file", "path", pidFile, "stale_pid", existingPID)
		os.Remove(pidFile)
		return m.acquirePIDFileWithRetry(pid, attempt+1)
	}

	// Another knxd instance is actually running
	return fmt.Errorf("another knxd instance is already running (PID %d, file %s)", existingPID, pidFile)
}

// isKnxdProcessAlive checks if a process with the given PID is running and is knxd.
func (m *Manager) isKnxdProcessAlive(pid int) bool {
	// Check if process exists and we can signal it
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds - send signal 0 to check if alive
	if signalErr := proc.Signal(syscall.Signal(0)); signalErr != nil {
		return false // Process is dead
	}

	// Process is alive - verify it's actually knxd
	commPath := fmt.Sprintf("/proc/%d/comm", pid)
	commData, err := os.ReadFile(commPath)
	if err != nil {
		// Can't verify identity, assume it's not our knxd
		return false
	}

	comm := strings.TrimSpace(string(commData))
	return comm == "knxd"
}

// removePIDFile removes the PID file if it exists.
func (m *Manager) removePIDFile() {
	// Use the stored path from acquisition, not a fresh determination.
	// This ensures we remove the same file we created, even if /var/run permissions changed.
	pidFile := m.activePIDFilePath
	if pidFile == "" {
		return // Never acquired a PID file
	}
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		m.logger.Warn("failed to remove PID file", "path", pidFile, "error", err)
	} else if err == nil {
		m.logger.Debug("removed PID file", "path", pidFile)
	}
	m.activePIDFilePath = "" // Clear after removal
}
