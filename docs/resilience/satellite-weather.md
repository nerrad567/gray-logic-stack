---
title: Satellite Weather Specification
version: 1.0.0
status: active
last_updated: 2026-01-13
depends_on:
  - intelligence/weather.md
  - resilience/offline.md
---

# Satellite Weather Specification

This document specifies Gray Logic's capability to receive weather data directly from satellites — enabling weather-aware automation even when internet connectivity is unavailable.

---

## Overview

### Why Satellite Weather?

Most weather integrations depend on internet-connected APIs. Satellite weather provides:

| Internet APIs | Satellite Decode |
|---------------|------------------|
| Requires internet | Works offline |
| Depends on third-party service | Direct from source |
| Can be rate-limited | Unlimited access |
| Service may change/discontinue | Satellite signals stable for decades |
| ~15-60 minute updates | Real-time (as satellite transmits) |

### Use Cases

1. **Off-grid properties** — No reliable internet
2. **Resilience** — Internet outage doesn't affect weather automation
3. **Storm awareness** — Direct imagery for approaching weather
4. **Self-sufficiency** — No dependency on external services

### Supported Satellites

| Satellite | Coverage | Data Type | Frequency |
|-----------|----------|-----------|-----------|
| **NOAA POES** | Global (polar orbit) | APT imagery | 137 MHz |
| **NOAA GOES** | Americas | HRIT/EMWIN | 1.7 GHz |
| **EUMETSAT** | Europe/Africa | LRIT/HRIT | 1.7 GHz |
| **Meteor-M** | Global (polar orbit) | LRPT imagery | 137 MHz |

---

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                        SATELLITE                                 │
│  (NOAA, GOES, EUMETSAT, Meteor)                                 │
└────────────────────────────┬─────────────────────────────────────┘
                             │ Radio Signal
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    ANTENNA + LNA                                 │
│  • V-dipole or QFH (137 MHz)                                    │
│  • Dish (1.7 GHz for geostationary)                             │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                  SOFTWARE DEFINED RADIO                          │
│  • RTL-SDR (basic)                                              │
│  • Airspy (better)                                              │
│  • SDRPlay (advanced)                                           │
└────────────────────────────┬─────────────────────────────────────┘
                             │ USB
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                 SATELLITE DECODER SERVICE                        │
│  • noaa-apt / SatDump (polar orbiters)                          │
│  • goestools / xrit-rx (geostationary)                          │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                 GRAY LOGIC WEATHER SERVICE                       │
│  • Image processing                                             │
│  • Cloud/rain detection                                         │
│  • Integration with automation                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

```yaml
data_flow:
  1_receive:
    component: "SDR + Antenna"
    action: "Capture raw radio signal from satellite pass"
    
  2_decode:
    component: "Decoder software"
    action: "Demodulate and decode to image data"
    
  3_process:
    component: "Image processor"
    action: "Apply map overlays, extract weather features"
    
  4_analyze:
    component: "Weather analyzer"
    action: "Detect clouds, precipitation, storm cells"
    
  5_integrate:
    component: "Weather service"
    action: "Update weather data, trigger automations"
```

---

## Hardware Requirements

### Basic Setup (NOAA APT)

Receives imagery from NOAA polar-orbiting satellites (~4 passes/day).

```yaml
basic_hardware:
  antenna:
    type: "V-dipole or QFH"
    frequency: "137 MHz"
    cost: "£10-50 (DIY) or £50-150 (commercial)"
    
  sdr:
    type: "RTL-SDR v3"
    cost: "£25-40"
    
  computer:
    requirement: "Any Linux (Raspberry Pi 4 sufficient)"
    
  cable:
    type: "50Ω coax (RG58 or better)"
    length: "As short as practical"
    
  lna:
    type: "LNA4ALL or similar (optional but recommended)"
    cost: "£20-40"
    
  total_cost: "£60-250"
```

### Advanced Setup (GOES/EUMETSAT HRIT)

Receives real-time imagery from geostationary satellites.

```yaml
advanced_hardware:
  antenna:
    type: "1m dish with scalar feed"
    frequency: "1.7 GHz (L-band)"
    cost: "£100-300"
    pointing: "Fixed (geostationary satellites)"
    
  sdr:
    type: "Airspy Mini or SDRPlay RSP1A"
    cost: "£100-150"
    
  lna:
    type: "SAWbird+ GOES or similar"
    cost: "£40-60"
    
  computer:
    requirement: "Linux with decent CPU"
    
  total_cost: "£300-600"
```

### Antenna Placement

```yaml
antenna_placement:
  noaa_apt:
    location: "Outdoors, clear sky view"
    elevation: "Higher is better"
    obstruction: "Avoid buildings to the north (UK) for polar passes"
    
  goes_hrit:
    location: "Fixed pointing to satellite"
    elevation: "GOES-16: ~30° elevation from UK (low on horizon)"
    obstruction: "Clear line of sight to satellite position"
    note: "EUMETSAT better for Europe (higher elevation angle)"
```

