package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/nerrad567/gray-logic-core/internal/auth"
)

// Auth constants.
const (
	// ticketTTL is how long a WebSocket ticket is valid.
	ticketTTL = 60 * time.Second
)

// loginRequest is the request body for POST /auth/login.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginResponse is the response body for POST /auth/login.
type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// refreshRequest is the request body for POST /auth/refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// refreshResponse is the response body for POST /auth/refresh.
type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// changePasswordRequest is the request body for POST /auth/change-password.
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ticketStore holds pending WebSocket authentication tickets.
// Tickets are single-use and expire after ticketTTL.
type ticketStore struct {
	tickets map[string]ticketEntry
	mu      sync.Mutex
}

type ticketEntry struct {
	userID    string // empty for panel tickets
	role      auth.Role
	panelID   string // non-empty for panel tickets
	expiresAt time.Time
}

var wsTickets = &ticketStore{
	tickets: make(map[string]ticketEntry),
}

// handleLogin authenticates a user and returns JWT access + refresh tokens.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) { //nolint:gocognit // auth login: credential check + token generation pipeline
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeBadRequest(w, "username and password are required")
		return
	}

	if s.userRepo == nil {
		writeInternalError(w, "auth not configured")
		return
	}

	// Look up user by username
	user, err := s.userRepo.GetByUsername(r.Context(), req.Username)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			writeUnauthorized(w, "invalid credentials")
			return
		}
		s.logger.Error("login: user lookup failed", "error", err)
		writeInternalError(w, "authentication failed")
		return
	}

	// Check account is active
	if !user.IsActive {
		writeUnauthorized(w, "account is inactive")
		return
	}

	// Verify password
	ok, err := auth.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil {
		s.logger.Error("login: password verification failed", "error", err)
		writeInternalError(w, "authentication failed")
		return
	}
	if !ok {
		s.auditLog("login_failed", "user", user.ID, user.ID, map[string]any{
			"username":   req.Username,
			"reason":     "invalid_password",
			"user_agent": r.UserAgent(),
		})
		writeUnauthorized(w, "invalid credentials")
		return
	}

	// Generate access token
	ttl := s.secCfg.JWT.AccessTokenTTL
	if ttl == 0 {
		ttl = 15 //nolint:mnd // default 15-minute access token TTL
	}

	accessToken, err := auth.GenerateAccessToken(user, s.secCfg.JWT.Secret, ttl)
	if err != nil {
		s.logger.Error("login: access token generation failed", "error", err)
		writeInternalError(w, "authentication failed")
		return
	}

	// Generate and store refresh token
	rawRefresh, err := auth.GenerateRefreshToken()
	if err != nil {
		s.logger.Error("login: refresh token generation failed", "error", err)
		writeInternalError(w, "authentication failed")
		return
	}

	refreshTTL := s.secCfg.JWT.RefreshTokenTTL
	if refreshTTL == 0 {
		refreshTTL = 10080 //nolint:mnd // default 7 days in minutes
	}

	rt := &auth.RefreshToken{
		UserID:     user.ID,
		TokenHash:  auth.HashToken(rawRefresh),
		DeviceInfo: r.UserAgent(),
		ExpiresAt:  time.Now().Add(time.Duration(refreshTTL) * time.Minute),
	}

	if err := s.tokenRepo.Create(r.Context(), rt); err != nil {
		s.logger.Error("login: refresh token storage failed", "error", err)
		writeInternalError(w, "authentication failed")
		return
	}

	s.logger.Info("user logged in", "user_id", user.ID, "username", user.Username, "role", user.Role)
	s.auditLog("login", "user", user.ID, user.ID, map[string]any{
		"username":   user.Username,
		"role":       user.Role,
		"user_agent": r.UserAgent(),
	})

	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    ttl * 60, //nolint:mnd // convert minutes to seconds
	})
}

