# DALI-2 Bridge Plan (Draft for Implementation)

Last updated: 2026-02-03

This plan captures the structure, interfaces, constraints, and staged implementation steps for a DALI-2 bridge in Gray Logic, aligned to existing KNX bridge patterns and Modbus bridge architecture notes.

---

## 1) Mandatory Constraints (Do Not Break)

From `docs/CONSTRAINTS.md` and DALI protocol spec:
- Physical controls always work; bridge only augments/observes.
- Life safety is read-only; never control safety systems.
- No cloud dependency for core; bridge operates offline.
- Open standards at field layer; DALI-2 only, no proprietary-only devices.
- Privacy by design; no external telemetry beyond local network.
- Multi-decade stability; avoid bleeding-edge libs, pin versions.
- Resilience order: remote → automation → manual (manual never degrades).
- MQTT bridge contract: LWT, health every 30s retained, QoS 1, immediate ack.
- DALI security: no native security; isolate gateway on control VLAN only.

---

## 2) Repository Observations That Shape the Design

- Only KNX bridge exists in code: `code/core/internal/bridges/knx`.
- KNX bridge uses the flat MQTT topic scheme: `graylogic/command/knx/{address}` etc.
- Infrastructure MQTT helpers use the same flat scheme: `graylogic/{category}/{protocol}/{address}`.
- DALI spec document uses the bridge scheme (per `docs/protocols/dali.md`).
- Config system includes `DALIConfig` stub in `internal/infrastructure/config`.
- Device registry provides a standard protocol address map (example DALI address fields listed).

**Decision required before implementation:** unify or reconcile MQTT topic scheme (see Section 4).

---

## 3) Proposed File Structure (mirrors `internal/bridges/knx`)

```
code/core/internal/bridges/dali/
  AGENTS.md            # DALI-2 bridge guidance + spec checklist
  doc.go               # package docs
  config.go            # YAML config + validation + env overrides
  bridge.go            # orchestrator (MQTT <-> DALI gateway)
  messages.go          # MQTT message structs + topic helpers
  health.go            # health reporter (LWT, status, metrics)
  errors.go            # domain errors
  levels.go            # DALI level <-> % conversions
  fadetime.go          # transition ms -> DALI fade time code
  colour.go            # kelvin <-> mirek, DT8 helpers
  gateway.go           # gateway client interface + adapters
  state.go             # state normalisation + diffing
  (tests)
    *_test.go          # unit tests per file
```

Additional (outside package):
- `code/core/configs/dali-bridge.yaml` (bridge configuration template).
- Optional: `code/core/docs/technical/packages/dali-bridge.md` (mirror knx-bridge doc).

---

## 4) MQTT Topic Scheme — Open Decision

Current sources conflict:
- KNX bridge code: `graylogic/command/knx/{address}`.
- MQTT helpers (`internal/infrastructure/mqtt/topics.go`): `graylogic/command/{protocol}/{device_id}`.
- DALI spec: `graylogic/command/dali/{address}` etc.

**Resolved:** All bridges use the flat topic scheme `graylogic/{category}/{protocol}/{address}` as defined in `internal/infrastructure/mqtt/topics.go` and `docs/protocols/mqtt.md`. KNX bridge already follows this pattern. DALI bridge should use the same scheme (e.g. `graylogic/command/dali/{address}`, `graylogic/state/dali/{address}`, `graylogic/health/dali`).

---

## 5) Interfaces Needed (No Code Yet)

### MQTT Client Interface (same shape as KNX bridge)
- `Publish(topic string, payload []byte, qos byte, retained bool) error`
- `Subscribe(topic string, qos byte, handler func(topic string, payload []byte)) error`
- `IsConnected() bool`
- `Disconnect(quiesce uint)`

### Device Registry Interface (same concept as KNX bridge)
- `SetDeviceState(ctx, id, state map[string]any) error`
- `SetDeviceHealth(ctx, id, status string) error`
- `CreateDeviceIfNotExists(ctx, seed DeviceSeed) error`
- `GetDALIDevices(ctx) ([]RegistryDevice, error)` (if needed to merge registry mappings)

