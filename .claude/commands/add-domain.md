---
description: Scaffold a new domain specification
---

# Add Domain

Creates a new domain specification following Gray Logic documentation patterns.

## Arguments

- `$ARGUMENTS` — Domain name in lowercase (e.g., `pool`, `irrigation`, `garage`)

## What Gets Created

```
docs/domains/$ARGUMENTS.md
```

## Template Structure

The generated domain spec includes:

1. **Overview** — What this domain controls
2. **Device Types** — Supported devices in this domain
3. **State Model** — Device states and transitions
4. **Commands** — Available control commands
5. **Automation** — Scene and schedule integration
6. **PHM Integration** — Predictive health monitoring (if applicable)
7. **Commissioning** — Installation and setup checklist

## Example: `/add-domain garage`

Creates `docs/domains/garage.md`:

```markdown
# Garage Domain

## Overview

The Garage domain manages garage doors, vehicle presence detection, and associated automation.

## Device Types

| Type | Protocol | Capabilities |
|------|----------|--------------|
| `garage_door` | Modbus/KNX | open, close, stop, position |
| `vehicle_detector` | KNX | presence |
| `obstruction_sensor` | KNX | binary |

## State Model

### Garage Door

```
    ┌──────┐   open    ┌────────┐
    │CLOSED├──────────►│OPENING │
    └──┬───┘          └───┬────┘
       │                  │
   close│              arrived
       │                  │
    ┌──┴───┐   close   ┌──▼───┐
    │CLOSING│◄─────────┤ OPEN │
    └──────┘           └──────┘
```

## Commands

| Command | Parameters | Description |
|---------|------------|-------------|
| `open` | — | Open garage door |
| `close` | — | Close garage door |
| `stop` | — | Stop door movement |
| `toggle` | — | Toggle current state |

## Automation

### Triggers
- Vehicle arrives (via Bluetooth/ANPR)
- Departure mode activated
- Time-based (close at night)

### Safety
- Obstruction detection before close
- Auto-close timeout (configurable)

## PHM Integration

| Metric | Threshold | Alert |
|--------|-----------|-------|
| Door cycles | > 10,000 | Maintenance due |
| Motor current | +20% baseline | Bearing wear |
| Travel time | +30% baseline | Track obstruction |

## Commissioning

- [ ] Install door controller
- [ ] Configure KNX group addresses
- [ ] Test open/close/stop
- [ ] Configure safety sensors
- [ ] Set auto-close timeout
- [ ] Test vehicle detection (if applicable)
```

## Checklist After Creation

- [ ] Review generated spec for domain-specific details
- [ ] Add to `docs/domains/` index
- [ ] Update `docs/data-model/entities.md` if new device types
- [ ] Add device types to `docs/data-model/schemas/common.schema.json`
- [ ] Cross-reference from related docs (if applicable)

## Reference

- Existing domains: `docs/domains/lighting.md`, `docs/domains/climate.md`
- Device types: `docs/data-model/entities.md`
- Automation patterns: `docs/automation/automation.md`
