"""Simple on/off light switch (DPT 1.001).

Group addresses (flexible naming — matches by pattern):
  *switch*, *cmd*, button_*, toggle — receives GroupWrite ON/OFF
  *status*, *feedback*              — sends GroupResponse with current state
"""

import logging

from .base import BaseDevice, decode_dpt1, encode_dpt1

logger = logging.getLogger("knxsim.devices")


class LightSwitch(BaseDevice):
    GA_DPT_MAP = {
        "switch_cmd": "1.001",
        "switch_status": "1.001",
    }

    def _is_status_ga(self, name: str) -> bool:
        """Check if GA name is for status/feedback (read-only output)."""
        if not name:
            return False
        n = name.lower()
        return any(pattern in n for pattern in [
            "status", "feedback", "state", "actual",
        ])

    def _get_status_ga(self) -> int | None:
        """Find the GA used for status output."""
        for name, ga in self.group_addresses.items():
            if self._is_status_ga(name):
                return ga
        return self.group_addresses.get("switch_status")

    def on_group_write(self, ga: int, payload: bytes) -> bytes | None:
        name = self.get_ga_name(ga)
        if not name:
            return None

        # Accept commands on any non-status GA
        if self._is_status_ga(name):
            # Status GAs receive updates from other devices - still update state
            value = decode_dpt1(payload)
            self.state["on"] = value
            logger.info("%s ← status update %s", self.device_id, "ON" if value else "OFF")
            return None  # Don't echo status back

        # This is a command GA
        value = decode_dpt1(payload)
        self.state["on"] = value

        status_ga = self._get_status_ga()
        if status_ga is not None:
            logger.info(
                "%s ← %s (via %s) → status",
                self.device_id,
                "ON" if value else "OFF",
                name,
            )
            return self._make_response(status_ga, encode_dpt1(value))

        logger.info("%s ← %s (via %s)", self.device_id, "ON" if value else "OFF", name)
        return None

    def on_group_read(self, ga: int) -> bytes | None:
        # Return current state for any GA read
        return self._make_response(ga, encode_dpt1(self.state.get("on", False)))
