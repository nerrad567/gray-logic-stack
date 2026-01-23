package automation

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// mockRepository is an in-memory implementation of Repository for testing.
type mockRepository struct {
	scenes     map[string]*Scene
	executions map[string]*SceneExecution
	mu         sync.RWMutex
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		scenes:     make(map[string]*Scene),
		executions: make(map[string]*SceneExecution),
	}
}

func (m *mockRepository) GetByID(_ context.Context, id string) (*Scene, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.scenes[id]
	if !ok {
		return nil, ErrSceneNotFound
	}
	return s.DeepCopy(), nil
}

func (m *mockRepository) GetBySlug(_ context.Context, slug string) (*Scene, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.scenes {
		if s.Slug == slug {
			return s.DeepCopy(), nil
		}
	}
	return nil, ErrSceneNotFound
}

func (m *mockRepository) List(_ context.Context) ([]Scene, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	scenes := make([]Scene, 0, len(m.scenes))
	for _, s := range m.scenes {
		scenes = append(scenes, *s.DeepCopy())
	}
	return scenes, nil
}

func (m *mockRepository) ListByRoom(_ context.Context, roomID string) ([]Scene, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var scenes []Scene
	for _, s := range m.scenes {
		if s.RoomID != nil && *s.RoomID == roomID {
			scenes = append(scenes, *s.DeepCopy())
		}
	}
	return scenes, nil
}

func (m *mockRepository) ListByArea(_ context.Context, areaID string) ([]Scene, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var scenes []Scene
	for _, s := range m.scenes {
		if s.AreaID != nil && *s.AreaID == areaID {
			scenes = append(scenes, *s.DeepCopy())
		}
	}
	return scenes, nil
}

func (m *mockRepository) ListByCategory(_ context.Context, category Category) ([]Scene, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var scenes []Scene
	for _, s := range m.scenes {
		if s.Category == category {
			scenes = append(scenes, *s.DeepCopy())
		}
	}
	return scenes, nil
}

func (m *mockRepository) Create(_ context.Context, scene *Scene) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.scenes[scene.ID]; ok {
		return ErrSceneExists
	}
	// Check slug uniqueness
	for _, s := range m.scenes {
		if s.Slug == scene.Slug {
			return ErrSceneExists
		}
	}
	m.scenes[scene.ID] = scene.DeepCopy()
	return nil
}

func (m *mockRepository) Update(_ context.Context, scene *Scene) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.scenes[scene.ID]; !ok {
		return ErrSceneNotFound
	}
	m.scenes[scene.ID] = scene.DeepCopy()
	return nil
}

func (m *mockRepository) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.scenes[id]; !ok {
		return ErrSceneNotFound
	}
	delete(m.scenes, id)
	return nil
}

func (m *mockRepository) CreateExecution(_ context.Context, exec *SceneExecution) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cpy := *exec
	m.executions[exec.ID] = &cpy
	return nil
}

func (m *mockRepository) UpdateExecution(_ context.Context, exec *SceneExecution) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.executions[exec.ID]; !ok {
		return ErrExecutionNotFound
	}
	cpy := *exec
	m.executions[exec.ID] = &cpy
	return nil
}

func (m *mockRepository) GetExecution(_ context.Context, id string) (*SceneExecution, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.executions[id]
	if !ok {
		return nil, ErrExecutionNotFound
	}
	cpy := *e
	return &cpy, nil
}

func (m *mockRepository) ListExecutions(_ context.Context, sceneID string, limit int) ([]SceneExecution, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var execs []SceneExecution
	for _, e := range m.executions {
		if e.SceneID == sceneID {
			execs = append(execs, *e)
		}
	}
	if limit > 0 && len(execs) > limit {
		execs = execs[:limit]
	}
	return execs, nil
}

func TestRegistry_RefreshCache(t *testing.T) {
	repo := newMockRepository()
	ctx := context.Background()

	// Pre-populate repo
	repo.scenes["s1"] = &Scene{ID: "s1", Name: "Scene 1", Slug: "scene-1", Priority: 50, Enabled: true, Actions: []SceneAction{{DeviceID: "d1", Command: "set", ContinueOnError: true}}}
	repo.scenes["s2"] = &Scene{ID: "s2", Name: "Scene 2", Slug: "scene-2", Priority: 50, Enabled: true, Actions: []SceneAction{{DeviceID: "d2", Command: "set", ContinueOnError: true}}}

	registry := NewRegistry(repo)

	if err := registry.RefreshCache(ctx); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	if registry.GetSceneCount() != 2 {
		t.Errorf("SceneCount = %d, want 2", registry.GetSceneCount())
	}
}

