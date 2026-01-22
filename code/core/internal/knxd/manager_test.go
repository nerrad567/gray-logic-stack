package knxd

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestNewManager_Defaults(t *testing.T) {
	cfg := Config{
		Managed: true,
		Backend: BackendConfig{Type: BackendUSB},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Verify defaults are applied
	if m.config.Binary != "/usr/bin/knxd" {
		t.Errorf("Binary = %q, want %q", m.config.Binary, "/usr/bin/knxd")
	}
	if m.config.PhysicalAddress != "0.0.1" {
		t.Errorf("PhysicalAddress = %q, want %q", m.config.PhysicalAddress, "0.0.1")
	}
	if m.config.ClientAddresses != "0.0.2:8" {
		t.Errorf("ClientAddresses = %q, want %q", m.config.ClientAddresses, "0.0.2:8")
	}
	if m.config.TCPPort != 6720 {
		t.Errorf("TCPPort = %d, want %d", m.config.TCPPort, 6720)
	}
	if m.config.RestartDelay != 5*time.Second {
		t.Errorf("RestartDelay = %v, want %v", m.config.RestartDelay, 5*time.Second)
	}
	if m.config.MaxRestartAttempts != 10 {
		t.Errorf("MaxRestartAttempts = %d, want %d", m.config.MaxRestartAttempts, 10)
	}
	if m.config.HealthCheckInterval != 30*time.Second {
		t.Errorf("HealthCheckInterval = %v, want %v", m.config.HealthCheckInterval, 30*time.Second)
	}
}

func TestNewManager_CustomConfig(t *testing.T) {
	cfg := Config{
		Managed:            true,
		Binary:             "/opt/knxd/bin/knxd",
		PhysicalAddress:    "1.1.1",
		ClientAddresses:    "1.1.2:4",
		TCPPort:            7720,
		RestartDelay:       10 * time.Second,
		MaxRestartAttempts: 5,
		Backend:            BackendConfig{Type: BackendUSB},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	if m.config.Binary != "/opt/knxd/bin/knxd" {
		t.Errorf("Binary = %q, want %q", m.config.Binary, "/opt/knxd/bin/knxd")
	}
	if m.config.PhysicalAddress != "1.1.1" {
		t.Errorf("PhysicalAddress = %q, want %q", m.config.PhysicalAddress, "1.1.1")
	}
	if m.config.TCPPort != 7720 {
		t.Errorf("TCPPort = %d, want %d", m.config.TCPPort, 7720)
	}
}

func TestNewManager_InvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "invalid physical address format",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "invalid",
				Backend:         BackendConfig{Type: BackendUSB},
			},
		},
		{
			name: "physical address area out of range",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "16.0.1",
				Backend:         BackendConfig{Type: BackendUSB},
			},
		},
		{
			name: "invalid client addresses",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "bad",
				Backend:         BackendConfig{Type: BackendUSB},
			},
		},
		{
			name: "TCP port out of range",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "0.0.2:8",
				TCPPort:         99999,
				Backend:         BackendConfig{Type: BackendUSB},
			},
		},
		{
			name: "log level out of range",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "0.0.2:8",
				TCPPort:         6720,
				LogLevel:        10,
				Backend:         BackendConfig{Type: BackendUSB},
			},
		},
		{
			name: "unknown backend type",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "0.0.2:8",
				TCPPort:         6720,
				Backend:         BackendConfig{Type: "unknown"},
			},
		},
		{
			name: "ipt backend missing host",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "0.0.2:8",
				TCPPort:         6720,
				Backend:         BackendConfig{Type: BackendIPTunnel},
			},
		},
		{
			name: "usb reset without IDs",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "0.0.2:8",
				TCPPort:         6720,
				Backend: BackendConfig{
					Type:            BackendUSB,
					USBResetOnRetry: true,
				},
			},
		},
		{
			name: "invalid USB vendor ID",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "0.0.2:8",
				TCPPort:         6720,
				Backend: BackendConfig{
					Type:        BackendUSB,
					USBVendorID: "ZZZZ",
				},
			},
		},
		{
			name: "invalid health check device address",
			cfg: Config{
				Managed:                  true,
				PhysicalAddress:          "0.0.1",
				ClientAddresses:          "0.0.2:8",
				TCPPort:                  6720,
				Backend:                  BackendConfig{Type: BackendUSB},
				HealthCheckDeviceAddress: "999/999/999",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewManager(tt.cfg)
			if err == nil {
				t.Error("NewManager() expected error, got nil")
			}
		})
	}
}

