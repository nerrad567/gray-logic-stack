---
title: Security Model
version: 1.1.0
status: active
last_updated: 2026-01-17
depends_on:
  - architecture/system-overview.md
  - architecture/core-internals.md
  - interfaces/api.md
  - overview/principles.md
  - operations/bootstrapping.md
changelog:
  - version: 1.1.0
    date: 2026-01-17
    changes:
      - "Added absolute session lifetime (90 days) for JWT refresh tokens"
      - "Added refresh token family tracking for theft detection"
      - "Changed API key default expiry to 1 year (was never)"
      - "Added explicit PIN rate limiting spec (3 attempts, 5min lockout)"
      - "Added 'Never Log Secrets' requirement"
      - "Updated claim token expiry to 1 hour with 15min rotation"
---

# Security Model

This document specifies Gray Logic's security architecture — authentication, authorization, encryption, and security best practices for deployment.

---

## Overview

### Security Philosophy

Gray Logic implements **defense in depth** with security at every layer:

1. **Offline-first authentication** — No cloud dependency for auth
2. **Least privilege** — Users only get permissions they need
3. **Layered security** — Network, application, and physical security
4. **Audit everything** — All security-relevant actions logged
5. **Fail secure** — Defaults to deny, not allow

### Security Layers

```
┌─────────────────────────────────────────────────────────────────────┐
│                         PHYSICAL SECURITY                            │
│  • Server room access • Equipment access • Network cabinet locks     │
├─────────────────────────────────────────────────────────────────────┤
│                         NETWORK SECURITY                             │
│  • VLAN segregation • Firewall rules • VPN for remote access        │
├─────────────────────────────────────────────────────────────────────┤
│                         TRANSPORT SECURITY                           │
│  • TLS everywhere • Certificate management • Encrypted MQTT         │
├─────────────────────────────────────────────────────────────────────┤
│                         APPLICATION SECURITY                         │
│  • Authentication • Authorization • Rate limiting • Input validation│
├─────────────────────────────────────────────────────────────────────┤
│                         DATA SECURITY                                │
│  • Encrypted secrets • Hashed passwords • Audit logging             │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Authentication

### Authentication Methods

Gray Logic supports multiple authentication methods:

| Method | Use Case | Security Level |
|--------|----------|----------------|
| **Username/Password** | Standard login | High |
| **PIN** | Wall panels, quick access | Medium |
| **API Key** | Service-to-service | High |
| **LDAP/AD** | Enterprise integration | High |
| **OIDC** | SSO integration (optional) | High |

### Authentication Decision Tree

Three primary authentication mechanisms serve different purposes:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     AUTHENTICATION DECISION TREE                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Is this the FIRST RUN of a new system?                                 │
│  │                                                                       │
│  ├── YES ──▶ CLAIM TOKEN                                                │
│  │           • Displayed on console during First Run Wizard             │
│  │           • One-time use, expires in 1 HOUR (not 24h)               │
│  │           • Rotates every 15 minutes while unclaimed                 │
│  │           • Setup mode times out after 24 hours                      │
│  │           • Creates first admin user account                         │
│  │           • After claim, system uses JWT auth                        │
│  │           • See: docs/operations/bootstrapping.md                    │
│  │                                                                       │
│  └── NO ──▶ Is this a HUMAN USER (web UI, mobile app, wall panel)?     │
│             │                                                            │
│             ├── YES ──▶ JWT + REFRESH TOKENS                            │
│             │           • Login with username/password (or LDAP/OIDC)   │
│             │           • Receive access token (1 hour) + refresh token │
│             │           • Access token in Authorization header          │
│             │           • Refresh token rotated on use (family tracked) │
│             │           • 90-day ABSOLUTE session limit (re-auth required)│
│             │           • PIN alternative for registered devices        │
│             │                                                            │
│             └── NO ──▶ API KEY                                          │
│                        • Service-to-service authentication              │
│                        • Home Assistant, monitoring, integrations       │
│                        • 1-year default expiry (explicit opt-out only)  │
│                        • Passed in X-API-Key header                     │
│                        • Last-used tracking for stale key detection     │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Mechanism Comparison

| Mechanism | Lifecycle | Token Format | Renewal | Revocation |
|-----------|-----------|--------------|---------|------------|
| **Claim Token** | One-time use, first run only | 6-char alphanumeric | N/A | Auto-expires in 1 hour, rotates every 15 min |
| **JWT Access Token** | 60 minutes | Signed JWT | Via refresh token | Logout, or wait for expiry |
| **JWT Refresh Token** | 30 days (90 day absolute max) | Opaque token | Rotated on each use, family tracked | Explicit revocation, theft detection |
| **API Key** | 1 year default (explicit opt-out) | `gl_` prefixed string | Regenerate | Admin revokes, expiry warnings |

### When to Use Each

```yaml
authentication_contexts:
  claim_token:
    when: "New system installation, First Run Wizard"
    purpose: "Bootstrap first admin account"
    security: "Physical access to console required"
    after_use: "System transitions to JWT authentication"

  jwt_tokens:
    when:
      - "Web Admin UI login"
      - "Mobile app login"
      - "Wall panel authentication"
    purpose: "Interactive user sessions"
    storage:
      access_token: "Memory only (not persisted)"
      refresh_token: "Secure storage (Keychain, encrypted prefs)"
    security: "Short-lived, renewable, revocable"

  api_keys:
    when:
      - "Home Assistant integration"
      - "External monitoring systems"
      - "Custom scripts/automation"
      - "Third-party integrations"
    purpose: "Machine-to-machine authentication"
    storage: "Secure secrets manager in client system"
    security: "Scoped permissions, rate-limited, audited"

  pin:
    when: "Quick access on trusted wall panels"
    purpose: "Convenience for frequent interactions"
    security: "Reduced permissions, registered devices only"
