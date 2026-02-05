package auth

import (
	"context"
	"testing"
)

func TestRoomAccessRepository_SetAndGetRoomAccess(t *testing.T) {
	db := testDB(t)
	seedTestRooms(t, db)
	user := seedTestUser(t, db, "jack", RoleUser)
	repo := NewRoomAccessRepository(db)
	ctx := context.Background()

	// Initially no access
	access, err := repo.GetRoomAccess(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetRoomAccess() error = %v", err)
	}
	if len(access) != 0 {
		t.Errorf("should have no access initially, got %d", len(access))
	}

	// Grant access to bedroom (with scenes) and kitchen (without scenes)
	grants := []RoomAccessGrant{
		{RoomID: "room-bedroom-jack", CanManageScenes: true},
		{RoomID: "room-kitchen", CanManageScenes: false},
	}
	if err := repo.SetRoomAccess(ctx, user.ID, grants, ""); err != nil { //nolint:govet // shadow: err re-declared in test
		t.Fatalf("SetRoomAccess() error = %v", err)
	}

	access, err = repo.GetRoomAccess(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetRoomAccess() error = %v", err)
	}
	if len(access) != 2 {
		t.Fatalf("GetRoomAccess() returned %d, want 2", len(access))
	}

	// Verify order (by room_id) and values
	if access[0].RoomID != "room-bedroom-jack" || !access[0].CanManageScenes {
		t.Errorf("access[0] = %+v, want bedroom-jack with scenes", access[0])
	}
	if access[1].RoomID != "room-kitchen" || access[1].CanManageScenes {
		t.Errorf("access[1] = %+v, want kitchen without scenes", access[1])
	}
}

func TestRoomAccessRepository_GetAccessibleRoomIDs(t *testing.T) {
	db := testDB(t)
	seedTestRooms(t, db)
	user := seedTestUser(t, db, "mum", RoleUser)
	repo := NewRoomAccessRepository(db)
	ctx := context.Background()

	grants := []RoomAccessGrant{
		{RoomID: "room-kitchen", CanManageScenes: true},
		{RoomID: "room-living", CanManageScenes: true},
		{RoomID: "room-bedroom-jack", CanManageScenes: false},
	}
	repo.SetRoomAccess(ctx, user.ID, grants, "") //nolint:errcheck // test setup

	roomIDs, err := repo.GetAccessibleRoomIDs(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetAccessibleRoomIDs() error = %v", err)
	}
	if len(roomIDs) != 3 {
		t.Errorf("GetAccessibleRoomIDs() returned %d, want 3", len(roomIDs))
	}
}

func TestRoomAccessRepository_GetSceneManageRoomIDs(t *testing.T) {
	db := testDB(t)
	seedTestRooms(t, db)
	user := seedTestUser(t, db, "teen", RoleUser)
	repo := NewRoomAccessRepository(db)
	ctx := context.Background()

	grants := []RoomAccessGrant{
		{RoomID: "room-bedroom-jack", CanManageScenes: true},
		{RoomID: "room-kitchen", CanManageScenes: false},
		{RoomID: "room-living", CanManageScenes: false},
	}
	repo.SetRoomAccess(ctx, user.ID, grants, "") //nolint:errcheck // test setup

	sceneRooms, err := repo.GetSceneManageRoomIDs(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetSceneManageRoomIDs() error = %v", err)
	}
	if len(sceneRooms) != 1 {
		t.Fatalf("GetSceneManageRoomIDs() returned %d, want 1", len(sceneRooms))
	}
	if sceneRooms[0] != "room-bedroom-jack" {
		t.Errorf("scene room = %q, want %q", sceneRooms[0], "room-bedroom-jack")
	}
}

func TestRoomAccessRepository_ClearRoomAccess(t *testing.T) {
	db := testDB(t)
	seedTestRooms(t, db)
	user := seedTestUser(t, db, "clearme", RoleUser)
	repo := NewRoomAccessRepository(db)
	ctx := context.Background()

	grants := []RoomAccessGrant{
		{RoomID: "room-kitchen", CanManageScenes: false},
	}
	repo.SetRoomAccess(ctx, user.ID, grants, "") //nolint:errcheck // test setup

	if err := repo.ClearRoomAccess(ctx, user.ID); err != nil {
		t.Fatalf("ClearRoomAccess() error = %v", err)
	}

	roomIDs, _ := repo.GetAccessibleRoomIDs(ctx, user.ID)
	if len(roomIDs) != 0 {
		t.Errorf("after clear, GetAccessibleRoomIDs() returned %d, want 0", len(roomIDs))
	}
}