func TestConnectionURL(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "TCP enabled",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "0.0.2:8",
				ListenTCP:       true,
				TCPPort:         6720,
				Backend:         BackendConfig{Type: BackendUSB},
			},
			want: "tcp://localhost:6720",
		},
		{
			name: "custom TCP port",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "0.0.2:8",
				ListenTCP:       true,
				TCPPort:         7720,
				Backend:         BackendConfig{Type: BackendUSB},
			},
			want: "tcp://localhost:7720",
		},
		{
			name: "unix socket only",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "0.0.2:8",
				ListenTCP:       false,
				UnixSocket:      "/tmp/knxd.sock",
				Backend:         BackendConfig{Type: BackendUSB},
			},
			want: "unix:///tmp/knxd.sock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewManager(tt.cfg)
			if err != nil {
				t.Fatalf("NewManager() error: %v", err)
			}
			if got := m.ConnectionURL(); got != tt.want {
				t.Errorf("ConnectionURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsManaged(t *testing.T) {
	cfg := Config{
		Managed: true,
		Backend: BackendConfig{Type: BackendUSB},
	}
	m, _ := NewManager(cfg)
	if !m.IsManaged() {
		t.Error("IsManaged() = false, want true")
	}
}

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		contains []string
	}{
		{
			name: "USB backend defaults",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "0.0.2:8",
				ListenTCP:       true,
				TCPPort:         6720,
				Backend:         BackendConfig{Type: BackendUSB},
			},
			contains: []string{"-e", "0.0.1", "-E", "0.0.2:8", "-i6720", "-b", "usb:"},
		},
		{
			name: "IPT backend with host",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "0.0.2:8",
				ListenTCP:       true,
				TCPPort:         6720,
				Backend: BackendConfig{
					Type: BackendIPTunnel,
					Host: "192.168.1.100",
					Port: 3671,
				},
			},
			contains: []string{"-b", "ipt:192.168.1.100:3671"},
		},
		{
			name: "IP routing backend",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "0.0.2:8",
				ListenTCP:       true,
				TCPPort:         6720,
				Backend: BackendConfig{
					Type:             BackendIPRouting,
					MulticastAddress: "224.0.23.12",
				},
			},
			contains: []string{"-b", "ip:224.0.23.12"},
		},
		{
			name: "with log level and trace flags",
			cfg: Config{
				Managed:         true,
				PhysicalAddress: "0.0.1",
				ClientAddresses: "0.0.2:8",
				ListenTCP:       true,
				TCPPort:         6720,
				LogLevel:        5,
				TraceFlags:      3,
				Backend:         BackendConfig{Type: BackendUSB},
			},
			contains: []string{"-f5", "-t3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.cfg.BuildArgs()
			for _, want := range tt.contains {
				found := false
				for _, arg := range args {
					if arg == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("BuildArgs() missing %q, got %v", want, args)
				}
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Managed {
		t.Error("Managed = false, want true")
	}
	if cfg.Backend.Type != BackendUSB {
		t.Errorf("Backend.Type = %q, want %q", cfg.Backend.Type, BackendUSB)
	}
	if cfg.TCPPort != 6720 {
		t.Errorf("TCPPort = %d, want 6720", cfg.TCPPort)
	}
	if cfg.PhysicalAddress != "0.0.1" {
		t.Errorf("PhysicalAddress = %q, want %q", cfg.PhysicalAddress, "0.0.1")
	}

	// Default config should validate cleanly
	if err := cfg.Validate(); err != nil {
		t.Errorf("DefaultConfig().Validate() error: %v", err)
	}
}

func TestParseGroupAddress(t *testing.T) {
	tests := []struct {
		addr    string
		want    uint16
		wantErr bool
	}{
		{"0/0/0", 0x0000, false},
		{"1/0/0", 0x0800, false},
		{"1/7/0", 0x0F00, false},
		{"1/2/3", 0x0A03, false},
		{"31/7/255", 0xFFFF, false},
		{"", 0, true},
		{"1/2", 0, true},
		{"32/0/0", 0, true},  // main > 31
		{"0/8/0", 0, true},   // middle > 7
		{"0/0/256", 0, true}, // sub > 255
		{"a/b/c", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got, err := ParseGroupAddress(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGroupAddress(%q) error = %v, wantErr %v", tt.addr, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseGroupAddress(%q) = 0x%04X, want 0x%04X", tt.addr, got, tt.want)
			}
		})
	}
}

