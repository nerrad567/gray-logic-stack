---
title: Error Catalog Specification
version: 1.0.0
status: active
implementation_status: not_started
target_version: v1.x
last_updated: 2026-01-19
depends_on:
  - resilience/offline.md
  - architecture/supervisor.md
  - operations/monitoring.md
---

# Error Catalog Specification

This document defines the machine-readable error catalog system — enabling AI agents and automated tools to diagnose issues and find remediation procedures.

---

## Overview

### The Problem

Current logs are human-readable but not machine-queryable:

```
ERROR: failed to connect to MQTT broker
```

An AI agent reading this log cannot:
- Know if this is a known failure pattern
- Find the documented recovery procedure
- Determine the expected recovery time
- Identify related symptoms

### The Solution

Every error includes a structured ID that references the catalog:

```go
log.Error("failed to connect to MQTT broker",
    "error", err,
    "error_id", "ERR-MQTT-CONN-001",
    "playbook", "PLAY-MQTT-RECOVER-001",
)
```

The catalog provides:
- Structured error metadata
- Symptoms and root causes
- Links to documentation
- Reference to remediation playbooks

---

## Error ID Format

### Structure

```
ERR-{DOMAIN}-{COMPONENT}-{NUMBER}

Example: ERR-MQTT-CONN-001
         ├── ERR       = Error prefix (vs WARN, INFO)
         ├── MQTT      = Domain (infrastructure component)
         ├── CONN      = Component/Category (connection)
         └── 001       = Unique number within category
```

### Domains

| Domain | Description | Examples |
|--------|-------------|----------|
| `INFRA` | Infrastructure layer | Database, MQTT, VictoriaMetrics |
| `DEVICE` | Device layer | Registry, state, discovery |
| `AUTO` | Automation layer | Scenes, scheduler, modes |
| `INTEL` | Intelligence layer | Voice, PHM, AI |
| `BRIDGE` | Protocol bridges | KNX, DALI, Modbus |
| `API` | API layer | REST, WebSocket |
| `SYS` | System-level | Startup, shutdown, resources |

---

## Catalog Structure

### File Format

```yaml
# docs/errors/catalog.yaml

version: "1.0.0"
last_updated: "2026-01-19"

errors:
  - id: "ERR-INFRA-MQTT-001"
    domain: "infrastructure"
    component: "mqtt"
    severity: "critical"
    summary: "MQTT broker connection failed"
    
    # How to detect
    symptoms:
      - metric: "graylogic_mqtt_connected"
        condition: "== 0"
      - log_pattern: "failed to connect to MQTT broker"
      - health_check: "/api/v1/health → mqtt.status == unhealthy"
      
    # What causes it
    root_causes:
      - cause: "Mosquitto service not running"
        likelihood: "high"
        verification: "systemctl status mosquitto"
        
      - cause: "Network partition"
        likelihood: "medium"
        verification: "ping mqtt_host"
        
      - cause: "TLS certificate expired"
        likelihood: "low"
        verification: "openssl s_client -connect mqtt_host:8883"
        
    # How to fix
    remediation:
      automatic: true
      playbook_ref: "PLAY-MQTT-RECOVER-001"
      manual_steps:
        - "Check mosquitto service: systemctl status mosquitto"
        - "Check logs: journalctl -u mosquitto -n 50"
        - "Restart if needed: systemctl restart mosquitto"
        
    # How long it takes
    recovery:
      expected_time_seconds: 30
      max_time_seconds: 120
      escalation_after_seconds: 180
      
    # Related documentation
    doc_refs:
      - path: "docs/architecture/mqtt-resilience.md"
        section: "Recovery Procedures"
      - path: "docs/resilience/offline.md"
        section: "MQTT Broker Down"
        
    # Tags for search
    tags:
      - "mqtt"
      - "connection"
      - "broker"
      - "infrastructure"
```

### Playbook Reference

