package api

import (
	"context"
	"net/http"
)

// hierarchyRoom is the room representation within the hierarchy response.
type hierarchyRoom struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	SortOrder   int               `json:"sort_order"`
	DeviceCount int               `json:"device_count"`
	SceneCount  int               `json:"scene_count"`
	Zones       map[string]string `json:"zones,omitempty"`
}

// hierarchyArea is the area representation within the hierarchy response.
type hierarchyArea struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	SortOrder int             `json:"sort_order"`
	Rooms     []hierarchyRoom `json:"rooms"`
	RoomCount int             `json:"room_count"`
}

// hierarchySite is the top-level site representation in the hierarchy response.
type hierarchySite struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	ModeCurrent string          `json:"mode_current"`
	Areas       []hierarchyArea `json:"areas"`
	AreaCount   int             `json:"area_count"`
}

// handleGetHierarchy returns the full site → areas → rooms tree in a single call.
// Room-scoped users see only their granted rooms; areas with zero visible rooms are omitted.
//
// GET /hierarchy
// Response: {"site": {..., "areas": [{..., "rooms": [...]}]}}
func (s *Server) handleGetHierarchy(w http.ResponseWriter, r *http.Request) { //nolint:gocognit,gocyclo // hierarchy builder: aggregates data from multiple sources into a single response tree
	ctx := r.Context()
	scope := requestRoomScope(ctx)

	// Get the site
	site, err := s.locationRepo.GetAnySite(ctx)
	if err != nil {
		writeNotFound(w, "no site configured")
		return
	}

	// Get all areas for this site
	areas, err := s.locationRepo.ListAreasBySite(ctx, site.ID)
	if err != nil {
		s.logger.Error("failed to list areas for hierarchy", "error", err)
		writeInternalError(w, "failed to build hierarchy")
		return
	}

	// Build device and scene counts per room
	deviceCounts, sceneCounts := s.buildRoomCounts(ctx)

	// Build zone membership per room (room_id → {domain: zone_id})
	roomZones := s.buildRoomZoneMap(ctx)

	// Build the hierarchy tree
	var hierarchyAreas []hierarchyArea
	for _, area := range areas {
		rooms, roomErr := s.locationRepo.ListRoomsByArea(ctx, area.ID)
		if roomErr != nil {
			s.logger.Error("failed to list rooms for area", "error", roomErr, "area_id", area.ID)
			continue
		}

		var visibleRooms []hierarchyRoom
		for _, room := range rooms {
			// Room scope filtering: skip rooms not in the user's access list
			if scope != nil && !scope.CanAccessRoom(room.ID) {
				continue
			}

			hr := hierarchyRoom{
				ID:          room.ID,
				Name:        room.Name,
				Type:        room.Type,
				SortOrder:   room.SortOrder,
				DeviceCount: deviceCounts[room.ID],
				SceneCount:  sceneCounts[room.ID],
			}
			if zones, ok := roomZones[room.ID]; ok {
				hr.Zones = zones
			}
			visibleRooms = append(visibleRooms, hr)
		}

		// Omit areas with zero visible rooms for scoped users
		if scope != nil && len(visibleRooms) == 0 {
			continue
		}
		if visibleRooms == nil {
			visibleRooms = []hierarchyRoom{}
		}

		hierarchyAreas = append(hierarchyAreas, hierarchyArea{
			ID:        area.ID,
			Name:      area.Name,
			Type:      area.Type,
			SortOrder: area.SortOrder,
			Rooms:     visibleRooms,
			RoomCount: len(visibleRooms),
		})
	}

	if hierarchyAreas == nil {
		hierarchyAreas = []hierarchyArea{}
	}

	result := hierarchySite{
		ID:          site.ID,
		Name:        site.Name,
		ModeCurrent: site.ModeCurrent,
		Areas:       hierarchyAreas,
		AreaCount:   len(hierarchyAreas),
	}

	writeJSON(w, http.StatusOK, map[string]any{"site": result})
}

// buildRoomCounts returns maps of room_id → device count and room_id → scene count.
func (s *Server) buildRoomCounts(ctx context.Context) (map[string]int, map[string]int) {
	deviceCounts := make(map[string]int)
	sceneCounts := make(map[string]int)

	// Count devices per room from the cache
	devices, err := s.registry.ListDevices(ctx)
	if err == nil {
		for _, d := range devices {
			if d.RoomID != nil && *d.RoomID != "" {
				deviceCounts[*d.RoomID]++
			}
		}
	}

	// Count scenes per room from the registry
	scenes, err := s.sceneRegistry.ListScenes(ctx)
	if err == nil {
		for _, sc := range scenes {
			if sc.RoomID != nil && *sc.RoomID != "" {
				sceneCounts[*sc.RoomID]++
			}
		}
	}

	return deviceCounts, sceneCounts
}

// buildRoomZoneMap returns a map of room_id → {domain: zone_id} for all rooms.
func (s *Server) buildRoomZoneMap(ctx context.Context) map[string]map[string]string {
	roomZones := make(map[string]map[string]string)

	if s.zoneRepo == nil {
		return roomZones
	}

	// Get all rooms, then look up zones for each
	rooms, err := s.locationRepo.ListRooms(ctx)
	if err != nil {
		return roomZones
	}

	for _, room := range rooms {
		zones, zoneErr := s.zoneRepo.GetZonesForRoom(ctx, room.ID)
		if zoneErr != nil || len(zones) == 0 {
			continue
		}
		zoneMap := make(map[string]string, len(zones))
		for _, z := range zones {
			zoneMap[string(z.Domain)] = z.ID
		}
		roomZones[room.ID] = zoneMap
	}

	return roomZones
}
