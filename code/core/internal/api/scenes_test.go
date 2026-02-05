package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/nerrad567/gray-logic-core/internal/automation"
	"github.com/nerrad567/gray-logic-core/internal/device"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/logging"
)

// ─── Mock Dependencies ─────────────────────────────────────────────

// mockDeviceRegistry implements automation.DeviceRegistry for engine tests.
type mockDeviceRegistry struct {
	devices map[string]automation.DeviceInfo
}

func (m *mockDeviceRegistry) GetDevice(_ context.Context, id string) (automation.DeviceInfo, error) {
	dev, ok := m.devices[id]
	if !ok {
		return automation.DeviceInfo{}, automation.ErrSceneNotFound
	}
	return dev, nil
}

// mockMQTTClient implements automation.MQTTClient for engine tests.
// Uses a mutex because the engine calls Publish from parallel goroutines.
type mockMQTTClient struct {
	mu        sync.Mutex
	published []mqttMessage
}

type mqttMessage struct {
	topic   string
	payload []byte
}

func (m *mockMQTTClient) Publish(topic string, payload []byte, _ byte, _ bool) error {
	m.mu.Lock()
	m.published = append(m.published, mqttMessage{topic: topic, payload: payload})
	m.mu.Unlock()
	return nil
}

// ─── Test Helpers ──────────────────────────────────────────────────

// testSceneServer creates a Server with a real scene registry backed by in-memory SQLite.
// It also creates an Engine with mock MQTT/Device dependencies for activation tests.
func testSceneServer(t *testing.T) (*Server, *automation.Registry, *mockMQTTClient) {
	t.Helper()

	db := setupSceneTestDB(t)
	repo := automation.NewSQLiteRepository(db)
	registry := automation.NewRegistry(repo)
	if err := registry.RefreshCache(context.Background()); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	log := logging.New(config.LoggingConfig{Level: "error", Format: "text", Output: "stdout"}, "test")

	mockMQTT := &mockMQTTClient{}
	mockDevices := &mockDeviceRegistry{
		devices: map[string]automation.DeviceInfo{
			"light-1":  {ID: "light-1", Protocol: "knx"},
			"light-2":  {ID: "light-2", Protocol: "knx"},
			"blind-1":  {ID: "blind-1", Protocol: "knx"},
			"dimmer-1": {ID: "dimmer-1", Protocol: "dali"},
		},
	}

	// Create hub for WebSocket broadcast
	hub := NewHub(config.WebSocketConfig{MaxMessageSize: 8192, PingInterval: 30, PongTimeout: 10}, log)
	go hub.Run(context.Background())

	engine := automation.NewEngine(registry, mockDevices, mockMQTT, hub, repo, nil)

	// Also need device registry for the base server
	deviceDB := setupTestDB(t)
	deviceRepo := device.NewSQLiteRepository(deviceDB)
	deviceRegistry := device.NewRegistry(deviceRepo)
	if err := deviceRegistry.RefreshCache(context.Background()); err != nil {
		t.Fatalf("device RefreshCache: %v", err)
	}

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
				Secret:         string(testJWTSecret),
				AccessTokenTTL: 15,
			},
		},
		Logger:        log,
		Registry:      deviceRegistry,
		MQTT:          nil,
		SceneEngine:   engine,
		SceneRegistry: registry,
		SceneRepo:     repo,
		ExternalHub:   hub,
		Version:       "test",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	return srv, registry, mockMQTT
}