```

### Token Flow Diagram

```
FIRST RUN (New Installation)
─────────────────────────────
Console ──[Display Claim Token]──▶ Admin ──[Enter in UI]──▶ Create Account ──▶ JWT Auth

USER SESSION (Normal Operation)
───────────────────────────────
User ──[Username/Password]──▶ POST /auth/login ──▶ { access_token, refresh_token }
                                                           │
                                    ┌──────────────────────┘
                                    ▼
                            Include in requests:
                            Authorization: Bearer {access_token}
                                    │
                                    ▼ (Token expires after 60min)
                            POST /auth/refresh
                            { refresh_token }
                                    │
                                    ▼
                            { new_access_token, new_refresh_token }

INTEGRATION (Service-to-Service)
────────────────────────────────
Admin ──[Create API Key in UI]──▶ gl_xxxxxxxx...
                                        │
Integration ──[Store securely]──────────┘
        │
        ▼
Include in requests:
X-API-Key: gl_xxxxxxxx...
```

### Local Authentication

Primary authentication method — stored in local SQLite database.

```yaml
auth:
  local:
    enabled: true
    
    # Password requirements
    password_policy:
      min_length: 12
      require_uppercase: true
      require_lowercase: true
      require_number: true
      require_special: false
      max_age_days: 0                # 0 = never expires
      
    # Account lockout
    lockout:
      enabled: true
      max_attempts: 5
      lockout_duration_minutes: 15
      
    # Session management
    session:
      access_token_lifetime_minutes: 60
      refresh_token_lifetime_days: 30
      max_sessions_per_user: 10
```

### Password Storage

Passwords are never stored in plain text:

```yaml
password_hashing:
  algorithm: "argon2id"             # Memory-hard, recommended
  parameters:
    memory: 65536                   # 64 MB
    iterations: 3
    parallelism: 4
    salt_length: 16
    key_length: 32
```

### JWT Tokens

Authentication uses JSON Web Tokens (JWT):

```yaml
jwt:
  algorithm: "HS256"                # or RS256 for multi-service
  secret_env: "JWT_SECRET"          # From environment

  access_token:
    lifetime_minutes: 60
    claims:
      - sub                          # User ID
      - role                         # User role
      - permissions                  # Permission list
      - exp                          # Expiry
      - jti                          # Unique token ID (for audit)
      - sid                          # Session ID (links to refresh family)

  refresh_token:
    lifetime_days: 30
    rotation: true                   # Issue new refresh on use
    absolute_lifetime_days: 90       # Max session length regardless of refresh
    family_tracking: true            # Detect token theft via refresh family
```

### Refresh Token Family Tracking

To detect and mitigate refresh token theft, Gray Logic implements **refresh token families**:

```yaml
refresh_token_family:
  # How it works:
  # 1. On login, a new "family" is created with a unique family_id
  # 2. Each refresh token has: family_id + generation number
  # 3. On refresh, generation increments, old token invalidated
  # 4. If old token is reused (theft detected), entire family revoked

  structure:
    family_id: "uuid-v4"             # Created at login
    generation: 1                    # Increments on each refresh
    created_at: "timestamp"          # Session start (for absolute lifetime)
    last_refresh_at: "timestamp"

  theft_detection:
    # If a refresh token with generation N is used, but generation N+1
    # already exists, the original token was stolen and reused.
    on_reuse_detected:
      - "Revoke entire token family"
      - "Force re-authentication"
      - "Log security event: token_theft_detected"
      - "Optional: notify user via email/push"

  storage:
    # Refresh token families stored in database
    table: "refresh_token_families"
    columns:
      - family_id                    # Primary key
      - user_id
      - current_generation
      - session_started_at           # For absolute lifetime check
      - last_activity_at
      - revoked_at                   # NULL if active
      - revocation_reason            # theft_detected, logout, expired, admin
```

### Absolute Session Lifetime

Even with rolling refresh tokens, sessions have a hard limit:

```yaml
session_limits:
  # Prevents indefinite session extension via refresh token rolling
  absolute_lifetime_days: 90

  # User must re-authenticate after 90 days, regardless of activity
  # This ensures:
  # - Compromised tokens have bounded exposure
  # - GDPR/compliance session limits are met
  # - Credential changes propagate within bounded time

  enforcement:
    check_on_refresh: true
    check_on_access: false           # Only check on refresh for performance

  user_notification:
    warn_before_expiry_days: 7
    message: "Your session will expire in {days} days. Please save your work."
