// Package audit provides access to the audit_logs table for
// querying system activity history.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// AuditLog represents a single audit trail entry.
type AuditLog struct {
	ID         string         `json:"id"`
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id,omitempty"`
	UserID     string         `json:"user_id,omitempty"`
	Source     string         `json:"source"`
	Details    map[string]any `json:"details,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Filter controls which audit logs to return.
type Filter struct {
	Action     string // optional: filter by action (create, update, delete, command, login)
	EntityType string // optional: filter by entity type (device, scene, site, etc.)
	EntityID   string // optional: filter by specific entity ID
	Limit      int    // default 50, max 200
	Offset     int    // pagination offset
}

// ListResult contains the paginated audit log results.
type ListResult struct {
	Logs   []AuditLog `json:"logs"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

// Repository defines the interface for audit log queries.
type Repository interface {
	List(ctx context.Context, filter Filter) (*ListResult, error)
}

// SQLiteRepository reads audit logs from SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new audit log repository.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// List returns audit logs matching the filter, ordered by most recent first.
func (r *SQLiteRepository) List(ctx context.Context, filter Filter) (*ListResult, error) {
	// Clamp limit.
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	// Build WHERE clause dynamically.
	var conditions []string
	var args []any

	if filter.Action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, filter.Action)
	}
	if filter.EntityType != "" {
		conditions = append(conditions, "entity_type = ?")
		args = append(args, filter.EntityType)
	}
	if filter.EntityID != "" {
		conditions = append(conditions, "entity_id = ?")
		args = append(args, filter.EntityID)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count.
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", where)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("counting audit logs: %w", err)
	}

	// Get paginated results.
	query := fmt.Sprintf(
		"SELECT id, action, entity_type, entity_id, user_id, source, details, created_at FROM audit_logs %s ORDER BY created_at DESC LIMIT ? OFFSET ?",
		where,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		var entityID, userID, detailsJSON sql.NullString
		var createdAt string

		if err := rows.Scan(&log.ID, &log.Action, &log.EntityType,
			&entityID, &userID, &log.Source, &detailsJSON, &createdAt); err != nil {
			return nil, fmt.Errorf("scanning audit log: %w", err)
		}

		if entityID.Valid {
			log.EntityID = entityID.String
		}
		if userID.Valid {
			log.UserID = userID.String
		}
		if detailsJSON.Valid && detailsJSON.String != "" {
			var details map[string]any
			if json.Unmarshal([]byte(detailsJSON.String), &details) == nil {
				log.Details = details
			}
		}

		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			t, _ = time.Parse("2006-01-02T15:04:05Z", createdAt)
		}
		log.CreatedAt = t

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating audit logs: %w", err)
	}

	if logs == nil {
		logs = []AuditLog{}
	}

	return &ListResult{
		Logs:   logs,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}
