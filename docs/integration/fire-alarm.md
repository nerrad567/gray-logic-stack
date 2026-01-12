---
title: Fire Alarm System Integration
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - overview/principles.md
  - data-model/entities.md
---

# Fire Alarm System Integration

This document specifies how Gray Logic interfaces with fire alarm systems. This is a **monitoring-only** integration — Gray Logic never controls fire safety equipment.

---

## Critical Safety Rules

> **HARD RULE**: Gray Logic **OBSERVES** fire alarm status. It **NEVER CONTROLS** fire safety equipment.

### What Gray Logic CAN Do

- ✅ Receive fire alarm activation signals
- ✅ Display fire alarm status on UIs
- ✅ Trigger automation responses (lights on, blinds open, unlock doors)
- ✅ Send notifications to occupants and managers
- ✅ Log fire events for audit
- ✅ Display zone information (which zone triggered)

### What Gray Logic CANNOT Do

- ❌ Silence fire alarms
- ❌ Reset fire alarm panels
- ❌ Arm/disarm fire detection
- ❌ Control sprinkler systems
- ❌ Override fire dampers
- ❌ Control smoke extraction
- ❌ Control fire doors
- ❌ Any action that could prevent alarm activation

### Why This Matters

Fire alarm systems are:
- Life safety certified (EN 54, NFPA 72, etc.)
- Installed and maintained by certified contractors
- Subject to insurance requirements
- Legally required to operate independently

Gray Logic integration adds convenience and awareness, not control.

---

## Integration Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    FIRE ALARM PANEL                                  │
│              (Certified, Independent Operation)                      │
│                                                                      │
│  Detection → Decision → Activation → Sounders/Beacons               │
│                                                                      │
│  ┌────────────────────────────────────────┐                         │
│  │        Auxiliary Output Contacts       │                         │
│  │   (Fire signal, Fault signal, etc.)    │                         │
│  └──────────────────┬─────────────────────┘                         │
└─────────────────────┼───────────────────────────────────────────────┘
                      │
                      │ Volt-free contacts
                      │
┌─────────────────────▼───────────────────────────────────────────────┐
│                    INTERFACE MODULE                                  │
│                                                                      │
│  ┌──────────────────┐    ┌──────────────────┐                       │
│  │  KNX Binary      │    │  Modbus Digital  │                       │
│  │  Input Module    │    │  Input Module    │                       │
│  └────────┬─────────┘    └────────┬─────────┘                       │
│           │                       │                                  │
└───────────┼───────────────────────┼─────────────────────────────────┘
            │                       │
            ▼                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    GRAY LOGIC CORE                                   │
│                                                                      │
│  Fire Input Device → State Manager → Automation Triggers            │
│                                   → UI Updates                      │
│                                   → Notifications                   │
│                                   → Audit Logging                   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Signal Types

### Fire Alarm Panel Outputs

| Signal | Type | Meaning |
|--------|------|---------|
| **Fire** | Normally Open | Alarm condition active (closes on fire) |
| **Fault** | Normally Closed | System fault (opens on fault) |
| **Isolate** | Normally Open | Zone(s) isolated |
| **Evacuate** | Normally Open | Full evacuation triggered |
| **Pre-Alarm** | Normally Open | Early warning (dual-knock systems) |

### Typical Wiring

```
Fire Alarm Panel          KNX Binary Input
┌─────────────┐           ┌─────────────┐
│ Fire (N/O)  │───────────│ Channel 1   │
│ Fault (N/C) │───────────│ Channel 2   │
│ Isolate     │───────────│ Channel 3   │
│ Common      │───────────│ Common      │
└─────────────┘           └─────────────┘
```

---

## Device Configuration

### Fire Input Device

