package automation

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Logger defines the logging interface used by the Registry and Engine.
// This allows different logging implementations to be used.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// noopLogger is a logger that does nothing.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

// Registry provides scene management with caching and thread safety.
// It wraps a Repository and adds an in-memory cache for fast lookups.
//
// The cache is populated on startup via RefreshCache() and kept in sync
// by cache-invalidating CRUD operations.
//
// Active scene tracking (roomID -> sceneID) is in-memory only — deliberately
// not persisted. On restart, no scene is considered active because physical
// device state may have changed independently (wall switches, power cycles).
//
// All public methods are thread-safe.
type Registry struct {
	repo         Repository
	cache        map[string]*Scene // Cached scenes by ID
	activeScenes map[string]string // roomID -> sceneID (in-memory only)
	cacheMu      sync.RWMutex      // Protects cache and activeScenes
	logger       Logger
}

// NewRegistry creates a new scene registry.
// The repository is used for persistence; the registry adds caching.
func NewRegistry(repo Repository) *Registry {
	return &Registry{
		repo:         repo,
		cache:        make(map[string]*Scene),
		activeScenes: make(map[string]string),
		logger:       noopLogger{},
	}
}

// SetLogger sets the logger for the registry.
func (r *Registry) SetLogger(logger Logger) {
	r.logger = logger
}

// RefreshCache reloads all scenes from the repository into the cache.
// This should be called on application startup.
func (r *Registry) RefreshCache(ctx context.Context) error {
	scenes, err := r.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("loading scenes: %w", err)
	}

	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	// Clear and rebuild cache with deep copies
	r.cache = make(map[string]*Scene, len(scenes))
	for i := range scenes {
		s := scenes[i]
		r.cache[s.ID] = s.DeepCopy()
	}

	r.logger.Info("scene cache refreshed", "count", len(scenes))
	return nil
}

// GetScene retrieves a scene by ID.
// The returned scene is a deep copy; callers can safely modify it.
func (r *Registry) GetScene(_ context.Context, id string) (*Scene, error) {
	r.cacheMu.RLock()
	cached, ok := r.cache[id]
	r.cacheMu.RUnlock()

	if ok {
		return cached.DeepCopy(), nil
	}
	return nil, ErrSceneNotFound
}

// GetSceneBySlug retrieves a scene by its slug.
// The returned scene is a deep copy.
func (r *Registry) GetSceneBySlug(_ context.Context, slug string) (*Scene, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	for _, s := range r.cache {
		if s.Slug == slug {
			return s.DeepCopy(), nil
		}
	}
	return nil, ErrSceneNotFound
}

// ListScenes retrieves all scenes from the cache.
// Returns deep copies sorted by sort_order then name for deterministic ordering.
func (r *Registry) ListScenes(_ context.Context) ([]Scene, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	scenes := make([]Scene, 0, len(r.cache))
	for _, s := range r.cache {
		scenes = append(scenes, *s.DeepCopy())
	}
	sortScenes(scenes)
	return scenes, nil
}

// ListScenesByRoom retrieves all scenes in a specific room.
func (r *Registry) ListScenesByRoom(_ context.Context, roomID string) ([]Scene, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	var scenes []Scene
	for _, s := range r.cache {
		if s.RoomID != nil && *s.RoomID == roomID {
			scenes = append(scenes, *s.DeepCopy())
		}
	}
	sortScenes(scenes)
	return scenes, nil
}

// ListScenesByArea retrieves all scenes in a specific area.
func (r *Registry) ListScenesByArea(_ context.Context, areaID string) ([]Scene, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	var scenes []Scene
	for _, s := range r.cache {
		if s.AreaID != nil && *s.AreaID == areaID {
			scenes = append(scenes, *s.DeepCopy())
		}
	}
	sortScenes(scenes)
	return scenes, nil
}

// ListScenesByCategory retrieves all scenes in a specific category.
func (r *Registry) ListScenesByCategory(_ context.Context, category Category) ([]Scene, error) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	var scenes []Scene
	for _, s := range r.cache {
		if s.Category == category {
			scenes = append(scenes, *s.DeepCopy())
		}
	}
	sortScenes(scenes)
	return scenes, nil
}

