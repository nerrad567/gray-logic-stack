# Audit Iteration 5: Holistic Consistency Audit

**Date:** January 18, 2026
**Auditor:** Antigravity (Gemini Agent)
**Focus:** Integration consistency, race conditions, logic gaps, unresolvable dependencies.

---

## 1. Executive Threat Summary

The documentation is high quality (Readiness ~9.0/10), but five distinct implementation risks remain that could cause catastrophic data loss or system deadlocks in the long term.

1.  **"The Time-Travel Deadlock" (High)**
    If the Core's RTC battery fails and internet is down (common 10-year scenario), the Core will boot with a 1970 timestamp. The "Clock Skew Protection" (`offline.md`) explicitly rejects all update timestamps >60s. Bridges (which might have valid time or their own drift) will be permanently rejected, locking the system in a "Data Rejected" loop until manual intervention.

2.  **"The Forgotten Archive" (High)**
    `maintenance.md` specifies that logs >1 year are moved to `/var/lib/graylogic/archive/`. However, `backup.md`'s automation script **only** backs up the SQLite database and configuration files. The archive directory is completely excluded from the backup strategy. A disk failure in Year 5 would result in the silent loss of Years 1-4 of security/audit logs, violating the "10-year stability" logic.

3.  **"The USB Schrodinger" (Medium)**
    `backup.md` lists USB backups as "Optional". `commissioning-checklist.md` lists a USB "Gold Master" left on site as "MANDATORY". This ambiguity invites installers to skip the USB drive, leaving the client with no physical recovery media if the cloud/remote backup fails or is never configured.

4.  **"Phantom Energy Loads" (Medium)**
    `energy.md` describes shedding loads by `device_id` (e.g., `fridge-kitchen`). However, `entities.md` defines appliances as potential "virtual" concepts unless on a smart plug. If logic tries to `shed` a device that is just a "monitored" entity (via CT clamp) rather than a "controlled" entity (relay/plug), the automation will fail silently or throw errors. The spec lacks the "Control Proxy" explicit requirement for `shed_loads`.

5.  **"The Broken Map" (Low)**
    `operations/maintenance.md` and `operations/infrastructure.md` depend on `operations/backup.md`, which does not exist. The file is actually `resilience/backup.md`. This breaks the documentation navigation tree.

---

## 2. Readiness Score: 9.0 / 10

**Verdict:** **Go for Code** (Conditioned on fixing the Time and Backup gaps).
We are extremely close. The data model, offline logic, and security model are robust. The findings today are subtle integration seams that only appear when cross-referencing multiple subsystems (Time vs Offline, Backup vs Maintenance). Fixing these will bring us to near-perfection.

---

## 3. Findings Table

| Severity | Issue | File / Section | Ref / Quote | Why It Hurts | Fix |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **HIGH** | **Archive Data Loss** | `resilience/backup.md` / SQLite Backup | `script: ... sqlite3 ... .backup` (Exclusive) | Log Archives (`/var/lib/graylogic/archive`) defined in `maintenance.md` are NEVER backed up. Disk failure = 100% loss of history >1 year. | Add `tar -rf ... /var/lib/graylogic/archive` to the backup script. |
| **HIGH** | **Time Sync Deadlock** | `resilience/offline.md` / Clock Skew | `Reject bridge timestamp ... >60s from Core clock` | If Core RTC fails (1970), valid Bridge data (2026) is rejected forever. System bricks itself. | Add "Trust on First Use" or "RTC Confidence" logic. If Core < 2025, accept ANY external timestamp > 2025. |
| **MED** | **USB Backup Contradiction** | `backup.md` vs `commissioning.md` | `usb: enabled: true ... (optional)` vs `MANDATORY ... USB stick` | Installers will skip the USB drive, violating the "Physical Handover" principle. | Change `backup.md` to `usb: required_for_handover`. |
| **MED** | **Load Shedding Target Ambiguity** | `domains/energy.md` / Load Management | `device_id: "fridge-kitchen"` | A fridge on a CT clamp cannot be shed. Logic implies control where none may exist. | Explicitly require `target_device_id` in Energy Load config to be a *controllable* switch/relay. |
| **LOW** | **Broken Doc Link** | `operations/maintenance.md` | `depends_on: - operations/backup.md` | File does not exist. (Is in `resilience/backup.md`). | Update path. |
| **ENHANCE**| **Offline Tariff Awareness** | `domains/energy.md` / Smart Meter | *New Feature* | User noted reliance on cloud API for pricing. | Added `LocalSmartMeter` spec to read HAN/P1 tariff index for offline price switching. |

---

## 4. Surgical Strikes (Next Steps)

These are the final polish tasks before we start the `cmd/graylogic/main.go` file.

1.  **Strike 1: The "History Saver" Update**
    *   **Goal:** Ensure long-term log archives are backed up.
    *   **Action:** Update `resilience/backup.md` scripts to include `/var/lib/graylogic/archive/`.
    *   **Action:** Update `operations/maintenance.md` link to `resilience/backup.md`.

2.  **Strike 2: The "Time Lord" Protocol**
    *   **Goal:** Prevent 1970-boot deadlock.
    *   **Action:** Update `resilience/offline.md` "Clock Skew Protection" to include a "Sanity Check" exception: "If Core Time < Build Date, accept Bridge Time and set System Time."

3.  **Strike 3: The "Mandatory USB" & Energy Logic**
    *   **Goal:** Align Handover specs and Energy control logic.
    *   **Action:** Update `resilience/backup.md` to mark USB as `required_for_commissioning`.
    *   **Action:** Update `domains/energy.md` to verify `shed_loads` targets allow control capabilities.
