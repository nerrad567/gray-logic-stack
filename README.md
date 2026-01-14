# Gray Logic

**Gray Logic** is a complete building intelligence platform — the central nervous system for high-end homes, small estates, and light commercial buildings. It rivals systems like Crestron, Savant, and Loxone while maintaining complete openness, true offline capability, and zero vendor lock-in.

## Status

**v1.0 Architecture Phase** (January 2026) — Documentation complete, implementation starting.

Pivoted from openHAB-based approach to custom **Gray Logic Core** written in Go for 10-year deployment stability.

## Quick Links

| Document | Description |
|----------|-------------|
| [Vision](docs/overview/vision.md) | What we're building and why |
| [Principles](docs/overview/principles.md) | Hard rules that can never be broken |
| [System Overview](docs/architecture/system-overview.md) | Technical architecture |
| [Entities](docs/data-model/entities.md) | Core data model |
| [CHANGELOG](CHANGELOG.md) | Version history |
| [Business Case](docs/business/business-case.md) | Market positioning and strategy |

## Documentation Structure

```
docs/
├── overview/           # Vision, principles, glossary
├── architecture/       # System design, bridges, energy model
├── data-model/         # Entities, schemas
├── protocols/          # KNX, DALI, Modbus, MQTT, BACnet
├── domains/            # Lighting, climate, blinds, plant
├── integration/        # Access control, CCTV, fire alarm
├── intelligence/       # AI, voice, PHM, weather
├── resilience/         # Offline, backup, mesh comms
├── deployment/         # Installation guides
├── business/           # Business case, pricing, sales
└── archive/            # v0.4 openHAB-era docs (zip)
```

## Core Principles

1. **Physical controls always work** — Wall switches function even if all software is down
2. **Life safety is independent** — Fire alarms, E-stops are certified hardware; we observe, never control
3. **No cloud dependencies** — 99%+ functionality without internet
4. **10-year deployment horizon** — Systems installed today must work reliably in 2036
5. **Open standards** — KNX, DALI, Modbus; no proprietary lock-in

## Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Core | Go | Single binary, no runtime, cross-compiles |
| Database | SQLite | Embedded, zero maintenance |
| Time-Series | InfluxDB | PHM data, energy monitoring |
| Message Bus | MQTT | Simple, proven, debuggable |
| Wall Panel/Mobile | Flutter | Cross-platform native |
| Voice STT | Whisper | Local, accurate, open |
| Voice TTS | Piper | Local, natural |

## Roadmap

| Year | Focus | Deliverables |
|------|-------|--------------|
| 1 | Foundation | Core + KNX + lighting in own home |
| 2 | Expansion | Scenes, modes, blinds, climate, BACnet |
| 3 | Integration | Audio, video, security, CCTV |
| 4 | Intelligence | Voice control, PHM, AI |
| 5 | Product | Commissioning tools, first customer |

## Archive

Previous openHAB-based documentation (v0.4) preserved in:
- `docs/archive/v0.4-openhab-era.zip`
- `code/archive/v0.4-openhab-era.zip`

## License

GPL v3 License — See [LICENSE](LICENSE) for details.

