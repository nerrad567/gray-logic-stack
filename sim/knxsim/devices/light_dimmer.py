"""Dimmable light (DPT 1.001 for switch, DPT 5.001 for brightness).

Group addresses (flexible naming — matches by pattern or DPT):
  *switch*, *cmd*, button_* — receives GroupWrite ON/OFF (DPT 1.xxx)
  *status*, *feedback*      — sends GroupResponse with on/off state
  *brightness*, *dim*, *level* — receives GroupWrite 0–100% (DPT 5.xxx)
  brightness_status         — sends GroupResponse with brightness level

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

    def _is_switch_ga(self, name: str) -> bool:
        """Check if GA name is for switch/on-off control (DPT 1.xxx)."""
        if not name:
            return False
        n = name.lower()
        # Match common patterns for switch/button GAs
        return any(pattern in n for pattern in [
            "switch", "button", "toggle", "onoff", "on_off",
        ]) or n.endswith("_cmd") or n.startswith("cmd")

    def _is_brightness_ga(self, name: str) -> bool:
        """Check if GA name is for brightness/dimming control (DPT 5.xxx)."""
        if not name:
            return False
        n = name.lower()
        return any(pattern in n for pattern in [
            "brightness", "dim", "level", "value", "percent",
        ])

    def _is_status_ga(self, name: str) -> bool:
        """Check if GA name is for status/feedback (read-only output)."""
        if not name:
            return False
        n = name.lower()
        return any(pattern in n for pattern in [
            "status", "feedback", "state", "actual",
        ])

    def _get_switch_status_ga(self) -> int | None:
        """Find the GA used for switch status output."""
        for name, ga in self.group_addresses.items():
            n = name.lower()
            if ("switch" in n or "on" in n) and self._is_status_ga(name):
                return ga
        # Fallback to any status GA
        for name, ga in self.group_addresses.items():
            if self._is_status_ga(name) and not self._is_brightness_ga(name):
                return ga
        return self.group_addresses.get("switch_status")

    def _get_brightness_status_ga(self) -> int | None:
        """Find the GA used for brightness status output."""
        for name, ga in self.group_addresses.items():
            if self._is_brightness_ga(name) and self._is_status_ga(name):
                return ga
        return self.group_addresses.get("brightness_status")

    def on_group_write(
        self, ga: int, payload: bytes
    ) -> bytes | list[bytes] | None:
        name = self.get_ga_name(ga)

        # Determine what kind of command this is based on name pattern
        # Skip if it's clearly a status/feedback GA (shouldn't receive commands)
        if name and self._is_status_ga(name) and not self._is_switch_ga(name) and not self._is_brightness_ga(name):
            # This is a status output GA, not a command input — but still update state
            # (This handles the case where multiple devices share a status GA)
            pass

        # Check for brightness command (DPT 5.xxx — multi-byte payload)
        if name and self._is_brightness_ga(name) and not self._is_status_ga(name):
            brightness = decode_dpt5(payload)
            self.state["brightness"] = brightness
            # Setting brightness > 0 implies ON
            self.state["on"] = brightness > 0

            logger.info("%s ← brightness %d%%", self.device_id, brightness)

            # Send both status values
            responses = []
            switch_ga = self._get_switch_status_ga()
            if switch_ga is not None:
                responses.append(
                    self._make_response(switch_ga, encode_dpt1(self.state["on"]))
                )
            brightness_ga = self._get_brightness_status_ga()
            if brightness_ga is not None:
                responses.append(
                    self._make_response(brightness_ga, encode_dpt5(brightness))
                )

            return responses if responses else None

        # Check for switch command (DPT 1.xxx — single byte/bit payload)
        # This is the default for any GA that looks like a command
        if name and (self._is_switch_ga(name) or not self._is_status_ga(name)):
            # Treat as switch command (DPT 1.001)
            value = decode_dpt1(payload)
            self.state["on"] = value
            # When turning on with brightness 0, default to 100%
            if value and self.state.get("brightness", 0) == 0:
                self.state["brightness"] = 100

            logger.info("%s ← switch %s (via %s)", self.device_id, "ON" if value else "OFF", name)

            # Send both switch and brightness status (like real KNX dimmers)
            responses = []
            switch_ga = self._get_switch_status_ga()
            if switch_ga is not None:
                responses.append(self._make_response(switch_ga, encode_dpt1(value)))
            brightness_ga = self._get_brightness_status_ga()
            if brightness_ga is not None:
                responses.append(
                    self._make_response(
                        brightness_ga, encode_dpt5(self.state.get("brightness", 0))
                    )
                )

            return responses if responses else None

        return None

    def on_group_read(self, ga: int) -> bytes | None:
        name = self.get_ga_name(ga)
        if not name:
            return None

        # Return brightness for brightness-related GAs
        if self._is_brightness_ga(name):
            return self._make_response(ga, encode_dpt5(self.state.get("brightness", 0)))

        # Return on/off state for everything else
        return self._make_response(ga, encode_dpt1(self.state.get("on", False)))
