"""Multi-channel switch actuator (DPT 1.001 per channel).

Handles 2/4/6/8/12/16/24-fold switch actuators where each channel has
independent on/off state.  Commands on channel C echo on channel C's
status GA — never on channel A's.

GA naming conventions (all resolved at __init__):
  channel_a_switch / channel_a_switch_status   (config.yaml style)
  a_switch / a_switch_status                   (short style)
  switch_a / switch_status_a                   (suffix style)
  switch_1 / switch_status_1                   (numeric — 1→a, 2→b, ...)
  ch1_switch_cmd / ch1_switch_status           (template style)
"""

import logging
import re

from .base import BaseDevice, decode_dpt1, encode_dpt1

logger = logging.getLogger("knxsim.devices")

# Maps numeric channel indices to letter IDs
_NUM_TO_LETTER = {str(i + 1): chr(ord("a") + i) for i in range(24)}


def _extract_channel_id(ga_name: str) -> str | None:
    """Extract channel letter (a-z) from a GA function name.

    Returns lowercase single letter, or None if no channel found.
    """
    n = ga_name.lower()

    # channel_a_switch / channel_a_switch_status
    m = re.match(r"channel_([a-z])_", n)
    if m:
        return m.group(1)

    # ch1_switch_cmd / ch1_switch_status  (template numeric)
    m = re.match(r"ch(\d+)_", n)
    if m:
        return _NUM_TO_LETTER.get(m.group(1))

    # a_switch / a_switch_status  (leading letter)
    m = re.match(r"^([a-z])_(?:switch|status|cmd)", n)
    if m:
        return m.group(1)

    # switch_a / switch_status_a  (trailing letter)
    m = re.search(r"_([a-z])$", n)
    if m:
        return m.group(1)

    # switch_1 / switch_status_1  (trailing number)
    m = re.search(r"_(\d+)$", n)
    if m:
        return _NUM_TO_LETTER.get(m.group(1))

    return None


def _is_status_name(ga_name: str) -> bool:
    """Check if a GA function name is a status/feedback GA."""
    n = ga_name.lower()
    return any(p in n for p in ("status", "feedback", "state", "actual"))


class SwitchActuator(BaseDevice):
    """Multi-channel switch actuator with independent per-channel state."""

    GA_DPT_MAP = {
        "switch": "1.001",
        "switch_cmd": "1.001",
        "switch_status": "1.001",
    }

    def __init__(
        self,
        device_id: str,
        individual_address: int,
        group_addresses: dict,
        initial_state: dict,
    ):
        super().__init__(device_id, individual_address, group_addresses, initial_state)

        # Per-channel state: {"a": {"cmd_ga": int, "status_ga": int, "on": False}, ...}
        self._channels: dict[str, dict] = {}

        # Reverse lookup: GA (int) → (channel_id, is_status)
        self._ga_to_channel: dict[int, tuple[str, bool]] = {}

        self._build_channel_map()

    def _build_channel_map(self):
        """Parse group_addresses to build per-channel structure."""
        # First pass: discover all channels and their GAs
        for ga_name, ga_int in self.group_addresses.items():
            ch_id = _extract_channel_id(ga_name)
            if ch_id is None:
                # Single-channel fallback — treat as channel "a"
                ch_id = "a"

            is_status = _is_status_name(ga_name)

            if ch_id not in self._channels:
                self._channels[ch_id] = {
                    "cmd_ga": None,
                    "status_ga": None,
                    "on": self.state.get(f"ch_{ch_id}_on", False),
                }

            if is_status:
                self._channels[ch_id]["status_ga"] = ga_int
            else:
                self._channels[ch_id]["cmd_ga"] = ga_int

            self._ga_to_channel[ga_int] = (ch_id, is_status)

        logger.debug(
            "%s channels: %s",
            self.device_id,
            {ch: {"cmd": c["cmd_ga"], "sts": c["status_ga"]} for ch, c in self._channels.items()},
        )

    def on_group_write(self, ga: int, payload: bytes) -> bytes | None:
        lookup = self._ga_to_channel.get(ga)
        if not lookup:
            return None

        ch_id, is_status = lookup
        channel = self._channels[ch_id]
        value = decode_dpt1(payload)

        # Update this channel's state
        channel["on"] = value
        self.state[f"ch_{ch_id}_on"] = value
        # Also set template-compatible "on" key — the persistence layer's
        # update_channel_state_by_ga() filters by keys that exist in the
        # channel state template ({"on": False}), and GA-routing ensures
        # only the correct channel is updated.
        self.state["on"] = value

        if is_status:
            # Status GA — absorb update, don't echo
            logger.info(
                "%s ch %s <- status update %s",
                self.device_id,
                ch_id.upper(),
                "ON" if value else "OFF",
            )
            return None

        # Command GA — echo on this channel's status GA
        status_ga = channel["status_ga"]
        if status_ga is not None:
            logger.info(
                "%s ch %s <- %s -> status",
                self.device_id,
                ch_id.upper(),
                "ON" if value else "OFF",
            )
            return self._make_response(status_ga, encode_dpt1(value))

        logger.info(
            "%s ch %s <- %s",
            self.device_id,
            ch_id.upper(),
            "ON" if value else "OFF",
        )
        return None

    def on_group_read(self, ga: int) -> bytes | None:
        lookup = self._ga_to_channel.get(ga)
        if not lookup:
            return None

        ch_id, _ = lookup
        channel = self._channels[ch_id]
        return self._make_response(ga, encode_dpt1(channel["on"]))
