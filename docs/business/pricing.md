---
title: Gray Logic Pricing Model
version: 1.0.0
status: active
last_updated: 2026-01-14
depends_on:
  - business-case.md
  - ../overview/vision.md
---

# Gray Logic Pricing Model

## Pricing Philosophy

Gray Logic pricing is designed to be:

1. **Accessible** — Professional automation shouldn't require being a millionaire
2. **Transparent** — Customers see exactly where their money goes
3. **Profitable** — Healthy margins for installers (us and future partners)
4. **Competitive** — Better value than Loxone/Control4, vastly better than Crestron/Savant

**Target price range:** £8,000 - £40,000 (comparable to an expensive home rewire)

**Benchmark:** A full electrical rewire of a 4-bed house costs £8,000 - £15,000. Gray Logic should sit at the upper end of this range for basic installations, scaling with complexity.

---

## Installation Tiers

### Overview

| Tier | Target Property | Domains Included | Price Range |
|------|-----------------|------------------|-------------|
| Essential | 2-3 bed home | Lighting, basic scenes | £8,000 - £15,000 |
| Standard | 3-4 bed home | + Climate, blinds | £15,000 - £25,000 |
| Premium | 4-5 bed / new build | + Audio, security | £25,000 - £40,000 |
| Estate | Multiple buildings | Full integration | £40,000+ (quoted) |

---

### Essential Tier (£8,000 - £15,000)

**Target:** 2-3 bedroom home, first-time automation buyer

**Included Domains:**
- Lighting control (all rooms)
- Scene management (whole house)
- Basic scheduling
- Mobile app control
- Local operation (no cloud dependency)

**Typical Hardware:**
- Gray Logic Hub
- KNX actuators (8-12 channels)
- KNX push buttons (6-10 rooms)
- Basic presence detection (2-3 sensors)

**What's NOT included:**
- Climate control (manual thermostats remain)
- Blind control
- Multi-room audio
- Security integration
- Advanced presence/occupancy

**Sample Breakdown (3-bed semi):**

| Item | Cost | Notes |
|------|------|-------|
| Gray Logic Hub | £800 | Core controller |
| KNX power supply | £150 | 640mA |
| KNX actuators (16ch) | £600 | 2 × 8-channel |
| KNX push buttons (8) | £1,200 | £150 avg each |
| Presence sensors (3) | £300 | PIR/occupancy |
| Cabling and sundries | £400 | Cat6, KNX bus |
| **Hardware subtotal** | **£3,450** | |
| Installation labour | £2,500 | 4-5 days |
| Configuration/commissioning | £1,200 | 2 days |
| Documentation/handover | £400 | Half day |
| **Labour subtotal** | **£4,100** | |
| **Total** | **£7,550** | |
| Contingency (10%) | £755 | |
| **Customer price** | **£8,300** | |

**Margin analysis:**
- Hardware cost: £3,450
- Hardware markup (15%): £520
- Labour cost (days × day rate): Variable
- Target gross margin: 35-40%

---

### Standard Tier (£15,000 - £25,000)

**Target:** 3-4 bedroom home, comfort-focused buyer

**Included Domains:**
- Everything in Essential, plus:
- Climate control (heating zones, optional cooling)
- Blind/shading control
- Enhanced presence detection
- Energy monitoring (main circuits)
- Weather-responsive automation

**Typical Hardware:**
- Gray Logic Hub
- KNX actuators (16-24 channels)
- KNX push buttons (8-12 rooms)
- KNX blind actuators (4-8 channels)
- KNX climate integration
- Enhanced presence sensors

**Sample Breakdown (4-bed detached):**

