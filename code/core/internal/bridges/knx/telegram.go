package knx

import (
	"encoding/binary"
	"fmt"
	"time"
)

// knxd protocol message types.
const (
	// EIBOpenTGroup opens group communication mode with knxd.
	// Format: type(2) + group_addr(2) + flags(1)
	// After sending this, the client can send/receive group telegrams.
	// Note: This is EIB_OPEN_T_GROUP (0x0022), not EIB_OPEN_GROUPCON (0x0026).
	EIBOpenTGroup uint16 = 0x0022

	// EIBGroupPacket is used to send and receive group telegrams.
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

	// telegramMinPayloadBytes is the minimum bytes for a telegram with multi-byte data.
	telegramMinPayloadBytes = 3
)

// Telegram represents a KNX group telegram.
//
// A telegram is the basic unit of communication on the KNX bus.
// It carries a command (read/write/response) and optional data
// to a destination group address.
type Telegram struct {
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
// The expected format is:
//
//	Byte 0-1: Destination group address (big-endian)
//	Byte 2:   APCI (high nibble) + data length indicator
//	Byte 3+:  Payload data (if any)
//
// Parameters:
//   - data: Raw bytes from knxd EIB_GROUP_PACKET message (after header)
//
// Returns:
//   - Telegram: Parsed telegram with timestamp set to now
//   - error: ErrInvalidTelegram if parsing fails
func ParseTelegram(data []byte) (Telegram, error) {
	if len(data) < 2 {
		return Telegram{}, fmt.Errorf("%w: too short (%d bytes)", ErrInvalidTelegram, len(data))
	}

	// Parse destination group address (big-endian uint16)
	destRaw := binary.BigEndian.Uint16(data[0:2])
	dest := GroupAddressFromUint16(destRaw)

	// For short telegrams (1-bit values), APCI and data are in byte 2
	if len(data) == 2 {
		// Read request with no data
		return Telegram{
			Destination: dest,
			APCI:        APCIRead,
			Data:        nil,
			Timestamp:   time.Now(),
		}, nil
	}

	// Parse APCI from byte 2 (high 2 bits indicate type)
	apci := data[2] & 0xC0

	// Extract data
	var payload []byte
	if len(data) > telegramMinPayloadBytes {
		// Multi-byte payload
		payload = make([]byte, len(data)-telegramMinPayloadBytes)
		copy(payload, data[telegramMinPayloadBytes:])
	} else if apci == APCIWrite || apci == APCIResponse {
		// Single-bit value encoded in low 6 bits of APCI byte
		payload = []byte{data[2] & 0x3F}
	}

	return Telegram{
		Destination: dest,
		APCI:        apci,
		Data:        payload,
		Timestamp:   time.Now(),
	}, nil
}

// Encode encodes a Telegram to knxd wire format.
//
// The output format is suitable for sending via EIB_GROUP_PACKET:
//
//	Byte 0-1: Destination group address (big-endian)
//	Byte 2:   APCI + small data (for 1-bit values)
//	Byte 3+:  Additional data bytes (if needed)
//
// Returns:
//   - []byte: Encoded telegram ready for knxd
func (t Telegram) Encode() []byte {
	// Determine if data fits in APCI byte (small values ≤ 0x3F)
	smallData := len(t.Data) == 1 && t.Data[0] <= 0x3F

	// Calculate size: 2 (GA) + 1 (APCI) + extra data bytes
	size := 3
	if len(t.Data) > 0 && !smallData {
		size = 3 + len(t.Data)
	}

	buf := make([]byte, size)

	// Destination group address (big-endian)
	binary.BigEndian.PutUint16(buf[0:2], t.Destination.ToUint16())

	// APCI byte with optional small data
	if len(t.Data) == 0 {
		// Read request or empty write
		buf[2] = t.APCI
	} else if smallData {
		// Small value (6 bits) fits in APCI byte
		buf[2] = t.APCI | (t.Data[0] & 0x3F)
	} else {
		// Data goes in separate bytes
		buf[2] = t.APCI
		copy(buf[3:], t.Data)
	}

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
//   - msgType: knxd message type (e.g., EIBOpenTGroup, EIBGroupPacket)
//   - payload: Message payload (may be nil)
//
// Returns:
//   - []byte: Complete knxd message ready to send over socket
func EncodeKNXDMessage(msgType uint16, payload []byte) []byte {
	totalSize := knxdHeaderSize + len(payload)
	buf := make([]byte, totalSize)

	// Size field = type(2) + payload length (does NOT include size field itself)
	// This matches the eibd protocol specification
	sizeField := 2 + len(payload) // type(2) + payload
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
