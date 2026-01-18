---
title: Gray Logic JSON Schemas
version: 1.0.0
status: active
implementation_status: specified
last_updated: 2026-01-18
depends_on:
  - data-model/entities.md
---

# Gray Logic JSON Schemas

This directory contains JSON Schema definitions for all Gray Logic entities. These schemas serve as the **single source of truth** for data contracts across the system.

## Purpose

These schemas enable:

1. **Runtime validation** — Validate configuration files and API requests
2. **Documentation** — Formal specification for API consumers
3. **Code generation** — Generate Go structs, TypeScript types, Dart classes

## Schema Files

| File | Entity | Description |
|------|--------|-------------|
| `common.schema.json` | (shared) | Enums, embedded types, reusable definitions |
| `site.schema.json` | Site | A Gray Logic installation |
| `area.schema.json` | Area | Floor, building, wing, zone |
| `room.schema.json` | Room | Physical space within an area |
| `device.schema.json` | Device | Controllable/monitorable entity |
| `scene.schema.json` | Scene | Collection of device states |
| `schedule.schema.json` | Schedule | Time-based automation trigger |
| `mode.schema.json` | Mode | System-wide operational state |
| `condition.schema.json` | Condition | Automation prerequisite |
| `user.schema.json` | User | Person interacting with system |
| `device-association.schema.json` | DeviceAssociation | Links between devices |
| `audio-zone.schema.json` | AudioZone | Multi-room audio zone |
| `climate-zone.schema.json` | ClimateZone | HVAC zone |

## Schema Version

All schemas use JSON Schema draft 2020-12 and are versioned under:

```
https://graylogic.uk/schemas/v1/
```

## Usage

### Validation (JavaScript/Node.js)

```javascript
const Ajv = require('ajv/dist/2020');
const addFormats = require('ajv-formats');

const ajv = new Ajv({ allErrors: true });
addFormats(ajv);

// Load schemas
const commonSchema = require('./common.schema.json');
const siteSchema = require('./site.schema.json');

ajv.addSchema(commonSchema);
const validate = ajv.compile(siteSchema);

// Validate data
const site = {
  id: "a7b3c9d2-1234-5678-9abc-def012345678",
  name: "My Home",
  slug: "my-home",
  location: {
    latitude: 51.5074,
    longitude: -0.1278,
    timezone: "Europe/London"
  },
  modes: {
    available: ["home", "away"],
    current: "home"
  }
};

if (validate(site)) {
  console.log('Valid!');
} else {
  console.log('Errors:', validate.errors);
}
```

### Validation (Go)

```go
import (
    "github.com/santhosh-tekuri/jsonschema/v5"
)

func validateSite(data []byte) error {
    compiler := jsonschema.NewCompiler()
    compiler.Draft = jsonschema.Draft2020
    
    schema, err := compiler.Compile("site.schema.json")
    if err != nil {
        return err
    }
    
    var v interface{}
    json.Unmarshal(data, &v)
    
    return schema.Validate(v)
}
```

### Code Generation (TypeScript)

Using `json-schema-to-typescript`:

```bash
npx json-schema-to-typescript schemas/*.schema.json -o types/
```

This generates TypeScript interfaces from all schemas.

### Code Generation (Go)

Using `go-jsonschema`:

```bash
go install github.com/atombender/go-jsonschema/cmd/gojsonschema@latest

gojsonschema -p entities schemas/*.schema.json -o entities/
```

### Code Generation (Dart/Flutter)

The schemas can be used with `json_serializable` by manually creating Dart classes that match the schema structure, or using tools like `quicktype`:

```bash
npx quicktype schemas/device.schema.json -o lib/models/device.dart --lang dart
```

## Schema Structure

Each schema follows this structure:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://graylogic.uk/schemas/v1/[entity].schema.json",
  "title": "EntityName",
  "description": "Description of the entity",
  "type": "object",
  "properties": { ... },
  "required": [ ... ],
  "examples": [ ... ]
}
```

### Key Features

- **`$ref`** — References to common definitions (DRY)
- **`description`** — Every property is documented
- **`format`** — Standard formats (uuid, date-time, email)
- **`examples`** — Real-world examples for each entity
- **`enum`** — Strict enumeration of allowed values

## Entity Relationships

```
Site
 └── Area (1:N)
      └── Room (1:N)
           ├── Device (1:N)
           ├── AudioZone (0:1)
           └── ClimateZone (0:1)

Device
 ├── belongs to Room or Area
 ├── has Capabilities (1:N)
 └── has Associations (0:N)

Scene
 ├── scoped to Room, Area, or Site
 └── contains Actions (1:N)

Schedule → triggers Scene or Actions
Mode → affects automation behaviour
User → has presence tracking
```

## Enums Reference

Key enums are defined in `common.schema.json`:

### Device Types

Lighting: `light_switch`, `light_dimmer`, `light_ct`, `light_rgb`, `light_rgbw`

Climate: `thermostat`, `temperature_sensor`, `humidity_sensor`, `air_quality_sensor`, `hvac_unit`, `valve_actuator`

Blinds: `blind_switch`, `blind_position`, `blind_tilt`

Sensors: `motion_sensor`, `presence_sensor`, `door_sensor`, `window_sensor`, `leak_sensor`, `smoke_sensor`, `co_sensor`

Audio/Video: `audio_zone`, `audio_source`, `video_output`, `video_source`, `video_matrix`

Security: `alarm_panel`, `camera`, `nvr`, `door_lock`, `door_station`

Plant: `pump`, `boiler`, `heat_pump`, `chiller`, `ahu`, `fcu`, `vav_box`, `vfd`, etc.

### Domains

`lighting`, `climate`, `blinds`, `audio`, `video`, `security`, `access`, `energy`, `plant`, `irrigation`, `safety`, `sensor`

### Protocols

`knx`, `dali`, `modbus_rtu`, `modbus_tcp`, `bacnet_ip`, `bacnet_mstp`, `mqtt`, `http`, `sip`, `rtsp`, `onvif`, `ocpp`, `rs232`, `rs485`

## Related Documents

- [Entities](../entities.md) — Human-readable entity documentation
- [API Specification](../../interfaces/api.md) — REST API using these schemas
- [System Overview](../../architecture/system-overview.md) — How entities are used

## Versioning

When making breaking changes to schemas:

1. Create new version directory (`v2/`)
2. Update `$id` URLs to new version
3. Maintain backward compatibility where possible
4. Document migration path