```

**Token Structure:**

```json
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "sub": "usr-001",
    "name": "Darren",
    "role": "admin",
    "permissions": ["all"],
    "iat": 1736755200,
    "exp": 1736758800
  }
}
```

### PIN Authentication

For wall panels and quick access on trusted devices:

```yaml
pin_auth:
  enabled: true

  # PIN requirements
  policy:
    min_length: 4
    max_length: 8
    allow_sequential: false          # Reject 1234, 4321
    allow_repeated: false            # Reject 1111

  # Restrictions
  restrictions:
    device_registration: true        # PIN only works on registered devices

  # Rate limiting (applies to both physical keypad and voice PIN)
  rate_limiting:
    max_attempts: 3                  # Attempts before lockout
    lockout_duration_minutes: 5      # Initial lockout period
    lockout_escalation:              # Exponential backoff on repeated lockouts
      enabled: true
      multiplier: 2                  # Each subsequent lockout doubles
      max_lockout_minutes: 60        # Cap at 1 hour
    reset_after_success: true        # Successful auth resets attempt counter
    reset_after_minutes: 30          # Counter resets after 30min of no attempts

  # Scope
  scope:
    full_access: false               # PIN grants reduced permissions
    allowed_permissions:
      - "devices:control"
      - "scenes:execute"
      - "modes:change"

  # Security logging
  logging:
    log_attempts: true               # Log all attempts (success and failure)
    log_lockouts: true               # Log when lockout triggered
    never_log_pin_value: true        # CRITICAL: Never log actual PIN digits
```

### Voice PIN Rate Limiting

Voice-based PIN entry uses the same rate limiting as physical keypads:

```yaml
voice_pin:
  # Inherits from pin_auth.rate_limiting
  rate_limiting:
    max_attempts: 3
    lockout_duration_minutes: 5
    lockout_escalation:
      enabled: true
      multiplier: 2
      max_lockout_minutes: 60

  # Additional voice-specific protections
  voice_specific:
    require_wake_word: true          # Must say wake word before PIN
    ambient_noise_threshold: 0.7     # Reject if too noisy (replay risk)
    speaker_verification: false      # Optional: verify voice matches user

  # Audit log entry format
  audit_entry:
    event: "pin.voice_attempt"
    fields:
      - user_id
      - success
      - timestamp
      - device_id                    # Which voice endpoint
      - attempt_number               # 1, 2, or 3
    never_log:
      - pin_value                    # NEVER log the actual PIN
      - audio_recording              # NEVER log raw audio

  # Implementation Reference
  # See docs/intelligence/voice.md "Voice Authentication" for interaction flows
  # and transcript sanitization rules.
```

### LDAP/Active Directory

For enterprise/commercial deployments:

```yaml
auth:
  ldap:
    enabled: true
    type: "ldaps"                    # Always use LDAPS
    
    server:
      host: "ldaps://dc.company.local"
      port: 636
      timeout_seconds: 10
      
    bind:
      dn: "CN=GrayLogic,OU=Service Accounts,DC=company,DC=local"
      password_env: "LDAP_BIND_PASSWORD"
      
    search:
      base_dn: "OU=Users,DC=company,DC=local"
      user_filter: "(sAMAccountName={username})"
      
    # Map AD groups to Gray Logic roles
    group_mapping:
      - ad_group: "CN=GL-Admins,OU=Groups,DC=company,DC=local"
        role: "admin"
        
      - ad_group: "CN=GL-FacilityManagers,OU=Groups,DC=company,DC=local"
        role: "facility_manager"
        
      - ad_group: "CN=Domain Users,DC=company,DC=local"
        role: "user"
```

### OIDC (Optional)

For SSO integration with identity providers:

```yaml
auth:
  oidc:
    enabled: false                   # Opt-in
    
    provider:
      issuer: "https://login.company.com"
      client_id_env: "OIDC_CLIENT_ID"
      client_secret_env: "OIDC_CLIENT_SECRET"
      
    # Claim mapping
    claims:
      username: "preferred_username"
      email: "email"
      groups: "groups"
      
    # Role mapping
    role_mapping:
      - claim_value: "gl-admins"
        role: "admin"
      - claim_value: "gl-users"
        role: "user"
