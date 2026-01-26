package knxd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
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

// ─── Integration Tests (require KNXSim running) ─────────────

// knxSimHost returns the host where KNXSim is reachable, or empty if not available.
// Tries multiple addresses: Docker hostname, container IP, localhost.
func knxSimHost() string {
	hosts := []string{
		"knxsim",     // Docker network hostname
		"172.21.0.2", // Docker container IP (may vary)
		"127.0.0.1",  // localhost if port is exposed
	}

	for _, host := range hosts {
		// UDP "connect" doesn't actually send packets, just sets up the socket
		// We need to try a real connection or check if something is listening
		addr := fmt.Sprintf("%s:3671", host)
		conn, err := net.DialTimeout("udp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			// For UDP we can't really tell if something is listening
			// Try TCP on the web UI port (9090) as a proxy check
			if host == "127.0.0.1" {
				// Check if web UI is accessible
				tcpConn, tcpErr := net.DialTimeout("tcp", "127.0.0.1:9090", 100*time.Millisecond)
				if tcpErr == nil {
					tcpConn.Close()
					return host
				}
			} else {
				return host
			}
		}
	}
	return ""
}

// skipIfNoKNXSim skips the test if KNXSim is not available
func skipIfNoKNXSim(t *testing.T) string {
	t.Helper()
	host := knxSimHost()
	if host == "" {
		t.Skip("KNXSim not available (tried knxsim:3671, 172.21.0.2:3671, 127.0.0.1:3671)")
	}
	return host
}

