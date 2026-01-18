---
title: Automation Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/core-internals.md
  - data-model/entities.md
  - interfaces/api.md
---

# Automation Specification

This document specifies how Gray Logic implements automation through Scenes, Schedules, Modes, Conditions, and Event-Driven Triggers. These building blocks work together to create intelligent, responsive environments.

---

## Overview

### Automation Philosophy

Gray Logic automation follows these principles:

1. **Explicit over implicit** — Users should understand why things happen
2. **Predictable** — Same inputs produce same outputs
3. **Fail-safe** — Automation failures don't compromise safety or basic operation
4. **Local-first** — All automation runs on-site, no cloud dependency
5. **Physical controls always win** — Wall switches override automation

### Automation Hierarchy

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              TRIGGERS                                        │
│                                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   Manual    │  │  Schedule   │  │    Event    │  │       Voice         │ │
│  │  (UI/Voice) │  │ (Time-based)│  │  (Sensors)  │  │   (NLU Intent)      │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
└─────────┼────────────────┼────────────────┼────────────────────┼────────────┘
          │                │                │                    │
          └────────────────┴────────────────┴────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CONDITION CHECK                                    │
│                                                                              │
│  "Should this automation run right now?"                                    │
│                                                                              │
│  • Mode check (home/away/night/holiday)                                     │
│  • Time window check                                                         │
│  • Presence check                                                            │
│  • Device state check                                                        │
│  • Custom conditions                                                         │
│                                                                              │
└─────────────────────────────────┬───────────────────────────────────────────┘
                                  │
                          ┌───────▼───────┐
                          │  Conditions   │
                          │     Met?      │
                          └───────┬───────┘
                                  │
                    ┌─────────────┴─────────────┐
                    │ Yes                       │ No
                    ▼                           ▼
┌─────────────────────────────────┐    ┌──────────────────┐
│         EXECUTE ACTIONS         │    │  Log & Skip      │
│                                 │    │  (No action)     │
│  • Scene activation             │    └──────────────────┘
│  • Device commands              │
│  • Mode changes                 │
│  • Notifications                │
│                                 │
└─────────────────────────────────┘
```

---

## Scenes

### What is a Scene?

A Scene is a predefined collection of device states that can be activated together. Scenes provide:

- **Convenience** — One action sets multiple devices
- **Consistency** — Same result every time
- **Ambiance** — Coordinated lighting, blinds, audio, climate

### Scene Structure

```yaml
Scene:
  id: uuid
  site_id: uuid
  name: string                      # "Cinema Mode"
  slug: string                      # "cinema-mode"
  icon: string                      # "movie"
  color: string                     # "#6B21A8"
  
  scope:
    type: "site" | "area" | "room"
    id: uuid
    
  triggers:
    manual: boolean                 # Available in UI
    voice_phrase: string | null     # "cinema mode", "movie time"
    keypads: [KeypadBinding]        # Physical button bindings
    
  actions: [Action]
  conditions: [Condition] | null
  
  priority: integer | null            # 1-100, higher wins conflicts (default: 50)
  category: string                  # "entertainment", "comfort", "security"
  sort_order: integer
```

### Scene Execution Tracking

Every scene activation creates a `SceneExecution` record to track progress and results:

```yaml
SceneExecution:
  id: uuid
  scene_id: uuid
  triggered_at: timestamp           # When activation was requested
  started_at: timestamp             # When first action began
  completed_at: timestamp | null    # When all actions finished (null if running)

  trigger:
    type: "manual" | "schedule" | "event" | "voice" | "automation"
    user_id: uuid | null            # If manual or voice
    schedule_id: uuid | null        # If scheduled
    event: object | null            # If event-triggered

  status: "pending" | "running" | "completed" | "partial" | "failed" | "cancelled"

  # Action tracking
  actions_total: integer            # Total actions in scene
  actions_completed: integer        # Successfully executed
  actions_failed: integer           # Failed to execute
  actions_skipped: integer          # Skipped due to conditions

  # Failure details (if any)
  failures: [ActionFailure]

  # Conflict tracking (see Scene Conflict Resolution)
  conflicts: [ConflictRecord] | null

  # Timing
  duration_ms: integer | null       # Total execution time
