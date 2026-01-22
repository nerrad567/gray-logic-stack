package process

import (
	"context"
	"testing"
	"time"
)

func TestNewManager_Defaults(t *testing.T) {
	cfg := Config{
		Name:   "test-proc",
		Binary: "/usr/bin/test",
		Args:   []string{"--flag"},
	}

	m := NewManager(cfg)

	if m.config.Name != "test-proc" {
		t.Errorf("Name = %q, want %q", m.config.Name, "test-proc")
	}
	if m.config.Binary != "/usr/bin/test" {
		t.Errorf("Binary = %q, want %q", m.config.Binary, "/usr/bin/test")
	}
	if m.config.RestartDelay != 5*time.Second {
		t.Errorf("RestartDelay = %v, want %v", m.config.RestartDelay, 5*time.Second)
	}
	if m.config.MaxRestartDelay != 5*time.Minute {
		t.Errorf("MaxRestartDelay = %v, want %v", m.config.MaxRestartDelay, 5*time.Minute)
	}
	if m.config.StableThreshold != 2*time.Minute {
		t.Errorf("StableThreshold = %v, want %v", m.config.StableThreshold, 2*time.Minute)
	}
	if m.config.GracefulTimeout != 10*time.Second {
		t.Errorf("GracefulTimeout = %v, want %v", m.config.GracefulTimeout, 10*time.Second)
	}
	if m.config.HealthCheckInterval != 30*time.Second {
		t.Errorf("HealthCheckInterval = %v, want %v", m.config.HealthCheckInterval, 30*time.Second)
	}
}

func TestNewManager_CustomConfig(t *testing.T) {
	cfg := Config{
		Name:                "custom",
		Binary:              "/opt/bin/daemon",
		Args:                []string{"-v", "--port=8080"},
		RestartDelay:        10 * time.Second,
		MaxRestartDelay:     10 * time.Minute,
		StableThreshold:     5 * time.Minute,
		GracefulTimeout:     30 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxRestartAttempts:  20,
	}

	m := NewManager(cfg)

	if m.config.RestartDelay != 10*time.Second {
		t.Errorf("RestartDelay = %v, want %v", m.config.RestartDelay, 10*time.Second)
	}
	if m.config.MaxRestartDelay != 10*time.Minute {
		t.Errorf("MaxRestartDelay = %v, want %v", m.config.MaxRestartDelay, 10*time.Minute)
	}
	if m.config.MaxRestartAttempts != 20 {
		t.Errorf("MaxRestartAttempts = %d, want 20", m.config.MaxRestartAttempts)
	}
}

func TestDefaultConfig_Function(t *testing.T) {
	cfg := DefaultConfig("myproc", "/usr/bin/myproc", []string{"--daemon"})

	if cfg.Name != "myproc" {
		t.Errorf("Name = %q, want %q", cfg.Name, "myproc")
	}
	if cfg.Binary != "/usr/bin/myproc" {
		t.Errorf("Binary = %q, want %q", cfg.Binary, "/usr/bin/myproc")
	}
	if len(cfg.Args) != 1 || cfg.Args[0] != "--daemon" {
		t.Errorf("Args = %v, want [--daemon]", cfg.Args)
	}
	if !cfg.RestartOnFailure {
		t.Error("RestartOnFailure = false, want true")
	}
	if cfg.MaxRestartAttempts != 10 {
		t.Errorf("MaxRestartAttempts = %d, want 10", cfg.MaxRestartAttempts)
	}
}

func TestManager_InitialState(t *testing.T) {
	m := NewManager(Config{
		Name:   "test",
		Binary: "/bin/true",
	})

	if m.Status() != StatusStopped {
		t.Errorf("initial Status() = %q, want %q", m.Status(), StatusStopped)
	}
	if m.IsRunning() {
		t.Error("IsRunning() = true, want false")
	}
	if m.PID() != 0 {
		t.Errorf("PID() = %d, want 0", m.PID())
	}
	if m.RestartCount() != 0 {
		t.Errorf("RestartCount() = %d, want 0", m.RestartCount())
	}
	if m.Uptime() != 0 {
		t.Errorf("Uptime() = %v, want 0", m.Uptime())
	}
	if m.LastError() != nil {
		t.Errorf("LastError() = %v, want nil", m.LastError())
	}
}

func TestManager_Stats(t *testing.T) {
	m := NewManager(Config{
		Name:   "stats-test",
		Binary: "/bin/echo",
	})

	stats := m.Stats()
	if stats.Name != "stats-test" {
		t.Errorf("Stats.Name = %q, want %q", stats.Name, "stats-test")
	}
	if stats.Status != StatusStopped {
		t.Errorf("Stats.Status = %q, want %q", stats.Status, StatusStopped)
	}
	if stats.PID != 0 {
		t.Errorf("Stats.PID = %d, want 0", stats.PID)
	}
	if stats.RestartCount != 0 {
		t.Errorf("Stats.RestartCount = %d, want 0", stats.RestartCount)
	}
	if stats.LastError != "" {
		t.Errorf("Stats.LastError = %q, want empty", stats.LastError)
	}
}

