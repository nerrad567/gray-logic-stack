package etsimport

import (
	"strings"
)

// Confidence boost constants for detection scoring.
const (
	optionalDPTBoostFactor = 0.1  // Boost per matched optional DPT
	keywordMatchBoost      = 0.05 // Boost per keyword match in address names
)

// Device domain constants for detection results.
// These mirror device.Domain values but are defined locally to avoid
// circular imports between commissioning and device packages.
const (
	domainLighting       = "lighting"
	domainClimate        = "climate"
	domainBlinds         = "blinds"
	domainSensor         = "sensor"
	domainInfrastructure = "infrastructure"
	domainSecurity       = "security"
	domainScene          = "scene"
	domainEnergy         = "energy"
)

// Device type constants for detection results.
const (
	typeLightSwitch     = "light_switch"
	typeLightDimmer     = "light_dimmer"
	typeBlindPosition   = "blind_position"
	typeThermostat      = "thermostat"
	typeHeatingActuator = "heating_actuator"
	typeSwitchActuator  = "switch_actuator"
	typePresenceSensor  = "presence_sensor"
	typeSensor          = "sensor"
)

// DetectionRule defines a pattern for detecting a device type from group addresses.
type DetectionRule struct {
	// Name is the device type name (dimmer, switch, blind, etc.).
	Name string

	// Domain is the device domain (lighting, blinds, climate, etc.).
	Domain string

	// RequiredDPTs are the DPT patterns that must be present.
	RequiredDPTs []DPTRequirement

	// OptionalDPTs are additional DPTs that may be present.
	OptionalDPTs []DPTRequirement

	// MaxAddresses limits matches to avoid false positives (0 = no limit).
	MaxAddresses int

	// StrictNameMatch disables the name-fallback in matchRequiredDPTs.
	// When true, required DPTs MUST match by name keywords — a DPT-only
	// match is not enough. Use this for rules whose DPT patterns overlap
	// with other device types (e.g., heating_actuator vs blind both use 5.001).
	StrictNameMatch bool

	// MinConfidence is the base confidence for this rule.
	MinConfidence float64
}

// DPTRequirement defines a required or optional DPT for detection.
type DPTRequirement struct {
	// DPT is the datapoint type pattern (e.g., "1.001", "5.*").
	DPT string

	// Function is the suggested function name for this address.
	Function string

	// NameContains are keywords that increase confidence if found in the name.
	NameContains []string

	// Flags are the suggested access flags.
	Flags []string
}

// TryMatch attempts to match a group of addresses against this rule.
func (r *DetectionRule) TryMatch(prefix string, addresses []GroupAddress) *DetectedDevice {
	if r.MaxAddresses > 0 && len(addresses) > r.MaxAddresses {
		return nil
	}

	// Check required DPTs
	matched := r.matchRequiredDPTs(addresses)
	if matched == nil {
		return nil
	}

	// Check optional DPTs
	r.matchOptionalDPTs(matched, addresses)

	// Calculate confidence and build device
	confidence := r.calculateConfidence(matched)
	device := r.buildDevice(prefix, matched, confidence)

	return device
}

// matchRequiredDPTs attempts to match all required DPTs. Returns nil if any required DPT is not found.
func (r *DetectionRule) matchRequiredDPTs(addresses []GroupAddress) map[string]GroupAddress {
	matched := make(map[string]GroupAddress)

	for _, req := range r.RequiredDPTs {
		if !r.tryMatchDPT(req, addresses, matched, true) {
			// Try without name matching as fallback (unless StrictNameMatch)
			if r.StrictNameMatch || !r.tryMatchDPT(req, addresses, matched, false) {
				return nil // Required DPT not found
			}
		}
	}

	return matched
}

