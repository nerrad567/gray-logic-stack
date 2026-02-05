package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/nerrad567/gray-logic-core/internal/auth"
	"github.com/nerrad567/gray-logic-core/internal/device"
)

// handleListDevices returns all devices, with optional query filters.
//
// Query parameters:
//   - room_id: filter by room
//   - area_id: filter by area
//   - domain: filter by domain (lighting, climate, etc.)
//   - protocol: filter by protocol (knx, dali, etc.)
//   - capability: filter by capability (on_off, dim, etc.)
//   - health: filter by health status (online, offline, degraded, unknown)
//   - tag: filter by device tag (escape_lighting, accent, etc.)
func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := requestRoomScope(ctx)
	handleScopedList(w, scope, "devices", "failed to list devices", func() ([]device.Device, string, error) {
		return s.listDevicesForRequest(ctx, r, scope)
	})
}

// handleGetDevice returns a single device by ID.
func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	scope := requestRoomScope(r.Context())
	if scope != nil && len(scope.RoomIDs) == 0 {
		writeForbidden(w, "device not in accessible rooms")
		return
	}

	dev, err := s.registry.GetDevice(r.Context(), id)
	if err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			writeNotFound(w, "device not found")
			return
		}
		writeInternalError(w, "failed to get device")
		return
	}

	if !deviceInScope(scope, dev) {
		writeForbidden(w, "device not in accessible rooms")
		return
	}

	writeJSON(w, http.StatusOK, dev)
}

// handleCreateDevice creates a new device.
func (s *Server) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var dev device.Device
	if err := json.NewDecoder(r.Body).Decode(&dev); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if err := s.registry.CreateDevice(r.Context(), &dev); err != nil {
		// Check for validation errors
		if isValidationError(err) {
			writeBadRequest(w, err.Error())
			return
		}
		writeInternalError(w, "failed to create device")
		return
	}

	writeJSON(w, http.StatusCreated, dev)
}

// handleUpdateDevice partially updates a device.
func (s *Server) handleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Get existing device
	existing, err := s.registry.GetDevice(r.Context(), id)
	if err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			writeNotFound(w, "device not found")
			return
		}
		writeInternalError(w, "failed to get device")
		return
	}

	// Decode partial update onto existing device
	if err := json.NewDecoder(r.Body).Decode(existing); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}
	existing.ID = id // Ensure ID cannot be changed

	if err := s.registry.UpdateDevice(r.Context(), existing); err != nil {
		if isValidationError(err) {
			writeBadRequest(w, err.Error())
			return
		}
		writeInternalError(w, "failed to update device")
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// handleDeleteDevice removes a device by ID.
func (s *Server) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.registry.DeleteDevice(r.Context(), id); err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			writeNotFound(w, "device not found")
			return
		}
		writeInternalError(w, "failed to delete device")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleDeviceStats returns device registry statistics.
func (s *Server) handleDeviceStats(w http.ResponseWriter, _ *http.Request) {
	stats := s.registry.GetStats()
	writeJSON(w, http.StatusOK, stats)
}

// handleGetDeviceState returns the current state of a device.
func (s *Server) handleGetDeviceState(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	scope := requestRoomScope(r.Context())
	if scope != nil && len(scope.RoomIDs) == 0 {
		writeForbidden(w, "device not in accessible rooms")
		return
	}

	dev, err := s.registry.GetDevice(r.Context(), id)
	if err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			writeNotFound(w, "device not found")
			return
		}
		writeInternalError(w, "failed to get device")
		return
	}

	if !deviceInScope(scope, dev) {
		writeForbidden(w, "device not in accessible rooms")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"device_id":        dev.ID,
		"state":            dev.State,
		"state_updated_at": dev.StateUpdatedAt,
		"health_status":    dev.HealthStatus,
	})
}

