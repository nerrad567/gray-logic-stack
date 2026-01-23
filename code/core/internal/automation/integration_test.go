package automation

import (
	"context"
	"testing"
)

// TestIntegration_SceneLifecycle tests the full lifecycle with a real SQLite database:
// create → list → activate → check execution → update → delete → verify gone.
func TestIntegration_SceneLifecycle(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Refresh empty cache
	if err := registry.RefreshCache(ctx); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}
	if registry.GetSceneCount() != 0 {
		t.Fatalf("expected 0 scenes, got %d", registry.GetSceneCount())
	}

	// Create a scene via registry
	scene := &Scene{
		Name:     "Integration Test Scene",
		Enabled:  true,
		Priority: 60,
		Category: CategoryDaily,
		Actions: []SceneAction{
			{DeviceID: "light-living-main", Command: "set", Parameters: map[string]any{"on": true}, ContinueOnError: true},
			{DeviceID: "blind-living-01", Command: "position", Parameters: map[string]any{"position": float64(100)}, Parallel: true, FadeMS: 2000, ContinueOnError: true},
		},
	}

	if err := registry.CreateScene(ctx, scene); err != nil {
		t.Fatalf("CreateScene: %v", err)
	}

	// Verify ID and slug were generated
	if scene.ID == "" {
		t.Error("scene ID not generated")
	}
	if scene.Slug != "integration-test-scene" {
		t.Errorf("slug = %q, want %q", scene.Slug, "integration-test-scene")
	}

	// List should return 1
	scenes, err := registry.ListScenes(ctx)
	if err != nil {
		t.Fatalf("ListScenes: %v", err)
	}
	if len(scenes) != 1 {
		t.Fatalf("expected 1 scene, got %d", len(scenes))
	}

	// Get by ID
	got, err := registry.GetScene(ctx, scene.ID)
	if err != nil {
		t.Fatalf("GetScene: %v", err)
	}
	if got.Name != "Integration Test Scene" {
		t.Errorf("Name = %q, want %q", got.Name, "Integration Test Scene")
	}
	if got.Category != CategoryDaily {
		t.Errorf("Category = %q, want %q", got.Category, CategoryDaily)
	}
	if len(got.Actions) != 2 {
		t.Fatalf("Actions count = %d, want 2", len(got.Actions))
	}
	if got.Actions[1].FadeMS != 2000 {
		t.Errorf("Action[1].FadeMS = %d, want 2000", got.Actions[1].FadeMS)
	}

	// Get by slug
	bySlug, err := registry.GetSceneBySlug(ctx, "integration-test-scene")
	if err != nil {
		t.Fatalf("GetSceneBySlug: %v", err)
	}
	if bySlug.ID != scene.ID {
		t.Errorf("GetSceneBySlug ID = %q, want %q", bySlug.ID, scene.ID)
	}

	// Update the scene
	got.Name = "Updated Integration Scene"
	got.Slug = "updated-integration-scene"
	got.Priority = 80
	if updateErr := registry.UpdateScene(ctx, got); updateErr != nil {
		t.Fatalf("UpdateScene: %v", updateErr)
	}

	updated, err := registry.GetScene(ctx, scene.ID)
	if err != nil {
		t.Fatalf("GetScene after update: %v", err)
	}
	if updated.Name != "Updated Integration Scene" {
		t.Errorf("Name after update = %q, want %q", updated.Name, "Updated Integration Scene")
	}
	if updated.Priority != 80 {
		t.Errorf("Priority after update = %d, want 80", updated.Priority)
	}

	// Delete the scene
	if err := registry.DeleteScene(ctx, scene.ID); err != nil {
		t.Fatalf("DeleteScene: %v", err)
	}

	if registry.GetSceneCount() != 0 {
		t.Errorf("SceneCount after delete = %d, want 0", registry.GetSceneCount())
	}
}

