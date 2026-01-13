---
title: OCPP Protocol Specification
version: 1.0.0
status: active
last_updated: 2026-01-13
depends_on:
  - architecture/system-overview.md
  - architecture/energy-model.md
  - domains/energy.md
---

# OCPP Protocol Specification

This document specifies Gray Logic's integration with EV chargers via OCPP (Open Charge Point Protocol) — enabling smart charging, load balancing, and energy management.

---

## Overview

### What is OCPP?

OCPP (Open Charge Point Protocol) is an open standard for communication between EV chargers and central management systems.

| Version | Status | Features |
|---------|--------|----------|
| **OCPP 1.6** | Widely deployed | JSON/SOAP, basic smart charging |
| **OCPP 2.0.1** | Emerging | Security, ISO 15118, device management |

### Why OCPP Integration?

| Without Gray Logic | With Gray Logic |
|--------------------|-----------------|
| Charger operates independently | Integrated with home energy |
| No solar optimization | Charge when solar is available |
| Fixed schedule | Dynamic load management |
| Separate app | Unified control interface |

### Use Cases

1. **Solar optimization** — Charge EV when solar production exceeds consumption
2. **Load management** — Limit charging to avoid circuit overload
3. **Time-of-use** — Charge during off-peak electricity rates
4. **Grid response** — Reduce charging during peak demand
5. **Monitoring** — Track charging sessions, energy usage

---

## Architecture

### Integration Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      GRAY LOGIC CORE                             │
│                                                                  │
│  ┌────────────────┐    ┌─────────────────────────────────────┐  │
│  │ Energy Manager │────│          OCPP Bridge                 │  │
│  │                │    │                                     │  │
│  │ • Solar data   │    │ • WebSocket server (OCPP-J)         │  │
│  │ • Grid import  │    │ • Message handling                  │  │
│  │ • Load limits  │    │ • Charger state management          │  │
│  └────────────────┘    └──────────────┬──────────────────────┘  │
│                                       │                          │
└───────────────────────────────────────┼──────────────────────────┘
                                        │
                            WebSocket (OCPP-J)
                                        │
              ┌─────────────────────────┼─────────────────────────┐
              │                         │                         │
              ▼                         ▼                         ▼
       ┌──────────────┐         ┌──────────────┐         ┌──────────────┐
       │  EV Charger  │         │  EV Charger  │         │  EV Charger  │
       │  (Wallbox)   │         │  (Easee)     │         │  (OpenEVSE)  │
       └──────────────┘         └──────────────┘         └──────────────┘
```

### OCPP Bridge

Gray Logic acts as a Central System (CSMS) to which chargers connect:

```yaml
ocpp_bridge:
  role: "central_system"
  
  # WebSocket server
  server:
    port: 9000
    path: "/ocpp/{charger_id}"
    protocol: "ocpp1.6"             # or "ocpp2.0.1"
    
  # Security
  security:
    tls: true
    auth_method: "basic"            # or "certificate" for OCPP 2.0
    
  # Connection handling
  connections:
    heartbeat_interval_seconds: 300
    reconnect_timeout_seconds: 60
```

---

## OCPP 1.6 Support

### Message Types

```yaml
ocpp_1_6_messages:
  # Core profile (required)
  core:
    # Charger → Central System
    from_charger:
      - Authorize
      - BootNotification
      - Heartbeat
      - MeterValues
      - StartTransaction
      - StatusNotification
      - StopTransaction
      
    # Central System → Charger
    to_charger:
      - ChangeAvailability
      - ChangeConfiguration
      - ClearCache
      - GetConfiguration
      - RemoteStartTransaction
      - RemoteStopTransaction
      - Reset
      - UnlockConnector
      
  # Smart charging profile
  smart_charging:
    to_charger:
      - ClearChargingProfile
      - GetCompositeSchedule
      - SetChargingProfile
      
  # Firmware management
  firmware:
    from_charger:
      - DiagnosticsStatusNotification
      - FirmwareStatusNotification
    to_charger:
      - GetDiagnostics
      - UpdateFirmware
