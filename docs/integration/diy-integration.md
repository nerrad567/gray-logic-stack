---
title: DIY Device Integration Specification
version: 1.0.0
status: active
last_updated: 2026-01-18
depends_on:
  - overview/principles.md
  - architecture/security-model.md
  - domains/appliances.md
---

# DIY Device Integration Specification

> **Philosophy**: Gray Logic welcomes customer-owned devices, but they must respect our core principles: **offline-first**, **local control**, and **no cloud dependencies for core functionality**.

---

## 1. Overview

Gray Logic is designed for professional installation with open standards (KNX, DALI, Modbus). However, we recognize that customers may wish to integrate their own consumer smart devices — smart plugs, lights, kettles, sensors, and more.

This document defines:
1. **Supported integration methods** for DIY devices
2. **Device requirements** for Gray Logic compatibility
3. **Security and isolation policies**
4. **What we support vs. what customers manage themselves**

---

## 2. Integration Tiers

### Tier 1: Matter/Thread (Recommended)

**The gold standard for consumer device integration.**

| Aspect | Details |
|--------|---------|
| **Protocol** | Matter over Thread or Matter over Wi-Fi |
| **Requirement** | Device must work 100% locally (no cloud required) |
| **Gray Logic Support** | Native Matter controller built into Core |
| **Examples** | Eve sensors, Nanoleaf lights, newer Philips Hue |

```yaml
matter_controller:
  enabled: true
  thread_border_router: true
  wifi_commissioning: true
  
  # Thread border router hardware
  thread_hardware:
    primary: "Built into Gray Logic Hub"
    alternative: "Apple HomePod Mini, Google Nest Hub (customer-owned)"
    standard: "Thread 1.3.0"
  
  device_categories:
    - lights
    - switches
    - sensors
    - thermostats
    - blinds
    - locks
    
  commissioning:
    method: "QR code or manual pairing code"
    network: "Thread mesh or local Wi-Fi"
    cloud_required: false
```

**Why Matter?**
- Industry standard backed by Apple, Google, Amazon, Samsung
- Guaranteed local operation (no cloud dependency)
- Interoperability across brands
- Thread mesh networking for reliability

---

### Tier 2: Zigbee (via Coordinator)

**For existing Zigbee device ecosystems.**

| Aspect | Details |
|--------|---------|
| **Protocol** | Zigbee 3.0 |
| **Hardware** | USB Zigbee coordinator (e.g., Sonoff ZBDongle-E, SLZB-06) |
| **Gray Logic Support** | Zigbee Bridge (zigbee2mqtt compatible) |
| **Examples** | Aqara sensors, IKEA Trådfri, older Philips Hue |

```yaml
zigbee_bridge:
  enabled: true
  coordinator: "/dev/ttyUSB0"     # USB Zigbee dongle
  
  supported_devices:
    - category: "lights"
      features: ["on_off", "brightness", "color_temp", "color_xy"]
    - category: "sensors"
      features: ["temperature", "humidity", "motion", "contact", "illuminance"]
    - category: "switches"
      features: ["on_off", "power_monitoring"]
    - category: "blinds"
      features: ["position", "tilt"]
      
  mesh_network:
    channel: 15                   # Recommended: 15, 20, 25, 26 (minimal Wi-Fi overlap)
    channel_selection: "Check Wi-Fi channels during commissioning"
    permit_join: false            # Enable only during pairing
```

**Supported Zigbee Brands:**
| Brand | Devices | Notes |
|-------|---------|-------|
| **Aqara** | Sensors, switches, blinds | Excellent quality, local-first |
| **IKEA** | Lights, blinds, sensors | Budget-friendly, reliable |
| **Philips Hue** | Lights | Direct pairing (bypass Hue Bridge) |
| **Sonoff** | Switches, sensors | Good value |
| **Tuya Zigbee** | Various | Quality varies, check reviews |

---

### Tier 3: Z-Wave (via Controller)

**For Z-Wave device ecosystems (popular in US/EU).**

| Aspect | Details |
|--------|---------|
| **Protocol** | Z-Wave (700/800 series) |
| **Hardware** | USB Z-Wave controller (e.g., Zooz ZST39, Aeotec Z-Stick) |
| **Gray Logic Support** | Z-Wave Bridge |
| **Examples** | Zooz switches, Aeotec sensors, Yale locks |

```yaml
zwave_bridge:
  enabled: true
  controller: "/dev/ttyACM0"
  region: "EU"                    # EU, US, ANZ frequency
  
  security:
    s2_enabled: true              # Require S2 security for new devices
    s0_fallback: false            # Reject insecure devices
```

---

### Tier 4: Local Wi-Fi API

**For Wi-Fi devices with documented local APIs.**

| Aspect | Details |
|--------|---------|
| **Protocol** | HTTP/MQTT over local network |
| **Requirement** | Device must have local API (no cloud polling) |
| **Gray Logic Support** | Generic REST/MQTT integration |
| **Examples** | Shelly devices, Tasmota-flashed plugs, ESPHome |

