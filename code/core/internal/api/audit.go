package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/nerrad567/gray-logic-core/internal/audit"
)

// auditChanSize is the buffer size for the async audit log channel.
// Entries beyond this are dropped (best-effort) to avoid back-pressure on requests.
const auditChanSize = 256

// auditLog enqueues an audit log entry for asynchronous write (best-effort).
// If the channel is full the entry is dropped and a warning is logged.
func (s *Server) auditLog(action, entityType, entityID, userID string, details map[string]any) {
	if s.auditRepo == nil || s.auditCh == nil {
		return
	}

	entry := &audit.AuditLog{
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		UserID:     userID,
		Source:     "api",
		Details:    details,
	}

	select {
	case s.auditCh <- entry:
	default:
		s.logger.Warn("audit log channel full — dropping entry",
			"action", action,
			"entity_type", entityType,
		)
	}
}

// drainAuditLog reads entries from the audit channel and writes them serially.
// This avoids unbounded goroutine creation and is kinder to SQLite's serial write model.
// It runs until the context is cancelled, then drains remaining entries.
func (s *Server) drainAuditLog(ctx context.Context) {
	for {
		select {
		case entry := <-s.auditCh:
			if err := s.auditRepo.Create(context.Background(), entry); err != nil {
				s.logger.Error("audit log write failed",
					"action", entry.Action,
					"entity_type", entry.EntityType,
					"error", err,
				)
			}
		case <-ctx.Done():
			// Drain remaining entries before exiting
			for {
				select {
				case entry := <-s.auditCh:
					if err := s.auditRepo.Create(context.Background(), entry); err != nil {
						s.logger.Error("audit log write failed during shutdown",
							"action", entry.Action,
							"error", err,
						)
					}
				default:
					return
				}
			}
		}
	}
}

// handleListAuditLogs returns paginated audit log entries with optional filters.
//
// Query parameters:
//   - action: filter by action type (create, update, delete, command, login)
//   - entity_type: filter by entity type (device, scene, site, etc.)
//   - entity_id: filter by specific entity ID
//   - limit: max results (default 50, max 200)
//   - offset: pagination offset
func (s *Server) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	if s.auditRepo == nil {
		writeInternalError(w, "audit logging not configured")
		return
	}

	q := r.URL.Query()
	filter := audit.Filter{
		Action:     q.Get("action"),
		EntityType: q.Get("entity_type"),
		EntityID:   q.Get("entity_id"),
	}

	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Offset = n
		}
	}

	result, err := s.auditRepo.List(r.Context(), filter)
	if err != nil {
		s.logger.Error("failed to list audit logs", "error", err)
		writeInternalError(w, "failed to list audit logs")
		return
	}

	writeJSON(w, http.StatusOK, result)
}
