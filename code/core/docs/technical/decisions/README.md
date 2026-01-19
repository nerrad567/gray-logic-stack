# Implementation Decision Records

> IMP-ADRs documenting significant implementation-level decisions

These records capture **how** we implement features, complementing the high-level Architecture Decision Records (ADRs) in `docs/architecture/decisions/` that document **what** technologies we use.

---

## Index

| ID | Title | Package | Status |
|----|-------|---------|--------|
| [IMP-001](IMP-001-embedded-migrations.md) | Embedded Migrations with embed.FS | database | Accepted |
| [IMP-002](IMP-002-wal-mode.md) | WAL Mode for SQLite | database | Accepted |
| [IMP-003](IMP-003-subscription-tracking.md) | Subscription Tracking for Reconnect | mqtt | Accepted |

---

## Template

When adding a new IMP-ADR, use this structure:

```markdown
# IMP-NNN: Title

## Status
Accepted | Proposed | Deprecated

## Date
YYYY-MM-DD

## Context
What problem are we solving? What constraints exist?

## Decision
What did we decide to do?

## Implementation
How was it implemented? Include code references.

## Consequences

### Advantages
- What becomes easier

### Disadvantages  
- What becomes harder

### Risks
- What could go wrong

## Alternatives Considered
What else did we consider and why didn't we choose it?
```

---

## Related Documents

- [docs/architecture/decisions/](../../../../../../docs/architecture/decisions/) — High-level ADRs
