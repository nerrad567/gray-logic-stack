# Sales Specification — Gray Logic Stack (Draft v0.4)

**Evergreen note:** This document is a planning/reference specification. Final scope, deliverables, pricing, and support terms are confirmed in the written quote and support agreement for each site.

## 1. Purpose (What This Document Is For)

This is the **canonical, technical sales specification** for the Gray Logic Stack.

It exists so we can later generate consistent marketing outputs (website copy, ads, social posts, print, proposals) **without drifting** away from the actual product boundaries.

This document is written to be:

- Detailed enough to drive accurate marketing.
- Honest about boundaries, failure modes, and constraints.
- Compatible with the “offline-first / internal vs external” split used throughout the stack documentation.

See also:

- [docs/gray-logic-stack.md](gray-logic-stack.md) (technical spec)
- [docs/business-case.md](business-case.md) (commercial reasoning; internal)
- [docs/architecture.md](architecture.md) (routing/security model)
- [docs/ai-premium-features.md](ai-premium-features.md) (optional AI features + data-handling defaults; internal)

## 2. One-Sentence Positioning

> “Mini-BMS quality, using open standards, for sites that are too small or too specialised for full BMS/SCADA, and too serious for toy smart home setups — designed offline-first, with optional remote and predictive-health bonuses layered on top.”

## 3. Target Sites (Fit Checklist)

Good fit:

- High-end homes / small estates (new builds or deep refurbs).
- Pools / spas / leisure sites with real plant (pumps, boilers, filtration, ventilation).
- Light commercial / mixed-use sites that want visibility + maintainability without full BMS/SCADA.

Usually not a fit (or requires re-scoping):

- Large commercial/industrial BMS/SCADA environments requiring redundant servers, 24/7 cover, and formal compliance reporting.
- Projects that require cloud dependency for core operation.
- “Bring any random consumer gadget and make it mission-critical” style projects.

## 4. Hard Rules (Must Match Technical Spec)

These phrases are intentionally reused verbatim from the technical spec:

1. **Physical controls always remain valid**

   - Wall switches, panic buttons, safety overrides, plant room controls must work even if:
     - The server is off.
     - The LAN is down.
     - The VPN / internet is down.
   - Automation **observes** physical state and follows it, not the other way around.

2. **Life safety is independent**

   - Fire alarm systems, emergency lighting, emergency stop circuits:
     - Use their own certified hardware, wiring, and power supplies.
     - May send signals into the Gray Logic Stack (e.g. a global “fire” relay),  
       but are never **controlled by** the Gray Logic Stack.
   - No remote reset/silence of fire alarms through the stack.

3. **No cloud-only dependencies for basic operation**

   - The site must continue to function if:
     - The internet is down.
     - A vendor cloud disappears.
   - Cloud/VPS is an **enhancement** (remote monitoring, backups, dashboards, updates), not a requirement.

4. **Consumer Overlay is non-critical**

   - The Consumer Overlay is always treated as:
     - Non-critical.
     - Best-effort.
   - It must not become a prerequisite for:
     - Life safety.
     - Basic lighting safety.
     - Core plant operation.

5. **Offline-first, remote as bonus**

   - At least **99% of everyday features** should keep working with the internet down:
     - Scenes, schedules, modes.
     - Local UI on the LAN.
     - Plant and lighting control.
   - Remote services (VPS, cloud APIs) are treated as **nice-to-have bonuses**, not hard dependencies.

## 5. Architecture Summary (What Gets Installed)

The Gray Logic Stack is delivered as an on-site Linux/Docker “mini-NOC” running:

- openHAB (automation brain)
- Node-RED (cross-system flows, PHM logic)
- Traefik (reverse proxy/front door)
- Mosquitto (optional MQTT broker for loose coupling)

Recommended network segmentation:

- Control/HMI network (Gray Logic node, controllers, gateways)
- CCTV/NVR network (cameras + NVR)
- Guest/general LAN
- IoT/Consumer network (if used)

Remote access model:

- Local-first access on site LAN.
- Remote access via VPN (WireGuard) as the default secure posture.
- Avoid direct public exposure of control ports.

