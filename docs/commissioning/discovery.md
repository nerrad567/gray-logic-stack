---
title: Device Discovery Specification
version: 1.0.0
status: active
last_updated: 2026-01-15
depends_on:
  - architecture/bridge-interface.md
  - protocols/knx.md
  - protocols/dali.md
  - protocols/modbus.md
---

# Device Discovery Specification

This document specifies how Gray Logic discovers devices on various protocols and the workflow for adding them to the system.

---

## Overview

### Discovery Philosophy

1. **Discovery assists, doesn't replace** — Discovery finds devices; humans name and configure them
2. **Protocol-specific** — Each protocol has different discovery capabilities
3. **Non-intrusive** — Discovery doesn't change device state
4. **Secure** — Discovery doesn't expose sensitive information
5. **Repeatable** — Re-discovery detects new/changed devices

### Discovery Capabilities by Protocol

| Protocol | Auto-Discovery | Device Info | Manual Entry Required |
|----------|---------------|-------------|----------------------|
| **KNX** | Limited (knxd scan) | Group addresses | Yes (function mapping) |
| **DALI** | Yes (query all) | Short address, device type | Yes (names, grouping) |
| **Modbus** | No | N/A | Yes (register maps) |
| **Audio Matrix** | Limited (RS-232 ID) | Model, zones | Yes (zone names) |
| **IP Devices** | mDNS/SSDP | Manufacturer, model | Yes (credentials) |

---

## Discovery Workflow

### High-Level Process

```
┌───────────────────────────────────────────────────────────────────────────┐
│                        DISCOVERY WORKFLOW                                  │
│                                                                            │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐ │
│  │   Trigger   │───▶│   Scan      │───▶│  Propose    │───▶│   Approve   │ │
│  │  Discovery  │    │  Protocol   │    │  Devices    │    │  & Configure│ │
│  └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘ │
│                                                                            │
│  Manual trigger      Bridge scans      Show candidates   User confirms,   │
│  or scheduled        physical bus      in staging area   names devices    │
└───────────────────────────────────────────────────────────────────────────┘
```

### State Diagram

```
                    ┌─────────────┐
                    │  Discovered │
                    │   (staged)  │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
       ┌───────────┐ ┌───────────┐ ┌───────────┐
       │  Approved │ │  Ignored  │ │  Merged   │
       │  (active) │ │  (hidden) │ │ (existing)│
       └───────────┘ └───────────┘ └───────────┘
```

---

## Protocol-Specific Discovery

### KNX Discovery

```yaml
knx_discovery:
  # Capabilities
  capabilities:
    auto_discovery: "limited"
    info_available: ["group_addresses"]
    info_not_available: ["device_names", "function_mapping"]

  # Discovery methods
  methods:
    # Method 1: Group address monitor
    group_monitor:
      description: "Listen for telegrams on the bus"
      duration: "5-30 minutes"
      captures:
        - "Group addresses in use"
        - "Data types (inferred)"
        - "Message frequency"
      requires: "User to activate devices during scan"

    # Method 2: ETS project import
    ets_import:
      description: "Import ETS project file"
      file_types: [".knxproj"]
      extracts:
        - "All group addresses"
        - "Device names"
        - "Datapoint types"
        - "Topology"
      recommended: true             # Best discovery method for KNX

    # Method 3: Individual address scan (limited)
    individual_scan:
      description: "Scan for devices by individual address"
      range: "1.1.1 to 15.15.255"
      info: "Confirms device presence only"

  # Discovery message format
  discovery_message:
    topic: "graylogic/discovery/knx"
    payload:
      devices:
        - address: "1/2/3"
          observed_dpt: "DPT_Switch"
          direction: "output"         # Inferred from traffic
          last_seen: "2026-01-15T10:30:00Z"
          suggested_type: "switch"
          suggested_name: "Switch 1/2/3"

  # Commissioning workflow
  workflow:
    1: "Import ETS project (preferred) or run group monitor"
    2: "Review discovered addresses"
    3: "Map addresses to logical devices"
    4: "Assign names and rooms"
    5: "Configure capabilities per device"
```

### DALI Discovery

