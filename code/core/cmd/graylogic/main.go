// Gray Logic Core - Building Intelligence Platform
//
// This is the main entry point for the Gray Logic Core application.
// Gray Logic is a complete building automation system designed for:
//   - Multi-decade deployment stability
//   - Offline-first operation (99%+ functionality without internet)
//   - Open standards (KNX, DALI, Modbus)
//   - Zero vendor lock-in
//
// For architecture details, see: docs/architecture/system-overview.md
// For coding standards, see: docs/development/CODING-STANDARDS.md
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/nerrad567/gray-logic-core/migrations"

	"github.com/nerrad567/gray-logic-core/internal/api"
	"github.com/nerrad567/gray-logic-core/internal/audit"
	"github.com/nerrad567/gray-logic-core/internal/auth"
	"github.com/nerrad567/gray-logic-core/internal/automation"
	"github.com/nerrad567/gray-logic-core/internal/bridges/knx"
	"github.com/nerrad567/gray-logic-core/internal/device"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/database"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/logging"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/mqtt"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/tsdb"
	"github.com/nerrad567/gray-logic-core/internal/knxd"
	"github.com/nerrad567/gray-logic-core/internal/location"
)

// Version information - set at build time via ldflags
// Example: go build -ldflags "-X main.version=1.0.0 -X main.commit=abc123"
var (
	version = "dev"     // Semantic version (e.g., "1.0.0")
	commit  = "unknown" // Git commit hash
	date    = "unknown" // Build date
)

// Default configuration file path
const defaultConfigPath = "configs/config.yaml"

const (
	stateHistoryRetention     = 30 * 24 * time.Hour
	stateHistoryPruneInterval = 24 * time.Hour
	stateHistoryPruneTimeout  = 30 * time.Second

	tokenCleanupInterval = 1 * time.Hour
	tokenCleanupTimeout  = 30 * time.Second
)

