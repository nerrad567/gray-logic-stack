---
title: Core Internals Architecture
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/system-overview.md
  - overview/principles.md
---

# Core Internals Architecture

This document describes the internal architecture of Gray Logic Core — the central Go binary that manages all automation, device state, and user interfaces.

---

## Overview

Gray Logic Core is a single Go binary (~30MB compiled) that runs on the on-site server. It contains all automation logic and provides APIs for user interfaces and protocol bridges.

### Design Goals

1. **Single binary** — No external runtime dependencies
2. **Low resource usage** — Target 30-50MB RAM for typical home
3. **Fast startup** — Ready in <5 seconds
4. **Resilient** — Survive component failures gracefully
5. **Observable** — Comprehensive logging and metrics
6. **Testable** — Unit testable without hardware

---

## Layer Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           GRAY LOGIC CORE                                        │
│                                                                                  │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │                        INTELLIGENCE LAYER                                  │  │
│  │                                                                            │  │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │  │
│  │   │  AI Engine  │  │ Voice / NLU │  │  Presence   │  │  Learning   │      │  │
│  │   │             │  │             │  │  Engine     │  │  Engine     │      │  │
│  │   └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘      │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                      │                                           │
│                                      ▼                                           │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │                        AUTOMATION LAYER                                    │  │
│  │                                                                            │  │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │  │
│  │   │   Scene     │  │  Scheduler  │  │    Mode     │  │   Event     │      │  │
│  │   │   Engine    │  │             │  │   Manager   │  │   Router    │      │  │
│  │   └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘      │  │
│  │                                                                            │  │
│  │   ┌─────────────────────────────────────────────────────────────────┐     │  │
│  │   │                    Conditional Logic Engine                      │     │  │
│  │   └─────────────────────────────────────────────────────────────────┘     │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                      │                                           │
│                                      ▼                                           │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │                          DEVICE LAYER                                      │  │
│  │                                                                            │  │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │  │
│  │   │   Device    │  │    State    │  │   Command   │  │  Discovery  │      │  │
│  │   │  Registry   │  │   Manager   │  │  Processor  │  │   Service   │      │  │
│  │   └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘      │  │
│  │                                                                            │  │
│  │   ┌─────────────────────────────────────────────────────────────────┐     │  │
│  │   │                   Health Monitor (PHM)                           │     │  │
│  │   └─────────────────────────────────────────────────────────────────┘     │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                      │                                           │
│                                      ▼                                           │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │                       INFRASTRUCTURE LAYER                                 │  │
│  │                                                                            │  │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │  │
│  │   │     API     │  │  WebSocket  │  │  Database   │  │   Message   │      │  │
│  │   │   Server    │  │   Server    │  │  (SQLite)   │  │ Bus (MQTT)  │      │  │
│  │   └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘      │  │
│  │                                                                            │  │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │  │
│  │   │  Security   │  │   Config    │  │   Logging   │  │   Metrics   │      │  │
│  │   │   & Auth    │  │   Manager   │  │             │  │             │      │  │
│  │   └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘      │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## Package Structure

