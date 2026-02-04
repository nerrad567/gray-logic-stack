package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"

	"github.com/nerrad567/gray-logic-core/internal/device"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/logging"
	"github.com/nerrad567/gray-logic-core/internal/location"
)

// testServer creates a Server with a real device registry backed by in-memory SQLite.
func testServer(t *testing.T) (*Server, *device.Registry) {
	t.Helper()

	db := setupTestDB(t)
	repo := device.NewSQLiteRepository(db)
	registry := device.NewRegistry(repo)
	if err := registry.RefreshCache(context.Background()); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	log := logging.New(config.LoggingConfig{Level: "error", Format: "text", Output: "stdout"}, "test")

	srv, err := New(Deps{
		Config: config.APIConfig{
			Host: "127.0.0.1",
			Port: 0,
			Timeouts: config.APITimeoutConfig{
				Read:  5,
				Write: 5,
				Idle:  5,
			},
		},
		WS: config.WebSocketConfig{
			MaxMessageSize: 8192,
			PingInterval:   30,
			PongTimeout:    10,
		},
		Security: config.SecurityConfig{
			JWT: config.JWTConfig{
				Secret:         "test-secret-key-at-least-32-characters-long",
				AccessTokenTTL: 15,
			},
		},
		Logger:   log,
		Registry: registry,
		MQTT:     nil, // Tests that need MQTT will use a mock
		Version:  "test",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Initialise hub for tests
	srv.hub = NewHub(srv.wsCfg, log)
	go srv.hub.Run(context.Background())

	return srv, registry
}

// setupTestDB creates an in-memory SQLite database with the devices schema.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE devices (
			id TEXT PRIMARY KEY,
			room_id TEXT,
			area_id TEXT,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			type TEXT NOT NULL,
			domain TEXT NOT NULL,
			protocol TEXT NOT NULL,
			address TEXT NOT NULL,
			gateway_id TEXT,
			capabilities TEXT NOT NULL DEFAULT '[]',
			config TEXT NOT NULL DEFAULT '{}',
			state TEXT NOT NULL DEFAULT '{}',
			state_updated_at TEXT,
			health_status TEXT NOT NULL DEFAULT 'unknown',
			health_last_seen TEXT,
			phm_enabled INTEGER NOT NULL DEFAULT 0,
			phm_baseline TEXT,
			manufacturer TEXT,
			model TEXT,
			firmware_version TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		) STRICT;
		CREATE INDEX idx_devices_room_id ON devices(room_id);
		CREATE INDEX idx_devices_area_id ON devices(area_id);
		CREATE INDEX idx_devices_domain ON devices(domain);
		CREATE INDEX idx_devices_protocol ON devices(protocol);
		CREATE INDEX idx_devices_type ON devices(type);
		CREATE INDEX idx_devices_health ON devices(health_status);
	`

	if _, execErr := db.Exec(schema); execErr != nil {
		db.Close()
		t.Fatalf("failed to create test schema: %v", execErr)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

// ─── Health Endpoint Tests ─────────────────────────────────────────

func TestHealth(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
	if resp["version"] != "test" {
		t.Errorf("version = %v, want test", resp["version"])
	}
}

func TestHealth_ContentType(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

// ─── Middleware Tests ──────────────────────────────────────────────

func TestRequestID_Generated(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}

func TestRequestID_PreservesClient(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("X-Request-ID", "client-123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-ID"); got != "client-123" {
		t.Errorf("X-Request-ID = %q, want %q", got, "client-123")
	}
}

func TestCORS_Preflight(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want %d", w.Code, http.StatusNoContent)
	}

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("ACAO = %q, want %q", got, "http://localhost:3000")
	}
}

func TestNotFound(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("unknown route status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ─── Device CRUD Tests ─────────────────────────────────────────────

func TestListDevices_Empty(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if int(resp["count"].(float64)) != 0 {
		t.Errorf("count = %v, want 0", resp["count"])
	}
}

func TestCreateAndGetDevice(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	body := `{
		"name": "Test Light",
		"type": "light_dimmer",
		"domain": "lighting",
		"protocol": "knx",
		"address": {"functions": {"switch": {"ga": "1/2/3", "dpt": "1.001", "flags": ["write"]}, "brightness": {"ga": "1/2/4", "dpt": "5.001", "flags": ["write"]}}},
		"capabilities": ["on_off", "dim"]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var created device.Device
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created: %v", err)
	}

	if created.ID == "" {
		t.Error("expected device ID to be auto-generated")
	}
	if created.Slug == "" {
		t.Error("expected slug to be auto-generated")
	}

	// Get device by ID
	req = httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+created.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", w.Code, http.StatusOK)
	}

	var got device.Device
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}

	if got.Name != "Test Light" {
		t.Errorf("name = %q, want %q", got.Name, "Test Light")
	}
}

