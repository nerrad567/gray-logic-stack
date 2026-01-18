---
title: System Capabilities & Benefits
version: 2.0.0
status: active
last_updated: 2026-01-18
depends_on:
  - vision.md
  - principles.md
  - ../domains/*.md
  - ../intelligence/phm.md
  - ../architecture/cloud-relay.md
---

# Gray Logic System: Capabilities & Benefits

> **The building intelligence platform that works when everything else fails.**

---

## 1. Executive Summary

Gray Logic is a **future-proof, offline-first building intelligence platform** designed as the "central nervous system" for properties. Unlike proprietary systems (Crestron, Savant) that lock users into specific dealers, or hobbyist platforms (Home Assistant) that lack stability, Gray Logic targets the underserved professional mid-market with an **electrician-first approach**.

### The Four Pillars

| Pillar | Promise | Proof |
|--------|---------|-------|
| **Offline-First** | 99%+ functionality without internet | Voice, automation, and all local control work identically with the cable cut |
| **No Lock-In** | Customer owns their system | Open standards (KNX, DALI, Modbus), open source core, full documentation handover |
| **Safety-First** | Physical controls always work | Wall switches work even if server, network, and internet all fail |
| **10-Year Horizon** | Install once, run for a decade | Version-pinned deployments, no forced updates, proven technologies |

### At a Glance

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                           GRAY LOGIC ARCHITECTURE                            │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │  INTELLIGENCE LAYER                                                  │   │
│   │  Voice • PHM • AI Insights • Presence Detection                     │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                     ▲                                        │
│   ┌─────────────────────────────────┴───────────────────────────────────┐   │
│   │  AUTOMATION LAYER                                                    │   │
│   │  Scenes • Schedules • Modes • Logic Engine                          │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                     ▲                                        │
│   ┌─────────────────────────────────┴───────────────────────────────────┐   │
│   │  DEVICE LAYER                                                        │   │
│   │  14 Domains • 6 Protocol Bridges • State Management                 │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                     ▲                                        │
│   ┌─────────────────────────────────┴───────────────────────────────────┐   │
│   │  HARDWARE BACKBONE (Always Works)                                    │   │
│   │  KNX • DALI • Modbus • Physical Switches                            │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Core Platform Capabilities

### Hardware Backbone ("The Spine")

| Technology | Purpose | Key Benefit |
|------------|---------|-------------|
| **KNX** | Switching, dimming, sensors, keypads | Works without software — wall switches directly control actuators |
| **DALI** | Advanced lighting control | Driver-level feedback, individual lamp addressing |
| **Modbus** | Plant equipment, energy monitoring | Industrial-grade reliability for HVAC, pumps, meters |
| **SIP** | Intercom, door stations | Local VoIP, no cloud dependency |
| **ONVIF/RTSP** | CCTV cameras | Open standard, vendor-agnostic |

**Resilience Guarantee**: If Gray Logic Core fails, lights still work. Heating still works. Blinds still work.

### Gray Logic Core ("The Brain")

| Component | Technology | Purpose |
|-----------|------------|---------|
| **Core Engine** | Go (single binary) | Automation, state management, API |
| **Database** | SQLite | Configuration, device state, audit logs |
| **Time-Series** | InfluxDB | Energy data, PHM telemetry, trends |
| **Message Bus** | MQTT | Bridge communication, events |
| **Local AI** | Whisper + Piper | Speech-to-text, text-to-speech |

### Feature Modules ("The Senses")

| Module | Capability | Offline? |
|--------|------------|----------|
| **Voice Bridge** | Local speech recognition, natural language | ✅ Yes |
| **PHM Engine** | Predictive health monitoring, anomaly detection | ✅ Yes |
| **Presence Engine** | Room occupancy, building modes (Home/Away/Night) | ✅ Yes |
| **Scheduler** | Cron-like and event-driven automation | ✅ Yes |
| **Scene Controller** | Multi-device coordinated actions | ✅ Yes |

---

## 3. The 14 Domains

Gray Logic provides comprehensive control across 14 integrated domains:

### Core Comfort

| Domain | Key Capabilities | Differentiators |
|--------|------------------|-----------------|
| **🌡️ Climate** | UFH, radiators, A/C, MVHR | Predictive pre-heating/cooling using weather forecasts; CO2-driven ventilation |
| **💡 Lighting** | Scenes, circadian rhythms, daylight harvesting | Lumen depreciation tracking; sub-500ms scene recall |
| **🪟 Blinds** | Sun tracking, privacy, wake simulation | Wind/rain safety override; slat angle optimisation |

### Security & Safety

| Domain | Key Capabilities | Differentiators |
|--------|------------------|-----------------|
| **🛡️ Security** | Panel integration (Texecom, Honeywell) | Voice PIN (sanitised from logs); hardware-independent |
| **📹 Video** | Local NVR (Frigate), ANPR, face detection | 100% local AI processing; no cloud dependency |
| **💧 Leak Protection** | Valve shutoff, floor sensors | Automatic shutoff in <5 seconds |
| **🚰 Water Management** | Tank levels, pumps, irrigation zones | Rainwater harvesting integration |

### Lifestyle & Efficiency

| Domain | Key Capabilities | Differentiators |
|--------|------------------|-----------------|
| **⚡ Energy** | Smart meter, solar, battery, EV, load shedding | Grid frequency monitoring; tariff-aware automation |
| **🔊 Audio** | Multi-room, streaming, TTS announcements | Local sources always work |
| **📺 Video** | Matrix switching, display control | HDMI/HDBaseT matrix integration |
| **🏊 Pool/Spa** | Temperature, chemistry, pumps, covers | Solar heating priority; safety interlocks |
| **🌿 Irrigation** | Zones, soil moisture, weather-aware | ET-based watering; rain delay |
| **🍔 Appliances** | Smart plugs, monitoring | Phantom load detection |
| **🌱 Plant** | HVAC, boilers, heat pumps | Modbus integration; efficiency monitoring |

---

## 4. Predictive Health Monitoring (PHM)

PHM transforms a reactive maintenance strategy into a proactive one — detecting equipment problems before they cause failures.

### Three Tiers of PHM

| Tier | Hardware Required | Capabilities | Target Market |
|------|-------------------|--------------|---------------|
| **Tier 1: Basic** | Smart devices with built-in telemetry | Runtime tracking, on/off patterns, error rates | All installations |
| **Tier 2: Enhanced** | CT clamps, external sensors | Power trending, efficiency curves, anomaly detection | High-end residential, light commercial |
| **Tier 3: Advanced** | Vibration, thermal, pressure sensors | Bearing wear prediction, time-to-failure estimates | Commercial plant, critical infrastructure |

### What PHM Detects

| Equipment | Monitored Parameters | Detectable Issues |
|-----------|---------------------|-------------------|
| **Pumps** | Current, vibration, runtime | Bearing wear, impeller damage, dry running |
| **HVAC** | Power, temperature differential | Refrigerant loss, filter blockage, efficiency drop |
| **Lighting** | Lumen output, power, errors | LED degradation, driver failure |
| **Motors** | Current, vibration, temperature | Winding faults, mechanical wear |

### PHM Value Proposition

| Without PHM | With PHM |
|-------------|----------|
| Equipment fails unexpectedly | 2-4 weeks warning before failure |
| Emergency callout fees | Planned maintenance at convenience |
| Cascade failures (pump → boiler → heating) | Isolated alerts prevent cascades |
| "It was working yesterday" | "PHM flagged this 3 weeks ago" |

---

## 5. Resilience & Offline Capabilities

### Degradation Hierarchy

When things fail, Gray Logic degrades gracefully in a predictable order:

```
ALWAYS WORKS (even if everything else fails):
├── Physical wall switches → actuators
├── Hardware frost protection (thermostat-based)
├── Life safety systems (fire, E-stop)
└── Alarm panel keypads

WORKS WITHOUT INTERNET:
├── All lighting, climate, blinds control
├── Voice commands (local processing)
├── Scenes and schedules
├── PHM monitoring
└── Mobile app (on LAN)

REQUIRES INTERNET (optional enhancements):
├── Remote access via cloud relay
├── Push notifications
├── External weather data
└── Cloud AI queries
```

### Failure Scenario Impact

| Failure | Impact | Recovery |
|---------|--------|----------|
| **Internet down** | Minimal — designed for this | Automatic on restore |
| **MQTT broker down** | Moderate — state becomes stale | Automatic reconnect |
| **Core down** | Automation stops, physical controls work | systemd auto-restart |
| **Database corrupted** | Config lost | USB backup restore |
| **Power outage** | Everything stops | Full state recovery from cache |

### Frost Protection Guarantee

**Frost protection is hardware-enforced and works regardless of software state:**

- ✅ Works during: Internet down, MQTT down, Core down, database corruption
- ✅ Enforced by: Thermostat hardware, NOT software
- ✅ Cannot be disabled: No UI or API can turn off frost protection

---

## 6. Cloud Services (Optional)

> **Philosophy**: Free core, premium cloud. The building works perfectly offline; cloud adds convenience.

### Subscription Tiers

| Tier | Monthly | Key Features |
|------|---------|--------------|
| **Free** | £0 | Full local functionality, VPN remote access |
| **Connect** | £9.99 | Cloud API relay, push notifications, config backup |
| **Secure** | £24.99 | + Remote CCTV (4 cameras), 7-day clip storage |
| **Premium** | £49.99 | + AI insights, 30-day storage, phone support |
| **Estate** | £99.99 | + Multi-site, 90-day storage, priority support, SLA |
| **Commercial** | Custom | + Unlimited sites, facility management features |

### Zero-Knowledge Architecture

| Principle | Implementation |
|-----------|---------------|
| **End-to-end encryption** | Control commands encrypted on-site, decrypted on-site |
| **Cloud sees only metadata** | Site ID, timestamp, message size — never content |
| **Video never stored unencrypted** | AES-256-GCM, keys held only by customer |
| **No behaviour profiling** | Cloud cannot analyse what you do in your home |

---

## 7. User Benefits

### For Homeowners

| Feature | Benefit | Impact |
|---------|---------|--------|
| **Energy Management** | Lower bills without effort | 20-40% reduction in heating/cooling costs |
| **Privacy First** | No creeping surveillance | 0 bytes of voice/video to cloud |
| **10-Year Horizon** | Investment protection | Won't be "obsolete" in 3 years |
| **One App** | Simplified control | Replaces 5-6 fragmented apps |
| **Healthy Home** | Better sleep and air | CO2 automation + circadian lighting |

### For Installers

| Feature | Benefit | Impact |
|---------|---------|--------|
| **Remote Diagnostics** | Reduced truck rolls | 80% of issues resolved remotely |
| **Standardised Config** | Predictable installation | 50% faster commissioning |
| **Upsell Path** | Modular capability add-ons | Voice, energy, PHM packages |
| **Documentation** | Transferable maintenance | Any qualified engineer can service |

### For Commercial/Facilities

| Feature | Benefit | Impact |
|---------|---------|--------|
| **PHM** | Prevent downtime | Catch failing pumps before total failure |
| **Compliance** | Auto-testing | Emergency lighting reports automated |
| **OpEx Reduction** | Automated efficiency | Lighting/HVAC off in empty rooms |
| **Multi-Site Dashboard** | Centralised visibility | "Which gym has a DALI fault?" |

---

## 8. Competitive Positioning

### Market Position

```
                    HIGH COMPLEXITY
                          │
         Crestron ●       │       ● Savant
                          │
                          │
    Control4 ●            │ ★ GRAY LOGIC
                          │   (Target Zone)
          Loxone ●        │
                          │
──────────────────────────┼──────────────────────────
    LOW PRICE             │            HIGH PRICE
                          │
                          │
Home Assistant ●          │
                          │
                    LOW COMPLEXITY
```

### Feature Comparison

| Feature | Gray Logic | Crestron | Control4 | Loxone | Home Assistant |
|---------|------------|----------|----------|--------|----------------|
| **Offline operation** | ✅ 99%+ | ❌ | ❌ | ✅ | ✅ |
| **Open source** | ✅ | ❌ | ❌ | ❌ | ✅ |
| **Local voice AI** | ✅ | ❌ | ❌ | ❌ | ⚠️ Complex |
| **PHM** | ✅ Tier 1-3 | ❌ | ❌ | ❌ | ❌ |
| **No dealer lock-in** | ✅ | ❌ | ❌ | ⚠️ | ✅ |
| **Professional support** | ✅ | ✅ | ✅ | ✅ | ❌ |
| **10-year stability** | ✅ Designed | ⚠️ | ⚠️ | ✅ | ❌ |
| **Typical project** | £8k-40k | £50k-200k | £15k-60k | £10k-30k | £2k-10k |

### Why Choose Gray Logic?

| If You Want... | Not This | Choose Gray Logic Because |
|----------------|----------|---------------------------|
| Professional quality | Home Assistant | Stability, documentation, support |
| No vendor lock-in | Crestron, Savant | Open source, open standards |
| Affordable mid-market | Control4 | Lower cost, no dealer markup |
| True offline | Most systems | Internet ≠ requirement |
| Privacy | Alexa-dependent | Local voice, no cloud surveillance |
| 10-year lifespan | Consumer IoT | Designed for longevity |

---

## 9. Integration & Extensibility

### Bridge Architecture

Each protocol has an independent bridge. If one crashes, others continue:

| Bridge | Protocol | Devices |
|--------|----------|---------|
| **KNX Bridge** | KNX/IP | Switches, actuators, sensors, keypads |
| **DALI Bridge** | DALI-2 | LED drivers, emergency lighting |
| **Modbus Bridge** | Modbus TCP/RTU | Energy meters, HVAC, heat pumps |
| **Audio Bridge** | Proprietary | Audio matrix (Sonance, RTI) |
| **Security Bridge** | Texecom, Honeywell | Alarm panels |
| **CCTV Bridge** | Frigate/MQTT | NVR, cameras |

### API Access

| Interface | Purpose | Authentication |
|-----------|---------|----------------|
| **REST API** | Integrations, dashboards | JWT tokens |
| **WebSocket** | Real-time state, notifications | Ticket-based |
| **MQTT** | Internal communication | mTLS |

### Upgrade Path

| Package | Installation Type | Additions |
|---------|-------------------|-----------|
| **Starter** | Single room | Lighting + heating |
| **Standard** | Full home | + Blinds, energy, security |
| **Estate** | Multi-building | + Plant, pool, central monitoring |
| **Commercial** | Light commercial | + RBAC, multi-tenant, SLA |

### DIY Device Integration

Gray Logic welcomes customer-owned smart devices — provided they respect our offline-first principle.

| Integration Tier | Protocol | Examples | Support Level |
|------------------|----------|----------|---------------|
| **Tier 1: Matter** | Matter/Thread | Eve, Nanoleaf, newer Hue | ✅ Native |
| **Tier 2: Zigbee** | Zigbee 3.0 | Aqara, IKEA, Sonoff | ✅ Full |
| **Tier 3: Z-Wave** | Z-Wave 700/800 | Zooz, Aeotec, Yale | ✅ Full |
| **Tier 4: Wi-Fi** | Local API | Shelly, Tasmota, ESPHome | ✅ Standard |
| **Tier 5: Cloud** | Vendor API | Ring, Nest | ⚠️ Tolerated |

**Key Requirements for DIY Devices:**
- Must work **100% locally** (no cloud dependency)
- Documented API (no reverse-engineering)
- Standard protocols only

**Ecosystem Bridges:**

| Bridge | Direction | Notes |
|--------|-----------|-------|
| **HomeKit** | Gray Logic → Apple Home | 100% local, Siri control |
| **Google Home** | Gray Logic → Google | Cloud required, opt-in |
| **Alexa** | Gray Logic → Amazon | Cloud required, opt-in |

> See [DIY Integration Specification](../integration/diy-integration.md) for full details on adding smart plugs, sensors, lights, and more.

---

## 10. Conclusion

Gray Logic reimagines the "smart home" not as a collection of gadgets, but as **infrastructure**. Like electrical wiring or plumbing, it's designed to be installed once and run reliably for decades.

### The Gray Logic Promise

1. **Works without internet** — The building keeps running
2. **No vendor lock-in** — You own your system
3. **10-year stability** — Install once, run for a decade
4. **Privacy by design** — Local processing, no surveillance
5. **Professional quality** — Not a hobby project

> **"This is the central nervous system for the next generation of buildings — built to last."**

---

## Related Documents

- [Vision](vision.md) — What Gray Logic is and why
- [Principles](principles.md) — Hard rules that cannot be broken
- [System Architecture](../architecture/system-overview.md) — Technical design
- [Cloud Relay](../architecture/cloud-relay.md) — Optional cloud services
- [DIY Integration](../integration/diy-integration.md) — Customer-owned devices
- [PHM Specification](../intelligence/phm.md) — Predictive health monitoring
- [Subscription Pricing](../business/subscription-pricing.md) — Cloud tier details
