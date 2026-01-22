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
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/nerrad567/gray-logic-core/internal/device"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/logging"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/mqtt"
)

// gracefulShutdownTimeout is the maximum time to wait for in-flight requests
// to complete during shutdown.
const gracefulShutdownTimeout = 10 * time.Second

// Deps holds the dependencies required by the API server.
type Deps struct {
	Config   config.APIConfig
	WS       config.WebSocketConfig
	Security config.SecurityConfig
	Logger   *logging.Logger
	Registry *device.Registry
	MQTT     *mqtt.Client
	Version  string
}

// Server is the HTTP API server for Gray Logic Core.
//
// It manages the HTTP listener, routes, middleware, and WebSocket hub.
// The server is created with New() and started with Start().
type Server struct {
	cfg      config.APIConfig
	wsCfg    config.WebSocketConfig
	secCfg   config.SecurityConfig
	logger   *logging.Logger
	registry *device.Registry
	mqtt     *mqtt.Client
	version  string
	server   *http.Server
	hub      *Hub
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
		cfg:      deps.Config,
		wsCfg:    deps.WS,
		secCfg:   deps.Security,
		logger:   deps.Logger,
		registry: deps.Registry,
		mqtt:     deps.MQTT,
		version:  deps.Version,
	}

	return s, nil
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
	// Create WebSocket hub
	s.hub = NewHub(s.wsCfg, s.logger)
	go s.hub.Run(ctx)

	// Subscribe to device state changes from bridges for WebSocket broadcast
	if err := s.subscribeStateUpdates(); err != nil {
		s.logger.Warn("failed to subscribe to state updates for WebSocket", "error", err)
	}

	// Build router
	router := s.buildRouter()

	// Create HTTP server
	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(s.cfg.Timeouts.Read) * time.Second,
		WriteTimeout: time.Duration(s.cfg.Timeouts.Write) * time.Second,
		IdleTimeout:  time.Duration(s.cfg.Timeouts.Idle) * time.Second,
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
