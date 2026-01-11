# Gray Logic Stack - Architecture Documentation

**Version**: 0.5  
**Date**: 2026-01-11  
**Status**: Design Phase

This document defines the technical architecture for the Gray Logic Stack, including component design, network topology, access patterns, and security boundaries.

---

## 1. System Overview

The Gray Logic Stack is a three-tier building automation platform:

```
┌─────────────────────────────────────────────────────────┐
│ Field Layer (Open Standards)                             │
│ KNX • DALI • Modbus • MQTT • Dry Contacts                │
└────────────────┬────────────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────────────┐
│ Control Layer (On-Site Linux/Docker "Mini-NOC")          │
│ ┌─────────┬──────────┬──────────┬───────────┬─────────┐ │
│ │ Traefik │ openHAB  │ Node-RED │ Mosquitto │ Monitor │ │
│ └─────────┴──────────┴──────────┴───────────┴─────────┘ │
└────────────────┬────────────────────────────────────────┘
                 │ WireGuard VPN (optional)
┌────────────────▼────────────────────────────────────────┐
│ Remote Layer (Optional VPS)                              │
│ Long-term data • Dashboards • VPN Hub • User Auth       │
└─────────────────────────────────────────────────────────┘
```

**Design Principles:**
- **Offline-First**: 99%+ functionality without internet
- **Open Standards**: No vendor lock-in at field layer
- **Safety Independence**: Life safety systems remain autonomous
- **Documented Boundaries**: Clear separation between critical and convenience features

---

## 2. Component Architecture

### 2.1 Control Layer (On-Site)

All services run in Docker containers via `docker-compose.yml`:

#### Traefik
- **Role**: Reverse proxy, HTTPS termination, routing
- **Image**: `traefik:2.11` (pinned)
- **Ports**: 80 (HTTP), 443 (HTTPS), 8080 (dashboard)
- **Responsibilities**:
  - Route requests to openHAB, Node-RED, NVR, custom APIs
  - Terminate TLS (Let's Encrypt via ACME or self-signed local)
  - Session validation (JWT tokens)
  - Access logging for audit

#### openHAB
- **Role**: Main automation brain
- **Image**: `openhab/openhab:4.1.0` (pinned)
- **Port**: 8080
- **Responsibilities**:
  - Device bindings (KNX, DALI, Modbus, MQTT)
  - Items, things, rules, scenes
  - Mode management (Home/Away/Night/Holiday)
  - Persistence (local time-series)
  - UI (Main UI, HABPanel)

**Logic Boundary**: Device-centric automation (scenes, schedules, individual device rules)

#### Node-RED
- **Role**: Cross-system glue logic, PHM flows
- **Image**: `nodered/node-red:3.1.0` (pinned)
- **Port**: 1880
- **Responsibilities**:
  - Multi-system flows (e.g., "alarm armed → heating + lights")
  - Predictive Health Monitoring (PHM) logic
  - Optional external data ingest and normalisation (weather nowcast, mesh comms, notifications)
    - Supports local ingest pipelines (e.g. satellite receiver/decoder) and optional internet enrichment
    - Publishes internal “weather products” plus freshness/staleness signals to MQTT and/or openHAB Items
    - Publishes internal “mesh comms products” (health/telemetry) plus freshness/staleness signals to MQTT and/or openHAB Items
  - Data transformation and routing

**Logic Boundary**: Cross-system coordination, complex logic requiring multiple inputs


#### Mosquitto
- **Role**: MQTT message broker (optional loose coupling)
- **Image**: `eclipse-mosquitto:2.0.18` (pinned)
- **Ports**: 1883 (MQTT), 9001 (WebSocket)
- **Responsibilities**:
  - Pub/sub messaging between openHAB, Node-RED, IoT devices
  - Retain important state messages
  - Not mandatory—used where it adds value

#### Prometheus (Optional for v0.5)
- **Role**: Metrics collection and alerting
- **Image**: `prom/prometheus:v2.48.0` (pinned)
- **Port**: 9090
- **Responsibilities**:
  - Container health metrics
  - PHM sensor data (pump current, temps, run hours)
  - Local alerting (high temps, deviations)

### 2.2 Remote Layer (VPS)

**Status**: Design phase, not implemented in v0.5

**Components:**
- **WireGuard Hub**: Central VPN endpoint (port 51820)
- **User Auth API**: Credential management, session tokens
- **Config Store**: Encrypted per-user WireGuard configs
- **Relay API**: Read-only status endpoint (future)
- **Central Logging**: Audit trail for remote actions
- **Long-term Storage**: Multi-year trend data for PHM

---

## 3. Network Topology

### 3.1 On-Site Networks

```
┌─────────────────────────────────────────────────────────┐
│ Physical Network (Single NUC/Server)                     │
│                                                           │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐    │
│  │ Control LAN │  │  CCTV VLAN   │  │  Guest VLAN │    │
│  │ 192.168.1.x │  │ 192.168.2.x  │  │ 192.168.99.x│    │
│  │             │  │              │  │             │    │
│  │ • Traefik   │  │ • NVR        │  │ • Guest WiFi│    │
│  │ • openHAB   │  │ • Cameras    │  │             │    │
│  │ • Node-RED  │  │              │  │             │    │
│  │ • KNX/DALI  │  │              │  │             │    │
│  └─────────────┘  └──────────────┘  └─────────────┘    │
│         │                │                  │            │
│         └────────────────┴──────────────────┘            │
│                          │                               │
│                   ┌──────▼──────┐                        │
│                   │   Router    │                        │
│                   │  Firewall   │                        │
│                   └──────┬──────┘                        │
│                          │                               │
└──────────────────────────┼───────────────────────────────┘
                           │
                      Internet
                           │
                   ┌───────▼────────┐
                   │  VPS (WireGuard│
                   │      Hub)       │
                   └────────────────┘
```

**Network Segmentation:**
- **Control LAN** (`192.168.1.0/24`): Core stack, KNX/DALI gateways, trusted clients
- **CCTV VLAN** (`192.168.2.0/24`): NVR and cameras (isolated, no internet)
- **Guest VLAN** (`192.168.99.0/24`): Guest devices (no access to control/CCTV)
- **IoT/Consumer Overlay** (`192.168.50.0/24`): Hue, smart plugs (isolated, conditional access)

**Firewall Rules:**
- Control LAN → All local networks (for orchestration)
- CCTV VLAN → Control LAN port 443 only (API health checks)
- Guest VLAN → Internet only (blocked from other VLANs)
- IoT Overlay → Control LAN MQTT port 1883 only (if integrated)

### 3.2 Docker Networks

```yaml
networks:
  control_lan:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.1.0/24
  
  monitoring:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.2.0/24
```

**Service Assignments:**
- `control_lan`: Traefik, openHAB, Node-RED, Mosquitto
- `monitoring`: Prometheus, Grafana (if used)
- Services can be on multiple networks where needed

### 3.3 WireGuard VPN Topology

**Hub-and-Spoke Model:**

```
Site A (10.100.1.0/24) ──┐
                          │
Site B (10.100.2.0/24) ──┼───► VPS Hub (10.100.0.1) ◄──── User Desktop Utility
                          │              ▲                        │
Site C (10.100.3.0/24) ──┘              │                        │
                                         │                        │
                              Persistent Tunnel         Dynamic Connection
                                (51821-51823)                  (51820)
```

**IP Allocation:**
- VPS Hub: `10.100.0.1/16` (can route to all sites)
- Site A: `10.100.1.0/24`
- Site B: `10.100.2.0/24`
- Site C: `10.100.3.0/24`
- User Pool: `10.100.50.0/24` (dynamic allocation)

**WireGuard Port Strategy:**
- VPS Hub: **51820** (standard, easy to remember)
- Site A tunnel: **51821** (pattern: 51820 + site_id)
- Site B tunnel: **51822**
- Site C tunnel: **51823**

**Why Hub-and-Spoke:**
✅ Each site maintains single persistent tunnel to VPS  
✅ Users connect to VPS, VPS routes to authorized sites  
✅ Simpler firewall rules (sites only whitelist VPS IP)  
✅ Central logging at VPS  
✅ Multi-site users connect once, access all sites  
✅ No complex site-to-site mesh  

**Site WireGuard Config Example:**
```ini
[Interface]
PrivateKey = <site_private_key>
Address = 10.100.1.1/24
ListenPort = 51821

[Peer]
PublicKey = <vps_public_key>
Endpoint = vpn.graylogic.cloud:51820
AllowedIPs = 10.100.0.0/16, 10.100.50.0/24  # VPS and user pool
PersistentKeepalive = 25
```

**User WireGuard Config Example (Split-Tunnel):**
```ini
[Interface]
PrivateKey = <user_private_key>
Address = 10.100.50.2/32
DNS = 10.100.1.1  # Route DNS to on-site for .local resolution

[Peer]
PublicKey = <vps_public_key>
Endpoint = vpn.graylogic.cloud:51820
AllowedIPs = 10.100.0.0/16  # ONLY Gray Logic networks (not 0.0.0.0/0)
PersistentKeepalive = 25
MTU = 1420  # Prevent fragmentation
```

**Split-Tunnel Rationale:**
- Regular internet traffic bypasses VPN (faster, lower bandwidth)
- Only Gray Logic site traffic goes through tunnel
- Better battery life on mobile devices
- User's ISP doesn't see encrypted home automation traffic
- VPN drop doesn't kill entire internet connection

---

## 4. Access Patterns & Security

### 4.1 Connection Tiers

**Tier 1: Local LAN Access (Full Control)**
- **Detection**: mDNS broadcast from NUC (`_graylogic-<siteslug>._tcp.local`)
- **Connection**: Direct HTTPS to on-site Traefik (`https://192.168.1.10`)
- **Authentication**: Required (session-based, stored in OS keyring)
- **Capabilities**: Full read/write access to all services
- **Offline**: Fully functional (no internet needed)

**Tier 2: VPN Access (Full Control, Remote)**
- **Detection**: Local network scan fails → desktop utility establishes WireGuard tunnel
- **Connection**: 
  1. Desktop utility connects to VPS hub (51820)
  2. VPS routes to site via persistent site tunnel (5182x)
  3. User accesses site via `10.100.x.x` addresses
- **Authentication**: Required (same as local)
- **Capabilities**: Full read/write access (identical to Tier 1)
- **Offline**: VPS or site offline → fails gracefully

**Tier 3: Relay Access (Read-Only, Future)**
- **Detection**: User chooses "Quick View" without VPN
- **Connection**: HTTPS to VPS relay API (`https://site-name.graylogic.cloud`)
- **Authentication**: Basic auth or API key
- **Capabilities**:
  - ✅ Dashboard (modes, temps, status)
  - ✅ (Optional) Camera snapshots (low-res, explicitly enabled per site)
  - ✅ Alarm state (read-only)
  - ❌ Control actions (requires VPN)
  - ❌ Video feeds (requires VPN)
  - ❌ Admin interfaces (local/VPN only)
- **Offline**: Shows "Site Offline" if site-to-VPS tunnel down

**Tier 4: Offline/Failsafe**
- **Scenario**: Internet down, VPN unreachable
- **Behavior**:
  - Local LAN access (Tier 1) continues normally
  - Remote users cannot connect (expected)
  - On-site operations unaffected (99%+ uptime goal)
  - Services continue: lighting, plant, modes, scenes, PHM

### 4.2 Security Boundaries

**Remote Control Philosophy:**

| Feature                  | Local | VPN | Relay | Rationale                          |
|--------------------------|-------|-----|-------|------------------------------------|
| Lighting control         | ✅    | ✅  | ❌    | Safe, logged                       |
| Heating setpoints        | ✅    | ✅  | ❌    | Rate-limited, logged               |
| Mode changes             | ✅    | ✅  | ❌    | Safe (Home/Away/Night)             |
| Camera feeds (live)      | ✅    | ✅  | ❌    | Bandwidth, privacy                 |
| Camera snapshots         | ✅    | ✅  | ⚪    | Optional; disabled by default      |
| Alarm arming/disarming   | ✅    | ⚠️  | ❌    | Requires PIN + explicit confirm    |
| Plant control (pool/DHW) | ✅    | ⚠️  | ❌    | Confirm dialog, safety checks      |
| Node-RED editor          | ✅    | ✅  | ❌    | Admin only, never relay            |
| openHAB admin            | ✅    | ✅  | ❌    | Admin only, never relay            |
| Emergency stops          | 🔴    | 🔴  | 🔴    | **Physical only** (never remote)   |
| Fire system              | 🔴    | 🔴  | 🔴    | **Independent** (stack observes)   |

**Legend:**
- ✅ Allowed
- ⚠️ Allowed with confirmation
- ❌ Blocked
- ⚪ Allowed if explicitly enabled
- 🔴 Never (safety-critical, physically independent)

### 4.2.1 AI Analytics Data Flow (Optional)

AI-assisted features (if commissioned) are treated as **premium bonuses**: they may summarise and explain PHM/trend signals, but they are never required for control.

**Default telemetry posture (least sensitive):**
- Export aggregated health metrics and PHM events only.
- Log a minimal audit trail of remote actions.

**Never export off-site by default:**
- CCTV media (video/audio/recordings/snapshots)
- Occupancy/presence timelines
- Detailed security timelines (zone-by-zone alarm history, door event logs)
- Secrets (passwords, API tokens, keys, WireGuard configs)
- Raw network identifiers (MAC/IP client lists, device fingerprint scans)

Opt-in exceptions (explicit per site):
- Low-res, low-frequency camera snapshots for status-only “quick view”
- Time-boxed diagnostic logging during a support window

### 4.3 Audit Logging

All remote actions logged to:

**On-Site:**
```
/var/log/graylogic/audit.log
```

**VPS (Central):**
```
Database: audit_events table
Fields: timestamp, user_id, site_id, action, resource, result, ip_address, vpn_session
```

**Logged Events:**
- User authentication (success/failure)
- VPN connections (start/end, duration)
- Mode changes
- Alarm arming/disarming
- Plant control commands
- Admin interface access
- Failed authorization attempts

**Retention:**
- On-site: 90 days (rolling)
- VPS: 2 years (compliance/forensics)

---

## 5. Desktop Utility Architecture

### 5.1 Purpose

The **Gray Logic Desktop Utility** solves the "WireGuard in browser" impossibility by acting as a local bridge between the web app and remote sites.

**Key Insight**: Web browsers cannot establish kernel-level VPN tunnels. The utility runs as a native application with necessary privileges, exposing a local HTTP API that the web app consumes.

### 5.2 Component Diagram

```
┌─────────────────────────────────────────────────────────┐
│ Web Browser (User Interface)                             │
│ http://localhost:8888                                    │
│                                                           │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Gray Logic Web App (React/Vue/Svelte)                │ │
│ │ • Dashboard                                          │ │
│ │ • Site Switcher                                      │ │
│ │ • Control Interfaces (Lights, Modes, Plant)          │ │
│ └───────────┬─────────────────────────────────────────┘ │
└─────────────┼─────────────────────────────────────────────┘
              │ HTTP/WebSocket (localhost:8888)
              │
┌─────────────▼─────────────────────────────────────────────┐
│ Gray Logic Desktop Utility (Go Binary)                     │
│                                                             │
│ ┌───────────────────┐  ┌──────────────────────────────┐   │
│ │ HTTP Proxy Server │  │ WireGuard Manager            │   │
│ │ • Serve web app   │  │ • Start/stop interfaces      │   │
│ │ • API endpoints   │  │ • Config encryption          │   │
│ │ • WebSocket       │  │ • Auto-reconnect             │   │
│ └───────────────────┘  └──────────────────────────────┘   │
│                                                             │
│ ┌───────────────────┐  ┌──────────────────────────────┐   │
│ │ mDNS Discovery    │  │ System Tray UI               │   │
│ │ • Scan for sites  │  │ • Status indicator           │   │
│ │ • Auto-switch     │  │ • Site menu                  │   │
│ │ • Network monitor │  │ • "Open Dashboard" action    │   │
│ └───────────────────┘  └──────────────────────────────┘   │
│                                                             │
│ ┌─────────────────────────────────────────────────────┐   │
│ │ Site Connection Manager                              │   │
│ │ • Connection state machine                           │   │
│ │ • Multi-site handling                                │   │
│ │ • Credential storage (OS keyring)                    │   │
│ └─────────────────────────────────────────────────────┘   │
└─────────────────┬───────────────────────────────────────────┘
                  │
    ┌─────────────┴─────────────┐
    │                           │
    ▼                           ▼
Local Site                 VPN Tunnel
(Direct)                   (via WireGuard)
```

### 5.3 API Specification

**Served by Desktop Utility on `http://localhost:8888`:**

#### Static Files
```http
GET  /                      # Serve web app (index.html)
GET  /assets/*              # CSS, JS, images
```

#### Status & Discovery
```http
GET  /api/status
Response: {
  "connected": true,
  "site": {
    "id": "a7b3c9d2-1234-5678",
    "name": "Oak Street Residence",
    "mode": "local" | "vpn" | "disconnected"
  },
  "vpnActive": false,
  "localSites": ["a7b3c9d2-1234-5678"]  # Detected via mDNS
}
```

#### Site Management
```http
GET  /api/sites
Response: {
  "sites": [
    {
      "id": "a7b3c9d2-1234-5678",
      "name": "Oak Street Residence",
      "slug": "123oak",
      "available": true,
      "local": true
    },
    {
      "id": "b8c4d3e5-2345-6789",
      "name": "Pool House",
      "slug": "poolhouse",
      "available": false,
      "local": false
    }
  ]
}

POST /api/sites/connect
Request: {"siteId": "a7b3c9d2-1234-5678"}
Response: {"status": "connecting"}

POST /api/sites/disconnect
Response: {"status": "disconnected"}
```

#### Authentication
```http
POST /api/auth/login
Request: {"username": "darren", "password": "***"}
Response: {
  "success": true,
  "token": "eyJhbGc...",
  "user": {"id": "user123", "name": "Darren"}
}

POST /api/auth/logout
Response: {"success": true}
```

#### Proxy (Forwards to Active Site)
```http
GET  /api/proxy/openhab/rest/items
→ Proxied to: https://10.100.1.10/openhab/rest/items (via VPN)
   or https://192.168.1.10/openhab/rest/items (local)

POST /api/proxy/openhab/rest/items/Kitchen_Light
Request: "ON"
→ Proxied to active site

GET  /api/proxy/nodered/flows
→ Proxied to Node-RED API
```

#### WebSocket (Real-Time Updates)
```http
WS   /api/ws
Messages:
  → {"type": "connection_state", "status": "connecting", "site": "oak-street"}
  → {"type": "connection_state", "status": "connected", "mode": "vpn"}
  → {"type": "local_site_detected", "siteId": "a7b3c9d2-1234-5678"}
  → {"type": "vpn_tunnel_up", "site": "oak-street"}
  → {"type": "vpn_tunnel_down", "reason": "timeout"}
```

### 5.4 WireGuard Integration

**Responsibilities:**

1. **Config Storage**
   - Per-site encrypted WireGuard configs
   - Stored in OS-specific directories:
     - Linux: `~/.config/graylogic/sites/<site-id>.wg.enc`
     - Windows: `%APPDATA%\GrayLogic\sites\<site-id>.wg.enc`
   - Encrypted with AES-256-GCM using user password-derived key

2. **Tunnel Lifecycle**
   ```
   User clicks "Connect to Oak Street"
     ↓
   Check mDNS for local site
     ↓ (not found)
   Load encrypted WireGuard config
     ↓
   Decrypt with user's key (from OS keyring)
     ↓
   Create WireGuard interface (wg0)
     ↓
   Bring interface up
     ↓
   Verify connectivity (ping 10.100.1.1)
     ↓
   Notify web app via WebSocket
   ```

3. **Auto-Reconnect**
   ```
   Monitor tunnel state (every 10s)
     ↓
   Ping test fails?
     ↓
   Wait 5s → retry
     ↓
   Still failing?
     ↓
   Exponential backoff: 10s, 20s, 40s, 60s (max)
     ↓
   After 5 failures → notify user, downgrade to "disconnected"
   ```

4. **Platform Differences**

   **Linux:**
   - Use `wg-quick` or `wgctrl` Go library
   - Requires `NET_ADMIN` capability:
     ```bash
     sudo setcap cap_net_admin+ep /usr/bin/graylogic-desktop
     ```
   - Or sudoers entry for `wg` commands

   **Windows:**
   - Use WireGuardNT driver (official implementation)
   - Requires admin privileges (installer handles this)
   - Alternative: Utilize existing WireGuard installation

### 5.5 mDNS Discovery

**Broadcast (On-Site NUC):**
```bash
# Avahi service file: /etc/avahi/services/graylogic.service
<?xml version="1.0" standalone='no'?>
<!DOCTYPE service-group SYSTEM "avahi-service.dtd">
<service-group>
  <name replace-wildcards="yes">Gray Logic - Oak Street</name>
  <service>
    <type>_graylogic-123oak._tcp</type>
    <port>443</port>
    <txt-record>siteid=a7b3c9d2-1234-5678</txt-record>
    <txt-record>version=0.5.0</txt-record>
    <txt-record>api=https://192.168.1.10</txt-record>
    <txt-record>name=Oak Street Residence</txt-record>
  </service>
</service-group>
```

**Discovery (Desktop Utility):**
```go
// Scan every 30s or on network change
func (d *Discovery) Scan() {
    entries, _ := mdns.Lookup("_graylogic-*._tcp", "")
    for _, entry := range entries {
        siteId := entry.InfoFields["siteid"]
        
        // Check if user authorized for this site
        if d.sites.IsAuthorized(siteId) {
            d.notifyLocalSite(siteId, entry.AddrV4)
        }
    }
}
```

**Auto-Switch Logic:**
```
Local site detected (mDNS)
  ↓
Currently connected via VPN to same site?
  ↓ YES
Switch to direct local connection
Disconnect VPN tunnel
Notify web app: {"mode": "local"}
  ↓ NO
Add to "available sites" menu
```

### 5.6 Multi-Site Handling

**QR Code Onboarding:**

1. **Site Installation Complete**
   - Installer generates QR code via admin panel
   - QR contains JWT with:
     ```json
     {
       "siteId": "a7b3c9d2-1234-5678",
       "siteName": "Oak Street Residence",
       "siteSlug": "123oak",
       "vpsEndpoint": "vpn.graylogic.cloud:51820",
       "setupToken": "eyJhbGc...",  // One-time use
       "expiresAt": "2026-01-12T00:00:00Z"
     }
     ```

2. **User Scans QR (Desktop Utility)**
   ```
   Scan QR code
     ↓
   Parse JWT
     ↓
   Validate signature (against VPS public key)
     ↓
   Prompt: "Add Oak Street Residence?"
     ↓
   User confirms, sets password
     ↓
   POST to VPS: /api/sites/provision
   {
     "setupToken": "eyJhbGc...",
     "userId": "user123",
     "password": "hashed_password"
   }
     ↓
   VPS generates WireGuard config for user
   Encrypts with password-derived key
   Returns encrypted config
     ↓
   Utility saves config locally
   Site added to user's account
   ```

3. **Site Switching**
   ```
   System Tray Menu:
   ┌──────────────────────────────┐
   │ 🟢 Oak Street (Connected)    │ ← Active, local
   │ ⚪ Pool House (Available)    │ ← On same network
   │ 🔴 Beach House (Offline)     │ ← VPS reports site down
   │ ────────────────────────────│
   │ Add Site (Scan QR)...        │
   │ Settings                     │
   │ Quit                         │
   └──────────────────────────────┘
   ```

**State Management:**
- Only one VPN tunnel active at a time
- Multiple local sites can be detected simultaneously
- Active site persisted to config (auto-reconnect on utility restart)
- Switching sites disconnects current tunnel, establishes new one

---

## 6. Configuration Management

### 6.1 On-Site Configuration

**Directory Structure:**
```
code/stack/
├── docker-compose.yml              # Pinned image versions
├── .env                            # Site-specific vars (site_id, vps_endpoint)
├── traefik/
│   ├── traefik.yml                 # Static config
│   └── dynamic/
│       └── routes.yml              # Service routing
└── volumes/                        # Bind mounts (backup target)
    ├── openhab/
    │   ├── conf/                   # Things, items, rules, sitemaps
    │   └── userdata/               # Persistence, logs, cache
    ├── nodered/
    │   └── data/                   # Flows, credentials, settings
    ├── mosquitto/
    │   ├── config/                 # mosquitto.conf
    │   └── data/                   # Retained messages, logs
    ├── traefik/
    │   └── acme/                   # Let's Encrypt certs
    └── prometheus/
        └── config/                 # prometheus.yml, alerts
```

**Version Control:**
- Each site has Git repository: `graylogic-site-<slug>`
- Tracked files:
  - `docker-compose.yml`
  - `traefik/*`
  - `volumes/openhab/conf/*`
  - `volumes/nodered/data/flows.json`
  - `volumes/mosquitto/config/*`
- Ignored: `volumes/*/logs`, `volumes/*/cache`, credentials

**Site-Specific Variables (`.env`):**
```bash
SITE_ID=a7b3c9d2-1234-5678
SITE_NAME=Oak Street Residence
SITE_SLUG=123oak
VPS_ENDPOINT=vpn.graylogic.cloud:51820
WG_LISTEN_PORT=51821
INTERNAL_NETWORK=10.100.1.0/24
TZ=Europe/London
```

### 6.2 Desktop Utility Configuration

**Linux:**
```
~/.config/graylogic/
├── config.json
├── sites/
│   ├── a7b3c9d2-1234-5678.json
│   └── a7b3c9d2-1234-5678.wg.enc
└── session.token
```

**Windows:**
```
%APPDATA%\GrayLogic\
├── config.json
├── sites\
│   ├── a7b3c9d2-1234-5678.json
│   └── a7b3c9d2-1234-5678.wg.enc
└── session.token
```

**config.json:**
```json
{
  "listenPort": 8888,
  "logLevel": "info",
  "autoStart": true,
  "activeSite": "a7b3c9d2-1234-5678",
  "preferences": {
    "theme": "dark",
    "showNotifications": true
  }
}
```

**Site Metadata (a7b3c9d2-1234-5678.json):**
```json
{
  "id": "a7b3c9d2-1234-5678",
  "name": "Oak Street Residence",
  "slug": "123oak",
  "vpsEndpoint": "vpn.graylogic.cloud:51820",
  "createdAt": "2026-01-11T10:00:00Z",
  "lastConnected": "2026-01-11T14:30:00Z"
}
```

**Encrypted WireGuard Config (a7b3c9d2-1234-5678.wg.enc):**
```
Encrypted binary blob (AES-256-GCM)
Plaintext format (after decryption):
[Interface]
PrivateKey = <user_private_key>
Address = 10.100.50.2/32
DNS = 10.100.1.1
[Peer]
PublicKey = <vps_public_key>
Endpoint = vpn.graylogic.cloud:51820
AllowedIPs = 10.100.0.0/16
PersistentKeepalive = 25
```

**Encryption Method:**
```
User password → PBKDF2(password, salt, 100k iterations) → 256-bit key
Plaintext config → AES-256-GCM(config, key, random nonce) → Ciphertext
Store: nonce || ciphertext || tag
Decrypt: Extract nonce, decrypt with key, verify tag
```

**OS Keyring (Master Key Storage):**
- Linux: `libsecret` (gnome-keyring, kwallet)
  ```go
  secret.Password{
      Schema: "com.graylogic.desktop",
      Label: "Gray Logic Master Key",
  }.Store("masterkey", derivedKey)
  ```
- Windows: `DPAPI` (Data Protection API)
  ```go
  encryptedKey := dpapi.Encrypt(derivedKey)
  registry.WriteKey(encryptedKey)
  ```

---

## 7. Backup & Disaster Recovery

### 7.1 Backup Strategy

**Automated Daily Backup:**
```bash
# /etc/cron.daily/graylogic-backup
#!/bin/bash
/opt/graylogic/scripts/backup.sh >> /var/log/graylogic/backup.log 2>&1
```

**Backup Contents:**
1. `docker-compose.yml` + `.env`
2. `traefik/` configs
3. `volumes/openhab/conf/` (items, things, rules)
4. `volumes/openhab/userdata/` (persistence, jsondb)
5. `volumes/nodered/data/` (flows, credentials)
6. `volumes/mosquitto/config/`
7. `volumes/traefik/acme/` (certificates)

**Excluded:**
- Logs (`*.log`)
- Caches (`cache/*`, `tmp/*`)
- Large time-series data (backed up separately if needed)

**Backup Destinations:**
- **On-Site**: `/var/backups/graylogic/` (last 30 days)
- **Off-Site** (via VPN): VPS rsync daily → `s3://graylogic-backups/<site-id>/`
- **Retention**:
  - On-site: 30 days rolling
  - VPS: 90 days
  - S3: 2 years (lifecycle to Glacier)

### 7.2 Restore Procedure

**Scenario: Hardware Failure / Rebuild**

1. **Prepare New Host**
   ```bash
   # Install base OS (Ubuntu 24.04 LTS recommended)
   sudo apt update && sudo apt upgrade -y
   
   # Install Docker + Docker Compose
   curl -fsSL https://get.docker.com | sh
   sudo usermod -aG docker $USER
   
   # Install WireGuard
   sudo apt install wireguard
   ```

2. **Retrieve Backup**
   ```bash
   # From USB/local backup
   tar -xzf /mnt/usb/graylogic-backup-20260111.tar.gz -C /opt/graylogic/
   
   # Or from VPS (requires temporary WireGuard connection)
   scp vps:/backups/site-a7b3c9d2/latest.tar.gz /tmp/
   tar -xzf /tmp/latest.tar.gz -C /opt/graylogic/
   ```

3. **Restore Volumes**
   ```bash
   cd /opt/graylogic/stack
   
   # Restore creates volumes and sets permissions
   ../scripts/restore.sh /tmp/graylogic-backup-20260111.tar.gz
   ```

4. **Verify Configuration**
   ```bash
   # Check site-specific .env
   cat .env
   
   # Verify WireGuard keys (site tunnel to VPS)
   sudo wg show
   ```

5. **Start Stack**
   ```bash
   docker compose up -d
   
   # Wait for services to start
   sleep 30
   
   # Check health
   docker compose ps
   curl http://localhost:8080  # openHAB UI
   ```

6. **Verify Functionality**
   - [ ] openHAB UI accessible on LAN
   - [ ] Items show correct states
   - [ ] Mode changes work
   - [ ] Node-RED flows execute
   - [ ] MQTT messages flow
   - [ ] Traefik routes correctly
   - [ ] VPN tunnel to VPS established

7. **Re-establish Remote Access**
   ```bash
   # Verify WireGuard tunnel to VPS
   sudo wg show
   ping 10.100.0.1  # VPS hub
   
   # Test remote user connection
   # (From desktop utility, connect to site)
   ```

**Recovery Time Objective (RTO)**: 4 hours (bare metal to fully operational)  
**Recovery Point Objective (RPO)**: 24 hours (daily backups)

---

## 8. Integration Points

### 8.1 Field Layer → Control Layer

**KNX Integration:**
- openHAB KNX binding connects to KNX/IP gateway
- Group addresses mapped to openHAB items
- Bidirectional: Stack reads sensor states, sends commands to actuators

**DALI Integration:**
- DALI gateway (e.g., Lunatone) with IP interface
- openHAB DALI binding or Modbus TCP
- Lighting scenes stored in openHAB, triggered by KNX buttons

**Modbus Integration:**
- Modbus TCP for VFDs, energy meters, plant controllers
- Node-RED can read Modbus directly for PHM data
- openHAB Modbus binding for device control

**MQTT Integration:**
- Consumer overlay devices (Hue, smart plugs) via MQTT
- Zigbee2MQTT or similar bridges
- Loose coupling: devices publish state, stack subscribes

### 8.2 Control Layer → Remote Layer

**Site → VPS Communication (via WireGuard):**
- Persistent tunnel (established at startup)
- Metrics push (Prometheus remote write to VPS)
- Audit log sync (rsync over tunnel)
- Backup transfers (rsync to VPS/S3)

**Desktop Utility → VPS:**
- User authentication (OAuth2 / JWT)
- WireGuard config retrieval (encrypted)
- Site authorization checks
- (Future) Relay API for read-only status

---

## 9. Security Hardening

### 9.1 On-Site Security

**OS Hardening:**
- Minimal Ubuntu Server installation
- UFW firewall (default deny, allow only necessary ports)
- Automatic security updates (`unattended-upgrades`)
- SSH key-only authentication (password auth disabled)

**Docker Security:**
- Run containers as non-root users where possible
- Read-only root filesystems
- Seccomp/AppArmor profiles
- Resource limits (CPU, memory)

**Network Security:**
- VLANs for segmentation (control, CCTV, guest, IoT)
- Firewall between VLANs
- No port forwarding for services (VPN only)
- mDNS only on control VLAN

**Certificate Management:**
- Let's Encrypt for public domains (if site has static IP)
- Self-signed for `.local` domains
- Cert rotation via Traefik ACME

### 9.2 VPN Security

**WireGuard Best Practices:**
- Unique keypairs per user and site
- Regular key rotation (6mo users, 12mo sites)
- PersistentKeepalive to prevent NAT timeout
- Split-tunnel to minimize attack surface

**Access Control:**
- Site authorizes users (whitelist of allowed user public keys)
- VPS enforces routing (users can only reach authorized sites)
- Failed auth attempts logged and rate-limited

### 9.3 Desktop Utility Security

**Config Encryption:**
- AES-256-GCM with password-derived key
- PBKDF2 (100k iterations, SHA-256)
- Configs never stored unencrypted on disk

**OS Integration:**
- Linux: `libsecret` for key storage
- Windows: DPAPI for key storage
- Session tokens in OS-protected storage (not plain files)

**Code Security:**
- Go binary compiled with stack protection
- No embedded secrets (all user-specific)
- Auto-update with signature verification

---

## 10. Performance & Scalability

### 10.1 Hardware Recommendations

**Minimum (Small Home):**
- Intel NUC (8GB RAM, 128GB SSD)
- Raspberry Pi 4 (8GB) - acceptable but not ideal

**Recommended (Typical Deployment):**
- Intel NUC 11/12 or equivalent
- 16GB RAM
- 256GB NVMe SSD
- Dual Gigabit Ethernet (VLAN trunking)

**High-End (Estate / Commercial):**
- Small rack server (Dell, HP, Supermicro)
- 32GB RAM
- 512GB NVMe SSD (RAID1)
- Redundant PSU
- 10GbE for CCTV backbone

### 10.2 Scaling Considerations

**Single Site Limits:**
- **Devices**: 500-1000 (KNX theoretical limit: 57,000)
- **CCTV Streams**: 32 cameras @ 1080p (4-8 concurrent viewers)
- **Concurrent Users**: 10-20 (web UI)
- **PHM Assets**: 50-100 monitored devices

**Multi-Site Considerations:**
- Each site is independent (horizontal scaling)
- VPS hub can support 50-100 sites (network bandwidth limit)
- Central monitoring/dashboards aggregate via VPS

---

## 11. Development Roadmap

### v0.5 (Current Phase)
- [x] Architecture documentation (this document)
- [ ] Docker Compose stack implementation
- [ ] Basic openHAB/Node-RED configuration
- [ ] Desktop utility (Go) - minimal viable
- [ ] Offline resilience testing
- [ ] Backup/restore scripts

### v0.6 (Domain Demos)
- [ ] Environment module (temps, CO₂, humidity)
- [ ] Lighting module (scenes, KNX integration)
- [ ] Security module (alarm states, CCTV health)
- [ ] PHM demo (simulated pump monitoring)

### v0.7 (VPS Implementation)
- [ ] VPS WireGuard hub setup
- [ ] User authentication API
- [ ] Config provisioning service
- [ ] Central audit logging
- [ ] Long-term metrics storage

### v0.8 (Mobile Apps)
- [ ] iOS app (Swift + WireGuardKit)
- [ ] Android app (Kotlin + wireguard-android)
- [ ] App Store / Play Store deployment

### v1.0 (Production Ready)
- [ ] First real site deployment
- [ ] Runbook documentation
- [ ] Installer package (QR code generation)
- [ ] Support processes
- [ ] Monitoring/alerting (PHM operational)

---

## 12. References

**Related Documentation:**
- [Technical Specification](gray-logic-stack.md) - Overall system design
- [Business Case](business-case.md) - Commercial justification
- [Copilot Instructions](../.github/copilot-instructions.md) - AI agent guidance
- [CHANGELOG](../CHANGELOG.md) - Spec evolution

**External Standards:**
- KNX: https://www.knx.org/
- DALI: https://www.dali-alliance.org/
- WireGuard: https://www.wireguard.com/
- Traefik: https://doc.traefik.io/traefik/
- openHAB: https://www.openhab.org/docs/
- Node-RED: https://nodered.org/docs/

---

**Document Version**: 0.5  
**Last Updated**: 2026-01-11  
**Status**: Design Phase (Pre-Implementation)
