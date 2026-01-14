---
title: Gray Logic Sales Specification
version: 1.0.0
status: active
last_updated: 2026-01-14
depends_on:
  - business-case.md
  - pricing.md
  - ../deployment/residential.md
  - ../deployment/handover-pack-template.md
---

# Gray Logic Sales Specification

## Overview

This document defines the complete customer journey from initial enquiry through installation to ongoing support. It serves as:

1. **Process guide** — Step-by-step sales and delivery process
2. **Quality standard** — Minimum requirements for every project
3. **Training material** — For future certified installers

---

## Customer Journey

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   ENQUIRY   │───▶│  DISCOVERY  │───▶│  PROPOSAL   │───▶│  CONTRACT   │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                                                │
                                                                ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   SUPPORT   │◀───│  HANDOVER   │◀───│COMMISSIONING│◀───│INSTALLATION │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

**Typical timeline:** 8-16 weeks from enquiry to handover

---

## Stage 1: Enquiry

### Lead Sources

| Source | Typical Quality | Conversion Rate | Effort |
|--------|-----------------|-----------------|--------|
| Referral | High | 40-60% | Low |
| Architect/builder | High | 30-50% | Medium |
| Website enquiry | Medium | 15-25% | Medium |
| Existing electrical customer | High | 30-40% | Low |
| Trade show/event | Variable | 10-20% | High |

### Initial Response

**Target:** Respond within 24 hours (business days)

**Initial contact should:**
- Thank them for enquiry
- Ask qualifying questions
- Provide brief overview of Gray Logic
- Offer discovery call/meeting
- Set expectations on timeline

**Email Template:**

```
Subject: Re: Home Automation Enquiry

Hi [Name],

Thank you for your enquiry about Gray Logic home automation.

To understand your requirements better, could you tell me:
- Is this a new build, renovation, or existing property?
- What's driving your interest in automation? (convenience, energy, security?)
- What's your approximate budget range?
- What's your timeline?

I'd be happy to arrange a call or site visit to discuss your project in detail.

Best regards,
[Name]
Gray Logic
```

### Qualification Criteria

**Proceed to Discovery if:**
- Budget aligns with our minimum (£8,000+)
- Timeline is realistic (8+ weeks)
- Property is suitable (cabling access, electrical capacity)
- Customer understands automation is a considered purchase

**Politely decline if:**
- Budget is under £5,000 (suggest DIY alternatives)
- Expecting "smart home in a box" with no installation
- Unrealistic timeline (wants it "next week")
- Red flags (aggressive negotiators, unclear decision-maker)

---

## Stage 2: Discovery

### Discovery Call (Remote)

**Duration:** 30-45 minutes

**Objectives:**
- Understand customer's goals and priorities
- Assess property suitability
- Identify decision-makers
- Qualify budget and timeline
- Determine if site survey needed

**Discovery Questions:**

| Area | Questions |
|------|-----------|
| Property | Type? Age? Size? New build or existing? |
| Goals | What problems are you trying to solve? |
| Priorities | If you could only have one thing, what would it be? |
| Experience | Used smart home tech before? What worked/didn't? |
| Budget | Have you set aside a budget for this? |
| Timeline | When would you like the work completed? |
| Decision | Who else is involved in this decision? |
| Other works | Any other building works planned? |

**Outcome:** Decide whether to proceed to site survey

---

### Site Survey

**Duration:** 2-3 hours (typical residential)

**Fee:** £400 (credited against project if proceeds)

**What to Assess:**

| Area | Check |
|------|-------|
| Electrical | Consumer unit capacity, wiring type, earth quality |
| Cabling routes | Can we run KNX bus? Cable access? |
| Distribution board | Space for actuators? Location suitable? |
| Network | Existing infrastructure? Router location? |
| Rooms | Count, usage, lighting points, window types |
| Existing systems | Heating type, existing automation, alarm |
| Access | Any areas difficult to reach? Listed building? |

**Site Survey Checklist:**

```
PROPERTY DETAILS
□ Address
□ Property type (detached/semi/terrace/flat)
□ Approximate size (m²)
□ Number of floors
□ Age of property
□ Listed/conservation area?

ELECTRICAL
□ Consumer unit location
□ Number of ways
□ Spare ways available
□ Earth type (TN-S, TN-C-S, TT)
□ Wiring type (T&E, singles in conduit)
□ Age of electrical installation

ROOMS (for each)
□ Room name/function
□ Lighting points (count, type)
□ Switching points (count)
□ Windows/blinds (count, type)
□ Heating (radiator, UFH, other)
□ Special requirements

INFRASTRUCTURE
□ Network cabinet/location
□ Existing structured cabling
□ WiFi coverage
□ Heating system type
□ Alarm system (make, model)
□ Existing automation

CABLE ROUTES
□ Loft access
□ Under floor access
□ Vertical runs possible
□ First-fix or retrofit?

PHOTOS REQUIRED
□ Consumer unit (open)
□ Each room (general)
□ Problem areas
□ Cable route locations
```

