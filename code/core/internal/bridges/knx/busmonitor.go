package knx

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/nerrad567/gray-logic-core/internal/knxd"
)

// EIB protocol constants for bus monitor mode.
const (
	// EIBOpenVBusmonitor opens virtual bus monitor mode.
	// This receives all bus traffic with full L_Data frames including source addresses.
	EIBOpenVBusmonitor uint16 = 0x0012

	// EIBVBusmonitorPacket is the message type for bus monitor packets.
	EIBVBusmonitorPacket uint16 = 0x0014

	// Minimum L_Data frame size: ctrl(1) + src(2) + dst(2) + info(1) + apci(1)
	minLDataFrameSize = 7

	// busMonitorReadTimeout is the read timeout for bus monitor connection.
	busMonitorReadTimeout = 60 * time.Second

	// busMonitorConnectTimeout is the connection timeout.
	busMonitorConnectTimeout = 10 * time.Second

	// busMonitorReadBuffer is the read buffer size.
	busMonitorReadBuffer = 256
)

// BusMonitor passively observes KNX bus traffic and records device individual addresses
// and group addresses. It connects to knxd in virtual bus monitor mode and extracts
// addresses from all telegrams, building a database of known devices and group
// addresses over time.
//
// This enables health checks to use any discovered device/address without manual configuration:
//   - Layer 4: Individual addresses for DeviceDescriptor_Read
//   - Layer 3: Group addresses for GroupValue_Read (fallback)
type BusMonitor struct {
	db     *sql.DB
	conn   net.Conn
	done   chan struct{}
	wg     sync.WaitGroup
	logger Logger

	// Prepared statements for upserts (created once, reused)
	deviceUpsertStmt *sql.Stmt
	groupUpsertStmt  *sql.Stmt

	mu      sync.RWMutex
	running bool
}

// NewBusMonitor creates a new bus monitor instance.
func NewBusMonitor(db *sql.DB) *BusMonitor {
	return &BusMonitor{
		db:   db,
		done: make(chan struct{}),
	}
}

// SetLogger sets the logger for the bus monitor.
func (m *BusMonitor) SetLogger(logger Logger) {
	m.logger = logger
}

// Start connects to knxd and begins monitoring bus traffic.
func (m *BusMonitor) Start(ctx context.Context, knxdURL string) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("bus monitor already running")
	}
	m.running = true
	m.mu.Unlock()

	// Prepare upsert statements
	var err error
	m.deviceUpsertStmt, err = m.db.Prepare(`
		INSERT INTO knx_devices (individual_address, last_seen, message_count)
		VALUES (?, ?, 1)
		ON CONFLICT(individual_address) DO UPDATE SET
			last_seen = excluded.last_seen,
			message_count = message_count + 1
	`)
	if err != nil {
		return fmt.Errorf("preparing device upsert statement: %w", err)
	}

	m.groupUpsertStmt, err = m.db.Prepare(`
		INSERT INTO knx_group_addresses (group_address, last_seen, message_count, has_read_response)
		VALUES (?, ?, 1, ?)
		ON CONFLICT(group_address) DO UPDATE SET
			last_seen = excluded.last_seen,
			message_count = message_count + 1,
			has_read_response = MAX(has_read_response, excluded.has_read_response)
	`)
	if err != nil {
		m.deviceUpsertStmt.Close()
		return fmt.Errorf("preparing group upsert statement: %w", err)
	}

	// Parse connection URL
	network, address, err := m.parseURL(knxdURL)
	if err != nil {
		return fmt.Errorf("invalid knxd URL: %w", err)
	}

	// Connect with timeout
	connectCtx, cancel := context.WithTimeout(ctx, busMonitorConnectTimeout)
	defer cancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(connectCtx, network, address)
	if err != nil {
		return fmt.Errorf("connecting to knxd: %w", err)
	}
	m.conn = conn

	// Open vbusmonitor mode
	if err := m.openVBusMonitor(); err != nil {
		m.conn.Close()
		return fmt.Errorf("opening vbusmonitor mode: %w", err)
	}

	m.log("bus monitor started", "url", knxdURL)

	// Start receive loop
	m.wg.Add(1)
	go m.receiveLoop()

	return nil
}

