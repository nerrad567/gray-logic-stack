package device

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupGroupResolverDB creates an in-memory SQLite database with group and tag tables.
func setupGroupResolverDB(t *testing.T, deviceIDs []string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE devices (
			id TEXT PRIMARY KEY
		) STRICT;

		CREATE TABLE device_tags (
			device_id TEXT NOT NULL,
			tag TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			PRIMARY KEY (device_id, tag),
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
		) STRICT;
		CREATE INDEX idx_device_tags_tag ON device_tags(tag);

		CREATE TABLE device_groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			description TEXT,
			type TEXT NOT NULL DEFAULT 'static',
			filter_rules TEXT,
			icon TEXT,
			colour TEXT,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		) STRICT;

		CREATE TABLE device_group_members (
			group_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			PRIMARY KEY (group_id, device_id),
			FOREIGN KEY (group_id) REFERENCES device_groups(id) ON DELETE CASCADE,
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
		) STRICT;
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		t.Fatalf("failed to create test schema: %v", err)
	}

	if len(deviceIDs) > 0 {
		values := make([]any, 0, len(deviceIDs))
		placeholders := make([]string, 0, len(deviceIDs))
		for _, id := range deviceIDs {
			placeholders = append(placeholders, "(?)")
			values = append(values, id)
		}
		query := "INSERT INTO devices (id) VALUES " + stringsJoin(placeholders, ",")
		if _, err := db.Exec(query, values...); err != nil {
			db.Close()
			t.Fatalf("failed to seed devices: %v", err)
		}
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// stringsJoin joins strings without importing extra packages in test helpers.
func stringsJoin(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}

	joined := parts[0]
	for i := 1; i < len(parts); i++ {
		joined += sep + parts[i]
	}
	return joined
}

func setupResolverRegistry(t *testing.T, devices []*Device, tagRepo TagRepository) *Registry {
	t.Helper()

	repo := NewMockRepository()
	for _, d := range devices {
		repo.addDevice(d)
	}

	registry := NewRegistry(repo)
	registry.SetTagRepository(tagRepo)
	if err := registry.RefreshCache(context.Background()); err != nil {
		t.Fatalf("RefreshCache() error = %v", err)
	}

	return registry
}

func baseDevices() []*Device {
	areaWest := "area-west"
	areaEast := "area-east"
	roomOne := "room-1"
	roomTwo := "room-2"
	roomThree := "room-3"

	accent := testDevice("dev-a", "Accent Light")
	accent.AreaID = &areaWest
	accent.RoomID = &roomOne
	accent.Type = DeviceTypeLightDimmer
	accent.Domain = DomainLighting
	accent.Capabilities = []Capability{CapOnOff, CapDim}

	escape := testDevice("dev-b", "Escape Light")
	escape.AreaID = &areaWest
	escape.RoomID = &roomTwo
	escape.Type = DeviceTypeLightSwitch
	escape.Domain = DomainLighting
	escape.Capabilities = []Capability{CapOnOff}

	thermo := testDevice("dev-c", "Main Thermostat")
	thermo.AreaID = &areaEast
	thermo.RoomID = &roomThree
	thermo.Type = DeviceTypeThermostat
	thermo.Domain = DomainClimate
	thermo.Capabilities = []Capability{CapTemperatureRead, CapTemperatureSet}
	thermo.Protocol = ProtocolModbusTCP
	thermo.Address = Address{"host": "1.1.1.1", "unit_id": 1}

	rgb := testDevice("dev-d", "RGB Light")
	rgb.AreaID = &areaEast
	rgb.RoomID = &roomThree
	rgb.Type = DeviceTypeLightRGBW
	rgb.Domain = DomainLighting
	rgb.Capabilities = []Capability{CapOnOff, CapDim, CapColorRGB}

	return []*Device{accent, escape, thermo, rgb}
}

func seedResolverTags(t *testing.T, tagRepo TagRepository) {
	t.Helper()
	ctx := context.Background()

	if err := tagRepo.SetTags(ctx, "dev-a", []string{"accent"}); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}
	if err := tagRepo.SetTags(ctx, "dev-b", []string{"escape_lighting"}); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}
	if err := tagRepo.SetTags(ctx, "dev-d", []string{"accent", "feature"}); err != nil {
		t.Fatalf("SetTags() error = %v", err)
	}
}

func deviceIDsFromSlice(devices []Device) []string {
	ids := make([]string, 0, len(devices))
	for _, d := range devices {
		ids = append(ids, d.ID)
	}
	return ids
}

