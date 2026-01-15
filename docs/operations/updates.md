---
title: Update and Upgrade Strategy
version: 1.0.0
status: active
last_updated: 2026-01-15
depends_on:
  - overview/principles.md
  - resilience/backup.md
  - architecture/system-overview.md
---

# Update and Upgrade Strategy

This document defines how Gray Logic systems are updated, ensuring the 10-year deployment horizon while maintaining security and stability.

---

## Overview

### Update Philosophy

Gray Logic follows a conservative update strategy aligned with the 10-year deployment horizon:

1. **Stability over features** — Security patches prioritized over new features
2. **No forced updates** — Customer chooses when to update
3. **Always rollback-ready** — Every update can be undone
4. **Tested before deployed** — Updates tested on staging before production
5. **Offline-capable** — Updates work without internet (local package repository)

### Update Types

| Type | Frequency | Contents | Risk Level |
|------|-----------|----------|------------|
| **Security patch** | As needed | CVE fixes, dependency updates | Low |
| **Bug fix** | Monthly | Defect corrections | Low |
| **Minor release** | Quarterly | New features, improvements | Medium |
| **Major release** | Annually | Significant changes, migrations | Higher |

---

## Version Scheme

### Semantic Versioning

Gray Logic uses semantic versioning: `MAJOR.MINOR.PATCH`

```
v2.3.15
│ │ └── Patch: Bug fixes, security patches
│ └──── Minor: New features, backward-compatible
└────── Major: Breaking changes, migrations required
```

### Update Channels

```yaml
update_channels:
  stable:
    description: "Production-ready releases"
    update_frequency: "Quarterly minor, monthly patches"
    testing: "Minimum 30 days in beta"
    recommended_for: "All production deployments"

  beta:
    description: "Pre-release testing"
    update_frequency: "As released"
    testing: "Internal testing complete"
    recommended_for: "Test environments, early adopters"

  lts:
    description: "Long-term support (2-year cycle)"
    update_frequency: "Security patches only"
    testing: "Extended stability testing"
    recommended_for: "Risk-averse deployments, commercial"
```

---

## Pre-Update Procedures

### Automated Pre-flight Checks

Before any update, the system performs:

```yaml
preflight_checks:
  - name: "Disk space"
    check: "Minimum 2GB free"
    fail_action: "abort"

  - name: "Database integrity"
    check: "sqlite3 integrity_check"
    fail_action: "abort"

  - name: "Backup exists"
    check: "Backup < 24 hours old"
    fail_action: "create_backup"

  - name: "All bridges healthy"
    check: "Health status = healthy"
    fail_action: "warn"

  - name: "No active scenes"
    check: "No scenes currently executing"
    fail_action: "wait_30s"

  - name: "System load"
    check: "Load average < 2.0"
    fail_action: "wait"
```

### Manual Pre-Update Checklist

```yaml
manual_checklist:
  - [ ] Read release notes
  - [ ] Check known issues for your configuration
  - [ ] Verify backup exists and is restorable
  - [ ] Schedule update during low-usage period
  - [ ] Notify users of potential brief disruption
  - [ ] Have rollback plan ready
  - [ ] Test update on staging first (if available)
```

---

## Update Procedures

### Core Update (Gray Logic Core)

```yaml
core_update:
  # Step 1: Download and verify
  download:
    source: "https://releases.graylogic.uk/core/{version}/graylogic-core"
    verify:
      checksum: "sha256"
      signature: "gpg"
    local_fallback: "/opt/graylogic/packages/"

  # Step 2: Pre-flight
  preflight:
    - backup_database
    - backup_config
    - stop_non_essential_services

  # Step 3: Apply update
  apply:
    method: "atomic_replace"
    steps:
      - "Stop Core service"
      - "Copy new binary to /opt/graylogic/bin/graylogic-core.new"
      - "Verify binary integrity"
      - "Move current to /opt/graylogic/bin/graylogic-core.bak"
      - "Move new to /opt/graylogic/bin/graylogic-core"
      - "Run database migrations (if any)"
      - "Start Core service"
      - "Verify health check passes"

  # Step 4: Post-update verification
  verify:
    - health_check: "/api/v1/health returns 200"
    - bridges_reconnect: "All bridges show healthy within 60s"
    - test_command: "Send test command to known device"
    - ui_accessible: "Wall panel connects successfully"

  # Step 5: Cleanup or rollback
  completion:
    success:
      - "Remove .bak file after 7 days"
      - "Log update completion"
      - "Notify admin"
    failure:
      - "Execute rollback procedure"
      - "Alert admin"
```

