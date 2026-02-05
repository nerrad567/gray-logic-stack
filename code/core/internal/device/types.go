package device

import "time"

// Device represents a controllable or monitorable entity in the system.
// This matches the database schema in migrations/20260118_200000_initial_schema.up.sql.
type Device struct {
	// Identity
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`

	// Location (one of RoomID or AreaID should be set for room-level or area-level devices)
	RoomID *string `json:"room_id,omitempty"`
	AreaID *string `json:"area_id,omitempty"`

	// Classification
	Type   DeviceType `json:"type"`
	Domain Domain     `json:"domain"`

	// Protocol information
	Protocol  Protocol `json:"protocol"`
	Address   Address  `json:"address"`
	GatewayID *string  `json:"gateway_id,omitempty"`

	// Capabilities and configuration
	Capabilities []Capability `json:"capabilities"`
	Config       Config       `json:"config"`

	// Current state
	State          State      `json:"state"`
	StateUpdatedAt *time.Time `json:"state_updated_at,omitempty"`

	// Health monitoring
	HealthStatus   HealthStatus `json:"health_status"`
	HealthLastSeen *time.Time   `json:"health_last_seen,omitempty"`
	PHMEnabled     bool         `json:"phm_enabled"`
	PHMBaseline    *PHMBaseline `json:"phm_baseline,omitempty"`

	// Metadata
	Manufacturer    *string `json:"manufacturer,omitempty"`
	Model           *string `json:"model,omitempty"`
	FirmwareVersion *string `json:"firmware_version,omitempty"`

	// Tags are free-form string labels for filtering and exception-based operations.
	// Example: ["escape_lighting", "accent", "entertainment"]
	Tags []string `json:"tags,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeepCopy creates a complete independent copy of the Device.
// All map and slice fields are cloned so modifications to the copy
// do not affect the original. This is essential for cache isolation.
func (d *Device) DeepCopy() *Device {
	if d == nil {
		return nil
	}

	cpy := *d // Shallow copy of value fields

	// Deep copy maps
	cpy.Address = deepCopyMap(d.Address)
	cpy.Config = deepCopyMap(d.Config)
	cpy.State = deepCopyMap(d.State)

	// Deep copy slice
	if d.Capabilities != nil {
		cpy.Capabilities = make([]Capability, len(d.Capabilities))
		copy(cpy.Capabilities, d.Capabilities)
	}

	if d.Tags != nil {
		cpy.Tags = make([]string, len(d.Tags))
		copy(cpy.Tags, d.Tags)
	}

	// Deep copy PHMBaseline pointer to map
	if d.PHMBaseline != nil {
		baseline := deepCopyMap(*d.PHMBaseline)
		cpy.PHMBaseline = (*PHMBaseline)(&baseline)
	}

	// Pointer fields (*string, *time.Time) don't need deep copy
	// because strings and time.Time are immutable in Go

	return &cpy
}

// deepCopyMap creates a deep copy of a map[string]any.
// Nested maps and slices are recursively copied.
func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	cpy := make(map[string]any, len(m))
	for k, v := range m {
		cpy[k] = deepCopyValue(v)
	}
	return cpy
}

// deepCopyValue recursively copies a value, handling nested maps and slices.
func deepCopyValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]any:
		return deepCopyMap(val)
	case []any:
		cpy := make([]any, len(val))
		for i, elem := range val {
			cpy[i] = deepCopyValue(elem)
		}
		return cpy
	default:
		// Primitives (string, bool, int, float64, etc.) are safe to copy by value
		return v
	}
}

// Address holds protocol-specific address information as a JSON map.
//
// Examples:
//
//	KNX (structured functions with DPT and flags):
//	  {
//	    "individual_address": "1.1.1",
//	    "application_program": "M-0001_...",
//	    "functions": {
//	      "switch":        {"ga": "1/0/1", "dpt": "1.001", "flags": ["write"]},
//	      "switch_status": {"ga": "1/0/2", "dpt": "1.001", "flags": ["read", "transmit"]}
//	    }
//	  }
//
//	DALI: {"gateway": "dali-gw-01", "short_address": 15, "group": 0}
//	Modbus: {"host": "192.168.1.100", "port": 502, "unit_id": 1, "registers": {...}}
type Address map[string]any

