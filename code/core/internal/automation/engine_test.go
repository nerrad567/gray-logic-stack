package automation

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

// ─── Mock Dependencies ──────────────────────────────────────────────────────

// mockDeviceRegistry returns device info for known device IDs.
type mockDeviceRegistry struct {
	devices map[string]DeviceInfo
	mu      sync.RWMutex
}

func newMockDeviceRegistry() *mockDeviceRegistry {
	return &mockDeviceRegistry{
		devices: map[string]DeviceInfo{
			"light-01":   {ID: "light-01", Protocol: "knx"},
			"light-02":   {ID: "light-02", Protocol: "knx"},
			"light-03":   {ID: "light-03", Protocol: "knx"},
			"blind-01":   {ID: "blind-01", Protocol: "knx"},
			"audio-01":   {ID: "audio-01", Protocol: "modbus"},
			"thermostat": {ID: "thermostat", Protocol: "modbus"},
		},
	}
}

func (m *mockDeviceRegistry) GetDevice(_ context.Context, id string) (DeviceInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	dev, ok := m.devices[id]
	if !ok {
		return DeviceInfo{}, errors.New("device: not found")
	}
	return dev, nil
}

// mockMQTT captures all published messages.
type mockMQTT struct {
	messages []mqttMessage
	mu       sync.Mutex
	failOn   string // Topic to fail on (for error testing)
}

type mqttMessage struct {
	Topic    string
	Payload  map[string]any
	QoS      byte
	Retained bool
}

func newMockMQTT() *mockMQTT {
	return &mockMQTT{}
}

func (m *mockMQTT) Publish(topic string, payload []byte, qos byte, retained bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failOn != "" && topic == m.failOn {
		return errors.New("MQTT publish failed")
	}

	var parsed map[string]any
	_ = json.Unmarshal(payload, &parsed)

	m.messages = append(m.messages, mqttMessage{
		Topic:    topic,
		Payload:  parsed,
		QoS:      qos,
		Retained: retained,
	})
	return nil
}

func (m *mockMQTT) getMessages() []mqttMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	cpy := make([]mqttMessage, len(m.messages))
	copy(cpy, m.messages)
	return cpy
}

// mockWSHub captures all broadcasts.
type mockWSHub struct {
	broadcasts []wsBroadcast
	mu         sync.Mutex
}

type wsBroadcast struct {
	Channel string
	Payload any
}

func newMockWSHub() *mockWSHub {
	return &mockWSHub{}
}

func (m *mockWSHub) Broadcast(channel string, payload any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcasts = append(m.broadcasts, wsBroadcast{Channel: channel, Payload: payload})
}

func (m *mockWSHub) getBroadcasts() []wsBroadcast {
	m.mu.Lock()
	defer m.mu.Unlock()
	cpy := make([]wsBroadcast, len(m.broadcasts))
	copy(cpy, m.broadcasts)
	return cpy
}

// ─── Helper ─────────────────────────────────────────────────────────────────

func setupEngine(t *testing.T) (*Engine, *mockMQTT, *mockWSHub, *mockRepository) {
	t.Helper()

	repo := newMockRepository()
	registry := NewRegistry(repo)
	devices := newMockDeviceRegistry()
	mqtt := newMockMQTT()
	hub := newMockWSHub()

	engine := NewEngine(registry, devices, mqtt, hub, repo, noopLogger{})
	return engine, mqtt, hub, repo
}

func createTestScene(repo *mockRepository, registry *Registry, id, name string, actions []SceneAction) {
	scene := &Scene{
		ID:       id,
		Name:     name,
		Slug:     GenerateSlug(name),
		Enabled:  true,
		Priority: 50,
		Actions:  actions,
	}
	repo.scenes[id] = scene
	_ = registry.RefreshCache(context.Background())
}

// ─── Tests ──────────────────────────────────────────────────────────────────