```

---

## Authorization

### Role-Based Access Control (RBAC)

Gray Logic uses RBAC with hierarchical permissions.

### Built-in Roles

| Role | Description | Typical Permissions |
|------|-------------|---------------------|
| **admin** | Full system access | `all` |
| **facility_manager** | Manage building operations | Control, configure, view all |
| **user** | Standard user | Control own area, view limited |
| **guest** | Visitor access | View only, limited areas |
| **integration** | API access | Scoped to API key |

### Role Definitions

```yaml
roles:
  - id: "admin"
    name: "Administrator"
    description: "Full system access"
    permissions:
      - "all"
      
  - id: "facility_manager"
    name: "Facility Manager"
    description: "Building operations management"
    permissions:
      - "devices:read"
      - "devices:control"
      - "devices:configure"
      - "scenes:read"
      - "scenes:execute"
      - "scenes:manage"
      - "schedules:read"
      - "schedules:manage"
      - "modes:read"
      - "modes:change"
      - "energy:read"
      - "phm:read"
      - "users:read"
      - "audit:read"
      
  - id: "user"
    name: "Standard User"
    description: "Normal building occupant"
    permissions:
      - "devices:read"
      - "devices:control"
      - "scenes:read"
      - "scenes:execute"
      - "modes:read"
    scope:
      areas: ["assigned"]            # Only their assigned areas
      
  - id: "guest"
    name: "Guest"
    description: "Visitor with limited access"
    permissions:
      - "devices:read"
    scope:
      areas: ["common"]              # Only common areas
      
  - id: "integration"
    name: "Integration"
    description: "API access for integrations"
    permissions: []                  # Defined per API key
```

### Permission Model

Permissions follow a `resource:action` pattern:

```yaml
permissions:
  # Device permissions
  devices:
    - "devices:read"                 # View device state
    - "devices:control"              # Control devices
    - "devices:configure"            # Change device settings
    
  # Scene permissions
  scenes:
    - "scenes:read"                  # View scenes
    - "scenes:execute"               # Activate scenes
    - "scenes:manage"                # Create/edit/delete scenes
    
  # Schedule permissions
  schedules:
    - "schedules:read"               # View schedules
    - "schedules:manage"             # Create/edit/delete schedules
    
  # Mode permissions
  modes:
    - "modes:read"                   # View current mode
    - "modes:change"                 # Change system mode
    
  # User permissions
  users:
    - "users:read"                   # View user list
    - "users:manage"                 # Create/edit/delete users
    
  # System permissions
  system:
    - "system:read"                  # View system status
    - "system:configure"             # Change system configuration
    
  # Energy permissions
  energy:
    - "energy:read"                  # View energy data
    
  # PHM permissions
  phm:
    - "phm:read"                     # View health data
    - "phm:configure"                # Configure PHM settings
    
  # Audit permissions
  audit:
    - "audit:read"                   # View audit logs
    
  # Special permissions
  special:
    - "all"                          # Full access (admin only)
    - "security:arm"                 # Arm security system
    - "security:disarm"              # Disarm security system
```

### Area-Based Scoping

Permissions can be scoped to specific areas:

```yaml
user:
  id: "usr-002"
  name: "Jane"
  role: "user"
  
  # Area scope
  scope:
    type: "areas"
    areas:
      - "area-floor-2"
      - "area-meeting-rooms"
      
# Jane can only control devices in floor 2 and meeting rooms
```

### Permission Checking

```go
// Example permission check in Core
func (s *Service) ControlDevice(ctx context.Context, deviceID string, cmd Command) error {
    user := auth.UserFromContext(ctx)
    device := s.devices.Get(deviceID)
    
    // Check permission
    if !user.HasPermission("devices:control") {
        return ErrInsufficientPermissions
    }
    
    // Check scope
    if !user.CanAccessArea(device.AreaID) {
        return ErrInsufficientPermissions
    }
    
    // Execute command
    return s.executeCommand(device, cmd)
}
```

---

## API Security

### API Key Management

For service-to-service and integration access:

```yaml
api_keys:
  # Default policy for all API keys
  defaults:
    expiry_days: 365                 # 1 year default expiry
    warn_before_expiry_days: 30      # Warn admin 30 days before expiry
    require_explicit_never_expires: true  # Must explicitly set never_expires

  # Example keys
  keys:
    - id: "key-home-assistant"
      name: "Home Assistant Integration"
      key_hash: "sha256:..."         # Never store plain key
      permissions:
        - "devices:read"
        - "devices:control"
        - "scenes:execute"
      scope:
        areas: ["all"]
      rate_limit:
        requests_per_minute: 60
      expires_at: "2027-01-17T00:00:00Z"  # 1 year from creation
      created_at: "2026-01-17T10:00:00Z"
      last_used_at: "2026-01-17T14:30:00Z"  # Track last usage

    - id: "key-monitoring"
      name: "External Monitoring"
      key_hash: "sha256:..."
      permissions:
        - "system:read"
        - "phm:read"
        - "energy:read"
      rate_limit:
        requests_per_minute: 30
      expires_at: "2027-01-01T00:00:00Z"
      last_used_at: "2026-01-17T12:00:00Z"
