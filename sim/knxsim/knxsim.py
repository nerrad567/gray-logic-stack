"""KNX/IP Gateway Simulator — Main Entry Point.

Loads device configuration from config.yaml, creates virtual devices,
starts the KNXnet/IP tunnelling server on UDP 3671, and runs scenarios.

This script is the single process that handles everything:
  - KNXnet/IP protocol (UDP server)
  - Virtual device state machines
  - Periodic sensor scenarios
"""

import logging
import os
import signal
import sys
import threading
import time
from typing import Optional

# Use PyYAML if available, otherwise fall back to a simple parser
try:
    import yaml
except ImportError:
    yaml = None

from devices.base import BaseDevice
from devices.blind import Blind
from devices.light_dimmer import LightDimmer
from devices.light_switch import LightSwitch
from devices.sensor import Sensor
from knxip import constants as C
from knxip import frames
from knxip.server import KNXIPServer
from scenarios.periodic import ScenarioRunner

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------


def load_config(path: str) -> dict:
    """Load config.yaml using PyYAML or a basic fallback parser."""
    with open(path, "r") as f:
        if yaml:
            return yaml.safe_load(f)
        else:
            # Minimal YAML subset parser for our config format
            return _parse_yaml_minimal(f.read())


def _parse_yaml_minimal(text: str) -> dict:
    """Very basic YAML parser — handles our specific config structure.

    Only supports: scalars, lists of dicts, nested dicts (2 levels).
    Good enough for config.yaml without requiring pip install pyyaml.
    """
    import re

    result = {"gateway": {}, "devices": [], "scenarios": []}
    current_section = None
    current_item = None
    current_sub = None

    for line in text.split("\n"):
        stripped = line.rstrip()
        if not stripped or stripped.startswith("#"):
            continue

        indent = len(line) - len(line.lstrip())

        # Top-level keys
        if indent == 0 and ":" in stripped:
            key = stripped.split(":")[0].strip()
            current_section = key
            current_item = None
            current_sub = None
            continue

        # List item start
        if stripped.lstrip().startswith("- "):
            if current_section in ("devices", "scenarios"):
                current_item = {}
                result[current_section].append(current_item)
                current_sub = None
                # Parse key on same line as -
                rest = stripped.lstrip()[2:]
                if ":" in rest:
                    k, v = rest.split(":", 1)
                    current_item[k.strip()] = _parse_value(v.strip())
            continue

        # Nested dict or value
        if ":" in stripped:
            k, v = stripped.split(":", 1)
            k = k.strip()
            v = v.strip()

            if current_item is not None:
                if v == "" or v == "{}":
                    # Start of nested dict within list item
                    current_sub = k
                    current_item[k] = {}
                elif current_sub and indent >= 8:
                    current_item[current_sub][k] = _parse_value(v)
                else:
                    if v:
                        current_item[k] = _parse_value(v)
                    else:
                        current_sub = k
                        current_item[k] = {}
            elif current_section == "gateway":
                result["gateway"][k] = _parse_value(v)

    return result


def _parse_value(v: str):
    """Parse a YAML scalar value."""
    # Remove quotes
    if (v.startswith('"') and v.endswith('"')) or (
        v.startswith("'") and v.endswith("'")
    ):
        return v[1:-1]
    # Booleans
    if v.lower() in ("true", "yes"):
        return True
    if v.lower() in ("false", "no"):
        return False
    # Numbers
    try:
        if "." in v:
            return float(v)
        return int(v)
    except ValueError:
        return v


# ---------------------------------------------------------------------------
# Device Registry
# ---------------------------------------------------------------------------

DEVICE_TYPES = {
    "light_switch": LightSwitch,
    "light_dimmer": LightDimmer,
    "blind": Blind,
    "sensor": Sensor,
}


def create_devices(config: dict) -> list[BaseDevice]:
    """Instantiate devices from config."""
    devices = []
    for dev_cfg in config.get("devices", []):
        dev_type = dev_cfg.get("type", "")
        cls = DEVICE_TYPES.get(dev_type)
        if not cls:
            logging.warning("Unknown device type: %s", dev_type)
            continue

        # Parse addresses
        individual = frames.parse_individual_address(dev_cfg["individual_address"])
        gas = {}
        for name, ga_str in dev_cfg.get("group_addresses", {}).items():
            gas[name] = frames.parse_group_address(ga_str)

        initial = dev_cfg.get("initial", {})
        device = cls(dev_cfg["id"], individual, gas, initial)
        devices.append(device)

        ga_list = ", ".join(
            f"{n}={frames.format_group_address(g)}" for n, g in gas.items()
        )
        logging.info(
            "Device: %s (%s) [%s] GAs: %s",
            dev_cfg["id"],
            dev_type,
            frames.format_individual_address(individual),
            ga_list,
        )

    return devices