func TestResolveGroup_Static(t *testing.T) {
	devices := baseDevices()
	deviceIDs := []string{"dev-a", "dev-b", "dev-c", "dev-d"}

	db := setupGroupResolverDB(t, deviceIDs)
	tagRepo := NewSQLiteTagRepository(db)
	groupRepo := NewSQLiteGroupRepository(db)
	seedResolverTags(t, tagRepo)

	group := &DeviceGroup{ID: "grp-1", Name: "Static", Slug: "static", Type: GroupTypeStatic}
	if err := groupRepo.Create(context.Background(), group); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := groupRepo.SetMembers(context.Background(), group.ID, []string{"dev-a", "dev-b", "dev-c"}); err != nil {
		t.Fatalf("SetMembers() error = %v", err)
	}

	registry := setupResolverRegistry(t, devices, tagRepo)

	resolved, err := ResolveGroup(context.Background(), group, registry, tagRepo, groupRepo)
	if err != nil {
		t.Fatalf("ResolveGroup() error = %v", err)
	}

	if !reflect.DeepEqual(deviceIDsFromSlice(resolved), []string{"dev-a", "dev-b", "dev-c"}) {
		t.Errorf("Resolved IDs = %v, want %v", deviceIDsFromSlice(resolved), []string{"dev-a", "dev-b", "dev-c"})
	}
}

func TestResolveGroup_Dynamic_ByArea(t *testing.T) {
	devices := baseDevices()
	db := setupGroupResolverDB(t, []string{"dev-a", "dev-b", "dev-c", "dev-d"})
	tagRepo := NewSQLiteTagRepository(db)
	seedResolverTags(t, tagRepo)
	groupRepo := NewSQLiteGroupRepository(db)

	group := &DeviceGroup{
		ID:   "grp-2",
		Name: "Area West",
		Slug: "area-west",
		Type: GroupTypeDynamic,
		FilterRules: &FilterRules{
			ScopeType: "area",
			ScopeID:   "area-west",
		},
	}

	registry := setupResolverRegistry(t, devices, tagRepo)
	resolved, err := ResolveGroup(context.Background(), group, registry, tagRepo, groupRepo)
	if err != nil {
		t.Fatalf("ResolveGroup() error = %v", err)
	}

	if !reflect.DeepEqual(deviceIDsFromSlice(resolved), []string{"dev-a", "dev-b"}) {
		t.Errorf("Resolved IDs = %v, want %v", deviceIDsFromSlice(resolved), []string{"dev-a", "dev-b"})
	}
}

func TestResolveGroup_Dynamic_ByDomain(t *testing.T) {
	devices := baseDevices()
	db := setupGroupResolverDB(t, []string{"dev-a", "dev-b", "dev-c", "dev-d"})
	tagRepo := NewSQLiteTagRepository(db)
	seedResolverTags(t, tagRepo)
	groupRepo := NewSQLiteGroupRepository(db)

	group := &DeviceGroup{
		ID:   "grp-3",
		Name: "Lighting",
		Slug: "lighting",
		Type: GroupTypeDynamic,
		FilterRules: &FilterRules{
			ScopeType: "site",
			Domains:   []string{"lighting"},
		},
	}

	registry := setupResolverRegistry(t, devices, tagRepo)
	resolved, err := ResolveGroup(context.Background(), group, registry, tagRepo, groupRepo)
	if err != nil {
		t.Fatalf("ResolveGroup() error = %v", err)
	}

	if !reflect.DeepEqual(deviceIDsFromSlice(resolved), []string{"dev-a", "dev-b", "dev-d"}) {
		t.Errorf("Resolved IDs = %v, want %v", deviceIDsFromSlice(resolved), []string{"dev-a", "dev-b", "dev-d"})
	}
}

func TestResolveGroup_Dynamic_ByTag(t *testing.T) {
	devices := baseDevices()
	db := setupGroupResolverDB(t, []string{"dev-a", "dev-b", "dev-c", "dev-d"})
	tagRepo := NewSQLiteTagRepository(db)
	seedResolverTags(t, tagRepo)
	groupRepo := NewSQLiteGroupRepository(db)

	group := &DeviceGroup{
		ID:   "grp-4",
		Name: "Accent",
		Slug: "accent",
		Type: GroupTypeDynamic,
		FilterRules: &FilterRules{
			ScopeType: "site",
			Tags:      []string{"accent"},
		},
	}

	registry := setupResolverRegistry(t, devices, tagRepo)
	resolved, err := ResolveGroup(context.Background(), group, registry, tagRepo, groupRepo)
	if err != nil {
		t.Fatalf("ResolveGroup() error = %v", err)
	}

	if !reflect.DeepEqual(deviceIDsFromSlice(resolved), []string{"dev-a", "dev-d"}) {
		t.Errorf("Resolved IDs = %v, want %v", deviceIDsFromSlice(resolved), []string{"dev-a", "dev-d"})
	}
}

