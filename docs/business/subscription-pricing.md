---
title: Gray Logic Cloud Subscription Pricing
version: 0.1.0
status: draft
implementation_status: Year 2+
last_updated: 2026-01-18
depends_on:
  - business/pricing.md
  - architecture/cloud-relay.md
roadmap: Year 2+
---

# Gray Logic Cloud Subscription Pricing

> **Status**: Draft for evaluation

This document specifies the subscription pricing model for Gray Logic Cloud services.

---

## Pricing Philosophy

### Principles

1. **Free core, premium cloud** — Local functionality is free; cloud features are subscription-based
2. **Value-aligned pricing** — Higher tiers provide significantly more value
3. **Sustainable revenue** — Pricing covers infrastructure + margin for reinvestment
4. **Competitive positioning** — Below Savant/Crestron monitoring, above DIY solutions

### Target Market Segments

| Segment | Monthly Spend Tolerance | Value Expectation |
|---------|------------------------|-------------------|
| **DIY/Prosumer** | £5-15/month | Basic remote access |
| **Residential Owner** | £20-50/month | Full remote + CCTV |
| **High-Net-Worth** | £100+/month | Premium everything, white-glove |
| **Commercial** | £200-500+/month | Multi-site, SLA, analytics |

---

## Subscription Tiers

### Overview

| Tier | Monthly | Annual | Target Customer |
|------|---------|--------|-----------------|
| **Free** | £0 | £0 | DIY, budget-conscious |
| **Connect** | £9.99 | £99/year | Basic remote access |
| **Secure** | £24.99 | £249/year | CCTV remote viewing |
| **Premium** | £49.99 | £499/year | Full features, residential |
| **Estate** | £99.99 | £999/year | Large residential, priority |
| **Commercial** | Custom | Custom | Offices, multi-site |

**Note**: Annual pricing = ~17% discount (2 months free)

---

### Free Tier (£0)

**Purpose**: Entry point, demonstrates value of paid tiers.

**Included**:
- ✅ Full local functionality (100%)
- ✅ VPN remote access (WireGuard → Web Interface)
- ✅ Mobile app: **Status Only** (see below)
- ✅ Software updates
- ✅ Community forum support

**Mobile App (Free Tier)**:
```
┌───────────────────────────┐
│      GRAY LOGIC APP       │
│                           │
│  🏠 Oak Street Home       │
│  ────────────────────     │
│  Status: ● ONLINE         │
│  Mode: Home               │
│  Last seen: Just now      │
│                           │
│  ┌─────────────────────┐  │
│  │ 🔒 Upgrade to        │  │
│  │    control remotely  │  │
│  └─────────────────────┘  │
│                           │
│  [VPN: Use web interface] │
└───────────────────────────┘
```
- Shows: Online/Offline status, current mode, alerts count
- Does NOT allow: Control, scene activation, camera viewing
- **Upgrade prompt**: "Upgrade to Connect for full remote control"

**VPN Alternative (Free)**:
- User can still control via VPN → Web interface
- Full functionality, just not through app

**Not Included**:
- ❌ App control (requires Connect+)
- ❌ Push notifications (requires Connect+)
- ❌ Cloud configuration backup
- ❌ Remote CCTV

**Target**: DIY enthusiasts, highly technical users, budget-conscious customers.

---

### Connect Tier (£9.99/month)

**Purpose**: Consumer-friendly remote access.

**Included**:
- ✅ Everything in Free, plus:
- ✅ Cloud API relay (remote control via app)
- ✅ Cloud configuration backup (encrypted)
- ✅ Enhanced push notifications (customisable)
- ✅ Email support (48-hour response)

**Not Included**:
- ❌ Remote CCTV viewing
- ❌ Video clip cloud storage
- ❌ AI insights
- ❌ Priority support

**Target**: Residential customers wanting "check my home from anywhere".

**Infrastructure Cost per Site**:
- API relay bandwidth: ~£1.50/month
- Backup storage (encrypted): ~£0.50/month
- **Margin**: ~80%

---

### Secure Tier (£24.99/month)

**Purpose**: Full CCTV remote access.

**Included**:
- ✅ Everything in Connect, plus:
- ✅ Remote CCTV live viewing (up to 4 cameras concurrent)
- ✅ Video clip cloud storage (7 days retention)
- ✅ Camera event snapshots (cloud-stored)
- ✅ MFA management portal
- ✅ Email + chat support (24-hour response)

**Not Included**:
- ❌ Extended video storage (30+ days)
- ❌ AI insights and reporting
- ❌ Multi-site dashboard
- ❌ Priority support

**Target**: Security-conscious homeowners wanting remote camera access.

**Infrastructure Cost per Site**:
- Video relay bandwidth: ~£5/month (variable)
- Video storage (7 days): ~£3/month
- API + backup: ~£2/month
- **Margin**: ~60%

