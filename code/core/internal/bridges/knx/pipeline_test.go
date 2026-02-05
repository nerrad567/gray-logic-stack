package knx

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// mockDeviceRegistry is a mock that returns pre-configured RegistryDevices.
type mockDeviceRegistry struct {
	devices      []RegistryDevice
	stateUpdates map[string]map[string]any // deviceID → last state update
}

func newMockDeviceRegistry(devices []RegistryDevice) *mockDeviceRegistry {
	return &mockDeviceRegistry{
		devices:      devices,
		stateUpdates: make(map[string]map[string]any),
	}
}

func (m *mockDeviceRegistry) GetKNXDevices(_ context.Context) ([]RegistryDevice, error) {
	return m.devices, nil
}

func (m *mockDeviceRegistry) SetDeviceState(_ context.Context, id string, state map[string]any) error {
	m.stateUpdates[id] = state
	return nil
}

func (m *mockDeviceRegistry) SetDeviceHealth(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockDeviceRegistry) CreateDeviceIfNotExists(_ context.Context, _ DeviceSeed) error {
	return nil
}

// pipelineTestDevices returns a representative set of devices covering all
// major device types: light switch, dimmer, blind, thermostat, presence sensor,
// heating actuator, and multi-channel infrastructure.
func pipelineTestDevices() []RegistryDevice {
	return []RegistryDevice{
		{
			ID:     "light-switch-1",
			Type:   "light_switch",
			Domain: "lighting",
			Functions: map[string]FunctionMapping{
				"switch":        {GA: "1/0/1", DPT: "1.001", Flags: []string{"write"}},
				"switch_status": {GA: "1/0/2", DPT: "1.001", Flags: []string{"read", "transmit"}},
			},
		},
		{
			ID:     "dimmer-1",
			Type:   "light_dimmer",
			Domain: "lighting",
			Functions: map[string]FunctionMapping{
				"switch":            {GA: "1/1/0", DPT: "1.001", Flags: []string{"write"}},
				"switch_status":     {GA: "1/1/1", DPT: "1.001", Flags: []string{"read", "transmit"}},
				"brightness":        {GA: "1/1/2", DPT: "5.001", Flags: []string{"write"}},
				"brightness_status": {GA: "1/1/3", DPT: "5.001", Flags: []string{"read", "transmit"}},
			},
		},
		{
			ID:     "blind-1",
			Type:   "blind_tilt",
			Domain: "blinds",
			Functions: map[string]FunctionMapping{
				"position":        {GA: "2/0/0", DPT: "5.001", Flags: []string{"write"}},
				"position_status": {GA: "2/0/1", DPT: "5.001", Flags: []string{"read", "transmit"}},
				"slat":            {GA: "2/0/2", DPT: "5.001", Flags: []string{"write"}},
				"slat_status":     {GA: "2/0/3", DPT: "5.001", Flags: []string{"read", "transmit"}},
				"move":            {GA: "2/0/4", DPT: "1.008", Flags: []string{"write"}},
				"stop":            {GA: "2/0/5", DPT: "1.007", Flags: []string{"write"}},
			},
		},
		{
			ID:     "thermostat-1",
			Type:   "thermostat",
			Domain: "climate",
			Functions: map[string]FunctionMapping{
				"temperature":    {GA: "3/0/0", DPT: "9.001", Flags: []string{"read", "transmit"}},
				"setpoint":       {GA: "3/0/1", DPT: "9.001", Flags: []string{"write", "read"}},
				"heating_output": {GA: "3/0/2", DPT: "5.001", Flags: []string{"write"}},
			},
		},
		{
			ID:     "pir-1",
			Type:   "presence_sensor",
			Domain: "sensor",
			Functions: map[string]FunctionMapping{
				"presence": {GA: "4/0/0", DPT: "1.018", Flags: []string{"read", "transmit"}},
				"lux":      {GA: "4/0/1", DPT: "9.004", Flags: []string{"read", "transmit"}},
			},
		},
		{
			ID:     "heating-actuator-1",
			Type:   "heating_actuator",
			Domain: "climate",
			Functions: map[string]FunctionMapping{
				"valve":        {GA: "5/0/0", DPT: "5.001", Flags: []string{"write"}},
				"valve_status": {GA: "5/0/1", DPT: "5.001", Flags: []string{"read", "transmit"}},
			},
		},
		{
			ID:     "switch-actuator-infra",
			Type:   "switch_actuator",
			Domain: "infrastructure",
			Functions: map[string]FunctionMapping{
				"ch_a_switch":        {GA: "6/0/0", DPT: "1.001", Flags: []string{"write"}},
				"ch_a_switch_status": {GA: "6/0/1", DPT: "1.001", Flags: []string{"read", "transmit"}},
				"ch_b_switch":        {GA: "6/0/2", DPT: "1.001", Flags: []string{"write"}},
				"ch_b_switch_status": {GA: "6/0/3", DPT: "1.001", Flags: []string{"read", "transmit"}},
			},
		},
	}
}

