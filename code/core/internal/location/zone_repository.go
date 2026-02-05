package location

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/nerrad567/gray-logic-core/internal/device"
)

// ZoneRepository defines persistence operations for infrastructure zones.
type ZoneRepository interface {
	// Zone CRUD
	CreateZone(ctx context.Context, zone *InfrastructureZone) error
	GetZone(ctx context.Context, id string) (*InfrastructureZone, error)
	ListZones(ctx context.Context, siteID string) ([]InfrastructureZone, error)
	ListZonesByDomain(ctx context.Context, siteID string, domain ZoneDomain) ([]InfrastructureZone, error)
	UpdateZone(ctx context.Context, zone *InfrastructureZone) error
	DeleteZone(ctx context.Context, id string) error

	// Room membership
	SetZoneRooms(ctx context.Context, zoneID string, roomIDs []string) error
	GetZoneRooms(ctx context.Context, zoneID string) ([]Room, error)
	GetZoneRoomIDs(ctx context.Context, zoneID string) ([]string, error)

	// Reverse lookup: which zone is this room in for a given domain?
	GetZoneForRoom(ctx context.Context, roomID string, domain ZoneDomain) (*InfrastructureZone, error)

	// List all zones a room belongs to (across all domains)
	GetZonesForRoom(ctx context.Context, roomID string) ([]InfrastructureZone, error)

	// GetAllRoomZoneMappings returns room_id → {domain: zone_id} for all rooms in one query.
	GetAllRoomZoneMappings(ctx context.Context) (map[string]map[string]string, error)
}

// SQLiteZoneRepository implements ZoneRepository using SQLite.
type SQLiteZoneRepository struct {
	db *sql.DB
}

// NewSQLiteZoneRepository creates a new SQLite-backed zone repository.
//
// Parameters:
//   - db: Open SQLite connection used for zone queries
//
// Returns:
//   - *SQLiteZoneRepository: Repository instance ready for use
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	repo := location.NewSQLiteZoneRepository(db)
func NewSQLiteZoneRepository(db *sql.DB) *SQLiteZoneRepository {
	return &SQLiteZoneRepository{db: db}
}

// CreateZone inserts a new infrastructure zone.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - zone: Zone definition to persist
//
// Returns:
//   - error: nil on success, otherwise the underlying database error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	err := repo.CreateZone(ctx, &location.InfrastructureZone{Name: "North Wing"})
func (r *SQLiteZoneRepository) CreateZone(ctx context.Context, zone *InfrastructureZone) error {
	if zone == nil {
		return fmt.Errorf("zone is required")
	}
	if zone.ID == "" {
		zone.ID = device.GenerateID()
	}
	if zone.Slug == "" {
		zone.Slug = device.GenerateSlug(zone.Name)
	}
	if err := ValidateZone(zone); err != nil {
		return err
	}

	settingsJSON, err := marshalSettings(zone.Settings)
	if err != nil {
		return err
	}

	query := `INSERT INTO infrastructure_zones (
			id, site_id, name, slug, domain, settings, sort_order
		) VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = r.db.ExecContext(ctx, query,
		zone.ID,
		zone.SiteID,
		zone.Name,
		zone.Slug,
		string(zone.Domain),
		settingsJSON,
		zone.SortOrder,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrZoneExists
		}
		return fmt.Errorf("inserting infrastructure zone: %w", err)
	}

	return nil
}

// GetZone retrieves an infrastructure zone by ID.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - id: Unique zone identifier
//
// Returns:
//   - *InfrastructureZone: Zone definition when found
//   - error: ErrZoneNotFound if missing, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	zone, err := repo.GetZone(ctx, "zone-1")
func (r *SQLiteZoneRepository) GetZone(ctx context.Context, id string) (*InfrastructureZone, error) {
	query := `SELECT id, site_id, name, slug, domain, settings, sort_order, created_at, updated_at
		FROM infrastructure_zones WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	zone, err := scanZoneRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrZoneNotFound
		}
		return nil, err
	}

	return zone, nil
}

// ListZones returns all zones for a site ordered by domain, sort_order, name.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - siteID: Site identifier to filter by
//
// Returns:
//   - []InfrastructureZone: Zones ordered by domain then sort_order
//   - error: nil on success, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	zones, err := repo.ListZones(ctx, "site-1")
func (r *SQLiteZoneRepository) ListZones(ctx context.Context, siteID string) ([]InfrastructureZone, error) {
	query := `SELECT id, site_id, name, slug, domain, settings, sort_order, created_at, updated_at
		FROM infrastructure_zones
		WHERE site_id = ?
		ORDER BY domain, sort_order, name`

	return r.queryZones(ctx, query, siteID)
}

