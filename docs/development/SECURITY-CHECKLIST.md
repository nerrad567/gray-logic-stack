---
title: Security Checklist
version: 1.0.0
status: active
last_updated: 2026-01-15
depends_on:
  - architecture/security-model.md
  - development/DEVELOPMENT-STRATEGY.md
  - overview/principles.md
---

# Gray Logic Security Checklist

This checklist must be completed for every component and every release.

---

## Purpose

Gray Logic controls critical building systems. A security breach could result in:
- Unauthorised access to the building (doors, gates)
- Surveillance of residents (cameras, microphones)
- Physical harm (heating/cooling sabotage, lighting manipulation)
- Privacy violations (occupancy data, behaviour patterns)

**This checklist is mandatory. Skipping checks is prohibited.**

---

## Hard Rules Verification

Before any security review, verify compliance with [principles.md](../overview/principles.md):

- [ ] **Physical controls still work** — Feature doesn't interfere with manual switches
- [ ] **Life safety independent** — No control of fire/emergency systems
- [ ] **Works offline** — Feature functions without internet
- [ ] **Privacy preserved** — Data processed locally by default

---

## Pre-Implementation Security Review

Complete **before** writing code for a new component.

### Threat Modeling

- [ ] **Attack surface documented**
  - What external inputs does this component accept?
  - What network interfaces does it expose?
  - What file system access does it require?

- [ ] **Trust boundaries identified**
  - What data comes from untrusted sources?
  - What privileges does this component run with?
  - What components does it trust implicitly?

- [ ] **Threats enumerated (STRIDE)**
  - **Spoofing:** Can an attacker impersonate a legitimate user/device?
  - **Tampering:** Can data be modified in transit or at rest?
  - **Repudiation:** Can an attacker deny their actions?
  - **Information Disclosure:** Can sensitive data be leaked?
  - **Denial of Service:** Can the component be overwhelmed or crashed?
  - **Elevation of Privilege:** Can a low-privilege user gain higher access?

- [ ] **Mitigations planned for each threat**
  - Document how each threat is addressed
  - If a threat is accepted (not mitigated), document rationale

### Security Requirements

- [ ] **Authentication required?**
  - If yes: What mechanism? (JWT, mTLS, API key)
  - If no: Justify why not (e.g., internal-only component)

- [ ] **Authorization required?**
  - What permissions are needed?
  - How are permissions checked?

- [ ] **Audit logging required?**
  - What actions should be logged?
  - What data should be included?

- [ ] **Encryption required?**
  - Data in transit: TLS? Which version?
  - Data at rest: What algorithm?

- [ ] **Rate limiting required?**
  - What operations can be abused?
  - What are the limits?

---

## Component Security Checklist

Complete **during development** and verify in **code review**.

### 1. Input Validation

- [ ] **All external inputs validated**
  - API requests (JSON, query params, headers)
  - MQTT messages
  - WebSocket messages
  - File uploads
  - Configuration files

- [ ] **Validation strategy documented**
  - Whitelist approach used (allowed values/patterns)
  - Reject invalid input, don't try to "fix" it

- [ ] **Length limits enforced**
  - Strings have maximum length
  - Arrays have maximum size
  - File uploads have size limits

- [ ] **Type safety enforced**
  - Numbers are validated for range
  - Enums are validated against allowed values
  - UUIDs are validated for format

- [ ] **Injection attacks prevented**
  - SQL injection: Parameterized queries only
  - Command injection: No shell execution with user input
  - Path traversal: Validate file paths, use `filepath.Clean()`
  - LDAP injection: Escape special characters
  - Template injection: Escape user content in templates