func TestRegistry_GetScene(t *testing.T) {
	desc := "Original description"
	roomID := "room-living"
	repo := newMockRepository()
	repo.scenes["s1"] = &Scene{
		ID: "s1", Name: "Test", Slug: "test", Priority: 50, Enabled: true,
		Description: &desc, RoomID: &roomID,
		Actions: []SceneAction{{DeviceID: "d1", Command: "set", Parameters: map[string]any{"on": true}, ContinueOnError: true}},
	}

	registry := NewRegistry(repo)
	ctx := context.Background()
	_ = registry.RefreshCache(ctx)

	t.Run("cache hit", func(t *testing.T) {
		scene, err := registry.GetScene(ctx, "s1")
		if err != nil {
			t.Fatalf("GetScene: %v", err)
		}
		if scene.Name != "Test" {
			t.Errorf("Name = %q, want %q", scene.Name, "Test")
		}
		// Verify deep copy (modifying returned value shouldn't affect cache)
		scene.Name = "Modified"
		original, _ := registry.GetScene(ctx, "s1")
		if original.Name != "Test" {
			t.Error("cache was mutated by returned copy")
		}
	})

	t.Run("pointer field isolation", func(t *testing.T) {
		scene, err := registry.GetScene(ctx, "s1")
		if err != nil {
			t.Fatalf("GetScene: %v", err)
		}
		// Modify pointer fields on the returned copy
		*scene.Description = "Corrupted"
		*scene.RoomID = "room-corrupted"

		// Original cache should be unaffected
		original, _ := registry.GetScene(ctx, "s1")
		if *original.Description != "Original description" {
			t.Errorf("cache Description corrupted: got %q", *original.Description)
		}
		if *original.RoomID != "room-living" {
			t.Errorf("cache RoomID corrupted: got %q", *original.RoomID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := registry.GetScene(ctx, "nonexistent")
		if !errors.Is(err, ErrSceneNotFound) {
			t.Errorf("expected ErrSceneNotFound, got: %v", err)
		}
	})
}

func TestRegistry_GetSceneBySlug(t *testing.T) {
	repo := newMockRepository()
	repo.scenes["s1"] = &Scene{ID: "s1", Name: "Cinema Mode", Slug: "cinema-mode", Priority: 50, Enabled: true, Actions: []SceneAction{{DeviceID: "d1", Command: "set", ContinueOnError: true}}}

	registry := NewRegistry(repo)
	ctx := context.Background()
	_ = registry.RefreshCache(ctx)

	t.Run("found", func(t *testing.T) {
		scene, err := registry.GetSceneBySlug(ctx, "cinema-mode")
		if err != nil {
			t.Fatalf("GetSceneBySlug: %v", err)
		}
		if scene.ID != "s1" {
			t.Errorf("ID = %q, want %q", scene.ID, "s1")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := registry.GetSceneBySlug(ctx, "nonexistent")
		if !errors.Is(err, ErrSceneNotFound) {
			t.Errorf("expected ErrSceneNotFound, got: %v", err)
		}
	})
}

func TestRegistry_ListScenes(t *testing.T) {
	repo := newMockRepository()
	repo.scenes["s1"] = &Scene{ID: "s1", Name: "A", Slug: "a", Priority: 50, Enabled: true, Actions: []SceneAction{{DeviceID: "d1", Command: "set", ContinueOnError: true}}}
	repo.scenes["s2"] = &Scene{ID: "s2", Name: "B", Slug: "b", Priority: 50, Enabled: true, Actions: []SceneAction{{DeviceID: "d2", Command: "set", ContinueOnError: true}}}

	registry := NewRegistry(repo)
	ctx := context.Background()
	_ = registry.RefreshCache(ctx)

	scenes, err := registry.ListScenes(ctx)
	if err != nil {
		t.Fatalf("ListScenes: %v", err)
	}
	if len(scenes) != 2 {
		t.Errorf("expected 2 scenes, got %d", len(scenes))
	}
}

func TestRegistry_ListScenesByRoom(t *testing.T) {
	roomID := "room-living"
	repo := newMockRepository()
	repo.scenes["s1"] = &Scene{ID: "s1", Name: "In Room", Slug: "in-room", RoomID: &roomID, Priority: 50, Enabled: true, Actions: []SceneAction{{DeviceID: "d1", Command: "set", ContinueOnError: true}}}
	repo.scenes["s2"] = &Scene{ID: "s2", Name: "No Room", Slug: "no-room", Priority: 50, Enabled: true, Actions: []SceneAction{{DeviceID: "d2", Command: "set", ContinueOnError: true}}}

	registry := NewRegistry(repo)
	ctx := context.Background()
	_ = registry.RefreshCache(ctx)

	scenes, err := registry.ListScenesByRoom(ctx, roomID)
	if err != nil {
		t.Fatalf("ListScenesByRoom: %v", err)
	}
	if len(scenes) != 1 {
		t.Errorf("expected 1 scene, got %d", len(scenes))
	}
}

func TestRegistry_ListScenesByArea(t *testing.T) {
	areaID := "area-ground"
	repo := newMockRepository()
	repo.scenes["s1"] = &Scene{ID: "s1", Name: "In Area", Slug: "in-area", AreaID: &areaID, Priority: 50, Enabled: true, Actions: []SceneAction{{DeviceID: "d1", Command: "set", ContinueOnError: true}}}
	repo.scenes["s2"] = &Scene{ID: "s2", Name: "No Area", Slug: "no-area", Priority: 50, Enabled: true, Actions: []SceneAction{{DeviceID: "d2", Command: "set", ContinueOnError: true}}}

	registry := NewRegistry(repo)
	ctx := context.Background()
	_ = registry.RefreshCache(ctx)

	scenes, err := registry.ListScenesByArea(ctx, areaID)
	if err != nil {
		t.Fatalf("ListScenesByArea: %v", err)
	}
	if len(scenes) != 1 {
		t.Errorf("expected 1 scene, got %d", len(scenes))
	}
}

func TestRegistry_ListScenesByCategory(t *testing.T) {
	repo := newMockRepository()
	repo.scenes["s1"] = &Scene{ID: "s1", Name: "Comfort", Slug: "comfort", Category: CategoryComfort, Priority: 50, Enabled: true, Actions: []SceneAction{{DeviceID: "d1", Command: "set", ContinueOnError: true}}}
	repo.scenes["s2"] = &Scene{ID: "s2", Name: "Energy", Slug: "energy", Category: CategoryEnergy, Priority: 50, Enabled: true, Actions: []SceneAction{{DeviceID: "d2", Command: "set", ContinueOnError: true}}}

	registry := NewRegistry(repo)
	ctx := context.Background()
	_ = registry.RefreshCache(ctx)

	scenes, err := registry.ListScenesByCategory(ctx, CategoryComfort)
	if err != nil {
		t.Fatalf("ListScenesByCategory: %v", err)
	}
	if len(scenes) != 1 {
		t.Errorf("expected 1 scene, got %d", len(scenes))
	}
}

func TestRegistry_CreateScene(t *testing.T) {
	repo := newMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	t.Run("success with ID generation", func(t *testing.T) {
		scene := &Scene{
			Name:     "New Scene",
			Priority: 50,
			Enabled:  true,
			Actions: []SceneAction{
				{DeviceID: "light-01", Command: "set", ContinueOnError: true},
			},
		}

		err := registry.CreateScene(ctx, scene)
		if err != nil {
			t.Fatalf("CreateScene: %v", err)
		}

		// ID and slug should be generated
		if scene.ID == "" {
			t.Error("ID not generated")
		}
		if scene.Slug == "" {
			t.Error("Slug not generated")
		}
		if scene.Slug != "new-scene" {
			t.Errorf("Slug = %q, want %q", scene.Slug, "new-scene")
		}

		// Should be in cache
		if registry.GetSceneCount() != 1 {
			t.Errorf("SceneCount = %d, want 1", registry.GetSceneCount())
		}
	})

	t.Run("success with provided ID", func(t *testing.T) {
		scene := &Scene{
			ID:       "custom-id",
			Name:     "Custom ID Scene",
			Slug:     "custom-id-scene",
			Priority: 70,
			Enabled:  true,
			Actions: []SceneAction{
				{DeviceID: "light-02", Command: "dim", ContinueOnError: true},
			},
		}

		err := registry.CreateScene(ctx, scene)
		if err != nil {
			t.Fatalf("CreateScene: %v", err)
		}
		if scene.ID != "custom-id" {
			t.Errorf("ID = %q, want %q", scene.ID, "custom-id")
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		scene := &Scene{
			Name:     "", // Invalid
			Priority: 50,
			Actions:  []SceneAction{{DeviceID: "d1", Command: "set"}},
		}

		err := registry.CreateScene(ctx, scene)
		if !errors.Is(err, ErrInvalidName) {
			t.Errorf("expected ErrInvalidName, got: %v", err)
		}
	})

	t.Run("default priority", func(t *testing.T) {
		scene := &Scene{
			Name:    "Default Priority",
			Slug:    "default-priority",
			Enabled: true,
			Actions: []SceneAction{
				{DeviceID: "light-01", Command: "set", ContinueOnError: true},
			},
		}

		err := registry.CreateScene(ctx, scene)
		if err != nil {
			t.Fatalf("CreateScene: %v", err)
		}
		if scene.Priority != defaultPriority {
			t.Errorf("Priority = %d, want %d", scene.Priority, defaultPriority)
		}
	})
}

func TestRegistry_UpdateScene(t *testing.T) {
	repo := newMockRepository()
	repo.scenes["s1"] = &Scene{ID: "s1", Name: "Original", Slug: "original", Priority: 50, Enabled: true, Actions: []SceneAction{{DeviceID: "d1", Command: "set", ContinueOnError: true}}}

	registry := NewRegistry(repo)
	ctx := context.Background()
	_ = registry.RefreshCache(ctx)

	t.Run("success", func(t *testing.T) {
		scene, _ := registry.GetScene(ctx, "s1")
		scene.Name = "Updated"
		scene.Slug = "updated"
		scene.Priority = 80

		err := registry.UpdateScene(ctx, scene)
		if err != nil {
			t.Fatalf("UpdateScene: %v", err)
		}

		// Verify cache is updated
		got, _ := registry.GetScene(ctx, "s1")
		if got.Name != "Updated" {
			t.Errorf("Name = %q, want %q", got.Name, "Updated")
		}
		if got.Priority != 80 {
			t.Errorf("Priority = %d, want 80", got.Priority)
		}
	})

	t.Run("not found", func(t *testing.T) {
		scene := &Scene{ID: "nonexistent", Name: "Nope", Slug: "nope", Priority: 50, Actions: []SceneAction{{DeviceID: "d1", Command: "set", ContinueOnError: true}}}
		err := registry.UpdateScene(ctx, scene)
		if !errors.Is(err, ErrSceneNotFound) {
			t.Errorf("expected ErrSceneNotFound, got: %v", err)
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		scene := &Scene{ID: "s1", Name: "", Slug: "test", Priority: 50, Actions: []SceneAction{{DeviceID: "d1", Command: "set"}}}
		err := registry.UpdateScene(ctx, scene)
		if !errors.Is(err, ErrInvalidName) {
			t.Errorf("expected ErrInvalidName, got: %v", err)
		}
	})
}

func TestRegistry_DeleteScene(t *testing.T) {
	repo := newMockRepository()
	repo.scenes["s1"] = &Scene{ID: "s1", Name: "Delete Me", Slug: "delete-me", Priority: 50, Enabled: true, Actions: []SceneAction{{DeviceID: "d1", Command: "set", ContinueOnError: true}}}

	registry := NewRegistry(repo)
	ctx := context.Background()
	_ = registry.RefreshCache(ctx)

	t.Run("success", func(t *testing.T) {
		err := registry.DeleteScene(ctx, "s1")
		if err != nil {
			t.Fatalf("DeleteScene: %v", err)
		}

		if registry.GetSceneCount() != 0 {
			t.Errorf("SceneCount = %d, want 0", registry.GetSceneCount())
		}

		_, err = registry.GetScene(ctx, "s1")
		if !errors.Is(err, ErrSceneNotFound) {
			t.Errorf("expected ErrSceneNotFound after delete, got: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := registry.DeleteScene(ctx, "nonexistent")
		if !errors.Is(err, ErrSceneNotFound) {
			t.Errorf("expected ErrSceneNotFound, got: %v", err)
		}
	})
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	repo := newMockRepository()
	registry := NewRegistry(repo)
	ctx := context.Background()

	// Pre-populate with some scenes
	for i := range 10 {
		scene := &Scene{
			ID:       GenerateID(),
			Name:     "Concurrent " + string(rune('A'+i)),
			Slug:     "concurrent-" + string(rune('a'+i)),
			Priority: 50,
			Enabled:  true,
			Actions:  []SceneAction{{DeviceID: "d1", Command: "set", ContinueOnError: true}},
		}
		repo.scenes[scene.ID] = scene
	}
	_ = registry.RefreshCache(ctx)

	// Hammer the registry with concurrent reads and writes
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(3)

		// Concurrent reads
		go func() {
			defer wg.Done()
			_, _ = registry.ListScenes(ctx)
		}()

		// Concurrent creates
		go func() {
			defer wg.Done()
			scene := &Scene{
				Name:     "Created " + GenerateID()[:8],
				Slug:     "created-" + GenerateID()[:8],
				Priority: 50,
				Enabled:  true,
				Actions:  []SceneAction{{DeviceID: "d1", Command: "set", ContinueOnError: true}},
			}
			_ = registry.CreateScene(ctx, scene)
		}()

		// Concurrent count reads
		go func() {
			defer wg.Done()
			_ = registry.GetSceneCount()
		}()
	}

	wg.Wait()

	// Should not have panicked — that's the main assertion
	if registry.GetSceneCount() < 10 {
		t.Errorf("SceneCount = %d, expected at least 10", registry.GetSceneCount())
	}
}