func TestResolveGroup_Dynamic_ExcludeTags(t *testing.T) {
	devices := baseDevices()
	db := setupGroupResolverDB(t, []string{"dev-a", "dev-b", "dev-c", "dev-d"})
	tagRepo := NewSQLiteTagRepository(db)
	seedResolverTags(t, tagRepo)
	groupRepo := NewSQLiteGroupRepository(db)

	group := &DeviceGroup{
		ID:   "grp-5",
		Name: "Lighting Excluding Escape",
		Slug: "lighting-excluding",
		Type: GroupTypeDynamic,
		FilterRules: &FilterRules{
			ScopeType:   "site",
			Domains:     []string{"lighting"},
			ExcludeTags: []string{"escape_lighting"},
		},
	}

	registry := setupResolverRegistry(t, devices, tagRepo)
	resolved, err := ResolveGroup(context.Background(), group, registry, tagRepo, groupRepo)
	if err != nil {
		t.Fatalf("ResolveGroup() error = %v", err)
	}

	if !reflect.DeepEqual(deviceIDsFromSlice(resolved), []string{"dev-a", "dev-d"}) {
		t.Errorf("Resolved IDs = %v, want %v", deviceIDsFromSlice(resolved), []string{"dev-a", "dev-d"})
	}
}

func TestResolveGroup_Dynamic_MultipleFilters(t *testing.T) {
	devices := baseDevices()
	db := setupGroupResolverDB(t, []string{"dev-a", "dev-b", "dev-c", "dev-d"})
	tagRepo := NewSQLiteTagRepository(db)
	seedResolverTags(t, tagRepo)
	groupRepo := NewSQLiteGroupRepository(db)

	group := &DeviceGroup{
		ID:   "grp-6",
		Name: "West Wing Dimmers",
		Slug: "west-wing-dimmers",
		Type: GroupTypeDynamic,
		FilterRules: &FilterRules{
			ScopeType:    "area",
			ScopeID:      "area-west",
			Domains:      []string{"lighting"},
			Capabilities: []string{"on_off", "dim"},
		},
	}

	registry := setupResolverRegistry(t, devices, tagRepo)
	resolved, err := ResolveGroup(context.Background(), group, registry, tagRepo, groupRepo)
	if err != nil {
		t.Fatalf("ResolveGroup() error = %v", err)
	}

	if !reflect.DeepEqual(deviceIDsFromSlice(resolved), []string{"dev-a"}) {
		t.Errorf("Resolved IDs = %v, want %v", deviceIDsFromSlice(resolved), []string{"dev-a"})
	}
}

func TestResolveGroup_Hybrid(t *testing.T) {
	devices := baseDevices()
	db := setupGroupResolverDB(t, []string{"dev-a", "dev-b", "dev-c", "dev-d"})
	tagRepo := NewSQLiteTagRepository(db)
	seedResolverTags(t, tagRepo)
	groupRepo := NewSQLiteGroupRepository(db)

	group := &DeviceGroup{ID: "grp-7", Name: "Hybrid", Slug: "hybrid", Type: GroupTypeHybrid, FilterRules: &FilterRules{
		ScopeType: "area",
		ScopeID:   "area-west",
	}}
	if err := groupRepo.Create(context.Background(), group); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := groupRepo.SetMembers(context.Background(), group.ID, []string{"dev-d"}); err != nil {
		t.Fatalf("SetMembers() error = %v", err)
	}

	registry := setupResolverRegistry(t, devices, tagRepo)
	resolved, err := ResolveGroup(context.Background(), group, registry, tagRepo, groupRepo)
	if err != nil {
		t.Fatalf("ResolveGroup() error = %v", err)
	}

	if !reflect.DeepEqual(deviceIDsFromSlice(resolved), []string{"dev-a", "dev-b", "dev-d"}) {
		t.Errorf("Resolved IDs = %v, want %v", deviceIDsFromSlice(resolved), []string{"dev-a", "dev-b", "dev-d"})
	}
}