func main() {
	// Create a context that cancels on interrupt signals (Ctrl+C, SIGTERM)
	// This is the Go pattern for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Run the application
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Startup banner - The Penguin Rises
const banner = `
    .--.
   |o_o |    Gray Logic Core %s
   |:_/ |    Building Intelligence Platform
  //   \ \
 (|     | )  The Penguin Rises
/'\_   _/` + "`" + `\
\___)=(___/
`

// run is the actual application logic, separated from main for testability.
// Returning an error allows main to handle exit codes consistently.
//
// Parameters:
//   - ctx: Context for cancellation and shutdown signals
//
// Returns:
//   - error: nil on clean shutdown, or error describing failure
func run(ctx context.Context) error { //nolint:gocognit,gocyclo // application bootstrap: sequential init steps are inherently linear
	// Print startup banner
	fmt.Printf(banner, version)

	// Use default logger until config is loaded
	log := logging.Default()
	log.Info("starting Gray Logic Core",
		"version", version,
		"commit", commit,
		"build_date", date,
	)

	// Load configuration
	configPath := getConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	log.Info("configuration loaded", "path", configPath)

	// Reinitialise logger with config settings
	log = logging.New(cfg.Logging, version)
	log.Info("logger initialised",
		"level", cfg.Logging.Level,
		"format", cfg.Logging.Format,
	)

	// Open database
	db, err := database.Open(ctx, database.Config{
		Path:        cfg.Database.Path,
		WALMode:     cfg.Database.WALMode,
		BusyTimeout: cfg.Database.BusyTimeout,
	})
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() {
		log.Info("closing database")
		if closeErr := db.Close(); closeErr != nil {
			log.Error("error closing database", "error", closeErr)
		}
	}()
	log.Info("database connected", "path", cfg.Database.Path)

	// Run migrations
	if migrateErr := db.Migrate(ctx); migrateErr != nil {
		return fmt.Errorf("running migrations: %w", migrateErr)
	}
	log.Info("database migrations complete")

	// Check if a site record exists. If not, the system is in "setup needed"
	// mode — the admin panel will detect this via GET /api/v1/site and prompt
	// the user to create one. Property data is user-owned, not config-seeded.
	var siteCount int
	if err := db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sites").Scan(&siteCount); err != nil { //nolint:govet // shadow: err re-declared in nested scope, checked immediately
		return fmt.Errorf("checking site: %w", err)
	}
	if siteCount == 0 {
		log.Info("no site configured — create one via the admin panel")
	}

	// Initialise device registry
	deviceRepo := device.NewSQLiteRepository(db.DB)
	deviceRegistry := device.NewRegistry(deviceRepo)
	deviceRegistry.SetLogger(log)

	// Initialise tag repository and wire into registry BEFORE RefreshCache
	// so that device tags are bulk-loaded during the initial cache population.
	tagRepo := device.NewSQLiteTagRepository(db.DB)
	deviceRegistry.SetTagRepository(tagRepo)

	if refreshErr := deviceRegistry.RefreshCache(ctx); refreshErr != nil {
		return fmt.Errorf("loading device registry: %w", refreshErr)
	}
	log.Info("device registry initialised", "devices", deviceRegistry.GetDeviceCount())

	// Migrate KNX devices from old flat address format to structured functions format.
	// This is a one-time migration that adds DPT and flags to existing devices.
	if migErr := deviceRegistry.MigrateKNXAddressFormat(ctx); migErr != nil {
		log.Error("KNX address format migration failed", "error", migErr)
		// Non-fatal: devices will still work via inference fallback
	}

	// Connect to MQTT broker
	mqttClient, err := mqtt.Connect(cfg.MQTT)
	if err != nil {
		return fmt.Errorf("connecting to MQTT: %w", err)
	}
	defer func() {
		log.Info("disconnecting from MQTT")
		if closeErr := mqttClient.Close(); closeErr != nil {
			log.Error("error closing MQTT", "error", closeErr)
		}
	}()
	log.Info("MQTT connected",
		"broker", fmt.Sprintf("%s:%d", cfg.MQTT.Broker.Host, cfg.MQTT.Broker.Port),
		"client_id", cfg.MQTT.Broker.ClientID,
	)

	// Set up MQTT logging callbacks
	mqttClient.SetOnConnect(func() {
		log.Info("MQTT reconnected")
	})
	mqttClient.SetOnDisconnect(func(err error) {
		log.Warn("MQTT disconnected", "error", err)
	})

	// Initialise scene automation
	sceneRepo := automation.NewSQLiteRepository(db.DB)
	sceneRegistry := automation.NewRegistry(sceneRepo)
	sceneRegistry.SetLogger(log)

	if refreshErr := sceneRegistry.RefreshCache(ctx); refreshErr != nil {
		return fmt.Errorf("loading scene registry: %w", refreshErr)
	}
	log.Info("scene registry initialised", "scenes", sceneRegistry.GetSceneCount())

	// Initialise location repository
	locationRepo := location.NewSQLiteRepository(db.DB)
	log.Info("location repository initialised")

	// Initialise group repository
	groupRepo := device.NewSQLiteGroupRepository(db.DB)
	log.Info("group repository initialised")

	// Initialise zone repository
	zoneRepo := location.NewSQLiteZoneRepository(db.DB)
	log.Info("zone repository initialised")

	// Initialise audit log repository
	auditRepo := audit.NewSQLiteRepository(db.DB)
	log.Info("audit log repository initialised")

	// Initialise auth repositories
	userRepo := auth.NewUserRepository(db.DB)
	tokenRepo := auth.NewTokenRepository(db.DB)
	panelRepo := auth.NewPanelRepository(db.DB)
	roomAccessRepo := auth.NewRoomAccessRepository(db.DB)
	log.Info("auth repositories initialised")

	// Seed owner account on first boot (no users in DB yet)
	if _, seedErr := auth.SeedOwner(ctx, userRepo, log.Logger); seedErr != nil {
		return fmt.Errorf("seeding owner account: %w", seedErr)
	}

	// Start background cleanup of expired refresh tokens
	startTokenCleanupLoop(ctx, tokenRepo, log)

	// Initialise state history repository
	stateHistoryRepo := device.NewSQLiteStateHistoryRepository(db.DB)
	log.Info("state history repository initialised")
	startStateHistoryPruneLoop(ctx, stateHistoryRepo, log)

	// Create WebSocket hub (shared between engine and API server)
	wsHub := api.NewHub(cfg.WebSocket, log)
	go wsHub.Run(ctx)

	// Create scene engine with adapters
	sceneDeviceAdapter := &sceneDeviceRegistryAdapter{registry: deviceRegistry}
	sceneMQTTAdapter := &sceneMQTTClientAdapter{client: mqttClient}
	sceneEngine := automation.NewEngine(sceneRegistry, sceneDeviceAdapter, sceneMQTTAdapter, wsHub, sceneRepo, log)

	// Connect to TSDB (VictoriaMetrics — optional)
	// NOTE: TSDB defer is registered here (after MQTT defer above) so that
	// Go's LIFO defer order shuts down TSDB first. However, the MQTT subscription
	// handler writes to TSDB. We register a late-closing func below to ensure
	// TSDB flushes after MQTT is disconnected.
	var tsdbClient *tsdb.Client
	if cfg.TSDB.Enabled {
		tsdbClient, err = tsdb.Connect(ctx, cfg.TSDB)
		if err != nil {
			return fmt.Errorf("connecting to TSDB: %w", err)
		}
		log.Info("TSDB connected (VictoriaMetrics)",
			"url", cfg.TSDB.URL,
		)

		// Set up TSDB error callback
		tsdbClient.SetOnError(func(err error) {
			log.Error("TSDB write error", "error", err)
		})
	} else {
		log.Info("TSDB disabled")
	}

	// Start API server
	apiServer, err := api.New(api.Deps{
		Config:         cfg.API,
		WS:             cfg.WebSocket,
		Security:       cfg.Security,
		SiteID:         cfg.Site.ID,
		Logger:         log,
		Registry:       deviceRegistry,
		MQTT:           mqttClient,
		DB:             db,
		SceneEngine:    sceneEngine,
		SceneRegistry:  sceneRegistry,
		SceneRepo:      sceneRepo,
		LocationRepo:   locationRepo,
		TagRepo:        tagRepo,
		GroupRepo:      groupRepo,
		ZoneRepo:       zoneRepo,
		AuditRepo:      auditRepo,
		UserRepo:       userRepo,
		TokenRepo:      tokenRepo,
		PanelRepo:      panelRepo,
		RoomAccessRepo: roomAccessRepo,
		StateHistory:   stateHistoryRepo,
		TSDB:           tsdbClient,
		DevMode:        cfg.DevMode,
		PanelDir:       cfg.PanelDir,
		ExternalHub:    wsHub,
		Version:        version + " (" + commit + ")",
	})
	if err != nil {
		return fmt.Errorf("creating API server: %w", err)
	}
	if err = apiServer.Start(ctx); err != nil {
		return fmt.Errorf("starting API server: %w", err)
	}
	defer func() {
		log.Info("stopping API server")
		if closeErr := apiServer.Close(); closeErr != nil {
			log.Error("error stopping API server", "error", closeErr)
		}
	}()
	log.Info("API server started", "address", fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port))

	// Start knxd daemon (if managed)
	var knxdManager *knxd.Manager
	var gaRecorder *knx.GARecorder
	if cfg.Protocols.KNX.Enabled && cfg.Protocols.KNX.KNXD.Managed {
		knxdManager, err = startKNXD(ctx, cfg, log)
		if err != nil {
			return fmt.Errorf("starting knxd: %w", err)
		}
		defer func() {
			log.Info("stopping knxd")
			if stopErr := knxdManager.Stop(); stopErr != nil {
				log.Error("error stopping knxd", "error", stopErr)
			}
		}()

		// Create GA recorder for passive discovery
		// Records all group addresses seen on the bus via the Bridge
		gaRecorder = knx.NewGARecorder(db.DB)
		gaRecorder.SetLogger(log)
		if startErr := gaRecorder.Start(); startErr != nil {
			log.Warn("GA recorder failed to start", "error", startErr)
			gaRecorder = nil
		} else {
			defer func() {
				log.Info("stopping GA recorder")
				gaRecorder.Stop()
			}()
			log.Info("GA recorder started")

			// Wire GA recorder as provider for Layer 3 health checks
			knxdManager.SetGroupAddressProvider(gaRecorder)
		}
	}

	// Start KNX bridge (if enabled)
	var knxBridge *knx.Bridge
	if cfg.Protocols.KNX.Enabled {
		knxBridge, err = startKNXBridge(ctx, cfg, knxdManager, mqttClient, log, deviceRegistry, gaRecorder)
		if err != nil {
			return fmt.Errorf("starting KNX bridge: %w", err)
		}
		defer func() {
			log.Info("stopping KNX bridge")
			knxBridge.Stop()
		}()

		// Wire KNX bridge to API server for device reload after ETS import
		apiServer.SetKNXBridge(knxBridge)

		// Wire KNX metrics provider to API server for /metrics endpoint
		apiServer.SetKNXMetricsProvider(&knxMetricsAdapter{bridge: knxBridge})
	} else {
		log.Info("KNX bridge disabled")
	}

	// Verify all connections are healthy
	if err := healthCheck(ctx, db, mqttClient, tsdbClient); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	log.Info("all health checks passed")

	log.Info("initialisation complete, waiting for shutdown signal")

	// Wait for shutdown signal
	<-ctx.Done()

	log.Info("shutdown signal received, cleaning up")

	// Explicit TSDB close AFTER deferred MQTT disconnect.
	// Deferred Close() calls run in LIFO order (KNX → API → MQTT → DB).
	// TSDB must flush after MQTT stops delivering messages, so we close it
	// explicitly here — before the defers run, but after ctx cancellation
	// has stopped new MQTT message processing.
	if tsdbClient != nil {
		log.Info("closing TSDB connection")
		if closeErr := tsdbClient.Close(); closeErr != nil {
			log.Error("error closing TSDB", "error", closeErr)
		}
	}

	log.Info("Gray Logic Core stopped")
	return nil
}

