package etsimport

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"
)

// Test constants to avoid magic strings.
const (
	testTypeLightDimmer    = "light_dimmer"
	testTypeLightSwitch    = "light_switch"
	testTypeBlind          = "blind_position"
	testTypeTempSensor     = "temperature_sensor"
	testTypePresenceSensor = "presence_sensor"
	testTypeLightSensor    = "light_sensor"
	testTypeHeatingAct     = "heating_actuator"
)

func TestNormaliseDPT(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"DPST-1-1", "1.001"},
		{"DPST-5-1", "5.001"},
		{"DPST-9-1", "9.001"},
		{"DPST-14-68", "14.068"},
		{"DPT-1", "1.001"},
		{"DPT1", "1.001"},
		{"1.001", "1.001"},
		{"5.001", "5.001"},
		{"9.1", "9.001"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normaliseDPT(tt.input)
			if result != tt.expected {
				t.Errorf("normaliseDPT(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidGA(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1/2/3", true},
		{"0/0/0", true},
		{"15/7/255", true},
		{"1/2", true},
		{"123", true},
		{"invalid", false},
		{"1/2/3/4", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isValidGA(tt.input)
			if result != tt.expected {
				t.Errorf("isValidGA(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormaliseGA(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"3-level format unchanged", "1/2/3", "1/2/3"},
		{"3-level with zeros", "0/0/0", "0/0/0"},
		{"3-level max values", "15/7/255", "15/7/255"},
		{"integer 0", "0", "0/0/0"},
		{"integer 2048 (1/0/0)", "2048", "1/0/0"},
		{"integer 2305 (1/1/1)", "2305", "1/1/1"},
		{"integer 4352 (2/1/0)", "4352", "2/1/0"},
		{"2-level to 3-level", "1/256", "1/1/0"},
		{"empty string", "", ""},
		{"whitespace", "  1/2/3  ", "1/2/3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normaliseGA(tt.input)
			if result != tt.expected {
				t.Errorf("normaliseGA(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractNamePrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Traditional ETS-style names (suffix-based)
		{"Kitchen Light Switch", "Kitchen Light"},
		{"Kitchen Light Dimming", "Kitchen Light"},
		{"Kitchen Light Status", "Kitchen Light"},
		{"Living Room Blind Position", "Living Room Blind"},
		{"Bedroom Temperature", "Bedroom Temperature"},
		{"Simple", "Simple"},
		// KNXSim-style names (colon-separated)
		{"Living Room Ceiling Light : Switch", "Living Room Ceiling Light"},
		{"Living Room Ceiling Light : Brightness", "Living Room Ceiling Light"},
		{"Living Room Ceiling Light : Brightness Status", "Living Room Ceiling Light"},
		{"Kitchen Pendant Light : Switch Status", "Kitchen Pendant Light"},
		{"Living Room Blinds : Position", "Living Room Blinds"},
		{"Living Room Blinds : Slat", "Living Room Blinds"},
		// Mixed formats - "Control" is a suffix word so it gets stripped
		{"Ch-3 - Blinds Control : Step/stop", "Ch-3 - Blinds"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractNamePrefix(tt.input)
			if result != tt.expected {
				t.Errorf("extractNamePrefix(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Kitchen Light", "kitchen-light"},
		{"Living Room", "living-room"},
		{"Küche Licht", "k-che-licht"},
		{"Test 123", "test-123"},
		{"  Spaces  ", "spaces"},
		{"", "device"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := generateSlug(tt.input)
			if result != tt.expected {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConfidenceLevel(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0.95, "high"},
		{0.80, "high"},
		{0.79, "medium"},
		{0.50, "medium"},
		{0.49, "low"},
		{0.0, "low"},
	}

	for _, tt := range tests {
		result := ConfidenceLevel(tt.input)
		if result != tt.expected {
			t.Errorf("ConfidenceLevel(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestParseCSV(t *testing.T) {
	csv := `"Address","Name","DatapointType"
"1/0/0","Kitchen Light Switch","DPST-1-1"
"1/0/1","Kitchen Light Dimming","DPST-5-1"
"1/0/2","Kitchen Light Status","DPST-1-1"
`

	parser := NewParser()
	result, err := parser.ParseBytes([]byte(csv), "test.csv")
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if result.Format != "csv" {
		t.Errorf("Format = %q, want %q", result.Format, "csv")
	}

	// Should detect a dimmer (switch + brightness)
	if len(result.Devices) == 0 {
		t.Error("Expected at least one detected device")
	}

	// Check statistics
	if result.Statistics.TotalGroupAddresses < 3 {
		t.Errorf("TotalGroupAddresses = %d, want >= 3", result.Statistics.TotalGroupAddresses)
	}
}

func TestParseXML(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<GroupAddresses>
  <GroupRange Name="Lighting" Address="1">
    <GroupRange Name="Kitchen" Address="0">
      <GroupAddress Id="GA-1" Address="1/0/0" Name="Kitchen Light Switch" DatapointType="DPST-1-1"/>
      <GroupAddress Id="GA-2" Address="1/0/1" Name="Kitchen Light Dimming" DatapointType="DPST-5-1"/>
    </GroupRange>
  </GroupRange>
</GroupAddresses>`

	parser := NewParser()
	result, err := parser.ParseBytes([]byte(xml), "test.xml")
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if result.Format != "xml" {
		t.Errorf("Format = %q, want %q", result.Format, "xml")
	}

	// Check that location was extracted
	found := false
	for _, dev := range result.Devices {
		if strings.Contains(dev.SourceLocation, "Lighting") && strings.Contains(dev.SourceLocation, "Kitchen") {
			found = true
			break
		}
	}
	// Also check unmapped addresses
	for _, ga := range result.UnmappedAddresses {
		if strings.Contains(ga.Location, "Lighting") && strings.Contains(ga.Location, "Kitchen") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected location hierarchy to be extracted")
	}

	// Check that Location objects were created from hierarchy
	if len(result.Locations) == 0 {
		t.Error("Expected locations to be extracted from XML hierarchy")
	}
	// "Lighting" is a domain name and should be filtered out;
	// "Kitchen" should appear as a room
	hasKitchen := false
	for _, loc := range result.Locations {
		if loc.ID == "kitchen" && loc.Type == "room" {
			hasKitchen = true
		}
	}
	if !hasKitchen {
		t.Errorf("Expected kitchen room in locations, got %v", result.Locations)
	}
}

func TestParseKNXProj(t *testing.T) {
	// Create a minimal .knxproj ZIP file
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Add GroupAddresses.xml
	gaXML := `<?xml version="1.0" encoding="utf-8"?>
<GroupAddresses>
  <GroupRange Name="Lighting" Address="1">
    <GroupAddress Id="GA-1" Address="1/1/0" Name="Living Room Light Switch" DatapointType="DPST-1-1"/>
    <GroupAddress Id="GA-2" Address="1/1/1" Name="Living Room Light Brightness" DatapointType="DPST-5-1"/>
    <GroupAddress Id="GA-3" Address="1/1/2" Name="Living Room Light Status" DatapointType="DPST-1-1"/>
  </GroupRange>
  <GroupRange Name="Blinds" Address="2">
    <GroupAddress Id="GA-4" Address="2/1/0" Name="Living Room Blind Move" DatapointType="DPST-1-8"/>
    <GroupAddress Id="GA-5" Address="2/1/1" Name="Living Room Blind Position" DatapointType="DPST-5-1"/>
  </GroupRange>
</GroupAddresses>`

	f, createErr := w.Create("P-TEST/GroupAddresses.xml")
	if createErr != nil {
		t.Fatalf("Failed to create zip entry: %v", createErr)
	}
	if _, writeErr := f.Write([]byte(gaXML)); writeErr != nil {
		t.Fatalf("Failed to write zip content: %v", writeErr)
	}

	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("Failed to close zip: %v", closeErr)
	}

	parser := NewParser()
	result, err := parser.ParseBytes(buf.Bytes(), "test.knxproj")
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if result.Format != "knxproj" {
		t.Errorf("Format = %q, want %q", result.Format, "knxproj")
	}

	// Should detect devices
	if len(result.Devices) < 2 {
		t.Errorf("Expected at least 2 detected devices, got %d", len(result.Devices))
	}

	// Check for dimmer detection
	foundDimmer := false
	foundBlind := false
	for _, dev := range result.Devices {
		if dev.DetectedType == testTypeLightDimmer {
			foundDimmer = true
			if dev.Confidence < ConfidenceHigh {
				t.Errorf("Dimmer confidence = %v, expected >= %v", dev.Confidence, ConfidenceHigh)
			}
		}
		if strings.Contains(dev.DetectedType, "blind") {
			foundBlind = true
		}
	}

	if !foundDimmer {
		t.Error("Expected to detect a dimmer device")
	}
	if !foundBlind {
		t.Error("Expected to detect a blind device")
	}

	// Check import ID format
	if !strings.HasPrefix(result.ImportID, "imp_") {
		t.Errorf("ImportID = %q, want prefix 'imp_'", result.ImportID)
	}
}

func TestDetectionRuleDimmer(t *testing.T) {
	addresses := []GroupAddress{
		{Address: "1/0/0", Name: "Kitchen Light Switch", DPT: "1.001"},
		{Address: "1/0/1", Name: "Kitchen Light Brightness", DPT: "5.001"},
		{Address: "1/0/2", Name: "Kitchen Light Status", DPT: "1.001"},
	}

	rules := DefaultDetectionRules()
	var dimmerRule *DetectionRule
	for i := range rules {
		if rules[i].Name == "light_dimmer" {
			dimmerRule = &rules[i]
			break
		}
	}

	if dimmerRule == nil {
		t.Fatal("Dimmer rule not found in default rules")
	}

	device := dimmerRule.TryMatch("Kitchen Light", addresses)
	if device == nil {
		t.Fatal("Expected dimmer rule to match")
	}

	if device.DetectedType != testTypeLightDimmer {
		t.Errorf("DetectedType = %q, want %q", device.DetectedType, testTypeLightDimmer)
	}

	if device.SuggestedDomain != "lighting" {
		t.Errorf("SuggestedDomain = %q, want %q", device.SuggestedDomain, "lighting")
	}

	if len(device.Addresses) < 2 {
		t.Errorf("Expected at least 2 addresses, got %d", len(device.Addresses))
	}

	// Check that switch and brightness functions are assigned
	functions := make(map[string]bool)
	for _, addr := range device.Addresses {
		functions[addr.SuggestedFunction] = true
	}

	if !functions["switch"] {
		t.Error("Expected 'switch' function to be assigned")
	}
	if !functions["brightness"] {
		t.Error("Expected 'brightness' function to be assigned")
	}
}

func TestDetectionRuleSwitch(t *testing.T) {
	addresses := []GroupAddress{
		{Address: "1/0/0", Name: "Hallway Light Switch", DPT: "1.001"},
		{Address: "1/0/1", Name: "Hallway Light Status", DPT: "1.001"},
	}

	rules := DefaultDetectionRules()
	var switchRule *DetectionRule
	for i := range rules {
		if rules[i].Name == "light_switch" {
			switchRule = &rules[i]
			break
		}
	}

	if switchRule == nil {
		t.Fatal("Switch rule not found in default rules")
	}

	device := switchRule.TryMatch("Hallway Light", addresses)
	if device == nil {
		t.Fatal("Expected switch rule to match")
	}

	if device.DetectedType != testTypeLightSwitch {
		t.Errorf("DetectedType = %q, want %q", device.DetectedType, testTypeLightSwitch)
	}
}

func TestDetectionRuleBlind(t *testing.T) {
	addresses := []GroupAddress{
		{Address: "2/0/0", Name: "Office Blind Move", DPT: "1.008"},
		{Address: "2/0/1", Name: "Office Blind Position", DPT: "5.001"},
		{Address: "2/0/2", Name: "Office Blind Stop", DPT: "1.007"},
	}

	rules := DefaultDetectionRules()
	var blindRule *DetectionRule
	for i := range rules {
		if rules[i].Name == "blind_position" {
			blindRule = &rules[i]
			break
		}
	}

	if blindRule == nil {
		t.Fatal("Blind rule not found in default rules")
	}

	device := blindRule.TryMatch("Office Blind", addresses)
	if device == nil {
		t.Fatal("Expected blind rule to match")
	}

	if device.DetectedType != testTypeBlind {
		t.Errorf("DetectedType = %q, want %q", device.DetectedType, testTypeBlind)
	}

	if device.SuggestedDomain != "blinds" {
		t.Errorf("SuggestedDomain = %q, want %q", device.SuggestedDomain, "blinds")
	}
}

func TestDetectionRuleTemperatureSensor(t *testing.T) {
	addresses := []GroupAddress{
		{Address: "4/0/0", Name: "Living Room Temperature", DPT: "9.001"},
	}

	rules := DefaultDetectionRules()
	var tempRule *DetectionRule
	for i := range rules {
		if rules[i].Name == "temperature_sensor" {
			tempRule = &rules[i]
			break
		}
	}

	if tempRule == nil {
		t.Fatal("Temperature sensor rule not found in default rules")
	}

	device := tempRule.TryMatch("Living Room Temperature", addresses)
	if device == nil {
		t.Fatal("Expected temperature sensor rule to match")
	}

	if device.DetectedType != testTypeTempSensor {
		t.Errorf("DetectedType = %q, want %q", device.DetectedType, testTypeTempSensor)
	}

	if device.Confidence < 0.9 {
		t.Errorf("Confidence = %v, expected >= 0.9", device.Confidence)
	}
}

func TestMatchesDPT(t *testing.T) {
	tests := []struct {
		actual   string
		pattern  string
		expected bool
	}{
		{"1.001", "1.001", true},
		{"1.001", "1.*", true},
		{"1.008", "1.*", true},
		{"5.001", "5.001", true},
		{"5.001", "1.001", false},
		{"", "1.001", false},
		{"1.001", "", false},
	}

	for _, tt := range tests {
		name := tt.actual + "_" + tt.pattern
		t.Run(name, func(t *testing.T) {
			result := matchesDPT(tt.actual, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesDPT(%q, %q) = %v, want %v", tt.actual, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestParseEmptyFile(t *testing.T) {
	parser := NewParser()
	_, err := parser.ParseBytes([]byte{}, "empty.csv")
	if !errors.Is(err, ErrNoGroupAddresses) {
		t.Errorf("Expected ErrNoGroupAddresses, got %v", err)
	}
}

func TestParseInvalidZip(t *testing.T) {
	parser := NewParser()
	_, err := parser.ParseBytes([]byte("not a zip file"), "test.knxproj")
	if err == nil {
		t.Error("Expected error for invalid zip")
	}
}

func TestParseFileTooLarge(t *testing.T) {
	// Create data larger than MaxFileSize
	data := make([]byte, MaxFileSize+1)
	parser := NewParser()
	_, err := parser.ParseBytes(data, "large.knxproj")
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("Expected ErrFileTooLarge, got %v", err)
	}
}

// ─── extractLocations Tests ────────────────────────────────────────

func TestExtractLocations_DomainFirst(t *testing.T) {
	parser := NewParser()
	result := &ParseResult{
		UnmappedAddresses: []GroupAddress{
			{Address: "1/0/0", Name: "Kitchen Light", Location: "Lighting > Kitchen"},
			{Address: "1/0/1", Name: "Living Room Light", Location: "Lighting > Living Room"},
			{Address: "2/0/0", Name: "Kitchen Temp", Location: "HVAC > Kitchen"},
		},
	}

	parser.extractLocations(result)

	// "Lighting" and "HVAC" are domains — should be filtered out
	// "Kitchen" and "Living Room" should appear as rooms
	// "Kitchen" appears under both domains — should be deduplicated
	rooms := 0
	for _, loc := range result.Locations {
		if loc.Type == "room" {
			rooms++
		}
	}
	if rooms != 2 {
		t.Errorf("Expected 2 rooms, got %d: %+v", rooms, result.Locations)
	}

	// Kitchen should be deduplicated
	kitchenCount := 0
	for _, loc := range result.Locations {
		if loc.ID == "kitchen" {
			kitchenCount++
		}
	}
	if kitchenCount != 1 {
		t.Errorf("Expected 1 kitchen location, got %d", kitchenCount)
	}
}

func TestExtractLocations_LocationFirst(t *testing.T) {
	parser := NewParser()
	result := &ParseResult{
		UnmappedAddresses: []GroupAddress{
			{Address: "1/0/0", Name: "Light", Location: "Ground Floor > Living Room"},
			{Address: "1/0/1", Name: "Light", Location: "Ground Floor > Kitchen"},
			{Address: "2/0/0", Name: "Light", Location: "First Floor > Bedroom"},
		},
	}

	parser.extractLocations(result)

	// "Ground Floor" and "First Floor" are non-leaf = areas (floors)
	// "Living Room", "Kitchen", "Bedroom" are leaves = rooms
	areas := 0
	rooms := 0
	for _, loc := range result.Locations {
		switch loc.Type {
		case "floor":
			areas++
		case "room":
			rooms++
		}
	}
	if areas != 2 {
		t.Errorf("Expected 2 areas, got %d: %+v", areas, result.Locations)
	}
	if rooms != 3 {
		t.Errorf("Expected 3 rooms, got %d: %+v", rooms, result.Locations)
	}

	// Areas should come before rooms in sorted output
	firstRoomIdx := -1
	lastAreaIdx := -1
	for i, loc := range result.Locations {
		if loc.Type == "floor" {
			lastAreaIdx = i
		}
		if loc.Type == "room" && firstRoomIdx == -1 {
			firstRoomIdx = i
		}
	}
	if lastAreaIdx > firstRoomIdx && firstRoomIdx >= 0 {
		t.Error("Expected areas to be sorted before rooms")
	}
}

func TestExtractLocations_ThreeLevel(t *testing.T) {
	parser := NewParser()
	result := &ParseResult{
		UnmappedAddresses: []GroupAddress{
			{Address: "1/0/0", Name: "Light", Location: "Lighting > Ground Floor > Kitchen"},
			{Address: "1/0/1", Name: "Light", Location: "Lighting > Ground Floor > Living Room"},
		},
	}

	parser.extractLocations(result)

	// "Lighting" is a domain — filtered out
	// "Ground Floor" is an intermediate node — area/floor
	// "Kitchen" and "Living Room" are leaves — rooms
	areas := 0
	rooms := 0
	for _, loc := range result.Locations {
		switch loc.Type {
		case "floor":
			areas++
		case "room":
			rooms++
		}
	}
	if areas != 1 {
		t.Errorf("Expected 1 area, got %d: %+v", areas, result.Locations)
	}
	if rooms != 2 {
		t.Errorf("Expected 2 rooms, got %d: %+v", rooms, result.Locations)
	}

	// Check parent-child relationship: rooms should have Ground Floor as parent
	for _, loc := range result.Locations {
		if loc.Type == "room" {
			if loc.ParentID != "ground-floor" {
				t.Errorf("Room %q has ParentID %q, want %q", loc.ID, loc.ParentID, "ground-floor")
			}
			if loc.SuggestedAreaID != "ground-floor" {
				t.Errorf("Room %q has SuggestedAreaID %q, want %q", loc.ID, loc.SuggestedAreaID, "ground-floor")
			}
		}
	}
}

func TestExtractLocations_Empty(t *testing.T) {
	parser := NewParser()
	result := &ParseResult{}
	parser.extractLocations(result)
	if len(result.Locations) != 0 {
		t.Errorf("Expected 0 locations for empty input, got %d", len(result.Locations))
	}
}

func TestExtractLocations_NoLocationPaths(t *testing.T) {
	parser := NewParser()
	result := &ParseResult{
		UnmappedAddresses: []GroupAddress{
			{Address: "1/0/0", Name: "Light"}, // No Location field
		},
	}
	parser.extractLocations(result)
	if len(result.Locations) != 0 {
		t.Errorf("Expected 0 locations when no paths, got %d", len(result.Locations))
	}
}

func TestExtractLocations_FromDeviceSourceLocation(t *testing.T) {
	parser := NewParser()
	result := &ParseResult{
		Devices: []DetectedDevice{
			{SuggestedID: "dev-1", SourceLocation: "Lighting > Bathroom"},
			{SuggestedID: "dev-2", SourceLocation: "Lighting > Hallway"},
		},
	}

	parser.extractLocations(result)

	rooms := 0
	for _, loc := range result.Locations {
		if loc.Type == "room" {
			rooms++
		}
	}
	if rooms != 2 {
		t.Errorf("Expected 2 rooms from device SourceLocation, got %d", rooms)
	}
}

// --- Detection rule tests for presence sensor, heating actuator, and ordering ---

func TestDetectionRulePresenceSensor(t *testing.T) {
	addresses := []GroupAddress{
		{Address: "4/0/0", Name: "Living Presence", DPT: "1.018"},
		{Address: "4/0/1", Name: "Living Lux", DPT: "9.004"},
	}

	rules := DefaultDetectionRules()
	var presenceRule *DetectionRule
	for i := range rules {
		if rules[i].Name == testTypePresenceSensor {
			presenceRule = &rules[i]
			break
		}
	}

	if presenceRule == nil {
		t.Fatal("Presence sensor rule not found in default rules")
	}

	device := presenceRule.TryMatch("Living", addresses)
	if device == nil {
		t.Fatal("Expected presence sensor rule to match")
	}

	if device.DetectedType != testTypePresenceSensor {
		t.Errorf("DetectedType = %q, want %q", device.DetectedType, testTypePresenceSensor)
	}

	if device.SuggestedDomain != "sensor" {
		t.Errorf("SuggestedDomain = %q, want %q", device.SuggestedDomain, "sensor")
	}
}

func TestDetectionPresenceBeatsLightSensor(t *testing.T) {
	// When a device has both presence (DPT 1.*) and lux (DPT 9.004) GAs,
	// the full detection pipeline should classify it as presence_sensor,
	// not light_sensor.
	addresses := []GroupAddress{
		{Address: "4/0/0", Name: "Presence Living : Presence", DPT: "1.018"},
		{Address: "4/0/1", Name: "Presence Living : Lux", DPT: "9.004"},
	}

	parser := NewParser()
	device := parser.tryDetectDevice("Presence Living", addresses)
	if device == nil {
		t.Fatal("Expected device detection to succeed")
	}

	if device.DetectedType != testTypePresenceSensor {
		t.Errorf("DetectedType = %q, want %q (should not be light_sensor)", device.DetectedType, testTypePresenceSensor)
	}
}

func TestDetectionRuleHeatingActuator(t *testing.T) {
	addresses := []GroupAddress{
		{Address: "3/0/0", Name: "Heating Actuator : Ch1 Valve", DPT: "5.001"},
		{Address: "3/0/1", Name: "Heating Actuator : Ch1 Valve Status", DPT: "5.001"},
	}

	rules := DefaultDetectionRules()
	var heatingRule *DetectionRule
	for i := range rules {
		if rules[i].Name == testTypeHeatingAct {
			heatingRule = &rules[i]
			break
		}
	}

	if heatingRule == nil {
		t.Fatal("Heating actuator rule not found in default rules")
	}

	device := heatingRule.TryMatch("Heating Actuator", addresses)
	if device == nil {
		t.Fatal("Expected heating actuator rule to match")
	}

	if device.DetectedType != testTypeHeatingAct {
		t.Errorf("DetectedType = %q, want %q", device.DetectedType, testTypeHeatingAct)
	}

	if device.SuggestedDomain != "climate" {
		t.Errorf("SuggestedDomain = %q, want %q", device.SuggestedDomain, "climate")
	}
}

func TestDetectionHeatingActuatorNotBlind(t *testing.T) {
	// Valve DPT 5.001 GAs should NOT be misdetected as blind_tilt.
	addresses := []GroupAddress{
		{Address: "3/0/0", Name: "Heating Actuator : Ch1 Valve", DPT: "5.001"},
		{Address: "3/0/1", Name: "Heating Actuator : Ch1 Valve Status", DPT: "5.001"},
		{Address: "3/0/2", Name: "Heating Actuator : Ch2 Valve", DPT: "5.001"},
		{Address: "3/0/3", Name: "Heating Actuator : Ch2 Valve Status", DPT: "5.001"},
	}

	parser := NewParser()
	device := parser.tryDetectDevice("Heating Actuator", addresses)
	if device == nil {
		t.Fatal("Expected device detection to succeed")
	}

	if device.DetectedType == "blind_tilt" || device.DetectedType == "blind_position" {
		t.Errorf("DetectedType = %q, valve GAs should not be detected as a blind type", device.DetectedType)
	}

	if device.DetectedType != testTypeHeatingAct {
		t.Errorf("DetectedType = %q, want %q", device.DetectedType, testTypeHeatingAct)
	}
}

func TestStandaloneLuxSensorStillWorks(t *testing.T) {
	// Regression test: a pure lux sensor (no presence GA) should still
	// be detected as light_sensor after reordering presence above it.
	addresses := []GroupAddress{
		{Address: "4/0/0", Name: "Outdoor Brightness", DPT: "9.004"},
	}

	parser := NewParser()
	device := parser.tryDetectDevice("Outdoor Brightness", addresses)
	if device == nil {
		t.Fatal("Expected device detection to succeed")
	}

	if device.DetectedType != testTypeLightSensor {
		t.Errorf("DetectedType = %q, want %q", device.DetectedType, testTypeLightSensor)
	}
}

// ─── Tier 1: Function Type Mapping Tests ───────────────────────────

func TestFunctionTypeMapping_SwitchableLight(t *testing.T) {
	devType, domain, confidence := functionTypeToDeviceType("SwitchableLight", "")
	if devType != "light_switch" {
		t.Errorf("devType = %q, want %q", devType, "light_switch")
	}
	if domain != "lighting" {
		t.Errorf("domain = %q, want %q", domain, "lighting")
	}
	if confidence != 0.99 {
		t.Errorf("confidence = %v, want 0.99", confidence)
	}
}

func TestFunctionTypeMapping_DimmableLight(t *testing.T) {
	devType, domain, confidence := functionTypeToDeviceType("DimmableLight", "")
	if devType != "light_dimmer" {
		t.Errorf("devType = %q, want %q", devType, "light_dimmer")
	}
	if domain != "lighting" {
		t.Errorf("domain = %q, want %q", domain, "lighting")
	}
	if confidence != 0.99 {
		t.Errorf("confidence = %v, want 0.99", confidence)
	}
}

func TestFunctionTypeMapping_Sunblind(t *testing.T) {
	devType, domain, confidence := functionTypeToDeviceType("Sunblind", "")
	if devType != "blind_position" {
		t.Errorf("devType = %q, want %q", devType, "blind_position")
	}
	if domain != "blinds" {
		t.Errorf("domain = %q, want %q", domain, "blinds")
	}
	if confidence != 0.95 {
		t.Errorf("confidence = %v, want 0.95", confidence)
	}
}

func TestFunctionTypeMapping_HeatingRadiator(t *testing.T) {
	devType, domain, confidence := functionTypeToDeviceType("HeatingRadiator", "")
	if devType != "thermostat" {
		t.Errorf("devType = %q, want %q", devType, "thermostat")
	}
	if domain != "climate" {
		t.Errorf("domain = %q, want %q", domain, "climate")
	}
	if confidence != 0.99 {
		t.Errorf("confidence = %v, want 0.99", confidence)
	}
}

func TestFunctionTypeMapping_HeatingFloor(t *testing.T) {
	devType, domain, confidence := functionTypeToDeviceType("HeatingFloor", "")
	if devType != "heating_actuator" {
		t.Errorf("devType = %q, want %q", devType, "heating_actuator")
	}
	if domain != "climate" {
		t.Errorf("domain = %q, want %q", domain, "climate")
	}
	if confidence != 0.99 {
		t.Errorf("confidence = %v, want 0.99", confidence)
	}
}

func TestFunctionTypeMapping_CustomWithComment(t *testing.T) {
	devType, domain, confidence := functionTypeToDeviceType("Custom", "presence_detector")
	if devType != "presence_sensor" {
		t.Errorf("devType = %q, want %q", devType, "presence_sensor")
	}
	if domain != "sensor" {
		t.Errorf("domain = %q, want %q", domain, "sensor")
	}
	if confidence != 0.98 {
		t.Errorf("confidence = %v, want 0.98", confidence)
	}
}

func TestFunctionTypeMapping_CustomUnknown(t *testing.T) {
	devType, _, confidence := functionTypeToDeviceType("Custom", "")
	if devType != "" {
		t.Errorf("devType = %q, want empty for unknown Custom", devType)
	}
	if confidence != 0 {
		t.Errorf("confidence = %v, want 0", confidence)
	}
}

func TestFunctionTypeMapping_UnknownType(t *testing.T) {
	devType, _, confidence := functionTypeToDeviceType("SomethingNew", "")
	if devType != "" {
		t.Errorf("devType = %q, want empty for unknown type", devType)
	}
	if confidence != 0 {
		t.Errorf("confidence = %v, want 0", confidence)
	}
}

func TestCommentToDeviceType_AllCategories(t *testing.T) {
	tests := []struct {
		comment    string
		wantType   string
		wantDomain string
	}{
		// Sensors
		{"temperature_sensor", "temperature_sensor", "sensor"},
		{"humidity_sensor", "humidity_sensor", "sensor"},
		{"co2_sensor", "co2_sensor", "sensor"},
		{"weather_station", "weather_station", "sensor"},
		// Energy
		{"energy_meter", "energy_meter", "energy"},
		{"solar_inverter", "solar_inverter", "energy"},
		{"ev_charger", "ev_charger", "energy"},
		// Controls
		{"scene_controller", "scene_controller", "lighting"},
		{"push_button_4", "push_button", "lighting"},
		{"binary_input", "binary_input", "sensor"},
		// System
		{"ip_router", "ip_router", "energy"},
		{"power_supply", "power_supply", "energy"},
		// Lighting actuators
		{"switch_actuator_8ch", "light_switch", "lighting"},
		{"dimmer_actuator_4ch", "light_dimmer", "lighting"},
	}

	for _, tt := range tests {
		t.Run(tt.comment, func(t *testing.T) {
			devType, domain, confidence := commentToDeviceType(tt.comment)
			if devType != tt.wantType {
				t.Errorf("devType = %q, want %q", devType, tt.wantType)
			}
			if domain != tt.wantDomain {
				t.Errorf("domain = %q, want %q", domain, tt.wantDomain)
			}
			if confidence != 0.98 {
				t.Errorf("confidence = %v, want 0.98", confidence)
			}
		})
	}
}

// ─── Tier 1 Integration: Full .knxproj with Functions ──────────────

func TestParseKNXProjWithFunctions(t *testing.T) { //nolint:gocognit // comprehensive integration test
	// Build a .knxproj ZIP with 0.xml containing Topology, ManufacturerData,
	// GroupAddresses, and Trades (Functions) — mimicking KNXSim export.
	projectXML := `<?xml version="1.0" encoding="utf-8"?>
<KNX xmlns="http://knx.org/xml/project/21" CreatedBy="KNXSim">
  <Project Id="P-TEST" Name="Test Project">
    <ProjectInformation Name="Test Project"/>
    <Installations>
      <Installation Name="Test" InstallationId="0">
        <GroupAddresses>
          <GroupRanges>
            <GroupRange Id="GR-1" Name="Lighting">
              <GroupRange Id="GR-2" Name="Ground Floor">
                <GroupRange Id="GR-3" Name="Kitchen">
                  <GroupAddress Id="GA-001" Address="1/0/1" Name="Kitchen Light : Switch" DatapointType="DPST-1-1"/>
                  <GroupAddress Id="GA-002" Address="1/0/2" Name="Kitchen Light : Brightness" DatapointType="DPST-5-1"/>
                  <GroupAddress Id="GA-003" Address="1/0/3" Name="Kitchen Light : Switch Status" DatapointType="DPST-1-1"/>
                </GroupRange>
              </GroupRange>
            </GroupRange>
            <GroupRange Id="GR-4" Name="Climate">
              <GroupRange Id="GR-5" Name="Ground Floor">
                <GroupRange Id="GR-6" Name="Kitchen">
                  <GroupAddress Id="GA-004" Address="3/0/1" Name="Kitchen Thermostat : Temperature" DatapointType="DPST-9-1"/>
                  <GroupAddress Id="GA-005" Address="3/0/2" Name="Kitchen Thermostat : Setpoint" DatapointType="DPST-9-1"/>
                </GroupRange>
              </GroupRange>
            </GroupRange>
            <GroupRange Id="GR-7" Name="Sensors">
              <GroupRange Id="GR-8" Name="Ground Floor">
                <GroupRange Id="GR-9" Name="Kitchen">
                  <GroupAddress Id="GA-006" Address="4/0/1" Name="Kitchen Presence : Presence" DatapointType="DPST-1-18"/>
                </GroupRange>
              </GroupRange>
            </GroupRange>
          </GroupRanges>
        </GroupAddresses>
        <Topology>
          <Area Id="A-1" Address="1" Name="Backbone">
            <Line Id="L-1" Address="1" Name="Main Line">
              <DeviceInstance Id="D-0001" Name="Dimming Actuator 4-fold"
                IndividualAddress="1.1.1"
                ProductRefId="M-0083_H-0001-HP-0001"
                ApplicationProgramRef="M-0083_A-0001">
                <ComObjectInstanceRefs>
                  <ComObjectInstanceRef RefId="D-0001-CO-001" DatapointType="DPST-1-1">
                    <Connectors><Send GroupAddressRefId="GA-001"/></Connectors>
                  </ComObjectInstanceRef>
                  <ComObjectInstanceRef RefId="D-0001-CO-002" DatapointType="DPST-5-1">
                    <Connectors><Send GroupAddressRefId="GA-002"/></Connectors>
                  </ComObjectInstanceRef>
                  <ComObjectInstanceRef RefId="D-0001-CO-003" DatapointType="DPST-1-1">
                    <Connectors><Send GroupAddressRefId="GA-003"/></Connectors>
                  </ComObjectInstanceRef>
                </ComObjectInstanceRefs>
              </DeviceInstance>
              <DeviceInstance Id="D-0002" Name="Room Thermostat"
                IndividualAddress="1.1.2"
                ProductRefId="M-0083_H-0002-HP-0001"
                ApplicationProgramRef="M-0083_A-0002">
                <ComObjectInstanceRefs>
                  <ComObjectInstanceRef RefId="D-0002-CO-001" DatapointType="DPST-9-1">
                    <Connectors><Send GroupAddressRefId="GA-004"/></Connectors>
                  </ComObjectInstanceRef>
                  <ComObjectInstanceRef RefId="D-0002-CO-002" DatapointType="DPST-9-1">
                    <Connectors><Send GroupAddressRefId="GA-005"/></Connectors>
                  </ComObjectInstanceRef>
                </ComObjectInstanceRefs>
              </DeviceInstance>
            </Line>
          </Area>
        </Topology>
        <Trades>
          <Trade Id="T-1" Name="Kitchen Light">
            <Function Id="F-1" Name="Kitchen Light" Type="DimmableLight">
              <GroupAddressRefs>
                <GroupAddressRef RefId="GA-001" Name="switch"/>
                <GroupAddressRef RefId="GA-002" Name="brightness"/>
                <GroupAddressRef RefId="GA-003" Name="switch_status"/>
              </GroupAddressRefs>
            </Function>
          </Trade>
          <Trade Id="T-2" Name="Kitchen Thermostat">
            <Function Id="F-2" Name="Kitchen Thermostat" Type="HeatingRadiator">
              <GroupAddressRefs>
                <GroupAddressRef RefId="GA-004" Name="temperature"/>
                <GroupAddressRef RefId="GA-005" Name="setpoint"/>
              </GroupAddressRefs>
            </Function>
          </Trade>
          <Trade Id="T-3" Name="Kitchen Presence">
            <Function Id="F-3" Name="Kitchen Presence" Type="Custom" Comment="presence_detector">
              <GroupAddressRefs>
                <GroupAddressRef RefId="GA-006" Name="presence"/>
              </GroupAddressRefs>
            </Function>
          </Trade>
        </Trades>
      </Installation>
    </Installations>
  </Project>
  <ManufacturerData>
    <Manufacturer Id="M-0083" Name="ABB">
      <Hardware Id="M-0083_H-0001" Name="UD/S 4.315.2.1"/>
      <Hardware Id="M-0083_H-0002" Name="RTC"/>
      <ApplicationProgram Id="M-0083_A-0001" Name="Dimming Actuator 4-fold" ApplicationVersion="1"/>
      <ApplicationProgram Id="M-0083_A-0002" Name="Room Thermostat Controller" ApplicationVersion="1"/>
    </Manufacturer>
  </ManufacturerData>
</KNX>`

	// Create a .knxproj ZIP containing this XML as 0.xml
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("P-TEST/0.xml")
	if err != nil {
		t.Fatalf("Failed to create zip entry: %v", err)
	}
	if _, err := f.Write([]byte(projectXML)); err != nil { //nolint:govet // shadow: idiomatic err check
		t.Fatalf("Failed to write zip content: %v", err)
	}
	if err := w.Close(); err != nil { //nolint:govet // shadow: idiomatic err check
		t.Fatalf("Failed to close zip: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseBytes(buf.Bytes(), "test.knxproj")
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	// Should have 3 Tier 1 devices
	if len(result.Devices) < 3 {
		t.Fatalf("Expected at least 3 devices, got %d", len(result.Devices))
	}

	// Verify each device type and confidence
	devicesByType := make(map[string]*DetectedDevice)
	for i := range result.Devices {
		dev := &result.Devices[i]
		devicesByType[dev.DetectedType] = dev
	}

	// Dimmer
	if dimmer, ok := devicesByType["light_dimmer"]; ok {
		if dimmer.Confidence != 0.99 {
			t.Errorf("Dimmer confidence = %v, want 0.99", dimmer.Confidence)
		}
		if dimmer.FunctionType != "DimmableLight" {
			t.Errorf("Dimmer FunctionType = %q, want %q", dimmer.FunctionType, "DimmableLight")
		}
		if len(dimmer.Addresses) != 3 {
			t.Errorf("Dimmer addresses = %d, want 3", len(dimmer.Addresses))
		}
		if dimmer.Manufacturer != "ABB" {
			t.Errorf("Dimmer Manufacturer = %q, want %q", dimmer.Manufacturer, "ABB")
		}
		if dimmer.ProductModel != "UD/S 4.315.2.1" {
			t.Errorf("Dimmer ProductModel = %q, want %q", dimmer.ProductModel, "UD/S 4.315.2.1")
		}
		if dimmer.ApplicationProgram != "Dimming Actuator 4-fold" {
			t.Errorf("Dimmer ApplicationProgram = %q, want %q", dimmer.ApplicationProgram, "Dimming Actuator 4-fold")
		}
		if dimmer.IndividualAddress != "1.1.1" {
			t.Errorf("Dimmer IndividualAddress = %q, want %q", dimmer.IndividualAddress, "1.1.1")
		}
	} else {
		t.Error("Expected light_dimmer in Tier 1 results")
	}

	// Thermostat
	if therm, ok := devicesByType["thermostat"]; ok {
		if therm.Confidence != 0.99 {
			t.Errorf("Thermostat confidence = %v, want 0.99", therm.Confidence)
		}
		if therm.FunctionType != "HeatingRadiator" {
			t.Errorf("Thermostat FunctionType = %q, want %q", therm.FunctionType, "HeatingRadiator")
		}
	} else {
		t.Error("Expected thermostat in Tier 1 results")
	}

	// Presence sensor (Custom + comment)
	if pres, ok := devicesByType["presence_sensor"]; ok {
		if pres.Confidence != 0.98 {
			t.Errorf("Presence confidence = %v, want 0.98", pres.Confidence)
		}
		if pres.FunctionType != "Custom" {
			t.Errorf("Presence FunctionType = %q, want %q", pres.FunctionType, "Custom")
		}
	} else {
		t.Error("Expected presence_sensor in Tier 1 results")
	}

	// All GAs should be consumed — no unmapped addresses
	if len(result.UnmappedAddresses) != 0 {
		t.Errorf("Expected 0 unmapped addresses, got %d: %v",
			len(result.UnmappedAddresses), result.UnmappedAddresses)
	}
}

func TestParseKNXProjFallbackTier2(t *testing.T) {
	// A .knxproj with GroupAddresses but NO Trades/Functions.
	// Tier 2 (DPT-based detection) should still work.
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	gaXML := `<?xml version="1.0" encoding="utf-8"?>
<GroupAddresses>
  <GroupRange Name="Lighting" Address="1">
    <GroupAddress Id="GA-1" Address="1/1/0" Name="Hall Light Switch" DatapointType="DPST-1-1"/>
    <GroupAddress Id="GA-2" Address="1/1/1" Name="Hall Light Brightness" DatapointType="DPST-5-1"/>
  </GroupRange>
</GroupAddresses>`

	f, _ := w.Create("P-TEST/GroupAddresses.xml")
	f.Write([]byte(gaXML))
	w.Close()

	parser := NewParser()
	result, err := parser.ParseBytes(buf.Bytes(), "test.knxproj")
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	// Should detect a dimmer via Tier 2
	foundDimmer := false
	for _, dev := range result.Devices {
		if dev.DetectedType == testTypeLightDimmer {
			foundDimmer = true
			// Tier 2 should NOT have FunctionType set
			if dev.FunctionType != "" {
				t.Errorf("Tier 2 device should not have FunctionType, got %q", dev.FunctionType)
			}
			// Tier 2 should NOT have manufacturer metadata
			if dev.Manufacturer != "" {
				t.Errorf("Tier 2 device should not have Manufacturer, got %q", dev.Manufacturer)
			}
		}
	}
	if !foundDimmer {
		t.Error("Expected Tier 2 to detect a dimmer from GroupAddresses")
	}
}

func TestParseCSVStillWorks(t *testing.T) {
	// CSV format has no XML metadata — should use Tier 2 only.
	csv := `"Address","Name","DatapointType"
"1/0/0","Office Light Switch","DPST-1-1"
"1/0/1","Office Light Brightness","DPST-5-1"
`

	parser := NewParser()
	result, err := parser.ParseBytes([]byte(csv), "test.csv")
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if result.Format != "csv" {
		t.Errorf("Format = %q, want %q", result.Format, "csv")
	}

	// Should detect a dimmer via Tier 2
	foundDimmer := false
	for _, dev := range result.Devices {
		if dev.DetectedType == testTypeLightDimmer {
			foundDimmer = true
		}
	}
	if !foundDimmer {
		t.Error("Expected Tier 2 to detect a dimmer from CSV")
	}
}