**Example validation:**
```go
func ValidateSceneName(name string) error {
    if name == "" {
        return errors.New("scene name required")
    }
    if len(name) > 50 {
        return errors.New("scene name too long (max 50)")
    }
    if !regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`).MatchString(name) {
        return errors.New("scene name contains invalid characters")
    }
    return nil
}
```

---

### 2. Authentication

- [ ] **Authentication implemented correctly**
  - Passwords never stored in plaintext
  - Use bcrypt, scrypt, or Argon2 for password hashing
  - Minimum password requirements enforced (12+ chars recommended)

- [ ] **JWT tokens secured (if used)**
  - Signed with strong secret (min 256 bits)
  - Secret stored securely (not in code)
  - `alg` header validated (prevent `none` algorithm attack)
  - Expiration (`exp`) enforced
  - Issuer (`iss`) and audience (`aud`) validated

- [ ] **Sessions secured**
  - Session IDs are cryptographically random
  - Session cookies have `HttpOnly`, `Secure`, `SameSite` flags
  - Session timeout enforced (e.g., 24 hours)
  - Logout properly invalidates session

- [ ] **Brute force protection**
  - Login attempts rate-limited
  - Account lockout after N failed attempts
  - CAPTCHA or delay on repeated failures

**Example JWT validation:**
```go
func ValidateJWT(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        // Prevent "none" algorithm attack
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return jwtSecret, nil
    })

    if err != nil {
        return nil, err
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token")
    }

    // Validate issuer
    if claims.Issuer != "gray-logic-core" {
        return nil, errors.New("invalid issuer")
    }

    return claims, nil
}
```

---

### 3. Authorization

- [ ] **Permissions checked before every action**
  - Check permissions at the entry point (API handler, not deep in business logic)
  - Use principle of least privilege

- [ ] **RBAC implemented** (see [security-model.md](../architecture/security-model.md))
  - Roles defined: Admin, Facility Manager, User, Guest
  - Permissions mapped to roles
  - Default deny (if no permission rule, deny)

- [ ] **Object-level authorization**
  - User can only access their own resources
  - Admin can access all resources
  - Check ownership before modification

- [ ] **Sensitive operations require confirmation**
  - Account deletion
  - Security setting changes
  - Remote arm/disarm (security systems)

**Example authorization:**
```go
func (s *SceneService) DeleteScene(ctx context.Context, sceneID uuid.UUID, userID uuid.UUID) error {
    // Load scene
    scene, err := s.repo.GetScene(ctx, sceneID)
    if err != nil {
        return err
    }

    // Check authorization: user must own the scene or be an admin
    user, err := s.authService.GetUser(ctx, userID)
    if err != nil {
        return err
    }

    if scene.OwnerID != userID && !user.IsAdmin {
        return ErrForbidden
    }

    // Delete scene
    return s.repo.DeleteScene(ctx, sceneID)
}
```

---

### 4. Cryptography

- [ ] **TLS configured correctly**
  - TLS 1.2 minimum (TLS 1.3 preferred)
  - Strong cipher suites only (no RC4, 3DES, MD5)
  - Certificate validation enforced (no `InsecureSkipVerify`)
  - Certificate expiration monitored

- [ ] **Secrets never hardcoded**
  - API keys from environment variables or vault
  - JWT secrets from configuration files (not in code)
  - Encryption keys from key management service

- [ ] **Random values are cryptographically secure**
  - Use `crypto/rand`, not `math/rand`
  - Session IDs, tokens, salts use `crypto/rand`

- [ ] **Sensitive data encrypted at rest**
  - Passwords hashed (bcrypt/Argon2)
  - API keys encrypted (AES-256)
  - Backups encrypted

- [ ] **Sensitive data encrypted in transit**
  - HTTPS for API
  - TLS for MQTT (if over internet)
  - WireGuard for VPN

**Example secure random generation:**
```go
func GenerateSessionID() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b), nil
}
```

---

### 5. Logging and Monitoring

- [ ] **Security events logged**
  - Login attempts (success and failure)
  - Permission denials
  - Configuration changes
  - Critical system actions (mode changes, scene activations)

- [ ] **Log entries include context**
  - Timestamp (UTC)
  - User ID or session ID
  - Action attempted
  - Outcome (success/failure)
  - IP address (if network request)

- [ ] **Sensitive data not logged**
  - Passwords never logged
  - Tokens not logged in plaintext
  - PII minimized (log user ID, not full name)

- [ ] **Logs protected from tampering**
  - Append-only log files
  - Log rotation configured
  - Logs backed up regularly

- [ ] **Alerting configured**
  - Alert on repeated login failures
  - Alert on permission denials
  - Alert on system errors

**Example security logging:**
```go
func (s *AuthService) Login(ctx context.Context, username, password string) (*User, error) {
    user, err := s.repo.GetUserByUsername(ctx, username)
    if err != nil {
        // Log failed login attempt
        log.WithFields(log.Fields{
            "event":    "login_failed",
            "username": username,
            "reason":   "user_not_found",
            "ip":       getClientIP(ctx),
        }).Warn("Login failed")
        return nil, ErrInvalidCredentials
    }

    if !s.checkPassword(password, user.PasswordHash) {
        // Log failed login attempt
        log.WithFields(log.Fields{
            "event":    "login_failed",
            "username": username,
            "user_id":  user.ID,
            "reason":   "invalid_password",
            "ip":       getClientIP(ctx),
        }).Warn("Login failed")
        return nil, ErrInvalidCredentials
    }

    // Log successful login
    log.WithFields(log.Fields{
        "event":   "login_success",
        "user_id": user.ID,
        "ip":      getClientIP(ctx),
    }).Info("Login successful")

    return user, nil
}
```

---

### 6. Error Handling

- [ ] **Errors don't leak sensitive information**
  - Generic error messages to users ("Authentication failed")
  - Detailed errors logged server-side
  - No stack traces in production responses

- [ ] **Errors handled consistently**
  - All errors logged
  - All errors returned to caller
  - No ignored errors (check `err != nil`)

- [ ] **Failures fail securely**
  - Deny access on error (don't allow by default)
  - Close connections on protocol errors
  - Invalidate sessions on auth errors

**Example error handling:**
```go
func (s *SceneService) ActivateScene(ctx context.Context, sceneID uuid.UUID, userID uuid.UUID) error {
    // Check authorization
    if err := s.checkPermission(ctx, userID, "scenes:activate"); err != nil {
        // Log detailed error server-side
        log.WithFields(log.Fields{
            "user_id":  userID,
            "scene_id": sceneID,
            "error":    err,
        }).Error("Permission denied for scene activation")

        // Return generic error to user
        return ErrForbidden // "You don't have permission to perform this action"
    }

    // Continue with scene activation...
}
```

---

### 7. Rate Limiting

- [ ] **Rate limits implemented**
  - API endpoints rate-limited
  - Login attempts rate-limited
  - Scene activations rate-limited (prevent DoS)

- [ ] **Rate limits appropriate**
  - Not too strict (blocks legitimate use)
  - Not too loose (allows abuse)

- [ ] **Rate limits per-user or per-IP**
  - Authenticated: per-user limits
  - Unauthenticated: per-IP limits

**Example rate limiting:**
```go
// Rate limiter: 10 requests per minute per user
var sceneActivationLimiter = rate.NewLimiter(rate.Every(6*time.Second), 10)