// getConfigPath returns the configuration file path.
// Uses GRAYLOGIC_CONFIG environment variable if set, otherwise default.
func getConfigPath() string {
	if path := os.Getenv("GRAYLOGIC_CONFIG"); path != "" {
		return path
	}
	return defaultConfigPath
}

// startStateHistoryPruneLoop starts a background task to prune old state history entries.
//
// Parameters:
//   - ctx: Context used for cancellation on shutdown
//   - repo: State history repository used for pruning
//   - log: Logger for pruning diagnostics
func startStateHistoryPruneLoop(ctx context.Context, repo *device.SQLiteStateHistoryRepository, log *logging.Logger) {
	if repo == nil {
		return
	}

	// Run pruning in a background goroutine to avoid blocking startup.
	go func() {
		prune := func() {
			pruneCtx, cancel := context.WithTimeout(ctx, stateHistoryPruneTimeout)
			defer cancel()

			deleted, err := repo.PruneHistory(pruneCtx, stateHistoryRetention)
			if err != nil {
				log.Warn("state history prune failed", "error", err)
				return
			}
			if deleted > 0 {
				log.Info("state history pruned", "deleted", deleted)
			} else {
				log.Debug("state history prune complete", "deleted", deleted)
			}
		}

		// Run once on startup.
		prune()

		ticker := time.NewTicker(stateHistoryPruneInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				prune()
			}
		}
	}()
}

