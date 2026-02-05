---
title: Offline Behavior Specification
version: 1.0.0
status: active
last_updated: 2026-01-13
depends_on:
  - overview/principles.md
  - architecture/system-overview.md
  - architecture/core-internals.md
see_also:
  - architecture/supervisor.md       # Future: Intelligent orchestration layer
  - errors/catalog.md                # Future: Machine-readable error IDs
---


# Offline Behavior Specification

This document specifies how Gray Logic behaves when components fail or become unavailable — ensuring the system degrades gracefully while maintaining core functionality.

---

## Overview

### Offline-First Philosophy

Gray Logic is **offline-first**, not merely offline-capable:

| Offline-Capable | Offline-First |
|-----------------|---------------|
| Works without internet (sometimes) | **Designed** to work without internet |
| Cloud is primary, local is fallback | **Local is primary**, cloud is bonus |
| Degrades unpredictably | **Predictable, documented** degradation |
| May lose functionality unexpectedly | Core functionality **always available** |

### Design Principles

1. **99%+ offline** — Almost all functionality works without internet
2. **Predictable degradation** — Users know exactly what stops working
3. **Fast recovery** — System restores full operation automatically
4. **No silent failures** — Degradation is visible and logged
5. **Physical controls always work** — Even if Gray Logic fails completely

### The Hierarchy of Availability

```
┌─────────────────────────────────────────────────────────────────┐
│ LEVEL 1: Physical Controls                                      │
│ • Wall switches control lights directly (KNX actuators)         │
│ • Always works, even with no software running                   │
├─────────────────────────────────────────────────────────────────┤
│ LEVEL 2: Gray Logic Core                                        │
│ • Scenes, schedules, automation                                 │
│ • Works without internet, without bridges (cached state)        │
├─────────────────────────────────────────────────────────────────┤
│ LEVEL 3: Protocol Bridges                                       │
│ • Real-time device control                                      │
│ • Each bridge is independent; one failure doesn't affect others │
├─────────────────────────────────────────────────────────────────┤
│ LEVEL 4: External Services                                      │
│ • Weather data, remote access, AI insights                      │
│ • Nice-to-have, not required for core operation                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Component Failure Matrix

### What Works When Things Fail

| Component Down | Still Works | Doesn't Work |
|----------------|-------------|--------------|
| **Internet** | All local control, scenes, schedules, voice, PHM | Remote access, weather API, cloud AI |
| **MQTT Broker** | Core API (cached state), physical controls | Real-time bridge communication |
| **Database** | In-memory state, physical controls | Persistence, configuration changes |
| **KNX Bridge** | Other bridges, scenes (partial) | KNX device control |
| **Voice Bridge** | All control except voice | Voice commands |
| **VictoriaMetrics** | All control, automation | PHM analysis, historical data |
| **Gray Logic Core** | Physical controls, **frost protection (hardware-based)**, security panel, fire alarm | All automation, scenes, UI, PHM |

### Detailed Failure Scenarios

#### Internet Down

**Impact:** Minimal — Gray Logic is designed for this.

```yaml
internet_down:
  works:
    - "All lighting control"
    - "All climate control"
    - "All blind control"
    - "Scenes and schedules"
    - "Local voice commands"
    - "Security monitoring"
    - "PHM monitoring"
    - "Local UI (wall panels, local web)"
    - "Audio/video control"
    
  degraded:
    - "Weather data (uses cached/local)"
    - "AI insights (uses local models only)"
    - "Streaming sources (Spotify, etc.)"
    
  unavailable:
    - "Remote access"
    - "Push notifications (external)"
    - "Cloud calendar sync"
    - "External weather APIs"
    
  recovery:
    automatic: true
    sync_required: false
```

#### MQTT Broker Down

**Impact:** Moderate — bridges cannot communicate with Core.

```yaml
mqtt_down:
  works:
    - "Physical controls (KNX switches → actuators)"
    - "REST API (cached state)"
    - "WebSocket (cached state)"
    - "Database access"
    
  degraded:
    - "Device state (stale until recovery)"
    - "Scene execution (uses last known state)"
    
  unavailable:
    - "Real-time device control via UI"
    - "Real-time state updates"
    - "Bridge communication"
    - "Voice command execution"
    
  recovery:
    automatic: true
    action: "Core monitors broker, reconnects when available"
    sync:
      - "Bridges re-announce device states"
      - "Core reconciles state differences"
