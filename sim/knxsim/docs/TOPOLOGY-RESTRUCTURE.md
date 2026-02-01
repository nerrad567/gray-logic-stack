# KNXSim Topology Restructure

**Status:** Proposed  
**Date:** 2026-02-01  
**Author:** Sir + Darren

## Overview

Restructure KNXSim to follow proper KNX topology patterns, separating physical topology (Area/Line/Device) from building visualization (Floor/Room).

## Current State

```
Premise
└── Floor (for UI)
    └── Room (for UI)
        └── Devices (flat list with room_id FK)
```

**Problems:**
- No concept of KNX topology (Area/Line)
- Individual addresses are just strings, not validated against topology
- Can't visualize or plan by bus structure
- Doesn't match ETS mental model

## Proposed State

```
Premise
├── TOPOLOGY VIEW (physical KNX bus structure)
│   └── Area (0-15)
│       ├── area_number: 0-15
│       ├── name: "Building A" / "Ground Floor Zone"
│       ├── line_coupler_address: "x.0.0" (optional)
│       └── Lines
│           └── Line (0-15)
│               ├── line_number: 0-15
│               ├── name: "Lighting" / "Living Area"
│               ├── line_coupler_address: "x.y.0" (optional)
│               └── Devices
│                   └── Device
│                       ├── device_number: 1-255 (0 = coupler)
│                       ├── individual_address: "area.line.device"
│                       └── ... (type, GAs, state, etc.)
│
├── BUILDING VIEW (visualization/floor plans)
│   └── Floor
│       ├── name, sort_order
│       └── Rooms
│           └── Room
│               ├── name, room_type, grid position
│               └── device_placements (links to devices)
│
└── GROUP ADDRESS VIEW (logical functions) [Phase 2]
    └── Main Group (0-31)
        ├── name: "Lighting" / "Blinds" / "HVAC"
        └── Middle Groups
            └── Middle Group (0-7)
                ├── name: "Ground Floor" / "First Floor"
                └── Sub addresses (0-255) - defined per device
```

## Database Schema Changes

### New Tables

```sql
-- Topology: Areas
CREATE TABLE areas (
    id TEXT PRIMARY KEY,
    premise_id TEXT NOT NULL REFERENCES premises(id) ON DELETE CASCADE,
    area_number INTEGER NOT NULL CHECK (area_number >= 0 AND area_number <= 15),
    name TEXT NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(premise_id, area_number)
);

-- Topology: Lines
CREATE TABLE lines (
    id TEXT PRIMARY KEY,
    area_id TEXT NOT NULL REFERENCES areas(id) ON DELETE CASCADE,
    line_number INTEGER NOT NULL CHECK (line_number >= 0 AND line_number <= 15),
    name TEXT NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(area_id, line_number)
);
```

### Modified Tables

```sql
-- Devices: Add topology references
ALTER TABLE devices ADD COLUMN line_id TEXT REFERENCES lines(id) ON DELETE SET NULL;
ALTER TABLE devices ADD COLUMN device_number INTEGER CHECK (device_number >= 0 AND device_number <= 255);

-- After migration, individual_address becomes computed/validated:
-- individual_address = f"{area.area_number}.{line.line_number}.{device.device_number}"
```

### Unchanged Tables

- `premises` - No changes needed
- `floors` - Keep for building view
- `rooms` - Keep for building view  
- `scenarios` - No changes needed

## Data Model

### Area
```python
@dataclass
class Area:
    id: str
    premise_id: str
    area_number: int      # 0-15
    name: str             # "Building A", "Main Zone"
    description: str | None
```

### Line
```python
@dataclass
class Line:
    id: str
    area_id: str
    line_number: int      # 0-15
    name: str             # "Lighting", "Sensors", "HVAC"
    description: str | None
```

### Device (modified)
```python
@dataclass
class Device:
    id: str
    premise_id: str
    line_id: str | None   # NEW - topology placement
    device_number: int    # NEW - 1-255 (0 reserved for couplers)
    room_id: str | None   # Keep - building view placement
    type: str
    individual_address: str  # Computed: "{area}.{line}.{device}"
    group_addresses: dict
    state: dict
    initial_state: dict
    config: dict | None
```

## API Changes

### New Endpoints

```
# Topology management
GET    /api/v1/premises/{id}/areas
POST   /api/v1/premises/{id}/areas
GET    /api/v1/premises/{id}/areas/{area_id}
PATCH  /api/v1/premises/{id}/areas/{area_id}
DELETE /api/v1/premises/{id}/areas/{area_id}

GET    /api/v1/premises/{id}/areas/{area_id}/lines
POST   /api/v1/premises/{id}/areas/{area_id}/lines
GET    /api/v1/premises/{id}/areas/{area_id}/lines/{line_id}
PATCH  /api/v1/premises/{id}/areas/{area_id}/lines/{line_id}
DELETE /api/v1/premises/{id}/areas/{area_id}/lines/{line_id}

# Topology view
GET    /api/v1/premises/{id}/topology
# Returns: { areas: [{ ...area, lines: [{ ...line, devices: [...] }] }] }
```

### Modified Endpoints

