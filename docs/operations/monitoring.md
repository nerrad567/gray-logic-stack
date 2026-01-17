---
title: Monitoring and Alerting
version: 1.0.0
status: active
last_updated: 2026-01-15
depends_on:
  - architecture/system-overview.md
  - architecture/core-internals.md
  - resilience/offline.md
---

# Monitoring and Alerting

This document specifies how Gray Logic systems are monitored — both for installers/administrators and for end customers via front-end indicators.

---

## Overview

### Monitoring Philosophy

Gray Logic implements a two-tier monitoring approach:

1. **Customer-Facing** — Visual and audible indicators on wall panels and displays
2. **Installer/Admin-Facing** — Remote monitoring via secure VPN with Prometheus/Grafana

### Design Principles

1. **Customers shouldn't need to be system admins** — Issues should be obvious and actionable
2. **Installers should know before customers** — Proactive remote monitoring
3. **No silent failures** — Every failure must be visible somewhere
4. **Privacy-preserving** — Remote monitoring doesn't expose personal data

---

## Customer-Facing Monitoring

### Wall Panel Health Indicator

Every wall panel displays a system health status:

```yaml
wall_panel_health:
  # Status indicator (always visible in corner)
  indicator:
    position: "top_right"
    size: "12px dot"

    states:
      healthy:
        color: "green"
        pulse: false
        tooltip: "System healthy"

      degraded:
        color: "amber"
        pulse: true                   # Slow pulse to attract attention
        tooltip: "Some features unavailable"
        on_tap: "Show degradation details"

      error:
        color: "red"
        pulse: true                   # Fast pulse
        tooltip: "System needs attention"
        on_tap: "Show error details"

      offline:
        color: "grey"
        pulse: false
        tooltip: "Cannot connect to Gray Logic"
```

### Dead Man's Switch (Front-End Heartbeat)

Wall panels implement a heartbeat check to detect Core failures:

```yaml
dead_man_switch:
  # Wall panel checks Core health periodically
  heartbeat:
    endpoint: "/api/v1/health"
    interval_seconds: 30
    timeout_seconds: 10

  # Behavior when heartbeat fails
  failure_detection:
    consecutive_failures: 3         # After 3 failures (90s)
    action: "enter_offline_mode"

  # Offline mode display
  offline_mode:
    screen_state: "dim"             # Dim screen to 30%
    overlay:
      visible: true
      message: "Connection to system lost"
      subtext: "Physical controls still work"
      icon: "cloud_off"
      color: "amber"

    # Retry indicator
    retry_indicator:
      show: true
      text: "Reconnecting..."
      spinner: true

  # Recovery
  recovery:
    action: "automatic"
    on_reconnect:
      - "Clear overlay"
      - "Refresh state"
      - "Show brief 'Connected' toast"
```

### Error Screen

When a critical error requires attention:

```yaml
error_screen:
  # Full-screen error for critical issues
  critical_errors:
    - type: "database_corruption"
      screen: "full"
      message: "System Error"
      subtext: "Please contact your installer"
      action_button: "Call Support"

    - type: "core_crash_loop"
      screen: "full"
      message: "System Restarting"
      subtext: "This may take a few minutes"
      progress: true

  # Banner errors (non-blocking)
  banner_errors:
    - type: "bridge_offline"
      position: "top"
      message: "{bridge_name} offline - some devices unavailable"
      dismissible: true
      auto_dismiss_seconds: 30

    - type: "internet_offline"
      position: "top"
      message: "No internet connection"
      subtext: "Local control still works"
      dismissible: false            # Stay until resolved
```

### Audible Alerts

For critical issues, optional audible notification:

