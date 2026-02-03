---
description: Nuke everything and rebuild from scratch — fresh virgin stack with all services online
---

# Nuke & Rebuild

Full scorched-earth teardown and rebuild. Kills all processes, removes all databases
and Docker volumes, recompiles everything from source, and brings up a fresh virgin
stack ready for testing.

**This is destructive by design.** All state, databases, and cached data will be lost.

## Phases

Execute each phase in order. Do NOT skip phases. Report progress after each phase.

### Phase 1: Kill — Terminate All Running Processes

Stop everything that might be running. Order matters: GLCore first, then knxd, then
Docker services.

```bash
# 1a. Kill any running GLCore process (nohup'd or otherwise)
pkill -f "build/graylogic" 2>/dev/null || true
sleep 1

# 1b. Kill any orphaned knxd managed by GLCore
#     (Match the socket path to avoid killing system knxd)
pkill -f "graylogic-knxd" 2>/dev/null || true

# 1c. Bring down Docker dev services
cd code/core && make dev-services-down

# 1d. Verify nothing remains
#     Ports 8090 (API), 6720 (knxd), 1883 (MQTT), 8086 (InfluxDB) should all be clear
ss -tlnp | grep -E '8090|6720|1883|8086|9090' || echo "All ports clear"
```

If any port is still occupied, identify and kill the process before proceeding.

### Phase 2: Purge — Remove All Data and Volumes

Delete every piece of persistent state for a true virgin install.

```bash
# 2a. Remove SQLite database and WAL files
rm -f code/core/data/graylogic.db \
      code/core/data/graylogic.db-shm \
      code/core/data/graylogic.db-wal

# 2b. Remove Docker volumes (MQTT data, InfluxDB data, KNX sim data)
docker volume rm gray-logic-stack_influxdb_data 2>/dev/null || true
docker volume rm gray-logic-stack_knxsim_data 2>/dev/null || true
docker volume rm gray-logic-stack_mosquitto_data 2>/dev/null || true
docker volume rm gray-logic-stack_mosquitto_log 2>/dev/null || true

# 2c. Remove Go build artifacts
cd code/core && make clean

# 2d. Remove knxd PID file
rm -f /tmp/graylogic-knxd.pid

# 2e. Remove stale GLCore log
rm -f /tmp/graylogic.log

# 2f. Verify clean slate
ls code/core/data/ 2>/dev/null || echo "data dir empty or absent"
docker volume ls | grep gray-logic || echo "No volumes remain"
```

### Phase 3: Rebuild — Compile Everything From Source

Build all components. Fail fast on any compilation error.

```bash
# 3a. Go dependencies
cd code/core && go mod tidy

# 3b. Build GLCore binary
cd code/core && make build

# 3c. Build Flutter wall panel (if flutter is available)
#     Skip gracefully if Flutter is not installed
if command -v flutter &>/dev/null; then
  cd code/ui/wallpanel && flutter pub get && flutter build web --release --base-href="/panel/" --no-tree-shake-icons
  cp -r build/web ../../core/internal/panel/web
  echo "Flutter panel built and deployed"
else
  echo "Flutter not installed — skipping panel build"
fi
```

If the Go build fails, STOP and report the error. Do not proceed to Phase 4.

### Phase 4: Launch — Bring Up Fresh Stack

Start services in dependency order: Docker services first, then GLCore.

```bash
# 4a. Start Docker dev services (mosquitto, influxdb, knxsim)
cd code/core && make dev-services

# 4b. Wait for services to be healthy
sleep 3

# 4c. Verify Docker services are up
docker compose -f ../../docker-compose.dev.yml ps

# 4d. Start GLCore as a background process that survives shell exit
cd code/core && nohup env GRAYLOGIC_PANEL_DIR=./internal/panel/web ./build/graylogic > /tmp/graylogic.log 2>&1 &
GLCORE_PID=$!
echo "GLCore started with PID: $GLCORE_PID"

# 4e. Wait for GLCore to initialise
sleep 3
```

**IMPORTANT**: GLCore is a long-running server. It MUST be launched with `nohup` and
backgrounded (`&`) so it is not killed when the shell session ends. Never run it with
a timeout — it will be killed when the timeout expires.

### Phase 5: Verify — Confirm Everything Is Healthy

Run health checks against every component.

```bash
# 5a. Check GLCore is still running
ps -p $GLCORE_PID -o pid,etime,rss,comm

# 5b. Check GLCore log for successful init
tail -3 /tmp/graylogic.log
# Should show: "initialisation complete, waiting for shutdown signal"

# 5c. Test API is responding
curl -s http://127.0.0.1:8090/api/v1/devices

# 5d. Check all expected ports are listening
ss -tlnp | grep -E '8090|6720|1883|8086|9090'

# 5e. Verify fresh database (0 devices, 0 scenes)
curl -s http://127.0.0.1:8090/api/v1/devices
curl -s http://127.0.0.1:8090/api/v1/scenes
```

## Success Criteria

ALL of the following must be true:

- [ ] No stale processes from previous runs
- [ ] All Docker volumes freshly created (no old data)
- [ ] SQLite database freshly created via migrations (0 devices, 0 scenes)
- [ ] GLCore binary compiled from current source
- [ ] Docker services running: mosquitto (1883), influxdb (8086), knxsim (9090)
- [ ] GLCore running and stable: API (8090), knxd (6720)
- [ ] API responding with empty device/scene lists
- [ ] GLCore log shows "initialisation complete, waiting for shutdown signal"

## Summary Report

After all phases complete, print a summary table:

```
=== NUKE & REBUILD COMPLETE ===

| Component        | Status | Details                    |
|------------------|--------|----------------------------|
| Docker Services  | ...    | mosquitto, influxdb, knxsim|
| GLCore           | ...    | PID, uptime                |
| knxd             | ...    | managed, port 6720         |
| KNX Bridge       | ...    | connected, 0 devices       |
| Database         | ...    | fresh, migrations applied  |
| Flutter Panel    | ...    | built / skipped            |
| API              | ...    | responding on :8090        |

Stack is virgin and ready for testing.
GLCore log: /tmp/graylogic.log (tail -f /tmp/graylogic.log)
```