```yaml
# docs/errors/playbooks.yaml

playbooks:
  - id: "PLAY-MQTT-RECOVER-001"
    name: "MQTT Broker Recovery"
    description: "Automated recovery from MQTT broker failure"
    
    # Triggering errors
    handles_errors:
      - "ERR-INFRA-MQTT-001"
      - "ERR-INFRA-MQTT-002"
      
    # Execution limits
    limits:
      max_attempts: 3
      cooldown_seconds: 60
      timeout_seconds: 120
      
    # Recovery steps
    steps:
      - step: 1
        name: "Check broker status"
        action: "check_service"
        target: "mosquitto"
        timeout_seconds: 5
        on_success: "goto:done"
        on_failure: "goto:2"
        
      - step: 2
        name: "Restart broker"
        action: "restart_service"
        target: "mosquitto"
        timeout_seconds: 30
        on_success: "goto:3"
        on_failure: "goto:escalate"
        
      - step: 3
        name: "Verify recovery"
        action: "wait_for_health"
        target: "mqtt"
        timeout_seconds: 30
        success_criteria: "status == healthy"
        on_success: "goto:done"
        on_failure: "goto:escalate"
        
      - step: "done"
        name: "Recovery complete"
        action: "log"
        message: "MQTT broker recovered successfully"
        
      - step: "escalate"
        name: "Escalate to human"
        action: "alert"
        severity: "critical"
        notify:
          - "admin"
          - "on_call"
        message: "MQTT broker unrecoverable after {attempts} attempts"
```

---

## Error Categories

### Infrastructure Errors (ERR-INFRA-*)

| ID | Summary | Severity | Auto-Recover |
|----|---------|----------|--------------|
| `ERR-INFRA-MQTT-001` | MQTT broker connection failed | Critical | Yes |
| `ERR-INFRA-MQTT-002` | MQTT publish timeout | High | Yes |
| `ERR-INFRA-DB-001` | Database connection failed | Critical | Partial |
| `ERR-INFRA-DB-002` | Database query timeout | High | Yes |
| `ERR-INFRA-DB-003` | Database integrity error | Critical | No |
| `ERR-INFRA-INFLUX-001` | VictoriaMetrics connection failed | Medium | Yes |
| `ERR-INFRA-INFLUX-002` | VictoriaMetrics write failed | Medium | Yes (queue) |

### Device Errors (ERR-DEVICE-*)

| ID | Summary | Severity | Auto-Recover |
|----|---------|----------|--------------|
| `ERR-DEVICE-OFFLINE-001` | Device stopped responding | Medium | Partial |
| `ERR-DEVICE-STATE-001` | State reconciliation conflict | Low | Yes |
| `ERR-DEVICE-CMD-001` | Command execution timeout | Medium | Yes (retry) |
| `ERR-DEVICE-CMD-002` | Command rejected by device | Medium | No |

### Bridge Errors (ERR-BRIDGE-*)

| ID | Summary | Severity | Auto-Recover |
|----|---------|----------|--------------|
| `ERR-BRIDGE-CONN-001` | Bridge disconnected | High | Yes |
| `ERR-BRIDGE-CLOCK-001` | Bridge clock skew detected | Medium | Partial |
| `ERR-BRIDGE-PROTO-001` | Protocol error from device | Medium | No |
| `ERR-BRIDGE-HEARTBEAT-001` | Bridge heartbeat missing | High | Yes |

### Automation Errors (ERR-AUTO-*)

| ID | Summary | Severity | Auto-Recover |
|----|---------|----------|--------------|
| `ERR-AUTO-SCENE-001` | Scene execution timeout | Medium | Yes |
| `ERR-AUTO-SCENE-002` | Scene partially executed | Low | No |
| `ERR-AUTO-SCHED-001` | Scheduler missed trigger | Low | Partial |
| `ERR-AUTO-CONFLICT-001` | Scene conflict detected | Low | Yes |

### System Errors (ERR-SYS-*)

| ID | Summary | Severity | Auto-Recover |
|----|---------|----------|--------------|
| `ERR-SYS-MEM-001` | Memory pressure critical | Critical | Partial |
| `ERR-SYS-DISK-001` | Disk space critical | Critical | Partial |
| `ERR-SYS-START-001` | Service startup failed | Critical | Yes |
| `ERR-SYS-CERT-001` | Certificate expiring soon | Medium | Yes |

---

## Integration with Logging

### Go Implementation

```go
// pkg/errors/catalog.go

package errors

// ErrorID represents a cataloged error
type ErrorID string

const (
    ErrMQTTConnection    ErrorID = "ERR-INFRA-MQTT-001"
    ErrMQTTPublishTimeout ErrorID = "ERR-INFRA-MQTT-002"
    ErrDatabaseConnection ErrorID = "ERR-INFRA-DB-001"
    // ... more constants
)

// CatalogEntry contains metadata for an error
type CatalogEntry struct {
    ID          ErrorID
    Severity    string
    Summary     string
    PlaybookRef string
    DocRefs     []string
}

// Catalog provides lookup for error metadata
var Catalog = map[ErrorID]CatalogEntry{
    ErrMQTTConnection: {
        ID:          ErrMQTTConnection,
        Severity:    "critical",
        Summary:     "MQTT broker connection failed",
        PlaybookRef: "PLAY-MQTT-RECOVER-001",
        DocRefs:     []string{"docs/architecture/mqtt-resilience.md"},
    },
    // ... more entries
}

// Lookup returns catalog entry for an error ID
func Lookup(id ErrorID) (CatalogEntry, bool) {
    entry, ok := Catalog[id]
    return entry, ok
}
```

