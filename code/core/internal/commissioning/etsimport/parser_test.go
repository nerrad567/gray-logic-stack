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
	testTypeLightDimmer = "light_dimmer"
	testTypeLightSwitch = "light_switch"
	testTypeBlind       = "blind_position"
	testTypeTempSensor  = "temperature_sensor"
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
