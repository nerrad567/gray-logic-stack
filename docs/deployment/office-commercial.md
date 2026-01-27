---
title: Office & Commercial Deployment Guide
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - overview/vision.md
  - overview/principles.md
  - architecture/system-overview.md
---

# Office & Commercial Deployment Guide

This guide covers deploying Gray Logic in commercial office buildings, light commercial spaces, and multi-tenant environments. It addresses the unique requirements of commercial deployments versus residential installations.

---

## Overview

### Commercial vs Residential

| Aspect | Residential | Commercial |
|--------|-------------|------------|
| **Users** | Family/guests (few) | Employees/visitors (many) |
| **Schedules** | Flexible | Fixed occupancy hours |
| **HVAC** | Zones by room | VAV/FCU zones, central plant |
| **Lighting** | Scene-based | Occupancy/daylight-based |
| **Access** | Single entry | Multiple entries, access control |
| **Booking** | Not applicable | Meeting room scheduling |
| **Energy** | Cost focus | Cost + compliance |
| **IT** | Home network | Enterprise network, IT policies |

### Target Buildings

Gray Logic commercial deployment suits:

- Small-medium offices (up to ~5,000 m²)
- Boutique hotels and B&Bs
- Medical clinics and dental practices
- Retail showrooms
- Light industrial with office space
- Multi-tenant buildings (per-tenant deployment)

For larger buildings with existing BMS, Gray Logic can complement or integrate via BACnet (Year 2+).

---

## Architecture

### Commercial Deployment Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         ENTERPRISE NETWORK                           │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    CORPORATE VLAN                            │    │
│  │  • User devices, corporate applications                     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    BUILDING CONTROL VLAN                     │    │
│  │  • Gray Logic Server                                        │    │
│  │  • KNX/IP Interface                                         │    │
│  │  • DALI Gateways                                            │    │
│  │  • BACnet Controllers (Year 2)                              │    │
│  │  • Access Control System                                    │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    CCTV VLAN                                 │    │
│  │  • NVR, Cameras                                             │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    GUEST NETWORK                             │    │
│  │  • Visitor WiFi (isolated)                                  │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### IT Integration Points

| System | Integration | Protocol |
|--------|-------------|----------|
| **Active Directory** | User authentication | LDAP |
| **Microsoft 365** | Room booking | Graph API |
| **Google Workspace** | Room booking | Calendar API |
| **Exchange** | Room booking | EWS |
| **SIEM** | Security logging | Syslog |
| **Network monitoring** | Health metrics | SNMP |

---

## Network Requirements

### Firewall Rules

```yaml
firewall_rules:
  # Building control VLAN → Corporate VLAN
  - name: "GL to AD"
    source: "building_control"
    destination: "corporate"
    port: 636                       # LDAPS
    protocol: "tcp"
    action: "allow"
    
  - name: "GL to Calendar"
    source: "building_control"
    destination: "internet"
    port: 443                       # Microsoft/Google APIs
    protocol: "tcp"
    action: "allow"
    
  # Corporate VLAN → Building control
  - name: "Admin access"
    source: "corporate"
    destination: "building_control"
    port: 443                       # Gray Logic UI
    protocol: "tcp"
    action: "allow"
    # Additional: Restrict to IT admin IPs
    
  # Building control → Internet
  - name: "Time sync"
    source: "building_control"
    destination: "internet"
    port: 123                       # NTP
    protocol: "udp"
    action: "allow"
```

### Server Placement

```yaml
server_requirements:
  location: "Server room or comms closet"
  
  network:
    interfaces: 2                   # Redundancy or separation
    vlan_access: ["building_control"]
    
  power:
    ups_protected: true
    runtime_minutes: 30
    
  cooling:
    ambient_max: 35                 # °C
```

---

## Occupancy Schedules

### Standard Office Schedule

