---
title: Weather Integration Specification
version: 1.0.0
status: active
last_updated: 2026-01-13
depends_on:
  - architecture/system-overview.md
  - automation/automation.md
  - domains/irrigation.md
  - domains/blinds.md
  - domains/climate.md
---

# Weather Integration Specification

This document specifies how Gray Logic integrates weather data for automation — using forecasts and current conditions to optimize irrigation, blinds, climate control, and energy management.

---

## Overview

### What Weather Integration Provides

| Use Case | Weather Data Used | Automation |
|----------|-------------------|------------|
| **Irrigation** | Rain forecast, evapotranspiration | Skip watering if rain expected |
| **Blinds** | Sun position, cloud cover | Optimize solar gain/glare |
| **Climate** | Temperature forecast | Pre-conditioning optimization |
| **Energy** | Solar forecast | Optimize battery/EV charging |
| **Pool** | Temperature, UV | Heating and cover control |
| **Alerts** | Storms, frost, high wind | Protective actions |

### Design Principles

1. **Optional enhancement** — System works without weather data
2. **Local caching** — Cached data works during internet outages
3. **Multiple sources** — Fallback between data sources
4. **Privacy** — Location data stays local

### Data Sources

```
┌─────────────────────────────────────────────────────────────────┐
│                      WEATHER SOURCES                             │
├─────────────────┬─────────────────┬─────────────────────────────┤
│  Online APIs    │  Local Stations │  Satellite Decode           │
│  (internet)     │  (on-site)      │  (off-grid)                 │
│                 │                 │                             │
│  • OpenWeather  │  • Davis        │  • NOAA GOES                │
│  • Met Office   │  • Netatmo      │  • EUMETSAT                 │
│  • Tomorrow.io  │  • Ecowitt      │  (See satellite-weather.md) │
│  • Open-Meteo   │  • HomeMatic    │                             │
└────────┬────────┴────────┬────────┴──────────────┬──────────────┘
         │                 │                       │
         └─────────────────┼───────────────────────┘
                           │
                    Weather Service
                    (Gray Logic Core)
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
    Irrigation       Blinds/Climate     Energy/Alerts
```

---

## Weather Data Model

### Current Conditions

```yaml
WeatherCurrent:
  timestamp: datetime               # When measured/fetched
  source: string                    # "openweather" | "local_station" | "satellite"
  
  # Temperature
  temperature_c: float
  feels_like_c: float
  humidity_percent: float
  dew_point_c: float
  
  # Atmospheric
  pressure_hpa: float
  pressure_trend: string            # "rising" | "falling" | "steady"
  
  # Wind
  wind_speed_ms: float
  wind_direction_deg: float
  wind_gust_ms: float
  
  # Precipitation
  precipitation_mm: float           # Current/recent
  precipitation_type: string        # "none" | "rain" | "snow" | "sleet"
  
  # Visibility and clouds
  visibility_km: float
  cloud_cover_percent: float
  
  # Solar
  uv_index: float
  solar_radiation_wm2: float
  
  # Derived
  is_daytime: boolean
  conditions: string                # "clear" | "cloudy" | "rain" | "snow" | "fog"
```

### Forecast

```yaml
WeatherForecast:
  generated: datetime               # When forecast was issued
  source: string
  
  # Hourly forecast (next 48 hours)
  hourly:
    - datetime: datetime
      temperature_c: float
      precipitation_probability: float
      precipitation_mm: float
      cloud_cover_percent: float
      wind_speed_ms: float
      uv_index: float
      
  # Daily forecast (next 7 days)
  daily:
    - date: date
      temperature_high_c: float
      temperature_low_c: float
      precipitation_probability: float
      precipitation_mm: float
      sunrise: datetime
      sunset: datetime
      uv_index_max: float
      conditions: string
```

### Solar Data