// tryMatchDPT attempts to match a single DPT requirement against addresses.
func (r *DetectionRule) tryMatchDPT(req DPTRequirement, addresses []GroupAddress, matched map[string]GroupAddress, requireNameMatch bool) bool {
	for _, addr := range addresses {
		if !matchesDPT(addr.DPT, req.DPT) || isAlreadyMatched(matched, addr.Address) {
			continue
		}

		if requireNameMatch && len(req.NameContains) > 0 {
			if !containsAny(addr.Name, req.NameContains) {
				continue
			}
		}

		matched[req.Function] = addr
		return true
	}
	return false
}

// matchOptionalDPTs attempts to match optional DPTs and adds them to the matched map.
func (r *DetectionRule) matchOptionalDPTs(matched map[string]GroupAddress, addresses []GroupAddress) {
	for _, opt := range r.OptionalDPTs {
		r.tryMatchDPT(opt, addresses, matched, len(opt.NameContains) > 0)
	}
}

// calculateConfidence computes the confidence score based on matched addresses.
func (r *DetectionRule) calculateConfidence(matched map[string]GroupAddress) float64 {
	confidence := r.MinConfidence

	// Boost for matching optional DPTs
	confidence += r.optionalDPTBoost(matched)

	// Boost for name keyword matches
	confidence += r.keywordMatchBoost(matched)

	// Cap at maximum confidence
	if confidence > maxConfidence {
		confidence = maxConfidence
	}

	return confidence
}

// optionalDPTBoost calculates confidence boost from matched optional DPTs.
func (r *DetectionRule) optionalDPTBoost(matched map[string]GroupAddress) float64 {
	if len(r.OptionalDPTs) == 0 {
		return 0
	}

	optionalMatched := 0
	for _, opt := range r.OptionalDPTs {
		if _, ok := matched[opt.Function]; ok {
			optionalMatched++
		}
	}

	return optionalDPTBoostFactor * float64(optionalMatched) / float64(len(r.OptionalDPTs))
}

// keywordMatchBoost calculates confidence boost from keyword matches in names.
func (r *DetectionRule) keywordMatchBoost(matched map[string]GroupAddress) float64 {
	keywordMatches := 0
	allDPTs := append(r.RequiredDPTs, r.OptionalDPTs...)

	for _, req := range allDPTs {
		if addr, ok := matched[req.Function]; ok {
			if containsAny(addr.Name, req.NameContains) {
				keywordMatches++
			}
		}
	}

	if keywordMatches > 0 {
		return keywordMatchBoost * float64(keywordMatches)
	}
	return 0
}

// buildDevice creates a DetectedDevice from matched addresses.
func (r *DetectionRule) buildDevice(prefix string, matched map[string]GroupAddress, confidence float64) *DetectedDevice {
	device := &DetectedDevice{
		SuggestedID:     generateSlug(prefix),
		SuggestedName:   cleanName(prefix),
		DetectedType:    r.Name,
		Confidence:      confidence,
		SuggestedDomain: r.Domain,
	}

	// Add required addresses
	r.addAddressesToDevice(device, r.RequiredDPTs, matched)

	// Add optional addresses
	r.addAddressesToDevice(device, r.OptionalDPTs, matched)

	// Extract room/area from location
	if device.SourceLocation != "" {
		device.SuggestedRoom = extractRoomFromLocation(device.SourceLocation)
		device.SuggestedArea = extractAreaFromLocation(device.SourceLocation)
	}

	return device
}

// addAddressesToDevice adds matched addresses to the device.
func (r *DetectionRule) addAddressesToDevice(device *DetectedDevice, reqs []DPTRequirement, matched map[string]GroupAddress) {
	for _, req := range reqs {
		addr, ok := matched[req.Function]
		if !ok {
			continue
		}

		device.Addresses = append(device.Addresses, DeviceAddress{
			GA:                addr.Address,
			Name:              addr.Name,
			DPT:               addr.DPT,
			SuggestedFunction: req.Function,
			SuggestedFlags:    req.Flags,
			Description:       addr.Description,
		})

		if device.SourceLocation == "" {
			device.SourceLocation = addr.Location
		}
	}
}