---

### Premium Tier (£49.99/month)

**Purpose**: Full feature set for discerning residential.

**Included**:
- ✅ Everything in Secure, plus:
- ✅ Extended video storage (30 days)
- ✅ AI insights and reporting (PHM triage, health digests)
- ✅ Advanced analytics dashboard
- ✅ Energy optimization recommendations
- ✅ Phone support (24-hour response)
- ✅ 10% discount on support callouts

**Not Included**:
- ❌ Multi-site management
- ❌ SLA guarantees
- ❌ Dedicated contact

**Target**: Premium residential, comfort + security focused.

**Infrastructure Cost per Site**:
- Video relay: ~£6/month
- Extended storage: ~£8/month
- AI processing: ~£5/month
- API + backup: ~£2/month
- **Margin**: ~58%

---

### Estate Tier (£99.99/month)

**Purpose**: Large properties, multiple buildings, high expectations.

**Included**:
- ✅ Everything in Premium, plus:
- ✅ Multi-site dashboard (up to 5 buildings)
- ✅ Extended video storage (90 days)
- ✅ Priority support (4-hour response)
- ✅ Quarterly system review call
- ✅ Dedicated account manager
- ✅ 20% discount on support callouts
- ✅ SLA guarantee (99.9% cloud uptime)

**Target**: Estates, large residential, high-net-worth individuals.

**Infrastructure Cost per Site**:
- Multi-site overhead: ~£15/month
- Extended storage: ~£15/month
- Priority support allocation: ~£10/month
- Base services: ~£15/month
- **Margin**: ~45%

---

### Commercial Tier (Custom Pricing)

**Purpose**: Offices, retail, hospitality, multi-site portfolios.

**Pricing Model**:
```
Base fee: £199/month
+ Per building: £49/month
+ Per camera (remote): £5/month
+ Advanced analytics: +£99/month
+ ANPR integration: +£49/month
```

**Example: 3-Building Office Complex**:
```
Base:           £199
3 buildings:    £147 (3 × £49)
20 cameras:     £100 (20 × £5)
Analytics:      £99
─────────────────────
Total:          £545/month
```

**Included**:
- ✅ Everything in Estate, plus:
- ✅ Unlimited buildings
- ✅ RBAC for facility managers
- ✅ Integration API access
- ✅ Custom analytics
- ✅ 2-hour priority response
- ✅ On-site support visits (quarterly)
- ✅ 99.95% SLA

**The Vision — Facility Management at Scale**:

Example: Gym chain with 50 locations managed by a facilities company.

```
┌──────────────────────────────────────────────────────────────────────────┐
│                    FACILITIES COMPANY DASHBOARD                           │
│                                                                           │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐           │
│  │  GYM LOCATION 1 │  │  GYM LOCATION 2 │  │  GYM LOCATION N │           │
│  │  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓ │  │  ▓▓▓▓▓▓▓▓▓▓░░░░ │  │  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓ │           │
│  │  Status: OK     │  │  Status: ALERT  │  │  Status: OK     │           │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘           │
│                              │                                            │
│                    ┌─────────▼─────────┐                                 │
│                    │ ⚠️ DALI Fault      │                                 │
│                    │ Gym 2, Zone B      │                                 │
│                    │ 3 lights offline   │                                 │
│                    │ [Dispatch Tech]    │                                 │
│                    └───────────────────┘                                 │
└──────────────────────────────────────────────────────────────────────────┘
```

- Real-time DALI/lighting fault reporting → Auto-dispatch technician
- HVAC/PHM metrics aggregated across all sites
- Energy benchmarking between locations
- Centralised firmware updates

---

## Hardware Sales (DIY/Installer)

> **Note**: Documented for future implementation. See also [Pricing](pricing.md).

| Product | Target | Suggested RRP |
|---------|--------|---------------|
| Hub Only | Existing installer | £800 |
| Starter Kit | DIY enthusiast | £2,500 |
| Standard Kit | Serious DIY | £5,000 |
| Installer Pack (5× Hubs) | Certified partner | £3,500 |

**Sales Channels**: Website direct, Amazon (Hub only for visibility).
**Support for DIY**: Full docs, community forum, optional paid setup calls.

---

## Value Comparison

### vs Competitors

| Feature | Gray Logic | Savant Monitoring | Control4 Care | Loxone Cloud |
|---------|------------|-------------------|---------------|--------------|
| Remote access | ✅ Connect+ | ✅ Included | ✅ Included | ✅ Included |
| Remote CCTV | ✅ Secure+ | ✅ Premium | ✅ Premium | ❌ |
| AI insights | ✅ Premium+ | ❌ | ❌ | ❌ |
| Local-first | ✅ Always | ❌ | ❌ | ✅ |
| Open source core | ✅ | ❌ | ❌ | ❌ |
| Pricing (typical) | **£25-100** | **£150-300** | **£100-200** | **£50-80** |