```

#### Database (SQLite) Down

**Impact:** Significant — no persistent configuration.

```yaml
database_down:
  works:
    - "Physical controls"
    - "In-memory state (if Core running)"
    - "MQTT communication"
    
  degraded:
    - "Automation (runs from cached config)"
    
  unavailable:
    - "Configuration changes"
    - "New scene creation"
    - "User management"
    - "State persistence"
    - "Audit logging"
    
  recovery:
    automatic: false
    action: "Investigate file system / disk issues"
    data_loss_risk: "Configuration changes since last backup"
```

#### Protocol Bridge Down (e.g., KNX Bridge)

**Impact:** Limited to that protocol only.

```yaml
bridge_down:
  # Example: KNX Bridge
  knx_bridge_down:
    works:
      - "All non-KNX devices"
      - "DALI lighting (if separate bridge)"
      - "Modbus plant equipment"
      - "Audio/video"
      - "Security (if not KNX)"
      
    degraded:
      - "Scenes (KNX devices won't respond)"
      - "Schedules (KNX devices won't respond)"
      
    unavailable:
      - "KNX device control"
      - "KNX device state updates"
      
    # Physical controls still work
    note: "KNX wall switches still control actuators directly"
    
    recovery:
      automatic: true
      action: "Core detects bridge offline, attempts restart"
      max_restart_attempts: 3
      backoff_seconds: [10, 30, 60]
```

#### Voice Bridge Down

**Impact:** Minimal — only voice is affected.

```yaml
voice_bridge_down:
  works:
    - "All device control via UI"
    - "Scenes, schedules, automation"
    - "Physical controls"
    - "Wall panels"
    - "Mobile app"
    
  unavailable:
    - "Voice commands"
    - "TTS announcements"
    
  recovery:
    automatic: true
    user_impact: "Low — voice is convenience, not primary control"
```

#### VictoriaMetrics Down

**Impact:** Low — only time-series data affected.

```yaml
victoriametrics_down:
  works:
    - "All device control"
    - "Scenes, schedules, automation"
    - "Real-time state"
    - "Configuration"
    
  degraded:
    - "PHM (no new analysis until recovery)"
    
  unavailable:
    - "Historical energy data"
    - "PHM trend analysis"
    - "Energy reports"
    
  data_handling:
    # Queue writes until recovery
    queue_writes: true
    max_queue_size_mb: 100
    queue_overflow: "drop_oldest"
    
  recovery:
    automatic: true
    action: "Flush queued writes on reconnection"
```

#### Gray Logic Core Down

**Impact:** High — but physical controls still work.

```yaml
core_down:
  works:
    - "Physical controls (KNX switches → actuators)"
    - "Frost protection (hardware-based, see below)"
    - "Security panel (independent)"
    - "Fire alarm (independent)"
    - "CCTV recording (NVR independent)"
    
  unavailable:
    - "All automation"
    - "Scenes and schedules"
    - "Voice control"
    - "UI (wall panels, app, web)"
    - "PHM monitoring"
    
  note: |
    This is why principle #1 exists: "Physical controls always work"
    The building is still livable without Gray Logic running.
    
  recovery:
    automatic: true
    action: "systemd auto-restart, watchdog monitoring"
    max_restart_attempts: 5
    notify_on_failure: true
```

#### Frost Protection During Failures

**Frost protection is hardware-based and continues regardless of software state.**

```yaml
frost_protection_resilience:
  # All these failures: frost protection continues
  continues_during:
    - "Internet down"
    - "MQTT broker down"
    - "Gray Logic Core down"
    - "Database corrupted"
    - "KNX bridge down"
    - "Modbus bridge down"

  # Frost protection is only affected if:
  affected_only_by:
    - "Thermostat hardware failure"
    - "Thermostat power loss"
    - "All heating equipment failure"

  # User notification when monitoring unavailable
  monitoring_unavailable:
    condition: "Core down but heating still needed"
    user_impact: "No alerts or logging, but protection continues"
    recovery: "Review logs on Core restart to check for activations"
