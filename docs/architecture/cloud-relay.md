---
title: Cloud Relay and Remote Connectivity
version: 0.1.0
status: draft
implementation_status: Year 2+
last_updated: 2026-01-18
depends_on:
  - overview/principles.md
  - architecture/security-model.md
  - integration/cctv.md
roadmap: Year 2+
---

# Cloud Relay and Remote Connectivity

> **Status**: Draft specification for Year 2+ implementation

This document specifies how Gray Logic provides secure remote access to residential and commercial installations via Gray Logic-hosted cloud infrastructure.

---

## Overview

### Design Philosophy

1. **Local-first remains paramount** — Cloud connectivity is an enhancement, never a dependency
2. **Zero-trust architecture** — Cloud relay never has access to decrypted user data
3. **End-to-end encryption** — All data encrypted from device to viewer
4. **Opt-in by default** — Users explicitly enable cloud features
5. **Subscription model** — Cloud infrastructure funded by recurring revenue

### What Cloud Relay Provides

- ✅ Remote API access without VPN configuration
- ✅ Push notifications to mobile devices
- ✅ Secure video relay for CCTV (opt-in)
- ✅ Cloud-based configuration backup
- ✅ Multi-site dashboard for estate/commercial
- ✅ AI-assisted insights and reporting

### What Cloud Relay Does NOT Provide

- ❌ Any control when local system is offline (relay only)
- ❌ Storage of unencrypted user data on cloud servers
- ❌ Access to data without user-held encryption keys
- ❌ Replacement for local functionality

---

## Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          GRAY LOGIC CLOUD                                    │
│                        (Gray Logic Hosted)                                   │
│                                                                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐             │
│  │   API Gateway   │  │  Video Relay    │  │  Push Service   │             │
│  │   (Pass-through)│  │  (E2E Encrypted)│  │  (FCM/APNs)     │             │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘             │
│           │                    │                    │                       │
│           └──────────────┬─────┴────────────────────┘                       │
│                          │                                                  │
│                   ┌──────▼──────┐                                          │
│                   │   Router    │                                          │
│                   │ (Site→Cloud)│                                          │
│                   └──────┬──────┘                                          │
│                          │                                                  │
└──────────────────────────┼──────────────────────────────────────────────────┘
                           │
                    WebSocket (WSS)
                    TLS 1.3 + mTLS
                           │
┌──────────────────────────▼──────────────────────────────────────────────────┐
│                         CUSTOMER SITE                                        │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │                      GRAY LOGIC CORE                                     ││
│  │                                                                          ││
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐       ││
│  │  │  Cloud Connector │  │  Video Proxy     │  │  Push Dispatcher │       ││
│  │  │  (API Relay)     │  │  (E2E Encrypts)  │  │  (Event→Push)    │       ││
│  │  └──────────────────┘  └──────────────────┘  └──────────────────┘       ││
│  │                                                                          ││
│  └─────────────────────────────────────────────────────────────────────────┘│
│                                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   Devices   │  │   Cameras   │  │   NVR       │  │   Local UI          │ │
│  │   (KNX etc) │  │   (ONVIF)   │  │  (Frigate)  │  │   (Panels/Web)      │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────────┘ │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Cloud Infrastructure (Gray Logic Hosted)

```yaml
cloud_infrastructure:
  provider: "Self-hosted"
  regions:
    primary: "UK (London)"
    future:
      - "EU (Frankfurt)"
      - "US (East)"

  components:
    api_gateway:
      technology: "Go (same stack as Core)"
      purpose: "Route authenticated requests to customer sites"
      stores_data: false           # Pass-through only

    video_relay:
      technology: "Go + WebRTC/SFU"
      purpose: "Relay encrypted video streams"
      stores_data: false           # Pass-through only
      encryption: "end-to-end"     # Gray Logic cloud cannot decrypt

    push_service:
      technology: "FCM (Android) + APNs (iOS)"
      purpose: "Deliver push notifications to mobile apps"
      stores_data: "24h queue only"

    site_registry:
      purpose: "Map site IDs to active connections"
      stores:
        - site_id
        - connection_metadata
        - subscription_tier
        - last_seen_timestamp
      encryption: "at rest"

  security:
    authentication: "mTLS (site→cloud) + JWT (user→API)"
    rate_limiting: true
    ddos_protection: true
    audit_logging: true
```

