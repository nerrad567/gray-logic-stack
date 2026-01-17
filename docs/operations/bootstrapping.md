---
title: Bootstrapping & First-Run Security
version: 1.0.0
status: active
last_updated: 2026-01-17
depends_on:
  - architecture/security-model.md
  - architecture/core-internals.md
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
2.  If missing, generate a **Claim Token** (UUIDv4) and print it to the console (stdout/logs).
3.  Start the Web Server in **Setup Mode**.
    - All standard API endpoints return `503 Service Unavailable`.
    - Only `/api/setup/*` endpoints are active.
4.  Advertise via mDNS (Bonjour): `GrayLogic-Setup-[Hostname]._http._tcp.local`.

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

1.  **Claim Token Exposure:** The token is only visible in local logs. If an attacker has access to logs, the box is already compromised.
2.  **Network Exposure:** In Setup Mode, the API should bind to `0.0.0.0` to allow LAN access, but the Claim Token prevents unauthorized initialization.
3.  **Secret Quality:** All generated secrets must use a Cryptographically Secure Pseudo-Random Number Generator (CSPRNG).

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
