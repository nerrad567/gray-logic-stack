package knx

import (
	"fmt"
	"math"
)

// KNX Datapoint Type encoding constants.
const (
	// dpt5MaxValue is the maximum raw value for DPT5 (1-byte unsigned).
	dpt5MaxValue = 255

	// dpt5AngleMax is the maximum angle in degrees for DPT5.003.
	dpt5AngleMax = 360

	// dpt9MaxExponent is the maximum exponent for DPT9 2-byte float.
	dpt9MaxExponent = 15

	// dpt17MaxScene is the maximum scene number for DPT17/18.
	dpt17MaxScene = 63

	// dpt17SceneMask is the mask for extracting scene number.
	dpt17SceneMask = 0x3F

	// dptRGBBytes is the number of bytes for DPT232 RGB colour.
	dptRGBBytes = 3

	// byteShift is the bit shift for byte extraction.
	byteShift = 8

	// dpt9MantissaMask is the mask for extracting mantissa from DPT9.
	dpt9MantissaMask = 0x07FF
)

// DPT represents a KNX Datapoint Type identifier.
//
// Format: "major.minor" (e.g., "1.001", "9.001")
type DPT string

// Common DPT identifiers used in building automation.
const (
	// 1-bit types (DPT 1.xxx)
	DPTSwitch    DPT = "1.001" // 0=Off, 1=On
	DPTBool      DPT = "1.002" // 0=False, 1=True
	DPTEnable    DPT = "1.003" // 0=Disable, 1=Enable
	DPTStep      DPT = "1.007" // 0=Decrease, 1=Increase
	DPTUpDown    DPT = "1.008" // 0=Up, 1=Down
	DPTOpenClose DPT = "1.009" // 0=Open, 1=Close
	DPTStart     DPT = "1.010" // 0=Stop, 1=Start
	DPTTrigger   DPT = "1.017" // 1=Trigger

	// 4-bit types (DPT 3.xxx)
	DPTDimmingControl DPT = "3.007" // Direction + steps
	DPTBlindControl   DPT = "3.008" // Direction + steps

	// 1-byte unsigned types (DPT 5.xxx)
	DPTPercentage DPT = "5.001" // 0-100%
	DPTAngle      DPT = "5.003" // 0-360°
	DPTPercentU8  DPT = "5.004" // 0-255 raw

	// 2-byte float types (DPT 9.xxx)
	DPTTemperature DPT = "9.001" // -273 to 670760 °C
	DPTLux         DPT = "9.004" // 0 to 670760 lux
	DPTSpeed       DPT = "9.005" // m/s
	DPTHumidity    DPT = "9.007" // 0-100%
	DPTAirQuality  DPT = "9.008" // ppm

	// 1-byte scene types (DPT 17/18.xxx)
	DPTSceneNumber  DPT = "17.001" // 0-63 scene number
	DPTSceneControl DPT = "18.001" // Scene + learn bit

	// 3-byte colour types (DPT 232.xxx)
	DPTColourRGB DPT = "232.600" // R, G, B
)

// EncodeDPT1 encodes a boolean value to 1-bit KNX format.
//
// Used for: switch, bool, enable, step, up/down, open/close, start, trigger
//
// Parameters:
//   - value: Boolean value to encode
//
// Returns:
//   - []byte: Single byte with LSB set to 0 or 1
func EncodeDPT1(value bool) []byte {
	if value {
		return []byte{0x01}
	}
	return []byte{0x00}
}

// DecodeDPT1 decodes a 1-bit KNX value to boolean.
//
// Parameters:
//   - data: KNX data (at least 1 byte)
//
// Returns:
//   - bool: Decoded value
//   - error: If data is empty
func DecodeDPT1(data []byte) (bool, error) {
	if len(data) < 1 {
		return false, fmt.Errorf("%w: DPT1 requires 1 byte, got %d", ErrDecodingFailed, len(data))
	}
	return (data[0] & 0x01) != 0, nil
}

// EncodeDPT3 encodes a dimming/blind control value.
//
// Parameters:
//   - increase: True for increase/up, false for decrease/down
//   - steps: Number of steps (0-7, where 0 means stop)
//
// Returns:
//   - []byte: Single byte with control bits
func EncodeDPT3(increase bool, steps uint8) []byte {
	var value byte
	if increase {
		value = 0x08 // Bit 3 = direction (1=increase)
	}
	value |= (steps & 0x07) // Bits 0-2 = steps
	return []byte{value}
}

