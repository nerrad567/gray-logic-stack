package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/nerrad567/gray-logic-core/internal/audit"
)

// auditLog writes an audit log entry asynchronously (best-effort).
// Audit write failures are logged but never block the request.
func (s *Server) auditLog(action, entityType, entityID, userID string, details map[string]any) {
	if s.auditRepo == nil {
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

	go func() {
		if err := s.auditRepo.Create(context.Background(), entry); err != nil {
			s.logger.Error("audit log write failed",
				"action", action,
				"entity_type", entityType,
				"error", err,
			)
		}
	}()
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