// DeviceCommand represents a command to send to a device via MQTT.
type DeviceCommand struct {
	Command    string         `json:"command"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

// handleSetDeviceState sends a command to a device via MQTT.
// This is an asynchronous operation — the command is published to MQTT and the
// response is 202 Accepted. The actual state change arrives via WebSocket.
func (s *Server) handleSetDeviceState(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	scope := requestRoomScope(r.Context())
	if scope != nil && len(scope.RoomIDs) == 0 {
		writeForbidden(w, "device not in accessible rooms")
		return
	}

	// Verify device exists and get protocol info for routing
	dev, err := s.registry.GetDevice(r.Context(), id)
	if err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			writeNotFound(w, "device not found")
			return
		}
		writeInternalError(w, "failed to get device")
		return
	}

	if !deviceInScope(scope, dev) {
		writeForbidden(w, "device not in accessible rooms")
		return
	}

	// Decode command
	var cmd DeviceCommand
	if decodeErr := json.NewDecoder(r.Body).Decode(&cmd); decodeErr != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}
	if cmd.Command == "" {
		writeBadRequest(w, "command field is required")
		return
	}

	// Build MQTT command message
	commandID := generateRequestID()
	mqttPayload := map[string]any{
		"id":         commandID,
		"device_id":  id,
		"command":    cmd.Command,
		"parameters": cmd.Parameters,
		"source":     "api",
	}

	payload, marshalErr := json.Marshal(mqttPayload)
	if marshalErr != nil {
		writeInternalError(w, "failed to encode command")
		return
	}

	// Publish to MQTT if available — the protocol bridge subscribes to this topic.
	// Topic format: graylogic/command/{protocol}/{device_id}
	if s.mqtt != nil {
		topic := "graylogic/command/" + string(dev.Protocol) + "/" + id
		if pubErr := s.mqtt.Publish(topic, payload, 1, false); pubErr != nil {
			s.logger.Debug("MQTT publish failed", "error", pubErr)
		}
	}

	newState, simulated := s.simulateDeviceStateChange(id, dev.State, cmd)

	logFields := []any{
		"device_id", id,
		"command", cmd.Command,
		"parameters", cmd.Parameters,
		"command_id", commandID,
	}
	if simulated {
		logFields = append(logFields, "new_state", newState)
	}
	s.logger.Info("device command sent", logFields...)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"command_id": commandID,
		"status":     "accepted",
		"message":    "command published, state update will follow via WebSocket",
	})
}

// commandToState translates a device command into the resulting state.
// Used in dev/demo mode when no protocol bridge is available to confirm the change.
func commandToState(command string, params map[string]any, current device.State) device.State { //nolint:gocyclo // command-to-state mapping: switch on command type
	newState := make(device.State, len(current))
	for k, v := range current {
		newState[k] = v
	}

	switch command {
	case "on", "turn_on":
		newState["on"] = true
	case "off", "turn_off":
		newState["on"] = false
	case "toggle":
		if on, ok := newState["on"].(bool); ok {
			newState["on"] = !on
		} else {
			newState["on"] = true
		}
	case "dim", "set_level":
		if level, ok := params["level"]; ok {
			newState["level"] = level
			newState["on"] = true
		}
	case "set_position":
		if pos, ok := params["position"]; ok {
			newState["position"] = pos
		}
	case "set_tilt":
		if tilt, ok := params["tilt"]; ok {
			newState["tilt"] = tilt
		}
	case "set_temperature", "set_setpoint":
		if temp, ok := params["setpoint"]; ok {
			newState["setpoint"] = temp
		}
	case "set_mode":
		if mode, ok := params["mode"]; ok {
			newState["mode"] = mode
		}
	default:
		// For unknown commands, merge all parameters into state
		for k, v := range params {
			newState[k] = v
		}
	}

	return newState
}

// simulateDeviceStateChange mimics bridge confirmations in dev mode.
func (s *Server) simulateDeviceStateChange(deviceID string, current device.State, cmd DeviceCommand) (device.State, bool) {
	// In dev mode WITHOUT a real bridge, simulate the bridge confirmation
	// loop: delay the state write + WebSocket broadcast to mimic the real
	// KNX bus round-trip time. When a bridge is active (knxBridge != nil),
	// the real bridge handles the confirmation via MQTT → WebSocket, so we
	// must not also inject a simulated state — that would race with the
	// real state updates and cause flickering / stale overwrites.
	if !s.devMode || s.knxBridge != nil {
		return nil, false
	}

	newState := commandToState(cmd.Command, cmd.Parameters, current)

	go func(state device.State) {
		// Simulate bridge processing + bus round-trip (KNX typical: 100-300ms,
		// exaggerated here so the pending UI is clearly visible during dev)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		select {
		case <-time.After(400 * time.Millisecond):
		case <-ctx.Done():
			s.logger.Debug("dev mode simulation cancelled", "device_id", deviceID)
			return
		}

		if updateErr := s.registry.SetDeviceState(ctx, deviceID, state); updateErr != nil {
			s.logger.Warn("failed to apply local state", "error", updateErr)
			return
		}
		s.hub.Broadcast("device.state_changed", map[string]any{
			"device_id": deviceID,
			"state":     state,
		})
		s.logger.Debug("dev mode: simulated bridge confirmation", "device_id", deviceID)
	}(newState)

	return newState, true
}

// isValidationError checks whether an error is a device validation error.
// ValidateDevice wraps various sentinel errors (ErrInvalidName, ErrInvalidAddress, etc.)
// so we check all of them rather than just ErrInvalidDevice.
func isValidationError(err error) bool {
	return errors.Is(err, device.ErrInvalidDevice) ||
		errors.Is(err, device.ErrInvalidName) ||
		errors.Is(err, device.ErrInvalidSlug) ||
		errors.Is(err, device.ErrInvalidDeviceType) ||
		errors.Is(err, device.ErrInvalidDomain) ||
		errors.Is(err, device.ErrInvalidProtocol) ||
		errors.Is(err, device.ErrInvalidAddress) ||
		errors.Is(err, device.ErrInvalidCapability) ||
		errors.Is(err, device.ErrInvalidState)
}

// deviceInScope returns true if the device is in an accessible room.
func deviceInScope(scope *auth.RoomScope, dev *device.Device) bool {
	if scope == nil {
		return true
	}
	return scope.CanAccessRoom(derefString(dev.RoomID))
}

// listDevicesForRequest returns devices matching query filters and room scope.
func (s *Server) listDevicesForRequest(ctx context.Context, r *http.Request, scope *auth.RoomScope) ([]device.Device, string, error) {
	query := r.URL.Query()

	filters := []deviceFilter{
		{key: "room_id", value: query.Get("room_id")},
		{key: "area_id", value: query.Get("area_id")},
		{key: "domain", value: query.Get("domain")},
		{key: "protocol", value: query.Get("protocol")},
		{key: "capability", value: query.Get("capability")},
		{key: "health", value: query.Get("health")},
		{key: "tag", value: query.Get("tag")},
	}

	for _, filter := range filters {
		if filter.value == "" {
			continue
		}
		return s.listDevicesByFilter(ctx, filter, scope)
	}

	devices, err := s.registry.ListDevices(ctx)
	if err != nil {
		return nil, "", err
	}
	return applyDeviceScope(devices, scope), "", nil
}

type deviceFilter struct {
	key   string
	value string
}

func (s *Server) listDevicesByFilter(ctx context.Context, filter deviceFilter, scope *auth.RoomScope) ([]device.Device, string, error) {
	switch filter.key {
	case "room_id":
		return s.listDevicesByRoom(ctx, filter.value, scope)
	case "area_id":
		return s.listDevicesByArea(ctx, filter.value, scope)
	case "domain":
		return s.listDevicesByDomain(ctx, filter.value, scope)
	case "protocol":
		return s.listDevicesByProtocol(ctx, filter.value, scope)
	case "capability":
		return s.listDevicesByCapability(ctx, filter.value, scope)
	case "health":
		return s.listDevicesByHealth(ctx, filter.value, scope)
	case "tag":
		return s.listDevicesByTag(ctx, filter.value, scope)
	default:
		return []device.Device{}, "", nil
	}
}

func validateDeviceFilterValue(param, value string) string {
	if len(value) > maxQueryParamLen {
		return param + " exceeds maximum length"
	}
	return ""
}

func (s *Server) listDevicesByRoom(ctx context.Context, value string, scope *auth.RoomScope) ([]device.Device, string, error) {
	if bad := validateDeviceFilterValue("room_id", value); bad != "" {
		return nil, bad, nil
	}
	if scope != nil && !scope.CanAccessRoom(value) {
		return []device.Device{}, "", nil
	}
	devices, err := s.registry.GetDevicesByRoom(ctx, value)
	if err != nil {
		return nil, "", err
	}
	return applyDeviceScope(devices, scope), "", nil
}

func (s *Server) listDevicesByArea(ctx context.Context, value string, scope *auth.RoomScope) ([]device.Device, string, error) {
	if bad := validateDeviceFilterValue("area_id", value); bad != "" {
		return nil, bad, nil
	}
	devices, err := s.registry.GetDevicesByArea(ctx, value)
	if err != nil {
		return nil, "", err
	}
	return applyDeviceScope(devices, scope), "", nil
}

func (s *Server) listDevicesByDomain(ctx context.Context, value string, scope *auth.RoomScope) ([]device.Device, string, error) {
	if bad := validateDeviceFilterValue("domain", value); bad != "" {
		return nil, bad, nil
	}
	devices, err := s.registry.GetDevicesByDomain(ctx, device.Domain(value))
	if err != nil {
		return nil, "", err
	}
	return applyDeviceScope(devices, scope), "", nil
}

func (s *Server) listDevicesByProtocol(ctx context.Context, value string, scope *auth.RoomScope) ([]device.Device, string, error) {
	if bad := validateDeviceFilterValue("protocol", value); bad != "" {
		return nil, bad, nil
	}
	devices, err := s.registry.GetDevicesByProtocol(ctx, device.Protocol(value))
	if err != nil {
		return nil, "", err
	}
	return applyDeviceScope(devices, scope), "", nil
}

func (s *Server) listDevicesByCapability(ctx context.Context, value string, scope *auth.RoomScope) ([]device.Device, string, error) {
	if bad := validateDeviceFilterValue("capability", value); bad != "" {
		return nil, bad, nil
	}
	devices, err := s.registry.GetDevicesByCapability(ctx, device.Capability(value))
	if err != nil {
		return nil, "", err
	}
	return applyDeviceScope(devices, scope), "", nil
}

func (s *Server) listDevicesByHealth(ctx context.Context, value string, scope *auth.RoomScope) ([]device.Device, string, error) {
	if bad := validateDeviceFilterValue("health", value); bad != "" {
		return nil, bad, nil
	}
	devices, err := s.registry.GetDevicesByHealthStatus(ctx, device.HealthStatus(value))
	if err != nil {
		return nil, "", err
	}
	return applyDeviceScope(devices, scope), "", nil
}

func (s *Server) listDevicesByTag(ctx context.Context, value string, scope *auth.RoomScope) ([]device.Device, string, error) {
	if bad := validateDeviceFilterValue("tag", value); bad != "" {
		return nil, bad, nil
	}
	devices, err := s.registry.GetDevicesByTag(ctx, value)
	if err != nil {
		return nil, "", err
	}
	return applyDeviceScope(devices, scope), "", nil
}

// applyDeviceScope filters devices when a room scope is present.
func applyDeviceScope(devices []device.Device, scope *auth.RoomScope) []device.Device {
	if scope == nil {
		return ensureDeviceSlice(devices)
	}
	return filterByRoomIDs(devices, scope.RoomIDs, func(dev device.Device) *string {
		return dev.RoomID
	})
}

// ensureDeviceSlice normalises nil slices to empty for JSON responses.
func ensureDeviceSlice(devices []device.Device) []device.Device {
	if devices == nil {
		return []device.Device{}
	}
	return devices
}