```
gray-logic-core/
├── cmd/
│   └── graylogic/
│       └── main.go                 # Entry point
│
├── internal/
│   ├── infrastructure/             # Infrastructure Layer
│   │   ├── api/                    # REST API server
│   │   │   ├── server.go
│   │   │   ├── routes.go
│   │   │   ├── middleware/
│   │   │   └── handlers/
│   │   ├── websocket/              # WebSocket server
│   │   │   ├── server.go
│   │   │   ├── hub.go
│   │   │   └── client.go
│   │   ├── database/               # SQLite database
│   │   │   ├── database.go
│   │   │   ├── migrations/
│   │   │   └── queries/
│   │   ├── mqtt/                   # MQTT client
│   │   │   ├── client.go
│   │   │   └── topics.go
│   │   ├── config/                 # Configuration management
│   │   │   ├── config.go
│   │   │   └── loader.go
│   │   ├── auth/                   # Authentication & authorization
│   │   │   ├── auth.go
│   │   │   ├── jwt.go
│   │   │   └── rbac.go
│   │   ├── logging/                # Structured logging
│   │   │   └── logger.go
│   │   └── metrics/                # Prometheus metrics
│   │       └── metrics.go
│   │
│   ├── device/                     # Device Layer
│   │   ├── registry/               # Device registry
│   │   │   ├── registry.go
│   │   │   └── device.go
│   │   ├── state/                  # State management
│   │   │   ├── manager.go
│   │   │   ├── store.go
│   │   │   └── history.go
│   │   ├── command/                # Command processing
│   │   │   ├── processor.go
│   │   │   ├── validator.go
│   │   │   └── router.go
│   │   ├── discovery/              # Device discovery
│   │   │   └── discovery.go
│   │   └── health/                 # PHM health monitoring
│   │       ├── monitor.go
│   │       ├── baseline.go
│   │       └── alerter.go
│   │
│   ├── automation/                 # Automation Layer
│   │   ├── scene/                  # Scene engine
│   │   │   ├── engine.go
│   │   │   ├── scene.go
│   │   │   └── action.go
│   │   ├── scheduler/              # Time-based triggers
│   │   │   ├── scheduler.go
│   │   │   ├── schedule.go
│   │   │   └── astronomical.go
│   │   ├── mode/                   # Mode management
│   │   │   ├── manager.go
│   │   │   └── mode.go
│   │   ├── event/                  # Event routing
│   │   │   ├── router.go
│   │   │   ├── event.go
│   │   │   └── handlers.go
│   │   └── logic/                  # Conditional logic
│   │       ├── engine.go
│   │       ├── condition.go
│   │       └── evaluator.go
│   │
│   ├── intelligence/               # Intelligence Layer
│   │   ├── ai/                     # AI engine
│   │   │   ├── engine.go
│   │   │   └── llm.go
│   │   ├── voice/                  # Voice processing
│   │   │   ├── processor.go
│   │   │   ├── stt.go              # Speech-to-text
│   │   │   ├── nlu.go              # Natural language understanding
│   │   │   └── tts.go              # Text-to-speech
│   │   ├── presence/               # Presence detection
│   │   │   ├── engine.go
│   │   │   └── tracker.go
│   │   └── learning/               # Pattern learning
│   │       ├── engine.go
│   │       └── patterns.go
│   │
│   └── domain/                     # Domain-specific logic
│       ├── lighting/
│       │   ├── service.go
│       │   └── circadian.go
│       ├── climate/
│       │   ├── service.go
│       │   ├── zone.go
│       │   └── adaptive.go
│       ├── blinds/
│       │   ├── service.go
│       │   └── suntracking.go
│       ├── audio/
│       │   └── service.go
│       ├── security/
│       │   └── service.go
│       └── energy/
│           ├── service.go
│           └── flows.go
│
├── pkg/                            # Public packages (reusable)
│   ├── models/                     # Data models
│   │   ├── site.go
│   │   ├── device.go
│   │   ├── scene.go
│   │   └── ...
│   ├── protocol/                   # Protocol definitions
│   │   ├── knx/
│   │   ├── dali/
│   │   └── modbus/
│   └── mqtt/                       # MQTT message types
│       ├── messages.go
│       └── topics.go
│
├── migrations/                     # Database migrations
│   ├── 001_initial.sql
│   └── ...
│
├── configs/                        # Default configurations
│   ├── config.yaml
│   └── ...
│
└── web/                            # Embedded web assets (optional)
    └── admin/
```

---

## Component Specifications

### Infrastructure Layer

#### API Server

HTTP REST API for user interfaces and external integrations.

```go
// internal/infrastructure/api/server.go

type Server struct {
    router      *chi.Mux
    auth        *auth.Service
    stateStore  *state.Manager
    cmdProc     *command.Processor
    sceneEngine *scene.Engine
}

// Endpoints
// GET    /api/v1/sites
// GET    /api/v1/sites/{id}
// GET    /api/v1/devices
// GET    /api/v1/devices/{id}
// POST   /api/v1/devices/{id}/command
// GET    /api/v1/scenes
// POST   /api/v1/scenes/{id}/activate
// GET    /api/v1/state
// ...
```

> **See Also:** [REST and WebSocket API Specification](../interfaces/api.md) for complete endpoint documentation, authentication details, request/response schemas, and WebSocket event types.

