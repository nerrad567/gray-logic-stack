"""Thermostat device with internal proportional controller.

Simulates a smart KNX thermostat (like MDT, Theben, ABB) that:
- Measures temperature (simulated)
- Compares to setpoint (like a physical dial in the room)
- Calculates heating demand via simple proportional control
- Outputs heating_output (0-100%) to control valve actuators

Group addresses:
  current_temperature — measured room temperature (DPT 9.001)
  setpoint           — target temperature, writable (DPT 9.001)
  setpoint_status    — current setpoint feedback (DPT 9.001)
  heating_output     — controller output 0-100% valve demand (DPT 5.001)

The thermostat runs its control loop internally. Gray Logic Core can
override the setpoint for scheduling/presence, but the control runs
on-device, ensuring heating works even if GLCore is offline.

This matches the offline-first architecture principle: KNX handles
the basics, GLCore is an overlay for intelligence.
"""

import logging
import time

from .base import BaseDevice, decode_dpt9, encode_dpt5, encode_dpt9

logger = logging.getLogger("knxsim.devices")


class Thermostat(BaseDevice):
    """Smart thermostat with internal proportional controller."""

    GA_DPT_MAP = {
        "current_temperature": "9.001",
        "actual_temperature": "9.001",  # Alias for current_temperature
        "temperature": "9.001",  # Another common name
        "setpoint": "9.001",
        "setpoint_status": "9.001",
        "heating_output": "5.001",
    }

    # Proportional control parameter
    # Real thermostats use full PID, but P-only is fine for simulation
    # Kp=10 means: 10% valve opening per 1°C error
    # So 10°C below setpoint = 100% open
    KP = 10.0
    OUTPUT_MIN = 0
    OUTPUT_MAX = 100

    def __init__(
        self,
        device_id: str,
        individual_address: int,
        group_addresses: dict,
        initial_state: dict,
    ):
        # Set defaults for thermostat state
        defaults = {
            "current_temperature": 20.0,
            "setpoint": 21.0,
            "heating_output": 0,
        }
        defaults.update(initial_state)
        super().__init__(device_id, individual_address, group_addresses, defaults)

        # Calculate initial output based on initial state
        self.state["heating_output"] = self._calculate_output()
        self._last_output = self.state["heating_output"]
        self._last_calculation = time.time()

    def _calculate_output(self) -> int:
        """Calculate heating output based on current temperature and setpoint.

        Simple proportional control: output = Kp × error
        Returns heating demand as percentage (0-100).
        """
        current_temp = self.state.get("current_temperature", 20.0)
        setpoint = self.state.get("setpoint", 21.0)

        # Error = how far below setpoint
        error = setpoint - current_temp

        # Only heat if below setpoint (no cooling in this simple model)
        if error <= 0:
            output = 0
        else:
            output = int(self.KP * error)

        # Clamp to 0-100%
        return max(self.OUTPUT_MIN, min(self.OUTPUT_MAX, output))

    def _send_heating_output(self) -> bytes | None:
        """Generate telegram for heating_output if GA is configured."""
        ga = self.group_addresses.get("heating_output")
        if ga is None:
            return None
        output = self.state["heating_output"]
        logger.info("%s → heating_output = %d%%", self.device_id, output)
        return self._make_indication(ga, encode_dpt5(output))

    def _send_setpoint_status(self) -> bytes | None:
        """Generate telegram for setpoint_status if GA is configured.

        Falls back to the setpoint GA when no dedicated status GA exists.
        Real KNX thermostats echo on the same GA in this case.
        """
        ga = self.group_addresses.get("setpoint_status")
        if ga is None:
            ga = self.group_addresses.get("setpoint")
        if ga is None:
            return None
        setpoint = self.state["setpoint"]
        return self._make_indication(ga, encode_dpt9(setpoint))

    def on_group_write(self, ga: int, payload: bytes) -> bytes | list[bytes] | None:
        """Handle incoming GroupWrite telegrams."""
        name = self.get_ga_name(ga)
        if not name:
            return None

        responses = []

        if name == "setpoint":
            # Setpoint change (user turned dial, or GLCore override)
            if len(payload) >= 2:
                value = decode_dpt9(payload)
                self.state["setpoint"] = value
                logger.info("%s ← setpoint = %.1f°C", self.device_id, value)

                # Recalculate with new setpoint
                self.state["heating_output"] = self._calculate_output()
                self._last_output = self.state["heating_output"]

                # Send status updates
                status = self._send_setpoint_status()
                if status:
                    responses.append(status)
                output = self._send_heating_output()
                if output:
                    responses.append(output)

        elif name in ("current_temperature", "actual_temperature", "temperature"):
            # Temperature update (sensor reading)
            if len(payload) >= 2:
                value = decode_dpt9(payload)
                self.state["current_temperature"] = value
                logger.info("%s ← temperature = %.1f°C", self.device_id, value)

                # Recalculate with new temperature
                new_output = self._calculate_output()
                if new_output != self._last_output:
                    self.state["heating_output"] = new_output
                    self._last_output = new_output
                    output = self._send_heating_output()
                    if output:
                        responses.append(output)

        return responses if responses else None

    def on_group_read(self, ga: int) -> bytes | None:
        """Handle GroupRead requests."""
        name = self.get_ga_name(ga)
        if not name:
            return None

        if name in ("current_temperature", "actual_temperature", "temperature"):
            value = self.state.get("current_temperature", 20.0)
            logger.info("%s → read temperature = %.1f°C", self.device_id, value)
            return self._make_response(ga, encode_dpt9(value))

        elif name in ("setpoint", "setpoint_status"):
            value = self.state.get("setpoint", 21.0)
            logger.info("%s → read setpoint = %.1f°C", self.device_id, value)
            return self._make_response(ga, encode_dpt9(value))

        elif name == "heating_output":
            value = self.state.get("heating_output", 0)
            logger.info("%s → read heating_output = %d%%", self.device_id, value)
            return self._make_response(ga, encode_dpt5(value))

        return None

    # Temperature field aliases — config may use any of these interchangeably.
    _TEMP_ALIASES = ("current_temperature", "actual_temperature", "temperature")

    def get_indication(self, field: str) -> bytes | None:
        """Build an unsolicited indication for a field (used by scenarios)."""
        ga = self.group_addresses.get(field)
        # Try alias keys when the primary lookup misses (e.g. scenario asks
        # for "current_temperature" but config uses "actual_temperature").
        if ga is None and field in self._TEMP_ALIASES:
            for alias in self._TEMP_ALIASES:
                ga = self.group_addresses.get(alias)
                if ga is not None:
                    break
        if ga is None:
            return None

        if field in self._TEMP_ALIASES:
            value = self.state.get("current_temperature", 20.0)
            return self._make_indication(ga, encode_dpt9(value))

        elif field in ("setpoint", "setpoint_status"):
            value = self.state.get("setpoint", 21.0)
            return self._make_indication(ga, encode_dpt9(value))

        elif field == "heating_output":
            value = self.state.get("heating_output", 0)
            return self._make_indication(ga, encode_dpt5(value))

        return None

    def update_temperature(self, temperature: float) -> bytes | None:
        """Update temperature and recalculate. Returns heating_output telegram if changed.

        Used by scenarios to simulate temperature changes.
        """
        self.state["current_temperature"] = temperature

        new_output = self._calculate_output()
        if new_output != self._last_output:
            self.state["heating_output"] = new_output
            self._last_output = new_output
            return self._send_heating_output()

        return None

    def recalculate_output(self) -> None:
        """Recalculate heating output based on current state.

        Called after state is restored from persistence to ensure
        heating_output reflects current temperature/setpoint, not
        stale persisted values.
        """
        self.state["heating_output"] = self._calculate_output()
        self._last_output = self.state["heating_output"]