## 6. What’s Included (Core, On-Site)

The “core” package is designed to keep working offline and typically includes:

- On-site control node (Linux + Docker)
- Core automation logic (modes/scenes/schedules)
- Integrations for open standards where applicable (KNX/DALI/Modbus/dry contacts)
- Local UI access on the LAN
- Local logging/history for key metrics (short-to-medium term)
- A documented handover pack (see section 11)

## 7. Optional Enhancements (Remote Bonuses)

Optional remote capabilities (where commissioned) include:

- Remote monitoring and alerts
- Remote access for tweaks/maintenance (via VPN)
- Long-term retention (years) of metrics for trends and PHM
- Multi-site dashboards (where relevant)
- Optional weather nowcast display (satellite-derived where licensed/feasible) with internet enrichment (not required); advisory/comfort use only
- Optional out-of-band mesh comms (Meshtastic-like) with a long-range node for coverage and integrated user messaging; remote access only via VPN; best-effort, not emergency/life-safety
- (Premium) Optional **AI-assisted insights and reporting** on top of PHM/trend data (advisory only; never required for control)

Default data-handling posture for remote bonuses:

- Remote monitoring uses **aggregated health/PHM signals**, not private household timelines.
- CCTV media (video/audio/snapshots) and detailed security/occupancy histories are **not exported off-site by default**.

If remote services are unavailable, the building continues to run; only these bonuses pause.


## 8. Predictive Health Monitoring (PHM)

PHM is a **condition-based early warning** approach for suitable plant.

What it is:

- Baseline “heartbeat” metrics (e.g., temperatures, current draw, run hours, flows).
- Rolling averages (e.g., 7-day baselines) to understand “normal”.
- Deviation thresholds sustained over time (e.g., ≥20% deviation for >2 hours) to reduce false positives.
- Alerts that help maintenance become proactive.

What it is not:

- A guarantee of “no breakdowns”.
- A substitute for maintenance schedules, servicing, or safety compliance.

PHM runs on-site; remote services mainly add nicer dashboards, alert routing, and long-term trend retention.

## 9. CCTV / NVR (Local-First)

CCTV is in-scope when it is **local-first** and supports stable local interfaces.

Supported pattern:

- Segregated CCTV VLAN/subnet.
- Local NVR (on-site, not cloud-hosted).
- Cameras and door intercoms that support local standards (e.g., RTSP/ONVIF and/or SIP where relevant).
- Optional integration points:
  - System health/availability (NVR up, camera streams alive).
  - Events/alerts where supported.

Not supported as part of the core package:

- Cloud-only consumer ecosystems with limited/no stable local interfaces.
- Any solution where core security/operation depends on a vendor cloud.

## 10. Consumer Overlay (Best-Effort)

Consumer and cloud-heavy devices can be integrated via an optional **Consumer Overlay**, which is:

- Segregated (network + logical separation).
- Best-effort.
- Never allowed to become critical to safety, core lighting safety, or core plant operation.

This is where “nice-to-have” devices can live without contaminating the reliability model.

## 11. Deliverables (Documentation & Handover Pack)

Each deployment should have a handover pack including:

- Template reference: [docs/handover-pack-template.md](handover-pack-template.md)

- Electrical schematics and panel schedules (where applicable).
- KNX/DALI group tables and I/O maps.
- Network diagram (VLANs, addressing, VPN).
- Site runbook covering:
  - How to restart services safely.
  - Where backups go and how to restore.
  - Contact details and support process.
  - A client summary page explaining:
    - Core stack vs Consumer Overlay.
    - On-site vs remote responsibilities.
    - What is considered critical vs best-effort.
    - What happens during an internet outage.
    - PHM gives early warning, not guarantees.

## 12. Support Model

### 12.1 Included Commissioning Support (Typical)

Included Commissioning Support is **typically ~3 months post-handover**, intended for snagging/teething issues and minor tuning.

- Cost is absorbed into the initial project pricing.
- Scope is intentionally limited (defects and minor configuration changes).
- New features, major change requests, third-party vendor changes, and site visits are quoted separately.
- This is not a promise of lifetime or unlimited support.

