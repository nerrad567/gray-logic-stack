---
description: Explain recently written code with intuitive diagrams and flows
---

# Explain Code — Understanding What Was Built

This command helps you understand code that was recently written or any code in the codebase.

## What This Command Does

When you run `/explain-code`, Claude will:

1. **Identify recent changes** (or ask what you want explained)
2. **Explain the purpose** — What problem does it solve?
3. **Show the architecture** — ASCII diagrams of components
4. **Trace the data flows** — Step-by-step what happens
5. **Explain key decisions** — Why was it built this way?

---

## How to Use

### Explain Recent Work
```
/explain-code
```
Claude will explain the most recent significant code written in this session.

### Explain Specific Code
```
/explain-code knx bridge
/explain-code mqtt client
/explain-code database migrations
```

### Explain a Specific File
```
/explain-code internal/bridges/knx/knxd.go
```

---

## Output Format

Claude should provide:

### 1. One-Line Summary
> "The KNX Bridge translates between physical KNX devices and Gray Logic Core via MQTT."

### 2. Architecture Diagram (ASCII)
```
┌─────────┐     ┌─────────┐     ┌─────────┐
│ Component A │────►│ Component B │────►│ Component C │
└─────────┘     └─────────┘     └─────────┘
```

### 3. Key Concepts
| Concept | What It Is | Example |
|---------|------------|---------|
| Group Address | Device identifier | 1/2/3 |

### 4. Data Flows
Show step-by-step what happens for the main use cases:
```
1. User does X
2. Component A receives...
3. Component B transforms...
4. Result: Y happens
```

### 5. Code Highlights
Point to specific files/functions that are most important:
- `knxd.go:Connect()` — Establishes connection to knxd daemon
- `telegram.go:ParseTelegram()` — Decodes KNX bus messages

### 6. Design Decisions
Explain *why* certain choices were made:
- Why separate binary? → Isolation, independent restarts
- Why TCP not Unix socket default? → Easier Docker deployment

---

## Example Output

When asked `/explain-code knx bridge`:

```markdown
## KNX Bridge — What It Does

**One-liner**: Translates between physical KNX building devices and Gray Logic Core.

### Architecture

    Physical World          Software World
    ─────────────          ──────────────

    [Wall Switch]              [Flutter App]
         │                          │
    [KNX Bus]                  [Core REST API]
         │                          │
    [KNX/IP Gateway]           [MQTT Broker]
         │                          │
    [knxd daemon] ◄────────► [KNX Bridge] ◄────────► [Gray Logic Core]
         ▲                                                   │
         │                                                   │
         └───────────────────────────────────────────────────┘
                        Bidirectional communication

### Key Files

| File | Purpose |
|------|---------|
| `telegram.go` | Parse/encode KNX messages |
| `knxd.go` | Connection to knxd daemon |
| `bridge.go` | Main orchestration logic |

### Main Flow: Light Switch Press

1. Physical switch pressed → KNX telegram sent
2. knxd receives → forwards to our bridge
3. Bridge parses telegram → looks up device mapping
4. Bridge publishes MQTT → Core receives
5. Core updates state → UI refreshes

### Why This Design?

- **Separate binary**: Can restart bridge without Core restart
- **MQTT between bridge/Core**: Decoupled, debuggable, standard protocol
- **Device mappings in YAML**: Human-readable, version-controllable
```

---

## Tips for Good Explanations

1. **Start with WHY** — What problem does this solve?
2. **Use diagrams** — ASCII art is fine, visual > text
3. **Trace real scenarios** — "When user presses switch, X happens..."
4. **Point to code** — Reference specific files and line numbers
5. **Explain trade-offs** — Why this approach vs alternatives?

---

## Related Commands

- `/code-audit` — Verify code quality
- `/health-check` — Check development environment
- `/pre-commit` — Quality gates before committing
