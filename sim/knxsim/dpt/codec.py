"""KNX Datapoint Type (DPT) Codec — encode and decode for 35+ types.

Each DPT defines how raw bytes on the KNX bus represent real-world values.
This module provides a registry of encode/decode functions, plus metadata
(unit, range, description) for UI display.

Usage:
    from dpt import encode, decode, get_dpt_info

    raw = encode("9.001", 21.5)        # → b'\\x0c\\x33'
    value = decode("9.001", b'\\x0c\\x33')  # → 21.5
    info = get_dpt_info("9.001")       # → {"name": "Temperature", "unit": "°C", ...}
"""

import struct
from typing import Any, Callable, Optional


class DPTInfo:
    """Metadata for a DPT."""

    __slots__ = ("id", "name", "unit", "min_val", "max_val", "encoding_size")

    def __init__(
        self,
        id: str,
        name: str,
        unit: str = "",
        min_val: Any = None,
        max_val: Any = None,
        encoding_size: int = 1,
    ):
        self.id = id
        self.name = name
        self.unit = unit
        self.min_val = min_val
        self.max_val = max_val
        self.encoding_size = encoding_size

    def to_dict(self) -> dict:
        return {
            "id": self.id,
            "name": self.name,
            "unit": self.unit,
            "min": self.min_val,
            "max": self.max_val,
            "encoding_size": self.encoding_size,
        }


# ---------------------------------------------------------------------------
# Encode/Decode functions
# ---------------------------------------------------------------------------


def _encode_dpt1(value) -> bytes:
    """DPT 1.x — Boolean (1 bit, packed in 6-bit APCI)."""
    return bytes([0x01 if value else 0x00])


def _decode_dpt1(data: bytes):
    """DPT 1.x — Boolean."""
    return bool(data[0] & 0x01) if data else False


def _encode_dpt2(value: dict) -> bytes:
    """DPT 2.x — Control + Boolean (2 bits: control | value)."""
    control = 1 if value.get("control", False) else 0
    val = 1 if value.get("value", False) else 0
    return bytes([(control << 1) | val])


def _decode_dpt2(data: bytes) -> dict:
    """DPT 2.x — Control + Boolean."""
    b = data[0] if data else 0
    return {"control": bool(b & 0x02), "value": bool(b & 0x01)}


def _encode_dpt3(value: dict) -> bytes:
    """DPT 3.x — Control + 3-bit dimming/blinds step."""
    control = 1 if value.get("control", False) else 0
    step = int(value.get("step", 0)) & 0x07
    return bytes([(control << 3) | step])


def _decode_dpt3(data: bytes) -> dict:
    """DPT 3.x — Control + 3-bit step."""
    b = data[0] if data else 0
    return {"control": bool(b & 0x08), "step": b & 0x07}


def _encode_dpt4(value: str) -> bytes:
    """DPT 4.x — Character (ASCII or ISO 8859-1)."""
    return bytes([ord(value[0]) & 0xFF]) if value else bytes([0])


def _decode_dpt4(data: bytes) -> str:
    """DPT 4.x — Character."""
    return chr(data[0]) if data else ""


def _encode_dpt5(value) -> bytes:
    """DPT 5.x — Unsigned 8-bit (0–255, often scaled to 0–100%)."""
    v = max(0, min(255, int(value)))
    return bytes([v])


def _decode_dpt5(data: bytes):
    """DPT 5.x — Unsigned 8-bit."""
    return data[0] if data else 0


def _encode_dpt5_001(value) -> bytes:
    """DPT 5.001 — Scaling (0–100% → 0–255)."""
    v = max(0.0, min(100.0, float(value)))
    return bytes([round(v * 255.0 / 100.0)])


def _decode_dpt5_001(data: bytes):
    """DPT 5.001 — Scaling (0–255 → 0–100%)."""
    return round(data[0] * 100.0 / 255.0, 1) if data else 0.0


def _encode_dpt5_003(value) -> bytes:
    """DPT 5.003 — Angle (0–360° → 0–255)."""
    v = max(0.0, min(360.0, float(value)))
    return bytes([round(v * 255.0 / 360.0)])


def _decode_dpt5_003(data: bytes):
    """DPT 5.003 — Angle (0–255 → 0–360°)."""
    return round(data[0] * 360.0 / 255.0, 1) if data else 0.0