| Item | Cost | Notes |
|------|------|-------|
| Gray Logic Hub | £800 | Core controller |
| KNX power supply | £200 | 960mA |
| KNX lighting actuators (24ch) | £900 | 3 × 8-channel |
| KNX blind actuators (8ch) | £450 | 2 × 4-channel |
| KNX push buttons (12) | £1,800 | £150 avg each |
| KNX thermostats (4) | £800 | Zone control |
| Presence sensors (6) | £600 | PIR/occupancy |
| Energy monitoring | £400 | CT clamps + interface |
| Weather station | £300 | Wind/rain sensors |
| Cabling and sundries | £600 | Cat6, KNX bus |
| **Hardware subtotal** | **£6,850** | |
| Installation labour | £4,500 | 7-8 days |
| Configuration/commissioning | £2,000 | 3 days |
| Documentation/handover | £600 | 1 day |
| **Labour subtotal** | **£7,100** | |
| **Total** | **£13,950** | |
| Contingency (10%) | £1,395 | |
| **Customer price** | **£15,350** | |

---

### Premium Tier (£25,000 - £40,000)

**Target:** 4-5 bedroom home, new build, full integration buyer

**Included Domains:**
- Everything in Standard, plus:
- Multi-room audio (4-6 zones)
- Security system integration
- CCTV integration (viewing, not supply)
- Intercom/access control integration
- Video distribution (basic)
- Voice control (local)
- Predictive health monitoring (basic)

**Typical Hardware:**
- Gray Logic Hub (enhanced spec)
- Full KNX installation
- Audio endpoints (Sonos/similar)
- Voice processing hardware
- Network infrastructure upgrades
- Touch panels (optional)

**Sample Breakdown (5-bed new build):**

| Item | Cost | Notes |
|------|------|-------|
| Gray Logic Hub (Pro) | £1,200 | Enhanced CPU/RAM |
| KNX power supply | £300 | Redundant |
| KNX lighting actuators (32ch) | £1,200 | 4 × 8-channel |
| KNX dimming actuators (8ch) | £800 | Feature lighting |
| KNX blind actuators (12ch) | £675 | 3 × 4-channel |
| KNX push buttons (16) | £2,400 | £150 avg each |
| KNX thermostats (6) | £1,200 | Zone control |
| Presence sensors (10) | £1,000 | PIR/occupancy |
| Energy monitoring (detailed) | £800 | Per-circuit |
| Weather station | £300 | Wind/rain sensors |
| Audio endpoints (6 zones) | £1,800 | Sonos or similar |
| Voice processing | £400 | Local STT/TTS |
| Network switch upgrade | £500 | Managed, PoE |
| Touch panel (1) | £800 | Main entrance |
| Cabling and sundries | £1,000 | Cat6, KNX bus |
| **Hardware subtotal** | **£14,375** | |
| Installation labour | £8,000 | 12-14 days |
| Configuration/commissioning | £4,000 | 5-6 days |
| Documentation/handover | £1,000 | 1.5 days |
| **Labour subtotal** | **£13,000** | |
| **Total** | **£27,375** | |
| Contingency (10%) | £2,738 | |
| **Customer price** | **£30,115** | |

---

### Estate Tier (£40,000+)

**Target:** Multiple buildings, large properties, complex requirements

**Scope:** Fully bespoke, quoted per project

**Typical additions:**
- Multiple Gray Logic Hubs (per building)
- Inter-building communication (fiber/wireless)
- Pool/spa automation
- Irrigation systems
- Outbuilding control
- Gate and access control
- Full CCTV integration
- Commercial-grade equipment

**Pricing approach:**
- Detailed site survey required
- Full specification developed
- Itemised quote with all components
- No "package" pricing — everything transparent

---

## Hardware Pricing

### Gray Logic Hub

The Gray Logic Hub is custom hardware designed for reliability and purpose.

| Model | Specification | Trade Cost | Retail Price | Notes |
|-------|---------------|------------|--------------|-------|
| Hub Standard | ARM64, 4GB, 64GB eMMC | £450 | £800 | Most installations |
| Hub Pro | ARM64, 8GB, 128GB NVMe | £700 | £1,200 | Large/complex sites |
| Hub Enterprise | x86, 16GB, 256GB NVMe | £1,200 | £2,000 | Multi-building, high I/O |