```

**ActionFailure Structure:**

```yaml
ActionFailure:
  action_index: integer             # Position in actions array
  device_id: uuid
  command: string
  error:
    code: string                    # "DEVICE_UNREACHABLE", "TIMEOUT", etc.
    message: string
    timestamp: timestamp
  acknowledged: boolean             # Has user seen this failure
```

**Scene Execution States:**

| Status | Description |
|--------|-------------|
| `pending` | Triggered, waiting for conditions check |
| `running` | Actions currently executing |
| `completed` | All actions executed successfully |
| `partial` | Some actions failed, but continue_on_error allowed completion |
| `failed` | Critical action failed, scene aborted |
| `cancelled` | User or system cancelled mid-execution |

**API Endpoint for Execution Status:**

```http
GET /api/v1/scenes/{scene_id}/executions?limit=10
```

**Response:**
```json
{
  "data": [
    {
      "id": "exec-001",
      "scene_id": "scene-cinema",
      "status": "partial",
      "triggered_at": "2026-01-15T20:00:00Z",
      "completed_at": "2026-01-15T20:00:02Z",
      "actions_total": 8,
      "actions_completed": 7,
      "actions_failed": 1,
      "failures": [
        {
          "action_index": 5,
          "device_id": "blind-living-1",
          "command": "position",
          "error": {
            "code": "TIMEOUT",
            "message": "Device did not respond within 5000ms"
          }
        }
      ],
      "duration_ms": 2150
    }
  ]
}
```

### Actions

Actions define what happens when a scene activates:

```yaml
Action:
  # Target specification (one required)
  device_id: uuid                   # Single device
  device_group: string              # Tag-based group
  room_id: uuid                     # All devices in room
  domain: string                    # Filter by domain (lighting, climate, etc.)
  
  # Command
  command: string                   # "set", "dim", "position", "activate"
  parameters: object                # Command-specific parameters
  
  # Timing
  delay_ms: integer                 # Wait before executing (default: 0)
  fade_ms: integer                  # Transition duration (default: 0)
  
  # Execution
  parallel: boolean                 # Run with previous action (default: false)
  continue_on_error: boolean        # Don't abort scene on failure (default: true)
```

### Action Targeting

#### Single Device

```yaml
- device_id: "light-living-main"
  command: "set"
  parameters:
    on: true
    brightness: 75
```

#### Device Group (Tag-Based)

```yaml
- device_group: "living_room_lights"
  command: "set"
  parameters:
    on: false
  fade_ms: 3000
```

#### Room + Domain

```yaml
# All lights in the living room
- room_id: "room-living"
  domain: "lighting"
  command: "set"
  parameters:
    on: false
```

#### All Devices in Room

```yaml
# Everything in the room (lights, blinds, etc.)
- room_id: "room-living"
  domain: null  # No filter
  command: "off"
```

### Timing and Sequencing

#### Sequential Execution

Actions execute in order by default:

```yaml
actions:
  - device_id: "light-main"      # Step 1: Runs first
    command: "set"
    parameters: { on: true }
    
  - device_id: "light-accent"    # Step 2: Runs after step 1 completes
    command: "set"
    parameters: { on: true }
```

#### Parallel Execution

Use `parallel: true` to run with the previous action:

```yaml
actions:
  - device_id: "light-main"
    command: "set"
    parameters: { on: false }
    fade_ms: 2000
    
  - device_id: "blind-main"      # Runs simultaneously with lights
    command: "position"
    parameters: { position: 0 }
    parallel: true
```

#### Delayed Execution

```yaml
actions:
  - device_id: "light-main"
    command: "set"
    parameters: { on: false }
    fade_ms: 3000
    
  - device_id: "av-receiver"
    command: "source"
    parameters: { input: "hdmi2" }
    delay_ms: 3500              # Wait for lights to fade, then switch source
