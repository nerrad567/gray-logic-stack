"""Premise — a single simulated KNX installation.

Wraps a KNXnet/IP server, device registry, telegram dispatcher,
and scenario runner into one cohesive unit. Multiple premises can
run concurrently on different UDP ports.
"""

import logging
from collections.abc import Callable

from devices.base import BaseDevice
from devices.blind import Blind
from devices.light_dimmer import LightDimmer
from devices.light_switch import LightSwitch
from devices.presence import PresenceSensor
from devices.sensor import Sensor
from devices.template_device import TemplateDevice
from knxip import frames
from knxip.server import KNXIPServer
from scenarios.periodic import ScenarioRunner

logger = logging.getLogger("knxsim.premise")

# Base device types
DEVICE_TYPES = {
    "light_switch": LightSwitch,
    "light_dimmer": LightDimmer,
    "blind": Blind,
    "sensor": Sensor,
    "presence": PresenceSensor,
    "thermostat": Sensor,  # Thermostat uses sensor-like behavior
    "template_device": TemplateDevice,
}

# Multi-channel device types mapped to appropriate base classes
# The multi-channel behavior is handled at DB/API level; runtime uses base class
MULTI_CHANNEL_DEVICE_TYPES = {
    # Switch actuators (2/4/8/12/16/24-fold)
    "switch_actuator_2fold": LightSwitch,
    "switch_actuator_4fold": LightSwitch,
    "switch_actuator_8fold": LightSwitch,
    "switch_actuator_12fold": LightSwitch,
    "switch_actuator_16fold": LightSwitch,
    "switch_actuator_24fold": LightSwitch,
    # Dimmer actuators (1/2/4-fold)
    "dimmer_actuator_1fold": LightDimmer,
    "dimmer_actuator_2fold": LightDimmer,
    "dimmer_actuator_4fold": LightDimmer,
    # Blind/shutter actuators (2/4/8-fold)
    "blind_actuator_2fold": Blind,
    "blind_actuator_4fold": Blind,
    "blind_actuator_8fold": Blind,
    # Push button interfaces (2/4/6/8-fold)
    "push_button_2fold": TemplateDevice,
    "push_button_4fold": TemplateDevice,
    "push_button_6fold": TemplateDevice,
    "push_button_8fold": TemplateDevice,
    # Binary inputs (4/8/16-fold)
    "binary_input_4fold": Sensor,
    "binary_input_8fold": Sensor,
    "binary_input_16fold": Sensor,
}

# Merge multi-channel types into main lookup
DEVICE_TYPES.update(MULTI_CHANNEL_DEVICE_TYPES)