**Configuration:**
```yaml
api:
  listen_address: "0.0.0.0:8080"
  tls:
    enabled: true
    cert_file: "/etc/graylogic/server.crt"
    key_file: "/etc/graylogic/server.key"
  cors:
    allowed_origins: ["https://admin.local"]
  rate_limit:
    requests_per_second: 100
    burst: 50
```

#### WebSocket Server

Real-time state updates to connected clients.

```go
// internal/infrastructure/websocket/hub.go

type Hub struct {
    clients    map[*Client]bool
    broadcast  chan StateUpdate
    register   chan *Client
    unregister chan *Client
}

type StateUpdate struct {
    DeviceID  string    `json:"device_id"`
    State     any       `json:"state"`
    Timestamp time.Time `json:"timestamp"`
}
```

**Message Types:**
```yaml
# Client → Server
- type: "subscribe"
  topics: ["device/*", "scene/*"]
  
- type: "command"
  device_id: "light-1"
  command: "dim"
  parameters: { level: 50 }

# Server → Client
- type: "state_update"
  device_id: "light-1"
  state: { on: true, level: 50 }
  
- type: "event"
  event_type: "scene_activated"
  scene_id: "scene-cinema"
```

#### Database (SQLite)

Embedded database for configuration and state persistence.

```go
// internal/infrastructure/database/database.go

type Database struct {
    db *sql.DB
}

// Core tables:
// - sites
// - areas
// - rooms
// - devices
// - device_state
// - device_state_history
// - scenes
// - schedules
// - modes
// - users
// - audit_log
```

**Schema excerpt:**
```sql
CREATE TABLE devices (
    id TEXT PRIMARY KEY,
    room_id TEXT REFERENCES rooms(id),
    area_id TEXT REFERENCES areas(id),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    domain TEXT NOT NULL,
    protocol TEXT NOT NULL,
    address JSON NOT NULL,
    capabilities JSON NOT NULL,
    config JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE device_state (
    device_id TEXT PRIMARY KEY REFERENCES devices(id),
    state JSON NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE device_state_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT REFERENCES devices(id),
    state JSON NOT NULL,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_history_device_time ON device_state_history(device_id, recorded_at);
```

#### MQTT Client

Internal message bus connection.

```go
// internal/infrastructure/mqtt/client.go

type Client struct {
    client      mqtt.Client
    handlers    map[string]MessageHandler
    stateStore  *state.Manager
}

type MessageHandler func(topic string, payload []byte) error

// Subscribe patterns:
// graylogic/state/+/+     - All device states
// graylogic/health/+      - Bridge health
// graylogic/discovery/+   - Device discovery
```

---

### Device Layer

#### Device Registry

Central catalog of all devices.

```go
// internal/device/registry/registry.go

type Registry struct {
    db      *database.Database
    devices map[string]*Device
    mu      sync.RWMutex
}

type Device struct {
    ID           string
    RoomID       string
    AreaID       string
    Name         string
    Type         DeviceType
    Domain       Domain
    Protocol     Protocol
    Address      map[string]any
    Capabilities []Capability
    Config       map[string]any
}

func (r *Registry) GetDevice(id string) (*Device, error)
func (r *Registry) GetDevicesByRoom(roomID string) ([]*Device, error)
func (r *Registry) GetDevicesByDomain(domain Domain) ([]*Device, error)
func (r *Registry) GetDevicesByProtocol(protocol Protocol) ([]*Device, error)
```

#### State Manager

Device state storage and notification.

```go
// internal/device/state/manager.go

type Manager struct {
    db         *database.Database
    cache      map[string]DeviceState
    mu         sync.RWMutex
    listeners  []StateChangeListener
    wsHub      *websocket.Hub
}

type DeviceState struct {
    DeviceID  string
    State     map[string]any
    UpdatedAt time.Time
}

type StateChangeListener func(deviceID string, oldState, newState map[string]any)

func (m *Manager) GetState(deviceID string) (DeviceState, error)
func (m *Manager) SetState(deviceID string, state map[string]any) error
func (m *Manager) Subscribe(listener StateChangeListener)
```

