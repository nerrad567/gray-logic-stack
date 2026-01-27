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

### 3. Multi-Decade Stability

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

Gray Logic is designed to scale from a single home to large commercial buildings. These are not priorities — they are **tiers of deployment complexity**.

### Tier 1: Residential

| Segment | Property Value | Typical System Cost | Key Needs |
|---------|----------------|---------------------|-----------|
| **High-end new builds** | £750k+ | £15,000 - £40,000 | Reliability, design integration, handover |
| **Deep refurbishments** | £500k+ | £10,000 - £30,000 | Retrofit-friendly, minimal disruption |
| **Self-builds** | £400k+ | £8,000 - £25,000 | Technical involvement, flexibility |
| **Small estates** | Multiple buildings | £25,000 - £80,000 | Unified control, outbuildings, remote management |

### Tier 2: Light Commercial

| Segment | Size | Typical System Cost | Key Needs |
|---------|------|---------------------|-----------|
| **Small offices** | <500m² | £10,000 - £30,000 | Energy monitoring, scheduling, presence |
| **Retail/showrooms** | <1,000m² | £15,000 - £50,000 | Lighting scenes, presence detection |
| **Boutique hospitality** | <20 rooms | £20,000 - £80,000 | Guest control, energy, HVAC |
| **Pool/spa facilities** | Variable | £15,000 - £60,000 | Plant monitoring, humidity, safety |

### Tier 3: Commercial

| Segment | Size | Typical System Cost | Key Needs |
|---------|------|---------------------|-----------|
| **Medium offices** | 500-5,000m² | £50,000 - £200,000 | BACnet, energy management, central monitoring |
| **Multi-site operations** | Chain/franchise | Per-site + central | Standardisation, central dashboard |
| **Industrial** | Variable | £100,000+ | PHM, predictive maintenance, safety systems |

**Technology requirements by tier:**
- **Tier 1:** KNX + Matter (Year 1-2)
- **Tier 2:** + DALI diagnostics + Modbus (Year 2)
- **Tier 3:** + BACnet + Multi-site dashboard (Year 3)

## Competitive Position

Gray Logic occupies whitespace between three markets:

| Market | Examples | Gray Logic Advantage |
|--------|----------|---------------------|
| **Premium proprietary** | Crestron, Savant, Control4 | Open source, no dealer lock-in, affordable |
| **Consumer KNX bridges** | 1Home, Gira X1, Thinka | Professional tools, PHM, multi-site management |
| **DIY platforms** | Home Assistant, openHAB | Reliability, professional support, stability |

### Key Differentiators

1. **Open source + Professional reliability** — Unlike Home Assistant (unstable) or Crestron (closed)
2. **Predictive Health Monitoring** — Detect failures before they happen (rare in market)
3. **Installer-first tools** — Multi-site dashboard, commissioning, handover docs
4. **Matter compatible** — Consumer voice control via Apple/Google/Alexa (Year 2)
5. **Commercial scale** — BACnet bridge for building systems (Year 3)
6. **No artificial limits** — Unlike Gira X1's 1,000 data point cap

### Technology Integration

| Technology | Purpose | Timeline |
|------------|---------|----------|
| **KNX** | Physical control layer (switches, relays, sensors) | Year 1 ✅ |
| **DALI** | Lighting intelligence — direct control via Lunatone IoT Gateway, full diagnostics for PHM | Year 2 |
| **Matter** | Consumer ecosystem bridge (Apple/Google/Alexa) via Matterbridge | Year 2 |
| **Modbus** | HVAC and plant equipment | Year 2 |
| **BACnet** | Commercial building systems | Year 3 |

**DALI Architecture:** KNX provides physical reliability (wall switches → relay → 230V → driver). DALI bus provides intelligence layer (dimming, CCT, diagnostics). If Gray Logic fails, lights still work at 100% via KNX. See `docs/architecture/decisions/0001-protocol-topology.md`.

See `docs/business/market-research.md` for detailed competitive analysis.

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

> "Will this still work and be supportable in 20 years, 
> without internet, without me, and without vendor cooperation?"

If the answer is no, find another way.

---

## Related Documents

- [Principles](principles.md) — Hard rules that cannot be broken
- [System Overview](../architecture/system-overview.md) — Technical architecture
- [Glossary](glossary.md) — Terms and definitions