**Margin:** 75-80% on hardware (covers R&D, support, warranty)

### Recommended Hardware Markup

For third-party hardware (KNX, audio, etc.):

| Category | Typical Markup | Rationale |
|----------|----------------|-----------|
| KNX actuators | 15-20% | Commodity, price-sensitive |
| KNX push buttons | 20-25% | Design element, less comparison |
| Sensors | 15-20% | Commodity |
| Audio equipment | 10-15% | Easily price-checked |
| Network equipment | 10-15% | Commodity |
| Custom/specialist | 25-30% | Harder to source, expertise required |

**Philosophy:** Lower markup on easily-compared items, higher on specialist equipment where we add value through selection and integration expertise.

---

## Labour Rates

### Day Rates

| Activity | Day Rate | Notes |
|----------|----------|-------|
| Site survey | £400 | May be credited against project |
| Installation (electrical) | £450 | Physical installation |
| Configuration | £500 | Software setup, testing |
| Commissioning | £500 | Final testing, optimisation |
| Training/handover | £400 | Customer education |
| Support callout | £500 | Emergency/priority |

### Hourly Rates (for small works)

| Activity | Hourly Rate | Minimum |
|----------|-------------|---------|
| Configuration changes | £75 | 1 hour |
| Remote support | £60 | 30 mins |
| On-site support | £85 | 2 hours |
| Emergency callout | £120 | 2 hours + travel |

---

## Support Tiers

### Overview

| Tier | Annual Fee | Response Time | Includes |
|------|------------|---------------|----------|
| Community | Free | Best effort | Forum, documentation |
| Standard | £300/year | 48 hours | Email, remote diagnostics |
| Premium | £600/year | 24 hours | Phone, annual health check |
| Enterprise | £1,200/year | 4 hours | Priority, quarterly reviews |

---

### Community Tier (Free)

**Included:**
- Access to community forum
- Full documentation
- Software updates
- Security patches

**Not included:**
- Direct support
- Remote diagnostics
- On-site visits
- Phone support

**Target:** DIY users of open source software, cost-conscious customers

---

### Standard Tier (£300/year)

**Included:**
- Everything in Community, plus:
- Email support (48-hour response)
- Remote diagnostic access (with permission)
- Priority security updates
- 10% discount on callout rates

**Not included:**
- Phone support
- On-site visits (charged separately)
- Hardware replacement

**Target:** Most residential customers

**Economics:**
- Expected support time: 2-3 hours/year
- Gross margin: ~70%

---

### Premium Tier (£600/year)

**Included:**
- Everything in Standard, plus:
- Phone support (24-hour response)
- Annual health check visit
- 20% discount on callout rates
- Priority scheduling for changes
- Proactive monitoring alerts

**Not included:**
- Hardware replacement (warranty separate)
- Major reconfigurations (quoted separately)

**Target:** Larger installations, customers who value peace of mind

**Economics:**
- Expected support time: 4-6 hours/year
- Annual visit: 2-3 hours
- Gross margin: ~50%

---

### Enterprise Tier (£1,200/year)

**Included:**
- Everything in Premium, plus:
- 4-hour response time
- Quarterly system reviews
- 30% discount on callout rates
- Dedicated contact
- SLA guarantees

**Target:** Estate installations, small commercial

**Economics:**
- Expected support time: 8-12 hours/year
- Quarterly visits: 8-10 hours
- Gross margin: ~40%

---

## Future Revenue Streams

### Training Courses

**2-Day Installer Certification Course**

| Item | Price | Notes |
|------|-------|-------|
| Course fee | £1,500 | Per attendee |
| Maximum attendees | 6 | Quality over quantity |
| Venue | Included | Or customer site (+travel) |
| Materials | Included | Printed + digital |

**Curriculum:**
- Day 1: Gray Logic architecture, KNX basics, installation
- Day 2: Configuration, commissioning, troubleshooting, handover