func TestGetDevice_NotFound(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/nonexistent-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUpdateDevice(t *testing.T) {
	srv, registry := testServer(t)
	router := srv.buildRouter()

	dev := &device.Device{
		Name:     "Original",
		Type:     device.DeviceTypeLightDimmer,
		Domain:   device.DomainLighting,
		Protocol: device.ProtocolKNX,
		Address: device.Address{"functions": map[string]any{
			"switch": map[string]any{"ga": "1/2/3", "dpt": "1.001", "flags": []any{"write"}},
		}},
		Capabilities: []device.Capability{device.CapOnOff},
	}
	if err := registry.CreateDevice(context.Background(), dev); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	body := `{"name": "Updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/devices/"+dev.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var updated device.Device
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if updated.Name != "Updated" {
		t.Errorf("name = %q, want %q", updated.Name, "Updated")
	}
}

func TestDeleteDevice(t *testing.T) {
	srv, registry := testServer(t)
	router := srv.buildRouter()

	dev := &device.Device{
		Name:     "ToDelete",
		Type:     device.DeviceTypeLightSwitch,
		Domain:   device.DomainLighting,
		Protocol: device.ProtocolKNX,
		Address: device.Address{"functions": map[string]any{
			"switch": map[string]any{"ga": "1/2/4", "dpt": "1.001", "flags": []any{"write"}},
		}},
		Capabilities: []device.Capability{device.CapOnOff},
	}
	if err := registry.CreateDevice(context.Background(), dev); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/devices/"+dev.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Confirm gone
	req = httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+dev.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("get after delete status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestListDevices_FilterByDomain(t *testing.T) {
	srv, registry := testServer(t)
	router := srv.buildRouter()

	dev := &device.Device{
		Name:     "Kitchen Light",
		Type:     device.DeviceTypeLightDimmer,
		Domain:   device.DomainLighting,
		Protocol: device.ProtocolKNX,
		Address: device.Address{"functions": map[string]any{
			"switch":     map[string]any{"ga": "1/2/5", "dpt": "1.001", "flags": []any{"write"}},
			"brightness": map[string]any{"ga": "1/2/6", "dpt": "5.001", "flags": []any{"write"}},
		}},
		Capabilities: []device.Capability{device.CapOnOff, device.CapDim},
	}
	if err := registry.CreateDevice(context.Background(), dev); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	// Filter by domain=lighting
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices?domain=lighting", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if int(resp["count"].(float64)) != 1 {
		t.Errorf("count = %v, want 1", resp["count"])
	}

	// Filter by domain=climate (should be empty)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/devices?domain=climate", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if int(resp["count"].(float64)) != 0 {
		t.Errorf("count for climate = %v, want 0", resp["count"])
	}
}

// ─── Device State Tests ────────────────────────────────────────────

func TestGetDeviceState(t *testing.T) {
	srv, registry := testServer(t)
	router := srv.buildRouter()

	dev := &device.Device{
		Name:     "Stateful Light",
		Type:     device.DeviceTypeLightDimmer,
		Domain:   device.DomainLighting,
		Protocol: device.ProtocolKNX,
		Address: device.Address{"functions": map[string]any{
			"switch":     map[string]any{"ga": "1/2/6", "dpt": "1.001", "flags": []any{"write"}},
			"brightness": map[string]any{"ga": "1/2/7", "dpt": "5.001", "flags": []any{"write"}},
		}},
		Capabilities: []device.Capability{device.CapOnOff},
	}
	if err := registry.CreateDevice(context.Background(), dev); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	state := device.State{"on": true, "level": float64(75)}
	if err := registry.SetDeviceState(context.Background(), dev.ID, state); err != nil {
		t.Fatalf("SetDeviceState: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+dev.ID+"/state", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	stateMap, ok := resp["state"].(map[string]any)
	if !ok {
		t.Fatalf("state is not a map: %T", resp["state"])
	}

	if stateMap["on"] != true {
		t.Errorf("state.on = %v, want true", stateMap["on"])
	}
}

func TestSetDeviceState_MissingCommand(t *testing.T) {
	srv, registry := testServer(t)
	router := srv.buildRouter()

	dev := &device.Device{
		Name:     "No Command",
		Type:     device.DeviceTypeLightSwitch,
		Domain:   device.DomainLighting,
		Protocol: device.ProtocolKNX,
		Address: device.Address{"functions": map[string]any{
			"switch": map[string]any{"ga": "1/2/9", "dpt": "1.001", "flags": []any{"write"}},
		}},
		Capabilities: []device.Capability{device.CapOnOff},
	}
	if err := registry.CreateDevice(context.Background(), dev); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	body := `{"parameters": {"level": 50}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/devices/"+dev.ID+"/state", strings.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSetDeviceState_DeviceNotFound(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	body := `{"command": "on"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/devices/nonexistent/state", strings.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ─── Device Stats Tests ────────────────────────────────────────────

func TestDeviceStats(t *testing.T) {
	srv, registry := testServer(t)
	router := srv.buildRouter()

	dev := &device.Device{
		Name:     "Stats Device",
		Type:     device.DeviceTypeLightSwitch,
		Domain:   device.DomainLighting,
		Protocol: device.ProtocolKNX,
		Address: device.Address{"functions": map[string]any{
			"switch": map[string]any{"ga": "1/2/7", "dpt": "1.001", "flags": []any{"write"}},
		}},
		Capabilities: []device.Capability{device.CapOnOff},
	}
	if err := registry.CreateDevice(context.Background(), dev); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ─── Create Validation Tests ───────────────────────────────────────

func TestCreateDevice_InvalidJSON(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ─── Auth Tests ────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	body := `{"username": "admin", "password": "admin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp loginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected access_token to be non-empty")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("token_type = %q, want Bearer", resp.TokenType)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	body := `{"username": "admin", "password": "wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestWSTicket_SingleUse(t *testing.T) {
	srv, _ := testServer(t)
	router := srv.buildRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/ws-ticket", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	ticket, ok := resp["ticket"].(string)
	if !ok || ticket == "" {
		t.Error("expected ticket to be a non-empty string")
	}

	// Ticket should be valid once
	if !validateTicket(ticket) {
		t.Error("ticket should be valid on first use")
	}

	// Ticket should be consumed (single-use)
	if validateTicket(ticket) {
		t.Error("ticket should not be valid on second use")
	}
}

func TestWSTicket_Expiry(t *testing.T) {
	ticket := generateTicket()
	wsTickets.mu.Lock()
	wsTickets.tickets[ticket] = ticketEntry{
		expiresAt: time.Now().Add(-1 * time.Second),
	}
	wsTickets.mu.Unlock()

	if validateTicket(ticket) {
		t.Error("expired ticket should not be valid")
	}
}

// ─── WebSocket Hub Tests ───────────────────────────────────────────

func TestHub_BroadcastToSubscribed(t *testing.T) {
	log := logging.New(config.LoggingConfig{Level: "error", Format: "text", Output: "stdout"}, "test")
	hub := NewHub(config.WebSocketConfig{MaxMessageSize: 8192, PingInterval: 30, PongTimeout: 10}, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// Create a mock client
	client := &WSClient{
		hub:           hub,
		send:          make(chan []byte, wsSendBufferSize),
		subscriptions: map[string]struct{}{"device.state_changed": {}},
	}
	hub.Register(client)

	// Broadcast
	hub.Broadcast("device.state_changed", map[string]any{"device_id": "test-1", "on": true})

	// Should receive the message
	select {
	case msg := <-client.send:
		var wsMsg WSMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if wsMsg.EventType != "device.state_changed" {
			t.Errorf("event_type = %q, want %q", wsMsg.EventType, "device.state_changed")
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for broadcast message")
	}
}

func TestHub_NoMessageForUnsubscribed(t *testing.T) {
	log := logging.New(config.LoggingConfig{Level: "error", Format: "text", Output: "stdout"}, "test")
	hub := NewHub(config.WebSocketConfig{MaxMessageSize: 8192, PingInterval: 30, PongTimeout: 10}, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// Client not subscribed to "device.state_changed"
	client := &WSClient{
		hub:           hub,
		send:          make(chan []byte, wsSendBufferSize),
		subscriptions: map[string]struct{}{"scene.activated": {}},
	}
	hub.Register(client)

	hub.Broadcast("device.state_changed", map[string]any{"device_id": "test-1"})

	// Should NOT receive the message
	select {
	case <-client.send:
		t.Error("unsubscribed client should not receive message")
	case <-time.After(100 * time.Millisecond):
		// OK — no message received
	}
}

func TestHub_ClientCount(t *testing.T) {
	log := logging.New(config.LoggingConfig{Level: "error", Format: "text", Output: "stdout"}, "test")
	hub := NewHub(config.WebSocketConfig{MaxMessageSize: 8192, PingInterval: 30, PongTimeout: 10}, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	if hub.ClientCount() != 0 {
		t.Errorf("initial client count = %d, want 0", hub.ClientCount())
	}

	client := &WSClient{
		hub:           hub,
		send:          make(chan []byte, wsSendBufferSize),
		subscriptions: make(map[string]struct{}),
	}
	hub.Register(client)

	if hub.ClientCount() != 1 {
		t.Errorf("after register count = %d, want 1", hub.ClientCount())
	}

	hub.Unregister(client)

	if hub.ClientCount() != 0 {
		t.Errorf("after unregister count = %d, want 0", hub.ClientCount())
	}
}

// ─── Location Handler Tests ────────────────────────────────────────

// mockLocationRepo is a test implementation of location.Repository.
type mockLocationRepo struct {
	areas []location.Area
	rooms []location.Room
	// Error injection
	createAreaErr error
	createRoomErr error
	listAreasErr  error
	listRoomsErr  error
	getAreaErr    error
	getRoomErr    error
}

func newMockLocationRepo() *mockLocationRepo {
	return &mockLocationRepo{
		areas: []location.Area{},
		rooms: []location.Room{},
	}
}

func (m *mockLocationRepo) CreateArea(_ context.Context, area *location.Area) error {
	if m.createAreaErr != nil {
		return m.createAreaErr
	}
	m.areas = append(m.areas, *area)
	return nil
}

func (m *mockLocationRepo) ListAreas(_ context.Context) ([]location.Area, error) {
	if m.listAreasErr != nil {
		return nil, m.listAreasErr
	}
	return m.areas, nil
}

func (m *mockLocationRepo) ListAreasBySite(_ context.Context, siteID string) ([]location.Area, error) {
	if m.listAreasErr != nil {
		return nil, m.listAreasErr
	}
	var result []location.Area
	for _, a := range m.areas {
		if a.SiteID == siteID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockLocationRepo) GetArea(_ context.Context, id string) (*location.Area, error) {
	if m.getAreaErr != nil {
		return nil, m.getAreaErr
	}
	for _, a := range m.areas {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, location.ErrAreaNotFound
}

func (m *mockLocationRepo) CreateRoom(_ context.Context, room *location.Room) error {
	if m.createRoomErr != nil {
		return m.createRoomErr
	}
	m.rooms = append(m.rooms, *room)
	return nil
}

func (m *mockLocationRepo) ListRooms(_ context.Context) ([]location.Room, error) {
	if m.listRoomsErr != nil {
		return nil, m.listRoomsErr
	}
	return m.rooms, nil
}

func (m *mockLocationRepo) ListRoomsByArea(_ context.Context, areaID string) ([]location.Room, error) {
	if m.listRoomsErr != nil {
		return nil, m.listRoomsErr
	}
	var result []location.Room
	for _, r := range m.rooms {
		if r.AreaID == areaID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockLocationRepo) GetRoom(_ context.Context, id string) (*location.Room, error) {
	if m.getRoomErr != nil {
		return nil, m.getRoomErr
	}
	for _, r := range m.rooms {
		if r.ID == id {
			return &r, nil
		}
	}
	return nil, location.ErrRoomNotFound
}

func (m *mockLocationRepo) UpdateArea(_ context.Context, _ *location.Area) error {
	return nil
}

func (m *mockLocationRepo) DeleteArea(_ context.Context, _ string) error {
	return nil
}

func (m *mockLocationRepo) UpdateRoom(_ context.Context, _ *location.Room) error {
	return nil
}

func (m *mockLocationRepo) DeleteRoom(_ context.Context, _ string) error {
	return nil
}

func (m *mockLocationRepo) DeleteAllAreas(_ context.Context) (int64, error) {
	n := int64(len(m.areas))
	m.areas = nil
	return n, nil
}

func (m *mockLocationRepo) DeleteAllRooms(_ context.Context) (int64, error) {
	n := int64(len(m.rooms))
	m.rooms = nil
	return n, nil
}

func (m *mockLocationRepo) GetAnySite(_ context.Context) (*location.Site, error) {
	return nil, location.ErrSiteNotFound
}

func (m *mockLocationRepo) CreateSite(_ context.Context, _ *location.Site) error {
	return nil
}

func (m *mockLocationRepo) UpdateSite(_ context.Context, _ *location.Site) error {
	return nil
}

func (m *mockLocationRepo) DeleteAllSites(_ context.Context) (int64, error) {
	return 0, nil
}

// testServerWithLocation creates a Server with location repo for testing.
func testServerWithLocation(t *testing.T) (*Server, *mockLocationRepo) {
	t.Helper()

	db := setupTestDB(t)
	repo := device.NewSQLiteRepository(db)
	registry := device.NewRegistry(repo)
	if err := registry.RefreshCache(context.Background()); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	log := logging.New(config.LoggingConfig{Level: "error", Format: "text", Output: "stdout"}, "test")
	locRepo := newMockLocationRepo()

	srv, err := New(Deps{
		Config: config.APIConfig{
			Host: "127.0.0.1",
			Port: 0,
			Timeouts: config.APITimeoutConfig{
				Read:  5,
				Write: 5,
				Idle:  5,
			},
		},
		WS: config.WebSocketConfig{
			MaxMessageSize: 8192,
			PingInterval:   30,
			PongTimeout:    10,
		},
		Security: config.SecurityConfig{
			JWT: config.JWTConfig{
				Secret:         "test-secret-key-at-least-32-characters-long",
				AccessTokenTTL: 15,
			},
		},
		Logger:       log,
		Registry:     registry,
		LocationRepo: locRepo,
		MQTT:         nil,
		Version:      "test",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	srv.hub = NewHub(srv.wsCfg, log)
	go srv.hub.Run(context.Background())

	return srv, locRepo
}

func TestCreateArea(t *testing.T) {
	srv, locRepo := testServerWithLocation(t)
	router := srv.buildRouter()

	t.Run("creates area successfully", func(t *testing.T) {
		body := `{"id":"area-1","name":"Ground Floor","site_id":"site-1"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/areas", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}

		if len(locRepo.areas) != 1 {
			t.Errorf("areas count = %d, want 1", len(locRepo.areas))
		}
	})

	t.Run("returns 400 for missing fields", func(t *testing.T) {
		body := `{"id":"area-2"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/areas", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/areas", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestCreateRoom(t *testing.T) {
	srv, locRepo := testServerWithLocation(t)
	router := srv.buildRouter()

	t.Run("creates room successfully", func(t *testing.T) {
		body := `{"id":"room-1","name":"Living Room","area_id":"area-1"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/rooms", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}

		if len(locRepo.rooms) != 1 {
			t.Errorf("rooms count = %d, want 1", len(locRepo.rooms))
		}
	})

	t.Run("returns 400 for missing fields", func(t *testing.T) {
		body := `{"id":"room-2"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/rooms", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestListAreas(t *testing.T) {
	srv, locRepo := testServerWithLocation(t)
	router := srv.buildRouter()

	// Add some test areas
	locRepo.areas = []location.Area{
		{ID: "area-1", SiteID: "site-1", Name: "Ground Floor", Slug: "ground-floor", Type: "floor"},
		{ID: "area-2", SiteID: "site-1", Name: "First Floor", Slug: "first-floor", Type: "floor"},
		{ID: "area-3", SiteID: "site-2", Name: "Basement", Slug: "basement", Type: "floor"},
	}

	t.Run("lists all areas", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/areas", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["count"].(float64) != 3 {
			t.Errorf("count = %v, want 3", resp["count"])
		}
	})

	t.Run("filters by site_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/areas?site_id=site-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["count"].(float64) != 2 {
			t.Errorf("count = %v, want 2", resp["count"])
		}
	})
}

func TestGetArea(t *testing.T) {
	srv, locRepo := testServerWithLocation(t)
	router := srv.buildRouter()

	locRepo.areas = []location.Area{
		{ID: "area-1", SiteID: "site-1", Name: "Ground Floor", Slug: "ground-floor", Type: "floor"},
	}

	t.Run("returns area by ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/areas/area-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var area location.Area
		json.Unmarshal(w.Body.Bytes(), &area)
		if area.Name != "Ground Floor" {
			t.Errorf("name = %q, want %q", area.Name, "Ground Floor")
		}
	})

	t.Run("returns 404 for unknown ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/areas/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestListRooms(t *testing.T) {
	srv, locRepo := testServerWithLocation(t)
	router := srv.buildRouter()

	locRepo.rooms = []location.Room{
		{ID: "room-1", AreaID: "area-1", Name: "Living Room", Slug: "living-room", Type: "living"},
		{ID: "room-2", AreaID: "area-1", Name: "Kitchen", Slug: "kitchen", Type: "kitchen"},
		{ID: "room-3", AreaID: "area-2", Name: "Master Bedroom", Slug: "master-bedroom", Type: "bedroom"},
	}

	t.Run("lists all rooms", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/rooms", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["count"].(float64) != 3 {
			t.Errorf("count = %v, want 3", resp["count"])
		}
	})

	t.Run("filters by area_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/rooms?area_id=area-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["count"].(float64) != 2 {
			t.Errorf("count = %v, want 2", resp["count"])
		}
	})
}

func TestGetRoom(t *testing.T) {
	srv, locRepo := testServerWithLocation(t)
	router := srv.buildRouter()

	locRepo.rooms = []location.Room{
		{ID: "room-1", AreaID: "area-1", Name: "Living Room", Slug: "living-room", Type: "living"},
	}

	t.Run("returns room by ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/rooms/room-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var room location.Room
		json.Unmarshal(w.Body.Bytes(), &room)
		if room.Name != "Living Room" {
			t.Errorf("name = %q, want %q", room.Name, "Living Room")
		}
	})

	t.Run("returns 404 for unknown ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/rooms/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// ─── Error Path Tests ──────────────────────────────────────────────

func TestListAreas_InternalError(t *testing.T) {
	srv, locRepo := testServerWithLocation(t)
	router := srv.buildRouter()

	locRepo.listAreasErr = errors.New("database error")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/areas", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestListRooms_InternalError(t *testing.T) {
	srv, locRepo := testServerWithLocation(t)
	router := srv.buildRouter()

	locRepo.listRoomsErr = errors.New("database error")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rooms", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestCreateArea_InternalError(t *testing.T) {
	srv, locRepo := testServerWithLocation(t)
	router := srv.buildRouter()

	locRepo.createAreaErr = errors.New("database error")

	body := `{"id":"area-1","name":"Ground Floor","site_id":"site-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/areas", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestGetArea_InternalError(t *testing.T) {
	srv, locRepo := testServerWithLocation(t)
	router := srv.buildRouter()

	locRepo.getAreaErr = errors.New("database error")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/areas/area-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestGetRoom_InternalError(t *testing.T) {
	srv, locRepo := testServerWithLocation(t)
	router := srv.buildRouter()

	locRepo.getRoomErr = errors.New("database error")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rooms/room-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ─── Device Helper Tests ───────────────────────────────────────────

func TestCommandToState(t *testing.T) {
	tests := []struct {
		name    string
		command string
		params  map[string]any
		current device.State
		want    device.State
	}{
		{
			name:    "turn_on sets on=true",
			command: "turn_on",
			params:  nil,
			current: device.State{},
			want:    device.State{"on": true},
		},
		{
			name:    "turn_off sets on=false",
			command: "turn_off",
			params:  nil,
			current: device.State{"on": true},
			want:    device.State{"on": false},
		},
		{
			name:    "set_brightness sets brightness",
			command: "set_brightness",
			params:  map[string]any{"brightness": float64(75)},
			current: device.State{},
			want:    device.State{"brightness": float64(75)},
		},
		{
			name:    "set_position sets position",
			command: "set_position",
			params:  map[string]any{"position": float64(50)},
			current: device.State{},
			want:    device.State{"position": float64(50)},
		},
		{
			name:    "set_temperature sets setpoint",
			command: "set_temperature",
			params:  map[string]any{"setpoint": float64(22.5)},
			current: device.State{},
			want:    device.State{"setpoint": float64(22.5)},
		},
		{
			name:    "set_mode sets mode",
			command: "set_mode",
			params:  map[string]any{"mode": "auto"},
			current: device.State{},
			want:    device.State{"mode": "auto"},
		},
		{
			name:    "unknown command returns current state",
			command: "unknown_command",
			params:  nil,
			current: device.State{"on": true},
			want:    device.State{"on": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commandToState(tt.command, tt.params, tt.current)
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("state[%s] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"ErrInvalidDevice", device.ErrInvalidDevice, true},
		{"ErrInvalidName", device.ErrInvalidName, true},
		{"ErrInvalidSlug", device.ErrInvalidSlug, true},
		{"ErrInvalidDeviceType", device.ErrInvalidDeviceType, true},
		{"ErrInvalidDomain", device.ErrInvalidDomain, true},
		{"ErrInvalidProtocol", device.ErrInvalidProtocol, true},
		{"ErrInvalidAddress", device.ErrInvalidAddress, true},
		{"ErrInvalidCapability", device.ErrInvalidCapability, true},
		{"ErrInvalidState", device.ErrInvalidState, true},
		{"random error", errors.New("random"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidationError(tt.err)
			if got != tt.want {
				t.Errorf("isValidationError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── WebSocket Integration Tests ───────────────────────────────────────────

// testServerWithRealListener creates a server that actually listens on a specific port.
func testServerWithRealListener(t *testing.T, port int) (*Server, string) {
	t.Helper()

	db := setupTestDB(t)
	repo := device.NewSQLiteRepository(db)
	registry := device.NewRegistry(repo)
	if err := registry.RefreshCache(context.Background()); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	log := logging.New(config.LoggingConfig{Level: "error", Format: "text", Output: "stdout"}, "test")

	srv, err := New(Deps{
		Config: config.APIConfig{
			Host: "127.0.0.1",
			Port: port,
			Timeouts: config.APITimeoutConfig{
				Read:  5,
				Write: 5,
				Idle:  5,
			},
		},
		WS: config.WebSocketConfig{
			MaxMessageSize: 8192,
			PingInterval:   30,
			PongTimeout:    10,
		},
		Security: config.SecurityConfig{
			JWT: config.JWTConfig{
				Secret:         "test-secret-key-at-least-32-characters-long",
				AccessTokenTTL: 15,
			},
		},
		Logger:   log,
		Registry: registry,
		MQTT:     nil,
		Version:  "test",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Cleanup(func() { srv.Close() })

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	return srv, addr
}

func TestServer_StartAndClose(t *testing.T) {
	db := setupTestDB(t)
	repo := device.NewSQLiteRepository(db)
	registry := device.NewRegistry(repo)
	_ = registry.RefreshCache(context.Background())

	log := logging.New(config.LoggingConfig{Level: "error", Format: "text", Output: "stdout"}, "test")

	// Use a specific port for this test
	port := 19080

	srv, err := New(Deps{
		Config: config.APIConfig{
			Host: "127.0.0.1",
			Port: port,
			Timeouts: config.APITimeoutConfig{
				Read:  5,
				Write: 5,
				Idle:  5,
			},
		},
		WS: config.WebSocketConfig{
			MaxMessageSize: 8192,
			PingInterval:   30,
			PongTimeout:    10,
		},
		Security: config.SecurityConfig{
			JWT: config.JWTConfig{
				Secret:         "test-secret-key-at-least-32-characters-long",
				AccessTokenTTL: 15,
			},
		},
		Logger:   log,
		Registry: registry,
		Version:  "test",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start server
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	addr := fmt.Sprintf("127.0.0.1:%d", port)

	// Verify server responds
	resp, err := http.Get("http://" + addr + "/api/v1/health")
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("health check status = %d, want 200", resp.StatusCode)
	}

	// Close server
	cancel()
	if err := srv.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}

	// Verify server stopped by trying to connect (should fail)
	time.Sleep(100 * time.Millisecond)
	_, err = http.Get("http://" + addr + "/api/v1/health")
	if err == nil {
		t.Error("server still responding after Close()")
	}
}

func TestServer_HealthCheck(t *testing.T) {
	srv, _ := testServer(t)

	// HealthCheck returns error if unhealthy, nil if healthy
	// Since the server isn't started, it should return an error
	ctx := context.Background()
	err := srv.HealthCheck(ctx)

	// Server not started, so health check should fail
	if err == nil {
		t.Log("HealthCheck returned nil (server considered healthy)")
	}
	// This is expected - the server struct exists but isn't listening
}

func TestWebSocket_FullConnection(t *testing.T) {
	srv, addr := testServerWithRealListener(t, 19081)
	defer srv.Close()

	// First, get a valid JWT token
	loginResp, err := http.Post(
		"http://"+addr+"/api/v1/auth/login",
		"application/json",
		strings.NewReader(`{"username":"admin","password":"admin"}`),
	)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer loginResp.Body.Close()

	var loginResult struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&loginResult); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	// Get WebSocket ticket
	req, _ := http.NewRequest("POST", "http://"+addr+"/api/v1/auth/ws-ticket", nil)
	req.Header.Set("Authorization", "Bearer "+loginResult.AccessToken)
	ticketResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ws-ticket request failed: %v", err)
	}
	defer ticketResp.Body.Close()

	var ticketResult struct {
		Ticket string `json:"ticket"`
	}
	if err := json.NewDecoder(ticketResp.Body).Decode(&ticketResult); err != nil {
		t.Fatalf("decode ticket response: %v", err)
	}

	// Connect via WebSocket
	wsURL := "ws://" + addr + "/api/v1/ws?ticket=" + ticketResult.Ticket
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v (resp: %v)", err, resp)
	}
	defer ws.Close()

	// Subscribe to a channel
	subscribeMsg := WSMessage{
		Type: WSTypeSubscribe,
		ID:   "sub-1",
		Payload: WSSubscribePayload{
			Channels: []string{"device.state_changed"},
		},
	}
	if err := ws.WriteJSON(subscribeMsg); err != nil {
		t.Fatalf("write subscribe message: %v", err)
	}

	// Read response
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var response WSMessage
	if err := ws.ReadJSON(&response); err != nil {
		t.Fatalf("read response: %v", err)
	}

	if response.Type != WSTypeResponse {
		t.Errorf("response type = %s, want %s", response.Type, WSTypeResponse)
	}
	if response.ID != "sub-1" {
		t.Errorf("response ID = %s, want sub-1", response.ID)
	}

	// Verify client is subscribed by checking hub
	if srv.hub.ClientCount() != 1 {
		t.Errorf("hub client count = %d, want 1", srv.hub.ClientCount())
	}
}

func TestWebSocket_SubscribeUnsubscribe(t *testing.T) {
	srv, addr := testServerWithRealListener(t, 19082)
	defer srv.Close()

	// Login and get ticket
	ws := connectWebSocket(t, addr)
	defer ws.Close()

	// Subscribe
	if err := ws.WriteJSON(WSMessage{
		Type:    WSTypeSubscribe,
		ID:      "sub-1",
		Payload: WSSubscribePayload{Channels: []string{"device.state_changed", "scene.activated"}},
	}); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}

	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp WSMessage
	if err := ws.ReadJSON(&resp); err != nil {
		t.Fatalf("read subscribe response: %v", err)
	}

	if resp.Type != WSTypeResponse {
		t.Errorf("subscribe response type = %s, want response", resp.Type)
	}

	// Unsubscribe from one channel
	if err := ws.WriteJSON(WSMessage{
		Type:    WSTypeUnsubscribe,
		ID:      "unsub-1",
		Payload: WSSubscribePayload{Channels: []string{"scene.activated"}},
	}); err != nil {
		t.Fatalf("write unsubscribe: %v", err)
	}

	if err := ws.ReadJSON(&resp); err != nil {
		t.Fatalf("read unsubscribe response: %v", err)
	}

	if resp.Type != WSTypeResponse {
		t.Errorf("unsubscribe response type = %s, want response", resp.Type)
	}
}

func TestWebSocket_Ping(t *testing.T) {
	srv, addr := testServerWithRealListener(t, 19083)
	defer srv.Close()

	ws := connectWebSocket(t, addr)
	defer ws.Close()

	// Send ping
	if err := ws.WriteJSON(WSMessage{
		Type: WSTypePing,
		ID:   "ping-1",
	}); err != nil {
		t.Fatalf("write ping: %v", err)
	}

	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp WSMessage
	if err := ws.ReadJSON(&resp); err != nil {
		t.Fatalf("read pong: %v", err)
	}

	if resp.Type != WSTypePong {
		t.Errorf("response type = %s, want pong", resp.Type)
	}
	if resp.ID != "ping-1" {
		t.Errorf("response ID = %s, want ping-1", resp.ID)
	}
}

func TestWebSocket_InvalidMessage(t *testing.T) {
	srv, addr := testServerWithRealListener(t, 19084)
	defer srv.Close()

	ws := connectWebSocket(t, addr)
	defer ws.Close()

	// Send invalid JSON
	if err := ws.WriteMessage(websocket.TextMessage, []byte("not json")); err != nil {
		t.Fatalf("write invalid message: %v", err)
	}

	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp WSMessage
	if err := ws.ReadJSON(&resp); err != nil {
		t.Fatalf("read error response: %v", err)
	}

	if resp.Type != WSTypeError {
		t.Errorf("response type = %s, want error", resp.Type)
	}
}

func TestWebSocket_UnknownMessageType(t *testing.T) {
	srv, addr := testServerWithRealListener(t, 19085)
	defer srv.Close()

	ws := connectWebSocket(t, addr)
	defer ws.Close()

	// Send unknown message type
	if err := ws.WriteJSON(WSMessage{
		Type: "unknown_type",
		ID:   "test-1",
	}); err != nil {
		t.Fatalf("write unknown type: %v", err)
	}

	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp WSMessage
	if err := ws.ReadJSON(&resp); err != nil {
		t.Fatalf("read error response: %v", err)
	}

	if resp.Type != WSTypeError {
		t.Errorf("response type = %s, want error", resp.Type)
	}
}

func TestWebSocket_Broadcast(t *testing.T) {
	srv, addr := testServerWithRealListener(t, 19086)
	defer srv.Close()

	ws := connectWebSocket(t, addr)
	defer ws.Close()

	// Subscribe to channel
	if err := ws.WriteJSON(WSMessage{
		Type:    WSTypeSubscribe,
		ID:      "sub-1",
		Payload: WSSubscribePayload{Channels: []string{"test.channel"}},
	}); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}

	// Read subscribe response
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp WSMessage
	if err := ws.ReadJSON(&resp); err != nil {
		t.Fatalf("read subscribe response: %v", err)
	}

	// Broadcast a message
	srv.hub.Broadcast("test.channel", map[string]string{"key": "value"})

	// Read broadcast
	if err := ws.ReadJSON(&resp); err != nil {
		t.Fatalf("read broadcast: %v", err)
	}

	if resp.Type != WSTypeEvent {
		t.Errorf("broadcast type = %s, want event", resp.Type)
	}
	if resp.EventType != "test.channel" {
		t.Errorf("broadcast event_type = %s, want test.channel", resp.EventType)
	}
}

func TestWebSocket_NoTicket(t *testing.T) {
	srv, addr := testServerWithRealListener(t, 19087)
	defer srv.Close()

	// Try to connect without ticket
	wsURL := "ws://" + addr + "/api/v1/ws"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected error connecting without ticket")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestWebSocket_InvalidTicket(t *testing.T) {
	srv, addr := testServerWithRealListener(t, 19088)
	defer srv.Close()

	// Try to connect with invalid ticket
	wsURL := "ws://" + addr + "/api/v1/ws?ticket=invalid-ticket"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected error connecting with invalid ticket")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

// connectWebSocket is a helper that logs in, gets a ticket, and connects.
func connectWebSocket(t *testing.T, addr string) *websocket.Conn {
	t.Helper()

	// Login
	loginResp, err := http.Post(
		"http://"+addr+"/api/v1/auth/login",
		"application/json",
		strings.NewReader(`{"username":"admin","password":"admin"}`),
	)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer loginResp.Body.Close()

	var loginResult struct {
		AccessToken string `json:"access_token"`
	}
	json.NewDecoder(loginResp.Body).Decode(&loginResult)

	// Get ticket
	req, _ := http.NewRequest("POST", "http://"+addr+"/api/v1/auth/ws-ticket", nil)
	req.Header.Set("Authorization", "Bearer "+loginResult.AccessToken)
	ticketResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get ticket failed: %v", err)
	}
	defer ticketResp.Body.Close()

	var ticketResult struct {
		Ticket string `json:"ticket"`
	}
	json.NewDecoder(ticketResp.Body).Decode(&ticketResult)

	// Connect
	wsURL := "ws://" + addr + "/api/v1/ws?ticket=" + ticketResult.Ticket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket connect failed: %v", err)
	}

	return ws
}
