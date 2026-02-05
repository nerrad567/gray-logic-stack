package knx

import (
	"encoding/binary"
	"fmt"
	"time"
)

// knxd protocol message types.
const (
	// EIBOpenGroupCon opens a group socket for sending/receiving group telegrams.
	// Format: type(2) + reserved(1) + write_only(1) + reserved(1)
	// This is the correct mode for bidirectional group communication via knxd.
	// Telegrams sent via this socket are forwarded to the KNX bus/backend.
	EIBOpenGroupCon uint16 = 0x0026

	// EIBGroupPacket is used to send and receive group telegrams.
	// Payload format: GA(2) + APDU (2+ bytes)
	//   Short APDU (value ≤ 0x3F): [0x00, APCI|value] (2 bytes)
	//   Long APDU: [0x00, APCI] + data (3+ bytes)
	EIBGroupPacket uint16 = 0x0027

	// EIBClose closes the knxd connection gracefully.
	EIBClose uint16 = 0x0006
)

// APCI (Application Protocol Control Information) codes.
// These define the type of group communication.
const (
	// APCIRead is a group read request (asks device for current value).
	APCIRead byte = 0x00

	// APCIResponse is a group read response (device answers read request).
	APCIResponse byte = 0x40

	// APCIWrite is a group write (sends value to devices listening on GA).
	APCIWrite byte = 0x80
)

// Telegram size constraints.
const (
	// knxdHeaderSize is the size of the knxd message header (size + type).
	knxdHeaderSize = 4
)

// Telegram represents a KNX group telegram.
//
// A telegram is the basic unit of communication on the KNX bus.
// It carries a command (read/write/response) and optional data
// to a destination group address.
type Telegram struct {
	// Source is the sender's individual address (e.g., "1.1.5").
	// Only populated for received telegrams; empty for outgoing.
	Source string

	// Destination is the target group address.
	Destination GroupAddress

	// APCI indicates the telegram type (read, response, or write).
	APCI byte

	// Data contains the DPT-encoded payload (may be empty for reads).
	Data []byte

	// Timestamp records when the telegram was received or created.
	Timestamp time.Time
}

// ParseTelegram parses a raw knxd group packet into a Telegram.
//
// The received format (EIB_OPEN_GROUPCON / EIB_GROUP_PACKET) is:
//
//	Byte 0-1: Source individual address (big-endian) — sender's address
//	Byte 2-3: Destination group address (big-endian)
//	Byte 4:   TPCI (transport control, usually 0x00)
//	Byte 5:   APCI (upper 2 bits) | data (lower 6 bits) for short frames
//	Byte 6+:  Additional data bytes for long frames
//
// Note: The receive format includes a source address prefix that the send
// format does not. This is an asymmetry in knxd's GROUPCON protocol.
//
// Parameters:
//   - data: Raw bytes from knxd EIB_GROUP_PACKET message (after header)
//
// Returns:
//   - Telegram: Parsed telegram with timestamp set to now
//   - error: ErrInvalidTelegram if parsing fails
func ParseTelegram(data []byte) (Telegram, error) {
	if len(data) < 6 { //nolint:mnd // CEMI frame minimum header length
		return Telegram{}, fmt.Errorf("%w: too short (%d bytes, need at least 6)", ErrInvalidTelegram, len(data))
	}

	// Bytes 0-1: Source individual address (big-endian uint16)
	srcRaw := binary.BigEndian.Uint16(data[0:2])
	source := formatIndividualAddress(srcRaw)

	// Bytes 2-3: Destination group address (big-endian uint16)
	destRaw := binary.BigEndian.Uint16(data[2:4])
	dest := GroupAddressFromUint16(destRaw)

	// Byte 4 = TPCI (usually 0x00 for group communication)
	// Byte 5 = APCI (upper 2 bits) | small data (lower 6 bits)
	apci := data[5] & 0xC0

	// Extract data
	var payload []byte
	if len(data) > 6 { //nolint:mnd // CEMI frame header length
		// Long frame: data bytes follow after the 6-byte header
		payload = make([]byte, len(data)-6) //nolint:mnd // CEMI frame header length
		copy(payload, data[6:])
	} else if apci == APCIWrite || apci == APCIResponse {
		// Short frame: value in lower 6 bits of APCI byte
		payload = []byte{data[5] & 0x3F}
	}
	// For APCIRead, payload stays nil

	return Telegram{
		Source:      source,
		Destination: dest,
		APCI:        apci,
		Data:        payload,
		Timestamp:   time.Now(),
	}, nil
}

// formatIndividualAddress converts a 16-bit individual address to "A.L.D" format.
// Individual addresses identify physical devices on the KNX bus.
func formatIndividualAddress(ia uint16) string {
	area := (ia >> 12) & 0x0F
	line := (ia >> 8) & 0x0F
	device := ia & 0xFF
	return fmt.Sprintf("%d.%d.%d", area, line, device)
}

