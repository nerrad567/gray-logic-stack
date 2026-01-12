---
title: REST and WebSocket API Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/system-overview.md
  - architecture/core-internals.md
  - data-model/entities.md
  - overview/principles.md
---

# REST and WebSocket API Specification

This document specifies the HTTP REST API and WebSocket interface for Gray Logic Core. All user interfaces (Flutter app, web dashboard, wall panels) and third-party integrations communicate with Core through these APIs.

---

## Overview

### API Philosophy

Gray Logic provides two complementary interfaces:

| Interface | Purpose | Use Case |
|-----------|---------|----------|
| **REST API** | CRUD operations, commands | Configuration, device control, queries |
| **WebSocket** | Real-time events | Live state updates, notifications |

### Design Principles

1. **Offline-first** — API works without internet; authentication is local
2. **10-year stability** — Versioned URLs; backwards compatibility guaranteed
3. **Simple and predictable** — RESTful conventions, consistent patterns
4. **Secure by default** — Authentication required, audit logging enabled

### Base URL

```
http://{host}:8080/api/v1/
```

- **Development**: `http://localhost:8080/api/v1/`
- **Production**: `https://graylogic.local/api/v1/` (with TLS)

### Versioning

URL-based versioning with long-term support:

| Version | Status | Support Until |
|---------|--------|---------------|
| `v1` | Active | 2036+ |
| `v2` | (Future) | - |

**Commitment**: `v1` endpoints will not have breaking changes. New features are additive.

### Content Type

All requests and responses use JSON:

```
Content-Type: application/json
Accept: application/json
```

---

## Authentication

### Overview

Gray Logic uses JWT (JSON Web Tokens) for authentication. Tokens are issued locally — no cloud dependency.

### Login

```http
POST /api/v1/auth/login
```

**Request:**
```json
{
  "username": "darren",
  "password": "secure_password"
}
```

**Successful Response (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": {
    "id": "usr-001",
    "name": "Darren",
    "role": "admin"
  }
}
```

**Error Response (401):**
```json
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Invalid username or password"
  }
}
```

### Token Refresh

```http
POST /api/v1/auth/refresh
```

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600
}
```

### PIN Authentication

For quick access on trusted devices (wall panels):

```http
POST /api/v1/auth/pin
```

**Request:**
```json
{
  "user_id": "usr-001",
  "pin": "1234"
}
```

### Logout

```http
POST /api/v1/auth/logout
```

Invalidates the current refresh token.

### Using Tokens

Include the access token in the Authorization header:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Token Lifetimes

| Token Type | Lifetime | Renewable |
|------------|----------|-----------|
| Access Token | 1 hour | Via refresh token |
| Refresh Token | 30 days | On use (rolling) |
| API Key | Never expires | Revocable |

### API Keys

For integrations and service accounts:

```http
POST /api/v1/auth/apikeys
```

**Request:**
```json
{
  "name": "Home Assistant Integration",
  "permissions": ["devices:read", "devices:control"],
  "expires_at": null
}
```

**Response (201):**
```json
{
  "id": "key-001",
  "key": "gl_live_abc123def456...",
  "name": "Home Assistant Integration",
  "permissions": ["devices:read", "devices:control"],
  "created_at": "2026-01-12T10:00:00Z"
}
```

**Usage:**
```http
Authorization: Bearer gl_live_abc123def456...
```

---

## Authorization

### Roles

| Role | Description | Permissions |
|------|-------------|-------------|
| `admin` | Full system access | All operations |
| `resident` | Normal user | Control devices, run scenes, view status |
| `guest` | Limited access | Control assigned rooms only |
| `installer` | Technical access | Configuration, no sensitive data |
| `integration` | API access | Scoped to API key permissions |

### Permission Model

Permissions follow the pattern: `{resource}:{action}`

| Permission | Description |
|------------|-------------|
| `devices:read` | View device state |
| `devices:control` | Control devices |
| `devices:configure` | Modify device configuration |
| `scenes:read` | View scenes |
| `scenes:execute` | Activate scenes |
| `scenes:manage` | Create/modify scenes |
| `schedules:read` | View schedules |
| `schedules:manage` | Create/modify schedules |
| `modes:read` | View current mode |
| `modes:change` | Change system mode |
| `users:read` | View user list |
| `users:manage` | Create/modify users |
| `system:read` | View system status |
| `system:configure` | Modify system configuration |
| `energy:read` | View energy data |
| `security:read` | View security status |
| `security:control` | Arm/disarm, unlock doors |

### Room-Level Access Control

Guests can be restricted to specific rooms:

```json
{
  "user_id": "usr-guest-01",
  "role": "guest",
  "access": {
    "rooms": ["room-guest-bedroom", "room-guest-bathroom"],
    "valid_until": "2026-01-15T12:00:00Z"
  }
}
```

---

## Response Format

### Successful Responses

**Single Resource:**
```json
{
  "data": {
    "id": "device-001",
    "name": "Living Room Light",
    "...": "..."
  }
}
```

