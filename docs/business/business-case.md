---
title: Gray Logic Business Case
version: 1.0.0
status: active
last_updated: 2026-01-14
depends_on:
  - ../overview/vision.md
  - ../overview/principles.md
---

# Gray Logic Business Case

## Executive Summary

Gray Logic is a professional building automation platform targeting the underserved middle market between expensive proprietary systems (Crestron, Savant) and unreliable DIY solutions (Home Assistant). By combining **open source software** with **purpose-built hardware** and **electrician-led installation**, Gray Logic offers professional-grade home automation accessible to quality-conscious homeowners — not just multi-millionaires.

**Key differentiators:**
- Open source software (transparency, no vendor lock-in)
- Custom hardware designed for reliability
- Built by an electrician who understands real-world installation
- Multi-decade deployment commitment
- True offline operation

**Target price point:** £8,000 - £40,000 (comparable to an expensive home rewire)

---

## Market Opportunity

### UK Smart Home Market

The UK smart home market continues to grow as homeowners increasingly expect integrated control of lighting, heating, security, and entertainment. However, the market is polarised:

| Segment | Market Size | Growth | Pain Points |
|---------|-------------|--------|-------------|
| Premium (Crestron, Savant) | ~£200M | Stable | Dealer lock-in, high cost, proprietary |
| Mid-market (Control4, Loxone) | ~£150M | Growing | Semi-proprietary, limited flexibility |
| DIY (Home Assistant, etc.) | ~£100M | Fast | Unreliable, no support, constant maintenance |
| Traditional electrical | ~£2B | Slow | No integration, no intelligence |

### The Underserved Middle

There exists a significant gap between:
- **Homeowners who can afford £50k+** for Crestron/Savant (and accept dealer dependency)
- **Technical enthusiasts** willing to maintain Home Assistant (and accept instability)

This middle market comprises homeowners who:
- Value quality and reliability
- Want professional installation and support
- Cannot justify £50k+ for home automation
- Don't have time or skills for DIY solutions
- Care about longevity and exit strategy

**This is Gray Logic's target market.**

### Market Size Estimate (UK)

| Segment | Annual New Builds | Refurbishments | Total Addressable |
|---------|-------------------|----------------|-------------------|
| High-end residential (£500k+) | ~15,000 | ~25,000 | ~40,000 homes/year |
| Realistic capture rate (Year 5) | 0.01% | 0.01% | ~4 projects/year |
| Average project value | - | - | £20,000 |
| **Annual revenue potential** | - | - | **£80,000** |

Note: These are conservative estimates for a boutique installer operation. The market opportunity is substantially larger for a scaled operation.

---

## Competitor Analysis

### Tier 1: Ultra-Premium (£50,000 - £200,000+)

**Crestron / Savant / Lutron RadioRA 3**

| Aspect | Assessment |
|--------|------------|
| Target market | Luxury homes, yachts, commercial |
| Typical project | £50,000 - £200,000+ |
| Strengths | Proven reliability, dealer network, prestige |
| Weaknesses | Extreme cost, total dealer dependency, proprietary |
| Lock-in level | **Absolute** — cannot self-maintain or switch |

*Pricing includes dealer markup, proprietary hardware, and ongoing "service agreements" that are effectively mandatory.*

### Tier 2: Premium (£15,000 - £80,000)

**Control4**

| Aspect | Assessment |
|--------|------------|
| Target market | Affluent homeowners |
| Typical project | £15,000 - £80,000 |
| Strengths | Good dealer network, reasonable UI, proven |
| Weaknesses | Dealer-dependent, semi-proprietary, cloud features |
| Lock-in level | **High** — dealer required for most changes |

**Loxone**

| Aspect | Assessment |
|--------|------------|
| Target market | New builds, energy-conscious |
| Typical project | £10,000 - £50,000 |
| Strengths | Good value, energy focus, direct sales option |
| Weaknesses | Closed ecosystem, limited third-party integration |
| Lock-in level | **Medium** — Loxone hardware only, but self-programmable |

### Tier 3: DIY / Open Source (£2,000 - £10,000)

**Home Assistant / openHAB / Hubitat**

| Aspect | Assessment |
|--------|------------|
| Target market | Technical enthusiasts |
| Typical project | £2,000 - £10,000 (hardware only) |
| Strengths | Flexible, cheap, large community |
| Weaknesses | Unstable, poor documentation, updates break things |
| Lock-in level | **None** — but also no support |

