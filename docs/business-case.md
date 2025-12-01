# Business Case – Gray Logic Stack

## 1. Purpose

This document explains **why** Gray Logic exists as a commercial offering:

- What problem it solves.
- Who it is for.
- How it makes money.
- Where it sits relative to existing platforms (Loxone, Crestron, Control4, “just use Home Assistant”).
- What success looks like and when I should walk away.

It is written **for me**, not for clients.

---

## 2. Problem Statement

### 2.1 Technical problem

In the UK market, I see a gap between:

1. **High-end proprietary platforms** (Loxone, Crestron, Control4, etc.)

   - Strong UX and integration, but:
     - Closed ecosystems and licence models.
     - Opaque to non-certified installers.
     - Painful to support or change if the original integrator disappears.

2. **DIY / enthusiast smart home setups** (Home Assistant, openHAB, etc.)

   - Powerful and flexible, but:
     - Usually not documented like a professional project.
     - Deployed on “whatever hardware” without proper network segmentation, backups, or runbooks.
     - Hard to justify as a paid, supportable product.

3. **Traditional electrical and small BMS work**
   - Safe and compliant, but:
     - Limited automation/integration.
     - No single pane of glass for plant, lighting, CCTV, and alarms.
     - Little or no structured remote monitoring.

**Key technical problem:**  
Clients with **high-end homes or small leisure/commercial sites** want BMS-like insight and control, but without going full industrial BMS or locking themselves into one proprietary smart home vendor.

---

### 2.2 Business problem (for me)

From my side:

- Standard electrical work is **limited in margin and scale**.
- I already have skills in:
  - Linux, Docker, VPNs, networking.
  - Documentation, runbooks, backup/restore.

Right now there is no **“Darren-grade”, open-standards automation product** I can sell repeatedly and support.

Gray Logic is my attempt to turn my **hybrid electrician + infra skillset** into:

- A **repeatable product** (not just day-rate engineering).
- With **recurring revenue** (support/maintenance).
- That can grow without me doing only raw labour forever.

---

## 3. Solution Overview

**Gray Logic** is a **productised building automation stack** aimed at:

- High-end homes / small estates.
- Pools, spas, and leisure facilities.
- Light commercial / mixed-use buildings with plant + lighting + CCTV.

It delivers:

1. **Open-standards field layer**

   - KNX, DALI, Modbus, MQTT, dry contacts.
   - Any competent electrician/controls engineer can understand and maintain the wiring.

2. **Open-source control layer**

   - openHAB as the main automation brain (devices, scenes, modes, schedules).
   - Node-RED for cross-system flows and “weird but useful” logic.
   - No second proprietary “brain” – Gray Logic owns the logic on a Gray Logic site.

3. **Professional infrastructure layer**
   - Linux + Docker.
   - Traefik, WireGuard, metrics, logging, backups.
   - Segmented networks (control, CCTV, guest).
   - Handover docs, I/O maps, network diagrams, runbooks.

In short:

> **“Mini-BMS quality, using open standards, for sites that are too small or too specialised for full BMS, and too serious for toy smart home setups.”**

---

## 4. Target Customers

### 4.1 Primary segments

1. **High-end residential clients**

   - New builds or deep refurbs.
   - Value comfort, lighting, cinema, security, and energy use.
   - Already spending on kitchens, AV, pools, etc.
   - Want **long-term maintainability**, not just a flashy app.

2. **Leisure / pool / spa sites**

   - Standalone pools or part of small hotels/leisure centres.
   - Have plant (boilers, filters, pumps), ventilation, and tight temperature/humidity needs.
   - Want simple dashboards, alarms, and remote access without paying for full BMS.

3. **Small mixed-use / light commercial**
   - Offices with showrooms, small multi-use buildings.
   - Need lighting control, environment monitoring, plant visibility, CCTV/alarm integration.

### 4.2 Stakeholders

- **End clients / owners** – care about comfort, reliability, remote visibility, and having “one person to call”.
- **Architects / builders / M&E designers** – care about clear scope, standards, and someone who can bridge electrical + IT.
- **Facilities/maintenance** – care about safe operation, documentation, and not being locked out of the system.

---

## 5. Value Proposition

### 5.1 For clients

Compared to “just smart home gadgets”:

