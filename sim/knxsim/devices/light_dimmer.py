"""Dimmable light (DPT 1.001 for switch, DPT 5.001 for brightness).

Group addresses:
  switch_cmd       — receives GroupWrite ON/OFF
  switch_status    — sends GroupResponse with on/off state
  brightness_cmd   — receives GroupWrite 0–100%
  brightness_status — sends GroupResponse with brightness level
"""

import logging
from typing import Optional

from .base import BaseDevice, decode_dpt1, decode_dpt5, encode_dpt1, encode_dpt5

logger = logging.getLogger("knxsim.devices")


class LightDimmer(BaseDevice):
    def on_group_write(self, ga: int, payload: bytes) -> Optional[bytes]:
        name = self.get_ga_name(ga)

        if name == "switch_cmd":
            value = decode_dpt1(payload)
            self.state["on"] = value
            # When turning on with brightness 0, default to 100%
            if value and self.state.get("brightness", 0) == 0:
                self.state["brightness"] = 100

            status_ga = self.group_addresses.get("switch_status")
            if status_ga is not None:
                logger.info("%s ← switch %s", self.device_id, "ON" if value else "OFF")
                return self._make_response(status_ga, encode_dpt1(value))

        elif name == "brightness_cmd":
            brightness = decode_dpt5(payload)
            self.state["brightness"] = brightness
            # Setting brightness > 0 implies ON
            self.state["on"] = brightness > 0

            status_ga = self.group_addresses.get("brightness_status")
            if status_ga is not None:
                logger.info("%s ← brightness %d%%", self.device_id, brightness)
                return self._make_response(status_ga, encode_dpt5(brightness))

        return None

    def on_group_read(self, ga: int) -> Optional[bytes]:
        name = self.get_ga_name(ga)
        if name in ("switch_cmd", "switch_status"):
            return self._make_response(ga, encode_dpt1(self.state.get("on", False)))
        if name in ("brightness_cmd", "brightness_status"):
            return self._make_response(ga, encode_dpt5(self.state.get("brightness", 0)))
        return None