### Bridge Update

```yaml
bridge_update:
  # Bridges update independently
  procedure:
    - "Download new bridge binary"
    - "Stop bridge service"
    - "Backup current binary"
    - "Install new binary"
    - "Start bridge service"
    - "Verify bridge health"

  # Bridges can be updated without Core downtime
  rolling_update:
    enabled: true
    order: "one_at_a_time"
    wait_between: 30                # Seconds between bridges
    verify_each: true
```

### Database Migration

```yaml
database_migration:
  # Migrations are versioned SQL files
  location: "/opt/graylogic/migrations/"
  naming: "YYYYMMDD_HHMMSS_description.sql"

  procedure:
    - "Check current schema version"
    - "Identify pending migrations"
    - "Backup database"
    - "Apply migrations in order"
    - "Verify schema integrity"
    - "Update schema version table"

  rollback:
    method: "restore_from_backup"
    # Down migrations are provided but restore is preferred
    down_migrations: true

  # Example migration
  example:
    file: "20260115_120000_add_scene_execution.sql"
    content: |
      -- Up migration
      CREATE TABLE scene_executions (
        id TEXT PRIMARY KEY,
        scene_id TEXT NOT NULL,
        status TEXT NOT NULL,
        triggered_at TIMESTAMP NOT NULL,
        ...
      );

      -- Down migration (in separate .down.sql file)
      DROP TABLE scene_executions;
```

---

## Rollback Procedures

### Automatic Rollback

If post-update verification fails:

```yaml
automatic_rollback:
  triggers:
    - "Health check fails after 60s"
    - "Bridges don't reconnect after 120s"
    - "Database migration fails"

  procedure:
    - "Stop Core service"
    - "Restore previous binary from .bak"
    - "Restore database from pre-update backup"
    - "Start Core service"
    - "Verify rollback successful"
    - "Alert admin with failure reason"

  max_automatic_attempts: 1
  after_failure: "Manual intervention required"
```

### Manual Rollback

```bash
# Rollback Gray Logic Core
sudo systemctl stop graylogic-core
sudo mv /opt/graylogic/bin/graylogic-core /opt/graylogic/bin/graylogic-core.failed
sudo mv /opt/graylogic/bin/graylogic-core.bak /opt/graylogic/bin/graylogic-core

# Restore database if needed
sudo graylogic-ctl restore --backup /var/backups/graylogic/pre-update.db

# Restart
sudo systemctl start graylogic-core

# Verify
curl -k https://localhost:8080/api/v1/health
```

---

## Offline Updates

For systems without internet access:

```yaml
offline_updates:
  # Package repository on USB drive or local server
  local_repository:
    path: "/opt/graylogic/packages/"
    structure:
      - "core/graylogic-core-{version}"
      - "bridges/graylogic-bridge-knx-{version}"
      - "migrations/{version}/"
      - "checksums.sha256"
      - "signatures.gpg"

  # USB update procedure
  usb_update:
    - "Insert USB drive with packages"
    - "Mount drive: mount /dev/sdb1 /mnt/update"
    - "Verify signatures: gpg --verify checksums.sha256.sig"
    - "Run updater: graylogic-ctl update --source /mnt/update"
    - "Updater follows standard procedure"
    - "Unmount drive"

  # Air-gapped update (maximum security)
  airgapped:
    - "Download packages on separate machine"
    - "Verify signatures"
    - "Transfer via USB"
    - "Apply update"
```

---

## Update Scheduling

### Maintenance Windows