// DefaultDetectionRules returns the standard device detection rules.
func DefaultDetectionRules() []DetectionRule {
	return []DetectionRule{
		// Dimmer: switch + brightness
		{
			Name:          "light_dimmer",
			Domain:        "lighting",
			MinConfidence: 0.85,
			RequiredDPTs: []DPTRequirement{
				{
					DPT:          "1.001",
					Function:     "switch",
					NameContains: []string{"switch", "on/off", "schalten", "ein/aus"},
					Flags:        []string{"write"},
				},
				{
					DPT:          "5.001",
					Function:     "brightness",
					NameContains: []string{"dim", "brightness", "level", "helligkeit", "wert"},
					Flags:        []string{"write"},
				},
			},
			OptionalDPTs: []DPTRequirement{
				{
					DPT:          "1.001",
					Function:     "switch_status",
					NameContains: []string{"switch", "status", "feedback", "rückmeldung", "state"},
					Flags:        []string{"read", "transmit"},
				},
				{
					DPT:          "5.001",
					Function:     "brightness_status",
					NameContains: []string{"brightness", "dim", "status", "feedback", "rückmeldung", "state", "actual"},
					Flags:        []string{"read", "transmit"},
				},
			},
		},

		// Thermostat (must be before heating_actuator — a device with
		// temperature GAs + valve GA is a thermostat, not a heating actuator)
		{
			Name:          "thermostat",
			Domain:        "climate",
			MinConfidence: 0.85,
			RequiredDPTs: []DPTRequirement{
				{
					DPT:          "9.001",
					Function:     "temperature",
					NameContains: []string{"temp", "actual", "ist"},
					Flags:        []string{"read", "transmit"},
				},
				{
					DPT:          "9.001",
					Function:     "setpoint",
					NameContains: []string{"setpoint", "target", "soll", "set"},
					Flags:        []string{"write", "read"},
				},
			},
			OptionalDPTs: []DPTRequirement{
				{
					DPT:          "5.001",
					Function:     "heating_output",
					NameContains: []string{"heating", "valve", "output", "stellgröße"},
					Flags:        []string{"write"},
				},
				{
					DPT:          "1.001",
					Function:     "heating",
					NameContains: []string{"heat", "heiz"},
					Flags:        []string{"read", "transmit"},
				},
				{
					DPT:          "1.001",
					Function:     "cooling",
					NameContains: []string{"cool", "kühl"},
					Flags:        []string{"read", "transmit"},
				},
			},
		},

		// Heating/valve actuator (must be before blind rules to prevent
		// valve DPT 5.001 GAs matching blind_tilt via name-fallback path)
		{
			Name:            "heating_actuator",
			Domain:          "climate",
			StrictNameMatch: true,
			MinConfidence:   0.85,
			RequiredDPTs: []DPTRequirement{
				{
					DPT:          "5.001",
					Function:     "valve",
					NameContains: []string{"valve", "heating", "actuator", "ventil"},
					Flags:        []string{"write"},
				},
			},
			OptionalDPTs: []DPTRequirement{
				{
					DPT:          "5.001",
					Function:     "valve_status",
					NameContains: []string{"valve", "status", "feedback", "rückmeldung"},
					Flags:        []string{"read", "transmit"},
				},
			},
		},

		// Blind with position and slat/tilt control
		{
			Name:          "blind_tilt",
			Domain:        "blinds",
			MinConfidence: 0.85,
			RequiredDPTs: []DPTRequirement{
				{
					DPT:          "5.001",
					Function:     "position",
					NameContains: []string{"position", "height", "höhe"},
					Flags:        []string{"write"},
				},
				{
					DPT:          "5.001",
					Function:     "slat",
					NameContains: []string{"slat", "lamelle", "tilt", "angle", "winkel"},
					Flags:        []string{"write"},
				},
			},
			OptionalDPTs: []DPTRequirement{
				{
					DPT:          "1.008",
					Function:     "move",
					NameContains: []string{"move", "up/down", "auf/ab", "fahren"},
					Flags:        []string{"write"},
				},
				{
					DPT:          "1.007",
					Function:     "stop",
					NameContains: []string{"stop", "stopp"},
					Flags:        []string{"write"},
				},
				{
					DPT:          "5.001",
					Function:     "position_status",
					NameContains: []string{"position", "status", "feedback", "rückmeldung", "actual"},
					Flags:        []string{"read", "transmit"},
				},
				{
					DPT:          "5.001",
					Function:     "slat_status",
					NameContains: []string{"slat", "lamelle", "tilt", "status"},
					Flags:        []string{"read", "transmit"},
				},
			},
		},

		// Blind with position only
		{
			Name:          "blind_position",
			Domain:        "blinds",
			MinConfidence: 0.80,
			RequiredDPTs: []DPTRequirement{
				{
					DPT:          "1.008",
					Function:     "move",
					NameContains: []string{"move", "up/down", "auf/ab", "fahren", "blind", "shutter"},
					Flags:        []string{"write"},
				},
				{
					DPT:          "5.001",
					Function:     "position",
					NameContains: []string{"position", "height", "höhe"},
					Flags:        []string{"write"},
				},
			},
			OptionalDPTs: []DPTRequirement{
				{
					DPT:          "1.007",
					Function:     "stop",
					NameContains: []string{"stop", "stopp"},
					Flags:        []string{"write"},
				},
				{
					DPT:          "5.001",
					Function:     "position_status",
					NameContains: []string{"status", "feedback", "rückmeldung"},
					Flags:        []string{"read", "transmit"},
				},
			},
		},

		// Blind with move/stop (channel-based naming like "Ch-3 - Blinds")
		{
			Name:          "blind_switch",
			Domain:        "blinds",
			MinConfidence: 0.75,
			RequiredDPTs: []DPTRequirement{
				{
					DPT:          "1.008",
					Function:     "move",
					NameContains: []string{"blind", "shutter", "jalousie", "rollo", "move", "up", "down"},
					Flags:        []string{"write"},
				},
			},
			OptionalDPTs: []DPTRequirement{
				{
					DPT:          "1.007",
					Function:     "stop",
					NameContains: []string{"stop", "step"},
					Flags:        []string{"write"},
				},
				{
					DPT:          "1.010",
					Function:     "stop",
					NameContains: []string{"stop", "step"},
					Flags:        []string{"write"},
				},
				{
					DPT:          "5.001",
					Function:     "position",
					NameContains: []string{"position", "height"},
					Flags:        []string{"write"},
				},
			},
		},

		// Simple switch
		{
			Name:          "light_switch",
			Domain:        "lighting",
			MinConfidence: 0.75,
			MaxAddresses:  3, // Avoid matching dimmers
			RequiredDPTs: []DPTRequirement{
				{
					DPT:          "1.001",
					Function:     "switch",
					NameContains: []string{"switch", "light", "licht", "on/off", "schalten", "ch-"},
					Flags:        []string{"write"},
				},
			},
			OptionalDPTs: []DPTRequirement{
				{
					DPT:          "1.001",
					Function:     "switch_status",
					NameContains: []string{"status", "feedback", "rückmeldung", "state"},
					Flags:        []string{"read", "transmit"},
				},
			},
		},

		// Temperature sensor
		{
			Name:          "temperature_sensor",
			Domain:        "sensor",
			MinConfidence: 0.90,
			MaxAddresses:  2,
			RequiredDPTs: []DPTRequirement{
				{
					DPT:          "9.001",
					Function:     "temperature",
					NameContains: []string{"temp", "temperatur"},
					Flags:        []string{"read", "transmit"},
				},
			},
		},

		// Humidity sensor
		{
			Name:          "humidity_sensor",
			Domain:        "sensor",
			MinConfidence: 0.90,
			MaxAddresses:  2,
			RequiredDPTs: []DPTRequirement{
				{
					DPT:          "9.007",
					Function:     "humidity",
					NameContains: []string{"humid", "feucht", "rh"},
					Flags:        []string{"read", "transmit"},
				},
			},
		},

		// Presence/motion sensor (must be before light_sensor so that a
		// device with both presence DPT 1.* and lux DPT 9.004 is detected
		// as presence_sensor, not light_sensor)
		{
			Name:          "presence_sensor",
			Domain:        "sensor",
			MinConfidence: 0.85,
			MaxAddresses:  3,
			RequiredDPTs: []DPTRequirement{
				{
					DPT:          "1.*",
					Function:     "presence",
					NameContains: []string{"presence", "motion", "bewegung", "präsenz", "occupancy"},
					Flags:        []string{"read", "transmit"},
				},
			},
			OptionalDPTs: []DPTRequirement{
				{
					DPT:          "9.004",
					Function:     "lux",
					NameContains: []string{"lux", "brightness", "helligkeit"},
					Flags:        []string{"read", "transmit"},
				},
			},
		},

		// Lux sensor (standalone — if device also has presence DPT, it
		// was already caught by the presence_sensor rule above)
		{
			Name:          "light_sensor",
			Domain:        "sensor",
			MinConfidence: 0.90,
			MaxAddresses:  2,
			RequiredDPTs: []DPTRequirement{
				{
					DPT:          "9.004",
					Function:     "lux",
					NameContains: []string{"lux", "brightness", "helligkeit", "light level"},
					Flags:        []string{"read", "transmit"},
				},
			},
		},
	}
}

