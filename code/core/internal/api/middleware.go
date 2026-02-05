package api

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nerrad567/gray-logic-core/internal/auth"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// ctxKeyRequestID is the context key for the request ID.
	ctxKeyRequestID contextKey = "request_id"
	// ctxKeyClaims holds the authenticated user's JWT claims.
	ctxKeyClaims contextKey = "claims"
	// ctxKeyPanel holds the authenticated panel's context.
	ctxKeyPanel contextKey = "panel"
	// ctxKeyRoomScope holds the resolved room scope for room-scoped roles.
	ctxKeyRoomScope contextKey = "room_scope"
)

// PanelContext holds the identity of an authenticated panel device.
type PanelContext struct {
	PanelID string
	RoomIDs []string // rooms this panel can access
}

// requestIDMiddleware generates a unique request ID for each request.
// If the client sends an X-Request-ID header, it is used; otherwise one is generated.
func (s *Server) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// loggingMiddleware logs each HTTP request with method, path, status, and duration.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		s.logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", r.Context().Value(ctxKeyRequestID),
		)
	})
}

// recoveryMiddleware catches panics in handlers and returns a 500 response.
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				s.logger.Error("panic recovered in HTTP handler",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
					"request_id", r.Context().Value(ctxKeyRequestID),
				)
				writeInternalError(w, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware handles Cross-Origin Resource Sharing headers.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && s.isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", joinOrDefault(s.cfg.CORS.AllowedMethods, "GET, POST, PUT, PATCH, DELETE, OPTIONS"))
			w.Header().Set("Access-Control-Allow-Headers", joinOrDefault(s.cfg.CORS.AllowedHeaders, "Authorization, Content-Type, X-Request-ID, X-Panel-Token"))
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		// Handle preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// maxRequestBodySize is the maximum allowed request body size (1 MB).
const maxRequestBodySize = 1 << 20

// bodySizeLimitMiddleware limits the size of incoming request bodies to prevent
// denial-of-service attacks via oversized payloads.
func (s *Server) bodySizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		}
		next.ServeHTTP(w, r)
	})
}

// securityHeadersMiddleware applies baseline security headers.
func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")

		// CSP applies to API responses only; skip the panel UI routes.
		if !strings.HasPrefix(r.URL.Path, "/panel") {
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
		}

		if s.cfg.TLS.Enabled {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// Rate limiting configuration.
const (
	rateLimitWindow          = 15 * time.Minute
	rateLimitCleanupInterval = 15 * time.Minute
)

// rateLimiter enforces request limits per key within a time window.
type rateLimiter struct {
	attempts sync.Map // key: string, value: *attemptRecord
}

type attemptRecord struct {
	count       int
	windowStart time.Time
	mu          sync.Mutex
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{}
}

// allow checks and records a request, returning whether it is allowed.
func (rl *rateLimiter) allow(key string, limit int, window time.Duration, now time.Time) (bool, time.Duration) {
	entry, _ := rl.attempts.LoadOrStore(key, &attemptRecord{windowStart: now})
	record, ok := entry.(*attemptRecord)
	if !ok {
		record = &attemptRecord{windowStart: now}
		rl.attempts.Store(key, record)
	}

	record.mu.Lock()
	defer record.mu.Unlock()

	if now.Sub(record.windowStart) >= window {
		record.windowStart = now
		record.count = 0
	}

	if record.count >= limit {
		retryAfter := window - now.Sub(record.windowStart)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}

	record.count++
	return true, 0
}

// cleanupLoop periodically removes expired rate limit records.
func (rl *rateLimiter) cleanupLoop(ctx context.Context, window time.Duration) {
	ticker := time.NewTicker(rateLimitCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.cleanupExpired(window, time.Now().UTC())
		}
	}
}

func (rl *rateLimiter) cleanupExpired(window time.Duration, now time.Time) {
	rl.attempts.Range(func(key, value any) bool {
		record, ok := value.(*attemptRecord)
		if !ok {
			rl.attempts.Delete(key)
			return true
		}

		record.mu.Lock()
		expired := now.Sub(record.windowStart) >= window
		record.mu.Unlock()
		if expired {
			rl.attempts.Delete(key)
		}
		return true
	})
}

