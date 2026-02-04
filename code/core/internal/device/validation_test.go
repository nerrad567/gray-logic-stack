package device

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "valid name",
			input:   "Living Room Dimmer",
			wantErr: nil,
		},
		{
			name:    "valid name with numbers",
			input:   "Light 1",
			wantErr: nil,
		},
		{
			name:    "valid name with special characters",
			input:   "Kitchen (Main) Light",
			wantErr: nil,
		},
		{
			name:    "empty name",
			input:   "",
			wantErr: ErrInvalidName,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: ErrInvalidName,
		},
		{
			name:    "name at max length",
			input:   strings.Repeat("a", maxNameLength),
			wantErr: nil,
		},
		{
			name:    "name exceeds max length",
			input:   strings.Repeat("a", maxNameLength+1),
			wantErr: ErrInvalidName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateName(%q) = %v, want nil", tt.input, err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateName(%q) = %v, want %v", tt.input, err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateSlug(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "valid lowercase slug",
			input:   "living-room-dimmer",
			wantErr: nil,
		},
		{
			name:    "valid slug with numbers",
			input:   "light-1",
			wantErr: nil,
		},
		{
			name:    "valid single word",
			input:   "kitchen",
			wantErr: nil,
		},
		{
			name:    "valid numbers only",
			input:   "123",
			wantErr: nil,
		},
		{
			name:    "empty slug",
			input:   "",
			wantErr: ErrInvalidSlug,
		},
		{
			name:    "uppercase letters",
			input:   "Living-Room",
			wantErr: ErrInvalidSlug,
		},
		{
			name:    "spaces",
			input:   "living room",
			wantErr: ErrInvalidSlug,
		},
		{
			name:    "underscores",
			input:   "living_room",
			wantErr: ErrInvalidSlug,
		},
		{
			name:    "leading hyphen",
			input:   "-living-room",
			wantErr: ErrInvalidSlug,
		},
		{
			name:    "trailing hyphen",
			input:   "living-room-",
			wantErr: ErrInvalidSlug,
		},
		{
			name:    "consecutive hyphens",
			input:   "living--room",
			wantErr: ErrInvalidSlug,
		},
		{
			name:    "special characters",
			input:   "living@room",
			wantErr: ErrInvalidSlug,
		},
		{
			name:    "slug at max length",
			input:   strings.Repeat("a", maxSlugLength),
			wantErr: nil,
		},
		{
			name:    "slug exceeds max length",
			input:   strings.Repeat("a", maxSlugLength+1),
			wantErr: ErrInvalidSlug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSlug(tt.input)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateSlug(%q) = %v, want nil", tt.input, err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateSlug(%q) = %v, want %v", tt.input, err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name    string
		input   Domain
		wantErr error
	}{
		{name: "lighting", input: DomainLighting, wantErr: nil},
		{name: "climate", input: DomainClimate, wantErr: nil},
		{name: "blinds", input: DomainBlinds, wantErr: nil},
		{name: "audio", input: DomainAudio, wantErr: nil},
		{name: "video", input: DomainVideo, wantErr: nil},
		{name: "security", input: DomainSecurity, wantErr: nil},
		{name: "access", input: DomainAccess, wantErr: nil},
		{name: "energy", input: DomainEnergy, wantErr: nil},
		{name: "plant", input: DomainPlant, wantErr: nil},
		{name: "irrigation", input: DomainIrrigation, wantErr: nil},
		{name: "safety", input: DomainSafety, wantErr: nil},
		{name: "sensor", input: DomainSensor, wantErr: nil},
		{name: "invalid domain", input: Domain("invalid"), wantErr: ErrInvalidDomain},
		{name: "empty domain", input: Domain(""), wantErr: ErrInvalidDomain},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDomain(tt.input)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateDomain(%q) = %v, want nil", tt.input, err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateDomain(%q) = %v, want %v", tt.input, err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateProtocol(t *testing.T) {
	tests := []struct {
		name    string
		input   Protocol
		wantErr error
	}{
		{name: "knx", input: ProtocolKNX, wantErr: nil},
		{name: "dali", input: ProtocolDALI, wantErr: nil},
		{name: "modbus_rtu", input: ProtocolModbusRTU, wantErr: nil},
		{name: "modbus_tcp", input: ProtocolModbusTCP, wantErr: nil},
		{name: "bacnet_ip", input: ProtocolBACnetIP, wantErr: nil},
		{name: "bacnet_mstp", input: ProtocolBACnetMSTP, wantErr: nil},
		{name: "mqtt", input: ProtocolMQTT, wantErr: nil},
		{name: "http", input: ProtocolHTTP, wantErr: nil},
		{name: "sip", input: ProtocolSIP, wantErr: nil},
		{name: "rtsp", input: ProtocolRTSP, wantErr: nil},
		{name: "onvif", input: ProtocolONVIF, wantErr: nil},
		{name: "ocpp", input: ProtocolOCPP, wantErr: nil},
		{name: "rs232", input: ProtocolRS232, wantErr: nil},
		{name: "rs485", input: ProtocolRS485, wantErr: nil},
		{name: "invalid protocol", input: Protocol("invalid"), wantErr: ErrInvalidProtocol},
		{name: "empty protocol", input: Protocol(""), wantErr: ErrInvalidProtocol},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProtocol(tt.input)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateProtocol(%q) = %v, want nil", tt.input, err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateProtocol(%q) = %v, want %v", tt.input, err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateDeviceType(t *testing.T) {
	tests := []struct {
		name    string
		input   DeviceType
		wantErr error
	}{
		// Sample of device types - testing a few from each category
		{name: "light_switch", input: DeviceTypeLightSwitch, wantErr: nil},
		{name: "light_dimmer", input: DeviceTypeLightDimmer, wantErr: nil},
		{name: "thermostat", input: DeviceTypeThermostat, wantErr: nil},
		{name: "blind_position", input: DeviceTypeBlindPosition, wantErr: nil},
		{name: "motion_sensor", input: DeviceTypeMotionSensor, wantErr: nil},
		{name: "pump", input: DeviceTypePump, wantErr: nil},
		{name: "camera", input: DeviceTypeCamera, wantErr: nil},
		{name: "energy_meter", input: DeviceTypeEnergyMeter, wantErr: nil},
		{name: "relay_module", input: DeviceTypeRelayModule, wantErr: nil},
		{name: "audio_zone", input: DeviceTypeAudioZone, wantErr: nil},
		{name: "gateway", input: DeviceTypeGateway, wantErr: nil},
		{name: "invalid type", input: DeviceType("invalid"), wantErr: ErrInvalidDeviceType},
		{name: "empty type", input: DeviceType(""), wantErr: ErrInvalidDeviceType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeviceType(tt.input)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateDeviceType(%q) = %v, want nil", tt.input, err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateDeviceType(%q) = %v, want %v", tt.input, err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateCapabilities(t *testing.T) {
	tests := []struct {
		name    string
		input   []Capability
		wantErr error
	}{
		{
			name:    "single valid capability",
			input:   []Capability{CapOnOff},
			wantErr: nil,
		},
		{
			name:    "multiple valid capabilities",
			input:   []Capability{CapOnOff, CapDim, CapColorTemp},
			wantErr: nil,
		},
		{
			name:    "all control capabilities",
			input:   []Capability{CapOnOff, CapDim, CapColorTemp, CapColorRGB, CapPosition, CapTilt, CapSpeed},
			wantErr: nil,
		},
		{
			name:    "empty capabilities",
			input:   []Capability{},
			wantErr: nil,
		},
		{
			name:    "nil capabilities",
			input:   nil,
			wantErr: nil,
		},
		{
			name:    "one invalid capability",
			input:   []Capability{Capability("invalid")},
			wantErr: ErrInvalidCapability,
		},
		{
			name:    "valid and invalid mixed",
			input:   []Capability{CapOnOff, Capability("invalid"), CapDim},
			wantErr: ErrInvalidCapability,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCapabilities(tt.input)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateCapabilities(%v) = %v, want nil", tt.input, err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateCapabilities(%v) = %v, want %v", tt.input, err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateHealthStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   HealthStatus
		wantErr error
	}{
		{name: "online", input: HealthStatusOnline, wantErr: nil},
		{name: "offline", input: HealthStatusOffline, wantErr: nil},
		{name: "degraded", input: HealthStatusDegraded, wantErr: nil},
		{name: "unknown", input: HealthStatusUnknown, wantErr: nil},
		{name: "invalid status", input: HealthStatus("invalid"), wantErr: ErrInvalidState},
		{name: "empty status", input: HealthStatus(""), wantErr: ErrInvalidState},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHealthStatus(tt.input)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateHealthStatus(%q) = %v, want nil", tt.input, err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateHealthStatus(%q) = %v, want %v", tt.input, err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateAddress(t *testing.T) {
	tests := []struct {
		name     string
		protocol Protocol
		address  Address
		wantErr  error
	}{
		// KNX addresses (new structured functions format)
		{
			name:     "valid KNX address with functions",
			protocol: ProtocolKNX,
			address: Address{"functions": map[string]any{
				"switch": map[string]any{"ga": "1/2/3", "dpt": "1.001", "flags": []any{"write"}},
			}},
			wantErr: nil,
		},
		{
			name:     "KNX with multiple functions",
			protocol: ProtocolKNX,
			address: Address{"functions": map[string]any{
				"switch":        map[string]any{"ga": "1/2/3", "dpt": "1.001", "flags": []any{"write"}},
				"switch_status": map[string]any{"ga": "1/2/4", "dpt": "1.001", "flags": []any{"read", "transmit"}},
			}},
			wantErr: nil,
		},
		{
			name:     "KNX missing functions map",
			protocol: ProtocolKNX,
			address:  Address{"individual_address": "1.1.1"},
			wantErr:  ErrInvalidAddress,
		},
		{
			name:     "KNX empty functions map",
			protocol: ProtocolKNX,
			address:  Address{"functions": map[string]any{}},
			wantErr:  ErrInvalidAddress,
		},
		{
			name:     "KNX function with empty ga",
			protocol: ProtocolKNX,
			address: Address{"functions": map[string]any{
				"switch": map[string]any{"ga": "", "dpt": "1.001", "flags": []any{"write"}},
			}},
			wantErr: ErrInvalidAddress,
		},

		// DALI addresses
		{
			name:     "valid DALI with short_address",
			protocol: ProtocolDALI,
			address:  Address{"gateway": "dali-gw-01", "short_address": 15},
			wantErr:  nil,
		},
		{
			name:     "valid DALI with group",
			protocol: ProtocolDALI,
			address:  Address{"gateway": "dali-gw-01", "group": 0},
			wantErr:  nil,
		},
		{
			name:     "DALI missing gateway",
			protocol: ProtocolDALI,
			address:  Address{"short_address": 15},
			wantErr:  ErrInvalidAddress,
		},
		{
			name:     "DALI missing address type",
			protocol: ProtocolDALI,
			address:  Address{"gateway": "dali-gw-01"},
			wantErr:  ErrInvalidAddress,
		},

		// Modbus addresses
		{
			name:     "valid Modbus TCP",
			protocol: ProtocolModbusTCP,
			address:  Address{"host": "192.168.1.100", "port": 502, "unit_id": 1},
			wantErr:  nil,
		},
		{
			name:     "valid Modbus RTU",
			protocol: ProtocolModbusRTU,
			address:  Address{"device": "/dev/ttyUSB0", "unit_id": 1},
			wantErr:  nil,
		},
		{
			name:     "Modbus missing host and device",
			protocol: ProtocolModbusTCP,
			address:  Address{"unit_id": 1},
			wantErr:  ErrInvalidAddress,
		},
		{
			name:     "Modbus missing unit_id",
			protocol: ProtocolModbusTCP,
			address:  Address{"host": "192.168.1.100"},
			wantErr:  ErrInvalidAddress,
		},

		// MQTT addresses
		{
			name:     "valid MQTT",
			protocol: ProtocolMQTT,
			address:  Address{"topic": "home/living-room/light"},
			wantErr:  nil,
		},
		{
			name:     "MQTT missing topic",
			protocol: ProtocolMQTT,
			address:  Address{"broker": "tcp://localhost:1883"},
			wantErr:  ErrInvalidAddress,
		},

		// Other protocols (just check non-empty)
		{
			name:     "HTTP with any address",
			protocol: ProtocolHTTP,
			address:  Address{"url": "http://192.168.1.100/api"},
			wantErr:  nil,
		},
		{
			name:     "empty address for HTTP",
			protocol: ProtocolHTTP,
			address:  Address{},
			wantErr:  ErrInvalidAddress,
		},
		{
			name:     "BACnet IP",
			protocol: ProtocolBACnetIP,
			address:  Address{"device_id": 1234},
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAddress(tt.protocol, tt.address)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateAddress(%q, %v) = %v, want nil", tt.protocol, tt.address, err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateAddress(%q, %v) = %v, want %v", tt.protocol, tt.address, err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateDevice(t *testing.T) {
	validDevice := func() *Device {
		return &Device{
			Name:     "Living Room Dimmer",
			Slug:     "living-room-dimmer",
			Type:     DeviceTypeLightDimmer,
			Domain:   DomainLighting,
			Protocol: ProtocolKNX,
			Address: Address{"functions": map[string]any{
				"switch":     map[string]any{"ga": "1/2/3", "dpt": "1.001", "flags": []any{"write"}},
				"brightness": map[string]any{"ga": "1/2/4", "dpt": "5.001", "flags": []any{"write"}},
			}},
			Capabilities: []Capability{CapOnOff, CapDim},
			HealthStatus: HealthStatusOnline,
		}
	}

	tests := []struct {
		name    string
		modify  func(*Device)
		wantErr error
	}{
		{
			name:    "valid device",
			modify:  func(d *Device) {},
			wantErr: nil,
		},
		{
			name:    "nil device",
			modify:  nil,
			wantErr: ErrInvalidDevice,
		},
		{
			name:    "empty name",
			modify:  func(d *Device) { d.Name = "" },
			wantErr: ErrInvalidName,
		},
		{
			name:    "invalid slug",
			modify:  func(d *Device) { d.Slug = "Invalid Slug" },
			wantErr: ErrInvalidSlug,
		},
		{
			name:    "empty slug allowed",
			modify:  func(d *Device) { d.Slug = "" },
			wantErr: nil, // Empty slug is allowed (will be generated)
		},
		{
			name:    "invalid domain",
			modify:  func(d *Device) { d.Domain = Domain("invalid") },
			wantErr: ErrInvalidDomain,
		},
		{
			name:    "invalid protocol",
			modify:  func(d *Device) { d.Protocol = Protocol("invalid") },
			wantErr: ErrInvalidProtocol,
		},
		{
			name:    "invalid device type",
			modify:  func(d *Device) { d.Type = DeviceType("invalid") },
			wantErr: ErrInvalidDeviceType,
		},
		{
			name:    "nil address",
			modify:  func(d *Device) { d.Address = nil },
			wantErr: ErrInvalidAddress,
		},
		{
			name:    "empty address",
			modify:  func(d *Device) { d.Address = Address{} },
			wantErr: ErrInvalidAddress,
		},
		{
			name:    "invalid address for protocol",
			modify:  func(d *Device) { d.Address = Address{"invalid": "value"} },
			wantErr: ErrInvalidAddress,
		},
		{
			name:    "invalid capability",
			modify:  func(d *Device) { d.Capabilities = []Capability{Capability("invalid")} },
			wantErr: ErrInvalidCapability,
		},
		{
			name:    "empty capabilities allowed",
			modify:  func(d *Device) { d.Capabilities = nil },
			wantErr: nil,
		},
		{
			name:    "invalid health status",
			modify:  func(d *Device) { d.HealthStatus = HealthStatus("invalid") },
			wantErr: ErrInvalidState,
		},
		{
			name:    "empty health status allowed",
			modify:  func(d *Device) { d.HealthStatus = "" },
			wantErr: nil, // Empty health status is allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d *Device
			if tt.modify != nil {
				d = validDevice()
				tt.modify(d)
			}

			err := ValidateDevice(d)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateDevice() = %v, want nil", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateDevice() = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple name",
			input: "Living Room",
			want:  "living-room",
		},
		{
			name:  "already lowercase",
			input: "kitchen",
			want:  "kitchen",
		},
		{
			name:  "with numbers",
			input: "Light 1",
			want:  "light-1",
		},
		{
			name:  "underscores to hyphens",
			input: "master_bedroom",
			want:  "master-bedroom",
		},
		{
			name:  "special characters removed",
			input: "Kitchen (Main) Light!",
			want:  "kitchen-main-light",
		},
		{
			name:  "multiple spaces",
			input: "Living   Room",
			want:  "living-room",
		},
		{
			name:  "leading/trailing spaces",
			input: "  Bedroom  ",
			want:  "bedroom",
		},
		{
			name:  "mixed case",
			input: "LiViNg RoOm DiMmEr",
			want:  "living-room-dimmer",
		},
		{
			name:  "truncates long names",
			input: strings.Repeat("a", 100),
			want:  strings.Repeat("a", maxSlugLength),
		},
		{
			name:  "truncation doesn't end with hyphen",
			input: strings.Repeat("ab-", 50),
			want:  strings.TrimRight(strings.Repeat("ab-", 50)[:maxSlugLength], "-"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlug(tt.input)
			if got != tt.want {
				t.Errorf("GenerateSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
			// Validate the generated slug is valid
			if got != "" {
				if err := ValidateSlug(got); err != nil {
					t.Errorf("GenerateSlug(%q) produced invalid slug %q: %v", tt.input, got, err)
				}
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	// Test that GenerateID produces valid UUIDs
	id1 := GenerateID()
	id2 := GenerateID()

	// Check format (should be 36 chars: 8-4-4-4-12)
	if len(id1) != 36 {
		t.Errorf("GenerateID() = %q, want 36 character UUID", id1)
	}

	// Check uniqueness
	if id1 == id2 {
		t.Errorf("GenerateID() produced duplicate IDs: %q", id1)
	}

	// Check UUID format
	parts := strings.Split(id1, "-")
	if len(parts) != 5 {
		t.Errorf("GenerateID() = %q, expected 5 hyphen-separated parts", id1)
	}
	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			t.Errorf("GenerateID() part %d has length %d, want %d", i, len(part), expectedLengths[i])
		}
	}
}

func TestAllDomains(t *testing.T) {
	domains := AllDomains()

	// Check we have all expected domains
	expected := []Domain{
		DomainLighting, DomainClimate, DomainBlinds, DomainAudio,
		DomainVideo, DomainSecurity, DomainAccess, DomainEnergy,
		DomainPlant, DomainIrrigation, DomainSafety, DomainSensor,
		DomainInfrastructure,
	}

	if len(domains) != len(expected) {
		t.Errorf("AllDomains() returned %d domains, want %d", len(domains), len(expected))
	}

	for _, d := range expected {
		found := false
		for _, got := range domains {
			if got == d {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AllDomains() missing domain %q", d)
		}
	}
}

func TestAllProtocols(t *testing.T) {
	protocols := AllProtocols()

	// Check we have all expected protocols
	if len(protocols) != 14 {
		t.Errorf("AllProtocols() returned %d protocols, want 14", len(protocols))
	}

	// Verify each protocol validates
	for _, p := range protocols {
		if err := ValidateProtocol(p); err != nil {
			t.Errorf("Protocol %q from AllProtocols() failed validation: %v", p, err)
		}
	}
}

func TestAllDeviceTypes(t *testing.T) {
	types := AllDeviceTypes()

	// Should have 50+ device types
	if len(types) < 50 {
		t.Errorf("AllDeviceTypes() returned %d types, want at least 50", len(types))
	}

	// Verify each type validates
	for _, dt := range types {
		if err := ValidateDeviceType(dt); err != nil {
			t.Errorf("DeviceType %q from AllDeviceTypes() failed validation: %v", dt, err)
		}
	}
}

func TestAllCapabilities(t *testing.T) {
	caps := AllCapabilities()

	// Should have 30+ capabilities
	if len(caps) < 30 {
		t.Errorf("AllCapabilities() returned %d capabilities, want at least 30", len(caps))
	}

	// Verify each capability validates
	if err := ValidateCapabilities(caps); err != nil {
		t.Errorf("AllCapabilities() contains invalid capability: %v", err)
	}
}

func TestAllHealthStatuses(t *testing.T) {
	statuses := AllHealthStatuses()

	expected := []HealthStatus{
		HealthStatusOnline, HealthStatusOffline, HealthStatusDegraded, HealthStatusUnknown,
	}

	if len(statuses) != len(expected) {
		t.Errorf("AllHealthStatuses() returned %d statuses, want %d", len(statuses), len(expected))
	}

	// Verify each status validates
	for _, s := range statuses {
		if err := ValidateHealthStatus(s); err != nil {
			t.Errorf("HealthStatus %q from AllHealthStatuses() failed validation: %v", s, err)
		}
	}
}