**Collection:**
```json
{
  "data": [
    { "id": "device-001", "...": "..." },
    { "id": "device-002", "...": "..." }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 127,
    "total_pages": 3
  }
}
```

**Action Response:**
```json
{
  "success": true,
  "message": "Scene activated",
  "request_id": "req-abc123"
}
```

### Error Responses

```json
{
  "error": {
    "code": "DEVICE_OFFLINE",
    "message": "Device is not responding",
    "details": {
      "device_id": "light-living-main",
      "last_seen": "2026-01-12T09:30:00Z"
    },
    "request_id": "req-xyz789"
  }
}
```

### HTTP Status Codes

| Code | Meaning | Use Case |
|------|---------|----------|
| 200 | OK | Successful GET, PUT, PATCH |
| 201 | Created | Successful POST (create) |
| 204 | No Content | Successful DELETE |
| 400 | Bad Request | Invalid request body |
| 401 | Unauthorized | Missing or invalid token |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Resource already exists |
| 422 | Unprocessable | Validation failed |
| 429 | Too Many Requests | Rate limited |
| 500 | Server Error | Internal error |
| 502 | Bad Gateway | Bridge/protocol error |
| 503 | Unavailable | Service temporarily unavailable |

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_CREDENTIALS` | 401 | Wrong username/password |
| `TOKEN_EXPIRED` | 401 | Access token expired |
| `INSUFFICIENT_PERMISSIONS` | 403 | User lacks required permission |
| `RESOURCE_NOT_FOUND` | 404 | Entity doesn't exist |
| `DEVICE_OFFLINE` | 502 | Device not responding |
| `BRIDGE_OFFLINE` | 502 | Protocol bridge not connected |
| `VALIDATION_ERROR` | 422 | Request body validation failed |
| `RATE_LIMITED` | 429 | Too many requests |
| `COMMAND_TIMEOUT` | 504 | Device didn't respond in time |
| `COMMAND_FAILED` | 502 | Device rejected command |

---

## Query Parameters

### Pagination

```
GET /api/v1/devices?page=2&limit=25
```

| Parameter | Default | Max | Description |
|-----------|---------|-----|-------------|
| `page` | 1 | - | Page number (1-indexed) |
| `limit` | 50 | 100 | Items per page |

### Filtering

```
GET /api/v1/devices?room_id=room-living&domain=lighting
GET /api/v1/devices?type=light_dimmer,light_rgb
GET /api/v1/devices?health.status=offline
```

Multiple values use comma separation.

### Sorting

```
GET /api/v1/devices?sort=name&order=asc
GET /api/v1/scenes?sort=updated_at&order=desc
```

| Parameter | Values | Default |
|-----------|--------|---------|
| `sort` | Any field | `name` |
| `order` | `asc`, `desc` | `asc` |

### Field Selection

```
GET /api/v1/devices?fields=id,name,state
```

Reduces response size for bandwidth-constrained clients.

### Time Ranges

```
GET /api/v1/energy/history?from=2026-01-01T00:00:00Z&to=2026-01-12T00:00:00Z
GET /api/v1/energy/history?period=7d
```

| Parameter | Format | Example |
|-----------|--------|---------|
| `from` | ISO 8601 | `2026-01-01T00:00:00Z` |
| `to` | ISO 8601 | `2026-01-12T00:00:00Z` |
| `period` | Duration | `1h`, `24h`, `7d`, `30d` |

### Include Related Resources

```
GET /api/v1/rooms/room-living?include=devices,climate_zone
```

Embeds related resources to reduce round trips.

---

## REST API Endpoints

### Site

#### Get Site Information

```http
GET /api/v1/site
```

**Response (200):**
```json
{
  "data": {
    "id": "site-001",
    "name": "Oak Street Residence",
    "slug": "oak-street",
    "location": {
      "address": "123 Oak Street, London",
      "latitude": 51.5074,
      "longitude": -0.1278,
      "timezone": "Europe/London",
      "elevation_m": 35
    },
    "modes": {
      "available": ["home", "away", "night", "holiday", "entertaining"],
      "current": "home"
    },
    "settings": {
      "units": {
        "temperature": "celsius",
        "distance": "metric"
      },
      "locale": "en-GB"
    }
  }
}
```

#### Update Site Settings

```http
PATCH /api/v1/site
```

**Request:**
```json
{
  "name": "Oak Street Home",
  "settings": {
    "units": {
      "temperature": "fahrenheit"
    }
  }
}
```

**Required Permission:** `system:configure`

---

### Areas

#### List Areas

```http
GET /api/v1/areas
```

**Response (200):**
```json
{
  "data": [
    {
      "id": "area-ground",
      "name": "Ground Floor",
      "slug": "ground-floor",
      "type": "floor",
      "sort_order": 1,
      "room_count": 6
    },
    {
      "id": "area-first",
      "name": "First Floor",
      "slug": "first-floor",
      "type": "floor",
      "sort_order": 2,
      "room_count": 5
    }
  ]
}
```

#### Get Area

```http
GET /api/v1/areas/{area_id}
GET /api/v1/areas/{area_id}?include=rooms
```

#### Create Area

```http
POST /api/v1/areas
```

**Request:**
```json
{
  "name": "Basement",
  "type": "floor",
  "sort_order": 0
}
```

**Required Permission:** `system:configure`

#### Update Area

```http
PATCH /api/v1/areas/{area_id}
```

#### Delete Area

```http
DELETE /api/v1/areas/{area_id}
```

---

### Rooms

#### List Rooms

```http
GET /api/v1/rooms
GET /api/v1/rooms?area_id=area-ground
```

**Response (200):**
```json
{
  "data": [
    {
      "id": "room-living",
      "area_id": "area-ground",
      "name": "Living Room",
      "slug": "living-room",
      "type": "living",
      "sort_order": 1,
      "climate_zone_id": "cz-001",
      "audio_zone_id": "az-001",
      "device_count": 12,
      "settings": {
        "default_scene": null,
        "vacancy_timeout_min": 15
      }
    }
  ]
}
```

#### Get Room

```http
GET /api/v1/rooms/{room_id}
GET /api/v1/rooms/{room_id}?include=devices
```

**Response with include (200):**
```json
{
  "data": {
    "id": "room-living",
    "name": "Living Room",
    "...": "...",
    "devices": [
      {
        "id": "light-living-main",
        "name": "Main Light",
        "type": "light_dimmer",
        "state": {
          "on": true,
          "brightness": 75
        }
      }
    ]
  }
}
```

#### Get Room Devices

```http
GET /api/v1/rooms/{room_id}/devices
GET /api/v1/rooms/{room_id}/devices?domain=lighting
```

#### Create Room

```http
POST /api/v1/rooms
```

**Request:**
```json
{
  "area_id": "area-ground",
  "name": "Study",
  "type": "office",
  "settings": {
    "vacancy_timeout_min": 30
  }
}
```

**Required Permission:** `system:configure`

#### Update Room

```http
PATCH /api/v1/rooms/{room_id}
```

#### Delete Room

```http
DELETE /api/v1/rooms/{room_id}
```

---

### Devices

#### List Devices

```http
GET /api/v1/devices
GET /api/v1/devices?room_id=room-living
GET /api/v1/devices?domain=lighting
GET /api/v1/devices?type=light_dimmer,light_rgb
GET /api/v1/devices?health.status=offline
```

**Response (200):**
```json
{
  "data": [
    {
      "id": "light-living-main",
      "room_id": "room-living",
      "name": "Main Light",
      "slug": "main-light",
      "type": "light_dimmer",
      "domain": "lighting",
      "protocol": "knx",
      "capabilities": ["on_off", "dim"],
      "state": {
        "on": true,
        "brightness": 75
      },
      "state_updated_at": "2026-01-12T14:30:00Z",
      "health": {
        "status": "online",
        "last_seen": "2026-01-12T14:30:00Z"
      }
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 127,
    "total_pages": 3
  }
}
```

#### Get Device

```http
GET /api/v1/devices/{device_id}
```

**Response (200):**
```json
{
  "data": {
    "id": "light-living-main",
    "room_id": "room-living",
    "area_id": null,
    "name": "Main Light",
    "slug": "main-light",
    "type": "light_dimmer",
    "domain": "lighting",
    "protocol": "knx",
    "address": {
      "group_address": "1/2/3",
      "feedback_address": "1/2/4"
    },
    "capabilities": ["on_off", "dim"],
    "config": {
      "min_brightness": 10,
      "transition_default_ms": 500
    },
    "state": {
      "on": true,
      "brightness": 75
    },
    "state_updated_at": "2026-01-12T14:30:00Z",
    "health": {
      "status": "online",
      "last_seen": "2026-01-12T14:30:00Z",
      "phm_enabled": true,
      "phm_baseline": {
        "typical_on_hours": 6.5,
        "typical_brightness": 65
      }
    },
    "manufacturer": "MDT",
    "model": "AKD-0424V.02",
    "firmware_version": "1.4"
  }
}
```

#### Get Device State

```http
GET /api/v1/devices/{device_id}/state
```

Lightweight endpoint for state polling:

**Response (200):**
```json
{
  "data": {
    "on": true,
    "brightness": 75
  },
  "updated_at": "2026-01-12T14:30:00Z"
}
```

#### Set Device State (Control)

```http
PUT /api/v1/devices/{device_id}/state
```

**Request (Light):**
```json
{
  "on": true,
  "brightness": 50,
  "transition_ms": 1000
}
```

**Request (Blind):**
```json
{
  "position": 75,
  "tilt": 45
}
```

**Request (Thermostat):**
```json
{
  "target_temp": 21.5,
  "mode": "heating"
}
```

**Response (200):**
```json
{
  "success": true,
  "request_id": "req-abc123",
  "state": {
    "on": true,
    "brightness": 50
  }
}
```

**Required Permission:** `devices:control`

#### Device Commands

For complex operations beyond simple state setting:

```http
POST /api/v1/devices/{device_id}/commands
```

**Request (Identify):**
```json
{
  "command": "identify",
  "parameters": {
    "duration_seconds": 10
  }
}
```

**Request (Calibrate):**
```json
{
  "command": "calibrate",
  "parameters": {
    "type": "full"
  }
}
```

**Response (202):**
```json
{
  "success": true,
  "request_id": "req-xyz789",
  "message": "Command queued"
}
```

#### Create Device

```http
POST /api/v1/devices
```

**Request:**
```json
{
  "room_id": "room-living",
  "name": "Accent Light",
  "type": "light_rgb",
  "domain": "lighting",
  "protocol": "dali",
  "address": {
    "gateway": "dali-gw-01",
    "short_address": 15
  },
  "capabilities": ["on_off", "dim", "color_rgb"]
}
```

**Required Permission:** `devices:configure`

#### Update Device

```http
PATCH /api/v1/devices/{device_id}
```

#### Delete Device

```http
DELETE /api/v1/devices/{device_id}
```

---

### Batch Device Control

Control multiple devices in a single request:

```http
POST /api/v1/devices/batch
```

**Request:**
```json
{
  "actions": [
    {
      "device_id": "light-living-main",
      "state": { "on": false }
    },
    {
      "device_id": "light-living-accent",
      "state": { "on": false }
    },
    {
      "device_id": "blind-living-01",
      "state": { "position": 100 }
    }
  ]
}
```

**Response (200):**
```json
{
  "success": true,
  "request_id": "req-batch-001",
  "results": [
    { "device_id": "light-living-main", "success": true },
    { "device_id": "light-living-accent", "success": true },
    { "device_id": "blind-living-01", "success": true }
  ]
}
```

### Room-Level Control

Control all devices of a domain in a room:

```http
PUT /api/v1/rooms/{room_id}/devices/state
```

**Request:**
```json
{
  "domain": "lighting",
  "state": {
    "on": false
  }
}
```

Turns off all lights in the room.

---

### Scenes

#### List Scenes

```http
GET /api/v1/scenes
GET /api/v1/scenes?scope.type=room&scope.id=room-living
GET /api/v1/scenes?category=entertainment
```

**Response (200):**
```json
{
  "data": [
    {
      "id": "scene-cinema",
      "name": "Cinema Mode",
      "slug": "cinema-mode",
      "icon": "movie",
      "color": "#6B21A8",
      "scope": {
        "type": "room",
        "id": "room-living"
      },
      "category": "entertainment",
      "triggers": {
        "manual": true,
        "voice_phrase": "cinema mode"
      },
      "action_count": 5
    }
  ]
}
```

#### Get Scene

```http
GET /api/v1/scenes/{scene_id}
```

Returns full scene definition including all actions.

#### Activate Scene

```http
POST /api/v1/scenes/{scene_id}/activate
```

**Request (optional overrides):**
```json
{
  "skip_conditions": false,
  "overrides": {
    "brightness_scale": 0.5
  }
}
```

**Response (200):**
```json
{
  "success": true,
  "request_id": "req-scene-001",
  "message": "Scene 'Cinema Mode' activated",
  "actions_executed": 5
}
```

**Required Permission:** `scenes:execute`

#### Create Scene

```http
POST /api/v1/scenes
```

**Request:**
```json
{
  "name": "Reading Mode",
  "scope": {
    "type": "room",
    "id": "room-living"
  },
  "icon": "book",
  "color": "#F59E0B",
  "triggers": {
    "manual": true,
    "voice_phrase": "reading mode"
  },
  "actions": [
    {
      "device_id": "light-living-main",
      "command": "set",
      "parameters": { "on": true, "brightness": 80 },
      "fade_ms": 2000
    },
    {
      "device_id": "light-living-accent",
      "command": "set",
      "parameters": { "on": true, "brightness": 30 },
      "fade_ms": 2000,
      "parallel": true
    }
  ],
  "category": "comfort"
}
```

**Required Permission:** `scenes:manage`

#### Update Scene

```http
PATCH /api/v1/scenes/{scene_id}
```

#### Delete Scene

```http
DELETE /api/v1/scenes/{scene_id}
```

---

### Schedules

#### List Schedules

```http
GET /api/v1/schedules
GET /api/v1/schedules?enabled=true
```

**Response (200):**
```json
{
  "data": [
    {
      "id": "sched-morning",
      "name": "Morning Wake Up",
      "enabled": true,
      "trigger": {
        "type": "time",
        "value": "06:30",
        "days": ["mon", "tue", "wed", "thu", "fri"]
      },
      "execute": {
        "type": "scene",
        "scene_id": "scene-morning"
      },
      "next_run": "2026-01-13T06:30:00Z"
    }
  ]
}
```

#### Get Schedule

```http
GET /api/v1/schedules/{schedule_id}
```

#### Create Schedule

```http
POST /api/v1/schedules
```

**Request:**
```json
{
  "name": "Evening Lights",
  "enabled": true,
  "trigger": {
    "type": "sunset",
    "value": "-15m",
    "days": null
  },
  "execute": {
    "type": "scene",
    "scene_id": "scene-evening"
  },
  "conditions": [
    {
      "type": "mode",
      "operator": "eq",
      "value": "home"
    }
  ]
}
```

**Required Permission:** `schedules:manage`

#### Update Schedule

```http
PATCH /api/v1/schedules/{schedule_id}
```

#### Enable/Disable Schedule

```http
POST /api/v1/schedules/{schedule_id}/enable
POST /api/v1/schedules/{schedule_id}/disable
```

#### Delete Schedule

```http
DELETE /api/v1/schedules/{schedule_id}
```

---

### Modes

#### List Available Modes

```http
GET /api/v1/modes
```

**Response (200):**
```json
{
  "data": [
    {
      "id": "home",
      "name": "Home",
      "icon": "home",
      "color": "#22C55E",
      "is_current": true,
      "behaviours": {
        "climate": {
          "setpoint_offset": 0,
          "eco_mode": false
        },
        "lighting": {
          "auto_off_enabled": true,
          "auto_off_delay_min": 15
        }
      }
    },
    {
      "id": "away",
      "name": "Away",
      "icon": "door-open",
      "color": "#EAB308",
      "is_current": false,
      "behaviours": {
        "climate": {
          "setpoint_offset": -3,
          "eco_mode": true
        }
      }
    }
  ]
}
```

#### Get Current Mode

```http
GET /api/v1/modes/current
```

**Response (200):**
```json
{
  "data": {
    "id": "home",
    "name": "Home",
    "activated_at": "2026-01-12T08:30:00Z",
    "activated_by": {
      "type": "user",
      "user_id": "usr-001"
    }
  }
}
```

#### Activate Mode

```http
POST /api/v1/modes/{mode_id}/activate
```

**Request (optional):**
```json
{
  "pin": "1234"
}
```

**Response (200):**
```json
{
  "success": true,
  "previous_mode": "home",
  "new_mode": "away",
  "message": "Mode changed to 'Away'"
}
```

**Required Permission:** `modes:change`

---

### Users

#### List Users

```http
GET /api/v1/users
```

**Required Permission:** `users:read`

**Response (200):**
```json
{
  "data": [
    {
      "id": "usr-001",
      "name": "Darren",
      "email": "darren@example.com",
      "role": "admin",
      "presence": {
        "is_home": true,
        "last_seen": "2026-01-12T14:30:00Z"
      }
    }
  ]
}
```

#### Get User

```http
GET /api/v1/users/{user_id}
```

#### Get Current User

```http
GET /api/v1/users/me
```

Returns the authenticated user's profile.

#### Create User

```http
POST /api/v1/users
```

**Request:**
```json
{
  "name": "Guest User",
  "role": "guest",
  "access": {
    "rooms": ["room-guest-bedroom", "room-guest-bathroom"],
    "valid_until": "2026-01-15T12:00:00Z"
  }
}
```

**Required Permission:** `users:manage`

#### Update User

```http
PATCH /api/v1/users/{user_id}
```

#### Delete User

```http
DELETE /api/v1/users/{user_id}
```

---

### Climate Zones

#### List Climate Zones

```http
GET /api/v1/climate/zones
```

**Response (200):**
```json
{
  "data": [
    {
      "id": "cz-001",
      "name": "Ground Floor",
      "rooms": ["room-living", "room-kitchen", "room-dining"],
      "state": {
        "current_temp": 21.2,
        "target_temp": 21.0,
        "mode": "heating",
        "humidity": 45
      },
      "setpoints": {
        "heating": 21.0,
        "cooling": 24.0
      }
    }
  ]
}
```

#### Get Climate Zone

```http
GET /api/v1/climate/zones/{zone_id}
```

#### Set Climate Zone

```http
PUT /api/v1/climate/zones/{zone_id}/state
```

**Request:**
```json
{
  "target_temp": 22.0,
  "mode": "auto"
}
```

#### Update Climate Zone Setpoints

```http
PATCH /api/v1/climate/zones/{zone_id}
```

**Request:**
```json
{
  "setpoints": {
    "heating": 21.5,
    "cooling": 24.5
  }
}
```

---

### Audio Zones

#### List Audio Zones

```http
GET /api/v1/audio/zones
```

**Response (200):**
```json
{
  "data": [
    {
      "id": "az-001",
      "name": "Living Room",
      "rooms": ["room-living"],
      "state": {
        "power": true,
        "source": 2,
        "source_name": "Spotify",
        "volume": 35,
        "mute": false
      },
      "now_playing": {
        "title": "Song Title",
        "artist": "Artist Name",
        "album": "Album Name"
      }
    }
  ]
}
```

#### Control Audio Zone

```http
PUT /api/v1/audio/zones/{zone_id}/state
```

**Request:**
```json
{
  "power": true,
  "source": 3,
  "volume": 40
}
```

---

### Energy

#### Current Energy Status

```http
GET /api/v1/energy/current
```

**Response (200):**
```json
{
  "data": {
    "timestamp": "2026-01-12T14:30:00Z",
    "grid": {
      "import_power_w": 2500,
      "export_power_w": 0
    },
    "solar": {
      "generation_power_w": 1200
    },
    "battery": {
      "power_w": -500,
      "soc_percent": 75,
      "state": "discharging"
    },
    "consumption": {
      "total_power_w": 3200
    },
    "ev": {
      "charging": true,
      "power_w": 7400,
      "soc_percent": 45
    }
  }
}
```

#### Energy History

```http
GET /api/v1/energy/history?period=24h&resolution=15m
GET /api/v1/energy/history?from=2026-01-01&to=2026-01-12&resolution=1d
```

**Query Parameters:**

| Parameter | Values | Default |
|-----------|--------|---------|
| `period` | `1h`, `24h`, `7d`, `30d`, `1y` | `24h` |
| `resolution` | `1m`, `5m`, `15m`, `1h`, `1d` | Auto |
| `metrics` | `consumption`, `solar`, `grid`, `battery` | All |

**Response (200):**
```json
{
  "data": {
    "from": "2026-01-11T14:30:00Z",
    "to": "2026-01-12T14:30:00Z",
    "resolution": "1h",
    "series": {
      "consumption": [
        { "timestamp": "2026-01-11T15:00:00Z", "value_wh": 1500 },
        { "timestamp": "2026-01-11T16:00:00Z", "value_wh": 1800 }
      ],
      "solar": [
        { "timestamp": "2026-01-11T15:00:00Z", "value_wh": 800 },
        { "timestamp": "2026-01-11T16:00:00Z", "value_wh": 600 }
      ]
    },
    "totals": {
      "consumption_kwh": 45.6,
      "solar_kwh": 12.3,
      "grid_import_kwh": 38.5,
      "grid_export_kwh": 5.2
    }
  }
}
```

**Required Permission:** `energy:read`

---

### Voice Commands

#### Submit Voice Command

```http
POST /api/v1/voice/command
```

**Request:**
```json
{
  "transcript": "turn on the living room lights",
  "context": {
    "room_id": "room-living",
    "user_id": "usr-001"
  }
}
```

**Response (200):**
```json
{
  "success": true,
  "intent": {
    "type": "device_control",
    "action": "turn_on",
    "targets": [
      { "device_id": "light-living-main" },
      { "device_id": "light-living-accent" }
    ]
  },
  "response_text": "Turning on the living room lights",
  "actions_executed": 2
}
```

**Response (Clarification Needed):**
```json
{
  "success": false,
  "intent": {
    "type": "ambiguous"
  },
  "response_text": "Which light? The main light or the accent light?",
  "options": [
    { "label": "Main Light", "device_id": "light-living-main" },
    { "label": "Accent Light", "device_id": "light-living-accent" }
  ]
}
```

---

### System

#### System Status

```http
GET /api/v1/system/status
```

**Response (200):**
```json
{
  "data": {
    "status": "running",
    "version": "1.0.0",
    "uptime_seconds": 604800,
    "mode": "home",
    "bridges": [
      {
        "id": "knx-bridge-01",
        "protocol": "knx",
        "status": "online",
        "devices_total": 45,
        "devices_online": 44
      },
      {
        "id": "dali-bridge-01",
        "protocol": "dali",
        "status": "online",
        "devices_total": 20,
        "devices_online": 20
      }
    ],
    "metrics": {
      "devices_total": 150,
      "devices_online": 148,
      "automations_active": 25,
      "memory_mb": 28,
      "cpu_percent": 2.5,
      "database_size_mb": 45
    },
    "time": {
      "current": "2026-01-12T14:30:00Z",
      "timezone": "Europe/London",
      "sunrise": "2026-01-12T08:01:00Z",
      "sunset": "2026-01-12T16:15:00Z"
    }
  }
}
```

#### Bridge Status

```http
GET /api/v1/system/bridges
GET /api/v1/system/bridges/{bridge_id}
```

#### System Time

```http
GET /api/v1/system/time
```

**Response (200):**
```json
{
  "data": {
    "timestamp": "2026-01-12T14:30:00Z",
    "timezone": "Europe/London",
    "offset_hours": 0,
    "sunrise": "2026-01-12T08:01:00Z",
    "sunset": "2026-01-12T16:15:00Z",
    "solar_noon": "2026-01-12T12:08:00Z"
  }
}
```

#### Trigger Backup

```http
POST /api/v1/system/backup
```

**Request:**
```json
{
  "type": "full",
  "include_timeseries": true
}
```

**Response (202):**
```json
{
  "success": true,
  "backup_id": "backup-20260112-143000",
  "message": "Backup started"
}
```

**Required Permission:** `system:configure`

#### List Backups

```http
GET /api/v1/system/backups
```

#### System Health Check

```http
GET /api/v1/system/health
```

Lightweight endpoint for monitoring:

**Response (200):**
```json
{
  "status": "healthy",
  "checks": {
    "database": "ok",
    "mqtt": "ok",
    "bridges": "ok"
  }
}
```

---

## WebSocket API

### Connection

```
ws://host:8080/api/v1/ws
wss://host:8080/api/v1/ws  (with TLS)
```

### Authentication

Include token as query parameter or in first message:

```
ws://host:8080/api/v1/ws?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

Or send auth message immediately after connection:

```json
{
  "type": "auth",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### Message Format

All WebSocket messages follow this structure:

**Client → Server:**
```json
{
  "type": "subscribe" | "unsubscribe" | "ping" | "auth",
  "id": "msg-001",
  "payload": { }
}
```

**Server → Client:**
```json
{
  "type": "event" | "response" | "error" | "pong",
  "id": "msg-001",
  "event_type": "device.state_changed",
  "timestamp": "2026-01-12T14:30:00Z",
  "payload": { }
}
```

### Subscriptions

#### Subscribe to All Events

```json
{
  "type": "subscribe",
  "id": "sub-001",
  "payload": {
    "channel": "*"
  }
}
```

#### Subscribe to Specific Channels

```json
{
  "type": "subscribe",
  "id": "sub-002",
  "payload": {
    "channels": [
      "device.state_changed",
      "scene.activated",
      "mode.changed"
    ]
  }
}
```

#### Subscribe to Room Events

```json
{
  "type": "subscribe",
  "id": "sub-003",
  "payload": {
    "channel": "device.state_changed",
    "filter": {
      "room_id": "room-living"
    }
  }
}
```

#### Subscribe to Specific Device

```json
{
  "type": "subscribe",
  "id": "sub-004",
  "payload": {
    "channel": "device.state_changed",
    "filter": {
      "device_id": "light-living-main"
    }
  }
}
```

#### Unsubscribe

```json
{
  "type": "unsubscribe",
  "id": "unsub-001",
  "payload": {
    "subscription_id": "sub-002"
  }
}
```

### Event Types

#### device.state_changed

```json
{
  "type": "event",
  "event_type": "device.state_changed",
  "timestamp": "2026-01-12T14:30:00.123Z",
  "payload": {
    "device_id": "light-living-main",
    "room_id": "room-living",
    "previous_state": {
      "on": false,
      "brightness": 0
    },
    "new_state": {
      "on": true,
      "brightness": 75
    },
    "trigger": {
      "type": "physical",
      "source": "knx"
    }
  }
}
```

#### scene.activated

```json
{
  "type": "event",
  "event_type": "scene.activated",
  "timestamp": "2026-01-12T14:30:00Z",
  "payload": {
    "scene_id": "scene-cinema",
    "scene_name": "Cinema Mode",
    "trigger": {
      "type": "voice",
      "user_id": "usr-001"
    },
    "actions_count": 5
  }
}
```

#### mode.changed

```json
{
  "type": "event",
  "event_type": "mode.changed",
  "timestamp": "2026-01-12T14:30:00Z",
  "payload": {
    "previous_mode": "home",
    "new_mode": "away",
    "trigger": {
      "type": "schedule",
      "schedule_id": "sched-workday"
    }
  }
}
```

#### schedule.triggered

```json
{
  "type": "event",
  "event_type": "schedule.triggered",
  "timestamp": "2026-01-12T06:30:00Z",
  "payload": {
    "schedule_id": "sched-morning",
    "schedule_name": "Morning Wake Up",
    "executed": {
      "type": "scene",
      "scene_id": "scene-morning"
    }
  }
}
```

#### health.alert

```json
{
  "type": "event",
  "event_type": "health.alert",
  "timestamp": "2026-01-12T14:30:00Z",
  "payload": {
    "severity": "warning",
    "device_id": "pump-circulation-01",
    "alert_type": "phm_deviation",
    "message": "Circulation pump current 15% above baseline",
    "details": {
      "metric": "current",
      "baseline": 2.3,
      "actual": 2.65,
      "deviation_percent": 15.2
    }
  }
}
```

#### bridge.status

```json
{
  "type": "event",
  "event_type": "bridge.status",
  "timestamp": "2026-01-12T14:30:00Z",
  "payload": {
    "bridge_id": "knx-bridge-01",
    "previous_status": "online",
    "new_status": "degraded",
    "reason": "High error rate detected"
  }
}
```

#### presence.changed

```json
{
  "type": "event",
  "event_type": "presence.changed",
  "timestamp": "2026-01-12T14:30:00Z",
  "payload": {
    "user_id": "usr-001",
    "user_name": "Darren",
    "is_home": true,
    "detection_method": "phone_wifi"
  }
}
```

### Heartbeat

Send periodic pings to keep connection alive:

**Client → Server:**
```json
{
  "type": "ping",
  "id": "ping-001"
}
```

**Server → Client:**
```json
{
  "type": "pong",
  "id": "ping-001",
  "timestamp": "2026-01-12T14:30:00Z"
}
```

Recommended interval: 30 seconds.

### Reconnection

On disconnect:

1. Wait 1 second
2. Attempt reconnection
3. If failed, exponential backoff (2s, 4s, 8s, max 30s)
4. On reconnect, re-authenticate and re-subscribe
5. Request missed events if needed

### Connection Limits

| Limit | Value |
|-------|-------|
| Max connections per user | 10 |
| Max subscriptions per connection | 50 |
| Message size limit | 64KB |
| Idle timeout | 5 minutes (without ping) |

---

## Rate Limiting

### Limits

| Category | Limit | Window |
|----------|-------|--------|
| Authentication | 10 requests | 1 minute |
| Device control | 100 requests | 1 minute |
| Read operations | 300 requests | 1 minute |
| Configuration changes | 30 requests | 1 minute |

### Headers

Rate limit information is included in response headers:

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1736691060
```

### Exceeded Response

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 30
```

```json
{
  "error": {
    "code": "RATE_LIMITED",
    "message": "Too many requests",
    "retry_after_seconds": 30
  }
}
```

---

## Security Considerations

### Transport Security

- **Development**: HTTP on localhost is acceptable
- **Production**: HTTPS required; HTTP redirects to HTTPS
- **Certificates**: Let's Encrypt or self-signed for local networks

### CORS Configuration

Default CORS headers for browser clients:

```http
Access-Control-Allow-Origin: https://graylogic.local
Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Max-Age: 86400
```

### Input Validation

- All inputs validated against JSON Schema
- Maximum request body: 1MB
- String length limits enforced
- SQL injection prevention via parameterized queries

### Audit Logging

Security-relevant operations are logged:

| Operation | Logged Data |
|-----------|-------------|
| Login attempt | Username, success/failure, IP |
| Mode change | User, previous mode, new mode |
| User creation | Admin user, new user details |
| Door unlock | User, door, method (remote/local) |
| Scene activation | User, scene, trigger type |
| Configuration change | User, resource, changes |

### Sensitive Actions

Actions requiring additional verification:

| Action | Requirement |
|--------|-------------|
| Unlock door remotely | Confirmation + optional PIN |
| Disarm security | PIN required |
| Delete user | Confirmation |
| Factory reset | Physical access + confirmation |

---

## Complete Examples

### Example 1: Authenticate and List Devices

```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "darren", "password": "password123"}'