### Site-to-Cloud Connection

```yaml
cloud_connector:
  # Outbound connection from Core to Cloud
  connection_type: "WebSocket (WSS)"
  direction: "outbound"            # No inbound ports required
  
  protocol: "TLS 1.3 + mTLS"
  mutual_tls:
    site_certificate: "/etc/graylogic/cloud/site.crt"
    site_private_key: "/etc/graylogic/cloud/site.key"
    cloud_ca: "/etc/graylogic/cloud/graylogic-cloud-ca.crt"
  
  keepalive:
    interval_seconds: 30
    timeout_seconds: 90
  
  reconnection:
    strategy: "exponential_backoff"
    initial_delay_ms: 1000
    max_delay_ms: 300000           # 5 minutes max
    jitter: true
  
  # Disable when internet unavailable
  health_check:
    interval_seconds: 60
    on_internet_loss: "disconnect_gracefully"
```

---

## API Relay

### How It Works

```
Mobile App ──[HTTPS]──▶ Gray Logic Cloud ──[WSS]──▶ Customer Site ──▶ Gray Logic Core
     │                        │                           │                   │
     │     JWT Auth           │     mTLS Auth             │    Local          │
     │     + Device Binding   │     + Site ID             │    Processing     │
     ▼                        ▼                           ▼                   ▼
  Response ◀──────────────────────────────────────────────────────────────────
```

### Request Flow

```yaml
api_relay:
  request_flow:
    1: "Mobile app sends HTTPS request to cloud API with JWT token"
    2: "Cloud validates JWT, extracts user_id and site_id"
    3: "Cloud looks up site's active WebSocket connection"
    4: "Cloud forwards request to site via WebSocket"
    5: "Gray Logic Core processes request locally"
    6: "Response returned via same path"
    7: "Mobile app receives response"
  
  latency_budget:
    cloud_processing: "50ms"
    site_round_trip: "variable (internet latency)"
    total_target: "<500ms typical"
  
  timeout:
    default: "30 seconds"
    file_transfer: "120 seconds"
  
  # Cloud never decrypts request body for control commands
  zero_knowledge:
    control_commands: true         # Encrypted end-to-end
    status_queries: true           # Encrypted end-to-end
    only_metadata_visible: true    # Cloud sees site_id, timestamp, size
```

### Authentication

```yaml
remote_authentication:
  # User → Cloud
  user_to_cloud:
    method: "JWT + Device Binding"
    token_source: "Gray Logic Core (issued locally)"
    token_validation: "Public key verification at cloud"
    
  # Cloud → Site
  cloud_to_site:
    method: "mTLS + Site Certificate"
    provisioned_at: "Cloud subscription activation"
    rotation: "Automatic, yearly"
    
  # Device binding (prevent token theft)
  device_binding:
    bind_to: "device_id + device_fingerprint"
    first_access_mfa: true
    allow_multiple_devices: true
    max_devices_per_user: 10
```

---

## Video Relay (CCTV Remote Viewing)

### End-to-End Encryption

```yaml
video_encryption:
  method: "End-to-end encryption"
  encryption_point: "Gray Logic Core (on-site)"
  decryption_point: "Mobile app (on-device)"
  cloud_can_decrypt: false         # Never has access to keys
  
  key_management:
    key_derivation: "User password-derived key + site master key"
    key_exchange: "ECDH (Curve25519)"
    key_rotation: "Per-session"
    
  encryption_algorithm:
    video: "AES-256-GCM"
    metadata: "ChaCha20-Poly1305"
```

### Video Stream Architecture