**State Change Flow:**
```
MQTT message received (graylogic/state/knx/1.2.3)
    │
    ▼
State Manager receives state update
    │
    ├──► Update in-memory cache
    ├──► Persist to SQLite
    ├──► Notify all StateChangeListeners
    │        ├── Event Router (automation triggers)
    │        ├── PHM Monitor (health analysis)
    │        └── Domain Services (domain logic)
    │
    └──► Broadcast via WebSocket to UIs
```

#### Command Processor

Validates and routes commands.

```go
// internal/device/command/processor.go

type Processor struct {
    registry    *registry.Registry
    mqtt        *mqtt.Client
    validator   *Validator
}

type Command struct {
    DeviceID   string
    Command    string
    Parameters map[string]any
    Source     CommandSource  // api, automation, voice
    UserID     string
}

func (p *Processor) Execute(cmd Command) error {
    // 1. Validate device exists
    // 2. Validate command is supported
    // 3. Validate parameters
    // 4. Check authorization
    // 5. Check for control associations (route via proxy if needed)
    // 6. Route to appropriate bridge via MQTT
    // 7. Log command execution
}
```

#### Association Resolver

Handles device associations for external monitoring and control proxying.

```go
// internal/device/association/resolver.go

type Resolver struct {
    db           *database.Database
    associations map[string][]*Association  // indexed by source and target
    mu           sync.RWMutex
}

type Association struct {
    ID             string
    SourceDeviceID string
    TargetDeviceID string
    TargetGroupID  string
    Type           AssociationType  // monitors, controls, monitors_and_controls
    Config         AssociationConfig
}

type AssociationType string

const (
    AssocMonitors            AssociationType = "monitors"
    AssocControls            AssociationType = "controls"
    AssocMonitorsAndControls AssociationType = "monitors_and_controls"
)

type AssociationConfig struct {
    Metrics         []string  // For monitoring: which metrics to attribute
    ControlFunction string    // For control: power, enable, speed
    SourceProperty  string    // Property name on source device
    TargetProperty  string    // Property name to set on target
    Scale           float64   // Multiply source value
    Offset          float64   // Add to source value
}

// Monitoring: Get associations where this device is the source (sensor)
func (r *Resolver) GetMonitoringTargets(sourceDeviceID string) []*Association

// Control: Get associations where this device is the target (equipment)
func (r *Resolver) GetControlProxy(targetDeviceID string) *Association

// Check if device has any associations
func (r *Resolver) HasAssociations(deviceID string) bool
```

**Monitoring Association Flow (Data Attribution):**

When a CT clamp reports power, the reading is attributed to the target device:

```
CT Clamp reports: { power: 5.2 kW }
        │
        ▼
State Manager receives state for "ct-clamp-1"
        │
        ▼
Association Resolver.GetMonitoringTargets("ct-clamp-1")
        │
        ▼
Returns: [{ target: "pump-chw-1", metrics: ["power_kw"] }]
        │
        ▼
State Manager ALSO updates "pump-chw-1":
   - state.power_kw = 5.2  (attributed from CT clamp)
   - state.power_source = "ct-clamp-1"
        │
        ├──► PHM sees pump-chw-1 power = 5.2 kW (for health monitoring)
        ├──► Energy sees pump-chw-1 consumption (for attribution)
        └──► UI shows pump power consumption
```

**Control Association Flow (Command Routing):**

When user commands the pump, the command is routed to the relay:

```
API request: POST /devices/pump-chw-1/commands { command: "power_on" }
        │
        ▼
Command Processor receives command for "pump-chw-1"
        │
        ▼
Association Resolver.GetControlProxy("pump-chw-1")
        │
        ▼
Returns: { source: "relay-module-1-ch3", control_function: "power" }
        │
        ▼
Command Processor routes to relay instead:
   - MQTT: graylogic/command/relay-module-1-ch3 { command: "on" }
        │
        ▼
Relay closes → Pump receives power → Pump turns on
```

**Combined Association (monitors_and_controls):**

For devices that both monitor AND control (e.g., smart plugs, VFDs with feedback):

```yaml
Association:
  source: "smart-plug-garage"
  target: "heater-garage"
  type: "monitors_and_controls"
  config:
    metrics: ["power_kw", "energy_kwh"]     # Monitoring attributes
    control_function: "power"                # Control function
```

**Behavior for `monitors_and_controls`:**