func TestManager_StopWhenNotRunning(t *testing.T) {
	m := NewManager(Config{
		Name:   "test",
		Binary: "/bin/true",
	})

	// Stopping a non-running process should be a no-op
	if err := m.Stop(); err != nil {
		t.Errorf("Stop() on stopped process error = %v, want nil", err)
	}
}

func TestManager_StartAlreadyRunning(t *testing.T) {
	m := NewManager(Config{
		Name:   "test",
		Binary: "/bin/sleep",
		Args:   []string{"10"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the process
	if err := m.Start(ctx); err != nil {
		t.Fatalf("first Start() error: %v", err)
	}
	defer m.Stop()

	// Starting again should fail
	err := m.Start(ctx)
	if err == nil {
		t.Error("second Start() expected error, got nil")
	}
}

func TestManager_StartAndStop(t *testing.T) {
	m := NewManager(Config{
		Name:            "test-sleep",
		Binary:          "/bin/sleep",
		Args:            []string{"60"},
		GracefulTimeout: 2 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Verify running state
	if !m.IsRunning() {
		t.Error("IsRunning() = false after Start()")
	}
	if m.PID() == 0 {
		t.Error("PID() = 0 after Start()")
	}
	if m.Status() != StatusRunning {
		t.Errorf("Status() = %q, want %q", m.Status(), StatusRunning)
	}

	// Stop the process
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}

	// Give the monitor goroutine time to update state
	time.Sleep(100 * time.Millisecond)

	if m.IsRunning() {
		t.Error("IsRunning() = true after Stop()")
	}
}

func TestManager_StartWithInvalidBinary(t *testing.T) {
	m := NewManager(Config{
		Name:   "bad-binary",
		Binary: "/nonexistent/binary",
	})

	ctx := context.Background()
	err := m.Start(ctx)
	if err == nil {
		t.Fatal("Start() with invalid binary expected error, got nil")
	}

	if m.Status() != StatusFailed {
		t.Errorf("Status() = %q, want %q", m.Status(), StatusFailed)
	}
}

func TestManager_SetLogger(t *testing.T) {
	m := NewManager(Config{
		Name:   "test",
		Binary: "/bin/true",
	})

	// Should not panic
	m.SetLogger(noopLogger{})
}

func TestCalculateBackoffDelay(t *testing.T) {
	m := NewManager(Config{
		Name:            "test",
		Binary:          "/bin/true",
		RestartDelay:    1 * time.Second,
		MaxRestartDelay: 30 * time.Second,
	})

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 1 * time.Second},  // First attempt: base delay
		{2, 2 * time.Second},  // 2nd: 1s * 2
		{3, 4 * time.Second},  // 3rd: 1s * 4
		{4, 8 * time.Second},  // 4th: 1s * 8
		{5, 16 * time.Second}, // 5th: 1s * 16
		{6, 30 * time.Second}, // 6th: capped at max
		{7, 30 * time.Second}, // 7th: stays at max
	}

	for _, tt := range tests {
		got := m.calculateBackoffDelay(tt.attempt)
		if got != tt.want {
			t.Errorf("calculateBackoffDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestIsRecoverable(t *testing.T) {
	t.Run("nil error is recoverable", func(t *testing.T) {
		if !IsRecoverable(nil) {
			t.Error("IsRecoverable(nil) = false, want true")
		}
	})

	t.Run("plain error is recoverable", func(t *testing.T) {
		err := context.DeadlineExceeded
		if !IsRecoverable(err) {
			t.Error("plain error should be recoverable by default")
		}
	})

	t.Run("recoverable error interface", func(t *testing.T) {
		err := &testRecoverableError{recoverable: true}
		if !IsRecoverable(err) {
			t.Error("recoverable error should return true")
		}
	})

	t.Run("non-recoverable error interface", func(t *testing.T) {
		err := &testRecoverableError{recoverable: false}
		if IsRecoverable(err) {
			t.Error("non-recoverable error should return false")
		}
	})
}

// testRecoverableError implements RecoverableError for testing.
type testRecoverableError struct {
	recoverable bool
}

func (e *testRecoverableError) Error() string       { return "test error" }
func (e *testRecoverableError) IsRecoverable() bool { return e.recoverable }

func TestManager_OnStartCallback(t *testing.T) {
	started := false
	m := NewManager(Config{
		Name:   "callback-test",
		Binary: "/bin/sleep",
		Args:   []string{"60"},
		OnStart: func() {
			started = true
		},
		GracefulTimeout: 2 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer m.Stop()

	if !started {
		t.Error("OnStart callback was not called")
	}
}
