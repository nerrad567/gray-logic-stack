package knx

import (
	"testing"
)

func TestNormalizeFunction_CanonicalNames(t *testing.T) {
	// Every canonical name should resolve to itself.
	for _, fn := range CanonicalFunctions {
		canon, known := NormalizeFunction(fn.Name)
		if !known {
			t.Errorf("NormalizeFunction(%q) not recognised", fn.Name)
		}
		if canon != fn.Name {
			t.Errorf("NormalizeFunction(%q) = %q, want %q", fn.Name, canon, fn.Name)
		}
	}
}

func TestNormalizeFunction_Aliases(t *testing.T) {
	tests := []struct {
		alias     string
		wantCanon string
	}{
		// Lighting aliases
		{"on_off", "switch"},
		{"switching", "switch"},
		{"switch_feedback", "switch_status"},
		{"dim", "brightness"},
		{"dimming", "brightness"},
		{"level", "brightness"},
		{"dim_status", "brightness_status"},
		{"ct", "color_temperature"},                 //nolint:misspell // KNX standard term
		{"colour_temperature", "color_temperature"}, //nolint:misspell // KNX standard term
		{"colour_temp", "color_temperature"},        //nolint:misspell // KNX standard term
		{"colour", "rgb"},

		// Climate aliases
		{"actual_temperature", "temperature"},
		{"current_temperature", "temperature"},
		{"temp", "temperature"},
		{"room_temperature", "temperature"},
		{"target_temperature", "setpoint"},
		{"set_temp", "setpoint"},
		{"heating_valve", "heating_output"},
		{"heat_demand", "heating"},
		{"cool_demand", "cooling"},
		{"mode", "hvac_mode"},
		{"valve_cmd", "valve"},
		{"valve_feedback", "valve_status"},
		{"rh", "humidity"},
		{"relative_humidity", "humidity"},

		// Sensor aliases
		{"motion", "presence"},
		{"occupancy", "presence"},
		{"occupied", "presence"},
		{"light_level", "lux"},
		{"illuminance", "lux"},
		{"carbon_dioxide", "co2"},
		{"wind", "wind_speed"},

		// Blind aliases
		{"blind_position", "position"},
		{"height", "position"},
		{"position_feedback", "position_status"},
		{"tilt", "slat"},
		{"lamelle", "slat"},
		{"angle", "slat"},
		{"tilt_status", "slat_status"},
		{"up_down", "move"},
		{"step", "stop"},

		// Energy aliases
		{"active_power", "power"},
		{"electric_current", "current"},
		{"active_energy_kwh", "active_energy"},
		{"energy_kwh", "active_energy"},

		// Scene aliases
		{"scene", "scene_number"},

		// Boolean aliases
		{"fault", "alarm"},
		{"rain_alarm", "rain"},
		{"contact", "open_close"},
	}

	for _, tt := range tests {
		canon, known := NormalizeFunction(tt.alias)
		if !known {
			t.Errorf("NormalizeFunction(%q) not recognised", tt.alias)
			continue
		}
		if canon != tt.wantCanon {
			t.Errorf("NormalizeFunction(%q) = %q, want %q", tt.alias, canon, tt.wantCanon)
		}
	}
}

func TestNormalizeFunction_Unknown(t *testing.T) {
	canon, known := NormalizeFunction("totally_unknown_function")
	if known {
		t.Errorf("NormalizeFunction(unknown) should return known=false")
	}
	if canon != "totally_unknown_function" {
		t.Errorf("NormalizeFunction(unknown) should pass through, got %q", canon)
	}
}

func TestNormalizeChannelFunction(t *testing.T) {
	tests := []struct {
		name      string
		wantPfx   string
		wantCanon string
		wantKnown bool
	}{
		{"ch_a_switch", "ch_a_", "switch", true},
		{"ch_b_valve_status", "ch_b_", "valve_status", true},
		{"ch_c_brightness", "ch_c_", "brightness", true},
		{"channel_a_switch_status", "channel_a_", "switch_status", true},
		{"ch_a_unknown_thing", "ch_a_", "unknown_thing", false},
		{"switch", "", "switch", false},                 // no prefix
		{"plain_function", "", "plain_function", false}, // no prefix
	}

	for _, tt := range tests {
		prefix, canon, known := NormalizeChannelFunction(tt.name)
		if prefix != tt.wantPfx {
			t.Errorf("NormalizeChannelFunction(%q) prefix = %q, want %q", tt.name, prefix, tt.wantPfx)
		}
		if canon != tt.wantCanon {
			t.Errorf("NormalizeChannelFunction(%q) canonical = %q, want %q", tt.name, canon, tt.wantCanon)
		}
		if known != tt.wantKnown {
			t.Errorf("NormalizeChannelFunction(%q) known = %v, want %v", tt.name, known, tt.wantKnown)
		}
	}
}