// startTokenCleanupLoop starts a background task to delete expired refresh tokens.
//
// Parameters:
//   - ctx: Context used for cancellation on shutdown
//   - repo: Token repository used for cleanup
//   - log: Logger for cleanup diagnostics
func startTokenCleanupLoop(ctx context.Context, repo auth.TokenRepository, log *logging.Logger) {
	if repo == nil {
		return
	}

	go func() {
		cleanup := func() {
			cleanupCtx, cancel := context.WithTimeout(ctx, tokenCleanupTimeout)
			defer cancel()

			deleted, err := repo.DeleteExpired(cleanupCtx)
			if err != nil {
				log.Warn("refresh token cleanup failed", "error", err)
				return
			}
			if deleted > 0 {
				log.Info("expired refresh tokens cleaned up", "deleted", deleted)
			} else {
				log.Debug("refresh token cleanup complete", "deleted", deleted)
			}
		}

		// Run once on startup.
		cleanup()

		ticker := time.NewTicker(tokenCleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanup()
			}
		}
	}()
}

// healthCheck verifies all infrastructure connections are healthy.
//
// Parameters:
//   - ctx: Context for timeout/cancellation
//   - db: Database connection to check
//   - mqttClient: MQTT client to check
//   - tsdbClient: TSDB client to check (may be nil if disabled)
//
// Returns:
//   - error: First health check failure, or nil if all healthy
func healthCheck(ctx context.Context, db *database.DB, mqttClient *mqtt.Client, tsdbClient *tsdb.Client) error {
	// Check database
	if err := db.HealthCheck(ctx); err != nil {
		return fmt.Errorf("database: %w", err)
	}

	// Check MQTT
	if err := mqttClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("mqtt: %w", err)
	}

	// Check TSDB (if enabled)
	if tsdbClient != nil {
		if err := tsdbClient.HealthCheck(ctx); err != nil {
			return fmt.Errorf("tsdb: %w", err)
		}
	}

	// KNX bridge health is verified during Start() - it connects to knxd
	// and sets up MQTT subscriptions before returning successfully.

	return nil
}