```yaml
video_relay:
  technology: "WebRTC-based relay (SFU mode)"
  
  flow:
    1: "User requests camera stream via mobile app"
    2: "Cloud verifies subscription tier (Secure or higher)"
    3: "Cloud forwards request to site via WebSocket"
    4: "Gray Logic Core fetches stream from camera/NVR"
    5: "Core encrypts stream with session key"
    6: "Encrypted stream relayed via cloud SFU"
    7: "Mobile app decrypts with session key"
    8: "User views live video"
  
  # Quality adaptation
  quality:
    adaptive_bitrate: true
    max_resolution: "1080p"        # Limit for bandwidth
    min_resolution: "360p"         # Minimum usable
    codec: "H.264 or VP9"
  
  # Cloud relay limits
  limits:
    max_streams_per_site: 4
    max_stream_duration_minutes: 60
    idle_timeout_seconds: 300
```

### Audit Logging

```yaml
video_access_audit:
  logged_events:
    - event: "video_stream_requested"
      fields: [user_id, camera_id, timestamp, ip_address]
    - event: "video_stream_started"
      fields: [user_id, camera_id, timestamp, quality]
    - event: "video_stream_ended"
      fields: [user_id, camera_id, timestamp, duration_seconds]
    - event: "video_access_denied"
      fields: [user_id, camera_id, timestamp, reason]
  
  never_logged:
    - video_content
    - audio_content
    - video_thumbnails
  
  retention:
    on_site: "1 year"
    on_cloud: "90 days (metadata only)"
```

---

## Push Notifications

### Architecture

```yaml
push_notifications:
  providers:
    android: "Firebase Cloud Messaging (FCM)"
    ios: "Apple Push Notification Service (APNs)"
  
  flow:
    1: "Event occurs on-site (doorbell, alarm, motion)"
    2: "Gray Logic Core filters event against notification rules"
    3: "Core sends notification request to cloud via WebSocket"
    4: "Cloud dispatches to FCM/APNs"
    5: "Mobile device receives push"
  
  # What triggers push notifications
  trigger_events:
    security:
      - "alarm_triggered"
      - "door_forced"
      - "access_denied"
    camera:
      - "motion_person_detected"
      - "doorbell_ring"
      - "package_delivered"
    system:
      - "phm_warning"
      - "system_offline"
      - "low_battery"
  
  # User preferences
  user_preferences:
    notification_schedule: true    # Quiet hours
    per_event_enable: true         # Per event type
    per_camera_enable: true        # Per camera
```

---

## Recommended NVR Platform (Frigate)

Based on research, **Frigate** is recommended as the preferred open-source NVR for Gray Logic integration:

### Why Frigate

| Feature | Benefit for Gray Logic |
|---------|------------------------|
| **Local AI** | Person/vehicle/face detection locally, no cloud |
| **MQTT native** | Direct integration with Gray Logic Core |
| **Home Assistant compatible** | Familiar ecosystem, proven stability |
| **Active development** | Face + license plate recognition (v0.16, Aug 2025) |
| **Hardware acceleration** | GPU/TPU support for efficient processing |
| **Open source** | No vendor lock-in |

### Integration Architecture

```yaml
frigate_integration:
  role: "Preferred NVR for new Gray Logic installations"
  deployment: "Co-located on Gray Logic Hub or separate HW"
  
  # Communication
  api:
    type: "REST + MQTT"
    url: "http://frigate:5000/api"
    mqtt_topics:
      detections: "frigate/+/+/+/state"
      events: "frigate/events"
      cameras: "frigate/cameras/+/state"
  
  # Gray Logic NVR Bridge
  nvr_bridge:
    purpose: "Translate Frigate events to Gray Logic format"
    protocol: "MQTT"
    mappings:
      frigate_person: "motion_person_detected"
      frigate_car: "motion_vehicle_detected"
      frigate_dog: "motion_animal_detected"
  
  # Remote API control
  remote_capabilities:
    via_gray_logic_cloud:
      - "View live streams"
      - "View recordings (via Frigate API)"
      - "Control PTZ cameras"
      - "Enable/disable detection zones"
      - "Trigger recording clips"
    not_available_remote:
      - "Frigate configuration changes"
      - "System administration"
```

