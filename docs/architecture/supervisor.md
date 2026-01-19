---
title: System Supervisor Specification
version: 0.1.0
status: planned
implementation_status: not_started
target_version: v2.0
last_updated: 2026-01-19
depends_on:
  - resilience/offline.md
  - architecture/mqtt-resilience.md
  - architecture/core-internals.md
  - operations/monitoring.md
---

# System Supervisor Specification

This document specifies the future System Supervisor — an intelligent orchestration layer that enhances system stability through proactive monitoring, automated remediation, and health aggregation.

> [!NOTE]
> **Status: Planned for v2.0 (Year 3+)**
> 
> This is a future enhancement. Current deployments (v1.x) use Level 1-2 recovery mechanisms documented in [offline.md](../resilience/offline.md) and [mqtt-resilience.md](mqtt-resilience.md).

---

## Overview

### Recovery Architecture Levels

The Supervisor is the third level of a layered recovery architecture:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ LEVEL 3: Supervisor Agent (v2.0+ Future)                                     │
│          Observes Level 2 recovery attempts                                  │
│          Adds intelligence: "restart hasn't worked 3x, try alternative"      │
│          Aggregates health across subsystems                                 │
│          Decides when to escalate vs. continue trying                        │
├─────────────────────────────────────────────────────────────────────────────┤
│ LEVEL 2: Basic Recovery (v1.0 Current — Deploy Now)                          │
│          systemd service restart policies                                    │
│          MQTT broker persistence + reconnection                              │
│          Database backup before migrations                                   │
│          Documented in: offline.md, mqtt-resilience.md                       │
├─────────────────────────────────────────────────────────────────────────────┤
│ LEVEL 1: Hardware/OS (Always Present — Cannot Remove)                        │
│          systemd watchdog, kernel OOM killer                                 │
│          This exists even if Gray Logic isn't installed                      │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Key Principle

**The Supervisor complements, never replaces, Level 1-2 mechanisms.**

- Level 2 (systemd restarts, MQTT persistence) continues to operate
- Supervisor observes their outcomes and makes higher-order decisions
- If Supervisor fails, Level 2 continues functioning independently

---

## Scope

### In Scope (v2.0)

| Capability | Description |
|------------|-------------|
| **Health Aggregation** | Unified view of all subsystem health states |
| **Recovery Orchestration** | Coordinate retries, backoff, and escalation |
| **Anomaly Detection** | Identify patterns indicating impending failure |
| **Alerting Escalation** | Progressive notification (UI → push → SMS) |
| **Remediation Logging** | Track what was tried, what worked |

### Out of Scope (Not in v2.0)

| Capability | Reason | Alternative |
|------------|--------|-------------|
| Backup hardware failover | Residential sites rarely have redundant buses | Document as commercial premium feature |
| AI-driven remediation selection | Requires significant training data | Manual playbooks first |
| Cross-site coordination | Single-site focus for now | Future multi-site management spec |

---

## Architecture

### Package Structure

```
internal/
├── infrastructure/          # Current packages (unchanged)
├── device/                  # Current packages (unchanged)
├── automation/              # Current packages (unchanged)
├── intelligence/            # Current packages (unchanged)
├── domain/                  # Current packages (unchanged)
│
└── resilience/              # NEW: Supervisor Layer
    ├── supervisor/          # Main orchestration
    │   ├── supervisor.go    # Background loop
    │   ├── strategy.go      # Decision trees
    │   └── config.go        # Supervisor configuration
    │
    ├── health/              # Health aggregation
    │   ├── aggregator.go    # Collects all subsystem health
    │   ├── reporter.go      # Exposes /api/v1/system/health
    │   └── checker.go       # Health check interface
    │
    ├── circuit/             # Circuit breaker pattern
    │   ├── breaker.go       # Generic circuit breaker
    │   └── states.go        # open/half-open/closed
    │
    └── playbook/            # Remediation actions
        ├── executor.go      # Runs playbook steps
        ├── registry.go      # Maps failure patterns → playbooks
        └── actions.go       # Available action types
```

### Integration Point

```go
// cmd/graylogic/main.go

func main() {
    // 1. Initialize infrastructure (existing)
    db := database.New(cfg.Database)
    mqtt := mqtt.NewClient(cfg.MQTT)
    influx := influxdb.NewClient(cfg.InfluxDB)
    
    // 2. Initialize Supervisor (new - v2.0)
    supervisor := resilience.NewSupervisor(
        resilience.WithHealthSources(db, mqtt, influx),
        resilience.WithPlaybooks(loadPlaybooks()),
        resilience.WithConfig(cfg.Supervisor),
    )
    
    // 3. Register subsystems
    supervisor.Register("database", db)
    supervisor.Register("mqtt", mqtt)
    supervisor.Register("scene_engine", sceneEngine)
    
    // 4. Start Supervisor loop (background goroutine)
    go supervisor.Run(ctx)
    
    // 5. Start other services (existing)
    ...
}
```

---

## Core Components

### Health Aggregator

Collects health status from all subsystems into a unified view.