```yaml
audible_alerts:
  enabled: true                     # User can disable

  alerts:
    - type: "fire_alarm_signal"
      sound: "alert_critical.wav"
      volume: 100                    # Always max volume
      repeat: true

    - type: "security_breach"
      sound: "alert_security.wav"
      volume: 80
      repeat: true

    - type: "system_error"
      sound: "alert_soft.wav"
      volume: 50
      repeat: false

  # Quiet hours (except fire/security)
  quiet_hours:
    enabled: true
    start: "22:00"
    end: "07:00"
    exceptions: ["fire_alarm_signal", "security_breach"]
```

---

### Installer/Admin Remote Monitoring

### Secure Remote Access

Installers can remotely monitor customer systems via WireGuard VPN:

```yaml
remote_monitoring:
  # VPN-based access only
  access:
    method: "wireguard"
    port: 51820

  # Time-limited access tokens
  access_tokens:
    installer:
      duration_hours: 24            # Valid for 24 hours
      renewable: true
      permissions: ["read", "diagnose"]

    emergency:
      duration_hours: 4
      permissions: ["read", "diagnose", "restart_services"]

  # Customer approval required
  customer_consent:
    required: true
    notification: "Your installer is connecting for maintenance"
    can_revoke: true
```

### Push Telemetry (Scalable Monitoring)

To allow monitoring of large fleets without maintaining persistent VPN tunnels to every site, Gray Logic supports **Push Telemetry**.

**Concept:**
The Core periodically pushes a lightweight, anonymized health packet to a central monitoring endpoint controlled by the installer.

**Architecture:**
```
[Site A Core] ──(HTTPS Push)──▶ [Installer Collector]
[Site B Core] ──(HTTPS Push)──▶ [      (Influx)     ]
                                [      (Grafana)    ]
```

**Security & Privacy:**
1.  **Transport:** HTTPS (TLS 1.2+) only.
2.  **Auth:** Site-specific API Key or Mutual TLS.
3.  **Data Minimization:** strictly **No PII** (No user names, no room names, no video/audio). Only structural IDs (`device-123`) and health metrics.
4.  **Opt-In:** Customer must explicitly enable "Remote Health Monitoring".

**Payload Example:**
```json
{
  "site_id": "a7b3c9d2...",
  "timestamp": "2026-01-15T10:30:00Z",
  "status": "degraded",
  "version": "1.0.0",
  "metrics": {
    "uptime": 86400,
    "error_rate": 0.05,
    "bridges_online": 3,
    "bridges_total": 4,
    "disk_free_gb": 15
  },
  "active_alerts": ["bridge_dali_offline"]
}
```

**Benefits:**
- **Scalable:** Monitor 1000+ sites easily.
- **Firewall Friendly:** Outbound HTTPS only; no inbound ports needed at customer site.
- **Low Bandwidth:** < 1KB per minute.

### Prometheus Metrics Endpoint

Core exposes metrics for remote collection:

```yaml
prometheus:
  endpoint: "/metrics"
  port: 9090                        # Internal port, VPN access only

  # Metrics exposed
  metrics:
    # System health
    - name: "graylogic_up"
      type: "gauge"
      description: "System is running"

    - name: "graylogic_uptime_seconds"
      type: "counter"
      description: "Seconds since start"

    # Component health
    - name: "graylogic_component_healthy"
      type: "gauge"
      labels: ["component"]
      description: "Component health status (1=healthy)"

    # Device metrics
    - name: "graylogic_devices_total"
      type: "gauge"
      labels: ["domain", "protocol"]
      description: "Total registered devices"

    - name: "graylogic_devices_online"
      type: "gauge"
      labels: ["domain", "protocol"]
      description: "Currently reachable devices"

    # Performance metrics
    - name: "graylogic_command_latency_seconds"
      type: "histogram"
      description: "Command execution latency"
      buckets: [0.05, 0.1, 0.2, 0.5, 1.0, 2.0]

    - name: "graylogic_api_requests_total"
      type: "counter"
      labels: ["method", "path", "status"]
      description: "API request count"

    # Error metrics
    - name: "graylogic_errors_total"
      type: "counter"
      labels: ["component", "type"]
      description: "Error count by component"

    # MQTT broker health (critical dependency)
    - name: "graylogic_mqtt_broker_up"
      type: "gauge"
      description: "MQTT broker reachable (1=up, 0=down)"

    - name: "graylogic_mqtt_broker_restarts_total"
      type: "counter"
      description: "Automatic broker restart count"
```