---

### Requirements Document

After discovery and site survey, create a Requirements Document:

**Section 1: Customer Profile**
- Name, contact details
- Property address
- Decision-makers
- Budget indication
- Timeline requirements

**Section 2: Priorities**
- Primary goals (ranked)
- Must-haves vs nice-to-haves
- Specific requirements or constraints

**Section 3: Technical Assessment**
- Property suitability
- Electrical assessment
- Cabling assessment
- Special considerations

**Section 4: Recommended Scope**
- Suggested tier (Essential/Standard/Premium)
- Domains to include
- Optional additions
- Items explicitly excluded

---

## Stage 3: Proposal

### Proposal Document Structure

Every proposal must include:

#### Cover Page
- Customer name
- Property address
- Proposal date
- Validity period (30 days typical)
- Version number

#### Executive Summary (1 page)
- What we're proposing (plain English)
- Total investment
- Key benefits
- Timeline summary

#### Scope of Works
- Domains included (with brief description)
- Room-by-room breakdown
- What's NOT included (explicit)

#### Technical Specification
- Hardware list (make, model, quantity)
- Software components
- Network requirements
- Integration points

#### Pricing Breakdown

| Category | Items | Cost |
|----------|-------|------|
| Hardware | Itemised list | £X,XXX |
| Installation labour | X days @ £XXX | £X,XXX |
| Configuration | X days @ £XXX | £X,XXX |
| Commissioning | X days @ £XXX | £X,XXX |
| Documentation/training | X days @ £XXX | £XXX |
| **Subtotal** | | **£XX,XXX** |
| Contingency (10%) | | £X,XXX |
| **Total** | | **£XX,XXX** |

*VAT shown separately if applicable*

#### Timeline
- Project phases
- Key milestones
- Dependencies (other trades, delivery lead times)
- Estimated completion date

#### Support Options
- Support tier options with pricing
- What each tier includes
- Recommendation

#### Terms and Conditions
- Payment terms
- Warranty coverage
- Cancellation policy
- Liability limitations

#### About Gray Logic
- Brief company overview
- Open source philosophy
- Relevant experience/portfolio
- Contact details

---

### Proposal Presentation

**Option A: In-person presentation**
- Walk through proposal
- Answer questions
- Discuss options/alternatives
- Leave physical copy

**Option B: Video call**
- Screen share proposal
- Same structure as in-person
- Follow up with PDF

**Option C: Email (simpler projects)**
- Send PDF with cover email
- Offer to discuss on call
- Follow up in 3-5 days

**Key messages to convey:**
1. "You'll own your system — no lock-in"
2. "This will work for 10+ years"
3. "We document everything — anyone can maintain it"
4. "The price is transparent — you see where every pound goes"

---

### Handling Objections

| Objection | Response |
|-----------|----------|
| "It's expensive" | "Let's look at what you could remove to reduce cost" / "Compare to Loxone/Control4 for same scope" |
| "Can't you do it cheaper?" | "We can adjust scope, but we don't reduce quality or margins" |
| "I've seen cheaper on Amazon" | "Those are consumer gadgets, not professional systems. Different reliability, no integration" |
| "What if you go out of business?" | "Software is open source — any developer can support it. Full documentation provided" |
| "Why not Home Assistant?" | "It's free to try but costs thousands in time and frustration. We provide reliability" |
| "Need to think about it" | "Of course. Proposal valid 30 days. Any questions I can answer?" |

---

## Stage 4: Contract

### Contract Structure

#### Terms and Conditions

**Payment Terms:**
- 30% deposit on contract signature
- 40% on hardware delivery / first fix complete
- 25% on commissioning complete
- 5% retained for 30 days (snagging)

**Change Control:**
- Changes requested after contract must be in writing
- Changes will be quoted separately
- Customer must approve quote before work proceeds
- Changes may affect timeline

**Warranty:**
- Hardware: Manufacturer warranty (typically 2 years)
- Installation workmanship: 12 months
- Software: Ongoing updates (see support tier)

**Liability:**
- Limit of liability = contract value
- No liability for consequential losses
- Professional indemnity insurance held

