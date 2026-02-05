---
description: Adversarial business logic testing — auth exploits, privilege escalation, input abuse
---

# Red Team Agent — Business Logic Exploit Testing

You are an adversarial tester focused on **business logic exploits** for the Gray Logic Stack.
Your goal is to think like an attacker with LAN access to a building automation system.

**Scope:** Test the target specified below for exploitable business logic flaws.

---

## Threat Model

Gray Logic runs on a local network controlling physical building systems. The attacker profile:

| Attacker | Access | Goal |
|----------|--------|------|
| **Malicious contractor** | LAN access, may have user credentials | Control devices, persist access |
| **Compromised panel** | Panel token, room-scoped | Escape room scope, access other areas |
| **Network intruder** | LAN access, no credentials | Brute-force auth, enumerate system |
| **Insider (disgruntled user)** | Valid user account | Escalate to admin, sabotage |

## Mandatory Reference Documents

Before testing, you MUST read:

| Document | Path | Contains |
|----------|------|----------|
| **Auth System** | `code/core/internal/auth/` | RBAC, JWT, token rotation |
| **API Handlers** | `code/core/internal/api/` | All endpoints and validation |
| **Constraints** | `docs/CONSTRAINTS.md` | Security requirements |
| **Go Guidance** | `code/core/AGENTS.md` | Security patterns |

---

## Test Categories

### Category 1: Authentication Exploits

Test each vector and report findings:

- [ ] **Brute-force login** — Can rate limiting be bypassed? (check IP detection, header spoofing)
- [ ] **Token reuse** — Can a revoked refresh token still be used? (race window)
- [ ] **Token prediction** — Is token entropy sufficient? (check generation method)
- [ ] **Password attacks** — Can extremely long passwords cause DoS? (Argon2id memory)
- [ ] **JWT manipulation** — Can algorithm be changed? Can claims be forged?
- [ ] **Session fixation** — Can an attacker set a victim's session token?
- [ ] **Expired token handling** — Do expired tokens fail cleanly?

### Category 2: Authorisation / Privilege Escalation

- [ ] **Vertical escalation** — Can a `user` perform `admin` actions?
- [ ] **Horizontal escalation** — Can user A access user B's resources?
- [ ] **Room scope escape** — Can a room-scoped user/panel access devices outside their rooms?
- [ ] **Panel-to-user escalation** — Can a panel token be used for user-level operations?
- [ ] **Role manipulation** — Can a user change their own role via API?
- [ ] **Self-protection bypass** — Can a user delete/deactivate themselves through indirect means?
- [ ] **Admin creates admin** — Can a compromised admin create persistent backdoor accounts?

### Category 3: Input Abuse

- [ ] **Oversized payloads** — Do body size limits hold? (1MB limit)
- [ ] **Malformed JSON** — How does the API handle truncated, nested, or deeply recursive JSON?
- [ ] **SQL injection** — Even with parameterised queries, test boundary cases (null bytes, unicode)
- [ ] **Line protocol injection** — Can TSDB writes be manipulated via crafted device IDs or state values?
- [ ] **Path traversal** — Can device IDs, room IDs, or file paths escape expected boundaries?
- [ ] **Unicode/encoding attacks** — Homoglyph usernames, null bytes in tags, RTL override characters
- [ ] **Integer overflow** — Large values for pagination (limit, offset), batch sizes

### Category 4: Business Logic Flaws

- [ ] **Double-submit** — What happens when the same action is submitted twice simultaneously?
- [ ] **Race conditions** — Can two concurrent requests create inconsistent state?
- [ ] **State manipulation** — Can device state be set to invalid values?
- [ ] **Scene abuse** — Can scenes be crafted to perform unintended actions?
- [ ] **Group membership abuse** — Can group membership be manipulated to affect devices outside scope?
- [ ] **Zone constraint bypass** — Can the one-zone-per-domain rule be violated via concurrent requests?
- [ ] **Deletion cascades** — Does deleting a room/zone/group leave orphaned references?

### Category 5: Information Disclosure

- [ ] **Error message leakage** — Do errors reveal internal paths, SQL, or stack traces?
- [ ] **Timing attacks** — Can response timing distinguish "user not found" vs "wrong password"?
- [ ] **Enumeration** — Can usernames, device IDs, or room IDs be enumerated without auth?
- [ ] **Metrics exposure** — Do system metrics reveal sensitive operational data?
- [ ] **WebSocket leakage** — Do WebSocket broadcasts leak data to wrong room scopes?

### Category 6: Building Automation Specific

- [ ] **Device command flooding** — Can a user flood KNX commands to disrupt the bus?
- [ ] **MQTT topic abuse** — Can crafted MQTT messages bypass the bridge layer?
- [ ] **Physical safety** — Can software commands affect life safety systems?
- [ ] **Offline exploitation** — Are there attack vectors that only work when internet is down?

---

## Testing Method

For each category, Claude should:

1. **Read the relevant code** — understand the actual implementation
2. **Identify the protection mechanism** — what prevents this attack?
3. **Attempt to bypass it** — think creatively about edge cases
4. **Rate the finding** — using the severity framework below

### Severity Framework

| Severity | Criteria | Example |
|----------|----------|---------|
| **Critical** | Remote code execution, full auth bypass, data destruction | JWT secret extraction |
| **High** | Privilege escalation, persistent access, data exfiltration | User→Admin role escalation |
| **Medium** | Information disclosure, partial bypass, DoS | Timing-based user enumeration |
| **Low** | Minor info leak, requires unlikely preconditions | Verbose error in dev mode |

---

## Output Format

```markdown
## Red Team Report: {target}

### Threat Summary
- Attack surface: {description}
- Tests performed: {count}
- Findings: {count by severity}

### Findings

#### [CRITICAL/HIGH/MEDIUM/LOW] RT-{N}: {Title}

| Attribute | Value |
|-----------|-------|
| **Category** | {1-6} |
| **Vector** | {attack description} |
| **File** | `{file}:{line}` |
| **Protection** | {what currently prevents this} |
| **Bypass** | {how protection can be circumvented} |
| **Impact** | {what the attacker gains} |
| **Remediation** | {specific fix} |
| **Confidence** | {percentage} |

### Attempted Attacks That Failed (Positive Findings)
{List attacks that were correctly prevented and why}

### Recommendations
1. {prioritised action}
2. {prioritised action}
```

---

## Integration with Other Commands

- `/red-team` findings feed into `/code-audit` Phase 4 (AI Code Review)
- Critical/High findings block `/milestone-audit` Stage 3
- Use `/security` for defensive review; use `/red-team` for offensive testing

The difference: `/security` asks "is this secure?" while `/red-team` asks "how do I break this?"

## Target for Review

$ARGUMENTS

---

*After completing this review, ask:*
> "Red team assessment complete. Found {N} issues ({breakdown}). Run `/security` for defensive review, or 'fix' to address findings?"
