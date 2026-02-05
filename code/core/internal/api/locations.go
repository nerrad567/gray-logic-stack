package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nerrad567/gray-logic-core/internal/location"
)

// handleCreateArea creates a new area.
//
//nolint:dupl // Area and Room handlers are structurally similar but differ in types, fields, and validation
func (s *Server) handleCreateArea(w http.ResponseWriter, r *http.Request) {
	var area location.Area
	if err := json.NewDecoder(r.Body).Decode(&area); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}
	if area.ID == "" || area.Name == "" || area.SiteID == "" {
		writeBadRequest(w, "id, name, and site_id are required")
		return
	}
	if area.Slug == "" {
		area.Slug = area.ID
	}
	if area.Type == "" {
		area.Type = "floor"
	}

	if err := s.locationRepo.CreateArea(r.Context(), &area); err != nil {
		writeInternalError(w, "failed to create area")
		return
	}
	writeJSON(w, http.StatusCreated, area)
}

// handleCreateRoom creates a new room.
//
//nolint:dupl // Area and Room handlers are structurally similar but differ in types, fields, and validation
func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	var room location.Room
	if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}
	if room.ID == "" || room.Name == "" || room.AreaID == "" {
		writeBadRequest(w, "id, name, and area_id are required")
		return
	}
	if room.Slug == "" {
		room.Slug = room.ID
	}
	if room.Type == "" {
		room.Type = "other"
	}

	if err := s.locationRepo.CreateRoom(r.Context(), &room); err != nil {
		writeInternalError(w, "failed to create room")
		return
	}
	writeJSON(w, http.StatusCreated, room)
}

// handleListAreas returns all areas, with optional site_id filter.
//
//nolint:dupl // Area and Room list handlers differ in entity types and filter params
func (s *Server) handleListAreas(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if siteID := r.URL.Query().Get("site_id"); siteID != "" {
		areas, err := s.locationRepo.ListAreasBySite(ctx, siteID)
		if err != nil {
			writeInternalError(w, "failed to list areas")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"areas": areas, "count": len(areas)})
		return
	}

	areas, err := s.locationRepo.ListAreas(ctx)
	if err != nil {
		writeInternalError(w, "failed to list areas")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"areas": areas, "count": len(areas)})
}

// handleGetArea returns a single area by ID.
func (s *Server) handleGetArea(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	area, err := s.locationRepo.GetArea(r.Context(), id)
	if err != nil {
		if errors.Is(err, location.ErrAreaNotFound) {
			writeNotFound(w, "area not found")
			return
		}
		writeInternalError(w, "failed to get area")
		return
	}
	writeJSON(w, http.StatusOK, area)
}

// handleUpdateArea partially updates an area via PATCH semantics.
func (s *Server) handleUpdateArea(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	area, err := s.locationRepo.GetArea(r.Context(), id)
	if err != nil {
		if errors.Is(err, location.ErrAreaNotFound) {
			writeNotFound(w, "area not found")
			return
		}
		writeInternalError(w, "failed to get area")
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
			area.Name = name
			area.Slug = slugify(name)
		}
	}
	if v, ok := raw["type"]; ok {
		var t string
		if json.Unmarshal(v, &t) == nil && t != "" {
			area.Type = t
		}
	}
	if v, ok := raw["sort_order"]; ok {
		var order int
		if json.Unmarshal(v, &order) == nil {
			area.SortOrder = order
		}
	}

	if err := s.locationRepo.UpdateArea(r.Context(), area); err != nil { //nolint:govet // shadow: err re-declared in nested scope, checked immediately
		s.logger.Error("failed to update area", "error", err, "id", id)
		writeInternalError(w, "failed to update area")
		return
	}

	// Re-read to get updated timestamp.
	updated, err := s.locationRepo.GetArea(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusOK, area)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// handleDeleteArea deletes an area by ID.
