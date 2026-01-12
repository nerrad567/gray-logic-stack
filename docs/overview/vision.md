---
title: Gray Logic Vision
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on: []
---

# Gray Logic Vision

## What Gray Logic Is

**Gray Logic** is a complete building intelligence platform — the central nervous system of a property. It rivals and aims to surpass systems like Crestron, Savant, Lutron, Control4, and Loxone while maintaining complete openness, true offline capability, and zero vendor lock-in.

Gray Logic is:

- **A product**, not a hobby project or integration exercise
- **A business**, designed to be installed, supported, and eventually sold
- **A 10+ year commitment** — systems deployed today must work reliably for a decade with minimal intervention
- **Built by an electrician** who understands real-world installation, maintenance, and what happens when things fail

## What Gray Logic Is Not

Gray Logic is **not**:

- A smart home toy or gadget platform
- A cloud-dependent service
- A wrapper around Home Assistant, openHAB, or other existing platforms
- A "good enough" solution — we aim to match or exceed commercial offerings
- A system that requires constant updates or internet connectivity

## The Problem We Solve

### For Property Owners

High-end homes, estates, pools/spas, and light commercial buildings need integrated control of:

- Lighting and scenes
- Climate and ventilation
- Blinds and shading
- Multi-room audio and video
- Security, CCTV, and access control
- Plant and equipment (pools, boilers, heat pumps)
- Energy monitoring

Current options are unsatisfactory:

| Option | Problem |
|--------|---------|
| **Crestron/Savant/Control4** | Vendor lock-in, proprietary tools, dealer-dependent, expensive ongoing costs |
| **Loxone** | Closed ecosystem, limited third-party integration |
| **DIY (Home Assistant, etc.)** | Unstable, poorly documented, no professional support, constant updates break things |
| **Traditional electrical** | No integration, no single view, no remote capability |

### For Gray Logic (The Business)

- Standard electrical work has limited margin and scale
- Existing skills (Linux, Docker, networking, documentation) are underutilised
- No existing product matches the "professional, open, offline-first" niche
- Recurring revenue opportunity through support tiers

## Core Value Propositions

### 1. True Offline Operation

**99%+ of functionality works without internet.**

- All lighting, scenes, schedules, modes
- All climate control
- All security monitoring
- All voice commands (local AI)
- All PHM monitoring
- All local UIs

Internet provides *optional enhancements*:
- Remote access
- Cloud AI for complex queries
- External weather data
- Remote notifications

**When the internet dies, the building doesn't.**

### 2. Zero Vendor Lock-in

- **Field layer**: Open standards only (KNX, DALI, Modbus)
- **Software**: We own and control Gray Logic Core
- **Documentation**: Complete handover pack for every installation
- **Exit strategy**: Any competent integrator can take over

**No golden handcuffs.** The customer owns their system.

### 3. 10-Year Stability

- Version-pinned deployments
- Security patches only (no feature upgrades forced)
- Hardware designed for 24/7 operation
- No dependency on external services that might disappear

**Install once, run for a decade.**

### 4. Native AI Integration

- Local voice control (no Alexa/Google dependency)
- Natural language commands ("turn on the lights", "it's cold in here")
- AI-assisted Predictive Health Monitoring
- Privacy-first (audio processed locally, never stored)

**Intelligence without surveillance.**

### 5. Predictive Health Monitoring (PHM)

- Learn normal equipment behaviour
- Detect deviations before failures
- Early warning, not magic promises
- Applies to any monitored asset

**Know before things fail.**

### 6. Resilience Beyond Internet

For critical installations:
- Direct satellite weather data (NOAA/EUMETSAT decode)
- LoRa mesh communication between buildings
- Amateur radio integration
- True off-grid capability for essentials

**If the world goes to shit, your home still works.**

## Target Markets

### Primary

1. **High-end residential** (new builds, deep refurbishments)
   - £500k+ property value
   - Clients who value quality and longevity
   - Architects and builders who want open systems

2. **Pool/spa/leisure facilities**
   - Real plant requiring monitoring
   - Humidity and ventilation control
   - Commercial-grade reliability needed

3. **Light commercial**
   - Small offices, showrooms, mixed-use
   - Need visibility without full BMS cost
   - No dedicated facilities team

### Secondary

4. **Small estates** (multiple buildings)
   - Outbuildings, guest houses, pool houses
   - Unified control and monitoring
   - Remote management essential

5. **Boutique hospitality**
   - Small hotels, B&Bs, holiday lets
   - Guest room control
   - Energy monitoring critical

## What Success Looks Like

### Technical Success

- [ ] Complete control of all domains (lighting, climate, audio, video, security, etc.)
- [ ] Local voice control working reliably
- [ ] PHM detecting real issues before failure
- [ ] 10+ year runtime on deployed systems
- [ ] Any competent integrator can maintain installed systems

### Business Success

- [ ] Multiple deployed customer sites
- [ ] Recurring revenue from support tiers
- [ ] Effective day rate exceeds electrical work
- [ ] Positive customer referrals
- [ ] Viable business to sell on retirement

### Personal Success

- [ ] Proud of the product
- [ ] Work-life balance maintained
- [ ] Skills developed and portfolio built
- [ ] Clear path to exit if desired

## Timeline

This is a **5-year part-time project** with no fixed deadline.

| Phase | Focus | Milestone |
|-------|-------|-----------|
| Year 1 | Foundation | Core + KNX + lighting working in own home |
| Year 2 | Automation | Full scenes, schedules, modes, blinds, climate |
| Year 3 | Integration | Audio, video, security, CCTV, access control |
| Year 4 | Intelligence | Voice control, PHM, AI features |
| Year 5 | Product | Commissioning tools, first customer sites |

## Guiding Question

When making any decision, ask:

> "Will this still work and be supportable in 10 years, 
> without internet, without me, and without vendor cooperation?"

If the answer is no, find another way.

---

## Related Documents

- [Principles](principles.md) — Hard rules that cannot be broken
- [System Overview](../architecture/system-overview.md) — Technical architecture
- [Glossary](glossary.md) — Terms and definitions
