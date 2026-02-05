package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nerrad567/gray-logic-core/internal/auth"
	"github.com/nerrad567/gray-logic-core/internal/device"
)

// handleListGroups returns all device groups.
//
// GET /device-groups
// Response: {"groups": [...], "count": N}
func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := s.groupRepo.List(r.Context())
	if err != nil {
		s.logger.Error("failed to list device groups", "error", err)
		writeInternalError(w, "failed to list device groups")
		return
	}
	if groups == nil {
		groups = []device.DeviceGroup{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": groups, "count": len(groups)})
}

// handleCreateGroup creates a new device group.
//
// POST /device-groups
// Body: DeviceGroup JSON
// Response: 201 Created with the created group
func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var group device.DeviceGroup
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if group.Name == "" || len(group.Name) > 128 { //nolint:mnd // max name length
		writeBadRequest(w, "name is required and must be at most 128 characters")
		return
	}
	if group.Type == "" {
		group.Type = device.GroupTypeStatic
	}
	if !isValidGroupType(group.Type) {
		writeBadRequest(w, "type must be static, dynamic, or hybrid")
		return
	}

	if err := s.groupRepo.Create(r.Context(), &group); err != nil {
		if errors.Is(err, device.ErrGroupExists) {
			writeConflict(w, "a group with this slug already exists")
			return
		}
		s.logger.Error("failed to create device group", "error", err)
		writeInternalError(w, "failed to create device group")
		return
	}

	writeJSON(w, http.StatusCreated, group)
}

// handleGetGroup returns a single device group by ID.
//
// GET /device-groups/{id}
// Response: DeviceGroup JSON
func (s *Server) handleGetGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	group, err := s.groupRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, device.ErrGroupNotFound) {
			writeNotFound(w, "device group not found")
			return
		}
		s.logger.Error("failed to get device group", "error", err, "id", id)
		writeInternalError(w, "failed to get device group")
		return
	}

	writeJSON(w, http.StatusOK, group)
}

// handleUpdateGroup partially updates a device group via PATCH semantics.
//
// PATCH /device-groups/{id}
// Body: partial DeviceGroup fields
// Response: updated DeviceGroup JSON
func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) { //nolint:gocognit,gocyclo // HTTP handler: validates and patches multiple optional fields
	id := chi.URLParam(r, "id")

	group, err := s.groupRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, device.ErrGroupNotFound) {
			writeNotFound(w, "device group not found")
			return
		}
		s.logger.Error("failed to get device group", "error", err, "id", id)
		writeInternalError(w, "failed to get device group")
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
			group.Name = name
			group.Slug = slugify(name)
		}
	}
	if v, ok := raw["description"]; ok {
		var desc string
		if json.Unmarshal(v, &desc) == nil {
			group.Description = &desc
		}
	}
	if v, ok := raw["type"]; ok {
		var t device.GroupType
		if json.Unmarshal(v, &t) == nil && isValidGroupType(t) {
			// Guard against inconsistent type changes
			if t != group.Type {
				// Changing to dynamic requires filter_rules
				if t == device.GroupTypeDynamic && group.FilterRules == nil {
					if _, hasRules := raw["filter_rules"]; !hasRules {
						writeBadRequest(w, "changing to dynamic type requires filter_rules")
						return
					}
				}
				// Changing away from static: warn if members exist
				if group.Type == device.GroupTypeStatic && (t == device.GroupTypeDynamic) {
					members, mErr := s.groupRepo.GetMemberDeviceIDs(r.Context(), id)
					if mErr == nil && len(members) > 0 {
						writeBadRequest(w, "clear explicit members before changing from static to dynamic")
						return
					}
				}
			}
			group.Type = t
		}
	}
	if v, ok := raw["filter_rules"]; ok {
		var rules device.FilterRules
		if json.Unmarshal(v, &rules) == nil {
			group.FilterRules = &rules
		}
	}
	if v, ok := raw["icon"]; ok {
		var icon string
		if json.Unmarshal(v, &icon) == nil {
			group.Icon = &icon
		}
	}
	if v, ok := raw["colour"]; ok {
		var colour string
		if json.Unmarshal(v, &colour) == nil {
			group.Colour = &colour
		}
	}
	if v, ok := raw["sort_order"]; ok {
		var order int
		if json.Unmarshal(v, &order) == nil {
			group.SortOrder = order
		}
	}

	if err := s.groupRepo.Update(r.Context(), group); err != nil { //nolint:govet // shadow: err re-declared in nested scope, checked immediately
		s.logger.Error("failed to update device group", "error", err, "id", id)
		writeInternalError(w, "failed to update device group")
		return
	}

	// Re-read to get updated timestamp
	updated, err := s.groupRepo.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusOK, group)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// handleDeleteGroup removes a device group by ID.
