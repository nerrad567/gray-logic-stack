# Prompt for Audit Iteration 5 Agent

**Objective:** Performance of the "Iteration 5 Holistic Consistency Audit" on the Gray Logic Stack.

**Context:** 
We have completed Iteration 3 (Functional Gaps) and Iteration 4 (Long-term/Forensic). The repository is at a high level of maturity (Readiness ~8.8/10). However, Iteration 4 flagged several "false positives" because the auditor missed existing specifications in `docs/operations/maintenance.md`.

**Your Mission:**
Conduct a deep **Integration & Consistency Audit**. Do not look for missing features (we have enough). Look for **contradictions**, **race conditions**, and **logic gaps** *between* the documented systems.

**The "False Positive" Defense Protocol (Strict Rules):**
1.  **Map Before You Judge:** You MUST read `docs/operations/maintenance.md`, `docs/resilience/offline.md`, `docs/architecture/core-internals.md`, and `docs/architecture/security-model.md` BEFORE flagging any operational gap.
2.  **Grep Before Gripe:** If you believe a mechanism (e.g., "Cert Rotation") is missing, you MUST perform a `grep_search` for keywords (e.g., "cert", "rotation", "expiry") across the *entire* `docs/` directory. Only report it if Grep returns nothing.
3.  **Assume Scope Exists:** Assume standard features exist somewhere. Your job is to find where they are *broken* or *contradictory*.

**Specific Target Areas (The "Seams" between systems):**
1.  **Backup vs. Security:** Does the backup functionality defined in `backup.md` inadvertently backup standard secrets or private keys in plaintext? Does restoration require keys that might be lost?
2.  **Offline vs. Time:** `offline.md` mandates no internet. `maintenance.md` might assume NTP from internet. Check for deadlocks where a system can't boot or auth because it has no time source.
3.  **Energy vs. PHM:** `energy.md` tracks consumption. `phm.md` uses power for anomaly detection. Are they using the same data streams? Do the update intervals match?
4.  **Privacy vs. Debug:** `security-model.md` forbids logging PINs. Check `api.md` or `core-internals.md` logging specs to ensure DEBUG levels don't leak this.

**Output Format:**
Produce `docs/audit-iteration-5-log.md`.
For each finding, you must include a "Verification" section proving you checked for existing coverage.

**Start by mapping the `docs/` folder.**
