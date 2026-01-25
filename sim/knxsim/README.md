# KNXSim — KNX/IP Simulator

A full-featured KNX/IP tunnelling server that simulates real KNX installations for development and testing.

## Quick Start

```bash
# Start the simulator
docker compose -f docker-compose.dev.yml up -d knxsim

# Check health
curl http://localhost:9090/api/v1/health

# Open Engineer UI
open http://localhost:9090/ui/
```

## What It Does

KNXSim acts as a **KNXnet/IP tunnelling server** on UDP port 3671. Connect your KNX client (knxd, ETS, Gray Logic Core) and interact with virtual devices that behave like real hardware.

**Virtual Devices:**
- Light switches and dimmers
- Blinds with position and slat control
- Temperature/humidity sensors
- Presence detectors with motion and lux

**Realistic Behaviour:**
- Devices respond to GroupRead requests
- Status telegrams broadcast on state changes
- Scenarios run autonomously (temperature drift, presence patterns)
- State persists across restarts

## Engineer UI

Access the web dashboard at `http://localhost:9090/ui/`:

- **Device List** — All devices with live state
- **Telegram Inspector** — Real-time bus traffic
- **Device Panel** — Click any device to:
  - View individual address and group addresses
  - See DPT types for each GA
  - Control the device (toggles, sliders)
  - Inspect raw state JSON

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/premises` | List premises |
| GET | `/api/v1/premises/{id}/devices` | List devices with state |
| POST | `/api/v1/premises/{id}/devices/{did}/command` | Send command |
| GET | `/api/v1/templates` | Device templates |
| WS | `/ws/telegrams?premise={id}` | Live telegram stream |
| WS | `/ws/state?premise={id}` | Live state updates |

### Command Examples

```bash
# Turn on a light
curl -X POST http://localhost:9090/api/v1/premises/default/devices/living-room-ceiling-light/command \
  -H "Content-Type: application/json" \
  -d '{"command": "switch", "value": true}'

# Set brightness to 75%
curl -X POST http://localhost:9090/api/v1/premises/default/devices/living-room-ceiling-light/command \
  -H "Content-Type: application/json" \
  -d '{"command": "brightness", "value": 75}'

# Trigger motion detection
curl -X POST http://localhost:9090/api/v1/premises/default/devices/living-room-presence/command \
  -H "Content-Type: application/json" \
  -d '{"command": "presence", "value": true}'

# Set lux level
curl -X POST http://localhost:9090/api/v1/premises/default/devices/living-room-presence/command \
  -H "Content-Type: application/json" \
  -d '{"command": "lux", "value": 850}'
```

## Configuration

Default devices are defined in `config.yaml`. Add your own:

```yaml
premises:
  default:
    knx_port: 3671
    devices:
      - id: my-dimmer
        type: light_dimmer
        individual_address: "1.1.10"
        group_addresses:
          switch_cmd: "1/0/0"
          switch_status: "1/0/1"
          brightness_cmd: "1/0/2"
          brightness_status: "1/0/3"
        initial_state:
          on: false
          brightness: 0
```

## Architecture

```
┌──────────────────────────────────────────────────┐
│              Web UI (localhost:9090/ui/)         │
│   Device List │ Telegram Inspector │ Controls    │
└──────────────────────────┬───────────────────────┘
                           │ REST + WebSocket
┌──────────────────────────▼───────────────────────┐
│              FastAPI Backend (Python)            │
│   Device CRUD │ State Management │ Persistence   │
└──────────────────────────┬───────────────────────┘
                           │
┌──────────────────────────▼───────────────────────┐
│           KNXnet/IP Server (UDP 3671)            │
│   Tunnelling Protocol │ Virtual Device Handlers  │
└──────────────────────────┬───────────────────────┘
                           │
            ┌──────────────┴──────────────┐
            │         KNX Clients          │
            │  knxd │ ETS │ Gray Logic    │
            └─────────────────────────────┘
```

## Development

```bash
# Run locally (without Docker)
cd sim/knxsim
pip install -r requirements.txt
python knxsim.py

# Rebuild container after changes
docker compose -f docker-compose.dev.yml up -d --build knxsim
```

## See Also

- [VISION.md](VISION.md) — Full roadmap and feature plans
- [Gray Logic Core](../../code/core/) — The automation system that connects to KNXSim