```

### Scene Examples

#### Cinema Mode

```yaml
id: "scene-cinema"
name: "Cinema Mode"
scope:
  type: "room"
  id: "room-living"
triggers:
  manual: true
  voice_phrase: "cinema mode"
actions:
  # Dim lights over 3 seconds
  - device_group: "living_lights"
    command: "set"
    parameters: { on: true, brightness: 5 }
    fade_ms: 3000
    
  # Close blinds (parallel with lights)
  - room_id: "room-living"
    domain: "blinds"
    command: "position"
    parameters: { position: 0 }
    parallel: true
    
  # Switch AV to correct input (after lights dimmed)
  - device_id: "av-receiver"
    command: "source"
    parameters: { input: "hdmi2" }
    delay_ms: 3000
    
  # Set comfortable volume
  - device_id: "av-receiver"
    command: "volume"
    parameters: { level: 35 }
    delay_ms: 3500
    
conditions:
  - type: "mode"
    operator: "in"
    value: ["home", "entertaining"]
```

#### Good Morning (Bedroom)

```yaml
id: "scene-morning-bedroom"
name: "Good Morning"
scope:
  type: "room"
  id: "room-master-bedroom"
triggers:
  manual: true
  voice_phrase: "good morning"
actions:
  # Gently raise blinds
  - device_id: "blind-bedroom-01"
    command: "position"
    parameters: { position: 100 }
    
  # Warm lights at low level
  - device_group: "bedroom_lights"
    command: "set"
    parameters:
      on: true
      brightness: 30
      color_temp_kelvin: 2700
    fade_ms: 10000
    delay_ms: 2000
    
  # Start coffee machine (if integrated)
  - device_id: "coffee-machine"
    command: "brew"
    parameters: { cups: 2 }
    delay_ms: 5000
    continue_on_error: true    # Don't fail scene if coffee machine offline
```

#### All Off (Leaving Home)

```yaml
id: "scene-all-off"
name: "All Off"
scope:
  type: "site"
  id: "site-main"
triggers:
  manual: true
  voice_phrase: "all off"
actions:
  # All lights off
  - domain: "lighting"
    command: "set"
    parameters: { on: false }
    
  # All audio off
  - domain: "audio"
    command: "set"
    parameters: { power: false }
    parallel: true
    
  # Close all blinds
  - domain: "blinds"
    command: "position"
    parameters: { position: 0 }
    parallel: true
    
  # Set climate to away setback
  - domain: "climate"
    command: "setpoint"
    parameters: { offset: -3 }
    parallel: true
```

### Scene Categories

Organize scenes by purpose:

| Category | Examples |
|----------|----------|
| `comfort` | Relax, Reading, Bright |
| `entertainment` | Cinema, Music, Gaming |
| `productivity` | Focus, Meeting, Presentation |
| `daily` | Good Morning, Good Night, Leaving, Arriving |
| `security` | Panic, Vacation Simulation |
| `energy` | Eco Mode, Away Setback |

### Scene Conflict Resolution

When multiple scenes target the same device within a short time window, Gray Logic applies conflict resolution to prevent flickering and ensure predictable behavior.

#### Conflict Detection

A **conflict** occurs when:
1. Two or more scenes are triggered within the **stabilization window** (default: 500ms)
2. Both scenes contain actions targeting the **same device**
3. The actions specify **different target states**

#### Resolution Rules

Conflicts are resolved using these rules, in priority order:

| Rule | Behavior |
|------|----------|
| **Physical control wins** | If a physical input (wall switch, keypad) triggers a command during scene execution, the physical command wins and clears pending scene actions for that device |
| **Higher priority wins** | Scene with higher `priority` value executes; lower priority scene's conflicting actions are skipped |
| **Last-write wins** | If priorities are equal, the most recently triggered scene wins |
| **Stabilization window** | After a device receives a command, subsequent scene commands to that device are held for 500ms; if a newer command arrives, the older one is discarded |

#### Priority Field

Scenes have an optional `priority` field (1-100, default: 50):

```yaml
Scene:
  priority: integer | null            # 1-100, higher wins conflicts (default: 50)
