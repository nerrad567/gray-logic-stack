"""Multi-channel heating actuator (DPT 5.001 / DPT 1.001 per channel).

Handles 2/4/6/8-fold heating actuators for UFH manifolds where each
channel has independent valve position.  Commands on channel C echo on
channel C's status GA — never on channel A's.

Supports both binary (thermal actuators) and percentage (modulating)
input, auto-detected from payload size — same logic as the single-channel
ValveActuator.

GA naming conventions (resolved at __init__):
  channel_a_valve / channel_a_valve_status     (config.yaml style)
  ch1_valve_cmd  / ch1_valve_status            (template style)
  a_valve / a_valve_status                     (short style)
"""

import logging
import re

from .base import BaseDevice, decode_dpt1, decode_dpt5, encode_dpt1, encode_dpt5

logger = logging.getLogger("knxsim.devices")

# Maps numeric channel indices to letter IDs
_NUM_TO_LETTER = {str(i + 1): chr(ord("a") + i) for i in range(24)}


def _extract_channel_id(ga_name: str) -> str | None:
    """Extract channel letter (a-z) from a GA function name.

    Returns lowercase single letter, or None if no channel found.
    """
    n = ga_name.lower()

    # channel_a_valve / channel_a_valve_status
    m = re.match(r"channel_([a-z])_", n)
    if m:
        return m.group(1)

    # ch1_valve_cmd / ch1_valve_status  (template numeric)
    m = re.match(r"ch(\d+)_", n)
    if m:
        return _NUM_TO_LETTER.get(m.group(1))

    # a_valve / a_valve_status  (leading letter)
    m = re.match(r"^([a-z])_(?:valve|status|cmd|position)", n)
    if m:
        return m.group(1)

    # valve_a / valve_status_a  (trailing letter)
    m = re.search(r"_([a-z])$", n)
    if m:
        return m.group(1)

    # valve_1 / valve_status_1  (trailing number)
    m = re.search(r"_(\d+)$", n)
    if m:
        return _NUM_TO_LETTER.get(m.group(1))

    return None


def _is_status_name(ga_name: str) -> bool:
    """Check if a GA function name is a status/feedback GA."""
    n = ga_name.lower()
    return any(p in n for p in ("status", "feedback"))


class HeatingActuator(BaseDevice):
    """Multi-channel heating actuator with independent per-channel valve state."""

    GA_DPT_MAP = {
        "valve": "5.001",
        "valve_cmd": "5.001",
        "valve_status": "5.001",
    }

    def __init__(
        self,
        device_id: str,
        individual_address: int,
        group_addresses: dict,
        initial_state: dict,
    ):
        defaults = {"position": 0}
        defaults.update(initial_state)
        super().__init__(device_id, individual_address, group_addresses, defaults)

        # Per-channel state: {"a": {"cmd_ga": int, "status_ga": int, "position": 0}, ...}
        self._channels: dict[str, dict] = {}

        # Reverse lookup: GA (int) → (channel_id, is_status)
        self._ga_to_channel: dict[int, tuple[str, bool]] = {}

        self._build_channel_map()

    def _build_channel_map(self):
        """Parse group_addresses to build per-channel structure."""
        for ga_name, ga_int in self.group_addresses.items():
            ch_id = _extract_channel_id(ga_name)
            if ch_id is None:
                # Single-channel fallback
                ch_id = "a"

            is_status = _is_status_name(ga_name)

            if ch_id not in self._channels:
                self._channels[ch_id] = {
                    "cmd_ga": None,
                    "status_ga": None,
                    "position": self.state.get(f"ch_{ch_id}_valve", 0),
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

        if is_status:
            # Status GA — absorb, don't echo
            position = self._decode_valve(payload)
            channel["position"] = position
            self.state[f"ch_{ch_id}_valve"] = position
            # Template-compatible key — persistence layer filters by
            # channel template keys ({"position": 0}), GA-routing scopes
            # the update to the correct channel.
            self.state["position"] = position
            logger.info(
                "%s ch %s <- status update %d%%",
                self.device_id,
                ch_id.upper(),
                position,
            )
            return None

        # Command GA — auto-detect binary vs percentage
        position = self._decode_valve(payload)
        channel["position"] = position
        self.state[f"ch_{ch_id}_valve"] = position
        self.state["position"] = position

        # Echo on this channel's status GA
        status_ga = channel["status_ga"]
        if status_ga is not None:
            logger.info(
                "%s ch %s <- %d%% -> status",
                self.device_id,
                ch_id.upper(),
                position,
            )
            return self._make_response(status_ga, encode_dpt5(position))

        logger.info(
            "%s ch %s <- %d%%",
            self.device_id,
            ch_id.upper(),
            position,
        )
        return None

    def on_group_read(self, ga: int) -> bytes | None:
        lookup = self._ga_to_channel.get(ga)
        if not lookup:
            return None

        ch_id, _ = lookup
        channel = self._channels[ch_id]
        return self._make_response(ga, encode_dpt5(channel["position"]))

    @staticmethod
    def _decode_valve(payload: bytes) -> int:
        """Auto-detect binary (DPT 1.001) vs percentage (DPT 5.001).

        Same heuristic as the single-channel ValveActuator:
        - 1-byte payload with value > 1 → percentage
        - Otherwise → binary ON/OFF mapped to 100/0
        """
        if len(payload) == 1 and payload[0] > 1:
            return decode_dpt5(payload)
        # Binary: ON → 100%, OFF → 0%
        return 100 if decode_dpt1(payload) else 0
