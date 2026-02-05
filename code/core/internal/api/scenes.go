package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nerrad567/gray-logic-core/internal/automation"
)

// maxQueryParamLen limits query parameter length to prevent DoS via oversized URL params.
const maxQueryParamLen = 100

// handleListScenes returns all scenes, with optional query filters.
//
// Query parameters:
//   - room_id: filter by room
//   - area_id: filter by area
//   - category: filter by category (comfort, entertainment, etc.)
func (s *Server) handleListScenes(w http.ResponseWriter, r *http.Request) { //nolint:gocognit // HTTP handler: filter parsing + query + response assembly
	ctx := r.Context()

	if roomID := r.URL.Query().Get("room_id"); roomID != "" {
		if len(roomID) > maxQueryParamLen {
			writeBadRequest(w, "room_id exceeds maximum length")
			return
		}
		scenes, err := s.sceneRegistry.ListScenesByRoom(ctx, roomID)
		if err != nil {
			writeInternalError(w, "failed to list scenes")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"scenes": scenes, "count": len(scenes)})
		return
	}

	if areaID := r.URL.Query().Get("area_id"); areaID != "" {
		if len(areaID) > maxQueryParamLen {
			writeBadRequest(w, "area_id exceeds maximum length")
			return
		}
		scenes, err := s.sceneRegistry.ListScenesByArea(ctx, areaID)
		if err != nil {
			writeInternalError(w, "failed to list scenes")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"scenes": scenes, "count": len(scenes)})
		return
	}

	if category := r.URL.Query().Get("category"); category != "" {
		if len(category) > maxQueryParamLen {
			writeBadRequest(w, "category exceeds maximum length")
			return
		}
		cat := automation.Category(category)
		valid := false
		for _, c := range automation.AllCategories() {
			if c == cat {
				valid = true
				break
			}
		}
		if !valid {
			writeBadRequest(w, "invalid category")
			return
		}
		scenes, err := s.sceneRegistry.ListScenesByCategory(ctx, cat)
		if err != nil {
			writeInternalError(w, "failed to list scenes")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"scenes": scenes, "count": len(scenes)})
		return
	}

	// No filter: return all scenes
	scenes, err := s.sceneRegistry.ListScenes(ctx)
	if err != nil {
		writeInternalError(w, "failed to list scenes")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scenes": scenes, "count": len(scenes)})
}

// handleGetScene returns a single scene by ID.
func (s *Server) handleGetScene(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" || len(id) > maxQueryParamLen {
		writeBadRequest(w, "invalid scene ID")
		return
	}

	scene, err := s.sceneRegistry.GetScene(r.Context(), id)
	if err != nil {
		if errors.Is(err, automation.ErrSceneNotFound) {
			writeNotFound(w, "scene not found")
			return
		}
		writeInternalError(w, "failed to get scene")
		return
	}

	writeJSON(w, http.StatusOK, scene)
}

// handleCreateScene creates a new scene.
func (s *Server) handleCreateScene(w http.ResponseWriter, r *http.Request) {
	var scene automation.Scene
	if err := json.NewDecoder(r.Body).Decode(&scene); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if err := s.sceneRegistry.CreateScene(r.Context(), &scene); err != nil {
		if errors.Is(err, automation.ErrInvalidScene) || errors.Is(err, automation.ErrInvalidName) ||
			errors.Is(err, automation.ErrNoActions) || errors.Is(err, automation.ErrInvalidAction) {
			writeBadRequest(w, err.Error())
			return
		}
		if errors.Is(err, automation.ErrSceneExists) {
			writeError(w, http.StatusConflict, ErrCodeConflict, err.Error())
			return
		}
		writeInternalError(w, "failed to create scene")
		return
	}

	writeJSON(w, http.StatusCreated, scene)
}

