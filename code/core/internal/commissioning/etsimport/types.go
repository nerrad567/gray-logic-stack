package etsimport

import "time"

// ParseResult contains the complete result of parsing an ETS project file.
type ParseResult struct {
	// ImportID is a unique identifier for this parse session.
	// Used to correlate parse and import operations.
	ImportID string `json:"import_id"`

	// SourceFile is the original filename.
	SourceFile string `json:"source_file"`

	// Format is the detected file format (knxproj, xml, csv).
	Format string `json:"format"`

	// ETSVersion is the ETS version that created the project (if detectable).
	ETSVersion string `json:"ets_version,omitempty"`

	// ParsedAt is when the file was parsed.
	ParsedAt time.Time `json:"parsed_at"`

	// Statistics summarises the parse results.
	Statistics ParseStatistics `json:"statistics"`

	// Devices contains detected device configurations.
	Devices []DetectedDevice `json:"devices"`

	// UnmappedAddresses contains GAs that couldn't be assigned to a device.
	UnmappedAddresses []GroupAddress `json:"unmapped_addresses,omitempty"`

	// Locations contains building structure extracted from ETS.
	Locations []Location `json:"locations,omitempty"`

	// Warnings contains non-fatal issues encountered during parsing.
	Warnings []ParseWarning `json:"warnings,omitempty"`
}

// ParseStatistics summarises parse results.
type ParseStatistics struct {
	TotalGroupAddresses int `json:"total_group_addresses"`
	DetectedDevices     int `json:"detected_devices"`
	HighConfidence      int `json:"high_confidence"`
	MediumConfidence    int `json:"medium_confidence"`
	LowConfidence       int `json:"low_confidence"`
	UnmappedAddresses   int `json:"unmapped_addresses"`
}

// DetectedDevice represents a device inferred from group addresses.
type DetectedDevice struct {
	// SuggestedID is the auto-generated device ID (can be overridden).
	SuggestedID string `json:"suggested_id"`

	// SuggestedName is the human-readable name derived from GA names.
	SuggestedName string `json:"suggested_name"`

	// DetectedType is the inferred device type (dimmer, switch, blind, etc.).
	DetectedType string `json:"detected_type"`

	// Confidence is the detection confidence (0.0 to 1.0).
	Confidence float64 `json:"confidence"`

	// SuggestedDomain is the inferred domain (lighting, blinds, climate, etc.).
	SuggestedDomain string `json:"suggested_domain"`

	// SuggestedRoom is the room ID inferred from ETS hierarchy.
	SuggestedRoom string `json:"suggested_room,omitempty"`

	// SuggestedArea is the area ID inferred from ETS hierarchy.
	SuggestedArea string `json:"suggested_area,omitempty"`

	// Addresses contains the group addresses for this device.
	Addresses []DeviceAddress `json:"addresses"`

	// SourceLocation is the ETS hierarchy path (for reference).
	SourceLocation string `json:"source_location,omitempty"`
}

// DeviceAddress represents a single group address mapping for a device.
type DeviceAddress struct {
	// GA is the group address in "main/middle/sub" format.
	GA string `json:"ga"`

	// Name is the original name from ETS.
	Name string `json:"name"`

	// DPT is the datapoint type in "X.YYY" format (e.g., "1.001").
	DPT string `json:"dpt"`

	// SuggestedFunction is the inferred function (switch, brightness, position, etc.).
	SuggestedFunction string `json:"suggested_function"`

	// SuggestedFlags are the inferred access flags.
	SuggestedFlags []string `json:"suggested_flags"`

	// Description is the optional description from ETS.
	Description string `json:"description,omitempty"`
}

// GroupAddress represents a raw group address from ETS.
type GroupAddress struct {
	// Address in "main/middle/sub" format.
	Address string `json:"address"`

	// Name from ETS.
	Name string `json:"name"`

	// DPT in "X.YYY" format (may be empty if not specified in ETS).
	DPT string `json:"dpt,omitempty"`

	// Description from ETS.
	Description string `json:"description,omitempty"`

	// Location is the ETS hierarchy path.
	Location string `json:"location,omitempty"`

	// LinkedDevices contains ETS device references (if available).
	LinkedDevices []string `json:"linked_devices,omitempty"`

	// Reason explains why this GA was not mapped to a device.
	Reason string `json:"reason,omitempty"`
}