```go
// internal/resilience/health/aggregator.go

type HealthChecker interface {
    HealthCheck(ctx context.Context) HealthStatus
    Name() string
}

type HealthStatus struct {
    Status      Status              // healthy | degraded | unhealthy
    Message     string              // Human-readable description
    CheckedAt   time.Time
    Metrics     map[string]float64  // Optional metrics
}

type Status string

const (
    StatusHealthy   Status = "healthy"
    StatusDegraded  Status = "degraded"
    StatusUnhealthy Status = "unhealthy"
)

type Aggregator struct {
    checkers []HealthChecker
    interval time.Duration  // How often to poll (default: 10s)
    results  map[string]HealthStatus
    mu       sync.RWMutex
}

func (a *Aggregator) GetSystemHealth() SystemHealth {
    a.mu.RLock()
    defer a.mu.RUnlock()
    
    overall := StatusHealthy
    components := make(map[string]HealthStatus)
    
    for name, status := range a.results {
        components[name] = status
        if status.Status == StatusUnhealthy {
            overall = StatusUnhealthy
        } else if status.Status == StatusDegraded && overall != StatusUnhealthy {
            overall = StatusDegraded
        }
    }
    
    return SystemHealth{
        Overall:    overall,
        Components: components,
        CheckedAt:  time.Now(),
    }
}
```

### Supervisor Loop

The main decision-making loop.

```go
// internal/resilience/supervisor/supervisor.go

type Supervisor struct {
    health     *health.Aggregator
    playbooks  *playbook.Registry
    config     Config
    state      map[string]*SubsystemState
}

type SubsystemState struct {
    Name            string
    Status          health.Status
    FailureCount    int
    LastFailure     time.Time
    RecoveryAttempt int
    InRecovery      bool
}

func (s *Supervisor) Run(ctx context.Context) {
    ticker := time.NewTicker(s.config.CheckInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.evaluate()
        }
    }
}

func (s *Supervisor) evaluate() {
    systemHealth := s.health.GetSystemHealth()
    
    for name, status := range systemHealth.Components {
        state := s.getOrCreateState(name)
        
        switch status.Status {
        case health.StatusHealthy:
            s.handleHealthy(state)
        case health.StatusDegraded:
            s.handleDegraded(state, status)
        case health.StatusUnhealthy:
            s.handleUnhealthy(state, status)
        }
    }
}

func (s *Supervisor) handleUnhealthy(state *SubsystemState, status health.HealthStatus) {
    state.FailureCount++
    state.LastFailure = time.Now()
    
    // Find applicable playbook
    playbook := s.playbooks.FindPlaybook(state.Name, status)
    if playbook == nil {
        s.escalate(state, "No playbook found for failure")
        return
    }
    
    // Determine next step based on recovery attempts
    if state.RecoveryAttempt >= playbook.MaxAttempts {
        s.escalate(state, "Max recovery attempts exceeded")
        return
    }
    
    // Execute next recovery step
    state.RecoveryAttempt++
    state.InRecovery = true
    
    go s.executePlaybook(state, playbook)
}
```

### Playbook System

Defines remediation actions for failure patterns.

```yaml
# config/playbooks/mqtt-recovery.yaml

id: "PLAY-MQTT-RECOVER-001"
name: "MQTT Broker Recovery"
trigger:
  component: "mqtt"
  status: "unhealthy"
  
max_attempts: 3
cooldown_seconds: 60

steps:
  - name: "Check broker status"
    action: "check_service"
    target: "mosquitto"
    success_criteria: "status == running"
    on_success: "done"
    on_failure: "next"
    
  - name: "Restart broker"
    action: "restart_service"
    target: "mosquitto"
    timeout_seconds: 30
    
  - name: "Verify recovery"
    action: "wait_for_health"
    target: "mqtt"
    timeout_seconds: 30
    success_criteria: "status == healthy"
    on_failure: "escalate"
    
escalation:
  notify:
    - "admin"
    - "on_call"
  message: "MQTT broker unrecoverable after 3 restart attempts"
```

---

## Monitoring Scope

### What the Supervisor Monitors

| Subsystem | Health Indicators | Failure Patterns |
|-----------|-------------------|------------------|
| **MQTT Broker** | Connection status, message latency | Connection lost, high latency |
| **Database** | Query time, disk space, integrity | Slow queries, disk full, corruption |
| **InfluxDB** | Write success rate, queue depth | Write failures, queue overflow |
| **Bridges** | Heartbeat presence, clock skew | Missing heartbeats, skew >60s |
| **Scene Engine** | Execution time, success rate | Timeout, high failure rate |
| **API Server** | Response time, error rate | Slow responses, 5xx errors |

### Anomaly Detection Patterns

```yaml
anomaly_detection:
  - pattern: "rapid_failure"
    description: "Multiple failures in short period"
    threshold: "5 failures in 60 seconds"
    action: "enter_safe_mode"
    
  - pattern: "state_oscillation"
    description: "Device state toggling rapidly"
    threshold: "10 state changes in 10 seconds"
    action: "stabilization_window"
    
  - pattern: "memory_pressure"
    description: "Memory usage approaching limit"
    threshold: "memory > 80% for 5 minutes"
    action: "alert_and_gc"
    
  - pattern: "disk_pressure"
    description: "Disk space running low"
    threshold: "disk > 90%"
    action: "alert_and_cleanup"
```

