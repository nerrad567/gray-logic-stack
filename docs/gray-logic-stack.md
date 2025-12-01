# Gray Logic Stack – Working Draft v0.1

## 1. What Gray Logic Is

Gray Logic is a **practical, field-first automation and infrastructure stack** designed by an electrician who runs real Linux/Docker systems.

It’s not “another smart home toy” – it’s a **repeatable architecture** for:

- High-end homes and small estates
- Leisure / pool / spa sites
- Light commercial and mixed-use buildings

…that need lighting, environment, media, security and monitoring that actually respects:

- Physical controls
- Regulations
- Reliability
- Long-term maintainability

---

## 2. Core Goals

### 2.1 Real-world reliability

- If the server dies, the building must still be usable and safe.
- Life safety (fire, emergency lighting, emergency stops) is always independent.

### 2.2 Electrician-friendly

- Installable and maintainable by competent sparkies / maintenance techs.
- Clear panels, labels, documentation – not “magic” hidden black boxes.

### 2.3 Tech-forward but sane

- Linux, Docker, and open-source components where possible.
- Traefik reverse proxy, metrics, dashboards – but kept simple and documented.

### 2.4 Scales from “posh house” to “small site”

Same mental model for:

- A cinema room + plant room in a house.
- A pool / leisure site with plant, lighting, CCTV, and access bits.

---

## 3. Non-Goals (at least v0.x)

- Not trying to replace full-fat BMS/SCADA in big industrial plants.
- Not trying to be a consumer IoT cloud platform.
- Not providing a generic “code anything” platform; this is **opinionated and tightly scoped**.

---

## 4. Design Rules

### 4.1 Hard Rules – Cannot Be Broken

1. **KISS (Keep It Simple, Sensible)**

   - Fewer moving parts > “clever”.
   - No unnecessary indirection or fragile chains.

2. **Physical controls always remain valid**

   - Wall switches, panic buttons, plant room overrides must work even if:
     - The server is off.
     - The network is down.
   - Automation reads physical state and follows it, not the other way around.

3. **Life safety is independent**

   - Fire alarms, emergency lighting, and emergency stop functions:
     - Have their own certified hardware and wiring.
     - May send signals to Gray Logic, but are never controlled by it.

4. **No cloud-only dependencies**

   - Site must continue to function if:
     - The internet is down.
     - A vendor cloud is gone.
   - Cloud is an enhancement (remote monitoring, off-site backup), not a requirement.

5. **Linux + Docker at the core**

   - Core stack runs on:
     - Linux host (Debian/Ubuntu style).
     - Docker / Docker Compose.
     - Traefik as primary edge / reverse proxy.

### 4.2 Strong Rules – Default Unless Really Justified

1. **Open-source first**

   - Prefer open-source components with sane licenses.
   - Use proprietary only if they bring clear, necessary value (e.g. KNX actuators, NVR).

2. **Modular and replaceable**

   - Clear module boundaries:
     - Core infra
     - Environment
     - Smart home / lighting
     - Media / cinema
     - Security / alarms / CCTV
   - Any module can be removed without collapsing the rest.

3. **On-site + remote pairing**

   Each site has:

   - An on-site node (mini PC / NUC / industrial box).
   - The option to connect to a remote “Gray Logic cloud” (VPS, WireGuard, backups).

4. **Everything is documented**

   - Panel schedules, I/O maps, KNX group tables.
   - A “site runbook” lives with the stack.

### 4.3 Preferences / Patterns

- **Electrical design friendly**

  - Use radials, contactors, and I/O in ways that an electrician instantly understands.

- **KNX / DALI / Modbus where it makes sense**

  - Use standard fieldbuses for lighting, HVAC, and controls when budget allows.

- **VPN for remote**
  - Remote admin and CCTV via WireGuard, not random forwarded ports.

---

## 5. Functional Modules

### 5.1 Core – Traefik + Dashboard + Metrics

**Purpose:** Single, consistent “brain” and front door.

- **Traefik reverse proxy**

  - Terminate HTTPS.
  - Route to:
    - HMI / dashboards.
    - Media apps.
    - Admin UIs.
  - Use ACME (Let’s Encrypt) where appropriate (internet-reachable sites).

- **Core dashboard**

  - “Is the site OK?” view:
    - Key statuses: power, critical services, VPN, alarm state, CCTV health.
  - Simple UI for:
    - House modes (Home / Away / Night / Holiday).
    - Basic plant overview (for pool / AHU / boiler etc. where used).

- **Metrics & logging**
  - Docker-level monitoring (container uptime, restarts).
  - Host metrics (CPU, load, disk, memory).
  - Optional: capture key I/O states and trends (energy, temperature, faults).

### 5.2 Environment Monitoring

**Goal:** Make climate and energy use visible and actionable.

- **Inputs:**

  - Temperature / humidity sensors (rooms, plant areas).
  - CO₂ and air quality (offices, cinemas, classrooms).
  - Water temperature (pools, DHW cylinders).
  - Optional: differential pressure (filters, AHU).

