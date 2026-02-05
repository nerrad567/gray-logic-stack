package device

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupTagTestDB creates an in-memory SQLite database with devices and device_tags tables.
func setupTagTestDB(t *testing.T) *sql.DB {
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

		CREATE TABLE device_tags (
			device_id TEXT NOT NULL,
			tag TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			PRIMARY KEY (device_id, tag),
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
		) STRICT;
		CREATE INDEX idx_device_tags_tag ON device_tags(tag);
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

func TestTagRepository_SetTags(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	deviceRepo := NewSQLiteRepository(db)
	tagRepo := NewSQLiteTagRepository(db)

	if err := deviceRepo.Create(ctx, testDevice("dev-1", "Device One")); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := tagRepo.SetTags(ctx, "dev-1", []string{"accent", "escape_lighting"}); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}

	tags, err := tagRepo.GetTags(ctx, "dev-1")
	if err != nil {
		t.Fatalf("GetTags() error = %v", err)
	}
	if !reflect.DeepEqual(tags, []string{"accent", "escape_lighting"}) {
		t.Errorf("Tags = %v, want %v", tags, []string{"accent", "escape_lighting"})
	}

	if err = tagRepo.SetTags(ctx, "dev-1", []string{"entertainment"}); err != nil {
		t.Fatalf("SetTags() replace error = %v", err)
	}

	tags, err = tagRepo.GetTags(ctx, "dev-1")
	if err != nil {
		t.Fatalf("GetTags() error = %v", err)
	}
	if !reflect.DeepEqual(tags, []string{"entertainment"}) {
		t.Errorf("Tags after replace = %v, want %v", tags, []string{"entertainment"})
	}
}

func TestTagRepository_SetTags_Empty(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	deviceRepo := NewSQLiteRepository(db)
	tagRepo := NewSQLiteTagRepository(db)

	if err := deviceRepo.Create(ctx, testDevice("dev-1", "Device One")); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := tagRepo.SetTags(ctx, "dev-1", []string{"accent"}); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}

	if err := tagRepo.SetTags(ctx, "dev-1", []string{}); err != nil {
		t.Fatalf("SetTags(empty) error = %v", err)
	}

	tags, err := tagRepo.GetTags(ctx, "dev-1")
	if err != nil {
		t.Fatalf("GetTags() error = %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("Tags length = %d, want 0", len(tags))
	}
}

func TestTagRepository_AddTag(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	deviceRepo := NewSQLiteRepository(db)
	tagRepo := NewSQLiteTagRepository(db)

	if err := deviceRepo.Create(ctx, testDevice("dev-1", "Device One")); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := tagRepo.AddTag(ctx, "dev-1", "accent"); err != nil {
		t.Fatalf("AddTag() error = %v", err)
	}
	if err := tagRepo.AddTag(ctx, "dev-1", "accent"); err != nil {
		t.Fatalf("AddTag(idempotent) error = %v", err)
	}

	tags, err := tagRepo.GetTags(ctx, "dev-1")
	if err != nil {
		t.Fatalf("GetTags() error = %v", err)
	}
	if !reflect.DeepEqual(tags, []string{"accent"}) {
		t.Errorf("Tags = %v, want %v", tags, []string{"accent"})
	}
}

func TestTagRepository_RemoveTag(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	deviceRepo := NewSQLiteRepository(db)
	tagRepo := NewSQLiteTagRepository(db)

	if err := deviceRepo.Create(ctx, testDevice("dev-1", "Device One")); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := tagRepo.SetTags(ctx, "dev-1", []string{"accent", "task"}); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}

	if err := tagRepo.RemoveTag(ctx, "dev-1", "accent"); err != nil {
		t.Fatalf("RemoveTag() error = %v", err)
	}

	tags, err := tagRepo.GetTags(ctx, "dev-1")
	if err != nil {
		t.Fatalf("GetTags() error = %v", err)
	}
	if !reflect.DeepEqual(tags, []string{"task"}) {
		t.Errorf("Tags after remove = %v, want %v", tags, []string{"task"})
	}
}