// setupSceneTestDB creates an in-memory SQLite database with the scenes schema.
func setupSceneTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE scenes (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			description TEXT,
			room_id TEXT,
			area_id TEXT,
			enabled INTEGER NOT NULL DEFAULT 1,
			priority INTEGER NOT NULL DEFAULT 50,
			icon TEXT,
			colour TEXT,
			category TEXT,
			actions TEXT NOT NULL DEFAULT '[]',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		) STRICT;
		CREATE UNIQUE INDEX idx_scenes_slug ON scenes(slug);

		CREATE TABLE scene_executions (
			id TEXT PRIMARY KEY,
			scene_id TEXT NOT NULL,
			triggered_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			started_at TEXT,
			completed_at TEXT,
			trigger_type TEXT NOT NULL DEFAULT 'manual',
			trigger_source TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			actions_total INTEGER NOT NULL DEFAULT 0,
			actions_completed INTEGER NOT NULL DEFAULT 0,
			actions_failed INTEGER NOT NULL DEFAULT 0,
			actions_skipped INTEGER NOT NULL DEFAULT 0,
			failures TEXT,
			duration_ms INTEGER,
			FOREIGN KEY (scene_id) REFERENCES scenes(id) ON DELETE CASCADE
		) STRICT;
		CREATE INDEX idx_scene_executions_scene_id ON scene_executions(scene_id);
	`

	if _, execErr := db.Exec(schema); execErr != nil {
		db.Close()
		t.Fatalf("failed to create test schema: %v", execErr)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

// ─── Scene CRUD Tests ──────────────────────────────────────────────

func TestListScenes_Empty(t *testing.T) {
	srv, _, _ := testSceneServer(t)
	router := srv.buildRouter()

	req := authReq(t, httptest.NewRequest(http.MethodGet, "/api/v1/scenes", nil))
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

func TestCreateAndGetScene(t *testing.T) {
	srv, _, _ := testSceneServer(t)
	router := srv.buildRouter()

	body := `{
		"name": "Movie Night",
		"description": "Dim lights for movie watching",
		"category": "entertainment",
		"actions": [
			{"device_id": "light-1", "command": "dim", "parameters": {"level": 20}, "continue_on_error": true},
			{"device_id": "blind-1", "command": "close", "parallel": true, "continue_on_error": true}
		]
	}`

	req := authReq(t, httptest.NewRequest(http.MethodPost, "/api/v1/scenes", strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var created automation.Scene
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created: %v", err)
	}

	if created.ID == "" {
		t.Error("expected scene ID to be auto-generated")
	}
	if created.Slug == "" {
		t.Error("expected slug to be auto-generated")
	}
	if created.Slug != "movie-night" {
		t.Errorf("slug = %q, want %q", created.Slug, "movie-night")
	}
	if len(created.Actions) != 2 {
		t.Errorf("actions count = %d, want 2", len(created.Actions))
	}

	// Get scene by ID
	req = authReq(t, httptest.NewRequest(http.MethodGet, "/api/v1/scenes/"+created.ID, nil))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", w.Code, http.StatusOK)
	}

	var got automation.Scene
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}

	if got.Name != "Movie Night" {
		t.Errorf("name = %q, want %q", got.Name, "Movie Night")
	}
	if got.Category != automation.CategoryEntertainment {
		t.Errorf("category = %q, want %q", got.Category, automation.CategoryEntertainment)
	}
}

func TestGetScene_NotFound(t *testing.T) {
	srv, _, _ := testSceneServer(t)
	router := srv.buildRouter()

	req := authReq(t, httptest.NewRequest(http.MethodGet, "/api/v1/scenes/nonexistent-id", nil))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestCreateScene_InvalidJSON(t *testing.T) {
	srv, _, _ := testSceneServer(t)
	router := srv.buildRouter()

	req := authReq(t, httptest.NewRequest(http.MethodPost, "/api/v1/scenes", strings.NewReader("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateScene_NoActions(t *testing.T) {
	srv, _, _ := testSceneServer(t)
	router := srv.buildRouter()

	body := `{"name": "Empty Scene", "actions": []}`
	req := authReq(t, httptest.NewRequest(http.MethodPost, "/api/v1/scenes", strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestCreateScene_NoName(t *testing.T) {
	srv, _, _ := testSceneServer(t)
	router := srv.buildRouter()

	body := `{"actions": [{"device_id": "light-1", "command": "on", "continue_on_error": true}]}`
	req := authReq(t, httptest.NewRequest(http.MethodPost, "/api/v1/scenes", strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestUpdateScene(t *testing.T) {
	srv, registry, _ := testSceneServer(t)
	router := srv.buildRouter()

	scene := &automation.Scene{
		Name:    "Original",
		Enabled: true,
		Actions: []automation.SceneAction{
			{DeviceID: "light-1", Command: "on", ContinueOnError: true},
		},
	}
	if err := registry.CreateScene(context.Background(), scene); err != nil {
		t.Fatalf("CreateScene: %v", err)
	}

	body := `{"name": "Updated Name"}`
	req := authReq(t, httptest.NewRequest(http.MethodPatch, "/api/v1/scenes/"+scene.ID, strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var updated automation.Scene
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("name = %q, want %q", updated.Name, "Updated Name")
	}
}

func TestUpdateScene_NotFound(t *testing.T) {
	srv, _, _ := testSceneServer(t)
	router := srv.buildRouter()

	body := `{"name": "Nope"}`
	req := authReq(t, httptest.NewRequest(http.MethodPatch, "/api/v1/scenes/nonexistent", strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteScene(t *testing.T) {
	srv, registry, _ := testSceneServer(t)
	router := srv.buildRouter()

	scene := &automation.Scene{
		Name:    "ToDelete",
		Enabled: true,
		Actions: []automation.SceneAction{
			{DeviceID: "light-1", Command: "off", ContinueOnError: true},
		},
	}
	if err := registry.CreateScene(context.Background(), scene); err != nil {
		t.Fatalf("CreateScene: %v", err)
	}

	req := authReq(t, httptest.NewRequest(http.MethodDelete, "/api/v1/scenes/"+scene.ID, nil))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Confirm gone
	req = authReq(t, httptest.NewRequest(http.MethodGet, "/api/v1/scenes/"+scene.ID, nil))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("get after delete status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteScene_NotFound(t *testing.T) {
	srv, _, _ := testSceneServer(t)
	router := srv.buildRouter()

	req := authReq(t, httptest.NewRequest(http.MethodDelete, "/api/v1/scenes/nonexistent", nil))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ─── Scene Filtering Tests ─────────────────────────────────────────

func TestListScenes_FilterByCategory(t *testing.T) {
	srv, registry, _ := testSceneServer(t)
	router := srv.buildRouter()

	scene := &automation.Scene{
		Name:     "Comfort Scene",
		Enabled:  true,
		Category: automation.CategoryComfort,
		Actions: []automation.SceneAction{
			{DeviceID: "light-1", Command: "on", ContinueOnError: true},
		},
	}
	if err := registry.CreateScene(context.Background(), scene); err != nil {
		t.Fatalf("CreateScene: %v", err)
	}

	// Filter by category=comfort
	req := authReq(t, httptest.NewRequest(http.MethodGet, "/api/v1/scenes?category=comfort", nil))
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

	// Filter by category=entertainment (should be empty)
	req = authReq(t, httptest.NewRequest(http.MethodGet, "/api/v1/scenes?category=entertainment", nil))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if int(resp["count"].(float64)) != 0 {
		t.Errorf("count for entertainment = %v, want 0", resp["count"])
	}
}

func TestListScenes_FilterByRoom(t *testing.T) {
	srv, registry, _ := testSceneServer(t)
	router := srv.buildRouter()

	roomID := "room-living"
	scene := &automation.Scene{
		Name:    "Living Room Scene",
		Enabled: true,
		RoomID:  &roomID,
		Actions: []automation.SceneAction{
			{DeviceID: "light-1", Command: "on", ContinueOnError: true},
		},
	}
	if err := registry.CreateScene(context.Background(), scene); err != nil {
		t.Fatalf("CreateScene: %v", err)
	}

	req := authReq(t, httptest.NewRequest(http.MethodGet, "/api/v1/scenes?room_id=room-living", nil))
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
}

// ─── Scene Activation Tests ────────────────────────────────────────

func TestActivateScene_Success(t *testing.T) {
	srv, registry, mockMQTT := testSceneServer(t)
	router := srv.buildRouter()

	scene := &automation.Scene{
		Name:    "Activate Me",
		Enabled: true,
		Actions: []automation.SceneAction{
			{DeviceID: "light-1", Command: "on", ContinueOnError: true},
			{DeviceID: "light-2", Command: "dim", Parameters: map[string]any{"level": float64(50)}, Parallel: true, ContinueOnError: true},
		},
	}
	if err := registry.CreateScene(context.Background(), scene); err != nil {
		t.Fatalf("CreateScene: %v", err)
	}

	body := `{"trigger_type": "manual", "trigger_source": "api"}`
	req := authReq(t, httptest.NewRequest(http.MethodPost, "/api/v1/scenes/"+scene.ID+"/activate", strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("activate status = %d, want %d; body: %s", w.Code, http.StatusAccepted, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp["execution_id"] == "" {
		t.Error("expected execution_id to be non-empty")
	}
	if resp["status"] != "accepted" {
		t.Errorf("status = %v, want accepted", resp["status"])
	}

	// Verify MQTT commands were published
	if len(mockMQTT.published) != 2 {
		t.Errorf("published messages = %d, want 2", len(mockMQTT.published))
	}
}

func TestActivateScene_DefaultTrigger(t *testing.T) {
	srv, registry, _ := testSceneServer(t)
	router := srv.buildRouter()

	scene := &automation.Scene{
		Name:    "Default Trigger",
		Enabled: true,
		Actions: []automation.SceneAction{
			{DeviceID: "light-1", Command: "on", ContinueOnError: true},
		},
	}
	if err := registry.CreateScene(context.Background(), scene); err != nil {
		t.Fatalf("CreateScene: %v", err)
	}

	// Empty body — should default to manual/api
	req := authReq(t, httptest.NewRequest(http.MethodPost, "/api/v1/scenes/"+scene.ID+"/activate", nil))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("activate status = %d, want %d; body: %s", w.Code, http.StatusAccepted, w.Body.String())
	}
}

func TestActivateScene_NotFound(t *testing.T) {
	srv, _, _ := testSceneServer(t)
	router := srv.buildRouter()

	req := authReq(t, httptest.NewRequest(http.MethodPost, "/api/v1/scenes/nonexistent/activate", nil))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestActivateScene_Disabled(t *testing.T) {
	srv, registry, _ := testSceneServer(t)
	router := srv.buildRouter()

	scene := &automation.Scene{
		Name:    "Disabled Scene",
		Enabled: false,
		Actions: []automation.SceneAction{
			{DeviceID: "light-1", Command: "on", ContinueOnError: true},
		},
	}
	if err := registry.CreateScene(context.Background(), scene); err != nil {
		t.Fatalf("CreateScene: %v", err)
	}

	req := authReq(t, httptest.NewRequest(http.MethodPost, "/api/v1/scenes/"+scene.ID+"/activate", nil))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

// ─── Scene Executions Tests ────────────────────────────────────────

func TestListSceneExecutions(t *testing.T) {
	srv, registry, _ := testSceneServer(t)
	router := srv.buildRouter()

	scene := &automation.Scene{
		Name:    "Execution History",
		Enabled: true,
		Actions: []automation.SceneAction{
			{DeviceID: "light-1", Command: "on", ContinueOnError: true},
		},
	}
	if err := registry.CreateScene(context.Background(), scene); err != nil {
		t.Fatalf("CreateScene: %v", err)
	}

	// Activate scene to create an execution record
	req := authReq(t, httptest.NewRequest(http.MethodPost, "/api/v1/scenes/"+scene.ID+"/activate", nil))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("activate status = %d, want %d", w.Code, http.StatusAccepted)
	}

	// List executions
	req = authReq(t, httptest.NewRequest(http.MethodGet, "/api/v1/scenes/"+scene.ID+"/executions", nil))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list executions status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if int(resp["count"].(float64)) != 1 {
		t.Errorf("execution count = %v, want 1", resp["count"])
	}
}

func TestListSceneExecutions_SceneNotFound(t *testing.T) {
	srv, _, _ := testSceneServer(t)
	router := srv.buildRouter()

	req := authReq(t, httptest.NewRequest(http.MethodGet, "/api/v1/scenes/nonexistent/executions", nil))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