### 12.2 Optional Ongoing Support Plans (Add-ons)

Ongoing support is available under optional plans. These are described using stable tier names:

- **Core Support** — on-site focused basics (no VPS required)
- **Enhanced Support** — adds PHM alerts + remote monitoring (VPN/VPS bonuses)
- **Premium Support** — deep PHM + long-term retention + reporting (remote bonuses); may include optional AI-assisted insights

Exact inclusions, response times, and pricing are defined per site in the support agreement.

### 12.3 Third-Party / Client-Supplied Hardware Integration

Core principle:

- If it is supplied/installed as part of the Gray Logic installation package, it is supported.

Client-supplied / third-party hardware is not included in the core package.

We can offer an **Integration Discovery & Implementation** service where:

- We assess feasibility (interfaces, local APIs, protocol support).
- We provide a written estimate before work begins.
- We are explicit that vendor lock-in and cloud dependency can reduce capability or make integration impractical.

## 13. Scope Assumptions Checklist (Anchors for Indicative Pricing)

Indicative pricing examples assume:

- A site with stable LAN and basic segmentation capability.
- A clear scope for modules and integrations (not “integrate everything in the house”).
- A documented list of plant assets and required points (I/O map / Modbus register list / KNX group table).
- Suitable interfaces exist (or are included) for required integrations.
- The client accepts the offline-first model and boundaries.

## 14. Indicative Costs (Non-Final)

These are planning numbers only and will change with scope, site conditions, and integration depth.

- Typical core design + implementation: **~£10–20k**
- Typical optional modules (overlay/CCTV/plant/remote setup): **~£2–5k+**

Optional ongoing support (planning numbers only; agreement defines reality):

- Core Support: from low £/month (or annual)
- Enhanced Support: £X–£Y/month
- Premium Support: £Y+–£Z/month

Third-party integration service (planning reference):

- Integration Discovery & Implementation: priced as a scoped piece of work. As a planning number, expect engineering effort in the **~£750/day-equivalent** range, with a written estimate provided once the target device(s) and requirements are known.

## 15. Example Copy (Non-Binding)

These blocks are suggestions for marketing generation; they are not contractual language.

- **Positioning (short):**
  “A safety-first, offline-first automation stack built on open standards — designed and installed like a professional electrical job, with proper runbooks, backups, and a clear exit path.”

- **Offline-first (reassurance):**
  “If the internet goes down, your home/building keeps working normally. Remote dashboards and extras pause — the core doesn’t.”

- **PHM (honest):**
  “We don’t promise magic. We do provide early warning when plant starts behaving differently from its normal baseline — so you can fix problems before they become failures.”

- **Lock-in (differentiator):**
  “Open protocols, clear documentation, and a handover pack — so you’re not trapped in a dealer-only ecosystem.”

## 16. Example Site Profiles (Archetypes)

1. **High-end new build (residential)** — lighting scenes, modes, heating/plant overview, minimal overlay.
2. **Home + pool plant room** — filtration/pumps/boilers with PHM, humidity/ventilation, early warning.
3. **Small leisure facility** — uptime focus, dashboards, alarms, remote visibility.
4. **Light commercial showroom/office** — lighting + environment monitoring + simple plant visibility.
5. **Small estate** — multiple outbuildings, CCTV/NVR segmentation, remote support add-ons.

## 17. Not Supported / Not Recommended (and Why)

These are excluded or discouraged because they conflict with offline-first, open-standards, and maintainability:

- Cloud-only consumer ecosystems with limited local interfaces (vendor lock-in, high support friction).
- Any attempt to control life safety systems via the stack.
- Architectures that require the internet/VPS for normal on-site operation.
- Making Consumer Overlay devices “mission critical”.

## Appendix A. Glossary

- **Consumer Overlay:** A segregated, best-effort layer for consumer/cloud-heavy devices.
- **Internal vs External:** On-site functions that must work offline vs optional remote bonuses.
- **PHM (Predictive Health Monitoring):** Baseline + deviation-based early warning on plant health.