```

### API Key Expiry Policy

API keys must have bounded lifetimes to limit exposure from compromised credentials:

```yaml
api_key_policy:
  # Default behavior
  default_expiry_days: 365           # 1 year

  # Never-expiring keys require explicit opt-in
  never_expires:
    allowed: true                    # Can create never-expiring keys
    requires_admin: true             # Only admins can create them
    requires_justification: true     # Must provide reason in audit log
    audit_event: "apikey.never_expires_created"

  # Expiry warnings
  expiry_warnings:
    - days_before: 30
      action: "email_admin"
    - days_before: 7
      action: "email_admin"
    - days_before: 1
      action: "email_admin"
      severity: "critical"

  # Last-used tracking
  last_used_tracking:
    enabled: true
    alert_if_unused_days: 90         # Alert if key unused for 90 days
    suggest_revoke_after_days: 180   # Suggest revoking if unused for 180 days

  # Rotation recommendations
  rotation:
    recommended_interval_days: 365
    grace_period_minutes: 5          # Old key valid for 5min after rotation
```

### API Key Creation with Expiry

When creating API keys, expiry is now mandatory unless explicitly overridden:

```yaml
# POST /api/v1/auth/apikeys
create_api_key:
  request:
    name: "Home Assistant"           # Required
    permissions: ["devices:read"]    # Required
    expires_in_days: 365             # Optional, defaults to 365
    never_expires: false             # Must be explicit if true

  # If never_expires: true
  never_expires_request:
    name: "Critical Infrastructure Monitor"
    permissions: ["system:read"]
    never_expires: true
    justification: "Required for 24/7 monitoring system with no maintenance window"
    # Creates audit log entry with justification

  response:
    id: "key-xxx"
    key: "gl_live_xxx..."            # Shown once only
    expires_at: "2027-01-17T00:00:00Z"
    warning: "This key expires in 365 days. Set a reminder to rotate."
```

### Rate Limiting

Prevent abuse and DoS attacks:

```yaml
rate_limiting:
  enabled: true
  
  # Global limits
  global:
    requests_per_second: 100
    burst: 200
    
  # Per-user limits
  per_user:
    requests_per_minute: 300
    burst: 50
    
  # Per-IP limits (unauthenticated)
  per_ip:
    requests_per_minute: 60
    burst: 20
    
  # Endpoint-specific limits
  endpoints:
    - path: "/api/v1/auth/login"
      requests_per_minute: 10        # Prevent brute force
      
    - path: "/api/v1/voice/*"
      requests_per_minute: 30        # Voice commands
```

### Request Validation

All API requests are validated:

```yaml
request_validation:
  # Input validation
  input:
    max_body_size_kb: 1024
    max_string_length: 10000
    sanitize_html: true
    validate_json: true
    
  # Headers
  headers:
    require_content_type: true
    allowed_content_types:
      - "application/json"
      
  # Query parameters
  query:
    max_array_size: 100
    max_string_length: 1000
```

---

## Transport Security

### TLS Configuration

All connections use TLS:

```yaml
tls:
  enabled: true
  
  # Certificate configuration
  certificate:
    cert_file: "/etc/graylogic/certs/server.crt"
    key_file: "/etc/graylogic/certs/server.key"
    ca_file: "/etc/graylogic/certs/ca.crt"
    
  # TLS version
  min_version: "1.2"
  max_version: "1.3"
  
  # Cipher suites (TLS 1.2)
  cipher_suites:
    - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
    - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
    - "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"
    - "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"
    
  # HSTS
  hsts:
    enabled: true
    max_age_seconds: 31536000
    include_subdomains: true
```

### Certificate Management

```yaml
certificates:
  # Self-signed (development/internal)
  self_signed:
    enabled: true
    validity_days: 365
    auto_renew: true
    renew_before_days: 30
    
  # Let's Encrypt (if internet available)
  letsencrypt:
    enabled: false
    email: "admin@example.com"
    domains:
      - "graylogic.example.com"
    challenge: "dns-01"              # For internal networks
    
  # Custom CA (enterprise)
  custom_ca:
    enabled: false
    ca_cert: "/path/to/ca.crt"
```

### MQTT Security

```yaml
mqtt:
  security:
    # TLS
    tls:
      enabled: true
      cert_file: "/etc/graylogic/certs/mqtt.crt"
      key_file: "/etc/graylogic/certs/mqtt.key"

    # Authentication
    authentication:
      enabled: true
      method: "password"             # password | certificate | mtls

    # ACLs
    acl:
      enabled: true
      rules:
        - user: "core"
          topic: "#"
          access: "readwrite"

        - user: "bridge-knx"
          topic: "graylogic/+/knx/#"
          access: "readwrite"

        - user: "bridge-knx"
          topic: "graylogic/state/#"
          access: "read"
