---
title: Simulation & Testing Framework
version: 1.0.0
status: planned
implementation_status: not_started
target_version: v2.0
last_updated: 2026-01-19
depends_on:
  - development/testing-strategy.md
  - architecture/supervisor.md
  - resilience/offline.md
---

# Simulation & Testing Framework

This document specifies the simulation and chaos testing framework for validating Gray Logic's resilience and self-healing capabilities without risking production hardware.

---

## Overview

### The Problem

Testing resilience code is inherently risky:
- Killing services to test recovery can corrupt data
- Simulating hardware failures requires real hardware
- Self-healing loops can cause cascading failures if buggy
- AI agents making changes could brick physical systems

### The Solution

A layered testing approach with increasing fidelity and risk:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Level 4: Production (Live Site)                                              │
│          Real hardware, real data, monitored                                 │
│          Risk: HIGH — Only after all other levels pass                       │
├─────────────────────────────────────────────────────────────────────────────┤
│ Level 3: Staging (Test Rig)                                                  │
│          Real hardware (lab bench), isolated network                         │
│          Risk: MEDIUM — Can break test equipment                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ Level 2: Integration (Containers)                                            │
│          Real Core + simulated bridges + real MQTT/VictoriaMetrics                  │
│          Risk: LOW — Can corrupt dev data                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│ Level 1: Unit (Mocks)                                                        │
│          All dependencies mocked, pure logic testing                         │
│          Risk: NONE — Fast, safe, runs in CI                                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Simulation Modes

### Mode 1: Unit Testing (Level 1)

Pure Go tests with mocked interfaces. No external dependencies.

```go
// Example: Testing Supervisor decision logic
func TestSupervisor_HandleUnhealthy(t *testing.T) {
    // Mock health checker
    mockHealth := &MockHealthChecker{
        Status: health.StatusUnhealthy,
    }
    
    // Mock playbook executor
    mockExecutor := &MockPlaybookExecutor{}
    
    supervisor := NewSupervisor(
        WithHealthSources(mockHealth),
        WithExecutor(mockExecutor),
    )
    
    // Trigger evaluation
    supervisor.evaluate()
    
    // Assert playbook was executed
    assert.True(t, mockExecutor.WasCalled)
    assert.Equal(t, "PLAY-MQTT-RECOVER-001", mockExecutor.PlaybookID)
}
```

**When to use:**
- Testing decision logic
- Testing state machine transitions
- Testing error handling paths

**Limitations:**
- Doesn't catch integration issues
- Mock behavior may differ from real components

---

### Mode 2: Integration Testing (Level 2)

Real Core, real MQTT broker, real VictoriaMetrics, but simulated bridges.

```yaml
# docker-compose.test.yml

services:
  mqtt:
    image: eclipse-mosquitto:2.0.18
    
  tsdb:
    image: victoriametrics/victoria-metrics:v1.135.0
    
  # Simulated bridges (respond to commands, generate fake state)
  mock-knx-bridge:
    image: graylogic/mock-bridge:latest
    environment:
      PROTOCOL: knx
      DEVICE_COUNT: 20
      FAILURE_RATE: 0.05  # 5% of commands fail
      
  mock-dali-bridge:
    image: graylogic/mock-bridge:latest
    environment:
      PROTOCOL: dali
      DEVICE_COUNT: 50
      FAILURE_RATE: 0.02
```

**Mock Bridge Behavior:**

```yaml
mock_bridge_configuration:
  # Simulated devices
  devices:
    - id: "mock-light-1"
      type: "dimmer"
      initial_state: { on: false, level: 0 }
      
  # Response behavior
  behavior:
    command_latency_ms: 50          # Simulated response time
    state_update_interval_s: 1      # Heartbeat frequency
    failure_rate: 0.05              # 5% of commands fail
    
  # Chaos injection hooks
  chaos:
    - type: "disconnect"
      trigger: "CHAOS_DISCONNECT"    # Publish to this topic to trigger
      duration_s: 30
      
    - type: "clock_skew"
      trigger: "CHAOS_CLOCK_SKEW"
      skew_seconds: 120
```

