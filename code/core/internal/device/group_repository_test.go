package device

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupGroupTestDB creates an in-memory SQLite database with group tables.
func setupGroupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE devices (
			id TEXT PRIMARY KEY
		) STRICT;

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
		CREATE INDEX idx_device_group_members_device ON device_group_members(device_id);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		t.Fatalf("failed to create test schema: %v", err)
	}

	if _, err := db.Exec("INSERT INTO devices (id) VALUES ('dev-1'), ('dev-2'), ('dev-3')"); err != nil {
		db.Close()
		t.Fatalf("failed to seed devices: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestGroupRepository_Create(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewSQLiteGroupRepository(db)
	ctx := context.Background()

	group := &DeviceGroup{
		Name:      "Cinema System",
		Type:      GroupTypeStatic,
		SortOrder: 1,
	}

	if err := repo.Create(ctx, group); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByID(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Name != "Cinema System" {
		t.Errorf("Name = %q, want %q", got.Name, "Cinema System")
	}
	if got.Slug != "cinema-system" {
		t.Errorf("Slug = %q, want %q", got.Slug, "cinema-system")
	}
	if got.Type != GroupTypeStatic {
		t.Errorf("Type = %q, want %q", got.Type, GroupTypeStatic)
	}
	if got.SortOrder != 1 {
		t.Errorf("SortOrder = %d, want 1", got.SortOrder)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Error("CreatedAt/UpdatedAt should be set")
	}
}

func TestGroupRepository_Create_WithFilterRules(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewSQLiteGroupRepository(db)
	ctx := context.Background()

	group := &DeviceGroup{
		Name: "West Wing Lighting",
		Type: GroupTypeDynamic,
		FilterRules: &FilterRules{
			ScopeType: "area",
			ScopeID:   "area-west",
			Domains:   []string{"lighting"},
			Tags:      []string{"accent"},
		},
	}

	if err := repo.Create(ctx, group); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByID(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.FilterRules == nil {
		t.Fatal("FilterRules should not be nil")
	}
	if got.FilterRules.ScopeType != "area" {
		t.Errorf("ScopeType = %q, want %q", got.FilterRules.ScopeType, "area")
	}
	if !reflect.DeepEqual(got.FilterRules.Tags, []string{"accent"}) {
		t.Errorf("Tags = %v, want %v", got.FilterRules.Tags, []string{"accent"})
	}
}

func TestGroupRepository_GetByID(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewSQLiteGroupRepository(db)
	ctx := context.Background()

	group := &DeviceGroup{Name: "Group A", Type: GroupTypeStatic}
	if err := repo.Create(ctx, group); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByID(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.ID != group.ID {
		t.Errorf("ID = %q, want %q", got.ID, group.ID)
	}
}

func TestGroupRepository_GetByID_NotFound(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewSQLiteGroupRepository(db)

	_, err := repo.GetByID(context.Background(), "missing")
	if !errors.Is(err, ErrGroupNotFound) {
		t.Errorf("GetByID() error = %v, want ErrGroupNotFound", err)
	}
}

func TestGroupRepository_List(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewSQLiteGroupRepository(db)
	ctx := context.Background()

	groups := []*DeviceGroup{
		{Name: "Bravo", SortOrder: 2, Type: GroupTypeStatic},
		{Name: "Alpha", SortOrder: 1, Type: GroupTypeStatic},
		{Name: "Charlie", SortOrder: 2, Type: GroupTypeStatic},
	}
	for _, g := range groups {
		if err := repo.Create(ctx, g); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("List() length = %d, want 3", len(list))
	}
	if list[0].Name != "Alpha" || list[1].Name != "Bravo" || list[2].Name != "Charlie" {
		t.Errorf("Order = %v, want [Alpha Bravo Charlie]", []string{list[0].Name, list[1].Name, list[2].Name})
	}
}

func TestGroupRepository_Update(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewSQLiteGroupRepository(db)
	ctx := context.Background()

	group := &DeviceGroup{Name: "Original", Type: GroupTypeStatic}
	if err := repo.Create(ctx, group); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	group.Name = "Updated"
	group.Type = GroupTypeDynamic
	group.Slug = ""
	group.FilterRules = &FilterRules{ScopeType: "site", Domains: []string{"lighting"}}

	if err := repo.Update(ctx, group); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Name != "Updated" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated")
	}
	if got.Slug != "updated" {
		t.Errorf("Slug = %q, want %q", got.Slug, "updated")
	}
	if got.Type != GroupTypeDynamic {
		t.Errorf("Type = %q, want %q", got.Type, GroupTypeDynamic)
	}
	if got.FilterRules == nil || got.FilterRules.ScopeType != "site" {
		t.Errorf("FilterRules = %v, want scope_type site", got.FilterRules)
	}
}

func TestGroupRepository_Delete(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewSQLiteGroupRepository(db)
	ctx := context.Background()

	group := &DeviceGroup{Name: "To Delete", Type: GroupTypeStatic}
	if err := repo.Create(ctx, group); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := repo.SetMembers(ctx, group.ID, []string{"dev-1", "dev-2"}); err != nil {
		t.Fatalf("SetMembers() error = %v", err)
	}

	if err := repo.Delete(ctx, group.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := repo.GetByID(ctx, group.ID)
	if !errors.Is(err, ErrGroupNotFound) {
		t.Errorf("GetByID() error = %v, want ErrGroupNotFound", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM device_group_members").Scan(&count); err != nil {
		t.Fatalf("count members error = %v", err)
	}
	if count != 0 {
		t.Errorf("member count = %d, want 0", count)
	}
}

func TestGroupRepository_SetMembers(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewSQLiteGroupRepository(db)
	ctx := context.Background()

	group := &DeviceGroup{Name: "Members", Type: GroupTypeStatic}
	if err := repo.Create(ctx, group); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.SetMembers(ctx, group.ID, []string{"dev-1", "dev-2", "dev-3"}); err != nil {
		t.Fatalf("SetMembers() error = %v", err)
	}

	members, err := repo.GetMembers(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetMembers() error = %v", err)
	}
	if len(members) != 3 {
		t.Fatalf("Members length = %d, want 3", len(members))
	}
	if members[0].DeviceID != "dev-1" || members[0].SortOrder != 0 {
		t.Errorf("First member = %+v, want dev-1 sort_order 0", members[0])
	}
}

func TestGroupRepository_SetMembers_Replace(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewSQLiteGroupRepository(db)
	ctx := context.Background()

	group := &DeviceGroup{Name: "Replace", Type: GroupTypeStatic}
	if err := repo.Create(ctx, group); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.SetMembers(ctx, group.ID, []string{"dev-1", "dev-2"}); err != nil {
		t.Fatalf("SetMembers() error = %v", err)
	}
	if err := repo.SetMembers(ctx, group.ID, []string{"dev-3"}); err != nil {
		t.Fatalf("SetMembers() replace error = %v", err)
	}

	ids, err := repo.GetMemberDeviceIDs(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetMemberDeviceIDs() error = %v", err)
	}
	if !reflect.DeepEqual(ids, []string{"dev-3"}) {
		t.Errorf("Device IDs = %v, want %v", ids, []string{"dev-3"})
	}
}

func TestGroupRepository_GetMemberDeviceIDs(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewSQLiteGroupRepository(db)
	ctx := context.Background()

	group := &DeviceGroup{Name: "IDs", Type: GroupTypeStatic}
	if err := repo.Create(ctx, group); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.SetMembers(ctx, group.ID, []string{"dev-2", "dev-1"}); err != nil {
		t.Fatalf("SetMembers() error = %v", err)
	}

	ids, err := repo.GetMemberDeviceIDs(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetMemberDeviceIDs() error = %v", err)
	}
	if !reflect.DeepEqual(ids, []string{"dev-2", "dev-1"}) {
		t.Errorf("Device IDs = %v, want %v", ids, []string{"dev-2", "dev-1"})
	}
}
