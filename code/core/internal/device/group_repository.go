package device

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// GroupRepository defines persistence operations for device groups.
type GroupRepository interface {
	// Create inserts a new device group.
	Create(ctx context.Context, group *DeviceGroup) error
	// GetByID retrieves a device group by ID.
	GetByID(ctx context.Context, id string) (*DeviceGroup, error)
	// List retrieves all device groups.
	List(ctx context.Context) ([]DeviceGroup, error)
	// Update modifies an existing device group.
	Update(ctx context.Context, group *DeviceGroup) error
	// Delete removes a device group by ID.
	Delete(ctx context.Context, id string) error

	// Member management (for static and hybrid groups)
	SetMembers(ctx context.Context, groupID string, deviceIDs []string) error
	GetMembers(ctx context.Context, groupID string) ([]GroupMember, error)
	GetMemberDeviceIDs(ctx context.Context, groupID string) ([]string, error)
}

// SQLiteGroupRepository implements GroupRepository using SQLite.
type SQLiteGroupRepository struct {
	db *sql.DB
}

// NewSQLiteGroupRepository creates a new SQLite-backed group repository.
//
// Parameters:
//   - db: Open SQLite connection used for group queries
//
// Returns:
//   - *SQLiteGroupRepository: Repository instance ready for use
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	repo := device.NewSQLiteGroupRepository(db)
func NewSQLiteGroupRepository(db *sql.DB) *SQLiteGroupRepository {
	return &SQLiteGroupRepository{db: db}
}

// Create inserts a new device group.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - group: Group definition to persist
//
// Returns:
//   - error: nil on success, ErrGroupExists on conflict, otherwise a database error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	err := repo.Create(ctx, &device.DeviceGroup{Name: "Cinema"})
func (r *SQLiteGroupRepository) Create(ctx context.Context, group *DeviceGroup) error {
	if group == nil {
		return fmt.Errorf("group is required")
	}

	if group.ID == "" {
		group.ID = GenerateID()
	}
	if group.Slug == "" {
		group.Slug = GenerateSlug(group.Name)
	}
	if group.Type == "" {
		group.Type = GroupTypeStatic
	}

	filterRules, err := marshalFilterRules(group.FilterRules)
	if err != nil {
		return err
	}

	query := `INSERT INTO device_groups (
			id, name, slug, description, type, filter_rules, icon, colour, sort_order
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = r.db.ExecContext(ctx, query,
		group.ID,
		group.Name,
		group.Slug,
		nullableString(group.Description),
		string(group.Type),
		filterRules,
		nullableString(group.Icon),
		nullableString(group.Colour),
		group.SortOrder,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrGroupExists
		}
		return fmt.Errorf("inserting device group: %w", err)
	}

	return nil
}

// GetByID retrieves a device group by ID.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - id: Unique device group identifier
//
// Returns:
//   - *DeviceGroup: Group definition when found
//   - error: ErrGroupNotFound if missing, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	group, err := repo.GetByID(ctx, "grp-1")
func (r *SQLiteGroupRepository) GetByID(ctx context.Context, id string) (*DeviceGroup, error) {
	query := `SELECT id, name, slug, description, type, filter_rules, icon, colour,
		sort_order, created_at, updated_at
		FROM device_groups WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	group, err := scanGroupRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		if errors.Is(err, ErrGroupNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("querying device group: %w", err)
	}

	return group, nil
}

// List retrieves all device groups.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - []DeviceGroup: Groups ordered by sort_order then name
//   - error: nil on success, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	groups, err := repo.List(ctx)
func (r *SQLiteGroupRepository) List(ctx context.Context) ([]DeviceGroup, error) {
	query := `SELECT id, name, slug, description, type, filter_rules, icon, colour,
		sort_order, created_at, updated_at
		FROM device_groups
		ORDER BY sort_order, name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying device groups: %w", err)
	}
	defer rows.Close()

	var groups []DeviceGroup
	for rows.Next() {
		group, err := scanGroupRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning device group: %w", err)
		}
		groups = append(groups, *group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating device groups: %w", err)
	}

	return groups, nil
}

// Update modifies an existing device group.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - group: Group definition with updated fields
//
// Returns:
//   - error: ErrGroupNotFound if missing, ErrGroupExists on conflict, otherwise the underlying database error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	err := repo.Update(ctx, group)
func (r *SQLiteGroupRepository) Update(ctx context.Context, group *DeviceGroup) error {
	if group == nil {
		return fmt.Errorf("group is required")
	}
	if group.Slug == "" {
		group.Slug = GenerateSlug(group.Name)
	}
	if group.Type == "" {
		group.Type = GroupTypeStatic
	}

	filterRules, err := marshalFilterRules(group.FilterRules)
	if err != nil {
		return err
	}

	query := `UPDATE device_groups SET
		name = ?, slug = ?, description = ?, type = ?, filter_rules = ?, icon = ?, colour = ?,
		sort_order = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		group.Name,
		group.Slug,
		nullableString(group.Description),
		string(group.Type),
		filterRules,
		nullableString(group.Icon),
		nullableString(group.Colour),
		group.SortOrder,
		group.ID,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrGroupExists
		}
		return fmt.Errorf("updating device group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrGroupNotFound
	}

	return nil
}

