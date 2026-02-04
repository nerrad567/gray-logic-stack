"""Premise CRUD routes for the KNX Simulator Management API."""

import logging
import sqlite3

from fastapi import APIRouter, HTTPException

from .models import PremiseCreate, PremiseResponse

logger = logging.getLogger("knxsim.routes.premises")

router = APIRouter(prefix="/api/v1/premises", tags=["premises"])


@router.get("", response_model=list[PremiseResponse])
def list_premises():
    """List all premises with runtime status."""
    manager = router.app.state.manager
    return manager.list_premises()


@router.get("/{premise_id}", response_model=PremiseResponse)
def get_premise(premise_id: str):
    """Get a single premise by ID."""
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")
    # Enrich with runtime info
    live = manager.premises.get(premise_id)
    premise["running"] = live.is_running if live else False
    premise["device_count"] = len(live.devices) if live else 0
    return premise


@router.post("", response_model=PremiseResponse, status_code=201)
def create_premise(body: PremiseCreate):
    """Create a new premise and start its KNX server."""
    manager = router.app.state.manager
    # Check for duplicate
    if manager.get_premise(body.id):
        raise HTTPException(status_code=409, detail="Premise already exists")
    result = manager.create_premise(body.model_dump())
    result["running"] = True
    result["device_count"] = 0
    return result