// functionTypeToDeviceType maps an ETS Function Type to a GLCore device type.
// Returns (deviceType, domain, confidence). Returns empty strings if unknown.
func functionTypeToDeviceType(funcType, comment string) (string, string, float64) {
	switch funcType {
	case "SwitchableLight":
		return typeLightSwitch, domainLighting, 0.99 //nolint:mnd // high-confidence match score
	case "DimmableLight":
		return typeLightDimmer, domainLighting, 0.99 //nolint:mnd // high-confidence match score
	case "Sunblind":
		return typeBlindPosition, domainBlinds, 0.95 //nolint:mnd // high-confidence match score
	case "HeatingRadiator":
		return typeThermostat, domainClimate, 0.99 //nolint:mnd // high-confidence match score
	case "HeatingFloor":
		return typeHeatingActuator, domainClimate, 0.99 //nolint:mnd // high-confidence match score
	case "HeatingContinuousVariable":
		return typeHeatingActuator, domainClimate, 0.99 //nolint:mnd // high-confidence match score
	case "HeatingSwitchingVariable":
		return typeHeatingActuator, domainClimate, 0.99 //nolint:mnd // high-confidence match score
	}

	// Custom → check Comment for KNXSim template ID hint
	if funcType == "Custom" && comment != "" {
		return commentToDeviceType(comment)
	}
	return "", "", 0
}