```yaml
SolarData:
  date: date
  latitude: float
  longitude: float
  
  # Times (local timezone)
  sunrise: time
  sunset: time
  solar_noon: time
  day_length_hours: float
  
  # Current sun position
  sun_elevation_deg: float
  sun_azimuth_deg: float
  
  # Solar irradiance (forecast)
  hourly_irradiance:
    - hour: int
      ghi_wm2: float               # Global Horizontal Irradiance
      dni_wm2: float               # Direct Normal Irradiance
      dhi_wm2: float               # Diffuse Horizontal Irradiance
```

---

## Data Sources

### Online APIs

#### Open-Meteo (Free, No API Key)

```yaml
weather_source:
  type: "open_meteo"
  
  config:
    base_url: "https://api.open-meteo.com/v1"
    
    # Location
    latitude: 51.5074
    longitude: -0.1278
    timezone: "Europe/London"
    
    # Data requested
    current:
      - temperature_2m
      - relative_humidity_2m
      - precipitation
      - wind_speed_10m
      - wind_direction_10m
      - cloud_cover
      
    hourly:
      - temperature_2m
      - precipitation_probability
      - precipitation
      - cloud_cover
      
    daily:
      - temperature_2m_max
      - temperature_2m_min
      - precipitation_sum
      - sunrise
      - sunset
      
  # Refresh interval
  refresh_interval_minutes: 30
  
  # Caching
  cache:
    current_ttl_minutes: 30
    forecast_ttl_hours: 6
```

#### OpenWeatherMap

```yaml
weather_source:
  type: "openweathermap"
  
  config:
    api_key_env: "OPENWEATHER_API_KEY"
    base_url: "https://api.openweathermap.org/data/3.0"
    
    # Location
    latitude: 51.5074
    longitude: -0.1278
    
    # Plan (determines rate limits)
    plan: "free"                    # free | startup | developer
    
  refresh_interval_minutes: 60      # Free tier: 1000 calls/day
```

#### Met Office DataPoint (UK)

```yaml
weather_source:
  type: "met_office"
  
  config:
    api_key_env: "METOFFICE_API_KEY"
    location_id: "352409"           # Nearest observation site
    
  refresh_interval_minutes: 60
```

#### Tomorrow.io

```yaml
weather_source:
  type: "tomorrow_io"
  
  config:
    api_key_env: "TOMORROW_API_KEY"
    latitude: 51.5074
    longitude: -0.1278
    
  refresh_interval_minutes: 15
```

### Local Weather Stations

#### Davis Instruments

```yaml
weather_source:
  type: "davis"
  
  config:
    # WeatherLink Live
    device_ip: "192.168.1.50"
    
    # Or WeatherLink cloud
    api_key_env: "DAVIS_API_KEY"
    station_id: "123456"
    
  refresh_interval_seconds: 30
```

#### Netatmo

```yaml
weather_source:
  type: "netatmo"
  
  config:
    client_id_env: "NETATMO_CLIENT_ID"
    client_secret_env: "NETATMO_CLIENT_SECRET"
    refresh_token_env: "NETATMO_REFRESH_TOKEN"
    device_id: "70:ee:50:xx:xx:xx"
    
  refresh_interval_minutes: 10
```

#### Ecowitt/Fine Offset

```yaml
weather_source:
  type: "ecowitt"
  
  config:
    # Local gateway
    gateway_ip: "192.168.1.51"
    
    # Or push to Gray Logic
    listen_port: 8088
    passkey: "${ECOWITT_PASSKEY}"
    
  refresh_interval_seconds: 60
```

#### Generic MQTT Weather Station

```yaml
weather_source:
  type: "mqtt"
  
  config:
    topic_prefix: "weather/station1"
    
    # Topic mapping
    topics:
      temperature: "weather/station1/temperature"
      humidity: "weather/station1/humidity"
      pressure: "weather/station1/pressure"
      wind_speed: "weather/station1/wind_speed"
      wind_direction: "weather/station1/wind_direction"
      rain_rate: "weather/station1/rain_rate"
```

### Source Priority

