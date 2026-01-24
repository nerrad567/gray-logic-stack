package knx

import (
	"bytes"
	"testing"
)

func TestParseTelegram(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    Telegram
		wantErr bool
	}{
		{
			name: "write 1-bit true to 1/2/3",
			// src=1.1.1(0x1101), GA 1/2/3=0x0A03, TPCI=0x00, APCI write|1=0x81
			data: []byte{0x11, 0x01, 0x0A, 0x03, 0x00, 0x81},
			want: Telegram{
				Destination: GroupAddress{Main: 1, Middle: 2, Sub: 3},
				APCI:        APCIWrite,
				Data:        []byte{0x01},
			},
		},
		{
			name: "write 1-bit false to 1/2/3",
			// src=1.1.1, GA 1/2/3, TPCI=0x00, APCI write|0=0x80
			data: []byte{0x11, 0x01, 0x0A, 0x03, 0x00, 0x80},
			want: Telegram{
				Destination: GroupAddress{Main: 1, Middle: 2, Sub: 3},
				APCI:        APCIWrite,
				Data:        []byte{0x00},
			},
		},
		{
			name: "write percentage (75%) to 2/0/1",
			// src=1.1.2, GA 2/0/1=0x1001, TPCI=0x00, APCI write=0x80, value=0xBF
			data: []byte{0x11, 0x02, 0x10, 0x01, 0x00, 0x80, 0xBF},
			want: Telegram{
				Destination: GroupAddress{Main: 2, Middle: 0, Sub: 1},
				APCI:        APCIWrite,
				Data:        []byte{0xBF},
			},
		},
		{
			name: "read request to 6/0/1",
			// src=0.0.1, GA 6/0/1=0x3001, TPCI=0x00, APCI read=0x00
			data: []byte{0x00, 0x01, 0x30, 0x01, 0x00, 0x00},
			want: Telegram{
				Destination: GroupAddress{Main: 6, Middle: 0, Sub: 1},
				APCI:        APCIRead,
				Data:        nil,
			},
		},
		{
			name: "response 1-bit true from 6/0/1",
			// src=1.1.4, GA 6/0/1=0x3001, TPCI=0x00, APCI response|1=0x41
			data: []byte{0x11, 0x04, 0x30, 0x01, 0x00, 0x41},
			want: Telegram{
				Destination: GroupAddress{Main: 6, Middle: 0, Sub: 1},
				APCI:        APCIResponse,
				Data:        []byte{0x01},
			},
		},
		{
			name: "write 2-byte temperature (21.5°C)",
			// src=1.1.4, GA 5/0/1=0x2801, TPCI=0x00, APCI write=0x80, DPT9 data
			data: []byte{0x11, 0x04, 0x28, 0x01, 0x00, 0x80, 0x0C, 0x66},
			want: Telegram{
				Destination: GroupAddress{Main: 5, Middle: 0, Sub: 1},
				APCI:        APCIWrite,
				Data:        []byte{0x0C, 0x66},
			},
		},
		{
			name: "write RGB colour",
			// src=1.1.5, GA 3/0/5=0x1805, TPCI=0x00, APCI write=0x80, RGB
			data: []byte{0x11, 0x05, 0x18, 0x05, 0x00, 0x80, 0xFF, 0x80, 0x00},
			want: Telegram{
				Destination: GroupAddress{Main: 3, Middle: 0, Sub: 5},
				APCI:        APCIWrite,
				Data:        []byte{0xFF, 0x80, 0x00},
			},
		},
		{
			name:    "too short - only 1 byte",
			data:    []byte{0x0A},
			wantErr: true,
		},
		{
			name:    "too short - only 5 bytes",
			data:    []byte{0x11, 0x01, 0x0A, 0x03, 0x00},
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTelegram(tt.data)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseTelegram() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseTelegram() unexpected error: %v", err)
				return
			}

			if got.Destination != tt.want.Destination {
				t.Errorf("Destination = %v, want %v", got.Destination, tt.want.Destination)
			}
			if got.APCI != tt.want.APCI {
				t.Errorf("APCI = 0x%02X, want 0x%02X", got.APCI, tt.want.APCI)
			}
			if !bytes.Equal(got.Data, tt.want.Data) {
				t.Errorf("Data = %X, want %X", got.Data, tt.want.Data)
			}
		})
	}
}