- **Functions:**
  - Log + graph trends.
  - Trigger alerts:
    - Over/under temperature.
    - Poor air quality.
    - Plant faults (via dry contacts / Modbus / KNX).
  - Drive scenes:
    - Lower brightness / close blinds when rooms are empty and sun is bright.
    - Boost ventilation when CO₂ is high.

### 5.3 Smart Home / Lighting

**Goal:** Make lighting intuitive, safe and efficient, without “smart bulb hell.”

- **Physical-first**

  - Hard rule: standard wall switches, KNX keypads, or retractive presses remain primary.
  - Failure of automation must not leave someone in the dark if avoidable.

- **Control strategies:**

  - Room controllers / mini panels:
    - Per-room I/O (relay/dimmer) with local manual override.
  - KNX / DALI:
    - For high-end / refits, use bus-dimmers and KNX/DALI gear.
    - Scenes: Cooking, Dining, Cinema, Night, Away, etc.

- **Logic examples:**
  - Manual press → override automation for N hours or until next occupied state change.
  - Night paths: low-level lighting automatically on route to bathroom/kitchen, based on time + sensors.

### 5.4 Media / Cinema

**Goal:** Tie AV into the wider behaviour of the house/site.

- **Inputs:**

  - “Cinema mode” from remote / scene button.
  - Media playback state from receiver / HTPC (where accessible).

- **Outputs:**

  - Dimming lights.
  - Closing blinds.
  - Quiet hours – reduce some notifications/alerts to visual only.

- **Integration patterns:**
  - Role for a “Media controller” (e.g. Raspberry Pi / HTPC) exposed behind Traefik.
  - Optional HLS / video assets hosted on Gray Logic (for intros, status boards, etc.).

### 5.5 Security, Alarms & CCTV

**Goal:** Add visibility and convenience without undermining safety or compliance.

#### Hard Security Rule

Intruder alarms, fire alarms and critical access control must still work if the Gray Logic server or LAN is down.

#### Intruder / KNX Security

- **Alarm panel is primary**

  - All arming, zone logic and signalling is owned by the certified alarm panel.
  - Gray Logic:
    - Reads states (armed / disarmed / alarm / zones).
    - Sends request pulses (arm away / stay / reset) where appropriate.

- **Field integration**

  - KNX motion / contacts for:
    - Presence-aware lighting.
    - “House empty” energy saving.
  - Panel outputs / relays into Gray Logic for:
    - Alarm active flags.
    - Trouble / tamper states.

- **Remote considerations**
  - Any remote arm/disarm:
    - Must be explicit.
    - Must be behind VPN + auth.
    - Must be logged.

#### Fire / Life Safety

- **Read-only integration**
  - Global fire relay into Gray Logic:
    - Trigger lighting states (all on).
    - Trigger non-essential load shedding.
  - No remote reset / silence from Gray Logic.

#### Access Control

- **Access system is primary**
  - Door controllers manage credentials and time profiles.
  - Gray Logic:
    - Shows door status, fault states.
    - Optional “request door release” with strict logging.

#### CCTV / NVR

- **Physical topology**

  - Cameras → PoE switch → NVR (or VMS).
  - NVR isolated on its own subnet/VLAN.

- **Integration**

  - RTSP / ONVIF for:
    - Preview tiles in Gray Logic dashboards (low fps / snapshots).
  - Events from NVR → Gray Logic:
    - Motion / line crossing → lighting scenes, “visitor at gate” popups.

- **Remote**
  - Full feeds via VPN only.
  - Web UI can show “CCTV healthy + last event” without exposing streams publicly.

---

## 6. Delivery Model

### 6.1 On-Site Node

- Small Linux box (NUC / industrial PC / microserver).
- Runs:
  - Docker + Traefik + monitoring + site modules.
- Local storage for:
  - Config.
  - Logs.
  - Short-term metrics / history.

### 6.2 Optional Remote Services

- VPS (like existing Hetzner setup) for:
  - Off-site backups (encrypted, rclone).
  - Centralised dashboards.
  - Remote admin via WireGuard.

### 6.3 Networking

- Internal networks segmented:

  - Control/HMI.
  - CCTV/NVR.
  - Guest / general LAN.

- Remote access:
  - WireGuard only; no random open ports where avoidable.

---

## 7. Roadmap (Very High-Level)

1. **v0.1 – Architecture & docs**

   - This document + tweaks.
   - A couple of reference diagrams (physical + logical).

2. **v0.2 – Core stack prototype**

   - Docker Compose with:
     - Traefik.
     - Basic dashboard.
     - Metrics/logs.
   - Running on a test box.

3. **v0.3 – Environment + Lighting demo**

   - Simulated or real sensors (MQTT / Modbus / KNX gateway).
   - Basic scenes and overrides.

4. **v0.4 – Security & CCTV integration demo**

   - Tie in a demo alarm output + dummy CCTV feed.
   - Show “site OK / alarm / last motion” in one place.

5. **v0.5 – First real site**
   - Small, controlled deployment (e.g. your own home or part of a leisure site).
   - Treat it as a “productised job” with full documentation.
