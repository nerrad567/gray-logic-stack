package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/nerrad567/gray-logic-core/internal/infrastructure/config"
)

func TestNew_JSONFormat(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	logger := New(cfg, "1.0.0")

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNew_TextFormat(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "debug",
		Format: "text",
		Output: "stderr",
	}

	logger := New(cfg, "1.0.0")

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected slog.Level
	}{
		{
			name:     "debug level",
			input:    "debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "info level",
			input:    "info",
			expected: slog.LevelInfo,
		},
		{
			name:     "warn level",
			input:    "warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "warning level",
			input:    "warning",
			expected: slog.LevelWarn,
		},
		{
			name:     "error level",
			input:    "error",
			expected: slog.LevelError,
		},
		{
			name:     "unknown defaults to info",
			input:    "unknown",
			expected: slog.LevelInfo,
		},
		{
			name:     "empty defaults to info",
			input:    "",
			expected: slog.LevelInfo,
		},
		{
			name:     "case insensitive",
			input:    "DEBUG",
			expected: slog.LevelDebug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLogger_With(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	logger := New(cfg, "1.0.0")
	childLogger := logger.With("component", "mqtt")

	if childLogger == nil {
		t.Fatal("expected non-nil child logger")
	}

	if childLogger == logger {
		t.Error("expected child logger to be different from parent")
	}
}

func TestDefault(t *testing.T) {
	logger := Default()

	if logger == nil {
		t.Fatal("expected non-nil default logger")
	}
}

func TestLogger_OutputContainsDefaultFields(t *testing.T) {
	var buf bytes.Buffer

	baseHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	handler := baseHandler.WithAttrs([]slog.Attr{
		slog.String("service", "graylogic"),
		slog.String("version", "test"),
	})

	logger := &Logger{Logger: slog.New(handler)}
	logger.Info("test message", "key", "value")

	output := buf.String()

	if !strings.Contains(output, "graylogic") {
		t.Error("expected output to contain service field")
	}

	if !strings.Contains(output, "test") {
		t.Error("expected output to contain version field")
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if logEntry["msg"] != "test message" {
		t.Errorf("expected msg='test message', got %v", logEntry["msg"])
	}

	if logEntry["key"] != "value" {
		t.Errorf("expected key='value', got %v", logEntry["key"])
	}
}
