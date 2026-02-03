# CLAUDE.md -- KNXSim

AI agent guidance for the KNXSim simulator codebase.

## HARD RULE: Real Hardware Replacement

KNXSim exists to replace real KNX hardware during development. You MUST be able to remove KNXSim, connect real KNX devices, and have the entire Gray Logic stack behave identically. If a behavior cannot exist on a real KNX bus, it MUST NOT exist in KNXSim.

This is not aspirational. It is a design constraint.

**Corollary**: Device mapping, implementation standards, telegram formats, DPT encoding, and ETS exports must all follow the KNX specification exactly. When in doubt, refer to the spec -- never invent behavior.

## Prerequisites (Read In Order)

1. `docs/protocols/knx-reference.md` -- KNX protocol specification (MANDATORY before any work)
2. `docs/CONSTRAINTS.md` -- Section 8.1: KNX-specific rules
3. `code/core/internal/bridges/knx/AGENTS.md` -- KNX bridge compliance checklist
4. `sim/knxsim/VISION.md` -- KNXSim roadmap and architecture
5. `sim/knxsim/README.md` -- Usage guide and API reference

## Architecture

```
FastAPI (9090)          KNXnet/IP UDP (3671)
   │                          │
   ├─ REST CRUD               ├─ CONNECT/DISCONNECT
   ├─ WebSocket streams       ├─ TUNNELLING_REQUEST/ACK
   └─ .knxproj export         └─ CONNECTION_STATE heartbeat
          │                          │
          └──── PremiseManager ──────┘
                     │
            ┌────────┼────────┐
            │        │        │
         Devices  Scenarios  Persistence
         (state    (thermal,  (SQLite)
          machines) presence)
```

- **Single Python process** managing multiple isolated premises
- **Pure Python KNXnet/IP** -- no external KNX libraries
- **SQLite persistence** -- state survives restarts
- **Scenario engine** -- background threads for realistic sensor simulation

## KNX Specification Compliance

All protocol implementation MUST comply with the KNX Standard v3.0.0:

| Area | Requirement | Reference |
|------|-------------|-----------|
| Telegram format | cEMI L_DATA frames with correct CTRL1/CTRL2 bytes | knx-reference.md sections 3-4 |
| DPT encoding | Exact bit-level encoding per KNX spec | knx-reference.md DPT sections |
| Individual addressing | Format `area.line.device` (e.g., `1.1.5`), unique per device | knx-reference.md section 2 |
| Group addressing | 3-level `main/middle/sub` (e.g., `1/2/3`) | knx-reference.md section 2 |
| APCI types | GroupRead (0x00), GroupResponse (0x40), GroupWrite (0x80) | knx-reference.md section 5 |
| Tunnelling protocol | Sequence numbers, ACK for every request, heartbeat keepalive | KNXnet/IP spec |
| Short vs long frames | 6-bit compact APCI for DPT 1.x; 2-byte header for larger payloads | knx-reference.md section 4 |

## Device Behavior Fidelity

Rules for all device implementations in `sim/knxsim/devices/`:

1. **GroupRead responses**: Every device MUST respond to GroupRead on its status GAs with the current state value. Real devices do this; KNXSim must too.
2. **Status telegrams on change**: When state changes (via command or scenario), devices MUST send unsolicited L_DATA.ind on their status GAs. This is how real actuators provide feedback.
3. **DPT accuracy**: Encoded values MUST match real device behavior exactly. Use the codec in `sim/knxsim/dpt/codec.py` and verify against KNX spec test vectors.
4. **No magic behaviors**: Devices MUST NOT exhibit behaviors impossible on a real KNX bus. No instant global state sync, no out-of-band communication, no responses without a request.
5. **Physical plausibility**: State transitions must be physically realistic. Blinds have travel time. Dimmers ramp. Valves don't teleport from 0% to 100%.
6. **Channel independence**: Multi-channel actuators MUST treat each channel independently. Commanding channel A must not affect channel B.
7. **GA_DPT_MAP**: Every device class MUST define a `GA_DPT_MAP` mapping GA function names to their DPT types. This is used for automatic encoding/decoding.

