---
title: ETS Project Import Specification
version: 1.0.0
status: draft
last_updated: 2026-01-22
target_milestone: Year 5 (Commissioning Tools)
depends_on:
  - commissioning/discovery.md
  - interfaces/api.md
  - data-model/entities.md
  - protocols/knx.md
---

# ETS Project Import Specification

This document specifies how Gray Logic imports device configurations from KNX ETS (Engineering Tool Software) project files, automating the commissioning process for KNX installations.

---

## Overview

### The Problem

Commissioning a KNX installation currently requires manual data entry:

1. Open ETS project, find group addresses
2. Manually type into `knx-bridge.yaml`
3. Manually create device records in database
4. Hope for no typos

For a typical home with 50+ devices and 200+ group addresses, this takes hours and is error-prone.

### The Solution

Import ETS project files directly:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          ETS IMPORT WORKFLOW                                 │
│                                                                              │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │
│   │   Upload    │───▶│   Parse &   │───▶│  Preview &  │───▶│   Import    │  │
│   │   .knxproj  │    │   Extract   │    │   Refine    │    │   Confirm   │  │
│   └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘  │
│                                                                              │
│   Drag & drop        Auto-detect        User reviews,      Creates devices  │
│   project file       device types       assigns rooms      & bridge config  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Design Principles

1. **Import assists, doesn't replace** — Import proposes; humans confirm
2. **Non-destructive** — Never overwrites existing devices without confirmation
3. **Reversible** — Import creates a snapshot; can roll back
4. **Offline-capable** — Works without internet connection
5. **ETS version tolerant** — Supports ETS4, ETS5, ETS6 exports

---

## Supported Formats

### Primary: `.knxproj` (ETS Native Project)

The `.knxproj` file is a ZIP archive containing XML files:

```
project.knxproj (ZIP)
├── 0.xml                    # Project metadata
├── knx_master.xml           # KNX product database (may be large)
├── P-XXXX/                  # Project folder
│   ├── project.xml          # Project structure
│   ├── 0.xml                # Building topology
│   └── GroupAddresses.xml   # ★ The goldmine - all GAs with metadata
└── Topology/
    └── ...
```

**Key file: `GroupAddresses.xml`**

```xml
<?xml version="1.0" encoding="utf-8"?>
<GroupAddresses>
  <GroupRange Name="Lighting" Address="1">
    <GroupRange Name="Ground Floor" Address="0">
      <GroupAddress Id="P-0341-0_GA-1" Address="1/0/0"
                    Name="Kitchen Light Switch"
                    DatapointType="DPST-1-1"
                    Description="Switching on/off">
        <Links>
          <Link RefId="P-0341-0_DI-1"/>  <!-- Links to device -->
        </Links>
      </GroupAddress>
      <GroupAddress Id="P-0341-0_GA-2" Address="1/0/1"
                    Name="Kitchen Light Dimming"
                    DatapointType="DPST-5-1"
                    Description="Brightness 0-100%">
      </GroupAddress>
    </GroupRange>
  </GroupRange>
</GroupAddresses>
```

### Secondary: XML/CSV Export

ETS can export group addresses as standalone files:

**XML Export:**
```xml
<GroupAddressExport>
  <GroupAddress Address="1/0/0" Name="Kitchen Light Switch" DPT="1.001"/>
  <GroupAddress Address="1/0/1" Name="Kitchen Light Dimming" DPT="5.001"/>
</GroupAddressExport>
```

**CSV Export:**
```csv
"Main","Middle","Sub","Address","Central","Unfiltered","Name","Description","DatapointType"
"1","0","0","1/0/0","","","Kitchen Light Switch","On/Off","DPST-1-1"
"1","0","1","1/0/1","","","Kitchen Light Dimming","0-100%","DPST-5-1"
```

### Format Detection

The importer auto-detects format:

| Format | Detection | Confidence |
|--------|-----------|------------|
| `.knxproj` | ZIP with `GroupAddresses.xml` | High |
| XML | Root element `<GroupAddressExport>` or `<GroupAddresses>` | High |
| CSV | Header contains "Address" and "DatapointType" | Medium |

---

## Data Extraction

### Extracted Fields

From each group address, extract:

