// Package logging provides structured logging for Gray Logic Core.
//
// This package wraps Go's standard log/slog package to provide
// consistent, structured logging across the entire application.
//
// # Features
//
//   - JSON output for production (machine-parsable)
//   - Text output for development (human-readable)
//   - Default fields (service, version) on all log entries
//   - Level-based filtering (debug, info, warn, error)
//   - Thread-safe for concurrent use
//
// # Configuration
//
// Logging is configured via the LoggingConfig in config.yaml:
//
//	logging:
//	  level: "info"      # debug, info, warn, error
//	  format: "json"     # json, text
//	  output: "stdout"   # stdout, stderr
//
// # Usage
//
//	logger := logging.New(cfg.Logging, "1.0.0")
//	logger.Info("starting service", "port", 8080)
//	logger.Error("failed to connect", "error", err)
//
// # Security
//
// Never log secrets, tokens, passwords, or API keys.
// Use field redaction for sensitive data:
//
//	logger.Info("API key used", "key_prefix", key[:8]+"...")
//
// See docs/development/CODING-STANDARDS.md for logging guidelines.
package logging