// commentToDeviceType maps KNXSim template IDs (carried in Function Comment
// attribute) to GLCore device types. This enables Tier 1 classification for
// device types that have no standard ETS Function Type.
func commentToDeviceType(comment string) (string, string, float64) {
	// Infrastructure devices carry JSON channel metadata in the Comment.
	// Detect by checking for the JSON prefix.
	if strings.HasPrefix(comment, "{\"infrastructure\"") {
		// Determine actuator subtype from channel GAs
		if strings.Contains(comment, "\"valve\"") {
			return typeHeatingActuator, domainInfrastructure, 0.99 //nolint:mnd // high-confidence match score
		}
		return typeSwitchActuator, domainInfrastructure, 0.99 //nolint:mnd // high-confidence match score
	}

	known := map[string][2]string{
		// Sensors
		"presence_detector":        {"presence_sensor", "sensor"},
		"presence_detector_360":    {"presence_sensor", "sensor"},
		"presence":                 {"presence_sensor", "sensor"},
		"presence_pattern":         {"presence_sensor", "sensor"},
		"motion_detector_outdoor":  {"presence_sensor", "sensor"},
		"temperature_sensor":       {"temperature_sensor", "sensor"},
		"humidity_sensor":          {"humidity_sensor", "sensor"},
		"co2_sensor":               {"co2_sensor", "sensor"},
		"lux_sensor":               {"light_sensor", "sensor"},
		"multi_sensor":             {"multi_sensor", "sensor"},
		"brightness_sensor_facade": {"light_sensor", "sensor"},
		"wind_sensor":              {"wind_sensor", "sensor"},
		"weather_station":          {"weather_station", "sensor"},
		// Energy
		"energy_meter":        {"energy_meter", "energy"},
		"energy_meter_3phase": {"energy_meter", "energy"},
		"solar_inverter":      {"solar_inverter", "energy"},
		"ev_charger":          {"ev_charger", "energy"},
		"load_controller":     {"load_controller", "energy"},
		// Controls (mapped to lighting — wall controls typically control lights)
		"scene_controller":        {"scene_controller", "lighting"},
		"push_button_2":           {"push_button", "lighting"},
		"push_button_4":           {"push_button", "lighting"},
		"push_button_2fold":       {"push_button", "lighting"},
		"glass_push_button_6":     {"push_button", "lighting"},
		"glass_push_button_8":     {"push_button", "lighting"},
		"binary_input":            {"binary_input", "sensor"},
		"binary_input_8ch":        {"binary_input", "sensor"},
		"room_controller_display": {"room_controller", "climate"},
		"logic_module":            {"logic_module", "lighting"},
		// Climate (non-standard types)
		"fan_coil_controller": {"fan_coil", "climate"},
		"air_handling_unit":   {"air_handling_unit", "climate"},
		"hvac_unit":           {"hvac_unit", "climate"},
		"valve":               {"heating_actuator", "climate"},
		// System (mapped to energy — infrastructure devices)
		"ip_router":    {"ip_router", "energy"},
		"line_coupler": {"line_coupler", "energy"},
		"power_supply": {"power_supply", "energy"},
		"timer_switch": {"timer_switch", "energy"},
		// Lighting actuators (multi-channel)
		"switch_actuator_4ch":  {"light_switch", "lighting"},
		"switch_actuator_8ch":  {"light_switch", "lighting"},
		"switch_actuator_12ch": {"light_switch", "lighting"},
		"dimmer_actuator_4ch":  {"light_dimmer", "lighting"},
		"dali_gateway":         {"dali_gateway", "lighting"},
		// Blind actuators (multi-channel)
		"shutter_actuator_4ch": {"blind_position", "blinds"},
		"shutter_actuator_8ch": {"blind_position", "blinds"},
		"awning_controller":    {"blind_position", "blinds"},
	}
	if t, ok := known[comment]; ok {
		return t[0], t[1], 0.98 //nolint:mnd // keyword-based match confidence
	}
	return "", "", 0
}

// Helper functions

func matchesDPT(actual, pattern string) bool {
	if actual == "" {
		return false
	}

	// Exact match
	if actual == pattern {
		return true
	}

	// Wildcard match (e.g., "1.*" matches "1.001", "1.008")
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(actual, prefix+".")
	}

	// Main type match (e.g., "1" matches "1.001")
	if !strings.Contains(pattern, ".") {
		return strings.HasPrefix(actual, pattern+".")
	}

	return false
}

func isAlreadyMatched(matched map[string]GroupAddress, address string) bool {
	for _, ga := range matched {
		if ga.Address == address {
			return true
		}
	}
	return false
}

func containsAny(text string, keywords []string) bool {
	lower := strings.ToLower(text)
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}
