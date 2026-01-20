package knx

import "errors"

// Domain errors for the KNX bridge package.
var (
	// ErrNotConnected is returned when an operation requires a connection
	// but the client is not connected to knxd.
	ErrNotConnected = errors.New("knx: not connected to knxd")

	// ErrConnectionFailed is returned when the connection to knxd fails.
	ErrConnectionFailed = errors.New("knx: connection to knxd failed")

	// ErrInvalidGroupAddress is returned when a group address string
	// cannot be parsed.
	ErrInvalidGroupAddress = errors.New("knx: invalid group address")

	// ErrInvalidDPT is returned when a datapoint type identifier is invalid.
	ErrInvalidDPT = errors.New("knx: invalid datapoint type")

	// ErrEncodingFailed is returned when encoding a value to KNX format fails.
	ErrEncodingFailed = errors.New("knx: encoding failed")

	// ErrDecodingFailed is returned when decoding KNX data to a value fails.
	ErrDecodingFailed = errors.New("knx: decoding failed")

	// ErrTelegramFailed is returned when sending a telegram to the bus fails.
	ErrTelegramFailed = errors.New("knx: telegram send failed")

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("knx: operation timed out")

	// ErrInvalidTelegram is returned when a received telegram is malformed.
	ErrInvalidTelegram = errors.New("knx: invalid telegram")
)
