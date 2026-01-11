
# Module: Environment Monitoring

**Status**: Draft

## Purpose

Make climate and energy-adjacent signals visible and actionable, without creating dependencies that break the offline-first model.

## Core Inputs (On-Site)

- Room and plant temperatures
- Humidity and dew point (where relevant)
- CO₂ / IAQ (offices, cinemas, classrooms)
- Plant and leisure temperatures (pools, DHW cylinders)
- Energy meters and key circuit monitoring (where installed)

## Weather Inputs (Optional)

Weather is treated as an **advisory input** for comfort optimisation and user-facing visibility. It must never become required for safe operation.

Supported sourcing modes:

1. **Satellite ingest (preferred where feasible)**
	- Dedicated on-site receiver hardware (e.g. dish/antenna + receiver/decoder) ingests openly-available meteorological broadcasts.
	- Data use is subject to the provider’s licensing/terms and may be access-gated.

2. **Internet enrichment (never required)**
	- Optional forecast/nowcast APIs may be used to enrich or fill gaps.
	- Loss of internet must only pause enrichment.

3. **Hybrid**
	- Prefer local ingest; use internet enrichment when available.

### Minimum “Weather Products” (MVP)

The minimum deliverable is something a client can actually use locally:

- **Latest imagery tile** (e.g. cloud/IR layer), with a clear timestamp
- **2-hour loop** (animation of recent frames)
- **Freshness/age indicator** (e.g. “Updated 12 minutes ago”)
- **Feed health badge** (`healthy` / `degraded` / `stale`) based on last successful frame time and ingest error rate

Staleness rule:

- If weather data is **stale** (older than a configured threshold), it must be treated as **unknown** for automation decisions (UI display may continue with a “stale” badge).

### Optional “Immediate Forecast” Heuristics (Premium)

Short-horizon (0–2h) outputs are explicitly **advisory** and must be presented with uncertainty (ranges/confidence), not false precision.

Examples suitable for UK use:

- **Cloud-motion nowcast** (ETA for clouding-over/clearing at the site)
- **Solar attenuation proxy** (near-term “sun likely blocked” indicator for shading/comfort/PV storytelling)
- **Clearing vs building trend** classification to reduce noisy behaviour
- **Fog/low stratus hint** (flag as “possible”, not certain)

## Allowed Uses (Non-Critical)

- Client-facing dashboards (“what’s coming in next hour”)
- Comfort optimisation suggestions (e.g. bias pre-heat or shading decisions)
- Notifications (e.g. “clouding over soon”) where appropriate

## Not Allowed / Guardrails

- No safety-critical decisions based on weather inputs
- No life safety interactions
- Weather-driven automations must fail safe (if unknown/stale: do nothing special; fall back to deterministic local rules)

## Placement (Where the Logic Lives)

- **Node-RED**: weather ingest/normalisation, caching, freshness/staleness, derived “products”
- **openHAB**: presentation (Items/UI), simple rules that only consume weather products when fresh