| Field | Source | Example |
|-------|--------|---------|
| `address` | `@Address` attribute | `1/0/0` |
| `name` | `@Name` attribute | `Kitchen Light Switch` |
| `dpt` | `@DatapointType` | `DPST-1-1` → `1.001` |
| `description` | `@Description` | `Switching on/off` |
| `location` | Parent `<GroupRange>` names | `Lighting > Ground Floor` |
| `linked_devices` | `<Link>` elements | Device references |

### DPT Normalisation

ETS uses `DPST-X-Y` format; Gray Logic uses `X.YYY`:

```
DPST-1-1   → 1.001   (Switch)
DPST-1-8   → 1.008   (Up/Down)
DPST-5-1   → 5.001   (Percentage 0-100%)
DPST-5-4   → 5.004   (Percentage 0-255)
DPST-9-1   → 9.001   (Temperature °C)
DPST-9-4   → 9.004   (Illuminance lux)
DPST-14-68 → 14.068  (Power W)
```

### Location Inference

Use ETS hierarchy to suggest room assignments:

```
ETS Structure                        Gray Logic Mapping
─────────────                        ──────────────────
Lighting/                            domain: lighting
  Ground Floor/                      area: ground-floor
    Kitchen/                         room: kitchen
      Kitchen Light Switch           device: kitchen-light
```

---

## Device Type Detection

### Heuristic Rules

The importer groups related GAs into devices using pattern matching:

```yaml
# Device detection rules (priority order)
rules:
  - name: dimmer
    requires:
      - dpt: "1.001"          # Switch
        function: switch
      - dpt: "5.001"          # Brightness
        function: brightness
    optional:
      - dpt: "1.001"          # Status feedback
        function: switch_status
        name_contains: ["status", "feedback", "rückmeldung"]

  - name: blind
    requires:
      - dpt: "1.008"          # Up/Down
        function: move
      - dpt: "5.001"          # Position
        function: position
    optional:
      - dpt: "1.007"          # Stop
        function: stop
      - dpt: "5.001"          # Tilt (slat angle)
        function: tilt
        name_contains: ["slat", "lamelle", "tilt"]

  - name: switch
    requires:
      - dpt: "1.001"
        function: switch
    max_addresses: 2          # Avoid matching dimmers

  - name: temperature_sensor
    requires:
      - dpt: "9.001"
        function: temperature

  - name: motion_sensor
    requires:
      - dpt: "1.001"
        function: motion
        name_contains: ["motion", "presence", "bewegung", "präsenz"]
```

### Grouping Strategy

GAs are grouped into devices by:

1. **Name prefix matching**: "Kitchen Light Switch" + "Kitchen Light Dimming" → same device
2. **Address proximity**: `1/0/0` and `1/0/1` likely same device
3. **ETS links**: If GAs link to same physical device
4. **Manual override**: User can drag/drop to regroup

### Confidence Scoring

Each detected device gets a confidence score:

| Confidence | Meaning | UI Treatment |
|------------|---------|--------------|
| `high` (>80%) | Strong pattern match | Auto-selected for import |
| `medium` (50-80%) | Partial match | Selected but flagged for review |
| `low` (<50%) | Uncertain | Not selected; user must confirm |

---

## Import Preview UI