```yaml
maintenance_windows:
  # Define acceptable update times
  allowed_times:
    - day: "Sunday"
      start: "02:00"
      end: "06:00"
    - day: "Wednesday"
      start: "02:00"
      end: "04:00"

  # Automatic updates (optional, off by default)
  automatic_updates:
    enabled: false
    channel: "stable"
    type: "security_only"           # Only security patches
    require_approval: true          # Notify admin, wait for approval

  # Pre-notification
  notification:
    hours_before: 24
    method: ["email", "dashboard"]
```

---

## Update Notifications

### Admin Notifications

```yaml
update_notifications:
  # Check for updates
  check_frequency: "daily"
  check_url: "https://releases.graylogic.uk/latest.json"

  # Notification methods
  notify:
    dashboard: true                 # Show in web admin
    email: true                     # Email to admin
    push: false                     # Mobile push (if configured)

  # Notification content
  content:
    - current_version
    - available_version
    - release_notes_summary
    - security_advisories
    - recommended_action
```

### Release Notes Format

```yaml
release_notes:
  version: "2.3.15"
  date: "2026-01-15"
  channel: "stable"

  security:
    - severity: "medium"
      cve: "CVE-2026-1234"
      description: "TLS certificate validation bypass"
      fixed: true

  features:
    - "Added scene execution tracking"
    - "Improved voice fallback handling"

  bug_fixes:
    - "Fixed race condition in state reconciliation"
    - "Fixed memory leak in WebSocket handler"

  breaking_changes: []

  migration_required: true
  migration_notes: "Run database migration for scene_executions table"

  known_issues:
    - "DALI bridge may require restart after update"
```

---

## Testing Updates

### Staging Environment

```yaml
staging:
  # Recommended: Test updates on staging before production
  setup:
    - "Mirror production configuration"
    - "Use subset of devices (or simulators)"
    - "Connect to test MQTT broker"

  test_procedure:
    - "Apply update to staging"
    - "Run automated tests"
    - "Manual verification"
    - "Monitor for 24-48 hours"
    - "If stable, apply to production"

  # Automated test suite
  tests:
    - "API health check"
    - "All bridges connect"
    - "Scene activation works"
    - "Schedule triggers correctly"
    - "Voice commands work (if enabled)"
    - "State persistence survives restart"
```

---

## Emergency Procedures

### Security Emergency Update

For critical security vulnerabilities:

```yaml
emergency_update:
  # Expedited process for critical CVEs
  criteria:
    - "CVSS score >= 9.0"
    - "Active exploitation in the wild"
    - "Remote code execution possible"

  procedure:
    - "Security advisory published immediately"
    - "Patch released within 24 hours"
    - "Admin notified via all channels"
    - "Update recommended immediately"

  # Temporary mitigations
  mitigations:
    - "Disable affected feature"
    - "Network isolation"
    - "Additional firewall rules"
```

### Recovery from Failed Update

```yaml
recovery:
  # If system is unbootable after update
  steps:
    - "Boot from recovery USB (if available)"
    - "Mount system partition"
    - "Restore from backup"
    - "Verify and restart"

  # Remote recovery (if SSH accessible)
  remote:
    - "SSH into system"
    - "Manual rollback procedure"
    - "If database corrupted, restore from backup"

  # Last resort
  factory_reset:
    - "Reinstall Gray Logic from scratch"
    - "Restore configuration from backup"
    - "Re-commission devices"
```

---

## Configuration

### Update Settings

```yaml
# /etc/graylogic/updates.yaml
updates:
  # Channel selection
  channel: "stable"

  # Automatic checking
  check:
    enabled: true
    frequency: "daily"
    time: "03:00"

  # Automatic updates (default: off)
  automatic:
    enabled: false
    security_only: true
    require_approval: true

  # Local package repository
  local_repository:
    enabled: false
    path: "/opt/graylogic/packages"

  # Notifications
  notifications:
    email: "admin@example.com"
    dashboard: true

  # Backup before update
  backup:
    automatic: true
    retain_days: 30
```

---

## Related Documents

- [Principles](../overview/principles.md) — 10-year deployment horizon
- [Backup and Recovery](../resilience/backup.md) — Backup procedures
- [Security Model](../architecture/security-model.md) — Security update handling
- [Monitoring](monitoring.md) — System health monitoring