**When to use:**
- Testing end-to-end command flow
- Testing Supervisor recovery sequences
- Testing state synchronization

---

### Mode 3: Hardware-in-the-Loop (Level 3)

Real Core connected to simulated PLCs or test hardware.

```yaml
hardware_in_loop:
  # PLC simulator (Codesys, OpenPLC)
  plc_simulator:
    type: "codesys"
    project: "test_rig.project"
    devices:
      - "simulated_knx_actuator"
      - "simulated_modbus_vfd"
      
  # Physical test bench
  test_bench:
    - "Real KNX IP router (for protocol testing)"
    - "Real DALI gateway (for lighting protocol)"
    - "Isolated network switch"
    - "UPS for power failure testing"
```

**When to use:**
- Protocol-level testing
- Timing-sensitive tests
- Power failure recovery tests

---

### Mode 4: Staging (Level 3.5)

Full system on isolated network with real (expendable) hardware.

**Test Rig Components:**
- Industrial PC running Gray Logic Core
- Real KNX actuators (2-3 for testing)
- Real DALI driver (1 for testing)
- Isolated VLAN
- Power distribution with remote switching (for power failure simulation)

---

## Chaos Injection Framework

### Chaos Types

| Chaos Type | Description | Injection Method |
|------------|-------------|------------------|
| `service_crash` | Kill a service process | `kill -9 <pid>` or Docker stop |
| `network_partition` | Isolate component from network | iptables rules |
| `clock_skew` | Introduce time drift | `date -s` or NTP manipulation |
| `disk_full` | Exhaust disk space | Create large temp file |
| `memory_pressure` | Cause OOM conditions | Allocate memory in subprocess |
| `slow_network` | Add latency/packet loss | `tc qdisc` rules |
| `state_oscillation` | Rapid state changes | Automated toggle loop |

### Chaos Controller

```yaml
# config/chaos/test_mqtt_recovery.yaml

name: "Test MQTT Broker Recovery"
description: "Verify Supervisor handles MQTT broker failure"

# Pre-conditions
preconditions:
  - "System healthy for 60 seconds"
  - "No active scenes"

# Chaos injection
inject:
  - type: "service_crash"
    target: "mosquitto"
    at: "0s"
    
# Expected behavior
expect:
  - at: "5s"
    condition: "supervisor.state.mqtt == 'unhealthy'"
    
  - at: "10s"
    condition: "playbook.PLAY-MQTT-RECOVER-001.step == 1"
    
  - at: "30s"
    condition: "mqtt.status == 'healthy'"
    
  - at: "60s"
    condition: "system.health == 'healthy'"

# Post-conditions
postconditions:
  - "No data loss in database"
  - "State reconciliation completed"
  - "Audit log contains recovery events"
```

### Chaos Test Runner

```go
// pkg/chaos/runner.go

type ChaosRunner struct {
    scenarios []ChaosScenario
    injectors map[string]Injector
}

func (r *ChaosRunner) Run(ctx context.Context, scenario ChaosScenario) TestResult {
    // 1. Verify preconditions
    if err := r.verifyPreconditions(scenario.Preconditions); err != nil {
        return TestResult{Status: "skipped", Reason: err.Error()}
    }
    
    // 2. Inject chaos
    for _, injection := range scenario.Inject {
        injector := r.injectors[injection.Type]
        injector.Inject(injection)
    }
    
    // 3. Monitor expectations
    for _, expectation := range scenario.Expect {
        if !r.waitForCondition(expectation) {
            return TestResult{Status: "failed", Expectation: expectation}
        }
    }
    
    // 4. Verify postconditions
    if err := r.verifyPostconditions(scenario.Postconditions); err != nil {
        return TestResult{Status: "failed", Reason: err.Error()}
    }
    
    return TestResult{Status: "passed"}
}
```