### Wireframe: Upload Screen

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  ← Back                    Import ETS Project                         Help  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │                                                                          ││
│  │                     ┌──────────────────────┐                            ││
│  │                     │   📁                  │                            ││
│  │                     │                       │                            ││
│  │                     │  Drop .knxproj file   │                            ││
│  │                     │  or click to browse   │                            ││
│  │                     │                       │                            ││
│  │                     └──────────────────────┘                            ││
│  │                                                                          ││
│  │                  Supports: .knxproj, .xml, .csv                         ││
│  │                                                                          ││
│  └─────────────────────────────────────────────────────────────────────────┘│
│                                                                              │
│  Recent Imports:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │ 📄 SmithResidence.knxproj    2026-01-15    47 devices    [Re-import]   ││
│  │ 📄 TestProject.knxproj       2026-01-10    12 devices    [Re-import]   ││
│  └─────────────────────────────────────────────────────────────────────────┘│
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Wireframe: Preview & Refine Screen

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  ← Back              Review Detected Devices                   [Import All] │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Found: 47 devices, 156 group addresses                    Filter: [All ▼]  │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │ ☑ ✓ Kitchen Light                Dimmer      Kitchen     [Edit] [Skip] ││
│  │     ├─ 1/0/0  Switch (DPT 1.001)                                        ││
│  │     ├─ 1/0/1  Brightness (DPT 5.001)                                    ││
│  │     └─ 1/0/2  Status (DPT 1.001)                                        ││
│  ├─────────────────────────────────────────────────────────────────────────┤│
│  │ ☑ ✓ Kitchen Blind                Blind       Kitchen     [Edit] [Skip] ││
│  │     ├─ 1/1/0  Move (DPT 1.008)                                          ││
│  │     ├─ 1/1/1  Position (DPT 5.001)                                      ││
│  │     └─ 1/1/2  Stop (DPT 1.007)                                          ││
│  ├─────────────────────────────────────────────────────────────────────────┤│
│  │ ☑ ⚠ Temperature Sensor 1         Sensor      ???         [Edit] [Skip] ││
│  │     └─ 3/0/0  Temperature (DPT 9.001)                                   ││
│  │     ⚠ Room could not be determined                                      ││
│  ├─────────────────────────────────────────────────────────────────────────┤│
│  │ ☐ ? Unknown Device               ???         ???         [Edit] [Skip] ││
│  │     └─ 9/9/0  Unknown (DPT 14.068)                                      ││
│  │     ? Low confidence - please review                                    ││
│  └─────────────────────────────────────────────────────────────────────────┘│
│                                                                              │
│  Legend: ✓ High confidence  ⚠ Needs review  ? Low confidence               │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────── ┐│
│  │ Summary: 42 ready │ 3 need review │ 2 skipped │ 156 group addresses    ││
│  └──────────────────────────────────────────────────────────────────────────┘│
│                                                                              │
│                                              [Cancel]  [Import 45 Devices]  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Wireframe: Edit Device Modal

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Edit Device                              [×]      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Name:        [Kitchen Light________________________]                       │
│  ID:          [kitchen-light________________________]  (auto-generated)     │
│                                                                              │
│  Type:        [Dimmer ▼]                                                    │
│  Domain:      [Lighting ▼]                                                  │
│  Room:        [Kitchen ▼]          [+ Create Room]                          │
│                                                                              │
│  ─────────────────────────────────────────────────────────────────────────  │
│  Group Addresses:                                                            │
│                                                                              │
│  │ Function     │ Address │ DPT   │ Flags           │                       │
│  ├──────────────┼─────────┼───────┼─────────────────┤                       │
│  │ switch       │ 1/0/0   │ 1.001 │ [☑W] [☑R] [☐T] │                       │
│  │ brightness   │ 1/0/1   │ 5.001 │ [☑W] [☑R] [☐T] │                       │
│  │ switch_status│ 1/0/2   │ 1.001 │ [☐W] [☑R] [☑T] │                       │
│  │ [+ Add GA]   │         │       │                 │                       │
│                                                                              │
│  Flags: W=Write, R=Read, T=Transmit                                         │
│                                                                              │
│                                              [Cancel]  [Save Changes]       │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## API Specification

### Upload & Parse

```http
POST /api/v1/commissioning/ets/parse
Content-Type: multipart/form-data

file: <binary data>
```

**Response:**

```json
{
  "import_id": "imp_abc123",
  "source_file": "SmithResidence.knxproj",
  "ets_version": "ETS5",
  "statistics": {
    "total_group_addresses": 156,
    "detected_devices": 47,
    "high_confidence": 42,
    "medium_confidence": 3,
    "low_confidence": 2,
    "unmapped_addresses": 12
  },
  "devices": [
    {
      "suggested_id": "kitchen-light",
      "suggested_name": "Kitchen Light",
      "detected_type": "dimmer",
      "confidence": 0.95,
      "suggested_domain": "lighting",
      "suggested_room": "kitchen",
      "addresses": [
        {
          "ga": "1/0/0",
          "name": "Kitchen Light Switch",
          "dpt": "1.001",
          "suggested_function": "switch",
          "suggested_flags": ["write", "read"]
        },
        {
          "ga": "1/0/1",
          "name": "Kitchen Light Dimming",
          "dpt": "5.001",
          "suggested_function": "brightness",
          "suggested_flags": ["write", "read"]
        }
      ]
    }
  ],
  "unmapped_addresses": [
    {
      "ga": "9/9/0",
      "name": "Unknown Function",
      "dpt": "14.068",
      "reason": "No matching device pattern"
    }
  ],
  "warnings": [
    {
      "code": "ROOM_NOT_FOUND",
      "message": "Room 'Utility' does not exist. Will be created on import.",
      "affected_devices": ["utility-sensor"]
    }
  ]
}
```