def _encode_dpt6(value) -> bytes:
    """DPT 6.x — Signed 8-bit (-128 to 127)."""
    v = max(-128, min(127, int(value)))
    return struct.pack("b", v)


def _decode_dpt6(data: bytes):
    """DPT 6.x — Signed 8-bit."""
    return struct.unpack("b", data[:1])[0] if data else 0


def _encode_dpt7(value) -> bytes:
    """DPT 7.x — Unsigned 16-bit (0–65535)."""
    v = max(0, min(65535, int(value)))
    return struct.pack("!H", v)


def _decode_dpt7(data: bytes):
    """DPT 7.x — Unsigned 16-bit."""
    return struct.unpack("!H", data[:2])[0] if len(data) >= 2 else 0


def _encode_dpt8(value) -> bytes:
    """DPT 8.x — Signed 16-bit (-32768 to 32767)."""
    v = max(-32768, min(32767, int(value)))
    return struct.pack("!h", v)


def _decode_dpt8(data: bytes):
    """DPT 8.x — Signed 16-bit."""
    return struct.unpack("!h", data[:2])[0] if len(data) >= 2 else 0


def _encode_dpt9(value) -> bytes:
    """DPT 9.x — 2-byte float (KNX-specific encoding).

    Format: MEEEEMMM MMMMMMMM
    value = 0.01 * M * 2^E
    where M is signed 11-bit mantissa, E is 4-bit exponent.
    """
    v = float(value)
    # Handle zero
    if v == 0:
        return bytes([0x00, 0x00])

    sign = 1 if v < 0 else 0
    if sign:
        v = -v

    # Find appropriate exponent
    exp = 0
    mantissa = v * 100.0
    while mantissa > 2047 and exp < 15:
        mantissa /= 2.0
        exp += 1

    mantissa = int(round(mantissa))
    if mantissa > 2047:
        mantissa = 2047

    if sign:
        # Two's complement for negative mantissa
        mantissa = (~mantissa + 1) & 0x7FF

    byte1 = (sign << 7) | (exp << 3) | ((mantissa >> 8) & 0x07)
    byte2 = mantissa & 0xFF
    return bytes([byte1, byte2])


def _decode_dpt9(data: bytes):
    """DPT 9.x — 2-byte float."""
    if len(data) < 2:
        return 0.0

    byte1, byte2 = data[0], data[1]
    sign = (byte1 >> 7) & 0x01
    exp = (byte1 >> 3) & 0x0F
    mantissa = ((byte1 & 0x07) << 8) | byte2

    if sign:
        # Two's complement: convert from unsigned 11-bit
        mantissa = mantissa - 2048

    value = 0.01 * mantissa * (2**exp)
    return round(value, 2)


def _encode_dpt10(value: dict) -> bytes:
    """DPT 10.x — Time of day (day, hour, minute, second)."""
    day = int(value.get("day", 0)) & 0x07
    hour = int(value.get("hour", 0)) & 0x1F
    minute = int(value.get("minute", 0)) & 0x3F
    second = int(value.get("second", 0)) & 0x3F
    return bytes([(day << 5) | hour, minute, second])


def _decode_dpt10(data: bytes) -> dict:
    """DPT 10.x — Time of day."""
    if len(data) < 3:
        return {"day": 0, "hour": 0, "minute": 0, "second": 0}
    return {
        "day": (data[0] >> 5) & 0x07,
        "hour": data[0] & 0x1F,
        "minute": data[1] & 0x3F,
        "second": data[2] & 0x3F,
    }


def _encode_dpt11(value: dict) -> bytes:
    """DPT 11.x — Date (day, month, year)."""
    day = int(value.get("day", 1)) & 0x1F
    month = int(value.get("month", 1)) & 0x0F
    year = int(value.get("year", 0)) & 0x7F
    return bytes([day, month, year])


def _decode_dpt11(data: bytes) -> dict:
    """DPT 11.x — Date."""
    if len(data) < 3:
        return {"day": 1, "month": 1, "year": 0}
    year = data[2] & 0x7F
    # KNX convention: 0-99, where 0-89 = 2000-2089, 90-99 = 1990-1999
    full_year = (2000 + year) if year < 90 else (1900 + year)
    return {
        "day": data[0] & 0x1F,
        "month": data[1] & 0x0F,
        "year": full_year,
    }