func TestResolveGroup_Dynamic_ByDeviceType(t *testing.T) {
	devices := baseDevices()
	db := setupGroupResolverDB(t, []string{"dev-a", "dev-b", "dev-c", "dev-d"})
	tagRepo := NewSQLiteTagRepository(db)
	seedResolverTags(t, tagRepo)
	groupRepo := NewSQLiteGroupRepository(db)

	group := &DeviceGroup{ID: "grp-8", Name: "RGB", Slug: "rgb", Type: GroupTypeDynamic, FilterRules: &FilterRules{
		ScopeType:   "site",
		DeviceTypes: []string{"light_rgbw"},
	}}

	registry := setupResolverRegistry(t, devices, tagRepo)
	resolved, err := ResolveGroup(context.Background(), group, registry, tagRepo, groupRepo)
	if err != nil {
		t.Fatalf("ResolveGroup() error = %v", err)
	}

	if !reflect.DeepEqual(deviceIDsFromSlice(resolved), []string{"dev-d"}) {
		t.Errorf("Resolved IDs = %v, want %v", deviceIDsFromSlice(resolved), []string{"dev-d"})
	}
}

func TestResolveGroup_Dynamic_ByCapability(t *testing.T) {
	devices := baseDevices()
	db := setupGroupResolverDB(t, []string{"dev-a", "dev-b", "dev-c", "dev-d"})
	tagRepo := NewSQLiteTagRepository(db)
	seedResolverTags(t, tagRepo)
	groupRepo := NewSQLiteGroupRepository(db)

	group := &DeviceGroup{ID: "grp-9", Name: "Dimmer", Slug: "dimmer", Type: GroupTypeDynamic, FilterRules: &FilterRules{
		ScopeType:    "site",
		Capabilities: []string{"on_off", "dim"},
	}}

	registry := setupResolverRegistry(t, devices, tagRepo)
	resolved, err := ResolveGroup(context.Background(), group, registry, tagRepo, groupRepo)
	if err != nil {
		t.Fatalf("ResolveGroup() error = %v", err)
	}

	if !reflect.DeepEqual(deviceIDsFromSlice(resolved), []string{"dev-a", "dev-d"}) {
		t.Errorf("Resolved IDs = %v, want %v", deviceIDsFromSlice(resolved), []string{"dev-a", "dev-d"})
	}
}

func TestResolveGroup_Empty(t *testing.T) {
	devices := baseDevices()
	db := setupGroupResolverDB(t, []string{"dev-a", "dev-b", "dev-c", "dev-d"})
	tagRepo := NewSQLiteTagRepository(db)
	groupRepo := NewSQLiteGroupRepository(db)

	group := &DeviceGroup{ID: "grp-10", Name: "Empty", Slug: "empty", Type: GroupTypeDynamic, FilterRules: &FilterRules{
		ScopeType: "area",
		ScopeID:   "area-missing",
	}}

	registry := setupResolverRegistry(t, devices, tagRepo)
	resolved, err := ResolveGroup(context.Background(), group, registry, tagRepo, groupRepo)
	if err != nil {
		t.Fatalf("ResolveGroup() error = %v", err)
	}

	if resolved == nil {
		t.Fatal("Resolved slice is nil")
	}
	if len(resolved) != 0 {
		t.Errorf("Resolved length = %d, want 0", len(resolved))
	}
}

func TestResolveGroup_ExcludeRemovesExplicitMembers(t *testing.T) {
	devices := baseDevices()
	db := setupGroupResolverDB(t, []string{"dev-a", "dev-b", "dev-c", "dev-d"})
	tagRepo := NewSQLiteTagRepository(db)
	seedResolverTags(t, tagRepo)
	groupRepo := NewSQLiteGroupRepository(db)

	group := &DeviceGroup{
		ID:   "grp-11",
		Name: "Hybrid Exclude",
		Slug: "hybrid-exclude",
		Type: GroupTypeHybrid,
		FilterRules: &FilterRules{
			ScopeType:   "site",
			Domains:     []string{"lighting"},
			ExcludeTags: []string{"escape_lighting"},
		},
	}
	if err := groupRepo.Create(context.Background(), group); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := groupRepo.SetMembers(context.Background(), group.ID, []string{"dev-b", "dev-a"}); err != nil {
		t.Fatalf("SetMembers() error = %v", err)
	}

	registry := setupResolverRegistry(t, devices, tagRepo)
	resolved, err := ResolveGroup(context.Background(), group, registry, tagRepo, groupRepo)
	if err != nil {
		t.Fatalf("ResolveGroup() error = %v", err)
	}

	if !reflect.DeepEqual(deviceIDsFromSlice(resolved), []string{"dev-a", "dev-d"}) {
		t.Errorf("Resolved IDs = %v, want %v", deviceIDsFromSlice(resolved), []string{"dev-a", "dev-d"})
	}
}
