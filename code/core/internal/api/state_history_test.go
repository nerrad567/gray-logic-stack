package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"

	"github.com/nerrad567/gray-logic-core/internal/device"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/logging"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/tsdb"
)

// setupStateHistoryAPIDB creates an in-memory SQLite database with device and state history tables.
func setupStateHistoryAPIDB(t *testing.T) *sql.DB {
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

		CREATE TABLE state_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL,
			state TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT 'mqtt',
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		) STRICT;
		CREATE INDEX idx_state_history_device ON state_history(device_id, created_at DESC);
		CREATE INDEX idx_state_history_time ON state_history(created_at DESC);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		t.Fatalf("failed to create test schema: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// setupStateHistoryServer builds a server with registry and history repository.
func setupStateHistoryServer(t *testing.T, tsdbClient *tsdb.Client) (*Server, *device.SQLiteStateHistoryRepository, string) {
	t.Helper()

	db := setupStateHistoryAPIDB(t)
	deviceRepo := device.NewSQLiteRepository(db)
	registry := device.NewRegistry(deviceRepo)
	log := logging.New(config.LoggingConfig{Level: "error", Format: "text", Output: "stdout"}, "test")
	registry.SetLogger(log)

	if err := registry.RefreshCache(context.Background()); err != nil {
		t.Fatalf("RefreshCache error: %v", err)
	}

	dev := &device.Device{
		ID:       "dev-1",
		Name:     "Test Device",
		Slug:     "test-device",
		Type:     device.DeviceTypeLightDimmer,
		Domain:   device.DomainLighting,
		Protocol: device.ProtocolKNX,
		Address: device.Address{"functions": map[string]any{
			"switch": map[string]any{"ga": "1/2/3", "dpt": "1.001", "flags": []any{"write"}},
		}},
		Capabilities: []device.Capability{device.CapOnOff},
		Config:       device.Config{},
		State:        device.State{},
		HealthStatus: device.HealthStatusUnknown,
	}
	if err := registry.CreateDevice(context.Background(), dev); err != nil {
		t.Fatalf("CreateDevice error: %v", err)
	}

	stateHistoryRepo := device.NewSQLiteStateHistoryRepository(db)

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
		StateHistory: stateHistoryRepo,
		TSDB:         tsdbClient,
		Version:      "test",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	return srv, stateHistoryRepo, dev.ID
}

// newRequestWithDeviceID creates a request with chi URL params set.
func newRequestWithDeviceID(method, target, deviceID string) *http.Request { //nolint:unparam // method kept for test helper flexibility
	req := httptest.NewRequest(method, target, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", deviceID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// setupTSDBClient creates a TSDB client backed by a test HTTP server.
func setupTSDBClient(t *testing.T, rangeHandler, instantHandler http.HandlerFunc) (*tsdb.Client, func()) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if rangeHandler != nil {
		mux.HandleFunc("/api/v1/query_range", rangeHandler)
	}
	if instantHandler != nil {
		mux.HandleFunc("/api/v1/query", instantHandler)
	}

	server := httptest.NewServer(mux)
	cfg := config.TSDBConfig{
		Enabled:       true,
		URL:           server.URL,
		BatchSize:     1,
		FlushInterval: 1,
	}

	client, err := tsdb.Connect(context.Background(), cfg)
	if err != nil {
		server.Close()
		t.Fatalf("Connect error: %v", err)
	}

	cleanup := func() {
		client.Close()
		server.Close()
	}

	return client, cleanup
}

// TestHandleGetDeviceHistory verifies history retrieval and response shape.
func TestHandleGetDeviceHistory(t *testing.T) {
	srv, repo, deviceID := setupStateHistoryServer(t, nil)

	if err := repo.RecordStateChange(context.Background(), deviceID, device.State{"on": true}, device.StateHistorySourceMQTT); err != nil {
		t.Fatalf("RecordStateChange error: %v", err)
	}
	if err := repo.RecordStateChange(context.Background(), deviceID, device.State{"on": false}, device.StateHistorySourceCommand); err != nil {
		t.Fatalf("RecordStateChange error: %v", err)
	}

	req := newRequestWithDeviceID(http.MethodGet, "/api/v1/devices/"+deviceID+"/history?limit=1", deviceID)
	rr := httptest.NewRecorder()

	srv.handleGetDeviceHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp struct {
		DeviceID string                     `json:"device_id"`
		History  []device.StateHistoryEntry `json:"history"`
		Count    int                        `json:"count"`
	}

	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.DeviceID != deviceID {
		t.Errorf("device_id = %q, want %q", resp.DeviceID, deviceID)
	}
	if resp.Count != 1 {
		t.Errorf("count = %d, want 1", resp.Count)
	}
	if len(resp.History) != 1 {
		t.Errorf("history length = %d, want 1", len(resp.History))
	}
}

// TestHandleGetDeviceHistory_InvalidParams verifies invalid query handling.
func TestHandleGetDeviceHistory_InvalidParams(t *testing.T) {
	srv, _, deviceID := setupStateHistoryServer(t, nil)

	tests := []struct {
		name  string
		query string
	}{
		{name: "limit too high", query: "?limit=201"},
		{name: "invalid since", query: "?since=not-a-time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequestWithDeviceID(http.MethodGet, "/api/v1/devices/"+deviceID+"/history"+tt.query, deviceID)
			rr := httptest.NewRecorder()

			srv.handleGetDeviceHistory(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
			}
		})
	}
}

// TestHandleGetDeviceMetrics verifies PromQL query construction and proxying.
func TestHandleGetDeviceMetrics(t *testing.T) {
	var gotQuery string
	var gotStart string
	var gotEnd string
	var gotStep string

	rangeHandler := func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("query")
		gotStart = r.URL.Query().Get("start")
		gotEnd = r.URL.Query().Get("end")
		gotStep = r.URL.Query().Get("step")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":{"result":[]}}`))
	}

	client, cleanup := setupTSDBClient(t, rangeHandler, nil)
	defer cleanup()

	srv, _, deviceID := setupStateHistoryServer(t, client)

	start := time.Date(2026, 2, 5, 8, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 5, 9, 0, 0, 0, time.UTC)
	step := 30 * time.Second

	query := "/api/v1/devices/" + deviceID + "/metrics?field=level" +
		"&start=" + start.Format(time.RFC3339) +
		"&end=" + end.Format(time.RFC3339) +
		"&step=" + step.String()

	req := newRequestWithDeviceID(http.MethodGet, query, deviceID)
	rr := httptest.NewRecorder()

	srv.handleGetDeviceMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	expectedQuery := "device_metrics{device_id=\"" + deviceID + "\",measurement=\"level\"}"
	if gotQuery != expectedQuery {
		t.Errorf("query = %q, want %q", gotQuery, expectedQuery)
	}

	expectedStart := strconv.FormatFloat(float64(start.UnixNano())/float64(time.Second), 'f', -1, 64)
	expectedEnd := strconv.FormatFloat(float64(end.UnixNano())/float64(time.Second), 'f', -1, 64)
	expectedStep := strconv.FormatFloat(step.Seconds(), 'f', -1, 64)

	if gotStart != expectedStart {
		t.Errorf("start = %q, want %q", gotStart, expectedStart)
	}
	if gotEnd != expectedEnd {
		t.Errorf("end = %q, want %q", gotEnd, expectedEnd)
	}
	if gotStep != expectedStep {
		t.Errorf("step = %q, want %q", gotStep, expectedStep)
	}
}

