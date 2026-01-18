---
title: Implementation Roadmap (Year 1)
version: 1.0.0
status: active
last_updated: 2026-01-18
depends_on:
  - DEVELOPMENT-STRATEGY.md
  - CODING-STANDARDS.md
---

# Year 1 Implementation Roadmap: Foundation

This document breaks down the Year 1 milestones defined in `DEVELOPMENT-STRATEGY.md` into actionable technical tasks.

---

## M1.1: Core Infrastructure Setup

**Goal**: Establish the Go project, database, and message bus.

### 1.1.1 Project Structure
- [ ] Initialize Go module `github.com/nerrad567/gray-logic-stack`
- [ ] Create directory structure per `CODING-STANDARDS.md`:
  - `cmd/graylogic/` (Main entry point)
  - `internal/core/` (Core domain logic)
  - `internal/infrastructure/` (Adapters)
  - `pkg/` (Shared types)
- [ ] Configure `golangci-lint` and `Makefile`

### 1.1.2 Database (SQLite)
- [ ] Create `migrations/` directory
- [ ] Write `001_initial_schema.sql` based on `entities.md`:
  - `sites`, `areas`, `rooms`
  - `devices`, `capabilities`
  - `audit_logs`
- [ ] Implement `infrastructure/sqlite` package with migration runner

### 1.1.3 Message Bus (MQTT)
- [ ] Create `docker-compose.yml` for Mosquitto and InfluxDB
- [ ] Configure Mosquitto persistence (`mosquitto.conf`)
- [ ] Implement `infrastructure/mqtt` package:
  - Connection handling with auto-reconnect
  - Publish/Subscribe interface
  - Quality of Service (QoS 1) enforcement

### 1.1.4 Time-Series (InfluxDB)
- [ ] Configure InfluxDB bucket `graylogic`
- [ ] Implement `infrastructure/influx` package for telemetry

---

## M1.2: KNX Bridge

**Goal**: Communicate with physical KNX bus.

### 1.2.1 Bridge Architecture
- [ ] Design `internal/bridges/knx` package
- [ ] Create `knxd` connection handler (TCP/IP)
- [ ] Implement group address (GA) parser (1/2/3 format)

### 1.2.2 State Bridging
- [ ] Implement `KnxMonitor` service:
  - Listen to `knxd` events
  - Convert to internal `DeviceState`
  - Publish to `graylogic/state/knx/{ga}`

### 1.2.3 Command Bridging
- [ ] Implement `KnxCommander` service:
  - Subscribe to `graylogic/command/knx/{ga}`
  - Write to `knxd`

---

## M1.3: Device Registry

**Goal**: Manage what exists in the system.

### 1.3.1 Data Access Layer
- [ ] Implement `internal/device/repository.go`:
  - `Create`, `Get`, `List`, `Update`, `Delete`
  - Capability management (JSON blob)

### 1.3.2 Device Service
- [ ] Implement `internal/device/service.go`:
  - Business logic for device management
  - Validation (device ID format, required fields)

### 1.3.3 Initial Population
- [ ] Create seed script to populate test devices (lab bench setup)

---

## M1.4: REST API + WebSocket

**Goal**: External control and real-time updates.

### 1.4.1 API Server
- [ ] Implement `internal/api/server.go`:
  - Router setup (chi or standard lib)
  - Middleware (logger, recovery, CORS)

### 1.4.2 REST Endpoints
- [ ] `GET /api/v1/devices`
- [ ] `GET /api/v1/devices/{id}`
- [ ] `POST /api/v1/devices/{id}/command`

### 1.4.3 WebSocket Hub
- [ ] Implement `internal/api/websocket.go`:
  - Client connection handling
  - Broadcast loop for state changes
  - Ticket-based authentication (M3 requirement, basic for now)

### 1.4.4 Security
- [ ] Configure TLS (self-signed for dev)
- [ ] Implement basic authentication placeholder

---

## M1.5: Flutter Wall Panel

**Goal**: Visual control.

### 1.5.1 Project Setup
- [ ] Initialize Flutter project `ui/wallpanel`
- [ ] Configure `dio` for HTTP and `web_socket_channel`

### 1.5.2 Data Layer
- [ ] Implement repository pattern for Devices
- [ ] Create device models (matching Go structs)

### 1.5.3 UI Components
- [ ] Create `RoomView` widget
- [ ] Create `DeviceTile` widget (dynamic based on capability)
  - `SwitchTile`: On/Off
  - `DimmerTile`: Slider

---

## M1.6: Basic Scenes

**Goal**: Coordinated control.

### 1.6.1 Scene Data Model
- [ ] Add `scenes` and `scene_actions` tables
- [ ] Define `Scene` struct

### 1.6.2 Scene Engine
- [ ] Implement `internal/automation/scene_engine.go`:
  - `Activate(sceneID)`
  - Iterate actions -> publish commands
  - Parallel execution logic

### 1.6.3 API
- [ ] `GET /api/v1/scenes`
- [ ] `POST /api/v1/scenes/{id}/activate`
