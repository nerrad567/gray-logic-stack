"""Telegram Inspector — ring buffer for decoded telegram history.

Stores the last N telegrams per premise for REST access. Each entry
includes timestamp, direction, decoded GA, APCI type, payload, and
which device handled it.

Thread-safe: uses a lock since telegrams arrive from the KNX UDP
handler thread.
"""

import logging
import threading
import time
from collections import deque
from typing import Any, Optional

from knxip import constants as C
from knxip import frames

logger = logging.getLogger("knxsim.telegram_inspector")

# APCI code to human-readable name
APCI_NAMES = {
    C.APCI_GROUP_WRITE: "GroupWrite",
    C.APCI_GROUP_READ: "GroupRead",
    C.APCI_GROUP_RESPONSE: "GroupResponse",
}

# Direction descriptions for clarity
DIRECTION_INFO = {
    "rx": {"label": "RX", "description": "Received from client (command)"},
    "tx": {"label": "TX", "description": "Sent by simulator (status/sensor)"},
}


class TelegramEntry:
    """A single decoded telegram record."""

    __slots__ = (
        "timestamp",
        "premise_id",
        "direction",
        "source",
        "destination",
        "apci",
        "apci_name",
        "payload_hex",
        "device_id",
        "dpt",
        "decoded_value",
        "unit",
        "ga_function",
    )

    def __init__(
        self,
        premise_id: str,
        direction: str,
        source: int,
        destination: int,
        apci: int,
        payload: bytes,
        device_id: Optional[str] = None,
        dpt: Optional[str] = None,
        decoded_value: Any = None,
        unit: Optional[str] = None,
        ga_function: Optional[str] = None,
    ):
        self.timestamp = time.time()
        self.premise_id = premise_id
        self.direction = direction  # "rx" or "tx"
        self.source = source
        self.destination = destination
        self.apci = apci
        self.apci_name = APCI_NAMES.get(apci, f"0x{apci:04X}")
        self.payload_hex = payload.hex() if payload else ""
        self.device_id = device_id
        self.dpt = dpt
        self.decoded_value = decoded_value
        self.unit = unit
        self.ga_function = ga_function

    def to_dict(self) -> dict:
        dir_info = DIRECTION_INFO.get(self.direction, {})
        return {
            "timestamp": self.timestamp,
            "premise_id": self.premise_id,
            "direction": self.direction,
            "direction_label": dir_info.get("label", self.direction.upper()),
            "direction_desc": dir_info.get("description", ""),
            "source": frames.format_individual_address(self.source),
            "destination": frames.format_group_address(self.destination),
            "apci": self.apci_name,
            "payload": self.payload_hex,
            "device_id": self.device_id,
            "dpt": self.dpt,
            "decoded_value": self.decoded_value,
            "unit": self.unit or "",
            "ga_function": self.ga_function,
        }


class TelegramInspector:
    """Ring buffer storing decoded telegram history per premise."""

    def __init__(self, max_size: int = 1000):
        self._max_size = max_size
        # premise_id -> deque of TelegramEntry
        self._buffers: dict[str, deque[TelegramEntry]] = {}
        self._lock = threading.Lock()
        self._total_count = 0

    def record(
        self,
        premise_id: str,
        cemi_dict: dict,
        direction: str = "rx",
        device_id: Optional[str] = None,
        dpt: Optional[str] = None,
        decoded_value: Any = None,
        unit: Optional[str] = None,
        ga_function: Optional[str] = None,
    ):
        """Record a telegram into the ring buffer.

        Args:
            premise_id: Which premise this telegram belongs to
            cemi_dict: Decoded cEMI dict with dst, src, apci, payload
            direction: "rx" (received from client) or "tx" (sent by simulator)
            device_id: Which device handled this telegram (if known)
            dpt: DPT ID string (e.g., "9.001") if known
            decoded_value: Decoded Python value (e.g., 21.5 for temperature)
            unit: Unit string from DPT info (e.g., "°C")
            ga_function: GA function name (e.g., "temperature", "switch_cmd")
        """
        entry = TelegramEntry(
            premise_id=premise_id,
            direction=direction,
            source=cemi_dict.get("src", 0),
            destination=cemi_dict.get("dst", 0),
            apci=cemi_dict.get("apci", 0),
            payload=cemi_dict.get("payload", b""),
            device_id=device_id,
            dpt=dpt,
            decoded_value=decoded_value,
            unit=unit,
            ga_function=ga_function,
        )

        with self._lock:
            if premise_id not in self._buffers:
                self._buffers[premise_id] = deque(maxlen=self._max_size)
            self._buffers[premise_id].append(entry)
            self._total_count += 1

    def get_history(
        self,
        premise_id: str,
        limit: int = 100,
        offset: int = 0,
    ) -> list[dict]:
        """Get telegram history for a premise (newest first).

        Args:
            premise_id: Which premise to query
            limit: Max entries to return (default 100)
            offset: Skip this many entries from the newest
        """
        with self._lock:
            buf = self._buffers.get(premise_id)
            if not buf:
                return []
            # Convert to list (newest first)
            entries = list(reversed(buf))
            return [e.to_dict() for e in entries[offset : offset + limit]]

    def get_stats(self, premise_id: Optional[str] = None) -> dict:
        """Get telegram statistics."""
        with self._lock:
            if premise_id:
                buf = self._buffers.get(premise_id)
                return {
                    "premise_id": premise_id,
                    "buffered": len(buf) if buf else 0,
                    "buffer_max": self._max_size,
                }
            return {
                "total_recorded": self._total_count,
                "premises": {pid: len(buf) for pid, buf in self._buffers.items()},
                "buffer_max": self._max_size,
            }

    def clear(self, premise_id: Optional[str] = None):
        """Clear telegram history."""
        with self._lock:
            if premise_id:
                self._buffers.pop(premise_id, None)
            else:
                self._buffers.clear()
                self._total_count = 0