```yaml
source_priority:
  # Order of preference
  priority:
    1: "local_station"              # Most accurate for site
    2: "open_meteo"                 # Free, reliable
    3: "openweathermap"             # Good coverage
    4: "satellite"                  # Off-grid fallback
    
  # Fallback behavior
  fallback:
    on_source_failure: "use_next"
    max_data_age_minutes: 120       # Use cached if fresh enough
    
  # Source health
  health:
    check_interval_seconds: 60
    failure_threshold: 3             # Failures before fallback
```

---

## Automation Integration

### Irrigation

```yaml
irrigation_weather:
  # Skip watering if rain expected
  rain_skip:
    enabled: true
    probability_threshold: 60       # >60% chance
    amount_threshold_mm: 5          # >5mm expected
    lookahead_hours: 24
    
  # Skip if recently rained
  recent_rain:
    enabled: true
    threshold_mm: 10
    lookback_hours: 24
    
  # Evapotranspiration adjustment
  et_adjustment:
    enabled: true
    method: "penman_monteith"       # or "hargreaves"
    base_et_mm: 5                   # Base daily ET
    
  # Frost protection
  frost_protection:
    enabled: true
    temperature_threshold_c: 2
    wind_chill_factor: true
```

**Example automation:**

```yaml
automation:
  - name: "Skip irrigation if rain expected"
    trigger:
      type: "schedule"
      schedule_id: "irrigation-lawn"
      event: "before_run"
      lead_time_minutes: 30
      
    conditions:
      - condition: "weather"
        parameter: "precipitation_probability_24h"
        operator: ">"
        value: 60
        
    actions:
      - action: "cancel_schedule_run"
        reason: "Rain expected (${weather.precipitation_probability_24h}%)"
        notify: true
```

### Blinds and Shading

```yaml
blinds_weather:
  # Sun tracking
  sun_tracking:
    enabled: true
    elevation_threshold_deg: 10     # Sun must be above horizon
    
  # Cloud-based adjustment
  cloud_adjustment:
    enabled: true
    mappings:
      - cloud_cover: [0, 20]
        position: 100                # Fully closed for sun protection
      - cloud_cover: [20, 60]
        position: 50                 # Partial
      - cloud_cover: [60, 100]
        position: 0                  # Open, no sun
        
  # Wind protection
  wind_protection:
    enabled: true
    retract_threshold_ms: 10        # 10 m/s (~22 mph)
    extend_threshold_ms: 6          # Hysteresis
    
  # Storm protection
  storm_protection:
    enabled: true
    conditions: ["thunderstorm", "hail"]
    action: "retract_all"
```

**Example automation:**

```yaml
automation:
  - name: "Retract blinds on high wind"
    trigger:
      type: "weather"
      parameter: "wind_speed_ms"
      operator: ">"
      value: 10
      
    actions:
      - action: "device_command"
        targets:
          type: "capability"
          capability: "exterior_blind"
        command: "retract"
        
  - name: "Adjust blinds based on sun and clouds"
    trigger:
      type: "interval"
      minutes: 15
      
    conditions:
      - condition: "time"
        between: ["sunrise", "sunset"]
        
    actions:
      - action: "execute_script"
        script: "blinds_solar_optimization"
```

### Climate Pre-Conditioning

```yaml
climate_weather:
  # Optimum start
  optimum_start:
    enabled: true
    use_forecast: true
    outdoor_temp_factor: 0.1        # Adjust lead time by outdoor temp
    
  # Pre-cooling
  pre_cooling:
    enabled: true
    trigger_temp_c: 25              # Start pre-cooling above this
    lookahead_hours: 4
    
  # Heat wave mode
  heat_wave:
    enabled: true
    threshold_c: 30
    consecutive_days: 2
    actions:
      - "increase_ac_setpoint_by: 1"
      - "close_blinds_south_east_west"
      - "notify_owner"
```

**Example automation:**

