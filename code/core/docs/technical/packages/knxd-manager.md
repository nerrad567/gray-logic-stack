# knxd Manager Package Design

> `internal/knxd/` — Managed knxd daemon for Gray Logic KNX integration

## Purpose

Provides automatic lifecycle management for the knxd daemon, including:
- Configuration-driven startup (no manual /etc/knxd.conf editing)
- Multi-layer health monitoring (Layers 0-4)
- Automatic restart on failure with exponential backoff
- USB device reset for recovery from LIBUSB_ERROR_BUSY
- PID file management to prevent duplicate instances

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         knxd Manager                                     │
│                                                                          │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐   │
│  │     Manager      │───▶│  process.Manager │───▶│   knxd process   │   │
│  │   (manager.go)   │    │   (../process/)  │    │   (subprocess)   │   │
│  │                  │    │                  │    │                  │   │
│  │ • Start/Stop     │    │ • Lifecycle      │    │ • TCP listener   │   │
│  │ • Health checks  │    │ • Auto-restart   │    │ • KNX bus access │   │
│  │ • USB reset      │    │ • Log capture    │    │                  │   │
│  └──────────────────┘    └──────────────────┘    └──────────────────┘   │
│           │                                                              │
│           ▼                                                              │
│  ┌──────────────────┐    ┌──────────────────┐                           │
│  │     Config       │    │  Health Checks   │                           │
│  │   (config.go)    │    │   (Layers 0-4)   │                           │
│  │                  │    │                  │                           │
│  │ • Backend types  │    │ • USB presence   │                           │
│  │ • Address valid. │    │ • Process state  │                           │
│  │ • BuildArgs()    │    │ • Group read     │                           │
│  └──────────────────┘    │ • Device desc.   │                           │
│                          └──────────────────┘                           │
└─────────────────────────────────────────────────────────────────────────┘
```

### Key Types

| Type | Purpose |
|------|---------|
| `Manager` | Main orchestrator for knxd lifecycle |
| `Config` | Configuration for knxd daemon |
| `BackendConfig` | KNX bus connection settings (USB/IPT/IP) |
| `BackendType` | Enum: `usb`, `ipt`, `ip` |
| `HealthError` | Health check failure with recoverability info |
| `Stats` | Runtime statistics |

### External Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `internal/process` | - | Generic subprocess management |

---

## How It Works

### Initialization

```go
cfg := knxd.Config{
    Managed:         true,
    Binary:          "/usr/bin/knxd",
    PhysicalAddress: "0.0.1",
    ClientAddresses: "0.0.2:8",
    Backend: knxd.BackendConfig{
        Type: knxd.BackendUSB,
        USBVendorID:  "0e77",
        USBProductID: "0104",
    },
}

manager, err := knxd.NewManager(cfg)
if err != nil {
    return err
}
manager.SetLogger(logger)

// Start knxd
if err := manager.Start(ctx); err != nil {
    return err
}
defer manager.Stop()

// knxd is now ready on tcp://localhost:6720
fmt.Println(manager.ConnectionURL())
```

### Multi-Layer Health Checks

The manager implements a layered health check approach for comprehensive monitoring:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Health Check Pipeline                         │
│                                                                  │
│  Layer 0: USB Presence ──┬─ FAIL ──▶ Not Recoverable (no restart)│
│           (~5ms)         │                                       │
│                          └─ PASS                                 │
│                              │                                   │
│  Layer 1: Process State ─┬─ FAIL ──▶ Recoverable (restart)       │
│           (~0.1ms)       │                                       │
│                          └─ PASS                                 │
│                              │                                   │
│  Layer 4: DeviceDesc ────┬─ FAIL ──┐                             │
│           (~100-500ms)   │         │                             │
│                          └─ PASS ──│──▶ SUCCESS                  │
│                                    ▼                             │
│  Layer 3: GroupRead ─────┬─ FAIL ──▶ Recoverable (restart)       │
│           (~100-500ms)   │                                       │
│                          └─ PASS ──▶ SUCCESS                     │
└─────────────────────────────────────────────────────────────────┘
```

| Layer | Check | What It Detects | Recoverable? |
|-------|-------|-----------------|--------------|
| 0 | USB presence (`lsusb`) | Hardware disconnection | No |
| 1 | Process state (`/proc/stat`) | SIGSTOP, zombie, dead | Yes |
| 3 | GroupValue_Read | Interface failure, bus issues | Yes |
| 4 | DeviceDescriptor_Read | End-to-end bus health | Yes |

**Why Layer 3 & 4?**
- Layer 4 (DeviceDescriptor_Read) works with all certified KNX hardware
- Layer 3 (GroupValue_Read) is a fallback for simulators (KNX Virtual) that don't support T_Connection

