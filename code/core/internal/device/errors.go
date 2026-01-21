package device

import "errors"

// Domain errors for the device package.
//
// These errors can be checked using errors.Is() for error handling:
//
//	if errors.Is(err, device.ErrDeviceNotFound) {
//	    // handle not found case
//	}
var (
	// ErrDeviceNotFound is returned when a device ID does not exist.
	ErrDeviceNotFound = errors.New("device: not found")

	// ErrDeviceExists is returned when creating a device with an ID that already exists.
	ErrDeviceExists = errors.New("device: already exists")

	// ErrInvalidDevice is returned when device validation fails.
	ErrInvalidDevice = errors.New("device: invalid")

	// ErrInvalidProtocol is returned when a protocol value is not recognised.
	ErrInvalidProtocol = errors.New("device: invalid protocol")

	// ErrInvalidDomain is returned when a domain value is not recognised.
	ErrInvalidDomain = errors.New("device: invalid domain")

	// ErrInvalidDeviceType is returned when a device type is not recognised.
	ErrInvalidDeviceType = errors.New("device: invalid type")

	// ErrInvalidCapability is returned when a capability is not recognised.
	ErrInvalidCapability = errors.New("device: invalid capability")

	// ErrInvalidAddress is returned when address validation fails.
	ErrInvalidAddress = errors.New("device: invalid address")

	// ErrInvalidState is returned when state validation fails.
	ErrInvalidState = errors.New("device: invalid state")

	// ErrInvalidName is returned when a device name is empty or too long.
	ErrInvalidName = errors.New("device: invalid name")

	// ErrInvalidSlug is returned when a slug format is invalid.
	ErrInvalidSlug = errors.New("device: invalid slug")

	// ErrRoomNotFound is returned when a referenced room does not exist.
	ErrRoomNotFound = errors.New("device: room not found")

	// ErrAreaNotFound is returned when a referenced area does not exist.
	ErrAreaNotFound = errors.New("device: area not found")

	// ErrGatewayNotFound is returned when a referenced gateway device does not exist.
	ErrGatewayNotFound = errors.New("device: gateway not found")
)
