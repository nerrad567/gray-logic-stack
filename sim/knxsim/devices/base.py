"""Base class for simulated KNX devices.

Each device has:
  - An individual address (its physical bus address)
  - One or more group addresses (for commands and status)
  - Internal state that updates on GroupWrite and is reported on GroupRead/Response
"""

import logging
import struct
from typing import Optional

from knxip import constants as C
from knxip import frames

logger = logging.getLogger("knxsim.devices")


# ---------------------------------------------------------------------------
# DPT encoding/decoding helpers
# ---------------------------------------------------------------------------


def encode_dpt1(value: bool) -> bytes:
    """DPT 1.001 — Boolean. Fits in 6-bit compact APCI encoding."""
    return bytes([0x01 if value else 0x00])


def decode_dpt1(payload: bytes) -> bool:
    """DPT 1.001 — Boolean from compact encoding."""
    return bool(payload[0] & 0x01)


def encode_dpt5(value: int) -> bytes:
    """DPT 5.001 — Unsigned 8-bit (0–255, typically 0–100%)."""
    value = max(0, min(255, int(value * 255 / 100)))
    return bytes([value])


def decode_dpt5(payload: bytes) -> int:
    """DPT 5.001 — Unsigned 8-bit → percentage 0–100."""
    return int(payload[0] * 100 / 255)


def encode_dpt9(value: float) -> bytes:
    """DPT 9.001 — 2-byte float (KNX specific encoding).

    Format: MEEEEMMM MMMMMMMM
      M = mantissa (11-bit signed, two's complement)
      E = exponent (4-bit unsigned)
      Value = 0.01 * M * 2^E
    """
    # Find the best exponent to represent the value
    raw = int(value * 100)
    exponent = 0
    while abs(raw) > 2047 and exponent < 15:
        raw = raw >> 1
        exponent += 1

    # Clamp mantissa to 11-bit signed range
    raw = max(-2048, min(2047, raw))

    # Encode as 2 bytes: MEEEEMMM MMMMMMMM
    if raw < 0:
        mantissa = raw + 0x800  # Two's complement for 11 bits
        sign = 1
    else:
        mantissa = raw
        sign = 0

    high = (sign << 7) | (exponent << 3) | ((mantissa >> 8) & 0x07)
    low = mantissa & 0xFF
    return bytes([high, low])


def decode_dpt9(payload: bytes) -> float:
    """DPT 9.001 — 2-byte float → Python float."""
    if len(payload) < 2:
        return 0.0
    high, low = payload[0], payload[1]
    sign = (high >> 7) & 0x01
    exponent = (high >> 3) & 0x0F
    mantissa = ((high & 0x07) << 8) | low
    if sign:
        mantissa = mantissa - 0x800
    return 0.01 * mantissa * (2**exponent)


# ---------------------------------------------------------------------------
# Base device
# ---------------------------------------------------------------------------


class BaseDevice:
    """Base class for all simulated KNX devices."""

    # Default DPT mappings by GA function name. Subclasses can override.
    # Maps ga_name → DPT ID string (e.g., "switch_cmd" → "1.001")
    GA_DPT_MAP: dict[str, str] = {}

    def __init__(
        self,
        device_id: str,
        individual_address: int,
        group_addresses: dict,
        initial_state: dict,
    ):
        self.device_id = device_id
        self.individual_address = individual_address
        self.group_addresses = group_addresses  # name → GA (int)
        self.state = dict(initial_state)

        # Build reverse lookup: GA (int) → ga_name
        self._ga_to_name: dict[int, str] = {}
        for name, ga in group_addresses.items():
            self._ga_to_name[ga] = name

    def handles_ga(self, ga: int) -> bool:
        """Check if this device handles the given group address."""
        return ga in self._ga_to_name

    def get_ga_name(self, ga: int) -> Optional[str]:
        """Get the semantic name of a GA (e.g., 'switch_cmd')."""
        return self._ga_to_name.get(ga)

    def get_dpt_for_ga(self, ga: int) -> Optional[str]:
        """Get the DPT ID for a group address (e.g., '9.001' for temperature).

        Returns None if the GA is unknown or has no DPT mapping.
        """
        ga_name = self._ga_to_name.get(ga)
        if ga_name:
            return self.GA_DPT_MAP.get(ga_name)
        return None

    def on_group_write(self, ga: int, payload: bytes) -> Optional[bytes]:
        """Handle a GroupWrite. Returns response cEMI if state changed, else None.

        Subclasses override this to decode DPT and update state.
        """
        raise NotImplementedError

    def on_group_read(self, ga: int) -> Optional[bytes]:
        """Handle a GroupRead. Returns response cEMI with current state.

        Subclasses override this to encode current state as DPT.
        """
        raise NotImplementedError

    def _make_response(self, ga: int, payload: bytes) -> bytes:
        """Build a cEMI L_DATA.ind GroupResponse frame."""
        return frames.encode_cemi(
            msg_code=C.L_DATA_IND,
            src=self.individual_address,
            dst=ga,
            apci=C.APCI_GROUP_RESPONSE,
            payload=payload,
            is_group=True,
        )

    def _make_indication(self, ga: int, payload: bytes) -> bytes:
        """Build a cEMI L_DATA.ind GroupWrite frame (unsolicited status update)."""
        return frames.encode_cemi(
            msg_code=C.L_DATA_IND,
            src=self.individual_address,
            dst=ga,
            apci=C.APCI_GROUP_WRITE,
            payload=payload,
            is_group=True,
        )