```yaml
wifi_device_integration:
  # Shelly devices (native local API)
  shelly:
    discovery: "mDNS"
    protocol: "HTTP + MQTT"
    features:
      - power_monitoring
      - relay_control
      - energy_metering
      
  # Tasmota-flashed devices
  tasmota:
    protocol: "MQTT"
    topic_pattern: "tasmota/{device_id}/#"
    
  # ESPHome devices
  esphome:
    protocol: "Native API"
    encryption: true
```

**Recommended Wi-Fi Devices:**
| Brand | Why Recommended |
|-------|-----------------|
| **Shelly** | Native local API, no cloud required, professional quality |
| **Tasmota** | Open-source firmware, full local control |
| **ESPHome** | DIY-friendly, highly customizable |

---

### Tier 5: Cloud API (Tolerated, Not Recommended)

**For devices with no local control option.**

| Aspect | Details |
|--------|---------|
| **Protocol** | Vendor cloud API |
| **Requirement** | Customer accepts cloud dependency |
| **Gray Logic Support** | Cloud Bridge (optional module) |
| **Examples** | Ring cameras, Nest thermostats, some Tuya devices |

> [!CAUTION]
> Cloud-dependent devices violate Gray Logic's offline-first principle. They will NOT function during internet outages and may break without warning when vendors change their APIs.

```yaml
cloud_bridge:
  enabled: false                  # Disabled by default
  
  # Customer explicitly enables per-service
  services:
    tuya:
      enabled: false
      note: "Prefer Zigbee or local-flash Tuya devices"
    google_home:
      enabled: false
      note: "Exposes Gray Logic TO Google, not the other way"
```

**Policy:**
- Cloud integrations are **customer-managed**
- Gray Logic provides no SLA for cloud-dependent devices
- If vendor API breaks, customer is responsible for resolution

---

## 3. Exposing Gray Logic to External Ecosystems

Customers may want to control Gray Logic devices from Apple Home, Google Home, or Alexa.

### HomeKit Bridge (Recommended)

**Expose Gray Logic devices to Apple Home.**

```yaml
homekit_bridge:
  enabled: true
  pin: "123-45-678"               # Custom during setup
  
  exposed_devices:
    - category: "lights"
      include_all: true
    - category: "switches"
      include_all: true
    - category: "thermostats"
      include_all: true
    - category: "blinds"
      include_all: true
    - category: "sensors"
      include_all: true
    - category: "locks"
      include_all: false          # Security: explicit opt-in only
      
  security:
    local_only: true              # Gray Logic does not use iCloud relay
    pairing_timeout_minutes: 10
    note: "User's Apple devices may still use iCloud if configured by user"
```

**HomeKit Benefits:**
- 100% local control (no cloud)
- Works with Apple Watch, Siri
- Home app for simple family use

### Google Home / Alexa (Cloud Required)

**Expose Gray Logic to Google/Amazon ecosystems.**

> [!WARNING]
> This requires internet connectivity and sends commands through Google/Amazon servers.

```yaml
google_home_integration:
  enabled: false                  # Opt-in only
  
  # Requires Gray Logic Cloud subscription (Connect tier+)
  cloud_link_required: true
  
  exposed_capabilities:
    - "action.devices.types.LIGHT"
    - "action.devices.types.SWITCH"
    - "action.devices.types.THERMOSTAT"
    - "action.devices.types.BLINDS"
    
alexa_integration:
  enabled: false
  cloud_link_required: true
  skill_name: "Gray Logic Smart Home"
```

---

## 4. Device Requirements

### Minimum Requirements for Integration

| Requirement | Reason |
|-------------|--------|
| **Local control** | Must work without internet |
| **Documented API** | No reverse-engineering unstable hacks |
| **Standard protocol** | Matter, Zigbee, Z-Wave, or local HTTP/MQTT |
| **No cloud-only** | Vendor cloud cannot be sole control path |

### Device Vetting Checklist

Before adding a DIY device, verify:

- [ ] **Offline test**: Unplug router — does device still respond to local commands?
- [ ] **Protocol check**: Is it Matter, Zigbee 3.0, Z-Wave, or local API?
- [ ] **Firmware updates**: Can firmware be updated locally (not forced cloud)?
- [ ] **Privacy**: Does device phone home with usage data?

---

## 5. Network Architecture

### Recommended VLAN Segmentation