// sortScenes sorts scenes by sort_order then name, matching the DB query ordering.
func sortScenes(scenes []Scene) {
	sort.Slice(scenes, func(i, j int) bool {
		if scenes[i].SortOrder != scenes[j].SortOrder {
			return scenes[i].SortOrder < scenes[j].SortOrder
		}
		return scenes[i].Name < scenes[j].Name
	})
}

// CreateScene validates, persists, and caches a new scene.
func (r *Registry) CreateScene(ctx context.Context, scene *Scene) error {
	// Generate ID and slug if not provided
	if scene.ID == "" {
		scene.ID = GenerateID()
	}
	if scene.Slug == "" {
		scene.Slug = GenerateSlug(scene.Name)
	}

	// Set default priority if not provided
	if scene.Priority == 0 {
		scene.Priority = defaultPriority
	}

	// Set default sort order for actions that don't specify one.
	// ContinueOnError defaults to false (fail-fast) which is safer for building automation.
	for i := range scene.Actions {
		if scene.Actions[i].SortOrder == 0 {
			scene.Actions[i].SortOrder = i
		}
	}

	// Validate
	if err := ValidateScene(scene); err != nil {
		return err
	}

	// Persist
	if err := r.repo.Create(ctx, scene); err != nil {
		return err
	}

	// Update cache
	r.cacheMu.Lock()
	r.cache[scene.ID] = scene.DeepCopy()
	r.cacheMu.Unlock()

	r.logger.Info("scene created", "id", scene.ID, "name", scene.Name)
	return nil
}

// UpdateScene validates, persists, and updates the cached scene.
func (r *Registry) UpdateScene(ctx context.Context, scene *Scene) error {
	// Validate
	if err := ValidateScene(scene); err != nil {
		return err
	}

	// Persist
	if err := r.repo.Update(ctx, scene); err != nil {
		return err
	}

	// Update cache
	r.cacheMu.Lock()
	r.cache[scene.ID] = scene.DeepCopy()
	r.cacheMu.Unlock()

	r.logger.Info("scene updated", "id", scene.ID, "name", scene.Name)
	return nil
}

// DeleteScene removes a scene from persistence and cache.
// If the scene is the active scene for its room, the active tracking is cleared.
func (r *Registry) DeleteScene(ctx context.Context, id string) error {
	if err := r.repo.Delete(ctx, id); err != nil {
		return err
	}

	r.cacheMu.Lock()
	// Clear active tracking if this scene is currently active in any room
	for roomID, activeID := range r.activeScenes {
		if activeID == id {
			delete(r.activeScenes, roomID)
			break // A scene can only be active in one room
		}
	}
	delete(r.cache, id)
	r.cacheMu.Unlock()

	r.logger.Info("scene deleted", "id", id)
	return nil
}

// GetSceneCount returns the number of cached scenes.
func (r *Registry) GetSceneCount() int {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	return len(r.cache)
}

// SetActiveScene records a scene as the active one for a room (last-wins).
func (r *Registry) SetActiveScene(roomID, sceneID string) {
	r.cacheMu.Lock()
	r.activeScenes[roomID] = sceneID
	r.cacheMu.Unlock()
}

// ClearActiveScene removes the active scene for a room.
func (r *Registry) ClearActiveScene(roomID string) {
	r.cacheMu.Lock()
	delete(r.activeScenes, roomID)
	r.cacheMu.Unlock()
}

// GetActiveScene returns the active scene ID for a room, or "" if none.
func (r *Registry) GetActiveScene(roomID string) string {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	return r.activeScenes[roomID]
}

// GetActiveScenes returns a copy of the active scenes map (roomID -> sceneID).
func (r *Registry) GetActiveScenes() map[string]string {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	result := make(map[string]string, len(r.activeScenes))
	for k, v := range r.activeScenes {
		result[k] = v
	}
	return result
}
