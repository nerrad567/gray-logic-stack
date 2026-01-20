---
description: Add a new entity to the Gray Logic data model
---

# Add Entity

Adds a new entity type to the Gray Logic data model following established patterns.

## Arguments

- `$ARGUMENTS` — Entity name in PascalCase (e.g., `AudioZone`, `WeatherSource`)

## Steps

1. **Read existing patterns**
   - Review `docs/data-model/entities.md` for structure
   - Check `docs/data-model/schemas/` for JSON Schema examples

2. **Add entity definition**
   - Add to `docs/data-model/entities.md` following existing format
   - Include: purpose, fields, relationships, state diagram if applicable

3. **Create JSON Schema**
   - Create `docs/data-model/schemas/$ARGUMENTS.schema.json`
   - Follow patterns from existing schemas (especially `common.schema.json`)
   - Include required fields, enums, validation rules

4. **Update glossary**
   - Add any new terminology to `docs/overview/glossary.md`

5. **Check principles**
   - Review against `docs/overview/principles.md`
   - Ensure no boundary violations

6. **Cross-reference**
   - Update any related domain specs in `docs/domains/`
   - Link from relevant protocol specs if applicable

## Entity Template

```markdown
### EntityName

**Purpose**: Brief description of what this entity represents.

**Fields**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | UUID | Yes | Unique identifier |
| name | string | Yes | Human-readable name |
| ... | ... | ... | ... |

**Relationships**:
- Belongs to: [Parent entity]
- Has many: [Child entities]

**State Model** (if applicable):
[State diagram or description]
```

## Reference Documents

- `docs/data-model/entities.md` — Existing entities
- `docs/data-model/schemas/common.schema.json` — Shared types
- `docs/overview/glossary.md` — Standard terminology
