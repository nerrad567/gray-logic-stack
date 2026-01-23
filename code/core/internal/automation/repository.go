package automation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Repository defines the interface for scene persistence.
// This abstraction allows different implementations (SQLite, mock, etc.)
// and enables unit testing without database dependencies.
type Repository interface {
	// Scene CRUD
	GetByID(ctx context.Context, id string) (*Scene, error)
	GetBySlug(ctx context.Context, slug string) (*Scene, error)
	List(ctx context.Context) ([]Scene, error)
	ListByRoom(ctx context.Context, roomID string) ([]Scene, error)
	ListByArea(ctx context.Context, areaID string) ([]Scene, error)
	ListByCategory(ctx context.Context, category Category) ([]Scene, error)
	Create(ctx context.Context, scene *Scene) error
	Update(ctx context.Context, scene *Scene) error
	Delete(ctx context.Context, id string) error

	// Execution logging
	CreateExecution(ctx context.Context, exec *SceneExecution) error
	UpdateExecution(ctx context.Context, exec *SceneExecution) error
	GetExecution(ctx context.Context, id string) (*SceneExecution, error)
	ListExecutions(ctx context.Context, sceneID string, limit int) ([]SceneExecution, error)
}

// sceneColumns is the SELECT column list for scene queries.
const sceneColumns = `id, name, slug, description, room_id, area_id, enabled, priority,
			icon, colour, category, actions, sort_order, created_at, updated_at`

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite-backed repository.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// GetByID retrieves a scene by its unique identifier.
func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*Scene, error) {
	query := `SELECT ` + sceneColumns + ` FROM scenes WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	scene, err := scanScene(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSceneNotFound
		}
		return nil, fmt.Errorf("querying scene by id: %w", err)
	}
	return scene, nil
}

// GetBySlug retrieves a scene by its slug.
func (r *SQLiteRepository) GetBySlug(ctx context.Context, slug string) (*Scene, error) {
	query := `SELECT ` + sceneColumns + ` FROM scenes WHERE slug = ?`

	row := r.db.QueryRowContext(ctx, query, slug)
	scene, err := scanScene(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSceneNotFound
		}
		return nil, fmt.Errorf("querying scene by slug: %w", err)
	}
	return scene, nil
}

// List retrieves all scenes ordered by sort_order then name.
func (r *SQLiteRepository) List(ctx context.Context) ([]Scene, error) {
	query := `SELECT ` + sceneColumns + ` FROM scenes ORDER BY sort_order, name`
	return r.queryScenes(ctx, query)
}

// ListByRoom retrieves all scenes in a specific room.
func (r *SQLiteRepository) ListByRoom(ctx context.Context, roomID string) ([]Scene, error) {
	query := `SELECT ` + sceneColumns + ` FROM scenes WHERE room_id = ? ORDER BY sort_order, name`
	return r.queryScenes(ctx, query, roomID)
}

// ListByArea retrieves all scenes in a specific area.
func (r *SQLiteRepository) ListByArea(ctx context.Context, areaID string) ([]Scene, error) {
	query := `SELECT ` + sceneColumns + ` FROM scenes WHERE area_id = ? ORDER BY sort_order, name`
	return r.queryScenes(ctx, query, areaID)
}

// ListByCategory retrieves all scenes in a specific category.
func (r *SQLiteRepository) ListByCategory(ctx context.Context, category Category) ([]Scene, error) {
	query := `SELECT ` + sceneColumns + ` FROM scenes WHERE category = ? ORDER BY sort_order, name`
	return r.queryScenes(ctx, query, string(category))
}

// Create inserts a new scene.
func (r *SQLiteRepository) Create(ctx context.Context, scene *Scene) error {
	actionsJSON, err := json.Marshal(scene.Actions)
	if err != nil {
		return fmt.Errorf("marshalling actions: %w", err)
	}

	now := time.Now().UTC()
	if scene.CreatedAt.IsZero() {
		scene.CreatedAt = now
	}
	scene.UpdatedAt = now

	query := `
		INSERT INTO scenes (
			id, name, slug, description, room_id, area_id, enabled, priority,
			icon, colour, category, actions, sort_order, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = r.db.ExecContext(ctx, query,
		scene.ID,
		scene.Name,
		scene.Slug,
		nullableString(scene.Description),
		nullableString(scene.RoomID),
		nullableString(scene.AreaID),
		boolToInt(scene.Enabled),
		scene.Priority,
		nullableString(scene.Icon),
		nullableString(scene.Colour),
		nullableCategory(scene.Category),
		string(actionsJSON),
		scene.SortOrder,
		scene.CreatedAt.Format(time.RFC3339),
		scene.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrSceneExists
		}
		return fmt.Errorf("inserting scene: %w", err)
	}
	return nil
}