```

See [Climate Domain: Frost Protection Enforcement](../domains/climate.md#frost-protection-enforcement) for full specification.

---

## State Management

### State Caching

Core maintains device state in memory and persists to database:

```yaml
state_management:
  # In-memory state
  memory:
    purpose: "Fast access, real-time updates"
    persistence: "None — lost on restart"
    source: "Bridge updates via MQTT"
    
  # Database state
  database:
    purpose: "Persistence, recovery"
    update_frequency: "On state change"
    fields:
      - "last_known_state"
      - "last_update_timestamp"
      - "state_source"               # bridge | manual | assumed
      
  # Startup behavior
  startup:
    action: "Load state from database, request refresh from bridges"
    timeout_seconds: 30
    fallback: "Use database state as 'assumed'"
```

### State Reconciliation

After component recovery, states are synchronized using **timestamp-based conflict resolution**:

```yaml
state_reconciliation:
  # Bridge reconnection
  bridge_reconnect:
    action: "Bridge announces all device states with timestamps"
    core_action: "Compare timestamps, accept newer state"
    conflict_resolution: "timestamp_wins"
    conflict_window_seconds: 5      # Within 5s, bridge wins (closer to hardware)

  # Core restart
  core_restart:
    action: "Load database state, request bridge refresh"
    timeout: 30                      # Seconds to wait for bridges
    fallback: "Use database state, mark as 'assumed'"

  # MQTT broker restart
  mqtt_restart:
    action: "All bridges reconnect and announce states"
    core_action: "Accept all state updates, reconcile timestamps"
```

### Timestamp-Based Conflict Resolution

Every state update includes a timestamp. When conflicts occur:

```yaml
conflict_resolution:
  # Rule 1: Newer timestamp wins
  primary_rule: "latest_timestamp_wins"

  # Rule 2: Within threshold, bridge wins (closer to physical reality)
  close_conflict_threshold_seconds: 5
  close_conflict_winner: "bridge"

  # Rule 3: Command in-flight protection
  pending_command_protection:
    enabled: true
    window_seconds: 10              # If command was sent <10s ago, wait for ack
    action: "hold_state_update"

  # Clock Skew Protection
  clock_skew_protection:
    enabled: true
    max_acceptable_skew_seconds: 60     # Reject timestamps >60s from Core clock
    
    # Deadlock Prevention: "Time Trust on First Use"
    # If Core RTC fails (e.g., year 1970), allow ONE jump to a valid external time.
    sanity_check:
      minimum_valid_year: 2026           # Timestamps before this are considered "RTC Failure"
      action_on_rtc_failure: "Accept external timestamp if > minimum_valid_year"
    
    actions:
      within_tolerance:
        behavior: "Accept timestamp as-is"
      
      beyond_tolerance:
        behavior: "Reject bridge timestamp, use Core timestamp"
        log: "warning"
        message: "Bridge {bridge_id} clock skew detected: {skew}s"
        bridge_health: "degraded"
        
    # Why 60 seconds?
    rationale: |
      5 seconds is the conflict window. 60 seconds provides margin for:
      - Network latency (typically <100ms local)
      - Bridge processing delay (typically <50ms)
      - NTP poll intervals (typically hourly)
      Drifts beyond 60s indicate a clock problem, not normal variance.

  # Bridge Clock Health Status
  bridge_clock_health:
    status_values:
      healthy: "Clock within 5s of Core"
      warning: "Clock 5-60s from Core"
      degraded: "Clock >60s from Core (timestamps rejected)"
      unknown: "No recent timestamp received"
    
    # Health check on each bridge message
    check_trigger: "every_bridge_message"
    
    # Exposed in health API
    api_path: "/api/v1/health/bridges/{bridge_id}/clock"
    response:
      clock_offset_seconds: float
      status: string
      last_checked: timestamp

  # State update message format
  state_message:
    required_fields:
      - device_id
      - timestamp                   # ISO8601, required for reconciliation
      - state
    timestamp_source: "bridge"      # Bridge timestamps state at observation time

  # Example conflict scenarios
  scenarios:
    # User sends command while bridge was disconnected
    command_during_disconnect:
      command_sent_at: "2026-01-15T10:00:00Z"
      bridge_reconnect_at: "2026-01-15T10:00:05Z"
      bridge_reports_state_at: "2026-01-15T09:59:55Z"  # Old state
      resolution: "Ignore bridge state (older than command)"
      action: "Re-send command to bridge"

    # Bridge state newer than Core's cached state
    bridge_state_newer:
      core_cached_at: "2026-01-15T10:00:00Z"
      bridge_reports_at: "2026-01-15T10:00:10Z"
      resolution: "Accept bridge state"

    # Multiple bridges report conflicting state (shouldn't happen)
    conflicting_bridges:
      resolution: "Most recent timestamp wins"
      log: "warning"