- **Safer and more robust**

  - Physical controls and life safety remain independent.
  - The system keeps working if the server or internet is down.

- **Better long-term maintainability**

  - Built on open protocols and open-source software.
  - Documented so another professional can support it in future.

- **Single pane of glass**
  - One place to see:
    - Plant status.
    - Lighting and modes.
    - CCTV health.
    - Alarm states.
    - Environment trends.

Compared to proprietary platforms (Loxone, Crestron, etc.):

- **Less lock-in**

  - Field gear is KNX/DALI/Modbus where possible.
  - Replacement parts and support are not tied to one brand.

- **More transparent**
  - I/O maps, runbooks, and diagrams come as part of the job.
  - No “dealer-only” configuration tools.

---

### 5.2 For me (the business)

- **Higher-margin projects**

  - I can charge for:
    - Design (architecture, controls, network).
    - Implementation/commissioning.
    - Documentation.
  - Not just “labour to run cables”.

- **Recurring revenue**

  - I can sell **support plans** (monthly/annual) including:
    - Remote monitoring.
    - Regular updates and backups.
    - Priority response.

- **Re-use and scale**

  - I can use the same Gray Logic stack across multiple sites.
  - Room, plant, and CCTV patterns can be reused.
  - Jobs get faster and more profitable as the patterns mature.

- **Strategic positioning**
  - I’m no longer “just an electrician”; I’m an **open-standards automation provider**.
  - That keeps the door open for collaborating with consultants, M&E firms, and OT/ICS-type clients.

---

## 6. Revenue Model

### 6.1 Project revenue

1. **Design & planning**

   - Site survey, requirements, network and controls design.
   - Fixed fee or day rate with clear deliverables (drawings, spec, I/O list).

2. **Implementation & commissioning**

   - On-site wiring (if I’m the spark) or supervision if others do the cabling.
   - Panel build, Gray Logic node installation.
   - openHAB/Node-RED configuration, testing, client sign-off.

3. **Documentation & handover**
   - Runbook, I/O maps, network diagrams, backup/restore procedure.
   - Training session for client / FM.

### 6.2 Recurring revenue

Example support tiers:

- **Core Support**

  - Remote access via VPN.
  - Quarterly updates and backup verification.
  - Email/remote support in working hours.

- **Enhanced Support**

  - Core support plus:
    - Basic monitoring (ping / service checks).
    - Faster SLA for remote changes.

- **Premium Support (for bigger sites)**
  - Monitoring with alerts.
  - Out-of-hours escalation.
  - Annual on-site preventative visit.

Even a few sites on support contracts gives:

- Predictable monthly income.
- Ongoing relationships with clients.
- More chances to upsell extensions and new modules.

---

## 7. Competitive Landscape

### 7.1 Proprietary smart home / automation platforms

**Examples:** Loxone, Crestron, Control4, Savant, etc.

- **Strengths**

  - Polished hardware and UI.
  - Strong dealer networks.
  - “One throat to choke” for clients.

- **Weaknesses (from my viewpoint)**
  - Vendor lock-in and licence models.
  - Proprietary tools and training requirements.
  - Harder for a non-certified engineer to support or extend.
  - Often more focused on AV/lifestyle than plant, energy, and infrastructure.

**Gray Logic position:**

> “Open, documented, infra-first system for people who care about standards and long-term ownership more than brand logos.”

### 7.2 DIY / enthusiast open-source setups

**Examples:** Home Assistant on a Pi, random Node-RED flows.

- **Strengths**

  - Cheap entry point.
  - Massive community and integration support.

- **Weaknesses**
  - Usually not treated as a product.
  - Weak or no documentation.
  - No proper network design, safety boundaries, or runbooks.
  - Hard to charge professional rates for “I put HA on a NUC”.

**Gray Logic position:**

> “Professionalised, documented, supportable open-source stack delivered as a productised service, not a hobby build.”

### 7.3 Traditional BMS / SCADA

**Examples:** Trend, Siemens, Schneider, Honeywell, etc.

- **Strengths**

  - Very mature and robust.
  - Deep industrial and commercial feature sets.
  - Established spec/consultant ecosystem.

- **Weaknesses**
  - Complex and expensive for small sites.
  - Overkill for a nice house + pool or a small mixed-use building.