// Update modifies an existing scene.
func (r *SQLiteRepository) Update(ctx context.Context, scene *Scene) error {
	actionsJSON, err := json.Marshal(scene.Actions)
	if err != nil {
		return fmt.Errorf("marshalling actions: %w", err)
	}

	scene.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE scenes SET
			name = ?, slug = ?, description = ?, room_id = ?, area_id = ?,
			enabled = ?, priority = ?, icon = ?, colour = ?, category = ?,
			actions = ?, sort_order = ?, updated_at = ?
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		scene.Name,
		scene.Slug,
		nullableString(scene.Description),
		nullableString(scene.RoomID),
		nullableString(scene.AreaID),
		boolToInt(scene.Enabled),
		scene.Priority,
		nullableString(scene.Icon),
		nullableString(scene.Colour),
		nullableCategory(scene.Category),
		string(actionsJSON),
		scene.SortOrder,
		scene.UpdatedAt.Format(time.RFC3339),
		scene.ID,
	)
	if err != nil {
		return fmt.Errorf("updating scene: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrSceneNotFound
	}
	return nil
}

// Delete removes a scene by ID.
func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM scenes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting scene: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrSceneNotFound
	}
	return nil
}

// CreateExecution inserts a new execution record.
func (r *SQLiteRepository) CreateExecution(ctx context.Context, exec *SceneExecution) error {
	failuresJSON, err := marshalFailures(exec.Failures)
	if err != nil {
		return fmt.Errorf("marshalling failures: %w", err)
	}

	query := `
		INSERT INTO scene_executions (
			id, scene_id, triggered_at, started_at, completed_at,
			trigger_type, trigger_source, status,
			actions_total, actions_completed, actions_failed, actions_skipped,
			failures, duration_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = r.db.ExecContext(ctx, query,
		exec.ID,
		exec.SceneID,
		exec.TriggeredAt.Format(time.RFC3339),
		nullableTime(exec.StartedAt),
		nullableTime(exec.CompletedAt),
		exec.TriggerType,
		nullableString(exec.TriggerSource),
		string(exec.Status),
		exec.ActionsTotal,
		exec.ActionsCompleted,
		exec.ActionsFailed,
		exec.ActionsSkipped,
		failuresJSON,
		exec.DurationMS,
	)
	if err != nil {
		return fmt.Errorf("inserting execution: %w", err)
	}
	return nil
}

// UpdateExecution updates an existing execution record.
func (r *SQLiteRepository) UpdateExecution(ctx context.Context, exec *SceneExecution) error {
	failuresJSON, err := marshalFailures(exec.Failures)
	if err != nil {
		return fmt.Errorf("marshalling failures: %w", err)
	}

	query := `
		UPDATE scene_executions SET
			started_at = ?, completed_at = ?, status = ?,
			actions_total = ?, actions_completed = ?, actions_failed = ?, actions_skipped = ?,
			failures = ?, duration_ms = ?
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		nullableTime(exec.StartedAt),
		nullableTime(exec.CompletedAt),
		string(exec.Status),
		exec.ActionsTotal,
		exec.ActionsCompleted,
		exec.ActionsFailed,
		exec.ActionsSkipped,
		failuresJSON,
		exec.DurationMS,
		exec.ID,
	)
	if err != nil {
		return fmt.Errorf("updating execution: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrExecutionNotFound
	}
	return nil
}

