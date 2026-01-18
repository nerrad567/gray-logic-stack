---
title: System Maintenance Procedures
version: 1.0.0
status: active
last_updated: 2026-01-17
depends_on:
  - data-model/entities.md
  - resilience/backup.md
---

# System Maintenance Procedures

This document defines standard operating procedures for maintaining a Gray Logic system over its multi-decade lifecycle, specifically focusing on hardware replacement (RMA) and data integrity.

---

## 1. Device Replacement (RMA) Workflow

When a physical device fails and is replaced, the system must update its records to point to the new hardware while **preserving all historical data, scenes, and automations**.

### The Problem
*   **Old Device:** `light-living-main` (KNX Address: 1.1.5) -> Has 3 years of energy history.
*   **New Device:** Replaced physical unit (KNX Address: 1.1.10).
*   **Naive Approach:** Delete `light-living-main`, create `light-living-new`.
*   **Result:** Broken scenes, lost history, broken schedules.

### The "Swap Device" Solution

The system provides a specific `swap_device` operation that updates the *protocol address* of an existing entity without changing its UUID.

### Workflow

1.  **Physical Installation:** Electrician installs new device and assigns a new protocol address (if required).
2.  **Discovery (Optional):** System discovers the new device in the "Staging Area".
3.  **Swap Action:**
    - Admin navigates to the **failed device** in the Web UI.
    - Selects **"Maintenance > Replace Device"**.
    - Chooses the replacement from the **Staging Area** OR enters the new address manually.
4.  **System Execution:**
    - Updates `devices` table: Sets `address = new_address`.
    - Updates `devices` table: Sets `capabilities` (if model changed, see Compatibility).
    - Logs event: `device_swapped` (Audit trail).
    - Clears "Device Offline" alerts.

### Compatibility Checks

*   **Exact Match:** Same Manufacturer/Model. Swap proceeds automatically.
*   **Compatible:** Different model but superset of capabilities (e.g., RGBW replaces RGB). Swap proceeds; new capabilities added.
*   **Incompatible:** Missing critical capabilities (e.g., On/Off replaces Dimmer). System warns: "Scenes A, B, C will be broken. Proceed?"

---

## 2. Controller Migration

If the Gray Logic Core server (Industrial PC) fails, the entire system configuration must be movable to new hardware.

### Restoration Workflow

1.  **Install OS:** Flash standard Gray Logic image to new hardware.
2.  **Boot:** System enters "First Run Wizard".
3.  **Restore:** Select "Restore from Backup".
4.  **Upload:** Upload the latest `.glb` (Gray Logic Backup) file.
5.  **Decrypt:** Enter the backup encryption password.
6.  **Verify:** System checks integrity of SQLite and InfluxDB data.
7.  **Re-Key (Optional):** If the old hardware was stolen, the admin can choose "Rotate Secrets" to invalidate old API keys and certificates.

---

## 3. Database Maintenance

To ensure performance over decades, the database requires periodic grooming.

### Time-Series Retention (InfluxDB)

Data is downsampled automatically to save space:

| Metric | High Res (Raw) | Medium Res (1h avg) | Low Res (24h avg) |
|--------|----------------|---------------------|-------------------|
| Energy | 30 days | 1 year | Forever |
| Temperature | 30 days | 1 year | 5 years |
| Presence | 14 days | 90 days | Deleted |
| Logs | 7 days | Deleted | Deleted |

### SQLite Vacuuming

*   **Frequency:** Monthly (during maintenance window).
*   **Action:** `VACUUM;` command runs to reclaim unused space and defragment indices.
*   **Impact:** < 500ms lock time (usually negligible).

---

## 4. Certificate Rotation

TLS certificates are critical for security but expire.

### Automatic Rotation
*   **Internal CA:** The internal CA (managed by Core) automatically issues new certificates to Bridges and UIs 30 days before expiration.
*   **Self-Healing:** If a Bridge connects with an expired cert, the Core rejects it but offers a specific "Renewal" handshake (requires shared secret).

### Certificate Renewal Handshake Protocol

When a bridge certificate has expired, the following recovery procedure applies:

```yaml
certificate_renewal_handshake:
  # Scenario: Bridge has expired certificate
  trigger: "TLS handshake fails due to expired certificate"

  # Recovery flow
  flow:
    1: "Core rejects TLS connection (expired cert)"
    2: "Bridge detects rejection reason from TLS error"
    3: "Bridge connects to renewal endpoint (localhost:8443/renew or unix socket)"
    4: "Bridge presents: { bridge_id, timestamp, hmac(bridge_id + timestamp, shared_secret) }"
    5: "Core validates HMAC against stored bridge secret"
    6: "Core issues new certificate to bridge (1 year validity)"
    7: "Bridge stores new certificate, retries normal connection"

  # Shared secret
  shared_secret:
    location: "/etc/graylogic/bridges/{bridge_id}.secret"
    generated: "During bridge provisioning (stored on both Core and Bridge)"
    length: "256-bit random"
    rotation: "When bridge is re-provisioned"

  # Security constraints
  security:
    renewal_endpoint: "localhost only (or mTLS with expired cert exception)"
    request_rate_limit: "1 per minute per bridge_id"
    max_attempts: 5                    # Before requiring manual intervention
    audit_log: true                    # Log all renewal attempts

  # Fallback: Manual renewal
  manual_renewal:
    when: "Renewal handshake fails or shared secret lost"
    procedure:
      1: "SSH to Core server"
      2: "Run: graylogic bridge renew-cert --bridge-id={id}"
      3: "Copy new certificate to bridge"
      4: "Restart bridge service"
```

---

## 5. Audit Log Archival

Security logs are immutable but consume space.

*   **Active Log:** Stored in SQLite for 1 year (searchable).
*   **Archival:** Every month, logs older than 1 year are exported to JSONL (JSON Lines), gzipped, and moved to `/var/lib/graylogic/archive/`.
*   **Retention:** Archives kept for 5 years (configurable) then deleted.