// TestPipeline_TelegramToState tests the full pipeline from simulated KNX
// telegram → DPT decoding → state key mapping → MQTT publication.
// This is the key regression test that ensures DPTs, state keys, and
// function names work correctly across all layers.
func TestPipeline_TelegramToState(t *testing.T) { //nolint:gocognit // comprehensive table-driven test with many DPT cases
	mqtt := NewMockMQTTClient()
	connector := NewMockConnector()
	registry := newMockDeviceRegistry(pipelineTestDevices())

	cfg := createTestConfig()
	b := createBridgeWithRegistry(t, cfg, mqtt, connector, registry)

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	// Wait for initial load
	time.Sleep(50 * time.Millisecond)
	mqtt.ClearPublished()

	tests := []struct {
		name       string
		ga         string
		data       []byte
		wantDevice string
		wantKey    string
		wantValue  any
	}{
		// ── Light switch: DPT 1.001 boolean ──
		{
			name:       "switch on telegram",
			ga:         "1/0/2", // switch_status GA
			data:       []byte{0x01},
			wantDevice: "light-switch-1",
			wantKey:    "on",
			wantValue:  true,
		},
		{
			name:       "switch off telegram",
			ga:         "1/0/2",
			data:       []byte{0x00},
			wantDevice: "light-switch-1",
			wantKey:    "on",
			wantValue:  false,
		},

		// ── Dimmer: DPT 5.001 percentage ──
		{
			name:       "brightness 75%",
			ga:         "1/1/3",      // brightness_status GA
			data:       []byte{0xBF}, // 191/255 ≈ 74.9% → rounds to 75
			wantDevice: "dimmer-1",
			wantKey:    "level",
			wantValue:  float64(75),
		},

		// ── Blind: DPT 5.001 position ──
		{
			name:       "blind position 50%",
			ga:         "2/0/1",      // position_status GA
			data:       []byte{0x80}, // 128/255 ≈ 50.2% → rounds to 50
			wantDevice: "blind-1",
			wantKey:    "position",
			wantValue:  float64(50),
		},

		// ── Blind slat: DPT 5.001 tilt ──
		{
			name:       "slat tilt 25%",
			ga:         "2/0/3",      // slat_status GA
			data:       []byte{0x40}, // 64/255 ≈ 25.1% → rounds to 25
			wantDevice: "blind-1",
			wantKey:    "tilt",
			wantValue:  float64(25),
		},

		// ── Thermostat: DPT 9.001 temperature ──
		{
			name:       "temperature 21.5°C",
			ga:         "3/0/0", // temperature GA
			data:       encodeDPT9(21.5),
			wantDevice: "thermostat-1",
			wantKey:    "temperature",
			wantValue:  21.5,
		},

		// ── Thermostat setpoint: DPT 9.001 ──
		{
			name:       "setpoint 22.0°C",
			ga:         "3/0/1", // setpoint GA
			data:       encodeDPT9(22.0),
			wantDevice: "thermostat-1",
			wantKey:    "setpoint",
			wantValue:  22.0,
		},

		// ── Heating actuator valve: DPT 5.001 ──
		{
			name:       "valve 80%",
			ga:         "5/0/1",      // valve_status GA
			data:       []byte{0xCC}, // 204/255 ≈ 80% → rounds to 80
			wantDevice: "heating-actuator-1",
			wantKey:    "valve",
			wantValue:  float64(80),
		},

		// ── Presence sensor: DPT 1.018 boolean ──
		{
			name:       "presence detected",
			ga:         "4/0/0", // presence GA
			data:       []byte{0x01},
			wantDevice: "pir-1",
			wantKey:    "presence",
			wantValue:  true,
		},

		// ── Lux sensor: DPT 9.004 lux ──
		{
			name:       "lux 350",
			ga:         "4/0/1", // lux GA
			data:       encodeDPT9(350.0),
			wantDevice: "pir-1",
			wantKey:    "lux",
			wantValue:  350.0,
		},

		// ── Infrastructure channel: DPT 1.001 per-channel state ──
		{
			name:       "infra ch_a on",
			ga:         "6/0/1", // ch_a_switch_status
			data:       []byte{0x01},
			wantDevice: "switch-actuator-infra",
			wantKey:    "ch_a_on",
			wantValue:  true,
		},
		{
			name:       "infra ch_b off",
			ga:         "6/0/3", // ch_b_switch_status
			data:       []byte{0x00},
			wantDevice: "switch-actuator-infra",
			wantKey:    "ch_b_on",
			wantValue:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqtt.ClearPublished()

			ga, err := ParseGroupAddress(tt.ga)
			if err != nil {
				t.Fatalf("ParseGroupAddress(%q): %v", tt.ga, err)
			}

			// Simulate an incoming KNX telegram (write from bus device)
			connector.SimulateTelegram(Telegram{
				Source:      "1.1.1",
				Destination: ga,
				APCI:        APCIWrite,
				Data:        tt.data,
			})

			// Wait for processing
			time.Sleep(50 * time.Millisecond)

			// Find the state publication for the expected device.
			// The bridge publishes to GA-based topics (graylogic/state/knx/<GA>)
			// with the device_id inside the JSON payload.
			published := mqtt.GetPublished()
			found := false
			for _, pub := range published {
				if !strings.Contains(pub.Topic, "/state/") {
					continue
				}

				var msg map[string]any
				if err := json.Unmarshal(pub.Payload, &msg); err != nil {
					t.Fatalf("unmarshal state payload: %v", err)
				}

				// Check device_id in the payload
				devID, _ := msg["device_id"].(string)
				if devID != tt.wantDevice {
					continue
				}

				stateMap, _ := msg["state"].(map[string]any)
				if stateMap == nil {
					continue
				}

				val, exists := stateMap[tt.wantKey]
				if !exists {
					continue
				}

				found = true

				// Compare values with tolerance for floats
				switch want := tt.wantValue.(type) {
				case float64:
					got, ok := val.(float64)
					if !ok {
						t.Errorf("state[%q] = %T(%v), want float64", tt.wantKey, val, val)
						break
					}
					if diff := got - want; diff > 1.0 || diff < -1.0 {
						t.Errorf("state[%q] = %v, want ≈%v (diff=%v)", tt.wantKey, got, want, diff)
					}
				case bool:
					got, ok := val.(bool)
					if !ok {
						t.Errorf("state[%q] = %T(%v), want bool", tt.wantKey, val, val)
						break
					}
					if got != want {
						t.Errorf("state[%q] = %v, want %v", tt.wantKey, got, want)
					}
				default:
					if val != want {
						t.Errorf("state[%q] = %v, want %v", tt.wantKey, val, want)
					}
				}
			}

			if !found {
				topics := make([]string, len(published))
				for i, p := range published {
					topics[i] = fmt.Sprintf("%s → %s", p.Topic, string(p.Payload))
				}
				t.Errorf("no state publication found for device %q key %q;\npublished: %v",
					tt.wantDevice, tt.wantKey, topics)
			}
		})
	}
}