// KNXFunctionConfig represents a single KNX function's address configuration.
// Stored inside the Address map under the "functions" key.
type KNXFunctionConfig struct {
	GA    string   `json:"ga"`
	DPT   string   `json:"dpt"`
	Flags []string `json:"flags"`
}

// GetKNXFunctions extracts the typed function map from a KNX Address.
// Returns nil if the "functions" key is missing or not in the expected format.
func GetKNXFunctions(addr Address) map[string]KNXFunctionConfig {
	raw, ok := addr["functions"]
	if !ok {
		return nil
	}

	// The "functions" value is stored as map[string]any in JSON.
	// Each value is itself a map[string]any with "ga", "dpt", "flags" keys.
	funcMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	result := make(map[string]KNXFunctionConfig, len(funcMap))
	for name, v := range funcMap {
		entry, ok := v.(map[string]any)
		if !ok {
			continue
		}

		fc := KNXFunctionConfig{}
		if ga, ok := entry["ga"].(string); ok {
			fc.GA = ga
		}
		if dpt, ok := entry["dpt"].(string); ok {
			fc.DPT = dpt
		}
		if flags, ok := entry["flags"].([]any); ok {
			for _, f := range flags {
				if s, ok := f.(string); ok {
					fc.Flags = append(fc.Flags, s)
				}
			}
		}

		result[name] = fc
	}

	return result
}

// Config holds device-specific configuration as a JSON map.
type Config map[string]any

// State holds the current device state as a JSON map.
//
// Examples:
//   - Light: {"on": true, "level": 75}
//   - Thermostat: {"temperature": 21.5, "setpoint": 22.0, "mode": "heat"}
//   - Blind: {"position": 50, "tilt": 45}
type State map[string]any

// PHMBaseline holds learned baseline data for Predictive Health Monitoring.
type PHMBaseline map[string]any

// Domain represents the functional area a device belongs to.
type Domain string

// Domain constants.
const (
	DomainLighting       Domain = "lighting"
	DomainClimate        Domain = "climate"
	DomainBlinds         Domain = "blinds"
	DomainAudio          Domain = "audio"
	DomainVideo          Domain = "video"
	DomainSecurity       Domain = "security"
	DomainAccess         Domain = "access"
	DomainEnergy         Domain = "energy"
	DomainPlant          Domain = "plant"
	DomainIrrigation     Domain = "irrigation"
	DomainSafety         Domain = "safety"
	DomainSensor         Domain = "sensor"
	DomainInfrastructure Domain = "infrastructure"
)

// AllDomains returns all valid domain values.
func AllDomains() []Domain {
	return []Domain{
		DomainLighting, DomainClimate, DomainBlinds, DomainAudio,
		DomainVideo, DomainSecurity, DomainAccess, DomainEnergy,
		DomainPlant, DomainIrrigation, DomainSafety, DomainSensor,
		DomainInfrastructure,
	}
}

// Protocol represents the communication protocol for a device.
type Protocol string

// Protocol constants.
const (
	ProtocolKNX        Protocol = "knx"
	ProtocolDALI       Protocol = "dali"
	ProtocolModbusRTU  Protocol = "modbus_rtu"
	ProtocolModbusTCP  Protocol = "modbus_tcp"
	ProtocolBACnetIP   Protocol = "bacnet_ip"
	ProtocolBACnetMSTP Protocol = "bacnet_mstp"
	ProtocolMQTT       Protocol = "mqtt"
	ProtocolHTTP       Protocol = "http"
	ProtocolSIP        Protocol = "sip"
	ProtocolRTSP       Protocol = "rtsp"
	ProtocolONVIF      Protocol = "onvif"
	ProtocolOCPP       Protocol = "ocpp"
	ProtocolRS232      Protocol = "rs232"
	ProtocolRS485      Protocol = "rs485"
)

// AllProtocols returns all valid protocol values.
func AllProtocols() []Protocol {
	return []Protocol{
		ProtocolKNX, ProtocolDALI, ProtocolModbusRTU, ProtocolModbusTCP,
		ProtocolBACnetIP, ProtocolBACnetMSTP, ProtocolMQTT, ProtocolHTTP,
		ProtocolSIP, ProtocolRTSP, ProtocolONVIF, ProtocolOCPP,
		ProtocolRS232, ProtocolRS485,
	}
}

