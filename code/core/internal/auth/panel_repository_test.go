package auth

import (
	"context"
	"errors"
	"testing"
)

func TestPanelRepository_CreateAndGetByID(t *testing.T) {
	db := testDB(t)
	repo := NewPanelRepository(db)
	ctx := context.Background()

	panel := &Panel{
		Name:      "Living Room Panel",
		TokenHash: HashToken("panel-secret-token"),
		IsActive:  true,
	}

	if err := repo.Create(ctx, panel); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if panel.ID == "" {
		t.Fatal("Create() should generate an ID")
	}

	got, err := repo.GetByID(ctx, panel.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Name != "Living Room Panel" {
		t.Errorf("Name = %q, want %q", got.Name, "Living Room Panel")
	}
	if !got.IsActive {
		t.Error("IsActive should be true")
	}
}

func TestPanelRepository_GetByTokenHash(t *testing.T) {
	db := testDB(t)
	repo := NewPanelRepository(db)
	ctx := context.Background()

	tokenHash := HashToken("unique-panel-token")
	panel := &Panel{
		Name:      "Front Door Panel",
		TokenHash: tokenHash,
		IsActive:  true,
	}
	repo.Create(ctx, panel) //nolint:errcheck // test setup

	got, err := repo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash() error = %v", err)
	}

	if got.ID != panel.ID {
		t.Errorf("ID = %q, want %q", got.ID, panel.ID)
	}
}

func TestPanelRepository_GetByTokenHash_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewPanelRepository(db)

	_, err := repo.GetByTokenHash(context.Background(), "nonexistent-hash")
	if !errors.Is(err, ErrPanelNotFound) {
		t.Errorf("error = %v, want ErrPanelNotFound", err)
	}
}

func TestPanelRepository_List(t *testing.T) {
	db := testDB(t)
	repo := NewPanelRepository(db)
	ctx := context.Background()

	// Empty list
	panels, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(panels) != 0 {
		t.Errorf("List() should return empty, got %d", len(panels))
	}

	// Add panels
	for _, name := range []string{"Panel A", "Panel B"} {
		p := &Panel{Name: name, TokenHash: HashToken(name), IsActive: true}
		repo.Create(ctx, p) //nolint:errcheck // test setup
	}

	panels, err = repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(panels) != 2 {
		t.Errorf("List() returned %d, want 2", len(panels))
	}
}

func TestPanelRepository_Delete(t *testing.T) {
	db := testDB(t)
	repo := NewPanelRepository(db)
	ctx := context.Background()

	panel := &Panel{Name: "Delete Me", TokenHash: HashToken("delete-me"), IsActive: true}
	repo.Create(ctx, panel) //nolint:errcheck // test setup

	if err := repo.Delete(ctx, panel.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := repo.GetByID(ctx, panel.ID)
	if !errors.Is(err, ErrPanelNotFound) {
		t.Errorf("after delete, error = %v, want ErrPanelNotFound", err)
	}
}

func TestPanelRepository_Delete_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewPanelRepository(db)

	err := repo.Delete(context.Background(), "nonexistent")
	if !errors.Is(err, ErrPanelNotFound) {
		t.Errorf("error = %v, want ErrPanelNotFound", err)
	}
}

func TestPanelRepository_UpdateLastSeen(t *testing.T) {
	db := testDB(t)
	repo := NewPanelRepository(db)
	ctx := context.Background()

	panel := &Panel{Name: "Heartbeat Panel", TokenHash: HashToken("heartbeat"), IsActive: true}
	repo.Create(ctx, panel) //nolint:errcheck // test setup

	// Initially no last_seen_at
	got, _ := repo.GetByID(ctx, panel.ID)
	if got.LastSeenAt != nil {
		t.Error("LastSeenAt should be nil initially")
	}

	if err := repo.UpdateLastSeen(ctx, panel.ID); err != nil {
		t.Fatalf("UpdateLastSeen() error = %v", err)
	}

	got, _ = repo.GetByID(ctx, panel.ID)
	if got.LastSeenAt == nil {
		t.Error("LastSeenAt should be set after UpdateLastSeen")
	}
}

func TestPanelRepository_SetAndGetRooms(t *testing.T) {
	db := testDB(t)
	seedTestRooms(t, db)
	repo := NewPanelRepository(db)
	ctx := context.Background()

	panel := &Panel{Name: "Multi-Room Panel", TokenHash: HashToken("multi"), IsActive: true}
	repo.Create(ctx, panel) //nolint:errcheck // test setup

	// Initially no rooms
	rooms, _ := repo.GetRoomIDs(ctx, panel.ID)
	if len(rooms) != 0 {
		t.Errorf("GetRoomIDs() should return empty, got %d", len(rooms))
	}

	// Assign rooms
	if err := repo.SetRooms(ctx, panel.ID, []string{"room-kitchen", "room-living"}); err != nil {
		t.Fatalf("SetRooms() error = %v", err)
	}

	rooms, err := repo.GetRoomIDs(ctx, panel.ID)
	if err != nil {
		t.Fatalf("GetRoomIDs() error = %v", err)
	}
	if len(rooms) != 2 {
		t.Errorf("GetRoomIDs() returned %d, want 2", len(rooms))
	}

	// Replace rooms (should remove old, add new)
	if err := repo.SetRooms(ctx, panel.ID, []string{"room-bedroom-jack"}); err != nil {
		t.Fatalf("SetRooms() replace error = %v", err)
	}

	rooms, _ = repo.GetRoomIDs(ctx, panel.ID)
	if len(rooms) != 1 {
		t.Errorf("after replace, GetRoomIDs() returned %d, want 1", len(rooms))
	}
	if rooms[0] != "room-bedroom-jack" {
		t.Errorf("room = %q, want %q", rooms[0], "room-bedroom-jack")
	}

	// Clear all rooms
	if err := repo.SetRooms(ctx, panel.ID, []string{}); err != nil {
		t.Fatalf("SetRooms(empty) error = %v", err)
	}

	rooms, _ = repo.GetRoomIDs(ctx, panel.ID)
	if len(rooms) != 0 {
		t.Errorf("after clear, GetRoomIDs() returned %d, want 0", len(rooms))
	}
}

func TestPanelRepository_DeleteCascadesRooms(t *testing.T) {
	db := testDB(t)
	seedTestRooms(t, db)
	repo := NewPanelRepository(db)
	ctx := context.Background()

	panel := &Panel{Name: "Cascade Panel", TokenHash: HashToken("cascade"), IsActive: true}
	repo.Create(ctx, panel)                                               //nolint:errcheck // test setup
	repo.SetRooms(ctx, panel.ID, []string{"room-kitchen", "room-living"}) //nolint:errcheck // test setup

	// Delete panel — room assignments should cascade
	repo.Delete(ctx, panel.ID) //nolint:errcheck // test setup

	// Verify room assignments are gone
	var count int
	db.QueryRow("SELECT COUNT(*) FROM panel_room_access WHERE panel_id = ?", panel.ID).Scan(&count) //nolint:errcheck // test assertion
	if count != 0 {
		t.Errorf("panel_room_access should be empty after panel delete, got %d", count)
	}
}
