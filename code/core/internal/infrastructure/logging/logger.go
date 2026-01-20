package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
)

// Logger wraps slog.Logger with Gray Logic-specific functionality.
//
// It provides structured logging with default fields and level-based filtering.
//
// Thread Safety:
//   - All methods are safe for concurrent use from multiple goroutines.
type Logger struct {
	*slog.Logger
}

// New creates a new Logger with the specified configuration.
//
// It configures:
//   - Output format (JSON for production, text for development)
//   - Log level filtering
//   - Default fields (service name, version)
//   - Output destination
//
// Parameters:
//   - cfg: Logging configuration from config.yaml
//   - version: Application version for default field
//
// Returns:
//   - *Logger: Configured logger ready for use
func New(cfg config.LoggingConfig, version string) *Logger {
	// Determine output writer
	var output io.Writer
	switch strings.ToLower(cfg.Output) {
	case "stderr":
		output = os.Stderr
	default:
		output = os.Stdout
	}

	// Parse log level
	level := parseLevel(cfg.Level)

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	switch strings.ToLower(cfg.Format) {
	case "text":
		handler = slog.NewTextHandler(output, opts)
	default:
		handler = slog.NewJSONHandler(output, opts)
	}

	// Add default fields
	handler = handler.WithAttrs([]slog.Attr{
		slog.String("service", "graylogic"),
		slog.String("version", version),
	})

	return &Logger{
		Logger: slog.New(handler),
	}
}

// parseLevel converts a string log level to slog.Level.
//
// Supported levels: debug, info, warn, error
// Defaults to info if unrecognised.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// With returns a new Logger with additional default attributes.
//
// Parameters:
//   - args: Key-value pairs to add as default attributes
//
// Returns:
//   - *Logger: New logger with added attributes
//
// Example:
//
//	mqttLogger := logger.With("component", "mqtt")
//	mqttLogger.Info("connected") // Includes component=mqtt
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
	}
}

// Default creates a default logger for use before configuration is loaded.
//
// This logger outputs to stdout in JSON format at info level.
// It should only be used during early startup before config is available.
//
// Returns:
//   - *Logger: Default logger
func Default() *Logger {
	return New(config.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}, "dev")
}
