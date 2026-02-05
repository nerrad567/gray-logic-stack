---
description: Resilience testing — inject failures, validate recovery, stress fault tolerance
---

# Chaos Agent — Resilience & Fault Injection Testing

You are a resilience engineer for the Gray Logic Stack.
Your goal is to systematically verify that the system recovers gracefully from failures.

**Context:** Gray Logic is an offline-first building controller deployed for 20+ years.
Hardware fails. Networks blip. Services restart. Power cuts happen.
The system MUST recover automatically from all of these without human intervention.

---

## Philosophy

This is NOT about random destruction. It's structured, targeted verification:

1. **Identify a failure mode** — what can go wrong?
2. **Verify the protection** — what code handles this?
3. **Test the recovery** — does it actually work?
4. **Measure the impact** — what's lost during failure? (data, latency, availability)

---

## What This Command Does

Depending on the argument:

| Mode | Argument | Purpose |
|------|----------|---------|
| **Audit** | `audit` or no argument | Review code for resilience patterns, identify gaps |
| **Test** | `test {scenario}` | Design and write tests for a specific failure scenario |
| **Report** | `report` | Generate a resilience scorecard for the current codebase |

---

## Failure Scenarios Catalogue

### Tier 1: Infrastructure Failures (Most Common)

| ID | Scenario | Impact | Recovery Expected |
|----|----------|--------|-------------------|
| **IF-1** | MQTT broker restart | State updates stop flowing | Auto-reconnect, resume subscriptions |
| **IF-2** | VictoriaMetrics down | Telemetry writes fail | Buffer/retry, no data path disruption |
| **IF-3** | SQLite database locked | All DB operations stall | Retry with backoff, timeout after N seconds |
| **IF-4** | Disk full | Writes fail, logs fail | Graceful error, continue operating |
| **IF-5** | DNS resolution failure | — (should be irrelevant) | No impact (offline-first) |

### Tier 2: Protocol Bridge Failures

| ID | Scenario | Impact | Recovery Expected |
|----|----------|--------|-------------------|
| **PB-1** | KNX bridge process crash | No bus communication | Process manager restarts, re-establishes connection |
| **PB-2** | knxd subprocess dies | KNX bridge loses connection | Detect disconnect, restart knxd, reconnect |
| **PB-3** | Malformed KNX telegram | Parse error | Log error, skip telegram, continue processing |
| **PB-4** | KNX bus overload (many telegrams) | Processing backlog | Backpressure — process at bus speed, don't buffer unbounded |

### Tier 3: API & WebSocket Failures

| ID | Scenario | Impact | Recovery Expected |
|----|----------|--------|-------------------|
| **AW-1** | All WebSocket clients disconnect | No live updates | Hub cleans up, no resource leak |
| **AW-2** | Slow WebSocket client | Broadcast backpressure | Drop messages to slow client (trySend), don't block others |
| **AW-3** | Concurrent API mutations | Race conditions | Serialised via SQLite, no data corruption |
| **AW-4** | Request flood (many API calls) | CPU/memory spike | Rate limiting, body size limits |

### Tier 4: Shutdown & Startup

| ID | Scenario | Impact | Recovery Expected |
|----|----------|--------|-------------------|
| **SS-1** | Graceful shutdown (SIGTERM) | Clean exit | Flush TSDB, close MQTT, close DB in correct order |
| **SS-2** | Hard kill (SIGKILL) | Immediate death | SQLite WAL recovers on restart, no corruption |
| **SS-3** | Startup with MQTT down | Can't connect to broker | Retry with backoff, start API server anyway |
| **SS-4** | Startup with corrupt config | Can't parse config | Clear error message, exit 1 |
| **SS-5** | Startup with locked database | Migration fails | Clear error, suggest checking for stale PID |

### Tier 5: Data Integrity

| ID | Scenario | Impact | Recovery Expected |
|----|----------|--------|-------------------|
| **DI-1** | Duplicate device IDs | Registry confusion | Rejected by unique constraint |
| **DI-2** | Orphaned references | Room deleted but devices reference it | FK constraints or application-level checks |
| **DI-3** | State history explosion | 100+ devices, 1 update/sec | Dedup + prune keeps table bounded |
| **DI-4** | TSDB batch lost on network error | Telemetry gap | Retry buffer (single retry) |

---

## Mode: Audit

Read the codebase and evaluate each failure scenario:

