# Container Build Requirements

This file documents critical requirements for building the Gray Logic Core container image.

## Required Packages

The container image **MUST** include these packages for USB KNX interface support:

```dockerfile
RUN apt-get update && apt-get install -y --no-install-recommends \
    knxd \
    usbutils \
    libusb-1.0-0 \
    libev4 \
    && rm -rf /var/lib/apt/lists/*
```

### Package Purposes

| Package | Utility | Purpose |
|---------|---------|---------|
| `knxd` | `/usr/bin/knxd` | KNX daemon for bus communication |
| `usbutils` | `/usr/bin/lsusb` | USB device presence check (health Layer 0) |
| `usbutils` | `/usr/bin/usbreset` | USB device reset for error recovery |
| `libusb-1.0-0` | library | USB access library for knxd |
| `libev4` | library | Event loop library for knxd |

## Why lsusb and usbreset are Critical

### lsusb (Health Check Layer 0)

The `lsusb` utility is used by the health check system to verify USB device presence.
This is the fastest check (Layer 0) and immediately detects hardware disconnection.

**Without lsusb:** Health checks cannot detect USB disconnection, leading to
misleading error messages and unnecessary restart attempts.

### usbreset (Error Recovery)

The `usbreset` utility allows Gray Logic to recover from `LIBUSB_ERROR_BUSY` errors
without requiring root privileges or sysfs access. This is used by the knxd manager
to reset the USB device before restart attempts.

**Without usbreset:** Manual intervention required to recover from USB errors.
**With usbreset:** Automatic recovery, no downtime.

## Verification

After building the image, verify:

```bash
docker run --rm graylogic/core:latest which lsusb
# Should output: /usr/bin/lsusb

docker run --rm graylogic/core:latest which usbreset
# Should output: /usr/bin/usbreset

docker run --rm graylogic/core:latest which knxd
# Should output: /usr/bin/knxd
```

## Host Requirements

See `docs/deployment/usb-device-management.md` for:
- udev rules setup
- Docker device passthrough
- Production deployment configuration

## Quick Reference

```yaml
# docker-compose.yml - USB device access
services:
  graylogic-core:
    devices:
      - /dev/bus/usb:/dev/bus/usb
    device_cgroup_rules:
      - 'c 189:* rmw'
    group_add:
      - "1000"  # graylogic group GID
```