// TestPipeline_CommandToTelegram tests the reverse direction:
// MQTT command → bridge handler → KNX telegram with correct DPT encoding.
func TestPipeline_CommandToTelegram(t *testing.T) { //nolint:gocognit // comprehensive table-driven test
	mqtt := NewMockMQTTClient()
	connector := NewMockConnector()
	registry := newMockDeviceRegistry(pipelineTestDevices())

	cfg := createTestConfig()
	b := createBridgeWithRegistry(t, cfg, mqtt, connector, registry)

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	time.Sleep(50 * time.Millisecond)
	connector.ClearSent()

	tests := []struct {
		name     string
		deviceID string
		command  string
		params   map[string]any
		wantGA   string
		wantData []byte
	}{
		{
			name:     "turn on light switch",
			deviceID: "light-switch-1",
			command:  "on",
			wantGA:   "1/0/1", // switch write GA
			wantData: []byte{0x01},
		},
		{
			name:     "turn off light switch",
			deviceID: "light-switch-1",
			command:  "off",
			wantGA:   "1/0/1",
			wantData: []byte{0x00},
		},
		{
			name:     "set dimmer brightness",
			deviceID: "dimmer-1",
			command:  "dim",
			params:   map[string]any{"level": float64(50)},
			wantGA:   "1/1/2",      // brightness write GA
			wantData: []byte{0x80}, // 50% of 255 ≈ 128 = 0x80
		},
		{
			name:     "set blind position",
			deviceID: "blind-1",
			command:  "set_position",
			params:   map[string]any{"position": float64(75)},
			wantGA:   "2/0/0",      // position write GA
			wantData: []byte{0xBF}, // 75% of 255 ≈ 191
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connector.ClearSent()

			cmd := CommandMessage{
				ID:         "cmd-" + tt.name,
				DeviceID:   tt.deviceID,
				Command:    tt.command,
				Parameters: tt.params,
				Timestamp:  time.Now().UTC(),
			}
			payload, _ := json.Marshal(cmd)
			b.handleMQTTMessage("graylogic/command/knx/"+tt.deviceID, payload)

			time.Sleep(50 * time.Millisecond)

			telegrams := connector.GetSentTelegrams()
			if len(telegrams) == 0 {
				t.Fatal("no KNX telegram sent")
			}

			// Find telegram to expected GA
			found := false
			for _, tg := range telegrams {
				if tg.GA.String() == tt.wantGA {
					found = true
					if len(tg.Data) != len(tt.wantData) {
						t.Errorf("telegram data length = %d, want %d", len(tg.Data), len(tt.wantData))
					} else {
						for i := range tt.wantData {
							if tg.Data[i] != tt.wantData[i] {
								t.Errorf("telegram data[%d] = 0x%02X, want 0x%02X", i, tg.Data[i], tt.wantData[i])
							}
						}
					}
					break
				}
			}

			if !found {
				gas := make([]string, len(telegrams))
				for i, tg := range telegrams {
					gas[i] = tg.GA.String()
				}
				t.Errorf("no telegram to GA %s; sent to: %v", tt.wantGA, gas)
			}
		})
	}
}

