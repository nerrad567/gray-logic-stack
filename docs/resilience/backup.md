---
title: Backup and Recovery Specification
version: 1.1.0
status: active
last_updated: 2026-01-17
depends_on:
  - architecture/system-overview.md
  - resilience/offline.md
  - deployment/handover-pack-template.md
---

# Backup and Recovery Specification

This document specifies Gray Logic's backup strategy, recovery procedures, and disaster recovery planning — ensuring systems can be restored after failures, hardware replacements, or catastrophic events.

---

## Overview

### Backup Philosophy

1. **Multiple layers** — Local snapshots + off-site backups
2. **Automated** — Scheduled backups with no manual intervention
3. **Verified** — Regular restore tests to confirm backups work
4. **Complete** — Everything needed to rebuild from scratch
5. **Documented** — Clear procedures anyone can follow

### What Needs Backup

| Data Type | Location | Importance | Backup Frequency |
|-----------|----------|------------|------------------|
| **Configuration** | SQLite database | Critical | Daily + on change |
| **Device registry** | SQLite database | Critical | Daily |
| **Scenes/Schedules** | SQLite database | Critical | Daily |
| **User accounts** | SQLite database | Critical | Daily |
| **Time-series data** | VictoriaMetrics | Important | Daily |
| **Certificates** | /etc/graylogic/certs | Critical | On change |
| **Bridge configs** | /etc/graylogic/bridges | Critical | On change |
| **System config** | /etc/graylogic/*.yaml | Critical | On change |

---

## Backup Strategy

### Backup Types

```yaml
backup_types:
  # Full backup
  full:
    includes:
      - "SQLite database"
      - "VictoriaMetrics data"
      - "Configuration files"
      - "Certificates"
      - "Bridge configurations"
    frequency: "Weekly"
    retention: "4 weeks"
    
  # Incremental backup
  incremental:
    includes:
      - "SQLite database"
      - "Configuration files (if changed)"
    frequency: "Daily"
    retention: "7 days"
    
  # Continuous backup
  continuous:
    includes:
      - "SQLite WAL (write-ahead log)"
    frequency: "Real-time"
    retention: "24 hours"

  # On-demand backup (migration/upgrade triggered)
  on_demand:
    includes:
      - "SQLite database"
    frequency: "Before upgrades/migrations"
    retention: "5 most recent"
    trigger: "Automatic before any database migration"
    location: "/var/backup/graylogic/pre-migration/"
```

### Backup Locations

```yaml
backup_locations:
  # Local backup (always)
  local:
    path: "/var/backup/graylogic"
    retention_days: 7
    purpose: "Fast recovery, short-term"
    
  # USB drive (optional)
  usb:
    enabled: true
    path: "/mnt/backup-usb"
    device: "/dev/sdb1"
    retention_days: 30
    purpose: "Physical off-site rotation"
    
  # Remote backup (optional)
  remote:
    enabled: false
    type: "rsync"                    # rsync | s3 | sftp
    destination: "backup.example.com:/backups/graylogic"
    retention_days: 90
    purpose: "Off-site disaster recovery"
    
  # Encrypted cloud (optional, Premium)
  cloud:
    enabled: false
    provider: "s3"
    bucket: "graylogic-backups"
    encryption: "client-side"
    retention_days: 365
```

---

## Backup Components

### SQLite Database

The primary configuration store.

```yaml
sqlite_backup:
  database: "/var/lib/graylogic/graylogic.db"
  
  # Online backup (no downtime)
  method: "sqlite3 .backup"
  
  # Backup script
  script: |
    #!/bin/bash
    BACKUP_DIR="/var/backup/graylogic/sqlite"
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    
    # Create backup using SQLite online backup
    sqlite3 /var/lib/graylogic/graylogic.db ".backup ${BACKUP_DIR}/graylogic_${TIMESTAMP}.db"
    
    # Compress
    gzip "${BACKUP_DIR}/graylogic_${TIMESTAMP}.db"
    
    # Verify
    gunzip -c "${BACKUP_DIR}/graylogic_${TIMESTAMP}.db.gz" | sqlite3 :memory: "PRAGMA integrity_check;"
    
  # Retention
  retention:
    daily: 7
    weekly: 4
    monthly: 3
```

### VictoriaMetrics Time-Series

Historical PHM and energy data.

```yaml
victoriametrics_backup:
  # Using VictoriaMetrics backup command
  method: "VictoriaMetrics snapshot API"
  
  # Backup script
  script: |
    #!/bin/bash
    BACKUP_DIR="/var/backup/graylogic/victoriametrics"
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    
    # Create backup
    VictoriaMetrics snapshot API "${BACKUP_DIR}/vm_snapshot_${TIMESTAMP}" \
       \
      --bucket phm \
      --token "${TSDB_TOKEN}"
    
    # Compress
    tar -czf "${BACKUP_DIR}/vm_snapshot_${TIMESTAMP}.tar.gz" "${BACKUP_DIR}/vm_snapshot_${TIMESTAMP}"
    rm -rf "${BACKUP_DIR}/vm_snapshot_${TIMESTAMP}"
    
  # Retention
  retention:
    daily: 7
    weekly: 4
```

### Configuration Files

```yaml
config_backup:
  paths:
    - "/etc/graylogic/*.yaml"
    - "/etc/graylogic/bridges/*.yaml"
    - "/etc/graylogic/certs/*"
    
  # Exclude secrets (backed up separately)
  exclude:
    - "/etc/graylogic/secrets.yaml"
    
  # Backup script
  script: |
    #!/bin/bash
    BACKUP_DIR="/var/backup/graylogic/config"
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    
    # Create config archive
    tar -czf "${BACKUP_DIR}/config_${TIMESTAMP}.tar.gz" \
      --exclude="secrets.yaml" \
      --exclude="*.key" \
      /etc/graylogic/
```

### Secrets

Secrets require special handling:

```yaml
secrets_backup:
  # Secrets are encrypted before backup
  encryption:
    method: "gpg"
    key_id: "backup@graylogic.local"
    
  # Backup script
  script: |
    #!/bin/bash
    BACKUP_DIR="/var/backup/graylogic/secrets"
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    
    # Encrypt secrets (tar first to include keys)
    # Include: Secrets, Keys, and Long-term Log Archives
    tar -cf - /etc/graylogic/secrets.yaml /etc/graylogic/*.key /var/lib/graylogic/archive/ | \
      gpg --encrypt --recipient backup@graylogic.local \
      --output "${BACKUP_DIR}/secrets_and_archive_${TIMESTAMP}.tar.gpg"
      
  # Key management
  key_storage:
    - "Password manager (owner)"
    - "Sealed envelope in doomsday pack"
```

### GPG Key Management

The GPG encryption key for secrets backup must be available both on the server (for automated backups) and offsite (for disaster recovery):

```yaml
gpg_key_management:
  # Key generation (during commissioning)
  generation:
    when: "During setup wizard or first commissioning"
    command: "graylogic setup --generate-backup-key"
    output:
      private_key: "/etc/graylogic/backup.key"   # Mode 0600, graylogic:graylogic
      public_key: "/etc/graylogic/backup.pub"

  # Automated backup access
  server_key:
    location: "/etc/graylogic/backup.key"
    permissions: "0600"
    owner: "graylogic:graylogic"
    purpose: "Automated nightly backups can encrypt without human intervention"

  # Offsite recovery (critical for disaster scenarios)
  offsite_key:
    location: "Doomsday Pack (physical envelope)"
    also_in: "Owner's password manager"
    purpose: "If server is destroyed, backup can still be decrypted"

  # Key never stored in:
  never_store:
    - "The backup itself (circular dependency)"
    - "Cloud services"
    - "Unencrypted email"
```

---

## Backup Schedule

### Automated Backup Schedule

```yaml
backup_schedule:
  # Daily at 3 AM
  daily:
    time: "03:00"
    actions:
      - "SQLite incremental backup"
      - "VictoriaMetrics backup"
      - "Config backup (if changed)"
      - "Rotate old backups"
      
  # Weekly on Sunday at 4 AM
  weekly:
    day: "sunday"
    time: "04:00"
    actions:
      - "Full SQLite backup"
      - "Full VictoriaMetrics backup"
      - "Full config backup"
      - "Secrets backup (encrypted)"
      - "Copy to USB drive"
      
  # Monthly on 1st at 5 AM
  monthly:
    day: 1
    time: "05:00"
    actions:
      - "Full backup"
      - "Backup verification test"
      - "Copy to off-site (if configured)"
```

### Systemd Timer

```ini
# /etc/systemd/system/graylogic-backup.timer
[Unit]
Description=Gray Logic Daily Backup

[Timer]
OnCalendar=*-*-* 03:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

```ini
# /etc/systemd/system/graylogic-backup.service
[Unit]
Description=Gray Logic Backup Service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/graylogic-backup.sh daily
User=graylogic
```

---

## Backup Verification

### Automated Verification

```yaml
backup_verification:
  # Verify after each backup
  post_backup:
    sqlite:
      action: "PRAGMA integrity_check"
      expected: "ok"
      
    tsdb:
      action: "Query test data from backup"
      expected: "Data matches source"
      
    config:
      action: "Validate YAML syntax"
      expected: "No errors"
      
  # Monthly full verification
  monthly:
    action: "Full restore to test environment"
    verify:
      - "All configuration present"
      - "All devices listed"
      - "All scenes available"
      - "Historical data queryable"
```

### Verification Script

```bash
#!/bin/bash
# /usr/local/bin/graylogic-backup-verify.sh

set -e

BACKUP_DIR="/var/backup/graylogic"
LATEST_SQLITE=$(ls -t ${BACKUP_DIR}/sqlite/*.db.gz | head -1)
TEMP_DIR=$(mktemp -d)

echo "Verifying SQLite backup: ${LATEST_SQLITE}"

# Decompress
gunzip -c "${LATEST_SQLITE}" > "${TEMP_DIR}/test.db"

# Integrity check
RESULT=$(sqlite3 "${TEMP_DIR}/test.db" "PRAGMA integrity_check;")
if [ "$RESULT" != "ok" ]; then
    echo "ERROR: SQLite integrity check failed"
    exit 1
fi

# Row counts
DEVICES=$(sqlite3 "${TEMP_DIR}/test.db" "SELECT COUNT(*) FROM devices;")
SCENES=$(sqlite3 "${TEMP_DIR}/test.db" "SELECT COUNT(*) FROM scenes;")
USERS=$(sqlite3 "${TEMP_DIR}/test.db" "SELECT COUNT(*) FROM users;")

echo "Verification passed:"
echo "  Devices: ${DEVICES}"
echo "  Scenes: ${SCENES}"
echo "  Users: ${USERS}"

# Cleanup
rm -rf "${TEMP_DIR}"
```

---

## Migration-Triggered Backups

### Automatic Pre-Migration Backup

When Gray Logic Core starts and detects pending database migrations, it **automatically** creates a verified backup before applying any changes.

```yaml
pre_migration_backup:
  # Automatic behavior
  trigger: "Pending migrations detected on startup"
  location: "/var/backup/graylogic/pre-migration/"
  naming: "graylogic_{timestamp}_before_{migration_version}.db"

  # Safety requirements
  verify_integrity: true     # PRAGMA integrity_check after backup
  abort_on_failure: true     # Do not migrate if backup fails

  # Retention
  keep_count: 5              # Keep last 5 pre-migration backups
```

### Why This Matters

Even with additive-only migrations (see [ADR-004](../architecture/decisions/004-additive-only-migrations.md)), migrations can fail:

| Failure Mode | Impact | Recovery |
|--------------|--------|----------|
| SQL syntax error | Migration aborts cleanly | Retry after fix |
| Data transformation bug | Corrupted data | **Restore from pre-migration backup** |
| Disk full | Partial migration | Restore from backup |
| Power loss | Database corruption | Restore from backup |

### Backup File Naming

```
/var/backup/graylogic/pre-migration/
├── graylogic_20260115_080000_before_045.db
├── graylogic_20260120_080000_before_046.db
├── graylogic_20260125_080000_before_047.db
└── graylogic_20260201_080000_before_048.db
```

The naming convention includes:
- Timestamp of backup creation
- Migration version number that was about to be applied

### Manual Pre-Upgrade Backup

Before major upgrades (even if no migrations are expected), manually trigger a backup:

```bash
# Create on-demand backup
graylogic backup --type on-demand --reason "Pre-upgrade to v1.5.0"

# Verify backup
graylogic backup --verify /var/backup/graylogic/pre-migration/LATEST.db
```

### Integration with Rollback

See [ADR-004: Additive-Only Database Migrations](../architecture/decisions/004-additive-only-migrations.md) for complete rollback procedures using these backups.

---

## Recovery Procedures

### Full System Recovery

Complete rebuild from backup:

```yaml
full_recovery:
  prerequisites:
    - "Fresh OS installation (Debian/Ubuntu)"
    - "Network connectivity"
    - "Access to backup files"
    
  steps:
    1_install_graylogic:
      commands:
        - "curl -fsSL https://install.graylogic.uk | sudo bash"
        - "sudo systemctl stop graylogic"
        
    2_restore_database:
      commands:
        - "gunzip -c backup/graylogic_latest.db.gz > /var/lib/graylogic/graylogic.db"
        - "chown graylogic:graylogic /var/lib/graylogic/graylogic.db"
        
    3_restore_config:
      commands:
        - "tar -xzf backup/config_latest.tar.gz -C /"
        - "# Restore secrets separately (decrypt first)"
        - "gpg --decrypt backup/secrets_latest.yaml.gpg > /etc/graylogic/secrets.yaml"
        - "chmod 600 /etc/graylogic/secrets.yaml"
        
    4_restore_victoriametrics:
      commands:
        - "tar -xzf backup/vm_snapshot_latest.tar.gz"
        - "VictoriaMetrics restore ./vm_snapshot_latest "
        
    5_start_services:
      commands:
        - "sudo systemctl start graylogic"
        - "sudo systemctl status graylogic"
        
    6_verify:
      actions:
        - "Check web UI is accessible"
        - "Verify device list is correct"
        - "Test device control"
        - "Verify scenes work"
```

### Database Recovery

Restore database only (e.g., after corruption):

```yaml
database_recovery:
  steps:
    1_stop_core:
      command: "sudo systemctl stop graylogic"
      
    2_backup_current:
      command: "mv /var/lib/graylogic/graylogic.db /var/lib/graylogic/graylogic.db.corrupt"
      
    3_restore:
      command: "gunzip -c /var/backup/graylogic/sqlite/graylogic_latest.db.gz > /var/lib/graylogic/graylogic.db"
      
    4_permissions:
      command: "chown graylogic:graylogic /var/lib/graylogic/graylogic.db"
      
    5_start_core:
      command: "sudo systemctl start graylogic"
      
    6_verify:
      actions:
        - "Check logs for errors"
        - "Verify configuration loaded"
        - "Test device control"
        
  data_loss:
    warning: "Changes since last backup will be lost"
    mitigation: "Consider extracting data from corrupt DB first"
```

### Configuration Recovery

Restore configuration files only:

```yaml
config_recovery:
  steps:
    1_stop_core:
      command: "sudo systemctl stop graylogic"
      
    2_backup_current:
      command: "tar -czf /tmp/config_corrupt.tar.gz /etc/graylogic/"
      
    3_restore:
      command: "tar -xzf /var/backup/graylogic/config/config_latest.tar.gz -C /"
      
    4_restore_secrets:
      commands:
        - "gpg --decrypt /var/backup/graylogic/secrets/secrets_latest.yaml.gpg > /etc/graylogic/secrets.yaml"
        - "chmod 600 /etc/graylogic/secrets.yaml"
        
    5_start_core:
      command: "sudo systemctl start graylogic"
```

### Partial Recovery

Restore specific data (e.g., scenes deleted by mistake):

```yaml
partial_recovery:
  # Restore from backup to temp database
  extract:
    commands:
      - "gunzip -c backup/graylogic_latest.db.gz > /tmp/restore.db"
      - "# Extract specific data"
      - "sqlite3 /tmp/restore.db '.dump scenes' > /tmp/scenes.sql"
      
  # Merge into production
  merge:
    warning: "Review SQL before importing"
    commands:
      - "# Backup current first"
      - "sqlite3 /var/lib/graylogic/graylogic.db '.backup /tmp/before_merge.db'"
      - "# Import extracted data"
      - "sqlite3 /var/lib/graylogic/graylogic.db < /tmp/scenes.sql"
```

---

## Disaster Recovery

### Disaster Scenarios

```yaml
disaster_scenarios:
  hardware_failure:
    description: "Server hardware dies"
    recovery_time: "2-4 hours"
    procedure:
      - "Replace hardware"
      - "Install OS"
      - "Restore from backup"
      - "Verify operation"
      
  fire_flood:
    description: "Physical destruction of site equipment"
    recovery_time: "1-2 days"
    procedure:
      - "Acquire replacement hardware"
      - "Retrieve off-site backup"
      - "Full system restore"
      - "Reconfigure network"
      - "Verify all integrations"
      
  ransomware:
    description: "System encrypted by malware"
    recovery_time: "4-8 hours"
    procedure:
      - "Disconnect from network"
      - "Wipe and reinstall OS"
      - "Restore from known-good backup"
      - "Change all credentials"
      - "Investigate attack vector"
      
  corruption:
    description: "Database or file corruption"
    recovery_time: "1-2 hours"
    procedure:
      - "Identify corrupted component"
      - "Restore from backup"
      - "Investigate cause"
```

### Recovery Time Objectives

```yaml
recovery_objectives:
  # RTO: Recovery Time Objective
  rto:
    critical_functions: "4 hours"
    full_restoration: "24 hours"
    
  # RPO: Recovery Point Objective
  rpo:
    configuration: "24 hours"        # Daily backups
    time_series: "24 hours"
    
  # Note: Physical controls work immediately
  # These RTOs are for Gray Logic automation only
```

---

## Doomsday Pack

### Concept

The "Doomsday Pack" is a complete handover package enabling another integrator (or the owner) to take over or rebuild the system if Gray Logic (the business) becomes unavailable.

### Contents

```yaml
doomsday_pack:
  location: "Sealed envelope with owner"
  
  contents:
    credentials:
      - "Admin username and password"
      - "Root/sudo password"
      - "Database encryption key"
      - "Backup encryption key (GPG private key)"
      - "WireGuard private keys"
      - "API keys for integrations"
      
    access:
      - "Server IP address and port"
      - "WireGuard configuration file"
      - "SSH keys (if applicable)"
      
    documentation:
      - "System architecture diagram"
      - "Device schedule (all devices, addresses)"
      - "Network diagram"
      - "Runbook (common procedures)"
      - "This backup specification"
      
    recovery:
      - "USB drive with latest full backup"
      - "Step-by-step recovery instructions"
      
    contacts:
      - "Alternative integrators (2-3 companies)"
      - "Equipment suppliers"
      - "Gray Logic emergency contact"
      
    physical:
      - "Server room access instructions"
      - "Network cabinet key location"
      - "Emergency shutdown procedure"
```

### Update Schedule

```yaml
doomsday_pack_updates:
  # When to update
  triggers:
    - "Password changes"
    - "Major configuration changes"
    - "New equipment added"
    - "Annual review"
    
  # Update procedure
  procedure:
    - "Generate new USB backup"
    - "Update credential sheet"
    - "Print updated documentation"
    - "Seal in new envelope"
    - "Deliver to owner"
    - "Destroy old pack"
```

### Emergency Shutdown

Included in doomsday pack:

```yaml
emergency_shutdown:
  purpose: "Safely disable Gray Logic while maintaining basic building function"
  
  steps:
    1_understand:
      note: "Without Gray Logic, physical controls still work"
      
    2_stop_core:
      command: "sudo systemctl stop graylogic"
      effect: "Stops automation, scenes, schedules"
      
    3_stop_bridges:
      command: "sudo systemctl stop graylogic-*"
      effect: "Stops all bridge processes"
      
    4_optional_power_off:
      action: "Power off server"
      effect: "Complete shutdown"
      
    5_building_operation:
      note: |
        Physical controls continue to work:
        - KNX wall switches control lights
        - Climate controls at thermostats
        - Security panel operates independently
        - CCTV records to NVR
```

---

## Configuration

### Backup Configuration

```yaml
# /etc/graylogic/backup.yaml
backup:
  # Enable/disable
  enabled: true
  
  # Local backup
  local:
    path: "/var/backup/graylogic"
    retention_days: 7
    
  # USB backup
  usb:
    enabled: true
    label: "Local USB"
    mount_point: "/mnt/backup_usb"
    status: "mandatory_for_commissioning"  # Required for "Gold Master" handover
    auto_mount: true
    retention_days: 30
    
  # Remote backup
  remote:
    enabled: false
    type: "rsync"
    destination: "user@backup.example.com:/backups/graylogic"
    ssh_key: "/etc/graylogic/backup_ssh_key"
    
  # Schedule
  schedule:
    daily_time: "03:00"
    weekly_day: "sunday"
    weekly_time: "04:00"
    
  # Verification
  verification:
    enabled: true
    monthly_restore_test: true
    
  # Notifications
  notifications:
    on_success: false
    on_failure: true
    recipients: ["admin@example.com"]
    
  # Encryption
  encryption:
    secrets:
      enabled: true
      gpg_key_id: "backup@graylogic.local"
```

---

## Monitoring and Alerts

### Backup Monitoring

```yaml
backup_monitoring:
  # Track backup status
  metrics:
    - "last_backup_timestamp"
    - "backup_size_bytes"
    - "backup_duration_seconds"
    - "verification_result"
    
  # Alerts
  alerts:
    backup_failed:
      condition: "backup_exit_code != 0"
      severity: "critical"
      notification: "immediate"
      
    backup_overdue:
      condition: "time_since_backup > 36 hours"
      severity: "warning"
      notification: "email"
      
    backup_size_anomaly:
      condition: "size_change > 50%"
      severity: "warning"
      notification: "email"
      
    verification_failed:
      condition: "verification_result != 'ok'"
      severity: "critical"
      notification: "immediate"
```

### Backup Status API

```http
GET /api/v1/system/backup/status
```

**Response:**
```json
{
  "last_backup": {
    "timestamp": "2026-01-13T03:00:00Z",
    "type": "daily",
    "status": "success",
    "duration_seconds": 45,
    "size_bytes": 52428800
  },
  "last_verification": {
    "timestamp": "2026-01-01T05:00:00Z",
    "status": "success"
  },
  "next_backup": {
    "timestamp": "2026-01-14T03:00:00Z",
    "type": "daily"
  },
  "storage": {
    "local": {
      "used_bytes": 524288000,
      "available_bytes": 10737418240
    },
    "usb": {
      "connected": true,
      "used_bytes": 1073741824
    }
  }
}
```

---

## Related Documents

- [Offline Behavior](offline.md) — Graceful degradation
- [System Overview](../architecture/system-overview.md) — Architecture
- [Handover Pack Template](../deployment/handover-pack-template.md) — Customer handover
- [Security Model](../architecture/security-model.md) — Secrets management