**Cancellation:**
- Customer may cancel within 14 days (cooling-off)
- After 14 days: deposit non-refundable
- After hardware ordered: hardware cost non-refundable
- Cancellation must be in writing

**Dispute Resolution:**
- Good faith negotiation first
- Mediation if negotiation fails
- UK courts for unresolved disputes

---

### Contract Checklist

Before contract signature:

```
□ Proposal accepted (signed or written confirmation)
□ Scope clearly defined
□ Price agreed (including any negotiated changes)
□ Payment terms agreed
□ Timeline agreed
□ Support tier selected
□ Terms and conditions provided
□ Customer questions answered
□ Decision-makers available to sign
```

---

## Stage 5: Installation

### Pre-Installation

**2-4 weeks before:**
- Order hardware (KNX, hub, sensors)
- Confirm delivery dates
- Schedule installation dates with customer
- Coordinate with other trades if applicable
- Confirm access arrangements

**1 week before:**
- Verify hardware received
- Test Gray Logic Hub (basic power-on)
- Prepare configuration template
- Confirm schedule with customer
- Check nothing has changed (building works, etc.)

---

### Installation Process

**Phase 1: First Fix** (if new build/renovation)
- Run KNX bus cable
- Run Cat6 to panel locations
- Install back boxes for push buttons
- Coordinate with electrician for lighting circuits

**Phase 2: Distribution Board**
- Install KNX power supply
- Install KNX actuators
- Connect to lighting circuits
- Label everything clearly

**Phase 3: Devices**
- Install push buttons
- Install sensors
- Install thermostats
- Install touch panels (if included)

**Phase 4: Infrastructure**
- Install Gray Logic Hub
- Configure network
- Connect to KNX (via knxd)
- Verify connectivity

**Phase 5: Initial Configuration**
- Import room/device structure
- Basic addressing
- Verify all devices responding
- Test basic switching

---

### Installation Quality Standards

| Area | Standard |
|------|----------|
| Cable management | Neat, labelled, accessible |
| Actuator mounting | Secure, correct orientation |
| Push buttons | Level, clean, properly aligned |
| Documentation | Updated as-built drawings |
| Testing | Every circuit tested |
| Photography | Progress photos for records |

---

## Stage 6: Commissioning

### Commissioning Process

**Duration:** 1-3 days (depending on complexity)

**Phase 1: Functional Testing**
- Test every light circuit
- Test every blind/shading
- Test every climate zone
- Test every sensor
- Document any issues

**Phase 2: Scene Programming**
- Create scenes per customer requirements
- Test scenes in all conditions
- Adjust lighting levels as needed
- Fine-tune timings

**Phase 3: Automation Setup**
- Configure schedules
- Set up presence-based automation
- Configure weather responses
- Test failure modes

**Phase 4: Integration Testing**
- Test all integrations (audio, security, etc.)
- Test mobile app access
- Test remote access (if configured)
- Test voice control (if included)

**Phase 5: Optimisation**
- Fine-tune settings based on feedback
- Adjust scene timings
- Optimise sensor sensitivity
- Performance testing

---

### Commissioning Checklist

```
LIGHTING
□ All circuits switch correctly
□ All dimmers respond full range
□ All push buttons function
□ Scenes trigger correctly
□ Schedules execute on time

CLIMATE
□ All zones respond to setpoint
□ Thermostats display correctly
□ Schedules work
□ Presence/absence modes work

BLINDS
□ All blinds respond up/down
□ Position control accurate
□ Sun protection triggers
□ Weather protection triggers

PRESENCE
□ All sensors detect occupancy
□ Sensitivity appropriate
□ False positives resolved
□ Timeout settings correct

INTEGRATION
□ Audio system responds
□ Security status visible
□ CCTV feeds accessible
□ Intercom functional

MOBILE/REMOTE
□ App connects locally
□ App connects remotely (if configured)
□ All controls functional
□ Notifications working

VOICE (if included)
□ Wake word responsive
□ Commands understood
□ Actions execute correctly
□ Feedback appropriate
```

---

## Stage 7: Handover

### Customer Training

**Duration:** 2-4 hours (depending on complexity)

**Training Agenda:**

| Topic | Duration | Content |
|-------|----------|---------|
| System overview | 15 mins | Architecture, what's where |
| Daily operation | 30 mins | Push buttons, app, voice |
| Scenes and modes | 20 mins | How to use, how to modify |
| Schedules | 15 mins | Viewing, basic changes |
| Troubleshooting | 20 mins | Common issues, what to try |
| Support process | 10 mins | How to get help |
| Q&A | 30 mins | Customer questions |

