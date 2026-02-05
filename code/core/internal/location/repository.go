package location

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Repository defines the interface for location persistence operations.
type Repository interface {
	CreateArea(ctx context.Context, area *Area) error
	ListAreas(ctx context.Context) ([]Area, error)
	ListAreasBySite(ctx context.Context, siteID string) ([]Area, error)
	GetArea(ctx context.Context, id string) (*Area, error)
	UpdateArea(ctx context.Context, area *Area) error
	DeleteArea(ctx context.Context, id string) error
	DeleteAllAreas(ctx context.Context) (int64, error)

	CreateRoom(ctx context.Context, room *Room) error
	ListRooms(ctx context.Context) ([]Room, error)
	ListRoomsByArea(ctx context.Context, areaID string) ([]Room, error)
	GetRoom(ctx context.Context, id string) (*Room, error)
	UpdateRoom(ctx context.Context, room *Room) error
	DeleteRoom(ctx context.Context, id string) error
	DeleteAllRooms(ctx context.Context) (int64, error)

	// Site operations (single-row table — one site per deployment).
	GetAnySite(ctx context.Context) (*Site, error)
	CreateSite(ctx context.Context, site *Site) error
	UpdateSite(ctx context.Context, site *Site) error
	DeleteAllSites(ctx context.Context) (int64, error)
}

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite-backed location repository.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// CreateArea inserts a new area into the database.
func (r *SQLiteRepository) CreateArea(ctx context.Context, area *Area) error {
	const query = `INSERT INTO areas (id, site_id, name, slug, type, sort_order)
		VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		area.ID, area.SiteID, area.Name, area.Slug, area.Type, area.SortOrder)
	if err != nil {
		return fmt.Errorf("inserting area %s: %w", area.ID, err)
	}
	return nil
}

// CreateRoom inserts a new room into the database.
func (r *SQLiteRepository) CreateRoom(ctx context.Context, room *Room) error {
	settings := "{}"
	if room.Settings != nil {
		b, err := json.Marshal(room.Settings)
		if err == nil {
			settings = string(b)
		}
	}
	const query = `INSERT INTO rooms (id, area_id, name, slug, type, sort_order,
		climate_zone_id, audio_zone_id, settings)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		room.ID, room.AreaID, room.Name, room.Slug, room.Type, room.SortOrder,
		nullStr(room.ClimateZoneID), nullStr(room.AudioZoneID), settings)
	if err != nil {
		return fmt.Errorf("inserting room %s: %w", room.ID, err)
	}
	return nil
}

// nullStr converts a *string to a sql.NullString for nullable columns.
func nullStr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// ListAreas returns all areas ordered by sort_order then name.
func (r *SQLiteRepository) ListAreas(ctx context.Context) ([]Area, error) {
	const query = `SELECT id, site_id, name, slug, type, sort_order, created_at, updated_at
		FROM areas ORDER BY sort_order, name`
	return r.queryAreas(ctx, query)
}

// ListAreasBySite returns areas for a specific site.
func (r *SQLiteRepository) ListAreasBySite(ctx context.Context, siteID string) ([]Area, error) {
	const query = `SELECT id, site_id, name, slug, type, sort_order, created_at, updated_at
		FROM areas WHERE site_id = ? ORDER BY sort_order, name`
	return r.queryAreas(ctx, query, siteID)
}

// GetArea returns a single area by ID.
func (r *SQLiteRepository) GetArea(ctx context.Context, id string) (*Area, error) {
	const query = `SELECT id, site_id, name, slug, type, sort_order, created_at, updated_at
		FROM areas WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	area, err := scanArea(row)
	if err != nil {
		return nil, err
	}
	return area, nil
}

// ListRooms returns all rooms ordered by sort_order then name.
func (r *SQLiteRepository) ListRooms(ctx context.Context) ([]Room, error) {
	const query = `SELECT id, area_id, name, slug, type, sort_order,
		climate_zone_id, audio_zone_id, settings, created_at, updated_at
		FROM rooms ORDER BY sort_order, name`
	return r.queryRooms(ctx, query)
}

// ListRoomsByArea returns rooms for a specific area.
func (r *SQLiteRepository) ListRoomsByArea(ctx context.Context, areaID string) ([]Room, error) {
	const query = `SELECT id, area_id, name, slug, type, sort_order,
		climate_zone_id, audio_zone_id, settings, created_at, updated_at
		FROM rooms WHERE area_id = ? ORDER BY sort_order, name`
	return r.queryRooms(ctx, query, areaID)
}

// GetRoom returns a single room by ID.
func (r *SQLiteRepository) GetRoom(ctx context.Context, id string) (*Room, error) {
	const query = `SELECT id, area_id, name, slug, type, sort_order,
		climate_zone_id, audio_zone_id, settings, created_at, updated_at
		FROM rooms WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	room, err := scanRoom(row)
	if err != nil {
		return nil, err
	}
	return room, nil
}

