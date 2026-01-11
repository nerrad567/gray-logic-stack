
# Module: Core

**Status**: Draft

## Purpose

Provide the consistent on-site “front door” (routing + auth), health visibility, and operational resilience features that make the Gray Logic Stack supportable.

This module is infrastructure-first and must preserve the stack hard rules:

- Offline-first: on-site operation continues without internet.
- Physical controls remain valid.
- Life safety remains independent.

## Core Responsibilities (On-Site)

- Traefik routing, TLS termination, and access controls
- Local UI availability (openHAB Main UI/HABPanel plus Gray Logic UI surfaces)
- Service health visibility (host + container status, key integrations)
- Audit logging (who did what, when) for remote actions
- VPN-first access model (WireGuard)

## Optional: Out-of-Band Mesh Comms (Meshtastic-like)

### What it is

An optional, **best-effort** comms capability that provides:

- A long-range on-site mesh node (and, where appropriate, a “repeater” / high-site node) to improve coverage.
- Integrated user messaging inside the stack UI/app on the LAN.
- Remote access to the same Meshtastic environment **only via secured VPN routes** (WireGuard).

This is treated like the Consumer Overlay: useful and integrator-friendly, but never allowed to become a safety dependency.

### Intended uses (non-critical)

- Convenience messaging for users on-site (and optionally when remote via VPN).
- Maintenance/support messaging and low-bandwidth status pings.
- Resilience for “we still want a message path” scenarios where phone signal is poor, without claiming emergency-grade availability.

### Not allowed / guardrails

- Not an emergency system and not a substitute for calling emergency services.
- Not life safety (fire alarms, emergency lighting, E-stops).
- Not primary intruder alarm signalling.
- Not a control channel for plant/security actions.
- Not exposed via any public/unauthenticated relay or “quick view” tier.

### Access model

- **Local (default):** usable from the site LAN via the stack UI.
- **Remote (optional):** usable only over WireGuard VPN with normal authentication.
- No payload access from public internet, and no payload access via read-only relay endpoints.

### Minimum “Comms Products” (MVP)

The minimum deliverable is something a client (and support) can actually use:

- **Mesh health badge** (`healthy` / `degraded` / `stale`) derived from last-heard time and error counters.
- **Gateway status** (bridge up/down, last successful sync, version info where available).
- **Link/transport stats** (delivery/ack rate where supported, drop/error counters).
- **Queue/backlog indicator** (queued count + oldest queued age).

Staleness rule:

- If mesh telemetry is **stale**, it must be treated as **unknown/unavailable** for any automation decisions (UI display may continue with a “stale” badge).

### Message handling defaults (privacy-first)

- Default UX is **session/short buffer**:
	- Messages are displayed live and may be held in a short-lived buffer for usability.
	- The stack does **not** build a message archive by default.
- **Opt-in**: on-site payload retention can be enabled per site with an explicit retention policy (typically measured in days, e.g. 7/30/90) documented in the handover pack.
- Off-site export of message payloads is **not** enabled by default.

### Placement (Node-RED vs openHAB)

- **Node-RED** owns the mesh gateway integration/bridge and normalises telemetry into “comms products” and freshness/staleness signals.
- **MQTT** can be used as the internal bus for those products.
- **openHAB/UI** presents status and user messaging surfaces, but does not treat mesh comms as authoritative for critical actions.

### UK compliance & ownership

This option must be configured to remain legally compliant in the UK:

- Use compliant radio hardware (e.g. UKCA/CE where required) suitable for the intended band.
- Configure frequency plan / region settings, transmit power, and antenna selection/placement to stay within licence-exempt SRD rules.

Ownership model:

- Gray Logic sets the initial configuration as part of commissioning.
- Ongoing changes/tweaks (including remote changes via VPN) are handled via an agreed support tier and must be logged.