### Alternative: Shinobi

For customers requiring:
- More cameras (Shinobi scales to 100+)
- Node.js ecosystem preference
- Existing Shinobi installations

```yaml
shinobi_support:
  status: "Supported but not preferred"
  integration: "REST API + MQTT"
  note: "User responsible for Shinobi configuration"
```

---

## Subscription Tiers

See [Subscription Pricing](../business/subscription-pricing.md) for detailed pricing.

### Feature Matrix

| Feature | Free | Connect | Secure | Premium | Estate |
|---------|------|---------|--------|---------|--------|
| **Local control** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **VPN access** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Push notifications** | ❌ | ✅ | ✅ | ✅ | ✅ |
| **Cloud API relay** | ❌ | ✅ | ✅ | ✅ | ✅ |
| **Cloud config backup** | ❌ | ✅ | ✅ | ✅ | ✅ |
| **Remote CCTV viewing** | ❌ | ❌ | ✅ | ✅ | ✅ |
| **Video clip storage** | ❌ | ❌ | 7 days | 30 days | 90 days |
| **AI insights** | ❌ | ❌ | ❌ | ✅ | ✅ |
| **Multi-site dashboard** | ❌ | ❌ | ❌ | ✅ | ✅ (5 sites) |
| **Priority support** | ❌ | ❌ | ❌ | ❌ | ✅ |
| **SLA guarantee** | ❌ | ❌ | ❌ | ❌ | 99.9% |

---

## Security Considerations

### Zero-Knowledge Design

```yaml
zero_knowledge:
  principle: "Gray Logic Cloud cannot access user data"
  
  implementation:
    control_commands:
      encryption: "End-to-end (user key → site key)"
      cloud_access: "Metadata only (routing)"
    
    video_streams:
      encryption: "End-to-end (session key)"
      cloud_access: "Encrypted bytes only"
    
    configuration_backup:
      encryption: "User-held key (GPG or AES)"
      cloud_access: "Encrypted blob only"
    
    push_notifications:
      content: "Minimal (event type, no sensitive data)"
      example: "Motion at Front Door" (not detailed description)
```

### Compromise Scenarios

```yaml
if_cloud_compromised:
  attacker_can_see:
    - "Site IDs and connection metadata"
    - "Encrypted video streams (cannot decrypt)"
    - "Request/response sizes and timing"
    - "Push notification metadata"
  
  attacker_cannot_see:
    - "Control commands (encrypted)"
    - "Video content (encrypted)"
    - "User credentials (not stored in cloud)"
    - "Site configuration (encrypted if backed up)"
  
  attacker_cannot_do:
    - "Control devices (requires site key)"
    - "View cameras (requires session key)"
    - "Access site directly (requires mTLS cert)"
```

---

## Provisioning

### Subscription Activation

```yaml
subscription_activation:
  flow:
    1: "User purchases subscription via website/app"
    2: "User provides site_id (from Gray Logic Core)"
    3: "Cloud generates site certificate for mTLS"
    4: "User enters activation code on Gray Logic Core"
    5: "Core downloads certificate and connects to cloud"
    6: "Subscription active"
  
  activation_code:
    format: "GLXXXX-XXXX-XXXX-XXXX"
    validity: "7 days"
    single_use: true
```

---

## Offline Behaviour

### When Internet Unavailable

```yaml
offline_behaviour:
  local_functionality: "100% operational"
  
  cloud_features:
    api_relay: "Unavailable (graceful timeout)"
    video_relay: "Unavailable"
    push_notifications: "Queued (24h max)"
  
  reconnection:
    automatic: true
    queued_notifications: "Delivered on reconnect"
    state_sync: "Full state sync on reconnect"
```

---

## References

- [Security Model](security-model.md) — Local security architecture
- [Subscription Pricing](../business/subscription-pricing.md) — Pricing tiers
- [CCTV Integration](../integration/cctv.md) — Camera integration
- [AI Premium Features](../intelligence/ai-premium-features.md) — AI capabilities