```

### MQTT Mutual TLS (mTLS)

For higher security deployments, bridges can authenticate via client certificates:

```yaml
mqtt_mtls:
  # Enable mutual TLS
  enabled: true

  # Broker configuration (Mosquitto)
  broker:
    listener: 8883
    cafile: "/etc/graylogic/certs/ca.crt"
    certfile: "/etc/graylogic/certs/mqtt-broker.crt"
    keyfile: "/etc/graylogic/certs/mqtt-broker.key"
    require_certificate: true
    use_identity_as_username: true  # CN becomes username for ACL

  # Client (Bridge) certificates
  bridge_certificates:
    # Each bridge has its own certificate
    knx_bridge:
      cn: "bridge-knx"
      cert: "/etc/graylogic/certs/bridge-knx.crt"
      key: "/etc/graylogic/certs/bridge-knx.key"

    dali_bridge:
      cn: "bridge-dali"
      cert: "/etc/graylogic/certs/bridge-dali.crt"
      key: "/etc/graylogic/certs/bridge-dali.key"

    core:
      cn: "graylogic-core"
      cert: "/etc/graylogic/certs/core.crt"
      key: "/etc/graylogic/certs/core.key"

  # Certificate generation
  certificate_management:
    ca:
      generate_on_install: true
      validity_years: 10
      key_size: 4096

    bridge_certs:
      validity_years: 5
      key_size: 2048
      auto_renew_days_before: 30

  # Benefits of mTLS
  benefits:
    - "No password management for bridges"
    - "Certificate revocation for compromised bridges"
    - "Stronger authentication than password"
    - "Bridge identity verified cryptographically"

  # When to use
  recommendations:
    use_mtls_when:
      - "High-security commercial deployments"
      - "Bridges on separate network segments"
      - "Regulatory compliance requirements"
    use_password_when:
      - "Simpler residential deployments"
      - "All components on same host"
      - "Easier initial setup needed"
```

---

## Secrets Management

### Environment Variables

Sensitive values from environment:

```yaml
secrets:
  method: "environment"
  
  variables:
    - JWT_SECRET
    - DB_ENCRYPTION_KEY
    - LDAP_BIND_PASSWORD
    - INFLUXDB_TOKEN
    - MQTT_PASSWORD
```

### Secrets File

For deployments without environment variable support:

```yaml
secrets:
  method: "file"
  file: "/etc/graylogic/secrets.yaml"
  permissions: "0600"
  owner: "graylogic"
```

**secrets.yaml** (encrypted at rest):
```yaml
jwt_secret: "randomly-generated-secret"
db_encryption_key: "another-random-secret"
ldap_bind_password: "ad-password"
```

### Secret Rotation

```yaml
secret_rotation:
  jwt_secret:
    auto_rotate: false               # Manual rotation recommended
    dual_secret_support: true        # Support old+new secret during rotation

  api_keys:
    max_age_days: 365
    warn_before_expiry_days: 30
    regeneration_allowed: true       # Keys can be regenerated
```

### JWT Secret Rotation Procedure

When JWT secret needs to be rotated (e.g., suspected compromise):

```yaml
jwt_rotation:
  # Dual-secret transition period
  transition:
    duration_hours: 24              # Both old and new secret valid for 24h
    verification_order: ["new", "old"]  # Try new first

  # Rotation procedure
  procedure:
    1: "Generate new JWT secret"
    2: "Add new secret to configuration (JWT_SECRET_NEW)"
    3: "Restart Core with dual-secret mode"
    4: "All new tokens issued with new secret"
    5: "Existing tokens verified against both secrets"
    6: "After 24h, remove old secret (JWT_SECRET_OLD)"
    7: "Restart Core with single secret"

  # Configuration during rotation
  config:
    jwt:
      secrets:
        current: "${JWT_SECRET_NEW}"
        previous: "${JWT_SECRET_OLD}"  # Only during rotation
      transition_mode: true

  # Emergency rotation (immediate, all sessions invalidated)
  emergency:
    procedure:
      1: "Generate new JWT secret"
      2: "Replace JWT_SECRET immediately"
      3: "Restart Core"
      4: "All existing tokens immediately invalid"
      5: "All users must re-authenticate"
    use_when: "Active compromise confirmed"
```

### API Key Regeneration

API keys can be regenerated without deleting and recreating:

```yaml
api_key_regeneration:
  # Regeneration endpoint
  endpoint:
    method: POST
    path: "/api/v1/api-keys/{id}/regenerate"
    auth: "admin only"

  # Regeneration behavior
  behavior:
    old_key_grace_period_minutes: 5  # Old key works for 5 more minutes
    notify_on_regenerate: true       # Email admin when key regenerated
    log_event: true                  # Audit log entry

  # Response
  response:
    new_key: "gl_xxxx..."            # Shown once, cannot be retrieved again
    expires_in_minutes: 5            # Old key expiry countdown
    note: "Save this key - it cannot be retrieved again"

  # Recovery if key lost
  recovery:
    method: "regenerate"             # Only option is to regenerate
    old_key_revoked: "immediately or after grace period"

  # Rate limiting
  rate_limit:
    regenerations_per_hour: 3        # Prevent abuse