### Proactive Monitoring (Installer Dashboard)

Installers can aggregate monitoring from multiple customer sites:

```yaml
installer_dashboard:
  # Central monitoring server (installer-operated)
  architecture:
    method: "pull"                  # Installer pulls metrics via VPN
    frequency: "5m"

  # Per-site health summary
  site_health:
    metrics:
      - "System up/down"
      - "Component health"
      - "Device connectivity %"
      - "Error rate (last 24h)"
      - "Last successful backup"

  # Alerting thresholds
  alerts:
    - name: "Site offline"
      condition: "graylogic_up == 0 for 5m"
      severity: "critical"
      action: "page_installer"

    - name: "High error rate"
      condition: "rate(graylogic_errors_total[1h]) > 10"
      severity: "warning"
      action: "email_installer"

    - name: "Bridge offline"
      condition: "graylogic_component_healthy{component=~'bridge.*'} == 0 for 5m"
      severity: "warning"
      action: "email_installer"

    - name: "Backup overdue"
      condition: "time() - graylogic_last_backup_timestamp > 86400 * 7"
      severity: "warning"
      action: "email_installer"
```

### Customer Notification Flow

When installer detects an issue remotely:

```yaml
notification_flow:
  # Installer detects issue
  detection:
    source: "prometheus_alert"

  # Installer decides to notify customer
  installer_action:
    - "Investigate remotely (if VPN access)"
    - "Attempt remote fix (if appropriate)"
    - "Contact customer if on-site visit needed"

  # Customer contact
  contact_methods:
    - method: "phone"
      message: "We've detected an issue with your system. We can fix it remotely / need to schedule a visit."

    - method: "email"
      template: "system_issue_notification"
      include:
        - issue_summary
        - recommended_action
        - contact_details

  # Dashboard notification (if customer has web access)
  dashboard_notification:
    show: true
    message: "Your installer has been notified of a system issue"
    include_ticket_number: true
```

---

## Health Checks

### Component Health Checks

```yaml
health_checks:
  core:
    check: "Process running, responds to health endpoint"
    interval: 10s

  database:
    check: "Can execute query"
    interval: 60s
    query: "SELECT 1"

  mqtt:
    check: "Connected to broker"
    interval: 10s

  influxdb:
    check: "Can ping InfluxDB"
    interval: 60s

  bridges:
    check: "Health message received within timeout"
    interval: 30s
    timeout: 60s

  internet:
    check: "Can reach external host"
    interval: 300s
    host: "dns.google:53"           # TCP connect test
```

### MQTT Broker Health & Auto-Recovery

The MQTT broker (Mosquitto) is a critical dependency. Core implements dedicated health monitoring with automatic recovery.