// Stop gracefully stops the bus monitor.
func (m *BusMonitor) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.mu.Unlock()

	close(m.done)

	if m.conn != nil {
		m.conn.Close()
	}

	m.wg.Wait()

	if m.deviceUpsertStmt != nil {
		m.deviceUpsertStmt.Close()
	}
	if m.groupUpsertStmt != nil {
		m.groupUpsertStmt.Close()
	}

	m.log("bus monitor stopped")
}

// parseURL parses a knxd connection URL into network and address.
func (m *BusMonitor) parseURL(connURL string) (network, address string, err error) {
	u, err := url.Parse(connURL)
	if err != nil {
		return "", "", err
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
		return "", "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
}

// openVBusMonitor sends the EIB_OPEN_VBUSMONITOR command to knxd.
func (m *BusMonitor) openVBusMonitor() error {
	// EIB_OPEN_VBUSMONITOR has no payload
	msg := EncodeKNXDMessage(EIBOpenVBusmonitor, nil)

	if err := m.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}

	if _, err := m.conn.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	// Read response - knxd protocol: 2-byte length + message body
	if err := m.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}

	// Read length prefix (2 bytes)
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(m.conn, lenBuf); err != nil {
		return fmt.Errorf("read length: %w", err)
	}
	msgLen := binary.BigEndian.Uint16(lenBuf)

	if msgLen < 2 {
		return fmt.Errorf("message length too short: %d", msgLen)
	}

	// Read message body
	respBuf := make([]byte, msgLen)
	if _, err := io.ReadFull(m.conn, respBuf); err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// Check response type (first 2 bytes of message body)
	respType := binary.BigEndian.Uint16(respBuf[0:2])
	if respType != EIBOpenVBusmonitor {
		return fmt.Errorf("unexpected response type: 0x%04X", respType)
	}

	return nil
}

// receiveLoop continuously reads bus monitor packets.
func (m *BusMonitor) receiveLoop() {
	defer m.wg.Done()

	buf := make([]byte, busMonitorReadBuffer)

	for {
		select {
		case <-m.done:
			return
		default:
		}

		// Set read deadline
		if err := m.conn.SetReadDeadline(time.Now().Add(busMonitorReadTimeout)); err != nil {
			m.logError("set read deadline", err)
			return
		}

		// Read message size (2 bytes)
		if _, err := io.ReadFull(m.conn, buf[:2]); err != nil {
			if m.isClosed() {
				return
			}
			// Timeout is normal, continue
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			// EOF during shutdown is normal (connection closed)
			if err == io.EOF {
				return
			}
			m.logError("read size", err)
			return
		}

		msgSize := binary.BigEndian.Uint16(buf[:2])
		if msgSize < 2 || int(msgSize) > len(buf)-2 {
			m.logError("invalid message size", fmt.Errorf("size: %d", msgSize))
			continue
		}

		// Read rest of message
		totalLen := 2 + int(msgSize)
		if _, err := io.ReadFull(m.conn, buf[2:totalLen]); err != nil {
			if m.isClosed() {
				return
			}
			m.logError("read message", err)
			continue
		}

		// Parse message type
		msgType := binary.BigEndian.Uint16(buf[2:4])
		if msgType != EIBVBusmonitorPacket {
			continue // Not a bus monitor packet
		}

		// Extract L_Data frame (payload starts at byte 4)
		frame := buf[4:totalLen]
		m.processFrame(frame)
	}
}