func TestTelegramEncode(t *testing.T) {
	tests := []struct {
		name     string
		telegram Telegram
		want     []byte
	}{
		{
			name: "write 1-bit true",
			telegram: Telegram{
				Destination: GroupAddress{Main: 1, Middle: 2, Sub: 3},
				APCI:        APCIWrite,
				Data:        []byte{0x01},
			},
			// GA(2) + TPCI(0x00) + APCI|value(0x81) = 4 bytes
			want: []byte{0x0A, 0x03, 0x00, 0x81},
		},
		{
			name: "write 1-bit false",
			telegram: Telegram{
				Destination: GroupAddress{Main: 1, Middle: 2, Sub: 3},
				APCI:        APCIWrite,
				Data:        []byte{0x00},
			},
			want: []byte{0x0A, 0x03, 0x00, 0x80},
		},
		{
			name: "read request",
			telegram: Telegram{
				Destination: GroupAddress{Main: 6, Middle: 0, Sub: 1},
				APCI:        APCIRead,
				Data:        nil,
			},
			want: []byte{0x30, 0x01, 0x00, 0x00},
		},
		{
			name: "write percentage",
			telegram: Telegram{
				Destination: GroupAddress{Main: 2, Middle: 0, Sub: 1},
				APCI:        APCIWrite,
				Data:        []byte{0xBF},
			},
			// Value > 0x3F so goes in long format: GA(2) + TPCI(0x00) + APCI(0x80) + data(0xBF)
			want: []byte{0x10, 0x01, 0x00, 0x80, 0xBF},
		},
		{
			name: "write 2-byte temperature",
			telegram: Telegram{
				Destination: GroupAddress{Main: 5, Middle: 0, Sub: 1},
				APCI:        APCIWrite,
				Data:        []byte{0x0C, 0x66},
			},
			// Long format: GA(2) + TPCI(0x00) + APCI(0x80) + data(0x0C, 0x66)
			want: []byte{0x28, 0x01, 0x00, 0x80, 0x0C, 0x66},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.telegram.Encode()
			if !bytes.Equal(got, tt.want) {
				t.Errorf("Encode() = %X, want %X", got, tt.want)
			}
		})
	}
}

func TestTelegramRoundTrip(t *testing.T) {
	// GROUPCON format is asymmetric:
	//   Encode() produces send format: GA(2) + TPCI + APCI|data
	//   ParseTelegram() expects receive format: src(2) + GA(2) + TPCI + APCI|data
	// To test round-trip, prepend a dummy source address to the encoded output.
	tests := []struct {
		name     string
		telegram Telegram
	}{
		{
			name: "1-bit write",
			telegram: Telegram{
				Destination: GroupAddress{Main: 1, Middle: 2, Sub: 3},
				APCI:        APCIWrite,
				Data:        []byte{0x01},
			},
		},
		{
			name: "read request",
			telegram: Telegram{
				Destination: GroupAddress{Main: 6, Middle: 0, Sub: 1},
				APCI:        APCIRead,
				Data:        nil,
			},
		},
		{
			name: "2-byte write",
			telegram: Telegram{
				Destination: GroupAddress{Main: 5, Middle: 0, Sub: 1},
				APCI:        APCIWrite,
				Data:        []byte{0x0C, 0x66},
			},
		},
		{
			name: "3-byte RGB write",
			telegram: Telegram{
				Destination: GroupAddress{Main: 3, Middle: 0, Sub: 5},
				APCI:        APCIWrite,
				Data:        []byte{0xFF, 0x80, 0x00},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := tt.telegram.Encode()
			// Prepend a dummy source address (1.1.1 = 0x1101) to simulate
			// what knxd sends back in GROUPCON receive format.
			withSrc := append([]byte{0x11, 0x01}, encoded...)
			parsed, err := ParseTelegram(withSrc)
			if err != nil {
				t.Fatalf("ParseTelegram() error: %v", err)
			}

			if parsed.Destination != tt.telegram.Destination {
				t.Errorf("Destination = %v, want %v", parsed.Destination, tt.telegram.Destination)
			}
			if parsed.APCI != tt.telegram.APCI {
				t.Errorf("APCI = 0x%02X, want 0x%02X", parsed.APCI, tt.telegram.APCI)
			}
			if !bytes.Equal(parsed.Data, tt.telegram.Data) {
				t.Errorf("Data = %X, want %X", parsed.Data, tt.telegram.Data)
			}
		})
	}
}

func TestTelegramHelpers(t *testing.T) {
	t.Run("IsWrite", func(t *testing.T) {
		tg := Telegram{APCI: APCIWrite}
		if !tg.IsWrite() {
			t.Error("IsWrite() = false, want true")
		}
		tg.APCI = APCIRead
		if tg.IsWrite() {
			t.Error("IsWrite() = true, want false")
		}
	})

	t.Run("IsRead", func(t *testing.T) {
		tg := Telegram{APCI: APCIRead}
		if !tg.IsRead() {
			t.Error("IsRead() = false, want true")
		}
		tg.APCI = APCIWrite
		if tg.IsRead() {
			t.Error("IsRead() = true, want false")
		}
	})

	t.Run("IsResponse", func(t *testing.T) {
		tg := Telegram{APCI: APCIResponse}
		if !tg.IsResponse() {
			t.Error("IsResponse() = false, want true")
		}
		tg.APCI = APCIWrite
		if tg.IsResponse() {
			t.Error("IsResponse() = true, want false")
		}
	})

	t.Run("String", func(t *testing.T) {
		tg := Telegram{
			Destination: GroupAddress{Main: 1, Middle: 2, Sub: 3},
			APCI:        APCIWrite,
			Data:        []byte{0x01},
		}
		s := tg.String()
		if s == "" {
			t.Error("String() returned empty string")
		}
		// Should contain GA and APCI type
		if !bytes.Contains([]byte(s), []byte("1/2/3")) {
			t.Errorf("String() = %q, should contain GA", s)
		}
		if !bytes.Contains([]byte(s), []byte("WRITE")) {
			t.Errorf("String() = %q, should contain WRITE", s)
		}
	})
}

