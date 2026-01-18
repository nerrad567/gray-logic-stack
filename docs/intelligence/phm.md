---
title: Predictive Health Monitoring (PHM) Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/core-internals.md
  - architecture/energy-model.md
  - protocols/mqtt.md
---

# Predictive Health Monitoring (PHM) Specification

This document specifies Gray Logic's approach to Predictive Health Monitoring — detecting equipment problems before they cause failures, optimizing maintenance schedules, and reducing operational costs.

---

## Overview

### What is PHM?

Predictive Health Monitoring uses statistical analysis of equipment telemetry to:

1. **Establish baselines** — Learn normal operating patterns
2. **Detect anomalies** — Identify deviations from normal behavior
3. **Predict failures** — Estimate time-to-failure based on degradation trends
4. **Prioritize maintenance** — Focus attention on equipment that needs it most

### Why PHM?

| Traditional Maintenance | With PHM |
|------------------------|----------|
| Fixed schedules (wasteful) | Condition-based (efficient) |
| Reactive repairs (expensive) | Predictive intervention (planned) |
| Unexpected downtime | Advance warning |
| Over-maintenance of healthy equipment | Targeted maintenance |

### Design Principles

1. **Local processing** — All PHM runs on-site, no cloud dependency
2. **Learn, don't assume** — Baseline from actual equipment behavior, not specs
3. **Context-aware** — Consider operating conditions when detecting anomalies
4. **Actionable alerts** — Every alert includes recommended action
5. **Non-intrusive** — PHM observes, never interferes with operation

---


---

## PHM Capability Tiers

To align expectations with installed hardware, PHM features are grouped into three capability tiers.

### Tier 1: Standard (Native Intelligence)
**Requires:** No additional hardware. Works with standard device telemetry and protocol diagnostics.
**Features:**
- **Protocol Diagnostics:** DALI lamp failures, KNX device connectivity, Zigbee signal quality (LQI).
- **Runtime Stats:** Runtime hours, cycle counting.
- **Device Health:** Battery levels, error codes, "device offline" alerts.
- **Inferred Logic:** "Blinds moving too slow" (timestamp delta), "Valve stuck" (retry count).

**Target:** ALL Gray Logic installations. Base level functionality.

### Tier 2: Enhanced (Power & Energy)
**Requires:** Power measurement hardware (CT clamps, smart plugs, energy meters).
**Features:**
- Energy consumption profiling
- Current draw anomalies ("Pump blocked", "Motor strain")
- "Phantom load" detection
- Efficiency warnings (e.g., HVAC usage vs. cooling effect)

**Target:** High-end residential, light commercial.

### Tier 3: Advanced (Physical Sensing)
**Requires:** Dedicated industrial sensors (Vibration, Temperature probes, Pressure).
**Features:**
- Vibration analysis (bearing wear)
- Thermal imaging/profiling
- Predictive failure modeling (time-to-failure)
- Precise efficiency calculation (COP)

**Target:** Commercial plant rooms, critical infrastructure, luxury marine.

### Typical Deployment Expectations

| Feature | Typical Residential | Commercial / High-End |
|---------|---------------------|-----------------------|
| Light diagnostics | ✅ Tier 1 (DALI Status) | ✅ Tier 1 (DALI Status) |
| Bridge health | ✅ Tier 1 | ✅ Tier 1 |
| Pump status | ✅ Tier 1 (On/Off) | ✅ Tier 2 (Power) / 3 (Vibration) |
| HVAC health | ✅ Tier 1 (Errors) | ✅ Tier 3 (Pressures/Deltas) |
| **Result** | Maintenance reminders | Predictive failure alerts |

---

## Architecture

### Data Flow

```
Equipment → Bridges → MQTT → Core → InfluxDB
                              ↓
                         PHM Engine
                              ↓
                      ┌───────┴───────┐
                      ↓               ↓
               Anomaly Alerts    Health Scores
                      ↓               ↓
               Notifications     Dashboard
```

### Components

| Component | Responsibility |
|-----------|----------------|
| **Bridges** | Collect raw telemetry from equipment |
| **InfluxDB** | Store time-series data for analysis |
| **PHM Engine** | Baseline learning, anomaly detection, predictions |
| **Alert Service** | Generate and route PHM alerts |
| **API/Dashboard** | Expose health scores and recommendations |

### PHM Engine Location

The PHM Engine runs within Gray Logic Core as a background service:

```yaml
# Core configuration
phm:
  enabled: true
  analysis_interval_minutes: 15     # Run analysis every 15 min
  baseline_learning_days: 7         # Days to learn baseline
  retention_days: 365               # Keep PHM history
  
  # InfluxDB connection
  influxdb:
    url: "http://localhost:8086"
    org: "graylogic"
    bucket: "phm"
```

---

## PHM Lifecycle

### 1. Data Collection

Equipment telemetry flows through bridges to InfluxDB:

