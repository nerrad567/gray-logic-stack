package location

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database with the areas and rooms tables.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE sites (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			timezone TEXT NOT NULL DEFAULT 'UTC',
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		) STRICT;

		CREATE TABLE areas (
			id TEXT PRIMARY KEY,
			site_id TEXT NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'floor',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
			UNIQUE (site_id, slug)
		) STRICT;

		CREATE TABLE rooms (
			id TEXT PRIMARY KEY,
			area_id TEXT NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'other',
			sort_order INTEGER NOT NULL DEFAULT 0,
			climate_zone_id TEXT,
			audio_zone_id TEXT,
			settings TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			FOREIGN KEY (area_id) REFERENCES areas(id) ON DELETE CASCADE,
			UNIQUE (area_id, slug)
		) STRICT;

		INSERT INTO sites (id, name, slug) VALUES ('site-001', 'Test Home', 'test-home');

		INSERT INTO areas (id, site_id, name, slug, type, sort_order) VALUES
			('area-gf', 'site-001', 'Ground Floor', 'ground-floor', 'floor', 0),
			('area-ff', 'site-001', 'First Floor', 'first-floor', 'floor', 1),
			('area-ext', 'site-001', 'External', 'external', 'outdoor', 2);

		INSERT INTO rooms (id, area_id, name, slug, type, sort_order) VALUES
			('room-living', 'area-gf', 'Living Room', 'living-room', 'living', 0),
			('room-kitchen', 'area-gf', 'Kitchen', 'kitchen', 'kitchen', 1),
			('room-master', 'area-ff', 'Master Bedroom', 'master-bedroom', 'bedroom', 0),
			('room-bath', 'area-ff', 'Bathroom', 'bathroom', 'bathroom', 1);
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

func TestListAreas(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	areas, err := repo.ListAreas(context.Background())
	if err != nil {
		t.Fatalf("ListAreas: %v", err)
	}

	if len(areas) != 3 {
		t.Fatalf("expected 3 areas, got %d", len(areas))
	}

	// Should be sorted by sort_order
	if areas[0].Name != "Ground Floor" {
		t.Errorf("first area: got %q, want %q", areas[0].Name, "Ground Floor")
	}
	if areas[1].Name != "First Floor" {
		t.Errorf("second area: got %q, want %q", areas[1].Name, "First Floor")
	}
	if areas[2].Name != "External" {
		t.Errorf("third area: got %q, want %q", areas[2].Name, "External")
	}
}

func TestListAreasBySite(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	areas, err := repo.ListAreasBySite(context.Background(), "site-001")
	if err != nil {
		t.Fatalf("ListAreasBySite: %v", err)
	}
	if len(areas) != 3 {
		t.Fatalf("expected 3 areas for site-001, got %d", len(areas))
	}

	// Non-existent site returns empty
	areas, err = repo.ListAreasBySite(context.Background(), "site-999")
	if err != nil {
		t.Fatalf("ListAreasBySite non-existent: %v", err)
	}
	if len(areas) != 0 {
		t.Errorf("expected 0 areas for site-999, got %d", len(areas))
	}
}

func TestGetArea(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	area, err := repo.GetArea(context.Background(), "area-gf")
	if err != nil {
		t.Fatalf("GetArea: %v", err)
	}
	if area.Name != "Ground Floor" {
		t.Errorf("area name: got %q, want %q", area.Name, "Ground Floor")
	}
	if area.Type != "floor" {
		t.Errorf("area type: got %q, want %q", area.Type, "floor")
	}
	if area.SiteID != "site-001" {
		t.Errorf("area site_id: got %q, want %q", area.SiteID, "site-001")
	}
}

func TestGetAreaNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	_, err := repo.GetArea(context.Background(), "area-nope")
	if err != ErrAreaNotFound {
		t.Errorf("expected ErrAreaNotFound, got %v", err)
	}
}

func TestListRooms(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	rooms, err := repo.ListRooms(context.Background())
	if err != nil {
		t.Fatalf("ListRooms: %v", err)
	}
	if len(rooms) != 4 {
		t.Fatalf("expected 4 rooms, got %d", len(rooms))
	}
}

func TestListRoomsByArea(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	rooms, err := repo.ListRoomsByArea(context.Background(), "area-gf")
	if err != nil {
		t.Fatalf("ListRoomsByArea: %v", err)
	}
	if len(rooms) != 2 {
		t.Fatalf("expected 2 rooms for area-gf, got %d", len(rooms))
	}

	// Verify sort order
	if rooms[0].Name != "Living Room" {
		t.Errorf("first room: got %q, want %q", rooms[0].Name, "Living Room")
	}
	if rooms[1].Name != "Kitchen" {
		t.Errorf("second room: got %q, want %q", rooms[1].Name, "Kitchen")
	}

	// First floor
	rooms, err = repo.ListRoomsByArea(context.Background(), "area-ff")
	if err != nil {
		t.Fatalf("ListRoomsByArea first floor: %v", err)
	}
	if len(rooms) != 2 {
		t.Fatalf("expected 2 rooms for area-ff, got %d", len(rooms))
	}

	// Non-existent area returns empty
	rooms, err = repo.ListRoomsByArea(context.Background(), "area-nope")
	if err != nil {
		t.Fatalf("ListRoomsByArea non-existent: %v", err)
	}
	if len(rooms) != 0 {
		t.Errorf("expected 0 rooms for area-nope, got %d", len(rooms))
	}
}

func TestGetRoom(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	room, err := repo.GetRoom(context.Background(), "room-living")
	if err != nil {
		t.Fatalf("GetRoom: %v", err)
	}
	if room.Name != "Living Room" {
		t.Errorf("room name: got %q, want %q", room.Name, "Living Room")
	}
	if room.AreaID != "area-gf" {
		t.Errorf("room area_id: got %q, want %q", room.AreaID, "area-gf")
	}
	if room.Type != "living" {
		t.Errorf("room type: got %q, want %q", room.Type, "living")
	}
	if room.Settings == nil {
		t.Error("room settings should not be nil")
	}
}

func TestGetRoomNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRepository(db)

	_, err := repo.GetRoom(context.Background(), "room-nope")
	if err != ErrRoomNotFound {
		t.Errorf("expected ErrRoomNotFound, got %v", err)
	}
}

func TestRoomNullableFields(t *testing.T) {
	db := setupTestDB(t)

	// Insert a room with nullable fields set
	_, err := db.Exec(`INSERT INTO rooms (id, area_id, name, slug, type, sort_order, climate_zone_id, audio_zone_id, settings)
		VALUES ('room-special', 'area-gf', 'Special Room', 'special-room', 'other', 5, 'climate-z1', 'audio-z2', '{"brightness": 80}')`)
	if err != nil {
		t.Fatalf("insert room: %v", err)
	}

	repo := NewSQLiteRepository(db)
	room, err := repo.GetRoom(context.Background(), "room-special")
	if err != nil {
		t.Fatalf("GetRoom: %v", err)
	}

	if room.ClimateZoneID == nil || *room.ClimateZoneID != "climate-z1" {
		t.Errorf("climate_zone_id: got %v, want %q", room.ClimateZoneID, "climate-z1")
	}
	if room.AudioZoneID == nil || *room.AudioZoneID != "audio-z2" {
		t.Errorf("audio_zone_id: got %v, want %q", room.AudioZoneID, "audio-z2")
	}
	if room.Settings["brightness"] != float64(80) {
		t.Errorf("settings.brightness: got %v, want 80", room.Settings["brightness"])
	}
}