```

**Recommended priority ranges:**

| Range | Use Case | Examples |
|-------|----------|----------|
| 80-100 | Safety/Security | Panic Mode, Alarm Response |
| 60-79 | User-initiated | Cinema Mode, Good Night (manual trigger) |
| 40-59 | Scheduled | Evening Lights, Morning Routine |
| 20-39 | Automation/Events | Motion-triggered, Presence-based |
| 1-19 | Background | Eco adjustments, Ambient scenes |

#### Stabilization Window

The stabilization window prevents rapid command flickering:

```yaml
conflict_resolution:
  stabilization_window_ms: 500        # Configurable per-site
```

**Behavior:**
1. Scene A sends `brightness: 5%` to `light-living-main`
2. Within 500ms, Scene B sends `brightness: 75%` to same device
3. Scene B's command replaces Scene A's command (if Scene B has equal or higher priority)
4. Only one command is sent to the device

#### Conflict Logging

All conflicts are logged for debugging:

```yaml
ConflictEvent:
  timestamp: timestamp
  device_id: uuid
  winning_scene_id: uuid
  losing_scene_id: uuid
  resolution_rule: "priority" | "last_write" | "physical_override"
  winning_command: object
  discarded_command: object
```

#### Conflict Record Structure

The `SceneExecution` record includes conflict information via `ConflictRecord`:

```yaml
ConflictRecord:
  device_id: uuid
  conflicting_scene_id: uuid
  resolution: "won" | "lost" | "physical_override"
  action_taken: object | null         # null if this scene lost
```

#### UI Behavior

When a conflict occurs:
- **Wall panel / Mobile app:** Shows toast: "Cinema Mode activated (Evening Lights skipped for conflicting devices)"
- **Scene execution API:** Returns `status: "completed"` with `conflicts` array listing skipped actions
- **Audit log:** Records full conflict details

> [!NOTE]
> Physical controls (wall switches, keypads) always override scene commands. If a user presses a switch during scene execution, that input is applied immediately and any pending scene commands to that device are cancelled. This ensures [Physical Controls Always Work](../overview/principles.md#1-physical-controls-always-work).

---

## Schedules

### What is a Schedule?

A Schedule triggers automation at specific times or astronomical events. Schedules are the "when" of automation.

### Schedule Structure

```yaml
Schedule:
  id: uuid
  site_id: uuid
  name: string
  enabled: boolean
  
  trigger:
    type: "time" | "sunrise" | "sunset" | "solar_noon" | "cron"
    value: string                   # Time, offset, or cron expression
    days: [string] | null           # ["mon", "tue", ...] or null for all
    
  execute:
    type: "scene" | "actions" | "mode"
    scene_id: uuid | null
    actions: [Action] | null
    mode_id: string | null
    
  conditions: [Condition] | null
  
  valid_from: date | null           # Optional start date
  valid_until: date | null          # Optional end date
  
  randomize_minutes: integer | null # Add randomness (0-N minutes)
```

### Trigger Types

#### Fixed Time

```yaml
trigger:
  type: "time"
  value: "07:30"
  days: ["mon", "tue", "wed", "thu", "fri"]
```

#### Sunrise/Sunset (Astronomical)

```yaml
# 30 minutes before sunset
trigger:
  type: "sunset"
  value: "-30m"
  days: null                        # Every day

# At sunrise
trigger:
  type: "sunrise"
  value: "0m"
  
# 1 hour after sunrise
trigger:
  type: "sunrise"
  value: "+60m"
```

#### Solar Noon

```yaml
trigger:
  type: "solar_noon"
  value: "0m"                       # Exactly at solar noon
```

#### Cron Expression

For complex patterns:

```yaml
# Every 2 hours
trigger:
  type: "cron"
  value: "0 */2 * * *"

# First Monday of each month at 9am
trigger:
  type: "cron"
  value: "0 9 1-7 * 1"
```

### Schedule Examples

#### Weekday Morning Routine

```yaml
id: "sched-weekday-morning"
name: "Weekday Morning"
enabled: true
trigger:
  type: "time"
  value: "06:30"
  days: ["mon", "tue", "wed", "thu", "fri"]
