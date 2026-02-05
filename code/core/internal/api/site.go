package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/nerrad567/gray-logic-core/internal/location"
)

// SiteRequest is the JSON body for creating or updating a site.
type SiteRequest struct {
	ID             string   `json:"id,omitempty"`
	Name           string   `json:"name"`
	Address        string   `json:"address,omitempty"`
	Latitude       *float64 `json:"latitude,omitempty"`
	Longitude      *float64 `json:"longitude,omitempty"`
	Timezone       string   `json:"timezone,omitempty"`
	ElevationM     *float64 `json:"elevation_m,omitempty"`
	ModesAvailable []string `json:"modes_available,omitempty"`
	ModeCurrent    string   `json:"mode_current,omitempty"`
}

// handleGetSite returns the site record, or 404 if none exists.
// A 404 signals to the Flutter panel that setup is needed.
func (s *Server) handleGetSite(w http.ResponseWriter, r *http.Request) {
	site, err := s.locationRepo.GetAnySite(r.Context())
	if err != nil {
		if errors.Is(err, location.ErrSiteNotFound) {
			writeNotFound(w, "no site configured")
			return
		}
		s.logger.Error("failed to get site", "error", err)
		writeInternalError(w, "failed to get site")
		return
	}

	writeJSON(w, http.StatusOK, site)
}

// handleCreateSite creates a new site record. Used by the setup wizard
// after a fresh install or factory reset with clear_site.
func (s *Server) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	var req SiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	// Validate required fields.
	if req.Name == "" {
		writeBadRequest(w, "name is required")
		return
	}

	// Check if a site already exists — only one site per deployment.
	if existing, err := s.locationRepo.GetAnySite(r.Context()); err == nil && existing != nil {
		writeError(w, http.StatusConflict, ErrCodeConflict,
			"site already exists — use PATCH to update")
		return
	}

	site := &location.Site{
		ID:             valueOrDefault(req.ID, s.siteID, "site-001"),
		Name:           req.Name,
		Slug:           slugify(req.Name),
		Address:        req.Address,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		Timezone:       valueOrDefault(req.Timezone, "UTC"),
		ElevationM:     req.ElevationM,
		ModesAvailable: req.ModesAvailable,
		ModeCurrent:    valueOrDefault(req.ModeCurrent, "home"),
	}

	if len(site.ModesAvailable) == 0 {
		site.ModesAvailable = []string{"home", "away", "night", "holiday"}
	}

	if err := s.locationRepo.CreateSite(r.Context(), site); err != nil {
		s.logger.Error("failed to create site", "error", err)
		writeInternalError(w, "failed to create site")
		return
	}

	s.logger.Info("site created", "id", site.ID, "name", site.Name)

	// Re-read from DB to get timestamps populated by SQLite defaults.
	created, err := s.locationRepo.GetAnySite(r.Context())
	if err != nil {
		// Creation succeeded but re-read failed — return what we have.
		writeJSON(w, http.StatusCreated, site)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleUpdateSite updates the existing site record via PATCH semantics.
// Only fields present in the JSON body are updated.
func (s *Server) handleUpdateSite(w http.ResponseWriter, r *http.Request) { //nolint:gocognit,gocyclo // HTTP handler: validates and patches many optional site fields
	// Fetch the current site.
	site, err := s.locationRepo.GetAnySite(r.Context())
	if err != nil {
		if errors.Is(err, location.ErrSiteNotFound) {
			writeNotFound(w, "no site configured — create one first")
			return
		}
		s.logger.Error("failed to get site for update", "error", err)
		writeInternalError(w, "failed to get site")
		return
	}

	// Decode the partial update into a raw map so we can detect which
	// fields were explicitly sent (PATCH semantics).
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil { //nolint:govet // shadow: err re-declared in nested scope, checked immediately
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if len(raw) == 0 {
		writeBadRequest(w, "no fields to update")
		return
	}

	// Apply each provided field to the existing site.
	if v, ok := raw["name"]; ok {
		var name string
		if json.Unmarshal(v, &name) == nil && name != "" {
			site.Name = name
			site.Slug = slugify(name)
		}
	}
	if v, ok := raw["address"]; ok {
		var addr string
		if json.Unmarshal(v, &addr) == nil {
			site.Address = addr
		}
	}
	if v, ok := raw["latitude"]; ok {
		var lat *float64
		if json.Unmarshal(v, &lat) == nil {
			site.Latitude = lat
		}
	}
	if v, ok := raw["longitude"]; ok {
		var lon *float64
		if json.Unmarshal(v, &lon) == nil {
			site.Longitude = lon
		}
	}
	if v, ok := raw["timezone"]; ok {
		var tz string
		if json.Unmarshal(v, &tz) == nil && tz != "" {
			site.Timezone = tz
		}
	}
	if v, ok := raw["elevation_m"]; ok {
		var elev *float64
		if json.Unmarshal(v, &elev) == nil {
			site.ElevationM = elev
		}
	}
	if v, ok := raw["modes_available"]; ok {
		var modes []string
		if json.Unmarshal(v, &modes) == nil && len(modes) > 0 {
			site.ModesAvailable = modes
		}
	}
	if v, ok := raw["mode_current"]; ok {
		var mode string
		if json.Unmarshal(v, &mode) == nil && mode != "" {
			// Validate the mode is in the available list.
			valid := false
			for _, m := range site.ModesAvailable {
				if m == mode {
					valid = true
					break
				}
			}
			if !valid {
				writeBadRequest(w, "mode_current must be one of modes_available: "+strings.Join(site.ModesAvailable, ", "))
				return
			}
			site.ModeCurrent = mode
		}
	}

	if err := s.locationRepo.UpdateSite(r.Context(), site); err != nil { //nolint:govet // shadow: err re-declared in nested scope, checked immediately
		s.logger.Error("failed to update site", "error", err)
		writeInternalError(w, "failed to update site")
		return
	}

	s.logger.Info("site updated", "id", site.ID, "name", site.Name)

	// Re-read from DB to get the updated_at timestamp.
	updated, err := s.locationRepo.GetAnySite(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, site)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// valueOrDefault returns the first non-empty string from the arguments.
func valueOrDefault(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