```

### Race Condition Prevention

To prevent race conditions during recovery:

```yaml
race_condition_prevention:
  # Command tracking
  pending_commands:
    storage: "in_memory"
    ttl_seconds: 60                 # Track commands for 60s
    fields:
      - command_id
      - device_id
      - sent_at
      - acknowledged: boolean

  # State update holdoff
  holdoff:
    enabled: true
    # Don't accept state updates for device if command pending
    condition: "pending_command_exists AND age < 10s"
    action: "queue_state_update"
    queue_max_size: 100

  # Reconciliation on bridge reconnect
  reconnect_sequence:
    1: "Bridge connects and publishes health status"
    2: "Core waits 2s for bridge to stabilize"
    3: "Core requests full state refresh"
    4: "Bridge publishes all device states with timestamps"
    5: "Core reconciles each state against pending commands"
    6: "Core marks devices as 'confirmed' or 're-sends pending commands'"
```

### Assumed State

When real state is unknown, system uses "assumed" state:

```yaml
assumed_state:
  # When state is assumed
  triggers:
    - "Core restart before bridge connection"
    - "Bridge communication timeout"
    - "Database recovery without state refresh"
    
  # How assumed state is handled
  handling:
    display: "Show state with 'assumed' indicator"
    commands: "Allow commands (may not match current state)"
    automation: "Run with warning in logs"
    
  # Resolution
  resolution:
    - "Next bridge state update clears 'assumed'"
    - "Manual command confirms state"
```

---

## Graceful Degradation

### Degradation Levels

```yaml
degradation_levels:
  - level: 0
    name: "Full Operation"
    description: "All systems normal"
    indicator: "green"
    
  - level: 1
    name: "Minor Degradation"
    description: "Non-critical services unavailable"
    examples:
      - "Internet down"
      - "Weather API unavailable"
      - "Voice bridge down"
    indicator: "yellow"
    
  - level: 2
    name: "Moderate Degradation"
    description: "Some devices unreachable"
    examples:
      - "One protocol bridge down"
      - "VictoriaMetrics down"
    indicator: "orange"
    
  - level: 3
    name: "Significant Degradation"
    description: "Core functionality impaired"
    examples:
      - "MQTT broker down"
      - "Multiple bridges down"
    indicator: "red"
    
  - level: 4
    name: "Minimal Operation"
    description: "Only physical controls work"
    examples:
      - "Core down"
      - "Database corrupted"
    indicator: "red + alert"
```

### User Notification

Users are informed of degradation:

```yaml
degradation_notification:
  # UI indicators
  ui:
    wall_panel: "Status bar color + icon"
    mobile_app: "Banner notification"
    web_admin: "System status widget"
    
  # Active notifications
  notifications:
    level_1: "Log only"
    level_2: "UI indicator + admin notification"
    level_3: "UI indicator + all users notification"
    level_4: "UI indicator + push notification + SMS (if configured)"
    
  # Message format
  message:
    template: "{component} is unavailable. {impact}"
    examples:
      - "Internet is unavailable. Remote access and weather data unavailable."
      - "KNX Bridge is unavailable. Living room lights not controllable via app."
```

---

## Recovery Procedures

### Automatic Recovery

Most failures recover automatically:

```yaml
automatic_recovery:
  # Service monitoring
  monitoring:
    method: "Health checks + watchdog"
    interval_seconds: 10
    
  # Restart policies
  restart:
    core:
      method: "systemd"
      max_attempts: 5
      delay_seconds: [5, 10, 30, 60, 120]
      
    bridges:
      method: "Core-managed"
      max_attempts: 3
      delay_seconds: [10, 30, 60]
      
    mqtt:
      method: "systemd"
      max_attempts: 5
      
  # Recovery actions
  on_recovery:
    - "Log recovery event"
    - "Request state refresh"
    - "Clear degradation indicator"
    - "Resume queued operations"
