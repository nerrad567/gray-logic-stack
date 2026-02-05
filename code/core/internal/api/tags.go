package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handleListAllTags returns all unique tags across all devices.
//
// GET /tags
// Response: {"tags": ["accent", "escape_lighting", "entertainment"], "count": 3}
func (s *Server) handleListAllTags(w http.ResponseWriter, r *http.Request) {
	tags, err := s.tagRepo.ListAllTags(r.Context())
	if err != nil {
		s.logger.Error("failed to list tags", "error", err)
		writeInternalError(w, "failed to list tags")
		return
	}
	if tags == nil {
		tags = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tags": tags, "count": len(tags)})
}

// handleSetDeviceTags replaces all tags on a device.
//
// PUT /devices/{id}/tags
// Body: {"tags": ["escape_lighting", "accent"]}
// Response: {"device_id": "dev-1", "tags": ["accent", "escape_lighting"]}
func (s *Server) handleSetDeviceTags(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verify device exists
	if _, err := s.registry.GetDevice(r.Context(), id); err != nil {
		writeNotFound(w, "device not found")
		return
	}

	var body struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if err := s.tagRepo.SetTags(r.Context(), id, body.Tags); err != nil {
		s.logger.Error("failed to set device tags", "error", err, "device_id", id)
		writeInternalError(w, "failed to set tags")
		return
	}

	// Re-read tags to return the normalised set
	tags, err := s.tagRepo.GetTags(r.Context(), id)
	if err != nil {
		s.logger.Error("failed to get device tags", "error", err, "device_id", id)
		writeInternalError(w, "failed to get tags")
		return
	}
	if tags == nil {
		tags = []string{}
	}

	// Update the registry cache so subsequent reads include the new tags
	if refreshErr := s.registry.RefreshCache(r.Context()); refreshErr != nil {
		s.logger.Warn("failed to refresh cache after tag update", "error", refreshErr)
	}

	writeJSON(w, http.StatusOK, map[string]any{"device_id": id, "tags": tags})
}

// handleGetDeviceTags returns the tags for a specific device.
//
// GET /devices/{id}/tags
// Response: {"device_id": "dev-1", "tags": ["accent", "escape_lighting"]}
func (s *Server) handleGetDeviceTags(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verify device exists
	if _, err := s.registry.GetDevice(r.Context(), id); err != nil {
		writeNotFound(w, "device not found")
		return
	}

	tags, err := s.tagRepo.GetTags(r.Context(), id)
	if err != nil {
		s.logger.Error("failed to get device tags", "error", err, "device_id", id)
		writeInternalError(w, "failed to get tags")
		return
	}
	if tags == nil {
		tags = []string{}
	}

	writeJSON(w, http.StatusOK, map[string]any{"device_id": id, "tags": tags})
}