func TestEngine_ActivateScene_Success(t *testing.T) {
	engine, mqtt, hub, repo := setupEngine(t)
	ctx := context.Background()

	createTestScene(repo, engine.registry, "cinema", "Cinema Mode", []SceneAction{
		{DeviceID: "light-01", Command: "set", Parameters: map[string]any{"on": false}, ContinueOnError: true},
		{DeviceID: "blind-01", Command: "position", Parameters: map[string]any{"position": float64(0)}, Parallel: true, ContinueOnError: true},
	})

	execID, err := engine.ActivateScene(ctx, "cinema", "manual", "api")
	if err != nil {
		t.Fatalf("ActivateScene: %v", err)
	}
	if execID == "" {
		t.Error("execution ID is empty")
	}

	// Check MQTT messages
	msgs := mqtt.getMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 MQTT messages, got %d", len(msgs))
	}

	// Verify topics
	topics := make(map[string]bool)
	for _, msg := range msgs {
		topics[msg.Topic] = true
		if msg.QoS != 1 {
			t.Errorf("QoS = %d, want 1", msg.QoS)
		}
		if msg.Retained {
			t.Error("command should not be retained")
		}
		// Verify payload structure
		if msg.Payload["source"] != "scene:cinema" {
			t.Errorf("source = %v, want %q", msg.Payload["source"], "scene:cinema")
		}
	}
	if !topics["graylogic/command/knx/light-01"] {
		t.Error("missing MQTT message for light-01")
	}
	if !topics["graylogic/command/knx/blind-01"] {
		t.Error("missing MQTT message for blind-01")
	}

	// Check WebSocket broadcast
	broadcasts := hub.getBroadcasts()
	if len(broadcasts) != 1 {
		t.Fatalf("expected 1 broadcast, got %d", len(broadcasts))
	}
	if broadcasts[0].Channel != "scene.activated" {
		t.Errorf("channel = %q, want %q", broadcasts[0].Channel, "scene.activated")
	}
	bPayload, ok := broadcasts[0].Payload.(map[string]any)
	if !ok {
		t.Fatal("broadcast payload is not map[string]any")
	}
	if bPayload["scene_id"] != "cinema" {
		t.Errorf("broadcast scene_id = %v, want %q", bPayload["scene_id"], "cinema")
	}
	if bPayload["status"] != "completed" {
		t.Errorf("broadcast status = %v, want %q", bPayload["status"], "completed")
	}

	// Check execution record
	exec, execErr := repo.GetExecution(ctx, execID)
	if execErr != nil {
		t.Fatalf("GetExecution: %v", execErr)
	}
	if exec.Status != StatusCompleted {
		t.Errorf("execution status = %q, want %q", exec.Status, StatusCompleted)
	}
	if exec.ActionsCompleted != 2 {
		t.Errorf("ActionsCompleted = %d, want 2", exec.ActionsCompleted)
	}
	if exec.ActionsFailed != 0 {
		t.Errorf("ActionsFailed = %d, want 0", exec.ActionsFailed)
	}
}

func TestEngine_ActivateScene_NotFound(t *testing.T) {
	engine, _, _, _ := setupEngine(t)
	ctx := context.Background()

	_, err := engine.ActivateScene(ctx, "nonexistent", "manual", "api")
	if !errors.Is(err, ErrSceneNotFound) {
		t.Errorf("expected ErrSceneNotFound, got: %v", err)
	}
}

func TestEngine_ActivateScene_Disabled(t *testing.T) {
	engine, _, _, repo := setupEngine(t)
	ctx := context.Background()

	scene := &Scene{
		ID:       "disabled",
		Name:     "Disabled Scene",
		Slug:     "disabled-scene",
		Enabled:  false,
		Priority: 50,
		Actions:  []SceneAction{{DeviceID: "light-01", Command: "set", ContinueOnError: true}},
	}
	repo.scenes["disabled"] = scene
	_ = engine.registry.RefreshCache(ctx)

	_, err := engine.ActivateScene(ctx, "disabled", "manual", "api")
	if !errors.Is(err, ErrSceneDisabled) {
		t.Errorf("expected ErrSceneDisabled, got: %v", err)
	}
}

