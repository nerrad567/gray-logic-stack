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

// Repository defines the interface for device persistence operations.
// This abstraction allows for different implementations (SQLite, mock, etc.)
// and enables unit testing without database dependencies.
type Repository interface {
	// GetByID retrieves a device by its unique identifier.
	// Returns ErrDeviceNotFound if the device does not exist.
	GetByID(ctx context.Context, id string) (*Device, error)

	// List retrieves all devices.
	List(ctx context.Context) ([]Device, error)

	// ListByRoom retrieves all devices in a specific room.
	ListByRoom(ctx context.Context, roomID string) ([]Device, error)

	// ListByArea retrieves all devices in a specific area.
	ListByArea(ctx context.Context, areaID string) ([]Device, error)

	// ListByDomain retrieves all devices in a specific domain (lighting, climate, etc.).
	ListByDomain(ctx context.Context, domain Domain) ([]Device, error)

	// ListByProtocol retrieves all devices using a specific protocol (KNX, DALI, etc.).
	ListByProtocol(ctx context.Context, protocol Protocol) ([]Device, error)

	// Create inserts a new device.
	// Returns ErrDeviceExists if a device with the same ID already exists.
	Create(ctx context.Context, device *Device) error

	// Update modifies an existing device.
	// Returns ErrDeviceNotFound if the device does not exist.
	Update(ctx context.Context, device *Device) error

	// Delete removes a device by ID.
	// Returns ErrDeviceNotFound if the device does not exist.
	Delete(ctx context.Context, id string) error

	// UpdateState updates only the state fields of a device.
	// This is optimised for frequent state changes from protocol bridges.
	UpdateState(ctx context.Context, id string, state State) error

	// UpdateHealth updates the health status and last seen timestamp.
	UpdateHealth(ctx context.Context, id string, status HealthStatus, lastSeen time.Time) error
}

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite-backed repository.
// The db parameter should be an open SQLite connection.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// GetByID retrieves a device by its unique identifier.
func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*Device, error) {
	query := `
		SELECT id, room_id, area_id, name, slug, type, domain, protocol, address,
			gateway_id, capabilities, config, state, state_updated_at,
			health_status, health_last_seen, phm_enabled, phm_baseline,
			manufacturer, model, firmware_version, created_at, updated_at
		FROM devices
		WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	device, err := scanDevice(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("querying device by id: %w", err)
	}
	return device, nil
}

// List retrieves all devices.
func (r *SQLiteRepository) List(ctx context.Context) ([]Device, error) {
	query := `
		SELECT id, room_id, area_id, name, slug, type, domain, protocol, address,
			gateway_id, capabilities, config, state, state_updated_at,
			health_status, health_last_seen, phm_enabled, phm_baseline,
			manufacturer, model, firmware_version, created_at, updated_at
		FROM devices
		ORDER BY name`

	return r.queryDevices(ctx, query)
}

// ListByRoom retrieves all devices in a specific room.
func (r *SQLiteRepository) ListByRoom(ctx context.Context, roomID string) ([]Device, error) {
	query := `
		SELECT id, room_id, area_id, name, slug, type, domain, protocol, address,
			gateway_id, capabilities, config, state, state_updated_at,
			health_status, health_last_seen, phm_enabled, phm_baseline,
			manufacturer, model, firmware_version, created_at, updated_at
		FROM devices
		WHERE room_id = ?
		ORDER BY name`

	return r.queryDevices(ctx, query, roomID)
}

// ListByArea retrieves all devices in a specific area.
func (r *SQLiteRepository) ListByArea(ctx context.Context, areaID string) ([]Device, error) {
	query := `
		SELECT id, room_id, area_id, name, slug, type, domain, protocol, address,
			gateway_id, capabilities, config, state, state_updated_at,
			health_status, health_last_seen, phm_enabled, phm_baseline,
			manufacturer, model, firmware_version, created_at, updated_at
		FROM devices
		WHERE area_id = ?
		ORDER BY name`

	return r.queryDevices(ctx, query, areaID)
}

// ListByDomain retrieves all devices in a specific domain.
func (r *SQLiteRepository) ListByDomain(ctx context.Context, domain Domain) ([]Device, error) {
	query := `
		SELECT id, room_id, area_id, name, slug, type, domain, protocol, address,
			gateway_id, capabilities, config, state, state_updated_at,
			health_status, health_last_seen, phm_enabled, phm_baseline,
			manufacturer, model, firmware_version, created_at, updated_at
		FROM devices
		WHERE domain = ?
		ORDER BY name`

	return r.queryDevices(ctx, query, string(domain))
}

// ListByProtocol retrieves all devices using a specific protocol.
func (r *SQLiteRepository) ListByProtocol(ctx context.Context, protocol Protocol) ([]Device, error) {
	query := `
		SELECT id, room_id, area_id, name, slug, type, domain, protocol, address,
			gateway_id, capabilities, config, state, state_updated_at,
			health_status, health_last_seen, phm_enabled, phm_baseline,
			manufacturer, model, firmware_version, created_at, updated_at
		FROM devices
		WHERE protocol = ?
		ORDER BY name`

	return r.queryDevices(ctx, query, string(protocol))
}

// Create inserts a new device.
func (r *SQLiteRepository) Create(ctx context.Context, device *Device) error {
	// Marshal JSON fields
	addressJSON, err := json.Marshal(device.Address)
	if err != nil {
		return fmt.Errorf("marshalling address: %w", err)
	}

	capsJSON, err := json.Marshal(device.Capabilities)
	if err != nil {
		return fmt.Errorf("marshalling capabilities: %w", err)
	}

	configJSON, err := json.Marshal(device.Config)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	stateJSON, err := json.Marshal(device.State)
	if err != nil {
		return fmt.Errorf("marshalling state: %w", err)
	}

	var phmBaselineJSON []byte
	if device.PHMBaseline != nil {
		phmBaselineJSON, err = json.Marshal(device.PHMBaseline)
		if err != nil {
			return fmt.Errorf("marshalling phm_baseline: %w", err)
		}
	}

	// Set timestamps if not set
	now := time.Now().UTC()
	if device.CreatedAt.IsZero() {
		device.CreatedAt = now
	}
	device.UpdatedAt = now

	query := `
		INSERT INTO devices (
			id, room_id, area_id, name, slug, type, domain, protocol, address,
			gateway_id, capabilities, config, state, state_updated_at,
			health_status, health_last_seen, phm_enabled, phm_baseline,
			manufacturer, model, firmware_version, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?, ?
		)`

	_, err = r.db.ExecContext(ctx, query,
		device.ID,
		nullableString(device.RoomID),
		nullableString(device.AreaID),
		device.Name,
		device.Slug,
		string(device.Type),
		string(device.Domain),
		string(device.Protocol),
		string(addressJSON),
		nullableString(device.GatewayID),
		string(capsJSON),
		string(configJSON),
		string(stateJSON),
		nullableTime(device.StateUpdatedAt),
		string(device.HealthStatus),
		nullableTime(device.HealthLastSeen),
		boolToInt(device.PHMEnabled),
		nullableBytes(phmBaselineJSON),
		nullableString(device.Manufacturer),
		nullableString(device.Model),
		nullableString(device.FirmwareVersion),
		device.CreatedAt.Format(time.RFC3339),
		device.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		// Check for unique constraint violation
		if isUniqueConstraintError(err) {
			return ErrDeviceExists
		}
		return fmt.Errorf("inserting device: %w", err)
	}

	return nil
}

// Update modifies an existing device.
func (r *SQLiteRepository) Update(ctx context.Context, device *Device) error {
	// First check the device exists
	exists, err := r.exists(ctx, device.ID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrDeviceNotFound
	}

	// Marshal JSON fields
	addressJSON, err := json.Marshal(device.Address)
	if err != nil {
		return fmt.Errorf("marshalling address: %w", err)
	}

	capsJSON, err := json.Marshal(device.Capabilities)
	if err != nil {
		return fmt.Errorf("marshalling capabilities: %w", err)
	}

	configJSON, err := json.Marshal(device.Config)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	stateJSON, err := json.Marshal(device.State)
	if err != nil {
		return fmt.Errorf("marshalling state: %w", err)
	}

	var phmBaselineJSON []byte
	if device.PHMBaseline != nil {
		phmBaselineJSON, err = json.Marshal(device.PHMBaseline)
		if err != nil {
			return fmt.Errorf("marshalling phm_baseline: %w", err)
		}
	}

	device.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE devices SET
			room_id = ?, area_id = ?, name = ?, slug = ?, type = ?,
			domain = ?, protocol = ?, address = ?, gateway_id = ?,
			capabilities = ?, config = ?, state = ?, state_updated_at = ?,
			health_status = ?, health_last_seen = ?, phm_enabled = ?, phm_baseline = ?,
			manufacturer = ?, model = ?, firmware_version = ?, updated_at = ?
		WHERE id = ?`

	_, err = r.db.ExecContext(ctx, query,
		nullableString(device.RoomID),
		nullableString(device.AreaID),
		device.Name,
		device.Slug,
		string(device.Type),
		string(device.Domain),
		string(device.Protocol),
		string(addressJSON),
		nullableString(device.GatewayID),
		string(capsJSON),
		string(configJSON),
		string(stateJSON),
		nullableTime(device.StateUpdatedAt),
		string(device.HealthStatus),
		nullableTime(device.HealthLastSeen),
		boolToInt(device.PHMEnabled),
		nullableBytes(phmBaselineJSON),
		nullableString(device.Manufacturer),
		nullableString(device.Model),
		nullableString(device.FirmwareVersion),
		device.UpdatedAt.Format(time.RFC3339),
		device.ID,
	)
	if err != nil {
		return fmt.Errorf("updating device: %w", err)
	}

	return nil
}

