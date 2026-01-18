---
title: Architecture Decision Records
version: 1.0.0
status: active
last_updated: 2026-01-17
---

# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records — documents that capture important architectural decisions made during Gray Logic development, along with their context and consequences.

## Why ADRs?

Gray Logic is designed for a multi-decade deployment horizon. ADRs ensure:
- Future maintainers understand *why* decisions were made
- Decisions aren't revisited without understanding original context
- Trade-offs are documented for future reference
- AI assistants can provide informed recommendations

## Format

Each ADR follows this template:

```markdown
# ADR-NNN: Title

## Status
[Proposed | Accepted | Deprecated | Superseded by ADR-XXX]

## Context
What is the issue that we're seeing that motivates this decision?

## Decision
What is the change that we're proposing/accepting?

## Consequences
What becomes easier or more difficult because of this decision?
```

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [001](001-go-for-core.md) | Go for Core Implementation | Accepted |
| [002](002-mqtt-internal-bus.md) | MQTT as Internal Message Bus | Accepted |
| [003](003-sqlite-primary-database.md) | SQLite as Primary Database | Accepted |
| [004](004-additive-only-migrations.md) | Additive-Only Database Migrations | Accepted |

## Creating a New ADR

1. Copy the template from `_template.md`
2. Number sequentially (next available number)
3. Use present tense for proposed, past tense for accepted
4. Link related ADRs
5. Update this index