```yaml
device:
  id: "fire-panel-main"
  name: "Main Fire Alarm Panel"
  type: "fire_input"
  domain: "safety"
  
  protocol: "knx"
  address:
    fire_signal: "10/0/1"
    fault_signal: "10/0/2"
    isolate_signal: "10/0/3"
    
  # Signal configuration
  signals:
    fire:
      address: "10/0/1"
      type: "normally_open"       # Closes on alarm
      invert: false
      debounce_ms: 100
      
    fault:
      address: "10/0/2"
      type: "normally_closed"     # Opens on fault
      invert: true                # So "true" = fault active
      debounce_ms: 100
      
    isolate:
      address: "10/0/3"
      type: "normally_open"
      invert: false
      
  # Zone information (if panel provides)
  zones:
    - zone_id: 1
      name: "Ground Floor"
      areas: ["area-ground-floor"]
    - zone_id: 2
      name: "First Floor"
      areas: ["area-first-floor"]
    - zone_id: 3
      name: "Plant Room"
      areas: ["area-plant"]
```

### State Model

```yaml
FireAlarmState:
  # Overall status
  status: enum                      # normal | fire | fault | isolate
  
  # Individual signals
  fire_active: boolean
  fault_active: boolean
  zones_isolated: boolean
  
  # Zone detail (if available)
  active_zones: [integer]           # Which zones in alarm
  isolated_zones: [integer]         # Which zones isolated
  
  # Timestamps
  last_fire_alarm: timestamp | null
  last_fault: timestamp | null
  last_state_change: timestamp
```

---

## Automation Responses

### Fire Alarm Triggered

When fire alarm activates, Gray Logic can trigger helpful responses:

```yaml
automation:
  id: "fire-alarm-response"
  name: "Fire Alarm Response"
  
  trigger:
    type: "device_state"
    device_id: "fire-panel-main"
    condition:
      property: "fire_active"
      operator: "eq"
      value: true
      
  actions:
    # Lighting
    - description: "All lights to 100%"
      target:
        type: "site"
      command: "turn_on"
      parameters:
        brightness: 100
        transition_ms: 0            # Immediate
        
    # Blinds (if safe to do so)
    - description: "Open all blinds"
      target:
        type: "site"
        domain: "blinds"
      command: "open"
      
    # Audio (pause any music)
    - description: "Mute audio"
      target:
        type: "site"
        domain: "audio"
      command: "mute"
      
    # Notifications
    - description: "Notify all users"
      type: "notification"
      severity: "critical"
      title: "🔥 FIRE ALARM"
      message: "Fire alarm activated. Evacuate immediately."
      recipients: ["all"]
      channels: ["push", "sms"]
      
    - description: "Notify facility manager"
      type: "notification"
      severity: "critical"
      recipients: ["facility_manager"]
      channels: ["push", "sms", "voice_call"]
      
  # Override normal mode behavior
  mode_override:
    ignore_current_mode: true       # Run regardless of Home/Away/etc.
```

### Fire Alarm Cleared

```yaml
automation:
  id: "fire-alarm-clear"
  name: "Fire Alarm Cleared"
  
  trigger:
    type: "device_state"
    device_id: "fire-panel-main"
    condition:
      property: "status"
      from: "fire"
      to: "normal"
      
  actions:
    - description: "Notify all clear"
      type: "notification"
      severity: "info"
      title: "Fire Alarm Cleared"
      message: "Fire alarm has been cleared and reset."
      recipients: ["all"]
      
    # Don't auto-restore lighting - leave that to occupants
    # Fire panel reset should be manual process
```

### Fault Notification

```yaml
automation:
  id: "fire-panel-fault"
  name: "Fire Panel Fault Alert"
  
  trigger:
    type: "device_state"
    device_id: "fire-panel-main"
    condition:
      property: "fault_active"
      operator: "eq"
      value: true
      
  actions:
    - description: "Alert maintenance"
      type: "notification"
      severity: "high"
      title: "⚠️ Fire Panel Fault"
      message: "Fire alarm panel reporting fault condition. Inspection required."
      recipients: ["facility_manager", "maintenance", "fire_contractor"]
      
    - description: "Log for compliance"
      type: "audit_log"
      category: "safety"
      retention: "permanent"
```

---

## UI Display

### Dashboard Widget

```yaml
dashboard_widget:
  type: "fire_status"
  device_id: "fire-panel-main"
  
  display:
    normal:
      icon: "fire-extinguisher"
      color: "green"
      text: "Fire System Normal"
      
    fire:
      icon: "fire"
      color: "red"
      text: "🔥 FIRE ALARM ACTIVE"
      animate: true
      audio_alert: true
      
    fault:
      icon: "alert-triangle"
      color: "amber"
      text: "⚠️ System Fault"
      
    isolate:
      icon: "alert-circle"
      color: "amber"
      text: "Zone(s) Isolated"
      
  # Show zone detail when in alarm
  show_zones: true
```