```yaml
# Example: Pump telemetry written to InfluxDB
measurement: equipment_telemetry
tags:
  device_id: "pump-chw-1"
  device_type: "pump"
  location: "plantroom"
fields:
  power_kw: 5.2
  current_a: 12.1
  speed_percent: 75
  vibration_mm_s: 2.3
  bearing_temp_c: 45
  flow_rate_lps: 8.5
timestamp: 2026-01-12T10:30:00Z
```

#### External Monitoring via Device Associations

Many devices don't have built-in monitoring capability. PHM supports **external sensors** via device associations — the sensor's readings are attributed to the target device.

```
┌─────────────────┐         ┌─────────────────┐
│ Pump (dumb)     │←────────│ CT Clamp        │
│ No monitoring   │ monitors│ Reports power   │
└─────────────────┘         └─────────────────┘
```

**Configuration:**

```yaml
# 1. Define the external sensor
devices:
  - id: "ct-clamp-pump-1"
    type: "ct_clamp"
    protocol: "modbus_tcp"
    address:
      host: "192.168.1.100"
      register: 40001
    capabilities: ["power_read", "energy_read", "current_read"]

# 2. Define the target device (no monitoring capability)
  - id: "pump-chw-1"
    type: "pump"
    protocol: "relay"            # Controlled via relay, no direct comms

# 3. Create association
associations:
  - id: "assoc-ct-pump-1"
    source_device_id: "ct-clamp-pump-1"
    target_device_id: "pump-chw-1"
    type: "monitors"
    config:
      metrics: ["power_kw", "energy_kwh", "current_a"]
```

**How it works:**

1. CT clamp reports `power_kw: 5.2` via Modbus bridge
2. State Manager receives state for `ct-clamp-pump-1`
3. Association Resolver finds monitoring association to `pump-chw-1`
4. State Manager attributes reading: `pump-chw-1.power_kw = 5.2`
5. PHM processes `pump-chw-1` power as if the pump reported it directly
6. Energy model attributes consumption to the pump

**Common external monitoring scenarios:**

| Target Equipment | External Sensor | PHM Metrics |
|-----------------|-----------------|-------------|
| Dumb pump | CT clamp | Power, current, energy |
| Old boiler | External temp sensors | Flue temp, flow temp |
| Mechanical fan | Vibration sensor | Vibration level |
| Water heater | Smart plug | Power, energy, on/off cycles |
| Lighting circuit | DIN rail meter | Circuit power, energy |

