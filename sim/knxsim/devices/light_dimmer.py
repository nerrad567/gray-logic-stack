"""Dimmable light (DPT 1.001 for switch, DPT 5.001 for brightness).

Group addresses:
  switch_cmd       — receives GroupWrite ON/OFF
  switch_status    — sends GroupResponse with on/off state
  brightness_cmd   — receives GroupWrite 0–100%
  brightness_status — sends GroupResponse with brightness level

Note: Real KNX dimmers send both switch and brightness status when state
changes. This implementation returns a list of responses to simulate that
behaviour.
"""

import logging

from .base import BaseDevice, decode_dpt1, decode_dpt5, encode_dpt1, encode_dpt5

logger = logging.getLogger("knxsim.devices")


class LightDimmer(BaseDevice):
    GA_DPT_MAP = {
        "switch_cmd": "1.001",
        "switch_status": "1.001",
        "brightness_cmd": "5.001",
        "brightness_status": "5.001",
    }

    def on_group_write(
        self, ga: int, payload: bytes
    ) -> bytes | list[bytes] | None:
        name = self.get_ga_name(ga)

        if name == "switch_cmd":
            value = decode_dpt1(payload)
            self.state["on"] = value
            # When turning on with brightness 0, default to 100%
            if value and self.state.get("brightness", 0) == 0:
                self.state["brightness"] = 100

            logger.info("%s ← switch %s", self.device_id, "ON" if value else "OFF")

            # Send both switch and brightness status (like real KNX dimmers)
            responses = []
            switch_ga = self.group_addresses.get("switch_status")
            if switch_ga is not None:
                responses.append(self._make_response(switch_ga, encode_dpt1(value)))
            brightness_ga = self.group_addresses.get("brightness_status")
            if brightness_ga is not None:
                responses.append(
                    self._make_response(
                        brightness_ga, encode_dpt5(self.state.get("brightness", 0))
                    )
                )

            return responses if responses else None

        elif name == "brightness_cmd":
            brightness = decode_dpt5(payload)
            self.state["brightness"] = brightness
            # Setting brightness > 0 implies ON
            self.state["on"] = brightness > 0

            logger.info("%s ← brightness %d%%", self.device_id, brightness)

            # Send both status values
            responses = []
            switch_ga = self.group_addresses.get("switch_status")
            if switch_ga is not None:
                responses.append(
                    self._make_response(switch_ga, encode_dpt1(self.state["on"]))
                )
            brightness_ga = self.group_addresses.get("brightness_status")
            if brightness_ga is not None:
                responses.append(
                    self._make_response(brightness_ga, encode_dpt5(brightness))
                )

            return responses if responses else None

        return None

    def on_group_read(self, ga: int) -> bytes | None:
        name = self.get_ga_name(ga)
        if name in ("switch_cmd", "switch_status"):
            return self._make_response(ga, encode_dpt1(self.state.get("on", False)))
        if name in ("brightness_cmd", "brightness_status"):
            return self._make_response(ga, encode_dpt5(self.state.get("brightness", 0)))
        return None
