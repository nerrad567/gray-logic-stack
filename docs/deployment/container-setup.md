---
title: Container Setup
version: 1.0.0
status: active
dependencies:
  - deployment/usb-device-management.md
  - deployment/residential.md
---

# Container Setup

This document defines the container image requirements for Gray Logic Core deployment.

## Base Image

Gray Logic Core uses a minimal Debian-based image for stability and long-term support.

```dockerfile
FROM debian:bookworm-slim AS base
```

**Rationale:**
- Debian Bookworm has LTS support until 2028
- Slim variant minimises attack surface
- Compatible with ARM64 (Raspberry Pi) and AMD64

## Required Packages

### Core Dependencies

| Package | Version | Purpose | Required |
|---------|---------|---------|----------|
| `ca-certificates` | latest | TLS certificate validation | Yes |
| `tzdata` | latest | Timezone support for schedules | Yes |

### KNX/USB Support

| Package | Version | Purpose | Required |
|---------|---------|---------|----------|
| `knxd` | 0.14+ | KNX daemon for bus communication | Yes (if managed) |
| `usbutils` | latest | USB reset utility (`usbreset`) | Yes (for USB backend) |
| `libusb-1.0-0` | latest | USB library for knxd | Yes (for USB backend) |

### Optional Packages

| Package | Purpose | When Needed |
|---------|---------|-------------|
| `mosquitto-clients` | MQTT debugging | Development only |
| `curl` | Health check probes | If using HTTP probes |
| `jq` | JSON parsing | Scripting/debugging |

## Dockerfile

```dockerfile
# Gray Logic Core Container Image
# ================================
# Multi-stage build for minimal production image

# ------------------------------------------------------------------------------
# Stage 1: Builder
# ------------------------------------------------------------------------------
FROM golang:1.22-bookworm AS builder

WORKDIR /build

# Copy go mod files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VERSION:-dev}" \
    -o graylogic \
    ./cmd/graylogic

# ------------------------------------------------------------------------------
# Stage 2: Production Image
# ------------------------------------------------------------------------------
FROM debian:bookworm-slim AS production

LABEL org.opencontainers.image.title="Gray Logic Core"
LABEL org.opencontainers.image.description="Building intelligence platform - automation core"
LABEL org.opencontainers.image.vendor="Gray Logic"
LABEL org.opencontainers.image.source="https://github.com/graylogic/core"

# Install runtime dependencies
# CRITICAL: usbutils provides usbreset for USB device recovery
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    knxd \
    usbutils \
    libusb-1.0-0 \
    && rm -rf /var/lib/apt/lists/* \
    && rm -rf /var/cache/apt/*

# Verify critical utilities are installed
RUN which usbreset && echo "usbreset: OK" \
    && which knxd && echo "knxd: OK"

# Create non-root user
RUN groupadd -r -g 1000 graylogic \
    && useradd -r -u 1000 -g graylogic -d /app -s /sbin/nologin graylogic

# Create application directories
RUN mkdir -p /app/bin /app/configs /app/data /app/logs \
    && chown -R graylogic:graylogic /app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/graylogic /app/bin/graylogic

# Copy default configs (can be overridden via volume mount)
COPY configs/config.yaml /app/configs/config.yaml.default
COPY configs/knx-bridge.yaml /app/configs/knx-bridge.yaml.default

# Set ownership
RUN chown -R graylogic:graylogic /app

# Switch to non-root user
USER graylogic

# Expose ports
EXPOSE 8080/tcp   # REST API
EXPOSE 8081/tcp   # WebSocket

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD ["/app/bin/graylogic", "health"] || exit 1

# Default command
ENTRYPOINT ["/app/bin/graylogic"]
CMD ["--config", "/app/configs/config.yaml"]
```

## Docker Compose

### Development

```yaml
# docker-compose.dev.yml
version: '3.8'

services:
  # Development MQTT broker
  mosquitto:
    image: eclipse-mosquitto:2
    container_name: graylogic-mqtt
    ports:
      - "1883:1883"
    volumes:
      - ./docker/mosquitto/mosquitto.conf:/mosquitto/config/mosquitto.conf:ro
      - mosquitto-data:/mosquitto/data

  # VictoriaMetrics for time-series (optional)
  tsdb:
    image: victoriametrics/victoria-metrics:v1.135.0
    container_name: graylogic-victoriametrics
    ports:
      - "8428:8428"
    environment:
    volumes:
      - victoriametrics-data:/victoria-metrics-data

volumes:
  mosquitto-data:
  victoriametrics-data:
```

### Production