```

### Manual Recovery

Some scenarios require intervention:

```yaml
manual_recovery:
  database_corruption:
    steps:
      - "Stop Gray Logic Core"
      - "Restore database from backup"
      - "Start Gray Logic Core"
      - "Verify configuration"
    documentation: "See backup.md for restore procedures"
    
  hardware_failure:
    steps:
      - "Identify failed component"
      - "Replace/repair hardware"
      - "Verify network connectivity"
      - "Restart affected services"
      
  configuration_error:
    steps:
      - "Review logs for error details"
      - "Correct configuration"
      - "Restart affected service"
      - "Verify operation"
```

### Recovery Verification

After recovery, verify operation:

```yaml
recovery_verification:
  automatic:
    - "All bridges connected and responding"
    - "Device state refresh completed"
    - "No error logs in last 5 minutes"
    - "API responding to health check"
    
  manual:
    - "Test device control from UI"
    - "Verify scene execution"
    - "Check schedule triggers"
    - "Confirm voice commands work (if applicable)"
```

---

## Offline Automation

### Schedule Handling

Schedules continue during outages:

```yaml
offline_schedules:
  # When internet is down
  internet_down:
    behavior: "Continue normally"
    time_source: "Local RTC (battery-backed)"
    astronomical: "Use cached sunrise/sunset (recalculate daily)"
    
  # When bridge is down
  bridge_down:
    behavior: "Queue commands for recovery"
    max_queue_time_minutes: 60
    overflow: "Drop oldest, log warning"
    
  # When Core is restarting
  core_restart:
    behavior: "Catch up missed schedules on restart"
    max_catch_up_minutes: 30
    catch_up_action: "Run most recent state only"
    
    # Context-Based Catch-Up Logic
    catch_up_context:
      global_conditions:
        - check: "mode"
          logic: "If current_mode == 'away', skip comfort/lighting schedules"
        - check: "presence"
          logic: "If location_empty, skip room-specific schedules"
          
      on_skip:
        log: "info"
        message: "Skipped catch-up for schedule '{name}': {reason}"
        ui_notification: true              # Alert user: "Skipped 'Good Morning' (House Empty)"
        notification_auto_dismiss_minutes: 60   # Auto-dismiss after 1 hour
```

### Context-Aware Catch-Up

To prevent "ghost actions" (e.g., blinds opening at 3 AM because Core restarted and replayed a missed "Sunset" schedule), schedules support explicit catch-up conditions.

**Configuration Field:** `catch_up_condition`

```yaml
# Example Schedule Definition
schedule_id: "good_morning"
cron: "0 7 * * *"
actions: [...]

# Catch-up rules
catch_up_condition:
  # 1. Mode check (whitelist or blacklist)
  mode_whitelist: ["home", "morning"]   # Only run if currently in these modes
  mode_blacklist: ["away", "vacation"]  # Never run if in these modes
  
  # 2. Presence check
  require_presence: true                # Only run if associated room/home is occupied
  
  # 3. Time window override
  max_delay_minutes: 15                 # Override global 30m. If >15m late, skip.
```

### Event-Based Automation

Events are processed with degraded components:

```yaml
offline_events:
  # Sensor events (bridge working)
  sensor_events:
    behavior: "Process normally"
    
  # External events (internet needed)
  external_events:
    weather_trigger:
      behavior: "Use cached weather data"
      fallback: "Skip trigger, log warning"
      
    calendar_trigger:
      behavior: "Use cached calendar (if recent)"
      fallback: "Skip trigger until sync restored"
      
  # Voice events (voice bridge needed)
  voice_events:
    behavior: "Fail silently (no TTS response)"
    user_feedback: "No audio response indicates failure"
```

---

## Health Monitoring

### Component Health Checks

```yaml
health_checks:
  core:
    endpoint: "/api/v1/health"
    interval_seconds: 30
    timeout_seconds: 5
    
  bridges:
    method: "MQTT heartbeat"
    topic: "graylogic/health/{bridge_id}"
    interval_seconds: 10
    timeout_seconds: 30
    
  mqtt:
    method: "Connection status"
    reconnect_interval_seconds: 5
    
  database:
    method: "Query test"
    query: "SELECT 1"
    interval_seconds: 60
    
  tsdb:
    method: "Ping"
    interval_seconds: 60