### Confirm Import

```http
POST /api/v1/commissioning/ets/import
Content-Type: application/json

{
  "import_id": "imp_abc123",
  "devices": [
    {
      "suggested_id": "kitchen-light",
      "id": "kitchen-light",           // User can override
      "name": "Kitchen Ceiling Light", // User can override
      "type": "dimmer",
      "domain": "lighting",
      "room_id": "room_kitchen",
      "addresses": [
        {
          "ga": "1/0/0",
          "function": "switch",
          "dpt": "1.001",
          "flags": ["write", "read"]
        },
        {
          "ga": "1/0/1",
          "function": "brightness",
          "dpt": "5.001",
          "flags": ["write", "read"]
        }
      ],
      "import": true                    // false to skip
    }
  ],
  "create_rooms": true,                 // Auto-create missing rooms
  "create_areas": true,                 // Auto-create missing areas
  "backup_before_import": true          // Create snapshot first
}
```

**Response:**

```json
{
  "success": true,
  "import_id": "imp_abc123",
  "backup_id": "backup_20260122_143052",
  "results": {
    "devices_created": 45,
    "devices_updated": 0,
    "devices_skipped": 2,
    "rooms_created": 8,
    "areas_created": 2,
    "bridge_config_updated": true
  },
  "next_steps": [
    "Restart KNX bridge to load new configuration",
    "Test device communication"
  ]
}
```

### Rollback Import

```http
POST /api/v1/commissioning/ets/rollback
Content-Type: application/json

{
  "import_id": "imp_abc123"
}
```

---

## Output Files

### Generated: `knx-bridge.yaml`

The import appends to (or creates) the bridge configuration:

```yaml
# Auto-generated by ETS Import on 2026-01-22
# Source: SmithResidence.knxproj
# Import ID: imp_abc123

devices:
  - id: "kitchen-light"
    name: "Kitchen Ceiling Light"
    type: "dimmer"
    addresses:
      switch:
        ga: "1/0/0"
        dpt: "1.001"
        flags: ["write", "read"]
      brightness:
        ga: "1/0/1"
        dpt: "5.001"
        flags: ["write", "read"]
      switch_status:
        ga: "1/0/2"
        dpt: "1.001"
        flags: ["read", "transmit"]

  - id: "kitchen-blind"
    name: "Kitchen Blind"
    type: "blind"
    addresses:
      move:
        ga: "1/1/0"
        dpt: "1.008"
        flags: ["write"]
      position:
        ga: "1/1/1"
        dpt: "5.001"
        flags: ["write", "read"]
      stop:
        ga: "1/1/2"
        dpt: "1.007"
        flags: ["write"]
```

### Generated: Database Records

Devices are inserted into SQLite:

```sql
INSERT INTO devices (id, name, slug, domain, type, protocol, room_id, capabilities, protocol_config)
VALUES (
  'kitchen-light',
  'Kitchen Ceiling Light',
  'kitchen-ceiling-light',
  'lighting',
  'dimmer',
  'knx',
  'room_kitchen',
  '["on_off", "brightness"]',
  '{"import_id": "imp_abc123", "ets_source": "SmithResidence.knxproj"}'
);
```

---

## Conflict Handling

### Existing Device Detection

Before import, check for conflicts:

| Conflict Type | Detection | Resolution Options |
|---------------|-----------|-------------------|
| Same ID exists | `device.id` match | Skip / Replace / Merge / Rename |
| Same GA exists | `address.ga` match | Skip / Replace / Reassign |
| Similar name | Levenshtein distance < 3 | Suggest merge |