```yaml
occupancy_schedule:
  id: "office-standard"
  name: "Standard Office Hours"
  
  weekly:
    monday:
      - start: "07:00"
        end: "19:00"
        state: "occupied"
    tuesday:
      - start: "07:00"
        end: "19:00"
        state: "occupied"
    wednesday:
      - start: "07:00"
        end: "19:00"
        state: "occupied"
    thursday:
      - start: "07:00"
        end: "19:00"
        state: "occupied"
    friday:
      - start: "07:00"
        end: "17:00"
        state: "occupied"
    saturday: []
    sunday: []
    
  # Pre-conditioning
  pre_condition:
    hvac_lead_minutes: 60
    lighting_lead_minutes: 15
    
  # Post-occupancy
  post_occupancy:
    delay_minutes: 30
```

### Extended Hours Area

```yaml
occupancy_schedule:
  id: "gym-extended"
  name: "Staff Gym Extended Hours"
  
  weekly:
    monday:
      - start: "06:00"
        end: "22:00"
        state: "occupied"
    # ... etc
```

### Holiday Calendar

```yaml
holiday_calendar:
  id: "uk-bank-holidays-2026"
  name: "UK Bank Holidays 2026"
  
  holidays:
    - date: "2026-01-01"
      name: "New Year's Day"
    - date: "2026-04-03"
      name: "Good Friday"
    - date: "2026-04-06"
      name: "Easter Monday"
    - date: "2026-05-04"
      name: "Early May Bank Holiday"
    - date: "2026-05-25"
      name: "Spring Bank Holiday"
    - date: "2026-08-31"
      name: "Summer Bank Holiday"
    - date: "2026-12-25"
      name: "Christmas Day"
    - date: "2026-12-28"
      name: "Boxing Day (substitute)"
```

---

## Lighting Control

### Open Plan Office

```yaml
open_plan_lighting:
  area_id: "open-plan-main"
  
  # Daylight zones
  zones:
    - id: "perimeter"
      lights: ["row-1", "row-2"]
      daylight_sensor: "lux-perimeter"
      min_level: 20                 # Safety minimum
      
    - id: "core"
      lights: ["row-3", "row-4", "row-5"]
      daylight_sensor: "lux-core"
      min_level: 20
      
  # Occupancy control
  occupancy:
    sensors:
      - "pir-zone-1"
      - "pir-zone-2"
      - "pir-zone-3"
    vacancy_timeout_minutes: 15
    
  # Schedule integration
  schedule:
    occupied:
      target_lux: 500
      daylight_harvesting: true
    unoccupied:
      level: 0                      # Off when unoccupied
    after_hours:
      level: 30                     # Reduced if someone present
      vacancy_timeout_minutes: 5
```

### Meeting Rooms

```yaml
meeting_room_lighting:
  room_id: "meeting-room-1"
  
  # Booking integration
  booking_source: "microsoft_365"
  room_email: "room1@company.com"
  
  # Scenes
  scenes:
    - id: "meeting"
      name: "Meeting"
      trigger: "booking_start"
      brightness: 100
      color_temp: 4000
      
    - id: "presentation"
      name: "Presentation"
      trigger: "manual"
      front_lights: 30
      rear_lights: 100
      
    - id: "video_call"
      name: "Video Call"
      trigger: "manual"
      brightness: 100
      color_temp: 5000
      
  # End of booking
  post_meeting:
    scene: "off"
    delay_minutes: 10
```

### Corridor and Circulation

```yaml
corridor_lighting:
  area_id: "corridor-ground"
  
  schedule:
    occupied_hours:
      base_level: 50                # Always on during day
      presence_level: 100
    after_hours:
      base_level: 10                # Security level
      presence_level: 80
      timeout_seconds: 120
```

---

## Climate Control

### VAV System Configuration

```yaml
vav_system:
  ahu_id: "ahu-main"
  
  zones:
    - zone_id: "zone-open-plan"
      vav_id: "vav-op-1"
      setpoints:
        occupied:
          heating: 21
          cooling: 23
        unoccupied:
          heating: 16
          cooling: 28
      demand_ventilation:
        enabled: true
        co2_sensor: "co2-op-1"
        
    - zone_id: "zone-meeting-1"
      vav_id: "vav-mr-1"
      # Responds to booking
      booking_aware: true
```

### Out-of-Hours Override