```
# Device creation now accepts line_id + device_number OR individual_address
POST   /api/v1/premises/{id}/devices
{
    "id": "living-room-light",
    "type": "light_dimmer",
    "line_id": "line-1-1",        # Topology placement
    "device_number": 5,            # Auto-assigns IA: 1.1.5
    // OR legacy:
    "individual_address": "1.1.5", # Parsed to find/create line
    ...
}

# Device update can change topology placement
PATCH  /api/v1/premises/{id}/devices/{device_id}
{
    "line_id": "line-1-2",         # Move to different line
    "device_number": 3             # New device number on that line
}
```

## UI Changes

### New: Topology View

Add a "Topology" tab alongside "Floor Plan":

```
[Floor Plan] [Topology] [Group Addresses]

┌─────────────────────────────────────────────────────┐
│ Area 1: Main Building                               │
│ ├── Line 1.1: Lighting (5 devices)                 │
│ │   ├── 1.1.1 living-room-ceiling-light (dimmer)   │
│ │   ├── 1.1.2 kitchen-light (switch)               │
│ │   └── [+ Add Device]                             │
│ ├── Line 1.2: Wall Controls (3 devices)            │
│ │   ├── 1.2.1 living-room-switch (push_button_2)   │
│ │   └── [+ Add Device]                             │
│ └── [+ Add Line]                                    │
│                                                     │
│ [+ Add Area]                                        │
└─────────────────────────────────────────────────────┘
```

### Modified: Device Modal

```
┌─────────────────────────────────────────────────────┐
│ Add Device                                      [×] │
├─────────────────────────────────────────────────────┤
│ Device ID: [living-room-light___________]           │
│ Type:      [Light Dimmer          ▼]                │
│                                                     │
│ ─── Topology (Physical Address) ─────────────────── │
│ Area: [1: Main Building    ▼]                       │
│ Line: [1: Lighting         ▼]                       │
│ Device #: [5] → Individual Address: 1.1.5           │
│            [Suggest Next]                           │
│                                                     │
│ ─── Building (Visualization) ────────────────────── │
│ Floor: [Ground Floor       ▼]                       │
│ Room:  [Living Room        ▼]                       │
│                                                     │
│ ─── Group Addresses ─────────────────────────────── │
│ [... existing GA editor ...]                        │
│                                                     │
├─────────────────────────────────────────────────────┤
│                              [Cancel] [Create]      │
└─────────────────────────────────────────────────────┘
```

## Migration Plan

### Phase 1: Schema + Backend (This PR)

1. Add `areas` and `lines` tables
2. Add `line_id`, `device_number` columns to `devices`
3. Migration script:
   - Parse existing `individual_address` values
   - Auto-create Areas and Lines as needed
   - Populate `line_id` and `device_number`
4. Update device CRUD to handle new fields
5. Add topology API endpoints
6. Backward compatible: `individual_address` still works

### Phase 2: UI Topology View

1. Add Topology tab to main navigation
2. Tree view for Areas → Lines → Devices
3. Drag-and-drop to move devices between lines
4. Quick device creation within a line

### Phase 3: Device Modal Update

1. Add Area/Line/Device dropdowns
2. Auto-suggest next device number
3. Show computed individual address
4. Keep Floor/Room for visualization

### Phase 4: Group Address Planning (Future)

1. Add main_groups / middle_groups tables
2. GA planning view with drag-and-drop
3. Conflict detection
4. ETS-style GA tree view

## Validation Rules

### Individual Address
- Must be unique within premise
- Area must be 0-15
- Line must be 0-15
- Device must be 1-255 (0 reserved for line coupler)
- Format: `{area}.{line}.{device}`

### Topology Constraints
- Each premise can have max 16 areas (0-15)
- Each area can have max 16 lines (0-15)
- Each line can have max 255 devices (1-255)
- Device 0 on each line reserved for line coupler

### Building View
- Device can exist without room placement (unassigned)
- Device can only be in one room at a time
- Room deletion sets device.room_id = NULL

## Open Questions

1. **Auto-create topology?** When user enters `individual_address` directly, should we auto-create the Area/Line if they don't exist?
   - **Proposed:** Yes, for ease of use. Parse IA, create Area 1/Line 1 etc. as needed.

2. **Default topology?** Should new premises start with a default Area 1, Line 1?
   - **Proposed:** Yes, create Area 1 "Default Area" with Line 1 "Default Line" on premise creation.

3. **Line naming conventions?** Should we suggest names based on common patterns?
   - Line 1: Actuators/Lighting
   - Line 2: Wall controls
   - Line 3: Sensors
   - Line 4: HVAC
   - **Proposed:** Offer as suggestions, not enforced.

4. **Migration of existing data?** Current devices have `individual_address` but no `line_id`.
   - **Proposed:** Migration script parses IA, creates topology, links devices.

## Success Criteria

- [ ] All existing tests pass
- [ ] Devices can be created via topology (Area/Line/Device#) or IA string
- [ ] Topology view shows all devices organized by Area/Line
- [ ] Building view still works (Floor/Room)
- [ ] IA uniqueness enforced at DB level
- [ ] ETS export still works (uses topology for correct addressing)