// DecodeDPT3 decodes a dimming/blind control value.
//
// Parameters:
//   - data: KNX data (at least 1 byte)
//
// Returns:
//   - increase: True for increase/up direction
//   - steps: Number of steps (0-7)
//   - error: If data is empty
func DecodeDPT3(data []byte) (increase bool, steps uint8, err error) {
	if len(data) < 1 {
		return false, 0, fmt.Errorf("%w: DPT3 requires 1 byte, got %d", ErrDecodingFailed, len(data))
	}
	increase = (data[0] & 0x08) != 0
	steps = data[0] & 0x07
	return increase, steps, nil
}

// EncodeDPT5 encodes a percentage (0-100) to 1-byte KNX format.
//
// DPT 5.001: Scales 0-100% to 0-255.
//
// Parameters:
//   - percent: Percentage value (0-100)
//
// Returns:
//   - []byte: Single byte with scaled value
func EncodeDPT5(percent float64) []byte {
	if percent < 0 {
		percent = 0
	} else if percent > 100 {
		percent = 100
	}
	value := uint8(math.Round(percent * 255 / 100))
	return []byte{value}
}

// DecodeDPT5 decodes a 1-byte KNX value to percentage.
//
// DPT 5.001: Scales 0-255 to 0-100%.
//
// Parameters:
//   - data: KNX data (at least 1 byte)
//
// Returns:
//   - float64: Percentage (0-100)
//   - error: If data is empty
func DecodeDPT5(data []byte) (float64, error) {
	if len(data) < 1 {
		return 0, fmt.Errorf("%w: DPT5 requires 1 byte, got %d", ErrDecodingFailed, len(data))
	}
	return float64(data[0]) * 100 / dpt5MaxValue, nil
}

// EncodeDPT5Angle encodes an angle (0-360) to 1-byte KNX format.
//
// DPT 5.003: Scales 0-360° to 0-255.
//
// Parameters:
//   - angle: Angle value (0-360)
//
// Returns:
//   - []byte: Single byte with scaled value
func EncodeDPT5Angle(angle float64) []byte {
	if angle < 0 {
		angle = 0
	} else if angle > dpt5AngleMax {
		angle = dpt5AngleMax
	}
	value := uint8(math.Round(angle * dpt5MaxValue / dpt5AngleMax))
	return []byte{value}
}

// DecodeDPT5Angle decodes a 1-byte KNX value to angle.
//
// DPT 5.003: Scales 0-255 to 0-360°.
//
// Parameters:
//   - data: KNX data (at least 1 byte)
//
// Returns:
//   - float64: Angle (0-360)
//   - error: If data is empty
func DecodeDPT5Angle(data []byte) (float64, error) {
	if len(data) < 1 {
		return 0, fmt.Errorf("%w: DPT5 angle requires 1 byte, got %d", ErrDecodingFailed, len(data))
	}
	return float64(data[0]) * dpt5AngleMax / dpt5MaxValue, nil
}

// EncodeDPT9 encodes a float value to 2-byte KNX floating point format.
//
// Used for: temperature, lux, humidity, etc.
//
// KNX 2-byte float format:
//
//	Byte 0: SEEE EMMM (Sign, Exponent high, Mantissa high)
//	Byte 1: MMMM MMMM (Mantissa low)
//
// Value = (0.01 × Mantissa) × 2^Exponent
//
// Parameters:
//   - value: Float value to encode
//
// Returns:
//   - []byte: Two bytes in KNX format
//   - error: If value is out of range
func EncodeDPT9(value float64) ([]byte, error) {
	if value < -671088.64 || value > 670760.96 {
		return nil, fmt.Errorf("%w: DPT9 value out of range: %.2f (valid: -671088.64 to 670760.96)", ErrEncodingFailed, value)
	}

	var sign uint16
	if value < 0 {
		sign = 0x8000
		value = -value
	}

	exp := 0
	mantissa := value * 100

	for mantissa > 2047 {
		mantissa /= 2
		exp++
	}

	if exp > dpt9MaxExponent {
		return nil, fmt.Errorf("%w: DPT9 exponent overflow for value %.2f", ErrEncodingFailed, value)
	}

	m := int16(mantissa)
	if sign != 0 {
		m = -m
	}

	// exp is validated ≤15 above, so conversion is safe
	encoded := sign | (uint16(exp) << 11) | (uint16(m) & 0x07FF) //nolint:gosec // exp bounded
	return []byte{byte(encoded >> byteShift), byte(encoded)}, nil
}