```yaml
out_of_hours_override:
  # Physical button at reception
  trigger:
    device_id: "keypad-reception"
    button: 4
    
  # Parameters
  duration_minutes: 120
  extend_allowed: true
  max_extensions: 2
  
  # Scope
  zones: ["zone-ground-floor"]
  
  # Setpoints during override
  setpoints:
    heating: 21
    cooling: 23
    
  # Logging
  log_user: true
  notify: ["facility_manager"]
```

---

## Access Control Integration

### Standard Entry Points

```yaml
access_points:
  - id: "main-entrance"
    type: "turnstile"
    direction: "bidirectional"
    occupancy_counting: true
    
  - id: "rear-entrance"
    type: "door"
    card_reader: true
    
  - id: "loading-bay"
    type: "shutter"
    scheduled_access: true
    hours: "07:00-18:00"
```

### Meeting Room Access

```yaml
meeting_room_access:
  room_id: "meeting-room-1"
  
  # Access follows booking
  booking_access:
    enabled: true
    lead_time_minutes: 5
    overstay_minutes: 15
    
  # Display outside room
  room_panel:
    device_id: "panel-mr1-door"
    show_booking: true
    show_available: true
    allow_ad_hoc_booking: true
```

---

## Meeting Room Booking

### Microsoft 365 Integration

```yaml
booking_integration:
  provider: "microsoft_365"
  
  # Azure AD app registration
  auth:
    tenant_id: "${AZURE_TENANT_ID}"
    client_id: "${AZURE_CLIENT_ID}"
    client_secret: "${AZURE_CLIENT_SECRET}"
    
  # Room resources
  rooms:
    - room_id: "meeting-room-1"
      email: "room1@company.com"
      capacity: 8
      
    - room_id: "meeting-room-2"
      email: "room2@company.com"
      capacity: 4
      
    - room_id: "boardroom"
      email: "boardroom@company.com"
      capacity: 20
      
  # Sync settings
  sync:
    interval_minutes: 5
    lookahead_days: 7
```

### Google Workspace Integration

```yaml
booking_integration:
  provider: "google_workspace"
  
  auth:
    service_account: "${GOOGLE_SERVICE_ACCOUNT}"
    subject: "admin@company.com"
    
  rooms:
    - room_id: "meeting-room-1"
      calendar_id: "room1@company.com"
```

### Room Panel Display

```yaml
room_panel:
  device_id: "panel-mr1-door"
  
  display:
    show_current_booking: true
    show_next_booking: true
    show_organizer: false           # Privacy
    show_subject: false             # Privacy
    
  actions:
    check_in: true
    extend: true
    end_early: true
    book_now: true
    
  # No-show release
  no_show:
    enabled: true
    timeout_minutes: 10
    release_to_available: true
```

---

## Energy Management

### Tenant Billing

For multi-tenant buildings:

```yaml
tenant_metering:
  building_id: "building-main"
  
  tenants:
    - tenant_id: "tenant-floor-1"
      name: "Company A"
      areas: ["area-floor-1"]
      meters:
        - meter_id: "meter-tenant-a-main"
          type: "electrical"
        - meter_id: "meter-tenant-a-hvac"
          type: "btu"
          
    - tenant_id: "tenant-floor-2"
      name: "Company B"
      areas: ["area-floor-2"]
      meters:
        - meter_id: "meter-tenant-b-main"
          type: "electrical"
          
  # Billing reports
  reporting:
    schedule: "monthly"
    format: "pdf"
    include:
      - consumption_kwh
      - peak_demand_kw
      - comparison_previous
```

### Energy Reporting

```yaml
energy_reporting:
  # Dashboard metrics
  dashboard:
    - metric: "total_consumption_today"
    - metric: "peak_demand_today"
    - metric: "hvac_consumption"
    - metric: "lighting_consumption"
    - metric: "co2_emissions"
    
  # Scheduled reports
  reports:
    - name: "Weekly Energy Summary"
      schedule: "weekly"
      recipients: ["facility_manager"]
      
    - name: "Monthly Energy Report"
      schedule: "monthly"
      recipients: ["operations@company.com"]
      include:
        - consumption_breakdown
        - comparison_previous_month
        - comparison_same_month_last_year
        - anomaly_detection
```

---

## User Management

### Active Directory Integration