```yaml
dali_discovery:
  # Capabilities
  capabilities:
    auto_discovery: "full"
    info_available:
      - "Short address (0-63)"
      - "Device type"
      - "GTIN (product code)"
      - "Firmware version"
      - "Current level"
    info_not_available:
      - "Location"
      - "User-friendly name"

  # Discovery methods
  methods:
    query_all:
      description: "Query all 64 short addresses"
      duration: "~10 seconds per gateway"
      commands:
        - "QUERY DEVICE TYPE"
        - "QUERY ACTUAL LEVEL"
        - "QUERY STATUS"
        - "READ MEMORY BANK 0"        # GTIN and serial

  # Discovery message format
  discovery_message:
    topic: "graylogic/discovery/dali"
    payload:
      gateway: "dali-gw-1"
      devices:
        - short_address: 15
          device_type: 6              # LED module
          gtin: "4050300000000"
          serial: "ABC12345"
          firmware: "1.2.3"
          actual_level: 254
          status:
            lamp_failure: false
            ballast_failure: false
          suggested_name: "DALI Driver 15"

  # Commissioning workflow
  workflow:
    1: "Connect DALI gateway"
    2: "Run discovery (automatic)"
    3: "Identify fixtures physically (use flash command)"
    4: "Assign names and rooms"
    5: "Create groups (optional)"
```

### Modbus Discovery

```yaml
modbus_discovery:
  # No auto-discovery - manual configuration required
  capabilities:
    auto_discovery: "none"
    reason: "Modbus has no standard discovery mechanism"

  # Manual commissioning
  manual_commissioning:
    required_info:
      - "Device IP/serial port"
      - "Slave ID"
      - "Register map (from datasheet)"
      - "Data types per register"

  # Template library
  templates:
    description: "Pre-configured register maps for common devices"
    examples:
      - manufacturer: "Schneider"
        model: "PM5xxx"
        template: "schneider_pm5xxx.yaml"

      - manufacturer: "Eastron"
        model: "SDM630"
        template: "eastron_sdm630.yaml"

  # Discovery message (manual entry)
  discovery_message:
    topic: "graylogic/discovery/modbus"
    payload:
      devices:
        - host: "192.168.1.100"
          port: 502
          unit_id: 1
          template: "eastron_sdm630"
          suggested_name: "Energy Meter - Main Panel"

  # Workflow
  workflow:
    1: "Identify Modbus device model"
    2: "Select template (or create custom)"
    3: "Configure connection parameters"
    4: "Test communication"
    5: "Add to system"
```

### IP Device Discovery

```yaml
ip_discovery:
  # Network-based discovery
  methods:
    mdns:
      description: "Multicast DNS service discovery"
      services:
        - "_http._tcp"
        - "_onvif._tcp"
        - "_rtsp._tcp"

    ssdp:
      description: "Simple Service Discovery Protocol"
      targets:
        - "upnp:rootdevice"

    arp_scan:
      description: "ARP table scan for MAC vendor lookup"
      identifies: "Manufacturer from MAC OUI"

  # Discovered device info
  discovery_message:
    topic: "graylogic/discovery/ip"
    payload:
      devices:
        - ip: "192.168.1.150"
          mac: "AA:BB:CC:DD:EE:FF"
          manufacturer: "Hikvision"
          model: "DS-2CD2143G0-I"
          services:
            - type: "onvif"
              port: 80
            - type: "rtsp"
              port: 554
          suggested_type: "camera"
          suggested_name: "Camera 192.168.1.150"

  # Workflow
  workflow:
    1: "Run network scan"
    2: "Review discovered devices"
    3: "Provide credentials for each device"
    4: "Test connection"
    5: "Assign names and configure"
```

---

## Staging Area

### Concept

Discovered devices land in a "staging area" before being added to the system:

```yaml
staging_area:
  # Purpose
  purpose:
    - "Review before committing"
    - "Merge with existing devices"
    - "Ignore unwanted discoveries"
    - "Batch process discoveries"

  # Staging entry
  staged_device:
    id: uuid                        # Temporary ID
    protocol: string
    address: object                 # Protocol-specific
    discovered_at: timestamp
    discovery_source: string        # "scan", "import", "manual"

    # Device info (from discovery)
    device_info:
      type: string | null
      manufacturer: string | null
      model: string | null
      capabilities: [string] | null

    # Suggestions (can be overridden)
    suggested:
      name: string
      domain: string
      room_id: uuid | null

    # Status
    status: "pending" | "approved" | "ignored" | "merged"

  # Retention
  retention:
    pending_days: 30                # Auto-delete after 30 days if not actioned
```

### API Endpoints