### Structured Logging

```go
// Using the catalog in logging

func (c *MQTTClient) Connect() error {
    err := c.client.Connect()
    if err != nil {
        entry, _ := errors.Lookup(errors.ErrMQTTConnection)
        
        c.logger.Error("failed to connect to MQTT broker",
            "error", err,
            "error_id", entry.ID,
            "severity", entry.Severity,
            "playbook", entry.PlaybookRef,
        )
        return fmt.Errorf("%s: %w", entry.ID, err)
    }
    return nil
}
```

### Log Output

```json
{
  "level": "error",
  "msg": "failed to connect to MQTT broker",
  "error": "dial tcp 127.0.0.1:1883: connection refused",
  "error_id": "ERR-INFRA-MQTT-001",
  "severity": "critical",
  "playbook": "PLAY-MQTT-RECOVER-001",
  "timestamp": "2026-01-19T16:00:00Z"
}
```

---

## AI Agent Usage

### Querying the Catalog

An AI agent can:

1. **Parse log output** for `error_id` field
2. **Load catalog.yaml** to get metadata
3. **Follow doc_refs** to read relevant documentation
4. **Reference playbook** for automated remediation steps

### Example Agent Workflow

```
Agent observes: error_id="ERR-INFRA-MQTT-001"
        │
        ▼
Load catalog.yaml → Find entry for ERR-INFRA-MQTT-001
        │
        ├── symptoms: Check if all symptoms present
        ├── root_causes: Evaluate likelihood of each
        ├── remediation: Follow manual_steps or trigger playbook
        └── doc_refs: Read related documentation for context
        │
        ▼
Execute remediation or escalate to human
```

### Catalog as Context

The catalog can be loaded as context for AI agents:

```python
# Agent prompt augmentation
catalog = load_yaml("docs/errors/catalog.yaml")
error_context = catalog['errors'].get(error_id)

prompt = f"""
The system reported error: {error_id}
Summary: {error_context['summary']}
Severity: {error_context['severity']}
Root causes (by likelihood):
{format_causes(error_context['root_causes'])}

Manual remediation steps:
{format_steps(error_context['remediation']['manual_steps'])}

Please diagnose and recommend action.
"""
```

---

## Maintenance

### Adding New Errors

1. Assign ID following naming convention
2. Add entry to `catalog.yaml`
3. Create Go constant in `pkg/errors/catalog.go`
4. Update logging calls to include `error_id`
5. Create or reference playbook if auto-recovery desired

### Reviewing Errors

Quarterly review:
- Are all logged errors cataloged?
- Are playbooks effective (success rate)?
- Are doc_refs still accurate?
- Any new patterns to catalog?

---

## Implementation Milestones

### M-ERR.1: Catalog Schema

**Deliverables:**
- [ ] Define YAML schema for catalog
- [ ] Create initial `catalog.yaml` with 20 core errors
- [ ] Create initial `playbooks.yaml` with 5 playbooks

**Acceptance:**
- Schema validates correctly
- All errors from offline.md are cataloged

### M-ERR.2: Go Integration

**Deliverables:**
- [ ] Create `pkg/errors/catalog.go`
- [ ] Define error ID constants
- [ ] Update logging utilities to include error_id

**Acceptance:**
- Logs include error_id field
- Constants match catalog.yaml

### M-ERR.3: Documentation Links

**Deliverables:**
- [ ] Verify all doc_refs are valid
- [ ] Add catalog references to key docs
- [ ] Create error index page

**Acceptance:**
- All links resolve correctly
- Docs cross-reference catalog

---

## See Also

- [Offline Behavior](../resilience/offline.md) — Failure scenarios
- [Supervisor Specification](../architecture/supervisor.md) — Automated recovery
- [Monitoring](../operations/monitoring.md) — Metrics and alerting
- [Development Strategy](../development/DEVELOPMENT-STRATEGY.md) — Milestones