---

## Software Stack

### NOAA APT Reception

```yaml
noaa_apt_software:
  # SDR driver
  rtl_sdr:
    package: "rtl-sdr"
    purpose: "Interface with RTL-SDR dongle"
    
  # Decoder options
  noaa_apt:
    url: "https://github.com/martinber/noaa-apt"
    purpose: "Decode APT signal to image"
    
  satdump:
    url: "https://github.com/SatDump/SatDump"
    purpose: "All-in-one decoder (recommended)"
    
  # Pass prediction
  gpredict:
    purpose: "Satellite pass prediction"
    
  # Automation
  scripts:
    predict_passes: "Calculate upcoming satellite passes"
    record_pass: "Capture SDR data during pass"
    decode_pass: "Convert recording to image"
    notify_core: "Send image to Gray Logic"
```

### GOES/EUMETSAT Reception

```yaml
geostationary_software:
  # GOES (Americas)
  goestools:
    url: "https://github.com/pietern/goestools"
    purpose: "Receive and decode GOES HRIT"
    
  # EUMETSAT (Europe)
  xrit_rx:
    purpose: "EUMETCast via satellite"
    
  satdump:
    purpose: "Universal decoder (supports both)"
```

### Gray Logic Integration

```yaml
satellite_integration:
  # Satellite receiver service
  service:
    name: "graylogic-satellite"
    purpose: "Coordinate reception and processing"
    
  # Image storage
  storage:
    path: "/var/lib/graylogic/satellite"
    retention_days: 7
    
  # Weather extraction
  processing:
    cloud_detection: true
    precipitation_detection: true
    storm_tracking: true
```

---

## Configuration

### Satellite Receiver Configuration

```yaml
# /etc/graylogic/satellite.yaml
satellite:
  enabled: true
  
  # Hardware
  hardware:
    sdr:
      type: "rtlsdr"
      device_index: 0
      ppm_correction: 0              # Frequency correction
      gain: 40                       # dB
      
    antenna:
      type: "qfh"
      frequency_mhz: 137.1
      
  # Satellites to receive
  satellites:
    noaa_15:
      enabled: true
      frequency_mhz: 137.620
      min_elevation_deg: 20          # Minimum pass elevation
      
    noaa_18:
      enabled: true
      frequency_mhz: 137.9125
      min_elevation_deg: 20
      
    noaa_19:
      enabled: true
      frequency_mhz: 137.100
      min_elevation_deg: 20
      
    meteor_m2:
      enabled: true
      frequency_mhz: 137.100
      protocol: "lrpt"               # Different from APT
      
  # Location (for pass prediction)
  location:
    latitude: 51.5074
    longitude: -0.1278
    elevation_m: 50
    
  # Processing
  processing:
    map_overlay: true
    enhancement: "contrast"          # none | contrast | thermal
    
  # Output
  output:
    image_path: "/var/lib/graylogic/satellite/images"
    latest_symlink: true             # /var/lib/graylogic/satellite/latest.png
```

### Pass Scheduling

```yaml
pass_scheduling:
  # Prediction
  prediction:
    tle_source: "celestrak"
    tle_update_hours: 24
    lookahead_hours: 24
    
  # Recording
  recording:
    start_before_aos_seconds: 30     # Before Acquisition of Signal
    end_after_los_seconds: 30        # After Loss of Signal
    
  # Priority
  priority:
    # If passes overlap
    prefer:
      - "higher_elevation"
      - "noaa_over_meteor"
```

---

## Image Processing

### Cloud Detection

Extract cloud cover from satellite imagery:

```yaml
cloud_detection:
  enabled: true
  
  # Visible channel analysis
  visible:
    method: "threshold"
    bright_threshold: 180            # Pixel value (0-255)
    cloud_if_above: true
    
  # Infrared analysis
  infrared:
    method: "temperature"
    cold_threshold_c: -20            # Cloud tops are cold
    
  # Output
  output:
    cloud_cover_percent: true
    cloud_mask_image: true
```

### Precipitation Detection

Estimate rain from thermal imagery:

```yaml
precipitation_detection:
  enabled: true
  
  # Cloud top temperature
  method: "cloud_top_temp"
  
  thresholds:
    light_rain_c: -20                # Cloud top < -20°C
    moderate_rain_c: -35             # Cloud top < -35°C
    heavy_rain_c: -50                # Cloud top < -50°C
    
  # Confidence
  note: "Satellite precipitation is indicative, not precise"
```

### Storm Tracking

Detect and track storm cells:

```yaml
storm_tracking:
  enabled: true
  
  # Detection
  detection:
    method: "ir_gradient"            # Strong temperature gradients
    min_size_km: 20
    
  # Tracking
  tracking:
    method: "frame_to_frame"
    max_movement_km: 50              # Per image interval
    
  # Alerting
  alerts:
    approaching:
      distance_km: 100
      eta_hours: 2
```