```yaml
# docker-compose.yml
version: '3.8'

services:
  graylogic-core:
    image: graylogic/core:${VERSION:-latest}
    container_name: graylogic-core
    restart: unless-stopped

    # USB device access for KNX
    devices:
      - /dev/bus/usb:/dev/bus/usb
    device_cgroup_rules:
      - 'c 189:* rmw'  # USB devices (major number 189)

    # Map to host graylogic group for USB access
    group_add:
      - "${GRAYLOGIC_GID:-1000}"

    ports:
      - "8080:8080"  # REST API
      - "8081:8081"  # WebSocket

    volumes:
      - ./configs:/app/configs:ro
      - ./data:/app/data
      - ./logs:/app/logs

    environment:
      - TZ=${TZ:-Europe/London}
      - GRAYLOGIC_DATABASE_PATH=/app/data/graylogic.db
      - GRAYLOGIC_TSDB_URL=${TSDB_TOKEN:-}

    depends_on:
      - mosquitto

    healthcheck:
      test: ["CMD", "/app/bin/graylogic", "health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

  mosquitto:
    image: eclipse-mosquitto:2
    container_name: graylogic-mqtt
    restart: unless-stopped
    ports:
      - "1883:1883"
    volumes:
      - ./docker/mosquitto/mosquitto.conf:/mosquitto/config/mosquitto.conf:ro
      - mosquitto-data:/mosquitto/data

volumes:
  mosquitto-data:
```

### Environment File

```bash
# .env
VERSION=1.0.0
TZ=Europe/London
GRAYLOGIC_GID=1000
TSDB_TOKEN=your-token-here
```

## Build Commands

```bash
# Build image
docker build -t graylogic/core:latest .

# Build with version tag
docker build -t graylogic/core:1.0.0 \
  --build-arg VERSION=1.0.0 .

# Multi-platform build (AMD64 + ARM64)
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t graylogic/core:latest \
  --push .
```

## Deployment Commands

```bash
# Start services
docker compose up -d

# View logs
docker compose logs -f graylogic-core

# Restart after config change
docker compose restart graylogic-core

# Stop all services
docker compose down

# Update to new version
docker compose pull
docker compose up -d
```

## Host Preparation Checklist

Before running Gray Logic in a container:

### USB/KNX Setup
- [ ] USB device connected and visible (`lsusb -d 0e77:0104`)
- [ ] udev rules installed (see [USB Device Management](./usb-device-management.md))
- [ ] graylogic group exists with correct GID
- [ ] System knxd.service disabled

### Docker Setup
- [ ] Docker or Podman installed
- [ ] User in docker group (if applicable)
- [ ] docker-compose installed (v2+)

### Network Setup
- [ ] Port 8080 available (or configure alternative)
- [ ] Port 1883 available for MQTT (or use external broker)
- [ ] Firewall rules configured

### Storage Setup
- [ ] configs/ directory with configuration files
- [ ] data/ directory for database (writable)
- [ ] logs/ directory for log files (writable)

## Verifying Installation

```bash
# Check container is running
docker ps | grep graylogic

# Check logs for successful startup
docker logs graylogic-core 2>&1 | grep -E "(started|ready|connected)"

# Expected output:
# "starting Gray Logic Core"
# "knxd ready"
# "connected to knxd"
# "KNX bridge started"
# "initialisation complete"

# Check USB device is accessible
docker exec graylogic-core lsusb -d 0e77:0104

# Check usbreset is available
docker exec graylogic-core which usbreset

# Test health endpoint
curl http://localhost:8080/health
```

## Troubleshooting

### Container Won't Start

```bash
# Check container logs
docker logs graylogic-core

# Common issues:
# - "permission denied" → Check volume permissions
# - "address already in use" → Port conflict
# - "no such file" → Missing config files
```

### USB Device Not Found

```bash
# Verify device on host
lsusb -d 0e77:0104

# Check device passthrough
docker inspect graylogic-core | jq '.[0].HostConfig.Devices'

# Ensure cgroup rules are set
docker inspect graylogic-core | jq '.[0].HostConfig.DeviceCgroupRules'
```

### knxd Fails to Start

```bash
# Check for existing knxd processes on host
pgrep -a knxd

# Check if system service is running
systemctl status knxd.service

# Manual USB reset
docker exec graylogic-core usbreset 0e77:0104
```

## Related Documents

- [USB Device Management](./usb-device-management.md) - Detailed USB setup guide
- [Residential Deployment](./residential.md) - Full residential installation guide
- [Commissioning Checklist](./commissioning-checklist.md) - Pre-handover checks
