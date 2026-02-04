"""PremiseManager — orchestrates multiple simulated KNX installations.

Bridges between SQLite persistence and live Premise objects.
Handles bootstrapping from config.yaml on first run and
dynamic device/premise management via the API.
"""

import logging
from collections.abc import Callable

from persistence.db import Database

from .premise import Premise

logger = logging.getLogger("knxsim.manager")

DEFAULT_PREMISE_ID = "default"


class PremiseManager:
    """Manages multiple Premise instances with SQLite-backed persistence."""

    def __init__(
        self,
        db: Database,
        on_telegram: Callable | None = None,
        on_state_change: Callable | None = None,
        template_loader=None,
    ):
        self.db = db
        self.premises: dict[str, Premise] = {}
        self._on_telegram = on_telegram
        self._on_state_change = on_state_change
        self._template_loader = template_loader

    def bootstrap_from_config(self, config: dict):
        """Bootstrap the default premise from config.yaml if it doesn't exist in DB.

        On first run, creates an empty premise with setup_complete=False.
        The user will see a welcome modal to choose: load samples or start empty.
        """
        existing = self.db.get_premise(DEFAULT_PREMISE_ID)
        if existing:
            logger.info("Default premise already exists in database, loading...")
            self._load_premise_from_db(DEFAULT_PREMISE_ID)
            return

        # First run: create empty premise (user chooses via welcome modal)
        gw_cfg = config.get("gateway", {})
        gateway_addr = gw_cfg.get("individual_address", "1.0.0")
        client_addr = gw_cfg.get("client_address", "1.0.255")

        self.db.create_premise(
            {
                "id": DEFAULT_PREMISE_ID,
                "name": "Default Installation",
                "gateway_address": gateway_addr,
                "client_address": client_addr,
                "port": 3671,
                "setup_complete": False,  # User hasn't made a choice yet
            }
        )

        logger.info("Created empty premise '%s' — awaiting user setup choice", DEFAULT_PREMISE_ID)

        # Load the empty premise (starts KNX server)
        self._load_premise_from_db(DEFAULT_PREMISE_ID)
        return

        # NOTE: The code below is now DEAD CODE — kept for reference
        # Device import now happens via reset-sample endpoint when user chooses

        # Import devices from config
        for dev_cfg in config.get("devices", []):
            device_data = {
                "id": dev_cfg["id"],
                "type": dev_cfg["type"],
                "individual_address": dev_cfg["individual_address"],
                "group_addresses": dev_cfg.get("group_addresses", {}),
                "initial_state": dev_cfg.get("initial", {}),
                "state": dev_cfg.get("initial", {}),
            }

            # Handle template_device types - load template definition
            if dev_cfg["type"] == "template_device" and self._template_loader:
                template_id = dev_cfg.get("template")
                if template_id:
                    template = self._template_loader.get_template(template_id)
                    if template:
                        device_data["config"] = {
                            "template_id": template_id,
                            "template_def": template.group_addresses,
                        }
                        # Use template's initial state as defaults, override with config
                        merged_state = dict(template.initial_state)
                        merged_state.update(dev_cfg.get("initial", {}))
                        device_data["initial_state"] = merged_state
                        device_data["state"] = merged_state
                    else:
                        logger.warning(
                            "Template not found for device %s: %s",
                            dev_cfg["id"],
                            template_id,
                        )

            self.db.create_device(DEFAULT_PREMISE_ID, device_data)

        # Import scenarios from config
        for i, sc_cfg in enumerate(config.get("scenarios", [])):
            self.db.create_scenario(
                DEFAULT_PREMISE_ID,
                {
                    "id": f"default-scenario-{i}",
                    "device_id": sc_cfg["device_id"],
                    "field": sc_cfg.get("field", "temperature"),
                    "type": sc_cfg.get("type", "sine_wave"),
                    "params": sc_cfg.get("params", {}),
                    "enabled": True,
                },
            )

        logger.info(
            "Bootstrapped default premise from config: %d devices, %d scenarios",
            len(config.get("devices", [])),
            len(config.get("scenarios", [])),
        )

        self._load_premise_from_db(DEFAULT_PREMISE_ID)

    def _load_premise_from_db(self, premise_id: str):
        """Load a premise from the database and start it."""
        premise_data = self.db.get_premise(premise_id)
        if not premise_data:
            raise ValueError(f"Premise not found: {premise_id}")

        premise = Premise(
            premise_id=premise_data["id"],
            name=premise_data["name"],
            gateway_address=premise_data["gateway_address"],
            client_address=premise_data["client_address"],
            port=premise_data["port"],
            on_telegram=self._on_telegram,
            on_state_change=self._on_state_change,
        )

        # Load devices
        devices = self.db.list_devices(premise_id)
        for dev in devices:
            premise.add_device(
                device_id=dev["id"],
                device_type=dev["type"],
                individual_address=dev["individual_address"],
                group_addresses=dev["group_addresses"],
                initial_state=dev.get("initial_state", {}),
                config=dev.get("config"),
            )
            # Restore persisted state if different from initial
            if dev.get("state") and dev["state"] != dev.get("initial_state", {}):
                device_obj = premise.devices[dev["id"]]
                device_obj.state.update(dev["state"])
                # Recalculate derived values (e.g., thermostat heating output)
                if hasattr(device_obj, "recalculate_output"):
                    device_obj.recalculate_output()

        # Register premise before starting so callbacks can resolve devices
        self.premises[premise_id] = premise

        # Start the premise (UDP server)
        premise.start()

        # Load and start scenarios
        scenarios = self.db.list_scenarios(premise_id)
        for sc in scenarios:
            if not sc["enabled"]:
                continue
            device = premise.devices.get(sc["device_id"])
            if device:
                premise.scenario_runner.add_scenario(
                    device=device,
                    field=sc["field"],
                    scenario_type=sc["type"],
                    params=sc.get("params", {}),
                )
        if scenarios:
            premise.scenario_runner.start()
        logger.info(
            "Premise loaded: %s (%d devices, %d scenarios)",
            premise_id,
            len(devices),
            len(scenarios),
        )

    def load_all_premises(self):
        """Load all premises from the database."""
        for premise_data in self.db.list_premises():
            if premise_data["id"] not in self.premises:
                self._load_premise_from_db(premise_data["id"])

    # ------------------------------------------------------------------
    # Premise CRUD
    # ------------------------------------------------------------------

    def create_premise(self, data: dict) -> dict:
        """Create a new premise and start its server."""
        result = self.db.create_premise(data)
        self._load_premise_from_db(data["id"])
        return result

    def get_premise(self, premise_id: str) -> dict | None:
        return self.db.get_premise(premise_id)

    def list_premises(self) -> list[dict]:
        premises = self.db.list_premises()
        # Enrich with runtime info
        for p in premises:
            live = self.premises.get(p["id"])
            p["running"] = live.is_running if live else False
            p["device_count"] = len(live.devices) if live else 0
        return premises

    def delete_premise(self, premise_id: str) -> bool:
        """Stop and remove a premise."""
        live = self.premises.pop(premise_id, None)
        if live:
            live.stop()
        return self.db.delete_premise(premise_id)

    # ------------------------------------------------------------------
    # Device CRUD (live + persisted)
    # ------------------------------------------------------------------

    def add_device(self, premise_id: str, data: dict) -> dict | None:
        """Add a device to a premise (persists + adds to live server)."""
        premise = self.premises.get(premise_id)
        if not premise:
            return None

        # Persist
        result = self.db.create_device(premise_id, data)

        # Add to live premise
        premise.add_device(
            device_id=result["id"],
            device_type=result["type"],
            individual_address=result["individual_address"],
            group_addresses=result.get("group_addresses", {}),
            initial_state=result.get("initial_state", {}),
            config=result.get("config"),
        )

        return result

    def remove_device(self, premise_id: str, device_id: str) -> bool:
        """Remove a device from a premise (persists + removes from live server)."""
        premise = self.premises.get(premise_id)
        if premise:
            premise.remove_device(device_id)
        return self.db.delete_device(device_id)

    def update_device(self, premise_id: str, device_id: str, data: dict) -> dict | None:
        """Update device configuration.

        If group_addresses are changed, the running device is also updated
        so that changes take effect immediately without restart.
        """
        result = self.db.update_device(device_id, data)

        if result:
            premise = self.premises.get(premise_id)
            if premise:
                device = premise.devices.get(device_id)
                if device:
                    # Update individual address if changed
                    if "individual_address" in data:
                        from knxip import frames

                        device.individual_address = frames.parse_individual_address(
                            result["individual_address"]
                        )

                    # If group_addresses changed, update the running device
                    if "group_addresses" in data:
                        new_gas = data["group_addresses"]
                        old_gas = device.group_addresses.copy()

                        # Remove old GA mappings from premise._ga_map
                        for ga_int in old_gas.values():
                            if ga_int in premise._ga_map:
                                premise._ga_map[ga_int] = [
                                    d for d in premise._ga_map[ga_int] if d != device
                                ]
                                if not premise._ga_map[ga_int]:
                                    del premise._ga_map[ga_int]

                        # Parse new GAs and update device
                        def _parse_ga(ga_value: str | dict) -> int | None:
                            """Convert group address to integer.

                            Handles both legacy string format ('1/2/3') and
                            extended object format ({'ga': '1/2/3', 'dpt': '1.001'}).

                            Returns None if the address is empty or invalid.
                            """
                            # Handle extended format: extract the 'ga' field
                            if isinstance(ga_value, dict):
                                ga_str = ga_value.get("ga", "")
                            else:
                                ga_str = ga_value

                            # Handle empty strings
                            if not ga_str or not isinstance(ga_str, str):
                                return None
                            ga_str = ga_str.strip()
                            if not ga_str:
                                return None

                            try:
                                parts = ga_str.split("/")
                                if len(parts) == 3:
                                    main, middle, sub = (
                                        int(parts[0]),
                                        int(parts[1]),
                                        int(parts[2]),
                                    )
                                    return (main << 11) | (middle << 8) | sub
                                if len(parts) == 2:
                                    main, sub = int(parts[0]), int(parts[1])
                                    return (main << 11) | sub
                                return int(parts[0])
                            except (ValueError, IndexError):
                                return None

                        parsed_gas = {}
                        for name, ga_str in new_gas.items():
                            parsed = _parse_ga(ga_str)
                            if parsed is not None:
                                parsed_gas[name] = parsed
                        device.group_addresses = parsed_gas

                        # Add new GA mappings to premise._ga_map
                        for ga_int in parsed_gas.values():
                            if ga_int not in premise._ga_map:
                                premise._ga_map[ga_int] = []
                            if device not in premise._ga_map[ga_int]:
                                premise._ga_map[ga_int].append(device)

                        # For template devices, also rebuild _ga_info
                        if hasattr(device, "_ga_info") and hasattr(device, "_template_def"):
                            device._ga_info = {}
                            for slot_name, ga_int in parsed_gas.items():
                                slot_def = device._template_def.get(slot_name, {})
                                dpt = slot_def.get("dpt", "1.001")
                                direction = slot_def.get("direction", "write")
                                # For buttons/leds, use slot name directly as field
                                if slot_name.startswith("button_") or slot_name.startswith("led_"):
                                    field = slot_name
                                else:
                                    field = device._slot_to_field(slot_name)
                                if ga_int not in device._ga_info:
                                    device._ga_info[ga_int] = []
                                device._ga_info[ga_int].append((slot_name, field, dpt, direction))

        return result

    def list_devices(self, premise_id: str) -> list[dict]:
        """List devices with live state from the running premise."""
        devices = self.db.list_devices(premise_id)
        premise = self.premises.get(premise_id)
        if premise:
            for dev in devices:
                live_dev = premise.devices.get(dev["id"])
                if live_dev:
                    dev["state"] = dict(live_dev.state)
        return devices

    def get_device(self, device_id: str) -> dict | None:
        return self.db.get_device(device_id)

    def update_device_placement(self, device_id: str, room_id: str | None) -> dict | None:
        """Assign a device to a room (for floor plan placement)."""
        return self.db.update_device(device_id, {"room_id": room_id})

    # ------------------------------------------------------------------
    # Floor/Room CRUD
    # ------------------------------------------------------------------

    def list_floors(self, premise_id: str) -> list[dict]:
        floors = self.db.list_floors(premise_id)
        # Attach rooms to each floor
        for floor in floors:
            floor["rooms"] = self.db.list_rooms(floor["id"])
        return floors

    def create_floor(self, premise_id: str, data: dict) -> dict:
        return self.db.create_floor(premise_id, data)

    def update_floor(self, floor_id: str, data: dict) -> dict | None:
        return self.db.update_floor(floor_id, data)

    def delete_floor(self, floor_id: str) -> bool:
        return self.db.delete_floor(floor_id)

    def create_room(self, floor_id: str, data: dict) -> dict:
        return self.db.create_room(floor_id, data)

    def update_room(self, room_id: str, data: dict) -> dict | None:
        return self.db.update_room(room_id, data)

    def delete_room(self, room_id: str) -> bool:
        return self.db.delete_room(room_id)

    # ------------------------------------------------------------------
    # Lifecycle
    # ------------------------------------------------------------------

    def stop_all(self):
        """Stop all running premises."""
        for premise in self.premises.values():
            premise.stop()
        self.premises.clear()
        logger.info("All premises stopped")