# Response: { "access_token": "eyJ...", ... }

# List devices
curl http://localhost:8080/api/v1/devices \
  -H "Authorization: Bearer eyJ..."
```

### Example 2: Control a Light

```bash
# Turn on with 50% brightness and 2-second fade
curl -X PUT http://localhost:8080/api/v1/devices/light-living-main/state \
  -H "Authorization: Bearer eyJ..." \
  -H "Content-Type: application/json" \
  -d '{
    "on": true,
    "brightness": 50,
    "transition_ms": 2000
  }'
```

### Example 3: Activate a Scene

```bash
curl -X POST http://localhost:8080/api/v1/scenes/scene-cinema/activate \
  -H "Authorization: Bearer eyJ..."
```

### Example 4: WebSocket Session

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws?token=eyJ...');

ws.onopen = () => {
  // Subscribe to device state changes
  ws.send(JSON.stringify({
    type: 'subscribe',
    id: 'sub-001',
    payload: {
      channel: 'device.state_changed',
      filter: { room_id: 'room-living' }
    }
  }));
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === 'event' && msg.event_type === 'device.state_changed') {
    console.log('Device changed:', msg.payload.device_id, msg.payload.new_state);
  }
};

// Heartbeat
setInterval(() => {
  ws.send(JSON.stringify({ type: 'ping', id: Date.now().toString() }));
}, 30000);
```

### Example 5: Create a Schedule

```bash
curl -X POST http://localhost:8080/api/v1/schedules \
  -H "Authorization: Bearer eyJ..." \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Sunset Lights",
    "enabled": true,
    "trigger": {
      "type": "sunset",
      "value": "-15m"
    },
    "execute": {
      "type": "scene",
      "scene_id": "scene-evening"
    },
    "conditions": [
      {
        "type": "mode",
        "operator": "in",
        "value": ["home", "entertaining"]
      }
    ]
  }'
```

---

## Related Documents

- [System Overview](../architecture/system-overview.md) — Architecture context
- [Core Internals](../architecture/core-internals.md) — API Server implementation
- [Automation Specification](../automation/automation.md) — Scenes, schedules, modes behavior
- [Data Model: Entities](../data-model/entities.md) — Entity definitions
- [MQTT Protocol](../protocols/mqtt.md) — Internal message bus
- [Principles](../overview/principles.md) — Security and privacy principles
