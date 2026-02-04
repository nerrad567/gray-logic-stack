"""Tests for GA normalization helpers."""

from __future__ import annotations

import logging

from devices.ga import normalise_group_addresses


def test_normalise_group_addresses_string_form_warns_and_normalises(caplog):
    """String-form GAs should normalise and emit a deprecation warning."""
    caplog.set_level(logging.WARNING, logger="knxsim.devices.ga")

    result = normalise_group_addresses(
        {"switch": "1/2/3"},
        device_type="light_switch",
    )

    assert result == {
        "switch": {
            "ga": "1/2/3",
            "dpt": "1.001",
            "flags": "CWU",
            "direction": "write",
            "description": "",
        }
    }
    assert "deprecated" in caplog.text


def test_normalise_group_addresses_dict_form_fills_defaults():
    """Dict-form GA should be filled with defaults when missing fields."""
    result = normalise_group_addresses(
        {"switch_status": {"ga": "1/2/4"}},
        device_type="light_switch",
    )

    assert result == {
        "switch_status": {
            "ga": "1/2/4",
            "dpt": "1.001",
            "flags": "CRT",
            "direction": "write",
            "description": "",
        }
    }


def test_normalise_group_addresses_empty_or_none():
    """Empty or None input should yield empty mapping."""
    assert normalise_group_addresses({}, device_type="light_switch") == {}
    assert normalise_group_addresses(None, device_type="light_switch") == {}
