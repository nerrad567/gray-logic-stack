package device

import "errors"

var (
	// ErrGroupNotFound is returned when a device group ID does not exist.
	ErrGroupNotFound = errors.New("device group: not found")

	// ErrGroupExists is returned when a device group with the same ID or slug already exists.
	ErrGroupExists = errors.New("device group: already exists")
)