// handleRefresh rotates a refresh token and issues new access + refresh tokens.
// Implements theft detection: reuse of a consumed token revokes the entire family.
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) { //nolint:gocognit,gocyclo // refresh: token validation + rotation + theft detection pipeline
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if req.RefreshToken == "" {
		writeBadRequest(w, "refresh_token is required")
		return
	}

	if s.tokenRepo == nil || s.userRepo == nil {
		writeInternalError(w, "auth not configured")
		return
	}

	// Hash the incoming token to find it in the database
	tokenHash := auth.HashToken(req.RefreshToken)

	storedToken, err := s.tokenRepo.GetByTokenHash(r.Context(), tokenHash)
	if err != nil {
		writeUnauthorized(w, "invalid refresh token")
		return
	}

	// Check if token was already revoked (theft detection)
	if storedToken.Revoked {
		// Token reuse detected! Revoke the entire family.
		s.logger.Warn("refresh token reuse detected — revoking family",
			"token_id", storedToken.ID,
			"family_id", storedToken.FamilyID,
			"user_id", storedToken.UserID,
		)
		if err := s.tokenRepo.RevokeFamily(r.Context(), storedToken.FamilyID); err != nil { //nolint:govet // shadow: err re-declared in nested scope
			s.logger.Error("failed to revoke token family", "error", err)
		}
		s.auditLog("token_reuse", "session", storedToken.FamilyID, storedToken.UserID, map[string]any{
			"token_id":   storedToken.ID,
			"user_agent": r.UserAgent(),
		})
		writeUnauthorized(w, "token reuse detected — all sessions revoked")
		return
	}

	// Check expiry
	if time.Now().After(storedToken.ExpiresAt) {
		writeUnauthorized(w, "refresh token expired")
		return
	}

	// Revoke the current token (it's been consumed)
	if err := s.tokenRepo.Revoke(r.Context(), storedToken.ID); err != nil { //nolint:govet // shadow: err re-declared in nested scope
		s.logger.Error("failed to revoke consumed refresh token", "error", err)
		writeInternalError(w, "token refresh failed")
		return
	}

	// Look up user to generate new access token
	user, err := s.userRepo.GetByID(r.Context(), storedToken.UserID)
	if err != nil {
		writeUnauthorized(w, "user not found")
		return
	}

	if !user.IsActive {
		writeUnauthorized(w, "account is inactive")
		return
	}

	// Generate new access token
	ttl := s.secCfg.JWT.AccessTokenTTL
	if ttl == 0 {
		ttl = 15 //nolint:mnd // default 15-minute access token TTL
	}

	accessToken, err := auth.GenerateAccessToken(user, s.secCfg.JWT.Secret, ttl)
	if err != nil {
		s.logger.Error("refresh: access token generation failed", "error", err)
		writeInternalError(w, "token refresh failed")
		return
	}

	// Generate new refresh token in the same family
	rawRefresh, err := auth.GenerateRefreshToken()
	if err != nil {
		s.logger.Error("refresh: refresh token generation failed", "error", err)
		writeInternalError(w, "token refresh failed")
		return
	}

	refreshTTL := s.secCfg.JWT.RefreshTokenTTL
	if refreshTTL == 0 {
		refreshTTL = 10080 //nolint:mnd // default 7 days in minutes
	}

	newRT := &auth.RefreshToken{
		UserID:     user.ID,
		FamilyID:   storedToken.FamilyID, // same family for theft detection
		TokenHash:  auth.HashToken(rawRefresh),
		DeviceInfo: r.UserAgent(),
		ExpiresAt:  time.Now().Add(time.Duration(refreshTTL) * time.Minute),
	}

	if err := s.tokenRepo.Create(r.Context(), newRT); err != nil {
		s.logger.Error("refresh: new token storage failed", "error", err)
		writeInternalError(w, "token refresh failed")
		return
	}

	writeJSON(w, http.StatusOK, refreshResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    ttl * 60, //nolint:mnd // convert minutes to seconds
	})
}

