package knx

import (
	"math"
	"testing"
)

// ─── DPT1 (Boolean) ────────────────────────────────────────────────

func TestEncodeDPT1(t *testing.T) {
	tests := []struct {
		name  string
		value bool
		want  byte
	}{
		{"true", true, 0x01},
		{"false", false, 0x00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeDPT1(tt.value)
			if len(got) != 1 || got[0] != tt.want {
				t.Errorf("EncodeDPT1(%v) = %v, want [%02X]", tt.value, got, tt.want)
			}
		})
	}
}

func TestDecodeDPT1(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    bool
		wantErr bool
	}{
		{"0x00 is false", []byte{0x00}, false, false},
		{"0x01 is true", []byte{0x01}, true, false},
		{"0xFF is true (LSB=1)", []byte{0xFF}, true, false},
		{"0x80 is false (LSB=0)", []byte{0x80}, false, false}, // DPT1 only checks bit 0
		{"0x03 is true", []byte{0x03}, true, false},
		{"empty data", []byte{}, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeDPT1(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeDPT1() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("DecodeDPT1(%v) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

// ─── DPT3 (Dimming Control) ────────────────────────────────────────

func TestEncodeDPT3(t *testing.T) {
	tests := []struct {
		name     string
		increase bool
		steps    uint8
		want     byte
	}{
		{"increase 7 steps", true, 7, 0x0F},   // 0000 1111
		{"decrease 7 steps", false, 7, 0x07},  // 0000 0111
		{"increase 1 step", true, 1, 0x09},    // 0000 1001
		{"decrease 1 step", false, 1, 0x01},   // 0000 0001
		{"increase stop (0)", true, 0, 0x08},  // 0000 1000
		{"decrease stop (0)", false, 0, 0x00}, // 0000 0000
		{"steps capped at 7", true, 15, 0x0F}, // Should mask to 7
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeDPT3(tt.increase, tt.steps)
			if len(got) != 1 || got[0] != tt.want {
				t.Errorf("EncodeDPT3(%v, %d) = %v, want [%02X]", tt.increase, tt.steps, got, tt.want)
			}
		})
	}
}

func TestDecodeDPT3(t *testing.T) {
	tests := []struct {
		name         string
		data         []byte
		wantIncrease bool
		wantSteps    uint8
		wantErr      bool
	}{
		{"increase 7", []byte{0x0F}, true, 7, false},
		{"decrease 7", []byte{0x07}, false, 7, false},
		{"increase 1", []byte{0x09}, true, 1, false},
		{"decrease stop", []byte{0x00}, false, 0, false},
		{"empty data", []byte{}, false, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			increase, steps, err := DecodeDPT3(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeDPT3() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if increase != tt.wantIncrease {
					t.Errorf("DecodeDPT3() increase = %v, want %v", increase, tt.wantIncrease)
				}
				if steps != tt.wantSteps {
					t.Errorf("DecodeDPT3() steps = %v, want %v", steps, tt.wantSteps)
				}
			}
		})
	}
}

// ─── DPT5 (Percentage 0-100%) ──────────────────────────────────────

func TestEncodeDPT5(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		want    byte
	}{
		{"0%", 0, 0x00},
		{"50%", 50, 0x80}, // 127.5 rounds to 128
		{"100%", 100, 0xFF},
		{"negative clamped", -10, 0x00},
		{"over 100 clamped", 150, 0xFF},
		{"25%", 25, 0x40}, // 63.75 rounds to 64
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeDPT5(tt.percent)
			if len(got) != 1 || got[0] != tt.want {
				t.Errorf("EncodeDPT5(%v) = %v, want [%02X]", tt.percent, got, tt.want)
			}
		})
	}
}