func TestManager_StartStop_Integration(t *testing.T) {
	knxSimAddr := skipIfNoKNXSim(t)

	// Use a unique port to avoid conflicts with any running knxd
	port := 16720

	cfg := Config{
		Managed:         true,
		Binary:          "/usr/bin/knxd",
		PhysicalAddress: "0.0.50",
		ClientAddresses: "0.0.51:4",
		ListenTCP:       true,
		TCPPort:         port,
		Backend: BackendConfig{
			Type: BackendIPTunnel,
			Host: knxSimAddr,
			Port: 3671,
		},
		RestartDelay:       1 * time.Second,
		MaxRestartAttempts: 3,
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start the manager
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Give it time to connect
	time.Sleep(500 * time.Millisecond)

	// Verify it's running
	if !m.IsRunning() {
		t.Error("IsRunning() = false after Start()")
	}

	// Check stats
	stats := m.Stats()
	if stats.Status != "running" {
		t.Errorf("Stats.Status = %q, want running", stats.Status)
	}

	// Stop the manager
	if err := m.Stop(); err != nil {
		t.Errorf("Stop() error: %v", err)
	}

	// Verify it stopped
	if m.IsRunning() {
		t.Error("IsRunning() = true after Stop()")
	}
}

func TestManager_HealthCheck_Integration(t *testing.T) {
	knxSimAddr := skipIfNoKNXSim(t)

	port := 16721

	cfg := Config{
		Managed:         true,
		Binary:          "/usr/bin/knxd",
		PhysicalAddress: "0.0.60",
		ClientAddresses: "0.0.61:4",
		ListenTCP:       true,
		TCPPort:         port,
		Backend: BackendConfig{
			Type: BackendIPTunnel,
			Host: knxSimAddr,
			Port: 3671,
		},
		HealthCheckInterval: 1 * time.Second,
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer m.Stop()

	// Wait for knxd to fully connect
	time.Sleep(1 * time.Second)

	// Run health check
	hctx, hcancel := context.WithTimeout(ctx, 5*time.Second)
	defer hcancel()

	err = m.HealthCheck(hctx)
	if err != nil {
		t.Errorf("HealthCheck() error: %v", err)
	}
}

func TestManager_Stats_Integration(t *testing.T) {
	knxSimAddr := skipIfNoKNXSim(t)

	port := 16722

	cfg := Config{
		Managed:         true,
		Binary:          "/usr/bin/knxd",
		PhysicalAddress: "0.0.70",
		ClientAddresses: "0.0.71:4",
		ListenTCP:       true,
		TCPPort:         port,
		Backend: BackendConfig{
			Type: BackendIPTunnel,
			Host: knxSimAddr,
			Port: 3671,
		},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stats before start
	stats := m.Stats()
	if stats.Status == "running" {
		t.Error("Stats.Status = running before Start()")
	}
	if stats.PID != 0 {
		t.Errorf("Stats.PID = %d before Start(), want 0", stats.PID)
	}

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer m.Stop()

	time.Sleep(500 * time.Millisecond)

	// Stats after start
	stats = m.Stats()
	if stats.Status != "running" {
		t.Errorf("Stats.Status = %q after Start(), want running", stats.Status)
	}
	if stats.PID == 0 {
		t.Error("Stats.PID = 0 after Start()")
	}
	if !stats.Managed {
		t.Error("Stats.Managed = false")
	}
}

func TestManager_SetLogger_Integration(t *testing.T) {
	cfg := Config{
		Managed: true,
		Backend: BackendConfig{Type: BackendUSB},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create a simple logger
	testLogger := &testKnxdLogger{
		onInfo: func(msg string, args ...any) {},
	}

	m.SetLogger(testLogger)

	// Verify SetLogger doesn't panic and logger is set
	// (we can't access m.logger directly as it's private, but test passes if no panic)
}

type testKnxdLogger struct {
	onDebug func(string, ...any)
	onInfo  func(string, ...any)
	onWarn  func(string, ...any)
	onError func(string, ...any)
}

func (l *testKnxdLogger) Debug(msg string, args ...any) {
	if l.onDebug != nil {
		l.onDebug(msg, args...)
	}
}
func (l *testKnxdLogger) Info(msg string, args ...any) {
	if l.onInfo != nil {
		l.onInfo(msg, args...)
	}
}
func (l *testKnxdLogger) Warn(msg string, args ...any) {
	if l.onWarn != nil {
		l.onWarn(msg, args...)
	}
}
func (l *testKnxdLogger) Error(msg string, args ...any) {
	if l.onError != nil {
		l.onError(msg, args...)
	}
}

// ─── Additional Unit Tests for Coverage ─────────────────────────────────────

func TestNoopLogger(t *testing.T) {
	// Test that noopLogger methods don't panic
	logger := noopLogger{}
	logger.Debug("test message", "key", "value")
	logger.Info("test message", "key", "value")
	logger.Warn("test message", "key", "value")
	logger.Error("test message", "key", "value")
	// If we get here without panic, the test passes
}

// mockGroupAddressProvider implements GroupAddressProvider for testing
type mockGroupAddressProvider struct {
	addresses      []string
	lastMarkedUsed string
}

func (m *mockGroupAddressProvider) GetHealthCheckGroupAddresses(ctx context.Context, limit int) ([]string, error) {
	if limit < len(m.addresses) {
		return m.addresses[:limit], nil
	}
	return m.addresses, nil
}

func (m *mockGroupAddressProvider) MarkHealthCheckUsed(ctx context.Context, ga string) error {
	m.lastMarkedUsed = ga
	return nil
}

func TestManager_SetGroupAddressProvider(t *testing.T) {
	cfg := Config{
		Managed: true,
		Backend: BackendConfig{Type: BackendUSB},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	provider := &mockGroupAddressProvider{addresses: []string{"1/2/3", "1/2/4"}}
	m.SetGroupAddressProvider(provider)

	if m.groupAddressProvider == nil {
		t.Error("groupAddressProvider is nil after SetGroupAddressProvider()")
	}
}

func TestValidateSafePathComponent(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"validname", false},
		{"valid-name", false},
		{"valid_name", false},
		{"valid/path", false},  // slashes allowed
		{"valid:colon", false}, // colons allowed
		{"123", false},
		{"", true},                     // empty
		{"valid.name", true},           // dots NOT allowed (prevents path traversal)
		{"../etc/passwd", true},        // path traversal (contains dot)
		{"name with spaces", true},     // spaces
		{"name\nwith\nnewlines", true}, // newlines
		{"cmd;injection", true},        // shell metacharacter
		{"cmd|pipe", true},             // shell metacharacter
		{"$var", true},                 // shell metacharacter
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := validateSafePathComponent(tt.input, "testField")
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSafePathComponent(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// timeoutError is a net.Error that reports Timeout() = true
type timeoutError struct{}

func (e timeoutError) Error() string   { return "timeout" }
func (e timeoutError) Timeout() bool   { return true }
func (e timeoutError) Temporary() bool { return true }

// nonTimeoutNetError is a net.Error that reports Timeout() = false
type nonTimeoutNetError struct{}

func (e nonTimeoutNetError) Error() string   { return "non-timeout net error" }
func (e nonTimeoutNetError) Timeout() bool   { return false }
func (e nonTimeoutNetError) Temporary() bool { return false }

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"regular error", errors.New("some error"), false},
		{"net.Error with Timeout()=true", timeoutError{}, true},
		{"net.Error with Timeout()=false", nonTimeoutNetError{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTimeoutError(tt.err)
			if got != tt.want {
				t.Errorf("isTimeoutError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── USB Hardware Tests (require Weinzierl USB interface) ───────────────────

// usbDevicePresent checks if the Weinzierl KNX-USB interface is connected
func usbDevicePresent() bool {
	cmd := exec.Command("lsusb", "-d", "0e77:0104")
	output, err := cmd.CombinedOutput()
	return err == nil && len(output) > 0
}

func skipIfNoUSB(t *testing.T) {
	t.Helper()
	if !usbDevicePresent() {
		t.Skip("Weinzierl KNX-USB interface not detected (0e77:0104)")
	}
}

func TestManager_USBDeviceCheck_Integration(t *testing.T) {
	skipIfNoUSB(t)

	cfg := Config{
		Managed:         true,
		Binary:          "/usr/bin/knxd",
		PhysicalAddress: "0.0.80",
		ClientAddresses: "0.0.81:4",
		Backend: BackendConfig{
			Type:         BackendUSB,
			USBVendorID:  "0e77",
			USBProductID: "0104",
		},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test checkUSBDevicePresent directly
	err = m.checkUSBDevicePresent(ctx)
	if err != nil {
		t.Errorf("checkUSBDevicePresent() error: %v", err)
	}
}

func TestManager_USBDeviceCheck_NotPresent(t *testing.T) {
	// Test with a non-existent USB device
	cfg := Config{
		Managed:         true,
		Binary:          "/usr/bin/knxd",
		PhysicalAddress: "0.0.80",
		ClientAddresses: "0.0.81:4",
		Backend: BackendConfig{
			Type:         BackendUSB,
			USBVendorID:  "dead", // Non-existent vendor
			USBProductID: "beef", // Non-existent product
		},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = m.checkUSBDevicePresent(ctx)
	if err == nil {
		t.Error("checkUSBDevicePresent() should fail for non-existent device")
	}
}

func TestManager_USBDeviceCheck_NoIDConfigured(t *testing.T) {
	// When USB IDs aren't configured, check should be skipped
	cfg := Config{
		Managed:         true,
		Binary:          "/usr/bin/knxd",
		PhysicalAddress: "0.0.80",
		ClientAddresses: "0.0.81:4",
		Backend: BackendConfig{
			Type: BackendUSB,
			// No USBVendorID or USBProductID
		},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should return nil (skip check) when IDs not configured
	err = m.checkUSBDevicePresent(ctx)
	if err != nil {
		t.Errorf("checkUSBDevicePresent() should skip when IDs not configured, got: %v", err)
	}
}

// ─── USB Reset Tests ────────────────────────────────────────────────────────

func TestManager_ResetUSBDevice_Integration(t *testing.T) {
	skipIfNoUSB(t)

	cfg := Config{
		Managed:         true,
		Binary:          "/usr/bin/knxd",
		PhysicalAddress: "0.0.80",
		ClientAddresses: "0.0.81:4",
		Backend: BackendConfig{
			Type:         BackendUSB,
			USBVendorID:  "0e77",
			USBProductID: "0104",
		},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Test the public ResetUSBDevice method
	err = m.ResetUSBDevice()
	if err != nil {
		t.Errorf("ResetUSBDevice() error: %v", err)
	}

	// Give the device time to reinitialise
	time.Sleep(1 * time.Second)

	// Verify device is still present after reset
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = m.checkUSBDevicePresent(ctx)
	if err != nil {
		t.Errorf("USB device not present after reset: %v", err)
	}
}

func TestManager_ResetUSBDeviceWithContext_Integration(t *testing.T) {
	skipIfNoUSB(t)

	cfg := Config{
		Managed:         true,
		Binary:          "/usr/bin/knxd",
		PhysicalAddress: "0.0.80",
		ClientAddresses: "0.0.81:4",
		Backend: BackendConfig{
			Type:         BackendUSB,
			USBVendorID:  "0e77",
			USBProductID: "0104",
		},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Test resetUSBDeviceWithContext directly
	err = m.resetUSBDeviceWithContext(ctx)
	if err != nil {
		t.Errorf("resetUSBDeviceWithContext() error: %v", err)
	}
}

func TestManager_ResetUSBDevice_NonUSBBackend(t *testing.T) {
	// Reset should be no-op for non-USB backends
	cfg := Config{
		Managed:         true,
		Binary:          "/usr/bin/knxd",
		PhysicalAddress: "0.0.80",
		ClientAddresses: "0.0.81:4",
		Backend: BackendConfig{
			Type: BackendIPTunnel,
			Host: "192.168.1.100",
		},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Should return nil (no-op) for non-USB backend
	err = m.ResetUSBDevice()
	if err != nil {
		t.Errorf("ResetUSBDevice() should be no-op for IPT backend, got: %v", err)
	}
}

func TestManager_ResetUSBDevice_NoIDConfigured(t *testing.T) {
	// Reset should be skipped when USB IDs aren't configured
	cfg := Config{
		Managed:         true,
		Binary:          "/usr/bin/knxd",
		PhysicalAddress: "0.0.80",
		ClientAddresses: "0.0.81:4",
		Backend: BackendConfig{
			Type: BackendUSB,
			// No USBVendorID or USBProductID
		},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Should return nil (skip) when IDs not configured
	err = m.ResetUSBDevice()
	if err != nil {
		t.Errorf("ResetUSBDevice() should skip when IDs not configured, got: %v", err)
	}
}

func TestManager_ResetUSBDevice_ContextCancelled(t *testing.T) {
	skipIfNoUSB(t)

	cfg := Config{
		Managed:         true,
		Binary:          "/usr/bin/knxd",
		PhysicalAddress: "0.0.80",
		ClientAddresses: "0.0.81:4",
		Backend: BackendConfig{
			Type:         BackendUSB,
			USBVendorID:  "0e77",
			USBProductID: "0104",
		},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = m.resetUSBDeviceWithContext(ctx)
	if err == nil {
		t.Error("resetUSBDeviceWithContext() should fail with cancelled context")
	}
}

// TestManager_USBBackend_FullIntegration tests the complete USB workflow:
// reset device -> start knxd -> verify EIB handshake -> clean shutdown
func TestManager_USBBackend_FullIntegration(t *testing.T) {
	skipIfNoUSB(t)

	// Use a unique port to avoid conflicts
	port := 16721

	cfg := Config{
		Managed:         true,
		Binary:          "/usr/bin/knxd",
		PhysicalAddress: "0.0.90",
		ClientAddresses: "0.0.91:4",
		ListenTCP:       true,
		TCPPort:         port,
		Backend: BackendConfig{
			Type:         BackendUSB,
			USBVendorID:  "0e77",
			USBProductID: "0104",
		},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}
	m.SetLogger(&testKnxdLogger{
		onInfo: func(msg string, args ...any) {
			t.Logf("[INFO] %s %v", msg, args)
		},
		onWarn: func(msg string, args ...any) {
			t.Logf("[WARN] %s %v", msg, args)
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: Reset USB device before starting knxd
	t.Log("Resetting USB device...")
	if err := m.ResetUSBDevice(); err != nil {
		t.Fatalf("ResetUSBDevice() error: %v", err)
	}
	time.Sleep(1 * time.Second)

	// Step 2: Start knxd with USB backend
	t.Log("Starting knxd with USB backend...")
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer m.Stop()

	// Step 3: Verify knxd is running
	if !m.IsRunning() {
		t.Fatal("knxd should be running")
	}
	t.Logf("knxd running with PID %d", m.Stats().PID)

	// Step 4: Connect and verify EIB protocol works
	t.Log("Testing EIB protocol handshake...")
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to knxd: %v", err)
	}
	defer conn.Close()

	// Send EIB_OPEN_GROUPCON handshake
	handshake := []byte{0x00, 0x05, 0x00, 0x26, 0x00, 0x00, 0x00}
	conn.SetWriteDeadline(time.Now().Add(time.Second))
	if _, err := conn.Write(handshake); err != nil {
		t.Fatalf("Failed to send handshake: %v", err)
	}

	// Read response
	conn.SetReadDeadline(time.Now().Add(time.Second))
	resp := make([]byte, 4)
	if _, err := io.ReadFull(conn, resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify response is EIB_OPEN_GROUPCON (0x0026)
	respType := uint16(resp[2])<<8 | uint16(resp[3])
	if respType != 0x0026 {
		t.Errorf("Expected EIB_OPEN_GROUPCON (0x0026), got 0x%04X", respType)
	}
	t.Logf("EIB handshake OK: response 0x%04X", respType)

	// Step 5: Clean shutdown
	t.Log("Stopping knxd...")
	if err := m.Stop(); err != nil {
		t.Errorf("Stop() error: %v", err)
	}

	if m.IsRunning() {
		t.Error("knxd should not be running after Stop()")
	}
}