def _encode_dpt12(value) -> bytes:
    """DPT 12.x — Unsigned 32-bit counter."""
    v = max(0, min(4294967295, int(value)))
    return struct.pack("!I", v)


def _decode_dpt12(data: bytes):
    """DPT 12.x — Unsigned 32-bit."""
    return struct.unpack("!I", data[:4])[0] if len(data) >= 4 else 0


def _encode_dpt13(value) -> bytes:
    """DPT 13.x — Signed 32-bit counter."""
    v = max(-2147483648, min(2147483647, int(value)))
    return struct.pack("!i", v)


def _decode_dpt13(data: bytes):
    """DPT 13.x — Signed 32-bit."""
    return struct.unpack("!i", data[:4])[0] if len(data) >= 4 else 0


def _encode_dpt14(value) -> bytes:
    """DPT 14.x — 4-byte IEEE 754 float."""
    return struct.pack("!f", float(value))


def _decode_dpt14(data: bytes):
    """DPT 14.x — 4-byte float."""
    return round(struct.unpack("!f", data[:4])[0], 4) if len(data) >= 4 else 0.0


def _encode_dpt16(value: str) -> bytes:
    """DPT 16.x — 14-character ASCII string."""
    encoded = value.encode("ascii", errors="replace")[:14]
    return encoded.ljust(14, b"\x00")


def _decode_dpt16(data: bytes) -> str:
    """DPT 16.x — 14-character string."""
    return data[:14].decode("ascii", errors="replace").rstrip("\x00")


def _encode_dpt17(value) -> bytes:
    """DPT 17.x — Scene number (0–63)."""
    return bytes([int(value) & 0x3F])


def _decode_dpt17(data: bytes):
    """DPT 17.x — Scene number."""
    return data[0] & 0x3F if data else 0


def _encode_dpt18(value: dict) -> bytes:
    """DPT 18.x — Scene control (learn + number)."""
    learn = 1 if value.get("learn", False) else 0
    number = int(value.get("number", 0)) & 0x3F
    return bytes([(learn << 7) | number])


def _decode_dpt18(data: bytes) -> dict:
    """DPT 18.x — Scene control."""
    b = data[0] if data else 0
    return {"learn": bool(b & 0x80), "number": b & 0x3F}


def _encode_dpt20(value) -> bytes:
    """DPT 20.x — HVAC mode (1 byte enum)."""
    return bytes([int(value) & 0xFF])


def _decode_dpt20(data: bytes):
    """DPT 20.x — HVAC mode."""
    return data[0] if data else 0


# ---------------------------------------------------------------------------
# DPT Registry
# ---------------------------------------------------------------------------

# (encoder, decoder, DPTInfo)
_REGISTRY: dict[str, tuple[Callable, Callable, DPTInfo]] = {}


def _register(dpt_id: str, encoder: Callable, decoder: Callable, info: DPTInfo):
    _REGISTRY[dpt_id] = (encoder, decoder, info)
    # Also register the main type (e.g., "1" for "1.001")
    main_type = dpt_id.split(".")[0]
    if main_type not in _REGISTRY:
        _REGISTRY[main_type] = (encoder, decoder, info)


# DPT 1.x — Boolean
_register("1", _encode_dpt1, _decode_dpt1, DPTInfo("1", "Boolean", "", False, True, 1))
_register(
    "1.001", _encode_dpt1, _decode_dpt1, DPTInfo("1.001", "Switch", "", False, True, 1)
)
_register(
    "1.002", _encode_dpt1, _decode_dpt1, DPTInfo("1.002", "Boolean", "", False, True, 1)
)
_register(
    "1.003", _encode_dpt1, _decode_dpt1, DPTInfo("1.003", "Enable", "", False, True, 1)
)
_register(
    "1.005", _encode_dpt1, _decode_dpt1, DPTInfo("1.005", "Alarm", "", False, True, 1)
)
_register(
    "1.008", _encode_dpt1, _decode_dpt1, DPTInfo("1.008", "Up/Down", "", False, True, 1)
)
_register(
    "1.009",
    _encode_dpt1,
    _decode_dpt1,
    DPTInfo("1.009", "Open/Close", "", False, True, 1),
)
_register(
    "1.010",
    _encode_dpt1,
    _decode_dpt1,
    DPTInfo("1.010", "Start/Stop", "", False, True, 1),
)
_register(
    "1.018",
    _encode_dpt1,
    _decode_dpt1,
    DPTInfo("1.018", "Occupancy", "", False, True, 1),
)

