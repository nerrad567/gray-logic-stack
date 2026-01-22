# Device Registry Package Design

> `internal/device/` — Central device catalogue for Gray Logic Core

## Purpose

Provides the Device Registry for Gray Logic — the central catalogue of all controllable and monitorable entities in an installation. It manages:
- Device CRUD operations with validation
- In-memory caching for fast lookups
- SQLite persistence via repository pattern
- Thread-safe concurrent access

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Device Registry                                 │
│                                                                          │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐   │
│  │     Registry     │    │    Repository    │    │    Validation    │   │
│  │   (registry.go)  │───▶│  (repository.go) │    │ (validation.go)  │   │
│  │                  │    │                  │    │                  │   │
│  │ • CRUD ops       │    │ • SQLite queries │    │ • Device checks  │   │
│  │ • In-memory cache│    │ • JSON marshal   │    │ • Address valid. │   │
│  │ • Thread safety  │    │ • Transactions   │    │ • Slug generation│   │
│  └──────────────────┘    └──────────────────┘    └──────────────────┘   │
│           │                       │                                      │
└───────────│───────────────────────│──────────────────────────────────────┘
            │                       │
            ▼                       ▼
┌──────────────────────┐   ┌──────────────────────┐
│    REST API (M1.4)   │   │   SQLite Database    │
│  • GET /devices      │   │   (devices table)    │
│  • POST /devices     │   └──────────────────────┘
│  • WebSocket state   │
└──────────────────────┘
```

### Key Types

| Type | Purpose |
|------|---------|
| `Device` | Core entity with identity, location, protocol, state |
| `Registry` | CRUD operations with caching |
| `Repository` | SQLite persistence interface |
| `Domain` | Functional area (lighting, climate, blinds, etc.) |
| `Protocol` | Communication protocol (KNX, DALI, Modbus, etc.) |
| `DeviceType` | Specific classification (light_dimmer, thermostat) |
| `Capability` | What a device can do (on_off, dim, temperature_read) |
| `HealthStatus` | Device health state (online, offline, degraded) |

### External Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `database/sql` | stdlib | Database operations |
| `github.com/google/uuid` | - | Device ID generation |

---

## How It Works

### Initialization

```go
// Create SQLite repository
db, _ := database.Open(cfg.Database)
repo := device.NewSQLiteRepository(db)

// Create registry with repository
registry := device.NewRegistry(repo)
registry.SetLogger(logger)

// Load devices into cache on startup
if err := registry.RefreshCache(ctx); err != nil {
    return err
}
```

### Creating a Device

```go
dev := &device.Device{
    Name:     "Living Room Dimmer",
    Type:     device.DeviceTypeLightDimmer,
    Domain:   device.DomainLighting,
    Protocol: device.ProtocolKNX,
    Address: device.Address{
        "group_address":    "1/2/3",
        "feedback_address": "1/2/4",
    },
    Capabilities: []device.Capability{
        device.CapOnOff,
        device.CapDim,
    },
    RoomID: &roomID, // Optional: assign to room
}

if err := registry.CreateDevice(ctx, dev); err != nil {
    return err
}
// dev.ID and dev.Slug are now populated
```

### Querying Devices

```go
// Get single device
dev, err := registry.GetDevice(ctx, "device-uuid")
dev, err := registry.GetDeviceBySlug(ctx, "living-room-dimmer")

// List all devices
devices, _ := registry.ListDevices(ctx)

// Filter by various criteria
byRoom, _ := registry.GetDevicesByRoom(ctx, roomID)
byDomain, _ := registry.GetDevicesByDomain(ctx, device.DomainLighting)
byProtocol, _ := registry.GetDevicesByProtocol(ctx, device.ProtocolKNX)
byCapability, _ := registry.GetDevicesByCapability(ctx, device.CapDim)
byHealth, _ := registry.GetDevicesByHealthStatus(ctx, device.HealthStatusOffline)
byGateway, _ := registry.GetDevicesByGateway(ctx, gatewayID)
```

### Updating State (High-Frequency)

```go
// Optimised for frequent updates from protocol bridges
state := device.State{"on": true, "level": 75}
registry.SetDeviceState(ctx, deviceID, state)

// Update health status
registry.SetDeviceHealth(ctx, deviceID, device.HealthStatusOnline)
```

### Deep Copy Semantics

All returned devices are deep copies to prevent cache corruption:

```go
dev, _ := registry.GetDevice(ctx, id)
dev.Name = "Modified" // Safe - doesn't affect cache