// DeviceType represents the specific kind of device.
type DeviceType string //nolint:revive // device.DeviceType is clearer than device.Type in calling code

// Lighting device types.
const (
	DeviceTypeLightSwitch DeviceType = "light_switch"
	DeviceTypeLightDimmer DeviceType = "light_dimmer"
	DeviceTypeLightCT     DeviceType = "light_ct"
	DeviceTypeLightRGB    DeviceType = "light_rgb"
	DeviceTypeLightRGBW   DeviceType = "light_rgbw"
	DeviceTypeDALIBallast DeviceType = "dali_ballast"
	DeviceTypeDALIGateway DeviceType = "dali_gateway"
)

// Climate device types.
const (
	DeviceTypeThermostat        DeviceType = "thermostat"
	DeviceTypeTemperatureSensor DeviceType = "temperature_sensor"
	DeviceTypeHumiditySensor    DeviceType = "humidity_sensor"
	DeviceTypeHVACUnit          DeviceType = "hvac_unit"
	DeviceTypeHeatingActuator   DeviceType = "heating_actuator"
	DeviceTypeValveActuator     DeviceType = "valve_actuator"
	DeviceTypeFCU               DeviceType = "fcu"
	DeviceTypeVAVBox            DeviceType = "vav_box"
	DeviceTypeAHU               DeviceType = "ahu"
)

// Blinds device types.
const (
	DeviceTypeBlindSwitch   DeviceType = "blind_switch"
	DeviceTypeBlindPosition DeviceType = "blind_position"
	DeviceTypeBlindTilt     DeviceType = "blind_tilt"
)

// Sensor device types.
const (
	DeviceTypeMotionSensor     DeviceType = "motion_sensor"
	DeviceTypePresenceSensor   DeviceType = "presence_sensor"
	DeviceTypeDoorSensor       DeviceType = "door_sensor"
	DeviceTypeWindowSensor     DeviceType = "window_sensor"
	DeviceTypeLeakSensor       DeviceType = "leak_sensor"
	DeviceTypeCO2Sensor        DeviceType = "co2_sensor"
	DeviceTypeLightSensor      DeviceType = "light_sensor"
	DeviceTypeAirQualitySensor DeviceType = "air_quality_sensor"
)

// Plant equipment device types.
const (
	DeviceTypePump       DeviceType = "pump"
	DeviceTypeBoiler     DeviceType = "boiler"
	DeviceTypeHeatPump   DeviceType = "heat_pump"
	DeviceTypeChiller    DeviceType = "chiller"
	DeviceTypeFan        DeviceType = "fan"
	DeviceTypeVFD        DeviceType = "vfd"
	DeviceTypeCompressor DeviceType = "compressor"
)

// Security device types.
const (
	DeviceTypeAlarmPanel  DeviceType = "alarm_panel"
	DeviceTypeCamera      DeviceType = "camera"
	DeviceTypeNVR         DeviceType = "nvr"
	DeviceTypeDoorLock    DeviceType = "door_lock"
	DeviceTypeDoorStation DeviceType = "door_station"
	DeviceTypeKeypad      DeviceType = "keypad"
)

// Energy device types.
const (
	DeviceTypeEnergyMeter    DeviceType = "energy_meter"
	DeviceTypeCTClamp        DeviceType = "ct_clamp"
	DeviceTypeEnergySubmeter DeviceType = "energy_submeter"
	DeviceTypeSolarInverter  DeviceType = "solar_inverter"
	DeviceTypeBatteryStorage DeviceType = "battery_storage"
	DeviceTypeEVCharger      DeviceType = "ev_charger"
)

// I/O device types.
const (
	DeviceTypeRelayModule   DeviceType = "relay_module"
	DeviceTypeRelayChannel  DeviceType = "relay_channel"
	DeviceTypeDigitalInput  DeviceType = "digital_input"
	DeviceTypeDigitalOutput DeviceType = "digital_output"
	DeviceTypeAnalogInput   DeviceType = "analog_input"
	DeviceTypeAnalogOutput  DeviceType = "analog_output"
)

// Audio/Video device types.
const (
	DeviceTypeAudioZone   DeviceType = "audio_zone"
	DeviceTypeAudioMatrix DeviceType = "audio_matrix"
	DeviceTypeVideoMatrix DeviceType = "video_matrix"
	DeviceTypeDisplay     DeviceType = "display"
	DeviceTypeProjector   DeviceType = "projector"
)