# DPT 2.x — Control Boolean
_register(
    "2", _encode_dpt2, _decode_dpt2, DPTInfo("2", "Control Boolean", "", None, None, 1)
)
_register(
    "2.001",
    _encode_dpt2,
    _decode_dpt2,
    DPTInfo("2.001", "Switch Control", "", None, None, 1),
)

# DPT 3.x — Dimming/Blinds Control
_register(
    "3", _encode_dpt3, _decode_dpt3, DPTInfo("3", "Control Dimming", "", None, None, 1)
)
_register(
    "3.007",
    _encode_dpt3,
    _decode_dpt3,
    DPTInfo("3.007", "Dimming Control", "", None, None, 1),
)
_register(
    "3.008",
    _encode_dpt3,
    _decode_dpt3,
    DPTInfo("3.008", "Blinds Control", "", None, None, 1),
)

# DPT 4.x — Character
_register("4", _encode_dpt4, _decode_dpt4, DPTInfo("4", "Character", "", None, None, 1))
_register(
    "4.001",
    _encode_dpt4,
    _decode_dpt4,
    DPTInfo("4.001", "ASCII Character", "", None, None, 1),
)

# DPT 5.x — Unsigned 8-bit
_register(
    "5", _encode_dpt5, _decode_dpt5, DPTInfo("5", "Unsigned 8-bit", "", 0, 255, 1)
)
_register(
    "5.001",
    _encode_dpt5_001,
    _decode_dpt5_001,
    DPTInfo("5.001", "Scaling", "%", 0, 100, 1),
)
_register(
    "5.003",
    _encode_dpt5_003,
    _decode_dpt5_003,
    DPTInfo("5.003", "Angle", "°", 0, 360, 1),
)
_register(
    "5.004",
    _encode_dpt5,
    _decode_dpt5,
    DPTInfo("5.004", "Percent (0–255)", "%", 0, 255, 1),
)
_register(
    "5.010",
    _encode_dpt5,
    _decode_dpt5,
    DPTInfo("5.010", "Counter Pulses", "pulses", 0, 255, 1),
)

# DPT 6.x — Signed 8-bit
_register(
    "6", _encode_dpt6, _decode_dpt6, DPTInfo("6", "Signed 8-bit", "", -128, 127, 1)
)
_register(
    "6.001",
    _encode_dpt6,
    _decode_dpt6,
    DPTInfo("6.001", "Percent (-128..127)", "%", -128, 127, 1),
)

# DPT 7.x — Unsigned 16-bit
_register(
    "7", _encode_dpt7, _decode_dpt7, DPTInfo("7", "Unsigned 16-bit", "", 0, 65535, 2)
)
_register(
    "7.001",
    _encode_dpt7,
    _decode_dpt7,
    DPTInfo("7.001", "Pulses", "pulses", 0, 65535, 2),
)
_register(
    "7.005",
    _encode_dpt7,
    _decode_dpt7,
    DPTInfo("7.005", "Time (seconds)", "s", 0, 65535, 2),
)
_register(
    "7.012",
    _encode_dpt7,
    _decode_dpt7,
    DPTInfo("7.012", "Current (mA)", "mA", 0, 65535, 2),
)

# DPT 8.x — Signed 16-bit
_register(
    "8", _encode_dpt8, _decode_dpt8, DPTInfo("8", "Signed 16-bit", "", -32768, 32767, 2)
)
_register(
    "8.001",
    _encode_dpt8,
    _decode_dpt8,
    DPTInfo("8.001", "Pulses Difference", "pulses", -32768, 32767, 2),
)