//
// DELETE /device-groups/{id}
// Response: 204 No Content
func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.groupRepo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, device.ErrGroupNotFound) {
			writeNotFound(w, "device group not found")
			return
		}
		s.logger.Error("failed to delete device group", "error", err, "id", id)
		writeInternalError(w, "failed to delete device group")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleSetGroupMembers replaces the explicit member list for a group.
//
// PUT /device-groups/{id}/members
// Body: {"device_ids": ["dev-1", "dev-2", "dev-3"]}
// Response: {"group_id": "grp-1", "device_ids": [...], "count": N}
func (s *Server) handleSetGroupMembers(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verify group exists
	if _, err := s.groupRepo.GetByID(r.Context(), id); err != nil {
		if errors.Is(err, device.ErrGroupNotFound) {
			writeNotFound(w, "device group not found")
			return
		}
		s.logger.Error("failed to get device group", "error", err, "id", id)
		writeInternalError(w, "failed to get device group")
		return
	}

	var body struct {
		DeviceIDs []string `json:"device_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if len(body.DeviceIDs) > 500 { //nolint:mnd // max members per group
		writeBadRequest(w, "too many members (max 500 per group)")
		return
	}

	if err := s.groupRepo.SetMembers(r.Context(), id, body.DeviceIDs); err != nil {
		s.logger.Error("failed to set group members", "error", err, "group_id", id)
		writeInternalError(w, "failed to set group members")
		return
	}

	// Read back the member IDs
	memberIDs, err := s.groupRepo.GetMemberDeviceIDs(r.Context(), id)
	if err != nil {
		s.logger.Error("failed to get group members", "error", err, "group_id", id)
		writeInternalError(w, "failed to get group members")
		return
	}
	if memberIDs == nil {
		memberIDs = []string{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"group_id":   id,
		"device_ids": memberIDs,
		"count":      len(memberIDs),
	})
}

// handleGetGroupMembers returns the explicit member device IDs for a group.
//
// GET /device-groups/{id}/members
// Response: {"group_id": "grp-1", "members": [...], "count": N}
func (s *Server) handleGetGroupMembers(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verify group exists
	if _, err := s.groupRepo.GetByID(r.Context(), id); err != nil {
		if errors.Is(err, device.ErrGroupNotFound) {
			writeNotFound(w, "device group not found")
			return
		}
		s.logger.Error("failed to get device group", "error", err, "id", id)
		writeInternalError(w, "failed to get device group")
		return
	}

	members, err := s.groupRepo.GetMembers(r.Context(), id)
	if err != nil {
		s.logger.Error("failed to get group members", "error", err, "group_id", id)
		writeInternalError(w, "failed to get group members")
		return
	}
	if members == nil {
		members = []device.GroupMember{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"group_id": id,
		"members":  members,
		"count":    len(members),
	})
}

// handleResolveGroup expands a device group into its concrete device list.
//
// GET /device-groups/{id}/resolve
// Response: {"group_id": "grp-1", "devices": [...], "count": N}
func (s *Server) handleResolveGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	group, err := s.groupRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, device.ErrGroupNotFound) {
			writeNotFound(w, "device group not found")
			return
		}
		s.logger.Error("failed to get device group", "error", err, "id", id)
		writeInternalError(w, "failed to get device group")
		return
	}

	devices, err := device.ResolveGroup(r.Context(), group, s.registry, s.tagRepo, s.groupRepo)
	if err != nil {
		s.logger.Error("failed to resolve device group", "error", err, "id", id)
		writeInternalError(w, "failed to resolve device group")
		return
	}
	if devices == nil {
		devices = []device.Device{}
	}

	// Apply room scope if present (resolved devices still respect permissions)
	scope := requestRoomScope(r.Context())
	devices = applyDeviceScopeSlice(devices, scope)

	writeJSON(w, http.StatusOK, map[string]any{
		"group_id": id,
		"devices":  devices,
		"count":    len(devices),
	})
}

// applyDeviceScopeSlice filters a pre-fetched device slice by room scope.
func applyDeviceScopeSlice(devices []device.Device, scope *auth.RoomScope) []device.Device {
	if scope == nil {
		return devices
	}
	var filtered []device.Device
	for _, d := range devices {
		if deviceInScope(scope, &d) {
			filtered = append(filtered, d)
		}
	}
	if filtered == nil {
		return []device.Device{}
	}
	return filtered
}

// isValidGroupType checks whether a group type string is valid.
func isValidGroupType(t device.GroupType) bool {
	return t == device.GroupTypeStatic || t == device.GroupTypeDynamic || t == device.GroupTypeHybrid
}