// ListZonesByDomain returns zones for a site filtered by domain.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - siteID: Site identifier to filter by
//   - domain: Zone domain to filter by
//
// Returns:
//   - []InfrastructureZone: Zones ordered by sort_order then name
//   - error: nil on success, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	zones, err := repo.ListZonesByDomain(ctx, "site-1", location.ZoneDomainAudio)
func (r *SQLiteZoneRepository) ListZonesByDomain(ctx context.Context, siteID string, domain ZoneDomain) ([]InfrastructureZone, error) {
	query := `SELECT id, site_id, name, slug, domain, settings, sort_order, created_at, updated_at
		FROM infrastructure_zones
		WHERE site_id = ? AND domain = ?
		ORDER BY sort_order, name`

	return r.queryZones(ctx, query, siteID, string(domain))
}

// UpdateZone modifies an existing infrastructure zone.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - zone: Zone definition with updated fields
//
// Returns:
//   - error: ErrZoneNotFound if missing, otherwise the underlying database error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	err := repo.UpdateZone(ctx, zone)
func (r *SQLiteZoneRepository) UpdateZone(ctx context.Context, zone *InfrastructureZone) error {
	if zone == nil {
		return fmt.Errorf("zone is required")
	}
	if zone.Slug == "" {
		zone.Slug = device.GenerateSlug(zone.Name)
	}
	if err := ValidateZone(zone); err != nil {
		return err
	}

	settingsJSON, err := marshalSettings(zone.Settings)
	if err != nil {
		return err
	}

	query := `UPDATE infrastructure_zones SET
		name = ?, slug = ?, domain = ?, settings = ?, sort_order = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		zone.Name,
		zone.Slug,
		string(zone.Domain),
		settingsJSON,
		zone.SortOrder,
		zone.ID,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrZoneExists
		}
		return fmt.Errorf("updating infrastructure zone: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrZoneNotFound
	}

	return nil
}

// DeleteZone removes an infrastructure zone by ID.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - id: Zone identifier
//
// Returns:
//   - error: ErrZoneNotFound if missing, otherwise the underlying database error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	err := repo.DeleteZone(ctx, "zone-1")
func (r *SQLiteZoneRepository) DeleteZone(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is no-op after commit

	if _, execErr := tx.ExecContext(ctx, "DELETE FROM infrastructure_zone_rooms WHERE zone_id = ?", id); execErr != nil {
		return fmt.Errorf("deleting zone rooms: %w", execErr)
	}

	result, err := tx.ExecContext(ctx, "DELETE FROM infrastructure_zones WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting infrastructure zone: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrZoneNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// SetZoneRooms replaces the room membership for a zone.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - zoneID: Zone identifier
//   - roomIDs: Room identifiers to associate with the zone
//
// Returns:
//   - error: ErrZoneNotFound if zone missing, ErrRoomAlreadyInZoneDomain on conflicts, otherwise a database error
//
// Security: Uses a single transaction with parameterised SQL statements.
// Example:
//
//	err := repo.SetZoneRooms(ctx, "zone-1", []string{"room-1", "room-2"})
func (r *SQLiteZoneRepository) SetZoneRooms(ctx context.Context, zoneID string, roomIDs []string) error {
	if zoneID == "" {
		return fmt.Errorf("zone id is required")
	}

	uniqueRooms := dedupeZoneIDs(roomIDs)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is no-op after commit

	domain, err := r.loadZoneDomain(ctx, tx, zoneID)
	if err != nil {
		return err
	}

	if err := r.ensureRoomDomainAvailable(ctx, tx, zoneID, domain, uniqueRooms); err != nil {
		return err
	}

	if err := r.replaceZoneRooms(ctx, tx, zoneID, uniqueRooms); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// GetZoneRooms returns rooms assigned to a zone.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - zoneID: Zone identifier
//
// Returns:
//   - []Room: Rooms in the zone ordered by sort_order then name
//   - error: nil on success, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	rooms, err := repo.GetZoneRooms(ctx, "zone-1")
func (r *SQLiteZoneRepository) GetZoneRooms(ctx context.Context, zoneID string) ([]Room, error) {
	query := `SELECT r.id, r.area_id, r.name, r.slug, r.type, r.sort_order,
		r.climate_zone_id, r.audio_zone_id, r.settings, r.created_at, r.updated_at
		FROM rooms r
		JOIN infrastructure_zone_rooms zr ON r.id = zr.room_id
		WHERE zr.zone_id = ?
		ORDER BY r.sort_order, r.name`

	rows, err := r.db.QueryContext(ctx, query, zoneID)
	if err != nil {
		return nil, fmt.Errorf("querying zone rooms: %w", err)
	}
	defer rows.Close()

	var rooms []Room
	for rows.Next() {
		room, err := scanRoomRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning zone room: %w", err)
		}
		rooms = append(rooms, *room)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating zone rooms: %w", err)
	}

	return rooms, nil
}

// GetZoneRoomIDs returns room IDs assigned to a zone.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - zoneID: Zone identifier
//
// Returns:
//   - []string: Room IDs ordered alphabetically
//   - error: nil on success, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	ids, err := repo.GetZoneRoomIDs(ctx, "zone-1")
func (r *SQLiteZoneRepository) GetZoneRoomIDs(ctx context.Context, zoneID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT room_id FROM infrastructure_zone_rooms WHERE zone_id = ? ORDER BY room_id",
		zoneID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying zone room ids: %w", err)
	}
	defer rows.Close()

	var roomIDs []string
	for rows.Next() {
		var roomID string
		if err := rows.Scan(&roomID); err != nil {
			return nil, fmt.Errorf("scanning zone room id: %w", err)
		}
		roomIDs = append(roomIDs, roomID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating zone room ids: %w", err)
	}

	return roomIDs, nil
}

// GetZoneForRoom finds the zone for a room in a given domain.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - roomID: Room identifier
//   - domain: Domain to filter by
//
// Returns:
//   - *InfrastructureZone: Zone when found
//   - error: ErrZoneNotFound if none exists, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	zone, err := repo.GetZoneForRoom(ctx, "room-1", location.ZoneDomainClimate)
func (r *SQLiteZoneRepository) GetZoneForRoom(ctx context.Context, roomID string, domain ZoneDomain) (*InfrastructureZone, error) {
	query := `SELECT z.id, z.site_id, z.name, z.slug, z.domain, z.settings, z.sort_order, z.created_at, z.updated_at
		FROM infrastructure_zones z
		JOIN infrastructure_zone_rooms zr ON z.id = zr.zone_id
		WHERE zr.room_id = ? AND z.domain = ?`

	row := r.db.QueryRowContext(ctx, query, roomID, string(domain))
	zone, err := scanZoneRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrZoneNotFound
		}
		return nil, err
	}

	return zone, nil
}

// GetZonesForRoom lists all zones a room belongs to across domains.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - roomID: Room identifier
//
// Returns:
//   - []InfrastructureZone: Zones ordered by domain then sort_order
//   - error: nil on success, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	zones, err := repo.GetZonesForRoom(ctx, "room-1")
func (r *SQLiteZoneRepository) GetZonesForRoom(ctx context.Context, roomID string) ([]InfrastructureZone, error) {
	query := `SELECT z.id, z.site_id, z.name, z.slug, z.domain, z.settings, z.sort_order, z.created_at, z.updated_at
		FROM infrastructure_zones z
		JOIN infrastructure_zone_rooms zr ON z.id = zr.zone_id
		WHERE zr.room_id = ?
		ORDER BY z.domain, z.sort_order, z.name`

	return r.queryZones(ctx, query, roomID)
}

// GetAllRoomZoneMappings returns room_id → {domain: zone_id} for all rooms in one bulk query.
// This replaces the N+1 per-room GetZonesForRoom calls used in hierarchy building.
func (r *SQLiteZoneRepository) GetAllRoomZoneMappings(ctx context.Context) (map[string]map[string]string, error) {
	query := `SELECT zr.room_id, z.domain, z.id
		FROM infrastructure_zone_rooms zr
		JOIN infrastructure_zones z ON z.id = zr.zone_id
		ORDER BY zr.room_id, z.domain`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying room zone mappings: %w", err)
	}
	defer rows.Close()

	result := make(map[string]map[string]string)
	for rows.Next() {
		var roomID, domain, zoneID string
		if err := rows.Scan(&roomID, &domain, &zoneID); err != nil {
			return nil, fmt.Errorf("scanning room zone mapping: %w", err)
		}
		if result[roomID] == nil {
			result[roomID] = make(map[string]string)
		}
		result[roomID][domain] = zoneID
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating room zone mappings: %w", err)
	}

	return result, nil
}

// queryZones runs a zone query and returns zone models.
func (r *SQLiteZoneRepository) queryZones(ctx context.Context, query string, args ...any) ([]InfrastructureZone, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying zones: %w", err)
	}
	defer rows.Close()

	var zones []InfrastructureZone
	for rows.Next() {
		zone, err := scanZoneRow(rows)
		if err != nil {
			return nil, err
		}
		zones = append(zones, *zone)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating zones: %w", err)
	}

	return zones, nil
}

// zoneRowScanner is implemented by sql.Row and sql.Rows.
type zoneRowScanner interface {
	Scan(dest ...any) error
}

// scanZoneRow scans a single zone row into an InfrastructureZone.
func scanZoneRow(scanner zoneRowScanner) (*InfrastructureZone, error) {
	var zone InfrastructureZone
	var domain string
	var settingsJSON string
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&zone.ID,
		&zone.SiteID,
		&zone.Name,
		&zone.Slug,
		&domain,
		&settingsJSON,
		&zone.SortOrder,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrZoneNotFound
		}
		return nil, fmt.Errorf("scanning zone: %w", err)
	}

	zone.Domain = ZoneDomain(domain)
	zone.Settings = parseSettings(settingsJSON)
	var parseErr error
	if zone.CreatedAt, parseErr = parseTime(createdAt); parseErr != nil {
		return nil, fmt.Errorf("zone %s created_at: %w", zone.ID, parseErr)
	}
	if zone.UpdatedAt, parseErr = parseTime(updatedAt); parseErr != nil {
		return nil, fmt.Errorf("zone %s updated_at: %w", zone.ID, parseErr)
	}
	return &zone, nil
}

// marshalSettings serialises settings to JSON for storage.
func marshalSettings(settings Settings) (string, error) {
	if settings == nil {
		return "{}", nil
	}
	b, err := json.Marshal(settings)
	if err != nil {
		return "", fmt.Errorf("marshalling settings: %w", err)
	}
	return string(b), nil
}

// dedupeZoneIDs removes duplicate IDs while preserving order.
func dedupeZoneIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

// loadZoneDomain returns the domain for a zone in a transaction.
func (r *SQLiteZoneRepository) loadZoneDomain(ctx context.Context, tx *sql.Tx, zoneID string) (string, error) {
	var domain string
	if err := tx.QueryRowContext(ctx, "SELECT domain FROM infrastructure_zones WHERE id = ?", zoneID).Scan(&domain); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrZoneNotFound
		}
		return "", fmt.Errorf("loading zone domain: %w", err)
	}
	if !ValidZoneDomain(domain) {
		return "", fmt.Errorf("invalid zone domain: %s", domain)
	}
	return domain, nil
}

// ensureRoomDomainAvailable enforces one-zone-per-domain membership for rooms.
func (r *SQLiteZoneRepository) ensureRoomDomainAvailable(ctx context.Context, tx *sql.Tx, zoneID string, domain string, roomIDs []string) error {
	for _, roomID := range roomIDs {
		if roomID == "" {
			continue
		}
		var existing string
		err := tx.QueryRowContext(ctx,
			`SELECT z.id
			FROM infrastructure_zones z
			JOIN infrastructure_zone_rooms r ON z.id = r.zone_id
			WHERE r.room_id = ? AND z.domain = ? AND z.id != ?
			LIMIT 1`,
			roomID, domain, zoneID,
		).Scan(&existing)
		if err == nil {
			return ErrRoomAlreadyInZoneDomain
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("checking room assignment: %w", err)
		}
	}
	return nil
}

// replaceZoneRooms deletes existing assignments and inserts the new list.
func (r *SQLiteZoneRepository) replaceZoneRooms(ctx context.Context, tx *sql.Tx, zoneID string, roomIDs []string) error {
	if _, execErr := tx.ExecContext(ctx, "DELETE FROM infrastructure_zone_rooms WHERE zone_id = ?", zoneID); execErr != nil {
		return fmt.Errorf("clearing zone rooms: %w", execErr)
	}

	if len(roomIDs) == 0 {
		return nil
	}

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO infrastructure_zone_rooms (zone_id, room_id) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("preparing room insert: %w", err)
	}
	defer stmt.Close()

	for _, roomID := range roomIDs {
		if roomID == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx, zoneID, roomID); err != nil {
			return fmt.Errorf("inserting zone room: %w", err)
		}
	}

	return nil
}

// isUniqueConstraintError checks for SQLite unique constraint violations.
func isUniqueConstraintError(err error) bool {
	return err != nil && !errors.Is(err, sql.ErrNoRows) &&
		(strings.Contains(err.Error(), "UNIQUE constraint failed") ||
			strings.Contains(err.Error(), "unique constraint"))
}