func TestTagRepository_ListDevicesByTag(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	deviceRepo := NewSQLiteRepository(db)
	tagRepo := NewSQLiteTagRepository(db)

	for _, id := range []string{"dev-1", "dev-2", "dev-3"} {
		if err := deviceRepo.Create(ctx, testDevice(id, "Device "+id)); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	if err := tagRepo.AddTag(ctx, "dev-1", "accent"); err != nil {
		t.Fatalf("AddTag() error = %v", err)
	}
	if err := tagRepo.AddTag(ctx, "dev-2", "accent"); err != nil {
		t.Fatalf("AddTag() error = %v", err)
	}
	if err := tagRepo.AddTag(ctx, "dev-3", "task"); err != nil {
		t.Fatalf("AddTag() error = %v", err)
	}

	ids, err := tagRepo.ListDevicesByTag(ctx, "accent")
	if err != nil {
		t.Fatalf("ListDevicesByTag() error = %v", err)
	}
	if !reflect.DeepEqual(ids, []string{"dev-1", "dev-2"}) {
		t.Errorf("Device IDs = %v, want %v", ids, []string{"dev-1", "dev-2"})
	}
}

func TestTagRepository_ListAllTags(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	deviceRepo := NewSQLiteRepository(db)
	tagRepo := NewSQLiteTagRepository(db)

	if err := deviceRepo.Create(ctx, testDevice("dev-1", "Device One")); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := deviceRepo.Create(ctx, testDevice("dev-2", "Device Two")); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := tagRepo.SetTags(ctx, "dev-1", []string{"accent", "escape_lighting"}); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}
	if err := tagRepo.SetTags(ctx, "dev-2", []string{"accent", "task"}); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}

	tags, err := tagRepo.ListAllTags(ctx)
	if err != nil {
		t.Fatalf("ListAllTags() error = %v", err)
	}
	if !reflect.DeepEqual(tags, []string{"accent", "escape_lighting", "task"}) {
		t.Errorf("Tags = %v, want %v", tags, []string{"accent", "escape_lighting", "task"})
	}
}

func TestTagRepository_GetTagsForDevices(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	deviceRepo := NewSQLiteRepository(db)
	tagRepo := NewSQLiteTagRepository(db)

	for _, id := range []string{"dev-1", "dev-2", "dev-3"} {
		if err := deviceRepo.Create(ctx, testDevice(id, "Device "+id)); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	if err := tagRepo.SetTags(ctx, "dev-1", []string{"accent", "task"}); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}
	if err := tagRepo.SetTags(ctx, "dev-2", []string{"escape_lighting"}); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}

	results, err := tagRepo.GetTagsForDevices(ctx, []string{"dev-1", "dev-2", "dev-3"})
	if err != nil {
		t.Fatalf("GetTagsForDevices() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Results length = %d, want 2", len(results))
	}

	if !reflect.DeepEqual(results["dev-1"], []string{"accent", "task"}) {
		t.Errorf("dev-1 tags = %v, want %v", results["dev-1"], []string{"accent", "task"})
	}
	if !reflect.DeepEqual(results["dev-2"], []string{"escape_lighting"}) {
		t.Errorf("dev-2 tags = %v, want %v", results["dev-2"], []string{"escape_lighting"})
	}
	if _, ok := results["dev-3"]; ok {
		t.Errorf("dev-3 tags should be absent")
	}
}

func TestTagRepository_GetTagsForDevices_Empty(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	tagRepo := NewSQLiteTagRepository(db)

	results, err := tagRepo.GetTagsForDevices(ctx, []string{})
	if err != nil {
		t.Fatalf("GetTagsForDevices() error = %v", err)
	}
	if results == nil {
		t.Fatal("Results map is nil")
	}
	if len(results) != 0 {
		t.Errorf("Results length = %d, want 0", len(results))
	}
}

func TestTagRepository_Normalisation(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	deviceRepo := NewSQLiteRepository(db)
	tagRepo := NewSQLiteTagRepository(db)

	if err := deviceRepo.Create(ctx, testDevice("dev-1", "Device One")); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := tagRepo.SetTags(ctx, "dev-1", []string{"  Accent ", "Escape_Lighting"}); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}

	tags, err := tagRepo.GetTags(ctx, "dev-1")
	if err != nil {
		t.Fatalf("GetTags() error = %v", err)
	}
	if !reflect.DeepEqual(tags, []string{"accent", "escape_lighting"}) {
		t.Errorf("Tags = %v, want %v", tags, []string{"accent", "escape_lighting"})
	}
}

func TestRegistry_GetDevicesByTag(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	deviceRepo := NewSQLiteRepository(db)
	tagRepo := NewSQLiteTagRepository(db)

	deviceOne := testDevice("dev-1", "Alpha Light")
	deviceTwo := testDevice("dev-2", "Bravo Light")
	if err := deviceRepo.Create(ctx, deviceOne); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := deviceRepo.Create(ctx, deviceTwo); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := tagRepo.SetTags(ctx, "dev-1", []string{"accent"}); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}
	if err := tagRepo.SetTags(ctx, "dev-2", []string{"task"}); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}

	registry := NewRegistry(deviceRepo)
	registry.SetTagRepository(tagRepo)
	if err := registry.RefreshCache(ctx); err != nil {
		t.Fatalf("RefreshCache() error = %v", err)
	}

	devices, err := registry.GetDevicesByTag(ctx, "accent")
	if err != nil {
		t.Fatalf("GetDevicesByTag() error = %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("Devices length = %d, want 1", len(devices))
	}
	if devices[0].ID != "dev-1" {
		t.Errorf("Device ID = %q, want %q", devices[0].ID, "dev-1")
	}
}
