package device

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// TagRepository manages device tag associations in SQLite.
type TagRepository interface {
	// SetTags replaces all tags for a device (delete + insert in transaction).
	SetTags(ctx context.Context, deviceID string, tags []string) error

	// GetTags returns all tags for a device.
	GetTags(ctx context.Context, deviceID string) ([]string, error)

	// AddTag adds a single tag to a device (idempotent).
	AddTag(ctx context.Context, deviceID, tag string) error

	// RemoveTag removes a single tag from a device.
	RemoveTag(ctx context.Context, deviceID, tag string) error

	// ListDevicesByTag returns all device IDs that have the given tag.
	ListDevicesByTag(ctx context.Context, tag string) ([]string, error)

	// ListAllTags returns all unique tags across all devices, sorted alphabetically.
	ListAllTags(ctx context.Context) ([]string, error)

	// GetTagsForDevices returns a map of deviceID to tags for bulk loading.
	// Used by Registry.RefreshCache to populate device tags efficiently.
	GetTagsForDevices(ctx context.Context, deviceIDs []string) (map[string][]string, error)
}

// SQLiteTagRepository implements TagRepository using SQLite.
type SQLiteTagRepository struct {
	db *sql.DB
}

// NewSQLiteTagRepository creates a new SQLite-backed tag repository.
//
// Parameters:
//   - db: Open SQLite connection used for tag queries
//
// Returns:
//   - *SQLiteTagRepository: Repository instance ready for use
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	repo := device.NewSQLiteTagRepository(db)
func NewSQLiteTagRepository(db *sql.DB) *SQLiteTagRepository {
	return &SQLiteTagRepository{db: db}
}

