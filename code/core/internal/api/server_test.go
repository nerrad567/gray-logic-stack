package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/nerrad567/gray-logic-core/internal/device"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/logging"
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
		"address": {"group_address": "1/2/3"},
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
		Name:         "Original",
		Type:         device.DeviceTypeLightDimmer,
		Domain:       device.DomainLighting,
		Protocol:     device.ProtocolKNX,
		Address:      device.Address{"group_address": "1/2/3"},
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
		Name:         "ToDelete",
		Type:         device.DeviceTypeLightSwitch,
		Domain:       device.DomainLighting,
		Protocol:     device.ProtocolKNX,
		Address:      device.Address{"group_address": "1/2/4"},
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
		Name:         "Kitchen Light",
		Type:         device.DeviceTypeLightDimmer,
		Domain:       device.DomainLighting,
		Protocol:     device.ProtocolKNX,
		Address:      device.Address{"group_address": "1/2/5"},
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
		Name:         "Stateful Light",
		Type:         device.DeviceTypeLightDimmer,
		Domain:       device.DomainLighting,
		Protocol:     device.ProtocolKNX,
		Address:      device.Address{"group_address": "1/2/6"},
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
		Name:         "No Command",
		Type:         device.DeviceTypeLightSwitch,
		Domain:       device.DomainLighting,
		Protocol:     device.ProtocolKNX,
		Address:      device.Address{"group_address": "1/2/9"},
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
		Name:         "Stats Device",
		Type:         device.DeviceTypeLightSwitch,
		Domain:       device.DomainLighting,
		Protocol:     device.ProtocolKNX,
		Address:      device.Address{"group_address": "1/2/7"},
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
