# Gray Logic Stack – Working Draft v0.2

## 1. What Gray Logic and the Gray Logic Stack Are

**Gray Logic** is my company and trading name.

The **Gray Logic Stack** is a practical, field-first automation and infrastructure stack that I design, install, and support under the Gray Logic brand. It’s built by an electrician who runs real Linux/Docker systems in production.

The Gray Logic Stack is **not** “another smart home toy”. It is a **repeatable architecture** for:

- High-end homes and small estates
- Leisure / pool / spa sites
- Light commercial and mixed-use buildings

…where the client wants **joined-up control and visibility** of:

- Lighting and scenes
- Environment and plant (temperature, humidity, air quality, pools, DHW)
- Media / cinema behaviour
- Security, access, and CCTV health

…and where the system must still respect:

- Physical controls
- Regulations and life safety boundaries
- Reliability and graceful failure
- Long-term maintainability by competent people

The Gray Logic Stack is intended to be:

- **Installable and maintainable by a competent electrician / maintenance tech**, not just me
- **Documented and supportable** as a product, not just a one-off hobby build

---

## 2. Scope & Non-Goals

### 2.1 Scope

The Gray Logic Stack targets:

- Single high-end dwellings or small estates
- Pool / spa / leisure facilities (standalone or attached to a home or small hotel)
- Small commercial / mixed-use buildings with:
  - Lighting zones and scenes
  - HVAC or basic plant (boilers, cylinders, AHUs, pool plant)
  - CCTV and intruder/fire systems

It is designed to work well where:

- There is **real plant** (pumps, filters, boilers, ventilation)
- There is a desire for **modes** (home/away/night/holiday)
- There is value in **remote visibility and support** via VPN

### 2.2 Non-Goals (at least v0.x)

The Gray Logic Stack is **not**:

- A replacement for a full-fat industrial **BMS/SCADA** on large commercial or industrial plants
- A mass-market **cloud smart home platform** competing with Google/Amazon/Apple
- A generic “**code anything**” platform for arbitrary software projects
- A way to glue together hundreds of random consumer IoT devices with weak safety boundaries

The stack is **opinionated and tightly scoped**. It prefers:

- Fewer, better-understood integrations
- Open standards at the field layer (KNX, DALI, Modbus, dry contacts)
- Boring, stable infra over chasing every new toy

---

## 3. Design Principles

### 3.1 Hard Rules – Cannot Be Broken

These are **absolute**. If I have to break one, the project is not a Gray Logic Stack deployment.

1. **KISS (Keep It Simple, Sensible)**

   - Fewer moving parts beat “cleverness”.
   - Avoid fragile chains like: device → cloud → webhook → cloud → on-site.
   - Prefer: sensor → wire/bus → controller → clear logic.

2. **Physical controls always remain valid**

   - Wall switches, panic buttons, safety overrides, plant room controls must work even if:
     - The server is off
     - The LAN is down
     - The VPN / internet is down
   - Automation **observes** physical state and follows it, not the other way around.

3. **Life safety is independent**

   - Fire alarm systems, emergency lighting, emergency stop circuits:
     - Use their own certified hardware, wiring, and power supplies
     - May send signals into the Gray Logic Stack (e.g. a global “fire” relay),  
       but are never **controlled by** the Gray Logic Stack
   - No remote reset/silence of fire alarms through the stack.

4. **No cloud-only dependencies for basic operation**

   - The site must continue to function if:
     - The internet is down
     - A vendor cloud disappears
   - Cloud/VPS is an **enhancement** (remote monitoring, backups, dashboards), not a requirement.

5. **Linux + Docker at the core**

   - Core stack runs on:
     - A Linux host (Debian/Ubuntu-style)
     - Docker / Docker Compose
     - Traefik as the primary edge / reverse proxy

6. **Documented boundaries between safety layers**
   - Clear distinction between:
     - Life safety systems (fire, emergency stops)
     - Intruder / access control
     - Automation / comfort / convenience
   - “What can turn what off” is written down and reviewed.

