package knx

import "strings"

// FunctionDef defines a canonical KNX function with its state key, default DPT,
// default flags, and accepted aliases. This is the single authoritative source
// for all recognised function names across the pipeline: ETS import detection
// rules, bridge state mapping, command handlers, and Flutter UI.
type FunctionDef struct {
	Name     string   // Canonical name (e.g. "switch")
	StateKey string   // State object key (e.g. "on")
	DPT      string   // Default DPT (e.g. "1.001")
	Flags    []string // Default flags
	Aliases  []string // Accepted aliases that normalise to this name
}

// CanonicalFunctions is the exhaustive list of recognised KNX functions.
// Infrastructure channel functions (e.g. "ch_a_switch_status") are NOT listed
// here — they use a channel prefix pattern handled by NormalizeChannelFunction.
var CanonicalFunctions = []FunctionDef{
	// ── Lighting ─────────────────────────────────────────────
	{Name: "switch", StateKey: "on", DPT: "1.001", Flags: []string{"write"}, Aliases: []string{"on_off", "switching"}},
	{Name: "switch_status", StateKey: "on", DPT: "1.001", Flags: []string{"read", "transmit"}, Aliases: []string{"switch_feedback"}},
	{Name: "brightness", StateKey: "level", DPT: "5.001", Flags: []string{"write"}, Aliases: []string{"dim", "dimming", "level"}},
	{Name: "brightness_status", StateKey: "level", DPT: "5.001", Flags: []string{"read", "transmit"}, Aliases: []string{"dim_status", "dim_feedback"}},
	{Name: "color_temperature", StateKey: "color_temp", DPT: "7.600", Flags: []string{"write"}, Aliases: []string{"ct", "colour_temperature", "colour_temp", "color_temp"}},                                  //nolint:misspell // KNX standard uses American "color" for DPT 7.600
	{Name: "color_temperature_status", StateKey: "color_temp", DPT: "7.600", Flags: []string{"read", "transmit"}, Aliases: []string{"colour_temperature_status", "colour_temp_status", "color_temp_status"}}, //nolint:misspell // KNX standard uses American "color" for DPT 7.600
	{Name: "rgb", StateKey: "rgb", DPT: "232.600", Flags: []string{"write"}, Aliases: []string{"colour"}},
	{Name: "rgb_status", StateKey: "rgb", DPT: "232.600", Flags: []string{"read", "transmit"}, Aliases: []string{}},
	{Name: "rgbw", StateKey: "rgbw", DPT: "251.600", Flags: []string{"write"}, Aliases: []string{}},
	{Name: "dimming_control", StateKey: "dimming_control", DPT: "3.007", Flags: []string{"write"}, Aliases: []string{"relative_dimming"}},

	// ── Blinds / Shutters ────────────────────────────────────
	{Name: "position", StateKey: "position", DPT: "5.001", Flags: []string{"write"}, Aliases: []string{"blind_position", "height"}},
	{Name: "position_status", StateKey: "position", DPT: "5.001", Flags: []string{"read", "transmit"}, Aliases: []string{"position_feedback"}},
	{Name: "slat", StateKey: "tilt", DPT: "5.001", Flags: []string{"write"}, Aliases: []string{"tilt", "lamelle", "angle"}},
	{Name: "slat_status", StateKey: "tilt", DPT: "5.001", Flags: []string{"read", "transmit"}, Aliases: []string{"tilt_status", "tilt_feedback"}},
	{Name: "move", StateKey: "moving", DPT: "1.008", Flags: []string{"write"}, Aliases: []string{"up_down"}},
	{Name: "stop", StateKey: "stop", DPT: "1.007", Flags: []string{"write"}, Aliases: []string{"step", "step_stop"}},
	{Name: "blind_control", StateKey: "blind_control", DPT: "3.008", Flags: []string{"write"}, Aliases: []string{"relative_position"}},

	// ── Climate ──────────────────────────────────────────────
	{Name: "temperature", StateKey: "temperature", DPT: "9.001", Flags: []string{"read", "transmit"}, Aliases: []string{"actual_temperature", "current_temperature", "temp", "room_temperature"}},
	{Name: "setpoint", StateKey: "setpoint", DPT: "9.001", Flags: []string{"write", "read"}, Aliases: []string{"target_temperature", "set_temp"}},
	{Name: "setpoint_status", StateKey: "setpoint", DPT: "9.001", Flags: []string{"read", "transmit"}, Aliases: []string{}},
	{Name: "heating_output", StateKey: "heating_output", DPT: "5.001", Flags: []string{"write"}, Aliases: []string{"heating_valve"}},
	{Name: "heating", StateKey: "heating", DPT: "1.001", Flags: []string{"read", "transmit"}, Aliases: []string{"heat_demand"}},
	{Name: "cooling", StateKey: "cooling", DPT: "1.001", Flags: []string{"read", "transmit"}, Aliases: []string{"cool_demand"}},
	{Name: "hvac_mode", StateKey: "hvac_mode", DPT: "20.102", Flags: []string{"write"}, Aliases: []string{"mode"}},
	{Name: "valve", StateKey: "valve", DPT: "5.001", Flags: []string{"write"}, Aliases: []string{"valve_cmd", "valve_position"}},
	{Name: "valve_status", StateKey: "valve", DPT: "5.001", Flags: []string{"read", "transmit"}, Aliases: []string{"valve_feedback"}},
	{Name: "humidity", StateKey: "humidity", DPT: "9.007", Flags: []string{"read", "transmit"}, Aliases: []string{"rh", "relative_humidity"}},

	// ── Sensors ──────────────────────────────────────────────
	{Name: "presence", StateKey: "presence", DPT: "1.018", Flags: []string{"read", "transmit"}, Aliases: []string{"motion", "occupancy", "occupied"}},
	{Name: "lux", StateKey: "lux", DPT: "9.004", Flags: []string{"read", "transmit"}, Aliases: []string{"light_level", "illuminance", "brightness_sensor"}},
	{Name: "co2", StateKey: "co2", DPT: "9.008", Flags: []string{"read", "transmit"}, Aliases: []string{"carbon_dioxide"}},
	{Name: "wind_speed", StateKey: "wind_speed", DPT: "9.005", Flags: []string{"read", "transmit"}, Aliases: []string{"wind"}},
	{Name: "rain", StateKey: "rain", DPT: "1.005", Flags: []string{"read", "transmit"}, Aliases: []string{"rain_alarm"}},
	{Name: "temperature_difference", StateKey: "temperature_difference", DPT: "9.002", Flags: []string{"read", "transmit"}, Aliases: []string{}},

	// ── Energy / Metering ────────────────────────────────────
	{Name: "power", StateKey: "power", DPT: "14.056", Flags: []string{"read", "transmit"}, Aliases: []string{"active_power"}},
	{Name: "voltage", StateKey: "voltage", DPT: "14.027", Flags: []string{"read", "transmit"}, Aliases: []string{}},
	{Name: "current", StateKey: "current", DPT: "14.019", Flags: []string{"read", "transmit"}, Aliases: []string{"electric_current"}},
	{Name: "active_energy", StateKey: "energy", DPT: "13.010", Flags: []string{"read", "transmit"}, Aliases: []string{"active_energy_kwh", "energy_kwh"}},

	// ── Scenes / Controls ────────────────────────────────────
	{Name: "scene_number", StateKey: "scene", DPT: "17.001", Flags: []string{"write", "transmit"}, Aliases: []string{"scene"}},
	{Name: "scene_control", StateKey: "scene_control", DPT: "18.001", Flags: []string{"write"}, Aliases: []string{}},

	// ── Push Buttons ─────────────────────────────────────────
	{Name: "button_1", StateKey: "button_1", DPT: "1.001", Flags: []string{"write", "transmit"}, Aliases: []string{}},
	{Name: "button_1_led", StateKey: "button_1_led", DPT: "1.001", Flags: []string{"write"}, Aliases: []string{}},
	{Name: "button_2", StateKey: "button_2", DPT: "1.001", Flags: []string{"write", "transmit"}, Aliases: []string{}},
	{Name: "button_2_led", StateKey: "button_2_led", DPT: "1.001", Flags: []string{"write"}, Aliases: []string{}},
	{Name: "button_3", StateKey: "button_3", DPT: "1.001", Flags: []string{"write", "transmit"}, Aliases: []string{}},
	{Name: "button_3_led", StateKey: "button_3_led", DPT: "1.001", Flags: []string{"write"}, Aliases: []string{}},
	{Name: "button_4", StateKey: "button_4", DPT: "1.001", Flags: []string{"write", "transmit"}, Aliases: []string{}},
	{Name: "button_4_led", StateKey: "button_4_led", DPT: "1.001", Flags: []string{"write"}, Aliases: []string{}},
	{Name: "button_5", StateKey: "button_5", DPT: "1.001", Flags: []string{"write", "transmit"}, Aliases: []string{}},
	{Name: "button_6", StateKey: "button_6", DPT: "1.001", Flags: []string{"write", "transmit"}, Aliases: []string{}},
	{Name: "button_7", StateKey: "button_7", DPT: "1.001", Flags: []string{"write", "transmit"}, Aliases: []string{}},
	{Name: "button_8", StateKey: "button_8", DPT: "1.001", Flags: []string{"write", "transmit"}, Aliases: []string{}},

	// ── Boolean Control ──────────────────────────────────────
	{Name: "enable", StateKey: "enable", DPT: "1.003", Flags: []string{"write"}, Aliases: []string{}},
	{Name: "alarm", StateKey: "alarm", DPT: "1.005", Flags: []string{"read", "transmit"}, Aliases: []string{"fault"}},
	{Name: "open_close", StateKey: "open_close", DPT: "1.009", Flags: []string{"write"}, Aliases: []string{"contact"}},
	{Name: "start_stop", StateKey: "start_stop", DPT: "1.010", Flags: []string{"write"}, Aliases: []string{}},
	{Name: "trigger", StateKey: "trigger", DPT: "1.017", Flags: []string{"write"}, Aliases: []string{}},

	// ── Generic ──────────────────────────────────────────────
	{Name: "percentage", StateKey: "percentage", DPT: "5.004", Flags: []string{"write"}, Aliases: []string{}},
}