execute:
  type: "scene"
  scene_id: "scene-morning-bedroom"
conditions:
  - type: "mode"
    operator: "neq"
    value: "holiday"
```

#### Evening Lights

```yaml
id: "sched-evening-lights"
name: "Evening Lights On"
enabled: true
trigger:
  type: "sunset"
  value: "-15m"
  days: null
execute:
  type: "actions"
  actions:
    - device_group: "outdoor_lights"
      command: "set"
      parameters: { on: true }
    - device_group: "pathway_lights"
      command: "set"
      parameters: { on: true, brightness: 50 }
conditions:
  - type: "mode"
    operator: "in"
    value: ["home", "entertaining"]
```

#### Night Mode Activation

```yaml
id: "sched-night-mode"
name: "Activate Night Mode"
enabled: true
trigger:
  type: "time"
  value: "23:00"
  days: null
execute:
  type: "mode"
  mode_id: "night"
conditions:
  - type: "mode"
    operator: "eq"
    value: "home"
```

#### Vacation Light Simulation

```yaml
id: "sched-vacation-sim"
name: "Vacation Light Simulation"
enabled: true
trigger:
  type: "sunset"
  value: "+30m"
  days: null
execute:
  type: "actions"
  actions:
    - device_group: "vacation_sim_lights"
      command: "set"
      parameters: { on: true }
conditions:
  - type: "mode"
    operator: "eq"
    value: "holiday"
randomize_minutes: 30              # Random delay 0-30 mins for realism
valid_from: null
valid_until: null
```

#### Commercial: Out-of-Hours Setback

```yaml
id: "sched-office-setback"
name: "Office Night Setback"
enabled: true
trigger:
  type: "time"
  value: "19:00"
  days: ["mon", "tue", "wed", "thu", "fri"]
execute:
  type: "actions"
  actions:
    - domain: "climate"
      command: "setpoint"
      parameters: { heating: 16, cooling: 28 }
    - domain: "lighting"
      room_id: "area-office"
      command: "set"
      parameters: { on: false }
conditions:
  - type: "presence"
    operator: "nobody_in_area"
    value: "area-office"
```

### Astronomical Calculations

Gray Logic calculates sunrise/sunset based on site location:

```yaml
# Site configuration
site:
  location:
    latitude: 51.5074
    longitude: -0.1278
    timezone: "Europe/London"
```

| Event | Calculation |
|-------|-------------|
| `sunrise` | Sun crosses horizon (standard) |
| `sunset` | Sun crosses horizon (standard) |
| `solar_noon` | Sun at highest point |
| `civil_dawn` | Sun 6° below horizon (light enough to work outside) |
| `civil_dusk` | Sun 6° below horizon |

---

## Modes

### What is a Mode?

A Mode is a system-wide operational state that modifies how automation behaves. Modes answer "what should the house be doing right now?"

### Standard Modes

| Mode | Icon | Purpose |
|------|------|---------|
| `home` | 🏠 | Normal occupied operation |
| `away` | 🚪 | Nobody home, energy saving |
| `night` | 🌙 | Sleeping, quiet operation |
| `holiday` | ✈️ | Extended absence, simulation |
| `entertaining` | 🎉 | Guests present, disable auto-off |

### Mode Structure

```yaml
Mode:
  id: string                        # "home", "away", "night"
  name: string                      # "Home", "Away", "Night"
  icon: string
  color: string                     # Hex color for UI
  
  behaviours:
    climate:
      setpoint_offset: float        # Add/subtract from normal setpoint
      eco_mode: boolean             # Enable eco/setback mode
      
    lighting:
      auto_off_enabled: boolean     # Turn off lights when vacant
      auto_off_delay_min: integer   # Minutes before auto-off
      max_brightness: integer       # Cap brightness (0-100)
      circadian_enabled: boolean    # Follow circadian rhythm
      
    blinds:
      auto_enabled: boolean         # Allow automatic blind control
      sun_protection: boolean       # Close for solar gain protection
      
    security:
      arm_state: string | null      # "disarm", "arm_stay", "arm_away"
      
    audio:
      max_volume: integer | null    # Volume cap (0-100)
      
    notifications:
      priority_only: boolean        # Suppress non-critical notifications
      
  can_activate:
    roles: [string]                 # ["admin", "resident"]
    require_pin: boolean            # Require PIN confirmation
    remote_allowed: boolean         # Can activate from outside LAN
