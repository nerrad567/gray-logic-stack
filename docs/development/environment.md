---
title: Development Environment
version: 1.0.0
status: active
last_updated: 2026-01-19
depends_on:
  - development/DEVELOPMENT-STRATEGY.md
  - operations/infrastructure.md
---

# Development Environment

This document defines how developers run Gray Logic Core locally during active development, balancing fast iteration with safety.

---

## Philosophy

1. **Fast Iteration** — No Docker rebuild for code changes
2. **Real Dependencies** — Use real MQTT broker and InfluxDB (containerized)
3. **Safety Isolation** — Prevent catastrophic operations from affecting the host system
4. **Production Parity** — Same dependency versions as production

---

## Environment Options

### Option A: Restricted User (Recommended for Active Development)

Run Gray Logic Core natively with a dedicated low-privilege user to isolate data and prevent accidental system damage.

```bash
# One-time setup
sudo useradd -m -s /bin/bash graylogic-dev
sudo mkdir -p /opt/graylogic-dev/{data,config}
sudo chown -R graylogic-dev:graylogic-dev /opt/graylogic-dev

# Grant current user ability to switch to this account
sudo usermod -aG graylogic-dev $USER
```

**Directory Structure:**
```
/opt/graylogic-dev/
├── data/
│   └── graylogic.db          # SQLite database (dev copy)
├── config/
│   ├── config.yaml           # Dev configuration
│   └── secrets.yaml          # Dev secrets (can be dummy values)
└── logs/                     # Optional: separate log location
```

**Running Core:**
```bash
# From project directory
go build -o ./bin/graylogic ./cmd/graylogic

# Run as restricted user
sudo -u graylogic-dev ./bin/graylogic \
  --config /opt/graylogic-dev/config/config.yaml \
  --data-dir /opt/graylogic-dev/data
```

**Benefits:**
- Fast: No container rebuild
- Isolated: Can't touch `/home/darren` or system files
- Easy reset: `rm -rf /opt/graylogic-dev/data/*`

**Limitations:**
- User must exist on system
- Some Go tooling (debugger) may need adjustment

---

### Option B: Containerized Dependencies Only

Run infrastructure in Docker, run Core natively.

```yaml
# docker-compose.dev.yml
version: "3.8"

services:
  mqtt:
    image: eclipse-mosquitto:2.0.18
    container_name: graylogic-dev-mqtt
    ports:
      - "1883:1883"
      - "9001:9001"  # WebSocket (optional)
    volumes:
      - ./dev/mosquitto/mosquitto.conf:/mosquitto/config/mosquitto.conf
      - graylogic-dev-mqtt-data:/mosquitto/data
    restart: unless-stopped

  influxdb:
    image: influxdb:2.7-alpine
    container_name: graylogic-dev-influxdb
    ports:
      - "8086:8086"
    environment:
      DOCKER_INFLUXDB_INIT_MODE: setup
      DOCKER_INFLUXDB_INIT_USERNAME: graylogic
      DOCKER_INFLUXDB_INIT_PASSWORD: devpassword123
      DOCKER_INFLUXDB_INIT_ORG: graylogic
      DOCKER_INFLUXDB_INIT_BUCKET: phm
      DOCKER_INFLUXDB_INIT_ADMIN_TOKEN: dev-token-do-not-use-in-prod
    volumes:
      - graylogic-dev-influxdb-data:/var/lib/influxdb2
    restart: unless-stopped

volumes:
  graylogic-dev-mqtt-data:
  graylogic-dev-influxdb-data:
```

**Mosquitto configuration for dev:**
```conf
# dev/mosquitto/mosquitto.conf
listener 1883
allow_anonymous true
persistence true
persistence_location /mosquitto/data/
```

**Workflow:**
```bash
# Start dependencies (once per session)
docker compose -f docker-compose.dev.yml up -d

# Run Core natively (fast iteration)
go run ./cmd/graylogic --config ./dev/config.yaml

# Stop dependencies
docker compose -f docker-compose.dev.yml down
```

---

### Option C: Full Container Stack (CI/Production Testing)

All components in containers. Use when testing Docker deployment specifically.

```yaml
# docker-compose.full.yml
# ... (same as production with dev overrides)
```

Use sparingly — only for testing containerization itself.

---

## Recommended Workflow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Development Workflow                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Daily Development:                                              │
│    1. docker compose -f docker-compose.dev.yml up -d             │
│    2. go run ./cmd/graylogic --config ./dev/config.yaml          │
│    3. Make changes, Ctrl+C, repeat                               │
│                                                                  │
│  Testing New Package:                                            │
│    1. Write unit tests (no containers needed)                    │
│    2. Run: go test ./internal/...                                │
│                                                                  │
│  Integration Testing:                                            │
│    1. Ensure containers running                                  │
│    2. Run: go test -tags=integration ./...                       │
│                                                                  │
│  Pre-Commit:                                                     │
│    1. Full test suite                                            │
│    2. Lint: golangci-lint run                                    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Safety Considerations

### Why Isolation Matters

When developing with AI agents, there's risk of unintended destructive operations:
- Mass deletions
- Configuration corruption
- Resource exhaustion

### Mitigations

| Risk | Mitigation |
|------|------------|
| Delete production data | Dev data lives in `/opt/graylogic-dev/` not `/var/lib/` |
| Corrupt system configs | Restricted user can't write to `/etc/` |
| Runaway processes | Use `systemd-run --scope -p MemoryMax=1G` if needed |
| Network operations | Dev containers on isolated Docker network |

### Quick Reset

If development data becomes corrupted or needs reset:

```bash
# Reset database only
rm /opt/graylogic-dev/data/graylogic.db

# Reset all dev data
rm -rf /opt/graylogic-dev/data/*
docker compose -f docker-compose.dev.yml down -v
docker compose -f docker-compose.dev.yml up -d
```

---

## Configuration

### Development Config Template

```yaml
# dev/config.yaml
server:
  listen_address: "127.0.0.1:8080"
  environment: development

database:
  path: "/opt/graylogic-dev/data/graylogic.db"
  # OR for non-restricted user:
  # path: "./dev/data/graylogic.db"

mqtt:
  broker: "tcp://localhost:1883"
  client_id: "graylogic-core-dev"

influxdb:
  url: "http://localhost:8086"
  token: "dev-token-do-not-use-in-prod"
  org: "graylogic"
  bucket: "phm"

logging:
  level: debug
  format: text  # Easier to read than JSON in terminal
```

---

## Next Steps

Once we move toward production deployment testing:

1. Create `Dockerfile` for Core (see [containerization.md](containerization.md))
2. Create `docker-compose.prod.yml` with production settings
3. Document CI/CD pipeline for building and pushing images

---

## See Also

- [Infrastructure Strategy](../operations/infrastructure.md) — Production deployment
- [Testing Strategy](testing-strategy.md) — Test categories and execution
- [Coding Standards](CODING-STANDARDS.md) — Code quality requirements
