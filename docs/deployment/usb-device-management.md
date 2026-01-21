---
title: USB Device Management
version: 1.0.0
status: active
dependencies:
  - protocols/knx-bridge.md
  - deployment/container-setup.md
---

# USB Device Management

This document covers the setup and management of USB devices (primarily KNX-USB interfaces) for Gray Logic deployments, including containerized environments.

## Overview

Gray Logic can communicate with KNX bus via USB interfaces (e.g., Weinzierl KNX-USB Interface). The knxd manager includes automatic USB device reset capability to recover from common issues like `LIBUSB_ERROR_BUSY` without requiring root privileges.

## Supported USB Devices

| Manufacturer | Model | Vendor ID | Product ID | Notes |
|-------------|-------|-----------|------------|-------|
| Weinzierl | KNX-USB Interface | 0e77 | 0104 | Primary supported device |
| ABB | USB/S 1.1 | 147b | 2110 | Verify with ETS |
| Siemens | USB/KNX Interface | 147b | 1102 | Verify with ETS |

## Host System Requirements

### 1. udev Rules

udev rules grant the Gray Logic process access to USB devices without root privileges.

**Create the rule file:**
```bash
sudo nano /etc/udev/rules.d/90-graylogic-knx.rules
```

**Content (for Weinzierl device):**
```udev
# Gray Logic KNX-USB Device Access
# Weinzierl KNX-USB Interface
SUBSYSTEM=="usb", ATTR{idVendor}=="0e77", ATTR{idProduct}=="0104", \
  GROUP="graylogic", MODE="0660", \
  SYMLINK+="knx-usb", \
  TAG+="uaccess"

# ABB USB/S 1.1 (if used)
# SUBSYSTEM=="usb", ATTR{idVendor}=="147b", ATTR{idProduct}=="2110", \
#   GROUP="graylogic", MODE="0660", SYMLINK+="knx-usb-abb"
```

**Apply the rules:**
```bash
sudo udevadm control --reload-rules
sudo udevadm trigger
```

**Verify:**
```bash
ls -la /dev/bus/usb/*/  # Check permissions
ls -la /dev/knx-usb     # Check symlink exists
```

### 2. User Group

Create the graylogic group and add the service user:
```bash
sudo groupadd -r graylogic
sudo usermod -aG graylogic $USER  # For development
# For production, the container user will need this group
```

### 3. Required Utilities

The `usbreset` utility is required for automatic USB device recovery:
```bash
# Debian/Ubuntu
sudo apt-get install usbutils

# Verify installation
which usbreset  # Should output /usr/bin/usbreset
```

## Container Deployment

### Required Packages in Container Image

The container image MUST include these packages:

```dockerfile
# Dockerfile snippet for Gray Logic Core
FROM debian:bookworm-slim

# USB device management utilities
RUN apt-get update && apt-get install -y --no-install-recommends \
    usbutils \
    libusb-1.0-0 \
    && rm -rf /var/lib/apt/lists/*

# Verify usbreset is available
RUN which usbreset
```

**Package purposes:**
| Package | Purpose |
|---------|---------|
| `usbutils` | Provides `usbreset`, `lsusb` utilities |
| `libusb-1.0-0` | USB library required by knxd |

### Docker Compose Configuration

```yaml
# docker-compose.yml
version: '3.8'

services:
  graylogic-core:
    image: graylogic/core:latest
    container_name: graylogic-core
    restart: unless-stopped

    # CRITICAL: USB device passthrough
    devices:
      - /dev/bus/usb:/dev/bus/usb

    # Alternative: Pass specific device only
    # devices:
    #   - /dev/knx-usb:/dev/knx-usb

    # Required for USB access
    privileged: false  # We don't need full privileges

    # Grant USB device access via cgroup
    device_cgroup_rules:
      - 'c 189:* rmw'  # USB devices (major 189)

    # Group mapping for udev rules
    group_add:
      - "graylogic"  # Must match host group GID

    volumes:
      - ./configs:/app/configs:ro
      - ./data:/app/data

    environment:
      - TZ=Europe/London
```

### Docker Run Command

```bash
docker run -d \
  --name graylogic-core \
  --device /dev/bus/usb \
  --device-cgroup-rule='c 189:* rmw' \
  --group-add $(getent group graylogic | cut -d: -f3) \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/data:/app/data \
  graylogic/core:latest
```