// TestHandleGetDeviceMetrics_TSDBDisabled verifies 503 when TSDB is unavailable.
func TestHandleGetDeviceMetrics_TSDBDisabled(t *testing.T) {
	srv, _, deviceID := setupStateHistoryServer(t, nil)

	req := newRequestWithDeviceID(http.MethodGet, "/api/v1/devices/"+deviceID+"/metrics?field=level", deviceID)
	rr := httptest.NewRecorder()

	srv.handleGetDeviceMetrics(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

// TestHandleGetDeviceMetricsSummary verifies summary parsing and response.
func TestHandleGetDeviceMetricsSummary(t *testing.T) {
	instantHandler := func(w http.ResponseWriter, _ *http.Request) {
		timestamp := time.Date(2026, 2, 5, 10, 30, 45, 0, time.UTC).Unix()
		payload := `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{"metric": {"device_id": "dev-1", "measurement": "on"}, "value": [` + strconv.FormatInt(timestamp, 10) + `, "1"]},
					{"metric": {"device_id": "dev-1", "measurement": "level"}, "value": [` + strconv.FormatInt(timestamp, 10) + `, "75"]}
				]
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}

	client, cleanup := setupTSDBClient(t, nil, instantHandler)
	defer cleanup()

	srv, _, deviceID := setupStateHistoryServer(t, client)

	req := newRequestWithDeviceID(http.MethodGet, "/api/v1/devices/"+deviceID+"/metrics/summary", deviceID)
	rr := httptest.NewRecorder()

	srv.handleGetDeviceMetricsSummary(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp struct {
		DeviceID string                        `json:"device_id"`
		Fields   []string                      `json:"fields"`
		Latest   map[string]metricsLatestEntry `json:"latest"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.DeviceID != deviceID {
		t.Errorf("device_id = %q, want %q", resp.DeviceID, deviceID)
	}
	if len(resp.Fields) != 2 {
		t.Fatalf("fields length = %d, want 2", len(resp.Fields))
	}
	if resp.Fields[0] != "level" || resp.Fields[1] != "on" {
		t.Errorf("fields = %v, want [level on]", resp.Fields)
	}

	if latest, ok := resp.Latest["level"]; !ok || latest.Value != 75 {
		t.Errorf("latest level = %v, want 75", resp.Latest["level"])
	}
	if latest, ok := resp.Latest["on"]; !ok || latest.Value != 1 {
		t.Errorf("latest on = %v, want 1", resp.Latest["on"])
	}
}