// queryAreas executes a query and returns a slice of Area.
func (r *SQLiteRepository) queryAreas(ctx context.Context, query string, args ...any) ([]Area, error) { //nolint:dupl // queryAreas and queryRooms differ in types and scan functions
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying areas: %w", err)
	}
	defer rows.Close()

	var areas []Area
	for rows.Next() {
		a, err := scanAreaRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning area row: %w", err)
		}
		areas = append(areas, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating area rows: %w", err)
	}
	return areas, nil
}

// queryRooms executes a query and returns a slice of Room.
func (r *SQLiteRepository) queryRooms(ctx context.Context, query string, args ...any) ([]Room, error) { //nolint:dupl // queryAreas and queryRooms differ in types and scan functions
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying rooms: %w", err)
	}
	defer rows.Close()

	var rooms []Room
	for rows.Next() {
		rm, err := scanRoomRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning room row: %w", err)
		}
		rooms = append(rooms, *rm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating room rows: %w", err)
	}
	return rooms, nil
}

// scanArea scans a single row into an Area (for QueryRow).
func scanArea(row *sql.Row) (*Area, error) {
	var a Area
	var createdAt, updatedAt string

	err := row.Scan(&a.ID, &a.SiteID, &a.Name, &a.Slug, &a.Type, &a.SortOrder, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAreaNotFound
		}
		return nil, fmt.Errorf("scanning area: %w", err)
	}
	a.CreatedAt = parseTime(createdAt)
	a.UpdatedAt = parseTime(updatedAt)
	return &a, nil
}

// scanAreaRow scans an area from a Rows cursor.
func scanAreaRow(rows *sql.Rows) (*Area, error) {
	var a Area
	var createdAt, updatedAt string

	err := rows.Scan(&a.ID, &a.SiteID, &a.Name, &a.Slug, &a.Type, &a.SortOrder, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("scanning area row: %w", err)
	}
	a.CreatedAt = parseTime(createdAt)
	a.UpdatedAt = parseTime(updatedAt)
	return &a, nil
}

// scanRoom scans a single row into a Room (for QueryRow).
func scanRoom(row *sql.Row) (*Room, error) {
	var rm Room
	var climateZoneID, audioZoneID sql.NullString
	var settingsJSON string
	var createdAt, updatedAt string

	err := row.Scan(&rm.ID, &rm.AreaID, &rm.Name, &rm.Slug, &rm.Type, &rm.SortOrder,
		&climateZoneID, &audioZoneID, &settingsJSON, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRoomNotFound
		}
		return nil, fmt.Errorf("scanning room: %w", err)
	}

	if climateZoneID.Valid {
		rm.ClimateZoneID = &climateZoneID.String
	}
	if audioZoneID.Valid {
		rm.AudioZoneID = &audioZoneID.String
	}
	rm.Settings = parseSettings(settingsJSON)
	rm.CreatedAt = parseTime(createdAt)
	rm.UpdatedAt = parseTime(updatedAt)
	return &rm, nil
}

// scanRoomRow scans a room from a Rows cursor.
func scanRoomRow(rows *sql.Rows) (*Room, error) {
	var rm Room
	var climateZoneID, audioZoneID sql.NullString
	var settingsJSON string
	var createdAt, updatedAt string

	err := rows.Scan(&rm.ID, &rm.AreaID, &rm.Name, &rm.Slug, &rm.Type, &rm.SortOrder,
		&climateZoneID, &audioZoneID, &settingsJSON, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("scanning room row: %w", err)
	}

	if climateZoneID.Valid {
		rm.ClimateZoneID = &climateZoneID.String
	}
	if audioZoneID.Valid {
		rm.AudioZoneID = &audioZoneID.String
	}
	rm.Settings = parseSettings(settingsJSON)
	rm.CreatedAt = parseTime(createdAt)
	rm.UpdatedAt = parseTime(updatedAt)
	return &rm, nil
}