## ETS Export Quality

The `.knxproj` export (`sim/knxsim/api/routes_export.py`) is a critical interface -- it feeds the Gray Logic ETS import pipeline. Requirements:

1. **DPT correctness**: Exported `DatapointType` attributes MUST match the device's actual DPT. Use `DPST-X-Y` format (e.g., `DPST-9-001`).
2. **GA naming convention**: `"Device Name : Function Name"` format (e.g., `"Thermostat Living : Actual Temperature"`). The GLCore parser uses the ` : ` separator for device grouping.
3. **3-level GA hierarchy**: GroupRanges MUST be structured as `Domain > Floor > Room` to enable location extraction by the import pipeline.
4. **Building structure**: The `<Locations>` element MUST include `Building > Floor > Room` hierarchy matching the premise's floor/room structure.
5. **Parseable by GLCore**: The exported file MUST successfully parse via `POST /api/v1/commissioning/ets/parse`. If it doesn't, the export is broken.
6. **Function type mapping**: Use standard ETS Function Type names (`SwitchableLight`, `DimmableLight`, `Sunblind`, `HeatingRadiator`, `HeatingFloor`, etc.). Non-standard types use `Custom` with the template ID in the Comment attribute.

## Known Divergences from Real Hardware

These are documented limitations with technical reasoning. Each MUST include justification for why the divergence is acceptable and what the plan is to close the gap:

| Divergence | Reason | Impact | Plan |
|-----------|--------|--------|------|
| No bus timing (instant telegrams) | Python threading model; no 50ms inter-telegram gap | Low -- GLCore doesn't depend on timing | Future: add configurable delay |
| No telegram collision/retry | Single-threaded dispatch per premise | Low -- single-line topology has no collisions | Future: congestion simulation |
| Thermostat uses P-only control | Simpler to implement; PI/PID adds complexity | Medium -- real thermostats use PI or PID | Plan: upgrade to PID controller |
| No programming mode | ETS device commissioning not simulated | Medium -- can't test commissioning workflows | Future: add physical address assignment |
| No bus voltage simulation | Software-only; no electrical model | None -- GLCore never reads bus voltage | Not planned |
| No line/area coupler behavior | Single-line topology only | Low -- most homes are single-line | Future: multi-line support |
| No manufacturer/product IDs in export | No real device catalog database | Medium -- ETS won't show product details | Future: device catalog |

**When adding a new divergence**: Document it in this table with all four columns filled. If you cannot justify why the divergence is acceptable, do not introduce it.

## Development Standards

### Python conventions
- Type hints on all function signatures
- Docstrings on public classes and functions
- Use `dataclass` or plain `dict` for state (follow existing patterns per module)
- FastAPI routes in `sim/knxsim/api/routes_*.py`
- Device implementations in `sim/knxsim/devices/`

### Adding a new device type
1. Create device class in `sim/knxsim/devices/` extending `BaseDevice`
2. Define `GA_DPT_MAP` class variable mapping function names to DPTs
3. Implement `handle_group_write()` for incoming commands
4. Implement `handle_group_read()` for status reads
5. Create YAML template in `sim/knxsim/templates/<domain>/`
6. Add to `config.yaml` sample premise if appropriate
7. Verify the ETS export includes correct DPTs and naming

### Template format
Device templates live in `sim/knxsim/templates/` organized by domain (lighting, climate, blinds, sensors). Templates define:
- Channel structure (for multi-channel actuators)
- Default GA naming patterns
- DPT assignments per GA function
- Initial state values

## Common Commands

```bash
# Start KNXSim (via dev services)
cd code/core && make dev-services

# Check health
curl http://localhost:9090/api/v1/health

# List devices
curl http://localhost:9090/api/v1/premises/default/devices

# Export .knxproj
curl http://localhost:9090/api/v1/premises/default/export/knxproj -o test.knxproj

# Factory reset (reload from config.yaml)
curl -X POST http://localhost:9090/api/v1/premises/default/reset-sample

# View container logs
docker logs graylogic-knxsim --tail 50
```