```

### Mode Behaviour Examples

#### Home Mode

```yaml
id: "home"
name: "Home"
icon: "home"
color: "#22C55E"
behaviours:
  climate:
    setpoint_offset: 0
    eco_mode: false
  lighting:
    auto_off_enabled: true
    auto_off_delay_min: 15
    max_brightness: 100
    circadian_enabled: true
  blinds:
    auto_enabled: true
    sun_protection: true
  security:
    arm_state: "disarm"
  notifications:
    priority_only: false
can_activate:
  roles: ["admin", "resident", "guest"]
  require_pin: false
  remote_allowed: true
```

#### Night Mode

```yaml
id: "night"
name: "Night"
icon: "moon"
color: "#6366F1"
behaviours:
  climate:
    setpoint_offset: -2            # Cooler for sleeping
    eco_mode: false
  lighting:
    auto_off_enabled: true
    auto_off_delay_min: 5          # Quick auto-off
    max_brightness: 30             # Dim lights only
    circadian_enabled: false       # Warm only at night
  blinds:
    auto_enabled: false            # Don't move blinds
  security:
    arm_state: "arm_stay"          # Perimeter armed
  audio:
    max_volume: 30
  notifications:
    priority_only: true            # No non-urgent notifications
can_activate:
  roles: ["admin", "resident"]
  require_pin: false
  remote_allowed: false            # Must be home to activate
```

#### Away Mode

```yaml
id: "away"
name: "Away"
icon: "door-open"
color: "#EAB308"
behaviours:
  climate:
    setpoint_offset: -3
    eco_mode: true
  lighting:
    auto_off_enabled: true
    auto_off_delay_min: 2          # Very quick auto-off
    max_brightness: 50
  blinds:
    auto_enabled: true
    sun_protection: true           # Protect from sun when away
  security:
    arm_state: "arm_away"
  notifications:
    priority_only: true
can_activate:
  roles: ["admin", "resident"]
  require_pin: false
  remote_allowed: true
```

### Mode Transitions

#### Automatic Transitions

Modes can transition automatically based on:

1. **Schedule** — Time-based (e.g., night mode at 23:00)
2. **Presence** — Last person leaves → Away mode
3. **Geofencing** — Approaching home → Home mode

```yaml
# Example: Auto-switch to Away when last person leaves
mode_automation:
  - trigger:
      type: "presence"
      event: "last_person_left"
    action:
      set_mode: "away"
    conditions:
      - type: "mode"
        operator: "eq"
        value: "home"
```

#### Manual Transitions

Users can change modes via:
- Mobile app
- Wall panel
- Voice command ("set mode to night")
- Physical keypad
- API call

### Mode and Scene Interaction

Scenes can:
1. **Check mode** as a condition (only run in certain modes)
2. **Change mode** as an action
3. **Behave differently** based on mode (via conditions)

```yaml
# Scene that only runs in home mode
conditions:
  - type: "mode"
    operator: "in"
    value: ["home", "entertaining"]

# Scene that changes mode
actions:
  - command: "set_mode"
    parameters: { mode: "night" }
```

---

## Conditions

### What is a Condition?

A Condition is a prerequisite that must be true for automation to execute. Conditions prevent unwanted automation.

### Condition Structure

```yaml
Condition:
  type: string                      # Condition type
  operator: string                  # Comparison operator
  value: any                        # Value to compare
  device_id: uuid | null            # For device-based conditions
  negate: boolean                   # Invert the result (default: false)
```

### Condition Types

#### Mode Condition

```yaml
# Current mode equals "home"
- type: "mode"
  operator: "eq"
  value: "home"

