package knx

import "testing"

// ─── DPT1 (Boolean — light on/off) ──────────────────────────────────

func BenchmarkEncodeDPT1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		EncodeDPT1(true)
	}
}

func BenchmarkDecodeDPT1(b *testing.B) {
	data := []byte{0x01}
	for i := 0; i < b.N; i++ {
		DecodeDPT1(data) //nolint:errcheck // benchmark
	}
}

// ─── DPT3 (Dimming control) ─────────────────────────────────────────

func BenchmarkEncodeDPT3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		EncodeDPT3(true, 7)
	}
}

func BenchmarkDecodeDPT3(b *testing.B) {
	data := []byte{0x0F}
	for i := 0; i < b.N; i++ {
		DecodeDPT3(data) //nolint:errcheck // benchmark
	}
}

// ─── DPT5 (Percentage — dimmer level) ───────────────────────────────

func BenchmarkEncodeDPT5(b *testing.B) {
	for i := 0; i < b.N; i++ {
		EncodeDPT5(75.0)
	}
}

func BenchmarkDecodeDPT5(b *testing.B) {
	data := []byte{0xBF}
	for i := 0; i < b.N; i++ {
		DecodeDPT5(data) //nolint:errcheck // benchmark
	}
}

// ─── DPT9 (2-byte float — temperature, lux) ────────────────────────

func BenchmarkEncodeDPT9(b *testing.B) {
	for i := 0; i < b.N; i++ {
		EncodeDPT9(21.5) //nolint:errcheck // benchmark
	}
}

func BenchmarkDecodeDPT9(b *testing.B) {
	data := []byte{0x0C, 0x66}
	for i := 0; i < b.N; i++ {
		DecodeDPT9(data) //nolint:errcheck // benchmark
	}
}

// ─── DPT17 (Scene number) ───────────────────────────────────────────

func BenchmarkEncodeDPT17(b *testing.B) {
	for i := 0; i < b.N; i++ {
		EncodeDPT17(5) //nolint:errcheck // benchmark
	}
}

func BenchmarkDecodeDPT17(b *testing.B) {
	data := []byte{0x05}
	for i := 0; i < b.N; i++ {
		DecodeDPT17(data) //nolint:errcheck // benchmark
	}
}

// ─── DPT232 (RGB colour) ────────────────────────────────────────────

func BenchmarkEncodeDPT232(b *testing.B) {
	rgb := RGB{R: 255, G: 128, B: 0}
	for i := 0; i < b.N; i++ {
		EncodeDPT232(rgb)
	}
}

func BenchmarkDecodeDPT232(b *testing.B) {
	data := []byte{0xFF, 0x80, 0x00}
	for i := 0; i < b.N; i++ {
		DecodeDPT232(data) //nolint:errcheck // benchmark
	}
}