**Gray Logic position:**

> “Mini-BMS philosophy for sites that are too small for full BMS but too serious for a random ‘smart home’ install.”

---

## 8. Risks & Mitigations

### 8.1 Technical risks

- **Risk:** openHAB / Node-RED / other components change or deprecate features.  
  **Mitigation:**

  - Stick to mature releases.
  - Maintain a tested “stack version” per site.
  - Use solid backups and a clear update policy.

- **Risk:** System becomes too complex to support on my own.  
  **Mitigation:**
  - Enforce KISS and clear module boundaries.
  - Document everything.
  - Reuse patterns instead of creating bespoke one-offs.

### 8.2 Market risks

- **Risk:** Clients default to known brands (Loxone, Control4).  
  **Mitigation:**

  - Target clients and architects who value openness and ownership.
  - Deliver 1–2 flagship Gray Logic sites and build a strong story around them.

- **Risk:** Price pressure from “guy who will just install smart bulbs + Alexa”.  
  **Mitigation:**
  - Be explicit that this is **not the same product**.
  - Emphasise safety boundaries, plant integration, long-term support, and documentation.

### 8.3 Personal/business capacity

- **Risk:** Too many custom one-offs, not enough standardisation.  
  **Mitigation:**
  - Keep refining patterns and templates (configs, flows, runbooks).
  - Say no to jobs that fight the architecture too hard.

---

## 9. Success Criteria & Go/No-Go

I’m not doing this just for fun – Gray Logic has to justify itself in my real life: time, money, and stress levels.

I’m building this alongside full-time electrical work and family life, so it’s a **slow-burn project**, not a “quit everything and raise VC” sprint. These criteria help me decide, over time, whether to keep pushing, change direction, or park it.

### 9.1 What “success” looks like for me

Rather than fixed dates, I’ll use **milestones**. Gray Logic is doing its job if, as the years go by, I can honestly say:

- **I have real deployments**

  - I’ve got at least **1–2 proper Gray Logic sites** (high-end home, leisure, or small commercial).
  - My own home/lab only counts if I treat it like a paying client: design, documentation, commissioning, and a support plan.

- **The projects make financial sense**

  - When I look at hours vs. income for Gray Logic jobs:
    - The effective day rate is **as good as or better than** good-quality electrical work.
    - I’m clearly being paid for **design + integration + documentation**, not just “fitting bits”.

- **There is some recurring income**

  - I have at least a couple of sites on **paid support/maintenance**, so there’s steady monthly/annual money coming in _because_ Gray Logic exists.
  - Even if it starts small, I can point to a real number per month in the accounts.

- **The stack has a stable v1**
  - I have a **known-good “v1” stack**:
    - I’d be comfortable supporting it for 5–10 years.
    - I can deploy it to another site with only minor tweaks.
  - I’ve nailed repeatable patterns for:
    - A “standard room”.
    - A typical bit of plant (boiler, pool kit, AHU).
    - Security/CCTV integration that isn’t a bodge.

If those boxes slowly tick up over time, Gray Logic is working for me, even if progress isn’t fast.

### 9.2 My review checkpoints

After each big milestone (lab build, first client site, major refactor, etc.) I’ll sit down and ask:

- **Technical sanity**

  - Is the stack **boring and stable enough** that I’d support it for 5–10 years?
  - Am I building on mature stuff (Linux, Docker, openHAB, Node-RED, KNX, etc.), or fighting my own cleverness and edge cases?

- **Economic sanity**

  - Am I **actually getting paid** for thinking, designing, documenting, and supporting?
  - If I divide total income by total hours, does it look competitive with, or better than, solid electrical work?

- **Life sanity**
  - Does Gray Logic make my work-life **better overall**, or just noisier and more stressful?
  - Am I proud of where it’s going, or quietly resenting it?

If the answers are mostly “yes” (even if slowly improving), that’s a good sign.  
If they drift towards “no” for a while, I need to change something (scope, pricing, target clients) rather than just grinding.

### 9.3 Sector & macro reality check (UK in £, with global context)

I also need to sanity-check Gray Logic against what’s happening in the **UK and global markets**, not just my gut.

#### UK smart home / small automation

