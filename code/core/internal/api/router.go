package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nerrad567/gray-logic-core/internal/panel"
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

	// Wall panel UI (Flutter web build, embedded via go:embed)
	r.Handle("/panel/*", http.StripPrefix("/panel", panel.Handler()))
	r.Handle("/panel", http.RedirectHandler("/panel/", http.StatusMovedPermanently))

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Health check (no auth required)
		r.Get("/health", s.handleHealth)

		// Auth endpoints (no auth required)
		r.Post("/auth/login", s.handleLogin)

		// System metrics (no auth required for basic monitoring)
		r.Get("/metrics", s.handleMetrics)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)

			// WS ticket requires authentication - user must be logged in to request a ticket
			r.Post("/auth/ws-ticket", s.handleWSTicket)

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

			// Area endpoints
			r.Route("/areas", func(r chi.Router) {
				r.Get("/", s.handleListAreas)
				r.Post("/", s.handleCreateArea)
				r.Get("/{id}", s.handleGetArea)
			})

			// Room endpoints
			r.Route("/rooms", func(r chi.Router) {
				r.Get("/", s.handleListRooms)
				r.Post("/", s.handleCreateRoom)
				r.Get("/{id}", s.handleGetRoom)
			})

			// Scene endpoints
			r.Route("/scenes", func(r chi.Router) {
				r.Get("/", s.handleListScenes)
				r.Post("/", s.handleCreateScene)

				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", s.handleGetScene)
					r.Patch("/", s.handleUpdateScene)
					r.Delete("/", s.handleDeleteScene)
					r.Post("/activate", s.handleActivateScene)
					r.Get("/executions", s.handleListSceneExecutions)
				})
			})

			// Commissioning endpoints (ETS import)
			r.Route("/commissioning", func(r chi.Router) {
				r.Route("/ets", func(r chi.Router) {
					r.Post("/parse", s.handleETSParse)
					r.Post("/import", s.handleETSImport)
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
