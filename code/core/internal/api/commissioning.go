package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/nerrad567/gray-logic-core/internal/commissioning/etsimport"
	"github.com/nerrad567/gray-logic-core/internal/device"
)

// handleETSParse parses an uploaded ETS project file (.knxproj, .xml, or .csv)
// and returns the detected devices for preview before import.
//
// This is a two-phase import: parse returns suggestions, then import commits.
//
// Request: multipart/form-data with "file" field containing the ETS export.
// Response: ParseResult with detected devices, warnings, and statistics.
func (s *Server) handleETSParse(w http.ResponseWriter, r *http.Request) {
	// Limit request body size (already applied by middleware, but be explicit)
	r.Body = http.MaxBytesReader(w, r.Body, etsimport.MaxFileSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(etsimport.MaxFileSize); err != nil {
		writeBadRequest(w, "failed to parse multipart form: file may be too large")
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		s.logger.Error("ETS parse: failed to get file from form",
			"error", err,
			"content_type", r.Header.Get("Content-Type"),
		)
		writeBadRequest(w, "missing required 'file' field in form data")
		return
	}
	defer file.Close()

	// Debug: log what we received
	s.logger.Info("ETS parse: file received",
		"filename", header.Filename,
		"size", header.Size,
		"content_type", header.Header.Get("Content-Type"),
	)

	// Read file content
	data, err := io.ReadAll(file)
	if err != nil {
		s.logger.Error("ETS parse: failed to read file", "error", err)
		writeBadRequest(w, "failed to read uploaded file")
		return
	}

	s.logger.Info("ETS parse: file read complete",
		"bytes_read", len(data),
		"first_bytes", string(data[:min(50, len(data))]),
	)

	// Parse the file
	parser := etsimport.NewParser()
	result, err := parser.ParseBytes(data, header.Filename)
	if err != nil {
		// Log ALL parse errors with details
		s.logger.Error("ETS parse error",
			"error", err,
			"error_type", fmt.Sprintf("%T", err),
			"filename", header.Filename,
		)

		// Map specific errors to appropriate HTTP responses
		switch {
		case errors.Is(err, etsimport.ErrFileTooLarge):
			writeError(w, http.StatusRequestEntityTooLarge, "file_too_large",
				"file exceeds maximum size of 50MB")
		case errors.Is(err, etsimport.ErrInvalidFile):
			writeBadRequest(w, "invalid file format: expected .knxproj, .xml, or .csv")
		case errors.Is(err, etsimport.ErrCorruptArchive):
			writeBadRequest(w, "corrupt archive: unable to read .knxproj file")
		case errors.Is(err, etsimport.ErrNoGroupAddresses):
			writeBadRequest(w, "no group addresses found in file")
		case errors.Is(err, etsimport.ErrUnsupportedVersion):
			writeBadRequest(w, "unsupported ETS version")
		default:
			s.logger.Error("ETS parse failed", "error", err, "filename", header.Filename)
			writeInternalError(w, "failed to parse ETS file")
		}
		return
	}

	s.logger.Info("ETS file parsed",
		"import_id", result.ImportID,
		"filename", header.Filename,
		"format", result.Format,
		"devices", len(result.Devices),
		"unmapped", len(result.UnmappedAddresses),
		"warnings", len(result.Warnings),
	)

	writeJSON(w, http.StatusOK, result)
}

// ETSImportRequest is the request body for committing an ETS import.
type ETSImportRequest struct {
	// ImportID from the parse response (for audit trail).
	ImportID string `json:"import_id"`

	// Devices to import, potentially modified by user during preview.
	Devices []ETSDeviceImport `json:"devices"`

	// Options for import behaviour.
	Options ETSImportOptions `json:"options,omitempty"`
}

// ETSDeviceImport represents a device to import from ETS.
type ETSDeviceImport struct {
	// Import toggles whether this device should be imported.
	Import bool `json:"import"`

	// ID is the device identifier (can be modified from suggested).
	ID string `json:"id"`

	// Name is the display name (can be modified from suggested).
	Name string `json:"name"`

	// Type is the device type.
	Type string `json:"type"`

	// Domain is the device domain (lighting, blinds, climate, etc.).
	Domain string `json:"domain"`

	// RoomID is the room to assign the device to (optional).
	RoomID string `json:"room_id,omitempty"`

	// AreaID is the area to assign the device to (optional).
	AreaID string `json:"area_id,omitempty"`

	// Addresses are the KNX group addresses for this device.
	Addresses []ETSAddressImport `json:"addresses"`
}

// ETSAddressImport represents a group address mapping for import.
type ETSAddressImport struct {
	// GA is the group address (e.g., "1/2/3").
	GA string `json:"ga"`

	// Function is the address function (switch, brightness, position, etc.).
	Function string `json:"function"`

	// DPT is the datapoint type.
	DPT string `json:"dpt"`

	// Flags are the access flags (read, write, transmit).
	Flags []string `json:"flags,omitempty"`
}

// ETSImportOptions configures import behaviour.
type ETSImportOptions struct {
	// SkipExisting skips devices that already exist (by ID).
	SkipExisting bool `json:"skip_existing,omitempty"`

	// UpdateExisting updates existing devices instead of skipping.
	UpdateExisting bool `json:"update_existing,omitempty"`

	// DryRun validates the import without committing changes.
	DryRun bool `json:"dry_run,omitempty"`
}

// ETSImportResponse is the response from a successful import.
type ETSImportResponse struct {
	// ImportID for audit trail.
	ImportID string `json:"import_id"`

	// Created is the count of newly created devices.
	Created int `json:"created"`

	// Updated is the count of updated devices (if update_existing enabled).
	Updated int `json:"updated"`

	// Skipped is the count of skipped devices.
	Skipped int `json:"skipped"`

	// Errors are per-device errors that occurred during import.
	Errors []ETSImportError `json:"errors,omitempty"`
}

// ETSImportError represents an error for a specific device during import.
type ETSImportError struct {
	DeviceID string `json:"device_id"`
	Message  string `json:"message"`
}

// handleETSImport commits an ETS import, creating devices in the registry.
//
// This is phase 2 of the import: the user has reviewed the parse results
// and confirmed which devices to import.
func (s *Server) handleETSImport(w http.ResponseWriter, r *http.Request) {
	var req ETSImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if err := s.validateETSImportRequest(&req); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	response := s.processETSImport(r.Context(), &req)

	s.logger.Info("ETS import completed",
		"import_id", req.ImportID,
		"created", response.Created,
		"updated", response.Updated,
		"skipped", response.Skipped,
		"errors", len(response.Errors),
		"dry_run", req.Options.DryRun,
	)

	// Reload KNX bridge device mappings if devices were created/updated
	// This allows the bridge to control newly imported devices without restart
	if !req.Options.DryRun && (response.Created > 0 || response.Updated > 0) {
		if s.knxBridge != nil {
			s.knxBridge.ReloadDevices(r.Context())
			s.logger.Info("KNX bridge devices reloaded after ETS import")
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// validateETSImportRequest validates the import request fields.
func (s *Server) validateETSImportRequest(req *ETSImportRequest) error {
	if req.ImportID == "" {
		return errors.New("import_id is required")
	}
	if len(req.Devices) == 0 {
		return errors.New("at least one device is required for import")
	}
	return nil
}

// processETSImport processes all devices in the import request.
func (s *Server) processETSImport(ctx context.Context, req *ETSImportRequest) ETSImportResponse {
	response := ETSImportResponse{
		ImportID: req.ImportID,
	}

	for i := range req.Devices {
		devImport := &req.Devices[i]
		s.processETSDevice(ctx, devImport, &req.Options, &response)
	}

	return response
}

// processETSDevice handles importing a single device.
func (s *Server) processETSDevice(ctx context.Context, devImport *ETSDeviceImport, opts *ETSImportOptions, response *ETSImportResponse) {
	if !devImport.Import {
		response.Skipped++
		return
	}

	if devImport.ID == "" {
		response.Errors = append(response.Errors, ETSImportError{
			DeviceID: "(unknown)",
			Message:  "device ID is required",
		})
		return
	}

	if devImport.Name == "" {
		devImport.Name = devImport.ID
	}

	// Check if device already exists
	existingDev, err := s.registry.GetDevice(ctx, devImport.ID)
	deviceExists := err == nil && existingDev != nil

	if deviceExists {
		s.handleExistingDevice(ctx, devImport, opts, response)
	} else {
		s.createNewDevice(ctx, devImport, opts, response)
	}
}

// handleExistingDevice handles the case when a device already exists.
func (s *Server) handleExistingDevice(ctx context.Context, devImport *ETSDeviceImport, opts *ETSImportOptions, response *ETSImportResponse) {
	if opts.SkipExisting {
		response.Skipped++
		return
	}

	if !opts.UpdateExisting {
		response.Errors = append(response.Errors, ETSImportError{
			DeviceID: devImport.ID,
			Message:  "device already exists (use skip_existing or update_existing option)",
		})
		return
	}

	// Update existing device
	if opts.DryRun {
		response.Updated++
		return
	}

	dev := s.buildDeviceFromImport(*devImport)
	if err := s.registry.UpdateDevice(ctx, dev); err != nil {
		response.Errors = append(response.Errors, ETSImportError{
			DeviceID: devImport.ID,
			Message:  err.Error(),
		})
		return
	}

	response.Updated++
}

// createNewDevice creates a new device in the registry.
func (s *Server) createNewDevice(ctx context.Context, devImport *ETSDeviceImport, opts *ETSImportOptions, response *ETSImportResponse) {
	if opts.DryRun {
		response.Created++
		return
	}

	dev := s.buildDeviceFromImport(*devImport)
	if err := s.registry.CreateDevice(ctx, dev); err != nil {
		response.Errors = append(response.Errors, ETSImportError{
			DeviceID: devImport.ID,
			Message:  err.Error(),
		})
		return
	}

	response.Created++
}

// buildDeviceFromImport converts an ETSDeviceImport to a device.Device.
func (s *Server) buildDeviceFromImport(imp ETSDeviceImport) *device.Device {
	dev := &device.Device{
		ID:       imp.ID,
		Name:     imp.Name,
		Type:     device.DeviceType(imp.Type),
		Domain:   device.Domain(imp.Domain),
		Protocol: device.ProtocolKNX,
	}

	// Set optional room/area IDs
	if imp.RoomID != "" {
		dev.RoomID = &imp.RoomID
	}
	if imp.AreaID != "" {
		dev.AreaID = &imp.AreaID
	}

	// Build addresses map for KNX protocol - stored in Address field
	// KNX validation requires a top-level "group_address" key
	addresses := make(device.Address)
	var primaryGA string

	for _, addr := range imp.Addresses {
		// Store each function's address details
		addresses[addr.Function] = addr.GA

		// Select primary GA (prefer write-flagged, or first one)
		if primaryGA == "" {
			primaryGA = addr.GA
		}
		for _, flag := range addr.Flags {
			if flag == "write" && primaryGA == "" {
				primaryGA = addr.GA
			}
		}
	}

	// Add the required top-level group_address for KNX validation
	if primaryGA != "" {
		addresses["group_address"] = primaryGA
	}

	dev.Address = addresses

	// Derive capabilities from addresses
	dev.Capabilities = deriveCapabilitiesFromAddresses(imp.Addresses)

	return dev
}

// deriveCapabilitiesFromAddresses infers device capabilities from address functions.
func deriveCapabilitiesFromAddresses(addresses []ETSAddressImport) []device.Capability {
	caps := make(map[device.Capability]bool)

	for _, addr := range addresses {
		switch addr.Function {
		case "switch", "switch_status":
			caps[device.CapOnOff] = true
		case "brightness", "brightness_status":
			caps[device.CapDim] = true
		case "position", "position_status":
			caps[device.CapPosition] = true
		case "slat", "slat_status", "tilt":
			caps[device.CapTilt] = true
		case "move", "stop":
			// Move/stop are implicit in position capability for blinds
			caps[device.CapPosition] = true
		case "temperature":
			caps[device.CapTemperatureRead] = true
		case "setpoint":
			caps[device.CapTemperatureSet] = true
		case "humidity":
			caps[device.CapHumidityRead] = true
		case "lux":
			caps[device.CapLightLevelRead] = true
		case "presence":
			caps[device.CapPresenceDetect] = true
		}
	}

	result := make([]device.Capability, 0, len(caps))
	for cap := range caps {
		result = append(result, cap)
	}
	return result
}