// startKNXD initialises and starts the knxd daemon.
//
// Parameters:
//   - ctx: Context for startup/cancellation
//   - cfg: Application configuration
//   - log: Logger instance
//
// Returns:
//   - *knxd.Manager: Running knxd manager
//   - error: If knxd fails to start
func startKNXD(ctx context.Context, cfg *config.Config, log *logging.Logger) (*knxd.Manager, error) {
	// Convert config types
	knxdCfg := knxd.Config{
		Managed:                  cfg.Protocols.KNX.KNXD.Managed,
		Binary:                   cfg.Protocols.KNX.KNXD.Binary,
		PhysicalAddress:          cfg.Protocols.KNX.KNXD.PhysicalAddress,
		ClientAddresses:          cfg.Protocols.KNX.KNXD.ClientAddresses,
		ListenTCP:                true, // Always listen on TCP for Gray Logic
		TCPPort:                  cfg.Protocols.KNX.KNXDPort,
		RestartOnFailure:         cfg.Protocols.KNX.KNXD.RestartOnFailure,
		RestartDelay:             time.Duration(cfg.Protocols.KNX.KNXD.RestartDelaySeconds) * time.Second,
		MaxRestartAttempts:       cfg.Protocols.KNX.KNXD.MaxRestartAttempts,
		HealthCheckInterval:      cfg.Protocols.KNX.KNXD.HealthCheckInterval,
		HealthCheckDeviceAddress: cfg.Protocols.KNX.KNXD.HealthCheckDeviceAddress,
		HealthCheckDeviceTimeout: cfg.Protocols.KNX.KNXD.HealthCheckDeviceTimeout,
		GroupCache:               cfg.Protocols.KNX.KNXD.GroupCache,
		LogLevel:                 cfg.Protocols.KNX.KNXD.LogLevel,
		Backend: knxd.BackendConfig{
			Type:                 knxd.BackendType(cfg.Protocols.KNX.KNXD.Backend.Type),
			Host:                 cfg.Protocols.KNX.KNXD.Backend.Host,
			Port:                 cfg.Protocols.KNX.KNXD.Backend.Port,
			MulticastAddress:     cfg.Protocols.KNX.KNXD.Backend.MulticastAddress,
			USBVendorID:          cfg.Protocols.KNX.KNXD.Backend.USBVendorID,
			USBProductID:         cfg.Protocols.KNX.KNXD.Backend.USBProductID,
			USBResetOnRetry:      cfg.Protocols.KNX.KNXD.Backend.USBResetOnRetry,
			USBResetOnBusFailure: cfg.Protocols.KNX.KNXD.Backend.USBResetOnBusFailure,
		},
	}

	manager, err := knxd.NewManager(knxdCfg)
	if err != nil {
		return nil, fmt.Errorf("creating knxd manager: %w", err)
	}
	manager.SetLogger(log)

	log.Info("starting knxd",
		"backend", knxdCfg.Backend.Type,
		"physical_address", knxdCfg.PhysicalAddress,
	)

	if err := manager.Start(ctx); err != nil {
		return nil, fmt.Errorf("starting knxd: %w", err)
	}

	log.Info("knxd started",
		"connection_url", manager.ConnectionURL(),
		"managed", manager.IsManaged(),
	)

	return manager, nil
}

