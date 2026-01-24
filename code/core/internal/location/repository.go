package location

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// Repository defines the interface for location persistence operations.
type Repository interface {
	ListAreas(ctx context.Context) ([]Area, error)
	ListAreasBySite(ctx context.Context, siteID string) ([]Area, error)
	GetArea(ctx context.Context, id string) (*Area, error)

	ListRooms(ctx context.Context) ([]Room, error)
	ListRoomsByArea(ctx context.Context, areaID string) ([]Room, error)
	GetRoom(ctx context.Context, id string) (*Room, error)
}

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite-backed location repository.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
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
func (r *SQLiteRepository) queryAreas(ctx context.Context, query string, args ...any) ([]Area, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var areas []Area
	for rows.Next() {
		a, err := scanAreaRow(rows)
		if err != nil {
			return nil, err
		}
		areas = append(areas, *a)
	}
	return areas, rows.Err()
}

// queryRooms executes a query and returns a slice of Room.
func (r *SQLiteRepository) queryRooms(ctx context.Context, query string, args ...any) ([]Room, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []Room
	for rows.Next() {
		rm, err := scanRoomRow(rows)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, *rm)
	}
	return rooms, rows.Err()
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
		return nil, err
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
		return nil, err
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
		return nil, err
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
		return nil, err
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