```yaml
automation:
  - name: "Pre-cool before hot afternoon"
    trigger:
      type: "schedule"
      time: "10:00"
      
    conditions:
      - condition: "weather"
        parameter: "temperature_high_today_c"
        operator: ">"
        value: 28
        
    actions:
      - action: "climate_setpoint"
        target: "zone-living"
        cooling: 22
        until: "14:00"
        note: "Pre-cooling before peak heat"
```

### Energy Optimization

```yaml
energy_weather:
  # Solar forecast for battery management
  solar_forecast:
    enabled: true
    source: "open_meteo"            # Solar forecast API
    
    # Battery charging strategy
    battery_strategy:
      low_solar_forecast:           # Cloudy day expected
        charge_from_grid: true
        charge_time: "02:00-06:00"  # Off-peak
        
      high_solar_forecast:          # Sunny day expected
        charge_from_grid: false
        reserve_capacity: 20        # Leave room for solar
        
  # EV charging
  ev_charging:
    use_solar_forecast: true
    prefer_solar_hours: true
    
  # Pool heating
  pool_heating:
    use_solar_forecast: true
    skip_heating_if_sunny: true     # Solar gain sufficient
```

### Alerts and Protection

```yaml
weather_alerts:
  # Frost alert
  frost:
    enabled: true
    threshold_c: 2
    forecast_hours: 6
    actions:
      - "notify_owner"
      - "protect_irrigation"
      - "increase_heating_setpoint"
      
  # Storm alert
  storm:
    enabled: true
    conditions: ["thunderstorm", "severe_weather", "hail"]
    actions:
      - "notify_owner"
      - "retract_awnings"
      - "close_skylights"
      - "secure_pool_cover"
      
  # High wind
  wind:
    enabled: true
    threshold_ms: 15                # ~34 mph
    actions:
      - "retract_exterior_blinds"
      - "retract_awnings"
      
  # Heat wave
  heat_wave:
    enabled: true
    threshold_c: 30
    days: 2
    actions:
      - "notify_owner"
      - "adjust_cooling_strategy"
```

---

## Weather Service

### Core Weather Service

```go
// internal/intelligence/weather/service.go
type WeatherService struct {
    sources    []WeatherSource
    cache      *WeatherCache
    automation *AutomationIntegration
}

type WeatherData struct {
    Current  CurrentConditions
    Forecast Forecast
    Solar    SolarData
    Alerts   []WeatherAlert
    Source   string
    Updated  time.Time
}

func (s *WeatherService) GetCurrent() (*CurrentConditions, error)
func (s *WeatherService) GetForecast(hours int) (*Forecast, error)
func (s *WeatherService) GetSolarData(date time.Time) (*SolarData, error)
func (s *WeatherService) Subscribe(callback WeatherCallback)
```

### Caching

```yaml
weather_cache:
  # Cache configuration
  current:
    ttl_minutes: 30
    refresh_interval_minutes: 15
    
  forecast:
    ttl_hours: 6
    refresh_interval_hours: 3
    
  solar:
    ttl_hours: 24
    refresh_at: "00:00"             # Daily refresh
    
  # Offline behavior
  offline:
    use_stale: true
    max_age_hours: 12
    fallback_to_historical: true    # Use same day last year
```

### MQTT Topics

```yaml
mqtt_topics:
  # Published by Weather Service
  current: "graylogic/weather/current"
  forecast: "graylogic/weather/forecast"
  solar: "graylogic/weather/solar"
  alerts: "graylogic/weather/alerts"
  
  # Example current conditions
  current_payload:
    temperature_c: 15.5
    humidity_percent: 65
    wind_speed_ms: 5.2
    precipitation_mm: 0
    cloud_cover_percent: 40
    conditions: "partly_cloudy"
    source: "local_station"
    timestamp: "2026-01-13T10:30:00Z"
```

---

## API Endpoints

### Current Weather

```http
GET /api/v1/weather/current
```

