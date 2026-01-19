---
description: Documentation requirements when implementing or modifying packages
---

# Package Development Documentation Workflow

When implementing a new package or making significant changes to an existing package, follow this documentation checklist.

## Before Implementation

1. Review existing documentation:
   - `docs/technical/packages/` for related packages
   - `docs/technical/decisions/` for relevant IMP-ADRs
   - `IMPLEMENTATION.md` for session context

## During Implementation

1. Add comprehensive inline godoc comments:
   - Package-level `doc.go` with Purpose, Security, Performance, Usage
   - All exported types with field descriptions
   - All exported functions with Parameters, Returns, Examples

2. Note significant decisions as you code:
   - Alternative approaches considered
   - Why this approach was chosen
   - Trade-offs made

## After Implementation

### Required Documentation Updates

// turbo-all

1. **Create/Update Package Design Doc**
   ```bash
   # Path: docs/technical/packages/{package-name}.md
   ```
   Must include:
   - Purpose
   - Architecture (with diagram)
   - How It Works (Init → Ops → Shutdown)
   - Design Decisions table
   - Interactions (dependencies + dependents)
   - Error Handling
   - Thread Safety
   - Configuration
   - Testing
   - Known Limitations

2. **Create IMP-ADRs for significant decisions**
   ```bash
   # Path: docs/technical/decisions/IMP-NNN-{title}.md
   # Get next number from docs/technical/decisions/README.md
   ```
   Required for:
   - Non-obvious implementation choices
   - Trade-offs that future maintainers should understand
   - Patterns that deviate from typical approaches

3. **Update decisions/README.md index**
   Add new IMP-ADRs to the index table.

4. **Update technical/README.md index**
   Add new package to the Quick Navigation table.

5. **Update IMPLEMENTATION.md session log**
   Add session entry with:
   - Goal
   - Steps Taken
   - Dependencies Added
   - Files Created
   - Technical Decisions table
   - Issues Encountered table
   - Outcome

6. **Update IMPLEMENTATION.md "Resume Here" section**
   - Update last session number
   - Update milestone percentage
   - Set next task
   - Mark completed tasks as ✅ Done

## Verification

Before committing, verify:
- [ ] Package doc covers all 10 sections
- [ ] All code references in docs link to correct files
- [ ] IMP-ADRs are indexed
- [ ] IMPLEMENTATION.md is current
- [ ] `make lint` passes
- [ ] `make test` passes

## Commit Message Format

```
docs(core): add {package} technical documentation

- Add package design doc for {package}
- Add IMP-NNN: {decision title}
- Update IMPLEMENTATION.md with Session N
```