// Delete removes a device group by ID.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - id: Unique device group identifier
//
// Returns:
//   - error: ErrGroupNotFound if missing, otherwise the underlying database error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	err := repo.Delete(ctx, "grp-1")
func (r *SQLiteGroupRepository) Delete(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is no-op after commit

	if _, execErr := tx.ExecContext(ctx, "DELETE FROM device_group_members WHERE group_id = ?", id); execErr != nil {
		return fmt.Errorf("deleting group members: %w", execErr)
	}

	result, err := tx.ExecContext(ctx, "DELETE FROM device_groups WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting device group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrGroupNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// SetMembers replaces the explicit member list for a group.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - groupID: Unique device group identifier
//   - deviceIDs: Device identifiers to assign as explicit members
//
// Returns:
//   - error: nil on success, otherwise the underlying database error
//
// Security: Uses a single transaction with parameterised SQL statements.
// Example:
//
//	err := repo.SetMembers(ctx, "grp-1", []string{"dev-1", "dev-2"})
func (r *SQLiteGroupRepository) SetMembers(ctx context.Context, groupID string, deviceIDs []string) error {
	if groupID == "" {
		return fmt.Errorf("group id is required")
	}

	uniqueIDs := dedupeOrdered(deviceIDs)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is no-op after commit

	if _, err := tx.ExecContext(ctx, "DELETE FROM device_group_members WHERE group_id = ?", groupID); err != nil {
		return fmt.Errorf("clearing group members: %w", err)
	}

	if len(uniqueIDs) > 0 {
		stmt, err := tx.PrepareContext(ctx,
			"INSERT INTO device_group_members (group_id, device_id, sort_order) VALUES (?, ?, ?)",
		)
		if err != nil {
			return fmt.Errorf("preparing member insert: %w", err)
		}
		defer stmt.Close()

		for i, deviceID := range uniqueIDs {
			if _, err := stmt.ExecContext(ctx, groupID, deviceID, i); err != nil {
				return fmt.Errorf("inserting group member: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// GetMembers returns explicit group members for a group.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - groupID: Unique device group identifier
//
// Returns:
//   - []GroupMember: Members ordered by sort_order
//   - error: nil on success, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	members, err := repo.GetMembers(ctx, "grp-1")
func (r *SQLiteGroupRepository) GetMembers(ctx context.Context, groupID string) ([]GroupMember, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT group_id, device_id, sort_order, created_at
		FROM device_group_members
		WHERE group_id = ?
		ORDER BY sort_order, device_id`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying group members: %w", err)
	}
	defer rows.Close()

	var members []GroupMember
	for rows.Next() {
		var member GroupMember
		var createdAt string
		if scanErr := rows.Scan(&member.GroupID, &member.DeviceID, &member.SortOrder, &createdAt); scanErr != nil {
			return nil, fmt.Errorf("scanning group member: %w", scanErr)
		}
		member.CreatedAt, err = parseGroupTimestamp(createdAt)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating group members: %w", err)
	}

	return members, nil
}

// GetMemberDeviceIDs returns explicit group member device IDs for a group.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - groupID: Unique device group identifier
//
// Returns:
//   - []string: Device IDs ordered by sort_order
//   - error: nil on success, otherwise the underlying query error
//
// Security: Uses parameterised SQL queries to prevent injection.
// Example:
//
//	ids, err := repo.GetMemberDeviceIDs(ctx, "grp-1")
func (r *SQLiteGroupRepository) GetMemberDeviceIDs(ctx context.Context, groupID string) ([]string, error) {
	return queryStringList(ctx, r.db,
		`SELECT device_id
		FROM device_group_members
		WHERE group_id = ?
		ORDER BY sort_order, device_id`,
		"querying group member ids",
		groupID,
	)
}

// scanGroupRow scans a device group from a row scanner.
func scanGroupRow(scanner rowScanner) (*DeviceGroup, error) {
	var group DeviceGroup
	var description sql.NullString
	var filterRules sql.NullString
	var icon sql.NullString
	var colour sql.NullString
	var groupType string
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&group.ID,
		&group.Name,
		&group.Slug,
		&description,
		&groupType,
		&filterRules,
		&icon,
		&colour,
		&group.SortOrder,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	group.Type = GroupType(groupType)
	if description.Valid {
		group.Description = &description.String
	}
	if icon.Valid {
		group.Icon = &icon.String
	}
	if colour.Valid {
		group.Colour = &colour.String
	}

	if filterRules.Valid && strings.TrimSpace(filterRules.String) != "" && strings.TrimSpace(filterRules.String) != "null" {
		var rules FilterRules
		if unmarshalErr := json.Unmarshal([]byte(filterRules.String), &rules); unmarshalErr != nil {
			return nil, fmt.Errorf("unmarshalling filter rules: %w", unmarshalErr)
		}
		group.FilterRules = &rules
	}

	group.CreatedAt, err = parseGroupTimestamp(createdAt)
	if err != nil {
		return nil, err
	}
	group.UpdatedAt, err = parseGroupTimestamp(updatedAt)
	if err != nil {
		return nil, err
	}

	return &group, nil
}

// marshalFilterRules serialises filter rules for storage.
func marshalFilterRules(rules *FilterRules) (any, error) {
	if rules == nil {
		return nil, nil
	}

	b, err := json.Marshal(rules)
	if err != nil {
		return nil, fmt.Errorf("marshalling filter rules: %w", err)
	}
	return string(b), nil
}

// parseGroupTimestamp parses a timestamp stored in SQLite.
func parseGroupTimestamp(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("timestamp is empty")
	}

	timestamp, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return timestamp, nil
	}

	fallback, fallbackErr := time.Parse("2006-01-02T15:04:05Z", value)
	if fallbackErr == nil {
		return fallback, nil
	}

	return time.Time{}, fmt.Errorf("parsing timestamp: %w", err)
}

// dedupeOrdered removes duplicate values while preserving order.
func dedupeOrdered(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}

	return unique
}
