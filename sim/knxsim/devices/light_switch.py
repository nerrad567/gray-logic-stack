"""Simple on/off light switch (DPT 1.001).

Group addresses:
  switch_cmd    — receives GroupWrite ON/OFF
  switch_status — sends GroupResponse with current state
"""

import logging
from typing import Optional

from .base import BaseDevice, decode_dpt1, encode_dpt1

logger = logging.getLogger("knxsim.devices")


class LightSwitch(BaseDevice):
    def on_group_write(self, ga: int, payload: bytes) -> Optional[bytes]:
        name = self.get_ga_name(ga)
        if name != "switch_cmd":
            return None

        value = decode_dpt1(payload)
        self.state["on"] = value

        status_ga = self.group_addresses.get("switch_status")
        if status_ga is not None:
            logger.info(
                "%s ← %s → status %s",
                self.device_id,
                "ON" if value else "OFF",
                "ON" if value else "OFF",
            )
            return self._make_response(status_ga, encode_dpt1(value))
        return None

    def on_group_read(self, ga: int) -> Optional[bytes]:
        name = self.get_ga_name(ga)
        if name == "switch_status":
            return self._make_response(ga, encode_dpt1(self.state.get("on", False)))
        if name == "switch_cmd":
            return self._make_response(ga, encode_dpt1(self.state.get("on", False)))
        return None