// DecodeDPT9 decodes a 2-byte KNX floating point value.
//
// Parameters:
//   - data: KNX data (at least 2 bytes)
//
// Returns:
//   - float64: Decoded value
//   - error: If data is too short
func DecodeDPT9(data []byte) (float64, error) {
	if len(data) < 2 {
		return 0, fmt.Errorf("%w: DPT9 requires 2 bytes, got %d", ErrDecodingFailed, len(data))
	}

	raw := uint16(data[0])<<8 | uint16(data[1])

	// KNX spec: 0x7FFF is the "invalid data" sentinel for all DPT 9.xxx types.
	if raw == 0x7FFF { //nolint:mnd // KNX DPT 9.x invalid/error sentinel value
		return 0, fmt.Errorf("%w: DPT9 invalid value 0x7FFF (sensor error or not available)", ErrDecodingFailed)
	}

	sign := (raw & 0x8000) != 0
	exp := (raw >> 11) & 0x0F
	mantissa := int16(raw & dpt9MantissaMask) //nolint:gosec // 11-bit value fits in int16

	if sign {
		mantissa |= -0x800 // Sign extend (0xF800 as int16 = -2048)
	}

	value := float64(mantissa) * 0.01 * math.Pow(2, float64(exp))
	return value, nil
}

// EncodeDPT17 encodes a scene number (0-63) to 1-byte format.
//
// Parameters:
//   - scene: Scene number (0-63)
//
// Returns:
//   - []byte: Single byte with scene number
//   - error: If scene is out of range
func EncodeDPT17(scene uint8) ([]byte, error) {
	if scene > dpt17MaxScene {
		return nil, fmt.Errorf("%w: DPT17 scene must be 0-%d, got %d", ErrEncodingFailed, dpt17MaxScene, scene)
	}
	return []byte{scene & dpt17SceneMask}, nil
}

// DecodeDPT17 decodes a scene number from 1-byte format.
//
// Parameters:
//   - data: KNX data (at least 1 byte)
//
// Returns:
//   - uint8: Scene number (0-63)
//   - error: If data is empty
func DecodeDPT17(data []byte) (uint8, error) {
	if len(data) < 1 {
		return 0, fmt.Errorf("%w: DPT17 requires 1 byte, got %d", ErrDecodingFailed, len(data))
	}
	return data[0] & dpt17SceneMask, nil
}

// EncodeDPT18 encodes a scene control value.
//
// Parameters:
//   - scene: Scene number (0-63)
//   - learn: True to learn/save scene, false to recall
//
// Returns:
//   - []byte: Single byte with scene and learn bit
//   - error: If scene is out of range
func EncodeDPT18(scene uint8, learn bool) ([]byte, error) {
	if scene > dpt17MaxScene {
		return nil, fmt.Errorf("%w: DPT18 scene must be 0-%d, got %d", ErrEncodingFailed, dpt17MaxScene, scene)
	}
	value := scene & dpt17SceneMask
	if learn {
		value |= 0x80
	}
	return []byte{value}, nil
}

// DecodeDPT18 decodes a scene control value.
//
// Parameters:
//   - data: KNX data (at least 1 byte)
//
// Returns:
//   - scene: Scene number (0-63)
//   - learn: True if learn/save, false if recall
//   - error: If data is empty
func DecodeDPT18(data []byte) (scene uint8, learn bool, err error) {
	if len(data) < 1 {
		return 0, false, fmt.Errorf("%w: DPT18 requires 1 byte, got %d", ErrDecodingFailed, len(data))
	}
	scene = data[0] & dpt17SceneMask
	learn = (data[0] & 0x80) != 0
	return scene, learn, nil
}

// RGB represents an RGB colour value.
type RGB struct {
	R uint8
	G uint8
	B uint8
}

// EncodeDPT232 encodes an RGB colour to 3-byte format.
//
// Parameters:
//   - rgb: RGB colour value
//
// Returns:
//   - []byte: Three bytes (R, G, B)
func EncodeDPT232(rgb RGB) []byte {
	return []byte{rgb.R, rgb.G, rgb.B}
}

// DecodeDPT232 decodes a 3-byte RGB colour value.
//
// Parameters:
//   - data: KNX data (at least 3 bytes)
//
// Returns:
//   - RGB: Decoded colour
//   - error: If data is too short
func DecodeDPT232(data []byte) (RGB, error) {
	if len(data) < dptRGBBytes {
		return RGB{}, fmt.Errorf("%w: DPT232 requires %d bytes, got %d", ErrDecodingFailed, dptRGBBytes, len(data))
	}
	return RGB{R: data[0], G: data[1], B: data[2]}, nil
}