```
┌─────────────────────────────────────────────────────────────────────┐
│                        NETWORK ARCHITECTURE                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  VLAN 10: Management (Trusted)                                       │
│  ├── Gray Logic Core                                                 │
│  ├── KNX/IP interface                                                │
│  └── Admin workstations                                              │
│                                                                      │
│  VLAN 20: IoT Devices (Untrusted)                                    │
│  ├── Matter devices                                                  │
│  ├── Zigbee coordinator                                              │
│  ├── Z-Wave controller                                               │
│  ├── Shelly devices                                                  │
│  └── All customer DIY devices                                        │
│                                                                      │
│  VLAN 30: Cameras (Isolated)                                         │
│  ├── NVR                                                             │
│  └── IP cameras (no internet access)                                 │
│                                                                      │
│  VLAN 40: Guest (Isolated)                                           │
│  └── Guest Wi-Fi                                                     │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Firewall Rules for DIY Devices

```yaml
firewall_rules:
  iot_vlan:
    # Allow: IoT → Core (for control)
    - action: allow
      source: "vlan20"
      destination: "core:1883"    # MQTT
      
    # Allow: Core → IoT (for commands)
    - action: allow
      source: "core"
      destination: "vlan20"
      
    # BLOCK: IoT → Internet (prevent phoning home)
    - action: block
      source: "vlan20"
      destination: "internet"
      log: true
      
    # BLOCK: IoT → Other VLANs
    - action: block
      source: "vlan20"
      destination: "vlan10,vlan30,vlan40"
```

---

## 6. Support Policy

### What Gray Logic Supports

| Category | Support Level |
|----------|---------------|
| **Matter devices** | Full — Native integration |
| **Zigbee/Z-Wave** | Full — Via official bridges |
| **Shelly/Tasmota/ESPHome** | Standard — Documented integrations |
| **HomeKit bridge** | Full — Native feature |

### What Customers Manage

| Category | Reason |
|----------|--------|
| **Cloud integrations** | Vendor APIs change without notice |
| **Reverse-engineered APIs** | Unstable, may break on firmware update |
| **Unsupported protocols** | No official bridge available |
| **Device firmware** | Customer responsibility to update |

### Community Resources

```yaml
community_support:
  forum: "community.graylogic.io"
  discord: "discord.gg/graylogic"
  
  diy_channels:
    - "#matter-devices"
    - "#zigbee-help"
    - "#zwave-help"
    - "#shelly-tasmota"
    - "#homekit-integration"
```

---

## 7. Example Configurations

### Example 1: Adding Aqara Sensors (Zigbee)

```yaml
# 1. Enable Zigbee bridge
zigbee_bridge:
  enabled: true
  coordinator: "/dev/ttyUSB0"
  
# 2. Pair device (temporary permit join)
# UI: Settings → Zigbee → Add Device → Put sensor in pairing mode

# 3. Device appears in Gray Logic
devices:
  - id: "aqara-temp-living-room"
    name: "Living Room Temperature"
    type: "sensor"
    protocol: "zigbee"
    capabilities:
      - temperature
      - humidity
      - battery
```

### Example 2: Adding Shelly Plug (Wi-Fi)

```yaml
# 1. Configure Shelly to use MQTT
# Shelly Web UI → Settings → MQTT → Enable
# Broker: 192.168.1.10 (Core IP)

# 2. Device auto-discovered via MQTT
devices:
  - id: "shelly-plug-kettle"
    name: "Kitchen Kettle"
    type: "switch"
    protocol: "shelly"
    capabilities:
      - on_off
      - power_monitoring
      - energy_metering
```

### Example 3: Exposing to HomeKit

```yaml
# 1. Enable HomeKit bridge
homekit_bridge:
  enabled: true
  
# 2. Scan QR code in Apple Home app
# 3. All exposed devices appear in Home app
# 4. Control via Siri: "Hey Siri, turn on the kitchen lights"
```

---

## 8. Commissioning Checklist

Before going live with DIY devices:

- [ ] **Network segmentation**: DIY devices on IoT VLAN
- [ ] **Firewall rules**: IoT VLAN blocked from internet
- [ ] **Coordinator hardware**: Zigbee/Z-Wave dongles installed
- [ ] **Matter controller**: Enabled in Core settings
- [ ] **Offline test**: All devices respond with router unplugged
- [ ] **HomeKit pairing**: Tested if enabled
- [ ] **Customer training**: User knows how to add/remove devices

---

## 9. Limitations & Honest Disclosure

> [!IMPORTANT]
> Gray Logic is designed for **professional wired infrastructure** (KNX, DALI, Modbus). Wireless DIY devices are supported but are inherently less reliable than wired systems.

| Limitation | Impact | Mitigation |
|------------|--------|------------|
| **Wireless reliability** | Zigbee/Wi-Fi can have interference | Use Thread where possible |
| **Battery devices** | Sensors need battery replacement | PHM tracks battery levels |
| **Firmware updates** | Vendor updates may break integration | Test updates in staging |
| **Device quality** | Cheap devices may fail | Recommend quality brands |

---

## References

- [Appliances Domain](../domains/appliances.md) — White goods integration
- [Security Model](../architecture/security-model.md) — Network security
- [API Specification](../interfaces/api.md) — REST/WebSocket access
- [Matter Specification](https://csa-iot.org/all-solutions/matter/) — Official Matter docs