// handleLogout revokes the refresh token family for the current session.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if req.RefreshToken == "" {
		// No refresh token provided — just acknowledge logout
		writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
		return
	}

	if s.tokenRepo == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
		return
	}

	// Find and revoke the token family
	tokenHash := auth.HashToken(req.RefreshToken)
	storedToken, err := s.tokenRepo.GetByTokenHash(r.Context(), tokenHash)
	if err == nil {
		if err := s.tokenRepo.RevokeFamily(r.Context(), storedToken.FamilyID); err != nil { //nolint:govet // shadow: err re-declared in nested scope
			s.logger.Error("logout: failed to revoke token family", "error", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

// handleChangePassword verifies the current password and sets a new one.
// All existing sessions are revoked after a password change.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if claims == nil {
		writeUnauthorized(w, "authentication required")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeBadRequest(w, "current_password and new_password are required")
		return
	}

	if len(req.NewPassword) < 8 { //nolint:mnd // minimum password length
		writeBadRequest(w, "new password must be at least 8 characters")
		return
	}

	if s.userRepo == nil || s.tokenRepo == nil {
		writeInternalError(w, "auth not configured")
		return
	}

	// Get user
	user, err := s.userRepo.GetByID(r.Context(), claims.Subject)
	if err != nil {
		writeInternalError(w, "failed to retrieve user")
		return
	}

	// Verify current password
	ok, err := auth.VerifyPassword(req.CurrentPassword, user.PasswordHash)
	if err != nil {
		s.logger.Error("change-password: verification failed", "error", err)
		writeInternalError(w, "password change failed")
		return
	}
	if !ok {
		writeUnauthorized(w, "current password is incorrect")
		return
	}

	// Hash new password
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		s.logger.Error("change-password: hashing failed", "error", err)
		writeInternalError(w, "password change failed")
		return
	}

	// Update password
	if err := s.userRepo.UpdatePassword(r.Context(), user.ID, newHash); err != nil {
		s.logger.Error("change-password: update failed", "error", err)
		writeInternalError(w, "password change failed")
		return
	}

	// Revoke all existing sessions (force re-login everywhere)
	if err := s.tokenRepo.RevokeAllForUser(r.Context(), user.ID); err != nil {
		s.logger.Error("change-password: session revocation failed", "error", err)
	}

	s.logger.Info("password changed", "user_id", user.ID)
	s.auditLog("password_change", "user", user.ID, user.ID, nil)

	writeJSON(w, http.StatusOK, map[string]string{"status": "password_changed"})
}

// handleWSTicket generates a single-use WebSocket authentication ticket.
// The ticket carries the caller's identity (user claims or panel context)
// so the WebSocket connection inherits the same auth context.
func (s *Server) handleWSTicket(w http.ResponseWriter, r *http.Request) {
	ticket := generateTicket()

	entry := ticketEntry{
		expiresAt: time.Now().Add(ticketTTL),
	}

	// Carry auth identity onto the ticket
	if claims := claimsFromContext(r.Context()); claims != nil {
		entry.userID = claims.Subject
		entry.role = claims.Role
	} else if pc := panelFromContext(r.Context()); pc != nil {
		entry.panelID = pc.PanelID
		entry.role = auth.RolePanel
	}

	wsTickets.mu.Lock()
	wsTickets.tickets[ticket] = entry
	wsTickets.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"ticket":     ticket,
		"expires_in": int(ticketTTL.Seconds()),
	})
}

// validateTicket checks if a ticket is valid and consumes it (single-use).
// Returns the ticket entry on success for identity propagation.
func validateTicket(ticket string) (ticketEntry, bool) { //nolint:unparam // ticketEntry carries identity to WebSocket
	wsTickets.mu.Lock()
	defer wsTickets.mu.Unlock()

	entry, ok := wsTickets.tickets[ticket]
	if !ok {
		return ticketEntry{}, false
	}

	// Remove ticket (single-use)
	delete(wsTickets.tickets, ticket)

	// Check expiry
	if time.Now().After(entry.expiresAt) {
		return ticketEntry{}, false
	}

	return entry, true
}

// ticketBytes is the number of random bytes used for WebSocket tickets.
const ticketBytes = 32

// generateTicket creates a cryptographically random ticket string.
func generateTicket() string {
	b := make([]byte, ticketBytes)
	//nolint:errcheck // crypto/rand.Read always returns len(b) on supported platforms
	rand.Read(b)
	return hex.EncodeToString(b)
}

// cleanExpiredTickets removes expired tickets from the store.
func cleanExpiredTickets() {
	wsTickets.mu.Lock()
	defer wsTickets.mu.Unlock()

	now := time.Now()
	for ticket, entry := range wsTickets.tickets {
		if now.After(entry.expiresAt) {
			delete(wsTickets.tickets, ticket)
		}
	}
}

// cleanTicketsLoop runs cleanExpiredTickets periodically until the context is cancelled.
func (s *Server) cleanTicketsLoop(ctx context.Context) {
	ticker := time.NewTicker(ticketTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanExpiredTickets()
		}
	}
}
