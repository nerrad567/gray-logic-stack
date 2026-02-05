package automation

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// setupBenchSceneRegistry creates a registry pre-populated with n scenes.
func setupBenchSceneRegistry(b *testing.B, n int) *Registry {
	b.Helper()
	repo := &mockRepository{scenes: make(map[string]*Scene)}
	ctx := context.Background()

	for i := 0; i < n; i++ {
		scene := &Scene{
			ID:        fmt.Sprintf("scene-%04d", i),
			Name:      fmt.Sprintf("Scene %d", i),
			Slug:      fmt.Sprintf("scene-%d", i),
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Actions: []SceneAction{
				{DeviceID: fmt.Sprintf("dev-%04d", i), Command: "set", Parameters: map[string]any{"on": true}},
			},
		}
		if err := repo.Create(ctx, scene); err != nil {
			b.Fatalf("creating scene %d: %v", i, err)
		}
	}

	reg := NewRegistry(repo)
	if err := reg.RefreshCache(ctx); err != nil {
		b.Fatalf("refreshing cache: %v", err)
	}
	return reg
}

func BenchmarkSceneRegistryGetScene(b *testing.B) {
	reg := setupBenchSceneRegistry(b, 50)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.GetScene(ctx, "scene-0025") //nolint:errcheck // benchmark
	}
}

func BenchmarkSceneRegistryGetSceneBySlug(b *testing.B) {
	reg := setupBenchSceneRegistry(b, 50)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.GetSceneBySlug(ctx, "scene-25") //nolint:errcheck // benchmark
	}
}

func BenchmarkSceneRegistryRefreshCache(b *testing.B) {
	repo := &mockRepository{scenes: make(map[string]*Scene)}
	ctx := context.Background()

	for i := 0; i < 50; i++ {
		scene := &Scene{
			ID:        fmt.Sprintf("scene-%04d", i),
			Name:      fmt.Sprintf("Scene %d", i),
			Slug:      fmt.Sprintf("scene-%d", i),
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.Create(ctx, scene) //nolint:errcheck // setup
	}

	reg := NewRegistry(repo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.RefreshCache(ctx) //nolint:errcheck // benchmark
	}
}