// GetExecution retrieves an execution by ID.
func (r *SQLiteRepository) GetExecution(ctx context.Context, id string) (*SceneExecution, error) {
	query := `
		SELECT id, scene_id, triggered_at, started_at, completed_at,
			trigger_type, trigger_source, status,
			actions_total, actions_completed, actions_failed, actions_skipped,
			failures, duration_ms
		FROM scene_executions
		WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	exec, err := scanExecution(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrExecutionNotFound
		}
		return nil, fmt.Errorf("querying execution: %w", err)
	}
	return exec, nil
}

// ListExecutions retrieves recent executions for a scene.
func (r *SQLiteRepository) ListExecutions(ctx context.Context, sceneID string, limit int) ([]SceneExecution, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT id, scene_id, triggered_at, started_at, completed_at,
			trigger_type, trigger_source, status,
			actions_total, actions_completed, actions_failed, actions_skipped,
			failures, duration_ms
		FROM scene_executions
		WHERE scene_id = ?
		ORDER BY triggered_at DESC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, sceneID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying executions: %w", err)
	}
	defer rows.Close()

	var executions []SceneExecution
	for rows.Next() {
		exec, scanErr := scanExecutionFromRows(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning execution: %w", scanErr)
		}
		executions = append(executions, *exec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating executions: %w", err)
	}
	return executions, nil
}

// queryScenes executes a query and returns a slice of scenes.
func (r *SQLiteRepository) queryScenes(ctx context.Context, query string, args ...any) ([]Scene, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying scenes: %w", err)
	}
	defer rows.Close()

	var scenes []Scene
	for rows.Next() {
		scene, scanErr := scanSceneFromRows(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning scene: %w", scanErr)
		}
		scenes = append(scenes, *scene)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating scenes: %w", err)
	}
	return scenes, nil
}

// ─── Row Scanning Helpers ───────────────────────────────────────────────────

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanScene scans a single sql.Row into a Scene.
func scanScene(row *sql.Row) (*Scene, error) {
	return scanSceneRow(row)
}

// scanSceneFromRows scans a sql.Rows result into a Scene.
func scanSceneFromRows(rows *sql.Rows) (*Scene, error) {
	return scanSceneRow(rows)
}

func scanSceneRow(scanner rowScanner) (*Scene, error) {
	var s Scene
	var description, roomID, areaID, icon, colour, category sql.NullString
	var actionsJSON string
	var enabled int
	var createdAt, updatedAt string

	err := scanner.Scan(
		&s.ID,
		&s.Name,
		&s.Slug,
		&description,
		&roomID,
		&areaID,
		&enabled,
		&s.Priority,
		&icon,
		&colour,
		&category,
		&actionsJSON,
		&s.SortOrder,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse nullable strings
	if description.Valid {
		s.Description = &description.String
	}
	if roomID.Valid {
		s.RoomID = &roomID.String
	}
	if areaID.Valid {
		s.AreaID = &areaID.String
	}
	if icon.Valid {
		s.Icon = &icon.String
	}
	if colour.Valid {
		s.Colour = &colour.String
	}
	if category.Valid {
		s.Category = Category(category.String)
	}

	s.Enabled = enabled != 0

	// Parse timestamps (stored as RFC3339 by SQLite default expressions)
	if t, parseErr := time.Parse(time.RFC3339, createdAt); parseErr == nil {
		s.CreatedAt = t
	}
	if t, parseErr := time.Parse(time.RFC3339, updatedAt); parseErr == nil {
		s.UpdatedAt = t
	}

	// Unmarshal actions JSON
	if actionsJSON != "" && actionsJSON != "[]" {
		if jsonErr := json.Unmarshal([]byte(actionsJSON), &s.Actions); jsonErr != nil {
			return nil, fmt.Errorf("unmarshalling actions: %w", jsonErr)
		}
	}
	if s.Actions == nil {
		s.Actions = []SceneAction{}
	}

	return &s, nil
}

// scanExecution scans a single sql.Row into a SceneExecution.
func scanExecution(row *sql.Row) (*SceneExecution, error) {
	return scanExecutionRow(row)
}

// scanExecutionFromRows scans a sql.Rows result into a SceneExecution.
func scanExecutionFromRows(rows *sql.Rows) (*SceneExecution, error) {
	return scanExecutionRow(rows)
}

func scanExecutionRow(scanner rowScanner) (*SceneExecution, error) {
	var e SceneExecution
	var triggeredAt string
	var startedAt, completedAt, triggerSource, failuresJSON sql.NullString
	var durationMS sql.NullInt64
	var status string

	err := scanner.Scan(
		&e.ID,
		&e.SceneID,
		&triggeredAt,
		&startedAt,
		&completedAt,
		&e.TriggerType,
		&triggerSource,
		&status,
		&e.ActionsTotal,
		&e.ActionsCompleted,
		&e.ActionsFailed,
		&e.ActionsSkipped,
		&failuresJSON,
		&durationMS,
	)
	if err != nil {
		return nil, err
	}

	e.Status = ExecutionStatus(status)
	if t, parseErr := time.Parse(time.RFC3339, triggeredAt); parseErr == nil {
		e.TriggeredAt = t
	}

	if startedAt.Valid {
		if t, parseErr := time.Parse(time.RFC3339, startedAt.String); parseErr == nil {
			e.StartedAt = &t
		}
	}
	if completedAt.Valid {
		if t, parseErr := time.Parse(time.RFC3339, completedAt.String); parseErr == nil {
			e.CompletedAt = &t
		}
	}
	if triggerSource.Valid {
		e.TriggerSource = &triggerSource.String
	}
	if durationMS.Valid {
		d := int(durationMS.Int64)
		e.DurationMS = &d
	}

	// Unmarshal failures JSON
	if failuresJSON.Valid && failuresJSON.String != "" && failuresJSON.String != "null" {
		if jsonErr := json.Unmarshal([]byte(failuresJSON.String), &e.Failures); jsonErr != nil {
			return nil, fmt.Errorf("unmarshalling failures: %w", jsonErr)
		}
	}

	return &e, nil
}

// ─── SQL Helpers ────────────────────────────────────────────────────────────

func nullableString(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullableCategory(c Category) sql.NullString {
	if c == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: string(c), Valid: true}
}

func nullableTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.Format(time.RFC3339), Valid: true}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func marshalFailures(failures []ActionFailure) (sql.NullString, error) {
	if len(failures) == 0 {
		return sql.NullString{}, nil
	}
	data, err := json.Marshal(failures)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(data), Valid: true}, nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "unique constraint")
}