// TestPipeline_DeviceCount verifies all test devices are loaded correctly.
func TestPipeline_DeviceCount(t *testing.T) {
	mqtt := NewMockMQTTClient()
	connector := NewMockConnector()
	registry := newMockDeviceRegistry(pipelineTestDevices())

	cfg := createTestConfig()
	b := createBridgeWithRegistry(t, cfg, mqtt, connector, registry)

	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer b.Stop()

	time.Sleep(50 * time.Millisecond)

	b.mappingMu.RLock()
	deviceCount := len(b.deviceToGAs)
	gaCount := len(b.gaToDevice)
	b.mappingMu.RUnlock()

	if deviceCount != 7 {
		t.Errorf("loaded %d devices, want 7", deviceCount)
	}

	// Total unique GAs: 2+4+6+3+2+2+4 = 23
	if gaCount != 23 {
		t.Errorf("mapped %d GAs, want 23", gaCount)
	}
}

// TestPipeline_ChannelStateKeys verifies infrastructure channel functions
// produce correctly prefixed state keys.
func TestPipeline_ChannelStateKeys(t *testing.T) {
	tests := []struct {
		function string
		wantKey  string
	}{
		{"ch_a_switch", "ch_a_on"},
		{"ch_a_switch_status", "ch_a_on"},
		{"ch_b_switch", "ch_b_on"},
		{"ch_b_switch_status", "ch_b_on"},
		{"ch_c_valve", "ch_c_valve"},
		{"ch_c_valve_status", "ch_c_valve"},
		{"channel_a_brightness", "channel_a_level"},
		{"channel_b_position_status", "channel_b_position"},
	}

	for _, tt := range tests {
		got := StateKeyForFunction(tt.function)
		if got != tt.wantKey {
			t.Errorf("StateKeyForFunction(%q) = %q, want %q", tt.function, got, tt.wantKey)
		}
	}
}

// createBridgeWithRegistry creates a test bridge that uses a mock DeviceRegistry
// instead of pre-loaded device configs.
func createBridgeWithRegistry(t *testing.T, cfg *Config, mqtt *MockMQTTClient, connector *MockConnector, registry DeviceRegistry) *Bridge {
	t.Helper()
	b, err := NewBridge(BridgeOptions{
		Config:     cfg,
		MQTTClient: mqtt,
		KNXDClient: connector,
	})
	if err != nil {
		t.Fatalf("NewBridge() error: %v", err)
	}
	b.registry = registry
	b.loadDevicesFromRegistry(context.Background())
	return b
}

// encodeDPT9 encodes a float64 as KNX DPT 9.xxx (2-byte float).
func encodeDPT9(value float64) []byte {
	// KNX DPT 9 format: MEEEEMMM MMMMMMMM
	// value = 0.01 * mantissa * 2^exponent
	v := value * 100 // Scale to 0.01 resolution

	var exponent int
	mantissa := v
	for mantissa < -2048 || mantissa > 2047 {
		mantissa /= 2
		exponent++
	}

	m := int(mantissa) & 0x7FF
	if value < 0 {
		m = (int(mantissa) + 2048) & 0x7FF
		return []byte{
			byte(0x80 | (exponent << 3) | (m >> 8)),
			byte(m & 0xFF),
		}
	}
	return []byte{
		byte((exponent << 3) | (m >> 8)),
		byte(m & 0xFF),
	}
}