### Gateway Client Interface
Support both polling and event-driven gateways:
- `Connect(ctx) error`
- `Close() error`
- `IsConnected() bool`
- `SendCommand(ctx, target, command, params) error`
- `QueryStatus(ctx, target) (Status, error)`
- `SubscribeEvents(ctx, handler func(Event)) (cancel func(), err error)` (optional)

### Translator Helpers
- DALI level ↔ percent conversions (log scale)
- transition ms ↔ fade time code (nearest)
- kelvin ↔ mirek conversion for DT8
- device type mapping (DALI type -> Gray Logic device type)

---

## 6) Configuration Model (Derived from DALI Spec)

Based on `docs/protocols/dali.md`:
- Bridge:
  - id, health interval, mqtt broker + creds
- Gateways: list, each with
  - id, type (tridonic/helvar/other), host, port, buses
- Devices:
  - device_id, gateway, bus, short address (0-63)
  - device_type (DALI DT), Gray Logic type, features
  - optional CCT range (mirek), RGB capabilities
- Groups:
  - group_id, gateway, bus, dali_group (0-15), devices[]

Plan: place a `dali-bridge.yaml` template under `code/core/configs/`.

---

## 7) Bridge Behaviour (High-Level)

### Startup
1. Load config; validate gateway/device/group mappings.
2. Connect MQTT, set LWT.
3. Connect gateway(s).
4. Build mapping indices: device→address, address→device, group maps.
5. Seed device registry (if enabled).
6. Subscribe to MQTT command/config/request topics.
7. Publish health: `starting` → `healthy`.

### Commands
- Validate device/group target.
- Convert parameters:
  - brightness % -> DALI level (log curve)
  - transition_ms -> fade time code
  - Kelvin -> mirek (DT8)
- Send command to gateway.
- Ack immediately to Core.
- On timeout/failure, send follow‑up ack and update device health if needed.

### State Updates
- If gateway supports events, subscribe and translate events.
- Otherwise poll status (query actual level, lamp failure) at intervals.
- Publish state updates only on change (diff cache).
- Update device registry state and health.

### Health Reporting
- Publish retained health every 30 seconds.
- Status transitions: starting → healthy → degraded/unhealthy → offline.

---

## 8) Testing Plan (Coverage + Tiers)

- Unit tests for conversions:
  - DALI level <-> % (roundtrip tolerances)
  - fade time mapping
  - Kelvin <-> mirek conversions
- Message/topic tests:
  - topic builders, payload encoding
- Config validation tests:
  - invalid addresses, missing gateway
- Bridge tests:
  - command validation, ack emission, error paths
- Integration tests (tagged):
  - mock gateway that simulates status queries and events

Target coverage: 80%+ new code, 100% for error paths (per `docs/CONSTRAINTS.md`).

---

## 9) Open Questions / Decisions Needed

1. **MQTT topic scheme**: bridge-style topics vs legacy KNX-style topics.
2. **Gateway support**: which gateway types to implement first (Modbus TCP vs REST vs MQTT)?
3. **Registry integration**: should bridge ingest device mappings from registry on startup (like KNX)?
4. **Polling cadence**: default intervals for status/lamp failure checks.

---

## 10) Step-by-Step Implementation Plan (Sequenced)

1. Finalise MQTT topic scheme (document decision).
2. Create `internal/bridges/dali/` structure + `doc.go`, `errors.go`.
3. Implement config loading + validation + sample config file.
4. Implement message structs + topic helpers (based on chosen scheme).
5. Implement conversion helpers (levels, fade time, mirek).
6. Implement gateway interface + mock gateway (test harness).
7. Implement bridge orchestrator (subscribe, ack, state cache, health).
8. Integrate in `cmd/graylogic/main.go` (startDALIBridge analog to KNX).
9. Add tests (unit + integration scaffolding).
10. Update docs (`docs/protocols/dali.md`/package docs if needed).

---

## 11) Files to Reference During Implementation

- `docs/CONSTRAINTS.md`
- `docs/architecture/bridge-interface.md`
- `docs/protocols/dali.md`
- `docs/protocols/mqtt.md`
- `code/core/internal/bridges/knx/*`
- `code/core/internal/infrastructure/mqtt/*`
- `code/core/internal/device/*`
- `code/core/internal/infrastructure/config/config.go`

