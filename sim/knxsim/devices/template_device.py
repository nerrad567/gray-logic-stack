"""TemplateDevice — generic KNX device driven by a YAML template.

Unlike the hard-coded device classes (LightSwitch, Sensor, etc.),
TemplateDevice uses the DPT codec to handle any group address
dynamically. The template definition specifies which GAs map to
which state fields and what DPT to use for encoding/decoding.

This enables adding new device types with just a YAML file —
no Python code required.
"""

import logging

from devices.base import BaseDevice
from dpt.codec import DPTCodec
from knxip import constants as C
from knxip import frames

logger = logging.getLogger("knxsim.template_device")


class TemplateDevice(BaseDevice):
    """A device whose behaviour is entirely defined by its template."""

    def __init__(
        self,
        device_id: str,
        individual_address: int,
        group_addresses: dict[str, int],
        initial_state: dict,
        template_def: dict | None = None,
    ):
        """Initialize a template-driven device.

        Args:
            device_id: Unique device identifier
            individual_address: KNX individual address (encoded)
            group_addresses: Mapping of slot_name → GA (encoded int)
            initial_state: Initial state dict
            template_def: Template group_addresses definition with DPT info:
                          {"switch_cmd": {"dpt": "1.001", "direction": "write"}, ...}
        """
        super().__init__(device_id, individual_address, group_addresses, initial_state)
        self._template_def = template_def or {}

        # Build reverse map: GA (int) → (field_name, dpt_id, direction)
        self._ga_info: dict[int, tuple[str, str, str]] = {}
        for slot_name, ga_int in group_addresses.items():
            slot_def = self._template_def.get(slot_name, {})
            dpt = slot_def.get("dpt", "1.001")
            direction = slot_def.get("direction", "write")
            # Derive state field name from slot name (strip _cmd/_status suffix)
            field = self._slot_to_field(slot_name)
            self._ga_info[ga_int] = (field, dpt, direction)

    def _slot_to_field(self, slot_name: str) -> str:
        """Convert a GA slot name to a state field name.

        Examples:
            switch_cmd → on (special case for switch/on_off)
            switch_status → on
            brightness_cmd → brightness
            position_status → position
            temperature → temperature
        """
        # Strip _cmd or _status suffix
        field = slot_name
        if field.endswith("_cmd"):
            field = field[:-4]
        elif field.endswith("_status"):
            field = field[:-7]

        # Map common KNX names to state field names
        field_map = {
            "switch": "on",
            "on_off": "on",
            "move": "moving",
            "stop": "stopped",
        }
        return field_map.get(field, field)

    def on_group_write(self, ga: int, payload: bytes) -> bytes | None:
        """Handle a GroupWrite telegram — decode value and update state."""
        info = self._ga_info.get(ga)
        if not info:
            return None

        field, dpt, direction = info

        # Decode the incoming value
        value = DPTCodec.decode(dpt, payload)
        self.state[field] = value

        logger.info(
            "%s ← GroupWrite %s=%s (DPT %s)",
            self.device_id,
            field,
            value,
            dpt,
        )

        # If this was a command GA, find and respond with the status GA
        if direction == "write":
            return self._build_status_response(field, dpt)

        return None

    def on_group_read(self, ga: int) -> bytes | None:
        """Handle a GroupRead — respond with current state value."""
        info = self._ga_info.get(ga)
        if not info:
            return None

        field, dpt, direction = info
        value = self.state.get(field)
        if value is None:
            return None

        # Encode the response
        payload = DPTCodec.encode(dpt, value)
        return self._build_cemi_response(ga, payload)

    def _build_status_response(self, field: str, dpt: str) -> bytes | None:
        """Build a GroupResponse for a status GA matching the given field."""
        # Find status GA for this field
        for _slot_name, ga_int in self.group_addresses.items():
            info = self._ga_info.get(ga_int)
            if not info:
                continue
            s_field, s_dpt, s_direction = info
            if s_field == field and s_direction == "status":
                value = self.state.get(field)
                if value is not None:
                    payload = DPTCodec.encode(s_dpt, value)
                    return self._build_cemi_response(ga_int, payload)
        return None

    def _build_cemi_response(self, ga: int, payload: bytes) -> bytes:
        """Build a cEMI GroupResponse frame."""
        return frames.encode_cemi(
            msg_code=0x29,  # L_Data.ind
            src=self.individual_address,
            dst=ga,
            apci=C.APCI_GROUP_RESPONSE,
            payload=payload,
        )

    def get_indication(self, field: str) -> bytes | None:
        """Build a cEMI GroupWrite indication for scenario updates.

        Called by scenarios to send state changes to the bus.
        Finds the appropriate status GA and encodes the current value.
        """
        for _slot_name, ga_int in self.group_addresses.items():
            info = self._ga_info.get(ga_int)
            if not info:
                continue
            s_field, s_dpt, s_direction = info
            if s_field == field and s_direction == "status":
                value = self.state.get(field)
                if value is not None:
                    payload = DPTCodec.encode(s_dpt, value)
                    return frames.encode_cemi(
                        msg_code=0x29,
                        src=self.individual_address,
                        dst=ga_int,
                        apci=C.APCI_GROUP_WRITE,
                        payload=payload,
                    )
        return None
