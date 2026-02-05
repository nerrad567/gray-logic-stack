---
description: Verify development environment is correctly set up
---

# Health Check

Verifies that your development environment has all required tools and services.

## Requirements Checked

| Tool | Required Version | Purpose |
|------|-----------------|---------|
| Go | 1.25+ | Core development |
| golangci-lint | 1.55+ | Code quality |
| Docker | 20.10+ | Dev services |
| Docker Compose | 2.0+ | Multi-container |
| git | 2.30+ | Version control |

## Services Checked

| Service | Port | Purpose |
|---------|------|---------|
| Mosquitto (MQTT) | 1883 | Message bus |
| VictoriaMetrics | 8428 | Time-series |

## Commands

```bash
# Tool versions
go version
golangci-lint --version
docker --version
docker compose version
git --version

# Check if Go compiles
cd code/core && go build ./...

# Check dev services
docker compose -f docker-compose.dev.yml ps

# Test MQTT connectivity
docker exec mosquitto mosquitto_pub -t test -m "ping" 2>/dev/null && echo "✓ MQTT OK"

# Test VictoriaMetrics connectivity
curl -s http://localhost:8428/ping && echo "✓ VictoriaMetrics OK"
```

## Quick Fix: Start Dev Services

If services aren't running:

```bash
# Start MQTT and VictoriaMetrics
docker compose -f docker-compose.dev.yml up -d

# Verify
docker compose -f docker-compose.dev.yml ps
```

## Example Output

```
=== Tool Versions ===
Go:             go1.23.0 ✓
golangci-lint:  1.55.2 ✓
Docker:         24.0.7 ✓
Docker Compose: v2.23.0 ✓
git:            2.43.0 ✓

=== Build Check ===
go build ./... ✓

=== Dev Services ===
mosquitto:      running (port 1883) ✓
tsdb:       running (port 8428) ✓

=== Connectivity ===
MQTT pub/sub:   working ✓
VictoriaMetrics ping:  working ✓

Environment healthy — ready to develop.
```

## Common Issues

| Issue | Fix |
|-------|-----|
| Go not found | Install from https://go.dev/dl/ |
| golangci-lint not found | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| Docker not running | `sudo systemctl start docker` |
| Services not started | `docker compose -f docker-compose.dev.yml up -d` |
| Port conflict | Check `lsof -i :1883` or `lsof -i :8428` |