// Lookup maps built once at init.
var (
	functionByName       map[string]*FunctionDef // canonical name → definition
	functionByAlias      map[string]*FunctionDef // alias → definition
	functionToState      map[string]string       // canonical name → state key
	functionToDefaultDPT map[string]string       // canonical name → default DPT
)

func init() {
	functionByName = make(map[string]*FunctionDef, len(CanonicalFunctions))
	functionByAlias = make(map[string]*FunctionDef, len(CanonicalFunctions)*2)
	functionToState = make(map[string]string, len(CanonicalFunctions))
	functionToDefaultDPT = make(map[string]string, len(CanonicalFunctions))

	for i := range CanonicalFunctions {
		fn := &CanonicalFunctions[i]
		functionByName[fn.Name] = fn
		functionToState[fn.Name] = fn.StateKey
		functionToDefaultDPT[fn.Name] = fn.DPT

		for _, alias := range fn.Aliases {
			functionByAlias[alias] = fn
		}
	}
}

// LookupFunction returns the canonical function definition for a name.
// Returns nil if the name is not recognised (neither canonical nor alias).
func LookupFunction(name string) *FunctionDef {
	if fn, ok := functionByName[name]; ok {
		return fn
	}
	if fn, ok := functionByAlias[name]; ok {
		return fn
	}
	return nil
}