# ---------------------------------------------------------------------------
# Telegram Dispatcher
# ---------------------------------------------------------------------------


class TelegramDispatcher:
    """Routes incoming cEMI telegrams to the appropriate device."""

    def __init__(self, devices: list[BaseDevice]):
        self.devices = devices
        # Build GA → device lookup for fast dispatch
        self._ga_map: dict[int, BaseDevice] = {}
        for dev in devices:
            for ga in dev.group_addresses.values():
                self._ga_map[ga] = dev

    def on_telegram(self, channel, cemi_dict: dict) -> Optional[bytes]:
        """Handle an incoming telegram from knxd.

        Returns response cEMI bytes if the device generates one, else None.
        """
        dst_ga = cemi_dict["dst"]
        apci = cemi_dict["apci"]
        payload = cemi_dict["payload"]

        device = self._ga_map.get(dst_ga)
        if not device:
            logging.debug("No device for GA %s", frames.format_group_address(dst_ga))
            return None

        ga_str = frames.format_group_address(dst_ga)

        if apci == C.APCI_GROUP_WRITE:
            logging.info("%s ← GroupWrite [%s]", ga_str, payload.hex())
            return device.on_group_write(dst_ga, payload)

        elif apci == C.APCI_GROUP_READ:
            logging.info("%s ← GroupRead", ga_str)
            return device.on_group_read(dst_ga)

        elif apci == C.APCI_GROUP_RESPONSE:
            logging.debug("%s ← GroupResponse (ignored)", ga_str)
            return None

        return None


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main():
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(name)s] %(levelname)s %(message)s",
        datefmt="%H:%M:%S",
    )
    logger = logging.getLogger("knxsim")

    # Load configuration
    config_path = os.environ.get("KNXSIM_CONFIG", "/app/config.yaml")
    if not os.path.exists(config_path):
        # Try relative path for local development
        config_path = os.path.join(os.path.dirname(__file__), "config.yaml")

    logger.info("Loading config from %s", config_path)
    config = load_config(config_path)

    # Parse gateway config
    gw_cfg = config.get("gateway", {})
    gateway_addr = frames.parse_individual_address(
        gw_cfg.get("individual_address", "1.0.0")
    )
    client_addr = frames.parse_individual_address(
        gw_cfg.get("client_address", "1.0.255")
    )

    # Create devices
    devices = create_devices(config)
    if not devices:
        logger.error("No devices configured — exiting")
        sys.exit(1)

    # Create dispatcher
    dispatcher = TelegramDispatcher(devices)

    # Create and start KNXnet/IP server
    port = int(os.environ.get("KNXSIM_PORT", "3671"))
    server = KNXIPServer(
        host="0.0.0.0",
        port=port,
        client_address=client_addr,
        gateway_address=gateway_addr,
        on_telegram=dispatcher.on_telegram,
    )
    server.start()

    # Create and start scenarios
    scenario_runner = ScenarioRunner(send_telegram=server.send_telegram)
    for sc_cfg in config.get("scenarios", []):
        device_id = sc_cfg.get("device_id")
        device = next((d for d in devices if d.device_id == device_id), None)
        if not device:
            logger.warning("Scenario references unknown device: %s", device_id)
            continue
        if not isinstance(device, Sensor):
            logger.warning("Scenario device %s is not a sensor", device_id)
            continue
        scenario_runner.add_scenario(
            device=device,
            field=sc_cfg.get("field", "temperature"),
            scenario_type=sc_cfg.get("type", "sine_wave"),
            params=sc_cfg.get("params", {}),
        )
    scenario_runner.start()

    logger.info(
        "KNX/IP Gateway Simulator running (gateway=%s, client=%s, port=%d)",
        frames.format_individual_address(gateway_addr),
        frames.format_individual_address(client_addr),
        port,
    )
    logger.info(
        "Simulating %d device(s), %d scenario(s)",
        len(devices),
        len(config.get("scenarios", [])),
    )

    # Wait for shutdown signal
    shutdown = threading.Event()

    def handle_signal(signum, frame):
        logger.info("Received signal %d — shutting down", signum)
        shutdown.set()

    signal.signal(signal.SIGTERM, handle_signal)
    signal.signal(signal.SIGINT, handle_signal)

    shutdown.wait()

    # Cleanup
    scenario_runner.stop()
    server.stop()
    logger.info("Shutdown complete")


if __name__ == "__main__":
    main()
