"""Thermostat device with internal PID controller.

Simulates a smart KNX thermostat (like MDT, Theben, ABB) that:
- Measures temperature (simulated)
- Compares to setpoint
- Calculates heating demand via PID
- Outputs heating_output (0-100%) to control valve actuators

Group addresses:
  current_temperature — measured room temperature (DPT 9.001)
  setpoint           — target temperature, writable (DPT 9.001)
  setpoint_status    — current active setpoint (DPT 9.001)
  heating_output     — PID output 0-100% valve demand (DPT 5.001)
  mode               — HVAC mode: comfort/standby/economy/protection (DPT 20.102)
  mode_status        — current active mode (DPT 20.102)

The thermostat runs its PID internally. Gray Logic Core observes all values
and can override the setpoint, but the PID runs here (simulating on-device
control), ensuring heating works even if GLCore is offline.
"""

import logging
import time

from .base import BaseDevice, decode_dpt1, decode_dpt5, decode_dpt9, encode_dpt5, encode_dpt9

logger = logging.getLogger("knxsim.devices")


# HVAC modes (DPT 20.102)
HVAC_MODES = {
    0: "auto",
    1: "comfort",
    2: "standby",
    3: "economy",
    4: "protection",
}
HVAC_MODE_NAMES = {v: k for k, v in HVAC_MODES.items()}