func TestRoomAccessRepository_SetRoomAccess_Replaces(t *testing.T) {
	db := testDB(t)
	seedTestRooms(t, db)
	user := seedTestUser(t, db, "replaceme", RoleUser)
	repo := NewRoomAccessRepository(db)
	ctx := context.Background()

	// Initial grants
	grants1 := []RoomAccessGrant{
		{RoomID: "room-kitchen", CanManageScenes: false},
		{RoomID: "room-living", CanManageScenes: false},
	}
	repo.SetRoomAccess(ctx, user.ID, grants1, "") //nolint:errcheck // test setup

	// Replace with different grants
	grants2 := []RoomAccessGrant{
		{RoomID: "room-bedroom-emma", CanManageScenes: true},
	}
	if err := repo.SetRoomAccess(ctx, user.ID, grants2, ""); err != nil {
		t.Fatalf("SetRoomAccess(replace) error = %v", err)
	}

	roomIDs, _ := repo.GetAccessibleRoomIDs(ctx, user.ID)
	if len(roomIDs) != 1 {
		t.Fatalf("after replace, got %d rooms, want 1", len(roomIDs))
	}
	if roomIDs[0] != "room-bedroom-emma" {
		t.Errorf("room = %q, want %q", roomIDs[0], "room-bedroom-emma")
	}
}

func TestRoomAccessRepository_ResolveRoomScope(t *testing.T) {
	db := testDB(t)
	seedTestRooms(t, db)
	user := seedTestUser(t, db, "scopeuser", RoleUser)
	repo := NewRoomAccessRepository(db)
	ctx := context.Background()

	grants := []RoomAccessGrant{
		{RoomID: "room-bedroom-jack", CanManageScenes: true},
		{RoomID: "room-kitchen", CanManageScenes: false},
		{RoomID: "room-living", CanManageScenes: false},
	}
	repo.SetRoomAccess(ctx, user.ID, grants, "") //nolint:errcheck // test setup

	scope, err := repo.ResolveRoomScope(ctx, user.ID)
	if err != nil {
		t.Fatalf("ResolveRoomScope() error = %v", err)
	}

	if len(scope.RoomIDs) != 3 {
		t.Errorf("RoomIDs count = %d, want 3", len(scope.RoomIDs))
	}
	if len(scope.SceneManageRoomIDs) != 1 {
		t.Errorf("SceneManageRoomIDs count = %d, want 1", len(scope.SceneManageRoomIDs))
	}

	// Test CanAccessRoom
	if !scope.CanAccessRoom("room-kitchen") {
		t.Error("should have access to kitchen")
	}
	if scope.CanAccessRoom("room-bedroom-emma") {
		t.Error("should NOT have access to emma's bedroom")
	}

	// Test CanManageScenesInRoom
	if !scope.CanManageScenesInRoom("room-bedroom-jack") {
		t.Error("should be able to manage scenes in jack's bedroom")
	}
	if scope.CanManageScenesInRoom("room-kitchen") {
		t.Error("should NOT be able to manage scenes in kitchen")
	}
}

func TestRoomAccessRepository_ResolveRoomScope_NoGrants(t *testing.T) {
	db := testDB(t)
	user := seedTestUser(t, db, "nogrants", RoleUser)
	repo := NewRoomAccessRepository(db)

	scope, err := repo.ResolveRoomScope(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("ResolveRoomScope() error = %v", err)
	}

	if len(scope.RoomIDs) != 0 {
		t.Errorf("RoomIDs should be empty for user with no grants, got %d", len(scope.RoomIDs))
	}
	if scope.CanAccessRoom("any-room") {
		t.Error("user with no grants should not have access to any room")
	}
}

func TestRoomScope_NilIsUnrestricted(t *testing.T) {
	var scope *RoomScope // nil = unrestricted (admin/owner)

	if !scope.CanAccessRoom("any-room") {
		t.Error("nil scope should allow access to any room")
	}
	if !scope.CanManageScenesInRoom("any-room") {
		t.Error("nil scope should allow scene management in any room")
	}
}
