---
title: CCTV and Video Surveillance Integration
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - overview/principles.md
  - architecture/system-overview.md
  - data-model/entities.md
---

# CCTV and Video Surveillance Integration

This document specifies how Gray Logic integrates with CCTV systems, IP cameras, and Network Video Recorders (NVRs) for both residential and commercial applications.

---

## Overview

Gray Logic provides unified access to video streams and camera events, while respecting privacy principles and maintaining independence from the recording system.

### What Gray Logic Does

- ✅ Display live camera feeds on wall panels, mobile apps, and web UI
- ✅ Receive motion detection and analytics events from cameras/NVR
- ✅ Trigger automation based on camera events (motion, line crossing, etc.)
- ✅ Show camera pop-ups on intercom/doorbell events
- ✅ Provide single UI for all video sources (cameras, intercoms, door stations)
- ✅ Log camera-related events for audit trail

### What Gray Logic Does NOT Do

- ❌ Record video (the NVR handles this)
- ❌ Store video footage long-term
- ❌ Perform video analytics (the camera/NVR handles this)
- ❌ Manage NVR storage or retention policies
- ❌ Replace proper security monitoring (alarm response centre)

### Privacy Principles

From [principles.md](../overview/principles.md):

1. **No cloud video** — All streams stay on local network
2. **No behaviour profiling** — Motion events trigger automation, not surveillance
3. **User control** — Residents can disable camera views in private spaces
4. **Audit logging** — Who viewed which camera, when

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              CCTV VLAN                                        │
│                                                                               │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────────────┐ │
│  │ Camera  │  │ Camera  │  │ Camera  │  │ Camera  │  │        NVR          │ │
│  │  (PTZ)  │  │ (Fixed) │  │ (Dome)  │  │(Doorbell│  │  (Recording/Events) │ │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  └──────────┬──────────┘ │
│       │            │            │            │                   │            │
│       └────────────┴────────────┴────────────┴───────────────────┤            │
│                                                                  │            │
└──────────────────────────────────────────────────────────────────┼────────────┘
                                                                   │
                                                    Firewall (RTSP, ONVIF only)
                                                                   │
┌──────────────────────────────────────────────────────────────────┼────────────┐
│                           CONTROL VLAN                           │            │
│                                                                  │            │
│  ┌────────────────────────────────────────────────────────────── ▼ ─────────┐ │
│  │                         CCTV BRIDGE                                       │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │ │
│  │  │   ONVIF     │  │    RTSP     │  │   Event     │  │   Stream        │  │ │
│  │  │  Discovery  │  │   Manager   │  │  Receiver   │  │   Proxy         │  │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────┘  │ │
│  └───────────────────────────────────────┬───────────────────────────────────┘ │
│                                          │                                     │
│                                    Internal MQTT                               │
│                                          │                                     │
│  ┌───────────────────────────────────────▼───────────────────────────────────┐ │
│  │                        GRAY LOGIC CORE                                     │ │
│  │                                                                            │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌───────────────┐  │ │
│  │  │   Device     │  │    Event     │  │  Automation  │  │  Stream URL   │  │ │
│  │  │   Registry   │  │    Router    │  │   Triggers   │  │   Provider    │  │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  └───────────────┘  │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                          │                                     │
└──────────────────────────────────────────┼─────────────────────────────────────┘
                                           │
                              REST API + WebSocket
                                           │
        ┌──────────────────────────────────┼──────────────────────────────────┐
        │                                  │                                  │
   ┌────▼─────┐                      ┌─────▼────┐                       ┌─────▼────┐
   │  Wall    │                      │  Mobile  │                       │   Web    │
   │  Panel   │                      │   App    │                       │   Admin  │
   │ (Flutter)│                      │ (Flutter)│                       │ (Svelte) │
   └──────────┘                      └──────────┘                       └──────────┘
```

---

## Supported Hardware

### Cameras

| Manufacturer | Models | Protocol | Notes |
|--------------|--------|----------|-------|
| **Uniview** | IPC3x, IPC6x series | ONVIF Profile S/T | Recommended for new installs |
| **Hikvision** | DS-2CDxxxx series | ONVIF Profile S/T | Widely available |
| **Dahua** | IPC-HDWxxxx series | ONVIF Profile S/T | Good value |
| **Axis** | P/M/Q series | ONVIF Profile S/T/G | Premium, excellent analytics |
| **Hanwha** | XNx series | ONVIF Profile S/T | Enterprise grade |
| **Generic** | ONVIF-compliant | ONVIF Profile S | Minimum requirement |

### NVRs

| Manufacturer | Models | Integration | Notes |
|--------------|--------|-------------|-------|
| **Uniview** | NVR30x series | ONVIF + API | Recommended |
| **Hikvision** | DS-7xxx/DS-9xxx | ONVIF + ISAPI | Disable cloud features |
| **Synology** | Surveillance Station | ONVIF + API | Good for small installs |
| **Milestone** | XProtect | ONVIF + API | Enterprise |
| **Blue Iris** | v5+ | ONVIF + API | Popular for DIY |

### Video Doorbells

| Type | Example | Integration |
|------|---------|-------------|
| **SIP-based** | 2N, Doorbird, Akuvox | Full integration (see [Access Control](access-control.md)) |
| **ONVIF** | Hikvision DS-KV series | Stream only |
| **Local API** | UniFi Protect G4 | Via HTTP API |
| **Cloud-only** | Ring, Nest | NOT SUPPORTED (violates offline-first) |

---

## Device Configuration

### Camera Entity

```yaml
Camera:
  id: uuid
  room_id: uuid | null              # Room where camera is located
  area_id: uuid | null              # Or area-level (outdoor, etc.)
  
  name: string                      # "Front Garden Camera"
  slug: string                      # "camera-front-garden"
  
  type: "camera"
  domain: "security"
  
  # Protocol
  protocol: "onvif"                 # or "rtsp" for stream-only
  
  # Connection
  address:
    host: "192.168.2.10"
    onvif_port: 80
    rtsp_port: 554
    username: "admin"               # Encrypted in database
    password: "********"            # Encrypted in database
    
  # Streams
  streams:
    main:
      url: "rtsp://192.168.2.10:554/main"
      resolution: "2560x1440"
      codec: "H.265"
      fps: 25
    sub:
      url: "rtsp://192.168.2.10:554/sub"
      resolution: "640x360"
      codec: "H.264"
      fps: 15
      
  # Capabilities
  capabilities:
    - video_stream
    - motion_detect
    - audio                         # If camera has microphone
    - ptz                           # If PTZ camera
    - ir_control                    # IR illuminator
    
  # Camera-specific config
  config:
    location: "outdoor"             # outdoor | indoor
    coverage: "perimeter"           # perimeter | entrance | general | private
    ptz:
      enabled: true
      presets:
        - id: 1
          name: "Gate View"
        - id: 2
          name: "Driveway Wide"
    motion:
      enabled: true
      sensitivity: 60               # 0-100
      regions:
        - name: "Driveway"
          enabled: true
        - name: "Pavement"
          enabled: false            # Ignore public footpath
          
  # Privacy settings
  privacy:
    hide_in_modes: ["night"]        # Don't show on panels at night
    require_pin_remote: true        # Require PIN to view remotely
    audio_disabled: true            # No audio capture
    
  # State
  state:
    online: boolean
    recording: boolean
    motion_active: boolean
    last_motion: timestamp
    
  created_at: timestamp
  updated_at: timestamp
```

### NVR Entity

```yaml
NVR:
  id: uuid
  site_id: uuid
  
  name: string                      # "Main NVR"
  slug: string                      # "nvr-main"
  
  type: "nvr"
  domain: "security"
  
  protocol: "onvif"
  
  address:
    host: "192.168.2.100"
    onvif_port: 80
    api_port: 443
    username: "admin"
    password: "********"
    
  # Managed cameras
  cameras:
    - camera_id: "camera-front-garden"
      nvr_channel: 1
    - camera_id: "camera-driveway"
      nvr_channel: 2
    - camera_id: "camera-rear"
      nvr_channel: 3
      
  # Recording status
  state:
    online: boolean
    recording: boolean
    disk_usage_percent: integer
    disk_health: "healthy" | "warning" | "failed"
    
  # Alerts
  alerts:
    disk_warning_percent: 80
    disk_critical_percent: 95
    notify: ["admin"]
```

---

## Residential CCTV

### Typical Camera Placement

| Location | Camera Type | Purpose |
|----------|-------------|---------|
| **Front door** | Doorbell camera or bullet | Visitor identification, package delivery |
| **Driveway** | Bullet or PTZ | Vehicle detection, approach warning |
| **Rear garden** | Bullet or dome | Perimeter coverage |
| **Side passage** | Bullet | Intruder detection |
| **Garage** | Indoor dome | Internal coverage, vehicle presence |
| **Pool area** | Dome (weather-rated) | Safety monitoring (not recording) |

### Residential Configuration Example

```yaml
residential_cctv:
  site_id: "site-oak-street"
  
  nvr:
    id: "nvr-main"
    manufacturer: "uniview"
    model: "NVR302-08S2"
    channels: 8
    storage_tb: 4
    retention_days: 30
    
  cameras:
    # Front entrance
    - id: "camera-front-door"
      name: "Front Door"
      room_id: null
      area_id: "area-exterior"
      type: "doorbell"              # Integrated with intercom
      location: "outdoor"
      coverage: "entrance"
      resolution: "1920x1080"
      config:
        motion_zones:
          - name: "Porch"
            action: "detect"
          - name: "Pavement"
            action: "ignore"
        package_detection: true     # AI feature if supported
        person_detection: true
        
    # Driveway
    - id: "camera-driveway"
      name: "Driveway"
      area_id: "area-exterior"
      type: "ptz"
      location: "outdoor"
      coverage: "perimeter"
      resolution: "2560x1440"
      config:
        ptz_enabled: true
        presets:
          - id: 1
            name: "Gate"
            default: true
          - id: 2
            name: "Garage"
        motion_zones:
          - name: "Driveway"
            action: "detect"
        vehicle_detection: true
        
    # Rear garden
    - id: "camera-rear-garden"
      name: "Rear Garden"
      area_id: "area-exterior"
      type: "bullet"
      location: "outdoor"
      coverage: "perimeter"
      resolution: "2560x1440"
      config:
        ir_enabled: true
        motion_zones:
          - name: "Lawn"
            action: "detect"
          - name: "Neighbour fence"
            action: "ignore"
            
    # Garage internal
    - id: "camera-garage"
      name: "Garage"
      room_id: "room-garage"
      type: "dome"
      location: "indoor"
      coverage: "general"
      resolution: "1920x1080"
      config:
        motion_enabled: true
        vehicle_presence: true      # Detect if car is in garage
```

### Residential Automation Triggers

```yaml
residential_camera_triggers:
  # Motion at front door
  - trigger:
      type: "camera_motion"
      camera_id: "camera-front-door"
      detection: "person"           # Or "motion" for any
      
    conditions:
      - mode: ["away", "night"]
      
    actions:
      - type: "notification"
        recipients: ["residents"]
        priority: "high"
        title: "Person at front door"
        include_snapshot: true
        
      - type: "record_clip"
        duration_seconds: 30
        
  # Vehicle approaching gate
  - trigger:
      type: "camera_motion"
      camera_id: "camera-driveway"
      detection: "vehicle"
      zone: "Gate"
      
    actions:
      - type: "ptz_preset"
        camera_id: "camera-driveway"
        preset: 1                   # "Gate" view
        
      - type: "show_camera"
        camera_id: "camera-driveway"
        targets: ["panel-kitchen"]
        duration_seconds: 30
        
  # Motion while on holiday
  - trigger:
      type: "camera_motion"
      detection: "person"
      # Any camera
      
    conditions:
      - mode: "holiday"
      
    actions:
      - type: "notification"
        recipients: ["admin"]
        priority: "critical"
        title: "Motion detected - Holiday mode"
        include_snapshot: true
        include_clip_link: true
```

### Doorbell/Intercom Camera Integration

When doorbell rings, automatically show camera on panels:

```yaml
doorbell_integration:
  intercom_id: "intercom-front"
  camera_id: "camera-front-door"    # May be same device
  
  on_ring:
    # Show on wall panels
    - action: "show_camera"
      targets:
        - "panel-hallway"
        - "panel-kitchen"
        - "panel-living"
      duration_seconds: 60
      with_audio: true              # Two-way if supported
      
    # Picture-in-picture on TV (if watching)
    - action: "tv_pip"
      targets: ["tv-living"]
      position: "top-right"
      size: "small"
      duration_seconds: 45
      
    # Mobile notification
    - action: "notification"
      recipients: ["residents"]
      title: "Someone at the door"
      include_live_view: true
      actions:
        - label: "Unlock"
          action: "unlock_door"
          door_id: "door-front"
          require_pin: true
        - label: "Talk"
          action: "open_intercom"
```

### Package Delivery

```yaml
package_detection:
  camera_id: "camera-front-door"
  
  # If camera supports package detection AI
  ai_detection:
    type: "package"
    confidence_threshold: 70
    
  # Fallback: motion + time of day
  fallback_detection:
    motion_zone: "Porch"
    time_window: "09:00-18:00"
    linger_seconds: 60              # Person leaves, object remains
    
  on_package:
    - action: "notification"
      recipients: ["residents"]
      title: "Package delivered"
      include_snapshot: true
      
    - action: "log_event"
      type: "package_delivery"
```

### Pet/Child Safety (Optional)

```yaml
# Pool camera safety monitoring (MONITORING ONLY - not a safety device)
pool_monitoring:
  camera_id: "camera-pool"
  
  # Only when pool gate open
  conditions:
    - device_state:
        device_id: "gate-pool"
        state: "open"
        
  detection:
    type: "motion"
    zone: "Pool Area"
    
  actions:
    - type: "notification"
      recipients: ["adults"]
      title: "Motion in pool area"
      priority: "high"
      
  # IMPORTANT: This is NOT a drowning detection system
  # It's a convenience notification only
  disclaimer: "This does not replace adult supervision"
```

---

## Commercial CCTV

### Typical Camera Placement (Office)

| Location | Camera Type | Purpose |
|----------|-------------|---------|
| **Reception** | PTZ or dome | Visitor identification |
| **Car park** | PTZ (multiple) | Vehicle tracking, ANPR |
| **Building entrance** | Bullet | Access control verification |
| **Corridors** | Dome | Movement tracking |
| **Server room** | Dome | High-security monitoring |
| **Loading bay** | Bullet | Delivery verification |
| **Perimeter** | Bullet + analytics | Intrusion detection |

### Commercial Configuration Example

```yaml
commercial_cctv:
  site_id: "site-office-building"
  
  nvr:
    id: "nvr-main"
    manufacturer: "milestone"
    model: "xprotect-corporate"
    channels: 64
    storage_tb: 48
    retention:
      default_days: 30
      high_security_days: 90        # Server room, entrance
      
  camera_groups:
    - name: "Perimeter"
      cameras:
        - "camera-north-fence"
        - "camera-south-fence"
        - "camera-east-fence"
        - "camera-west-fence"
      analytics:
        line_crossing: true
        intrusion_zone: true
        
    - name: "Car Park"
      cameras:
        - "camera-carpark-1"
        - "camera-carpark-2"
        - "camera-carpark-ptz"
      analytics:
        vehicle_tracking: true
        anpr_enabled: true
        
    - name: "Internal"
      cameras:
        - "camera-reception"
        - "camera-corridor-g"
        - "camera-corridor-1"
        - "camera-corridor-2"
      analytics:
        people_counting: true
        
  access_control_linking:
    # Show camera on access denied
    - event: "access_denied"
      door_id: "door-main-entrance"
      camera_id: "camera-main-entrance"
      action: "record_clip"
      duration_seconds: 60
      flag_for_review: true
      
    # Tailgating detection
    - event: "tailgate_detected"
      camera_id: "camera-reception"
      action: "alert"
      recipients: ["security"]
```

### Analytics Integration

```yaml
camera_analytics:
  # Line crossing (perimeter)
  line_crossing:
    camera_id: "camera-north-fence"
    lines:
      - name: "Fence Line"
        direction: "into"           # Alert on entering only
        schedule:
          enabled: "always"         # Or specific hours
          
    on_trigger:
      - action: "alert"
        priority: "high"
        recipients: ["security"]
        title: "Perimeter breach - North fence"
        
      - action: "record_clip"
        duration_seconds: 120
        
      - action: "ptz_track"
        tracking_camera: "camera-ptz-north"
        
  # People counting (reception)
  people_counting:
    camera_id: "camera-reception"
    
    counting_lines:
      - name: "Main entrance"
        direction: "bidirectional"
        
    output:
      publish_to_mqtt: true
      topic: "graylogic/analytics/reception/occupancy"
      interval_seconds: 60
      
  # ANPR (car park)
  anpr:
    camera_id: "camera-carpark-entrance"
    
    on_plate_read:
      - action: "log_event"
        type: "vehicle_entry"
        data: ["plate", "timestamp", "confidence"]
        
      - action: "check_whitelist"
        whitelist_id: "staff_vehicles"
        on_match:
          - action: "open_barrier"
            barrier_id: "barrier-carpark"
        on_no_match:
          - action: "alert"
            recipients: ["reception"]
            title: "Unknown vehicle at barrier"
```

### Multi-Site/Building Video Wall

```yaml
video_wall:
  # Security control room
  location: "security-office"
  
  displays:
    - id: "wall-1"
      resolution: "3840x2160"
      grid: "2x2"
      
    - id: "wall-2"
      resolution: "3840x2160"
      grid: "3x3"
      
  layouts:
    - name: "Default"
      assignments:
        wall-1:
          - "camera-main-entrance"
          - "camera-reception"
          - "camera-carpark-ptz"
          - "camera-loading-bay"
        wall-2:
          - "camera-corridor-g"
          - "camera-corridor-1"
          - "camera-corridor-2"
          - "camera-server-room"
          - "camera-north-fence"
          - "camera-south-fence"
          - "camera-east-fence"
          - "camera-west-fence"
          - "camera-roof"
          
    - name: "Incident"
      # Larger view of incident area
      auto_switch_on:
        - event: "alarm_triggered"
        - event: "access_denied_multiple"
```

---

## Stream Management

### Stream Selection

UIs should use appropriate stream quality:

| Context | Stream | Rationale |
|---------|--------|-----------|
| Wall panel thumbnail | Sub-stream | Low bandwidth, small display |
| Wall panel full-screen | Main stream | Full quality needed |
| Mobile on WiFi | Sub-stream → Main | Adaptive based on available bandwidth |
| Mobile on cellular | Sub-stream | Bandwidth conservation |
| Web admin | Sub-stream (list), Main (single) | Balance |
| Recording | Main stream | Always full quality |

### Stream Proxy

The CCTV Bridge can proxy streams to:
- Handle authentication centrally
- Provide consistent URLs to UIs
- Enable stream transcoding if needed
- Add access control

```yaml
stream_proxy:
  enabled: true
  
  # Internal proxy URLs
  url_template: "http://graylogic:8554/{camera_id}/{stream}"
  
  # Example:
  # http://graylogic:8554/camera-front-door/main
  # http://graylogic:8554/camera-front-door/sub
  
  # Authentication
  auth:
    type: "jwt"                     # Use Gray Logic auth tokens
    token_header: "Authorization"
    
  # Transcoding (optional, for incompatible clients)
  transcode:
    enabled: false                  # Enable if needed
    target_codec: "h264"            # For older devices
```

---

## Events and MQTT

### Event Topics

```
graylogic/cctv/{camera_id}/status          # Online/offline
graylogic/cctv/{camera_id}/motion          # Motion detection
graylogic/cctv/{camera_id}/analytics/{type} # Analytics events
graylogic/cctv/{nvr_id}/recording          # Recording status
graylogic/cctv/{nvr_id}/storage            # Storage alerts
```

### Event Payloads

```json
// Motion detection
{
  "camera_id": "camera-front-door",
  "event": "motion_start",
  "timestamp": "2026-01-12T14:32:15Z",
  "zone": "Porch",
  "detection_type": "person",
  "confidence": 85,
  "snapshot_url": "/api/cctv/camera-front-door/snapshot/20260112143215"
}

// Motion end
{
  "camera_id": "camera-front-door",
  "event": "motion_end",
  "timestamp": "2026-01-12T14:32:45Z",
  "duration_seconds": 30
}

// Analytics: line crossing
{
  "camera_id": "camera-north-fence",
  "event": "line_crossing",
  "timestamp": "2026-01-12T22:15:33Z",
  "line_name": "Fence Line",
  "direction": "into",
  "object_type": "person",
  "snapshot_url": "/api/cctv/camera-north-fence/snapshot/20260112221533"
}

// Storage warning
{
  "nvr_id": "nvr-main",
  "event": "storage_warning",
  "timestamp": "2026-01-12T08:00:00Z",
  "disk_usage_percent": 85,
  "estimated_days_remaining": 7
}
```

---

## UI Integration

### Wall Panel Camera View

```yaml
panel_camera_widget:
  type: "camera_grid"
  
  # Thumbnail grid
  grid:
    columns: 2
    cameras:
      - "camera-front-door"
      - "camera-driveway"
      - "camera-rear-garden"
      - "camera-garage"
      
  # Tap to expand
  on_tap: "fullscreen"
  
  # Swipe between cameras in fullscreen
  swipe_navigation: true
  
  # Auto-popup on motion (configurable)
  auto_popup:
    enabled: true
    cameras: ["camera-front-door"]  # Only doorbell
    duration_seconds: 30
    modes: ["home", "night"]        # Not when away (use mobile)
```

### Mobile App Camera View

```yaml
mobile_camera:
  # Live view
  live:
    default_stream: "sub"           # Start with sub-stream
    quality_selector: true          # Allow user to switch
    two_way_audio: true             # If supported
    
  # Playback (via NVR)
  playback:
    timeline: true                  # Show motion events on timeline
    clip_download: true             # Allow downloading clips
    
  # Notifications
  notifications:
    motion:
      include_snapshot: true
      include_live_button: true
      
  # Remote viewing
  remote:
    via_vpn_only: true              # No cloud relay
    require_pin: true               # Additional auth
```

---

## Privacy Controls

### Per-Camera Privacy Settings

```yaml
privacy_settings:
  camera_id: "camera-bedroom"
  
  # Completely disabled in these modes
  disabled_in_modes: ["home", "night"]
  
  # Or just hide from casual view
  hidden_in_modes: ["home"]
  require_pin_to_view: true
  
  # Audio
  audio_capture: false
  
  # Recording
  recording_enabled: false          # Or only in "away"
  recording_modes: ["away", "holiday"]
  
  # Analytics
  analytics_disabled: true
```

### Privacy Zones (Camera-Side)

Configure camera's built-in privacy masking for sensitive areas:

```yaml
# Configure via ONVIF or camera's native API
privacy_zones:
  camera_id: "camera-rear-garden"
  
  zones:
    - name: "Neighbour Window"
      type: "blackout"              # Black rectangle
      coordinates:
        x: 1800
        y: 200
        width: 400
        height: 600
        
    - name: "Hot Tub"
      type: "blur"                  # If camera supports
      coordinates:
        x: 100
        y: 800
        width: 600
        height: 400
```

### Audit Logging

```yaml
cctv_audit_log:
  events:
    - type: "live_view"
      log: true
      fields: ["user_id", "camera_id", "timestamp", "duration", "remote"]
      
    - type: "playback"
      log: true
      fields: ["user_id", "camera_id", "time_range", "timestamp", "remote"]
      
    - type: "snapshot_download"
      log: true
      
    - type: "clip_download"
      log: true
      
    - type: "settings_change"
      log: true
      require_reason: true          # User must state why
      
  retention_days: 365               # Keep audit logs 1 year
```

---

## Commissioning Checklist

### Residential

- [ ] All cameras discovered via ONVIF
- [ ] Stream URLs verified (main and sub)
- [ ] Motion detection zones configured
- [ ] Irrelevant zones excluded (public paths, neighbour areas)
- [ ] Privacy zones configured where needed
- [ ] NVR recording verified for each camera
- [ ] Retention period set appropriately
- [ ] Doorbell camera integrated with intercom
- [ ] Wall panel showing camera grid
- [ ] Mobile app receiving motion notifications
- [ ] Automation triggers tested (motion → notification)
- [ ] Remote viewing via VPN tested
- [ ] PIN required for remote camera access
- [ ] Audit logging enabled
- [ ] Client briefed on privacy controls

### Commercial

- [ ] All cameras discovered and named
- [ ] Camera groups configured
- [ ] Analytics configured (line crossing, counting, ANPR)
- [ ] Access control linking verified
- [ ] Video wall layouts configured
- [ ] Security team trained on interface
- [ ] Alert routing configured
- [ ] Incident response procedures documented
- [ ] Retention periods comply with regulations
- [ ] GDPR signage in place (where applicable)
- [ ] Data protection impact assessment completed
- [ ] Export/evidence procedures documented

---

## Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Stream not loading | Network/firewall | Check VLAN routing, RTSP port (554) |
| Choppy video | Bandwidth | Use sub-stream, check network capacity |
| Motion events missing | Camera config | Verify motion detection enabled on camera |
| High latency | Stream path | Use direct stream, not via NVR |
| Recording gaps | NVR storage | Check disk space, health |
| Discovery fails | ONVIF disabled | Enable ONVIF on camera, check port |

### Diagnostic Commands

```bash
# Test RTSP stream
ffprobe rtsp://admin:password@192.168.2.10:554/main

# Test ONVIF discovery
onvif-util discover

# Check camera status via API
curl -X GET http://graylogic/api/cctv/camera-front-door/status
```

---

## Related Documents

- [Principles](../overview/principles.md) — Privacy and security philosophy
- [System Overview](../architecture/system-overview.md) — Network architecture
- [Entities](../data-model/entities.md) — Camera and NVR device types
- [Access Control](access-control.md) — Intercom/doorbell integration
- [Fire Alarm](fire-alarm.md) — Emergency lighting and evacuation
- [Security Domain](../domains/security.md) — Alarm system integration (to be created)
