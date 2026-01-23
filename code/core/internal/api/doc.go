// Package api implements the HTTP REST API and WebSocket server for Gray Logic Core.
//
// This package provides:
//   - REST endpoints for device CRUD, state management, and commands
//   - WebSocket hub for real-time state change broadcasts
//   - JWT authentication with ticket-based WebSocket auth
//   - Middleware stack (request ID, logging, recovery, CORS)
//   - TLS support for production deployments
//
// # Architecture
//
// The API server sits between user interfaces (Flutter wall panels, mobile apps,
// web admin) and the device registry + MQTT bus. Commands flow from the API to
// protocol bridges via MQTT, and state changes flow back via MQTT subscriptions
// which are broadcast to WebSocket clients.
//
// # Security
//
// Authentication uses JWT tokens (placeholder dev credentials in M1.4).
// WebSocket connections use single-use tickets to prevent token leakage in URLs.
// Full auth with RBAC is deferred to M1.5.
//
// # Graceful Degradation
//
// The server operates without MQTT — reads and WebSocket connections work,
// only device commands fail. This enables testing and partial operation.
//
// See docs/interfaces/api.md for the full API specification.
package api
