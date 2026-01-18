// Gray Logic Core - Building Intelligence Platform
//
// This is the main entry point for the Gray Logic Core application.
// Gray Logic is a complete building automation system designed for:
//   - 10-year deployment stability
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
)

// Version information - set at build time via ldflags
// Example: go build -ldflags "-X main.version=1.0.0 -X main.commit=abc123"
var (
	version = "dev"     // Semantic version (e.g., "1.0.0")
	commit  = "unknown" // Git commit hash
	date    = "unknown" // Build date
)

func main() {
	// Print startup banner
	fmt.Printf("Gray Logic Core %s (%s) built %s\n", version, commit, date)
	fmt.Println("Building Intelligence Platform")
	fmt.Println("---")

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
//
//nolint:unparam // error return will be used when components are implemented
func run(ctx context.Context) error {
	fmt.Println("Starting Gray Logic Core...")

	// TODO: Initialise components in order:
	// 1. Load configuration
	// 2. Connect to SQLite database
	// 3. Connect to MQTT broker
	// 4. Start API server
	// 5. Start WebSocket hub

	fmt.Println("Initialisation complete. Waiting for shutdown signal...")

	// Wait for shutdown signal
	<-ctx.Done()

	fmt.Println("\nShutdown signal received. Cleaning up...")

	// TODO: Graceful shutdown in reverse order:
	// 1. Stop accepting new API requests
	// 2. Close WebSocket connections
	// 3. Disconnect from MQTT
	// 4. Close database

	fmt.Println("Gray Logic Core stopped.")
	return nil
}