// SetTags replaces all tags for a device.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - deviceID: Unique device identifier
//   - tags: Tag values to replace the current set
//
// Returns:
//   - error: nil on success, otherwise the underlying database error
//
// Security: Uses a single transaction with parameterised SQL statements.
// Example:
//
//	err := repo.SetTags(ctx, "dev-1", []string{"accent", "escape_lighting"})
func (r *SQLiteTagRepository) SetTags(ctx context.Context, deviceID string, tags []string) error {
	if deviceID == "" {
		return fmt.Errorf("device id is required")
	}

	normalised := normaliseTags(tags)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is no-op after commit

	if _, err := tx.ExecContext(ctx, "DELETE FROM device_tags WHERE device_id = ?", deviceID); err != nil {
		return fmt.Errorf("clearing device tags: %w", err)
	}

	if len(normalised) > 0 {
		stmt, err := tx.PrepareContext(ctx, "INSERT INTO device_tags (device_id, tag) VALUES (?, ?)")
		if err != nil {
			return fmt.Errorf("preparing tag insert: %w", err)
		}
		defer stmt.Close()

		for _, tag := range normalised {
			if _, err := stmt.ExecContext(ctx, deviceID, tag); err != nil {
				return fmt.Errorf("inserting tag: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// GetTags returns all tags for a device.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - deviceID: Unique device identifier
//
// Returns:
//   - []string: Tags associated with the device, sorted alphabetically
//   - error: nil on success, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	tags, err := repo.GetTags(ctx, "dev-1")
func (r *SQLiteTagRepository) GetTags(ctx context.Context, deviceID string) ([]string, error) {
	return queryStringList(ctx, r.db, "SELECT tag FROM device_tags WHERE device_id = ? ORDER BY tag", "querying device tags", deviceID)
}

// AddTag adds a single tag to a device.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - deviceID: Unique device identifier
//   - tag: Tag value to add
//
// Returns:
//   - error: nil on success, otherwise the underlying database error
//
// Security: Uses INSERT OR IGNORE with parameterised SQL to avoid duplicates.
// Example:
//
//	err := repo.AddTag(ctx, "dev-1", "accent")
func (r *SQLiteTagRepository) AddTag(ctx context.Context, deviceID, tag string) error {
	if deviceID == "" {
		return fmt.Errorf("device id is required")
	}

	normalised := normaliseTag(tag)
	if normalised == "" {
		return fmt.Errorf("tag is required")
	}

	_, err := r.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO device_tags (device_id, tag) VALUES (?, ?)",
		deviceID,
		normalised,
	)
	if err != nil {
		return fmt.Errorf("adding device tag: %w", err)
	}

	return nil
}

// RemoveTag removes a single tag from a device.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - deviceID: Unique device identifier
//   - tag: Tag value to remove
//
// Returns:
//   - error: nil on success, otherwise the underlying database error
//
// Security: Uses parameterised SQL statements.
// Example:
//
//	err := repo.RemoveTag(ctx, "dev-1", "accent")
func (r *SQLiteTagRepository) RemoveTag(ctx context.Context, deviceID, tag string) error {
	if deviceID == "" {
		return fmt.Errorf("device id is required")
	}

	normalised := normaliseTag(tag)
	if normalised == "" {
		return fmt.Errorf("tag is required")
	}

	_, err := r.db.ExecContext(ctx,
		"DELETE FROM device_tags WHERE device_id = ? AND tag = ?",
		deviceID,
		normalised,
	)
	if err != nil {
		return fmt.Errorf("removing device tag: %w", err)
	}

	return nil
}

// ListDevicesByTag returns all device IDs that have the given tag.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - tag: Tag value to filter by
//
// Returns:
//   - []string: Device IDs associated with the tag, sorted alphabetically
//   - error: nil on success, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	ids, err := repo.ListDevicesByTag(ctx, "accent")
func (r *SQLiteTagRepository) ListDevicesByTag(ctx context.Context, tag string) ([]string, error) {
	normalised := normaliseTag(tag)
	if normalised == "" {
		return []string{}, nil
	}

	rows, err := r.db.QueryContext(ctx,
		"SELECT device_id FROM device_tags WHERE tag = ? ORDER BY device_id",
		normalised,
	)
	if err != nil {
		return nil, fmt.Errorf("querying devices by tag: %w", err)
	}
	defer rows.Close()

	var deviceIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning device id: %w", err)
		}
		deviceIDs = append(deviceIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating device ids: %w", err)
	}

	return deviceIDs, nil
}

// ListAllTags returns all unique tags across all devices, sorted alphabetically.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - []string: All unique tags in the system
//   - error: nil on success, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	tags, err := repo.ListAllTags(ctx)
func (r *SQLiteTagRepository) ListAllTags(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT DISTINCT tag FROM device_tags ORDER BY tag")
	if err != nil {
		return nil, fmt.Errorf("querying all tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scanning tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tags: %w", err)
	}

	return tags, nil
}

// GetTagsForDevices returns tags for multiple device IDs in a single query.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - deviceIDs: Device identifiers to fetch tags for
//
// Returns:
//   - map[string][]string: Device ID to tags mapping
//   - error: nil on success, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	tagsByDevice, err := repo.GetTagsForDevices(ctx, []string{"dev-1", "dev-2"})
func (r *SQLiteTagRepository) GetTagsForDevices(ctx context.Context, deviceIDs []string) (map[string][]string, error) {
	result := make(map[string][]string)
	if len(deviceIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(deviceIDs))
	args := make([]any, 0, len(deviceIDs))
	for i, id := range deviceIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	var builder strings.Builder
	builder.WriteString("SELECT device_id, tag FROM device_tags WHERE device_id IN (")
	builder.WriteString(strings.Join(placeholders, ","))
	builder.WriteString(") ORDER BY device_id, tag")
	query := builder.String()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying tags for devices: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var deviceID string
		var tag string
		if err := rows.Scan(&deviceID, &tag); err != nil {
			return nil, fmt.Errorf("scanning device tag: %w", err)
		}
		result[deviceID] = append(result[deviceID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating device tags: %w", err)
	}

	return result, nil
}

// normaliseTag trims whitespace and lowercases tag values.
func normaliseTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}

// normaliseTags normalises and deduplicates a tag slice.
func normaliseTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(tags))
	var normalised []string
	for _, tag := range tags {
		n := normaliseTag(tag)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		normalised = append(normalised, n)
	}

	sort.Strings(normalised)
	return normalised
}

// queryStringList executes a query and returns a string slice result.
func queryStringList(ctx context.Context, db *sql.DB, query string, op string, args ...any) ([]string, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return values, nil
}