### Podman Configuration

```bash
podman run -d \
  --name graylogic-core \
  --device /dev/bus/usb \
  --group-add keep-groups \
  -v ./configs:/app/configs:ro \
  -v ./data:/app/data \
  graylogic/core:latest
```

## Configuration

### Gray Logic config.yaml

```yaml
protocols:
  knx:
    enabled: true
    knxd:
      managed: true
      backend:
        type: "usb"
        # USB device identification for reset operations
        usb_vendor_id: "0e77"
        usb_product_id: "0104"
        # Enable automatic USB reset before restart attempts
        usb_reset_on_retry: true
```

### How USB Reset Works

1. **knxd crashes** (e.g., USB disconnect, bus error)
2. **Process Manager** detects unexpected exit
3. **OnRestart callback** fires before restart attempt
4. **resetUSBDevice()** executes:
   ```bash
   usbreset 0e77:0104
   ```
5. **USB device resets** via USBDEVFS_RESET ioctl
6. **500ms delay** for device reinitialisation
7. **knxd restarts** with fresh USB connection

This process does NOT require root because:
- `usbreset` uses ioctl (not sysfs bind/unbind)
- udev rules grant write access to the device

## Troubleshooting

### LIBUSB_ERROR_BUSY

**Symptoms:** knxd fails to start with "USBLowLevelDriver: setup config: LIBUSB_ERROR_BUSY"

**Causes:**
1. Another process has claimed the USB device
2. Previous knxd instance didn't clean up properly
3. System knxd.service is running

**Solutions:**
```bash
# Check what's using the device
lsof /dev/bus/usb/002/004  # Adjust path as needed

# Kill stray knxd processes
sudo pkill -9 knxd

# Disable system knxd if installed
sudo systemctl disable knxd.service
sudo systemctl disable knxd.socket

# Manual USB reset (if automatic fails)
usbreset 0e77:0104
```

### Permission Denied

**Symptoms:** "Permission denied" when accessing USB device

**Solutions:**
```bash
# Verify udev rules are loaded
sudo udevadm control --reload-rules
sudo udevadm trigger

# Check device permissions
ls -la /dev/bus/usb/002/  # Adjust bus number

# Verify user is in graylogic group
groups $USER

# Re-login or use newgrp to pick up new group
newgrp graylogic
```

### Container Can't See USB Device

**Symptoms:** Device not visible inside container

**Solutions:**
```bash
# Verify device is passed through
docker exec graylogic-core lsusb

# Check cgroup rules
docker inspect graylogic-core | grep -A5 DeviceCgroupRules

# Ensure /dev/bus/usb is mounted
docker exec graylogic-core ls -la /dev/bus/usb/
```

### USB Device Disconnected

**Symptoms:** Device disappears from system

**Solutions:**
```bash
# Check USB bus for device
lsusb -d 0e77:0104

# Check dmesg for USB errors
dmesg | tail -50 | grep -i usb

# Physical checks:
# - USB cable secure
# - Try different USB port
# - Check hub power (if using hub)
```

## Deployment Checklist

Before deploying Gray Logic with USB KNX interface:

- [ ] USB device physically connected and visible (`lsusb`)
- [ ] udev rules installed (`/etc/udev/rules.d/90-graylogic-knx.rules`)
- [ ] udev rules reloaded (`udevadm control --reload-rules && udevadm trigger`)
- [ ] graylogic group created (`getent group graylogic`)
- [ ] Service user in graylogic group
- [ ] usbreset utility installed (`which usbreset`)
- [ ] System knxd.service disabled (if installed)
- [ ] Container image includes usbutils package
- [ ] Docker/Podman device passthrough configured
- [ ] Config has correct vendor/product IDs
- [ ] Config has `usb_reset_on_retry: true`
- [ ] Test: Start Gray Logic and verify KNX bridge connects
- [ ] Test: Simulate restart and verify USB reset works

## Security Considerations

1. **Avoid world-writable permissions** (MODE="0666") in production
2. **Use group-based access** (GROUP="graylogic", MODE="0660")
3. **Don't run containers as root** - use proper device passthrough
4. **Audit USB access** - only grant access to required devices

## Related Documents

- [KNX Bridge Protocol](../protocols/knx-bridge.md)
- [Container Setup](./container-setup.md)
- [knxd Setup Guide](../protocols/knxd-setup.md)
