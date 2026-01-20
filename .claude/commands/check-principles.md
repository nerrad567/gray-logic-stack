---
description: Review recent changes against Gray Logic critical boundaries
---

# Principles Check

Review changes against `docs/overview/principles.md` hard rules:

## Checklist

1. **Physical controls still work independently?**
   - Wall switches, buttons must function even if all software is down
   - Check: Does this change affect KNX/DALI direct bindings?

2. **Life safety untouched?**
   - Fire alarms, E-stops use certified hardware — we observe, never control
   - Check: Does this change attempt to control any safety system?

3. **Works offline?**
   - 99%+ functionality without internet
   - Check: Does this add any cloud API calls for core functionality?

4. **No cloud dependencies for core?**
   - Internet down = everything except remote access still works
   - Check: Will this feature break if there's no internet?

5. **Open standards maintained?**
   - KNX, DALI, Modbus at field layer
   - Check: Does this introduce proprietary protocol dependencies?

6. **No vendor lock-in?**
   - Customer owns their system, full documentation
   - Check: Can this component be replaced independently?

7. **Privacy by design?**
   - Voice processed locally, no cloud surveillance
   - Check: Does this send sensitive data externally?

## How to Use

Run this check before committing significant changes:

```bash
# Review principle violations in recent commits
git diff HEAD~5 --name-only | xargs grep -l -E "(cloud|api\.external|vendor-specific)"
```

Report any violations or concerns before proceeding.