func TestEngine_ActivateScene_MQTTUnavailable(t *testing.T) {
	repo := newMockRepository()
	registry := NewRegistry(repo)
	devices := newMockDeviceRegistry()

	// Engine with nil MQTT
	engine := NewEngine(registry, devices, nil, nil, repo, noopLogger{})
	ctx := context.Background()

	createTestScene(repo, registry, "test", "Test", []SceneAction{
		{DeviceID: "light-01", Command: "set", ContinueOnError: true},
	})

	_, err := engine.ActivateScene(ctx, "test", "manual", "api")
	if !errors.Is(err, ErrMQTTUnavailable) {
		t.Errorf("expected ErrMQTTUnavailable, got: %v", err)
	}
}

func TestEngine_ActivateScene_Parallel(t *testing.T) {
	engine, mqtt, _, repo := setupEngine(t)
	ctx := context.Background()

	// All actions parallel — should execute concurrently
	createTestScene(repo, engine.registry, "parallel", "Parallel Test", []SceneAction{
		{DeviceID: "light-01", Command: "set", ContinueOnError: true},
		{DeviceID: "light-02", Command: "set", Parallel: true, ContinueOnError: true},
		{DeviceID: "light-03", Command: "set", Parallel: true, ContinueOnError: true},
	})

	start := time.Now()
	_, err := engine.ActivateScene(ctx, "parallel", "manual", "api")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ActivateScene: %v", err)
	}

	// All 3 should have been published
	msgs := mqtt.getMessages()
	if len(msgs) != 3 {
		t.Errorf("expected 3 MQTT messages, got %d", len(msgs))
	}

	// Parallel execution should be fast (well under 100ms without delays)
	if elapsed > 100*time.Millisecond {
		t.Errorf("parallel execution took %v, expected < 100ms", elapsed)
	}
}

func TestEngine_ActivateScene_Sequential(t *testing.T) {
	engine, mqtt, _, repo := setupEngine(t)
	ctx := context.Background()

	// All actions sequential (parallel=false)
	createTestScene(repo, engine.registry, "sequential", "Sequential Test", []SceneAction{
		{DeviceID: "light-01", Command: "set", ContinueOnError: true},
		{DeviceID: "light-02", Command: "set", ContinueOnError: true},                 // parallel=false → new group
		{DeviceID: "light-03", Command: "set", Parallel: true, ContinueOnError: true}, // joins group 2
	})

	_, err := engine.ActivateScene(ctx, "sequential", "manual", "api")
	if err != nil {
		t.Fatalf("ActivateScene: %v", err)
	}

	msgs := mqtt.getMessages()
	if len(msgs) != 3 {
		t.Errorf("expected 3 MQTT messages, got %d", len(msgs))
	}
}

func TestEngine_ActivateScene_Delay(t *testing.T) {
	engine, mqtt, _, repo := setupEngine(t)
	ctx := context.Background()

	createTestScene(repo, engine.registry, "delay", "Delay Test", []SceneAction{
		{DeviceID: "light-01", Command: "set", DelayMS: 50, ContinueOnError: true},
	})

	start := time.Now()
	_, err := engine.ActivateScene(ctx, "delay", "manual", "api")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ActivateScene: %v", err)
	}

	// Should have taken at least 50ms
	if elapsed < 40*time.Millisecond {
		t.Errorf("expected delay of ~50ms, took %v", elapsed)
	}

	msgs := mqtt.getMessages()
	if len(msgs) != 1 {
		t.Errorf("expected 1 MQTT message, got %d", len(msgs))
	}
}

