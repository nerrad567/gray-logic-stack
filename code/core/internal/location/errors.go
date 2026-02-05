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

	// ErrZoneNotFound is returned when an infrastructure zone ID does not exist.
	ErrZoneNotFound = errors.New("infrastructure zone not found")

	// ErrZoneExists is returned when a zone with the same slug already exists for a site.
	ErrZoneExists = errors.New("infrastructure zone already exists")

	// ErrRoomAlreadyInZoneDomain is returned when assigning a room to a zone
	// but the room is already in another zone of the same domain.
	ErrRoomAlreadyInZoneDomain = errors.New("room already assigned to a zone in this domain")
)