### 3.2 Strong Defaults – Only Break With Justification

These are the default patterns; I only break them with a clear reason.

1. **Open-source first**

   - Prefer open-source components with sane licences (openHAB, Node-RED, Mosquitto, etc.)
   - Use proprietary kit where it adds clear value:
     - KNX/DALI actuators and keypads
     - NVRs and professional CCTV kit
     - Alarm panels certified to relevant standards

2. **Modular and replaceable**

   - Clear module boundaries:
     - Core infra (Traefik, monitoring, VPN)
     - Environment monitoring
     - Smart home / lighting
     - Media / cinema
     - Security / alarms / CCTV
   - Any module can be removed or replaced without collapsing the rest.

3. **On-site + remote pairing**

   - Each site has:
     - An on-site node (NUC / industrial PC / microserver)
     - Optional “Gray Logic remote” presence (VPS) for:
       - Off-site backups
       - Central dashboards
       - Remote admin via WireGuard

4. **Everything is documented**

   - Panel schedules and circuit lists
   - I/O maps and KNX group tables
   - Network diagrams and VLAN plans
   - A site runbook describing:
     - What lives where
     - How to restart things safely
     - Backup/restore procedure

5. **Respect standard electrical practices**
   - Use radials, contactors, and I/O in a way an electrician instantly understands
   - Control layers are **additive**, not magical replacements for good design

### 3.3 Preferred Patterns

- **Fieldbuses where appropriate**

  - KNX for buttons, relays, and room controllers
  - DALI for lighting where dimming/colour is needed
  - Modbus (RTU/TCP) for some plant and meters

- **VPN for remote access**

  - WireGuard as the default
  - No random port-forwards to cameras, NVRs, or control UIs

- **Dashboards for state, not everything**
  - Show “is it healthy / what mode are we in / what’s wrong”
  - Avoid turning dashboards into another fragile control dependency

---

## 4. Architecture Overview

### 4.1 Layers

The Gray Logic Stack is conceptually split into four layers:

1. **Field Layer (Physical World)**

   - Circuits, contactors, actuators
   - KNX/DALI/Modbus devices
   - Sensors (temperature, humidity, CO₂, PIRs, reed contacts, pressure, flow)
   - Safety systems (fire alarm, emergency lighting, E-stops)

2. **Controller Layer (Logic & State)**

   - openHAB:
     - Devices, items, scenes, schedules, modes
     - User-facing control where appropriate (web/app)
   - Node-RED:
     - Cross-system flows (e.g. “alarm set” → “energy save mode”)
     - Integrations with oddball APIs
   - Rules and automations live here.

3. **Infrastructure Layer (Platform)**

   - Linux host (Debian/Ubuntu)
   - Docker + Docker Compose
   - Traefik reverse proxy (HTTPS, routing, access control)
   - Monitoring & logging stack:
     - Container health
     - Host metrics
     - Optional time-series for key signals
   - WireGuard, backup jobs, etc.

4. **Remote / Cloud Layer (Optional)**
   - VPS (e.g. Hetzner) hosting:
     - Central dashboards
     - Off-site encrypted backups
     - WireGuard hub / remote admin
   - Used to enhance on-site node, not replace it.

### 4.2 Data & Control Flows (Typical)

- Field devices → KNX/DALI/Modbus → openHAB → (optional) Node-RED → Scenes / Modes / Commands
- Alarms / NVR → dry contact or integration → Stack → dashboard / notifications
- Remote admin → WireGuard → Traefik → specific UIs (openHAB, NVR, routers)

Control never crosses safety boundaries in ways that would violate the hard rules.

---

## 5. Core Technology Choices

### 5.1 Automation Brain – openHAB

openHAB is the **primary automation brain** for a Gray Logic site.

Responsibilities:

- Represent devices, groups, and scenes as **Items**
- Provide automation through Rules (time-based, event-based)
- Offer a structured UI (Main UI) for:
  - House modes
  - Basic room control
  - Status views

Why openHAB:

