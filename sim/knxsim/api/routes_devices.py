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
    from devices.template_device import TemplateDevice
    from dpt.codec import DPTCodec
    from knxip import constants as C
    from knxip import frames

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
    telegrams_sent = []

    # Handle template devices (e.g., wall switches with buttons)
    # These send GroupWrite telegrams to their configured GAs
    if isinstance(device, TemplateDevice):
        ga = device.group_addresses.get(command)
        if ga is not None and premise.server:
            # Get DPT info from template definition
            # _ga_info is now a list of tuples: [(slot_name, field, dpt, direction), ...]
            info_list = device._ga_info.get(ga)
            if info_list:
                # Find the matching slot for this command
                dpt = None
                field = command  # Default to command name as field
                for slot_name, slot_field, slot_dpt, direction in info_list:
                    if slot_name == command:
                        field = slot_field
                        dpt = slot_dpt
                        break
                # Fallback to first entry if no exact match
                if dpt is None and info_list:
                    _, field, dpt, _ = info_list[0]

                # Encode value using the template's DPT
                payload = DPTCodec.encode(dpt, value)
                # Build GroupWrite telegram (simulates button press sending to bus)
                cemi = frames.encode_cemi(
                    msg_code=0x29,  # L_Data.ind
                    src=device.individual_address,
                    dst=ga,
                    apci=C.APCI_GROUP_WRITE,
                    payload=payload,
                )
                # Send to KNX bus - other devices listening on this GA will react
                premise._send_telegram_with_hook(cemi)
                # Also dispatch to local devices on same GA (simulate bus behavior)
                premise._dispatch_telegram(ga, payload)
                telegrams_sent.append(command)
                # Update our own state
                device.state[field] = value

        # Trigger state change callback
        if premise._on_state_change:
            premise._on_state_change(premise_id, device_id, dict(device.state))

        return {
            "status": "ok",
            "device_id": device_id,
            "command": command,
            "value": value,
            "telegrams_sent": telegrams_sent,
        }

    # Standard device handling (lights, blinds, sensors, etc.)
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


@router.post("/{device_id}/channels/{channel_id}/command", status_code=200)
def send_channel_command(premise_id: str, device_id: str, channel_id: str, body: DeviceCommand):
    """Send a command to a specific channel of a multi-channel device.
    
    Similar to send_command but targets a specific channel's group objects.
    The channel_id should match a channel's 'id' field (e.g., "A", "B").
    """
    manager = router.app.state.manager
    premise = manager.premises.get(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found or not running")

    device = premise.devices.get(device_id)
    if not device:
        raise HTTPException(status_code=404, detail="Device not found")

    # Get the device from DB to access channels
    db_device = manager.get_device(device_id)
    if not db_device:
        raise HTTPException(status_code=404, detail="Device not found in database")

    channels = db_device.get("channels", [])
    channel = next((c for c in channels if c.get("id") == channel_id), None)
    if not channel:
        raise HTTPException(status_code=404, detail=f"Channel {channel_id} not found")

    command = body.command.lower()
    value = body.value
    telegrams_sent = []

    # Map command to state field (depends on device type)
    device_type = db_device.get("type", "")
    is_push_button = "push_button" in device_type or "pushbutton" in device_type
    
    if is_push_button and command == "switch":
        # Push buttons use 'pressed' state, not 'on'
        field = "pressed"
    else:
        field_map = {
            "switch": "on",
            "brightness": "brightness",
            "position": "position",
            "slat": "tilt",
            "state": "active",
        }
        field = field_map.get(command, command)

    # Update channel state in DB
    manager.db.update_channel_state(device_id, channel_id, {field: value})

    # Find the GA for this command in the channel's group_objects
    group_objects = channel.get("group_objects", {})
    
    # Look for matching group object (command or command_status)
    ga_name = None
    ga_str = None
    for go_name, go_data in group_objects.items():
        if go_name == command or go_name.replace("_status", "") == command:
            if isinstance(go_data, dict) and go_data.get("ga"):
                ga_name = go_name
                ga_str = go_data["ga"]
                break

    # Send telegram if we found a GA
    if ga_str and premise.server:
        from devices.base import encode_dpt1, encode_dpt5, encode_dpt9
        from knxip import frames, constants as C

        # Parse GA
        ga = frames.parse_group_address(ga_str)

        # Encode value based on command type
        if command in ("switch", "state"):
            payload = encode_dpt1(bool(value))
        elif command in ("brightness", "position", "slat"):
            payload = encode_dpt5(int(value))
        else:
            payload = None

        if payload is not None:
            # Build and send telegram
            cemi = frames.encode_cemi(
                msg_code=0x29,  # L_Data.ind
                src=device.individual_address,
                dst=ga,
                apci=C.APCI_GROUP_WRITE,
                payload=payload,
            )
            premise._send_telegram_with_hook(cemi)
            # Also dispatch to local devices (simulate KNX bus behavior)
            premise._dispatch_telegram(ga, payload)
            telegrams_sent.append(ga_name)

    # Update runtime device state as well (for consistency)
    device.state[field] = value

    # Trigger state change callback
    if premise._on_state_change:
        premise._on_state_change(premise_id, device_id, dict(device.state))

    return {
        "status": "ok",
        "device_id": device_id,
        "channel_id": channel_id,
        "command": command,
        "value": value,
        "telegrams_sent": telegrams_sent,
    }