---

## Weather Data Extraction

### Nowcast Generation

Create short-term forecast from satellite imagery:

```yaml
nowcast:
  # Current conditions
  current:
    cloud_cover:
      source: "latest_visible_image"
      region: "50km_radius"
      
    precipitation_likelihood:
      source: "cloud_top_temperature"
      
  # Short-term forecast (0-2 hours)
  forecast:
    method: "persistence"            # Clouds continue moving same direction
    confidence_decay_hours: 2
    
  # Integration
  integration:
    update_weather_service: true
    trigger_automations: true
```

### Data Output

```yaml
satellite_weather_output:
  # MQTT topics
  mqtt:
    prefix: "graylogic/weather/satellite"
    
    topics:
      cloud_cover: "graylogic/weather/satellite/cloud_cover"
      precipitation: "graylogic/weather/satellite/precipitation"
      storm_alert: "graylogic/weather/satellite/storm"
      latest_image: "graylogic/weather/satellite/image"
      
  # Data format
  cloud_cover_payload:
    percent: 65
    source: "noaa_19"
    captured: "2026-01-13T10:30:00Z"
    
  storm_alert_payload:
    detected: true
    distance_km: 80
    direction_deg: 270              # Coming from west
    eta_minutes: 90
    severity: "moderate"
```

---

## Integration with Automation

### Weather Triggers

```yaml
satellite_triggers:
  # Storm approaching
  - trigger: "storm_approaching"
    conditions:
      - distance_km: "<100"
      - eta_hours: "<2"
    actions:
      - "notify_owner"
      - "retract_awnings"
      - "close_skylights"
      - "secure_pool_cover"
      
  # Clearing weather
  - trigger: "clearing_detected"
    conditions:
      - cloud_trend: "decreasing"
      - current_precipitation: false
    actions:
      - "resume_irrigation_schedule"
      - "open_blinds"
```

### Fallback Behavior

```yaml
fallback:
  # If no recent satellite data
  stale_threshold_hours: 6
  
  # Actions when stale
  on_stale:
    - "Mark satellite data as stale"
    - "Fall back to API weather (if available)"
    - "Continue with last known conditions"
    
  # If no satellite passes (polar)
  no_passes:
    note: "Polar satellites pass ~4 times/day; gaps are normal"
    max_gap_hours: 6
```

---

## Maintenance

### Routine Checks

```yaml
maintenance:
  # SDR health
  sdr:
    check_interval: "daily"
    test: "Confirm SDR responds to frequency query"
    
  # Antenna
  antenna:
    check_interval: "monthly"
    test: "Visual inspection, signal strength during known pass"
    
  # TLE updates
  tle:
    auto_update: true
    interval_hours: 24
    fallback_age_days: 7             # TLEs still usable for ~week
    
  # Storage
  storage:
    check_interval: "weekly"
    warn_at_percent: 80
    auto_cleanup: true
    retention_days: 7
```

### Troubleshooting

```yaml
troubleshooting:
  no_signal:
    checks:
      - "SDR connected and recognized"
      - "Correct frequency set"
      - "Antenna connected and oriented correctly"
      - "Pass actually occurring (check prediction)"
      
  poor_image:
    checks:
      - "Antenna gain/LNA working"
      - "SDR gain not too high (clipping)"
      - "Pass elevation was sufficient (>20°)"
      
  decode_failure:
    checks:
      - "Recording captured enough of pass"
      - "Correct decoder for satellite type"
      - "Sufficient CPU for real-time decode"
```

---

## Limitations

### What Satellite Weather Can Do

- Detect cloud cover and movement
- Indicate precipitation likelihood
- Track large storm systems
- Provide imagery for situational awareness
- Work completely offline

### What Satellite Weather Cannot Do

- Precise temperature readings (use ground sensors)
- Exact precipitation amounts (indicative only)
- Long-range forecasts (nowcast only, 0-2 hours)
- Replace internet APIs for detailed forecasts
- Work in all weather (heavy rain affects 1.7 GHz)

### Practical Considerations

```yaml
practical_considerations:
  # Polar orbiters (NOAA, Meteor)
  polar:
    passes_per_day: 4-6
    pass_duration_minutes: 10-15
    coverage: "Global, but periodic"
    
  # Geostationary (GOES, EUMETSAT)
  geostationary:
    updates: "Every 15 minutes"
    coverage: "Continuous for coverage area"
    antenna: "More complex (dish required)"
    
  # Recommended approach
  recommendation: |
    For most UK installations:
    - Start with NOAA APT (simple, cheap)
    - Add EUMETSAT if continuous coverage needed
    - Use alongside internet APIs, not instead of
```

---

## Related Documents

- [Weather Integration](../intelligence/weather.md) — Main weather specification
- [Offline Behavior](offline.md) — System resilience
- [Irrigation Domain](../domains/irrigation.md) — Weather-triggered irrigation
- [Blinds Domain](../domains/blinds.md) — Storm protection