// NormalizeFunction resolves a function name to its canonical form.
// Returns the canonical name and whether it was recognised.
func NormalizeFunction(name string) (canonical string, known bool) {
	if _, ok := functionByName[name]; ok {
		return name, true
	}
	if fn, ok := functionByAlias[name]; ok {
		return fn.Name, true
	}
	return name, false
}

// channelPrefixes are the recognised prefixes for multi-channel infrastructure
// devices (e.g. "ch_a_switch", "channel_b_valve_status").
var channelPrefixes = []string{
	"ch_a_", "ch_b_", "ch_c_", "ch_d_", "ch_e_", "ch_f_",
	"ch_g_", "ch_h_", "ch_i_", "ch_j_", "ch_k_", "ch_l_",
	"channel_a_", "channel_b_", "channel_c_", "channel_d_",
	"channel_e_", "channel_f_", "channel_g_", "channel_h_",
}

// NormalizeChannelFunction handles multi-channel function names like
// "ch_a_switch" → prefix="ch_a_", canonical="switch".
// Returns empty prefix if no channel prefix was found.
func NormalizeChannelFunction(name string) (prefix string, canonical string, known bool) {
	lower := strings.ToLower(name)
	for _, p := range channelPrefixes {
		if strings.HasPrefix(lower, p) {
			base := lower[len(p):]
			canon, ok := NormalizeFunction(base)
			if ok {
				return p, canon, true
			}
			return p, base, false
		}
	}
	return "", name, false
}

// StateKeyForFunction returns the state key for a canonical or alias function name.
// For channel-prefixed functions, returns "prefix + base_state_key" (e.g. "ch_a_on").
// Returns the function name itself as fallback for unrecognised functions.
func StateKeyForFunction(name string) string {
	// Direct lookup
	if sk, ok := functionToState[name]; ok {
		return sk
	}

	// Try alias
	if fn, ok := functionByAlias[name]; ok {
		return fn.StateKey
	}

	// Try channel prefix
	prefix, canon, known := NormalizeChannelFunction(name)
	if prefix != "" && known {
		if fn := LookupFunction(canon); fn != nil {
			return prefix + fn.StateKey
		}
	}

	// Fallback: use the function name as the state key
	return name
}

// DefaultDPTForFunction returns the default DPT for a canonical or alias function name.
// Returns empty string for unrecognised functions.
func DefaultDPTForFunction(name string) string {
	if dpt, ok := functionToDefaultDPT[name]; ok {
		return dpt
	}
	if fn, ok := functionByAlias[name]; ok {
		return fn.DPT
	}

	// Try channel prefix
	_, canon, known := NormalizeChannelFunction(name)
	if known {
		if fn := LookupFunction(canon); fn != nil {
			return fn.DPT
		}
	}

	return ""
}

// DefaultFlagsForFunction returns the default flags for a canonical or alias function name.
// Returns nil for unrecognised functions.
func DefaultFlagsForFunction(name string) []string {
	if fn, ok := functionByName[name]; ok {
		return fn.Flags
	}
	if fn, ok := functionByAlias[name]; ok {
		return fn.Flags
	}

	// Try channel prefix
	_, canon, known := NormalizeChannelFunction(name)
	if known {
		if fn := LookupFunction(canon); fn != nil {
			return fn.Flags
		}
	}

	return nil
}