1. **Monitoring Flow (Source → Target):**
   - When smart plug reports `power_kw: 1.5` → attributed to heater
   - Heater state updated: `{ power_kw: 1.5, power_source: "smart-plug-garage" }`
   - PHM and Energy see heater power consumption

2. **Control Flow (Target → Source):**
   - When user sends `POST /devices/heater-garage/command { command: "power_on" }`
   - Resolver finds control association → routes to smart plug
   - MQTT: `graylogic/command/smart-plug-garage { command: "on" }`
   - Smart plug turns on → heater receives power

3. **State Synchronization:**
   - Smart plug state: `{ on: true, power_kw: 1.5 }`
   - Heater derived state: `{ powered: true, power_kw: 1.5, control_proxy: "smart-plug-garage" }`

**Resolution Priority:**

When a device has multiple associations, resolver applies:
1. Most specific match (exact device_id over group)
2. Type priority: `monitors_and_controls` > `controls` > `monitors`
3. Most recently configured (for conflict resolution)

#### Health Monitor (PHM)

Predictive Health Monitoring for equipment.

```go
// internal/device/health/monitor.go

type Monitor struct {
    db         *database.Database
    influx     *influxdb.Client
    baselines  map[string]*Baseline
    alerter    *Alerter
}

type Baseline struct {
    DeviceID   string
    Metrics    map[string]*MetricBaseline
}

type MetricBaseline struct {
    Name              string
    Method            BaselineMethod  // rolling_average, seasonal, etc.
    Value             float64
    StandardDeviation float64
    SampleCount       int
    LastUpdated       time.Time
}

func (m *Monitor) RecordMetric(deviceID, metric string, value float64)
func (m *Monitor) CheckDeviation(deviceID, metric string, value float64) *Alert
```

---

### Automation Layer

#### Scene Engine

Executes scene actions.

```go
// internal/automation/scene/engine.go

type Engine struct {
    registry   *registry.Registry
    cmdProc    *command.Processor
    stateStore *state.Manager
    scenes     map[string]*Scene
}

type Scene struct {
    ID         string
    Name       string
    Scope      Scope
    Actions    []Action
    Conditions []Condition
}

type Action struct {
    DeviceID    string
    DeviceGroup string
    RoomID      string
    Domain      string
    Command     string
    Parameters  map[string]any
    DelayMs     int
    FadeMs      int
    Parallel    bool
}

func (e *Engine) Activate(sceneID string, source string) error {
    // 1. Load scene definition
    // 2. Evaluate conditions
    // 3. Execute actions (respecting timing, parallelism)
    // 4. Log activation
}
```

**Execution Algorithm:**
```
For each action in scene.Actions:
    If action.Parallel == false:
        Wait for previous action to complete
    
    If action.DelayMs > 0:
        Sleep for delay
    
    Build command from action
    Add fade_ms to command parameters if set
    
    Execute command via CommandProcessor
```

#### Scheduler

Time-based automation triggers.

```go
// internal/automation/scheduler/scheduler.go

type Scheduler struct {
    schedules  map[string]*Schedule
    cron       *cron.Cron
    sceneEngine *scene.Engine
    location   *time.Location
    sunCalc    *Astronomical
}

type Schedule struct {
    ID       string
    Name     string
    Enabled  bool
    Trigger  Trigger
    Execute  Execution
    Conditions []Condition
}

type Trigger struct {
    Type  TriggerType  // time, sunrise, sunset, cron
    Value string
    Days  []time.Weekday
}

func (s *Scheduler) Start()
func (s *Scheduler) EvaluateTrigger(schedule *Schedule) bool
```

**Astronomical Calculations:**
```go
// internal/automation/scheduler/astronomical.go

type Astronomical struct {
    latitude  float64
    longitude float64
}

func (a *Astronomical) Sunrise(date time.Time) time.Time
func (a *Astronomical) Sunset(date time.Time) time.Time
func (a *Astronomical) SolarNoon(date time.Time) time.Time
func (a *Astronomical) SunPosition(t time.Time) (azimuth, elevation float64)
```

#### Mode Manager

System-wide mode state.

