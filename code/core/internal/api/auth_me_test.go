package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/nerrad567/gray-logic-core/internal/auth"
	"github.com/nerrad567/gray-logic-core/internal/location"
)

// setupMeTestDB creates an in-memory SQLite database with full auth + location schemas
// needed by handleMe (which calls locationRepo.GetRoom, ListRooms, GetArea, ListAreas).
func setupMeTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE sites (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			address TEXT,
			latitude REAL,
			longitude REAL,
			timezone TEXT NOT NULL DEFAULT 'UTC',
			elevation_m REAL,
			modes_available TEXT NOT NULL DEFAULT '["home","away","night","holiday"]',
			mode_current TEXT NOT NULL DEFAULT 'home',
			settings TEXT NOT NULL DEFAULT '{}',
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
			FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE
		) STRICT;

		CREATE TABLE rooms (
			id TEXT PRIMARY KEY,
			area_id TEXT NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'standard',
			sort_order INTEGER NOT NULL DEFAULT 0,
			climate_zone_id TEXT,
			audio_zone_id TEXT,
			settings TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			FOREIGN KEY (area_id) REFERENCES areas(id) ON DELETE CASCADE
		) STRICT;

		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			email TEXT,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_by TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
		) STRICT;

		CREATE TABLE refresh_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			family_id TEXT NOT NULL,
			token_hash TEXT NOT NULL,
			device_info TEXT,
			expires_at TEXT NOT NULL,
			revoked INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) STRICT;

		CREATE TABLE panels (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			token_hash TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			last_seen_at TEXT,
			created_by TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
		) STRICT;

		CREATE TABLE panel_room_access (
			panel_id TEXT NOT NULL,
			room_id TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			PRIMARY KEY (panel_id, room_id),
			FOREIGN KEY (panel_id) REFERENCES panels(id) ON DELETE CASCADE,
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
		) STRICT;

		CREATE TABLE user_room_access (
			user_id TEXT NOT NULL,
			room_id TEXT NOT NULL,
			can_manage_scenes INTEGER NOT NULL DEFAULT 0,
			created_by TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			PRIMARY KEY (user_id, room_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
			FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
		) STRICT;

		-- Seed location data
		INSERT INTO sites (id, name, slug, timezone) VALUES ('site-1', 'Test Site', 'test-site', 'UTC');
		INSERT INTO areas (id, site_id, name, slug) VALUES ('area-1', 'site-1', 'Ground Floor', 'ground-floor');
		INSERT INTO areas (id, site_id, name, slug) VALUES ('area-2', 'site-1', 'First Floor', 'first-floor');
		INSERT INTO rooms (id, area_id, name, slug) VALUES ('room-a', 'area-1', 'Living Room', 'living-room');
		INSERT INTO rooms (id, area_id, name, slug) VALUES ('room-b', 'area-1', 'Kitchen', 'kitchen');
		INSERT INTO rooms (id, area_id, name, slug) VALUES ('room-c', 'area-2', 'Bedroom', 'bedroom');
	`
	if _, execErr := db.Exec(schema); execErr != nil {
		db.Close()
		t.Fatalf("failed to create me test schema: %v", execErr)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

// testServerWithMe creates a server with full auth + location repos for handleMe tests.
func testServerWithMe(t *testing.T) *Server {
	t.Helper()

	srv, _ := testServer(t)
	db := setupMeTestDB(t)

	srv.userRepo = auth.NewUserRepository(db)
	srv.tokenRepo = auth.NewTokenRepository(db)
	srv.panelRepo = auth.NewPanelRepository(db)
	srv.roomAccessRepo = auth.NewRoomAccessRepository(db)
	srv.locationRepo = location.NewSQLiteRepository(db)

	return srv
}

// ─── handleMe Tests ────────────────────────────────────────────────

func TestHandleMe_AdminUser(t *testing.T) {
	srv := testServerWithMe(t)
	router := srv.buildRouter()

	createTestUser(t, srv.userRepo, "admin-1", "admin", "testpass123", auth.RoleAdmin, true)
	token := testRoleToken(t, auth.RoleAdmin, "admin-1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp meResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Type != "user" {
		t.Errorf("type = %q, want user", resp.Type)
	}
	if resp.User == nil {
		t.Fatal("user field is nil")
	}
	if resp.User.ID != "admin-1" {
		t.Errorf("user.id = %q, want admin-1", resp.User.ID)
	}
	if resp.User.Username != "admin" {
		t.Errorf("user.username = %q, want admin", resp.User.Username)
	}
	if resp.User.Role != auth.RoleAdmin {
		t.Errorf("user.role = %q, want admin", resp.User.Role)
	}

	// Admin should see all 3 rooms
	if len(resp.Rooms) != 3 {
		t.Errorf("rooms count = %d, want 3", len(resp.Rooms))
	}

	// All rooms should have can_manage_scenes=true for admin
	for _, rm := range resp.Rooms {
		if !rm.CanManageScenes {
			t.Errorf("room %s: can_manage_scenes = false, want true", rm.ID)
		}
		if rm.AreaName == "" {
			t.Errorf("room %s: area_name is empty", rm.ID)
		}
	}

	// Should have admin permissions
	if len(resp.Permissions) == 0 {
		t.Error("permissions should not be empty for admin")
	}

	// Check key permissions are present
	permSet := make(map[string]bool)
	for _, p := range resp.Permissions {
		permSet[p] = true
	}
	for _, expected := range []string{"device:read", "device:configure", "location:manage", "system:admin"} {
		if !permSet[expected] {
			t.Errorf("missing permission: %s", expected)
		}
	}
}

func TestHandleMe_OwnerUser(t *testing.T) {
	srv := testServerWithMe(t)
	router := srv.buildRouter()

	createTestUser(t, srv.userRepo, "owner-1", "owner", "testpass123", auth.RoleOwner, true)
	token := testRoleToken(t, auth.RoleOwner, "owner-1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp meResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Owner should have system:dangerous permission
	permSet := make(map[string]bool)
	for _, p := range resp.Permissions {
		permSet[p] = true
	}
	if !permSet["system:dangerous"] {
		t.Error("owner should have system:dangerous permission")
	}

	// Owner should see all rooms
	if len(resp.Rooms) != 3 {
		t.Errorf("rooms count = %d, want 3", len(resp.Rooms))
	}
}

func TestHandleMe_RegularUser_RoomScoped(t *testing.T) {
	srv := testServerWithMe(t)
	router := srv.buildRouter()

	createTestUser(t, srv.userRepo, "admin-setup", "admin-setup", "testpass123", auth.RoleAdmin, true)
	createTestUser(t, srv.userRepo, "user-1", "alice", "testpass123", auth.RoleUser, true)

	// Grant access to room-a (with scene manage) and room-b (read only)
	grants := []auth.RoomAccessGrant{
		{RoomID: "room-a", CanManageScenes: true},
		{RoomID: "room-b", CanManageScenes: false},
	}
	if err := srv.roomAccessRepo.SetRoomAccess(context.Background(), "user-1", grants, "admin-setup"); err != nil {
		t.Fatalf("set room access: %v", err)
	}

	token := testRoleToken(t, auth.RoleUser, "user-1")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp meResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Type != "user" {
		t.Errorf("type = %q, want user", resp.Type)
	}

	// Should only see 2 rooms (not room-c)
	if len(resp.Rooms) != 2 {
		t.Fatalf("rooms count = %d, want 2; rooms: %v", len(resp.Rooms), resp.Rooms)
	}

	// Check room-a has can_manage_scenes=true, room-b has false
	roomMap := make(map[string]meRoom)
	for _, rm := range resp.Rooms {
		roomMap[rm.ID] = rm
	}

	if rm, ok := roomMap["room-a"]; !ok {
		t.Error("missing room-a")
	} else {
		if !rm.CanManageScenes {
			t.Error("room-a: can_manage_scenes should be true")
		}
		if rm.Name != "Living Room" {
			t.Errorf("room-a: name = %q, want Living Room", rm.Name)
		}
		if rm.AreaName != "Ground Floor" {
			t.Errorf("room-a: area_name = %q, want Ground Floor", rm.AreaName)
		}
	}

	if rm, ok := roomMap["room-b"]; !ok {
		t.Error("missing room-b")
	} else {
		if rm.CanManageScenes {
			t.Error("room-b: can_manage_scenes should be false")
		}
	}

	// User should have basic permissions only
	permSet := make(map[string]bool)
	for _, p := range resp.Permissions {
		permSet[p] = true
	}
	if !permSet["device:read"] {
		t.Error("user should have device:read")
	}
	if permSet["system:admin"] {
		t.Error("user should NOT have system:admin")
	}
}

func TestHandleMe_RegularUser_NoRoomAccess(t *testing.T) {
	srv := testServerWithMe(t)
	router := srv.buildRouter()

	createTestUser(t, srv.userRepo, "user-locked", "locked", "testpass123", auth.RoleUser, true)
	// No room access grants

	token := testRoleToken(t, auth.RoleUser, "user-locked")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp meResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Rooms) != 0 {
		t.Errorf("rooms count = %d, want 0 for user with no grants", len(resp.Rooms))
	}
}

func TestHandleMe_PanelAuth(t *testing.T) {
	srv := testServerWithMe(t)
	router := srv.buildRouter()

	rawToken := "panel-me-test-token"
	panel := &auth.Panel{
		ID:        "panel-me",
		Name:      "Living Room Panel",
		TokenHash: auth.HashToken(rawToken),
		IsActive:  true,
	}
	if err := srv.panelRepo.Create(context.Background(), panel); err != nil {
		t.Fatalf("create panel: %v", err)
	}
	if err := srv.panelRepo.SetRooms(context.Background(), panel.ID, []string{"room-a", "room-b"}); err != nil {
		t.Fatalf("set panel rooms: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("X-Panel-Token", rawToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp meResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Type != "panel" {
		t.Errorf("type = %q, want panel", resp.Type)
	}
	if resp.Panel == nil {
		t.Fatal("panel field is nil")
	}
	if resp.Panel.ID != "panel-me" {
		t.Errorf("panel.id = %q, want panel-me", resp.Panel.ID)
	}
	if resp.Panel.Name != "Living Room Panel" {
		t.Errorf("panel.name = %q, want Living Room Panel", resp.Panel.Name)
	}
	if resp.User != nil {
		t.Error("user field should be nil for panel")
	}

	// Panel should see 2 rooms
	if len(resp.Rooms) != 2 {
		t.Fatalf("rooms count = %d, want 2", len(resp.Rooms))
	}

	// Rooms should have area names
	for _, rm := range resp.Rooms {
		if rm.AreaName == "" {
			t.Errorf("room %s: area_name is empty", rm.ID)
		}
	}

	// Panel response should have no permissions field
	if len(resp.Permissions) != 0 {
		t.Errorf("permissions should be empty for panel, got %v", resp.Permissions)
	}
}

func TestHandleMe_Unauthenticated(t *testing.T) {
	srv := testServerWithMe(t)
	router := srv.buildRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