```

### BootNotification

Charger registration on connection:

```yaml
boot_notification:
  # Request (from charger)
  request:
    chargePointVendor: "Wallbox"
    chargePointModel: "Pulsar Plus"
    chargePointSerialNumber: "12345"
    firmwareVersion: "5.2.1"
    
  # Response (from Gray Logic)
  response:
    status: "Accepted"              # or "Pending" | "Rejected"
    currentTime: "2026-01-13T10:00:00Z"
    interval: 300                   # Heartbeat interval (seconds)
```

### StatusNotification

Charger status updates:

```yaml
status_notification:
  # Connector states
  states:
    - Available                     # Ready to charge
    - Preparing                     # Connector plugged in
    - Charging                      # Actively charging
    - SuspendedEV                   # Paused by EV
    - SuspendedEVSE                 # Paused by charger/CSMS
    - Finishing                     # Charging complete
    - Reserved                      # Reserved for user
    - Unavailable                   # Not available
    - Faulted                       # Error state
    
  # Error codes
  errors:
    - NoError
    - ConnectorLockFailure
    - EVCommunicationError
    - GroundFailure
    - HighTemperature
    - InternalError
    - LocalListConflict
    - OtherError
    - OverCurrentFailure
    - OverVoltage
    - PowerMeterFailure
    - PowerSwitchFailure
    - ReaderFailure
    - ResetFailure
    - UnderVoltage
    - WeakSignal
```

### Transactions

```yaml
transactions:
  # Start transaction
  start:
    request:
      connectorId: 1
      idTag: "RFID123"
      meterStart: 12345             # Wh
      timestamp: "2026-01-13T10:00:00Z"
      
    response:
      idTagInfo:
        status: "Accepted"
      transactionId: 1001
      
  # Stop transaction
  stop:
    request:
      transactionId: 1001
      meterStop: 19345              # Wh
      timestamp: "2026-01-13T12:00:00Z"
      reason: "Local"               # or "Remote" | "EVDisconnected"
      
    response:
      idTagInfo:
        status: "Accepted"
```

### MeterValues

Energy and power readings:

```yaml
meter_values:
  # Reading types
  measurands:
    - Energy.Active.Import.Register  # Total energy (Wh)
    - Power.Active.Import            # Current power (W)
    - Current.Import                 # Current (A)
    - Voltage                        # Voltage (V)
    - Temperature                    # Temperature (°C)
    - SoC                            # State of charge (%)
    
  # Sample configuration
  configuration:
    interval_seconds: 60
    measurands:
      - "Energy.Active.Import.Register"
      - "Power.Active.Import"
```

---

## Smart Charging

### Charging Profiles

Control charging power dynamically:

```yaml
charging_profile:
  # Profile structure
  profile:
    chargingProfileId: 1
    stackLevel: 0                   # Priority (higher = override)
    chargingProfilePurpose: "TxDefaultProfile"
    chargingProfileKind: "Absolute"
    
    # Schedule
    chargingSchedule:
      duration: 86400               # 24 hours
      startSchedule: "2026-01-13T00:00:00Z"
      chargingRateUnit: "W"         # or "A"
      
      # Power limits over time
      chargingSchedulePeriod:
        - startPeriod: 0            # Midnight
          limit: 3000               # 3 kW
          
        - startPeriod: 25200        # 07:00
          limit: 0                  # Stop (peak)
          
        - startPeriod: 36000        # 10:00
          limit: 7000               # 7 kW (solar available)
          
        - startPeriod: 57600        # 16:00
          limit: 0                  # Stop (peak)
          
        - startPeriod: 75600        # 21:00
          limit: 7000               # 7 kW (off-peak)