**Response:**
```json
{
  "temperature_c": 15.5,
  "feels_like_c": 14.2,
  "humidity_percent": 65,
  "pressure_hpa": 1015,
  "wind_speed_ms": 5.2,
  "wind_direction_deg": 270,
  "precipitation_mm": 0,
  "cloud_cover_percent": 40,
  "uv_index": 3,
  "conditions": "partly_cloudy",
  "is_daytime": true,
  "source": "local_station",
  "updated": "2026-01-13T10:30:00Z"
}
```

### Forecast

```http
GET /api/v1/weather/forecast?hours=24
```

**Response:**
```json
{
  "hourly": [
    {
      "datetime": "2026-01-13T11:00:00Z",
      "temperature_c": 16,
      "precipitation_probability": 10,
      "precipitation_mm": 0,
      "cloud_cover_percent": 35,
      "wind_speed_ms": 4.5
    }
  ],
  "daily": [
    {
      "date": "2026-01-13",
      "temperature_high_c": 18,
      "temperature_low_c": 8,
      "precipitation_probability": 20,
      "precipitation_mm": 2,
      "conditions": "partly_cloudy"
    }
  ],
  "source": "open_meteo",
  "generated": "2026-01-13T06:00:00Z"
}
```

### Solar Data

```http
GET /api/v1/weather/solar?date=2026-01-13
```

**Response:**
```json
{
  "date": "2026-01-13",
  "sunrise": "07:45",
  "sunset": "16:30",
  "solar_noon": "12:07",
  "day_length_hours": 8.75,
  "sun_elevation_deg": 25.3,
  "sun_azimuth_deg": 165,
  "hourly_irradiance": [
    {"hour": 8, "ghi_wm2": 50},
    {"hour": 9, "ghi_wm2": 150},
    {"hour": 10, "ghi_wm2": 280}
  ]
}
```

### Weather Alerts

```http
GET /api/v1/weather/alerts
```

**Response:**
```json
{
  "alerts": [
    {
      "type": "frost",
      "severity": "warning",
      "message": "Frost expected tonight",
      "starts": "2026-01-13T22:00:00Z",
      "ends": "2026-01-14T08:00:00Z",
      "actions_taken": [
        "Irrigation paused",
        "Heating setpoint increased"
      ]
    }
  ]
}
```

---

## Configuration

### Complete Weather Configuration

```yaml
# /etc/graylogic/weather.yaml
weather:
  enabled: true
  
  # Location
  location:
    latitude: 51.5074
    longitude: -0.1278
    timezone: "Europe/London"
    elevation_m: 11
    
  # Primary source
  primary_source:
    type: "open_meteo"
    refresh_interval_minutes: 30
    
  # Local station (optional, preferred if available)
  local_station:
    enabled: true
    type: "ecowitt"
    gateway_ip: "192.168.1.51"
    
  # Backup sources
  backup_sources:
    - type: "openweathermap"
      api_key_env: "OPENWEATHER_API_KEY"
      
  # Caching
  cache:
    current_ttl_minutes: 30
    forecast_ttl_hours: 6
    
  # Integrations
  integrations:
    irrigation:
      enabled: true
      rain_skip: true
      et_adjustment: true
      
    blinds:
      enabled: true
      sun_tracking: true
      wind_protection: true
      
    climate:
      enabled: true
      optimum_start: true
      pre_cooling: true
      
    energy:
      enabled: true
      solar_forecast: true
      
  # Alerts
  alerts:
    frost:
      enabled: true
      threshold_c: 2
      
    wind:
      enabled: true
      threshold_ms: 15
      
    storm:
      enabled: true
```

---

## Related Documents

- [Automation](../automation/automation.md) — Weather-triggered automation
- [Irrigation Domain](../domains/irrigation.md) — Weather-based irrigation
- [Blinds Domain](../domains/blinds.md) — Sun tracking and wind protection
- [Climate Domain](../domains/climate.md) — Weather-aware climate control
- [Energy Domain](../domains/energy.md) — Solar forecasting
- [Pool Domain](../domains/pool.md) — Weather-based pool control
- [Satellite Weather](../resilience/satellite-weather.md) — Off-grid weather data