// rateLimitMiddleware enforces a fixed request limit per client IP.
func (s *Server) rateLimitMiddleware(limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.rateLimiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			key := rateLimitKey(r)
			allowed, retryAfter := s.rateLimiter.allow(key, limit, window, time.Now().UTC())
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(retryAfter)))
				writeTooManyRequests(w, "too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// authMiddleware validates authentication on protected routes.
// It supports two authentication methods:
//  1. Panel token: X-Panel-Token header → SHA-256 hash → lookup in panels table
//  2. JWT bearer: Authorization: Bearer <token> → parse + validate signature/expiry
//
// On success, it injects either PanelContext or CustomClaims into the request context.
func (s *Server) authMiddleware(next http.Handler) http.Handler { //nolint:gocognit,gocyclo // dual-path auth: panel token + JWT bearer validation
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Path 1: Panel token authentication
		if panelToken := r.Header.Get("X-Panel-Token"); panelToken != "" {
			if s.panelRepo == nil {
				writeUnauthorized(w, "panel auth not configured")
				return
			}

			tokenHash := auth.HashToken(panelToken)
			panel, err := s.panelRepo.GetByTokenHash(r.Context(), tokenHash)
			if err != nil {
				writeUnauthorized(w, "invalid panel token")
				return
			}
			if !panel.IsActive {
				writeUnauthorized(w, "panel is inactive")
				return
			}

			// Update last seen (best-effort, don't block the request)
			go func() {
				//nolint:errcheck // best-effort last-seen update
				s.panelRepo.UpdateLastSeen(context.Background(), panel.ID)
			}()

			// Resolve panel's room assignments
			roomIDs, err := s.panelRepo.GetRoomIDs(r.Context(), panel.ID)
			if err != nil {
				s.logger.Error("failed to resolve panel rooms", "panel_id", panel.ID, "error", err)
				writeInternalError(w, "failed to resolve panel access")
				return
			}

			pc := &PanelContext{
				PanelID: panel.ID,
				RoomIDs: roomIDs,
			}
			ctx := context.WithValue(r.Context(), ctxKeyPanel, pc)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Path 2: JWT bearer token authentication
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeUnauthorized(w, "authentication required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2) //nolint:mnd // "Bearer <token>"
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeUnauthorized(w, "invalid authorization header format")
			return
		}

		claims, err := auth.ParseToken(parts[1], s.secCfg.JWT.Secret)
		if err != nil {
			writeUnauthorized(w, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyClaims, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requirePermission returns middleware that checks the authenticated caller
// has the specified permission. For panels, it checks panelPermissions.
// For users, it checks the role-permission map.
func (s *Server) requirePermission(perm auth.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check panel permissions
			if pc := panelFromContext(r.Context()); pc != nil {
				if !auth.HasPanelPermission(perm) {
					writeForbidden(w, "panels do not have permission: "+string(perm))
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Check user permissions
			claims := claimsFromContext(r.Context())
			if claims == nil {
				writeUnauthorized(w, "authentication required")
				return
			}

			if !auth.HasPermission(claims.Role, perm) {
				s.auditLog("permission_denied", "system", "", claims.Subject, map[string]any{
					"permission": string(perm),
					"role":       string(claims.Role),
					"method":     r.Method,
					"path":       r.URL.Path,
				})
				writeForbidden(w, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// resolveRoomScopeMiddleware resolves the room scope for room-scoped roles (user)
// and injects it into the request context. Admin/owner get nil scope (unrestricted).
// Panels already have their room scope in PanelContext.
func (s *Server) resolveRoomScopeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Panels carry their room scope in PanelContext — nothing to resolve
		if panelFromContext(r.Context()) != nil {
			next.ServeHTTP(w, r)
			return
		}

		claims := claimsFromContext(r.Context())
		if claims == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Only resolve room scope for room-scoped roles (user)
		if !auth.IsRoomScoped(claims.Role) {
			// Admin/owner — unrestricted (nil scope)
			next.ServeHTTP(w, r)
			return
		}

		if s.roomAccessRepo == nil {
			writeForbidden(w, "room access not configured")
			return
		}

		scope, err := s.roomAccessRepo.ResolveRoomScope(r.Context(), claims.Subject)
		if err != nil {
			s.logger.Error("failed to resolve room scope", "user_id", claims.Subject, "error", err)
			writeInternalError(w, "failed to resolve room access")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyRoomScope, scope)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireUsersOnly rejects panel requests — only user JWT auth is accepted.
func requireUsersOnly(next http.Handler) http.Handler { //nolint:unused // reserved for future route protection
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if panelFromContext(r.Context()) != nil {
			writeForbidden(w, "this endpoint is not available to panels")
			return
		}
		if claimsFromContext(r.Context()) == nil {
			writeUnauthorized(w, "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// claimsFromContext extracts CustomClaims from the request context.
// Returns nil if no user claims are present (e.g. panel request or unauthenticated).
func claimsFromContext(ctx context.Context) *auth.CustomClaims {
	claims, _ := ctx.Value(ctxKeyClaims).(*auth.CustomClaims) //nolint:errcheck // type assertion returns nil on miss
	return claims
}

// panelFromContext extracts PanelContext from the request context.
// Returns nil if no panel context is present (e.g. user request or unauthenticated).
func panelFromContext(ctx context.Context) *PanelContext {
	pc, _ := ctx.Value(ctxKeyPanel).(*PanelContext) //nolint:errcheck // type assertion returns nil on miss
	return pc
}

// roomScopeFromContext extracts the resolved RoomScope from the request context.
// Returns nil for admin/owner (unrestricted access).
func roomScopeFromContext(ctx context.Context) *auth.RoomScope {
	scope, _ := ctx.Value(ctxKeyRoomScope).(*auth.RoomScope) //nolint:errcheck // type assertion returns nil on miss
	return scope
}

// requestRoomScope resolves the effective room scope for a request.
// It returns nil for unrestricted access (admin/owner).
// Panels derive scope from their PanelContext.
func requestRoomScope(ctx context.Context) *auth.RoomScope {
	scope := roomScopeFromContext(ctx)
	if scope != nil {
		return scope
	}

	if pc := panelFromContext(ctx); pc != nil {
		return &auth.RoomScope{RoomIDs: pc.RoomIDs}
	}

	return nil
}

// derefString safely dereferences a string pointer.
func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// filterByRoomIDs filters a slice to items whose room ID is in the allowed set.
// Items without a room ID are excluded for room-scoped requests.
func filterByRoomIDs[T any](items []T, roomIDs []string, roomID func(T) *string) []T {
	if len(roomIDs) == 0 {
		return []T{}
	}

	roomSet := make(map[string]struct{}, len(roomIDs))
	for _, id := range roomIDs {
		roomSet[id] = struct{}{}
	}

	filtered := make([]T, 0, len(items))
	for _, item := range items {
		id := roomID(item)
		if id == nil {
			continue
		}
		if _, ok := roomSet[*id]; ok {
			filtered = append(filtered, item)
		}
	}

	if filtered == nil {
		return []T{}
	}
	return filtered
}

func handleScopedList[T any](w http.ResponseWriter, scope *auth.RoomScope, key string, internalError string, listFn func() ([]T, string, error)) {
	if scope != nil && len(scope.RoomIDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{key: []T{}, "count": 0})
		return
	}

	items, badRequest, err := listFn()
	if badRequest != "" {
		writeBadRequest(w, badRequest)
		return
	}
	if err != nil {
		writeInternalError(w, internalError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{key: items, "count": len(items)})
}

func rateLimitKey(r *http.Request) string {
	return r.URL.Path + "|" + clientIP(r)
}

func retryAfterSeconds(duration time.Duration) int {
	if duration <= 0 {
		return 1
	}
	seconds := int(duration.Seconds())
	if duration%time.Second != 0 {
		seconds++
	}
	if seconds < 1 {
		return 1
	}
	return seconds
}

// clientIP extracts the client IP from the TCP connection's RemoteAddr.
// X-Forwarded-For and X-Real-IP are intentionally ignored because they are
// trivially spoofable on a LAN (Gray Logic's deployment target) and would
// allow rate-limit bypass. If a trusted reverse proxy is added later,
// introduce a "trusted proxy" config to selectively honour forwarded headers.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

// isAllowedOrigin checks if the origin is in the allowed list.
// An empty list allows all origins (dev mode).
func (s *Server) isAllowedOrigin(origin string) bool {
	if len(s.cfg.CORS.AllowedOrigins) == 0 {
		return true
	}
	for _, allowed := range s.cfg.CORS.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (w *statusWriter) WriteHeader(status int) {
	if w.written {
		return // Already sent to client; subsequent calls are no-ops
	}
	w.written = true
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	//nolint:wrapcheck // Passthrough: statusWriter is a transparent wrapper
	return w.ResponseWriter.Write(b)
}

// Hijack implements http.Hijacker, required for WebSocket upgrades.
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack() //nolint:wrapcheck // thin pass-through to underlying http.Hijacker
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

// requestIDBytes is the number of random bytes used for request IDs.
const requestIDBytes = 8

// generateRequestID creates a random hex request ID.
func generateRequestID() string {
	b := make([]byte, requestIDBytes)
	//nolint:errcheck // crypto/rand.Read always returns len(b) on supported platforms
	rand.Read(b)
	return hex.EncodeToString(b)
}

// joinOrDefault joins a string slice with ", " or returns the default if empty.
func joinOrDefault(values []string, defaultVal string) string {
	if len(values) == 0 {
		return defaultVal
	}
	result := values[0]
	for _, v := range values[1:] {
		result += ", " + v
	}
	return result
}
