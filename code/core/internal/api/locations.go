package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nerrad567/gray-logic-core/internal/location"
)

// handleCreateArea creates a new area.
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

// handleListRooms returns all rooms, with optional area_id filter.
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
