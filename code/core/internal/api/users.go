package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nerrad567/gray-logic-core/internal/auth"
)

// ─── Request/Response Types ────────────────────────────────────────

type createUserRequest struct {
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email,omitempty"`
	Password    string    `json:"password"`
	Role        auth.Role `json:"role"`
}

type updateUserRequest struct {
	DisplayName *string    `json:"display_name,omitempty"`
	Email       *string    `json:"email,omitempty"`
	Role        *auth.Role `json:"role,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
}

type setRoomAccessRequest struct {
	Rooms []auth.RoomAccessGrant `json:"rooms"`
}

// ─── Handlers ──────────────────────────────────────────────────────

// handleListUsers returns all user accounts.
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.userRepo.List(r.Context())
	if err != nil {
		s.logger.Error("list users failed", "error", err)
		writeInternalError(w, "failed to list users")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users": users,
		"count": len(users),
	})
}

// handleCreateUser creates a new user account.
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) { //nolint:gocognit // user creation: validation + permission checks + password hashing pipeline
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if req.Username == "" || req.Password == "" || req.DisplayName == "" {
		writeBadRequest(w, "username, password, and display_name are required")
		return
	}

	if len(req.Password) < 8 { //nolint:mnd // minimum password length
		writeBadRequest(w, "password must be at least 8 characters")
		return
	}

	if req.Role == "" {
		req.Role = auth.RoleUser
	}

	if !auth.IsValidUserRole(req.Role) {
		writeBadRequest(w, "invalid role: must be user, admin, or owner")
		return
	}

	// Only owners can create owner accounts
	claims := claimsFromContext(r.Context())
	if req.Role == auth.RoleOwner && !auth.HasPermission(claims.Role, auth.PermUserManageAll) {
		writeForbidden(w, "only owners can create owner accounts")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("hash password failed", "error", err)
		writeInternalError(w, "failed to create user")
		return
	}

	user := &auth.User{
		Username:     req.Username,
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         req.Role,
		IsActive:     true,
		CreatedBy:    claims.Subject,
	}

	if err := s.userRepo.Create(r.Context(), user); err != nil {
		if errors.Is(err, auth.ErrUsernameExists) {
			writeConflict(w, "username already exists")
			return
		}
		s.logger.Error("create user failed", "error", err)
		writeInternalError(w, "failed to create user")
		return
	}

	s.logger.Info("user created", "user_id", user.ID, "username", user.Username, "role", user.Role, "created_by", claims.Subject)
	s.auditLog("create", "user", user.ID, claims.Subject, map[string]any{
		"username": user.Username,
		"role":     user.Role,
	})

	writeJSON(w, http.StatusCreated, user)
}

// handleGetUser returns a single user by ID.
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := s.userRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			writeNotFound(w, "user not found")
			return
		}
		s.logger.Error("get user failed", "error", err)
		writeInternalError(w, "failed to get user")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// handleUpdateUser modifies a user's mutable fields.
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) { //nolint:gocognit,gocyclo // user update: field patching + self-protection + role escalation guards
	id := chi.URLParam(r, "id")
	claims := claimsFromContext(r.Context())

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	user, err := s.userRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			writeNotFound(w, "user not found")
			return
		}
		s.logger.Error("get user for update failed", "error", err)
		writeInternalError(w, "failed to update user")
		return
	}

	// Self-protection: cannot deactivate yourself
	if req.IsActive != nil && !*req.IsActive && id == claims.Subject {
		writeForbidden(w, "cannot deactivate your own account")
		return
	}

	// Self-protection: cannot demote yourself
	if req.Role != nil && id == claims.Subject && *req.Role != claims.Role {
		writeForbidden(w, "cannot change your own role")
		return
	}

	// Only owners can modify owner accounts or promote to owner
	if user.Role == auth.RoleOwner && !auth.HasPermission(claims.Role, auth.PermUserManageAll) {
		writeForbidden(w, "only owners can modify owner accounts")
		return
	}
	if req.Role != nil && *req.Role == auth.RoleOwner && !auth.HasPermission(claims.Role, auth.PermUserManageAll) {
		writeForbidden(w, "only owners can promote users to owner")
		return
	}

	// Apply patches
	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := s.userRepo.Update(r.Context(), user); err != nil {
		s.logger.Error("update user failed", "error", err)
		writeInternalError(w, "failed to update user")
		return
	}

	s.logger.Info("user updated", "user_id", id, "updated_by", claims.Subject)
	s.auditLog("update", "user", id, claims.Subject, nil)

	writeJSON(w, http.StatusOK, user)
}

// handleDeleteUser removes a user account.
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims := claimsFromContext(r.Context())

	// Cannot delete yourself
	if id == claims.Subject {
		writeForbidden(w, "cannot delete your own account")
		return
	}

	// Check the target user exists and check permissions
	user, err := s.userRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			writeNotFound(w, "user not found")
			return
		}
		s.logger.Error("get user for delete failed", "error", err)
		writeInternalError(w, "failed to delete user")
		return
	}

	// Only owners can delete owner accounts
	if user.Role == auth.RoleOwner && !auth.HasPermission(claims.Role, auth.PermUserManageAll) {
		writeForbidden(w, "only owners can delete owner accounts")
		return
	}

	if err := s.userRepo.Delete(r.Context(), id); err != nil {
		s.logger.Error("delete user failed", "error", err)
		writeInternalError(w, "failed to delete user")
		return
	}

	// Revoke all sessions
	if s.tokenRepo != nil {
		if err := s.tokenRepo.RevokeAllForUser(r.Context(), id); err != nil {
			s.logger.Error("revoke sessions after delete failed", "error", err)
		}
	}

	s.logger.Info("user deleted", "user_id", id, "deleted_by", claims.Subject)
	s.auditLog("delete", "user", id, claims.Subject, map[string]any{
		"username": user.Username,
	})

	w.WriteHeader(http.StatusNoContent)
}

// handleListUserSessions returns active refresh tokens for a user.
func (s *Server) handleListUserSessions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	tokens, err := s.tokenRepo.ListActiveByUser(r.Context(), id)
	if err != nil {
		s.logger.Error("list user sessions failed", "error", err)
		writeInternalError(w, "failed to list sessions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessions": tokens,
		"count":    len(tokens),
	})
}

// handleRevokeUserSessions revokes all refresh tokens for a user.
func (s *Server) handleRevokeUserSessions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims := claimsFromContext(r.Context())

	if err := s.tokenRepo.RevokeAllForUser(r.Context(), id); err != nil {
		s.logger.Error("revoke user sessions failed", "error", err)
		writeInternalError(w, "failed to revoke sessions")
		return
	}

	s.logger.Info("user sessions revoked", "user_id", id, "revoked_by", claims.Subject)
	s.auditLog("revoke_sessions", "user", id, claims.Subject, nil)

	writeJSON(w, http.StatusOK, map[string]string{"status": "sessions_revoked"})
}

// handleGetUserRooms returns a user's room access assignments.
func (s *Server) handleGetUserRooms(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	access, err := s.roomAccessRepo.GetRoomAccess(r.Context(), id)
	if err != nil {
		s.logger.Error("get user rooms failed", "error", err)
		writeInternalError(w, "failed to get room access")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"rooms": access,
		"count": len(access),
	})
}

// handleSetUserRooms replaces all room access grants for a user.
// Empty array = revoke all room access (user becomes locked out).
func (s *Server) handleSetUserRooms(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims := claimsFromContext(r.Context())

	var req setRoomAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	// Verify user exists
	if _, err := s.userRepo.GetByID(r.Context(), id); err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			writeNotFound(w, "user not found")
			return
		}
		s.logger.Error("get user for room access failed", "error", err)
		writeInternalError(w, "failed to set room access")
		return
	}

	if err := s.roomAccessRepo.SetRoomAccess(r.Context(), id, req.Rooms, claims.Subject); err != nil {
		s.logger.Error("set user rooms failed", "error", err)
		writeInternalError(w, "failed to set room access")
		return
	}

	s.logger.Info("user room access updated", "user_id", id, "room_count", len(req.Rooms), "updated_by", claims.Subject)
	s.auditLog("update_room_access", "user", id, claims.Subject, map[string]any{
		"room_count": len(req.Rooms),
	})

	// Return the updated access
	access, err := s.roomAccessRepo.GetRoomAccess(r.Context(), id)
	if err != nil {
		s.logger.Error("get updated room access failed", "error", err)
		writeInternalError(w, "room access updated but failed to retrieve")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"rooms": access,
		"count": len(access),
	})
}
