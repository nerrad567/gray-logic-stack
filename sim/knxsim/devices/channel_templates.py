"""Channel template helpers for multi-channel KNX device types."""

from __future__ import annotations

# Multi-channel device templates
# Defines how many channels each multi-channel device type has and their group objects
MULTI_CHANNEL_TEMPLATES = {
    "switch_actuator_2fold": {
        "channel_count": 2,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CWU"},
            {"name": "switch_status", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"on": False},
    },
    "switch_actuator_4fold": {
        "channel_count": 4,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CWU"},
            {"name": "switch_status", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"on": False},
    },
    "switch_actuator_6fold": {
        "channel_count": 6,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CWU"},
            {"name": "switch_status", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"on": False},
    },
    "switch_actuator_8fold": {
        "channel_count": 8,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CWU"},
            {"name": "switch_status", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"on": False},
    },
    # Heating actuators (proportional valve control for UFH manifolds)
    "heating_actuator_2fold": {
        "channel_count": 2,
        "group_objects": [
            {"name": "valve", "dpt": "5.001", "flags": "CWU"},
            {"name": "valve_status", "dpt": "5.001", "flags": "CRT"},
        ],
        "state_fields": {"position": 0},
    },
    "heating_actuator_4fold": {
        "channel_count": 4,
        "group_objects": [
            {"name": "valve", "dpt": "5.001", "flags": "CWU"},
            {"name": "valve_status", "dpt": "5.001", "flags": "CRT"},
        ],
        "state_fields": {"position": 0},
    },
    "heating_actuator_6fold": {
        "channel_count": 6,
        "group_objects": [
            {"name": "valve", "dpt": "5.001", "flags": "CWU"},
            {"name": "valve_status", "dpt": "5.001", "flags": "CRT"},
        ],
        "state_fields": {"position": 0},
    },
    "heating_actuator_8fold": {
        "channel_count": 8,
        "group_objects": [
            {"name": "valve", "dpt": "5.001", "flags": "CWU"},
            {"name": "valve_status", "dpt": "5.001", "flags": "CRT"},
        ],
        "state_fields": {"position": 0},
    },
    "dimmer_actuator_2fold": {
        "channel_count": 2,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CWU"},
            {"name": "switch_status", "dpt": "1.001", "flags": "CRT"},
            {"name": "brightness", "dpt": "5.001", "flags": "CWU"},
            {"name": "brightness_status", "dpt": "5.001", "flags": "CRT"},
        ],
        "state_fields": {"on": False, "brightness": 0},
    },
    "dimmer_actuator_4fold": {
        "channel_count": 4,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CWU"},
            {"name": "switch_status", "dpt": "1.001", "flags": "CRT"},
            {"name": "brightness", "dpt": "5.001", "flags": "CWU"},
            {"name": "brightness_status", "dpt": "5.001", "flags": "CRT"},
        ],
        "state_fields": {"on": False, "brightness": 0},
    },
    "blind_actuator_2fold": {
        "channel_count": 2,
        "group_objects": [
            {"name": "move", "dpt": "1.008", "flags": "CW"},
            {"name": "stop", "dpt": "1.017", "flags": "CW"},
            {"name": "position", "dpt": "5.001", "flags": "CWU"},
            {"name": "position_status", "dpt": "5.001", "flags": "CRT"},
        ],
        "state_fields": {"position": 0, "moving": False},
    },
    "blind_actuator_4fold": {
        "channel_count": 4,
        "group_objects": [
            {"name": "move", "dpt": "1.008", "flags": "CW"},
            {"name": "stop", "dpt": "1.017", "flags": "CW"},
            {"name": "position", "dpt": "5.001", "flags": "CWU"},
            {"name": "position_status", "dpt": "5.001", "flags": "CRT"},
        ],
        "state_fields": {"position": 0, "moving": False},
    },
    "push_button_2fold": {
        "channel_count": 2,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"pressed": False},
    },
    "push_button_4fold": {
        "channel_count": 4,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"pressed": False},
    },
    "push_button_6fold": {
        "channel_count": 6,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"pressed": False},
    },
    "binary_input_4fold": {
        "channel_count": 4,
        "group_objects": [
            {"name": "state", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"active": False},
    },
    "binary_input_8fold": {
        "channel_count": 8,
        "group_objects": [
            {"name": "state", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"active": False},
    },
}

CHANNEL_LABELS = ["A", "B", "C", "D", "E", "F", "G", "H"]


def _default_channel_name(device_type: str) -> str:
    """Generate default channel name based on device type."""
    type_lower = device_type.lower()

    if "light" in type_lower or "dimmer" in type_lower:
        return "Light Output"
    if "blind" in type_lower or "shutter" in type_lower:
        return "Blind Output"
    if "switch" in type_lower and "actuator" in type_lower:
        return "Switch Output"
    if "sensor" in type_lower or "presence" in type_lower:
        return "Sensor"
    if "thermostat" in type_lower:
        return "Thermostat"
    if "button" in type_lower:
        return "Button"
    if "input" in type_lower:
        return "Input"

    return "Channel A"


def _generate_channels_from_template(device_type: str, group_addresses: dict = None) -> list[dict]:
    """Generate channel structure from multi-channel template.

    If group_addresses is provided (from config.yaml), distributes them across channels.
    Otherwise creates channels with empty GA assignments.
    """
    template = MULTI_CHANNEL_TEMPLATES.get(device_type)
    if not template:
        return None  # Not a multi-channel type

    channel_count = template["channel_count"]
    channels = []

    for i in range(channel_count):
        label = CHANNEL_LABELS[i] if i < len(CHANNEL_LABELS) else str(i + 1)

        # Build group_objects for this channel
        group_objects = {}
        for go in template["group_objects"]:
            # Check if we have a GA for this channel's group object
            # Convention: channel_A_switch, channel_B_switch OR button_1, button_2
            ga = None
            if group_addresses:
                # Try various naming conventions
                for key_pattern in [
                    f"channel_{label.lower()}_{go['name']}",
                    f"{label.lower()}_{go['name']}",
                    f"{go['name']}_{label.lower()}",
                    f"{go['name']}_{i + 1}",
                    f"button_{i + 1}" if go["name"] == "switch" else None,
                    f"led_{i + 1}" if go["name"] == "switch_status" else None,
                ]:
                    if key_pattern and key_pattern in group_addresses:
                        ga = group_addresses[key_pattern]
                        break

            # Handle extended GA format: {"ga": "1/2/3", "dpt": "5.001"}
            if isinstance(ga, dict):
                ga_str = ga.get("ga")
                dpt = ga.get("dpt", go["dpt"])
            else:
                ga_str = ga
                dpt = go["dpt"]

            group_objects[go["name"]] = {
                "ga": ga_str,
                "dpt": dpt,
                "flags": go["flags"],
            }

        channels.append(
            {
                "id": label,
                "name": f"Channel {label}",
                "group_objects": group_objects,
                "state": dict(template.get("state_fields", {})),
                "initial_state": dict(template.get("state_fields", {})),
                "parameters": {},
            }
        )

    return channels
