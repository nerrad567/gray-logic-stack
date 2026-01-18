---
title: Testing Strategy & Virtual Site Simulator
version: 1.0.0
status: active
last_updated: 2026-01-17
depends_on:
  - architecture/bridge-interface.md
  - development/DEVELOPMENT-STRATEGY.md
---

# Testing Strategy & Virtual Site Simulator

This document defines how we ensure the stability and correctness of Gray Logic without relying on physical hardware labs for every test case.

---

## The Philosophy

> "If it isn't tested, it doesn't work."
> "If it requires hardware to test, it won't get tested enough."

We prioritize **Unit Tests** for logic and **Integration Tests with Simulation** for system behavior.

---

## The Virtual Site Simulator

To validate complex automation (e.g., "Fire Alarm opens blinds"), we cannot rely on physical sensors. We need a **Virtual Site**.

### Architecture

The Virtual Site is a specialized Protocol Bridge (`bridge-sim`) that mimics the behavior of real devices.

```
┌─────────────────┐       MQTT        ┌──────────────────┐
│ Gray Logic Core │ <------------->   │ Simulated Bridge │
└─────────────────┘                   └────────┬─────────┘
                                               │
                                      ┌────────▼─────────┐
                                      │  Virtual State   │
                                      │    (In-Memory)   │
                                      └──────────────────┘
```

### Capabilities

The Simulator can:
1.  **Mock Devices:** Pretend to be 50 KNX lights, 10 Modbus pumps, and a Security Panel.
2.  **Inject Events:** Publish "Motion Detected" or "Temperature = 25.0" messages to MQTT.
3.  **Verify Commands:** Listen for commands (e.g., "Turn Light On") and assert they were received.
4.  **Simulate Physics (Optional):** "If Valve Open > 50%, increase Temperature by 0.1°C per minute."

### Usage in Integration Tests

We use Go's testing framework to spin up a transient Core and Simulator.

```go
func TestFireAlarmOpensBlinds(t *testing.T) {
    // 1. Setup
    env := testenv.New(t)
    defer env.Teardown()
    
    // 2. Define Virtual Site
    env.Simulator.AddDevice("sensor-fire", "contact_sensor")
    env.Simulator.AddDevice("blind-bedroom", "blind_position")
    
    // 3. Define Automation Rule
    env.Core.LoadRule("If fire detected, open blinds")
    
    // 4. Trigger Event
    env.Simulator.SetState("sensor-fire", "state", true)
    
    // 5. Assert Outcome
    env.Simulator.AssertStateEventually("blind-bedroom", "position", 0, 2*time.Second)
}
```

---

## Testing Pyramid

### 1. Unit Tests (Go)
- **Scope:** Individual functions, structs, logic engines.
- **Deps:** None (mocked interfaces).
- **Speed:** < 10ms.
- **Coverage Goal:** 80%+.
- **Example:** Testing the `SunPosition(lat, lon, time)` algorithm.

### 2. Integration Tests (Simulated)
- **Scope:** Core + SQLite + MQTT + Simulator.
- **Deps:** Docker (for Mosquitto/InfluxDB) or embedded replacements.
- **Speed:** < 5s.
- **Coverage Goal:** Critical paths (Scenes, Scheduler, Rules).
- **Example:** "Verify 'Good Night' scene turns off all simulated lights."

### 3. End-to-End Tests (Black Box)
- **Scope:** Full binary running via API.
- **Deps:** Full Docker stack.
- **Tooling:** Cypress or Playwright against the Web Admin.
- **Example:** "User logs in, clicks 'Living Room', toggles light, verifies icon updates."

---

## Continuous Integration (CI)

Every PR triggers:
1.  `go test ./...` (Unit Tests).
2.  `docker-compose up -d` (Infrastructure).
3.  `go test -tags=integration ./...` (Runs tests requiring MQTT).
4.  `golangci-lint run`.

---

## Manual Hardware Validation (Soak Testing)

While simulation catches logic bugs, it misses timing/driver issues.

**The "Lab Bench":**
A physical board containing:
- 1x KNX Interface + Actuator + Switch.
- 1x Modbus TCP Slave (simulated or PLC).
- 1x DALI Gateway + LED Driver.
- 1x Industrial PC (Core).

**Soak Test:**
- Run the "Lab Bench" 24/7.
- Randomly toggle devices every minute.
- Monitor for memory leaks, panic restarts, or missed ACKs.
