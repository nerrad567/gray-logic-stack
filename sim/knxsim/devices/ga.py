"""Group address normalization helpers."""

from __future__ import annotations

import logging

from .dpt_inference import _guess_dpt_for_ga, _guess_flags_for_ga

logger = logging.getLogger("knxsim.devices.ga")


def normalise_ga(raw, device_type: str = "", ga_name: str = "") -> dict:
    """Normalise a GA value to canonical dict form.

    Canonical form: {ga, dpt, flags, direction, description}
    """
    if isinstance(raw, str):
        ga = raw.strip()
        if ga:
            logger.warning(
                "String-form GA '%s' is deprecated - use {ga, dpt, ...} format",
                ga,
            )
        return {
            "ga": ga,
            "dpt": _guess_dpt_for_ga(device_type, ga_name),
            "flags": _guess_flags_for_ga(ga_name),
            "direction": "write",
            "description": "",
        }

    if isinstance(raw, dict):
        ga = raw.get("ga", "")
        dpt = raw.get("dpt") or _guess_dpt_for_ga(device_type, ga_name)
        flags = raw.get("flags") or _guess_flags_for_ga(ga_name)
        direction = raw.get("direction", "write")
        description = raw.get("description", "")
        return {
            "ga": ga,
            "dpt": dpt,
            "flags": flags,
            "direction": direction,
            "description": description,
        }

    return {
        "ga": "",
        "dpt": _guess_dpt_for_ga(device_type, ga_name),
        "flags": _guess_flags_for_ga(ga_name),
        "direction": "write",
        "description": "",
    }


def normalise_group_addresses(group_addresses: dict | None, device_type: str = "") -> dict:
    """Normalise a mapping of GA slot names to values."""
    if not group_addresses:
        return {}

    return {
        ga_name: normalise_ga(raw, device_type=device_type, ga_name=ga_name)
        for ga_name, raw in group_addresses.items()
    }