```yaml
mqtt_broker_health:
  # Health check configuration
  health_check:
    method: "tcp_connect"           # Connect to broker port
    endpoint: "localhost:1883"
    interval_seconds: 10
    timeout_seconds: 5
    consecutive_failures: 3         # Trigger recovery after 3 failures

  # Prometheus metrics
  metrics:
    - name: "graylogic_mqtt_broker_up"
      type: "gauge"
      description: "MQTT broker reachable (1=up, 0=down)"

    - name: "graylogic_mqtt_broker_restarts_total"
      type: "counter"
      description: "Number of automatic broker restarts"

    - name: "graylogic_mqtt_last_message_received_timestamp"
      type: "gauge"
      description: "Unix timestamp of last MQTT message received"

  # Auto-restart configuration (Docker)
  docker_restart:
    enabled: true
    policy: "unless-stopped"        # Docker Compose restart policy
    max_restarts: 5                 # Within restart_window
    restart_window_seconds: 300     # 5 restarts in 5 minutes = give up

  # Auto-restart configuration (systemd alternative)
  systemd_restart:
    enabled: false                  # Use if not running Docker
    unit: "mosquitto.service"
    restart_delay_seconds: 5

  # Recovery actions
  recovery:
    on_broker_down:
      - action: "log_warning"
        message: "MQTT broker health check failed"

      - action: "set_system_status"
        status: "degraded"

      - action: "notify_ui"
        message: "Communication system restarting..."

    on_broker_recovered:
      - action: "log_info"
        message: "MQTT broker recovered"

      - action: "reconnect_all_bridges"
        delay_seconds: 2            # Wait for broker to stabilise

      - action: "republish_state"
        scope: "all_devices"        # Re-sync state after reconnect

      - action: "clear_degraded"
        condition: "no_other_issues"

  # Alerting
  alerts:
    - name: "MQTT broker down"
      condition: "graylogic_mqtt_broker_up == 0 for 30s"
      severity: "critical"
      customer_action: "Show 'System communication issue' banner"
      installer_action: "Page on-call"

    - name: "MQTT broker restart loop"
      condition: "increase(graylogic_mqtt_broker_restarts_total[5m]) > 3"
      severity: "critical"
      customer_action: "Show error screen, contact installer"
      installer_action: "Page on-call, requires manual intervention"
```

**Recovery Sequence:**

```
1. Health check fails (10s interval)
2. Retry 2 more times (30s total)
3. Mark system "degraded", notify UI
4. Docker/systemd restarts Mosquitto
5. Core detects broker available
6. Core reconnects all bridge subscriptions
7. Core republishes current device state to ensure consistency
8. Clear "degraded" status if no other issues
```

**Manual Recovery:**

If auto-restart fails repeatedly, manual intervention is required:

```bash
# Check Mosquitto logs
docker logs graylogic-mqtt --tail 100

# Force restart
docker restart graylogic-mqtt

# If persistent failure, check config
docker exec graylogic-mqtt cat /mosquitto/config/mosquitto.conf
```

### Health API Response

```http
GET /api/v1/health
```

```json
{
  "status": "degraded",
  "timestamp": "2026-01-15T10:30:00Z",
  "uptime_seconds": 86400,
  "degradation_level": 2,

  "components": {
    "core": {
      "status": "healthy",
      "details": {}
    },
    "database": {
      "status": "healthy",
      "last_write": "2026-01-15T10:29:55Z"
    },
    "mqtt": {
      "status": "healthy",
      "connected": true
    },
    "bridges": {
      "knx": {
        "status": "healthy",
        "devices": 45,
        "last_message": "2026-01-15T10:29:58Z"
      },
      "dali": {
        "status": "offline",
        "last_seen": "2026-01-15T09:15:00Z",
        "error": "Connection timeout"
      }
    },
    "internet": {
      "status": "unavailable"
    }
  },

  "issues": [
    {
      "component": "bridges.dali",
      "severity": "warning",
      "message": "DALI bridge offline",
      "since": "2026-01-15T09:15:00Z",
      "customer_visible": true,
      "customer_message": "Some lights unavailable"
    }
  ]
}
```

---

## Alerting Rules

### Severity Levels

```yaml
severity_levels:
  critical:
    description: "Immediate attention required"
    examples:
      - "Core down"
      - "Fire alarm integration failure"
      - "Database corruption"
    customer_notification: "full_screen_error"
    installer_notification: "page"

  warning:
    description: "Attention needed soon"
    examples:
      - "Bridge offline"
      - "Device unreachable"
      - "Backup failed"
    customer_notification: "banner"
    installer_notification: "email"

  info:
    description: "Informational, no action needed"
    examples:
      - "Internet temporarily unavailable"
      - "Update available"
    customer_notification: "status_dot"
    installer_notification: "dashboard_only"
```

### Alert Definitions

