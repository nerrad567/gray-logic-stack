package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nerrad567/gray-logic-core/internal/location"
)

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