func TestDecodeDPT5(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    float64
		wantErr bool
	}{
		{"0x00 is 0%", []byte{0x00}, 0, false},
		{"0xFF is 100%", []byte{0xFF}, 100, false},
		{"0x80 is ~50%", []byte{0x80}, 50.196, false}, // 128/255*100
		{"empty data", []byte{}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeDPT5(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeDPT5() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && math.Abs(got-tt.want) > 0.01 {
				t.Errorf("DecodeDPT5(%v) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

// ─── DPT5 Angle (0-360°) ───────────────────────────────────────────

func TestEncodeDPT5Angle(t *testing.T) {
	tests := []struct {
		name  string
		angle float64
		want  byte
	}{
		{"0°", 0, 0x00},
		{"180°", 180, 0x80},
		{"360°", 360, 0xFF},
		{"negative clamped", -10, 0x00},
		{"over 360 clamped", 400, 0xFF},
		{"90°", 90, 0x40}, // 64
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeDPT5Angle(tt.angle)
			if len(got) != 1 || got[0] != tt.want {
				t.Errorf("EncodeDPT5Angle(%v) = %v, want [%02X]", tt.angle, got, tt.want)
			}
		})
	}
}

func TestDecodeDPT5Angle(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    float64
		wantErr bool
	}{
		{"0x00 is 0°", []byte{0x00}, 0, false},
		{"0xFF is 360°", []byte{0xFF}, 360, false},
		{"0x80 is 180°", []byte{0x80}, 180.7, false},
		{"empty data", []byte{}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeDPT5Angle(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeDPT5Angle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && math.Abs(got-tt.want) > 0.1 {
				t.Errorf("DecodeDPT5Angle(%v) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

// ─── DPT9 (2-byte Float) ───────────────────────────────────────────

func TestEncodeDPT9(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		wantErr bool
	}{
		{"zero", 0, false},
		{"room temperature", 21.5, false},
		{"negative", -10.5, false},
		{"lux value", 500.0, false},
		{"humidity", 65.5, false},
		{"out of range positive", 700000, true},
		{"out of range negative", -700000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EncodeDPT9(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeDPT9(%v) error = %v, wantErr %v", tt.value, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != 2 {
					t.Errorf("EncodeDPT9(%v) returned %d bytes, want 2", tt.value, len(got))
				}
			}
		})
	}
}

func TestDecodeDPT9(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    float64
		wantErr bool
	}{
		{"zero", []byte{0x00, 0x00}, 0, false},
		{"21°C encoded", []byte{0x0C, 0x1A}, 21.0, false}, // Approximate
		{"empty data", []byte{}, 0, true},
		{"one byte only", []byte{0x0C}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeDPT9(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeDPT9() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && math.Abs(got-tt.want) > 1.0 {
				t.Errorf("DecodeDPT9(%v) = %v, want ~%v", tt.data, got, tt.want)
			}
		})
	}
}

func TestDPT9_RoundTrip(t *testing.T) {
	// Test that encode → decode returns approximately the same value
	values := []float64{0, 21.5, -10.0, 100.0, 500.0, -40.0}

	for _, v := range values {
		encoded, err := EncodeDPT9(v)
		if err != nil {
			t.Errorf("EncodeDPT9(%v) error = %v", v, err)
			continue
		}

		decoded, err := DecodeDPT9(encoded)
		if err != nil {
			t.Errorf("DecodeDPT9() error = %v", err)
			continue
		}

		// DPT9 has limited precision, allow 1% tolerance
		tolerance := math.Abs(v) * 0.01
		if tolerance < 0.1 {
			tolerance = 0.1
		}
		if math.Abs(decoded-v) > tolerance {
			t.Errorf("DPT9 round trip: %v → %v → %v (diff: %v)", v, encoded, decoded, decoded-v)
		}
	}
}

// ─── DPT17 (Scene Number) ──────────────────────────────────────────

func TestEncodeDPT17(t *testing.T) {
	tests := []struct {
		name    string
		scene   uint8
		want    byte
		wantErr bool
	}{
		{"scene 0", 0, 0x00, false},
		{"scene 1", 1, 0x01, false},
		{"scene 63", 63, 0x3F, false},
		{"scene 64 (invalid)", 64, 0, true},
		{"scene 255 (invalid)", 255, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EncodeDPT17(tt.scene)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeDPT17(%v) error = %v, wantErr %v", tt.scene, err, tt.wantErr)
				return
			}
			if !tt.wantErr && (len(got) != 1 || got[0] != tt.want) {
				t.Errorf("EncodeDPT17(%v) = %v, want [%02X]", tt.scene, got, tt.want)
			}
		})
	}
}

func TestDecodeDPT17(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    uint8
		wantErr bool
	}{
		{"scene 0", []byte{0x00}, 0, false},
		{"scene 1", []byte{0x01}, 1, false},
		{"scene 63", []byte{0x3F}, 63, false},
		{"masked to 6 bits", []byte{0xFF}, 63, false}, // Should mask
		{"empty data", []byte{}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeDPT17(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeDPT17() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("DecodeDPT17(%v) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

// ─── DPT18 (Scene Control) ─────────────────────────────────────────

func TestEncodeDPT18(t *testing.T) {
	tests := []struct {
		name    string
		scene   uint8
		learn   bool
		want    byte
		wantErr bool
	}{
		{"recall scene 0", 0, false, 0x00, false},
		{"recall scene 1", 1, false, 0x01, false},
		{"learn scene 0", 0, true, 0x80, false},
		{"learn scene 63", 63, true, 0xBF, false},
		{"scene 64 (invalid)", 64, false, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EncodeDPT18(tt.scene, tt.learn)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeDPT18(%v, %v) error = %v, wantErr %v", tt.scene, tt.learn, err, tt.wantErr)
				return
			}
			if !tt.wantErr && (len(got) != 1 || got[0] != tt.want) {
				t.Errorf("EncodeDPT18(%v, %v) = %v, want [%02X]", tt.scene, tt.learn, got, tt.want)
			}
		})
	}
}

func TestDecodeDPT18(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantScene uint8
		wantLearn bool
		wantErr   bool
	}{
		{"recall scene 0", []byte{0x00}, 0, false, false},
		{"recall scene 1", []byte{0x01}, 1, false, false},
		{"learn scene 0", []byte{0x80}, 0, true, false},
		{"learn scene 63", []byte{0xBF}, 63, true, false},
		{"empty data", []byte{}, 0, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene, learn, err := DecodeDPT18(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeDPT18() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if scene != tt.wantScene {
					t.Errorf("DecodeDPT18() scene = %v, want %v", scene, tt.wantScene)
				}
				if learn != tt.wantLearn {
					t.Errorf("DecodeDPT18() learn = %v, want %v", learn, tt.wantLearn)
				}
			}
		})
	}
}

// ─── DPT232 (RGB Colour) ───────────────────────────────────────────

func TestEncodeDPT232(t *testing.T) {
	tests := []struct {
		name      string
		rgb       RGB
		wantBytes []byte
	}{
		{"black", RGB{0, 0, 0}, []byte{0x00, 0x00, 0x00}},
		{"white", RGB{255, 255, 255}, []byte{0xFF, 0xFF, 0xFF}},
		{"red", RGB{255, 0, 0}, []byte{0xFF, 0x00, 0x00}},
		{"green", RGB{0, 255, 0}, []byte{0x00, 0xFF, 0x00}},
		{"blue", RGB{0, 0, 255}, []byte{0x00, 0x00, 0xFF}},
		{"purple", RGB{128, 0, 128}, []byte{0x80, 0x00, 0x80}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeDPT232(tt.rgb)
			if len(got) != 3 {
				t.Errorf("EncodeDPT232() returned %d bytes, want 3", len(got))
				return
			}
			for i, b := range tt.wantBytes {
				if got[i] != b {
					t.Errorf("EncodeDPT232()[%d] = %02X, want %02X", i, got[i], b)
				}
			}
		})
	}
}

func TestDecodeDPT232(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    RGB
		wantErr bool
	}{
		{"black", []byte{0x00, 0x00, 0x00}, RGB{0, 0, 0}, false},
		{"white", []byte{0xFF, 0xFF, 0xFF}, RGB{255, 255, 255}, false},
		{"red", []byte{0xFF, 0x00, 0x00}, RGB{255, 0, 0}, false},
		{"empty data", []byte{}, RGB{}, true},
		{"too short", []byte{0xFF, 0x00}, RGB{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeDPT232(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeDPT232() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got != tt.want {
					t.Errorf("DecodeDPT232() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
