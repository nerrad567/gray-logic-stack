package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nerrad567/gray-logic-core/internal/auth"
)

// ─── Request/Response Types ────────────────────────────────────────

type createPanelRequest struct {
	Name    string   `json:"name"`
	RoomIDs []string `json:"rooms"`
}

type createPanelResponse struct {
	Panel      *auth.Panel `json:"panel"`
	PanelToken string      `json:"panel_token"` // shown once, never again
}

type updatePanelRequest struct {
	Name *string `json:"name,omitempty"`
}

type setPanelRoomsRequest struct {
	RoomIDs []string `json:"room_ids"`
}

// panelTokenBytes is the number of random bytes for panel tokens (256-bit).
const panelTokenBytes = 32

// ─── Handlers ──────────────────────────────────────────────────────

// handleListPanels returns all registered panels.
func (s *Server) handleListPanels(w http.ResponseWriter, r *http.Request) {
	panels, err := s.panelRepo.List(r.Context())
	if err != nil {
		s.logger.Error("list panels failed", "error", err)
		writeInternalError(w, "failed to list panels")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"panels": panels,
		"count":  len(panels),
	})
}

// handleCreatePanel registers a new panel device with room assignments.
// Returns the panel token exactly once — it cannot be retrieved later.
func (s *Server) handleCreatePanel(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())

	var req createPanelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if req.Name == "" {
		writeBadRequest(w, "name is required")
		return
	}

	// Generate panel token (256-bit random)
	tokenBytes := make([]byte, panelTokenBytes)
	//nolint:errcheck // crypto/rand.Read always returns len(b) on supported platforms
	rand.Read(tokenBytes)
	rawToken := hex.EncodeToString(tokenBytes)

	panel := &auth.Panel{
		Name:      req.Name,
		TokenHash: auth.HashToken(rawToken),
		IsActive:  true,
		CreatedBy: claims.Subject,
	}

	if err := s.panelRepo.Create(r.Context(), panel); err != nil {
		s.logger.Error("create panel failed", "error", err)
		writeInternalError(w, "failed to create panel")
		return
	}

	// Set room assignments if provided
	if len(req.RoomIDs) > 0 {
		if err := s.panelRepo.SetRooms(r.Context(), panel.ID, req.RoomIDs); err != nil {
			s.logger.Error("set panel rooms failed", "error", err)
			// Panel was created but rooms failed — still return the token
		}
	}

	s.logger.Info("panel registered", "panel_id", panel.ID, "name", panel.Name, "rooms", len(req.RoomIDs), "created_by", claims.Subject)
	s.auditLog("create", "panel", panel.ID, claims.Subject, map[string]any{
		"name":       panel.Name,
		"room_count": len(req.RoomIDs),
	})

	writeJSON(w, http.StatusCreated, createPanelResponse{
		Panel:      panel,
		PanelToken: rawToken,
	})
}

// handleGetPanel returns a single panel by ID.
func (s *Server) handleGetPanel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	panel, err := s.panelRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, auth.ErrPanelNotFound) {
			writeNotFound(w, "panel not found")
			return
		}
		s.logger.Error("get panel failed", "error", err)
		writeInternalError(w, "failed to get panel")
		return
	}

	// Include room assignments
	roomIDs, err := s.panelRepo.GetRoomIDs(r.Context(), id)
	if err != nil {
		s.logger.Error("get panel rooms failed", "error", err)
		writeInternalError(w, "failed to get panel rooms")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"panel": panel,
		"rooms": roomIDs,
	})
}

// handleUpdatePanel modifies a panel's mutable fields.
func (s *Server) handleUpdatePanel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updatePanelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	panel, err := s.panelRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, auth.ErrPanelNotFound) {
			writeNotFound(w, "panel not found")
			return
		}
		s.logger.Error("get panel for update failed", "error", err)
		writeInternalError(w, "failed to update panel")
		return
	}

	if req.Name != nil {
		panel.Name = *req.Name
	}

	if err := s.panelRepo.UpdateName(r.Context(), id, panel.Name); err != nil {
		s.logger.Error("update panel failed", "error", err)
		writeInternalError(w, "failed to update panel")
		return
	}

	writeJSON(w, http.StatusOK, panel)
}

// handleDeletePanel revokes a panel (deletes token + room assignments).
func (s *Server) handleDeletePanel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims := claimsFromContext(r.Context())

	if err := s.panelRepo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, auth.ErrPanelNotFound) {
			writeNotFound(w, "panel not found")
			return
		}
		s.logger.Error("delete panel failed", "error", err)
		writeInternalError(w, "failed to delete panel")
		return
	}

	s.logger.Info("panel revoked", "panel_id", id, "deleted_by", claims.Subject)
	s.auditLog("delete", "panel", id, claims.Subject, nil)

	w.WriteHeader(http.StatusNoContent)
}

// handleGetPanelRooms returns a panel's room assignments.
func (s *Server) handleGetPanelRooms(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verify panel exists
	if _, err := s.panelRepo.GetByID(r.Context(), id); err != nil {
		if errors.Is(err, auth.ErrPanelNotFound) {
			writeNotFound(w, "panel not found")
			return
		}
		s.logger.Error("get panel for rooms failed", "error", err)
		writeInternalError(w, "failed to get panel rooms")
		return
	}

	roomIDs, err := s.panelRepo.GetRoomIDs(r.Context(), id)
	if err != nil {
		s.logger.Error("get panel rooms failed", "error", err)
		writeInternalError(w, "failed to get panel rooms")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"room_ids": roomIDs,
		"count":    len(roomIDs),
	})
}

// handleSetPanelRooms replaces all room assignments for a panel.
func (s *Server) handleSetPanelRooms(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims := claimsFromContext(r.Context())

	var req setPanelRoomsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	// Verify panel exists
	if _, err := s.panelRepo.GetByID(r.Context(), id); err != nil {
		if errors.Is(err, auth.ErrPanelNotFound) {
			writeNotFound(w, "panel not found")
			return
		}
		s.logger.Error("get panel for room update failed", "error", err)
		writeInternalError(w, "failed to set panel rooms")
		return
	}

	if err := s.panelRepo.SetRooms(r.Context(), id, req.RoomIDs); err != nil {
		s.logger.Error("set panel rooms failed", "error", err)
		writeInternalError(w, "failed to set panel rooms")
		return
	}

	s.logger.Info("panel rooms updated", "panel_id", id, "room_count", len(req.RoomIDs), "updated_by", claims.Subject)
	s.auditLog("update_rooms", "panel", id, claims.Subject, map[string]any{
		"room_count": len(req.RoomIDs),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"room_ids": req.RoomIDs,
		"count":    len(req.RoomIDs),
	})
}
