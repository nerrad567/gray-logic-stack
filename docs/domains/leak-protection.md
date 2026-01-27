---
title: Leak Protection Domain Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/core-internals.md
  - data-model/entities.md
  - overview/principles.md
---

# Leak Protection Domain Specification

This document specifies how Gray Logic detects water leaks and executes automatic protective responses to prevent water damage.

---

## Overview

### Philosophy

Leak protection is a **safety-critical system** — fast detection and response can prevent thousands in damage:

| Without Protection | With Gray Logic |
|-------------------|-----------------|
| Leak runs for hours | Detected in seconds |
| Floors/ceilings damaged | Valves shut automatically |
| Discovered by accident | Immediate notification |
| Expensive repairs | Minimal damage |

### Design Principles

1. **Speed matters** — Every minute of delay = more damage
2. **Redundancy** — Multiple sensors, multiple detection methods
3. **Fail-safe** — Valve failure defaults to closed (if safe)
4. **Confirmation** — Avoid false positives, but err on side of safety
5. **Manual override** — Users can always restore water

### Safety Priority

```
Detection
    ↓
Confirmation (if configured)
    ↓
Automatic Response
    ├── Sound alarm
    ├── Close valves
    ├── Stop pumps
    └── Notify immediately
    ↓
Manual Investigation Required
```

---

## Components

### Leak Sensors

Point sensors that detect water presence.

```yaml
LeakSensor:
  id: uuid
  name: string                        # "Under Kitchen Sink"
  
  # Location
  room_id: uuid
  location_description: string        # "Below dishwasher connection"
  
  # Hardware
  type: enum                          # See sensor types
  protocol: string                    # "zigbee", "zwave", "wired"
  
  # Configuration
  config:
    sensitivity: "high" | "normal" | "low"
    probe_count: integer              # Number of detection probes
    
  # Zone association
  zone_id: uuid                       # Leak protection zone
  
  # Current state
  state:
    leak_detected: boolean
    last_triggered: timestamp | null
    battery_percent: integer | null
    tamper: boolean | null
```

**Sensor Types:**

| Type | Description | Response Time |
|------|-------------|---------------|
| `point` | Single point detection | Immediate |
| `cable` | Water-sensing cable | Immediate |
| `rope` | Rope sensor (long runs) | Immediate |
| `probe` | Multiple probes | Immediate |
| `flow_anomaly` | Flow-based detection | Seconds-minutes |

### Water Shutoff Valve

Automatic valve that stops water flow.

```yaml
ShutoffValve:
  id: uuid
  name: string                        # "Main Water Shutoff"
  
  # Location
  location: string                    # "Water meter entry"
  
  # Hardware
  type: enum                          # motorized, solenoid
  manufacturer: string
  model: string
  
  # Connection
  connection:
    type: "zwave" | "zigbee" | "relay" | "modbus"
    device_id: string
    
  # Configuration
  config:
    normal_state: "open" | "closed"   # Normal operating state
    fail_state: "open" | "closed"     # State on power/comm failure
    travel_time_seconds: integer      # Time to fully open/close
    
  # Zone association
  zone_id: uuid                       # Leak protection zone this protects
  
  # Current state
  state:
    position: "open" | "closed" | "moving" | "unknown"
    last_operated: timestamp | null
    fault: boolean
```

### Leak Protection Zone

Logical grouping of sensors, valves, and equipment.

```yaml
LeakProtectionZone:
  id: uuid
  name: string                        # "Plant Room", "Kitchen"
  
  # Associated areas
  rooms: [uuid]
  
  # Sensors in this zone
  sensors: [uuid]
  
  # Protective actions
  actions:
    valves: [uuid]                    # Valves to close
    pumps: [uuid]                     # Pumps to stop
    
  # Configuration
  config:
    confirmation_required: boolean    # Require 2+ sensors
    confirmation_sensors: integer     # How many must trigger
    confirmation_timeout_seconds: integer  # Max time to wait
    
    # Response settings
    auto_shutoff: boolean             # Automatically close valves
    auto_stop_pumps: boolean          # Automatically stop pumps
    alarm_siren: boolean              # Sound audible alarm
    
  # Current state
  state:
    status: "normal" | "leak_detected" | "shutoff_active"
    triggered_sensors: [uuid]
    valves_closed: [uuid]
    pumps_stopped: [uuid]
```

---

## Detection Methods

### Point Sensor Detection