// Encode encodes a Telegram to knxd wire format for EIB_OPEN_GROUPCON.
//
// The output format is suitable for sending via EIB_GROUP_PACKET on a GROUPCON socket:
//
//	Byte 0-1: Destination group address (big-endian)
//	Byte 2+:  APDU (2+ bytes): [TPCI|APCI_high, APCI_low|data, extra_data...]
//
// Short APDU (value ≤ 0x3F): GA(2) + [0x00, APCI|value] = 4 bytes total
// Long APDU (multi-byte data): GA(2) + [0x00, APCI] + data = 4+ bytes total
// Read request (no data): GA(2) + [0x00, 0x00] = 4 bytes total
//
// Returns:
//   - []byte: Encoded telegram ready for knxd
func (t Telegram) Encode() []byte {
	// Determine if data fits in APCI byte (small values ≤ 0x3F)
	smallData := len(t.Data) == 1 && t.Data[0] <= 0x3F

	if len(t.Data) == 0 || smallData {
		// Short APDU: GA(2) + [TPCI=0x00, APCI|value] = 4 bytes
		buf := make([]byte, 4) //nolint:mnd // knxd group socket header size
		binary.BigEndian.PutUint16(buf[0:2], t.Destination.ToUint16())
		buf[2] = 0x00 // TPCI
		if smallData {
			buf[3] = t.APCI | (t.Data[0] & 0x3F)
		} else {
			buf[3] = t.APCI // Read or empty write
		}
		return buf
	}

	// Long APDU: GA(2) + [TPCI=0x00, APCI] + data
	buf := make([]byte, 4+len(t.Data)) //nolint:mnd // knxd group socket header size
	binary.BigEndian.PutUint16(buf[0:2], t.Destination.ToUint16())
	buf[2] = 0x00   // TPCI
	buf[3] = t.APCI // APCI
	copy(buf[4:], t.Data)
	return buf
}

// IsWrite returns true if this is a group write telegram.
func (t Telegram) IsWrite() bool {
	return t.APCI == APCIWrite
}

// IsRead returns true if this is a group read request.
func (t Telegram) IsRead() bool {
	return t.APCI == APCIRead
}

// IsResponse returns true if this is a group read response.
func (t Telegram) IsResponse() bool {
	return t.APCI == APCIResponse
}

// String returns a human-readable representation of the telegram.
func (t Telegram) String() string {
	apciStr := "UNKNOWN"
	switch t.APCI {
	case APCIRead:
		apciStr = "READ"
	case APCIResponse:
		apciStr = "RESPONSE"
	case APCIWrite:
		apciStr = "WRITE"
	}

	return fmt.Sprintf("Telegram{GA:%s, APCI:%s, Data:%X}", t.Destination, apciStr, t.Data)
}

// NewWriteTelegram creates a new write telegram.
//
// Parameters:
//   - dest: Target group address
//   - data: DPT-encoded payload
//
// Returns:
//   - Telegram: Ready to send via knxd
func NewWriteTelegram(dest GroupAddress, data []byte) Telegram {
	return Telegram{
		Destination: dest,
		APCI:        APCIWrite,
		Data:        data,
		Timestamp:   time.Now(),
	}
}

// NewReadTelegram creates a new read request telegram.
//
// Parameters:
//   - dest: Target group address to read from
//
// Returns:
//   - Telegram: Ready to send via knxd
func NewReadTelegram(dest GroupAddress) Telegram {
	return Telegram{
		Destination: dest,
		APCI:        APCIRead,
		Data:        nil,
		Timestamp:   time.Now(),
	}
}

// EncodeKNXDMessage wraps a payload in the knxd message format.
//
// Format:
//
//	Byte 0-1: Total message size (big-endian, includes header)
//	Byte 2-3: Message type (big-endian)
//	Byte 4+:  Payload
//
// Parameters:
//   - msgType: knxd message type (e.g., EIBOpenGroupCon, EIBGroupPacket)
//   - payload: Message payload (may be nil)
//
// Returns:
//   - []byte: Complete knxd message ready to send over socket
func EncodeKNXDMessage(msgType uint16, payload []byte) []byte {
	totalSize := knxdHeaderSize + len(payload)
	buf := make([]byte, totalSize)

	// Size field = type(2) + payload length (does NOT include size field itself)
	// This matches the eibd protocol specification
	sizeField := 2 + len(payload)                           // type(2) + payload
	binary.BigEndian.PutUint16(buf[0:2], uint16(sizeField)) //nolint:gosec // bounded by small message sizes

	// Message type
	binary.BigEndian.PutUint16(buf[2:4], msgType)

	// Payload
	if len(payload) > 0 {
		copy(buf[4:], payload)
	}

	return buf
}

// ParseKNXDMessage parses a raw knxd message from the socket.
//
// Parameters:
//   - data: Raw bytes read from knxd socket
//
// Returns:
//   - msgType: The knxd message type
//   - payload: The message payload (may be empty)
//   - error: If message is malformed
func ParseKNXDMessage(data []byte) (msgType uint16, payload []byte, err error) {
	if len(data) < knxdHeaderSize {
		return 0, nil, fmt.Errorf("%w: message too short (%d bytes)", ErrInvalidTelegram, len(data))
	}

	// Validate size field (size = type(2) + payload, does NOT include size field itself)
	declaredSize := binary.BigEndian.Uint16(data[0:2])
	expectedSize := len(data) - 2 // total bytes minus the 2-byte size field
	if int(declaredSize) != expectedSize {
		return 0, nil, fmt.Errorf("%w: size mismatch (declared %d, expected %d)",
			ErrInvalidTelegram, declaredSize, expectedSize)
	}

	msgType = binary.BigEndian.Uint16(data[2:4])
	if len(data) > knxdHeaderSize {
		payload = data[knxdHeaderSize:]
	}

	return msgType, payload, nil
}
