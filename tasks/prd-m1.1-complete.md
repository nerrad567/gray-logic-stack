# PRD: Complete M1.1 Core Infrastructure

## Overview

Complete the remaining M1.1 milestone items to establish Gray Logic Core infrastructure.

## Current State

- ✅ Go module initialised
- ✅ Directory structure created
- ✅ Makefile with build automation
- ✅ golangci-lint configured
- ✅ Configuration system (YAML + env vars)
- ✅ SQLite database package with migrations
- ✅ MQTT client package
- ✅ InfluxDB client package
- ✅ Docker Compose for dev services

## Remaining Work

### Story 1: Fix MQTT Race Condition
**Priority**: P0 (blocking)
**Acceptance Criteria**:
- [ ] `go test -race ./internal/infrastructure/mqtt/...` passes
- [ ] OnConnect callback is thread-safe
- [ ] All MQTT tests pass with race detector

### Story 2: Wire main.go
**Priority**: P1
**Acceptance Criteria**:
- [ ] `cmd/graylogic/main.go` initialises config, database, MQTT, InfluxDB
- [ ] Graceful shutdown on SIGTERM/SIGINT
- [ ] `./bin/graylogic --version` prints version
- [ ] `./bin/graylogic --config config.yaml` loads config and connects

### Story 3: Add Health Endpoint
**Priority**: P2
**Acceptance Criteria**:
- [ ] HTTP server on configurable port
- [ ] `/health` endpoint returns JSON with component statuses
- [ ] Returns 200 if all components healthy, 503 if any unhealthy

### Story 4: Database Device Registry Schema
**Priority**: P2
**Acceptance Criteria**:
- [ ] Migration adds `devices` table per `docs/data-model/entities.md`
- [ ] Migration adds `areas` and `rooms` tables
- [ ] Migration is additive-only (no ALTER/DROP)
- [ ] Tests verify schema creation

### Story 5: Documentation Update
**Priority**: P3
**Acceptance Criteria**:
- [ ] PROJECT-STATUS.md updated with completed items
- [ ] GETTING-STARTED.md has accurate build/run instructions
- [ ] All package doc.go files are complete

## Constraints

- All code must pass `golangci-lint run`
- All tests must pass with `-race` flag
- No external HTTP calls in core code
- No cloud dependencies
- Follow patterns in `docs/development/CODING-STANDARDS.md`

## Branch Name

`m1.1-core-infrastructure`