Most common — sensors at specific risk points:

```yaml
PointDetection:
  sensors:
    - location: "Under kitchen sink"
      risk: "Dishwasher, disposal connections"
      
    - location: "Behind washing machine"
      risk: "Hose connections"
      
    - location: "Water heater base"
      risk: "Tank leak, relief valve"
      
    - location: "HVAC condensate pan"
      risk: "Drain blockage"
      
    - location: "Under bathroom vanity"
      risk: "Sink connections"
      
  response:
    single_sensor:
      action: "shutoff"               # Immediately act
    multiple_sensors:
      action: "shutoff"               # Also immediately act
```

### Flow-Based Detection

Detect leaks by abnormal water usage:

```yaml
FlowDetection:
  flow_meter_id: "flow-meter-main"
  
  rules:
    # Continuous flow when no demand
    - name: "no_demand_flow"
      condition: "flow_when_idle"
      threshold_lpm: 2
      duration_minutes: 15
      action: "alert"
      
    # High flow for extended period
    - name: "extended_high_flow"
      condition: "flow_above"
      threshold_lpm: 50
      duration_minutes: 60
      action: "alert_and_shutoff"
      
    # Any flow when away mode
    - name: "flow_while_away"
      condition: "any_flow"
      mode: "away"
      duration_minutes: 30
      action: "alert"
```

### Multi-Sensor Confirmation

Reduce false positives in critical areas:

```yaml
ConfirmationLogic:
  zone_id: "zone-plant-room"
  
  # Require 2 of 4 sensors to trigger shutoff
  confirmation:
    required_sensors: 2
    total_sensors: 4
    timeout_seconds: 30
    
  # First sensor triggers
  on_first_sensor:
    - alert: "warning"
    - message: "Possible leak detected in plant room"
    
  # Confirmation (2+ sensors)
  on_confirmation:
    - alert: "critical"
    - close_valves: true
    - stop_pumps: true
    - message: "LEAK CONFIRMED - Plant room shutoff activated"
    
  # Timeout (only 1 sensor)
  on_timeout:
    - alert: "warning"
    - message: "Single sensor triggered - manual investigation required"
```

---

## Response Actions

### Automatic Shutoff

```yaml
AutoShutoff:
  trigger:
    zone_id: "zone-plant-room"
    condition: "leak_confirmed"
    
  actions:
    # Close all zone valves
    - type: "close_valve"
      valves: ["valve-mains", "valve-plant-room"]
      
    # Stop all zone pumps
    - type: "stop_pump"
      pumps: ["pump-chw-1", "pump-chw-2", "pump-heating"]
      
    # Close isolation valves
    - type: "close_valve"
      valves: ["valve-chw-supply", "valve-chw-return"]
      
    # Sound local alarm
    - type: "alarm"
      device_id: "siren-plant-room"
      pattern: "leak_alert"
      
    # Log event
    - type: "log"
      severity: "critical"
      message: "Automatic shutoff activated - plant room leak"
```

### Notification Cascade

```yaml
Notifications:
  leak_detected:
    priority: "critical"
    
    # Immediate
    - channel: "push"
      recipients: ["all_users"]
      message: "WATER LEAK DETECTED in ${zone_name}"
      
    - channel: "sms"
      recipients: ["owner", "property_manager"]
      message: "ALERT: Water leak at ${site_name}. Automatic shutoff activated."
      
    # If not acknowledged in 5 minutes
    - delay_minutes: 5
      condition: "not_acknowledged"
      channel: "phone_call"
      recipients: ["owner"]
      message: "Emergency: Water leak detected. Press 1 to acknowledge."
      
    # Escalation
    - delay_minutes: 15
      condition: "not_acknowledged"
      channel: "sms"
      recipients: ["emergency_contact"]
```

### Integration with Other Domains

```yaml
LeakResponse:
  zone_id: "zone-kitchen"
  
  actions:
    # Core leak response
    - domain: "leak_protection"
      action: "close_valves"
      
    # Lighting - turn on for visibility
    - domain: "lighting"
      scope: "room-kitchen"
      action: "on"
      parameters:
        brightness: 100
        
    # Audio - announce
    - domain: "audio"
      action: "announce"
      parameters:
        message: "Water leak detected in kitchen. Water supply shut off."
        priority: "critical"
        zones: "all"
        
    # HVAC - stop if water-related
    - domain: "climate"
      scope: "zone-kitchen"
      action: "stop"
```