// KNX controls device types.
const (
	DeviceTypeSceneController DeviceType = "scene_controller"
	DeviceTypePushButton      DeviceType = "push_button"
	DeviceTypeBinaryInput     DeviceType = "binary_input"
	DeviceTypeRoomController  DeviceType = "room_controller"
	DeviceTypeLogicModule     DeviceType = "logic_module"
)

// KNX system/infrastructure device types.
const (
	DeviceTypeIPRouter       DeviceType = "ip_router"
	DeviceTypeLineCoupler    DeviceType = "line_coupler"
	DeviceTypePowerSupply    DeviceType = "power_supply"
	DeviceTypeTimerSwitch    DeviceType = "timer_switch"
	DeviceTypeLoadController DeviceType = "load_controller"
	DeviceTypeSwitchActuator DeviceType = "switch_actuator"
)

// Additional sensor types.
const (
	DeviceTypeMultiSensor DeviceType = "multi_sensor"
	DeviceTypeWindSensor  DeviceType = "wind_sensor"
)

// Other device types.
const (
	DeviceTypeGateway        DeviceType = "gateway"
	DeviceTypeIrrigation     DeviceType = "irrigation_controller"
	DeviceTypePoolPump       DeviceType = "pool_pump"
	DeviceTypePoolHeater     DeviceType = "pool_heater"
	DeviceTypeWeatherStation DeviceType = "weather_station"
)

// AllDeviceTypes returns all valid device type values.
func AllDeviceTypes() []DeviceType {
	return []DeviceType{
		// Lighting
		DeviceTypeLightSwitch, DeviceTypeLightDimmer, DeviceTypeLightCT,
		DeviceTypeLightRGB, DeviceTypeLightRGBW, DeviceTypeDALIBallast, DeviceTypeDALIGateway,
		// Climate
		DeviceTypeThermostat, DeviceTypeTemperatureSensor, DeviceTypeHumiditySensor,
		DeviceTypeHVACUnit, DeviceTypeHeatingActuator, DeviceTypeValveActuator, DeviceTypeFCU, DeviceTypeVAVBox, DeviceTypeAHU,
		// Blinds
		DeviceTypeBlindSwitch, DeviceTypeBlindPosition, DeviceTypeBlindTilt,
		// Sensors
		DeviceTypeMotionSensor, DeviceTypePresenceSensor, DeviceTypeDoorSensor,
		DeviceTypeWindowSensor, DeviceTypeLeakSensor, DeviceTypeCO2Sensor,
		DeviceTypeLightSensor, DeviceTypeAirQualitySensor,
		// Plant
		DeviceTypePump, DeviceTypeBoiler, DeviceTypeHeatPump, DeviceTypeChiller,
		DeviceTypeFan, DeviceTypeVFD, DeviceTypeCompressor,
		// Security
		DeviceTypeAlarmPanel, DeviceTypeCamera, DeviceTypeNVR,
		DeviceTypeDoorLock, DeviceTypeDoorStation, DeviceTypeKeypad,
		// Energy
		DeviceTypeEnergyMeter, DeviceTypeCTClamp, DeviceTypeEnergySubmeter,
		DeviceTypeSolarInverter, DeviceTypeBatteryStorage, DeviceTypeEVCharger,
		// I/O
		DeviceTypeRelayModule, DeviceTypeRelayChannel, DeviceTypeDigitalInput,
		DeviceTypeDigitalOutput, DeviceTypeAnalogInput, DeviceTypeAnalogOutput,
		// Audio/Video
		DeviceTypeAudioZone, DeviceTypeAudioMatrix, DeviceTypeVideoMatrix,
		DeviceTypeDisplay, DeviceTypeProjector,
		// KNX Controls
		DeviceTypeSceneController, DeviceTypePushButton, DeviceTypeBinaryInput,
		DeviceTypeRoomController, DeviceTypeLogicModule,
		// KNX System/Infrastructure
		DeviceTypeIPRouter, DeviceTypeLineCoupler, DeviceTypePowerSupply,
		DeviceTypeTimerSwitch, DeviceTypeLoadController, DeviceTypeSwitchActuator,
		// Additional Sensors
		DeviceTypeMultiSensor, DeviceTypeWindSensor,
		// Other
		DeviceTypeGateway, DeviceTypeIrrigation, DeviceTypePoolPump,
		DeviceTypePoolHeater, DeviceTypeWeatherStation,
	}
}