# Current mode is one of these
- type: "mode"
  operator: "in"
  value: ["home", "entertaining"]

# Current mode is NOT "holiday"
- type: "mode"
  operator: "neq"
  value: "holiday"
```

#### Time Condition

```yaml
# Between 22:00 and 06:00
- type: "time"
  operator: "between"
  value: ["22:00", "06:00"]

# After 18:00
- type: "time"
  operator: "after"
  value: "18:00"

# Before sunrise
- type: "time"
  operator: "before"
  value: "sunrise"
```

#### Day of Week

```yaml
# Only on weekdays
- type: "day"
  operator: "in"
  value: ["mon", "tue", "wed", "thu", "fri"]

# Not on weekends
- type: "day"
  operator: "not_in"
  value: ["sat", "sun"]
```

#### Presence Condition

```yaml
# Anyone is home
- type: "presence"
  operator: "any_home"
  value: null

# Nobody is home
- type: "presence"
  operator: "nobody_home"
  value: null

# Specific user is home
- type: "presence"
  operator: "user_home"
  value: "usr-001"

# Room is occupied
- type: "presence"
  operator: "room_occupied"
  value: "room-living"
```

#### Device State Condition

```yaml
# Light is on
- type: "device_state"
  device_id: "light-living-main"
  operator: "eq"
  value: { "on": true }

# Brightness above 50%
- type: "device_state"
  device_id: "light-living-main"
  operator: "gt"
  value: { "brightness": 50 }

# Blind is closed
- type: "device_state"
  device_id: "blind-living-01"
  operator: "eq"
  value: { "position": 0 }

# Temperature below setpoint
- type: "device_state"
  device_id: "thermostat-living"
  operator: "lt"
  value: { "current_temp": 19.0 }
```

#### Sun Position

```yaml
# Sun is above horizon
- type: "sun"
  operator: "above_horizon"
  value: null

# Sun elevation > 10 degrees
- type: "sun"
  operator: "elevation_gt"
  value: 10

# Sun azimuth between 90-180 (morning, east-facing)
- type: "sun"
  operator: "azimuth_between"
  value: [90, 180]
```

#### Weather Condition

```yaml
# It's raining (from weather integration)
- type: "weather"
  operator: "eq"
  value: { "condition": "rain" }

# Wind speed above threshold
- type: "weather"
  operator: "gt"
  value: { "wind_speed_kph": 30 }
```

### Combining Conditions

Multiple conditions are combined with AND logic by default:

```yaml
conditions:
  - type: "mode"
    operator: "eq"
    value: "home"
  - type: "time"
    operator: "between"
    value: ["18:00", "23:00"]
  - type: "presence"
    operator: "room_occupied"
    value: "room-living"
# All three must be true
```

For OR logic, use separate automations or a grouped condition:

```yaml
conditions:
  - type: "or"
    conditions:
      - type: "mode"
        operator: "eq"
        value: "home"
      - type: "mode"
        operator: "eq"
        value: "entertaining"
```

---

## Event-Driven Triggers

### What is an Event Trigger?

Event triggers respond to real-time changes in the system rather than scheduled times. They answer "when X happens, do Y."

### Event Trigger Structure

```yaml
EventTrigger:
  id: uuid
  name: string
  enabled: boolean
  
  trigger:
    type: string                    # Event type
    source: object                  # What generates the event
    
  execute:
    type: "scene" | "actions"
    scene_id: uuid | null
    actions: [Action] | null
    
  conditions: [Condition] | null
  
  throttle_seconds: integer | null  # Minimum time between triggers
  cooldown_seconds: integer | null  # Time to wait after triggering
```

### Event Types

#### Device State Changed

```yaml
trigger:
  type: "device_state_changed"
  source:
    device_id: "motion-hallway"
    from_state: { "motion": false }
    to_state: { "motion": true }
```

#### Motion Detected

```yaml
trigger:
  type: "motion_detected"
  source:
    device_id: "motion-hallway"
    # or
    room_id: "room-hallway"