// processFrame extracts addresses from an L_Data frame:
//   - Source individual address (for Layer 4 health checks)
//   - Destination group address (for Layer 3 health checks)
func (m *BusMonitor) processFrame(frame []byte) {
	if len(frame) < minLDataFrameSize {
		return // Frame too short
	}

	// L_Data frame format:
	// Byte 0: Control byte
	// Byte 1-2: Source individual address (big-endian)
	// Byte 3-4: Destination address
	// Byte 5: Address type (bit 7) + routing counter (bits 4-6) + length (bits 0-3)
	// Byte 6+: TPDU/APDU

	sourceIA := binary.BigEndian.Uint16(frame[1:3])
	destAddr := binary.BigEndian.Uint16(frame[3:5])
	addrTypeByte := frame[5]

	// Bit 7 of byte 5: 0 = individual address, 1 = group address
	isGroupAddr := (addrTypeByte & 0x80) != 0

	// Record source individual address (skip 0.0.0 invalid/broadcast)
	if sourceIA != 0 {
		addr := knxd.FormatIndividualAddress(sourceIA)
		m.recordDevice(addr)
	}

	// Record destination group address
	if isGroupAddr && destAddr != 0 {
		// Check if this is a response (APCI in TPDU)
		// APDU starts at byte 6, APCI is in bytes 6-7
		// GroupValue_Response has APCI = 0x0040 (bits 6-7 of byte 7 = 01)
		isResponse := false
		if len(frame) >= 8 {
			apciLow := frame[7] & 0xC0 // High 2 bits indicate response type
			isResponse = (apciLow == 0x40) // GroupValue_Response
		}

		ga := knxd.FormatGroupAddress(destAddr)
		m.recordGroupAddress(ga, isResponse)
	}
}

// recordDevice updates the database with a seen device.
func (m *BusMonitor) recordDevice(addr string) {
	if m.deviceUpsertStmt == nil {
		return
	}

	now := time.Now().Unix()
	if _, err := m.deviceUpsertStmt.Exec(addr, now); err != nil {
		m.logError("recording device", err)
	}
}

// recordGroupAddress updates the database with a seen group address.
func (m *BusMonitor) recordGroupAddress(addr string, isResponse bool) {
	if m.groupUpsertStmt == nil {
		return
	}

	hasResponse := 0
	if isResponse {
		hasResponse = 1
	}

	now := time.Now().Unix()
	if _, err := m.groupUpsertStmt.Exec(addr, now, hasResponse); err != nil {
		m.logError("recording group address", err)
	}
}

// GetHealthCheckDevices returns the most recently active devices for health checks.
func (m *BusMonitor) GetHealthCheckDevices(ctx context.Context, limit int) ([]string, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT individual_address FROM knx_devices
		ORDER BY last_seen DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []string
	for rows.Next() {
		var addr string
		if err := rows.Scan(&addr); err != nil {
			return nil, err
		}
		devices = append(devices, addr)
	}

	return devices, rows.Err()
}

// GetHealthCheckGroupAddresses returns group addresses for Layer 3 health checks.
// Prioritizes addresses that have previously responded to read requests.
func (m *BusMonitor) GetHealthCheckGroupAddresses(ctx context.Context, limit int) ([]string, error) {
	// Prefer addresses with known read responses, then most recently active
	rows, err := m.db.QueryContext(ctx, `
		SELECT group_address FROM knx_group_addresses
		ORDER BY has_read_response DESC, last_seen DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []string
	for rows.Next() {
		var addr string
		if err := rows.Scan(&addr); err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}

	return addresses, rows.Err()
}

// DeviceCount returns the number of discovered devices.
func (m *BusMonitor) DeviceCount(ctx context.Context) (int, error) {
	var count int
	err := m.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM knx_devices`).Scan(&count)
	return count, err
}

// GroupAddressCount returns the number of discovered group addresses.
func (m *BusMonitor) GroupAddressCount(ctx context.Context) (int, error) {
	var count int
	err := m.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM knx_group_addresses`).Scan(&count)
	return count, err
}

// isClosed returns true if the monitor has been stopped.
func (m *BusMonitor) isClosed() bool {
	select {
	case <-m.done:
		return true
	default:
		return false
	}
}

// log logs an info message if logger is set.
func (m *BusMonitor) log(msg string, keysAndValues ...any) {
	if m.logger != nil {
		m.logger.Info(msg, keysAndValues...)
	}
}

// logError logs an error if logger is set.
func (m *BusMonitor) logError(msg string, err error) {
	if m.logger != nil {
		m.logger.Error(msg, "error", err)
	}
}