func TestEngine_ActivateScene_FadeMS(t *testing.T) {
	engine, mqtt, _, repo := setupEngine(t)
	ctx := context.Background()

	createTestScene(repo, engine.registry, "fade", "Fade Test", []SceneAction{
		{DeviceID: "light-01", Command: "dim", Parameters: map[string]any{"brightness": float64(30)}, FadeMS: 3000, ContinueOnError: true},
	})

	_, err := engine.ActivateScene(ctx, "fade", "manual", "api")
	if err != nil {
		t.Fatalf("ActivateScene: %v", err)
	}

	msgs := mqtt.getMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 MQTT message, got %d", len(msgs))
	}

	// fade_ms should be added to parameters
	params, ok := msgs[0].Payload["parameters"].(map[string]any)
	if !ok {
		t.Fatal("parameters is not map[string]any")
	}
	fadeMS, ok := params["fade_ms"]
	if !ok {
		t.Fatal("fade_ms not found in parameters")
	}
	if fadeMS != float64(3000) {
		t.Errorf("fade_ms = %v, want 3000", fadeMS)
	}
	// Original brightness should still be present
	if params["brightness"] != float64(30) {
		t.Errorf("brightness = %v, want 30", params["brightness"])
	}
}

func TestEngine_ActivateScene_ContinueOnError(t *testing.T) {
	engine, mqtt, _, repo := setupEngine(t)
	ctx := context.Background()

	// Second action targets nonexistent device but has ContinueOnError=true
	createTestScene(repo, engine.registry, "continue", "Continue Test", []SceneAction{
		{DeviceID: "light-01", Command: "set", ContinueOnError: true},
		{DeviceID: "nonexistent-device", Command: "set", ContinueOnError: true},
		{DeviceID: "light-02", Command: "set", ContinueOnError: true},
	})

	execID, err := engine.ActivateScene(ctx, "continue", "manual", "api")
	if err != nil {
		t.Fatalf("ActivateScene: %v", err)
	}

	// Should have published 2 successful commands (light-01 and light-02)
	msgs := mqtt.getMessages()
	if len(msgs) != 2 {
		t.Errorf("expected 2 MQTT messages, got %d", len(msgs))
	}

	// Execution should be "partial"
	exec, _ := repo.GetExecution(ctx, execID)
	if exec.Status != StatusPartial {
		t.Errorf("status = %q, want %q", exec.Status, StatusPartial)
	}
	if exec.ActionsFailed != 1 {
		t.Errorf("ActionsFailed = %d, want 1", exec.ActionsFailed)
	}
	if exec.ActionsCompleted != 2 {
		t.Errorf("ActionsCompleted = %d, want 2", exec.ActionsCompleted)
	}
}

func TestEngine_ActivateScene_AbortOnError(t *testing.T) {
	engine, mqtt, _, repo := setupEngine(t)
	ctx := context.Background()

	// Second action has ContinueOnError=false and targets nonexistent device
	createTestScene(repo, engine.registry, "abort", "Abort Test", []SceneAction{
		{DeviceID: "light-01", Command: "set", ContinueOnError: true},
		{DeviceID: "nonexistent-device", Command: "set", ContinueOnError: false}, // New group, will fail
		{DeviceID: "light-02", Command: "set", ContinueOnError: true},            // New group, should be skipped
	})

	execID, err := engine.ActivateScene(ctx, "abort", "manual", "api")
	if err != nil {
		t.Fatalf("ActivateScene: %v", err)
	}

	// light-01 published (group 1), nonexistent fails (group 2), light-02 skipped (group 3)
	msgs := mqtt.getMessages()
	if len(msgs) != 1 {
		t.Errorf("expected 1 MQTT message (only light-01), got %d", len(msgs))
	}

	exec, _ := repo.GetExecution(ctx, execID)
	if exec.Status != StatusFailed {
		t.Errorf("status = %q, want %q", exec.Status, StatusFailed)
	}
	if exec.ActionsSkipped != 1 {
		t.Errorf("ActionsSkipped = %d, want 1", exec.ActionsSkipped)
	}
}

