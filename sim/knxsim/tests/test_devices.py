"""Unit tests for KNXSim device classes."""

from __future__ import annotations

import pytest

from devices.base import encode_dpt1, encode_dpt5, encode_dpt9
from devices.blind import Blind
from devices.light_dimmer import LightDimmer
from devices.light_switch import LightSwitch
from devices.presence import PresenceSensor
from devices.sensor import Sensor
from devices.template_device import TemplateDevice
from devices.thermostat import Thermostat
from devices.valve_actuator import ValveActuator
from knxip import frames


def test_light_switch_toggle():
    dev = LightSwitch(
        device_id="ls-1",
        individual_address=frames.parse_individual_address("1.1.1"),
        group_addresses={
            "switch_cmd": frames.parse_group_address("0/0/1"),
            "switch_status": frames.parse_group_address("0/0/2"),
        },
        initial_state={"on": False},
    )
    resp = dev.on_group_write(frames.parse_group_address("0/0/1"), encode_dpt1(True))
    assert dev.state["on"] is True
    assert resp is not None


def test_light_dimmer_brightness():
    dev = LightDimmer(
        device_id="ld-1",
        individual_address=frames.parse_individual_address("1.1.2"),
        group_addresses={
            "switch_cmd": frames.parse_group_address("0/0/1"),
            "switch_status": frames.parse_group_address("0/0/2"),
            "brightness_cmd": frames.parse_group_address("0/0/3"),
            "brightness_status": frames.parse_group_address("0/0/4"),
        },
        initial_state={"on": False, "brightness": 0},
    )
    resp = dev.on_group_write(
        frames.parse_group_address("0/0/3"), encode_dpt5(50)
    )
    assert dev.state["brightness"] == pytest.approx(50, abs=1)
    assert dev.state["on"] is True
    assert isinstance(resp, list)


def test_blind_position_and_tilt():
    dev = Blind(
        device_id="blind-1",
        individual_address=frames.parse_individual_address("1.1.3"),
        group_addresses={
            "position_cmd": frames.parse_group_address("0/0/5"),
            "position_status": frames.parse_group_address("0/0/6"),
            "slat_cmd": frames.parse_group_address("0/0/7"),
            "slat_status": frames.parse_group_address("0/0/8"),
        },
        initial_state={"position": 0, "tilt": 0},
    )
    resp = dev.on_group_write(
        frames.parse_group_address("0/0/5"), encode_dpt5(25)
    )
    assert dev.state["position"] == pytest.approx(25, abs=1)
    assert resp is not None


def test_thermostat_setpoint_and_output():
    dev = Thermostat(
        device_id="th-1",
        individual_address=frames.parse_individual_address("1.1.4"),
        group_addresses={
            "current_temperature": frames.parse_group_address("1/0/1"),
            "setpoint": frames.parse_group_address("1/0/2"),
            "setpoint_status": frames.parse_group_address("1/0/3"),
            "heating_output": frames.parse_group_address("1/0/4"),
        },
        initial_state={"current_temperature": 20.0, "setpoint": 21.0},
    )
    resp = dev.on_group_write(
        frames.parse_group_address("1/0/2"), encode_dpt9(23.0)
    )
    assert dev.state["setpoint"] == pytest.approx(23.0, abs=0.1)
    assert isinstance(resp, list)
    assert dev.state["heating_output"] >= 0


def test_sensor_value_updates():
    dev = Sensor(
        device_id="sensor-1",
        individual_address=frames.parse_individual_address("1.1.5"),
        group_addresses={"temperature": frames.parse_group_address("2/0/1")},
        initial_state={"temperature": 20.0},
    )
    resp = dev.on_group_write(
        frames.parse_group_address("2/0/1"), encode_dpt9(21.5)
    )
    assert dev.state["temperature"] == pytest.approx(21.5, abs=0.1)
    assert resp is not None


def test_presence_sensor_updates():
    dev = PresenceSensor(
        device_id="ps-1",
        individual_address=frames.parse_individual_address("1.1.6"),
        group_addresses={
            "presence": frames.parse_group_address("3/0/1"),
            "lux": frames.parse_group_address("3/0/2"),
        },
        initial_state={"presence": False, "lux": 0.0},
    )
    resp = dev.on_group_write(
        frames.parse_group_address("3/0/1"), encode_dpt1(True)
    )
    assert dev.state["presence"] is True
    assert resp is not None


def test_valve_actuator_binary_and_percentage():
    dev = ValveActuator(
        device_id="va-1",
        individual_address=frames.parse_individual_address("1.1.7"),
        group_addresses={
            "valve_cmd": frames.parse_group_address("4/0/1"),
            "valve_status": frames.parse_group_address("4/0/2"),
            "position": frames.parse_group_address("4/0/3"),
            "position_status": frames.parse_group_address("4/0/4"),
        },
        initial_state={"position": 0, "on": False},
    )
    resp = dev.on_group_write(
        frames.parse_group_address("4/0/3"), encode_dpt5(75)
    )
    assert dev.state["position"] == pytest.approx(75, abs=1)
    assert isinstance(resp, list)

    resp = dev.on_group_write(
        frames.parse_group_address("4/0/1"), encode_dpt1(True)
    )
    assert dev.state["on"] is True
    assert dev.state["position"] == 100
    assert isinstance(resp, list)


def test_template_device_mapping():
    group_addresses = {
        "switch_cmd": frames.parse_group_address("5/0/1"),
        "switch_status": frames.parse_group_address("5/0/2"),
    }
    template_def = {
        "switch_cmd": {"dpt": "1.001", "direction": "write"},
        "switch_status": {"dpt": "1.001", "direction": "status"},
    }
    dev = TemplateDevice(
        device_id="td-1",
        individual_address=frames.parse_individual_address("1.1.8"),
        group_addresses=group_addresses,
        initial_state={"on": False},
        template_def=template_def,
    )
    resp = dev.on_group_write(
        frames.parse_group_address("5/0/1"), encode_dpt1(True)
    )
    assert dev.state["on"] is True
    assert resp is not None