```

---

## Audit Logging

### What is Logged

All security-relevant events are logged:

```yaml
audit:
  enabled: true

  events:
    # Authentication
    - "auth.login.success"
    - "auth.login.failure"
    - "auth.logout"
    - "auth.token.refresh"
    - "auth.lockout"
    - "auth.token_theft_detected"    # Refresh token family reuse

    # Authorization
    - "auth.permission.denied"

    # User management
    - "user.created"
    - "user.updated"
    - "user.deleted"
    - "user.password.changed"

    # API keys
    - "apikey.created"
    - "apikey.revoked"
    - "apikey.regenerated"
    - "apikey.never_expires_created" # Requires justification
    - "apikey.expiry_warning"

    # System
    - "system.config.changed"
    - "system.backup.created"
    - "system.restore.performed"

    # Security
    - "security.arm"
    - "security.disarm"
    - "security.alarm"
    - "security.pin_attempt"         # PIN auth attempts
    - "security.pin_lockout"         # PIN lockout triggered

    # Sensitive commands
    - "device.command"               # All device commands
    - "scene.activated"
    - "mode.changed"
```

### Never Log Secrets (CRITICAL)

Certain values MUST NEVER appear in logs at any level (debug, info, warn, error, audit):

```yaml
never_log:
  # Authentication secrets
  secrets:
    - password                       # User passwords (plaintext or hashed)
    - pin                            # PIN values, even on failed attempts
    - api_key                        # Full API key values (log key ID only)
    - jwt_token                      # Full JWT tokens (log jti claim only)
    - refresh_token                  # Full refresh tokens
    - claim_token                    # Setup claim tokens

  # Sensitive data
  sensitive:
    - voice_audio                    # Raw voice recordings
    - biometric_data                 # Any biometric identifiers

  # What TO log instead
  safe_alternatives:
    password: "Log 'password_changed' event without value"
    pin: "Log 'pin_attempt' with success/failure, never the PIN"
    api_key: "Log key_id, e.g., 'key-home-assistant'"
    jwt_token: "Log jti claim, e.g., 'jti: abc123'"
    refresh_token: "Log family_id, e.g., 'family: xyz789'"

  # Enforcement
  enforcement:
    code_review_required: true       # PRs must verify no secret logging
    static_analysis: true            # Linter rules to detect patterns
    runtime_scrubbing: true          # Scrub known patterns from log output

  # Example: WRONG vs RIGHT
  examples:
    wrong:
      - "PIN 1234 failed for user admin"
      - "Invalid password 'hunter2' for user admin"
      - "API key gl_live_abc123... used from 192.168.1.50"

    right:
      - "PIN attempt failed for user admin (attempt 2/3)"
      - "Login failed for user admin: invalid_password"
      - "API key key-home-assistant used from 192.168.1.50"
```

### Audit Log Format

```yaml
audit_log:
  format: "json"
  
  fields:
    - timestamp                      # ISO 8601
    - event_type                     # Event category
    - user_id                        # Who
    - user_ip                        # From where
    - resource                       # What resource
    - action                         # What action
    - result                         # Success/failure
    - details                        # Additional context
```

**Example log entry:**
```json
{
  "timestamp": "2026-01-13T10:30:00Z",
  "event_type": "auth.login.failure",
  "user_id": null,
  "user_ip": "192.168.1.50",
  "resource": "auth",
  "action": "login",
  "result": "failure",
  "details": {
    "username": "admin",
    "reason": "invalid_password",
    "attempt": 3
  }
}
```

### Log Storage

```yaml
audit_log:
  storage:
    type: "file"
    path: "/var/log/graylogic/audit.log"
    
  rotation:
    max_size_mb: 100
    max_files: 10
    compress: true
    
  retention:
    days: 365                        # Keep for 1 year
    
  # Optional: forward to SIEM
  siem:
    enabled: false
    type: "syslog"
    host: "siem.company.local"
    port: 514
    protocol: "tcp"
```

---

## Network Security

### VLAN Segregation

Recommended network architecture:

```yaml
network_segregation:
  vlans:
    - id: 100
      name: "building_control"
      devices:
        - "Gray Logic Server"
        - "KNX/IP Interface"
        - "DALI Gateways"
        - "Modbus Gateways"
      internet_access: false
      
    - id: 200
      name: "security"
      devices:
        - "CCTV NVR"
        - "Alarm Panel"
        - "Access Control"
      internet_access: false
      
    - id: 300
      name: "user"
      devices:
        - "Wall Panels"
        - "User devices"
      internet_access: true
```

### Firewall Rules

```yaml
firewall:
  default_policy: "deny"
  
  rules:
    # User VLAN → Control VLAN (Gray Logic API only)
    - name: "User to GL API"
      source: "vlan_300"
      destination: "gray_logic_server"
      port: 443
      protocol: "tcp"
      action: "allow"
      
    # Control VLAN → Security VLAN (limited)
    - name: "GL to NVR"
      source: "gray_logic_server"
      destination: "nvr"
      port: 554                      # RTSP
      protocol: "tcp"
      action: "allow"
      
    # No direct access from internet
    - name: "Block WAN"
      source: "wan"
      destination: "vlan_100"
      action: "deny"
