---
title: Infrastructure & Data Strategy
version: 1.0.0
status: active
last_updated: 2026-01-17
depends_on:
  - architecture/system-overview.md
  - operations/backup.md
---

# Infrastructure & Data Strategy

To ensure "10-year stability", the infrastructure (OS, containers, database) must be reproducible, version-controlled, and robust against upgrades.

---

## Infrastructure as Code (IaC)

We treat the server configuration as code. No manual `apt-get install` on production servers.

### 1. Docker Composition

The primary deployment artifact is a `docker-compose.yml` file. This locks the versions of all dependencies.

**Structure:**
```yaml
services:
  # The Core Logic
  core:
    image: graylogic/core:1.0.0
    restart: unless-stopped
    network_mode: host  # Required for mDNS/KNX multicast
    volumes:
      - ./config:/etc/graylogic
      - ./data:/var/lib/graylogic

  # Message Bus
  mqtt:
    image: eclipse-mosquitto:2.0.18
    restart: always
    ports:
      - "1883:1883"
    volumes:
      - ./mosquitto/config:/mosquitto/config
      - ./mosquitto/data:/mosquitto/data

  # Time Series Data
  influxdb:
    image: influxdb:2.7-alpine
    restart: always
    volumes:
      - ./influxdb/data:/var/lib/influxdb2
      - ./influxdb/config:/etc/influxdb2

  # Bridges (Separate Containers)
  bridge-knx:
    image: graylogic/bridge-knx:1.0.0
    environment:
      - MQTT_HOST=localhost
```

### 2. OS Provisioning (Ansible)

For bare-metal (Industrial PC) provisioning, we use Ansible.

**Playbook Responsibilities:**
1.  **Security Hardening:** Configure UFW firewall, fail2ban, SSH keys.
2.  **Docker Setup:** Install Docker Engine, Docker Compose.
3.  **Networking:** Configure static IPs, VLAN interfaces (if needed).
4.  **Deploy:** Copy `docker-compose.yml` and pull images.

**Repository:** `ops/ansible/site.yml`

---

## Database Migration Strategy

Gray Logic uses SQLite. Schema changes must be non-destructive and strictly versioned.

### Tooling

We use **`golang-migrate`** to manage SQL migrations.

**Directory:** `gray-logic-core/migrations/`

**File Format:**
- `001_initial_schema.up.sql`
- `001_initial_schema.down.sql`
- `002_add_device_profiles.up.sql`

### Migration Process

1.  **On Startup:** Core checks the current schema version in `schema_migrations` table.
2.  **Auto-Migrate:** If new migration files exist, Core applies them automatically inside a transaction.
3.  **Failure Safety:** If a migration fails, the transaction rolls back, and Core exits with a fatal error (preventing data corruption).

### Rules for Schema Changes

1.  **Additive Only:** Never rename or drop columns in an active system.
    - *Bad:* `ALTER TABLE devices RENAME COLUMN type TO kind;`
    - *Good:* Add `kind`, backfill data, deprecate `type`.
2.  **Data Integrity:** All `CREATE TABLE` must include `STRICT` mode (SQLite 3.37+).
3.  **Idempotency:** Migrations should use `IF NOT EXISTS` where possible.

---

## Data Persistence & Backup

### Volume Strategy

All persistent data lives in a single directory structure:

```
/opt/graylogic/
├── config/          # YAML config, secrets, certificates
├── data/            # SQLite db (core.db)
├── influxdb/        # History (metrics)
└── mosquitto/       # Retained messages
```

### Backup Routine

A strict backup strategy is enforced by the Core (see `system/backup` API).

**Process:**
1.  **Lock:** SQLite `VACUUM INTO '/backup/core.db'` (safe hot backup).
2.  **Snapshot:** InfluxDB `backup` command.
3.  **Archive:** Tar/Gzip config + SQLite backup + Influx snapshot.
4.  **Export:** Copy to external USB or upload to remote target (if configured).

**Retention:**
- Local: Last 7 days.
- External: Configurable.

---

## Upgrade Strategy

**"Blue/Green" Container Update:**
1.  Pull new images.
2.  Stop Core container (downtime < 5s).
3.  Start new Core container.
4.  Core runs DB migrations.
5.  System resumes.

**Rollback:**
If the new container crashes on startup (e.g., migration failure), the watchdog (systemd/docker) can revert to the previous tag (requires manual intervention or advanced orchestrator).