// handleUpdateScene partially updates a scene.
func (s *Server) handleUpdateScene(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" || len(id) > maxQueryParamLen {
		writeBadRequest(w, "invalid scene ID")
		return
	}

	// Get existing scene
	existing, err := s.sceneRegistry.GetScene(r.Context(), id)
	if err != nil {
		if errors.Is(err, automation.ErrSceneNotFound) {
			writeNotFound(w, "scene not found")
			return
		}
		writeInternalError(w, "failed to get scene")
		return
	}

	// Decode partial update onto existing scene
	if err := json.NewDecoder(r.Body).Decode(existing); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}
	existing.ID = id // Ensure ID cannot be changed

	if err := s.sceneRegistry.UpdateScene(r.Context(), existing); err != nil {
		if errors.Is(err, automation.ErrInvalidScene) || errors.Is(err, automation.ErrInvalidName) ||
			errors.Is(err, automation.ErrNoActions) || errors.Is(err, automation.ErrInvalidAction) {
			writeBadRequest(w, err.Error())
			return
		}
		if errors.Is(err, automation.ErrSceneExists) {
			writeError(w, http.StatusConflict, ErrCodeConflict, err.Error())
			return
		}
		writeInternalError(w, "failed to update scene")
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// handleDeleteScene removes a scene by ID.
func (s *Server) handleDeleteScene(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" || len(id) > maxQueryParamLen {
		writeBadRequest(w, "invalid scene ID")
		return
	}

	if err := s.sceneRegistry.DeleteScene(r.Context(), id); err != nil {
		if errors.Is(err, automation.ErrSceneNotFound) {
			writeNotFound(w, "scene not found")
			return
		}
		writeInternalError(w, "failed to delete scene")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// activateRequest is the request body for POST /scenes/{id}/activate.
type activateRequest struct {
	TriggerType   string `json:"trigger_type"`
	TriggerSource string `json:"trigger_source"`
}

// handleActivateScene activates a scene and returns the execution ID.
// This is an asynchronous operation — MQTT commands are published to bridges
// and the response is 202 Accepted. Device state changes arrive via WebSocket.
func (s *Server) handleActivateScene(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" || len(id) > maxQueryParamLen {
		writeBadRequest(w, "invalid scene ID")
		return
	}

	// Decode optional activation parameters
	var req activateRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "invalid JSON body")
			return
		}
	}

	// Default trigger type to "manual" if not specified
	triggerType := req.TriggerType
	if triggerType == "" {
		triggerType = "manual"
	}
	triggerSource := req.TriggerSource
	if triggerSource == "" {
		triggerSource = "api"
	}

	executionID, err := s.sceneEngine.ActivateScene(r.Context(), id, triggerType, triggerSource)
	if err != nil {
		if errors.Is(err, automation.ErrSceneNotFound) {
			writeNotFound(w, "scene not found")
			return
		}
		if errors.Is(err, automation.ErrSceneDisabled) {
			writeError(w, http.StatusConflict, ErrCodeConflict, "scene is disabled")
			return
		}
		if errors.Is(err, automation.ErrMQTTUnavailable) {
			writeInternalError(w, "MQTT not available")
			return
		}
		writeInternalError(w, "failed to activate scene")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"execution_id": executionID,
		"status":       "accepted",
		"message":      "scene activation started, state updates will follow via WebSocket",
	})
}

// handleListSceneExecutions returns execution history for a scene.
func (s *Server) handleListSceneExecutions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" || len(id) > maxQueryParamLen {
		writeBadRequest(w, "invalid scene ID")
		return
	}

	// Verify scene exists
	if _, err := s.sceneRegistry.GetScene(r.Context(), id); err != nil {
		if errors.Is(err, automation.ErrSceneNotFound) {
			writeNotFound(w, "scene not found")
			return
		}
		writeInternalError(w, "failed to get scene")
		return
	}

	const maxExecutions = 50
	executions, err := s.sceneRepo.ListExecutions(r.Context(), id, maxExecutions)
	if err != nil {
		writeInternalError(w, "failed to list executions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"executions": executions, "count": len(executions)})
}