*The "free" software often costs thousands in time, frustration, and unreliability.*

### Gray Logic Positioning

| Aspect | Gray Logic |
|--------|------------|
| Target market | Quality-conscious homeowners, £500k+ properties |
| Typical project | **£8,000 - £40,000** |
| Strengths | Professional quality, open source, offline-first, electrician-built |
| Weaknesses | New entrant, limited brand recognition |
| Lock-in level | **Zero** — open source, full documentation, any electrician can maintain |

```
Price/Complexity Matrix

                    HIGH COMPLEXITY
                          │
         Crestron ●       │       ● Savant
                          │
                          │
    Control4 ●            │
                          │
              ┌───────────┼───────────┐
              │   GRAY    │  LOGIC    │
              │   TARGET  │  ZONE     │
    Loxone ●  │           │           │
              └───────────┼───────────┘
                          │
                          │
    Home Assistant ●      │
                          │
         LOW PRICE ───────┼─────── HIGH PRICE
                          │
                    LOW COMPLEXITY
```

---

## Open Source Strategy

### Why Open Source?

Gray Logic software is released as open source. This is a deliberate strategic decision, not an afterthought.

#### 1. Trust Through Transparency

Proprietary systems are black boxes. Customers must trust:
- That the software does what it claims
- That there are no backdoors or data collection
- That the company will exist in 20 years
- That they won't be held hostage for support

Open source eliminates these concerns. Customers (or their representatives) can verify everything.

#### 2. Longevity Guarantee

The multi-decade deployment commitment is credible because:
- Source code is publicly available
- Any competent developer can maintain it
- Community can continue development if original author stops
- No dependency on a company's continued operation

#### 3. Competitive Moat

Paradoxically, open source creates a stronger competitive position:
- **Impossible to replicate** — competitors cannot match the transparency
- **Community contributions** — improvements benefit everyone
- **Reduced support burden** — community answers common questions
- **Trust advantage** — "we have nothing to hide"

#### 4. Revenue Model Without Software Licensing

Open source does not mean "no revenue." Gray Logic generates revenue from:

| Revenue Stream | Open Source? | Notes |
|----------------|--------------|-------|
| Installation services | N/A | Primary revenue (Phase 1) |
| Custom hardware | No | Proprietary, purpose-built |
| Support contracts | No | Optional but valuable |
| Training courses | No | Knowledge transfer |
| Certification programme | No | Quality-controlled network |
| Consultancy | No | Complex project design |

The software is the **platform** that enables these revenue streams, not the product itself.

### What Is Open Source?

| Component | License | Rationale |
|-----------|---------|-----------|
| Gray Logic Core | GPL v3 | Copyleft ensures derivatives stay open |
| Bridge implementations | GPL v3 | Improvements return to community |
| Documentation | CC BY-SA 4.0 | Share knowledge, require attribution |
| Hardware designs | Proprietary | Revenue stream, quality control |

### What Is NOT Open Source?

| Component | Status | Rationale |
|-----------|--------|-----------|
| Hardware designs | Proprietary | Primary revenue stream |
| Certification materials | Proprietary | Quality control |
| Customer configurations | Private | Customer data |
| Internal business docs | Private | Competitive information |

### Trademark Strategy

The **license** (GPL v3) controls the code. The **trademark** ("Gray Logic") controls the name.

| Action | GPL v3 Allows? | Trademark Allows? |
|--------|----------------|-------------------|
| Use the code as-is | Yes | Yes (with attribution) |
| Modify and improve | Yes (must share source) | Must rename OR get permission |
| Call a fork "Gray Logic" | N/A | No (trademark violation) |
| Remove "Powered by Gray Logic" | No (copyright notices protected) | No |
| Claim to be "Gray Logic Certified" | N/A | Only if certified |

**For Certified Installers:**
- Licensed to use "Gray Logic" name and branding
- Customer clearly knows it's a Gray Logic system

**For Non-Certified Users:**
- Can use the software freely (GPL v3)
- Must share any modifications (copyleft)
- Cannot use "Gray Logic" name without permission
- Cannot remove attribution notices
- Customer still sees "Powered by Gray Logic"

**Trademark Registration:**
- Register "Gray Logic" with UK IPO
- Estimated cost: £170-270
- Classes: Software, electrical installation services, training

---

### Community Model

