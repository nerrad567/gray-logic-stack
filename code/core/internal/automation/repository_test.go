package automation

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database with the scenes schema.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}

	// Create the scenes table (matches migration)
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
		) STRICT;`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

// testScene creates a test scene with the given ID and name.
func testScene(id, name string) *Scene {
	return &Scene{
		ID:       id,
		Name:     name,
		Slug:     GenerateSlug(name),
		Enabled:  true,
		Priority: 50,
		Actions: []SceneAction{
			{
				DeviceID:        "light-living-main",
				Command:         "set",
				Parameters:      map[string]any{"on": true, "brightness": float64(80)},
				ContinueOnError: true,
			},
		},
	}
}

func TestSQLiteRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	t.Run("create success", func(t *testing.T) {
		scene := testScene("scene-01", "Cinema Mode")
		scene.Category = CategoryEntertainment
		desc := "Dims lights for movie watching"
		scene.Description = &desc
		roomID := "room-living"
		scene.RoomID = &roomID

		err := repo.Create(ctx, scene)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		// Verify timestamps were set
		if scene.CreatedAt.IsZero() {
			t.Error("CreatedAt not set")
		}
		if scene.UpdatedAt.IsZero() {
			t.Error("UpdatedAt not set")
		}
	})

	t.Run("duplicate ID", func(t *testing.T) {
		scene := testScene("scene-01", "Duplicate")
		scene.Slug = "duplicate" // Different slug to avoid that constraint

		err := repo.Create(ctx, scene)
		if !errors.Is(err, ErrSceneExists) {
			t.Errorf("expected ErrSceneExists, got: %v", err)
		}
	})

	t.Run("duplicate slug", func(t *testing.T) {
		scene := testScene("scene-99", "Cinema Mode") // Same name → same slug
		err := repo.Create(ctx, scene)
		if !errors.Is(err, ErrSceneExists) {
			t.Errorf("expected ErrSceneExists, got: %v", err)
		}
	})
}

func TestSQLiteRepository_GetByID(t *testing.T) { //nolint:gocognit // comprehensive table-driven test
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create a scene with all fields
	scene := testScene("scene-get", "Evening Relax")
	scene.Category = CategoryComfort
	desc := "Relaxing evening ambiance"
	scene.Description = &desc
	icon := "moon"
	scene.Icon = &icon
	colourHex := "#FFD700"
	scene.Colour = &colourHex
	scene.Priority = 60
	scene.SortOrder = 2
	scene.Actions = []SceneAction{
		{DeviceID: "light-01", Command: "dim", Parameters: map[string]any{"brightness": float64(30)}, FadeMS: 3000, ContinueOnError: true},
		{DeviceID: "blind-01", Command: "position", Parameters: map[string]any{"position": float64(80)}, Parallel: true, ContinueOnError: true},
	}

	if err := repo.Create(ctx, scene); err != nil {
		t.Fatalf("Create: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetByID(ctx, "scene-get")
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}

		if got.ID != "scene-get" {
			t.Errorf("ID = %q, want %q", got.ID, "scene-get")
		}
		if got.Name != "Evening Relax" {
			t.Errorf("Name = %q, want %q", got.Name, "Evening Relax")
		}
		if got.Category != CategoryComfort {
			t.Errorf("Category = %q, want %q", got.Category, CategoryComfort)
		}
		if got.Description == nil || *got.Description != "Relaxing evening ambiance" {
			t.Errorf("Description = %v, want %q", got.Description, "Relaxing evening ambiance")
		}
		if got.Icon == nil || *got.Icon != "moon" {
			t.Errorf("Icon = %v, want %q", got.Icon, "moon")
		}
		if got.Colour == nil || *got.Colour != "#FFD700" {
			t.Errorf("Colour = %v, want %q", got.Colour, "#FFD700")
		}
		if got.Priority != 60 {
			t.Errorf("Priority = %d, want 60", got.Priority)
		}
		if got.SortOrder != 2 {
			t.Errorf("SortOrder = %d, want 2", got.SortOrder)
		}
		if !got.Enabled {
			t.Error("Enabled = false, want true")
		}
		if len(got.Actions) != 2 {
			t.Fatalf("Actions count = %d, want 2", len(got.Actions))
		}
		if got.Actions[0].FadeMS != 3000 {
			t.Errorf("Action[0].FadeMS = %d, want 3000", got.Actions[0].FadeMS)
		}
		if !got.Actions[1].Parallel {
			t.Error("Action[1].Parallel = false, want true")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "nonexistent")
		if !errors.Is(err, ErrSceneNotFound) {
			t.Errorf("expected ErrSceneNotFound, got: %v", err)
		}
	})
}

func TestSQLiteRepository_GetBySlug(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	scene := testScene("scene-slug", "Good Morning")
	if err := repo.Create(ctx, scene); err != nil {
		t.Fatalf("Create: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetBySlug(ctx, "good-morning")
		if err != nil {
			t.Fatalf("GetBySlug: %v", err)
		}
		if got.ID != "scene-slug" {
			t.Errorf("ID = %q, want %q", got.ID, "scene-slug")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetBySlug(ctx, "nonexistent")
		if !errors.Is(err, ErrSceneNotFound) {
			t.Errorf("expected ErrSceneNotFound, got: %v", err)
		}
	})
}

func TestSQLiteRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	t.Run("empty", func(t *testing.T) {
		scenes, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(scenes) != 0 {
			t.Errorf("expected 0 scenes, got %d", len(scenes))
		}
	})

	// Insert test scenes
	for i, name := range []string{"Cinema Mode", "All Off", "Good Morning"} {
		s := testScene("scene-list-"+string(rune('a'+i)), name)
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Create %q: %v", name, err)
		}
	}

	t.Run("multiple", func(t *testing.T) {
		scenes, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(scenes) != 3 {
			t.Errorf("expected 3 scenes, got %d", len(scenes))
		}
	})
}

func TestSQLiteRepository_ListByRoom(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	roomID := "room-living"
	s1 := testScene("scene-room-1", "Room Scene 1")
	s1.Slug = "room-scene-1"
	s1.RoomID = &roomID
	s2 := testScene("scene-room-2", "Room Scene 2")
	s2.Slug = "room-scene-2"
	s2.RoomID = &roomID
	s3 := testScene("scene-other", "Other Scene")
	s3.Slug = "other-scene"

	for _, s := range []*Scene{s1, s2, s3} {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Create %q: %v", s.Name, err)
		}
	}

	scenes, err := repo.ListByRoom(ctx, roomID)
	if err != nil {
		t.Fatalf("ListByRoom: %v", err)
	}
	if len(scenes) != 2 {
		t.Errorf("expected 2 scenes in room, got %d", len(scenes))
	}
}

func TestSQLiteRepository_ListByArea(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	areaID := "area-ground"
	s1 := testScene("scene-area-1", "Area Scene")
	s1.Slug = "area-scene"
	s1.AreaID = &areaID
	s2 := testScene("scene-noarea", "No Area")
	s2.Slug = "no-area"

	for _, s := range []*Scene{s1, s2} {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Create %q: %v", s.Name, err)
		}
	}

	scenes, err := repo.ListByArea(ctx, areaID)
	if err != nil {
		t.Fatalf("ListByArea: %v", err)
	}
	if len(scenes) != 1 {
		t.Errorf("expected 1 scene in area, got %d", len(scenes))
	}
}

func TestSQLiteRepository_ListByCategory(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	s1 := testScene("scene-cat-1", "Scene Comfort")
	s1.Slug = "scene-comfort"
	s1.Category = CategoryComfort
	s2 := testScene("scene-cat-2", "Scene Energy")
	s2.Slug = "scene-energy"
	s2.Category = CategoryEnergy

	for _, s := range []*Scene{s1, s2} {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Create %q: %v", s.Name, err)
		}
	}

	scenes, err := repo.ListByCategory(ctx, CategoryComfort)
	if err != nil {
		t.Fatalf("ListByCategory: %v", err)
	}
	if len(scenes) != 1 {
		t.Errorf("expected 1 comfort scene, got %d", len(scenes))
	}
	if scenes[0].Category != CategoryComfort {
		t.Errorf("Category = %q, want %q", scenes[0].Category, CategoryComfort)
	}
}

func TestSQLiteRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	scene := testScene("scene-upd", "Original Name")
	if err := repo.Create(ctx, scene); err != nil {
		t.Fatalf("Create: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		scene.Name = "Updated Name"
		scene.Slug = "updated-name"
		scene.Priority = 80
		scene.Enabled = false

		err := repo.Update(ctx, scene)
		if err != nil {
			t.Fatalf("Update: %v", err)
		}

		got, err := repo.GetByID(ctx, "scene-upd")
		if err != nil {
			t.Fatalf("GetByID after update: %v", err)
		}
		if got.Name != "Updated Name" {
			t.Errorf("Name = %q, want %q", got.Name, "Updated Name")
		}
		if got.Priority != 80 {
			t.Errorf("Priority = %d, want 80", got.Priority)
		}
		if got.Enabled {
			t.Error("Enabled = true, want false")
		}
	})

	t.Run("not found", func(t *testing.T) {
		notFound := testScene("nonexistent", "Nope")
		err := repo.Update(ctx, notFound)
		if !errors.Is(err, ErrSceneNotFound) {
			t.Errorf("expected ErrSceneNotFound, got: %v", err)
		}
	})
}

func TestSQLiteRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	scene := testScene("scene-del", "Delete Me")
	if err := repo.Create(ctx, scene); err != nil {
		t.Fatalf("Create: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		err := repo.Delete(ctx, "scene-del")
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}

		_, err = repo.GetByID(ctx, "scene-del")
		if !errors.Is(err, ErrSceneNotFound) {
			t.Errorf("expected ErrSceneNotFound after delete, got: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := repo.Delete(ctx, "nonexistent")
		if !errors.Is(err, ErrSceneNotFound) {
			t.Errorf("expected ErrSceneNotFound, got: %v", err)
		}
	})
}

func TestSQLiteRepository_CreateExecution(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create a scene first (foreign key)
	scene := testScene("scene-exec", "Exec Scene")
	if err := repo.Create(ctx, scene); err != nil {
		t.Fatalf("Create scene: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	source := "api"
	exec := &SceneExecution{
		ID:            "exec-01",
		SceneID:       "scene-exec",
		TriggeredAt:   now,
		TriggerType:   "manual",
		TriggerSource: &source,
		Status:        StatusPending,
		ActionsTotal:  3,
	}

	err := repo.CreateExecution(ctx, exec)
	if err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}

	// Retrieve and verify
	got, err := repo.GetExecution(ctx, "exec-01")
	if err != nil {
		t.Fatalf("GetExecution: %v", err)
	}
	if got.SceneID != "scene-exec" {
		t.Errorf("SceneID = %q, want %q", got.SceneID, "scene-exec")
	}
	if got.Status != StatusPending {
		t.Errorf("Status = %q, want %q", got.Status, StatusPending)
	}
	if got.ActionsTotal != 3 {
		t.Errorf("ActionsTotal = %d, want 3", got.ActionsTotal)
	}
	if got.TriggerSource == nil || *got.TriggerSource != "api" {
		t.Errorf("TriggerSource = %v, want %q", got.TriggerSource, "api")
	}
}

func TestSQLiteRepository_UpdateExecution(t *testing.T) { //nolint:gocognit // comprehensive table-driven test
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	scene := testScene("scene-exec-upd", "Exec Update Scene")
	if err := repo.Create(ctx, scene); err != nil {
		t.Fatalf("Create scene: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	exec := &SceneExecution{
		ID:           "exec-upd-01",
		SceneID:      "scene-exec-upd",
		TriggeredAt:  now,
		TriggerType:  "manual",
		Status:       StatusPending,
		ActionsTotal: 2,
	}
	if err := repo.CreateExecution(ctx, exec); err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}

	t.Run("transition to completed", func(t *testing.T) {
		started := now.Add(10 * time.Millisecond)
		completed := now.Add(150 * time.Millisecond)
		duration := 140
		exec.StartedAt = &started
		exec.CompletedAt = &completed
		exec.Status = StatusCompleted
		exec.ActionsCompleted = 2
		exec.DurationMS = &duration

		err := repo.UpdateExecution(ctx, exec)
		if err != nil {
			t.Fatalf("UpdateExecution: %v", err)
		}

		got, err := repo.GetExecution(ctx, "exec-upd-01")
		if err != nil {
			t.Fatalf("GetExecution: %v", err)
		}
		if got.Status != StatusCompleted {
			t.Errorf("Status = %q, want %q", got.Status, StatusCompleted)
		}
		if got.ActionsCompleted != 2 {
			t.Errorf("ActionsCompleted = %d, want 2", got.ActionsCompleted)
		}
		if got.DurationMS == nil || *got.DurationMS != 140 {
			t.Errorf("DurationMS = %v, want 140", got.DurationMS)
		}
	})

	t.Run("with failures", func(t *testing.T) {
		exec.Status = StatusPartial
		exec.ActionsFailed = 1
		exec.ActionsCompleted = 1
		exec.Failures = []ActionFailure{
			{ActionIndex: 1, DeviceID: "light-02", Command: "set", ErrorCode: "DEVICE_OFFLINE", ErrorMsg: "device unreachable"},
		}

		err := repo.UpdateExecution(ctx, exec)
		if err != nil {
			t.Fatalf("UpdateExecution: %v", err)
		}

		got, err := repo.GetExecution(ctx, "exec-upd-01")
		if err != nil {
			t.Fatalf("GetExecution: %v", err)
		}
		if len(got.Failures) != 1 {
			t.Fatalf("Failures count = %d, want 1", len(got.Failures))
		}
		if got.Failures[0].DeviceID != "light-02" {
			t.Errorf("Failure DeviceID = %q, want %q", got.Failures[0].DeviceID, "light-02")
		}
		if got.Failures[0].ErrorCode != "DEVICE_OFFLINE" {
			t.Errorf("Failure ErrorCode = %q, want %q", got.Failures[0].ErrorCode, "DEVICE_OFFLINE")
		}
	})

	t.Run("not found", func(t *testing.T) {
		notFound := &SceneExecution{ID: "nonexistent", Status: StatusFailed}
		err := repo.UpdateExecution(ctx, notFound)
		if !errors.Is(err, ErrExecutionNotFound) {
			t.Errorf("expected ErrExecutionNotFound, got: %v", err)
		}
	})
}

func TestSQLiteRepository_ListExecutions(t *testing.T) { //nolint:gocognit // comprehensive table-driven test
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	scene := testScene("scene-exec-list", "Exec List Scene")
	if err := repo.Create(ctx, scene); err != nil {
		t.Fatalf("Create scene: %v", err)
	}

	// Insert 5 executions with different times
	now := time.Now().UTC().Truncate(time.Second)
	for i := range 5 {
		exec := &SceneExecution{
			ID:           GenerateID(),
			SceneID:      "scene-exec-list",
			TriggeredAt:  now.Add(time.Duration(i) * time.Second),
			TriggerType:  "manual",
			Status:       StatusCompleted,
			ActionsTotal: 1,
		}
		if err := repo.CreateExecution(ctx, exec); err != nil {
			t.Fatalf("CreateExecution %d: %v", i, err)
		}
	}

	t.Run("default limit", func(t *testing.T) {
		execs, err := repo.ListExecutions(ctx, "scene-exec-list", 0)
		if err != nil {
			t.Fatalf("ListExecutions: %v", err)
		}
		if len(execs) != 5 {
			t.Errorf("expected 5 executions, got %d", len(execs))
		}
	})

	t.Run("with limit", func(t *testing.T) {
		execs, err := repo.ListExecutions(ctx, "scene-exec-list", 3)
		if err != nil {
			t.Fatalf("ListExecutions: %v", err)
		}
		if len(execs) != 3 {
			t.Errorf("expected 3 executions, got %d", len(execs))
		}
	})

	t.Run("ordered by triggered_at DESC", func(t *testing.T) {
		execs, err := repo.ListExecutions(ctx, "scene-exec-list", 5)
		if err != nil {
			t.Fatalf("ListExecutions: %v", err)
		}
		if len(execs) < 2 {
			t.Fatal("need at least 2 executions for ordering check")
		}
		// Most recent first
		if !execs[0].TriggeredAt.After(execs[1].TriggeredAt) {
			t.Errorf("expected descending order: %v should be after %v",
				execs[0].TriggeredAt, execs[1].TriggeredAt)
		}
	})

	t.Run("nonexistent scene", func(t *testing.T) {
		execs, err := repo.ListExecutions(ctx, "nonexistent", 10)
		if err != nil {
			t.Fatalf("ListExecutions: %v", err)
		}
		if len(execs) != 0 {
			t.Errorf("expected 0 executions, got %d", len(execs))
		}
	})

	t.Run("limit capped at 100", func(t *testing.T) {
		// Should not error even with limit > 100
		_, err := repo.ListExecutions(ctx, "scene-exec-list", 500)
		if err != nil {
			t.Fatalf("ListExecutions with large limit: %v", err)
		}
	})
}

func TestSQLiteRepository_GetExecution_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	_, err := repo.GetExecution(ctx, "nonexistent")
	if !errors.Is(err, ErrExecutionNotFound) {
		t.Errorf("expected ErrExecutionNotFound, got: %v", err)
	}
}
