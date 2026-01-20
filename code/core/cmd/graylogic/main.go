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

	_ "github.com/nerrad567/gray-logic-core/migrations"

	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/database"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/influxdb"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/logging"
	"github.com/nerrad567/gray-logic-core/internal/infrastructure/mqtt"
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
	db, err := database.Open(database.Config{
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
		influxClient, err = influxdb.Connect(cfg.InfluxDB)
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

	return nil
}