See [Data Model: DeviceAssociation](../data-model/entities.md#deviceassociation) for full association specification.

### 1a. Sensor Health Validation

Before data enters the baseline or anomaly engines, it must pass validation to filter out sensor failures.

```yaml
sensor_validation:
  # 1. Range Validity (Physical Plausibility)
  range_check:
    - parameter: "temperature_c"
      min: -30
      max: 120
      action: "discard_and_flag_sensor"
    - parameter: "power_kw"
      min: 0
      max: 1000
      action: "discard"

  # 2. State Correlation (Logical Consistency)
  # Prevents "Machine Running + Zero Power" scenarios
  # 2. State Correlation (Logical Consistency)
  # Prevents "Machine Running + Zero Power" scenarios
  correlation_check:
    - device_id: "pump-chw-1"
      condition: "state == 'on'"
      expect:
        parameter: "power_kw"
        operator: ">"
        value: 0.1
      on_fail:
        # Differentiate based on state confidence
        logic: |
          IF state.source == 'feedback' THEN
             flag_sensor("ct-clamp-pump-1", "Pump confirmed ON (Feedback), but Power is 0")
          ELSE
             flag_ambiguous("Pump Commanded ON, but Power is 0. Check Pump Start or Contactor.")


  # 3. Stuck Value Detection
  stuck_check:
    method: "variance_over_time"
    window_minutes: 60
    min_variance: 0.001
    action: "warn_sensor_stuck"
```

**Handling Invalid Data:**
*   **Invalid readings** trigger a `sensor_fault` alert, NOT a machinery alert.
*   **Sensor health score** degrades (affecting the monitoring confidence).
*   **Baseline learning** pauses for that parameter until sensor is fixed.

### 1b. Dual Fault Awareness

When sensor correlation fails, PHM must consider BOTH sensor failure AND equipment failure possibilities:

```yaml
DualFaultAwareness:
  # When sensor correlation fails (e.g., "Pump ON but Power = 0")
  on_correlation_failure:
    primary_action: "flag_sensor"
    secondary_action:
      notify: true
      severity: "warning"
      title: "Correlation Failure - Verify Equipment"
      message: |
        Correlation failure detected for {device_id}.
        Possible causes:
        1. Sensor failure ({sensor_id}) - most likely
        2. Equipment failure ({device_id}) - verify manually
        3. Wiring issue - check connections

    # Require human verification
    require_verification: true
    verification_prompt: "Please confirm equipment status after sensor repair"

  # Extra caution during learning phase
  during_learning:
    correlation_failure_action:
      escalate: true
      require_manual_verification: true
      prevent_assumption: true   # Don't assume sensor-only failure
```

### 2. Baseline Learning

During the learning period, PHM establishes normal operating patterns:

```yaml
PHMBaseline:
  device_id: "pump-chw-1"
  parameter: "vibration_mm_s"
  
  # Learning configuration
  learning:
    start: "2026-01-01T00:00:00Z"
    end: "2026-01-08T00:00:00Z"
    samples: 10080                  # 1-minute samples
    status: "complete"
  
  # Calculated baseline
  baseline:
    mean: 2.1
    std_dev: 0.3
    min: 1.5
    max: 3.2
    percentile_95: 2.8
    percentile_99: 3.1
    
  # Context-aware baselines (optional)
  contexts:
    # Different baselines for different operating speeds
    speed_percent:
      50: { mean: 1.8, std_dev: 0.2 }
      75: { mean: 2.1, std_dev: 0.25 }
      100: { mean: 2.5, std_dev: 0.3 }
      
  # Auto-calculated thresholds
  thresholds:
    warning: 3.3                    # mean + 4σ
    alarm: 3.9                      # mean + 6σ
    critical: 7.1                   # External limit (ISO 10816)
```

### Device-Type-Specific Baseline Requirements

Different device types have vastly different feedback characteristics. PHM must account for this:

```yaml
baseline_requirements:
  # Category 1: Immediate Feedback Devices
  # These devices report status immediately - anomalies detectable right away
  immediate_feedback:
    description: "Devices that report status/faults immediately"
    examples:
      - "DALI emergency lighting (battery status, lamp failures)"
      - "Smart plugs (power, energy)"
      - "Digital sensors (temperature, humidity, CO2)"

    baseline_config:
      min_data_points: 100          # Just need enough for statistical validity
      typical_learning_hours: 24    # Can establish baseline in hours
      anomaly_detection: "threshold_based"  # Simple threshold usually sufficient

    # DALI emergency lighting example
    dali_emergency:
      feedback_type: "immediate"
      telemetry:
        - battery_level: "query on schedule"
        - lamp_failure: "broadcast immediately"
        - duration_test_result: "after test"
      baseline_learning: "minimal"  # Mostly threshold-based, not statistical
      anomaly_examples:
        - "Battery level drops below 80%"
        - "Lamp failure flag set"
        - "Duration test failed"

  # Category 2: Gradual Degradation Devices
  # These devices develop problems over time - need trend analysis
  gradual_degradation:
    description: "Devices where anomalies develop over days/weeks/months"
    examples:
      - "Motors (current increase, vibration, bearing temperature)"
      - "Pumps (flow rate decrease, pressure differential)"
      - "HVAC compressors (efficiency degradation)"
      - "Fans (vibration, noise, airflow reduction)"

    baseline_config:
      min_data_points: 2000         # Need substantial history
      typical_learning_days: 14-30  # Weeks to establish reliable baseline
      anomaly_detection: "statistical_trend"  # Need trend analysis

    # Motor/pump example
    motor_pump:
      feedback_type: "gradual"
      telemetry:
        - power_kw: "continuous via CT clamp"
        - current_a: "continuous via CT clamp"
        - vibration_mm_s: "periodic via sensor (if equipped)"
        - temperature_c: "periodic via sensor"
      baseline_learning: "extensive"
      protection_limits:
        vibration_max: 7.1          # mm/s (ISO 10816)
        temp_max_c: 90              # Bearing limit
      typical_degradation_patterns:
        - pattern: "Bearing wear"
          indicators: ["vibration increase", "temperature increase"]
          typical_timeline: "weeks to months"
        - pattern: "Impeller wear"
          indicators: ["flow decrease", "power increase for same speed"]
          typical_timeline: "months to years"

  # Category 3: Event-Based Devices
  # Anomalies detected by event patterns, not continuous telemetry
  event_based:
    description: "Devices where anomalies appear in event patterns"
    examples:
      - "Motorized blinds (travel time increase)"
      - "Door locks (operation count, battery)"
      - "Valve actuators (stroke time)"

    baseline_config:
      min_events: 50                # Need enough events to establish pattern
      typical_learning_days: 7-14   # Depends on usage frequency
      anomaly_detection: "event_pattern"

    # Motorized blind example
    blind_actuator:
      feedback_type: "event_pattern"
      events:
        - open_command: "timestamp"
        - fully_open: "timestamp"
        - travel_time_ms: "calculated"
      baseline: "travel_time_ms distribution"
      anomaly_indicator: "travel_time exceeds baseline by 2σ"

  # Category 4: Inferred Health Devices
  # No direct health telemetry - infer from operation patterns
  inferred_health:
    description: "Devices with no health telemetry - infer from usage"
    examples:
      - "Simple relays (cycle count, estimated lifespan)"
      - "Contactors (cycle count, thermal stress estimation)"
      - "Solenoid valves (operation count)"

    baseline_config:
      method: "lifecycle_estimation"
      data_required: ["operation_count", "load_at_switch"]

    # Relay example
    relay:
      feedback_type: "inferred"
      tracked_metrics:
        - operation_count: "from command log"
        - load_current_at_switch: "from associated CT clamp"
      health_estimation:
        method: "cycle_counting"
        rated_cycles: 100000        # From datasheet
        derating_for_inductive: 0.5 # 50% derating for inductive loads
        alert_at_percent: 80        # Alert at 80% of estimated life
```

### Baseline Learning Status

PHM reports baseline status per device to indicate readiness:

```yaml
baseline_status:
  states:
    insufficient_data:
      description: "Not enough data points yet"
      phm_enabled: false
      ui_indicator: "Learning... ({percent}%)"

    learning:
      description: "Actively collecting baseline data"
      phm_enabled: "limited"        # Only extreme anomalies flagged
      ui_indicator: "Learning ({days_remaining} days)"

    ready:
      description: "Baseline established, full PHM active"
      phm_enabled: true
      ui_indicator: "Monitoring"

    stale:
      description: "Baseline too old, may not reflect current normal"
      stale_after_days: 90             # Baseline older than 90 days is considered stale
      phm_enabled: "with_warning"
      ui_indicator: "Baseline outdated"
      action: "Offer to re-learn baseline"

  # Behavior during learning period
  during_learning:
    protection:
      # Level 1: Safe Operating Limits (Absolute physical limits)
      safe_operating_limits:
        enabled: true
        source: "device_profile"    # e.g., "Pump -> ISO 10816 limit"
        
      # Level 2: Reference Baseline (Generic "good" values)
      reference_baseline:
        enabled: true
        source: "manufacturer_spec" # or "similar_device_average"
        multiplier: 1.5             # Alert if > 1.5x reference
        
      # Level 3: Learning Baseline (The one being calculated)
      learning_baseline:
        extreme_only: true          # Only alert if > 3x current (unstable) mean
```

### Safe Operating Limits (Day 0 Protection)

To prevent the "learning blind spot" (where developing faults validly occur during the 7-30 day learning phase), PHM enforces multiple protection layers:

1.  **Safe Operating Limits (SOL):** Hard physical limits that apply **always**, regardless of learning state.
    *   *Example:* ISO 10816 Class II vibration limit (7.1 mm/s). If exceeded, PHM alerts immediately, even on Day 1.
    *   *Example:* Bearing temperature > 90°C.
2.  **Commissioning Baseline (Golden Trace):**
    *   For critical equipment, the [Commissioning Checklist](../deployment/commissioning-checklist.md) requires capturing a "Golden Trace" (short-term baseline).
    *   This serves as a temporary reference until the full statistical baseline is established.
3.  **Reference Baselines:**
    *   Generic "known good" profiles for common equipment (e.g., "Standard 10W LED Driver").
    *   Used as a fallback comparison point during the learning phase.

### 3. Anomaly Detection

PHM continuously compares current values against baseline:

```yaml
PHMAnalysis:
  device_id: "pump-chw-1"
  parameter: "vibration_mm_s"
  timestamp: "2026-01-12T10:30:00Z"
  
  # Current reading
  current:
    value: 3.8
    context:
      speed_percent: 75
      
  # Analysis against baseline
  analysis:
    expected_mean: 2.1              # From baseline for speed=75
    expected_std_dev: 0.25
    zscore: 6.8                     # (3.8 - 2.1) / 0.25
    percentile: 99.99
    
  # Detection result
  detection:
    anomaly: true
    severity: "warning"             # Crossed warning threshold
    confidence: 0.95
```

### 4. Trend Analysis & Prediction

PHM tracks degradation over time to predict failures:

```yaml
PHMTrend:
  device_id: "pump-chw-1"
  parameter: "vibration_mm_s"
  
  # Rolling window analysis
  window:
    start: "2026-01-05T00:00:00Z"
    end: "2026-01-12T00:00:00Z"
    
  # Trend calculation
  trend:
    direction: "increasing"
    rate: 0.15                      # mm/s per week
    r_squared: 0.87                 # Trend fit quality
    
  # Failure prediction
  prediction:
    threshold: "alarm"              # Predicting when we'll hit alarm
    threshold_value: 3.9
    current_value: 3.8
    days_to_threshold: 0.7          # Less than 1 day!
    confidence: 0.82
    
  # Recommendation
  recommendation:
    action: "schedule_inspection"
    urgency: "high"
    message: "Vibration trending toward alarm level. Schedule bearing inspection within 24 hours."
```

### 5. Health Score Calculation

Each device gets an overall health score (0-100):

```yaml
PHMHealthScore:
  device_id: "pump-chw-1"
  timestamp: "2026-01-12T10:30:00Z"
  
  # Overall score
  health_score: 62                  # 0-100, lower = worse
  status: "degraded"                # healthy, degraded, critical
  
  # Component scores
  components:
    - parameter: "vibration_mm_s"
      score: 45
      weight: 0.4
      status: "warning"
      
    - parameter: "bearing_temp_c"
      score: 78
      weight: 0.3
      status: "healthy"
      
    - parameter: "power_kw"
      score: 85
      weight: 0.3
      status: "healthy"
      
  # Calculation: weighted average
  # 45*0.4 + 78*0.3 + 85*0.3 = 18 + 23.4 + 25.5 = 66.9 ≈ 62 (with penalties)
```

---

## Domain-Specific PHM

### Equipment Categories

| Category | PHM Value | Typical Parameters |
|----------|-----------|-------------------|
| **Plant Equipment** | ★★★★★ | Vibration, temperature, current, pressure |
| **HVAC** | ★★★★☆ | COP, runtime, cycle count, temperature delta |
| **Lighting** | ★★★☆☆ | Runtime hours, lumen depreciation, error rate |
| **Blinds/Shades** | ★★☆☆☆ | Motor runtime, travel time, operation count |
| **Audio** | ★★☆☆☆ | Amplifier temperature, response latency |
| **Network/Bridges** | ★★★☆☆ | Error rate, latency, packet loss |

### Plant Equipment

**Primary PHM targets** — highest value, most data available.

| Equipment | Key Parameters | Alert Indicators |
|-----------|---------------|------------------|
| **Pumps** | Vibration, bearing temp, current, flow | Bearing wear, impeller damage, cavitation |
| **Fans** | Vibration, current, speed, DP | Belt wear, bearing failure, imbalance |
| **Compressors** | Current, pressures, temperature | Refrigerant leak, valve wear, motor degradation |
| **VFDs** | DC bus voltage, heatsink temp, current | Capacitor aging, overheating |
| **Boilers** | Flue temp, ignition cycles, modulation | Burner fouling, heat exchanger scaling |

```yaml
phm_plant:
  devices:
    - device_id: "pump-chw-1"
      type: "pump"
      parameters:
        - name: "vibration_mm_s"
          source: "modbus:40001"
          baseline_method: "context_aware"   # By speed
          context_parameter: "speed_percent"
          external_limit: 7.1                # ISO 10816 Class II
          
        - name: "bearing_temp_c"
          source: "modbus:40003"
          baseline_method: "rolling_mean"
          deviation_threshold_c: 10
          absolute_limit: 80
          
        - name: "power_kw"
          source: "modbus:40005"
          baseline_method: "context_aware"
          context_parameter: "speed_percent"
          deviation_threshold_percent: 15
```

### Climate/HVAC Equipment

| Equipment | Key Parameters | Alert Indicators |
|-----------|---------------|------------------|
| **Heat pumps** | COP, defrost frequency, compressor current | Refrigerant loss, fouling |
| **Boilers** | Efficiency, ignition cycles, modulation | Scaling, burner issues |
| **Valve actuators** | Runtime, travel time, position accuracy | Actuator wear, valve sticking |
| **Thermostats** | Temperature accuracy, battery | Sensor drift, battery end-of-life |

```yaml
phm_climate:
  devices:
    - device_id: "heatpump-main"
      type: "heat_pump"
      parameters:
        - name: "cop"
          calculation: "heat_output_kw / electrical_input_kw"
          baseline_method: "seasonal_context"
          context_parameter: "outdoor_temp_c"
          deviation_threshold: 0.5
          
        - name: "defrost_count"
          source: "modbus:40020"
          baseline_method: "daily_count"
          threshold_per_day: 10
          
        - name: "compressor_current_a"
          source: "modbus:40015"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 20
```

### Lighting Systems

| Equipment | Key Parameters | Alert Indicators |
|-----------|---------------|------------------|
| **LED drivers** | Runtime hours, output current | Driver aging, thermal issues |
| **Lamps/luminaires** | Runtime, lumen output, failures | Lumen depreciation, end-of-life |
| **DALI buses** | Error rate, response time | Wiring issues, device failures |
| **Switches/sensors** | Activation count, battery | Mechanical wear, battery EOL |

```yaml
phm_lighting:
  devices:
    - device_id: "dali-driver-1"
      type: "led_driver"
      parameters:
        - name: "runtime_hours"
          source: "dali:query_runtime"
          threshold_hours: 50000           # L70 rated life
          alert_at_percent: 80             # Alert at 40,000 hours
          
        - name: "actual_vs_commanded"
          calculation: "abs(actual_level - commanded_level)"
          threshold: 10                    # >10% deviation
          
        - name: "failure_count"
          source: "dali:query_failures"
          baseline_method: "none"
          threshold: 1                     # Any failure
          
    - device_id: "dali-bus-1"
      type: "dali_bus"
      parameters:
        - name: "error_rate"
          calculation: "errors / total_commands"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 200  # 2x baseline
          
        - name: "response_time_ms"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 50
```

### Blinds and Shades

| Equipment | Key Parameters | Alert Indicators |
|-----------|---------------|------------------|
| **Motors** | Runtime, current draw | Motor wear, binding |
| **Mechanisms** | Travel time, operation count | Track wear, tension issues |
| **Actuators** | Position accuracy | Calibration drift |

```yaml
phm_blinds:
  devices:
    - device_id: "blind-living-1"
      type: "motorized_blind"
      parameters:
        - name: "travel_time_s"
          baseline_method: "initial_calibration"
          deviation_threshold_percent: 20
          alert: "Travel time increased - check mechanism"
          
        - name: "motor_current_a"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 30
          alert: "Motor current elevated - check for binding"
          
        - name: "operation_count"
          threshold: 100000
          alert_at_percent: 80
```

### Audio Systems

| Equipment | Key Parameters | Alert Indicators |
|-----------|---------------|------------------|
| **Amplifiers** | Temperature, current draw | Overheating, component aging |
| **Speakers** | Response (if measurable) | Driver degradation |
| **Matrix systems** | Response latency, errors | Control board issues |

```yaml
phm_audio:
  devices:
    - device_id: "amp-zone-1"
      type: "amplifier"
      parameters:
        - name: "heatsink_temp_c"
          baseline_method: "context_aware"
          context_parameter: "output_power_w"
          deviation_threshold_c: 15
          absolute_limit: 85
          
        - name: "response_latency_ms"
          baseline_method: "rolling_mean"
          threshold_ms: 500
          
    - device_id: "matrix-main"
      type: "audio_matrix"
      parameters:
        - name: "command_error_rate"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 200
```

### Network and Bridges

| Component | Key Parameters | Alert Indicators |
|-----------|---------------|------------------|
| **Protocol bridges** | Error rate, latency, reconnects | Communication issues |
| **MQTT broker** | Message rate, queue depth | Performance degradation |
| **Network switches** | Port errors, bandwidth | Hardware issues |

```yaml
phm_infrastructure:
  devices:
    - device_id: "bridge-knx"
      type: "protocol_bridge"
      parameters:
        - name: "error_rate"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 200
          
        - name: "reconnect_count"
          baseline_method: "daily_count"
          threshold_per_day: 3
          
        - name: "message_latency_ms"
          baseline_method: "percentile_95"
          threshold_ms: 100
```

---

## Baseline Methods

### Available Methods

| Method | Description | Best For |
|--------|-------------|----------|
| `rolling_mean` | Mean over sliding window | Steady-state values |
| `rolling_percentile` | Percentile over window | Values with outliers |
| `seasonal_context` | Baseline varies by season | Outdoor-dependent equipment |
| `context_aware` | Baseline varies by operating point | Variable-speed equipment |
| `initial_calibration` | Fixed from commissioning | Mechanical systems |
| `daily_count` | Count per day | Event-based metrics |
| `none` | No baseline, fixed threshold | Binary conditions |

### Context-Aware Baselines

For equipment where "normal" depends on operating conditions:

```yaml
# Pump vibration varies with speed
baseline:
  method: "context_aware"
  parameter: "vibration_mm_s"
  context:
    parameter: "speed_percent"
    ranges:
      - range: [0, 40]
        baseline: { mean: 1.2, std_dev: 0.15 }
      - range: [40, 70]
        baseline: { mean: 1.8, std_dev: 0.2 }
      - range: [70, 100]
        baseline: { mean: 2.5, std_dev: 0.3 }
```

### Seasonal Baselines

For equipment affected by outdoor conditions:

```yaml
# Heat pump COP varies with outdoor temperature
baseline:
  method: "seasonal_context"
  parameter: "cop"
  context:
    parameter: "outdoor_temp_c"
    ranges:
      - range: [-10, 0]
        baseline: { mean: 2.5, std_dev: 0.2 }
      - range: [0, 10]
        baseline: { mean: 3.2, std_dev: 0.25 }
      - range: [10, 20]
        baseline: { mean: 4.0, std_dev: 0.3 }
```

---

## Alerting

### Alert Severities

| Severity | Meaning | Response |
|----------|---------|----------|
| `info` | Notable but not actionable | Log only |
| `warning` | Degradation detected | Schedule maintenance |
| `alarm` | Significant issue | Prioritize maintenance |
| `critical` | Imminent failure risk | Immediate action required |

### Alert Structure

```yaml
PHMAlert:
  id: "phm-alert-12345"
  timestamp: "2026-01-12T10:30:00Z"
  
  # Source
  device_id: "pump-chw-1"
  device_name: "Chilled Water Pump 1"
  parameter: "vibration_mm_s"
  
  # Alert details
  severity: "warning"
  type: "threshold_exceeded"        # or "trend_detected", "anomaly"
  
  # Values
  current_value: 3.8
  threshold: 3.3
  baseline_mean: 2.1
  deviation_percent: 81
  
  # Context
  context:
    speed_percent: 75
    runtime_hours: 12450
    
  # Prediction (if available)
  prediction:
    days_to_alarm: 14
    confidence: 0.78
    
  # Recommendation
  recommendation:
    action: "schedule_inspection"
    urgency: "medium"
    message: "Vibration level elevated. Schedule bearing inspection within 2 weeks."
    work_order_type: "preventive_maintenance"
```

### Alert Routing

```yaml
alert_routing:
  - severity: "critical"
    actions:
      - notify:
          channels: ["push", "sms", "email"]
          recipients: ["facility_manager", "on_call"]
      - create_work_order:
          priority: "emergency"
      - log_audit: true
      
  - severity: "alarm"
    actions:
      - notify:
          channels: ["push", "email"]
          recipients: ["facility_manager"]
      - create_work_order:
          priority: "high"
          
  - severity: "warning"
    actions:
      - notify:
          channels: ["email"]
          recipients: ["maintenance_team"]
      - create_work_order:
          priority: "normal"
          
  - severity: "info"
    actions:
      - log_only: true
```

---

## Dashboard & Reporting

### Health Overview

```yaml
PHMDashboard:
  timestamp: "2026-01-12T10:30:00Z"
  
  # Summary stats
  summary:
    total_devices: 45
    healthy: 38
    degraded: 5
    warning: 2
    critical: 0
    
  # Overall health score
  site_health_score: 87
  
  # By category
  categories:
    - name: "Plant Equipment"
      device_count: 8
      health_score: 72
      issues: 2
      
    - name: "HVAC"
      device_count: 12
      health_score: 91
      issues: 1
      
    - name: "Lighting"
      device_count: 15
      health_score: 95
      issues: 0
      
    - name: "Blinds"
      device_count: 10
      health_score: 88
      issues: 2
```

### Equipment Detail View

```yaml
PHMDeviceDetail:
  device_id: "pump-chw-1"
  device_name: "Chilled Water Pump 1"
  
  # Health overview
  health_score: 62
  status: "degraded"
  last_analysis: "2026-01-12T10:30:00Z"
  
  # Parameters
  parameters:
    - name: "Vibration"
      current: 3.8
      unit: "mm/s"
      baseline: 2.1
      status: "warning"
      trend: "increasing"
      sparkline: [2.1, 2.3, 2.5, 2.9, 3.2, 3.5, 3.8]
      
    - name: "Bearing Temperature"
      current: 45
      unit: "°C"
      baseline: 42
      status: "healthy"
      trend: "stable"
      
    - name: "Power Consumption"
      current: 5.2
      unit: "kW"
      baseline: 5.0
      status: "healthy"
      trend: "stable"
      
  # Active alerts
  alerts:
    - id: "phm-alert-12345"
      severity: "warning"
      message: "Vibration trending toward alarm level"
      timestamp: "2026-01-12T10:30:00Z"
      
  # Recommendations
  recommendations:
    - priority: "high"
      message: "Schedule vibration analysis - bearing wear suspected"
      suggested_action: "Create work order for bearing inspection"
      
  # History
  maintenance_history:
    - date: "2025-08-15"
      type: "Bearing replacement"
      notes: "Replaced both bearings, vibration returned to baseline"
```

### Reports

#### Weekly Health Digest

```yaml
PHMWeeklyReport:
  period: "2026-01-06 to 2026-01-12"
  
  summary:
    average_site_health: 85
    alerts_generated: 12
    alerts_by_severity:
      warning: 10
      alarm: 2
      critical: 0
    work_orders_created: 3
    
  highlights:
    - "Pump CHW-1 showing increased vibration - bearing inspection scheduled"
    - "Heat pump COP improved after filter cleaning"
    - "DALI bus error rate returned to normal after driver replacement"
    
  equipment_needing_attention:
    - device: "pump-chw-1"
      issue: "Elevated vibration"
      trend: "Worsening"
      recommendation: "Inspect bearings"
      
  upcoming_maintenance:
    - device: "ahu-1-fan"
      reason: "Belt runtime approaching 10,000 hours"
      suggested_date: "2026-01-20"
```

---

## API Endpoints

### Health Overview

```
GET /api/v1/phm/health
```

Response:
```json
{
  "site_health_score": 87,
  "summary": {
    "total_devices": 45,
    "healthy": 38,
    "degraded": 5,
    "warning": 2,
    "critical": 0
  },
  "categories": [...]
}
```

### Device Health

```
GET /api/v1/phm/devices/{device_id}
```

Response: `PHMDeviceDetail` object

### Active Alerts

```
GET /api/v1/phm/alerts?severity=warning,alarm&status=active
```

### Parameter History

```
GET /api/v1/phm/devices/{device_id}/parameters/{parameter}/history
    ?start=2026-01-01&end=2026-01-12&resolution=1h
```

### Trigger Re-Analysis

```
POST /api/v1/phm/devices/{device_id}/analyze
```

### Update Baseline

```
POST /api/v1/phm/devices/{device_id}/parameters/{parameter}/rebaseline
{
  "learning_days": 7,
  "reason": "After bearing replacement"
}
```

---

## Configuration

### Global PHM Settings

```yaml
# /etc/graylogic/phm.yaml
phm:
  enabled: true
  
  # Analysis schedule
  analysis:
    interval_minutes: 15            # Run every 15 minutes
    batch_size: 50                  # Devices per batch
    
  # Baseline learning
  baseline:
    default_learning_days: 7
    min_samples: 1000
    auto_rebaseline: false          # Require manual approval
    
  # Alerting
  alerts:
    deduplication_hours: 24         # Don't repeat same alert
    auto_clear_hours: 48            # Auto-clear if condition resolves
    
  # Data retention
  retention:
    raw_telemetry_days: 30
    aggregated_data_days: 365
    alerts_days: 730
    
  # Storage
  influxdb:
    url: "http://localhost:8086"
    org: "graylogic"
    bucket: "phm"
    token_env: "INFLUXDB_TOKEN"
```

### Device-Specific Configuration

```yaml
# In device configuration
device:
  id: "pump-chw-1"
  name: "Chilled Water Pump 1"
  type: "pump"
  
  phm:
    enabled: true
    category: "plant"
    
    parameters:
      - name: "vibration_mm_s"
        enabled: true
        source:
          type: "modbus"
          register: 40001
          scale: 0.1
        baseline:
          method: "context_aware"
          context_parameter: "speed_percent"
        thresholds:
          warning: null             # Auto-calculate
          alarm: null               # Auto-calculate
          critical: 7.1             # ISO 10816 limit
          
      - name: "bearing_temp_c"
        enabled: true
        source:
          type: "modbus"
          register: 40003
        baseline:
          method: "rolling_mean"
          window_days: 7
        thresholds:
          warning: null
          alarm: null
          critical: 80              # Absolute limit
```

---

## Implementation Notes

### Commissioning Workflow

1. **Add device to PHM** — Enable PHM in device configuration
2. **Verify data flow** — Confirm telemetry arriving in InfluxDB
3. **Start baseline learning** — Typically 7 days of normal operation
4. **Review baseline** — Check calculated thresholds are sensible
5. **Adjust if needed** — Set external limits, modify thresholds
6. **Enable alerting** — Device now monitored

### Re-Baselining

After maintenance or significant changes, re-baseline to avoid false alerts:

```yaml
# Trigger re-baseline via API or UI
POST /api/v1/phm/devices/pump-chw-1/parameters/vibration_mm_s/rebaseline
{
  "learning_days": 7,
  "reason": "Bearing replacement on 2026-01-10",
  "operator": "john.smith"
}
```

### Handling Missing Data

```yaml
missing_data:
  # If no data for threshold, consider device offline
  offline_threshold_minutes: 15
  
  # Don't alert during known outages
  respect_maintenance_windows: true
  
  # Interpolation for short gaps
  interpolate_gaps_under_minutes: 5
```

---

## Best Practices

### Do's

1. **Start with high-value equipment** — Pumps, fans, compressors first
2. **Allow adequate learning time** — 7 days minimum for baseline
3. **Use context-aware baselines** — Operating point matters
4. **Set external limits** — Use manufacturer/ISO limits as backstop
5. **Review weekly reports** — Catch trends before they become alerts
6. **Re-baseline after maintenance** — Prevent false alerts

### Don'ts

1. **Don't enable PHM without data verification** — Garbage in, garbage out
2. **Don't set thresholds too tight** — Alert fatigue kills PHM value
3. **Don't ignore "info" trends** — They become warnings
4. **Don't skip commissioning** — Proper baseline is critical
5. **Don't auto-rebaseline** — Manual review catches real issues

---

## Related Documents

- [Core Internals](../architecture/core-internals.md) — PHM Engine architecture
- [Energy Model](../architecture/energy-model.md) — Energy equipment PHM
- [Plant Domain](../domains/plant.md) — Plant equipment PHM details
- [Climate Domain](../domains/climate.md) — HVAC PHM integration
- [Lighting Domain](../domains/lighting.md) — Lighting PHM integration
- [Blinds Domain](../domains/blinds.md) — Blind motor PHM
- [Audio Domain](../domains/audio.md) — Audio equipment PHM
- [Modbus Protocol](../protocols/modbus.md) — Equipment telemetry collection
- [API Specification](../interfaces/api.md) — PHM API endpoints