# DPT 9.x — 2-byte float
_register(
    "9",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9", "2-byte Float", "", -671088.64, 670760.96, 2),
)
_register(
    "9.001",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9.001", "Temperature", "°C", -273, 670760, 2),
)
_register(
    "9.002",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9.002", "Temperature Diff", "K", -670760, 670760, 2),
)
_register(
    "9.003",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9.003", "Kelvin/Hour", "K/h", -670760, 670760, 2),
)
_register(
    "9.004", _encode_dpt9, _decode_dpt9, DPTInfo("9.004", "Lux", "lux", 0, 670760, 2)
)
_register(
    "9.005",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9.005", "Speed (m/s)", "m/s", 0, 670760, 2),
)
_register(
    "9.006",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9.006", "Pressure (Pa)", "Pa", 0, 670760, 2),
)
_register(
    "9.007", _encode_dpt9, _decode_dpt9, DPTInfo("9.007", "Humidity", "%", 0, 670760, 2)
)
_register(
    "9.008",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9.008", "Air Quality (ppm)", "ppm", 0, 670760, 2),
)
_register(
    "9.010",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9.010", "Time (s)", "s", -670760, 670760, 2),
)
_register(
    "9.011",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9.011", "Time (ms)", "ms", -670760, 670760, 2),
)
_register(
    "9.020",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9.020", "Voltage (mV)", "mV", -670760, 670760, 2),
)
_register(
    "9.021",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9.021", "Current (mA)", "mA", -670760, 670760, 2),
)
_register(
    "9.022",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9.022", "Power Density", "W/m²", -670760, 670760, 2),
)
_register(
    "9.024",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9.024", "Power (kW)", "kW", -670760, 670760, 2),
)
_register(
    "9.028",
    _encode_dpt9,
    _decode_dpt9,
    DPTInfo("9.028", "Wind Speed (km/h)", "km/h", 0, 670760, 2),
)

# DPT 10.x — Time of Day
_register(
    "10", _encode_dpt10, _decode_dpt10, DPTInfo("10", "Time of Day", "", None, None, 3)
)
_register(
    "10.001",
    _encode_dpt10,
    _decode_dpt10,
    DPTInfo("10.001", "Time of Day", "", None, None, 3),
)

# DPT 11.x — Date
_register("11", _encode_dpt11, _decode_dpt11, DPTInfo("11", "Date", "", None, None, 3))
_register(
    "11.001", _encode_dpt11, _decode_dpt11, DPTInfo("11.001", "Date", "", None, None, 3)
)

# DPT 12.x — Unsigned 32-bit
_register(
    "12",
    _encode_dpt12,
    _decode_dpt12,
    DPTInfo("12", "Unsigned 32-bit", "", 0, 4294967295, 4),
)
_register(
    "12.001",
    _encode_dpt12,
    _decode_dpt12,
    DPTInfo("12.001", "Counter Pulses", "pulses", 0, 4294967295, 4),
)

# DPT 13.x — Signed 32-bit
_register(
    "13",
    _encode_dpt13,
    _decode_dpt13,
    DPTInfo("13", "Signed 32-bit", "", -2147483648, 2147483647, 4),
)
_register(
    "13.001",
    _encode_dpt13,
    _decode_dpt13,
    DPTInfo("13.001", "Counter Pulses", "pulses", -2147483648, 2147483647, 4),
)
_register(
    "13.010",
    _encode_dpt13,
    _decode_dpt13,
    DPTInfo("13.010", "Active Energy (Wh)", "Wh", -2147483648, 2147483647, 4),
)
_register(
    "13.011",
    _encode_dpt13,
    _decode_dpt13,
    DPTInfo("13.011", "Reactive Energy (VARh)", "VARh", -2147483648, 2147483647, 4),
)
_register(
    "13.013",
    _encode_dpt13,
    _decode_dpt13,
    DPTInfo("13.013", "Active Energy (kWh)", "kWh", -2147483648, 2147483647, 4),
)

# DPT 14.x — 4-byte float
_register(
    "14", _encode_dpt14, _decode_dpt14, DPTInfo("14", "4-byte Float", "", None, None, 4)
)
_register(
    "14.019",
    _encode_dpt14,
    _decode_dpt14,
    DPTInfo("14.019", "Current (A)", "A", None, None, 4),
)
_register(
    "14.027",
    _encode_dpt14,
    _decode_dpt14,
    DPTInfo("14.027", "Electric Potential (V)", "V", None, None, 4),
)
_register(
    "14.033",
    _encode_dpt14,
    _decode_dpt14,
    DPTInfo("14.033", "Frequency (Hz)", "Hz", None, None, 4),
)
_register(
    "14.056",
    _encode_dpt14,
    _decode_dpt14,
    DPTInfo("14.056", "Power (W)", "W", None, None, 4),
)
_register(
    "14.068",
    _encode_dpt14,
    _decode_dpt14,
    DPTInfo("14.068", "Temperature (°C)", "°C", None, None, 4),
)
_register(
    "14.076",
    _encode_dpt14,
    _decode_dpt14,
    DPTInfo("14.076", "Volume Flow (m³/h)", "m³/h", None, None, 4),
)