func TestFormatGroupAddress(t *testing.T) {
	tests := []struct {
		ga   uint16
		want string
	}{
		{0x0000, "0/0/0"},
		{0x0800, "1/0/0"},
		{0x0A03, "1/2/3"},
		{0xFFFF, "31/7/255"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := FormatGroupAddress(tt.ga); got != tt.want {
				t.Errorf("FormatGroupAddress(0x%04X) = %q, want %q", tt.ga, got, tt.want)
			}
		})
	}
}

func TestParseIndividualAddress(t *testing.T) {
	tests := []struct {
		addr    string
		want    uint16
		wantErr bool
	}{
		{"0.0.0", 0x0000, false},
		{"0.0.1", 0x0001, false},
		{"1.1.1", 0x1101, false},
		{"15.15.255", 0xFFFF, false},
		{"", 0, true},
		{"1.1", 0, true},
		{"16.0.0", 0, true},  // area > 15
		{"0.16.0", 0, true},  // line > 15
		{"0.0.256", 0, true}, // device > 255
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got, err := ParseIndividualAddress(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIndividualAddress(%q) error = %v, wantErr %v", tt.addr, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseIndividualAddress(%q) = 0x%04X, want 0x%04X", tt.addr, got, tt.want)
			}
		})
	}
}

func TestFormatIndividualAddress(t *testing.T) {
	tests := []struct {
		ia   uint16
		want string
	}{
		{0x0000, "0.0.0"},
		{0x0001, "0.0.1"},
		{0x1101, "1.1.1"},
		{0xFFFF, "15.15.255"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := FormatIndividualAddress(tt.ia); got != tt.want {
				t.Errorf("FormatIndividualAddress(0x%04X) = %q, want %q", tt.ia, got, tt.want)
			}
		})
	}
}

func TestHealthError(t *testing.T) {
	t.Run("recoverable error", func(t *testing.T) {
		err := newHealthError(3, true, fmt.Errorf("bus timeout"))
		if !err.IsRecoverable() {
			t.Error("IsRecoverable() = false, want true")
		}
		if err.Layer != 3 {
			t.Errorf("Layer = %d, want 3", err.Layer)
		}
		if err.Error() == "" {
			t.Error("Error() should not be empty")
		}
	})

	t.Run("non-recoverable error", func(t *testing.T) {
		err := newHealthError(0, false, fmt.Errorf("USB device missing"))
		if err.IsRecoverable() {
			t.Error("IsRecoverable() = true, want false")
		}
		if err.Layer != 0 {
			t.Errorf("Layer = %d, want 0", err.Layer)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		inner := fmt.Errorf("inner error")
		err := newHealthError(1, true, inner)
		if !errors.Is(err, inner) {
			t.Error("errors.Is() did not match inner error")
		}
	})
}

func TestStats_Unmanaged(t *testing.T) {
	cfg := Config{
		Managed: false,
		Backend: BackendConfig{Type: BackendUSB},
	}

	// Need to set minimum valid fields for NewManager
	cfg.PhysicalAddress = "0.0.1"
	cfg.ClientAddresses = "0.0.2:8"
	cfg.TCPPort = 6720

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	stats := m.Stats()
	if stats.Status != "external" {
		t.Errorf("Status = %q, want %q", stats.Status, "external")
	}
	if stats.Managed {
		t.Error("Stats.Managed = true, want false (config.Managed is false)")
	}
}

func TestValidateClientAddresses(t *testing.T) {
	tests := []struct {
		addr    string
		wantErr bool
	}{
		{"0.0.2:8", false},
		{"1.1.10:4", false},
		{"0.0.1:255", false},
		{"0.0.1:0", true},   // count < 1
		{"0.0.1:256", true}, // count > 255
		{"bad:8", true},
		{"0.0.1", true}, // missing count
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			err := validateClientAddresses(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateClientAddresses(%q) error = %v, wantErr %v", tt.addr, err, tt.wantErr)
			}
		})
	}
}
