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
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/nerrad567/gray-logic-core/migrations"

	"github.com/nerrad567/gray-logic-core/internal/bridges/knx"
	"github.com/nerrad567/gray-logic-core/internal/device"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/database"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/influxdb"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/logging"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/mqtt"
	"github.com/nerrad567/gray-logic-core/internal/knxd"
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

// run is the actual application logic, separated from main for testability.
// Returning an error allows main to handle exit codes consistently.
//
// Parameters:
//   - ctx: Context for cancellation and shutdown signals
//
// Returns:
//   - error: nil on clean shutdown, or error describing failure
func run(ctx context.Context) error {
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

	// Initialise device registry
	deviceRepo := device.NewSQLiteRepository(db.DB)
	deviceRegistry := device.NewRegistry(deviceRepo)
	deviceRegistry.SetLogger(log)

	if refreshErr := deviceRegistry.RefreshCache(ctx); refreshErr != nil {
		return fmt.Errorf("loading device registry: %w", refreshErr)
	}
	log.Info("device registry initialised", "devices", deviceRegistry.GetDeviceCount())

	// Mark deviceRegistry as used (will be needed by REST API in M1.4)
	_ = deviceRegistry

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

	// Connect to InfluxDB (optional)
	var influxClient *influxdb.Client
	if cfg.InfluxDB.Enabled {
		influxClient, err = influxdb.Connect(ctx, cfg.InfluxDB)
		if err != nil {
			return fmt.Errorf("connecting to InfluxDB: %w", err)
		}
		defer func() {
			log.Info("closing InfluxDB connection")
			if closeErr := influxClient.Close(); closeErr != nil {
				log.Error("error closing InfluxDB", "error", closeErr)
			}
		}()
		log.Info("InfluxDB connected",
			"url", cfg.InfluxDB.URL,
			"org", cfg.InfluxDB.Org,
			"bucket", cfg.InfluxDB.Bucket,
		)

		// Set up InfluxDB error callback
		influxClient.SetOnError(func(err error) {
			log.Error("InfluxDB write error", "error", err)
		})
	} else {
		log.Info("InfluxDB disabled")
	}

	// Start knxd daemon (if managed)
	var knxdManager *knxd.Manager
	var busMonitor *knx.BusMonitor
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

		// Start bus monitor for passive device discovery
		// This learns KNX device addresses from bus traffic for health checks
		busMonitor = knx.NewBusMonitor(db.DB)
		busMonitor.SetLogger(log)
		if startErr := busMonitor.Start(ctx, knxdManager.ConnectionURL()); startErr != nil {
			log.Warn("bus monitor failed to start (health checks will use fallback)", "error", startErr)
			busMonitor = nil
		} else {
			defer func() {
				log.Info("stopping bus monitor")
				busMonitor.Stop()
			}()
			log.Info("bus monitor started", "url", knxdManager.ConnectionURL())

			// Wire bus monitor as provider for health checks
			knxdManager.SetDeviceProvider(busMonitor)       // Layer 4: individual addresses
			knxdManager.SetGroupAddressProvider(busMonitor) // Layer 3: group addresses
		}
	}

	// Start KNX bridge (if enabled)
	var knxBridge *knx.Bridge
	if cfg.Protocols.KNX.Enabled {
		knxBridge, err = startKNXBridge(ctx, cfg, knxdManager, mqttClient, log)
		if err != nil {
			return fmt.Errorf("starting KNX bridge: %w", err)
		}
		defer func() {
			log.Info("stopping KNX bridge")
			knxBridge.Stop()
		}()
	} else {
		log.Info("KNX bridge disabled")
	}

	// Verify all connections are healthy
	if err := healthCheck(ctx, db, mqttClient, influxClient); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	log.Info("all health checks passed")

	log.Info("initialisation complete, waiting for shutdown signal")

	// Wait for shutdown signal
	<-ctx.Done()

	log.Info("shutdown signal received, cleaning up")

	// Deferred Close() calls will run in reverse order:
	// 1. InfluxDB (if enabled)
	// 2. MQTT
	// 3. Database

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

// healthCheck verifies all infrastructure connections are healthy.
//
// Parameters:
//   - ctx: Context for timeout/cancellation
//   - db: Database connection to check
//   - mqttClient: MQTT client to check
//   - influxClient: InfluxDB client to check (may be nil if disabled)
//
// Returns:
//   - error: First health check failure, or nil if all healthy
func healthCheck(ctx context.Context, db *database.DB, mqttClient *mqtt.Client, influxClient *influxdb.Client) error {
	// Check database
	if err := db.HealthCheck(ctx); err != nil {
		return fmt.Errorf("database: %w", err)
	}

	// Check MQTT
	if err := mqttClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("mqtt: %w", err)
	}

	// Check InfluxDB (if enabled)
	if influxClient != nil {
		if err := influxClient.HealthCheck(ctx); err != nil {
			return fmt.Errorf("influxdb: %w", err)
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
//
// Returns:
//   - *knx.Bridge: Running KNX bridge
//   - error: If bridge fails to start
func startKNXBridge(ctx context.Context, cfg *config.Config, knxdManager *knxd.Manager, mqttClient *mqtt.Client, log *logging.Logger) (*knx.Bridge, error) {
	// Load KNX bridge configuration (devices, group addresses, mappings)
	knxBridgeCfg, err := knx.LoadConfig(cfg.Protocols.KNX.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("loading KNX bridge config: %w", err)
	}
	log.Info("KNX bridge config loaded",
		"path", cfg.Protocols.KNX.ConfigFile,
		"devices", len(knxBridgeCfg.Devices),
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
	log.Info("connected to knxd", "url", connURL)

	// Create MQTT adapter to satisfy KNX bridge interface
	mqttAdapter := &mqttBridgeAdapter{client: mqttClient, log: log}

	// Create the bridge
	bridge, err := knx.NewBridge(knx.BridgeOptions{
		Config:     knxBridgeCfg,
		MQTTClient: mqttAdapter,
		KNXDClient: knxdClient,
		Logger:     log,
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