```go
// internal/automation/mode/manager.go

type Manager struct {
    db          *database.Database
    current     *Mode
    available   []*Mode
    listeners   []ModeChangeListener
}

type Mode struct {
    ID         string
    Name       string
    Behaviors  Behaviors
}

type Behaviors struct {
    Climate   ClimateBehavior
    Lighting  LightingBehavior
    Security  SecurityBehavior
    Audio     AudioBehavior
}

func (m *Manager) GetCurrent() *Mode
func (m *Manager) SetMode(modeID string, userID string) error
func (m *Manager) Subscribe(listener ModeChangeListener)
```

#### Event Router

Routes device events to automation.

```go
// internal/automation/event/router.go

type Router struct {
    sceneEngine *scene.Engine
    scheduler   *scheduler.Scheduler
    handlers    map[EventType][]EventHandler
}

type Event struct {
    Type       EventType
    DeviceID   string
    OldState   map[string]any
    NewState   map[string]any
    Timestamp  time.Time
}

type EventHandler func(event Event) error

func (r *Router) OnStateChange(deviceID string, oldState, newState map[string]any) {
    // Generate appropriate events
    // Route to handlers
    // Check scene triggers
}
```

#### Conditional Logic Engine

Evaluates complex conditions.

```go
// internal/automation/logic/engine.go

type Engine struct {
    stateStore  *state.Manager
    modeManager *mode.Manager
    sunCalc     *astronomical.Calculator
}

type Condition struct {
    Type     ConditionType
    Operator Operator
    Value    any
    DeviceID string  // for device_state conditions
}

func (e *Engine) Evaluate(conditions []Condition) bool {
    for _, cond := range conditions {
        if !e.evaluateCondition(cond) {
            return false  // AND logic (all must pass)
        }
    }
    return true
}
```

---

### Intelligence Layer

#### Voice Processor

Speech-to-text, NLU, and text-to-speech.

```go
// internal/intelligence/voice/processor.go

type Processor struct {
    stt     *STT       // Whisper
    nlu     *NLU       // Local LLM or rule-based
    tts     *TTS       // Piper
    cmdProc *command.Processor
}

type VoiceCommand struct {
    RoomID     string
    AudioData  []byte
    WakeWordAt time.Time
}

type Intent struct {
    Name       string
    Domain     string
    Action     string
    Room       string
    Parameters map[string]any
    Confidence float64
}

func (p *Processor) Process(cmd VoiceCommand) (response string, err error) {
    // 1. Speech-to-text
    transcript := p.stt.Transcribe(cmd.AudioData)
    
    // 2. Extract intent
    intent := p.nlu.ParseIntent(transcript, cmd.RoomID)
    
    // 3. Execute command
    err = p.executeIntent(intent)
    
    // 4. Generate response
    response = p.generateResponse(intent, err)
    
    return response, err
}
```

#### Presence Engine

Tracks occupancy and user location.

```go
// internal/intelligence/presence/engine.go

type Engine struct {
    trackers   []Tracker
    users      map[string]*UserPresence
    rooms      map[string]*RoomOccupancy
    listeners  []PresenceChangeListener
}

type Tracker interface {
    Type() TrackerType
    Detect() []Detection
}

// Tracker implementations:
// - WiFiTracker (phone MAC detection)
// - MotionTracker (PIR/mmWave sensors)
// - GeofenceTracker (GPS geofencing)

type UserPresence struct {
    UserID    string
    IsHome    bool
    LastSeen  time.Time
    Location  string  // room ID if known
}

type RoomOccupancy struct {
    RoomID    string
    Occupied  bool
    OccupiedSince time.Time
    LastMotion    time.Time
}
```

---

## Startup Sequence