### Value Proposition by Tier

| Tier | Monthly Cost | Comparable To | Justification |
|------|--------------|---------------|---------------|
| Connect | £9.99 | Netflix subscription | "Check your home from anywhere" |
| Secure | £24.99 | Home insurance add-on | "Peace of mind, see your cameras" |
| Premium | £49.99 | Premium gym membership | "Full smart home experience" |
| Estate | £99.99 | Property management | "White-glove service for estates" |

---

## Revenue Projections

### Year 2 (Launch Year)

**Assumptions**:
- 50 new installations in Year 1
- 70% conversion to paid tier
- Distribution: 40% Connect, 35% Secure, 20% Premium, 5% Estate

| Tier | Sites | Monthly Revenue | Annual Revenue |
|------|-------|-----------------|----------------|
| Free | 15 | £0 | £0 |
| Connect | 14 | £140 | £1,680 |
| Secure | 12 | £300 | £3,600 |
| Premium | 7 | £350 | £4,200 |
| Estate | 2 | £200 | £2,400 |
| **Total** | **50** | **£990** | **£11,880** |

### Year 3 (Scaling)

**Assumptions**:
- 150 total installations
- 75% conversion to paid
- Better distribution toward premium tiers

| Tier | Sites | Monthly Revenue | Annual Revenue |
|------|-------|-----------------|----------------|
| Free | 38 | £0 | £0 |
| Connect | 34 | £340 | £4,080 |
| Secure | 39 | £975 | £11,700 |
| Premium | 28 | £1,400 | £16,800 |
| Estate | 11 | £1,100 | £13,200 |
| **Total** | **150** | **£3,815** | **£45,780** |

### Year 5 (Mature)

**Assumptions**:
- 500 residential + 50 commercial sites
- 80% paid conversion
- Commercial average: £400/month

| Category | Sites | Monthly Revenue | Annual Revenue |
|----------|-------|-----------------|----------------|
| Residential Paid | 400 | £16,000 | £192,000 |
| Commercial | 50 | £20,000 | £240,000 |
| **Total** | **550** | **£36,000** | **£432,000** |

---

## Infrastructure Costs

### Per-Site Costs (Estimated)

| Component | Connect | Secure | Premium | Estate |
|-----------|---------|--------|---------|--------|
| API relay | £1.50 | £1.50 | £1.50 | £2.00 |
| Push notifications | £0.20 | £0.20 | £0.20 | £0.20 |
| Config backup | £0.50 | £0.50 | £0.50 | £1.00 |
| Video relay | — | £5.00 | £6.00 | £8.00 |
| Video storage | — | £3.00 | £8.00 | £15.00 |
| AI processing | — | — | £5.00 | £5.00 |
| Support allocation | — | — | £2.00 | £10.00 |
| **Total/site** | **£2.20** | **£10.20** | **£23.20** | **£41.20** |

### Fixed Costs (Monthly)

| Item | Cost | Notes |
|------|------|-------|
| Cloud servers | £500 | Scales with usage |
| CDN/bandwidth | £200 | Video-heavy |
| Monitoring | £100 | 24/7 |
| Security | £150 | DDoS, WAF |
| Support tools | £100 | Ticketing, etc. |
| **Total fixed** | **£1,050** | Break-even ~50 paid sites |

---

## Upselling Strategy

### Free → Connect

Trigger: "You're using VPN — try our app for easier access!"
Offer: First month free

### Connect → Secure

Trigger: CCTV cameras detected but not viewable remotely
Offer: "View your cameras from anywhere — upgrade now"

### Secure → Premium

Trigger: PHM alerts, energy usage
Offer: "Get AI-powered insights — understand your home better"

### Premium → Estate

Trigger: Multi-building setup detected
Offer: "Manage all your properties from one dashboard"

---

## Pricing Evaluation Points

### Questions for Decision

1. **Connect at £9.99**: Is this competitive enough to capture DIY market?
2. **Secure at £24.99**: Right price for CCTV access?
3. **Premium at £49.99**: Does AI justify premium over Secure?
4. **Estate at £99.99**: Is £100/month right for high-net-worth?
5. **Commercial base at £199**: Competitive with facility management tools?

### Market Research Needed

- [ ] Survey existing customers on price sensitivity
- [ ] Benchmark against Loxone Cloud pricing in detail
- [ ] Assess AWS/GCP costs more precisely
- [ ] Validate video storage costs with sample data

---

## References

- [Cloud Relay Architecture](../architecture/cloud-relay.md) — Technical specification
- [Installation Pricing](pricing.md) — One-time installation costs
- [AI Premium Features](../intelligence/ai-premium-features.md) — What AI provides