// TestIntegration_PersistAcrossRestart verifies that scenes persist across
// cache refreshes (simulating application restarts).
func TestIntegration_PersistAcrossRestart(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create scenes with first registry instance
	registry1 := NewRegistry(repo)
	_ = registry1.RefreshCache(ctx)

	scene := &Scene{
		ID:       "persist-test",
		Name:     "Persist Test",
		Slug:     "persist-test",
		Enabled:  true,
		Priority: 50,
		Category: CategoryComfort,
		Actions: []SceneAction{
			{DeviceID: "light-01", Command: "dim", Parameters: map[string]any{"brightness": float64(50)}, FadeMS: 1000, ContinueOnError: true},
		},
	}

	if err := registry1.CreateScene(ctx, scene); err != nil {
		t.Fatalf("CreateScene: %v", err)
	}

	// Simulate restart: create a new registry with the same repo
	registry2 := NewRegistry(repo)
	if err := registry2.RefreshCache(ctx); err != nil {
		t.Fatalf("RefreshCache (restart): %v", err)
	}

	// Scene should still be there
	if registry2.GetSceneCount() != 1 {
		t.Fatalf("SceneCount after restart = %d, want 1", registry2.GetSceneCount())
	}

	got, err := registry2.GetScene(ctx, "persist-test")
	if err != nil {
		t.Fatalf("GetScene after restart: %v", err)
	}
	if got.Name != "Persist Test" {
		t.Errorf("Name = %q, want %q", got.Name, "Persist Test")
	}
	if got.Category != CategoryComfort {
		t.Errorf("Category = %q, want %q", got.Category, CategoryComfort)
	}
	if len(got.Actions) != 1 {
		t.Fatalf("Actions count = %d, want 1", len(got.Actions))
	}
	if got.Actions[0].FadeMS != 1000 {
		t.Errorf("Action FadeMS = %d, want 1000", got.Actions[0].FadeMS)
	}
}

// TestIntegration_ExecutionLifecycle tests creating and tracking executions
// through the full lifecycle with a real SQLite database.
func TestIntegration_ExecutionLifecycle(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	registry := NewRegistry(repo)
	devices := newMockDeviceRegistry()
	mqtt := newMockMQTT()
	hub := newMockWSHub()
	ctx := context.Background()

	engine := NewEngine(registry, devices, mqtt, hub, repo, noopLogger{})

	// Create a scene
	scene := &Scene{
		ID:       "exec-lifecycle",
		Name:     "Execution Lifecycle",
		Slug:     "execution-lifecycle",
		Enabled:  true,
		Priority: 50,
		Actions: []SceneAction{
			{DeviceID: "light-01", Command: "set", Parameters: map[string]any{"on": true}, ContinueOnError: true},
			{DeviceID: "light-02", Command: "dim", Parameters: map[string]any{"brightness": float64(50)}, Parallel: true, ContinueOnError: true},
		},
	}
	if err := registry.CreateScene(ctx, scene); err != nil {
		t.Fatalf("CreateScene: %v", err)
	}

	// Activate the scene
	execID, err := engine.ActivateScene(ctx, "exec-lifecycle", "manual", "api")
	if err != nil {
		t.Fatalf("ActivateScene: %v", err)
	}

	// Verify execution record
	exec, err := repo.GetExecution(ctx, execID)
	if err != nil {
		t.Fatalf("GetExecution: %v", err)
	}
	if exec.Status != StatusCompleted {
		t.Errorf("Status = %q, want %q", exec.Status, StatusCompleted)
	}
	if exec.ActionsTotal != 2 {
		t.Errorf("ActionsTotal = %d, want 2", exec.ActionsTotal)
	}
	if exec.ActionsCompleted != 2 {
		t.Errorf("ActionsCompleted = %d, want 2", exec.ActionsCompleted)
	}
	if exec.DurationMS == nil {
		t.Error("DurationMS is nil")
	}
	if exec.StartedAt == nil {
		t.Error("StartedAt is nil")
	}
	if exec.CompletedAt == nil {
		t.Error("CompletedAt is nil")
	}

	// List executions
	execs, err := repo.ListExecutions(ctx, "exec-lifecycle", 10)
	if err != nil {
		t.Fatalf("ListExecutions: %v", err)
	}
	if len(execs) != 1 {
		t.Errorf("expected 1 execution, got %d", len(execs))
	}

	// Activate again
	execID2, err := engine.ActivateScene(ctx, "exec-lifecycle", "manual", "mobile")
	if err != nil {
		t.Fatalf("ActivateScene (2): %v", err)
	}
	if execID2 == execID {
		t.Error("second execution has same ID as first")
	}

	// Should now have 2 executions
	execs, err = repo.ListExecutions(ctx, "exec-lifecycle", 10)
	if err != nil {
		t.Fatalf("ListExecutions (2): %v", err)
	}
	if len(execs) != 2 {
		t.Errorf("expected 2 executions, got %d", len(execs))
	}
}