@router.delete("/{premise_id}", status_code=204)
def delete_premise(premise_id: str):
    """Stop and delete a premise (cascades to devices, floors, rooms)."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    manager.delete_premise(premise_id)


@router.post("/{premise_id}/reset-sample")
def reset_to_sample(premise_id: str):
    """Reset premise to sample installation from config.yaml.

    This will:
    1. Delete all existing devices, floors, and rooms
    2. Create sample topology (floors and rooms)
    3. Create sample devices and assign them to appropriate rooms
    4. Mark the premise as setup_complete

    Useful for starting fresh or restoring the default learning environment.
    """
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")

    # Load config
    try:
        import yaml
    except ImportError as e:
        raise HTTPException(status_code=500, detail=f"Failed to import PyYAML: {e}") from e

    try:
        with open("/app/config.yaml") as f:
            config = yaml.safe_load(f)
    except (OSError, yaml.YAMLError) as e:
        raise HTTPException(status_code=500, detail=f"Failed to load config: {e}") from e

    if "devices" not in config:
        raise HTTPException(status_code=500, detail="No devices found in config.yaml")

    # Delete existing loads first (before devices, since loads reference devices)
    existing_loads = manager.db.list_loads(premise_id)
    for load in existing_loads:
        manager.db.delete_load(load["id"])

    # Delete existing devices
    existing_devices = manager.db.list_devices(premise_id)
    for device in existing_devices:
        manager.remove_device(premise_id, device["id"])

    # Delete existing floors (cascades to rooms)
    existing_floors = manager.db.list_floors(premise_id)
    for floor in existing_floors:
        manager.db.delete_floor(floor["id"])

    # ─────────────────────────────────────────────────────────────
    # Create sample topology
    # ─────────────────────────────────────────────────────────────

    SAMPLE_TOPOLOGY = {
        "ground": {
            "name": "Ground Floor",
            "sort_order": 0,
            "rooms": [
                {"id": "living-room", "name": "Living Room", "room_type": "living"},
                {"id": "kitchen", "name": "Kitchen", "room_type": "kitchen"},
                {"id": "hallway", "name": "Hallway", "room_type": "hallway"},
                {"id": "distribution-board", "name": "Distribution Board", "room_type": "utility"},
            ],
        },
        "first": {
            "name": "First Floor",
            "sort_order": 1,
            "rooms": [
                {"id": "bedroom", "name": "Bedroom", "room_type": "bedroom"},
                {"id": "bathroom", "name": "Bathroom", "room_type": "bathroom"},
            ],
        },
    }

    # Device-to-room mapping based on device ID patterns
    # Format: substring in device_id -> room_id
    # Order matters: more specific patterns should come first
    DEVICE_ROOM_MAP = [
        # Centralized equipment (actuators in distribution board)
        ("actuator", "distribution-board"),
        ("binary-input", "distribution-board"),
        # Room-specific devices (lights, sensors, controls, switches)
        # Match room name in device ID
        ("-living", "living-room"),
        ("-kitchen", "kitchen"),
        ("-hallway", "hallway"),
        ("-bedroom", "bedroom"),
        ("-bathroom", "bathroom"),
    ]

    # For non-default premises, prefix IDs to avoid collisions
    prefix = f"{premise_id}-" if premise_id != "default" else ""

    floors_created = 0
    rooms_created = 0

    # Map from bare room ID → scoped room ID (for device/load assignment)
    room_id_map = {}

    for floor_id, floor_data in SAMPLE_TOPOLOGY.items():
        scoped_floor_id = f"{prefix}{floor_id}"
        manager.db.create_floor(
            premise_id,
            {
                "id": scoped_floor_id,
                "name": floor_data["name"],
                "sort_order": floor_data["sort_order"],
            },
        )
        floors_created += 1

        for room_data in floor_data["rooms"]:
            scoped_room = {**room_data, "id": f"{prefix}{room_data['id']}"}
            room_id_map[room_data["id"]] = scoped_room["id"]
            manager.db.create_room(scoped_floor_id, scoped_room)
            rooms_created += 1

    # ─────────────────────────────────────────────────────────────
    # Create sample devices
    # ─────────────────────────────────────────────────────────────

    def guess_room_for_device(device_id: str, group_addresses: dict) -> str | None:
        """Guess which room a device belongs to based on its ID."""
        device_id_lower = device_id.lower()
        # Check patterns in order (more specific first)
        for pattern, bare_room_id in DEVICE_ROOM_MAP:
            if pattern in device_id_lower:
                return room_id_map.get(bare_room_id, bare_room_id)
        # Default: unassigned
        return None

    devices_created = 0
    # Map from bare device ID → prefixed device ID (for scenarios/loads)
    device_id_map = {}

    for dev_config in config.get("devices", []):
        bare_id = dev_config["id"]
        prefixed_id = f"{prefix}{bare_id}"
        device_id_map[bare_id] = prefixed_id

        device_data = {
            "id": prefixed_id,
            "type": dev_config["type"],
            "individual_address": dev_config["individual_address"],
            "group_addresses": dev_config.get("group_addresses", {}),
            "initial_state": dev_config.get("initial", {}),
            "room_id": dev_config.get("room_id")
            or guess_room_for_device(
                dev_config["id"],
                dev_config.get("group_addresses", {}),
            ),
        }

        # Handle template devices
        if dev_config.get("template"):
            device_data["config"] = {"template": dev_config["template"]}

        # Inject manufacturer metadata from matching template (if available)
        loader = router.app.state.template_loader
        template = loader.get_template(dev_config["type"])
        if template and template.manufacturer_name:
            dev_cfg = device_data.get("config") or {}
            dev_cfg["template_id"] = template.id
            dev_cfg["manufacturer_id"] = template.manufacturer_id
            dev_cfg["manufacturer_name"] = template.manufacturer_name
            dev_cfg["product_model"] = template.product_model
            dev_cfg["application_program"] = template.application_program
            dev_cfg["hardware_type"] = template.hardware_type
            device_data["config"] = dev_cfg

        try:
            manager.add_device(premise_id, device_data)
            devices_created += 1
        except (KeyError, ValueError, TypeError) as e:
            logger.warning("Failed to create device %s: %s", prefixed_id, e)

    # ─────────────────────────────────────────────────────────────
    # Reload scenarios
    # ─────────────────────────────────────────────────────────────

    scenarios_created = 0
    live_premise = manager.premises.get(premise_id)
    if live_premise and live_premise.scenario_runner:
        live_premise.scenario_runner.stop()
        live_premise.scenario_runner = None

    for scenario_config in config.get("scenarios", []):
        bare_dev_id = scenario_config["device_id"]
        prefixed_dev_id = device_id_map.get(bare_dev_id, f"{prefix}{bare_dev_id}")
        try:
            manager.db.create_scenario(
                premise_id,
                {
                    "id": f"{prefix}scenario-{bare_dev_id}-{scenario_config['field']}",
                    "device_id": prefixed_dev_id,
                    "field": scenario_config["field"],
                    "type": scenario_config["type"],
                    "params": scenario_config.get("params", {}),
                },
            )
            scenarios_created += 1
        except sqlite3.IntegrityError:
            pass  # May already exist

    # ─────────────────────────────────────────────────────────────
    # Create loads (physical equipment controlled by actuators)
    # ─────────────────────────────────────────────────────────────

    # Load-to-room mapping based on load ID patterns
    LOAD_ROOM_MAP = [
        ("-living", "living-room"),
        ("-kitchen", "kitchen"),
        ("-hallway", "hallway"),
        ("-bedroom", "bedroom"),
        ("-bathroom", "bathroom"),
    ]

    def guess_room_for_load(load_id: str) -> str | None:
        """Guess which room a load belongs to based on its ID."""
        load_id_lower = load_id.lower()
        for pattern, bare_room_id in LOAD_ROOM_MAP:
            if pattern in load_id_lower:
                return room_id_map.get(bare_room_id, bare_room_id)
        return None

    loads_created = 0
    for load_config in config.get("loads", []):
        bare_actuator_id = load_config.get("actuator_device_id")
        load_data = {
            "id": f"{prefix}{load_config['id']}",
            "name": load_config["name"],
            "type": load_config["type"],
            "icon": load_config.get("icon"),
            "actuator_device_id": device_id_map.get(bare_actuator_id, bare_actuator_id)
            if bare_actuator_id
            else None,
            "actuator_channel_id": load_config.get("actuator_channel_id"),
            "room_id": load_config.get("room_id") or guess_room_for_load(load_config["id"]),
        }

        try:
            manager.db.create_load(premise_id, load_data)
            loads_created += 1
        except (sqlite3.IntegrityError, KeyError, ValueError, TypeError) as e:
            logger.warning("Failed to create load %s: %s", load_config.get("id"), e)

    # Reload premise to pick up new devices and scenarios
    if live_premise:
        # Stop the old premise first (to free the UDP port)
        live_premise.stop()
        # Remove from manager
        del manager.premises[premise_id]
    # Load and start fresh
    manager._load_premise_from_db(premise_id)

    # Mark setup as complete
    manager.db.mark_premise_setup_complete(premise_id)

    return {
        "status": "ok",
        "floors_created": floors_created,
        "rooms_created": rooms_created,
        "devices_deleted": len(existing_devices),
        "devices_created": devices_created,
        "loads_deleted": len(existing_loads),
        "loads_created": loads_created,
        "scenarios_created": scenarios_created,
    }


@router.post("/{premise_id}/mark-setup-complete")
def mark_setup_complete(premise_id: str):
    """Mark premise setup as complete (user chose to start empty).

    Called when user dismisses the welcome modal without loading samples.
    """
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")

    manager.db.mark_premise_setup_complete(premise_id)

    return {"status": "ok"}


@router.post("/{premise_id}/factory-reset")
def factory_reset(premise_id: str):
    """Factory reset — delete everything and show welcome modal again.

    Clears all loads, devices, floors, rooms, and sets setup_complete=false
    so the user sees the welcome modal to choose their setup.
    """
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")

    # Delete all loads first (they reference devices)
    existing_loads = manager.db.list_loads(premise_id)
    for load in existing_loads:
        manager.db.delete_load(load["id"])

    # Delete all devices
    existing_devices = manager.db.list_devices(premise_id)
    for device in existing_devices:
        manager.remove_device(premise_id, device["id"])

    # Delete all floors (cascades to rooms)
    existing_floors = manager.db.list_floors(premise_id)
    for floor in existing_floors:
        manager.db.delete_floor(floor["id"])

    # Reset setup_complete to show welcome modal
    now = __import__("datetime").datetime.now(__import__("datetime").timezone.utc).isoformat()
    manager.db.conn.execute(
        "UPDATE premises SET setup_complete = 0, updated_at = ? WHERE id = ?",
        (now, premise_id),
    )
    manager.db.conn.commit()

    # Reload premise
    live_premise = manager.premises.get(premise_id)
    if live_premise:
        manager._load_premise_from_db(premise_id)

    return {
        "status": "ok",
        "loads_deleted": len(existing_loads),
        "devices_deleted": len(existing_devices),
        "floors_deleted": len(existing_floors),
    }