### Wall Panel Behavior

```yaml
wall_panel_fire_behavior:
  # On fire alarm
  fire_alarm:
    force_display: true             # Wake screen
    show_evacuation_info: true
    show_assembly_point: true
    show_exit_route: true           # If building map available
    
    # Don't allow dismissal
    lockout_navigation: false       # Still allow other controls
    
    # Audio alert (if panel has speaker)
    audio_alert:
      enabled: true
      sound: "fire_alarm"
      volume: 100
```

---

## Audit and Compliance

### Event Logging

All fire-related events are permanently logged:

```yaml
audit_log:
  category: "fire_safety"
  retention: "permanent"            # Never auto-delete
  
  events:
    - type: "fire_alarm_activated"
      timestamp: true
      zones: true
      
    - type: "fire_alarm_cleared"
      timestamp: true
      duration: true
      
    - type: "fault_detected"
      timestamp: true
      fault_type: true
      
    - type: "fault_cleared"
      timestamp: true
      
    - type: "zone_isolated"
      timestamp: true
      zone_id: true
      
    - type: "zone_restored"
      timestamp: true
      zone_id: true
```

### Compliance Reporting

```yaml
compliance_report:
  name: "Fire System Integration Log"
  schedule: "monthly"
  format: "pdf"
  
  contents:
    - section: "Event Summary"
      include:
        - alarm_count
        - fault_count
        - false_alarm_count
        
    - section: "Event Detail"
      include:
        - all_fire_events
        - all_fault_events
        
    - section: "System Status"
      include:
        - current_status
        - zones_status
        - communication_status
        
  recipients:
    - "compliance@company.com"
    - "fire_contractor@example.com"
```

---

## Testing and Commissioning

### Integration Test Procedure

1. **Coordinate with fire contractor** — Never test without their involvement
2. **Notify building occupants** — Planned test, no evacuation required
3. **Silence audible alarms** — Via fire panel, not Gray Logic
4. **Trigger test activation** — Via fire panel test mode
5. **Verify Gray Logic receives signal** — Check state change
6. **Verify automation triggers** — Lights, blinds, notifications
7. **Clear alarm at panel** — Verify clear signal received
8. **Test fault signal** — Disconnect cable to simulate fault
9. **Document results** — Record in commissioning log

### Commissioning Checklist

- [ ] Fire contractor approval obtained
- [ ] Auxiliary contacts identified on fire panel
- [ ] Wiring installed to interface module
- [ ] Fire signal polarity verified (N/O vs N/C)
- [ ] Fault signal polarity verified
- [ ] KNX/Modbus addresses configured
- [ ] Device added to Gray Logic
- [ ] State correctly reflects panel status
- [ ] Fire alarm automation tested
- [ ] Fault notification tested
- [ ] All occupants notified of integration
- [ ] Audit logging verified
- [ ] Documentation updated

---

## Limitations and Disclaimers

### What This Integration Does NOT Provide

- Life safety functionality
- Fire detection
- Alarm notification (sounders/beacons)
- Evacuation management
- Fire suppression control
- Compliance with fire regulations
- Replacement for certified fire systems

### Responsibility

- **Fire detection and alarm**: Fire alarm system and contractor
- **Evacuation**: Building management and fire procedures
- **System maintenance**: Certified fire contractor
- **Gray Logic integration**: Convenience and awareness only

### Insurance and Certification

This integration:
- Does not affect fire system certification
- Should not affect insurance (verify with insurer)
- Does not require fire system recertification
- Uses only auxiliary/monitoring outputs

---

## Related Documents

- [Principles](../overview/principles.md) — Safety-first philosophy
- [Entities](../data-model/entities.md) — Fire input device type
- [Access Control Integration](access-control.md) — Emergency egress
- [CCTV Integration](cctv.md) — Evacuation camera views
- [Lighting Domain](../domains/lighting.md) — Emergency lighting monitoring