### Conflict Resolution UI

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  ⚠ Conflict Detected                                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Device "kitchen-light" already exists.                                      │
│                                                                              │
│  Existing:                        Importing:                                 │
│  ─────────                        ──────────                                 │
│  Name: Kitchen Light              Name: Kitchen Ceiling Light                │
│  GAs: 1/0/0, 1/0/1               GAs: 1/0/0, 1/0/1, 1/0/2                   │
│  Room: Kitchen                    Room: Kitchen                              │
│                                                                              │
│  What would you like to do?                                                  │
│                                                                              │
│  ○ Skip - Keep existing device, don't import                                │
│  ○ Replace - Delete existing, create new                                    │
│  ● Merge - Add new GAs (1/0/2) to existing device                           │
│  ○ Rename - Import as "kitchen-light-2"                                     │
│                                                                              │
│  ☑ Apply to all similar conflicts                                           │
│                                                                              │
│                                              [Cancel]  [Continue]           │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Error Handling

### Parse Errors

| Error Code | Meaning | User Message |
|------------|---------|--------------|
| `INVALID_FILE` | Not a valid ETS export | "File doesn't appear to be an ETS project or export" |
| `UNSUPPORTED_VERSION` | ETS version too old/new | "ETS version X not supported. Supported: ETS4-ETS6" |
| `CORRUPT_ARCHIVE` | ZIP extraction failed | "Project file appears to be corrupted" |
| `NO_GROUP_ADDRESSES` | Empty project | "No group addresses found in project" |
| `ENCODING_ERROR` | Character encoding issue | "File contains invalid characters" |

### Import Errors

| Error Code | Meaning | Recovery |
|------------|---------|----------|
| `DB_WRITE_FAILED` | Database error | Automatic rollback |
| `CONFIG_WRITE_FAILED` | Can't write YAML | Automatic rollback |
| `VALIDATION_FAILED` | Invalid device data | Show validation errors |

---

## Security Considerations

### File Upload Security

1. **Size limit**: Max 50MB (typical ETS project is 1-10MB)
2. **Type validation**: Verify ZIP structure before extracting
3. **Path traversal**: Sanitise all paths from ZIP
4. **Resource limits**: Timeout parsing after 60 seconds
5. **Temp file cleanup**: Delete uploaded files after processing

### Data Privacy

ETS projects may contain:
- Building addresses (strip if present)
- Installer contact info (strip if present)
- Device serial numbers (keep for reference)

---

## Implementation Phases

### Phase 1: Basic Import (MVP)

- [ ] Parse `.knxproj` ZIP files
- [ ] Extract group addresses with DPT
- [ ] Basic device detection (dimmer, switch, blind)
- [ ] Simple preview UI (table view)
- [ ] Generate `knx-bridge.yaml`
- [ ] Create device records

### Phase 2: Smart Detection

- [ ] Advanced pattern matching
- [ ] Confidence scoring
- [ ] Room inference from ETS hierarchy
- [ ] Drag-drop device grouping in UI
- [ ] Conflict detection and resolution

### Phase 3: Full Integration

- [ ] XML/CSV export support
- [ ] Re-import detection (update existing)
- [ ] Backup/rollback functionality
- [ ] Import history with diff view
- [ ] Bulk edit in preview

---

## Testing Strategy

### Test Data

Create test ETS projects covering:

1. **Simple home**: 10 devices, clean naming
2. **Complex home**: 100+ devices, mixed languages
3. **Edge cases**: Missing DPTs, duplicate names, deep hierarchies
4. **Real projects**: Anonymised exports from actual installations

### Validation Tests

| Test | Input | Expected Output |
|------|-------|-----------------|
| Dimmer detection | GA with 1.001 + 5.001 | Detects dimmer, high confidence |
| Blind detection | GA with 1.008 + 5.001 | Detects blind, high confidence |
| Unknown DPT | GA with 14.068 | Marked as unknown, low confidence |
| Duplicate GA | Same GA in two devices | Conflict warning |
| German names | "Küche Licht Schalten" | Correct detection + room |

---

## Related Documents

- [Device Discovery](discovery.md) — Runtime device discovery
- [KNX Protocol](../protocols/knx.md) — KNX protocol details
- [Data Model](../data-model/entities.md) — Device entity schema
- [REST API](../interfaces/api.md) — API conventions