```yaml
alerts:
  - name: "Core down"
    condition: "health endpoint unreachable for 60s"
    severity: "critical"
    customer_action: "Show offline screen"
    installer_action: "Page on-call"

  - name: "All bridges offline"
    condition: "No bridge healthy for 120s"
    severity: "critical"
    customer_action: "Show error, explain physical controls work"
    installer_action: "Alert via email"

  - name: "Single bridge offline"
    condition: "Bridge health not received for 60s"
    severity: "warning"
    customer_action: "Banner notification"
    installer_action: "Dashboard alert"

  - name: "Device unreachable"
    condition: "Device not responding for 5m"
    severity: "warning"
    customer_action: "Dim device in UI"
    installer_action: "Log only"

  - name: "High command latency"
    condition: "p95 latency > 500ms for 5m"
    severity: "info"
    customer_action: "None"
    installer_action: "Dashboard metric"

  - name: "Disk space low"
    condition: "Free space < 10%"
    severity: "warning"
    customer_action: "Banner notification"
    installer_action: "Email alert"
```

---

## Logging

### Log Aggregation

```yaml
logging:
  # Local logs (always)
  local:
    path: "/var/log/graylogic/"
    files:
      - "core.log"
      - "api.log"
      - "audit.log"
      - "bridges/*.log"
    rotation:
      max_size: "100MB"
      max_files: 10

  # Remote logging (optional, for support)
  remote:
    enabled: false                  # Customer must opt-in
    method: "syslog_tls"
    destination: "logs.graylogic.uk:6514"
    filter: "no_personal_data"      # Strip user IDs, device names
```

### Log Levels for Troubleshooting

```yaml
log_levels:
  production:
    core: "info"
    api: "warning"
    bridges: "info"

  debug:                            # Installer can enable temporarily
    core: "debug"
    api: "debug"
    bridges: "debug"
    retention: "24h"                # Auto-revert after 24h
```

---

## Dashboard Views

### Customer Dashboard (Simple)

```yaml
customer_dashboard:
  # Simple status page accessible on local network
  url: "https://graylogic.local/status"

  shows:
    - "Overall status (healthy/degraded/error)"
    - "Active issues (plain language)"
    - "Last backup date"
    - "Support contact button"

  hides:
    - "Technical metrics"
    - "Log files"
    - "Configuration details"
```

### Installer Dashboard (Detailed)

```yaml
installer_dashboard:
  # Grafana dashboards (VPN access)
  dashboards:
    - name: "System Overview"
      panels:
        - "Component health matrix"
        - "Device connectivity %"
        - "API response times"
        - "Error rate graph"

    - name: "Device Status"
      panels:
        - "Devices by protocol"
        - "Offline devices list"
        - "Command success rate"

    - name: "Performance"
      panels:
        - "CPU/Memory usage"
        - "Command latency histogram"
        - "MQTT message rate"

    - name: "Alerts"
      panels:
        - "Active alerts"
        - "Alert history"
        - "Resolution time"
```

---

## Configuration

### Monitoring Configuration

```yaml
# /etc/graylogic/monitoring.yaml
monitoring:
  # Customer-facing
  customer:
    wall_panel_indicator: true
    dead_man_switch: true
    audible_alerts: true
    quiet_hours:
      enabled: true
      start: "22:00"
      end: "07:00"

  # Prometheus
  prometheus:
    enabled: true
    listen: "127.0.0.1:9090"        # Local only, VPN for remote

  # Remote monitoring consent
  remote:
    enabled: false                  # Customer must opt-in
    installer_vpn: true

  # Alerting
  alerts:
    email:
      enabled: true
      recipient: "admin@example.com"
    push:
      enabled: false
```

---

## Related Documents

- [System Overview](../architecture/system-overview.md) — Architecture
- [Offline Behavior](../resilience/offline.md) — Graceful degradation
- [Security Model](../architecture/security-model.md) — Remote access security
- [Update Strategy](updates.md) — System updates