---

## Recovery Procedures

### Manual Reset

After a leak event, water must be manually restored:

```yaml
RecoveryProcedure:
  # Cannot auto-restore water
  auto_restore: false
  
  # User must:
  steps:
    1: "Investigate and fix leak source"
    2: "Dry affected areas"
    3: "Reset sensors if needed"
    4: "Acknowledge alarm in app/UI"
    5: "Manually open shutoff valve (physical or app)"
    
  # App provides restore option
  restore_command:
    requires_pin: true
    confirmation: "Confirm you have fixed the leak and want to restore water?"
    
  # Log restore action
  logging:
    user_id: true
    timestamp: true
    reason_required: false
```

### Sensor Reset

```yaml
SensorReset:
  # Some sensors auto-clear when dry
  auto_clear:
    - type: "resistance"
      clear_when: "dry"
      
  # Others need manual reset
  manual_reset:
    - type: "latching"
      reset_method: "button"
      
  # System clears alert when all sensors clear
  zone_clear:
    condition: "all_sensors_clear"
    actions:
      - clear_zone_alert: true
      - log_event: "Zone returned to normal"
```

---

## MQTT Topics

### Sensor State

```yaml
topic: graylogic/leak/sensor/{sensor_id}/state
payload:
  sensor_id: "sensor-kitchen-sink"
  timestamp: "2026-01-12T14:30:00Z"
  state:
    leak_detected: true
    battery_percent: 85
```

### Zone State

```yaml
topic: graylogic/leak/zone/{zone_id}/state
payload:
  zone_id: "zone-plant-room"
  timestamp: "2026-01-12T14:30:00Z"
  state:
    status: "leak_detected"
    triggered_sensors: ["sensor-1", "sensor-3"]
    valves_closed: ["valve-mains"]
    pumps_stopped: ["pump-chw-1"]
```

### Commands

```yaml
# Manual valve control
topic: graylogic/leak/command
payload:
  target: "valve-mains"
  command: "open"
  parameters:
    user_pin: "1234"
    reason: "Leak repaired"
  request_id: "req-12345"

# Acknowledge alert
topic: graylogic/leak/command
payload:
  target: "zone-plant-room"
  command: "acknowledge"
  parameters:
    user_id: "user-123"
  request_id: "req-12346"
```

---

## PHM Integration

### PHM Value for Leak Protection

| Equipment | PHM Value | Key Indicators |
|-----------|-----------|----------------|
| Shutoff valves | ★★★★☆ | Operation count, travel time |
| Leak sensors | ★★★☆☆ | Battery, false alarm rate |
| Flow meters | ★★★☆☆ | Accuracy, communication |

### PHM Configuration

```yaml
phm_leak:
  devices:
    - device_id: "valve-mains"
      type: "shutoff_valve"
      parameters:
        - name: "travel_time_seconds"
          baseline_method: "initial_calibration"
          deviation_threshold_percent: 50
          alert: "Valve slow to operate - may be stuck"
          
        - name: "operation_count"
          threshold: 5000             # Typical valve life
          alert_at_percent: 80
          alert: "Valve approaching end of life"
          
    - device_id: "sensor-kitchen-sink"
      type: "leak_sensor"
      parameters:
        - name: "battery_percent"
          threshold: 20
          alert: "Leak sensor battery low"
          
        - name: "false_alarm_count"
          baseline_method: "yearly_count"
          threshold: 3
          alert: "Possible sensor issue - multiple false alarms"
```

### Regular Testing

```yaml
TestSchedule:
  # Monthly valve exercise
  valve_exercise:
    schedule: "monthly"
    day: 1
    time: "03:00"
    action:
      - close_valve: true
      - wait_seconds: 5
      - open_valve: true
      - verify_operation: true
    notify_before: true
    
  # Sensor test reminder
  sensor_test:
    schedule: "quarterly"
    action: "notify"
    message: "Reminder: Test leak sensors by dripping water on probes"
```

---

## Commercial Considerations

### Multi-Zone Protection

```yaml
CommercialZones:
  zones:
    - id: "zone-server-room"
      priority: "critical"
      sensors: 6
      confirmation_required: 1        # Any sensor triggers
      response_time_seconds: 0        # Immediate
      actions:
        - close_hvac_condensate_valves
        - alert_it_team
        
    - id: "zone-restrooms"
      priority: "high"
      sensors: 4
      confirmation_required: 2
      
    - id: "zone-plant-room"
      priority: "critical"
      sensors: 8
      confirmation_required: 2
      actions:
        - close_mains
        - stop_all_pumps
        - close_all_isolation_valves
```