// Location represents a building structure element from ETS.
type Location struct {
	// ID is the generated location ID.
	ID string `json:"id"`

	// Name is the location name from ETS.
	Name string `json:"name"`

	// Type is the location type (building, floor, room, etc.).
	Type string `json:"type"`

	// ParentID is the parent location ID (empty for root).
	ParentID string `json:"parent_id,omitempty"`

	// SuggestedAreaID is the suggested Gray Logic area ID.
	SuggestedAreaID string `json:"suggested_area_id,omitempty"`

	// SuggestedRoomID is the suggested Gray Logic room ID.
	SuggestedRoomID string `json:"suggested_room_id,omitempty"`
}

// ParseWarning represents a non-fatal issue during parsing.
type ParseWarning struct {
	// Code is a machine-readable warning code.
	Code string `json:"code"`

	// Message is a human-readable description.
	Message string `json:"message"`

	// AffectedDevices lists device IDs affected by this warning.
	AffectedDevices []string `json:"affected_devices,omitempty"`

	// AffectedAddresses lists GAs affected by this warning.
	AffectedAddresses []string `json:"affected_addresses,omitempty"`
}

// ImportRequest represents a confirmed import operation.
type ImportRequest struct {
	// ImportID must match a previous parse result.
	ImportID string `json:"import_id"`

	// Devices contains the devices to import (with user modifications).
	Devices []ImportDevice `json:"devices"`

	// CreateRooms enables auto-creation of missing rooms.
	CreateRooms bool `json:"create_rooms"`

	// CreateAreas enables auto-creation of missing areas.
	CreateAreas bool `json:"create_areas"`

	// BackupBeforeImport creates a snapshot before importing.
	BackupBeforeImport bool `json:"backup_before_import"`
}

// ImportDevice represents a device to be imported (with user overrides).
type ImportDevice struct {
	// SuggestedID is the original suggested ID (for correlation).
	SuggestedID string `json:"suggested_id"`

	// ID is the final device ID (may be user-modified).
	ID string `json:"id"`

	// Name is the final device name (may be user-modified).
	Name string `json:"name"`

	// Type is the final device type.
	Type string `json:"type"`

	// Domain is the device domain.
	Domain string `json:"domain"`

	// RoomID is the room to assign the device to.
	RoomID string `json:"room_id,omitempty"`

	// AreaID is the area to assign the device to (if no room).
	AreaID string `json:"area_id,omitempty"`

	// Addresses contains the GA mappings.
	Addresses []ImportAddress `json:"addresses"`

	// Import indicates whether to import this device (false to skip).
	Import bool `json:"import"`
}

// ImportAddress represents a GA mapping for import.
type ImportAddress struct {
	// GA is the group address.
	GA string `json:"ga"`

	// Function is the address function (switch, brightness, etc.).
	Function string `json:"function"`

	// DPT is the datapoint type.
	DPT string `json:"dpt"`

	// Flags are the access flags (read, write, transmit).
	Flags []string `json:"flags"`
}

// ImportResult contains the result of an import operation.
type ImportResult struct {
	// Success indicates whether the import completed.
	Success bool `json:"success"`

	// ImportID is the import session ID.
	ImportID string `json:"import_id"`

	// BackupID is the backup snapshot ID (if backup was requested).
	BackupID string `json:"backup_id,omitempty"`

	// Results summarises what was created/updated.
	Results ImportResultStats `json:"results"`

	// NextSteps contains recommended actions after import.
	NextSteps []string `json:"next_steps,omitempty"`

	// Errors contains any errors encountered.
	Errors []string `json:"errors,omitempty"`
}

// ImportResultStats summarises import results.
type ImportResultStats struct {
	DevicesCreated      int  `json:"devices_created"`
	DevicesUpdated      int  `json:"devices_updated"`
	DevicesSkipped      int  `json:"devices_skipped"`
	RoomsCreated        int  `json:"rooms_created"`
	AreasCreated        int  `json:"areas_created"`
	BridgeConfigUpdated bool `json:"bridge_config_updated"`
}

// Confidence level thresholds.
const (
	ConfidenceHigh   = 0.80
	ConfidenceMedium = 0.50
)

// ConfidenceLevel returns a human-readable confidence level.
func ConfidenceLevel(c float64) string {
	switch {
	case c >= ConfidenceHigh:
		return "high"
	case c >= ConfidenceMedium:
		return "medium"
	default:
		return "low"
	}
}
