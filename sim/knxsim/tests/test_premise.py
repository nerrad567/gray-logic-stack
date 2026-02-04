"""Unit tests for Premise."""

from __future__ import annotations

from core.premise import Premise
from devices.base import encode_dpt1
from knxip import constants as C
from knxip import frames


def _create_premise() -> Premise:
    return Premise(
        premise_id="premise-1",
        name="Test Premise",
        gateway_address="1.0.0",
        client_address="1.0.255",
        port=0,
    )


def test_device_creation_and_state_updates():
    premise = _create_premise()
    premise.add_device(
        device_id="light-1",
        device_type="light_switch",
        individual_address="1.1.1",
        group_addresses={"switch_cmd": "0/0/1", "switch_status": "0/0/2"},
        initial_state={"on": False},
    )

    ga = frames.parse_group_address("0/0/1")
    premise._dispatch_telegram(ga, encode_dpt1(True))

    assert premise.devices["light-1"].state["on"] is True
    states = premise.get_device_states()
    assert states["light-1"]["on"] is True


def test_on_telegram_dispatch():
    premise = _create_premise()
    premise.add_device(
        device_id="light-1",
        device_type="light_switch",
        individual_address="1.1.2",
        group_addresses={"switch_cmd": "0/0/1", "switch_status": "0/0/2"},
        initial_state={"on": False},
    )

    cemi = {
        "dst": frames.parse_group_address("0/0/1"),
        "apci": C.APCI_GROUP_WRITE,
        "payload": encode_dpt1(True),
    }
    responses = premise.on_telegram(None, cemi)
    assert responses is not None
    assert premise.devices["light-1"].state["on"] is True


def test_scenario_lifecycle():
    premise = _create_premise()
    try:
        premise.start()
        assert premise.is_running is True
    finally:
        premise.stop()
    assert premise.is_running is False