Different analysts give different numbers, but the direction is consistent:

- One report estimates the **UK smart home market** at about **USD 8.43 billion in 2023**, rising to **USD 25 billion by 2030** (around 16–17% CAGR). With a rough 2025 exchange rate, that’s **~£6–6.5 billion today**, heading towards **~£18–19 billion** by 2030.
- Another puts the **UK smart home market** at **USD 7.7 billion in 2024**, projected to reach **USD 16.9 billion by 2033** (around 9% CAGR) – roughly **~£5.8 billion growing to ~£12.5–13 billion** in today’s money.

Even if I discount the most optimistic forecasts, the UK market for smart/connected homes is clearly growing in the **mid-single to high-teens % per year**, in a space measured in **billions of pounds**, not millions.

The **UK home improvement market** is also forecast to grow from around **USD 14.4 billion in 2024 to USD 21.6 billion by 2033** (roughly 4% CAGR), with smart/energy-efficient upgrades called out as a driver – roughly **£10–11 billion rising towards £16 billion** in equivalent terms.

For me, that means:

- UK homeowners and small commercial operators **are** spending real money on upgrading properties.
- Smart / energy-efficient / automated systems are part of that spend, not just a side fad.

#### Global smart home & building automation context

Globally:

- The **global smart home market** is estimated at around **USD ~125–130 billion in 2024**, with forecasts up to **USD ~500+ billion by 2030**, implying very strong double-digit growth.
- The **global building automation systems (BAS) market** is estimated at **around USD 200 billion in the mid-2020s**, with forecasts to **USD 340–350 billion by 2030** (roughly 10–12% CAGR).

I’m not trying to build a global platform; I’m carving out a tiny slice. But these numbers tell me:

- The **global tide is rising** for smart homes and building automation.
- As long as I stay in the lane of:

  - high-end homes,
  - pools/leisure,
  - small “serious” sites (plant + lighting + security),

  there should be space for an **open-standards, infra-first, UK-based offering** like Gray Logic.

As part of each review, I’ll explicitly ask:

- In my patch, are architects, builders, and clients **talking more or less** about this kind of thing?
- Does the UK data still show **growth in the high-end / smart / energy-aware parts of the market**, or has it flattened?
- Given those trends, does spending my limited time on Gray Logic still look like a **good bet** compared to just doing more standard electrical jobs?

If the macro picture is strong but I’m not landing projects, that’s a **positioning and sales problem** I might be able to fix.  
If the macro picture weakens _and_ Gray Logic isn’t paying its way, that’s a sign to be more ruthless.

### 9.4 Permission to pause or pivot

I’m allowed to stop or park this.

If, after a few real attempts (lab builds, a site or two, real quotes to real people), I find that:

- The stack is **OK technically**, but
- It isn’t paying me fairly for the effort, **and**
- It isn’t obviously moving me towards the income and lifestyle I want, **and**
- The UK/global trends don’t look especially favourable,

…then I give myself explicit permission to:

- Freeze Gray Logic at **“good enough for my own use and portfolio”**.
- Keep the GitHub repo public as a solid example of how I design and think.
- Shift focus back to better-paying electrical work or other opportunities (including OT/ICS or infra roles) that fit my situation better.

The goal isn’t to cling to Gray Logic out of sunk cost.  
The goal is to give it a **fair, informed shot** – and then make a clear decision based on my reality, not just hope.

---

## 10. Next Steps (Business Side)

1. **Document the product packages**

   - Gray Logic Core.
   - Lighting & Scenes.
   - Environment & Plant.
   - Security & CCTV.
   - Write one-page descriptions and rough budget bands for each.

2. **Pick one “hero” use-case**

   - For example: “Gray Logic for Pool & Leisure” or “Gray Logic for Cinema + Plant”.
   - Build and document it end-to-end (lab, then a real site).

3. **Refine the support offering**

   - Decide on 2–3 support tiers and what I’m genuinely comfortable committing to.

4. **Integrate messaging into my public sites**
   - Add a Gray Logic section to graylogic.uk / electrician.onl.
   - Keep technical detail in GitHub; keep the client pitch simple and outcome-focused.

This business case will evolve with each real project, but this version is enough to justify pushing forward and seeing whether Gray Logic can truly pay its way.
