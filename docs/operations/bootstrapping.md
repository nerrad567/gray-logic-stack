---
title: Bootstrapping & First-Run Security
version: 1.1.0
status: active
last_updated: 2026-01-17
depends_on:
  - architecture/security-model.md
  - architecture/core-internals.md
changelog:
  - version: 1.1.0
    date: 2026-01-17
    changes:
      - "Added claim token expiry (1 hour) and rotation (every 15 minutes)"
      - "Added setup mode timeout (24 hours)"
      - "Added detailed security considerations for claim token handling"
---

# Bootstrapping & First-Run Security

This document defines the process for initializing a new Gray Logic installation. The goal is to solve the "Chicken and Egg" problem of secret generation and ensuring the system is secure by default from the very first boot.

---

## The Problem

A raw installation (e.g., a fresh SD card image or Docker container) needs:
1.  Cryptographically secure secrets (JWT keys, DB encryption keys).
2.  Unique passwords for infrastructure (MQTT, InfluxDB).
3.  An initial Administrator account.
4.  Site-specific configuration (Timezone, Location).

Hardcoding default passwords (e.g., `admin/admin`) is a violation of our security principles.

## The Solution: First-Run Wizard

The Core detects if it is uninitialized (missing `secrets.yaml` or `config.yaml`) and enters **Setup Mode**.

### State Machine

1.  **Uninitialized:** No config found. Core starts a temporary HTTP server on port 8080.
2.  **Setup Mode:** API accepts unauthenticated requests *only* from `localhost` or via a special **Claim Token**.
3.  **Provisioning:** Wizard generates secrets, creates admin, writes config.
4.  **Production:** Core restarts in normal secure mode.

---

## Bootstrapping Workflow

### 1. Initial Boot

When the Gray Logic Core binary starts:
1.  Check for `/etc/graylogic/config.yaml` and `/etc/graylogic/secrets.yaml`.
2.  If missing, generate a **Claim Token** and print it to the console (stdout only).
3.  Start the Web Server in **Setup Mode**.
    - All standard API endpoints return `503 Service Unavailable`.
    - Only `/api/setup/*` endpoints are active.
4.  Advertise via mDNS (Bonjour): `GrayLogic-Setup-[Hostname]._http._tcp.local`.
5.  Start the **Setup Mode Timer** (24-hour timeout).

### Claim Token Specification

The Claim Token prevents unauthorized initialization of unconfigured systems:

```yaml
claim_token:
  # Token format
  format:
    type: "alphanumeric"
    length: 6                        # e.g., "A3F7K2"
    case: "uppercase"                # Easy to read/type
    exclude_ambiguous: true          # No 0/O, 1/I/L

  # Lifecycle
  lifecycle:
    expiry_minutes: 60               # Token expires after 1 hour
    rotation_interval_minutes: 15    # New token generated every 15 minutes
    single_use: true                 # Token invalidated after successful claim

  # Generation
  generation:
    algorithm: "CSPRNG"              # Cryptographically secure
    entropy_bits: 36                 # 6 chars * 6 bits each (36 possibilities)

  # Display
  display:
    console_format: |
      ══════════════════════════════════════════════════════════════
      GRAY LOGIC SETUP MODE

      Claim Token: {TOKEN}
      Expires: {EXPIRY_TIME} ({MINUTES_REMAINING} minutes remaining)

      Enter this token at: http://{IP_ADDRESS}:8080
      ══════════════════════════════════════════════════════════════
    refresh_display: true            # Re-print on rotation
    log_level: "INFO"                # Visible in normal logs

  # Security
  security:
    never_log_after_claim: true      # Remove from logs after successful claim
    rate_limit_attempts: 5           # Max 5 wrong attempts per 15 minutes
    lockout_minutes: 15              # Lockout after 5 failures
```

### Setup Mode Timeout

Setup Mode has a hard timeout to prevent abandoned installations from remaining vulnerable:

```yaml
setup_mode:
  timeout:
    duration_hours: 24               # Max time in setup mode

  timeout_behavior:
    action: "reboot_to_safe_mode"
    safe_mode:
      network: "disabled"            # No network access
      console_only: true             # Only local console access
      message: "Setup timed out. Reboot to restart setup process."

  # What happens on timeout
  on_timeout:
    1: "Log security event: setup_timeout"
    2: "Clear any partial configuration"
    3: "Disable network interfaces"
    4: "Display timeout message on console"
    5: "Require physical reboot to restart setup"

  # Rationale
  rationale: |
    An unclaimed system sitting on a network for days/weeks is a security risk.
    Even with claim token rotation, prolonged exposure increases attack surface.
    24 hours is sufficient for any legitimate installation scenario.
```

