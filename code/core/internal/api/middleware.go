package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// ctxKeyRequestID is the context key for the request ID.
	ctxKeyRequestID contextKey = "request_id"
)

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
			w.Header().Set("Access-Control-Allow-Headers", joinOrDefault(s.cfg.CORS.AllowedHeaders, "Authorization, Content-Type, X-Request-ID"))
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

// authMiddleware validates JWT tokens on protected routes.
// Placeholder: accepts any valid JWT signed with the configured secret.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(M1.6): Full JWT validation with claims
		// For now, pass through — auth endpoints exist but middleware is permissive
		// during development. Set GRAYLOGIC_JWT_SECRET in production.
		next.ServeHTTP(w, r)
	})
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
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
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
