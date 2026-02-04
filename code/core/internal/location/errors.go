package location

import "errors"

var (
	// ErrSiteNotFound is returned when no site record exists.
	ErrSiteNotFound = errors.New("site not found")

	// ErrAreaNotFound is returned when an area ID does not exist.
	ErrAreaNotFound = errors.New("area not found")

	// ErrAreaHasRooms is returned when trying to delete an area that still has rooms.
	ErrAreaHasRooms = errors.New("area has rooms: delete rooms first")

	// ErrRoomNotFound is returned when a room ID does not exist.
	ErrRoomNotFound = errors.New("room not found")
)