```

### Profile Types

```yaml
profile_purposes:
  ChargePointMaxProfile:
    description: "Maximum power limit for entire charger"
    use_case: "Circuit protection"
    stack_level: 0
    
  TxDefaultProfile:
    description: "Default profile for all transactions"
    use_case: "Time-of-use optimization"
    stack_level: 1
    
  TxProfile:
    description: "Profile for specific transaction"
    use_case: "Solar surplus charging"
    stack_level: 2
```

### Dynamic Load Balancing

```yaml
load_balancing:
  # Circuit limit
  circuit:
    max_current_a: 32
    phases: 3
    
  # Connected chargers
  chargers:
    - id: "charger-garage"
      max_current_a: 32
      
    - id: "charger-driveway"
      max_current_a: 32
      
  # Algorithm
  algorithm: "proportional"         # or "priority" | "round_robin"
  
  # Priority (if algorithm = "priority")
  priority:
    - "charger-garage"              # Higher priority
    - "charger-driveway"
    
  # Rebalance trigger
  rebalance:
    on_charger_change: true
    on_power_change: true
    min_interval_seconds: 10
```

---

## Energy Integration

### Solar Optimization

```yaml
solar_charging:
  enabled: true
  
  # Surplus calculation
  surplus:
    source: "energy_manager"
    formula: "solar_production - house_consumption"
    min_surplus_w: 1400             # Min to start charging (~6A single phase)
    
  # Behavior
  behavior:
    mode: "solar_only"              # or "solar_priority" | "mixed"
    
    solar_only:
      charge_only_from_solar: true
      stop_if_no_surplus: true
      
    solar_priority:
      prefer_solar: true
      allow_grid_minimum: 1400      # Min charge rate if no solar
      
    mixed:
      use_available_solar: true
      top_up_from_grid: true
      max_grid_w: 3000
      
  # Response time
  update_interval_seconds: 60
  
  # Smoothing
  smoothing:
    enabled: true
    window_seconds: 300             # 5-minute average
```

### Time-of-Use Optimization

```yaml
tou_charging:
  enabled: true
  
  # Rate periods
  periods:
    - name: "off_peak"
      start: "00:00"
      end: "07:00"
      rate_per_kwh: 0.10
      priority: 1                   # Prefer this period
      
    - name: "peak"
      start: "16:00"
      end: "20:00"
      rate_per_kwh: 0.35
      priority: 3                   # Avoid this period
      
    - name: "standard"
      start: null                   # All other times
      end: null
      rate_per_kwh: 0.20
      priority: 2
      
  # Behavior
  behavior:
    avoid_peak: true                # Don't charge during peak
    prefer_off_peak: true           # Charge during off-peak if possible
    
  # Target
  target:
    mode: "time"                    # or "soc" | "energy"
    ready_by: "07:00"               # Car ready by this time
```

### Grid Response

```yaml
grid_response:
  enabled: true
  
  # Demand response
  demand_response:
    # External signal source
    source: "grid_operator_api"     # or "frequency" | "manual"
    
    # Response levels
    levels:
      normal:
        limit_percent: 100
        
      moderate:
        limit_percent: 50
        
      high:
        limit_percent: 0            # Stop charging
        
  # Frequency response (future)
  frequency_response:
    enabled: false
    low_frequency_hz: 49.5
    high_frequency_hz: 50.5
```

---

## Charger Management

### Charger State Model

```yaml
charger_state:
  id: "charger-garage"
  
  # Connection
  connection:
    status: "connected"             # connected | disconnected
    last_heartbeat: "2026-01-13T10:30:00Z"
    protocol: "ocpp1.6"
    
  # Connector (most chargers have 1)
  connector:
    id: 1
    status: "Charging"
    error: "NoError"
    
  # Current transaction
  transaction:
    id: 1001
    started: "2026-01-13T09:00:00Z"
    meter_start_wh: 12345
    current_meter_wh: 15678
    energy_delivered_wh: 3333
    
  # Current power
  power:
    current_w: 7200
    current_a: 31.3
    voltage_v: 230
    
  # Limits
  limits:
    max_power_w: 7400
    current_limit_w: 7400
    current_limit_source: "solar"