- Mature, stable, long-running project
- Good support for KNX, Modbus, MQTT, and lots of vendor bridges
- Suitable for a “boring, long-term” automation brain

### 5.2 Glue Logic – Node-RED

Node-RED is used as a **glue / power tool**:

- Bridge weird APIs or webhooks into/from the stack
- Implement flows that cross system boundaries:
  - “Alarm set to Away” → “Lower heating setpoints + enable perimeter lights”
  - “Pool plant fault + time window” → “Send Telegram and log to dashboard”

Rules of thumb:

- If logic is mostly about **devices and scenes**, it belongs in openHAB
- If logic is about **multiple systems and APIs**, it probably belongs in Node-RED

### 5.3 Messaging – MQTT (Where Appropriate)

MQTT is used to:

- Normalise messages from devices that speak MQTT natively
- Act as a loose coupling point between:
  - openHAB
  - Node-RED
  - Custom scripts/services

Not everything must use MQTT; it’s a tool, not a religion.

### 5.4 Infrastructure – Linux, Docker, Traefik

- Linux host (typically Debian or Ubuntu)
- Docker + Docker Compose:
  - openHAB container
  - Node-RED container
  - Mosquitto (if using MQTT)
  - Traefik
  - Monitoring/logging stack

Traefik responsibilities:

- Terminate HTTPS for on-site UIs
- Handle ACME (Let’s Encrypt) where the site is internet-reachable
- Provide SSO / auth where appropriate for admin UIs

### 5.5 Remote Access – WireGuard

WireGuard is the default for:

- Remote admin from my laptop/phone
- Linking site → VPS (backups, dashboards)
- Exposing NVR/Web-UIs to me without port forwards

---

## 6. Functional Domains / Modules

Each module has its own doc in `docs/modules/` for detail. This section describes the high-level shape.

### 6.1 Core – Traefik, Dashboard, Metrics

**Purpose:** Provide a single, consistent “front door” and health overview.

Components:

- Traefik reverse proxy
- Core dashboard (openHAB UI plus custom panels as needed)
- Monitoring stack (e.g. Prometheus-style + simple dashboards)

Key questions the core view must answer:

- “Is the site OK?”

  - Host up?
  - Core containers running?
  - VPN connected?
  - Alarm state?
  - CCTV / NVR healthy?

- “What mode are we in?”

  - Home / Away / Night / Holiday (or equivalent per site)

- “What’s wrong?”
  - Obvious faults (plant, communications, sensors offline) flagged clearly

### 6.2 Environment Monitoring

**Goal:** Make climate and energy use visible and actionable.

Inputs may include:

- Room and plant area temperature / humidity
- CO₂ and air quality (offices, cinemas, classrooms)
- Water temperatures (pools, DHW cylinders)
- Optional: differential pressure across filters, AHUs

Functions:

- Logging and trending (graph history)
- Alerts:
  - Over/under temperature
  - Persistent poor air quality
  - Plant faults / out-of-range conditions
- Driving behaviour:
  - Lower brightness / close blinds when unoccupied and bright
  - Boost ventilation when CO₂ is high
  - Adjust setpoints or modes based on occupancy and time

### 6.3 Lighting & Scenes

**Goal:** Intuitive, safe, efficient lighting without “smart bulb hell”.

Principles:

- **Physical-first**:

  - Standard wall switches, KNX keypads, or retractive pushes remain primary
  - Failure of automation must not leave someone in the dark if avoidable

- **Local control where possible**:
  - Room controllers or KNX actuators with local manual override
  - DALI gear for dimming / colour temperature where required

Typical behaviour:

- Scenes such as:
  - Cooking / Dining / Cinema / Night / Away
- Logic examples:
  - Manual press → override automation for N hours or until next occupancy change
  - Night paths: low-level lighting on routes to bathroom/kitchen in certain time windows
  - “Away” mode: lights off, some security presence patterns, etc.

### 6.4 Media / Cinema

**Goal:** Integrate AV with the wider behaviour of the house/site.

Inputs:

- “Cinema mode” from keypads or control app
- Playback state from AV receiver / HTPC (where practical)