---

## Safety Guardrails

### For Development Environment

See [Development Environment](environment.md) for isolated development setup.

**Key protections:**
- Restricted user account (`graylogic-dev`)
- Dev data in `/opt/graylogic-dev/` (not `/var/lib/`)
- Containerized dependencies
- Easy reset procedures

### For Chaos Testing

```yaml
chaos_safety:
  # Kill switch
  kill_switch:
    enabled: true
    trigger: "Ctrl+C or SIGTERM"
    action: "Immediately stop all chaos, restore normal operation"
    
  # Automatic cleanup
  cleanup:
    on_test_end: true
    on_test_failure: true
    actions:
      - "Restart crashed services"
      - "Remove network rules"
      - "Clear disk fill files"
      
  # Scope limits
  limits:
    max_chaos_duration_s: 300      # 5 minutes max
    max_concurrent_injections: 2
    forbidden_targets:
      - "host_system"
      - "network_switch"
      - "power_controller"  # Only in controlled tests
      
  # Isolation
  isolation:
    network: "isolated_vlan"
    no_internet: true
    no_production_access: true
```

### For AI Agent Operations

When AI agents are developing/testing:

```yaml
agent_safety:
  # Read-only by default
  default_mode: "read_only"
  
  # Require explicit approval for:
  approval_required:
    - "Database schema changes"
    - "Configuration file modifications"
    - "Service restarts"
    - "Chaos injection"
    
  # Automatic rollback
  rollback:
    enabled: true
    checkpoint_before: ["database_write", "config_change"]
    auto_rollback_on: "5 consecutive errors"
    
  # Audit
  audit:
    log_all_actions: true
    include_intent: true  # Why the agent took this action
```

---

## Integration with Supervisor

The Simulation framework validates Supervisor behavior:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        TESTING FEEDBACK LOOP                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   Chaos Injection ──────► System Failure ──────► Supervisor Detects          │
│                                                         │                    │
│                                                         ▼                    │
│   Test Assertion ◄────── Recovery Complete ◄───── Playbook Executes          │
│                                                                              │
│   If test passes: Supervisor behavior validated                              │
│   If test fails: Supervisor or playbook needs refinement                     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Implementation Milestones

### M-SIM.1: Mock Bridge Framework

**Deliverables:**
- [ ] Define mock bridge Docker image
- [ ] Implement configurable device simulation
- [ ] Implement chaos injection hooks
- [ ] Document mock bridge configuration

**Acceptance:**
- Mock bridge responds to commands like real bridge
- Chaos injection triggers work via MQTT

### M-SIM.2: Chaos Controller

**Deliverables:**
- [ ] Define chaos scenario YAML schema
- [ ] Implement chaos runner
- [ ] Implement injectors: service_crash, network_partition
- [ ] Implement cleanup and kill switch

**Acceptance:**
- Chaos scenarios execute reliably
- Cleanup occurs even on test failure

### M-SIM.3: Integration Test Suite

**Deliverables:**
- [ ] Create docker-compose.test.yml
- [ ] Write minimal Supervisor recovery tests
- [ ] Integrate with CI pipeline

**Acceptance:**
- Tests run in under 5 minutes
- Tests are deterministic (no flaky tests)

### M-SIM.4: Hardware-in-Loop Setup

**Deliverables:**
- [ ] Document test bench requirements
- [ ] Create PLC simulator project
- [ ] Create power cycling test procedures

**Acceptance:**
- Protocol tests pass on simulated hardware
- Power failure recovery verified

---

## See Also

- [Testing Strategy](testing-strategy.md) — Overall testing approach
- [Development Environment](environment.md) — Local dev setup
- [Supervisor Specification](../architecture/supervisor.md) — What we're testing
- [Offline Behavior](../resilience/offline.md) — Expected failure behavior