```

### Health Status API

```http
GET /api/v1/health
```

**Response:**
```json
{
  "status": "degraded",
  "degradation_level": 1,
  "timestamp": "2026-01-13T10:30:00Z",
  "components": {
    "core": {
      "status": "healthy",
      "uptime_seconds": 86400
    },
    "mqtt": {
      "status": "healthy",
      "connected": true
    },
    "database": {
      "status": "healthy",
      "last_write": "2026-01-13T10:29:55Z"
    },
    "victoriametrics": {
      "status": "healthy",
      "connected": true
    },
    "bridges": {
      "knx": {
        "status": "healthy",
        "connected": true,
        "devices": 45
      },
      "dali": {
        "status": "unhealthy",
        "connected": false,
        "last_seen": "2026-01-13T10:25:00Z",
        "error": "Connection timeout"
      }
    },
    "internet": {
      "status": "unavailable",
      "last_seen": "2026-01-13T09:00:00Z"
    }
  },
  "issues": [
    {
      "component": "bridges.dali",
      "severity": "warning",
      "message": "DALI bridge not responding"
    },
    {
      "component": "internet",
      "severity": "info",
      "message": "Internet connection unavailable"
    }
  ]
}
```

---

## Testing Offline Behavior

### Commissioning Tests

```yaml
commissioning_offline_tests:
  - name: "Internet disconnection"
    steps:
      - "Disconnect internet"
      - "Verify all local control works"
      - "Verify voice commands work"
      - "Verify schedules run"
      - "Reconnect internet"
      - "Verify recovery"
    pass_criteria: "All local functions work without internet"
    
  - name: "Bridge failure simulation"
    steps:
      - "Stop KNX bridge"
      - "Verify other devices work"
      - "Verify physical KNX controls work"
      - "Restart KNX bridge"
      - "Verify state synchronization"
    pass_criteria: "Graceful degradation, automatic recovery"
    
  - name: "Core restart"
    steps:
      - "Restart Gray Logic Core"
      - "Verify state restoration"
      - "Verify schedules resume"
      - "Verify no missed automations"
    pass_criteria: "Clean restart with state recovery"
```

### Periodic Verification

```yaml
periodic_verification:
  schedule: "Monthly"
  
  tests:
    - name: "Simulated internet outage"
      duration_minutes: 30
      verify:
        - "All local control"
        - "Schedules run"
        - "Voice works"
        
    - name: "Database backup restore"
      procedure: "Restore from backup, verify config"
      
  documentation: "Record results in maintenance log"
```

---

## Time Synchronization

Accurate time is critical for schedules, astronomical calculations, and state reconciliation.

### NTP Configuration

```yaml
time_synchronization:
  # Primary: Local NTP server (if available)
  ntp:
    enabled: true
    servers:
      - "192.168.1.1"               # Local router/firewall
      - "time.google.com"           # Fallback (if internet available)
      - "pool.ntp.org"              # Secondary fallback
    poll_interval_seconds: 3600     # Check every hour

  # Bridge NTP Requirements (MANDATORY)
  bridge_ntp:
    requirement: "All bridges MUST synchronize to NTP"
    
    # Local NTP pool (offline-first)
    preferred_sources:
      - "Core server (local NTP relay)"    # Primary: Core relays NTP locally
      - "Local router/gateway"              # Fallback: Router as NTP source
      - "LAN time server"                   # Alternative: Dedicated LAN server
    
    # Internet NTP (fallback only)
    fallback_sources:
      - "pool.ntp.org"                      # Only if local sources unavailable
    
    # Why local-first?
    rationale: |
      Bridges must work during internet outages. Using the Core server or 
      local router as the NTP source ensures time sync continues offline.
      This maintains the offline-first principle while ensuring clock accuracy.
    
    # Configuration examples
    examples:
      raspberry_pi: |
        # /etc/systemd/timesyncd.conf
        [Time]
        NTP=192.168.1.10 192.168.1.1     # Core server, router
        FallbackNTP=pool.ntp.org
        
      embedded_linux: |
        # BusyBox ntpd
        ntpd -p 192.168.1.10 -p 192.168.1.1

  # Hardware RTC (battery-backed)
  rtc:
    required: true                  # Server must have battery-backed RTC
    drift_check_interval_hours: 24
    max_acceptable_drift_seconds: 60

  # Time uncertainty handling
  uncertainty:
    max_tolerable_drift_minutes: 5
    actions_when_uncertain:
      schedules: "use_last_known_time"
      state_timestamps: "mark_uncertain"
      astronomical: "skip_until_sync"