#### Contribution Guidelines
- Bug reports and fixes welcome
- Feature requests tracked publicly
- Pull requests reviewed and merged
- Major features discussed before implementation

#### Support Tiers
- **Community** (free): GitHub issues, community forum
- **Professional** (paid): Direct support, guaranteed response times

#### Governance
- Benevolent dictator model (initially)
- Clear roadmap published
- Breaking changes announced well in advance
- LTS (Long Term Support) releases for stability

---

## Competitive Advantages

### 1. No Vendor Lock-in

**The Problem:** Every competitor creates lock-in through proprietary software, dealer networks, or closed ecosystems. Customers are trapped.

**Gray Logic Solution:**
- Open source software — inspect, modify, fork
- Open standards (KNX, DALI, Modbus) — hardware works with any system
- Complete documentation handover — any electrician can maintain
- No dealer network required — direct relationship with installer

**Customer benefit:** "You own your system. If you're unhappy with us, you can leave."

### 2. True Offline Operation

**The Problem:** Most "smart" systems require cloud connectivity for basic functions. Internet outage = broken home.

**Gray Logic Solution:**
- 99%+ functionality works without internet
- Local voice processing (no Alexa/Google dependency)
- Local AI and automation
- Internet provides optional enhancements only

**Customer benefit:** "When the internet dies, your home doesn't."

### 3. Built by an Electrician

**The Problem:** Most home automation companies are software companies who don't understand electrical installation, building regulations, or what happens when things fail at 2 AM.

**Gray Logic Solution:**
- Founder is a qualified electrician
- Understands real-world installation constraints
- Designs for maintainability by other electricians
- Knows what fails and how to prevent it

**Customer benefit:** "Designed by someone who's actually installed these systems."

### 4. Multi-Decade Deployment Commitment

**The Problem:** Technology companies chase new features, deprecate old systems, and force upgrades. A system installed today may be unsupported in 3 years.

**Gray Logic Solution:**
- Explicit multi-decade deployment target
- Version-pinned deployments (no forced updates)
- Security patches without feature churn
- Technology choices made for longevity

**Customer benefit:** "Install once, run for a decade."

### 5. Transparent Pricing

**The Problem:** Competitor pricing is opaque. Dealers provide quotes without breakdowns. Customers cannot compare or verify value.

**Gray Logic Solution:**
- Published pricing tiers
- Itemised quotes showing hardware, labour, margin
- No hidden fees or mandatory service contracts
- Customer understands exactly what they're paying for

**Customer benefit:** "You'll know exactly where every pound goes."

---

## Business Model

### Phase 1: Boutique Installer (Years 1-3)

**Focus:** Build the product, refine the process, establish reputation.

| Revenue Stream | Description | Target |
|----------------|-------------|--------|
| Installation services | Primary income | £60,000 - £100,000/year |
| Hardware sales | Gray Logic Hub + recommended equipment | £10,000 - £20,000/year |
| Support contracts | Optional annual contracts | £2,000 - £5,000/year |

**Constraints:**
- Maximum 4-6 projects per year (part-time)
- Geographic limitation (travel distance)
- Single point of failure (founder only)

**Goals:**
- Prove the product works
- Build reference installations
- Collect testimonials and case studies
- Refine processes and documentation

### Phase 2: Expansion (Years 3-5)

**Focus:** Leverage proven product through training and partnerships.

| Revenue Stream | Description | Target |
|----------------|-------------|--------|
| Installation (own) | Continued direct installation | £50,000 - £80,000/year |
| Training courses | 2-day intensive for electricians | £10,000 - £20,000/year |
| Certification fees | Annual partner fees | £5,000 - £10,000/year |
| Hardware wholesale | Sales to certified installers | £20,000 - £40,000/year |
| Consultancy | Complex project design | £10,000 - £20,000/year |

**Expanded reach:**
- Certified installers extend geographic coverage
- Hardware revenue scales without direct involvement
- Training creates recurring touchpoints

### Phase 3: Maturity (Year 5+)

**Focus:** Sustainable business with multiple revenue streams.

**Options:**
1. **Continue boutique** — maintain small, profitable operation
2. **Scale training** — become primarily a training/certification business
3. **Product company** — focus on hardware and software, leave installation to partners
4. **Exit** — sell business to larger integrator or investor

**Exit viability:**
- Documented processes and systems
- Established customer base
- Recurring revenue from support and certification
- Trained partner network
- Open source ensures continuity for customers

---

