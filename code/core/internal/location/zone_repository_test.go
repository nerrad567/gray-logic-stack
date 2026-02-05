package location

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupZoneTestDB creates an in-memory SQLite database with zone tables.
func setupZoneTestDB(t *testing.T) *sql.DB {
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

		CREATE TABLE infrastructure_zones (
			id TEXT PRIMARY KEY,
			site_id TEXT NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			domain TEXT NOT NULL,
			settings TEXT NOT NULL DEFAULT '{}',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
			UNIQUE (site_id, slug)
		) STRICT;

		CREATE TABLE infrastructure_zone_rooms (
			zone_id TEXT NOT NULL,
			room_id TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			PRIMARY KEY (zone_id, room_id),
			FOREIGN KEY (zone_id) REFERENCES infrastructure_zones(id) ON DELETE CASCADE,
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
		) STRICT;

		INSERT INTO sites (id, name, slug) VALUES ('site-001', 'Test Home', 'test-home');

		INSERT INTO areas (id, site_id, name, slug, type, sort_order) VALUES
			('area-gf', 'site-001', 'Ground Floor', 'ground-floor', 'floor', 0),
			('area-ff', 'site-001', 'First Floor', 'first-floor', 'floor', 1);

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

func TestZoneRepository_CreateZone(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	zone := &InfrastructureZone{
		ID:     "zone-1",
		SiteID: "site-001",
		Name:   "Ground Floor Climate",
		Slug:   "ground-floor-climate",
		Domain: ZoneDomainClimate,
		Settings: Settings{
			"thermostat_id": "dev-123",
		},
	}

	if err := repo.CreateZone(ctx, zone); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}

	got, err := repo.GetZone(ctx, "zone-1")
	if err != nil {
		t.Fatalf("GetZone() error = %v", err)
	}
	if got.Name != "Ground Floor Climate" {
		t.Errorf("Name = %q, want %q", got.Name, "Ground Floor Climate")
	}
	if got.Domain != ZoneDomainClimate {
		t.Errorf("Domain = %q, want %q", got.Domain, ZoneDomainClimate)
	}
	if got.Settings["thermostat_id"] != "dev-123" {
		t.Errorf("Settings thermostat_id = %v, want dev-123", got.Settings["thermostat_id"])
	}
}

func TestZoneRepository_CreateZone_AudioDomain(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	zone := &InfrastructureZone{
		ID:     "zone-2",
		SiteID: "site-001",
		Name:   "Audio Zone 1",
		Domain: ZoneDomainAudio,
		Settings: Settings{
			"matrix_zone": float64(4),
			"max_volume":  float64(80),
		},
	}

	if err := repo.CreateZone(ctx, zone); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}

	got, err := repo.GetZone(ctx, "zone-2")
	if err != nil {
		t.Fatalf("GetZone() error = %v", err)
	}
	if got.Domain != ZoneDomainAudio {
		t.Errorf("Domain = %q, want %q", got.Domain, ZoneDomainAudio)
	}
	if got.Settings["matrix_zone"] != float64(4) {
		t.Errorf("Settings matrix_zone = %v, want 4", got.Settings["matrix_zone"])
	}
}

func TestZoneRepository_GetZone(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	zone := &InfrastructureZone{
		ID:     "zone-3",
		SiteID: "site-001",
		Name:   "Lighting Zone",
		Domain: ZoneDomainLighting,
		Settings: Settings{
			"gateway_id": "dev-456",
		},
	}

	if err := repo.CreateZone(ctx, zone); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}

	got, err := repo.GetZone(ctx, "zone-3")
	if err != nil {
		t.Fatalf("GetZone() error = %v", err)
	}
	if got.Settings["gateway_id"] != "dev-456" {
		t.Errorf("Settings gateway_id = %v, want dev-456", got.Settings["gateway_id"])
	}
}

func TestZoneRepository_GetZone_NotFound(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)

	_, err := repo.GetZone(context.Background(), "missing")
	if !errors.Is(err, ErrZoneNotFound) {
		t.Errorf("GetZone() error = %v, want ErrZoneNotFound", err)
	}
}