// Delete removes a device by ID.
func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM devices WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting device: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// UpdateState merges the given state fields into the device's existing state.
// This allows partial updates (e.g., updating "on" without losing "level").
func (r *SQLiteRepository) UpdateState(ctx context.Context, id string, state State) error {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshalling state: %w", err)
	}

	now := time.Now().UTC()
	// Use json_patch to merge new state into existing state.
	// json_patch(target, patch) applies patch keys to target, preserving
	// existing keys not present in patch.
	query := `
		UPDATE devices
		SET state = json_patch(COALESCE(state, '{}'), ?),
		    state_updated_at = ?,
		    updated_at = ?
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		string(stateJSON),
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
		id,
	)
	if err != nil {
		return fmt.Errorf("updating device state: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// UpdateHealth updates the health status and last seen timestamp.
func (r *SQLiteRepository) UpdateHealth(ctx context.Context, id string, status HealthStatus, lastSeen time.Time) error {
	now := time.Now().UTC()
	query := `
		UPDATE devices
		SET health_status = ?, health_last_seen = ?, updated_at = ?
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		string(status),
		lastSeen.UTC().Format(time.RFC3339),
		now.Format(time.RFC3339),
		id,
	)
	if err != nil {
		return fmt.Errorf("updating device health: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// queryDevices executes a query and returns a slice of devices.
func (r *SQLiteRepository) queryDevices(ctx context.Context, query string, args ...any) ([]Device, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying devices: %w", err)
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		device, err := scanDeviceFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning device: %w", err)
		}
		devices = append(devices, *device)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating devices: %w", err)
	}

	return devices, nil
}