```

### Remote Control

```yaml
remote_control:
  # Start charging
  start:
    command: "RemoteStartTransaction"
    parameters:
      connectorId: 1
      idTag: "GrayLogic"
      
  # Stop charging
  stop:
    command: "RemoteStopTransaction"
    parameters:
      transactionId: 1001
      
  # Set power limit
  set_limit:
    command: "SetChargingProfile"
    parameters:
      connectorId: 1
      chargingProfile: {...}
      
  # Unlock connector
  unlock:
    command: "UnlockConnector"
    parameters:
      connectorId: 1
```

---

## MQTT Integration

### Topics

```yaml
mqtt_topics:
  # Charger state
  state: "graylogic/charger/{charger_id}/state"
  
  # Connector state
  connector: "graylogic/charger/{charger_id}/connector/{connector_id}"
  
  # Transaction
  transaction: "graylogic/charger/{charger_id}/transaction"
  
  # Commands
  command: "graylogic/charger/{charger_id}/command"
  
  # Meter values
  meter: "graylogic/charger/{charger_id}/meter"
```

### State Payload

```json
{
  "charger_id": "charger-garage",
  "connection_status": "connected",
  "connector_status": "Charging",
  "power_w": 7200,
  "energy_session_wh": 3333,
  "limit_w": 7400,
  "limit_source": "solar",
  "timestamp": "2026-01-13T10:30:00Z"
}
```

### Command Payload

```json
{
  "action": "set_limit",
  "limit_w": 3000,
  "reason": "Load balancing",
  "duration_seconds": null
}
```

---

## API Endpoints

### Charger List

```http
GET /api/v1/chargers
```

**Response:**
```json
{
  "chargers": [
    {
      "id": "charger-garage",
      "name": "Garage Charger",
      "model": "Wallbox Pulsar Plus",
      "status": "connected",
      "connector_status": "Charging"
    }
  ]
}
```

### Charger Detail

```http
GET /api/v1/chargers/{charger_id}
```

**Response:**
```json
{
  "id": "charger-garage",
  "name": "Garage Charger",
  "vendor": "Wallbox",
  "model": "Pulsar Plus",
  "firmware": "5.2.1",
  "connection": {
    "status": "connected",
    "protocol": "ocpp1.6",
    "last_heartbeat": "2026-01-13T10:30:00Z"
  },
  "connector": {
    "id": 1,
    "status": "Charging",
    "error": "NoError"
  },
  "transaction": {
    "id": 1001,
    "started": "2026-01-13T09:00:00Z",
    "energy_wh": 3333,
    "power_w": 7200
  },
  "limits": {
    "max_power_w": 7400,
    "current_limit_w": 7400,
    "limit_source": "solar"
  }
}
```

### Start Charging

```http
POST /api/v1/chargers/{charger_id}/start
```

### Stop Charging

```http
POST /api/v1/chargers/{charger_id}/stop
```

### Set Power Limit

```http
POST /api/v1/chargers/{charger_id}/limit
```

**Request:**
```json
{
  "limit_w": 3000,
  "reason": "Load balancing"
}
```

### Charging History

```http
GET /api/v1/chargers/{charger_id}/sessions
```

**Response:**
```json
{
  "sessions": [
    {
      "id": 1001,
      "started": "2026-01-13T09:00:00Z",
      "ended": "2026-01-13T12:00:00Z",
      "energy_wh": 21000,
      "cost": 2.10,
      "solar_percent": 65
    }
  ]
}
```

---

## Configuration

### OCPP Bridge Configuration

```yaml
# /etc/graylogic/ocpp.yaml
ocpp:
  enabled: true
  
  # Server
  server:
    port: 9000
    tls:
      enabled: true
      cert_file: "/etc/graylogic/certs/ocpp.crt"
      key_file: "/etc/graylogic/certs/ocpp.key"
      
  # Protocol
  protocol:
    version: "1.6"                  # or "2.0.1"
    heartbeat_interval: 300
    
  # Authentication
  auth:
    method: "basic"                 # or "certificate"
    credentials:
      - charger_id: "charger-garage"
        password_env: "OCPP_CHARGER_GARAGE_PW"
        
  # Charger configuration
  chargers:
    - id: "charger-garage"
      name: "Garage Charger"
      location: "garage"
      max_power_w: 7400
      phases: 1
      
    - id: "charger-driveway"
      name: "Driveway Charger"
      location: "driveway"
      max_power_w: 22000
      phases: 3