Outputs:

- Dimming lights, closing blinds
- Enabling “quiet hours” where some alerts become visual only
- Optional pre-rolls/intros from local HLS / media hosted in the stack

Integration pattern:

- A “media controller” (e.g. HTPC / Pi) exposed behind Traefik
- openHAB / Node-RED reacting to media state where integrations allow

### 6.5 Security, Alarms & CCTV

**Goal:** Add visibility and convenience without undermining safety or compliance.

#### Hard Security Rule

Intruder alarms, fire alarms, and critical access control must still work if the Gray Logic node or LAN is down.

#### Intruder / KNX Security

Alarm panel:

- Owns all arming, zone logic, signalling
- Is the **primary system**

Gray Logic Stack:

- **Reads** states (armed, disarmed, alarm, fault, zones)
- Can send **request** pulses (arm away, arm stay, reset) where appropriate
- Any remote arming/disarming:
  - Must be explicit (no accidental one-tap)
  - Must be behind VPN + auth
  - Must be logged

KNX field integration:

- Use motions/contacts for:
  - Presence-aware lighting
  - “House empty” energy saving
- Panel outputs / relays into the stack for:
  - Alarm active flags
  - Trouble / tamper states

#### Fire / Life Safety

Fire system:

- Maintains its own circuits and power
- Is not remotely silenced/reset by the stack

Gray Logic Stack:

- May receive a global “fire” relay
- May:
  - Turn all relevant lighting on
  - Shed non-essential loads
- Logs the event for review

#### Access Control

Access control system:

- Manages credentials and time profiles
- Is the primary system for doors

Gray Logic Stack:

- Shows door states, faults, last events
- May offer a “request door release” function:
  - Requires strong logging and trust
  - Only where coherent with site security policy

#### CCTV / NVR

Topology:

- Cameras → PoE switch → NVR/VMS
- NVR on its own VLAN/subnet
- No direct internet exposure

Integration:

- RTSP / ONVIF into dashboards for:
  - Health tiles
  - Low-fps previews or snapshots
- Event feeds:
  - Motion / line crossing → lighting scenes, visitor alerts

Remote:

- Full feeds only over VPN
- Dashboard may show “CCTV healthy + last event” without exposing streams publicly

---

## 7. Delivery Model

### 7.1 On-Site Node

Typical hardware:

- Small Linux box (NUC, industrial PC, or microserver)
- SSD for OS + containers + logs

Responsibilities:

- Run Docker + Traefik + monitoring + site modules
- Store config, logs, and short-term metrics
- Maintain local resilience (keep running if internet is down)

### 7.2 Optional Remote Services

VPS (e.g. Hetzner-style) for:

- Off-site encrypted backups (rclone + GPG, etc.)
- Centralised dashboards (Grafana-style)
- Remote admin via WireGuard hub

Constraints:

- Remote is an enhancement, not a control dependency
- Site must be usable and safe with VPS offline

### 7.3 Networking & Segmentation

Default segmentation:

- **Control / HMI network** – Gray Logic node, KNX/IP gateway, openHAB, Node-RED
- **CCTV / NVR** – Cameras and NVR on separate VLAN/subnet
- **Guest / general LAN** – Client Wi-Fi, general devices

Remote access:

- WireGuard into:
  - Gray Logic node
  - Router / firewall
  - NVR (indirectly)
- Avoid direct public exposure of control ports

### 7.4 Documentation & Handover

Every Gray Logic Stack deployment should include:

- Electrical schematics and panel schedules
- KNX/DALI group tables and I/O maps
- Network diagram (VLANs, addressing, VPN)
- Site runbook covering:
  - How to restart services safely
  - Where backups go and how to restore
  - Contact details and support process

---

## 8. “Why Not Loxone / Crestron / Control4 / Just Use Home Assistant?”

This is about **positioning**, not trashing other platforms.

### 8.1 Why not Loxone / Crestron / Control4 etc.

These are strong systems with real advantages:

