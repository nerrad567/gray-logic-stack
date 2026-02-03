"""Valve actuator for heating/cooling control.

Supports both binary (thermal actuators) and percentage (modulating actuators):
- Binary mode (DPT 1.001): ON/OFF control, >0% input = ON
- Percentage mode (DPT 5.001): 0-100% valve position

Group addresses:
  valve / valve_cmd    — command input (accepts binary OR percentage)
  valve_status         — current state output (matches input type)
  position             — percentage position (alternative to valve)
  position_status      — percentage position feedback

The actuator auto-detects the DPT from the incoming telegram:
- 1-byte compact (6-bit) → binary
- 1-byte data → percentage
- This allows linking thermostat heating_output (%) directly to actuator valve.
"""

import logging

from .base import BaseDevice, decode_dpt1, decode_dpt5, encode_dpt1, encode_dpt5

logger = logging.getLogger("knxsim.devices")


class ValveActuator(BaseDevice):
    """Valve actuator supporting binary and percentage control."""

    GA_DPT_MAP = {
        "valve": "1.001",  # Default to binary, but accepts percentage too
        "valve_cmd": "1.001",
        "valve_status": "1.001",
        "position": "5.001",
        "position_status": "5.001",
    }

    def __init__(
        self,
        device_id: str,
        individual_address: int,
        group_addresses: dict,
        initial_state: dict,
    ):
        defaults = {
            "on": False,
            "position": 0,
        }
        defaults.update(initial_state)
        super().__init__(device_id, individual_address, group_addresses, defaults)

    def _is_status_ga(self, name: str) -> bool:
        """Check if GA name is for status/feedback."""
        if not name:
            return False
        n = name.lower()
        return "status" in n or "feedback" in n

    def _is_position_ga(self, name: str) -> bool:
        """Check if GA is for percentage position."""
        if not name:
            return False
        n = name.lower()
        return "position" in n

    def _get_status_ga(self, for_position: bool = False) -> int | None:
        """Find the appropriate status GA."""
        # Look for position_status if we have position
        if for_position:
            for name, ga in self.group_addresses.items():
                if "position" in name.lower() and "status" in name.lower():
                    return ga
        # Otherwise look for valve_status
        for name, ga in self.group_addresses.items():
            if "valve" in name.lower() and "status" in name.lower():
                return ga
        return self.group_addresses.get("valve_status")

    def on_group_write(self, ga: int, payload: bytes) -> bytes | list[bytes] | None:
        """Handle incoming commands (binary or percentage)."""
        name = self.get_ga_name(ga)
        if not name:
            return None

        # Skip if this is a status GA (just receiving an update)
        if self._is_status_ga(name):
            return None

        responses = []

        # Determine if this is percentage or binary based on GA name and payload
        is_percentage_ga = self._is_position_ga(name)

        if is_percentage_ga or (len(payload) == 1 and payload[0] > 1):
            # Percentage input (DPT 5.001)
            # Either explicitly a position GA, or value > 1 indicates percentage
            position = decode_dpt5(payload)
            self.state["position"] = position
            # Only derive 'on' if the device was initialised with it (binary valves).
            # Proportional-only devices (heating actuator channels) use position only.
            if "on" in self.state:
                self.state["on"] = position > 0

            logger.info(
                "%s ← position = %d%%",
                self.device_id,
                position,
            )

            # Send position status if available
            status_ga = self._get_status_ga(for_position=True)
            if status_ga:
                responses.append(self._make_response(status_ga, encode_dpt5(position)))

            # Also send binary status if available (only for devices with binary state)
            if "on" in self.state:
                binary_status_ga = self._get_status_ga(for_position=False)
                if binary_status_ga and binary_status_ga != status_ga:
                    responses.append(
                        self._make_response(binary_status_ga, encode_dpt1(self.state["on"]))
                    )

        else:
            # Binary input (DPT 1.001)
            value = decode_dpt1(payload)
            self.state["on"] = value
            self.state["position"] = 100 if value else 0

            logger.info("%s ← valve = %s", self.device_id, "OPEN" if value else "CLOSED")

            # Send binary status
            status_ga = self._get_status_ga(for_position=False)
            if status_ga:
                responses.append(self._make_response(status_ga, encode_dpt1(value)))

        return responses if responses else None

    def on_group_read(self, ga: int) -> bytes | None:
        """Handle GroupRead requests."""
        name = self.get_ga_name(ga)
        if not name:
            return None

        if self._is_position_ga(name):
            position = self.state.get("position", 0)
            logger.info("%s → read position = %d%%", self.device_id, position)
            return self._make_response(ga, encode_dpt5(position))
        else:
            on = self.state.get("on", False)
            logger.info("%s → read valve = %s", self.device_id, "OPEN" if on else "CLOSED")
            return self._make_response(ga, encode_dpt1(on))