class Thermostat(BaseDevice):
    """Smart thermostat with internal PID controller."""

    GA_DPT_MAP = {
        "current_temperature": "9.001",
        "actual_temperature": "9.001",  # Alias for current_temperature
        "temperature": "9.001",  # Another common name
        "setpoint": "9.001",
        "setpoint_status": "9.001",
        "heating_output": "5.001",
        "mode": "20.102",
        "mode_status": "20.102",
    }

    # PID parameters (simplified proportional-only for simulation)
    # Real thermostats have full PID with integral/derivative terms
    KP = 10.0  # Proportional gain: 10% output per 1°C error
    OUTPUT_MIN = 0
    OUTPUT_MAX = 100

    # Mode setpoint offsets
    MODE_OFFSETS = {
        "comfort": 0,
        "standby": -2,
        "economy": -4,
        "protection": -10,  # Frost protection
        "auto": 0,
    }

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
            "mode": "comfort",
        }
        defaults.update(initial_state)
        super().__init__(device_id, individual_address, group_addresses, defaults)

        # Calculate initial PID output based on initial state
        self.state["heating_output"] = self._calculate_pid()
        self._last_output = self.state["heating_output"]
        self._last_calculation = time.time()

    def _calculate_pid(self) -> int:
        """Calculate PID output based on current temperature and setpoint.

        Returns heating demand as percentage (0-100).
        """
        current_temp = self.state.get("current_temperature", 20.0)
        base_setpoint = self.state.get("setpoint", 21.0)
        mode = self.state.get("mode", "comfort")

        # Apply mode offset
        offset = self.MODE_OFFSETS.get(mode, 0)
        effective_setpoint = base_setpoint + offset

        # Simple proportional control
        error = effective_setpoint - current_temp

        # Only heat if below setpoint (no cooling in this simple model)
        if error <= 0:
            output = 0
        else:
            output = int(self.KP * error)

        # Clamp output
        output = max(self.OUTPUT_MIN, min(self.OUTPUT_MAX, output))

        return output

    def _send_heating_output(self) -> bytes | None:
        """Generate telegram for heating_output if GA is configured."""
        ga = self.group_addresses.get("heating_output")
        if ga is None:
            return None
        output = self.state["heating_output"]
        logger.info("%s → heating_output = %d%%", self.device_id, output)
        return self._make_indication(ga, encode_dpt5(output))

    def _send_setpoint_status(self) -> bytes | None:
        """Generate telegram for setpoint_status if GA is configured."""
        ga = self.group_addresses.get("setpoint_status")
        if ga is None:
            return None
        setpoint = self.state["setpoint"]
        return self._make_indication(ga, encode_dpt9(setpoint))

    def _send_mode_status(self) -> bytes | None:
        """Generate telegram for mode_status if GA is configured."""
        ga = self.group_addresses.get("mode_status")
        if ga is None:
            return None
        mode = self.state.get("mode", "comfort")
        mode_val = HVAC_MODE_NAMES.get(mode, 1)
        return self._make_indication(ga, bytes([mode_val]))

    def on_group_write(self, ga: int, payload: bytes) -> bytes | list[bytes] | None:
        """Handle incoming GroupWrite telegrams."""
        name = self.get_ga_name(ga)
        if not name:
            return None

        responses = []

        if name == "setpoint":
            # Setpoint change from external source (e.g., GLCore override)
            if len(payload) >= 2:
                value = decode_dpt9(payload)
                self.state["setpoint"] = value
                logger.info("%s ← setpoint = %.1f°C", self.device_id, value)

                # Recalculate PID with new setpoint
                self.state["heating_output"] = self._calculate_pid()
                self._last_output = self.state["heating_output"]

                # Send status updates
                status = self._send_setpoint_status()
                if status:
                    responses.append(status)
                output = self._send_heating_output()
                if output:
                    responses.append(output)

        elif name in ("current_temperature", "actual_temperature", "temperature"):
            # Temperature update (from external sensor or simulation)
            if len(payload) >= 2:
                value = decode_dpt9(payload)
                self.state["current_temperature"] = value
                logger.info("%s ← temperature = %.1f°C", self.device_id, value)

                # Recalculate PID with new temperature
                new_output = self._calculate_pid()
                if new_output != self._last_output:
                    self.state["heating_output"] = new_output
                    self._last_output = new_output
                    output = self._send_heating_output()
                    if output:
                        responses.append(output)

        elif name == "mode":
            # Mode change
            if len(payload) >= 1:
                mode_val = payload[0]
                mode_name = HVAC_MODES.get(mode_val, "comfort")
                self.state["mode"] = mode_name
                logger.info("%s ← mode = %s", self.device_id, mode_name)

                # Recalculate PID with new mode
                self.state["heating_output"] = self._calculate_pid()
                self._last_output = self.state["heating_output"]

                # Send status updates
                status = self._send_mode_status()
                if status:
                    responses.append(status)
                output = self._send_heating_output()
                if output:
                    responses.append(output)

        return responses if responses else None

    def on_group_read(self, ga: int) -> bytes | None:
        """Handle GroupRead requests."""
        name = self.get_ga_name(ga)
        if not name:
            return None

        if name in ("current_temperature", "actual_temperature", "temperature", "setpoint", "setpoint_status"):
            if name == "setpoint_status":
                key = "setpoint"
            elif name in ("actual_temperature", "temperature"):
                key = "current_temperature"
            else:
                key = name
            value = self.state.get(key, 20.0)
            logger.info("%s → read %s = %.1f°C", self.device_id, name, value)
            return self._make_response(ga, encode_dpt9(value))

        elif name == "heating_output":
            value = self.state.get("heating_output", 0)
            logger.info("%s → read heating_output = %d%%", self.device_id, value)
            return self._make_response(ga, encode_dpt5(value))

        elif name in ("mode", "mode_status"):
            mode = self.state.get("mode", "comfort")
            mode_val = HVAC_MODE_NAMES.get(mode, 1)
            logger.info("%s → read %s = %s", self.device_id, name, mode)
            return self._make_response(ga, bytes([mode_val]))

        return None

    def get_indication(self, field: str) -> bytes | None:
        """Build an unsolicited indication for a field (used by scenarios)."""
        ga = self.group_addresses.get(field)
        if ga is None:
            return None

        if field in ("current_temperature", "actual_temperature", "temperature", "setpoint", "setpoint_status"):
            if field == "setpoint_status":
                key = "setpoint"
            elif field in ("actual_temperature", "temperature"):
                key = "current_temperature"
            else:
                key = field
            value = self.state.get(key, 20.0)
            return self._make_indication(ga, encode_dpt9(value))

        elif field == "heating_output":
            value = self.state.get("heating_output", 0)
            return self._make_indication(ga, encode_dpt5(value))

        elif field in ("mode", "mode_status"):
            mode = self.state.get("mode", "comfort")
            mode_val = HVAC_MODE_NAMES.get(mode, 1)
            return self._make_indication(ga, bytes([mode_val]))

        return None

    def update_temperature(self, temperature: float) -> bytes | None:
        """Update temperature and recalculate PID. Returns heating_output telegram if changed.

        Used by scenarios to simulate temperature changes.
        """
        self.state["current_temperature"] = temperature

        new_output = self._calculate_pid()
        if new_output != self._last_output:
            self.state["heating_output"] = new_output
            self._last_output = new_output
            return self._send_heating_output()

        return None

    def recalculate_pid(self) -> None:
        """Recalculate PID output based on current state.

        Called after state is restored from persistence to ensure
        heating_output reflects current temperature/setpoint, not
        stale persisted values.
        """
        self.state["heating_output"] = self._calculate_pid()
        self._last_output = self.state["heating_output"]