# DPT 16.x — String
_register(
    "16", _encode_dpt16, _decode_dpt16, DPTInfo("16", "String", "", None, None, 14)
)
_register(
    "16.000",
    _encode_dpt16,
    _decode_dpt16,
    DPTInfo("16.000", "ASCII String", "", None, None, 14),
)
_register(
    "16.001",
    _encode_dpt16,
    _decode_dpt16,
    DPTInfo("16.001", "ISO 8859-1 String", "", None, None, 14),
)

# DPT 17.x — Scene Number
_register(
    "17", _encode_dpt17, _decode_dpt17, DPTInfo("17", "Scene Number", "", 0, 63, 1)
)
_register(
    "17.001",
    _encode_dpt17,
    _decode_dpt17,
    DPTInfo("17.001", "Scene Number", "", 0, 63, 1),
)

# DPT 18.x — Scene Control
_register(
    "18",
    _encode_dpt18,
    _decode_dpt18,
    DPTInfo("18", "Scene Control", "", None, None, 1),
)
_register(
    "18.001",
    _encode_dpt18,
    _decode_dpt18,
    DPTInfo("18.001", "Scene Control", "", None, None, 1),
)

# DPT 20.x — HVAC Mode
_register("20", _encode_dpt20, _decode_dpt20, DPTInfo("20", "HVAC Mode", "", 0, 255, 1))
_register(
    "20.102", _encode_dpt20, _decode_dpt20, DPTInfo("20.102", "HVAC Mode", "", 0, 4, 1)
)


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


class DPTCodec:
    """Central access point for DPT encoding/decoding."""

    @staticmethod
    def encode(dpt_id: str, value: Any) -> bytes:
        """Encode a Python value to KNX bytes for the given DPT."""
        entry = _REGISTRY.get(dpt_id)
        if not entry:
            # Try main type fallback
            main = dpt_id.split(".")[0]
            entry = _REGISTRY.get(main)
        if not entry:
            raise ValueError(f"Unknown DPT: {dpt_id}")
        return entry[0](value)

    @staticmethod
    def decode(dpt_id: str, data: bytes) -> Any:
        """Decode KNX bytes to a Python value for the given DPT."""
        entry = _REGISTRY.get(dpt_id)
        if not entry:
            main = dpt_id.split(".")[0]
            entry = _REGISTRY.get(main)
        if not entry:
            raise ValueError(f"Unknown DPT: {dpt_id}")
        return entry[1](data)

    @staticmethod
    def get_info(dpt_id: str) -> Optional[DPTInfo]:
        """Get metadata for a DPT."""
        entry = _REGISTRY.get(dpt_id)
        if not entry:
            main = dpt_id.split(".")[0]
            entry = _REGISTRY.get(main)
        return entry[2] if entry else None

    @staticmethod
    def list_dpts() -> list[dict]:
        """List all registered DPTs with metadata."""
        seen = set()
        result = []
        for dpt_id, (_, _, info) in sorted(_REGISTRY.items()):
            if dpt_id not in seen:
                seen.add(dpt_id)
                result.append(info.to_dict())
        return result

    @staticmethod
    def is_supported(dpt_id: str) -> bool:
        """Check if a DPT is supported."""
        if dpt_id in _REGISTRY:
            return True
        return dpt_id.split(".")[0] in _REGISTRY


# Module-level convenience functions
def encode(dpt_id: str, value: Any) -> bytes:
    return DPTCodec.encode(dpt_id, value)


def decode(dpt_id: str, data: bytes) -> Any:
    return DPTCodec.decode(dpt_id, data)


def get_dpt_info(dpt_id: str) -> Optional[dict]:
    info = DPTCodec.get_info(dpt_id)
    return info.to_dict() if info else None