```

### Remote Access

Remote access via WireGuard VPN only:

```yaml
remote_access:
  method: "wireguard"
  
  wireguard:
    listen_port: 51820
    interface: "wg0"
    
    # Server configuration
    server:
      private_key_env: "WG_PRIVATE_KEY"
      address: "10.100.0.1/24"
      
    # Authorized peers
    peers:
      - name: "Owner Mobile"
        public_key: "peer-public-key"
        allowed_ips: "10.100.0.2/32"
        
      - name: "Support"
        public_key: "support-public-key"
        allowed_ips: "10.100.0.10/32"
        # Time-limited access
        valid_until: "2026-02-01T00:00:00Z"
```

---

## Security Best Practices

### Deployment Checklist

```yaml
deployment_security_checklist:
  network:
    - [ ] VLANs configured and tested
    - [ ] Firewall rules in place
    - [ ] No unnecessary internet access
    - [ ] WireGuard for remote access only
    
  authentication:
    - [ ] Strong admin password set
    - [ ] Default accounts disabled/renamed
    - [ ] API keys generated securely
    - [ ] JWT secret is random and secure
    
  encryption:
    - [ ] TLS enabled on all endpoints
    - [ ] Valid certificates installed
    - [ ] MQTT TLS enabled
    - [ ] Secrets stored securely
    
  monitoring:
    - [ ] Audit logging enabled
    - [ ] Log retention configured
    - [ ] Failed login alerts configured
    
  physical:
    - [ ] Server in secure location
    - [ ] Network cabinet locked
    - [ ] USB ports disabled (optional)
```

### Security Hardening

```yaml
hardening:
  # Disable unnecessary features
  disable:
    - "remote_debugging"
    - "anonymous_access"
    - "auto_discovery"               # After commissioning
    
  # Timeouts
  timeouts:
    idle_session_minutes: 60
    api_request_seconds: 30
    
  # Headers
  security_headers:
    - "X-Content-Type-Options: nosniff"
    - "X-Frame-Options: DENY"
    - "X-XSS-Protection: 1; mode=block"
    - "Content-Security-Policy: default-src 'self'"
```

### Incident Response

```yaml
incident_response:
  # Account compromise
  account_compromise:
    - "Disable affected account immediately"
    - "Revoke all sessions and API keys"
    - "Review audit logs"
    - "Reset password"
    - "Enable additional monitoring"
    
  # Suspected intrusion
  intrusion:
    - "Isolate system from network"
    - "Preserve audit logs"
    - "Document timeline"
    - "Contact security professional"
    
  # Data breach
  data_breach:
    - "Identify scope of breach"
    - "Notify affected parties (if applicable)"
    - "Preserve evidence"
    - "Remediate vulnerability"
```

---

## Compliance Considerations

### Data Protection

```yaml
data_protection:
  # Personal data handling
  personal_data:
    - "User names and credentials"
    - "Presence/occupancy patterns"
    - "Voice recordings (if enabled)"
    
  # Privacy by design
  privacy:
    - "Minimize data collection"
    - "Local processing preferred"
    - "No cloud storage of personal data by default"
    - "Clear retention policies"
    
  # GDPR considerations (EU deployments)
  gdpr:
    data_subject_rights:
      - "Right to access"
      - "Right to rectification"
      - "Right to erasure"
    data_controller: "Property owner"
    data_processor: "Gray Logic (if providing support)"
```

### Industry Standards

```yaml
standards_alignment:
  # Relevant standards
  standards:
    - name: "ISO 27001"
      scope: "Information security management"
      status: "Aligned (not certified)"
      
    - name: "IEC 62443"
      scope: "Industrial automation security"
      status: "Aligned for relevant controls"
      
    - name: "NIST Cybersecurity Framework"
      scope: "Cybersecurity best practices"
      status: "Aligned"
```

---

## Configuration Reference

### Complete Security Configuration

```yaml
# /etc/graylogic/security.yaml
security:
  # Authentication
  auth:
    local:
      enabled: true
      password_policy:
        min_length: 12
      lockout:
        enabled: true
        max_attempts: 5
        
    ldap:
      enabled: false
      
    oidc:
      enabled: false
      
  # Session management
  session:
    access_token_lifetime_minutes: 60
    refresh_token_lifetime_days: 30
    
  # TLS
  tls:
    enabled: true
    min_version: "1.2"
    cert_file: "/etc/graylogic/certs/server.crt"
    key_file: "/etc/graylogic/certs/server.key"
    
  # Rate limiting
  rate_limiting:
    enabled: true
    per_user:
      requests_per_minute: 300
      
  # Audit logging
  audit:
    enabled: true
    retention_days: 365
    
  # Remote access
  remote_access:
    method: "wireguard"
```

---

## Related Documents

- [System Overview](system-overview.md) — Overall architecture
- [Core Internals](core-internals.md) — Core security implementation
- [API Specification](../interfaces/api.md) — API authentication details
- [Principles](../overview/principles.md) — Security principles
- [Office/Commercial Deployment](../deployment/office-commercial.md) — AD integration
- [Residential Deployment](../deployment/residential.md) — Home security
