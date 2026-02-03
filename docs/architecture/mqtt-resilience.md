---
title: MQTT Resilience Specification
version: 1.0.0
status: active
last_updated: 2026-01-17
depends_on:
  - protocols/mqtt.md
  - architecture/core-internals.md
  - architecture/decisions/002-mqtt-internal-bus.md
---

# MQTT Resilience Specification

This document specifies how Gray Logic handles MQTT broker failures, message persistence, and recovery procedures to minimize automation disruption.

---

## Overview

MQTT is a single point of failure by design (see [ADR-002](decisions/002-mqtt-internal-bus.md)). This document specifies the mitigations that make this acceptable for residential/commercial deployments.

**Design Goals:**
- Broker restarts within 10 seconds (watchdog)
- Message loss window < 30 seconds for any single failure
- No manual intervention required for recovery
- Physical controls always work regardless of MQTT state

---

## Message Persistence

### What Is Persisted

Mosquitto persistence stores:

| Data Type | Persisted | Recovery Behavior |
|-----------|-----------|-------------------|
| Retained messages | Yes | Restored on broker restart |
| QoS 1/2 in-flight (to offline clients) | Yes | Redelivered when client reconnects |
| QoS 1/2 in-flight (broker crash) | No | Lost if broker crashes mid-delivery |
| QoS 0 messages | No | Fire-and-forget by design |
| Client sessions (clean_session=false) | Yes | Subscriptions restored |

### Mosquitto Persistence Configuration

**`/etc/mosquitto/mosquitto.conf`** (additions to base config):

```conf
# === PERSISTENCE ===
persistence true
persistence_location /var/lib/mosquitto/

# Persist every 60 seconds OR every 100 messages (whichever first)
autosave_interval 60
autosave_on_changes true

# Queue limits for offline clients
max_queued_messages 1000
max_queued_bytes 0          # 0 = use max_queued_messages only
```

### Persistence File Location

```
/var/lib/mosquitto/
├── mosquitto.db           # Main persistence file
└── mosquitto.db.backup    # Automatic backup before writes
```

**Backup strategy:** Include `/var/lib/mosquitto/` in system backups. The persistence file is typically < 1MB for a 500-device installation.

---

## Message Loss Budget

### Acceptable Loss Window

| Scenario | Max Downtime | Consequence |
|----------|--------------|-------------|
| Broker restart (clean) | < 5 seconds | QoS 1 messages queued, no loss |
| Broker crash + restart | < 10 seconds | In-flight QoS 1 may be lost |
| Broker hardware failure | < 30 seconds | Commands during window lost |

**Design constraint:** Scene commands lost during a 10-second broker outage should not result in dangerous states. This is ensured by:
1. Physical controls work independently of software
2. Safety-critical systems (fire, e-stop) don't rely on MQTT
3. Partial scene execution is visually obvious (user re-triggers)

### Unacceptable Scenarios

| Scenario | Why Unacceptable | Mitigation |
|----------|------------------|------------|
| Scene leaves room half-lit | Confusing but not dangerous | User re-activates scene |
| Climate setpoint lost | Comfort impact | Thermostat has local setpoint memory |
| Security arm command lost | Security gap | Security panel has physical keypad backup |
| Frost protection command lost | **Pipe damage** | Frost protection runs on thermostat, not Core |

---

## QoS Policy Enforcement

### Required QoS Levels by Topic

| Topic Pattern | Required QoS | Enforcement |
|---------------|--------------|-------------|
| `graylogic/command/+/+` | QoS 1 | Core publishes QoS 1; bridges must subscribe QoS 1 |
| `graylogic/state/+/+` | QoS 1 | Bridges must publish QoS 1 |
| `graylogic/health/+` | QoS 1 | Retained, QoS 1 |
| `graylogic/core/device/*/state` | QoS 1 | Retained, QoS 1 |
| `graylogic/system/time` | QoS 0 | Loss acceptable |
| `graylogic/system/status` | QoS 1 | Retained |

### Enforcement Mechanism

QoS is enforced by convention (bridges must follow spec). Runtime validation:

```go
// Core validates incoming messages
func (c *MQTTClient) validateMessage(msg mqtt.Message) error {
    topic := msg.Topic()
    qos := msg.Qos()

    // Log QoS violations for debugging (don't reject)
    if isCommandTopic(topic) && qos < 1 {
        c.logger.Warn("QoS violation: command topic received with QoS 0",
            "topic", topic,
            "expected_qos", 1,
            "actual_qos", qos)
        c.metrics.QoSViolations.Inc()
    }

    return nil // Process anyway; logging is sufficient
}
```

**Rationale:** Rejecting messages is worse than logging violations. A misconfigured bridge that sends QoS 0 should still work; we just want visibility.

---

## Bridge Reconnection Strategy

### Exponential Backoff with Jitter

When a bridge loses MQTT connection, it must use exponential backoff **with jitter** to prevent thundering herd:

```go
// Bridge reconnection logic
type ReconnectConfig struct {
    InitialDelay  time.Duration  // 1 second
    MaxDelay      time.Duration  // 30 seconds
    Multiplier    float64        // 2.0
    JitterFactor  float64        // 0.3 (+-30%)
}

func (b *Bridge) reconnectLoop() {
    delay := b.config.InitialDelay

    for {
        err := b.mqtt.Connect()
        if err == nil {
            b.onConnected()
            return
        }

        // Add jitter: +-30% of delay
        jitter := delay * time.Duration(b.config.JitterFactor * (rand.Float64()*2 - 1))
        sleepTime := delay + jitter

        b.logger.Info("MQTT reconnect failed, retrying",
            "delay", sleepTime,
            "error", err)

        time.Sleep(sleepTime)

        // Exponential backoff
        delay = time.Duration(float64(delay) * b.config.Multiplier)
        if delay > b.config.MaxDelay {
            delay = b.config.MaxDelay
        }
    }
}

func (b *Bridge) onConnected() {
    // 1. Re-subscribe to command topics
    b.mqtt.Subscribe("graylogic/command/"+b.protocol+"/+", b.handleCommand)

    // 2. Publish online status (overwrites LWT offline status)
    b.publishHealth(StatusOnline)

    // 3. Request state sync from protocol (get current device states)
    b.protocol.QueryAllDevices()

    // 4. Publish current states to Core
    for _, device := range b.devices {
        b.publishState(device.ID, device.CurrentState())
    }
}
```

### Reconnection Timeline (Worst Case)

With 5 bridges reconnecting after broker restart:

| Bridge | Attempt 1 | Attempt 2 | Attempt 3 | Connected By |
|--------|-----------|-----------|-----------|--------------|
| KNX | 0.8s | - | - | ~1s |
| DALI | 1.2s | - | - | ~1.5s |
| Modbus | 1.0s | - | - | ~1s |
| Audio | 0.7s | - | - | ~1s |
| Video | 1.3s | - | - | ~1.5s |

**Jitter ensures:** Bridges don't all hit the broker at exactly t=1s, t=2s, etc.

---

## Systemd Watchdog Configuration

### Mosquitto Service Unit

**`/etc/systemd/system/mosquitto.service.d/override.conf`**:

```ini
[Unit]
Description=Mosquitto MQTT Broker
Documentation=man:mosquitto(8)
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
NotifyAccess=main
ExecStart=
ExecStart=/usr/sbin/mosquitto -c /etc/mosquitto/mosquitto.conf
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=3
WatchdogSec=30

# Resource limits
LimitNOFILE=65535
MemoryMax=256M

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/mosquitto /var/log/mosquitto

[Install]
WantedBy=multi-user.target
```

### Key Settings

| Setting | Value | Rationale |
|---------|-------|-----------|
| `Restart=always` | Always restart on failure | Auto-recovery |
| `RestartSec=3` | 3 second delay before restart | Prevents restart loops |
| `WatchdogSec=30` | Watchdog timeout | systemd kills if unresponsive for 30s |
| `MemoryMax=256M` | Memory limit | Prevents runaway memory usage |

### Gray Logic Core Service Unit

**`/etc/systemd/system/graylogic-core.service`**:

```ini
[Unit]
Description=Gray Logic Core
After=network-online.target mosquitto.service
Wants=mosquitto.service
BindsTo=mosquitto.service

[Service]
Type=simple
ExecStart=/usr/local/bin/graylogic-core
Restart=always
RestartSec=5
WatchdogSec=60

# MQTT reconnection is handled internally, not by systemd
# Core survives brief MQTT outages

[Install]
WantedBy=multi-user.target
```

