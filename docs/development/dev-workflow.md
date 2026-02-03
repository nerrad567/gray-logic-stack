---
title: Development Workflow
version: 1.0.0
status: current
dependencies:
  - docs/development/CODING-STANDARDS.md
  - docs/development/environment.md
---

# Development Workflow

> **Canonical reference for daily Gray Logic Core development.**
> Dev mode is the default — Docker is only for support services.

## Overview

```
┌─────────────────────────────────────────────────────────────┐
│  DEV MODE (default)                                         │
│                                                             │
│  Host machine:                                              │
│    Go core binary ──► localhost:8090 (API + Panel)          │
│         │                                                   │
│         ├── MQTT ─────► localhost:1883  (Docker: mosquitto)  │
│         ├── InfluxDB ─► localhost:8086  (Docker: influxdb)   │
│         └── KNX sim ──► localhost:3671  (Docker: knxsim)     │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  PROD MODE (explicit request only)                          │
│                                                             │
│  Docker network (graylogic):                                │
│    graylogic ──► mosquitto:1883                              │
│              ──► influxdb:8086                                │
│              ──► knxsim:3671                                  │
│    All containers communicate via Docker DNS                 │
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

- **Go 1.22+** with CGO enabled (required for SQLite)
- **knxd** — KNX daemon, managed as a subprocess by Gray Logic Core
- **Docker + Docker Compose** (for support services)
- **Flutter 3.x** (for wall panel UI)

Install knxd (Ubuntu/Debian):
```bash
sudo apt-get install -y knxd
sudo systemctl stop knxd knxd.socket
sudo systemctl disable knxd knxd.socket   # Gray Logic manages knxd as a subprocess
```

Install Flutter (standalone SDK — no snap):
```bash
# Download latest stable from https://docs.flutter.dev/install/archive
curl -Lo /tmp/flutter.tar.xz https://storage.googleapis.com/flutter_infra_release/releases/stable/linux/flutter_linux_<VERSION>-stable.tar.xz
sudo tar xf /tmp/flutter.tar.xz -C /opt/
sudo chown -R $USER:$USER /opt/flutter
echo 'export PATH="/opt/flutter/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
flutter --disable-analytics
```

Verify:
```bash
go version               # go1.22+
which knxd               # /usr/bin/knxd
flutter --version        # Flutter 3.x
docker compose version   # v2.x
CGO_ENABLED=1 go env CGO_ENABLED  # 1
```

## Quick Start (60 seconds)

```bash
# 1. Start support services
cd code/core && make dev-services

# 2. Build and run
make dev-run

# 3. Open in browser
#    API:   http://localhost:8090/api/v1/health
#    Panel: http://localhost:8090/panel/

# 4. When done
make dev-services-down
```

## Support Services

| Service    | Port      | Protocol | Purpose                    |
|------------|-----------|----------|----------------------------|
| mosquitto  | 1883      | MQTT     | Internal message bus       |
| influxdb   | 8086      | HTTP     | Time-series (PHM, energy)  |
| knxsim     | 3671/udp  | KNXnet/IP| KNX gateway simulator      |
| knxsim     | 9090      | HTTP     | Simulator web UI           |

All ports are bound to `127.0.0.1` (localhost only).

### InfluxDB Credentials (dev only)

| Field    | Value               |
|----------|---------------------|
| Username | graylogic           |
| Password | dev-password-only   |
| Org      | graylogic           |
| Bucket   | graylogic           |

## Native Go Core Workflow

### Standard (recommended)

```bash
make dev-run       # fmt + vet + build + run (~3s total)
```

### Fastest iteration

```bash
make dev-run-quick # build + run only (~2s)
```

### Just build (no run)

```bash
make build         # produces build/graylogic
```

### Environment variables

The config file (`configs/config.yaml`) defaults to `localhost` for all services.
Override at runtime with environment variables:

| Variable                    | Default     | Purpose                      |
|-----------------------------|-------------|------------------------------|
| `GRAYLOGIC_MQTT_HOST`      | localhost   | MQTT broker hostname         |
| `GRAYLOGIC_KNXD_BACKEND_HOST` | (from config) | KNX gateway host         |
| `GRAYLOGIC_PANEL_DIR`      | (empty)     | Filesystem dir for panel UI  |
| `GRAYLOGIC_DATABASE_PATH`  | ./data/graylogic.db | SQLite database path  |

## Flutter Panel Development

The wall panel is a Flutter web app. In dev mode, it's served from the filesystem
so you don't need to recompile the Go binary after Flutter changes.

### Build and copy

```bash
cd code/ui/wallpanel
flutter build web --release --base-href="/panel/"
cp -r build/web ../../core/internal/panel/web
```

### How it works

1. `make dev-run` sets `GRAYLOGIC_PANEL_DIR=./internal/panel/web`
2. The panel handler checks if the directory exists
3. If yes → serves files from filesystem (dev mode)
4. If no → serves from embedded `go:embed` assets (production)
5. SPA fallback works identically in both modes

After rebuilding Flutter, just refresh the browser — no Go restart needed.

## Production Docker Workflow

Only use this when explicitly requested or for CI testing.

```bash
# Build and start full stack
make docker-up

# Stop
make docker-down
```

This uses `docker-compose.prod.yml` which:
- Builds the Go binary inside Docker (multi-stage)
- Runs all 4 services on a Docker bridge network
- Overrides `GRAYLOGIC_MQTT_HOST=mosquitto` for container DNS
- Embeds the Flutter panel into the Go binary

## Troubleshooting

### CGO / SQLite build errors

Gray Logic uses SQLite via `go-sqlite3`, which requires CGO:
```bash
CGO_ENABLED=1 go build ./cmd/graylogic
```

On some systems you may need `gcc` or `build-essential` installed.

### Port conflicts

If port 1883 or 8086 is already in use:
```bash
# Check what's using the port
ss -tlnp | grep 1883
# Stop the conflicting service, or change the port in docker-compose.dev.yml
```

### knxsim UDP port (3671)

The KNX simulator uses UDP port 3671. Installing `knxd` via apt enables it as a
system service by default, which binds this port and conflicts with knxsim.
Disable both the service and its socket activation unit:

```bash
sudo systemctl stop knxd knxd.socket
sudo systemctl disable knxd knxd.socket
```

Gray Logic Core manages knxd as a subprocess — the system service is not needed.

### MQTT connection refused

Ensure support services are running:
```bash
make dev-services
docker compose -f ../../docker-compose.dev.yml ps
```

The Go core expects MQTT on `localhost:1883` by default.