**Economics:**
- Course revenue: £9,000 (6 attendees)
- Venue/materials cost: £500
- Instructor time: 3 days (prep + delivery)
- Gross margin: ~60%

**Frequency:** 4-6 courses per year (when demand exists)

---

### Certification Programme

**Annual Certification Fee**

| Level | Annual Fee | Requirements |
|-------|------------|--------------|
| Certified Installer | £500/year | Training + 1 installation |
| Certified Partner | £1,000/year | 3+ installations, quality audit |
| Master Partner | £2,000/year | 10+ installations, training delivery rights |

**Benefits by Level:**

| Benefit | Installer | Partner | Master |
|---------|-----------|---------|--------|
| Hardware discount | 10% | 15% | 20% |
| Lead referrals | Local | Regional | National |
| Logo usage | Badge | Full | Full + "Master" |
| Directory listing | Basic | Featured | Premium |
| Technical support | Standard | Priority | Dedicated |

---

### Hardware Wholesale

**Certified Installer Pricing:**

| Product | Retail | Certified (10%) | Partner (15%) | Master (20%) |
|---------|--------|-----------------|---------------|--------------|
| Hub Standard | £800 | £720 | £680 | £640 |
| Hub Pro | £1,200 | £1,080 | £1,020 | £960 |
| Hub Enterprise | £2,000 | £1,800 | £1,700 | £1,600 |

**Minimum order:** None (order as needed)
**Payment terms:** 30 days for Partners, immediate for Installers

---

### Consultancy

**Day Rate:** £600-800 (depending on complexity)

**Typical Engagements:**
- System design for complex projects
- Integration specification
- Third-party installer support
- Troubleshooting difficult issues
- Architecture review

---

## Margin Guidelines

### Target Margins

| Revenue Stream | Target Gross Margin |
|----------------|---------------------|
| Installation projects | 35-40% |
| Hardware sales (own) | 75-80% |
| Hardware sales (third-party) | 15-25% |
| Support contracts | 50-70% |
| Training courses | 55-65% |
| Consultancy | 70-80% |

### For Certified Installers

When training future installers, provide guidance on sustainable margins:

| Category | Recommended Markup |
|----------|-------------------|
| Gray Logic Hub | 20-30% on trade price |
| KNX equipment | 15-25% |
| Labour | £400-600/day depending on market |
| Support contracts | Match our pricing or adjust for local market |

**Key message:** "Don't race to the bottom. Quality work commands fair prices."

---

## Pricing Presentation

### Quotation Format

All quotes should include:

1. **Executive Summary**
   - Total price
   - What's included (plain English)
   - Timeline

2. **Detailed Breakdown**
   - Hardware (itemised with costs)
   - Labour (days × rate)
   - Contingency
   - Support options

3. **What's NOT Included**
   - Clear exclusions
   - Optional upgrades

4. **Payment Terms**
   - Deposit (typically 30%)
   - Stage payments
   - Final payment

### Transparency Principle

Show customers:
- What hardware costs
- What labour costs
- What markup is applied

**Why:** Builds trust, differentiates from opaque competitors, justifies pricing.

---

## Price Adjustments

### Annual Review

Prices reviewed annually (January) considering:
- Hardware cost changes
- Labour market rates
- Competitor pricing
- Exchange rates (EU expansion)

### Regional Variation

For future EU expansion:
- Convert GBP to EUR at prevailing rate
- Adjust for local labour costs
- Consider VAT differences

### Inflation Clause

Long projects (6+ months) may include inflation clause:
- Hardware priced at order date
- Labour rates may adjust if project extends beyond 12 months

---

## Related Documents

- [Institutional Principles](institutional-principles.md) — Building for generations
- [Business Case](business-case.md) — Strategic positioning
- [Sales Specification](sales-spec.md) — How to quote and sell
- [Go-to-Market](go-to-market.md) — Growth strategy
- [Residential Deployment](../deployment/residential.md) — What's involved in installation