### USB Device Reset

For USB backends, the manager can reset the device to recover from:
- `LIBUSB_ERROR_BUSY` conditions
- USB interface lockups
- Driver-level issues

```go
// Automatic reset before restart attempts
Backend: knxd.BackendConfig{
    Type:            knxd.BackendUSB,
    USBVendorID:     "0e77",
    USBProductID:    "0104",
    USBResetOnRetry: true,  // Reset before each restart
}

// Proactive reset on bus health failure
Backend: knxd.BackendConfig{
    USBResetOnBusFailure: true,  // Reset when Layer 3/4 fail
}
```

**Requires:**
- `usbreset` utility (standard on most Linux systems)
- udev rule for write access:
  ```
  SUBSYSTEM=="usb", ATTR{idVendor}=="0e77", ATTR{idProduct}=="0104", MODE="0666"
  ```

---

## Configuration

### Backend Types

```yaml
protocols:
  knx:
    knxd:
      backend:
        # USB Interface (Weinzierl KNX-USB, etc.)
        type: "usb"
        usb_vendor_id: "0e77"
        usb_product_id: "0104"
        usb_reset_on_retry: true
        usb_reset_on_bus_failure: true

        # IP Tunnelling (KNX/IP gateway)
        # type: "ipt"
        # host: "192.168.1.100"
        # port: 3671

        # IP Routing (multicast)
        # type: "ip"
        # multicast_address: "224.0.23.12"
        # interface: "eth0"
```

### Full Configuration

```yaml
protocols:
  knx:
    enabled: true
    knxd:
      managed: true                    # Gray Logic manages knxd lifecycle
      binary: "/usr/bin/knxd"
      physical_address: "0.0.1"        # knxd's address on the bus
      client_addresses: "0.0.2:8"      # Pool for clients (8 addresses)

      # Backend configuration
      backend:
        type: "usb"
        usb_vendor_id: "0e77"
        usb_product_id: "0104"
        usb_reset_on_retry: true
        usb_reset_on_bus_failure: true

      # Restart behaviour
      restart_on_failure: true
      restart_delay_seconds: 5
      max_restart_attempts: 10

      # Health monitoring
      health_check_interval: 30s
      health_check_device_address: "1/7/0"  # Optional: PSU diagnostic
      health_check_device_timeout: 3s

      # Logging
      log_level: 0                     # 0-9 (0 = minimal)
```

---

## Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| **Subprocess management** | knxd is a mature daemon; reimplementing EIB protocol would be complex | Embedded library, custom implementation |
| **Multi-layer health checks** | Different layers catch different failures at different speeds | Single TCP ping check |
| **USB reset capability** | USB interfaces can lock up; reset recovers without reboot | Manual intervention, ignore |
| **PID file management** | Prevents duplicate knxd instances which corrupt bus state | Process name matching |
| **RecoverableError interface** | Hardware issues (USB unplugged) shouldn't trigger restart loops | Always restart |

---

## Interactions

### Dependencies

| Package | Purpose |
|---------|---------|
| `internal/process` | Generic subprocess lifecycle management |

### Dependents

| Package | Purpose |
|---------|---------|
| `cmd/graylogic/main.go` | Starts knxd manager at application boot |
| `internal/bridges/knx` | Connects to knxd for KNX bus communication |

---

## Error Handling

Errors include layer information and recoverability:

```go
// HealthError includes layer and recoverability
err := manager.HealthCheck(ctx)
var healthErr *knxd.HealthError
if errors.As(err, &healthErr) {
    fmt.Printf("Layer %d failed: %v (recoverable: %v)\n",
        healthErr.Layer, healthErr.Err, healthErr.Recoverable)
}
```

---

## Thread Safety

The Manager is safe for concurrent use:
- Health checks can run while process is monitored
- Stats() can be called from any goroutine
- Stop() is safe to call multiple times

---

## Testing

```bash
# Run tests (requires knxd installed for some tests)
cd code/core
go test -v ./internal/knxd/...

# With race detector
go test -race ./internal/knxd/...
```

Test scenarios:
- Configuration validation (addresses, backends)
- Backend argument building
- Address parsing (group, individual)
- Health check error types

---

## Related Documents

- [Process Manager](./process-manager.md) — Generic subprocess management
- [KNX Bridge](./knx-bridge.md) — KNX protocol bridge using knxd
- [KNX Protocol](../../../docs/protocols/knx.md) — KNX protocol specification
- [configs/config.yaml](../../../configs/config.yaml) — Configuration reference
