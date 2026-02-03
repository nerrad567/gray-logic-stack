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
- DALI spec document uses the same flat scheme (per `docs/protocols/dali.md`).
- Config system includes `DALIConfig` stub in `internal/infrastructure/config`.
- Device registry provides a standard protocol address map (example DALI address fields listed).

**MQTT topic scheme is resolved:** all bridges use the flat scheme `graylogic/{category}/{protocol}/{address}` (see `docs/protocols/mqtt.md` and `internal/infrastructure/mqtt/topics.go`).

---

## 3) Proposed File Structure (mirrors `internal/bridges/knx`)

```
code/core/internal/bridges/dali/
  AGENTS.md            # DALI-2 bridge guidance + spec checklist
  doc.go               # package docs
  config.go            # YAML config + validation + env overrides
  bridge.go            # orchestrator (MQTT <-> DALI gateway)
  messages.go          # MQTT message structs + topic helpers (follow knx/messages.go pattern)
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

## 4) Interfaces Needed (No Code Yet)

### MQTT Client Interface (same shape as KNX bridge)
- `Publish(topic string, payload []byte, qos byte, retained bool) error`
- `Subscribe(topic string, qos byte, handler func(topic string, payload []byte)) error`
- `IsConnected() bool`
- `Disconnect(quiesce uint)`

Topics use the flat scheme `graylogic/{category}/dali/{address}`:
- `graylogic/command/dali/{address}` — commands from Core
- `graylogic/state/dali/{address}` — state updates to Core
- `graylogic/ack/dali/{address}` — command acknowledgements
- `graylogic/health/dali` — bridge health (retained, LWT)
- `graylogic/discovery/dali` — device discovery
- `graylogic/config/dali` — bridge configuration from Core
- `graylogic/request/dali/{request_id}` — ad-hoc requests from Core
- `graylogic/response/dali/{request_id}` — responses to requests

Use `mqtt.Topics{}` helpers (e.g. `BridgeCommand("dali", addr)`) — never construct topics manually.

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

## 5) Configuration Model (Derived from DALI Spec)

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

## 6) Bridge Behaviour (High-Level)

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

## 7) Testing Plan (Coverage + Tiers)

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

## 8) Open Questions / Decisions Needed

1. **Gateway support**: which gateway types to implement first (Modbus TCP vs REST vs MQTT)?
2. **Registry integration**: should bridge ingest device mappings from registry on startup (like KNX)?
3. **Polling cadence**: default intervals for status/lamp failure checks.

---

## 9) Step-by-Step Implementation Plan (Sequenced)

1. Create `internal/bridges/dali/` structure + `doc.go`, `errors.go`.
2. Implement config loading + validation + sample config file.
3. Implement message structs + topic helpers (follow `knx/messages.go` pattern, use `mqtt.Topics{}` helpers).
4. Implement conversion helpers (levels, fade time, mirek).
5. Implement gateway interface + mock gateway (test harness).
6. Implement bridge orchestrator (subscribe, ack, state cache, health).
7. Integrate in `cmd/graylogic/main.go` (startDALIBridge analog to KNX).
8. Add tests (unit + integration scaffolding).
9. Update docs (`docs/protocols/dali.md`/package docs if needed).

---

## 10) Files to Reference During Implementation

- `docs/CONSTRAINTS.md` — hard rules that apply to all bridges
- `docs/architecture/bridge-interface.md` — bridge contract spec
- `docs/protocols/dali.md` — DALI-2 protocol spec and examples
- `docs/protocols/mqtt.md` — unified flat topic scheme reference
- `code/core/internal/bridges/knx/*` — primary reference implementation
- `code/core/internal/bridges/knx/messages.go` — topic helpers and message structs to mirror
- `code/core/internal/infrastructure/mqtt/topics.go` — `Topics{}` builder methods (use these, don't hardcode topics)
- `code/core/internal/infrastructure/mqtt/client.go` — MQTT client interface
- `code/core/internal/device/*` — device registry, types, validation
- `code/core/internal/infrastructure/config/config.go` — config system with `DALIConfig` stub

