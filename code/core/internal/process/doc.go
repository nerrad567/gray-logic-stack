// Package process provides generic subprocess lifecycle management.
//
// This package is designed for managing long-running child processes like
// protocol daemons (knxd, DALI gateways, etc.) that Gray Logic depends on.
//
// Features:
//   - Start/stop subprocess with graceful shutdown
//   - Automatic restart on failure with configurable backoff
//   - Health monitoring and status reporting
//   - Log capture from subprocess stdout/stderr
//   - Context-based cancellation for clean shutdown
//
// Example usage:
//
//	mgr := process.NewManager(process.Config{
//	    Name:              "knxd",
//	    Binary:            "/usr/bin/knxd",
//	    Args:              []string{"-e", "0.0.1", "-b", "usb:"},
//	    RestartOnFailure:  true,
//	    RestartDelay:      5 * time.Second,
//	    MaxRestartAttempts: 10,
//	})
//
//	if err := mgr.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer mgr.Stop()
package process
