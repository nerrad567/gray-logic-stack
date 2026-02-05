package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nerrad567/gray-logic-core/internal/auth"
	"github.com/nerrad567/gray-logic-core/internal/panel"
)

const (
	loginRateLimit   = 5
	refreshRateLimit = 10
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
	r.Use(s.securityHeadersMiddleware)

	// Wall panel UI (Flutter web build — filesystem in dev, embedded in prod)
	r.Handle("/panel/*", http.StripPrefix("/panel", panel.Handler(s.panelDir)))
	r.Handle("/panel", http.RedirectHandler("/panel/", http.StatusMovedPermanently))

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Health check (no auth required)
		r.Get("/health", s.handleHealth)

		// Auth endpoints (no auth required)
		r.With(s.rateLimitMiddleware(loginRateLimit, rateLimitWindow)).Post("/auth/login", s.handleLogin)
		r.With(s.rateLimitMiddleware(refreshRateLimit, rateLimitWindow)).Post("/auth/refresh", s.handleRefresh)

		// System metrics (no auth required for basic monitoring)
		r.Get("/metrics", s.handleMetrics)

		// Discovery data (passive KNX bus scan results)
		r.Get("/discovery", s.handleListDiscovery)

		// WebSocket (auth via ticket, validated in handler — not behind JWT middleware)
		r.Get("/ws", s.handleWebSocket)

		// ── Any authenticated (user JWT or panel token) ───────────
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)

			// Auth session management (users only)
			r.Post("/auth/ws-ticket", s.handleWSTicket)
			r.Post("/auth/logout", s.handleLogout)
			r.Post("/auth/change-password", s.handleChangePassword)

			// ── device:read — user (room-scoped), admin, owner, panels ──
			r.Group(func(r chi.Router) {
				r.Use(s.requirePermission(auth.PermDeviceRead))
				r.Use(s.resolveRoomScopeMiddleware)

				r.Get("/devices", s.handleListDevices)
				r.Get("/devices/stats", s.handleDeviceStats)
				r.Get("/devices/{id}", s.handleGetDevice)
				r.Get("/devices/{id}/state", s.handleGetDeviceState)
				r.Get("/devices/{id}/tags", s.handleGetDeviceTags)
				r.Get("/devices/{id}/history", s.handleGetDeviceHistory)
				r.Get("/devices/{id}/metrics", s.handleGetDeviceMetrics)
				r.Get("/devices/{id}/metrics/summary", s.handleGetDeviceMetricsSummary)

				// Tags listing (all unique tags across devices)
				r.Get("/tags", s.handleListAllTags)

				// Device groups (read + resolve)
				r.Get("/device-groups", s.handleListGroups)
				r.Get("/device-groups/{id}", s.handleGetGroup)
				r.Get("/device-groups/{id}/members", s.handleGetGroupMembers)
				r.Get("/device-groups/{id}/resolve", s.handleResolveGroup)
			})

			// ── device:operate — user (room-scoped), admin, owner, panels ──
			r.Group(func(r chi.Router) {
				r.Use(s.requirePermission(auth.PermDeviceOperate))
				r.Use(s.resolveRoomScopeMiddleware)

				r.Put("/devices/{id}/state", s.handleSetDeviceState)
			})

			// ── device:configure — admin, owner only ──
			r.Group(func(r chi.Router) {
				r.Use(s.requirePermission(auth.PermDeviceConfigure))

				r.Post("/devices", s.handleCreateDevice)
				r.Patch("/devices/{id}", s.handleUpdateDevice)
				r.Delete("/devices/{id}", s.handleDeleteDevice)
				r.Put("/devices/{id}/tags", s.handleSetDeviceTags)

				// Device groups (write)
				r.Post("/device-groups", s.handleCreateGroup)
				r.Patch("/device-groups/{id}", s.handleUpdateGroup)
				r.Delete("/device-groups/{id}", s.handleDeleteGroup)
				r.Put("/device-groups/{id}/members", s.handleSetGroupMembers)
			})

			// ── scene:execute — user (room-scoped), admin, owner, panels ──
			r.Group(func(r chi.Router) {
				r.Use(s.requirePermission(auth.PermSceneExecute))
				r.Use(s.resolveRoomScopeMiddleware)

				r.Get("/scenes", s.handleListScenes)
				r.Get("/scenes/{id}", s.handleGetScene)
				r.Post("/scenes/{id}/activate", s.handleActivateScene)
				r.Get("/scenes/{id}/executions", s.handleListSceneExecutions)
			})

			// ── scene:manage — user (room-scoped, can_manage_scenes), admin, owner ──
			r.Group(func(r chi.Router) {
				r.Use(s.requirePermission(auth.PermSceneManage))
				r.Use(s.resolveRoomScopeMiddleware)

				r.Post("/scenes", s.handleCreateScene)
				r.Patch("/scenes/{id}", s.handleUpdateScene)
				r.Delete("/scenes/{id}", s.handleDeleteScene)
			})

			// ── location:manage — admin, owner ──
			r.Group(func(r chi.Router) {
				r.Use(s.requirePermission(auth.PermLocationManage))

				// Hierarchy (single-call site tree)
				r.Get("/hierarchy", s.handleGetHierarchy)

				r.Get("/areas", s.handleListAreas)
				r.Post("/areas", s.handleCreateArea)
				r.Get("/areas/{id}", s.handleGetArea)
				r.Patch("/areas/{id}", s.handleUpdateArea)
				r.Delete("/areas/{id}", s.handleDeleteArea)

				r.Get("/rooms", s.handleListRooms)
				r.Post("/rooms", s.handleCreateRoom)
				r.Get("/rooms/{id}", s.handleGetRoom)
				r.Patch("/rooms/{id}", s.handleUpdateRoom)
				r.Delete("/rooms/{id}", s.handleDeleteRoom)

				// Infrastructure zones
				r.Get("/zones", s.handleListZones)
				r.Post("/zones", s.handleCreateZone)
				r.Get("/zones/{id}", s.handleGetZone)
				r.Patch("/zones/{id}", s.handleUpdateZone)
				r.Delete("/zones/{id}", s.handleDeleteZone)
				r.Get("/zones/{id}/rooms", s.handleGetZoneRooms)
				r.Put("/zones/{id}/rooms", s.handleSetZoneRooms)
			})

			// ── commission:manage — admin, owner ──
			r.Group(func(r chi.Router) {
				r.Use(s.requirePermission(auth.PermCommissionManage))

				r.Post("/commissioning/ets/parse", s.handleETSParse)
				r.Post("/commissioning/ets/import", s.handleETSImport)
			})

			// ── system:admin — admin, owner ──
			r.Group(func(r chi.Router) {
				r.Use(s.requirePermission(auth.PermSystemAdmin))

				r.Get("/site", s.handleGetSite)
				r.Post("/site", s.handleCreateSite)
				r.Patch("/site", s.handleUpdateSite)

				r.Get("/audit-logs", s.handleListAuditLogs)

				// User management
				r.Get("/users", s.handleListUsers)
				r.Post("/users", s.handleCreateUser)
				r.Get("/users/{id}", s.handleGetUser)
				r.Patch("/users/{id}", s.handleUpdateUser)
				r.Delete("/users/{id}", s.handleDeleteUser)
				r.Get("/users/{id}/sessions", s.handleListUserSessions)
				r.Delete("/users/{id}/sessions", s.handleRevokeUserSessions)
				r.Get("/users/{id}/rooms", s.handleGetUserRooms)
				r.Put("/users/{id}/rooms", s.handleSetUserRooms)

				// Panel management
				r.Get("/panels", s.handleListPanels)
				r.Post("/panels", s.handleCreatePanel)
				r.Get("/panels/{id}", s.handleGetPanel)
				r.Patch("/panels/{id}", s.handleUpdatePanel)
				r.Delete("/panels/{id}", s.handleDeletePanel)
				r.Get("/panels/{id}/rooms", s.handleGetPanelRooms)
				r.Put("/panels/{id}/rooms", s.handleSetPanelRooms)
			})

			// ── system:dangerous — owner only ──
			r.Group(func(r chi.Router) {
				r.Use(s.requirePermission(auth.PermSystemDangerous))

				r.Post("/system/factory-reset", s.handleFactoryReset)
			})
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
