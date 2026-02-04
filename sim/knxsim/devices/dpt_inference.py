"""Heuristics for inferring DPTs and flags from GA names."""

from __future__ import annotations


def _guess_dpt_for_ga(device_type: str, ga_name: str) -> str:
    """Guess DPT based on device type and GA function name."""
    ga_lower = ga_name.lower()

    # Switch/status commands
    if "switch" in ga_lower or "on_off" in ga_lower:
        return "1.001"
    if "move" in ga_lower:
        return "1.008"
    if "stop" in ga_lower:
        return "1.017"
    if "presence" in ga_lower or "occupancy" in ga_lower:
        return "1.018"

    # Percentage values
    if "brightness" in ga_lower or "position" in ga_lower or "slat" in ga_lower:
        return "5.001"

    # Dimming
    if "dim" in ga_lower:
        return "3.007"

    # Temperature
    if "temp" in ga_lower or "setpoint" in ga_lower:
        return "9.001"

    # Humidity
    if "humidity" in ga_lower:
        return "9.007"

    # Lux
    if "lux" in ga_lower or "light_level" in ga_lower:
        return "9.004"

    # HVAC mode
    if "mode" in ga_lower and (
        "hvac" in device_type.lower() or "thermostat" in device_type.lower()
    ):
        return "20.102"

    # Default to switch for unknown
    return "1.001"


def _guess_flags_for_ga(ga_name: str) -> str:
    """Guess communication flags based on GA function name."""
    ga_lower = ga_name.lower()

    # Status objects are typically read + transmit
    if "status" in ga_lower:
        return "CRT"

    # Command objects are typically write + update
    if any(
        cmd in ga_lower
        for cmd in ["switch", "brightness", "position", "move", "stop", "setpoint", "mode"]
    ):
        return "CWU"

    # Sensor outputs are read + transmit
    if any(sensor in ga_lower for sensor in ["temperature", "humidity", "lux", "presence", "co2"]):
        return "CRT"

    # Default to write
    return "CW"