// Returns 409 Conflict if rooms still reference this area.
func (s *Server) handleDeleteArea(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.locationRepo.DeleteArea(r.Context(), id); err != nil {
		if errors.Is(err, location.ErrAreaNotFound) {
			writeNotFound(w, "area not found")
			return
		}
		if errors.Is(err, location.ErrAreaHasRooms) {
			writeError(w, http.StatusConflict, ErrCodeConflict, err.Error())
			return
		}
		s.logger.Error("failed to delete area", "error", err, "id", id)
		writeInternalError(w, "failed to delete area")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateRoom partially updates a room via PATCH semantics.
func (s *Server) handleUpdateRoom(w http.ResponseWriter, r *http.Request) { //nolint:gocognit,gocyclo // HTTP handler: validates and patches multiple optional fields
	id := chi.URLParam(r, "id")

	room, err := s.locationRepo.GetRoom(r.Context(), id)
	if err != nil {
		if errors.Is(err, location.ErrRoomNotFound) {
			writeNotFound(w, "room not found")
			return
		}
		writeInternalError(w, "failed to get room")
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
			room.Name = name
			room.Slug = slugify(name)
		}
	}
	if v, ok := raw["type"]; ok {
		var t string
		if json.Unmarshal(v, &t) == nil && t != "" {
			room.Type = t
		}
	}
	if v, ok := raw["sort_order"]; ok {
		var order int
		if json.Unmarshal(v, &order) == nil {
			room.SortOrder = order
		}
	}
	if v, ok := raw["area_id"]; ok {
		var areaID string
		if json.Unmarshal(v, &areaID) == nil && areaID != "" {
			// Validate area exists.
			if _, err := s.locationRepo.GetArea(r.Context(), areaID); err != nil { //nolint:govet // shadow: err re-declared in nested scope, checked immediately
				writeBadRequest(w, "area_id does not exist")
				return
			}
			room.AreaID = areaID
		}
	}

	if err := s.locationRepo.UpdateRoom(r.Context(), room); err != nil { //nolint:govet // shadow: err re-declared in nested scope, checked immediately
		s.logger.Error("failed to update room", "error", err, "id", id)
		writeInternalError(w, "failed to update room")
		return
	}

	updated, err := s.locationRepo.GetRoom(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusOK, room)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// handleDeleteRoom deletes a room by ID.
// Returns 409 Conflict if devices or scenes still reference this room.
func (s *Server) handleDeleteRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	// Verify room exists before checking references
	if _, err := s.locationRepo.GetRoom(ctx, id); err != nil {
		if errors.Is(err, location.ErrRoomNotFound) {
			writeNotFound(w, "room not found")
			return
		}
		writeInternalError(w, "failed to get room")
		return
	}

	// Referential safety: check for devices assigned to this room
	devices, err := s.registry.GetDevicesByRoom(ctx, id)
	if err == nil && len(devices) > 0 {
		writeConflict(w, "room has devices: reassign or delete them first")
		return
	}

	// Referential safety: check for scenes assigned to this room
	scenes, err := s.sceneRegistry.ListScenesByRoom(ctx, id)
	if err == nil && len(scenes) > 0 {
		writeConflict(w, "room has scenes: reassign or delete them first")
		return
	}

	if err := s.locationRepo.DeleteRoom(ctx, id); err != nil {
		if errors.Is(err, location.ErrRoomNotFound) {
			writeNotFound(w, "room not found")
			return
		}
		s.logger.Error("failed to delete room", "error", err, "id", id)
		writeInternalError(w, "failed to delete room")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleListRooms returns all rooms, with optional area_id filter.
//
//nolint:dupl // Area and Room list handlers differ in entity types and filter params
func (s *Server) handleListRooms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if areaID := r.URL.Query().Get("area_id"); areaID != "" {
		rooms, err := s.locationRepo.ListRoomsByArea(ctx, areaID)
		if err != nil {
			writeInternalError(w, "failed to list rooms")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"rooms": rooms, "count": len(rooms)})
		return
	}

	rooms, err := s.locationRepo.ListRooms(ctx)
	if err != nil {
		writeInternalError(w, "failed to list rooms")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rooms": rooms, "count": len(rooms)})
}

// handleGetRoom returns a single room by ID.
func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	room, err := s.locationRepo.GetRoom(r.Context(), id)
	if err != nil {
		if errors.Is(err, location.ErrRoomNotFound) {
			writeNotFound(w, "room not found")
			return
		}
		writeInternalError(w, "failed to get room")
		return
	}
	writeJSON(w, http.StatusOK, room)
}
