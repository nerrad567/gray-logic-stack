package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nerrad567/gray-logic-core/internal/auth"
	"github.com/nerrad567/gray-logic-core/internal/automation"
)

// maxQueryParamLen limits query parameter length to prevent DoS via oversized URL params.
const (
	maxQueryParamLen            = 100
	sceneNotAccessibleMessage   = "scene not in accessible rooms"
	sceneManageForbiddenMessage = "scene management not permitted in this room"
)

// handleListScenes returns all scenes, with optional query filters.
//
// Query parameters:
//   - room_id: filter by room
//   - area_id: filter by area
//   - category: filter by category (comfort, entertainment, etc.)
func (s *Server) handleListScenes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := requestRoomScope(ctx)
	handleScopedList(w, scope, "scenes", "failed to list scenes", func() ([]automation.Scene, string, error) {
		return s.listScenesForRequest(ctx, r, scope)
	})
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

	scope := requestRoomScope(r.Context())
	if denied, message := sceneAccessDenied(scope, derefString(scene.RoomID)); denied {
		writeForbidden(w, message)
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

	scope := requestRoomScope(r.Context())
	roomID := derefString(scene.RoomID)
	if denied, message := sceneManageDenied(scope, roomID); denied {
		writeForbidden(w, message)
		return
	}

	// Validate action device_ids are within the user's accessible scope.
	if err := s.validateActionDeviceScope(r.Context(), scope, scene.Actions); err != nil {
		writeForbidden(w, err.Error())
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

	scope := requestRoomScope(r.Context())
	if denied, message := sceneAccessDenied(scope, derefString(existing.RoomID)); denied {
		writeForbidden(w, message)
		return
	}

	// Decode partial update onto existing scene
	if err := json.NewDecoder(r.Body).Decode(existing); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}
	existing.ID = id // Ensure ID cannot be changed

	roomID := derefString(existing.RoomID)
	if denied, message := sceneManageDenied(scope, roomID); denied {
		writeForbidden(w, message)
		return
	}

	// Validate action device_ids are within the user's accessible scope.
	if err := s.validateActionDeviceScope(r.Context(), scope, existing.Actions); err != nil {
		writeForbidden(w, err.Error())
		return
	}

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

	scene, err := s.sceneRegistry.GetScene(r.Context(), id)
	if err != nil {
		if errors.Is(err, automation.ErrSceneNotFound) {
			writeNotFound(w, "scene not found")
			return
		}
		writeInternalError(w, "failed to get scene")
		return
	}

	scope := requestRoomScope(r.Context())
	roomID := derefString(scene.RoomID)
	if denied, message := sceneManageDenied(scope, roomID); denied {
		writeForbidden(w, message)
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

	scope := requestRoomScope(r.Context())
	scene, err := s.sceneRegistry.GetScene(r.Context(), id)
	if err != nil {
		if errors.Is(err, automation.ErrSceneNotFound) {
			writeNotFound(w, "scene not found")
			return
		}
		writeInternalError(w, "failed to get scene")
		return
	}
	if denied, message := sceneAccessDenied(scope, derefString(scene.RoomID)); denied {
		writeForbidden(w, message)
		return
	}

	// Decode optional activation parameters
	req, err := parseActivateRequest(r)
	if err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	// Default trigger type to "manual" if not specified
	triggerType, triggerSource := normaliseTrigger(req)

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
	scene, err := s.sceneRegistry.GetScene(r.Context(), id)
	if err != nil {
		if errors.Is(err, automation.ErrSceneNotFound) {
			writeNotFound(w, "scene not found")
			return
		}
		writeInternalError(w, "failed to get scene")
		return
	}

	scope := requestRoomScope(r.Context())
	if denied, message := sceneAccessDenied(scope, derefString(scene.RoomID)); denied {
		writeForbidden(w, message)
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

// listScenesForRequest returns scenes matching query filters and room scope.
func (s *Server) listScenesForRequest(ctx context.Context, r *http.Request, scope *auth.RoomScope) ([]automation.Scene, string, error) {
	query := r.URL.Query()

	filters := []sceneFilter{
		{key: "room_id", value: query.Get("room_id")},
		{key: "area_id", value: query.Get("area_id")},
		{key: "category", value: query.Get("category")},
	}

	for _, filter := range filters {
		if filter.value == "" {
			continue
		}
		return s.listScenesByFilter(ctx, filter, scope)
	}

	scenes, err := s.sceneRegistry.ListScenes(ctx)
	if err != nil {
		return nil, "", err
	}
	return applySceneScope(scenes, scope), "", nil
}

type sceneFilter struct {
	key   string
	value string
}

func (s *Server) listScenesByFilter(ctx context.Context, filter sceneFilter, scope *auth.RoomScope) ([]automation.Scene, string, error) {
	switch filter.key {
	case "room_id":
		if len(filter.value) > maxQueryParamLen {
			return nil, "room_id exceeds maximum length", nil
		}
		if scope != nil && !scope.CanAccessRoom(filter.value) {
			return []automation.Scene{}, "", nil
		}
		scenes, err := s.sceneRegistry.ListScenesByRoom(ctx, filter.value)
		if err != nil {
			return nil, "", err
		}
		return applySceneScope(scenes, scope), "", nil
	case "area_id":
		if len(filter.value) > maxQueryParamLen {
			return nil, "area_id exceeds maximum length", nil
		}
		scenes, err := s.sceneRegistry.ListScenesByArea(ctx, filter.value)
		if err != nil {
			return nil, "", err
		}
		return applySceneScope(scenes, scope), "", nil
	case "category":
		if len(filter.value) > maxQueryParamLen {
			return nil, "category exceeds maximum length", nil
		}
		cat := automation.Category(filter.value)
		if !isValidSceneCategory(cat) {
			return nil, "invalid category", nil
		}
		scenes, err := s.sceneRegistry.ListScenesByCategory(ctx, cat)
		if err != nil {
			return nil, "", err
		}
		return applySceneScope(scenes, scope), "", nil
	default:
		return []automation.Scene{}, "", nil
	}
}

// applySceneScope filters scenes when a room scope is present.
func applySceneScope(scenes []automation.Scene, scope *auth.RoomScope) []automation.Scene {
	if scope == nil {
		return ensureSceneSlice(scenes)
	}
	return filterByRoomIDs(scenes, scope.RoomIDs, func(scene automation.Scene) *string {
		return scene.RoomID
	})
}

// ensureSceneSlice normalises nil slices to empty for JSON responses.
func ensureSceneSlice(scenes []automation.Scene) []automation.Scene {
	if scenes == nil {
		return []automation.Scene{}
	}
	return scenes
}

// sceneAccessDenied checks whether a scene is outside the accessible scope.
func sceneAccessDenied(scope *auth.RoomScope, roomID string) (bool, string) {
	if scope == nil {
		return false, ""
	}
	if len(scope.RoomIDs) == 0 {
		return true, sceneNotAccessibleMessage
	}
	if !scope.CanAccessRoom(roomID) {
		return true, sceneNotAccessibleMessage
	}
	return false, ""
}

// sceneManageDenied checks whether scene management is permitted for a room.
func sceneManageDenied(scope *auth.RoomScope, roomID string) (bool, string) {
	if scope == nil {
		return false, ""
	}
	if len(scope.RoomIDs) == 0 {
		return true, sceneManageForbiddenMessage
	}
	if !scope.CanAccessRoom(roomID) {
		return true, sceneNotAccessibleMessage
	}
	if !scope.CanManageScenesInRoom(roomID) {
		return true, sceneManageForbiddenMessage
	}
	return false, ""
}

// parseActivateRequest decodes the optional scene activation payload.
func parseActivateRequest(r *http.Request) (activateRequest, error) {
	var req activateRequest
	if r.Body == nil || r.ContentLength == 0 {
		return req, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return activateRequest{}, err
	}
	return req, nil
}

// normaliseTrigger applies defaults for trigger fields.
func normaliseTrigger(req activateRequest) (string, string) {
	triggerType := req.TriggerType
	if triggerType == "" {
		triggerType = "manual"
	}
	triggerSource := req.TriggerSource
	if triggerSource == "" {
		triggerSource = "api"
	}
	return triggerType, triggerSource
}

// isValidSceneCategory validates a scene category against allowed values.
func isValidSceneCategory(category automation.Category) bool {
	for _, c := range automation.AllCategories() {
		if c == category {
			return true
		}
	}
	return false
}

// validateActionDeviceScope checks that all scene action device_ids are
// accessible to the current user's room scope. Admins (nil scope) skip this.
// Devices not yet in the registry are allowed (pre-commissioning use case).
func (s *Server) validateActionDeviceScope(ctx context.Context, scope *auth.RoomScope, actions []automation.SceneAction) error {
	if scope == nil {
		return nil // admins/owners have full access
	}
	for _, action := range actions {
		if action.DeviceID == "" {
			continue
		}
		dev, err := s.registry.GetDevice(ctx, action.DeviceID)
		if err != nil {
			continue // device not in registry yet — allow (harmless at execution time)
		}
		if !deviceInScope(scope, dev) {
			return fmt.Errorf("action targets device %q outside accessible rooms", action.DeviceID)
		}
	}
	return nil
}
