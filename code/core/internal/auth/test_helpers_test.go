package auth

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// testDB creates a temporary SQLite database with the auth schema applied.
// The database file is cleaned up when the test completes.
func testDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use a temp file so WAL mode works (in-memory doesn't support it)
	f, err := os.CreateTemp("", "auth-test-*.db")
	if err != nil {
		t.Fatalf("creating temp db: %v", err)
	}
	dbPath := f.Name()
	f.Close()
	t.Cleanup(func() { os.Remove(dbPath) })

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=ON")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create prerequisite tables (rooms depends on areas, which the auth migration references)
	prerequisiteSQL := `
		CREATE TABLE areas (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL
		) STRICT;

		CREATE TABLE rooms (
			id TEXT PRIMARY KEY,
			area_id TEXT NOT NULL,
			name TEXT NOT NULL,
			FOREIGN KEY (area_id) REFERENCES areas(id) ON DELETE CASCADE
		) STRICT;
	`
	if _, err := db.Exec(prerequisiteSQL); err != nil {
		t.Fatalf("creating prerequisite tables: %v", err)
	}

	// Apply the auth migration
	migrationSQL := `
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			email TEXT,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_by TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%%H:%%M:%%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%%H:%%M:%%SZ', 'now')),
			FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
		) STRICT;

		CREATE UNIQUE INDEX idx_users_username ON users(username);
		CREATE INDEX idx_users_role ON users(role);

		CREATE TABLE refresh_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			family_id TEXT NOT NULL,
			token_hash TEXT NOT NULL,
			device_info TEXT,
			expires_at TEXT NOT NULL,
			revoked INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%%H:%%M:%%SZ', 'now')),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) STRICT;

		CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
		CREATE INDEX idx_refresh_tokens_family ON refresh_tokens(family_id);
		CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at);

		CREATE TABLE panels (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			token_hash TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			last_seen_at TEXT,
			created_by TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%%H:%%M:%%SZ', 'now')),
			FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
		) STRICT;

		CREATE TABLE panel_room_access (
			panel_id TEXT NOT NULL,
			room_id TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%%H:%%M:%%SZ', 'now')),
			PRIMARY KEY (panel_id, room_id),
			FOREIGN KEY (panel_id) REFERENCES panels(id) ON DELETE CASCADE,
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
		) STRICT;

		CREATE INDEX idx_panel_room_access_room ON panel_room_access(room_id);

		CREATE TABLE user_room_access (
			user_id TEXT NOT NULL,
			room_id TEXT NOT NULL,
			can_manage_scenes INTEGER NOT NULL DEFAULT 0,
			created_by TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%%H:%%M:%%SZ', 'now')),
			PRIMARY KEY (user_id, room_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
			FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
		) STRICT;

		CREATE INDEX idx_user_room_access_room ON user_room_access(room_id);
	`
	if _, err := db.Exec(migrationSQL); err != nil {
		t.Fatalf("applying auth migration: %v", err)
	}

	return db
}

// seedTestRooms inserts test areas and rooms for room scoping tests.
func seedTestRooms(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO areas (id, name) VALUES ('area-ground', 'Ground Floor');
		INSERT INTO rooms (id, area_id, name) VALUES ('room-kitchen', 'area-ground', 'Kitchen');
		INSERT INTO rooms (id, area_id, name) VALUES ('room-living', 'area-ground', 'Living Room');
		INSERT INTO rooms (id, area_id, name) VALUES ('room-bedroom-jack', 'area-ground', 'Jack Bedroom');
		INSERT INTO rooms (id, area_id, name) VALUES ('room-bedroom-emma', 'area-ground', 'Emma Bedroom');
	`)
	if err != nil {
		t.Fatalf("seeding test rooms: %v", err)
	}
}

// seedTestUser inserts a test user and returns it.
func seedTestUser(t *testing.T, db *sql.DB, username string, role Role) *User {
	t.Helper()

	hash, err := HashPassword("test-password")
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}

	repo := NewUserRepository(db)
	user := &User{
		Username:     username,
		DisplayName:  username,
		PasswordHash: hash,
		Role:         role,
		IsActive:     true,
	}
	if err := repo.Create(t.Context(), user); err != nil {
		t.Fatalf("creating test user %s: %v", username, err)
	}
	return user
}