**Training Materials:**
- Quick-start guide (laminated, stays in property)
- Full user manual (PDF)
- Support contact details
- Login credentials (sealed envelope)

---

### Handover Documentation

Every project receives a complete handover pack per [Handover Pack Template](../deployment/handover-pack-template.md):

**Contents:**
1. System overview document
2. As-built drawings
3. Device inventory
4. Configuration backup
5. Login credentials
6. User manual
7. Support information
8. Warranty documentation

**Format:**
- Physical folder (for property)
- Digital copy (email to customer, stored by us)

---

### Sign-Off Process

**Practical Completion:**
- All contracted works complete
- Customer walkthrough completed
- Snagging list created (if any)
- Handover documentation provided
- Training completed

**Sign-off Document:**
```
PRACTICAL COMPLETION CERTIFICATE

Project: [Address]
Date: [Date]
Contract Value: £[Amount]

I confirm that:
□ All contracted works have been completed
□ I have received training on system operation
□ I have received all handover documentation
□ I understand the support process

Snagging items (if any):
1. [Item]
2. [Item]

Signed: _________________ Date: _________
        (Customer)

Signed: _________________ Date: _________
        (Gray Logic)
```

---

## Stage 8: Post-Installation Support

### 30-Day Settling-In Period

**Included in all projects:**
- Unlimited phone/email support
- Remote adjustments
- One return visit for adjustments
- Scene/schedule tweaks

**Purpose:** Let customer live with system, identify any issues or preferences

---

### First-Year Review

**3 months after handover:**
- Phone check-in
- Any issues?
- Any desired changes?
- Feedback collection

**12 months after handover:**
- On-site visit (Premium tier) or phone call
- System health check
- Any desired changes?
- Support renewal discussion

---

### Ongoing Support

As per selected support tier (see [Pricing Model](pricing.md)):

| Tier | Annual Fee | Response | Includes |
|------|------------|----------|----------|
| Community | Free | Best effort | Forum, docs |
| Standard | £300 | 48 hours | Email, remote diagnostics |
| Premium | £600 | 24 hours | Phone, annual visit |
| Enterprise | £1,200 | 4 hours | Quarterly reviews |

---

### Upsell Opportunities

**After successful installation, customers may want:**
- Additional rooms/zones
- New domains (audio, security, pool)
- Touch panels
- Voice control upgrade
- Integration expansions

**Approach:** Don't push at handover. Wait 6-12 months, then mention possibilities during support interactions.

---

### Referral Programme

**Referral reward:** £500 credit or cash for successful referral

**Qualification:**
- Referred customer must complete a project
- Minimum project value £10,000
- Referrer must be existing customer

**Process:**
1. Existing customer refers friend/colleague
2. We quote and complete project
3. On final payment, referrer receives reward
4. Both parties thanked appropriately

---

## Sales Tools

### Required Materials

| Material | Format | Purpose |
|----------|--------|---------|
| Company brochure | PDF/print | Initial enquiry |
| Portfolio/case studies | PDF/web | Demonstrate capability |
| Proposal template | Word/PDF | Quotations |
| Contract template | Word/PDF | Agreements |
| Handover pack template | Word/PDF | Deliverables |
| Quick-start guide | PDF/print | Customer reference |
| Business cards | Print | Networking |

---

### CRM Requirements

Track for each opportunity:

| Field | Purpose |
|-------|---------|
| Contact details | Communication |
| Property details | Project scoping |
| Enquiry source | Marketing effectiveness |
| Discovery date | Pipeline tracking |
| Proposal date | Conversion tracking |
| Contract date | Revenue forecasting |
| Estimated value | Pipeline value |
| Stage | Current status |
| Next action | Follow-up |
| Notes | Context |

---

## Quality Metrics

### Track and Review

| Metric | Target | Review Frequency |
|--------|--------|------------------|
| Enquiry response time | < 24 hours | Monthly |
| Discovery to proposal | < 2 weeks | Monthly |
| Proposal to decision | < 4 weeks | Monthly |
| Proposal win rate | > 30% | Quarterly |
| Project completion on time | > 80% | Per project |
| Project completion on budget | > 90% | Per project |
| Customer satisfaction | > 4.5/5 | Per project |
| Referral rate | > 25% | Annually |

---

## Related Documents

- [Institutional Principles](institutional-principles.md) — Building for generations
- [Business Case](business-case.md) — Strategic context
- [Pricing Model](pricing.md) — Pricing details
- [Go-to-Market](go-to-market.md) — Marketing approach
- [Residential Deployment](../deployment/residential.md) — Technical installation
- [Handover Pack Template](../deployment/handover-pack-template.md) — Deliverables
