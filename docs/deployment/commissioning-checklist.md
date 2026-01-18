---
title: Gray Logic Commissioning Checklist
version: 1.0.0
status: active
last_updated: 2026-01-18
---

# Commissioning Verification Checklist

**Mandatory Requirement for Warranty Validity**

This checklist must be completed by the lead integrator before the system is handed over to the client. It verifies that the "Hard Rules" of Gray Logic—specifically safety, offline capability, and resilience—are functionally active.

## Site Information

| Field | Value |
|-------|-------|
| **Project Name/ID** | __________________________________________________ |
| **Site Address** | __________________________________________________ |
| **Lead Integrator** | __________________________________________________ |
| **Commissioning Date** | __________________________________________________ |
| **Core Version** | __________________________________________________ |

---

## Phase 1: Resilience & Infrastructure

### 1.1 Clock Trust (Time Synchronization)
*Reference: [Strike 2] docs/resilience/offline.md*

- [ ] **Core Time**: Verified Core system time is correct (UTC).
- [ ] **Bridge Sync**: Verified ALL bridges (KNX, DALI, Modbus) are synced to Core time (limit: <5s deviation).
    - *Command used*: `date` check via SSH on bridges or status API.
- [ ] **NTP Configuration**: Confirmed local NTP source configuration for all devices (no cloud reliance).

### 1.2 Offline Capability
*Reference: docs/overview/principles.md*

- [ ] **WAN Disconnect Test**: Physically disconnected WAN uplink.
- [ ] **Local Control**: Verified mobile app works locally (WebSocket connects).
- [ ] **Voice Control**: Verified local voice commands (Piper/Whisper) function without internet.
- [ ] **Automation**: Verified scene triggers function without internet.
- [ ] **Recovery**: Reconnected WAN, verified graceful recovery (no restart required).

---

## Phase 2: Life Safety & Protection (CRITICAL)

> [!CAUTION]
> **STOP.** Do not proceed until Phase 2 is fully verified. These checks prevent property damage and are **MANDATORY**.

### 2.1 Frost Protection Hardware Gate
*Reference: docs/domains/climate.md*

**The "Hard Rule": Frost protection must work if Gray Logic Core is physically destroyed.**

- [ ] **Thermostat Audit**: List all thermostat models installed.
    - Model: _______________________ | Built-in Frost Mode? [Yes/No]
    - Model: _______________________ | Built-in Frost Mode? [Yes/No]
- [ ] **Hardware Configuration**: Verified frost protection threshold is configured ON THE DEVICE (typically 5°C), not just in software.
- [ ] **Functional Test**: Simulated bus failure (disconnected KNX/Modbus cable) to one thermostat and verified it remains powered and sensing.

**Integrator Declaration:**
> "I have verified that the thermostat models listed above have built-in hardware frost protection active at the hardware/firmware level, configured to a safe threshold (e.g., 5°C). I confirm this protection operates independently of the Gray Logic Core."
> 
> **Signed:** ___________________________________ **Date:** _____________

### 2.2 Fire & Safety
- [ ] **Independent Operation**: Verified smoke/heat detectors trigger alarms directly (e.g., KNX interlink) without passing through Core logic.
- [ ] **E-Stop**: Verified any physical E-Stops cut power to dangerous loads functionality at the hardware relay level.

---

## Phase 3: Domain Verification

### 3.1 Climate
- [ ] **Status Feedback**: Verified that manual adjustments on thermostats flow back to the UI (2-way sync).
- [ ] **Fail-Safe**: Verified heating valves close (or open, if design requires) on power loss.

### 3.2 Lighting & Shading
- [ ] **Status Feedback**: Verified manual wall switch toggles update the UI status.
- [ ] **Blind Safety**: Verified wind protection (if applicable) retracts blinds hardware-direct (weather station -> actuator direct link).

### 3.3 Power & PHM
- [ ] **Sensor Sanity**: Checked PHM dashboard for "impossible" readings (e.g., 0W power on running pump).
- [ ] **Baseline Training**: Initiated PHM learning mode for critical equipment.

---

## Phase 4: Handover

- [ ] **Handover Pack**: Generated and printed/PDF'd `docs/deployment/handover-pack-template.md`.
- [ ] **User Access**: Transferred admin credentials to owner.
- [ ] **Backup**: USB stick with "Gold Master" full system backup physically left on site.

---

**Commissioning Complete**

**Integrator Signature:** ___________________________________