```

#### Door/Window Opened

```yaml
trigger:
  type: "contact_opened"
  source:
    device_id: "door-front"
```

#### Presence Changed

```yaml
trigger:
  type: "presence_changed"
  source:
    user_id: "usr-001"
    event: "arrived" | "departed"
```

#### Button Pressed

```yaml
trigger:
  type: "button_pressed"
  source:
    device_id: "keypad-living"
    button: 3
    press_type: "short" | "long" | "double"
```

#### Alarm Event

```yaml
trigger:
  type: "alarm_event"
  source:
    event: "triggered" | "armed" | "disarmed"
```

### Event Trigger Examples

#### Motion-Activated Lights

```yaml
id: "trigger-hallway-motion"
name: "Hallway Motion Lights"
enabled: true
trigger:
  type: "motion_detected"
  source:
    room_id: "room-hallway"
execute:
  type: "actions"
  actions:
    - room_id: "room-hallway"
      domain: "lighting"
      command: "set"
      parameters:
        on: true
        brightness: 70
conditions:
  - type: "time"
    operator: "between"
    value: ["sunset", "sunrise"]
  - type: "device_state"
    device_id: "light-hallway"
    operator: "eq"
    value: { "on": false }
throttle_seconds: 30               # Don't re-trigger for 30 seconds
```

#### Welcome Home

```yaml
id: "trigger-welcome-home"
name: "Welcome Home"
enabled: true
trigger:
  type: "presence_changed"
  source:
    event: "first_person_arrived"
execute:
  type: "scene"
  scene_id: "scene-welcome"
conditions:
  - type: "mode"
    operator: "eq"
    value: "away"
```

#### Doorbell Response

```yaml
id: "trigger-doorbell"
name: "Doorbell Response"
enabled: true
trigger:
  type: "button_pressed"
  source:
    device_id: "doorbell-front"
    press_type: "short"
execute:
  type: "actions"
  actions:
    # Flash hallway lights
    - device_id: "light-hallway"
      command: "flash"
      parameters: { count: 3 }
    # Send notification
    - command: "notify"
      parameters:
        title: "Doorbell"
        message: "Someone is at the front door"
        priority: "high"
    # Display camera on living room TV
    - device_id: "tv-living"
      command: "show_camera"
      parameters: { camera_id: "camera-front-door" }
conditions:
  - type: "mode"
    operator: "in"
    value: ["home", "entertaining"]
```

---

## Automation Best Practices

### Do's

1. **Use conditions** — Always add mode/time/presence conditions to prevent unexpected behavior
2. **Set appropriate timeouts** — Use throttle/cooldown to prevent rapid re-triggering
3. **Use descriptive names** — "Evening Lights - Weekdays" not "Schedule 1"
4. **Group related actions** — One scene for "Cinema Mode" rather than multiple
5. **Test in stages** — Enable new automations gradually
6. **Document intent** — Add descriptions explaining why automation exists

### Don'ts

1. **Don't create circular triggers** — Light turns on → triggers scene → turns on light
2. **Don't override safety** — Never automate fire doors, emergency systems
3. **Don't ignore failures** — Log and alert on repeated automation failures
4. **Don't be too aggressive** — Auto-off timers should be generous
5. **Don't forget edge cases** — What happens at midnight? During DST change?

### Conflict Resolution

When multiple automations could trigger:

1. **User action wins** — Manual override always takes precedence
2. **Most specific wins** — Room-level beats site-level
3. **Most recent wins** — Latest trigger timestamp
4. **Safety wins** — Security/safety automations override comfort

---

## Related Documents

- [Data Model: Entities](../data-model/entities.md) — Entity definitions for Scene, Schedule, Mode
- [Core Internals](../architecture/core-internals.md) — Scene Engine, Scheduler, Mode Manager
- [REST API Specification](../interfaces/api.md) — API endpoints for automation
- [Lighting Domain](../domains/lighting.md) — Lighting-specific automation
- [Climate Domain](../domains/climate.md) — Climate-specific automation
- [Audio Domain](../domains/audio.md) — Audio-specific automation and announcements