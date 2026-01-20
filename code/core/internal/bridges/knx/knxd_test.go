package knx

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

func TestParseConnectionURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantNetwork string
		wantAddress string
		wantErr     bool
	}{
		{
			name:        "unix socket",
			url:         "unix:///run/knxd",
			wantNetwork: "unix",
			wantAddress: "/run/knxd",
		},
		{
			name:        "unix socket with var run",
			url:         "unix:///var/run/knxd",
			wantNetwork: "unix",
			wantAddress: "/var/run/knxd",
		},
		{
			name:        "tcp with host and port",
			url:         "tcp://localhost:6720",
			wantNetwork: "tcp",
			wantAddress: "localhost:6720",
		},
		{
			name:        "tcp with IP",
			url:         "tcp://192.168.1.100:6720",
			wantNetwork: "tcp",
			wantAddress: "192.168.1.100:6720",
		},
		{
			name:        "tcp without host defaults",
			url:         "tcp://",
			wantNetwork: "tcp",
			wantAddress: "localhost:6720",
		},
		{
			name:    "unsupported scheme",
			url:     "http://localhost:6720",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			url:     "://invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network, address, err := parseConnectionURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Error("parseConnectionURL() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseConnectionURL() unexpected error: %v", err)
				return
			}

			if network != tt.wantNetwork {
				t.Errorf("network = %q, want %q", network, tt.wantNetwork)
			}
			if address != tt.wantAddress {
				t.Errorf("address = %q, want %q", address, tt.wantAddress)
			}
		})
	}
}

func TestKNXDConfigDefaults(t *testing.T) {
	// Test that Connect applies defaults (we can't actually connect,
	// but we can verify the logic with a mock server)

	// Verify defaults exist as constants (can't test actual application
	// without a connection, which would use the cfg)
	_ = KNXDConfig{Connection: "tcp://localhost:16720"}

	// The actual defaults are applied inside Connect,
	// we verify they exist as constants
	if defaultConnectTimeout != 10*time.Second {
		t.Errorf("defaultConnectTimeout = %v, want 10s", defaultConnectTimeout)
	}
	if defaultReadTimeout != 30*time.Second {
		t.Errorf("defaultReadTimeout = %v, want 30s", defaultReadTimeout)
	}
	if defaultReconnectInterval != 5*time.Second {
		t.Errorf("defaultReconnectInterval = %v, want 5s", defaultReconnectInterval)
	}
}

func TestKNXDStats(t *testing.T) {
	// Create a client without connecting to test stats
	client := &KNXDClient{
		done: make(chan struct{}),
	}
	client.lastActivity.Store(time.Now().Unix())

	// Initial stats should be zero
	stats := client.Stats()
	if stats.TelegramsTx != 0 {
		t.Errorf("TelegramsTx = %d, want 0", stats.TelegramsTx)
	}
	if stats.TelegramsRx != 0 {
		t.Errorf("TelegramsRx = %d, want 0", stats.TelegramsRx)
	}
	if stats.ErrorsTotal != 0 {
		t.Errorf("ErrorsTotal = %d, want 0", stats.ErrorsTotal)
	}
	if stats.Connected {
		t.Error("Connected = true, want false")
	}

	// Simulate activity
	client.telegramsTx.Add(5)
	client.telegramsRx.Add(10)
	client.errorsTotal.Add(2)
	client.connMu.Lock()
	client.connected = true
	client.connMu.Unlock()

	stats = client.Stats()
	if stats.TelegramsTx != 5 {
		t.Errorf("TelegramsTx = %d, want 5", stats.TelegramsTx)
	}
	if stats.TelegramsRx != 10 {
		t.Errorf("TelegramsRx = %d, want 10", stats.TelegramsRx)
	}
	if stats.ErrorsTotal != 2 {
		t.Errorf("ErrorsTotal = %d, want 2", stats.ErrorsTotal)
	}
	if !stats.Connected {
		t.Error("Connected = false, want true")
	}
}

func TestKNXDClientIsConnected(t *testing.T) {
	client := &KNXDClient{
		done: make(chan struct{}),
	}

	if client.IsConnected() {
		t.Error("IsConnected() = true, want false (initial)")
	}

	client.connMu.Lock()
	client.connected = true
	client.connMu.Unlock()

	if !client.IsConnected() {
		t.Error("IsConnected() = false, want true")
	}

	client.connMu.Lock()
	client.connected = false
	client.connMu.Unlock()

	if client.IsConnected() {
		t.Error("IsConnected() = true, want false (after disconnect)")
	}
}

func TestKNXDClientSetOnTelegram(t *testing.T) {
	client := &KNXDClient{
		done: make(chan struct{}),
	}

	callback := func(_ Telegram) {
		// Callback set for testing
	}

	client.SetOnTelegram(callback)

	// Verify callback is set
	client.callbackMu.RLock()
	if client.onTelegram == nil {
		t.Error("onTelegram callback not set")
	}
	client.callbackMu.RUnlock()
}

func TestKNXDClientHealthCheck(t *testing.T) {
	client := &KNXDClient{
		done: make(chan struct{}),
	}

	// Not connected should return error
	err := client.HealthCheck(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("HealthCheck() = %v, want ErrNotConnected", err)
	}

	// Mark connected
	client.connMu.Lock()
	client.connected = true
	client.connMu.Unlock()

	err = client.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("HealthCheck() = %v, want nil", err)
	}
}

func TestKNXDClientSendNotConnected(t *testing.T) {
	client := &KNXDClient{
		done: make(chan struct{}),
	}

	err := client.Send(context.Background(), GroupAddress{1, 2, 3}, []byte{0x01})
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("Send() = %v, want ErrNotConnected", err)
	}

	err = client.SendRead(context.Background(), GroupAddress{1, 2, 3})
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("SendRead() = %v, want ErrNotConnected", err)
	}
}