```

### Drift Handling

When time source is unavailable or RTC has drifted:

```yaml
drift_handling:
  # Detection
  detect_drift:
    method: "compare_ntp_vs_rtc"
    threshold_seconds: 60

  # Behavior when drift detected
  on_drift_detected:
    - log_warning: "System time may be inaccurate"
    - mark_timestamps: "uncertain"
    - notify_admin: true

  # Behavior when NTP unavailable
  ntp_unavailable:
    fallback: "rtc"
    max_rtc_age_days: 7             # After 7 days without NTP, warn user
    actions:
      schedules: "continue"          # Use RTC time
      astronomical: "continue"       # May be off by minutes
      state_reconciliation: "widen_conflict_window"

  # Recovery when NTP restored
  ntp_restored:
    - sync_immediately: true
    - log_time_jump: true
    - re_evaluate_missed_schedules: true
```

### Time Requirements

| Component | Time Accuracy Required | Consequence if Inaccurate |
|-----------|----------------------|--------------------------|
| Schedules | ±5 minutes | Triggers at wrong time |
| Astronomical | ±2 minutes | Sunrise/sunset off |
| State timestamps | ±5 seconds | Reconciliation may fail |
| Audit logs | ±1 second | Investigation harder |
| TLS certificates | ±1 hour | Connection failures |

### Commissioning Checklist

```yaml
time_commissioning:
  # Core Server Commissioning
  core_server:
    - [ ] Server has battery-backed RTC
    - [ ] RTC battery tested (holds time on power loss)
    - [ ] NTP server configured and reachable
    - [ ] Timezone set correctly
    - [ ] Time verified accurate after reboot
    - [ ] Astronomical calculations verified (sunrise time matches reality)

  # Bridge Time Commissioning (MANDATORY for each bridge)
  bridge_time_commissioning:
    - [ ] Bridge has working hardware RTC or persistent clock
    - [ ] NTP client configured on bridge
    - [ ] NTP source is local (Core server or router), not internet-only
    - [ ] Verified: Bridge time matches Core within 5 seconds
    - [ ] Document: NTP source used for this bridge
    
  # Verification Command (run from Core server)
  verification:
    command: "mosquitto_pub -t 'graylogic/command/{bridge_id}/time_check' -m '{}'"
    expected_response:
      topic: "graylogic/status/{bridge_id}/time_check"
      payload:
        bridge_time: "ISO8601 timestamp"
        offset_from_core_seconds: "float (must be <5)"
```

---

## Configuration

### Offline Behavior Settings

```yaml
# /etc/graylogic/resilience.yaml
offline:
  # State management
  state:
    persist_on_change: true
    refresh_on_startup: true
    assumed_state_timeout_minutes: 5
    
  # Recovery
  recovery:
    auto_restart_bridges: true
    max_bridge_restart_attempts: 3
    bridge_restart_delay_seconds: 10
    
  # Queue management
  queues:
    command_queue_size: 1000
    tsdb_queue_size_mb: 100
    queue_overflow_action: "drop_oldest"
    
  # Health monitoring
  health:
    check_interval_seconds: 30
    bridge_heartbeat_timeout_seconds: 60
    
  # Notifications
  notifications:
    notify_on_degradation: true
    notify_on_recovery: true
    min_severity: 2                  # Only level 2+ degradation
```

---

## Related Documents

- [Principles](../overview/principles.md) — Offline-first philosophy
- [System Overview](../architecture/system-overview.md) — Architecture
- [Core Internals](../architecture/core-internals.md) — State management
- [Backup and Recovery](backup.md) — Disaster recovery
- [MQTT Protocol](../protocols/mqtt.md) — Bridge communication
