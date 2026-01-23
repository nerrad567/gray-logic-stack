package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// buildRouter creates the HTTP router with all routes and middleware.
func (s *Server) buildRouter() http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(s.requestIDMiddleware)
	r.Use(s.loggingMiddleware)
	r.Use(s.recoveryMiddleware)
	r.Use(s.corsMiddleware)
	r.Use(s.bodySizeLimitMiddleware)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Health check (no auth required)
		r.Get("/health", s.handleHealth)

		// Auth endpoints (no auth required)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/ws-ticket", s.handleWSTicket)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)

			// Device endpoints
			r.Route("/devices", func(r chi.Router) {
				r.Get("/", s.handleListDevices)
				r.Post("/", s.handleCreateDevice)
				r.Get("/stats", s.handleDeviceStats)

				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", s.handleGetDevice)
					r.Patch("/", s.handleUpdateDevice)
					r.Delete("/", s.handleDeleteDevice)
					r.Get("/state", s.handleGetDeviceState)
					r.Put("/state", s.handleSetDeviceState)
				})
			})

			// WebSocket (auth via ticket, validated in handler)
			r.Get("/ws", s.handleWebSocket)
		})
	})

	return r
}

// handleHealth returns the server health status.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": s.version,
	})
}
