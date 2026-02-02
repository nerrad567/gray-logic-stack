---
title: Climate Control Domain
version: "1.0"
status: draft
dependencies:
  - docs/overview/principles.md
  - docs/architecture/system-overview.md
---

# Climate Control Domain

## Overview

Climate control encompasses heating, cooling, and ventilation systems. This document defines Gray Logic Core's architectural approach to climate control, emphasizing the **offline-first** principle: the building must continue to heat and cool even when GLCore is completely offline.

## Core Principle: GLCore as Overlay

```
┌─────────────────────────────────────────────────────────────┐
│                     GRAY LOGIC CORE                         │
│         Observer + Override + Intelligence Layer            │
│   (Can be offline — building climate still functions)       │
└─────────────────────────────┬───────────────────────────────┘
                              │ observe / override setpoints
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    KNX BUS LAYER                            │
│   Thermostats ←──PID──→ Actuators (always running)          │
│   FCUs ←────────────→ Valves (always controllable)          │
│   Plant equipment ←──→ Controls (independent operation)     │
└─────────────────────────────────────────────────────────────┘
```

**Key rule**: GLCore is **never** the primary controller for climate. The PID control loop runs on KNX hardware. GLCore observes all values and can adjust setpoints, but if GLCore dies, heating/cooling continues at the last setpoint.

## Thermostat Architecture

### Smart Thermostat with Internal PID (Recommended)

Most professional KNX thermostats (MDT, Theben, ABB, Siemens) include an internal PID controller. This is the recommended approach for Gray Logic installations.

#### How It Works

1. **Thermostat** measures room temperature
2. **Internal PID** compares to setpoint, calculates heating demand (0-100%)
3. **Output** sent to valve actuator:
   - **Modulating actuators**: Percentage sent directly (DPT 5.001)
   - **Thermal actuators**: Thermostat PWMs the signal internally (e.g., 50% = ON 7.5min / OFF 7.5min in a 15-minute cycle)
4. **GLCore observes** temperature, setpoint, mode, and valve demand
5. **GLCore can override** setpoint based on presence, schedule, or user request

#### KNX Group Objects

| Object | DPT | Direction | Description |
|--------|-----|-----------|-------------|
| `current_temperature` | 9.001 | Read | Measured room temperature |
| `setpoint` | 9.001 | Read/Write | Target temperature (GLCore can override) |
| `setpoint_status` | 9.001 | Read | Currently active setpoint |
| `heating_output` | 5.001 | Read | PID-calculated valve demand (0-100%) |
| `mode` | 20.102 | Read/Write | HVAC mode (comfort/standby/economy/protection) |
| `mode_status` | 20.102 | Read | Currently active mode |

#### Fail-Safe Behavior

| Scenario | Behavior |
|----------|----------|
| GLCore offline | Thermostat continues at current setpoint — PID keeps running |
| KNX bus failure | Actuator goes to configured safety position (see below) |
| Thermostat failure | Actuator goes to safety position; GLCore alerts |

### Alternative: Separate Sensor + KNX Logic Module

For complex zones or when using simple temperature sensors, a KNX logic module (ABB, Siemens, Weinzierl) can run the PID:

```
Temp Sensor ──→ KNX Logic Module ──→ Valve Actuator
                     ↑
              GLCore (setpoint)
```

This achieves the same offline-first behavior — the logic module runs the PID, not GLCore.

**Note**: KNXSim does not simulate logic modules. For simulator purposes, use the smart thermostat model.

### Anti-Pattern: GLCore as Primary Controller ❌

**Do NOT do this:**

```
Temp Sensor ──→ GLCore (PID) ──→ Valve Actuator
```

This violates offline-first. If GLCore fails, there's no heating control.

## Valve Actuators

### Types

| Type | Signal | Behavior | Use Case |
|------|--------|----------|----------|
| **Thermal (NC)** | Binary | Normally Closed — spring closes valve when no power | Most common for UFH |
| **Thermal (NO)** | Binary | Normally Open — spring opens valve when no power | Freeze protection priority |
| **Modulating** | 0-100% | Proportional valve position | Commercial, high-end residential |

### PWM for Thermal Actuators

Smart thermostats convert their percentage output to PWM for thermal actuators:

- **Cycle time**: Typically 15 minutes (configurable in ETS)
- **50% demand**: Valve ON for 7.5 min, OFF for 7.5 min
- **Thermal averaging**: The actuator's thermal mass smooths out the on/off cycling

The actuator doesn't "know" it's receiving PWM — it just sees on/off commands. The averaging happens due to its physical response time.