class Premise:
    """A single simulated KNX installation with its own server and devices."""

    def __init__(
        self,
        premise_id: str,
        name: str,
        gateway_address: str = "1.0.0",
        client_address: str = "1.0.255",
        port: int = 3671,
        on_telegram: Callable | None = None,
        on_state_change: Callable | None = None,
    ):
        self.id = premise_id
        self.name = name
        self.port = port
        self.gateway_address = frames.parse_individual_address(gateway_address)
        self.client_address = frames.parse_individual_address(client_address)

        self.devices: dict[str, BaseDevice] = {}
        self._ga_map: dict[int, BaseDevice] = {}
        self._on_telegram = on_telegram
        self._on_state_change = on_state_change

        self.server: KNXIPServer | None = None
        self.scenario_runner: ScenarioRunner | None = None
        self._running = False

    def add_device(
        self,
        device_id: str,
        device_type: str,
        individual_address: str,
        group_addresses: dict,
        initial_state: dict,
        config: dict | None = None,
    ) -> BaseDevice:
        """Create and register a device in this premise."""
        cls = DEVICE_TYPES.get(device_type)
        if not cls:
            raise ValueError(f"Unknown device type: {device_type}")

        ind_addr = frames.parse_individual_address(individual_address)
        gas = {}
        for name, ga_str in group_addresses.items():
            parsed = frames.parse_group_address(ga_str)
            if parsed is not None:
                gas[name] = parsed

        if cls is TemplateDevice:
            template_def = config.get("template_def", {}) if config else {}
            device = cls(device_id, ind_addr, gas, initial_state, template_def=template_def)
        else:
            device = cls(device_id, ind_addr, gas, initial_state)
        self.devices[device_id] = device

        # Update GA lookup (multiple devices can share the same GA)
        for ga in gas.values():
            if ga not in self._ga_map:
                self._ga_map[ga] = []
            if device not in self._ga_map[ga]:
                self._ga_map[ga].append(device)

        logger.info(
            "Device added: %s (%s) [%s] port=%d",
            device_id,
            device_type,
            frames.format_individual_address(ind_addr),
            self.port,
        )
        return device

    def remove_device(self, device_id: str) -> bool:
        """Remove a device from this premise."""
        device = self.devices.pop(device_id, None)
        if not device:
            return False

        # Remove device from GA mappings
        for ga, devices in list(self._ga_map.items()):
            if device in devices:
                devices.remove(device)
            if not devices:
                del self._ga_map[ga]
        logger.info("Device removed: %s from premise %s", device_id, self.id)
        return True

    def on_telegram(self, channel, cemi_dict: dict) -> list[bytes] | None:
        """Handle an incoming telegram from a KNXnet/IP client.

        Returns a list of response cEMI frames (may be empty or contain multiple).
        All devices listening on the destination GA receive the telegram.
        """
        from knxip import constants as C

        dst_ga = cemi_dict["dst"]
        apci = cemi_dict["apci"]
        payload = cemi_dict["payload"]

        # Notify observers
        if self._on_telegram:
            self._on_telegram(self.id, cemi_dict)

        devices = self._ga_map.get(dst_ga)
        if not devices:
            return None

        responses = []

        if apci == C.APCI_GROUP_WRITE:
            # Dispatch to ALL devices listening on this GA (real KNX behavior)
            for device in devices:
                result = device.on_group_write(dst_ga, payload)
                # Notify state change
                if self._on_state_change:
                    self._on_state_change(self.id, device.device_id, dict(device.state))
                # Collect responses
                if result:
                    if isinstance(result, bytes):
                        responses.append(result)
                    elif isinstance(result, list):
                        responses.extend(result)
            return responses if responses else None

        elif apci == C.APCI_GROUP_READ:
            # For read, return first device's response
            for device in devices:
                result = device.on_group_read(dst_ga)
                if result:
                    return [result]
            return None

        return None

    def _dispatch_telegram(self, ga: int, payload: bytes):
        """Dispatch a GroupWrite to local devices listening on the GA.

        This simulates the KNX bus behavior where all devices on the same GA
        receive the telegram. Used when a wall switch button sends a command
        that should be received by lights/actuators on the same GA.
        """
        devices = self._ga_map.get(ga)
        if not devices:
            return

        for device in devices:
            # Call the device's GroupWrite handler
            result = device.on_group_write(ga, payload)

            # Notify state change
            if self._on_state_change:
                self._on_state_change(self.id, device.device_id, dict(device.state))

            # If device returned a response (status telegram), send it
            if result and self.server:
                if isinstance(result, bytes):
                    self._send_telegram_with_hook(result)
                elif isinstance(result, list):
                    for r in result:
                        self._send_telegram_with_hook(r)

    def _send_telegram_with_hook(self, cemi: bytes):
        """Wrap server.send_telegram to also notify observers of outgoing telegrams."""
        self.server.send_telegram(cemi)

        # Decode the cEMI to notify observers
        if len(cemi) >= 8:
            try:
                decoded = frames.decode_cemi(cemi)
                if decoded:
                    # Notify telegram observers
                    if self._on_telegram:
                        decoded["_direction"] = "tx"
                        self._on_telegram(self.id, decoded)

                    # Notify state change (scenario already updated device.state)
                    if self._on_state_change:
                        src = decoded.get("src", 0)
                        for dev in self.devices.values():
                            if dev.individual_address == src:
                                self._on_state_change(self.id, dev.device_id, dict(dev.state))
                                break
            except Exception:
                pass  # Don't break scenario on decode errors

    def start(self):
        """Start the KNXnet/IP server and scenario runner."""
        if self._running:
            return

        self.server = KNXIPServer(
            host="0.0.0.0",
            port=self.port,
            client_address=self.client_address,
            gateway_address=self.gateway_address,
            on_telegram=self.on_telegram,
        )
        self.server.start()

        self.scenario_runner = ScenarioRunner(send_telegram=self._send_telegram_with_hook)

        self._running = True
        logger.info(
            "Premise started: %s (%s) on port %d with %d device(s)",
            self.id,
            self.name,
            self.port,
            len(self.devices),
        )

    def stop(self):
        """Stop the server and scenarios."""
        if not self._running:
            return

        if self.scenario_runner:
            self.scenario_runner.stop()
        if self.server:
            self.server.stop()

        self._running = False
        logger.info("Premise stopped: %s", self.id)

    @property
    def is_running(self) -> bool:
        return self._running

    def get_device_states(self) -> dict[str, dict]:
        """Get current state of all devices."""
        return {dev_id: dict(dev.state) for dev_id, dev in self.devices.items()}