// MockKNXDServer simulates a knxd server for testing.
type MockKNXDServer struct {
	listener net.Listener
	conn     net.Conn
	received [][]byte
	mu       sync.Mutex
	done     chan struct{}
}

// NewMockKNXDServer creates a mock knxd server.
func NewMockKNXDServer(t *testing.T) *MockKNXDServer {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	server := &MockKNXDServer{
		listener: listener,
		done:     make(chan struct{}),
	}

	go server.acceptLoop(t)
	return server
}

func (s *MockKNXDServer) acceptLoop(t *testing.T) {
	conn, err := s.listener.Accept()
	if err != nil {
		select {
		case <-s.done:
			return
		default:
			t.Logf("Accept error: %v", err)
		}
		return
	}

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	// Handle connection
	buf := make([]byte, 256)
	for {
		select {
		case <-s.done:
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				continue
			}
			return
		}

		s.mu.Lock()
		s.received = append(s.received, append([]byte{}, buf[:n]...))
		s.mu.Unlock()

		// Respond to EIB_OPEN_GROUPCON
		if n >= 4 {
			msgType, _, _ := ParseKNXDMessage(buf[:n])
			if msgType == EIBOpenGroupcon {
				resp := EncodeKNXDMessage(EIBOpenGroupcon, nil)
				conn.Write(resp)
			}
		}
	}
}

func (s *MockKNXDServer) Address() string {
	return s.listener.Addr().String()
}

func (s *MockKNXDServer) Close() {
	close(s.done)
	if s.conn != nil {
		s.conn.Close()
	}
	s.listener.Close()
}

func (s *MockKNXDServer) Received() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.received
}

func (s *MockKNXDServer) SendTelegram(t *testing.T, telegram Telegram) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()

	if conn == nil {
		t.Fatal("No connection to send telegram")
	}

	payload := telegram.Encode()
	msg := EncodeKNXDMessage(EIBGroupPacket, payload)
	conn.Write(msg)
}

func TestKNXDClientConnectAndSend(t *testing.T) {
	server := NewMockKNXDServer(t)
	defer server.Close()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	cfg := KNXDConfig{
		Connection:     "tcp://" + server.Address(),
		ConnectTimeout: 2 * time.Second,
		ReadTimeout:    1 * time.Second,
	}

	ctx := context.Background()
	client, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("IsConnected() = false after Connect")
	}

	// Send a telegram
	ga := GroupAddress{Main: 1, Middle: 2, Sub: 3}
	data := []byte{0x01}
	err = client.Send(ctx, ga, data)
	if err != nil {
		t.Errorf("Send() error: %v", err)
	}

	// Verify stats updated
	stats := client.Stats()
	if stats.TelegramsTx != 1 {
		t.Errorf("TelegramsTx = %d, want 1", stats.TelegramsTx)
	}
}

func TestKNXDClientReceiveTelegram(t *testing.T) {
	server := NewMockKNXDServer(t)
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	cfg := KNXDConfig{
		Connection:     "tcp://" + server.Address(),
		ConnectTimeout: 2 * time.Second,
		ReadTimeout:    1 * time.Second,
	}

	client, err := Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer client.Close()

	// Set up callback
	received := make(chan Telegram, 1)
	client.SetOnTelegram(func(t Telegram) {
		received <- t
	})

	// Give time for receive loop to start
	time.Sleep(50 * time.Millisecond)

	// Send telegram from server
	testTelegram := Telegram{
		Destination: GroupAddress{Main: 5, Middle: 0, Sub: 1},
		APCI:        APCIWrite,
		Data:        []byte{0x0C, 0x66},
	}
	server.SendTelegram(t, testTelegram)

	// Wait for callback
	select {
	case got := <-received:
		if got.Destination != testTelegram.Destination {
			t.Errorf("Destination = %v, want %v", got.Destination, testTelegram.Destination)
		}
		if got.APCI != testTelegram.APCI {
			t.Errorf("APCI = 0x%02X, want 0x%02X", got.APCI, testTelegram.APCI)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for telegram callback")
	}

	// Verify stats
	stats := client.Stats()
	if stats.TelegramsRx != 1 {
		t.Errorf("TelegramsRx = %d, want 1", stats.TelegramsRx)
	}
}

func TestKNXDClientClose(t *testing.T) {
	server := NewMockKNXDServer(t)
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	cfg := KNXDConfig{
		Connection:     "tcp://" + server.Address(),
		ConnectTimeout: 2 * time.Second,
		ReadTimeout:    500 * time.Millisecond,
	}

	client, err := Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	if !client.IsConnected() {
		t.Error("IsConnected() = false after Connect")
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
	}

	if client.IsConnected() {
		t.Error("IsConnected() = true after Close")
	}
}

func TestKNXDClientConnectFailure(t *testing.T) {
	// Try to connect to non-existent server
	cfg := KNXDConfig{
		Connection:     "tcp://127.0.0.1:19999", // Non-existent port
		ConnectTimeout: 500 * time.Millisecond,
	}

	ctx := context.Background()
	_, err := Connect(ctx, cfg)
	if err == nil {
		t.Error("Connect() expected error for non-existent server")
	}
}

func TestKNXDClientContextCancellation(t *testing.T) {
	server := NewMockKNXDServer(t)
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	cfg := KNXDConfig{
		Connection:     "tcp://" + server.Address(),
		ConnectTimeout: 2 * time.Second,
		ReadTimeout:    1 * time.Second,
	}

	client, err := Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer client.Close()

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = client.Send(ctx, GroupAddress{1, 2, 3}, []byte{0x01})
	if err == nil {
		t.Error("Send() with cancelled context should fail")
	}
}