### Safety Configuration

KNX actuators must be configured with safety parameters in ETS:

| Parameter | Recommended Value | Purpose |
|-----------|-------------------|---------|
| Safety timeout | 30 minutes | Time without telegram before entering safety mode |
| Safety position (heating) | 0% (closed) | Prevents overheating if control fails |
| Safety position (freeze risk) | 100% (open) | Prevents pipe freeze in vulnerable areas |

**GLCore's role**: Document recommended settings. The actual configuration is done in ETS.

## GLCore Functions

### Observation

GLCore reads all climate data from the bus:

- Room temperatures
- Setpoints (current and active)
- Heating/cooling demands
- Valve states
- HVAC modes

This data is used for:
- Dashboard display
- Historical trending (InfluxDB)
- Energy monitoring
- Predictive Health Monitoring (PHM)

### Setpoint Override

GLCore can adjust setpoints based on:

| Trigger | Action | Example |
|---------|--------|---------|
| **Presence** | Set to economy mode when room empty | Unoccupied → setback 3°C |
| **Schedule** | Time-based setpoint changes | Night mode 22:00-06:00 |
| **Scene** | Scene-triggered temperature | "Away" scene → all zones to frost protect |
| **User request** | Manual adjustment via app/voice | "Set living room to 22°C" |
| **Weather** | Pre-emptive adjustment based on forecast | Cold snap coming → pre-heat |

### Coordination

GLCore coordinates between zones:

- Don't heat rooms that are unoccupied (presence integration)
- Balance plant capacity across zones
- Prevent conflicting demands (don't heat and cool simultaneously)

### Intelligence (Future)

- **Learning**: Learn occupancy patterns, optimal pre-heat times
- **Optimization**: Minimize energy while maintaining comfort
- **PHM**: Detect failing actuators (slow response, stuck valves)

## Simulator Implementation

KNXSim models smart thermostats with internal PID:

### Simulated Behavior

1. Temperature changes based on:
   - Heating output (if > 0%, temperature rises)
   - Ambient heat loss (gradual decay toward ambient)
   - External factors (window open, occupancy)

2. PID calculation (simplified):
   - Error = setpoint - actual_temperature
   - Output = Kp × error (simplified proportional control)
   - Real thermostats use full PID with integral/derivative terms

3. All values readable via KNX telegrams

### Why Percentage Output?

The simulator uses percentage output (DPT 5.001) because:

1. It shows the actual PID calculation
2. Works for both actuator types conceptually
3. More informative for debugging/learning
4. Real thermostats output percentages internally (converted to PWM for thermal actuators)

## Device Types

### Thermostat

See [Device Registry: Thermostat](/docs/data-model/entities.md#thermostat) for state schema.

State fields:
```json
{
  "current_temperature": 21.5,
  "setpoint": 22.0,
  "heating_output": 45,
  "mode": "comfort"
}
```

### Valve Actuator

State fields:
```json
{
  "position": 45,
  "on": true
}
```

### Temperature Sensor (Standalone)

For use with KNX logic modules (not simulated):

```json
{
  "temperature": 21.5
}
```

## Installation Guidance

### Recommended Hardware

| Component | Examples | Notes |
|-----------|----------|-------|
| **Smart thermostat** | MDT SCN-RT1.01, Theben RAMSES 718, ABB RTC | Must have internal PID |
| **Thermal actuator (NC)** | MDT AKH, ABB VC/S | Normally closed for heating |
| **Modulating actuator** | Belimo, Siemens | For higher-end installations |

### ETS Configuration Checklist

- [ ] Thermostat PID parameters tuned for the zone
- [ ] Cycle time appropriate for actuator type (15 min typical for thermal)
- [ ] Setpoint limits configured (min/max)
- [ ] Safety timeout set on actuators (30 min)
- [ ] Safety position appropriate for zone (0% or 100%)
- [ ] Mode objects linked if using schedules
- [ ] Feedback GAs configured for GLCore observation

### Wiring Notes

- Thermal actuators: Typically 24V AC from heating actuator controller
- Ensure correct NC/NO selection during ordering
- Label manifold zones clearly for commissioning

## Summary

| Aspect | Approach |
|--------|----------|
| **PID location** | KNX thermostat (on-device) |
| **GLCore role** | Observer + setpoint override |
| **Offline behavior** | Heating continues at last setpoint |
| **Simulator model** | Smart thermostat with internal PID |
| **Output type** | Percentage (0-100%), thermostat handles PWM |
| **Fail-safe** | Configured in ETS on actuators |
