package knx

import "testing"

func BenchmarkParseTelegram_1Bit(b *testing.B) {
	// Write true to 1/2/3 — most common telegram type
	data := []byte{0x11, 0x01, 0x0A, 0x03, 0x00, 0x81}
	for i := 0; i < b.N; i++ {
		ParseTelegram(data) //nolint:errcheck // benchmark
	}
}

func BenchmarkParseTelegram_2Byte(b *testing.B) {
	// Write 2-byte temperature to 5/0/1
	data := []byte{0x11, 0x04, 0x28, 0x01, 0x00, 0x80, 0x0C, 0x66}
	for i := 0; i < b.N; i++ {
		ParseTelegram(data) //nolint:errcheck // benchmark
	}
}

func BenchmarkParseTelegram_RGB(b *testing.B) {
	// Write 3-byte RGB to 3/0/5
	data := []byte{0x11, 0x05, 0x18, 0x05, 0x00, 0x80, 0xFF, 0x80, 0x00}
	for i := 0; i < b.N; i++ {
		ParseTelegram(data) //nolint:errcheck // benchmark
	}
}

func BenchmarkTelegramEncode(b *testing.B) {
	tg := Telegram{
		Destination: GroupAddress{Main: 1, Middle: 2, Sub: 3},
		APCI:        APCIWrite,
		Data:        []byte{0x01},
	}
	for i := 0; i < b.N; i++ {
		tg.Encode()
	}
}
