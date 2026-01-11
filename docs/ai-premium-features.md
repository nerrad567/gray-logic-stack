# AI Premium Features — Gray Logic Stack (Draft)

**Date**: 2026-01-11  
**Status**: Design notes / product boundaries

## 1. Purpose

This document defines where “AI” can be used in the Gray Logic Stack **without** breaking the core product rules:

- Offline-first (remote services are bonuses, not dependencies)
- Physical controls always remain valid
- Life safety remains independent
- Consumer Overlay remains best-effort

It is written as an internal reference to keep future implementation and marketing consistent.

## 2. What “AI” Means Here

In this context, “AI” is used for **optional insights and summarisation** on top of existing data (PHM metrics, alarms, service health).

AI is **not** a second automation brain. openHAB and Node-RED remain the control layer.

## 3. Hard Guardrails (Must Not Break)

1. **Advisory, not authority**

   - AI may suggest or summarise.
   - AI must not directly execute plant/security actions.
   - Any change to modes, setpoints, or plant behaviour uses the existing control paths (openHAB/Node-RED) with explicit confirmation and audit logging.

2. **No cloud-only dependency for core operation**

   - If AI services (local or remote) are down, core operation continues:
     - lighting/scenes/modes
     - plant control
     - PHM checks and early-warning flags
     - local UI on the LAN

3. **Data minimisation by default**

   - “AI insights” defaults to using **aggregated metrics and events**.
   - Sensitive categories are never exported off-site by default (see section 5).

4. **Auditability**

   - AI-generated recommendations that are shown to users should be logged (what was suggested, when, and what data category it was based on).
   - Any user-confirmed actions remain logged via the normal audit pipeline.

## 4. Candidate AI Features (Optional)

### 4.1 PHM anomaly triage (Premium)

Goal: reduce “what does this warning actually mean?” time.

- Input: existing PHM flags/events + a small window of relevant metrics (e.g. pump current + temperature + run command).
- Output: plain-English explanation and suggested checks (e.g. “check strainer basket / filter pressure / bearing noise”).
- Constraint: the underlying PHM detection remains deterministic and on-site; AI assists interpretation.

### 4.2 Weekly / monthly health digest (Enhanced-lite or Premium)

Goal: a simple, low-noise summary for clients and support.

- Enhanced-lite version: template-based digest driven by counts, durations, and thresholds.
- Premium version: AI-assisted summarisation that clusters related events and highlights trend changes.

### 4.3 “What changed?” reporting (Premium)

Goal: help explain drift without implying certainty.

- Examples:
  - “Heating flow temperature is taking longer to reach setpoint than last month.”
  - “Pump run duration is higher than its 7-day baseline.”

### 4.4 Support-facing troubleshooting assistant (internal)

Goal: speed up support work using the site runbook + known patterns.

- Reads local runbook content and recent fault summaries.
- Suggests next checks (purely advisory).
- Never runs commands or changes configuration.

### 4.5 Weather nowcast explanation (Premium)

Goal: turn the Weather Nowcast “products” (imagery/loop, freshness, basic heuristics) into a short, plain-English summary.

- Input: on-site weather products and timestamps (no raw CCTV/security/occupancy data).
- Output: “what it looks like now” + “what may happen in the next 0–2 hours” with explicit uncertainty.
- Constraint: ingest and any heuristics remain deterministic and on-site; AI only assists interpretation and wording.

### 4.6 Mesh comms health / incident digest (Premium)

Goal: make out-of-band mesh comms issues understandable without exposing message payloads.

- Input: mesh “comms products” (health badge, last-heard, delivery/error counters, backlog age) + relevant timestamps.
- Output: plain-English summary (e.g. “Mesh gateway hasn’t been heard for 45 minutes; messages may not deliver. Check power/antenna and local interference.”).
- Constraint: metadata/health only; do not use or export message payloads by default.

## 5. Data Handling Defaults (Never Export by Default)

“Never export” means: **do not send off-site (VPS/cloud/third parties) as part of the default product telemetry**.

Default never-export categories:

- **CCTV media**: video, audio, recordings, and still images/snapshots
- **Occupancy/presence history**: any timeline that could reconstruct “who was home when”
- **Detailed security timelines**: zone-by-zone history, door events, motion event lists
- **Secrets**: passwords, API tokens, private keys, WireGuard configs
- **Raw network identifiers**: MAC/IP client lists, device fingerprinting scans
- **Raw logs containing payloads**: prefer exporting only minimal, structured health/PHM events
- **Mesh message payloads**: message contents from out-of-band comms; export/retention is opt-in per site

Opt-in exceptions (explicit per site):

- Low-res, low-frequency camera snapshots for *status only* (e.g. relay quick view)
- Expanded logging for diagnostics during a time-boxed support window

## 6. Placement: On-Site vs Remote

- **Always on-site**:
  - openHAB / Node-RED control logic
  - PHM detection and early-warning flags
  - Local UI availability

- **Optional remote bonuses**:
  - Long-term retention of aggregated metrics/events
  - Premium “AI insights” that benefit from multi-month/multi-year history

The offline-first rule applies: remote features can pause; on-site continues.

## 7. Support Tier Mapping (Recommended)

- **Core Support**
  - No AI features required.

- **Enhanced Support**
  - PHM alerts + remote monitoring.
  - Optional “weekly digest” in a template-based form (kept simple, low-risk).

- **Premium Support**
  - Optional AI-assisted insights/reporting (PHM triage, trend summaries, long-term comparisons).
  - Still advisory; still not a dependency for control.

## 8. Not Supported

- AI-driven autonomous control of plant or security
- Exporting CCTV media or behavioural timelines by default
- Any design where loss of internet/VPS degrades core operation

## 9. References

- [docs/gray-logic-stack.md](gray-logic-stack.md)
- [docs/architecture.md](architecture.md)
- [docs/sales-spec.md](sales-spec.md)
- [docs/business-case.md](business-case.md)