---

## Integration with Existing Specs

### Relationship to offline.md

The Supervisor uses the failure matrices and degradation levels defined in [offline.md](../resilience/offline.md) as inputs:

- Degradation levels (0-4) map to Supervisor response severity
- Component failure impacts inform playbook selection
- Recovery procedures become automated playbooks

### Relationship to mqtt-resilience.md

The MQTT-specific recovery documented in [mqtt-resilience.md](mqtt-resilience.md) becomes a playbook:

- Reconnection backoff algorithm → Playbook step timing
- Health check mechanism → Health checker implementation
- QoS enforcement → Anomaly detection pattern

### Relationship to monitoring.md

The monitoring infrastructure in [monitoring.md](../operations/monitoring.md) provides data to the Supervisor:

- Prometheus metrics → Health indicators
- Alert thresholds → Anomaly detection triggers
- Health endpoint → Data for aggregator

---

## Configuration

```yaml
# /etc/graylogic/supervisor.yaml

supervisor:
  enabled: true
  check_interval_seconds: 10
  
  # Safe mode triggers
  safe_mode:
    enabled: true
    trigger: "5 failures in 60 seconds"
    actions:
      - "disable_automation"
      - "alert_admin"
    recovery: "manual_only"
    
  # Escalation chain
  escalation:
    levels:
      - delay_seconds: 0
        notify: ["ui_banner"]
      - delay_seconds: 300
        notify: ["push_notification"]
      - delay_seconds: 900
        notify: ["sms"]
        
  # Per-component settings
  components:
    mqtt:
      check_interval_seconds: 5
      max_recovery_attempts: 3
      playbook: "PLAY-MQTT-RECOVER-001"
      
    database:
      check_interval_seconds: 30
      max_recovery_attempts: 1
      playbook: "PLAY-DB-RECOVER-001"
```

---

## Implementation Milestones

> [!IMPORTANT]
> **These milestones are for Year 3+ (v2.0).** Current development focuses on M1.x milestones in [implementation-roadmap.md](../development/implementation-roadmap.md).

### M-SUP.1: Health Aggregation

**Deliverables:**
- [ ] Define `HealthChecker` interface
- [ ] Implement health checkers for: Database, MQTT, InfluxDB
- [ ] Implement `Aggregator` to collect health states
- [ ] Expose `/api/v1/system/health` endpoint

**Acceptance:**
- Health endpoint returns status of all components
- Status updates within 10 seconds of failure

### M-SUP.2: Supervisor Loop

**Deliverables:**
- [ ] Implement `Supervisor` main loop
- [ ] Implement subsystem state tracking
- [ ] Implement basic recovery decision tree (restart-only)

**Acceptance:**
- Supervisor restarts failed service automatically
- Recovery attempts are logged with outcomes

### M-SUP.3: Playbook System

**Deliverables:**
- [ ] Define playbook YAML schema
- [ ] Implement playbook parser and executor
- [ ] Create initial playbooks: MQTT, Database, Bridge recovery

**Acceptance:**
- Playbooks execute correct steps in order
- Step outcomes affect next step selection

### M-SUP.4: Circuit Breaker

**Deliverables:**
- [ ] Implement circuit breaker pattern
- [ ] Apply to MQTT client connection
- [ ] Apply to InfluxDB client writes

**Acceptance:**
- After N failures, circuit opens (stops retrying)
- After cooldown, circuit half-opens (tries one request)
- Successful request closes circuit

### M-SUP.5: Anomaly Detection

**Deliverables:**
- [ ] Implement pattern matchers for anomaly types
- [ ] Implement safe mode entry/exit
- [ ] Implement stabilization window for oscillations

**Acceptance:**
- Rapid failures trigger safe mode
- State oscillation triggers stabilization (commands rejected for 5s)

---

## Testing Strategy

### Unit Tests

- Health checker returns correct status
- Aggregator computes overall status correctly
- Supervisor state machine transitions correctly
- Playbook steps execute in order

### Integration Tests

- Kill MQTT broker → Supervisor detects → Playbook runs → Recovery verified
- Simulate 5 rapid failures → Safe mode entered
- State oscillation → Stabilization window activated

### Chaos Testing (Future)

See [simulation.md](../development/simulation.md) for chaos injection framework.

---

## See Also

- [Offline Behavior](../resilience/offline.md) — Failure matrices and degradation
- [MQTT Resilience](mqtt-resilience.md) — MQTT-specific recovery
- [Monitoring](../operations/monitoring.md) — Metrics and alerting
- [Development Strategy](../development/DEVELOPMENT-STRATEGY.md) — Year 3+ milestones
- [Simulation & Testing](../development/simulation.md) — Chaos testing framework
- [Failure Memory](failure-memory.md) — Learning from actual failures
- [Workflow Learning](workflow-learning.md) — Improving playbooks over time
- [Error Catalog](../errors/catalog.md) — Machine-readable error system