```go
func main() {
    // 1. Load configuration
    cfg := config.Load()
    
    // 2. Initialize logging
    logger := logging.New(cfg.Logging)
    
    // 3. Open database
    db := database.Open(cfg.Database.Path)
    db.Migrate()
    
    // 4. Initialize infrastructure
    mqttClient := mqtt.Connect(cfg.MQTT)
    
    // 5. Initialize device layer
    registry := registry.New(db)
    stateManager := state.New(db, mqttClient)
    cmdProcessor := command.New(registry, mqttClient)
    healthMonitor := health.New(db, cfg.Influx)
    
    // 6. Initialize automation layer
    modeManager := mode.New(db)
    sceneEngine := scene.New(registry, cmdProcessor, stateManager)
    scheduler := scheduler.New(sceneEngine, cfg.Site)
    eventRouter := event.New(sceneEngine, scheduler)
    
    // 7. Connect state changes to event router
    stateManager.Subscribe(eventRouter.OnStateChange)
    
    // 8. Initialize intelligence layer (optional)
    if cfg.Voice.Enabled {
        voiceProcessor := voice.New(cfg.Voice, cmdProcessor)
    }
    presenceEngine := presence.New(registry, stateManager)
    
    // 9. Initialize API and WebSocket servers
    wsHub := websocket.NewHub()
    apiServer := api.New(cfg.API, registry, stateManager, cmdProcessor, sceneEngine)
    
    // 10. Start all components
    go wsHub.Run()
    go scheduler.Start()
    go healthMonitor.Start()
    go presenceEngine.Start()
    
    // 11. Start API server (blocking)
    apiServer.ListenAndServe()
}
```

---

## Error Handling

### Graceful Degradation

```go
// Components fail independently
// System continues with reduced functionality

type ComponentStatus string

const (
    StatusHealthy  ComponentStatus = "healthy"
    StatusDegraded ComponentStatus = "degraded"
    StatusFailed   ComponentStatus = "failed"
)

type HealthStatus struct {
    Overall    ComponentStatus
    Components map[string]ComponentStatus
}

// Example: MQTT connection lost
// - Device commands fail (expected)
// - State updates stop
// - API continues serving cached state
// - UI shows "degraded" status
// - Reconnect in background
```

### Panic Recovery

```go
// All goroutines wrapped with recovery
func safeGoroutine(name string, fn func()) {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                logger.Error("Panic in goroutine",
                    "name", name,
                    "error", r,
                    "stack", debug.Stack())
                // Optionally restart
            }
        }()
        fn()
    }()
}
```

---

## Configuration

### Main Configuration File

```yaml
# /etc/graylogic/config.yaml

site:
  id: "site-oakstreet"
  name: "Oak Street Residence"
  timezone: "Europe/London"
  location:
    latitude: 51.5074
    longitude: -0.1278

database:
  path: "/var/lib/graylogic/graylogic.db"
  
influxdb:
  url: "http://localhost:8086"
  token: "${INFLUX_TOKEN}"
  org: "graylogic"
  bucket: "phm"

mqtt:
  broker: "tcp://localhost:1883"
  client_id: "graylogic-core"
  username: "graylogic"
  password: "${MQTT_PASSWORD}"
  
api:
  listen_address: "0.0.0.0:8080"
  tls:
    enabled: true
    cert_file: "/etc/graylogic/server.crt"
    key_file: "/etc/graylogic/server.key"

voice:
  enabled: true
  whisper:
    model: "base.en"
  piper:
    model: "en_GB-alan-medium"

logging:
  level: "info"
  format: "json"
  output: "/var/log/graylogic/core.log"

metrics:
  enabled: true
  listen_address: "0.0.0.0:9090"
```

---

## Metrics

Prometheus metrics exposed at `/metrics`:

```
# Device metrics
graylogic_devices_total{domain="lighting"} 42
graylogic_device_commands_total{device="light-1", command="dim"} 156
graylogic_device_state_updates_total{device="light-1"} 892

# Scene metrics
graylogic_scene_activations_total{scene="cinema"} 23
graylogic_scene_execution_duration_seconds{scene="cinema"} 2.3

# System metrics
graylogic_mqtt_messages_received_total 45678
graylogic_api_requests_total{method="GET", path="/devices"} 1234
graylogic_websocket_clients_connected 3
```

---

## Related Documents

- [System Overview](system-overview.md) — High-level architecture
- [Automation Specification](../automation/automation.md) — Scenes, schedules, modes, conditions
- [REST and WebSocket API Specification](../interfaces/api.md) — API endpoints and WebSocket events
- [Bridge Interface Specification](bridge-interface.md) — MQTT bridge contract
- [MQTT Protocol Specification](../protocols/mqtt.md) — Message formats
- [Data Model: Entities](../data-model/entities.md) — Entity definitions

