// Package api provides the HTTP REST API and WebSocket server for Gray Logic Core.
//
// It exposes device registry operations, real-time state updates, and system
// management endpoints to user interfaces (Flutter wall panels, mobile apps,
// web admin).
//
// The server follows the same lifecycle pattern as other infrastructure components:
//
//	server, err := api.New(deps)
//	server.Start(ctx)
//	defer server.Close()
//
// Thread Safety: All methods are safe for concurrent use from multiple goroutines.
package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/nerrad567/gray-logic-core/internal/audit"
	"github.com/nerrad567/gray-logic-core/internal/auth"
	"github.com/nerrad567/gray-logic-core/internal/automation"
	"github.com/nerrad567/gray-logic-core/internal/device"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/logging"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/mqtt"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/tsdb"
	"github.com/nerrad567/gray-logic-core/internal/location"
)

// gracefulShutdownTimeout is the maximum time to wait for in-flight requests
// to complete during shutdown.
const gracefulShutdownTimeout = 10 * time.Second

// KNXBridgeReloader is an interface for reloading KNX bridge device mappings.
// This allows the API server to trigger a device reload after ETS import
// without creating a circular dependency on the knx package.
type KNXBridgeReloader interface {
	ReloadDevices(ctx context.Context)
}

// KNXBridgeMetrics contains metrics data from the KNX bridge.
type KNXBridgeMetrics struct {
	Connected      bool
	Status         string
	TelegramsTx    uint64
	TelegramsRx    uint64
	DevicesManaged int
}

// KNXMetricsProvider is an interface for getting KNX bridge metrics.
// This allows the API server to collect metrics without importing the knx package.
type KNXMetricsProvider interface {
	GetMetrics() KNXBridgeMetrics
}

// DBStatsProvider is an interface for getting database statistics and access.
type DBStatsProvider interface {
	Stats() sql.DBStats
	SQLDB() *sql.DB // Returns the underlying *sql.DB for queries
}

// Deps holds the dependencies required by the API server.
type Deps struct {
	Config         config.APIConfig
	WS             config.WebSocketConfig
	Security       config.SecurityConfig
	SiteID         string // Site ID for location hierarchy (from config)
	Logger         *logging.Logger
	Registry       *device.Registry
	MQTT           *mqtt.Client
	DB             DBStatsProvider // Optional: for metrics endpoint
	SceneEngine    *automation.Engine
	SceneRegistry  *automation.Registry
	SceneRepo      automation.Repository
	LocationRepo   location.Repository
	TagRepo        device.TagRepository
	GroupRepo      device.GroupRepository
	ZoneRepo       location.ZoneRepository
	AuditRepo      audit.Repository
	UserRepo       auth.UserRepository
	TokenRepo      auth.TokenRepository
	PanelRepo      auth.PanelRepository
	RoomAccessRepo auth.RoomAccessRepository
	StateHistory   device.StateHistoryRepository
	TSDB           *tsdb.Client // Optional: time-series database for device telemetry
	ExternalHub    *Hub         // If set, the server uses this hub instead of creating its own
	DevMode        bool         // When true, commands apply state locally without bridge confirmation
	PanelDir       string       // Dev only: serve Flutter panel from filesystem instead of embed
	Version        string
}

// Server is the HTTP API server for Gray Logic Core.
//
// It manages the HTTP listener, routes, middleware, and WebSocket hub.
// The server is created with New() and started with Start().
type Server struct {
	cfg                config.APIConfig
	wsCfg              config.WebSocketConfig
	secCfg             config.SecurityConfig
	siteID             string
	logger             *logging.Logger
	registry           *device.Registry
	mqtt               *mqtt.Client
	db                 DBStatsProvider
	sceneEngine        *automation.Engine
	sceneRegistry      *automation.Registry
	sceneRepo          automation.Repository
	locationRepo       location.Repository
	tagRepo            device.TagRepository
	groupRepo          device.GroupRepository
	zoneRepo           location.ZoneRepository
	auditRepo          audit.Repository
	userRepo           auth.UserRepository
	tokenRepo          auth.TokenRepository
	panelRepo          auth.PanelRepository
	roomAccessRepo     auth.RoomAccessRepository
	stateHistory       device.StateHistoryRepository
	tsdb               *tsdb.Client // optional: time-series telemetry writes
	devMode            bool
	panelDir           string
	version            string
	startTime          time.Time // server start time for uptime calculation
	server             *http.Server
	hub                *Hub
	externalHub        bool               // true if hub was injected externally
	cancel             context.CancelFunc // cancels background goroutines on Close()
	rateLimiter        *rateLimiter
	panelCache         *panelAuthCache      // in-memory cache for panel token auth lookups
	scopeCache         *roomScopeCache      // in-memory cache for user room scope resolution
	wsTickets          *ticketStore         // WebSocket auth ticket store (moved from package-level)
	jwtSecretBytes     []byte               // pre-converted JWT secret to avoid per-request allocation
	auditCh            chan *audit.AuditLog // buffered channel for async audit log writes
	knxBridge          KNXBridgeReloader    // optional: for reloading devices after ETS import
	knxMetricsProvider KNXMetricsProvider   // optional: for metrics endpoint
}

