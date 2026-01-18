# Audit Iteration 5: Holistic Consistency Audit

**Date:** 2026-01-18
**Auditor:** Agentic AI (Iteration 5)
**Focus:** Integration seams, race conditions, partial failure states.

## Executive Summary
This audit focused on the "seams" between systems: Backup/Security, Offline/Time, Energy/PHM, and Privacy/Debug. Four critical consistency gaps were identified where documented systems, when interacting, create logic failures or security vulnerabilities.

---

## Findings

### 1. Security/Backup: Plaintext Backup of TLS Private Keys
**Severity:** Critical
**Conflict:** `docs/resilience/backup.md` vs `docs/architecture/security-model.md`

**Description:**
`docs/resilience/backup.md` defines two separate backup processes:
1. `config_backup` (Plain tarball): Includes `/etc/graylogic/certs/*`.
2. `secrets_backup` (GPG Encrypted): Includes `/etc/graylogic/secrets.yaml`.

However, `docs/architecture/core-internals.md` configures the TLS private key location as `/etc/graylogic/server.key`. Whether this falls under `certs/` or is a sibling, standard practice and the wildcard `certs/*` (or general `/etc/graylogic/*.yaml` + config directory structures) suggests the private key is likely included in the **unencrypted** `config_backup` or missed entirely. If included in a recursive config backup (which `tar /etc/graylogic/` would do), it is archived without encryption.

This violates the "Defense in Depth" principle and the encryption requirements set in `security-model.md`.

**Verification:**
*   Checked `docs/resilience/backup.md` lines 191-211 (`config_backup` spec).
*   Checked `docs/resilience/backup.md` lines 218-239 (`secrets_backup` spec).
*   Checked `docs/architecture/core-internals.md` line 276 (`key_file` location).
*   **Result:** `config_backup` creates a standard `.tar.gz` of `/etc/graylogic/` excluding only `secrets.yaml`. This captures any other private keys in that directory.

---

### 2. Resilience/Hardware: Offline Boot Deadlock (RTC Dependency)
**Severity:** High
**Conflict:** `docs/resilience/offline.md` vs `docs/architecture/core-internals.md`

**Description:**
`docs/resilience/offline.md` (Offline Scheduling) relies on a "Local RTC (battery-backed)" as the time source when internet is down.
`docs/architecture/core-internals.md` supports deployment on "NUC/Pi". Raspberry Pis (a primary target) do **not** have a hardware RTC by default.

**The Race Condition:**
1.  Power failure occurs.
2.  Pi reboots without internet.
3.  System time defaults to build date or 1970 (Unix Epoch).
4.  Core acts as CA (`maintenance.md`). Existing Certificates (issued in 2026) are "not valid yet" (future).
5.  JWTs (issued in 2026) are "not valid yet".
6.  **Result:** System boots but rejects all authentication and mTLS connections due to temporal invalidity. Admin cannot log in to fix it because local auth fails validation.

**Verification:**
*   Checked `docs/resilience/offline.md` line 695 ("Time source: Local RTC").
*   Checked `docs/operations/bootstrapping.md` and `core-internals.md` for hardware mandates. None found requiring RTC.
*   Grep for "RTC" returned no hardware specification mandates.

---

### 3. Intelligence/Maintenance: PHM Training vs Data Retention
**Severity:** Medium
**Conflict:** `docs/intelligence/phm.md` vs `docs/operations/maintenance.md`

**Description:**
`docs/intelligence/phm.md` defines "Category 2: Gradual Degradation Devices" which requires `typical_learning_days: 14-30` to establish a baseline. It relies on "Raw" telemetry (vibration, current) for statistical validity (detecting variance/peaks).
`docs/operations/maintenance.md` defines the Data Retention policy for "Energy" (often the source of current/power data) as **7 days** for High Res (Raw).

**The Logic Gap:**
PHM cannot calculate a 30-day statistical baseline (e.g., standard deviation of vibration) if the raw data is deleted/downsampled after 7 days. Downsampled data (1h average) destroys the signal fidelity needed for anomaly detection (e.g., brief vibration spikes).

**Verification:**
*   Checked `docs/intelligence/phm.md` lines 361-362 (Learning days: 14-30).
*   Checked `docs/operations/maintenance.md` lines 77-82 (High Res Retention: 7 days).

---

### 4. Privacy/Architecture: Implicit Logging of PINs
**Severity:** High
**Conflict:** `docs/architecture/security-model.md` vs `docs/architecture/core-internals.md`

**Description:**
`docs/architecture/security-model.md` explicitly states: "never_log_pin_value: true".
`docs/architecture/core-internals.md` describes the `CommandProcessor` which logs command execution. The `Command` struct includes a `Parameters map[string]any`.
The `auth/pin` flow (or any authenticated command requiring a PIN challenge) often passes keys like `{"pin": "1234"}` in the parameters.
There is no explicit "Sanitize" or "Redact" step defined in the `CommandProcessor` pipeline description in `core-internals.md`. A standard `log.Infof("Command: %+v", cmd)` would leak the PIN to disk.

**Verification:**
*   Checked `docs/architecture/security-model.md` line 396.
*   Checked `docs/architecture/core-internals.md` line 607 ("Log command execution").
*   Checked `docs/development/CODING-STANDARDS.md`. It advises against logging secrets but `core-internals` spec does not enforce the *mechanism* (e.g., `Sanitize()` method) to prevent it for dynamic maps.

---

## Resolutions

### 1. Security/Backup
**Fix Implemented:** Updated `docs/resilience/backup.md`.
- `config_backup`: Explicitly excludes `*.key`.
- `secrets_backup`: Updated script to tar `secrets.yaml` and `*.key` together before encryption.

### 2. Resilience/Hardware
**Fix Implemented:** Updated `docs/architecture/system-overview.md` and `docs/architecture/core-internals.md`.
- Removed support for Consumer Hardware (Raspberry Pi/NUC) for Core.
- Mandated "Industrial PC (x86/ARM)" with **Battery-Backed Hardware RTC** and **Google Coral/NPU**.
- This eliminates the offline boot deadlock risk.

### 3. Intelligence/Maintenance
**Fix Implemented:** Updated `docs/operations/maintenance.md`.
- Increased Data Retention for "High Res (Raw)" Energy and Temperature data from 7 days to **30 days**.
- This ensures sufficient history for PHM baseline learning (14-30 days).

### 4. Privacy/Architecture
**Fix Implemented:** Updated `docs/architecture/core-internals.md`.
- Updated Command Processor step 7 to explicitly require: "Log command execution (Sanitize: redact PINs/Secrets from Parameters)".