// Maps and slices are also deep copied
dev.State["on"] = false // Safe
dev.Capabilities = append(dev.Capabilities, device.CapColorRGB) // Safe
```

---

## Device Model

```go
type Device struct {
    // Identity
    ID   string `json:"id"`   // UUID
    Name string `json:"name"` // Human-readable
    Slug string `json:"slug"` // URL-safe (auto-generated)

    // Location
    RoomID *string `json:"room_id,omitempty"`
    AreaID *string `json:"area_id,omitempty"`

    // Classification
    Type   DeviceType `json:"type"`   // e.g., "light_dimmer"
    Domain Domain     `json:"domain"` // e.g., "lighting"

    // Protocol
    Protocol  Protocol `json:"protocol"` // e.g., "knx"
    Address   Address  `json:"address"`  // Protocol-specific map
    GatewayID *string  `json:"gateway_id,omitempty"`

    // Capabilities and State
    Capabilities   []Capability `json:"capabilities"`
    State          State        `json:"state"`
    StateUpdatedAt *time.Time   `json:"state_updated_at,omitempty"`

    // Health Monitoring
    HealthStatus   HealthStatus `json:"health_status"`
    HealthLastSeen *time.Time   `json:"health_last_seen,omitempty"`
    PHMEnabled     bool         `json:"phm_enabled"`

    // Metadata
    Manufacturer    *string   `json:"manufacturer,omitempty"`
    Model           *string   `json:"model,omitempty"`
    FirmwareVersion *string   `json:"firmware_version,omitempty"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}
```

### Protocol-Specific Address Examples

```go
// KNX
Address{"group_address": "1/2/3", "feedback_address": "1/2/4"}

// DALI
Address{"gateway": "dali-gw-01", "short_address": 15, "group": 0}

// Modbus TCP
Address{"host": "192.168.1.100", "port": 502, "unit_id": 1, "register": 100}
```

---

## Domains and Device Types

### Domains

| Domain | Description |
|--------|-------------|
| `lighting` | Lights, dimmers, RGB fixtures |
| `climate` | HVAC, thermostats, sensors |
| `blinds` | Blinds, curtains, shutters |
| `audio` | Audio zones, matrices |
| `video` | Video matrices, displays |
| `security` | Alarms, cameras, access |
| `energy` | Meters, EV chargers, solar |
| `sensor` | Environmental sensors |
| `plant` | Mechanical equipment |

### Device Types (Partial List)

```go
// Lighting
DeviceTypeLightSwitch  = "light_switch"
DeviceTypeLightDimmer  = "light_dimmer"
DeviceTypeLightRGB     = "light_rgb"

// Climate
DeviceTypeThermostat        = "thermostat"
DeviceTypeTemperatureSensor = "temperature_sensor"
DeviceTypeHVACUnit          = "hvac_unit"

// Security
DeviceTypeCamera     = "camera"
DeviceTypeAlarmPanel = "alarm_panel"
DeviceTypeDoorLock   = "door_lock"
```

### Capabilities

```go
// Control
CapOnOff     = "on_off"
CapDim       = "dim"
CapColorTemp = "color_temp"
CapPosition  = "position"  // Blinds

// Sensing
CapTemperatureRead = "temperature_read"
CapMotionDetect    = "motion_detect"
CapPowerRead       = "power_read"
```

---

## Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| **In-memory cache** | Fast lookups for state, REST API | Query DB every time (slow) |
| **Deep copy on read** | Prevents cache corruption from external modifications | Immutable types (complex) |
| **Repository pattern** | Testable, swappable persistence | Direct SQL in registry |
| **JSON for Address/State/Config** | Protocol-specific flexibility | Structured columns (rigid) |
| **Slug auto-generation** | URL-friendly identifiers | UUID-only URLs |
| **RWMutex for cache** | Multiple readers, single writer | Channel-based (complex) |

---

## Interactions

### Dependencies

| Package | Purpose |
|---------|---------|
| `internal/infrastructure/database` | SQLite database connection |

### Dependents

| Package | Purpose |
|---------|---------|
| `internal/bridges/knx` | Auto-register discovered KNX devices |
| `internal/api` (future) | REST API for device queries |
| `internal/automation` (future) | Scene engine device resolution |

---

## Error Handling

Domain-specific errors with context:

```go
var ErrDeviceNotFound = errors.New("device not found")
var ErrDeviceExists = errors.New("device already exists")
var ErrInvalidDevice = errors.New("invalid device")

// Check error type
if errors.Is(err, device.ErrDeviceNotFound) {
    // Handle 404
}
```

Validation errors include field information:

```go
if err := registry.CreateDevice(ctx, dev); err != nil {
    // err might be: "validation: device name is required"
    // err might be: "validation: invalid protocol 'foo'"
}
```

---

## Thread Safety

The Registry is safe for concurrent use:

| Operation | Lock Type |
|-----------|-----------|
| `GetDevice`, `ListDevices`, `GetDevicesByX` | Read lock |
| `CreateDevice`, `UpdateDevice`, `DeleteDevice` | Write lock |
| `SetDeviceState`, `SetDeviceHealth` | Write lock |
| `RefreshCache` | Write lock |

All returned devices are deep copies, so callers can safely modify them.

---

## Testing

```bash
cd code/core
go test -v ./internal/device/...

# With race detector
go test -race ./internal/device/...

# Coverage
go test -cover ./internal/device/...
```

Test scenarios:
- CRUD operations
- Cache consistency
- Deep copy isolation
- Validation rules
- Concurrent access

---

## Statistics

```go
stats := registry.GetStats()
fmt.Printf("Total devices: %d\n", stats.TotalDevices)
fmt.Printf("By domain: %v\n", stats.ByDomain)
fmt.Printf("By protocol: %v\n", stats.ByProtocol)
fmt.Printf("By health: %v\n", stats.ByHealthStatus)
```

---

## Related Documents

- [Entity Model](../../../docs/data-model/entities.md) — Device entity specification
- [Database Schema](../../migrations/20260118_200000_initial_schema.up.sql) — SQLite schema
- [KNX Bridge](./knx-bridge.md) — Device auto-registration from KNX
- [REST API](./api.md) (future) — Device endpoints