// startKNXBridge initialises and starts the KNX protocol bridge.
//
// Parameters:
//   - ctx: Context for connection/cancellation
//   - cfg: Application configuration
//   - knxdManager: knxd manager (may be nil if not managed)
//   - mqttClient: MQTT client for publishing/subscribing
//   - log: Logger instance
//   - deviceRegistry: Device registry for state/health persistence
//
// Returns:
//   - *knx.Bridge: Running KNX bridge
//   - error: If bridge fails to start
func startKNXBridge(ctx context.Context, cfg *config.Config, knxdManager *knxd.Manager, mqttClient *mqtt.Client, log *logging.Logger, deviceRegistry *device.Registry, gaRecorder *knx.GARecorder) (*knx.Bridge, error) {
	// Load KNX bridge configuration (connection settings, MQTT, logging)
	knxBridgeCfg, err := knx.LoadConfig(cfg.Protocols.KNX.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("loading KNX bridge config: %w", err)
	}
	log.Info("KNX bridge config loaded",
		"path", cfg.Protocols.KNX.ConfigFile,
	)

	// Determine connection URL:
	// - If knxd is managed, use its connection URL
	// - Otherwise, use the configured host/port
	var connURL string
	if knxdManager != nil {
		connURL = knxdManager.ConnectionURL()
	} else {
		connURL = fmt.Sprintf("tcp://%s:%d", cfg.Protocols.KNX.KNXDHost, cfg.Protocols.KNX.KNXDPort)
	}

	// Connect to knxd daemon
	knxdClient, err := knx.Connect(ctx, knx.KNXDConfig{
		Connection: connURL,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to knxd: %w", err)
	}
	knxdClient.SetLogger(log)
	log.Info("connected to knxd", "url", connURL)

	// Create MQTT adapter to satisfy KNX bridge interface
	mqttAdapter := &mqttBridgeAdapter{client: mqttClient, log: log}

	// Create registry adapter to convert string health status to device.HealthStatus
	registryAdapter := &deviceRegistryAdapter{registry: deviceRegistry}

	// Create the bridge
	bridge, err := knx.NewBridge(knx.BridgeOptions{
		Config:     knxBridgeCfg,
		MQTTClient: mqttAdapter,
		KNXDClient: knxdClient,
		Logger:     log,
		Registry:   registryAdapter,
		GARecorder: gaRecorder, // May be nil if not started
	})
	if err != nil {
		// Clean up knxd connection on error
		_ = knxdClient.Close()
		return nil, fmt.Errorf("creating KNX bridge: %w", err)
	}

	// Start the bridge
	if err := bridge.Start(ctx); err != nil {
		_ = knxdClient.Close()
		return nil, fmt.Errorf("starting KNX bridge: %w", err)
	}
	log.Info("KNX bridge started")

	return bridge, nil
}

// mqttBridgeAdapter adapts the infrastructure MQTT client to the KNX bridge's
// MQTTClient interface. The primary difference is the Subscribe handler signature:
// - Infrastructure mqtt: func(topic, payload []byte) error
// - KNX bridge expects: func(topic, payload []byte)
type mqttBridgeAdapter struct {
	client *mqtt.Client
	log    *logging.Logger
}

// Publish implements knx.MQTTClient.
func (a *mqttBridgeAdapter) Publish(topic string, payload []byte, qos byte, retained bool) error {
	return a.client.Publish(topic, payload, qos, retained)
}

// Subscribe implements knx.MQTTClient.
func (a *mqttBridgeAdapter) Subscribe(topic string, qos byte, handler func(topic string, payload []byte)) error {
	// Wrap the void handler to return nil error (KNX handlers don't return errors)
	return a.client.Subscribe(topic, qos, func(t string, p []byte) error {
		handler(t, p)
		return nil
	})
}

// IsConnected implements knx.MQTTClient.
func (a *mqttBridgeAdapter) IsConnected() bool {
	return a.client.IsConnected()
}

// Disconnect implements knx.MQTTClient.
// Note: When wired into main.go, the MQTT client is managed by the Core,
// so this is a no-op. The actual disconnect happens via the defer chain.
func (a *mqttBridgeAdapter) Disconnect(_ uint) {
	// No-op: MQTT client lifecycle is managed by Core's defer chain
}

// deviceRegistryAdapter adapts the device.Registry to the knx.DeviceRegistry interface.
// The primary difference is the HealthStatus type:
// - device.Registry expects: device.HealthStatus (typed string)
// - knx.DeviceRegistry expects: string (untyped)
type deviceRegistryAdapter struct {
	registry *device.Registry
}

// SetDeviceState implements knx.DeviceRegistry.
func (a *deviceRegistryAdapter) SetDeviceState(ctx context.Context, id string, state map[string]any) error {
	return a.registry.SetDeviceState(ctx, id, state)
}

// SetDeviceHealth implements knx.DeviceRegistry.
func (a *deviceRegistryAdapter) SetDeviceHealth(ctx context.Context, id string, status string) error {
	return a.registry.SetDeviceHealth(ctx, id, device.HealthStatus(status))
}

// CreateDeviceIfNotExists implements knx.DeviceRegistry.
// Seeds a device record from bridge config if it doesn't already exist.
func (a *deviceRegistryAdapter) CreateDeviceIfNotExists(ctx context.Context, seed knx.DeviceSeed) error {
	_, err := a.registry.GetDevice(ctx, seed.ID)
	if err == nil {
		return nil // Already exists
	}
	if !errors.Is(err, device.ErrDeviceNotFound) {
		return err
	}

	caps := make([]device.Capability, len(seed.Capabilities))
	for i, c := range seed.Capabilities {
		caps[i] = device.Capability(c)
	}
	addr := make(device.Address, len(seed.Address))
	for k, v := range seed.Address {
		addr[k] = v
	}

	dev := &device.Device{
		ID:           seed.ID,
		Name:         seed.Name,
		Type:         device.DeviceType(seed.Type),
		Domain:       device.Domain(seed.Domain),
		Protocol:     device.Protocol(seed.Protocol),
		Capabilities: caps,
		Address:      addr,
		HealthStatus: device.HealthStatusUnknown,
	}
	return a.registry.CreateDevice(ctx, dev)
}

// GetKNXDevices implements knx.DeviceRegistry.
// Returns all devices with protocol "knx" for bridge device mapping.
func (a *deviceRegistryAdapter) GetKNXDevices(ctx context.Context) ([]knx.RegistryDevice, error) {
	devices, err := a.registry.GetDevicesByProtocol(ctx, device.ProtocolKNX)
	if err != nil {
		return nil, err
	}

	result := make([]knx.RegistryDevice, len(devices))
	for i, dev := range devices {
		caps := make([]string, len(dev.Capabilities))
		for j, c := range dev.Capabilities {
			caps[j] = string(c)
		}

		// Parse structured functions map from device address
		functions := make(map[string]knx.FunctionMapping)
		if knxFuncs := device.GetKNXFunctions(dev.Address); knxFuncs != nil {
			for name, fc := range knxFuncs {
				functions[name] = knx.FunctionMapping{
					GA:    fc.GA,
					DPT:   fc.DPT,
					Flags: fc.Flags,
				}
			}
		}

		result[i] = knx.RegistryDevice{
			ID:           dev.ID,
			Name:         dev.Name,
			Type:         string(dev.Type),
			Domain:       string(dev.Domain),
			Functions:    functions,
			Capabilities: caps,
		}
	}
	return result, nil
}

// sceneDeviceRegistryAdapter adapts the device.Registry to the
// automation.DeviceRegistry interface. It extracts only the minimal
// DeviceInfo (ID, Protocol, GatewayID) needed for MQTT command routing.
type sceneDeviceRegistryAdapter struct {
	registry *device.Registry
}

// GetDevice implements automation.DeviceRegistry.
func (a *sceneDeviceRegistryAdapter) GetDevice(ctx context.Context, id string) (automation.DeviceInfo, error) {
	dev, err := a.registry.GetDevice(ctx, id)
	if err != nil {
		return automation.DeviceInfo{}, err
	}
	var gatewayID *string
	if dev.GatewayID != nil {
		gid := *dev.GatewayID
		gatewayID = &gid
	}
	return automation.DeviceInfo{
		ID:        dev.ID,
		Protocol:  string(dev.Protocol),
		GatewayID: gatewayID,
	}, nil
}

// sceneMQTTClientAdapter adapts the infrastructure mqtt.Client to the
// automation.MQTTClient interface (which only requires Publish).
type sceneMQTTClientAdapter struct {
	client *mqtt.Client
}

// Publish implements automation.MQTTClient.
func (a *sceneMQTTClientAdapter) Publish(topic string, payload []byte, qos byte, retained bool) error {
	return a.client.Publish(topic, payload, qos, retained)
}

// knxMetricsAdapter adapts knx.Bridge to api.KNXMetricsProvider.
type knxMetricsAdapter struct {
	bridge *knx.Bridge
}

// GetMetrics implements api.KNXMetricsProvider.
func (a *knxMetricsAdapter) GetMetrics() api.KNXBridgeMetrics {
	m := a.bridge.GetMetrics()
	return api.KNXBridgeMetrics{
		Connected:      m.Connected,
		Status:         m.Status,
		TelegramsTx:    m.TelegramsTx,
		TelegramsRx:    m.TelegramsRx,
		DevicesManaged: m.DevicesManaged,
	}
}