- Polished hardware and UX
- Established dealer networks
- Integrated ecosystems

But for what I’m trying to do with the Gray Logic Stack, they have downsides:

- **Vendor lock-in**

  - Hardware, licensing, and config tools tied to one vendor
  - Harder for other professionals to support later

- **Opaque to non-dealers**

  - Configuration tools and training often restricted to certified partners
  - Harder for a generic electrician or maintenance team to maintain long-term

- **Less flexible at the infra level**
  - The platform decisions (cloud, on-prem, containerisation) are mostly made for you
  - Harder to integrate with my existing infra patterns (VPS, VPN, backups)

I’m not trying to beat Loxone or Crestron at **their** game. I’m offering:

> An **open-standards, infra-first stack** that lives comfortably in a Linux/Docker world, is wired like a good electrical job, and can be documented and supported by more than one person over its lifetime.

### 8.2 Why not “just use Home Assistant”?

Home Assistant is excellent, with:

- Huge community
- Tons of integrations
- Great pace of development

But for Gray Logic Stack as a **product**:

- The pace of change and breaking config is higher than I’m comfortable with for long-term support
- The configuration style and patterns I’ve chosen (openHAB + Node-RED, documented infra) fit my “boring, supportable, slow-burn” approach better

I may still **interface** with sites that use Home Assistant, but the Gray Logic Stack itself:

- Standardises on **openHAB as the primary brain**
- Uses **Node-RED for glue**
- Keeps clear docs and update policies so I can support the same stack for years

---

## 9. Roadmap (High Level)

This is about turning the Gray Logic Stack from a nice spec into a real, repeatable product.

### 9.1 v0.1 – Architecture & Docs (done-ish)

- First write-up of:
  - What Gray Logic Stack is
  - Design rules
  - Module outlines
- Simple diagrams:
  - Field vs controller vs infra
  - Network segmentation

### 9.2 v0.2 – Spec & Business Case (this document)

- Clarify:
  - Gray Logic (company) vs Gray Logic Stack (product)
  - Technology choices (openHAB, Node-RED, Traefik, WireGuard)
  - Safety boundaries and “why not Loxone / Crestron / Home Assistant as the base”
- Add:
  - Business case document (`docs/business-case.md`)
  - Clear success criteria and go/no-go thinking

### 9.3 v0.3 – Core Stack Prototype

On a lab/test box:

- Docker Compose with at least:
  - Traefik
  - openHAB
  - Node-RED
  - MQTT (if used)
  - Basic monitoring
- A minimal “site”:
  - Simulated or real KNX/DALI/Modbus/MQTT devices
  - Example modes (Home/Away/Night)
  - Simple dashboard answering “is the site OK?”

### 9.4 v0.4 – Domain Demos

- **Environment + Lighting demo**
  - Real or simulated sensors
  - Scenes and physical override logic
- **Security & CCTV demo**
  - Tie in at least:
    - A real or simulated alarm output
    - A dummy CCTV feed or health indicator
  - Show a single view of “alarm state + last motion + CCTV health”

### 9.5 v0.5 – First Real Site

- Deploy to:
  - My own home, **or**
  - A small, controlled client site (e.g. pool/leisure)
- Treat it as a **productised job**:
  - Full design + documentation
  - Support agreement
  - Lessons learned fed back into the stack

### 9.6 v1.0 – Production-Ready Pattern

After at least one real site:

- Lock in:
  - A supported set of versions (openHAB, Node-RED, base images)
  - A standard repo structure and deployment playbook
- Only then call it “v1.0” for external clients.

---

## 10. Living Document

This spec is a **living document**:

- If reality (projects, clients, tech) shows a better way, the spec gets updated
- If something turns out too complex to support, it should be simplified or dropped
- The business case sits alongside this spec and helps decide:
  - Is Gray Logic Stack paying its way?
  - Does it still justify the effort?

The aim is not perfection on paper. The aim is:

> A **real, supportable stack** I can deploy to multiple sites, that respects safety and standards, fits my skills, and can actually earn money under the Gray Logic name.