func TestZoneRepository_ListZones(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	zones := []*InfrastructureZone{
		{ID: "zone-a", SiteID: "site-001", Name: "Audio Zone", Domain: ZoneDomainAudio, SortOrder: 2},
		{ID: "zone-b", SiteID: "site-001", Name: "Climate Zone", Domain: ZoneDomainClimate, SortOrder: 1},
		{ID: "zone-c", SiteID: "site-001", Name: "Lighting Zone", Domain: ZoneDomainLighting, SortOrder: 1},
	}

	for _, z := range zones {
		if err := repo.CreateZone(ctx, z); err != nil {
			t.Fatalf("CreateZone() error = %v", err)
		}
	}

	list, err := repo.ListZones(ctx, "site-001")
	if err != nil {
		t.Fatalf("ListZones() error = %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("ListZones length = %d, want 3", len(list))
	}
	if list[0].Domain != ZoneDomainAudio || list[1].Domain != ZoneDomainClimate || list[2].Domain != ZoneDomainLighting {
		t.Errorf("Order by domain incorrect: %v", []ZoneDomain{list[0].Domain, list[1].Domain, list[2].Domain})
	}
}

func TestZoneRepository_ListZonesByDomain(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	if err := repo.CreateZone(ctx, &InfrastructureZone{ID: "zone-1", SiteID: "site-001", Name: "Climate A", Domain: ZoneDomainClimate}); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}
	if err := repo.CreateZone(ctx, &InfrastructureZone{ID: "zone-2", SiteID: "site-001", Name: "Audio A", Domain: ZoneDomainAudio}); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}

	list, err := repo.ListZonesByDomain(ctx, "site-001", ZoneDomainClimate)
	if err != nil {
		t.Fatalf("ListZonesByDomain() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListZonesByDomain length = %d, want 1", len(list))
	}
	if list[0].Domain != ZoneDomainClimate {
		t.Errorf("Domain = %q, want %q", list[0].Domain, ZoneDomainClimate)
	}
}

func TestZoneRepository_UpdateZone(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	zone := &InfrastructureZone{ID: "zone-1", SiteID: "site-001", Name: "Old Name", Domain: ZoneDomainClimate}
	if err := repo.CreateZone(ctx, zone); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}

	zone.Name = "New Name"
	zone.Settings = Settings{"thermostat_id": "dev-999"}
	if err := repo.UpdateZone(ctx, zone); err != nil {
		t.Fatalf("UpdateZone() error = %v", err)
	}

	got, err := repo.GetZone(ctx, "zone-1")
	if err != nil {
		t.Fatalf("GetZone() error = %v", err)
	}
	if got.Name != "New Name" {
		t.Errorf("Name = %q, want %q", got.Name, "New Name")
	}
	if got.Settings["thermostat_id"] != "dev-999" {
		t.Errorf("Settings thermostat_id = %v, want dev-999", got.Settings["thermostat_id"])
	}
}

func TestZoneRepository_DeleteZone(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	zone := &InfrastructureZone{ID: "zone-1", SiteID: "site-001", Name: "Climate Zone", Domain: ZoneDomainClimate}
	if err := repo.CreateZone(ctx, zone); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}
	if err := repo.SetZoneRooms(ctx, "zone-1", []string{"room-living", "room-kitchen"}); err != nil {
		t.Fatalf("SetZoneRooms() error = %v", err)
	}

	if err := repo.DeleteZone(ctx, "zone-1"); err != nil {
		t.Fatalf("DeleteZone() error = %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM infrastructure_zone_rooms").Scan(&count); err != nil {
		t.Fatalf("count rooms error = %v", err)
	}
	if count != 0 {
		t.Errorf("room assignments count = %d, want 0", count)
	}
}

func TestZoneRepository_SetZoneRooms(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	zone := &InfrastructureZone{ID: "zone-1", SiteID: "site-001", Name: "Climate Zone", Domain: ZoneDomainClimate}
	if err := repo.CreateZone(ctx, zone); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}

	if err := repo.SetZoneRooms(ctx, "zone-1", []string{"room-living", "room-kitchen", "room-master"}); err != nil {
		t.Fatalf("SetZoneRooms() error = %v", err)
	}

	rooms, err := repo.GetZoneRooms(ctx, "zone-1")
	if err != nil {
		t.Fatalf("GetZoneRooms() error = %v", err)
	}
	if len(rooms) != 3 {
		t.Fatalf("Rooms length = %d, want 3", len(rooms))
	}
}

func TestZoneRepository_SetZoneRooms_Replace(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	zone := &InfrastructureZone{ID: "zone-1", SiteID: "site-001", Name: "Climate Zone", Domain: ZoneDomainClimate}
	if err := repo.CreateZone(ctx, zone); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}

	if err := repo.SetZoneRooms(ctx, "zone-1", []string{"room-living", "room-kitchen"}); err != nil {
		t.Fatalf("SetZoneRooms() error = %v", err)
	}
	if err := repo.SetZoneRooms(ctx, "zone-1", []string{"room-master"}); err != nil {
		t.Fatalf("SetZoneRooms() replace error = %v", err)
	}

	ids, err := repo.GetZoneRoomIDs(ctx, "zone-1")
	if err != nil {
		t.Fatalf("GetZoneRoomIDs() error = %v", err)
	}
	if !reflect.DeepEqual(ids, []string{"room-master"}) {
		t.Errorf("Room IDs = %v, want %v", ids, []string{"room-master"})
	}
}