## Risk Analysis

### Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Core software instability | Medium | High | Extensive testing, staged rollouts |
| KNX/DALI integration issues | Medium | Medium | Proven protocols, community knowledge |
| Hardware reliability | Low | High | Quality components, burn-in testing |
| Security vulnerabilities | Medium | High | Security-first design, regular audits |

### Business Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Low customer demand | Medium | High | Start with own home, friends/family |
| Competitor response | Low | Medium | Open source moat, niche positioning |
| Pricing pressure | Medium | Medium | Value differentiation, not price competition |
| Key person dependency | High | High | Documentation, eventual training programme |

### Market Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Economic downturn | Medium | High | Diversified customer base, support revenue |
| Technology shift | Low | Medium | Open standards, modular architecture |
| Regulatory changes | Low | Low | UK/EU compliance built-in |

---

## Financial Projections

### Year 1 (Foundation)

| Item | Projection |
|------|------------|
| Projects | 1 (own home) |
| Revenue | £0 (investment phase) |
| Hardware investment | £3,000 - £5,000 |
| Software development | Time investment |

### Year 2 (Validation)

| Item | Projection |
|------|------------|
| Projects | 2-3 (friends/family, discounted) |
| Revenue | £15,000 - £25,000 |
| Average project value | £8,000 - £12,000 |

### Year 3 (Growth)

| Item | Projection |
|------|------------|
| Projects | 3-4 (paying customers) |
| Revenue | £50,000 - £80,000 |
| Support contracts | £2,000 - £5,000 |
| Average project value | £15,000 - £20,000 |

### Year 4 (Expansion)

| Item | Projection |
|------|------------|
| Projects (own) | 3-4 |
| Training courses | 2-3 |
| Revenue | £80,000 - £120,000 |
| Support contracts | £5,000 - £10,000 |

### Year 5 (Maturity)

| Item | Projection |
|------|------------|
| Projects (own) | 2-3 |
| Certified installer projects | 4-6 |
| Training/certification | £15,000 - £25,000 |
| Hardware wholesale | £15,000 - £30,000 |
| Revenue | £100,000 - £150,000 |

*All projections are conservative estimates based on part-time operation. Full-time operation would scale proportionally.*

---

## Success Criteria

### Technical Success
- [ ] Complete control of all domains (lighting, climate, audio, video, security)
- [ ] Local voice control working reliably
- [ ] PHM detecting real issues before failure
- [ ] 10+ year runtime on deployed systems
- [ ] Any competent electrician can maintain installed systems

### Business Success
- [ ] Multiple deployed customer sites
- [ ] Recurring revenue from support tiers
- [ ] Effective day rate exceeds standard electrical work
- [ ] Positive customer referrals
- [ ] Viable business to sell on retirement

### Personal Success
- [ ] Proud of the product
- [ ] Work-life balance maintained
- [ ] Skills developed and portfolio built
- [ ] Clear path to exit if desired

### Institutional Success
- [ ] Knowledge preserved in documentation that outlives founder
- [ ] Network of certified installers operating independently
- [ ] Brand recognised and trusted in the market
- [ ] Succession options available (family, sale, or managed)
- [ ] Institution capable of continuing without any single person

---

## Conclusion

Gray Logic addresses a genuine market gap with a differentiated approach. By combining open source transparency with professional installation and purpose-built hardware, it offers homeowners a credible alternative to both expensive proprietary systems and unreliable DIY solutions.

The phased approach — starting as a boutique installer, validating the product, then expanding through training and partnerships — minimises risk while building toward a sustainable institution.

Gray Logic is being built not just as a business, but as an institution to serve generations. The specific technologies will change. Economic systems may transform. But the fundamental human need — buildings that work well for the people inside them — will remain. That is what we serve.

The open source strategy creates competitive advantage through trust, longevity, and community contribution. The certification network builds an asset that compounds over time. The obsessive documentation ensures knowledge survives beyond any individual.

**Gray Logic: Making buildings work well for people, for generations.**

---

## Related Documents

- [Institutional Principles](institutional-principles.md) — Building for generations
- [Vision](../overview/vision.md) — Strategic vision and goals
- [Principles](../overview/principles.md) — Non-negotiable design rules
- [Pricing Model](pricing.md) — Detailed pricing structure
- [Sales Specification](sales-spec.md) — Customer journey and process
- [Go-to-Market Strategy](go-to-market.md) — Growth plan
