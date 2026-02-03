package etsimport

import (
	"strings"
)

// Confidence boost constants for detection scoring.
const (
	optionalDPTBoostFactor = 0.1  // Boost per matched optional DPT
	keywordMatchBoost      = 0.05 // Boost per keyword match in address names
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

		// Thermostat
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
	}
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