func TestZoneRepository_GetZoneRooms(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	zone := &InfrastructureZone{ID: "zone-1", SiteID: "site-001", Name: "Climate Zone", Domain: ZoneDomainClimate}
	if err := repo.CreateZone(ctx, zone); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}
	if err := repo.SetZoneRooms(ctx, "zone-1", []string{"room-living", "room-kitchen"}); err != nil {
		t.Fatalf("SetZoneRooms() error = %v", err)
	}

	rooms, err := repo.GetZoneRooms(ctx, "zone-1")
	if err != nil {
		t.Fatalf("GetZoneRooms() error = %v", err)
	}
	if rooms[0].Name == "" {
		t.Errorf("Room name should be populated")
	}
}

func TestZoneRepository_GetZoneForRoom(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	zone := &InfrastructureZone{ID: "zone-1", SiteID: "site-001", Name: "Climate Zone", Domain: ZoneDomainClimate}
	if err := repo.CreateZone(ctx, zone); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}
	if err := repo.SetZoneRooms(ctx, "zone-1", []string{"room-living"}); err != nil {
		t.Fatalf("SetZoneRooms() error = %v", err)
	}

	got, err := repo.GetZoneForRoom(ctx, "room-living", ZoneDomainClimate)
	if err != nil {
		t.Fatalf("GetZoneForRoom() error = %v", err)
	}
	if got.ID != "zone-1" {
		t.Errorf("Zone ID = %q, want %q", got.ID, "zone-1")
	}
}

func TestZoneRepository_GetZoneForRoom_NoDomain(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)

	_, err := repo.GetZoneForRoom(context.Background(), "room-living", ZoneDomainClimate)
	if !errors.Is(err, ErrZoneNotFound) {
		t.Errorf("GetZoneForRoom() error = %v, want ErrZoneNotFound", err)
	}
}

func TestZoneRepository_GetZonesForRoom(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	climate := &InfrastructureZone{ID: "zone-1", SiteID: "site-001", Name: "Climate Zone", Domain: ZoneDomainClimate}
	audio := &InfrastructureZone{ID: "zone-2", SiteID: "site-001", Name: "Audio Zone", Domain: ZoneDomainAudio}
	if err := repo.CreateZone(ctx, climate); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}
	if err := repo.CreateZone(ctx, audio); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}
	if err := repo.SetZoneRooms(ctx, "zone-1", []string{"room-living"}); err != nil {
		t.Fatalf("SetZoneRooms() error = %v", err)
	}
	if err := repo.SetZoneRooms(ctx, "zone-2", []string{"room-living"}); err != nil {
		t.Fatalf("SetZoneRooms() error = %v", err)
	}

	zones, err := repo.GetZonesForRoom(ctx, "room-living")
	if err != nil {
		t.Fatalf("GetZonesForRoom() error = %v", err)
	}
	if len(zones) != 2 {
		t.Fatalf("Zones length = %d, want 2", len(zones))
	}
}

func TestZoneRepository_OneZonePerDomain(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	zoneA := &InfrastructureZone{ID: "zone-a", SiteID: "site-001", Name: "Climate A", Domain: ZoneDomainClimate}
	zoneB := &InfrastructureZone{ID: "zone-b", SiteID: "site-001", Name: "Climate B", Domain: ZoneDomainClimate}
	if err := repo.CreateZone(ctx, zoneA); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}
	if err := repo.CreateZone(ctx, zoneB); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}
	if err := repo.SetZoneRooms(ctx, "zone-a", []string{"room-living"}); err != nil {
		t.Fatalf("SetZoneRooms() error = %v", err)
	}

	err := repo.SetZoneRooms(ctx, "zone-b", []string{"room-living"})
	if !errors.Is(err, ErrRoomAlreadyInZoneDomain) {
		t.Errorf("SetZoneRooms() error = %v, want ErrRoomAlreadyInZoneDomain", err)
	}
}

func TestZoneRepository_DifferentDomainsAllowed(t *testing.T) {
	db := setupZoneTestDB(t)
	repo := NewSQLiteZoneRepository(db)
	ctx := context.Background()

	climate := &InfrastructureZone{ID: "zone-a", SiteID: "site-001", Name: "Climate", Domain: ZoneDomainClimate}
	audio := &InfrastructureZone{ID: "zone-b", SiteID: "site-001", Name: "Audio", Domain: ZoneDomainAudio}
	if err := repo.CreateZone(ctx, climate); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}
	if err := repo.CreateZone(ctx, audio); err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}
	if err := repo.SetZoneRooms(ctx, "zone-a", []string{"room-living"}); err != nil {
		t.Fatalf("SetZoneRooms() error = %v", err)
	}

	if err := repo.SetZoneRooms(ctx, "zone-b", []string{"room-living"}); err != nil {
		t.Fatalf("SetZoneRooms() error = %v", err)
	}
}
