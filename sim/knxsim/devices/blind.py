"""Blind/shutter with position and slat/tilt (DPT 5.001 for both).

Group addresses:
  position_cmd    — receives GroupWrite 0–100% (0=open, 100=closed)
  position_status — sends GroupResponse with current position
  slat_cmd        — receives GroupWrite 0–100% tilt angle
  slat_status     — sends GroupResponse with current tilt
"""

import logging
from typing import Optional

from .base import BaseDevice, decode_dpt5, encode_dpt5

logger = logging.getLogger("knxsim.devices")


class Blind(BaseDevice):
    GA_DPT_MAP = {
        "position_cmd": "5.001",
        "position_status": "5.001",
        "slat_cmd": "5.001",
        "slat_status": "5.001",
    }

    def on_group_write(self, ga: int, payload: bytes) -> Optional[bytes]:
        name = self.get_ga_name(ga)

        if name == "position_cmd":
            position = decode_dpt5(payload)
            self.state["position"] = position

            status_ga = self.group_addresses.get("position_status")
            if status_ga is not None:
                logger.info("%s ← position %d%%", self.device_id, position)
                return self._make_response(status_ga, encode_dpt5(position))

        elif name == "slat_cmd":
            tilt = decode_dpt5(payload)
            self.state["tilt"] = tilt

            status_ga = self.group_addresses.get("slat_status")
            if status_ga is not None:
                logger.info("%s ← tilt %d%%", self.device_id, tilt)
                return self._make_response(status_ga, encode_dpt5(tilt))

        return None

    def on_group_read(self, ga: int) -> Optional[bytes]:
        name = self.get_ga_name(ga)
        if name in ("position_cmd", "position_status"):
            return self._make_response(ga, encode_dpt5(self.state.get("position", 0)))
        if name in ("slat_cmd", "slat_status"):
            return self._make_response(ga, encode_dpt5(self.state.get("tilt", 0)))
        return None
