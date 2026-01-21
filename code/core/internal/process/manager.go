package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Status represents the current state of a managed process.
type Status string

const (
	StatusStopped  Status = "stopped"
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusFailed   Status = "failed"
)

// outputBufferSize is the buffer size for capturing subprocess stdout/stderr.
const outputBufferSize = 4096

// Config holds configuration for a managed subprocess.
type Config struct {
	// Name is a human-readable identifier for logging.
	Name string

	// Binary is the path to the executable.
	Binary string

	// Args are command-line arguments to pass to the binary.
	Args []string

	// Env are additional environment variables (key=value format).
	// If nil, inherits from parent process.
	Env []string

	// WorkDir is the working directory for the process.
	// If empty, inherits from parent process.
	WorkDir string

	// RestartOnFailure enables automatic restart when the process exits unexpectedly.
	RestartOnFailure bool

	// RestartDelay is the time to wait before restarting after a failure.
	RestartDelay time.Duration

	// MaxRestartAttempts limits restart attempts. 0 means unlimited.
	MaxRestartAttempts int

	// GracefulTimeout is how long to wait for graceful shutdown before SIGKILL.
	GracefulTimeout time.Duration

	// HealthCheckFunc is called periodically to verify the process is healthy.
	// If nil, process is considered healthy if running.
	HealthCheckFunc func(ctx context.Context) error

	// HealthCheckInterval is how often to run health checks.
	HealthCheckInterval time.Duration

	// OnStart is called when the process starts successfully.
	OnStart func()

	// OnStop is called when the process stops (either normally or due to failure).
	OnStop func(err error)

	// OnRestart is called before each restart attempt.
	OnRestart func(attempt int)
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(name, binary string, args []string) Config {
	return Config{
		Name:                name,
		Binary:              binary,
		Args:                args,
		RestartOnFailure:    true,
		RestartDelay:        5 * time.Second,
		MaxRestartAttempts:  10,
		GracefulTimeout:     10 * time.Second,
		HealthCheckInterval: 30 * time.Second,
	}
}

// Logger defines the logging interface for the process manager.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// noopLogger is a logger that does nothing.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

// Manager manages the lifecycle of a subprocess.
type Manager struct {
	config Config
	logger Logger

	mu            sync.RWMutex
	cmd           *exec.Cmd
	status        Status
	restartCount  int
	lastError     error
	startTime     time.Time
	stopRequested bool

	// Channels for coordination
	done chan struct{}
}

// NewManager creates a new process manager with the given configuration.
func NewManager(cfg Config) *Manager {
	// Apply defaults for zero values
	if cfg.RestartDelay == 0 {
		cfg.RestartDelay = 5 * time.Second
	}
	if cfg.GracefulTimeout == 0 {
		cfg.GracefulTimeout = 10 * time.Second
	}
	if cfg.HealthCheckInterval == 0 {
		cfg.HealthCheckInterval = 30 * time.Second
	}

	return &Manager{
		config: cfg,
		logger: noopLogger{},
		status: StatusStopped,
	}
}

// SetLogger sets the logger for the manager.
func (m *Manager) SetLogger(logger Logger) {
	m.logger = logger
}

// Start launches the subprocess and begins monitoring it.
// Returns an error if the process fails to start.
// The process will be automatically restarted on failure if configured.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.status == StatusRunning || m.status == StatusStarting {
		m.mu.Unlock()
		return fmt.Errorf("process %s is already running", m.config.Name)
	}
	m.status = StatusStarting
	m.stopRequested = false
	m.done = make(chan struct{})
	m.mu.Unlock()

	if err := m.startProcess(ctx); err != nil {
		m.mu.Lock()
		m.status = StatusFailed
		m.lastError = err
		m.mu.Unlock()
		return err
	}

	// Start the monitor goroutine
	go m.monitor(ctx)

	return nil
}

// startProcess actually starts the subprocess.
func (m *Manager) startProcess(ctx context.Context) error {
	m.logger.Info("starting process",
		"name", m.config.Name,
		"binary", m.config.Binary,
		"args", m.config.Args,
	)

	cmd := exec.CommandContext(ctx, m.config.Binary, m.config.Args...) //nolint:gosec // Binary path is validated in knxd.Config.Validate()

	// Create a new process group so we can signal all children on shutdown
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Set environment
	if m.config.Env != nil {
		cmd.Env = append(os.Environ(), m.config.Env...)
	}

	// Set working directory
	if m.config.WorkDir != "" {
		cmd.Dir = m.config.WorkDir
	}

	// Capture stdout/stderr for logging
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", m.config.Name, err)
	}

	m.mu.Lock()
	m.cmd = cmd
	m.status = StatusRunning
	m.startTime = time.Now()
	m.mu.Unlock()

	// Start log capture goroutines
	go m.captureOutput("stdout", stdout)
	go m.captureOutput("stderr", stderr)

	m.logger.Info("process started",
		"name", m.config.Name,
		"pid", cmd.Process.Pid,
	)

	if m.config.OnStart != nil {
		m.config.OnStart()
	}

	return nil
}