```

### Energy Integration Configuration

```yaml
# Energy integration settings
energy:
  # Solar optimization
  solar_charging:
    enabled: true
    mode: "solar_priority"
    min_surplus_w: 1400
    update_interval_seconds: 60
    
  # Load balancing
  load_balancing:
    enabled: true
    circuit_limit_a: 63
    algorithm: "proportional"
    
  # Time-of-use
  tou:
    enabled: true
    avoid_peak: true
    ready_by: "07:00"
```

---

## Commissioning

### Setup Checklist

```yaml
commissioning:
  # Gray Logic setup
  gray_logic:
    - [ ] OCPP bridge enabled
    - [ ] TLS certificates configured
    - [ ] Charger credentials set
    
  # Charger setup
  charger:
    - [ ] OCPP enabled in charger settings
    - [ ] Central System URL set (wss://graylogic.local:9000/ocpp/{id})
    - [ ] Credentials configured
    - [ ] Charger rebooted
    
  # Verification
  verification:
    - [ ] Charger connects and sends BootNotification
    - [ ] Status updates received
    - [ ] Remote start/stop works
    - [ ] Power limits applied correctly
    - [ ] Meter values received
    
  # Energy integration
  energy:
    - [ ] Solar data flowing
    - [ ] Load balancing tested
    - [ ] TOU schedule verified
```

### Supported Chargers

```yaml
tested_chargers:
  # Confirmed working
  confirmed:
    - vendor: "Wallbox"
      models: ["Pulsar Plus", "Commander 2"]
      ocpp: "1.6"
      
    - vendor: "Easee"
      models: ["Home", "Charge"]
      ocpp: "1.6"
      note: "Requires Easee OCPP license"
      
    - vendor: "OpenEVSE"
      models: ["All"]
      ocpp: "1.6"
      
    - vendor: "go-e"
      models: ["Charger Gemini", "Charger HOME+"]
      ocpp: "1.6"
      
  # Should work (OCPP compliant)
  expected:
    - vendor: "ABB"
    - vendor: "Schneider Electric"
    - vendor: "Siemens"
    - vendor: "Delta"
```

---

## Troubleshooting

### Common Issues

```yaml
troubleshooting:
  connection_refused:
    symptom: "Charger won't connect"
    checks:
      - "Verify URL in charger settings"
      - "Check TLS certificate validity"
      - "Confirm port 9000 is accessible"
      - "Check firewall rules"
      
  authentication_failed:
    symptom: "Charger connects then disconnects"
    checks:
      - "Verify charger ID matches configuration"
      - "Check password/credentials"
      
  charging_won't_start:
    symptom: "RemoteStartTransaction fails"
    checks:
      - "Connector is in Available state"
      - "Vehicle is plugged in"
      - "No error conditions"
      
  power_limit_not_applied:
    symptom: "Charger ignores SetChargingProfile"
    checks:
      - "Charger supports Smart Charging profile"
      - "Profile format is correct"
      - "Stack level is appropriate"
```

---

## Related Documents

- [Energy Model](../architecture/energy-model.md) — Energy flow architecture
- [Energy Domain](../domains/energy.md) — Energy management
- [Weather Integration](../intelligence/weather.md) — Solar forecasting
- [System Overview](../architecture/system-overview.md) — Overall architecture