```yaml
ad_integration:
  type: "ldaps"
  
  server:
    host: "ldaps://dc.company.local"
    port: 636
    base_dn: "DC=company,DC=local"
    
  # Bind account
  bind:
    dn: "CN=Gray Logic Service,OU=Service Accounts,DC=company,DC=local"
    password: "${AD_BIND_PASSWORD}"
    
  # User search
  user_search:
    base: "OU=Users,DC=company,DC=local"
    filter: "(objectClass=user)"
    
  # Group mapping
  group_mapping:
    - ad_group: "CN=Building Admins,OU=Groups,DC=company,DC=local"
      gl_role: "admin"
      
    - ad_group: "CN=Facility Managers,OU=Groups,DC=company,DC=local"
      gl_role: "facility_manager"
      
    - ad_group: "CN=Staff,OU=Groups,DC=company,DC=local"
      gl_role: "user"
```

### Role Definitions

```yaml
roles:
  - id: "admin"
    name: "Administrator"
    permissions:
      - "all"
      
  - id: "facility_manager"
    name: "Facility Manager"
    permissions:
      - "view_all"
      - "control_hvac"
      - "control_lighting"
      - "view_access_logs"
      - "override_schedules"
      - "view_energy"
      
  - id: "user"
    name: "Standard User"
    permissions:
      - "view_own_area"
      - "control_meeting_room"
      - "request_override"
      
  - id: "visitor"
    name: "Visitor"
    permissions:
      - "view_wayfinding"
```

---

## Commissioning

### Pre-Installation

- [ ] IT network requirements confirmed
- [ ] VLANs created and configured
- [ ] Firewall rules approved
- [ ] Server rack space allocated
- [ ] Power and UPS confirmed
- [ ] AD integration account created
- [ ] Calendar API access granted

### Installation

- [ ] Server installed and networked
- [ ] Gray Logic Core installed
- [ ] KNX/DALI gateways connected
- [ ] All devices discovered and named
- [ ] Areas and rooms defined
- [ ] Occupancy schedules programmed

### Integration Testing

- [ ] AD authentication working
- [ ] Calendar sync working
- [ ] Lighting occupancy control verified
- [ ] HVAC schedule following occupancy
- [ ] Meeting room automation tested
- [ ] Access control events flowing
- [ ] Out-of-hours override working
- [ ] Energy metering accurate

### Training

- [ ] IT team trained on administration
- [ ] Facility team trained on operation
- [ ] Reception trained on overrides
- [ ] End-user guide distributed

### Documentation

- [ ] As-built drawings updated
- [ ] Device schedule created
- [ ] Network diagram documented
- [ ] Runbook for facility team
- [ ] Emergency procedures documented

---

## Ongoing Operations

### Routine Tasks

| Task | Frequency | Responsible |
|------|-----------|-------------|
| Review energy reports | Weekly | Facility Manager |
| Check system health | Daily | Auto-alert |
| Verify calendar sync | Weekly | IT |
| Review access logs | Monthly | Security |
| Test emergency lighting | Monthly | Maintenance |
| System backup verify | Weekly | IT |
| Update holiday calendar | Annually | Facility Manager |

### Support Escalation

```yaml
support_escalation:
  level_1:
    scope: "User questions, basic troubleshooting"
    contact: "Internal IT helpdesk"
    sla: "4 hours"
    
  level_2:
    scope: "Configuration changes, integration issues"
    contact: "Facility Manager"
    sla: "1 business day"
    
  level_3:
    scope: "System issues, hardware failures"
    contact: "Gray Logic Support"
    sla: "Next business day"
```

---

## Related Documents

- [System Overview](../architecture/system-overview.md) — Technical architecture
- [Climate Domain](../domains/climate.md) — Commercial HVAC control
- [Lighting Domain](../domains/lighting.md) — Commercial lighting
- [Plant Domain](../domains/plant.md) — AHU and plant equipment
- [Access Control](../integration/access-control.md) — Access system integration
- [CCTV Integration](../integration/cctv.md) — Surveillance systems
- [Fire Alarm](../integration/fire-alarm.md) — Fire system integration
- [BACnet Protocol](../protocols/bacnet.md) — Commercial HVAC integration

