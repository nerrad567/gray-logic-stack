package knxd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
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

	// healthCheckTimeout is the timeout for health check connection attempts.
	healthCheckTimeout = 5 * time.Second
)

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

// Manager manages the knxd daemon process.
type Manager struct {
	config  Config
	process *process.Manager
	logger  Logger
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
	return m.process.Stop()
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

// HealthCheck verifies knxd is healthy using a multi-layer defense-in-depth approach:
//
// Layer 0: USB device presence check (USB backend only)
//   - Detects: USB device disconnection, hardware failure
//   - Speed: ~0.1ms (filesystem check)
//
// Layer 1: Process state check (/proc/PID/stat)
//   - Detects: SIGSTOP (T), zombie (Z), dead (X) states
//   - Speed: ~0.1ms
//
// Layer 2: TCP connectivity
//   - Detects: Process crash, port not bound
//   - Speed: ~1ms
//
// Layer 3: EIB protocol handshake
//   - Detects: Application deadlocks, infinite loops
//   - Speed: ~10ms
//
// Layer 4: Bus-level device read (optional, if HealthCheckDeviceAddress configured)
//   - Detects: Interface failure, bus disconnection, PSU failure
//   - Speed: ~100-500ms (depends on bus)
//   - Requires a responsive device on the bus (e.g., PSU diagnostic address)
//
// Each layer catches issues the others miss, providing comprehensive health verification.
// For USB backends, Layer 0 provides immediate detection of hardware disconnection.
func (m *Manager) HealthCheck(ctx context.Context) error {
	if !m.config.Managed && !m.config.ListenTCP {
		return nil // Can't health check if no TCP
	}

	// Layer 0: USB device presence check (USB backend only)
	// This is the fastest check and catches hardware disconnection immediately
	if m.config.Backend.Type == BackendUSB {
		if err := m.checkUSBDevicePresent(); err != nil {
			return err
		}
	}

	// Layer 1: Verify process state via /proc (fast, catches SIGSTOP/zombie)
	if m.process != nil {
		pid := m.process.PID()
		if pid > 0 {
			if err := m.checkProcessState(pid); err != nil {
				return err
			}
		}
	}

	// Layer 2+3: TCP connect + EIB protocol handshake
	// This verifies knxd is actually processing messages, not just accepting connections
	if err := m.checkProtocolHealth(ctx); err != nil {
		return err
	}

	// Layer 4: Bus-level device read (optional)
	// If a health check device is configured, verify actual KNX bus communication
	if m.config.HealthCheckDeviceAddress != "" {
		if err := m.checkBusHealth(ctx); err != nil {
			// For USB backend, attempt USB reset before reporting failure
			// This can recover from LIBUSB errors without waiting for full restart
			if m.config.Backend.Type == BackendUSB && m.config.Backend.USBResetOnBusFailure {
				m.logger.Warn("bus health check failed, attempting proactive USB reset",
					"error", err,
					"device", fmt.Sprintf("%s:%s", m.config.Backend.USBVendorID, m.config.Backend.USBProductID),
				)
				if resetErr := m.resetUSBDevice(); resetErr != nil {
					m.logger.Warn("USB reset failed", "error", resetErr)
				} else {
					m.logger.Info("USB reset completed, will retry on next health check")
				}
			}
			return err
		}
	}

	return nil
}

// checkUSBDevicePresent verifies the USB KNX interface is physically connected.
// This is Layer 0 of the health check - the fastest possible check for USB backends.
// It uses lsusb to check if the device with the configured vendor:product ID exists.
func (m *Manager) checkUSBDevicePresent() error {
	vendorID := m.config.Backend.USBVendorID
	productID := m.config.Backend.USBProductID

	// If USB IDs aren't configured, skip this check
	if vendorID == "" || productID == "" {
		return nil
	}

	// Use lsusb to check if device is present
	// Format: lsusb -d vendor:product
	deviceID := fmt.Sprintf("%s:%s", vendorID, productID)
	cmd := exec.Command("lsusb", "-d", deviceID)
	output, err := cmd.CombinedOutput()

	if err != nil {
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

// checkProtocolHealth connects to knxd and performs an EIB protocol handshake.
// This verifies that knxd is actually processing protocol messages, catching
// application-level hangs that /proc/stat cannot detect (deadlocks, infinite loops, etc).
func (m *Manager) checkProtocolHealth(ctx context.Context) error {
	addr := fmt.Sprintf("localhost:%d", m.config.TCPPort)

	checkCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	// Connect to knxd
	var d net.Dialer
	conn, err := d.DialContext(checkCtx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("knxd health check failed (connect): %w", err)
	}
	defer conn.Close()

	// Set deadlines for the handshake
	deadline := time.Now().Add(healthCheckTimeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("knxd health check failed (set deadline): %w", err)
	}

	// Send EIB_OPEN_T_GROUP handshake
	// Format: size(2) + type(2) + group_addr(2) + flags(1)
	// We use group 0/0/0 (0x0000) with flags 0xFF
	handshake := []byte{
		0x00, 0x05, // size = 5 (type + payload)
		0x00, 0x22, // type = EIB_OPEN_T_GROUP (0x0022)
		0x00, 0x00, // group address = 0/0/0
		0xFF, // flags
	}

	if _, err := conn.Write(handshake); err != nil {
		return fmt.Errorf("knxd health check failed (write handshake): %w", err)
	}

	// Read the response - knxd should echo back the message type
	// Response format: size(2) + type(2)
	response := make([]byte, 4)
	if _, err := io.ReadFull(conn, response); err != nil {
		return fmt.Errorf("knxd health check failed (read response): %w", err)
	}

	// Verify response type matches (0x0022 = success)
	respType := uint16(response[2])<<8 | uint16(response[3])
	if respType != eibOpenTGroup {
		return fmt.Errorf("knxd health check failed: unexpected response type 0x%04X (expected 0x%04X)", respType, eibOpenTGroup)
	}

	// Success - knxd responded to protocol message
	return nil
}

// EIB protocol constants for bus-level health check.
const (
	eibGroupPacket uint16 = 0x0027 // EIB_GROUP_PACKET - send/receive group telegrams
	apciRead       byte   = 0x00   // APCI: group read request
	apciResponse   byte   = 0x40   // APCI: group read response
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

	// Step 1: Send EIB_OPEN_T_GROUP handshake
	handshake := []byte{
		0x00, 0x05, // size = 5 (type + payload)
		0x00, 0x22, // type = EIB_OPEN_T_GROUP (0x0022)
		0x00, 0x00, // group address = 0/0/0 (subscribe to all)
		0xFF, // flags
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

	// Step 2: Send group read request to health check device
	// Format: size(2) + type(2) + dest_addr(2) + apci_data(2)
	// APCI for read: 0x00 0x00 (read request, no data)
	readRequest := []byte{
		0x00, 0x06, // size = 6 (type + payload)
		0x00, 0x27, // type = EIB_GROUP_PACKET (0x0027)
		byte(ga >> 8), byte(ga & 0xFF), // destination group address
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
	for {
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
		if pktSize < 2 {
			continue // Invalid packet, skip
		}
		payload := make([]byte, pktSize-2)
		if len(payload) > 0 {
			if _, err := io.ReadFull(conn, payload); err != nil {
				return fmt.Errorf("bus health check failed (read payload): %w", err)
			}
		}

		// Check if this is a group packet response
		if pktType != eibGroupPacket || len(payload) < 4 {
			continue // Not what we're looking for
		}

		// Parse response: dest_addr(2) + apci_data(2+)
		respGA := uint16(payload[0])<<8 | uint16(payload[1])
		apci := payload[3] & 0xC0 // High 2 bits of second APCI byte

		// Check if this is a response to our address
		if respGA == ga && apci == apciResponse {
			m.logger.Debug("bus health check: received response",
				"address", m.config.HealthCheckDeviceAddress,
				"data_len", len(payload)-2,
			)
			return nil // Success!
		}

		// Not our response, keep waiting
		m.logger.Debug("bus health check: received unrelated packet",
			"type", fmt.Sprintf("0x%04X", pktType),
			"ga", FormatGroupAddress(respGA),
		)
	}
}

// isTimeoutError checks if an error is a timeout error.
func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
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
		// D (uninterruptible sleep) is usually temporary (disk I/O)
		// We log but don't fail immediately - let consecutive failure logic handle it
		m.logger.Debug("knxd process in uninterruptible sleep (state=D)")
		return nil
	default:
		// R, S, I are all healthy states
		return nil
	}
}

// resetUSBDevice resets the USB KNX interface using the usbreset utility.
// This helps recover from LIBUSB_ERROR_BUSY conditions without requiring
// root privileges, as long as proper udev rules are in place.
//
// The usbreset utility uses the USBDEVFS_RESET ioctl, which only requires
// write access to the USB device (granted via udev rules).
//
// Required udev rule (e.g., /etc/udev/rules.d/90-knx-usb.rules):
//
//	SUBSYSTEM=="usb", ATTR{idVendor}=="0e77", ATTR{idProduct}=="0104", MODE="0666"
func (m *Manager) resetUSBDevice() error {
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
	cmd := exec.Command("usbreset", deviceID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("usbreset failed: %w (output: %s)", err, string(output))
	}

	m.logger.Info("USB device reset successful", "device", deviceID)

	// Brief delay to allow the device to fully reinitialise
	time.Sleep(500 * time.Millisecond)

	return nil
}

// ResetUSBDevice is the public method to manually reset the USB device.
// This can be called externally when USB issues are detected.
func (m *Manager) ResetUSBDevice() error {
	return m.resetUSBDevice()
}