### 2. Accessing the Wizard

The installer connects to the server IP (e.g., `http://192.168.1.50:8080`).

**Security Gate:**
The UI prompts for the **Claim Token**.
> *"Please enter the Claim Token printed in the server logs."*

This proves physical access/administrative control of the server.

### 3. The Setup Wizard

Once the token is validated, the Wizard guides the user:

#### Step A: System Identity
- Site Name (e.g., "Oak Street")
- Location (Geolocation for sunrise/sunset)

#### Step B: Admin Account
- Username
- Password (enforce complexity: min 12 chars)

#### Step C: Infrastructure Security (Auto-Generated)
The Wizard locally generates:
- **JWT Secret:** 32-byte random hex.
- **MQTT Password:** 16-byte random.
- **Database Key:** 32-byte random.
- **Bridge Credentials:** Unique passwords for KNX/DALI bridges.

### 4. Provisioning

The UI sends the payload to `POST /api/setup/provision`.

**Core Actions:**
1.  **Write Secrets:** Create `/etc/graylogic/secrets.yaml` (chmod 600).
2.  **Write Config:** Create `/etc/graylogic/config.yaml` with site details.
3.  **Configure MQTT:**
    - Generate `mosquitto_passwd` file.
    - Generate `mosquitto.acl` restricting anonymous access.
4.  **Initialize Database:** Run migrations, create Admin user.
5.  **Restart:** The Core process restarts itself.

### 5. Post-Setup

The Core restarts in **Production Mode**:
- `secrets.yaml` is loaded.
- Standard API endpoints active.
- Setup endpoints disabled.
- Anonymous access blocked.

---

## Automated Provisioning (Headless)

For fleet deployments (e.g., using Ansible), the First-Run Wizard can be bypassed by placing a pre-generated `provision.json` file in the config directory.

**File:** `/etc/graylogic/provision.json`

```json
{
  "site": {
    "name": "Automated Site",
    "timezone": "Europe/London"
  },
  "admin": {
    "username": "admin",
    "password_hash": "$argon2id$..."  // Pre-hashed password
  },
  "secrets": {
    "jwt_secret": "...",
    "mqtt_password": "..."
  }
}
```

On boot, if Core finds this file:
1.  Validates schema.
2.  Applies configuration.
3.  Deletes `provision.json`.
4.  Starts normally.

---

## Security Considerations

### Claim Token Security

1.  **Token Exposure Window:** The token is only visible in local logs for a maximum of 1 hour before rotation. This limits the window for log-based attacks.
2.  **Token Rotation:** Every 15 minutes, a new token is generated. Old tokens are immediately invalidated. An attacker who captures a token from logs has at most 15 minutes to use it.
3.  **Ephemeral Display:** The Claim Token is printed to `stdout` for console visibility. It SHOULD NOT be written to persistent logs (e.g., journald) where possible. If logged by the supervisor, the 15-minute rotation serves as the primary mitigation.
4.  **Rate Limiting:** 5 failed claim attempts trigger a 15-minute lockout, preventing brute-force attacks on the 6-character token.

### Network Exposure

1.  **Setup Mode Binding:** The API binds to `0.0.0.0` to allow LAN access, but the Claim Token prevents unauthorized initialization.
2.  **24-Hour Timeout:** Setup mode automatically disables after 24 hours, preventing indefinite network exposure.
3.  **mDNS Advertisement:** The system advertises via mDNS only during setup mode; this stops after provisioning.

### Secret Quality

1.  **CSPRNG Required:** All generated secrets (JWT, MQTT, DB keys) must use a Cryptographically Secure Pseudo-Random Number Generator.
2.  **Entropy Requirements:**
    - JWT Secret: 256 bits (32 bytes)
    - MQTT Password: 128 bits (16 bytes)
    - Database Key: 256 bits (32 bytes)
    - Claim Token: 36 bits (sufficient for 15-minute window with rate limiting)

### Threat Model

| Threat | Mitigation |
|--------|------------|
| Attacker reads claim token from logs | 1-hour expiry + 15-min rotation limits window |
| Attacker brute-forces claim token | Rate limiting (5 attempts / 15 min lockout) |
| System left in setup mode indefinitely | 24-hour timeout → safe mode |
| Attacker on same network during setup | Claim token required; cannot proceed without it |
| Installer forgets to complete setup | Timeout forces attention; system becomes safe |

---

## Recovery (Lost Admin Password)

If the admin password is lost:

1.  SSH into the server.
2.  Run the CLI command:
    ```bash
    graylogic users reset-password admin
    ```
3.  Enter new password when prompted.

(This relies on SSH access security).