// UpdateArea updates an existing area record.
func (r *SQLiteRepository) UpdateArea(ctx context.Context, area *Area) error {
	const query = `UPDATE areas SET name = ?, slug = ?, type = ?, sort_order = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query,
		area.Name, area.Slug, area.Type, area.SortOrder, area.ID)
	if err != nil {
		return fmt.Errorf("updating area %s: %w", area.ID, err)
	}
	n, _ := result.RowsAffected() //nolint:errcheck // SQLite always supports RowsAffected
	if n == 0 {
		return ErrAreaNotFound
	}
	return nil
}

// DeleteArea removes a single area by ID.
// Returns ErrAreaNotFound if the area does not exist.
// Returns ErrAreaHasRooms if rooms still reference this area.
func (r *SQLiteRepository) DeleteArea(ctx context.Context, id string) error {
	// Check for child rooms.
	var roomCount int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM rooms WHERE area_id = ?", id).Scan(&roomCount); err != nil {
		return fmt.Errorf("counting rooms for area %s: %w", id, err)
	}
	if roomCount > 0 {
		return ErrAreaHasRooms
	}

	result, err := r.db.ExecContext(ctx, "DELETE FROM areas WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting area %s: %w", id, err)
	}
	n, _ := result.RowsAffected() //nolint:errcheck // SQLite always supports RowsAffected
	if n == 0 {
		return ErrAreaNotFound
	}
	return nil
}

// UpdateRoom updates an existing room record.
func (r *SQLiteRepository) UpdateRoom(ctx context.Context, room *Room) error {
	settings := "{}"
	if room.Settings != nil {
		b, err := json.Marshal(room.Settings)
		if err == nil {
			settings = string(b)
		}
	}
	const query = `UPDATE rooms SET name = ?, slug = ?, type = ?, sort_order = ?,
		area_id = ?, climate_zone_id = ?, audio_zone_id = ?, settings = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query,
		room.Name, room.Slug, room.Type, room.SortOrder,
		room.AreaID, nullStr(room.ClimateZoneID), nullStr(room.AudioZoneID),
		settings, room.ID)
	if err != nil {
		return fmt.Errorf("updating room %s: %w", room.ID, err)
	}
	n, _ := result.RowsAffected() //nolint:errcheck // SQLite always supports RowsAffected
	if n == 0 {
		return ErrRoomNotFound
	}
	return nil
}

// DeleteRoom removes a single room by ID.
// Returns ErrRoomNotFound if the room does not exist.
func (r *SQLiteRepository) DeleteRoom(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM rooms WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting room %s: %w", id, err)
	}
	n, _ := result.RowsAffected() //nolint:errcheck // SQLite always supports RowsAffected
	if n == 0 {
		return ErrRoomNotFound
	}
	return nil
}

// DeleteAllRooms removes all rooms from the database.
// Returns the number of rows deleted.
func (r *SQLiteRepository) DeleteAllRooms(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM rooms")
	if err != nil {
		return 0, fmt.Errorf("deleting all rooms: %w", err)
	}
	n, _ := result.RowsAffected() //nolint:errcheck // SQLite always supports RowsAffected
	return n, nil
}

// DeleteAllAreas removes all areas from the database.
// Rooms must be deleted first due to FK constraints.
func (r *SQLiteRepository) DeleteAllAreas(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM areas")
	if err != nil {
		return 0, fmt.Errorf("deleting all areas: %w", err)
	}
	n, _ := result.RowsAffected() //nolint:errcheck // SQLite always supports RowsAffected
	return n, nil
}

// GetAnySite returns the first site record, or ErrSiteNotFound if none exists.
func (r *SQLiteRepository) GetAnySite(ctx context.Context) (*Site, error) {
	const query = `SELECT id, name, slug, address, latitude, longitude, timezone,
		elevation_m, modes_available, mode_current, settings, created_at, updated_at
		FROM sites LIMIT 1`
	row := r.db.QueryRowContext(ctx, query)
	return scanSite(row)
}

