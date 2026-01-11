# Handover Pack Template — Gray Logic Stack (Draft)

**Status**: Template / working draft

This document is a **template** for the site handover pack. It is intended to be delivered per-site with project-specific details filled in.

## 1. Cover Sheet

- Site name / slug:
- Address (if appropriate):
- Install / commissioning date:
- Primary installer:
- Support tier (if any): Core / Enhanced / Premium
- Key contacts (client / facilities / installer):

## 2. Scope & Hard Boundaries (Must Not Drift)

- **Offline-first**: Core operation continues with no internet.
- **Physical controls always remain valid**: wall switches and plant room controls must still work during LAN/server outages.
- **Life safety is independent**: fire alarms, emergency stops, emergency lighting remain on certified hardware; the stack may observe signals but never controls them.
- **Consumer Overlay is best-effort**: convenience devices must never become critical dependencies.

## 3. System Inventory (As Installed)

### 3.1 Hardware

- On-site node model/serial:
- Storage (SSD size, RAID if any):
- UPS model and runtime estimate:
- Network gear (router/switch/APs):

### 3.2 Network

- VLANs/subnets/IP plan:
- DNS/DHCP notes:
- Firewall posture (egress allowlist notes, if used):

### 3.3 Software / Container Stack

- OS version:
- Docker + Compose versions:
- Container list + pinned image tags:
- Primary endpoints (LAN URLs):

## 4. Access & Credentials Policy (No Secrets In This Document)

This pack must not contain plaintext secrets.

- Where credentials are stored (choose one):
  - Encrypted password manager vault (preferred)
  - Sealed printed envelope (fallback)
  - Encrypted USB (fallback)

### 4.1 “God Mode Exports” Checklist (Sealed & Encrypted)

- Host (Linux) admin access
- Docker/Compose project + any secrets handling notes
- WireGuard configs (site + admin users)
- openHAB configs (things/items/rules/UI)
- Node-RED flows + credentials
- KNX project file(s) (if applicable)
- Plant controller exports (VFD/Modbus mappings, register lists) (if applicable)

### 4.2 Ownership

Record who owns and can recover:

- Domains / DNS
- Certificates/ACME accounts (if used)
- VPS account (if any)
- Third-party vendor accounts (NVR, alarm vendor portals, etc.)

## 5. Backup, Restore & Disaster Recovery

- Backup scope:
- Backup schedule:
- Backup storage locations:
- Retention:
- Restore drill date + outcome:

**Minimum safe state guidance:** If the stack node is down, the site must remain safe and usable using physical controls and independent life safety systems.

## 6. Data Sources, Licensing & Terms

This section exists to keep the installation within legal/contractual boundaries, and to make the client aware of any third-party terms.

For each external data source or subscription, record:

- Provider name:
- Product/dataset name:
- Access method (satellite ingest / internet API / hybrid):
- Licence summary:
- Attribution requirements:
- Caching/retention limits:
- Redistribution constraints:
- Rate limits (if applicable):
- SLA/availability expectations (if any):

### 6.1 Weather Nowcast (Satellite Ingest + Optional Internet Enrichment)

- Source(s):
- Licensing/terms notes:
- Local retention policy:
- Redisplay rules (eg. client dashboard only):

**Important disclaimer:** Weather/nowcast features are informational and advisory only. They must not be relied upon for safety-critical decisions, and loss of weather input must not affect core operation.

### 6.2 Out-of-Band Mesh Comms (Meshtastic-like)

- Purpose (site-specific):
- Hardware inventory (nodes, repeaters/high-site node, antennas, power):
- Region/band plan:
- Configuration ownership (commissioning vs ongoing support):
- Local payload retention (opt-in?) and retention period (days, e.g. 7/30/90):
- Off-site export posture (default: none):

Compliance notes:

- Confirm radio hardware is suitable for UK use (e.g. UKCA/CE where required).
- Confirm transmit power, antenna selection/placement, and region settings are configured to remain within licence-exempt SRD rules.

**Important disclaimer:** Mesh comms are best-effort convenience features. They are not emergency/life-safety signalling and must not be relied upon as a primary alarm path.

## 7. Support & Change Control


- Link/reference to signed support agreement:
- What is included vs change request:
- Planned update cadence (if any):
- Audit logging expectations:

## 8. Exit Path / Doomsday Appendix (Non-Secret)

- Location of sealed credential package:
- Dead Man’s Switch clause reference (contract location):
- “Yellow Pages” list (alternative integrators):

