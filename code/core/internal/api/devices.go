package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

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
func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check for filters
	if roomID := r.URL.Query().Get("room_id"); roomID != "" {
		devices, err := s.registry.GetDevicesByRoom(ctx, roomID)
		if err != nil {
			writeInternalError(w, "failed to list devices")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"devices": devices, "count": len(devices)})
		return
	}

	if areaID := r.URL.Query().Get("area_id"); areaID != "" {
		devices, err := s.registry.GetDevicesByArea(ctx, areaID)
		if err != nil {
			writeInternalError(w, "failed to list devices")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"devices": devices, "count": len(devices)})
		return
	}

	if domainStr := r.URL.Query().Get("domain"); domainStr != "" {
		devices, err := s.registry.GetDevicesByDomain(ctx, device.Domain(domainStr))
		if err != nil {
			writeInternalError(w, "failed to list devices")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"devices": devices, "count": len(devices)})
		return
	}

	if protocolStr := r.URL.Query().Get("protocol"); protocolStr != "" {
		devices, err := s.registry.GetDevicesByProtocol(ctx, device.Protocol(protocolStr))
		if err != nil {
			writeInternalError(w, "failed to list devices")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"devices": devices, "count": len(devices)})
		return
	}

	if capStr := r.URL.Query().Get("capability"); capStr != "" {
		devices, err := s.registry.GetDevicesByCapability(ctx, device.Capability(capStr))
		if err != nil {
			writeInternalError(w, "failed to list devices")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"devices": devices, "count": len(devices)})
		return
	}

	if healthStr := r.URL.Query().Get("health"); healthStr != "" {
		devices, err := s.registry.GetDevicesByHealthStatus(ctx, device.HealthStatus(healthStr))
		if err != nil {
			writeInternalError(w, "failed to list devices")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"devices": devices, "count": len(devices)})
		return
	}

	// No filter: return all devices
	devices, err := s.registry.ListDevices(ctx)
	if err != nil {
		writeInternalError(w, "failed to list devices")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": devices, "count": len(devices)})
}

// handleGetDevice returns a single device by ID.
func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	dev, err := s.registry.GetDevice(r.Context(), id)
	if err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			writeNotFound(w, "device not found")
			return
		}
		writeInternalError(w, "failed to get device")
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

	dev, err := s.registry.GetDevice(r.Context(), id)
	if err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			writeNotFound(w, "device not found")
			return
		}
		writeInternalError(w, "failed to get device")
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

	// In dev mode, simulate the bridge confirmation loop: delay the state
	// write + WebSocket broadcast to mimic the real KNX bus round-trip time.
	// In production, only the bridge's confirmed state update (via MQTT)
	// should modify the database.
	var newState device.State
	if s.devMode {
		newState = commandToState(cmd.Command, cmd.Parameters, dev.State)
		go func(deviceID string, state device.State) {
			// Simulate bridge processing + bus round-trip (KNX typical: 100-300ms,
			// exaggerated here so the pending UI is clearly visible during dev)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			select {
			case <-time.After(800 * time.Millisecond):
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
		}(id, newState)
	}

	logFields := []any{
		"device_id", id,
		"command", cmd.Command,
		"parameters", cmd.Parameters,
		"command_id", commandID,
	}
	if s.devMode {
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
func commandToState(command string, params map[string]any, current device.State) device.State {
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
