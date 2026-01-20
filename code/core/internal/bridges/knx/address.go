package knx

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// GroupAddress represents a KNX group address in 3-level format.
//
// Format: Main/Middle/Sub
//   - Main:   0-31 (5 bits)
//   - Middle: 0-7  (3 bits)
//   - Sub:    0-255 (8 bits)
//
// Total: 16 bits (0x0000 - 0xFFFF)
type GroupAddress struct {
	Main   uint8
	Middle uint8
	Sub    uint8
}

// Group address limits per KNX specification.
const (
	maxMain   = 31
	maxMiddle = 7
	maxSub    = 255

	// gaLevelCount is the number of levels in a 3-level group address.
	gaLevelCount = 3

	// Bit masks for extracting group address parts from uint16.
	gaMainMask   = 0x1F // 5 bits
	gaMiddleMask = 0x07 // 3 bits
	gaSubMask    = 0xFF // 8 bits
)

// ParseGroupAddress parses a 3-level group address string.
//
// Accepts formats:
//   - "1/2/3" — Standard 3-level format
//
// Parameters:
//   - s: Group address string
//
// Returns:
//   - GroupAddress: Parsed address
//   - error: ErrInvalidGroupAddress if parsing fails
//
// Example:
//
//	addr, err := ParseGroupAddress("1/2/3")
//	if err != nil {
//	    return err
//	}
func ParseGroupAddress(s string) (GroupAddress, error) {
	parts := strings.Split(s, "/")
	if len(parts) != gaLevelCount {
		return GroupAddress{}, fmt.Errorf("%w: expected 3-level format (main/middle/sub), got %q", ErrInvalidGroupAddress, s)
	}

	main, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil || main > maxMain {
		return GroupAddress{}, fmt.Errorf("%w: main group must be 0-%d, got %q", ErrInvalidGroupAddress, maxMain, parts[0])
	}

	middle, err := strconv.ParseUint(parts[1], 10, 8)
	if err != nil || middle > maxMiddle {
		return GroupAddress{}, fmt.Errorf("%w: middle group must be 0-%d, got %q", ErrInvalidGroupAddress, maxMiddle, parts[1])
	}

	sub, err := strconv.ParseUint(parts[2], 10, 8)
	if err != nil || sub > maxSub {
		return GroupAddress{}, fmt.Errorf("%w: sub group must be 0-%d, got %q", ErrInvalidGroupAddress, maxSub, parts[2])
	}

	return GroupAddress{
		Main:   uint8(main),
		Middle: uint8(middle),
		Sub:    uint8(sub),
	}, nil
}

// String returns the group address in 3-level format.
//
// Example: "1/2/3"
func (ga GroupAddress) String() string {
	return fmt.Sprintf("%d/%d/%d", ga.Main, ga.Middle, ga.Sub)
}

// ToUint16 converts the group address to a 16-bit integer.
//
// Layout: MMMM MSSS SSSS SSSS
//   - M = Main (5 bits)
//   - S = Middle (3 bits) + Sub (8 bits)
func (ga GroupAddress) ToUint16() uint16 {
	return uint16(ga.Main)<<11 | uint16(ga.Middle)<<8 | uint16(ga.Sub)
}

// GroupAddressFromUint16 creates a GroupAddress from a 16-bit integer.
//
// Parameters:
//   - value: 16-bit group address value
//
// Returns:
//   - GroupAddress: Decoded address
func GroupAddressFromUint16(value uint16) GroupAddress {
	// Bit masks ensure values fit in uint8 (no overflow possible).
	return GroupAddress{
		Main:   uint8((value >> 11) & gaMainMask),  //nolint:gosec // masked to 5 bits (0-31)
		Middle: uint8((value >> 8) & gaMiddleMask), //nolint:gosec // masked to 3 bits (0-7)
		Sub:    uint8(value & gaSubMask),           //nolint:gosec // masked to 8 bits (0-255)
	}
}

// URLEncode returns the group address as a URL-encoded string.
//
// This is used in MQTT topics where "/" is a level separator.
//
// Example: "1/2/3" → "1%2F2%2F3"
func (ga GroupAddress) URLEncode() string {
	return url.PathEscape(ga.String())
}

// ParseGroupAddressFromURL parses a URL-encoded group address.
//
// Parameters:
//   - encoded: URL-encoded group address (e.g., "1%2F2%2F3")
//
// Returns:
//   - GroupAddress: Parsed address
//   - error: If decoding or parsing fails
func ParseGroupAddressFromURL(encoded string) (GroupAddress, error) {
	decoded, err := url.PathUnescape(encoded)
	if err != nil {
		return GroupAddress{}, fmt.Errorf("%w: URL decode failed: %w", ErrInvalidGroupAddress, err)
	}
	return ParseGroupAddress(decoded)
}

// IsValid returns true if the group address values are within valid ranges.
func (ga GroupAddress) IsValid() bool {
	return ga.Main <= maxMain && ga.Middle <= maxMiddle && ga.Sub <= maxSub
}
