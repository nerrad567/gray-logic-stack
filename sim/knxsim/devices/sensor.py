"""Sensor device for temperature, humidity, lux (DPT 9.x — 2-byte float).

Group addresses:
  temperature — reports current temperature value
  humidity    — reports current humidity value (optional)
  lux         — reports current lux value (optional)

Sensors are read-only from the bus perspective — they only respond to
GroupRead and send periodic updates via scenarios.
"""

import logging

from .base import BaseDevice, decode_dpt9, encode_dpt9

logger = logging.getLogger("knxsim.devices")


class Sensor(BaseDevice):
    GA_DPT_MAP = {
        "temperature": "9.001",
        "humidity": "9.007",
        "lux": "9.004",
    }

    def on_group_write(self, ga: int, payload: bytes) -> bytes | None:
        # Sensors can accept write to update their simulated value
        name = self.get_ga_name(ga)
        if name and len(payload) >= 2:
            value = decode_dpt9(payload)
            self.state[name] = value
            logger.info("%s ← %s = %.1f", self.device_id, name, value)
            # Respond with the updated value
            return self._make_response(ga, encode_dpt9(value))
        return None

    def on_group_read(self, ga: int) -> bytes | None:
        name = self.get_ga_name(ga)
        if name and name in self.state:
            value = self.state[name]
            logger.info("%s → read %s = %.1f", self.device_id, name, value)
            return self._make_response(ga, encode_dpt9(value))
        return None

    def get_indication(self, field: str) -> bytes | None:
        """Build an unsolicited L_DATA.ind for a sensor field (used by scenarios)."""
        ga = self.group_addresses.get(field)
        if ga is None:
            return None
        value = self.state.get(field, 0.0)
        return self._make_indication(ga, encode_dpt9(value))