// captureOutput reads from the given reader and logs each line.
func (m *Manager) captureOutput(stream string, r io.Reader) {
	buf := make([]byte, outputBufferSize)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			m.logger.Debug("process output",
				"name", m.config.Name,
				"stream", stream,
				"output", string(buf[:n]),
			)
		}
		if err != nil {
			if err != io.EOF {
				m.logger.Debug("output stream closed",
					"name", m.config.Name,
					"stream", stream,
				)
			}
			return
		}
	}
}

// waitForExitOrHealthFailure waits for the process to exit or for a health check to fail.
// If a health check fails, it kills the process and returns an error.
// This implements watchdog functionality to detect hung processes.
func (m *Manager) waitForExitOrHealthFailure(ctx context.Context, cmd *exec.Cmd) error {
	// Channel to receive process exit
	exitCh := make(chan error, 1)
	go func() {
		exitCh <- cmd.Wait()
	}()

	// If no health check function, just wait for exit
	if m.config.HealthCheckFunc == nil {
		return <-exitCh
	}

	// Health check ticker
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()

	consecutiveFailures := 0
	const maxConsecutiveFailures = 3 // Kill after 3 consecutive failures

	for {
		select {
		case err := <-exitCh:
			// Process exited normally or crashed
			return err

		case <-ctx.Done():
			// Context cancelled, return to let monitor handle it
			return ctx.Err()

		case <-ticker.C:
			// Run health check
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := m.config.HealthCheckFunc(checkCtx)
			cancel()

			if err != nil {
				consecutiveFailures++
				m.logger.Warn("health check failed",
					"name", m.config.Name,
					"error", err,
					"consecutive_failures", consecutiveFailures,
				)

				if consecutiveFailures >= maxConsecutiveFailures {
					m.logger.Error("health check failed repeatedly, killing process",
						"name", m.config.Name,
						"failures", consecutiveFailures,
					)

					// Kill the hung process
					if cmd.Process != nil {
						cmd.Process.Kill()
					}

					// Wait for exit and return
					select {
					case exitErr := <-exitCh:
						if exitErr != nil {
							return fmt.Errorf("killed due to health check failure: %w", exitErr)
						}
						return fmt.Errorf("killed due to health check failure after %d consecutive failures", consecutiveFailures)
					case <-time.After(5 * time.Second):
						return fmt.Errorf("process did not exit after kill (health check failure)")
					}
				}
			} else {
				// Health check passed, reset counter
				if consecutiveFailures > 0 {
					m.logger.Info("health check recovered",
						"name", m.config.Name,
						"previous_failures", consecutiveFailures,
					)
				}
				consecutiveFailures = 0
			}
		}
	}
}