func TestEngine_ActivateScene_ContextCancelled(t *testing.T) {
	engine, _, _, repo := setupEngine(t)

	// Create a scene with a long delay
	createTestScene(repo, engine.registry, "cancel", "Cancel Test", []SceneAction{
		{DeviceID: "light-01", Command: "set", DelayMS: 5000, ContinueOnError: true},
	})

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	execID, err := engine.ActivateScene(ctx, "cancel", "manual", "api")
	if err != nil {
		t.Fatalf("ActivateScene: %v", err)
	}

	exec, _ := repo.GetExecution(context.Background(), execID)
	if exec.Status != StatusCancelled {
		t.Errorf("status = %q, want %q", exec.Status, StatusCancelled)
	}
}

func TestEngine_ActivateScene_MQTTPublishFailure(t *testing.T) {
	engine, mqtt, _, repo := setupEngine(t)
	ctx := context.Background()

	// Make MQTT fail for light-01
	mqtt.failOn = "graylogic/command/knx/light-01"

	createTestScene(repo, engine.registry, "mqttfail", "MQTT Fail Test", []SceneAction{
		{DeviceID: "light-01", Command: "set", ContinueOnError: true},
		{DeviceID: "light-02", Command: "set", ContinueOnError: true},
	})

	execID, err := engine.ActivateScene(ctx, "mqttfail", "manual", "api")
	if err != nil {
		t.Fatalf("ActivateScene: %v", err)
	}

	exec, _ := repo.GetExecution(ctx, execID)
	if exec.Status != StatusPartial {
		t.Errorf("status = %q, want %q", exec.Status, StatusPartial)
	}
	if exec.ActionsFailed != 1 {
		t.Errorf("ActionsFailed = %d, want 1", exec.ActionsFailed)
	}
}

func TestEngine_ActivateScene_GatewayRouting(t *testing.T) {
	repo := newMockRepository()
	registry := NewRegistry(repo)

	// Device with custom gateway — topic still uses protocol, not gateway ID.
	// GatewayID is for the bridge's internal routing, not MQTT topics.
	gatewayID := "knx-bridge-02"
	devices := &mockDeviceRegistry{
		devices: map[string]DeviceInfo{
			"light-gw": {ID: "light-gw", Protocol: "knx", GatewayID: &gatewayID},
		},
	}

	mqtt := newMockMQTT()
	engine := NewEngine(registry, devices, mqtt, nil, repo, noopLogger{})
	ctx := context.Background()

	createTestScene(repo, registry, "gateway", "Gateway Test", []SceneAction{
		{DeviceID: "light-gw", Command: "set", ContinueOnError: true},
	})

	_, err := engine.ActivateScene(ctx, "gateway", "manual", "api")
	if err != nil {
		t.Fatalf("ActivateScene: %v", err)
	}

	msgs := mqtt.getMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	expectedTopic := "graylogic/command/knx/light-gw"
	if msgs[0].Topic != expectedTopic {
		t.Errorf("topic = %q, want %q", msgs[0].Topic, expectedTopic)
	}
}

func TestEngine_ActivateScene_Performance(t *testing.T) {
	engine, _, _, repo := setupEngine(t)
	ctx := context.Background()

	// Create a scene with 10 parallel actions (M1.6 performance target: <500ms)
	actions := make([]SceneAction, 10)
	for i := range 10 {
		actions[i] = SceneAction{
			DeviceID:        "light-01", // All use same device (mock returns instantly)
			Command:         "set",
			Parameters:      map[string]any{"brightness": float64(i * 10)},
			Parallel:        i > 0, // First action starts group, rest join
			ContinueOnError: true,
		}
	}
	createTestScene(repo, engine.registry, "perf", "Performance Test", actions)

	start := time.Now()
	_, err := engine.ActivateScene(ctx, "perf", "manual", "api")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ActivateScene: %v", err)
	}

	// Performance target: <500ms for 10 devices
	if elapsed > 500*time.Millisecond {
		t.Errorf("10-device scene took %v, target is <500ms", elapsed)
	}
	t.Logf("10-device scene execution: %v", elapsed)
}

