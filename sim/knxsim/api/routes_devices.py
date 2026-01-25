"""Device CRUD routes for the KNX Simulator Management API.

Devices are scoped to a premise. Adding a device via the API immediately
registers it on the live KNX bus — no restart required.
"""

from fastapi import APIRouter, HTTPException

from .models import (
    DeviceCommand,
    DeviceCreate,
    DevicePlacement,
    DeviceResponse,
    DeviceUpdate,
)

router = APIRouter(prefix="/api/v1/premises/{premise_id}/devices", tags=["devices"])


@router.get("", response_model=list[DeviceResponse])
def list_devices(premise_id: str):
    """List all devices in a premise with live state."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    return manager.list_devices(premise_id)


@router.get("/{device_id}", response_model=DeviceResponse)
def get_device(premise_id: str, device_id: str):
    """Get a single device with current state."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    device = manager.get_device(device_id)
    if not device or device.get("premise_id") != premise_id:
        raise HTTPException(status_code=404, detail="Device not found")
    # Enrich with live state
    live_premise = manager.premises.get(premise_id)
    if live_premise:
        live_dev = live_premise.devices.get(device_id)
        if live_dev:
            device["state"] = dict(live_dev.state)
    return device


@router.post("", response_model=DeviceResponse, status_code=201)
def create_device(premise_id: str, body: DeviceCreate):
    """Add a device to a premise (immediately active on KNX bus)."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    # Check duplicate
    existing = manager.get_device(body.id)
    if existing:
        raise HTTPException(status_code=409, detail="Device ID already exists")
    result = manager.add_device(premise_id, body.model_dump())
    if not result:
        raise HTTPException(status_code=500, detail="Failed to add device")
    return result


@router.patch("/{device_id}", response_model=DeviceResponse)
def update_device(premise_id: str, device_id: str, body: DeviceUpdate):
    """Update device configuration (group addresses, room assignment)."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    device = manager.get_device(device_id)
    if not device or device.get("premise_id") != premise_id:
        raise HTTPException(status_code=404, detail="Device not found")
    updates = body.model_dump(exclude_none=True)
    if not updates:
        return device
    result = manager.update_device(premise_id, device_id, updates)
    if not result:
        raise HTTPException(status_code=500, detail="Failed to update device")
    return result


@router.patch("/{device_id}/placement", response_model=DeviceResponse)
def update_device_placement(premise_id: str, device_id: str, body: DevicePlacement):
    """Assign a device to a room for floor plan placement."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    device = manager.get_device(device_id)
    if not device or device.get("premise_id") != premise_id:
        raise HTTPException(status_code=404, detail="Device not found")
    result = manager.update_device_placement(device_id, body.room_id)
    if not result:
        raise HTTPException(status_code=500, detail="Failed to update placement")
    return result


@router.delete("/{device_id}", status_code=204)
def delete_device(premise_id: str, device_id: str):
    """Remove a device from the premise (immediately removed from KNX bus)."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    device = manager.get_device(device_id)
    if not device or device.get("premise_id") != premise_id:
        raise HTTPException(status_code=404, detail="Device not found")
    manager.remove_device(premise_id, device_id)


@router.post("/{device_id}/command", status_code=200)
def send_command(premise_id: str, device_id: str, body: DeviceCommand):
    """Send a command to a device (triggers state update and KNX telegram).

    This simulates a physical switch press: updates device state and sends
    a status telegram onto the KNX bus so connected systems (like Core) see
    the state change.

    Commands map to group addresses:
      - switch: on/off (bool)
      - brightness: 0-100 (int)
      - position: blind position 0-100 (int)
      - slat: blind slat angle 0-100 (int)
      - setpoint: thermostat setpoint in °C (float, DPT9)
      - scene: scene number 0-63 (int)
      - presence: motion detected (bool, DPT1)
      - lux: ambient light level in lux (float, DPT9)
    """
    from devices.base import encode_dpt1, encode_dpt5, encode_dpt9

    manager = router.app.state.manager
    premise = manager.premises.get(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found or not running")

    device = premise.devices.get(device_id)
    if not device:
        raise HTTPException(status_code=404, detail="Device not found")

    # Map command to state field and GA
    command = body.command.lower()
    value = body.value

    # Command to state field and status GA mapping
    field_map = {
        "switch": ("on", "switch_status"),
        "brightness": ("brightness", "brightness_status"),
        "position": ("position", "position_status"),
        "slat": ("tilt", "tilt_status"),
        "setpoint": ("setpoint", "setpoint_status"),
        "scene": ("scene", None),
        "presence": ("presence", "presence"),
        "lux": ("lux", "lux"),
    }

    mapping = field_map.get(command, (command, None))
    field, status_ga_name = mapping

    # Update device state
    device.state[field] = value

    # Send status telegram onto KNX bus (simulates physical device reporting state)
    telegrams_sent = []
    if status_ga_name and premise.server:
        status_ga = device.group_addresses.get(status_ga_name)
        if status_ga is not None:
            # Encode value based on command type
            if command in ("switch", "presence"):
                payload = encode_dpt1(bool(value))
            elif command in ("brightness", "position", "slat"):
                payload = encode_dpt5(int(value))
            elif command in ("lux", "setpoint"):
                payload = encode_dpt9(float(value))
            else:
                payload = None

            if payload is not None:
                # Build and send the status telegram
                cemi = device._make_indication(status_ga, payload)
                premise._send_telegram_with_hook(cemi)
                telegrams_sent.append(status_ga_name)

    # Also trigger state change callback (broadcasts to WebSocket + persists)
    if premise._on_state_change:
        premise._on_state_change(premise_id, device_id, dict(device.state))

    return {
        "status": "ok",
        "device_id": device_id,
        "command": command,
        "value": value,
        "telegrams_sent": telegrams_sent,
    }