// exists checks if a device with the given ID exists.
func (r *SQLiteRepository) exists(ctx context.Context, id string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM devices WHERE id = ?", id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking device exists: %w", err)
	}
	return count > 0, nil
}

// rowScanner is an interface that sql.Row and sql.Rows both implement.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanDevice scans a single row into a Device.
func scanDevice(row *sql.Row) (*Device, error) {
	return scanDeviceRow(row)
}

// scanDeviceFromRows scans a rows result into a Device.
func scanDeviceFromRows(rows *sql.Rows) (*Device, error) {
	return scanDeviceRow(rows)
}

// scanDeviceRow scans a row or rows result into a Device.
func scanDeviceRow(scanner rowScanner) (*Device, error) { //nolint:gocognit,gocyclo // scans many nullable columns into Device struct
	var d Device
	var roomID, areaID, gatewayID sql.NullString
	var stateUpdatedAt, healthLastSeen sql.NullString
	var manufacturer, model, firmwareVersion sql.NullString
	var addressJSON, capsJSON, configJSON, stateJSON string
	var phmBaselineJSON sql.NullString
	var phmEnabled int
	var createdAt, updatedAt string
	var deviceType, domain, protocol, healthStatus string

	err := scanner.Scan(
		&d.ID,
		&roomID,
		&areaID,
		&d.Name,
		&d.Slug,
		&deviceType,
		&domain,
		&protocol,
		&addressJSON,
		&gatewayID,
		&capsJSON,
		&configJSON,
		&stateJSON,
		&stateUpdatedAt,
		&healthStatus,
		&healthLastSeen,
		&phmEnabled,
		&phmBaselineJSON,
		&manufacturer,
		&model,
		&firmwareVersion,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Set type fields
	d.Type = DeviceType(deviceType)
	d.Domain = Domain(domain)
	d.Protocol = Protocol(protocol)
	d.HealthStatus = HealthStatus(healthStatus)
	d.PHMEnabled = phmEnabled != 0

	// Set nullable strings
	if roomID.Valid {
		d.RoomID = &roomID.String
	}
	if areaID.Valid {
		d.AreaID = &areaID.String
	}
	if gatewayID.Valid {
		d.GatewayID = &gatewayID.String
	}
	if manufacturer.Valid {
		d.Manufacturer = &manufacturer.String
	}
	if model.Valid {
		d.Model = &model.String
	}
	if firmwareVersion.Valid {
		d.FirmwareVersion = &firmwareVersion.String
	}

	// Parse timestamps
	if stateUpdatedAt.Valid {
		t, err := time.Parse(time.RFC3339, stateUpdatedAt.String)
		if err == nil {
			d.StateUpdatedAt = &t
		}
	}
	if healthLastSeen.Valid {
		t, err := time.Parse(time.RFC3339, healthLastSeen.String)
		if err == nil {
			d.HealthLastSeen = &t
		}
	}

	var parseErr error
	d.CreatedAt, parseErr = time.Parse(time.RFC3339, createdAt)
	if parseErr != nil {
		return nil, fmt.Errorf("parsing created_at: %w", parseErr)
	}
	d.UpdatedAt, parseErr = time.Parse(time.RFC3339, updatedAt)
	if parseErr != nil {
		return nil, fmt.Errorf("parsing updated_at: %w", parseErr)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal([]byte(addressJSON), &d.Address); err != nil {
		return nil, fmt.Errorf("unmarshalling address: %w", err)
	}

	if err := json.Unmarshal([]byte(capsJSON), &d.Capabilities); err != nil {
		return nil, fmt.Errorf("unmarshalling capabilities: %w", err)
	}

	if err := json.Unmarshal([]byte(configJSON), &d.Config); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	if err := json.Unmarshal([]byte(stateJSON), &d.State); err != nil {
		return nil, fmt.Errorf("unmarshalling state: %w", err)
	}

	if phmBaselineJSON.Valid && phmBaselineJSON.String != "" {
		var baseline PHMBaseline
		if err := json.Unmarshal([]byte(phmBaselineJSON.String), &baseline); err != nil {
			return nil, fmt.Errorf("unmarshalling phm_baseline: %w", err)
		}
		d.PHMBaseline = &baseline
	}

	return &d, nil
}

// nullableString returns a sql.NullString for optional string pointers.
func nullableString(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// nullableTime returns a sql.NullString for optional time pointers (as RFC3339 strings).
func nullableTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.UTC().Format(time.RFC3339), Valid: true}
}

// nullableBytes returns a sql.NullString for optional byte slices.
func nullableBytes(b []byte) sql.NullString {
	if b == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}

// boolToInt converts a boolean to 0/1 for SQLite storage.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// isUniqueConstraintError checks if an error is a SQLite unique constraint violation.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "unique constraint")
}