// CreateSite inserts a new site record.
func (r *SQLiteRepository) CreateSite(ctx context.Context, site *Site) error {
	modesJSON, err := json.Marshal(site.ModesAvailable)
	if err != nil {
		modesJSON = []byte(`["home","away","night","holiday"]`)
	}
	settings := "{}"
	if site.Settings != nil {
		if b, err := json.Marshal(site.Settings); err == nil { //nolint:govet // shadow: err re-declared in nested scope, checked immediately
			settings = string(b)
		}
	}
	const query = `INSERT INTO sites (id, name, slug, address, latitude, longitude,
		timezone, elevation_m, modes_available, mode_current, settings)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = r.db.ExecContext(ctx, query,
		site.ID, site.Name, site.Slug, site.Address,
		nullFloat(site.Latitude), nullFloat(site.Longitude),
		site.Timezone, nullFloat(site.ElevationM),
		string(modesJSON), site.ModeCurrent, settings)
	if err != nil {
		return fmt.Errorf("inserting site %s: %w", site.ID, err)
	}
	return nil
}

// UpdateSite updates an existing site record.
func (r *SQLiteRepository) UpdateSite(ctx context.Context, site *Site) error {
	modesJSON, err := json.Marshal(site.ModesAvailable)
	if err != nil {
		modesJSON = []byte(`["home","away","night","holiday"]`)
	}
	settings := "{}"
	if site.Settings != nil {
		if b, err := json.Marshal(site.Settings); err == nil { //nolint:govet // shadow: err re-declared in nested scope, checked immediately
			settings = string(b)
		}
	}
	const query = `UPDATE sites SET name = ?, slug = ?, address = ?,
		latitude = ?, longitude = ?, timezone = ?, elevation_m = ?,
		modes_available = ?, mode_current = ?, settings = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?`
	_, err = r.db.ExecContext(ctx, query,
		site.Name, site.Slug, site.Address,
		nullFloat(site.Latitude), nullFloat(site.Longitude),
		site.Timezone, nullFloat(site.ElevationM),
		string(modesJSON), site.ModeCurrent, settings,
		site.ID)
	if err != nil {
		return fmt.Errorf("updating site %s: %w", site.ID, err)
	}
	return nil
}

// DeleteAllSites removes all site records from the database.
// Areas must be deleted first due to FK constraints.
func (r *SQLiteRepository) DeleteAllSites(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM sites")
	if err != nil {
		return 0, fmt.Errorf("deleting all sites: %w", err)
	}
	n, _ := result.RowsAffected() //nolint:errcheck // SQLite always supports RowsAffected
	return n, nil
}

// scanSite scans a single row into a Site.
func scanSite(row *sql.Row) (*Site, error) {
	var s Site
	var address sql.NullString
	var lat, lon, elev sql.NullFloat64
	var modesJSON, settingsJSON string
	var createdAt, updatedAt string

	err := row.Scan(&s.ID, &s.Name, &s.Slug, &address, &lat, &lon,
		&s.Timezone, &elev, &modesJSON, &s.ModeCurrent, &settingsJSON,
		&createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrSiteNotFound
		}
		return nil, fmt.Errorf("scanning site: %w", err)
	}

	if address.Valid {
		s.Address = address.String
	}
	if lat.Valid {
		s.Latitude = &lat.Float64
	}
	if lon.Valid {
		s.Longitude = &lon.Float64
	}
	if elev.Valid {
		s.ElevationM = &elev.Float64
	}
	if err := json.Unmarshal([]byte(modesJSON), &s.ModesAvailable); err != nil {
		s.ModesAvailable = []string{"home", "away", "night", "holiday"}
	}
	s.Settings = parseSettings(settingsJSON)
	s.CreatedAt = parseTime(createdAt)
	s.UpdatedAt = parseTime(updatedAt)
	return &s, nil
}

// nullFloat converts a *float64 to sql.NullFloat64 for nullable columns.
func nullFloat(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}

// parseTime parses an ISO 8601 timestamp from SQLite.
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try the SQLite default format without timezone.
		// Zero time is returned if both formats fail (should not
		// happen with schema-enforced DEFAULT strftime).
		t, err = time.Parse("2006-01-02T15:04:05Z", s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

// parseSettings deserializes a JSON string into a Settings map.
func parseSettings(s string) Settings {
	if s == "" || s == "{}" {
		return Settings{}
	}
	var m Settings
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return Settings{}
	}
	return m
}