func (s *SceneService) ActivateScene(ctx context.Context, sceneID uuid.UUID, userID uuid.UUID) error {
    // Check rate limit
    if !sceneActivationLimiter.Allow() {
        log.WithFields(log.Fields{
            "user_id":  userID,
            "scene_id": sceneID,
        }).Warn("Rate limit exceeded for scene activation")
        return ErrRateLimitExceeded
    }

    // Continue with scene activation...
}
```

---

### 8. Dependency Security

- [ ] **Dependencies audited**
  - Run `go list -m all` to list dependencies
  - Check for known vulnerabilities (`govulncheck`)
  - Review dependency licenses

- [ ] **Dependencies pinned**
  - Use `go.mod` with exact versions
  - Vendor dependencies (`go mod vendor`)

- [ ] **Dependencies minimized**
  - Only include necessary dependencies
  - Avoid large frameworks

- [ ] **Transitive dependencies reviewed**
  - Run `go mod graph` to see full dependency tree
  - Check for suspicious dependencies

**Example vulnerability check:**
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

---

### 9. Network Security

- [ ] **Network segmentation enforced**
  - Control VLAN isolated from user VLAN
  - CCTV VLAN isolated from other networks
  - Firewall rules documented

- [ ] **Unnecessary ports closed**
  - Only expose required ports (e.g., 443 for HTTPS)
  - MQTT, database not exposed to internet

- [ ] **Services bound to correct interface**
  - Internal services on `127.0.0.1` or private IP
  - Public services on `0.0.0.0` (if needed)

**Example secure binding:**
```go
// API server: bind to all interfaces (but firewalled externally)
apiServer := &http.Server{
    Addr:    ":8443",
    Handler: apiHandler,
}