### Integration with BMS

```yaml
BMSIntegration:
  # Report to building management
  report_events:
    - event: "leak_detected"
      protocol: "bacnet"
      object: "BinaryValue:100"
      
    - event: "shutoff_active"
      protocol: "bacnet"
      object: "BinaryValue:101"
      
  # Accept overrides from BMS
  accept_commands:
    - command: "valve_control"
      authorization: "bms_supervisor"
```

---

## Configuration Examples

### Residential: Basic

```yaml
leak_protection:
  zones:
    - id: "zone-whole-house"
      name: "Whole House"
      
      sensors:
        - id: "sensor-kitchen"
          location: "Under kitchen sink"
        - id: "sensor-laundry"
          location: "Behind washing machine"
        - id: "sensor-water-heater"
          location: "Water heater base"
        - id: "sensor-bathroom"
          location: "Under bathroom vanity"
          
      valves:
        - id: "valve-mains"
          location: "Main water entry"
          type: "motorized"
          
      config:
        auto_shutoff: true
        confirmation_required: false  # Single sensor triggers
```

### Residential: Advanced

```yaml
leak_protection:
  zones:
    - id: "zone-kitchen"
      sensors: ["sensor-sink", "sensor-dishwasher", "sensor-fridge"]
      valves: ["valve-kitchen-supply"]
      
    - id: "zone-laundry"
      sensors: ["sensor-washer", "sensor-drain"]
      valves: ["valve-laundry-supply"]
      
    - id: "zone-hvac"
      sensors: ["sensor-condensate", "sensor-pan"]
      actions:
        hvac_shutoff: true            # Stop HVAC on leak
        
  main_shutoff:
    valve_id: "valve-mains"
    trigger_on_any_zone: true
    
  flow_detection:
    enabled: true
    flow_meter: "flow-meter-main"
    away_mode_monitoring: true
```

### Commercial: Plant Room

```yaml
leak_protection:
  zones:
    - id: "zone-plant-room"
      name: "Plant Room"
      
      sensors:
        - id: "sensor-chiller-1"
          location: "Under chiller 1"
        - id: "sensor-chiller-2"
          location: "Under chiller 2"
        - id: "sensor-pump-bay"
          location: "Pump bay floor"
        - id: "sensor-expansion"
          location: "Expansion tank area"
        - id: "cable-perimeter"
          type: "cable"
          location: "Perimeter of room"
          
      config:
        confirmation_required: true
        confirmation_sensors: 2
        confirmation_timeout_seconds: 30
        
      valves:
        - id: "valve-chw-supply"
        - id: "valve-chw-return"
        - id: "valve-heating-supply"
        - id: "valve-heating-return"
        - id: "valve-makeup-water"
        
      pumps:
        - id: "pump-chw-primary"
        - id: "pump-chw-secondary"
        - id: "pump-heating"
        
      notifications:
        immediate: ["facility_manager", "on_call"]
        escalation_minutes: 10
        escalate_to: ["building_owner"]
```

---

## Best Practices

### Do's

1. **Sensor placement** — Under all water connections, appliances
2. **Test regularly** — Monthly valve exercise, quarterly sensor test
3. **Battery monitoring** — Replace before they die
4. **Fast notification** — Critical alerts to phones immediately
5. **Document locations** — Map all sensors and valves
6. **Redundancy** — Multiple sensors in critical areas

### Don'ts

1. **Don't ignore low battery** — Dead sensor = no protection
2. **Don't disable auto-shutoff** — The whole point is automation
3. **Don't skip testing** — Stuck valve when needed is disaster
4. **Don't rely on single sensor** — Critical areas need redundancy
5. **Don't forget the valve** — Sensors without shutoff just watch damage happen

---

## Related Documents

- [Principles](../overview/principles.md) — Safety-first philosophy
- [Plant Domain](plant.md) — Pump and valve control
- [Irrigation Domain](irrigation.md) — Outdoor water management
- [Water Management](water-management.md) — Broader water infrastructure
- [PHM Specification](../intelligence/phm.md) — Equipment health monitoring
- [Automation Specification](../automation/automation.md) — Event-driven responses
