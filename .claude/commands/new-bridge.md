---
description: Create a new protocol bridge skeleton
---

# Create Protocol Bridge

Creates the scaffold for a new protocol bridge following Gray Logic patterns.

## Arguments

- `$ARGUMENTS` — Protocol name (e.g., `bacnet`, `zigbee`, `matter`)

## Structure to Create

```
code/bridges/$ARGUMENTS/
├── main.go           # Entry point, MQTT client, signal handling
├── config.go         # Configuration struct (YAML + env vars)
├── bridge.go         # Protocol interface implementation
├── doc.go            # Package documentation
└── README.md         # Protocol-specific documentation
```

## Template Requirements

1. **Follow existing patterns** — Reference `docs/protocols/knx.md` for structure
2. **MQTT topics** — Use `graylogic/{protocol}/...` namespace per `docs/protocols/mqtt.md`
3. **Offline-first** — Bridge must function without Core (queue messages)
4. **Configuration** — Support YAML config + environment variable overrides
5. **Graceful shutdown** — Handle SIGTERM/SIGINT for clean disconnect

## Checklist

- [ ] Create bridge directory structure
- [ ] Implement MQTT client connection
- [ ] Implement protocol-specific device discovery
- [ ] Add to `docs/architecture/system-overview.md` bridge list
- [ ] Create `docs/protocols/$ARGUMENTS.md` specification
- [ ] Update `docker-compose.dev.yml` if needed

## Reference Documents

- `docs/architecture/bridge-interface.md` — MQTT contract
- `docs/protocols/mqtt.md` — Topic structure
- `docs/development/CODING-STANDARDS.md` — Go conventions
