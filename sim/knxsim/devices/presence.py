"""Presence detector device (DPT 1.x — Boolean + optional DPT 9.x lux).

Group addresses:
  presence — reports occupancy (True = occupied, False = empty)
  lux      — reports ambient light level (optional, DPT 9.x)

Presence detectors are primarily transmit-only (they push state to the bus),
but respond to GroupRead requests for current status.
"""

import logging
from typing import Optional

from .base import BaseDevice, decode_dpt1, decode_dpt9, encode_dpt1, encode_dpt9

logger = logging.getLogger("knxsim.devices")


class PresenceSensor(BaseDevice):
    GA_DPT_MAP = {
        "presence": "1.018",
        "lux": "9.004",
    }

    def on_group_write(self, ga: int, payload: bytes) -> Optional[bytes]:
        name = self.get_ga_name(ga)
        if not name:
            return None

        if name == "presence":
            value = decode_dpt1(payload)
            self.state["presence"] = value
            logger.info("%s ← presence = %s", self.device_id, value)
            return self._make_response(ga, encode_dpt1(value))
        elif name == "lux" and len(payload) >= 2:
            value = decode_dpt9(payload)
            self.state["lux"] = value
            logger.info("%s ← lux = %.1f", self.device_id, value)
            return self._make_response(ga, encode_dpt9(value))

        return None

    def on_group_read(self, ga: int) -> Optional[bytes]:
        name = self.get_ga_name(ga)
        if not name:
            return None

        if name == "presence":
            value = self.state.get("presence", False)
            logger.info("%s → read presence = %s", self.device_id, value)
            return self._make_response(ga, encode_dpt1(value))
        elif name == "lux":
            value = self.state.get("lux", 0.0)
            logger.info("%s → read lux = %.1f", self.device_id, value)
            return self._make_response(ga, encode_dpt9(value))

        return None

    def get_indication_bool(self, field: str) -> Optional[bytes]:
        """Build an unsolicited L_DATA.ind for a boolean field (presence)."""
        ga = self.group_addresses.get(field)
        if ga is None:
            return None
        value = self.state.get(field, False)
        return self._make_indication(ga, encode_dpt1(value))

    def get_indication_float(self, field: str) -> Optional[bytes]:
        """Build an unsolicited L_DATA.ind for a float field (lux)."""
        ga = self.group_addresses.get(field)
        if ga is None:
            return None
        value = self.state.get(field, 0.0)
        return self._make_indication(ga, encode_dpt9(value))