func TestEngine_ActivateScene_WebSocketBroadcast(t *testing.T) {
	engine, _, hub, repo := setupEngine(t)
	ctx := context.Background()

	createTestScene(repo, engine.registry, "ws-test", "WS Broadcast", []SceneAction{
		{DeviceID: "light-01", Command: "set", ContinueOnError: true},
	})

	_, err := engine.ActivateScene(ctx, "ws-test", "manual", "wall_panel")
	if err != nil {
		t.Fatalf("ActivateScene: %v", err)
	}

	broadcasts := hub.getBroadcasts()
	if len(broadcasts) != 1 {
		t.Fatalf("expected 1 broadcast, got %d", len(broadcasts))
	}

	bc := broadcasts[0]
	if bc.Channel != "scene.activated" {
		t.Errorf("channel = %q, want %q", bc.Channel, "scene.activated")
	}

	payload, ok := bc.Payload.(map[string]any)
	if !ok {
		t.Fatal("payload is not map[string]any")
	}
	if payload["scene_name"] != "WS Broadcast" {
		t.Errorf("scene_name = %v, want %q", payload["scene_name"], "WS Broadcast")
	}
}

func TestEngine_ActivateScene_NilHub(t *testing.T) {
	repo := newMockRepository()
	registry := NewRegistry(repo)
	devices := newMockDeviceRegistry()
	mqtt := newMockMQTT()

	// Engine with nil hub — should not panic
	engine := NewEngine(registry, devices, mqtt, nil, repo, noopLogger{})
	ctx := context.Background()

	createTestScene(repo, registry, "nilhub", "Nil Hub", []SceneAction{
		{DeviceID: "light-01", Command: "set", ContinueOnError: true},
	})

	_, err := engine.ActivateScene(ctx, "nilhub", "manual", "api")
	if err != nil {
		t.Fatalf("ActivateScene with nil hub: %v", err)
	}
}

func TestGroupActions(t *testing.T) {
	tests := []struct {
		name       string
		actions    []SceneAction
		wantGroups int
		wantSizes  []int // Size of each group
	}{
		{
			name:       "empty",
			actions:    nil,
			wantGroups: 0,
		},
		{
			name: "single action",
			actions: []SceneAction{
				{DeviceID: "d1", Command: "set"},
			},
			wantGroups: 1,
			wantSizes:  []int{1},
		},
		{
			name: "all sequential",
			actions: []SceneAction{
				{DeviceID: "d1", Command: "set"},
				{DeviceID: "d2", Command: "set"},
				{DeviceID: "d3", Command: "set"},
			},
			wantGroups: 3,
			wantSizes:  []int{1, 1, 1},
		},
		{
			name: "all parallel",
			actions: []SceneAction{
				{DeviceID: "d1", Command: "set"},
				{DeviceID: "d2", Command: "set", Parallel: true},
				{DeviceID: "d3", Command: "set", Parallel: true},
			},
			wantGroups: 1,
			wantSizes:  []int{3},
		},
		{
			name: "mixed groups",
			actions: []SceneAction{
				{DeviceID: "d1", Command: "set"},                 // Group 1
				{DeviceID: "d2", Command: "set", Parallel: true}, // Group 1
				{DeviceID: "d3", Command: "set"},                 // Group 2
				{DeviceID: "d4", Command: "set", Parallel: true}, // Group 2
				{DeviceID: "d5", Command: "set", Parallel: true}, // Group 2
				{DeviceID: "d6", Command: "set"},                 // Group 3
			},
			wantGroups: 3,
			wantSizes:  []int{2, 3, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := groupActions(tt.actions)
			if len(groups) != tt.wantGroups {
				t.Errorf("groups count = %d, want %d", len(groups), tt.wantGroups)
				return
			}
			for i, size := range tt.wantSizes {
				if len(groups[i]) != size {
					t.Errorf("group[%d] size = %d, want %d", i, len(groups[i]), size)
				}
			}
		})
	}
}