func TestStateKeyForFunction(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// Direct canonical
		{"switch", "on"},
		{"switch_status", "on"},
		{"brightness", "level"},
		{"brightness_status", "level"},
		{"position", "position"},
		{"slat", "tilt"},
		{"temperature", "temperature"},
		{"setpoint", "setpoint"},
		{"valve", "valve"},
		{"valve_status", "valve"},
		{"presence", "presence"},
		{"lux", "lux"},
		{"humidity", "humidity"},
		{"move", "moving"},
		{"stop", "stop"},
		{"button_1", "button_1"},
		{"scene_number", "scene"},

		// Via alias
		{"actual_temperature", "temperature"},
		{"dim", "level"},
		{"tilt", "tilt"},
		{"motion", "presence"},

		// Channel prefixed
		{"ch_a_switch", "ch_a_on"},
		{"ch_b_valve_status", "ch_b_valve"},

		// Unknown fallback
		{"completely_unknown", "completely_unknown"},
	}

	for _, tt := range tests {
		got := StateKeyForFunction(tt.name)
		if got != tt.want {
			t.Errorf("StateKeyForFunction(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDefaultDPTForFunction(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"switch", "1.001"},
		{"brightness", "5.001"},
		{"temperature", "9.001"},
		{"humidity", "9.007"},
		{"position", "5.001"},
		{"move", "1.008"},
		{"stop", "1.007"},
		{"valve", "5.001"},
		{"presence", "1.018"},
		{"lux", "9.004"},
		{"color_temperature", "7.600"}, //nolint:misspell // KNX standard term
		{"hvac_mode", "20.102"},
		{"power", "14.056"},
		{"active_energy", "13.010"},
		{"scene_number", "17.001"},

		// Via alias
		{"actual_temperature", "9.001"},
		{"dim", "5.001"},

		// Channel prefix
		{"ch_a_switch", "1.001"},
		{"ch_b_valve", "5.001"},

		// Unknown → empty
		{"unknown", ""},
	}

	for _, tt := range tests {
		got := DefaultDPTForFunction(tt.name)
		if got != tt.want {
			t.Errorf("DefaultDPTForFunction(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestLookupFunction(t *testing.T) {
	// Known canonical
	fn := LookupFunction("switch")
	if fn == nil {
		t.Fatal("LookupFunction(switch) returned nil")
	}
	if fn.Name != "switch" {
		t.Errorf("Name = %q, want switch", fn.Name)
	}
	if fn.StateKey != "on" {
		t.Errorf("StateKey = %q, want on", fn.StateKey)
	}

	// Known alias
	fn = LookupFunction("actual_temperature")
	if fn == nil {
		t.Fatal("LookupFunction(actual_temperature) returned nil")
	}
	if fn.Name != "temperature" {
		t.Errorf("Name = %q, want temperature", fn.Name)
	}

	// Unknown
	fn = LookupFunction("totally_unknown")
	if fn != nil {
		t.Errorf("LookupFunction(totally_unknown) should return nil, got %+v", fn)
	}
}

func TestCanonicalFunctions_NoDuplicateNames(t *testing.T) {
	seen := make(map[string]bool)
	for _, fn := range CanonicalFunctions {
		if seen[fn.Name] {
			t.Errorf("duplicate canonical name: %q", fn.Name)
		}
		seen[fn.Name] = true
	}
}

func TestCanonicalFunctions_NoDuplicateAliases(t *testing.T) {
	seen := make(map[string]string)
	for _, fn := range CanonicalFunctions {
		for _, alias := range fn.Aliases {
			if prev, ok := seen[alias]; ok {
				t.Errorf("duplicate alias %q: used by both %q and %q", alias, prev, fn.Name)
			}
			seen[alias] = fn.Name
		}
	}
}

func TestCanonicalFunctions_AliasNotCanonicalName(t *testing.T) {
	// No alias should collide with a canonical name
	names := make(map[string]bool)
	for _, fn := range CanonicalFunctions {
		names[fn.Name] = true
	}
	for _, fn := range CanonicalFunctions {
		for _, alias := range fn.Aliases {
			if names[alias] {
				t.Errorf("alias %q of %q collides with canonical name", alias, fn.Name)
			}
		}
	}
}

func TestCanonicalFunctions_AllHaveDPT(t *testing.T) {
	for _, fn := range CanonicalFunctions {
		if fn.DPT == "" {
			t.Errorf("canonical function %q has empty DPT", fn.Name)
		}
	}
}

func TestCanonicalFunctions_AllHaveStateKey(t *testing.T) {
	for _, fn := range CanonicalFunctions {
		if fn.StateKey == "" {
			t.Errorf("canonical function %q has empty StateKey", fn.Name)
		}
	}
}

func TestCanonicalFunctions_AllHaveFlags(t *testing.T) {
	for _, fn := range CanonicalFunctions {
		if len(fn.Flags) == 0 {
			t.Errorf("canonical function %q has no flags", fn.Name)
		}
	}
}