```yaml
staging_api:
  # List staged devices
  list:
    method: GET
    path: "/api/v1/commissioning/staging"
    response:
      devices: [StagedDevice]
      total: integer

  # Approve staged device
  approve:
    method: POST
    path: "/api/v1/commissioning/staging/{id}/approve"
    body:
      name: string                  # Required
      room_id: uuid                 # Required
      domain: string                # Required
      capabilities: [string]        # Optional override
      config: object                # Optional config

  # Ignore staged device
  ignore:
    method: POST
    path: "/api/v1/commissioning/staging/{id}/ignore"
    body:
      reason: string                # Optional

  # Merge with existing
  merge:
    method: POST
    path: "/api/v1/commissioning/staging/{id}/merge"
    body:
      target_device_id: uuid        # Existing device to update

  # Re-discover
  rediscover:
    method: POST
    path: "/api/v1/commissioning/discover"
    body:
      protocol: string              # "knx", "dali", etc.
      options: object               # Protocol-specific options
```

---

## Security Considerations

### Discovery Security

```yaml
discovery_security:
  # Network isolation
  network:
    - "Discovery only on building control VLAN"
    - "No discovery from user VLAN"

  # Authentication
  auth:
    - "Discovery endpoints require admin role"
    - "Device credentials not stored until approved"

  # Information exposure
  exposure:
    - "Don't expose device credentials in staging"
    - "Don't expose internal addresses externally"

  # Auto-discovery risks
  risks:
    - risk: "Rogue device added to network"
      mitigation: "All devices require manual approval"

    - risk: "Discovery floods network"
      mitigation: "Rate limit discovery requests"
```

### Post-Commissioning

```yaml
post_commissioning:
  # Disable auto-discovery after initial setup
  recommendation: "Disable continuous discovery in production"

  settings:
    auto_discovery:
      enabled: false                # Disable after commissioning
      manual_trigger_only: true

  # Periodic re-scan option
  periodic_scan:
    enabled: false
    interval: "monthly"
    notify_admin: true
    auto_add: false                 # Always require approval
```

---

## Web Admin UI

### Commissioning Dashboard

```yaml
commissioning_ui:
  # Discovery panel
  discovery_panel:
    actions:
      - "Scan KNX bus"
      - "Import ETS project"
      - "Scan DALI gateways"
      - "Scan network"
      - "Add Modbus device"

  # Staging table
  staging_table:
    columns:
      - "Protocol"
      - "Address"
      - "Suggested Name"
      - "Type"
      - "Discovered"
      - "Actions"

    actions_per_row:
      - "Approve"
      - "Ignore"
      - "Merge"
      - "Flash/Identify"            # Protocol-dependent

  # Bulk actions
  bulk_actions:
    - "Approve all"
    - "Ignore all"
    - "Assign room to selected"
```

### Device Identification

Help users match discovered devices to physical locations:

```yaml
identification:
  # Flash/blink for visual identification
  flash:
    knx: "Send flash telegram to group address"
    dali: "RECALL MAX LEVEL for 3 seconds"
    modbus: "N/A"
    camera: "Trigger motion detection LED"

  # UI flow
  flow:
    1: "User clicks 'Identify' in staging table"
    2: "Device flashes/activates"
    3: "User confirms physical location"
    4: "User enters name and room"
```

---

## Configuration

### Discovery Settings

```yaml
# /etc/graylogic/commissioning.yaml
commissioning:
  # Auto-discovery (recommend: disable after setup)
  auto_discovery:
    enabled: false
    on_bridge_connect: false        # Don't auto-scan when bridge connects

  # Discovery retention
  staging:
    retention_days: 30
    auto_cleanup: true

  # Protocol-specific settings
  knx:
    group_monitor_duration: 300     # 5 minutes
    import_on_project_change: true

  dali:
    scan_on_gateway_connect: true
    flash_duration_ms: 3000

  ip:
    mdns_enabled: true
    ssdp_enabled: true
    scan_interval: 0                # 0 = manual only
```

---

## Related Documents

- [Bridge Interface](../architecture/bridge-interface.md) — Discovery MQTT topics
- [KNX Protocol](../protocols/knx.md) — KNX-specific discovery
- [DALI Protocol](../protocols/dali.md) — DALI-specific discovery
- [Modbus Protocol](../protocols/modbus.md) — Modbus configuration
- [API Specification](../interfaces/api.md) — API endpoints