// New creates a new API server with the given dependencies.
//
// The server is not started until Start() is called.
//
// Parameters:
//   - deps: Required dependencies (config, logger, registry, MQTT)
//
// Returns:
//   - *Server: Configured server ready to start
//   - error: If required dependencies are missing
func New(deps Deps) (*Server, error) {
	if deps.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if deps.Registry == nil {
		return nil, fmt.Errorf("device registry is required")
	}
	// MQTT is optional — commands won't work without it but reads/WebSocket still function

	s := &Server{
		cfg:            deps.Config,
		wsCfg:          deps.WS,
		secCfg:         deps.Security,
		siteID:         deps.SiteID,
		logger:         deps.Logger,
		registry:       deps.Registry,
		mqtt:           deps.MQTT,
		db:             deps.DB,
		sceneEngine:    deps.SceneEngine,
		sceneRegistry:  deps.SceneRegistry,
		sceneRepo:      deps.SceneRepo,
		locationRepo:   deps.LocationRepo,
		tagRepo:        deps.TagRepo,
		groupRepo:      deps.GroupRepo,
		zoneRepo:       deps.ZoneRepo,
		auditRepo:      deps.AuditRepo,
		userRepo:       deps.UserRepo,
		tokenRepo:      deps.TokenRepo,
		panelRepo:      deps.PanelRepo,
		roomAccessRepo: deps.RoomAccessRepo,
		stateHistory:   deps.StateHistory,
		tsdb:           deps.TSDB,
		devMode:        deps.DevMode,
		panelDir:       deps.PanelDir,
		version:        deps.Version,
		startTime:      time.Now(),
		rateLimiter:    newRateLimiter(),
		panelCache:     newPanelAuthCache(),
		scopeCache:     newRoomScopeCache(),
		wsTickets:      newTicketStore(),
		jwtSecretBytes: []byte(deps.Security.JWT.Secret),
		auditCh:        make(chan *audit.AuditLog, auditChanSize),
	}

	// Use externally-provided hub if available (needed when Engine also
	// requires the hub for WebSocket broadcasting).
	if deps.ExternalHub != nil {
		s.hub = deps.ExternalHub
		s.externalHub = true
	}

	return s, nil
}

// SetKNXBridge sets the KNX bridge for device reload after ETS import.
// This is called after both the API server and KNX bridge are created,
// since they have a startup order dependency.
func (s *Server) SetKNXBridge(bridge KNXBridgeReloader) {
	s.knxBridge = bridge
}

// SetKNXMetricsProvider sets the KNX metrics provider for the metrics endpoint.
func (s *Server) SetKNXMetricsProvider(provider KNXMetricsProvider) {
	s.knxMetricsProvider = provider
}

// Start begins listening for HTTP connections.
//
// It sets up the router, starts the WebSocket hub, subscribes to MQTT state
// topics for real-time WebSocket broadcast, and launches the HTTP listener
// in a background goroutine. The server can be stopped with Close().
//
// Parameters:
//   - ctx: Context for cancellation (not used for listener lifetime)
//
// Returns:
//   - error: If the server fails to start (port in use, etc.)
func (s *Server) Start(ctx context.Context) error {
	// Create internal context so Close() can stop background goroutines
	// independently of the parent context.
	var srvCtx context.Context
	srvCtx, s.cancel = context.WithCancel(ctx)

	// Create WebSocket hub (unless one was injected externally)
	if s.hub == nil {
		s.hub = NewHub(s.wsCfg, s.logger)
		go s.hub.Run(srvCtx)
	}

	// Start audit log drain goroutine (replaces per-call goroutines)
	if s.auditRepo != nil {
		go s.drainAuditLog(srvCtx)
	}

	// Start periodic ticket cleanup to prevent memory leaks
	go s.cleanTicketsLoop(srvCtx)
	if s.rateLimiter != nil {
		go s.rateLimiter.cleanupLoop(srvCtx, rateLimitWindow)
	}

	// Subscribe to device state changes from bridges for WebSocket broadcast
	if err := s.subscribeStateUpdates(); err != nil {
		s.logger.Warn("failed to subscribe to state updates for WebSocket", "error", err)
	}

	// Build router
	router := s.buildRouter()

	// Create HTTP server
	s.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port),
		Handler:           router,
		ReadTimeout:       time.Duration(s.cfg.Timeouts.Read) * time.Second,
		ReadHeaderTimeout: time.Duration(s.cfg.Timeouts.Read) * time.Second,
		WriteTimeout:      time.Duration(s.cfg.Timeouts.Write) * time.Second,
		IdleTimeout:       time.Duration(s.cfg.Timeouts.Idle) * time.Second,
	}

	// Start listening in background
	go func() {
		var err error
		if s.cfg.TLS.Enabled {
			s.logger.Info("API server starting with TLS",
				"address", s.server.Addr,
				"cert", s.cfg.TLS.CertFile,
			)
			err = s.server.ListenAndServeTLS(s.cfg.TLS.CertFile, s.cfg.TLS.KeyFile)
		} else {
			err = s.server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("API server error", "error", err)
		}
	}()

	return nil
}

// Close gracefully shuts down the API server.
//
// It waits up to 10 seconds for in-flight requests to complete,
// then forcefully closes remaining connections.
//
// Returns:
//   - error: If shutdown encounters an error
func (s *Server) Close() error {
	if s.server == nil {
		return nil
	}

	// Cancel background goroutines (hub, ticket cleanup)
	if s.cancel != nil {
		s.cancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()

	s.logger.Info("API server shutting down")
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutting down API server: %w", err)
	}
	return nil
}

// HealthCheck verifies the API server is running and responsive.
//
// Parameters:
//   - ctx: Context for timeout/cancellation
//
// Returns:
//   - error: nil if healthy, error describing the issue otherwise
func (s *Server) HealthCheck(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("api health check: %w", ctx.Err())
	default:
	}

	if s.server == nil {
		return fmt.Errorf("api server not started")
	}

	return nil
}