// MQTT broker: bind to localhost only (internal use)
mqttBroker := &mqtt.Server{
    Addr:    "127.0.0.1:1883",
    Handler: mqttHandler,
}
```

---

### 10. Data Privacy

- [ ] **PII minimized**
  - Collect only necessary data
  - Don't log full names, addresses unnecessarily

- [ ] **Data retention defined**
  - Logs rotated after 30 days
  - Old backups deleted after 1 year
  - User data deleted on account deletion

- [ ] **Local processing default** (from [principles.md](../overview/principles.md))
  - Voice commands processed locally
  - Video analytics local (no cloud upload)

- [ ] **User consent for cloud features**
  - External AI requires opt-in
  - Remote access requires explicit enable

---

## Code Review Security Checklist

Complete **during code review** before merging.

### General

- [ ] **Code compiles without warnings**
- [ ] **All tests pass**
- [ ] **Linter passes (golangci-lint with gosec)**
- [ ] **No commented-out code**
- [ ] **No TODO comments for security issues** (fix or create ticket)

### Security-Specific

- [ ] **Input validation present**
  - All external inputs validated
  - Validation logic is correct

- [ ] **No hardcoded secrets**
  - No passwords, API keys, tokens in code
  - Secrets loaded from environment or config

- [ ] **SQL queries parameterized**
  - No string concatenation for SQL
  - ORM used correctly (no raw queries with user input)

- [ ] **Authentication checked**
  - Protected endpoints require valid JWT/session
  - Unauthenticated access denied

- [ ] **Authorization checked**
  - User permissions verified
  - Object ownership verified

- [ ] **Error handling secure**
  - Errors logged, not exposed to user
  - Failures fail securely

- [ ] **Logging appropriate**
  - Security events logged
  - Sensitive data not logged

- [ ] **TLS configured correctly**
  - No `InsecureSkipVerify`
  - Certificate validation enabled

- [ ] **Random values use crypto/rand**
  - No `math/rand` for security-sensitive values

---

## Release Security Checklist

Complete **before every release** (Alpha, Beta, RC, GA).

### Pre-Release

- [ ] **All code merged**
  - All PRs merged and tested
  - No pending security fixes

- [ ] **Dependency audit**
  - `govulncheck` passes
  - No known vulnerabilities in dependencies
  - Dependencies up-to-date (security patches)

- [ ] **Secrets audit**
  - No secrets in code
  - No secrets in git history
  - `.env.example` provided (no real secrets)

- [ ] **Configuration reviewed**
  - Default configuration is secure
  - Insecure defaults documented with warnings

### Security Testing

- [ ] **Unit tests pass**
  - 80%+ coverage
  - Security-critical code 100% covered

- [ ] **Integration tests pass**
  - End-to-end scenarios tested
  - Error paths tested

- [ ] **Static analysis passes**
  - `golangci-lint` with `gosec` enabled
  - No critical issues

- [ ] **Fuzz testing (if applicable)**
  - Input parsers fuzz-tested
  - No crashes on malformed input

- [ ] **Penetration test (Year 5+)**
  - External security audit
  - Findings remediated or risk-accepted

### Deployment

- [ ] **TLS certificates valid**
  - Certificates not expired
  - Certificates trusted (not self-signed for production)

- [ ] **Secrets rotated**
  - JWT signing keys rotated
  - Database passwords rotated (if shared with dev)

- [ ] **Backup tested**
  - Backup process works
  - Restore process tested
  - Backup encryption verified

- [ ] **Monitoring enabled**
  - Logs collected
  - Alerts configured
  - Health checks enabled

- [ ] **Firewall rules active**
  - VLAN segmentation enforced
  - Only required ports open
  - Rate limiting configured

### Documentation

- [ ] **Security documentation updated**
  - `docs/architecture/security-model.md` current
  - Deployment security guide current
  - Firewall rules documented

- [ ] **Release notes include security fixes**
  - Security fixes listed
  - CVEs referenced (if applicable)

- [ ] **Upgrade guide includes security steps**
  - Secrets rotation documented
  - Breaking security changes highlighted

---

## Incident Response Checklist

Use **in the event of a security incident**.

### Immediate Response

- [ ] **Contain the breach**
  - Disconnect compromised systems
  - Revoke compromised credentials
  - Block attacker IP addresses

- [ ] **Assess the damage**
  - What systems were accessed?
  - What data was exposed?
  - How did the attacker get in?

- [ ] **Notify stakeholders**
  - Inform affected users
  - Notify authorities (if required by law)

### Investigation

- [ ] **Preserve evidence**
  - Don't delete logs
  - Take disk/memory snapshots
  - Document timeline

- [ ] **Root cause analysis**
  - Identify vulnerability exploited
  - Determine how attacker gained access
  - Assess scope of compromise

### Remediation

- [ ] **Fix vulnerability**
  - Patch code
  - Update dependencies
  - Harden configuration

- [ ] **Rotate all secrets**
  - Passwords
  - API keys
  - JWT signing keys
  - TLS certificates

- [ ] **Verify no backdoors**
  - Check for unauthorized users
  - Check for unauthorized devices
  - Review audit logs for suspicious activity

### Post-Incident

- [ ] **Post-mortem written**
  - Timeline of events
  - Root cause
  - Remediation steps
  - Lessons learned

- [ ] **Security improvements implemented**
  - Address root cause
  - Improve detection
  - Improve response

- [ ] **Users notified (if applicable)**
  - Explain what happened
  - What data was affected
  - What actions users should take

---

## Annual Security Review

Complete **once per year** for deployed systems.

### Review

- [ ] **Threat model updated**
  - New threats identified
  - Mitigations still effective

- [ ] **Security audit**
  - External security firm
  - Penetration test
  - Code review

- [ ] **Dependency audit**
  - All dependencies current
  - No unpatched vulnerabilities

- [ ] **Access review**
  - Remove inactive users
  - Review admin accounts
  - Review API keys

- [ ] **Incident review**
  - Review past incidents
  - Verify fixes still in place
  - Update incident response plan

### Updates

- [ ] **Security patches applied**
  - OS patches
  - Application patches
  - Firmware updates (KNX gateways, etc.)

- [ ] **Certificates renewed**
  - TLS certificates
  - VPN certificates

- [ ] **Secrets rotated**
  - Passwords
  - API keys
  - Signing keys

- [ ] **Backups verified**
  - Backup integrity checked
  - Restore tested
  - Off-site backups confirmed

---

## Security Contacts

**Report security vulnerabilities to:**
- Email: security@graylogic.uk (when deployed)
- PGP key: [To be generated per deployment]

**Do not report security issues publicly (e.g., GitHub issues).**

---

## Related Documents

- [Security Model](../architecture/security-model.md) — Authentication, authorization, and encryption architecture
- [Development Strategy](DEVELOPMENT-STRATEGY.md) — Security-first development approach
- [Coding Standards](CODING-STANDARDS.md) — Secure coding practices
- [Principles](../overview/principles.md) — Privacy and security principles