**Note:** `BindsTo=mosquitto.service` means Core stops if Mosquitto stops. This is intentional — Core without MQTT is minimally useful.

---

## Health Monitoring

### MQTT Broker Health Check

Core performs active health checks on the MQTT broker:

```go
type MQTTHealthChecker struct {
    client       mqtt.Client
    checkTopic   string  // "graylogic/system/healthcheck"
    lastResponse time.Time
    status       HealthStatus
}

func (h *MQTTHealthChecker) Check() HealthStatus {
    // Publish ping
    token := h.client.Publish(h.checkTopic, 1, false, time.Now().Unix())
    if !token.WaitTimeout(5 * time.Second) {
        return HealthStatus{
            Status:  "failed",
            Message: "MQTT publish timeout",
        }
    }

    if token.Error() != nil {
        return HealthStatus{
            Status:  "failed",
            Message: token.Error().Error(),
        }
    }

    // Check connection state
    if !h.client.IsConnected() {
        return HealthStatus{
            Status:  "failed",
            Message: "MQTT client disconnected",
        }
    }

    return HealthStatus{
        Status:    "healthy",
        Latency:   time.Since(h.lastResponse),
    }
}
```

### Health Status Thresholds

| Metric | Healthy | Degraded | Failed |
|--------|---------|----------|--------|
| Connection state | Connected | - | Disconnected |
| Publish latency | < 100ms | 100ms - 1s | > 1s or timeout |
| Bridge heartbeats | All received | 1-2 missing | > 2 missing |
| Queue depth | < 100 | 100-500 | > 500 |

### Prometheus Metrics

```
# MQTT health metrics
graylogic_mqtt_connected{broker="localhost:1883"} 1
graylogic_mqtt_publish_latency_seconds{quantile="0.99"} 0.015
graylogic_mqtt_messages_queued 12
graylogic_mqtt_reconnections_total 3
graylogic_mqtt_qos_violations_total{topic_pattern="command"} 0
```

---

## Recovery Procedures

### Automatic Recovery (Normal)

```
Broker crashes
     |
     v (0s)
systemd detects exit
     |
     v (3s)
systemd restarts Mosquitto
     |
     v (4s)
Mosquitto loads persistence file
     |
     v (5s)
Mosquitto ready, listening
     |
     v (5-8s)
Bridges detect connection, reconnect with jitter
     |
     v (6-10s)
Bridges publish current device states
     |
     v (10s)
System fully recovered, automation operational
```

### Manual Recovery (Persistence Corruption)

If `/var/lib/mosquitto/mosquitto.db` is corrupted:

```bash
# 1. Stop services
sudo systemctl stop graylogic-core mosquitto

# 2. Remove corrupted persistence
sudo rm /var/lib/mosquitto/mosquitto.db*

# 3. Restart broker (will start fresh)
sudo systemctl start mosquitto

# 4. Restart Core (will republish retained states)
sudo systemctl start graylogic-core

# 5. Bridges will reconnect and republish device states
# System recovers within 30 seconds
```

**Data loss:** Retained messages are lost. Bridges will republish current states on connect, so state is recovered. Historical messages (events, alerts) are lost.

---

## Testing Checklist

### Commissioning Tests

- [ ] Kill Mosquitto process; verify auto-restart within 10s
- [ ] Kill Mosquitto during scene execution; verify partial execution is obvious
- [ ] Disconnect bridge network; verify LWT publishes offline status
- [ ] Reconnect bridge; verify state sync occurs
- [ ] Simulate 5 bridges reconnecting simultaneously; verify no broker overload
- [ ] Corrupt mosquitto.db; verify manual recovery procedure works

### Monitoring Tests

- [ ] Verify `graylogic_mqtt_connected` metric updates on disconnect/reconnect
- [ ] Verify `graylogic_mqtt_reconnections_total` increments on each reconnect
- [ ] Verify health endpoint shows "degraded" during MQTT reconnection

---

## Related Documents

- [MQTT Protocol Specification](../protocols/mqtt.md) — Message formats and topics
- [ADR-002: MQTT as Internal Message Bus](decisions/002-mqtt-internal-bus.md) — Why MQTT
- [Core Internals Architecture](core-internals.md) — MQTT client implementation
- [Backup and Recovery](../resilience/backup.md) — System-wide backup strategy