### For Each Scenario, Report:

```markdown
### {ID}: {Scenario Name}

**Protection exists:** Yes/No/Partial
**Code location:** `{file}:{line}` — {description of mechanism}
**Test exists:** Yes/No
**Recovery verified:** Yes/No

**Assessment:**
- What happens: {step-by-step failure sequence}
- What the code does: {actual recovery behaviour}
- Gap: {what's missing, if anything}
- Risk: {Low/Medium/High — likelihood × impact}

**Recommendation:** {Specific action or "Adequate"}
```

### Resilience Patterns to Look For

| Pattern | Where to Find It | What It Protects |
|---------|-------------------|-----------------|
| **Retry with backoff** | Connection loops (MQTT, KNX, knxd) | Transient failures |
| **Transaction rollback** | SQLite repos (tags, groups, zones, tokens) | Data consistency |
| **Bounded buffers** | TSDB batch, WebSocket send channel | Memory exhaustion |
| **Panic recovery** | `trySend()`, channel operations | Goroutine survival |
| **Context cancellation** | All long-running operations | Clean shutdown |
| **Mutex/RWMutex** | Device registry, WebSocket hub | Race conditions |
| **Health checks** | `IsConnected()` guards | Operating on dead connections |
| **Graceful degradation** | TSDB disabled mode, optional features | Partial availability |

---

## Mode: Test

When asked to test a specific scenario, write a Go test that:

1. Sets up the component under test
2. Injects the failure (mock, timeout, cancel context, close connection)
3. Verifies recovery behaviour
4. Checks for resource leaks (goroutines, file handles, memory)

### Test Naming Convention

```go
// Pattern: TestResilience_{Component}_{FailureScenario}
func TestResilience_MQTT_BrokerRestart(t *testing.T) { ... }
func TestResilience_TSDB_FlushFailureRetry(t *testing.T) { ... }
func TestResilience_WebSocket_SlowClient(t *testing.T) { ... }
func TestResilience_Registry_ConcurrentMutations(t *testing.T) { ... }
```

### Test Template

```go
func TestResilience_Component_Scenario(t *testing.T) {
    // 1. Setup
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    sut := setupComponent(t)

    // 2. Verify normal operation
    err := sut.DoNormalThing(ctx)
    require.NoError(t, err, "should work normally")

    // 3. Inject failure
    sut.InjectFailure() // e.g., close connection, return error from mock

    // 4. Verify degraded behaviour (not crash)
    err = sut.DoNormalThing(ctx)
    // Should either succeed (retry worked) or return a clean error (not panic)

    // 5. Remove failure
    sut.RemoveFailure()

    // 6. Verify recovery
    err = sut.DoNormalThing(ctx)
    require.NoError(t, err, "should recover after failure cleared")

    // 7. Check for resource leaks
    // (goroutine count, open connections, etc.)
}
```

---

## Mode: Report

Generate a resilience scorecard:

```markdown
## Resilience Scorecard — Gray Logic Core

**Date:** {date}
**Commit:** {hash}

### Overall Score: {X}/10

### By Category

| Category | Scenarios | Protected | Tested | Score |
|----------|-----------|-----------|--------|-------|
| Infrastructure (IF) | 5 | 4 | 2 | 7/10 |
| Protocol Bridge (PB) | 4 | 4 | 3 | 8/10 |
| API/WebSocket (AW) | 4 | 3 | 2 | 6/10 |
| Shutdown/Startup (SS) | 5 | 4 | 1 | 5/10 |
| Data Integrity (DI) | 4 | 3 | 2 | 6/10 |

### Critical Gaps (No Protection)
{List scenarios with no protection mechanism}

### Untested Protections (Protection exists but no test)
{List scenarios where code exists but isn't tested}

### Fully Verified (Protection + Test)
{List scenarios that are both protected and tested}

### Recommendations
1. {Highest priority gap to close}
2. {Next priority}
3. {Next priority}
```

---

## Integration with Other Commands

- `/chaos audit` produces the resilience scorecard that feeds into `/milestone-audit`
- `/chaos test {scenario}` writes tests that run as part of `go test -race ./...`
- `/stability` reviews code quality of resilience patterns; `/chaos` verifies they actually work
- Resilience tests use the `TestResilience_` prefix for easy filtering:
  ```bash
  go test -run TestResilience -race ./...
  ```

## Target

$ARGUMENTS
