package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nerrad567/gray-logic-core/internal/location"
)

// handleListZones returns infrastructure zones, optionally filtered by domain.
//
// GET /zones
// GET /zones?domain=climate
// Response: {"zones": [...], "count": N}
func (s *Server) handleListZones(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	domain := r.URL.Query().Get("domain")

	if domain != "" {
		if !location.ValidZoneDomain(domain) {
			writeBadRequest(w, "invalid zone domain")
			return
		}
		zones, err := s.zoneRepo.ListZonesByDomain(ctx, s.siteID, location.ZoneDomain(domain))
		if err != nil {
			s.logger.Error("failed to list zones by domain", "error", err, "domain", domain)
			writeInternalError(w, "failed to list zones")
			return
		}
		if zones == nil {
			zones = []location.InfrastructureZone{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"zones": zones, "count": len(zones)})
		return
	}

	zones, err := s.zoneRepo.ListZones(ctx, s.siteID)
	if err != nil {
		s.logger.Error("failed to list zones", "error", err)
		writeInternalError(w, "failed to list zones")
		return
	}
	if zones == nil {
		zones = []location.InfrastructureZone{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"zones": zones, "count": len(zones)})
}

// handleCreateZone creates a new infrastructure zone.
//
// POST /zones
// Body: InfrastructureZone JSON (domain in body)
// Response: 201 Created with the created zone
func (s *Server) handleCreateZone(w http.ResponseWriter, r *http.Request) {
	var zone location.InfrastructureZone
	if err := json.NewDecoder(r.Body).Decode(&zone); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if zone.Name == "" || len(zone.Name) > 128 { //nolint:mnd // max name length
		writeBadRequest(w, "name is required and must be at most 128 characters")
		return
	}
	if zone.Domain == "" {
		writeBadRequest(w, "domain is required")
		return
	}
	if !location.ValidZoneDomain(string(zone.Domain)) {
		writeBadRequest(w, "invalid zone domain")
		return
	}

	// Default site_id to the configured site
	if zone.SiteID == "" {
		zone.SiteID = s.siteID
	}

	if err := s.zoneRepo.CreateZone(r.Context(), &zone); err != nil {
		if errors.Is(err, location.ErrZoneExists) {
			writeConflict(w, "a zone with this slug already exists for this site")
			return
		}
		s.logger.Error("failed to create zone", "error", err)
		writeInternalError(w, "failed to create zone")
		return
	}

	writeJSON(w, http.StatusCreated, zone)
}

// handleGetZone returns a single infrastructure zone by ID, including its room list.
//
// GET /zones/{id}
// Response: zone JSON with embedded "rooms" array
func (s *Server) handleGetZone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	zone, err := s.zoneRepo.GetZone(r.Context(), id)
	if err != nil {
		if errors.Is(err, location.ErrZoneNotFound) {
			writeNotFound(w, "zone not found")
			return
		}
		s.logger.Error("failed to get zone", "error", err, "id", id)
		writeInternalError(w, "failed to get zone")
		return
	}

	// Also fetch the zone's rooms
	rooms, err := s.zoneRepo.GetZoneRooms(r.Context(), id)
	if err != nil {
		s.logger.Error("failed to get zone rooms", "error", err, "id", id)
		writeInternalError(w, "failed to get zone rooms")
		return
	}
	if rooms == nil {
		rooms = []location.Room{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"zone":  zone,
		"rooms": rooms,
	})
}