// monitor watches the process and handles restarts.
func (m *Manager) monitor(ctx context.Context) {
	defer close(m.done)

	for {
		m.mu.RLock()
		cmd := m.cmd
		m.mu.RUnlock()

		if cmd == nil {
			return
		}

		// Wait for process to exit OR health check failure
		err := m.waitForExitOrHealthFailure(ctx, cmd)

		m.mu.Lock()
		stopRequested := m.stopRequested
		m.mu.Unlock()

		// If stop was requested, don't restart
		if stopRequested {
			m.logger.Info("process stopped as requested", "name", m.config.Name)
			m.mu.Lock()
			m.status = StatusStopped
			m.mu.Unlock()
			if m.config.OnStop != nil {
				m.config.OnStop(nil)
			}
			return
		}

		// Process exited unexpectedly
		m.logger.Warn("process exited unexpectedly",
			"name", m.config.Name,
			"error", err,
		)

		m.mu.Lock()
		m.lastError = err
		m.status = StatusFailed
		m.mu.Unlock()

		if m.config.OnStop != nil {
			m.config.OnStop(err)
		}

		// Check if we should restart
		if !m.config.RestartOnFailure {
			m.logger.Info("restart disabled, not restarting", "name", m.config.Name)
			return
		}

		m.mu.Lock()
		m.restartCount++
		attempt := m.restartCount
		m.mu.Unlock()

		if m.config.MaxRestartAttempts > 0 && attempt > m.config.MaxRestartAttempts {
			m.logger.Error("max restart attempts reached",
				"name", m.config.Name,
				"attempts", attempt,
			)
			return
		}

		// Wait before restarting
		m.logger.Info("restarting process",
			"name", m.config.Name,
			"attempt", attempt,
			"delay", m.config.RestartDelay,
		)

		if m.config.OnRestart != nil {
			m.config.OnRestart(attempt)
		}

		select {
		case <-ctx.Done():
			m.logger.Info("context cancelled, not restarting", "name", m.config.Name)
			return
		case <-time.After(m.config.RestartDelay):
		}

		// Check if stop was requested during the delay
		m.mu.RLock()
		stopRequested = m.stopRequested
		m.mu.RUnlock()
		if stopRequested {
			return
		}

		// Attempt restart
		if err := m.startProcess(ctx); err != nil {
			m.logger.Error("failed to restart process",
				"name", m.config.Name,
				"error", err,
			)
			// Continue loop to try again
		}
	}
}

// Stop gracefully stops the subprocess.
// It sends SIGTERM and waits for graceful shutdown, then SIGKILL if needed.
func (m *Manager) Stop() error {
	m.mu.Lock()
	if m.status != StatusRunning && m.status != StatusStarting {
		m.mu.Unlock()
		return nil
	}
	m.stopRequested = true
	cmd := m.cmd
	done := m.done // Capture done channel under lock to avoid race
	m.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// done channel may be nil if Stop() is called before Start() completes
	if done == nil {
		return nil
	}

	pid := cmd.Process.Pid
	m.logger.Info("stopping process", "name", m.config.Name, "pid", pid)

	// Send SIGTERM to the entire process group for graceful shutdown
	// Use negative PID to signal the process group (created via Setpgid)
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		// Process might have already exited
		if !errors.Is(err, syscall.ESRCH) {
			m.logger.Warn("failed to send SIGTERM to process group", "name", m.config.Name, "error", err)
		}
	}

	// Wait for graceful shutdown or timeout
	select {
	case <-done:
		m.logger.Info("process stopped gracefully", "name", m.config.Name)
		return nil
	case <-time.After(m.config.GracefulTimeout):
		m.logger.Warn("graceful shutdown timeout, sending SIGKILL",
			"name", m.config.Name,
			"timeout", m.config.GracefulTimeout,
		)
	}

	// Force kill the entire process group
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		if !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("killing process group %s: %w", m.config.Name, err)
		}
	}

	// Wait for process to fully exit
	<-done
	m.logger.Info("process killed", "name", m.config.Name)

	return nil
}

// Status returns the current status of the managed process.
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// IsRunning returns true if the process is currently running.
func (m *Manager) IsRunning() bool {
	return m.Status() == StatusRunning
}

// LastError returns the last error that caused the process to exit.
func (m *Manager) LastError() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastError
}

// RestartCount returns the number of times the process has been restarted.
func (m *Manager) RestartCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.restartCount
}

// Uptime returns how long the process has been running.
// Returns 0 if the process is not running.
func (m *Manager) Uptime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status != StatusRunning {
		return 0
	}
	return time.Since(m.startTime)
}

// PID returns the process ID, or 0 if not running.
func (m *Manager) PID() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cmd != nil && m.cmd.Process != nil {
		return m.cmd.Process.Pid
	}
	return 0
}

// Stats returns statistics about the managed process.
type Stats struct {
	Name         string        `json:"name"`
	Status       Status        `json:"status"`
	PID          int           `json:"pid,omitempty"`
	Uptime       time.Duration `json:"uptime,omitempty"`
	RestartCount int           `json:"restart_count"`
	LastError    string        `json:"last_error,omitempty"`
}

// Stats returns current statistics for the process.
func (m *Manager) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := Stats{
		Name:         m.config.Name,
		Status:       m.status,
		RestartCount: m.restartCount,
	}

	if m.cmd != nil && m.cmd.Process != nil {
		stats.PID = m.cmd.Process.Pid
	}

	if m.status == StatusRunning {
		stats.Uptime = time.Since(m.startTime)
	}

	if m.lastError != nil {
		stats.LastError = m.lastError.Error()
	}

	return stats
}
