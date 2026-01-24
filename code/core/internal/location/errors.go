package location

import "errors"

var (
	// ErrAreaNotFound is returned when an area ID does not exist.
	ErrAreaNotFound = errors.New("area not found")

	// ErrRoomNotFound is returned when a room ID does not exist.
	ErrRoomNotFound = errors.New("room not found")
)