// handleUpdateZone partially updates an infrastructure zone via PATCH semantics.
//
// PATCH /zones/{id}
// Body: partial zone fields (name, settings, sort_order)
// Response: updated zone JSON
func (s *Server) handleUpdateZone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	zone, err := s.zoneRepo.GetZone(r.Context(), id)
	if err != nil {
		if errors.Is(err, location.ErrZoneNotFound) {
			writeNotFound(w, "zone not found")
			return
		}
		s.logger.Error("failed to get zone", "error", err, "id", id)
		writeInternalError(w, "failed to get zone")
		return
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil { //nolint:govet // shadow: err re-declared in nested scope, checked immediately
		writeBadRequest(w, "invalid JSON body")
		return
	}
	if len(raw) == 0 {
		writeBadRequest(w, "no fields to update")
		return
	}

	if v, ok := raw["name"]; ok {
		var name string
		if json.Unmarshal(v, &name) == nil && name != "" {
			if len(name) > 128 { //nolint:mnd // max name length
				writeBadRequest(w, "name must be at most 128 characters")
				return
			}
			zone.Name = name
			zone.Slug = slugify(name)
		}
	}
	if v, ok := raw["settings"]; ok {
		var settings location.Settings
		if json.Unmarshal(v, &settings) == nil {
			zone.Settings = settings
		}
	}
	if v, ok := raw["sort_order"]; ok {
		var order int
		if json.Unmarshal(v, &order) == nil {
			zone.SortOrder = order
		}
	}

	if err := s.zoneRepo.UpdateZone(r.Context(), zone); err != nil { //nolint:govet // shadow: err re-declared in nested scope, checked immediately
		s.logger.Error("failed to update zone", "error", err, "id", id)
		writeInternalError(w, "failed to update zone")
		return
	}

	// Re-read to get updated timestamp
	updated, err := s.zoneRepo.GetZone(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusOK, zone)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// handleDeleteZone removes an infrastructure zone by ID.
// Room assignments are cascade-deleted by the FK constraint.
//
// DELETE /zones/{id}
// Response: 204 No Content
func (s *Server) handleDeleteZone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.zoneRepo.DeleteZone(r.Context(), id); err != nil {
		if errors.Is(err, location.ErrZoneNotFound) {
			writeNotFound(w, "zone not found")
			return
		}
		s.logger.Error("failed to delete zone", "error", err, "id", id)
		writeInternalError(w, "failed to delete zone")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleSetZoneRooms replaces the room membership for a zone.
// Enforces one-zone-per-domain: a room cannot belong to two climate zones.
//
// PUT /zones/{id}/rooms
// Body: {"room_ids": ["room-1", "room-2"]}
// Response: {"zone_id": "zone-1", "room_ids": [...], "count": N}
func (s *Server) handleSetZoneRooms(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verify zone exists
	if _, err := s.zoneRepo.GetZone(r.Context(), id); err != nil {
		if errors.Is(err, location.ErrZoneNotFound) {
			writeNotFound(w, "zone not found")
			return
		}
		s.logger.Error("failed to get zone", "error", err, "id", id)
		writeInternalError(w, "failed to get zone")
		return
	}

	var body struct {
		RoomIDs []string `json:"room_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if err := s.zoneRepo.SetZoneRooms(r.Context(), id, body.RoomIDs); err != nil {
		if errors.Is(err, location.ErrRoomAlreadyInZoneDomain) {
			writeConflict(w, err.Error())
			return
		}
		s.logger.Error("failed to set zone rooms", "error", err, "zone_id", id)
		writeInternalError(w, "failed to set zone rooms")
		return
	}

	// Read back the room IDs
	roomIDs, err := s.zoneRepo.GetZoneRoomIDs(r.Context(), id)
	if err != nil {
		s.logger.Error("failed to get zone room IDs", "error", err, "zone_id", id)
		writeInternalError(w, "failed to get zone rooms")
		return
	}
	if roomIDs == nil {
		roomIDs = []string{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"zone_id":  id,
		"room_ids": roomIDs,
		"count":    len(roomIDs),
	})
}

// handleGetZoneRooms returns the rooms assigned to a zone.
//
// GET /zones/{id}/rooms
// Response: {"zone_id": "zone-1", "rooms": [...], "count": N}
func (s *Server) handleGetZoneRooms(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verify zone exists
	if _, err := s.zoneRepo.GetZone(r.Context(), id); err != nil {
		if errors.Is(err, location.ErrZoneNotFound) {
			writeNotFound(w, "zone not found")
			return
		}
		s.logger.Error("failed to get zone", "error", err, "id", id)
		writeInternalError(w, "failed to get zone")
		return
	}

	rooms, err := s.zoneRepo.GetZoneRooms(r.Context(), id)
	if err != nil {
		s.logger.Error("failed to get zone rooms", "error", err, "zone_id", id)
		writeInternalError(w, "failed to get zone rooms")
		return
	}
	if rooms == nil {
		rooms = []location.Room{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"zone_id": id,
		"rooms":   rooms,
		"count":   len(rooms),
	})
}