func TestNewWriteTelegram(t *testing.T) {
	dest := GroupAddress{Main: 1, Middle: 2, Sub: 3}
	data := []byte{0x01}

	tg := NewWriteTelegram(dest, data)

	if tg.Destination != dest {
		t.Errorf("Destination = %v, want %v", tg.Destination, dest)
	}
	if tg.APCI != APCIWrite {
		t.Errorf("APCI = 0x%02X, want 0x%02X", tg.APCI, APCIWrite)
	}
	if !bytes.Equal(tg.Data, data) {
		t.Errorf("Data = %X, want %X", tg.Data, data)
	}
	if tg.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestNewReadTelegram(t *testing.T) {
	dest := GroupAddress{Main: 6, Middle: 0, Sub: 1}

	tg := NewReadTelegram(dest)

	if tg.Destination != dest {
		t.Errorf("Destination = %v, want %v", tg.Destination, dest)
	}
	if tg.APCI != APCIRead {
		t.Errorf("APCI = 0x%02X, want 0x%02X", tg.APCI, APCIRead)
	}
	if tg.Data != nil {
		t.Errorf("Data = %X, want nil", tg.Data)
	}
	if tg.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestEncodeKNXDMessage(t *testing.T) {
	tests := []struct {
		name    string
		msgType uint16
		payload []byte
		want    []byte
	}{
		{
			name:    "open GROUPCON (with payload)",
			msgType: EIBOpenGroupCon,
			payload: []byte{0x00, 0x00, 0x00},                         // reserved, write_only=0, reserved
			want:    []byte{0x00, 0x05, 0x00, 0x26, 0x00, 0x00, 0x00}, // size=5 (type+payload)
		},
		{
			name:    "group packet with telegram",
			msgType: EIBGroupPacket,
			payload: []byte{0x0A, 0x03, 0x81},                         // GA 1/2/3, write true
			want:    []byte{0x00, 0x05, 0x00, 0x27, 0x0A, 0x03, 0x81}, // size=5 (type+payload)
		},
		{
			name:    "close connection",
			msgType: EIBClose,
			payload: nil,
			want:    []byte{0x00, 0x02, 0x00, 0x06}, // size=2 (type only, no payload)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeKNXDMessage(tt.msgType, tt.payload)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("EncodeKNXDMessage() = %X, want %X", got, tt.want)
			}
		})
	}
}

func TestParseKNXDMessage(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		wantType    uint16
		wantPayload []byte
		wantErr     bool
	}{
		{
			name:        "open GROUPCON response",
			data:        []byte{0x00, 0x02, 0x00, 0x26}, // size=2 (type only)
			wantType:    EIBOpenGroupCon,
			wantPayload: nil,
		},
		{
			name:        "group packet with telegram",
			data:        []byte{0x00, 0x05, 0x00, 0x27, 0x0A, 0x03, 0x81}, // size=5 (type+payload)
			wantType:    EIBGroupPacket,
			wantPayload: []byte{0x0A, 0x03, 0x81},
		},
		{
			name:    "too short",
			data:    []byte{0x00, 0x02, 0x00},
			wantErr: true,
		},
		{
			name:    "size mismatch",
			data:    []byte{0x00, 0x10, 0x00, 0x27, 0x0A}, // declared 16, expected 3 (5-2)
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotPayload, err := ParseKNXDMessage(tt.data)

			if tt.wantErr {
				if err == nil {
					t.Error("ParseKNXDMessage() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseKNXDMessage() unexpected error: %v", err)
				return
			}

			if gotType != tt.wantType {
				t.Errorf("msgType = 0x%04X, want 0x%04X", gotType, tt.wantType)
			}
			if !bytes.Equal(gotPayload, tt.wantPayload) {
				t.Errorf("payload = %X, want %X", gotPayload, tt.wantPayload)
			}
		})
	}
}

func TestKNXDMessageRoundTrip(t *testing.T) {
	// Test that encode → parse gives back the same message
	tests := []struct {
		name    string
		msgType uint16
		payload []byte
	}{
		{
			name:    "open GROUPCON",
			msgType: EIBOpenGroupCon,
			payload: []byte{0x00, 0x00, 0x00},
		},
		{
			name:    "group packet",
			msgType: EIBGroupPacket,
			payload: []byte{0x0A, 0x03, 0x81, 0xBF},
		},
		{
			name:    "close",
			msgType: EIBClose,
			payload: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeKNXDMessage(tt.msgType, tt.payload)
			gotType, gotPayload, err := ParseKNXDMessage(encoded)
			if err != nil {
				t.Fatalf("ParseKNXDMessage() error: %v", err)
			}

			if gotType != tt.msgType {
				t.Errorf("msgType = 0x%04X, want 0x%04X", gotType, tt.msgType)
			}
			if !bytes.Equal(gotPayload, tt.payload) {
				t.Errorf("payload = %X, want %X", gotPayload, tt.payload)
			}
		})
	}
}