// Capability represents what a device can do.
type Capability string

// Control capabilities.
const (
	CapOnOff     Capability = "on_off"
	CapDim       Capability = "dim"
	CapColorTemp Capability = "color_temp" //nolint:misspell // KNX standard uses American "color"
	CapColorRGB  Capability = "color_rgb"  //nolint:misspell // KNX standard uses American "color"
	CapPosition  Capability = "position"
	CapTilt      Capability = "tilt"
	CapSpeed     Capability = "speed"
)

// Reading capabilities.
const (
	CapTemperatureRead Capability = "temperature_read"
	CapTemperatureSet  Capability = "temperature_set"
	CapHumidityRead    Capability = "humidity_read"
	CapPressureRead    Capability = "pressure_read"
	CapFlowRead        Capability = "flow_read"
	CapCO2Read         Capability = "co2_read"
	CapPowerRead       Capability = "power_read"
	CapEnergyRead      Capability = "energy_read"
	CapVoltageRead     Capability = "voltage_read"
	CapCurrentRead     Capability = "current_read"
	CapLightLevelRead  Capability = "light_level_read"
)

// Detection capabilities.
const (
	CapMotionDetect   Capability = "motion_detect"
	CapPresenceDetect Capability = "presence_detect"
	CapContactState   Capability = "contact_state"
	CapLeakDetect     Capability = "leak_detect"
	CapSmokeDetect    Capability = "smoke_detect"
)

// Security capabilities.
const (
	CapLockUnlock  Capability = "lock_unlock"
	CapArmDisarm   Capability = "arm_disarm"
	CapCardAccess  Capability = "card_access"
	CapVideoStream Capability = "video_stream"
)

// Equipment capabilities.
const (
	CapRunStop       Capability = "run_stop"
	CapSpeedControl  Capability = "speed_control"
	CapFaultStatus   Capability = "fault_status"
	CapEnableDisable Capability = "enable_disable"
	CapFilterStatus  Capability = "filter_status"
	CapModeSelect    Capability = "mode_select"
)

// Health monitoring capabilities.
const (
	CapVibrationRead   Capability = "vibration_read"
	CapBearingTempRead Capability = "bearing_temp_read"
	CapOilPressureRead Capability = "oil_pressure_read"
	CapBatteryStatus   Capability = "battery_status"
	CapRuntimeHours    Capability = "runtime_hours"
)

// AllCapabilities returns all valid capability values.
func AllCapabilities() []Capability {
	return []Capability{
		// Control
		CapOnOff, CapDim, CapColorTemp, CapColorRGB, CapPosition, CapTilt, CapSpeed,
		// Reading
		CapTemperatureRead, CapTemperatureSet, CapHumidityRead, CapPressureRead,
		CapFlowRead, CapCO2Read, CapPowerRead, CapEnergyRead, CapVoltageRead,
		CapCurrentRead, CapLightLevelRead,
		// Detection
		CapMotionDetect, CapPresenceDetect, CapContactState, CapLeakDetect, CapSmokeDetect,
		// Security
		CapLockUnlock, CapArmDisarm, CapCardAccess, CapVideoStream,
		// Equipment
		CapRunStop, CapSpeedControl, CapFaultStatus, CapEnableDisable, CapFilterStatus, CapModeSelect,
		// Health
		CapVibrationRead, CapBearingTempRead, CapOilPressureRead, CapBatteryStatus, CapRuntimeHours,
	}
}

// HealthStatus represents the device health state.
type HealthStatus string

// HealthStatus constants.
const (
	HealthStatusOnline   HealthStatus = "online"
	HealthStatusOffline  HealthStatus = "offline"
	HealthStatusDegraded HealthStatus = "degraded"
	HealthStatusUnknown  HealthStatus = "unknown"
)

// AllHealthStatuses returns all valid health status values.
func AllHealthStatuses() []HealthStatus {
	return []HealthStatus{
		HealthStatusOnline, HealthStatusOffline, HealthStatusDegraded, HealthStatusUnknown,
	}
}
