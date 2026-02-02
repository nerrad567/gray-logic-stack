"""Premise CRUD routes for the KNX Simulator Management API."""

from fastapi import APIRouter, HTTPException

from .models import PremiseCreate, PremiseResponse

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
    import yaml
    
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")
    
    # Load config
    try:
        with open("/app/config.yaml") as f:
            config = yaml.safe_load(f)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to load config: {e}")
    
    if "devices" not in config:
        raise HTTPException(status_code=500, detail="No devices found in config.yaml")
    
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
    # Actuators go to Distribution Board (they're DIN-rail mounted)
    DEVICE_ROOM_MAP = {
        # Room-specific devices (sensors, controls)
        "living": "living-room",
        "kitchen": "kitchen",
        "hallway": "hallway",
        "bedroom": "bedroom",
        "bathroom": "bathroom",
        "dining": "living-room",  # Dining is often part of living area
        # Centralized equipment (actuators in distribution board)
        "actuator": "distribution-board",
        "binary-input": "distribution-board",
    }
    
    floors_created = 0
    rooms_created = 0
    
    for floor_id, floor_data in SAMPLE_TOPOLOGY.items():
        manager.db.create_floor(premise_id, {
            "id": floor_id,
            "name": floor_data["name"],
            "sort_order": floor_data["sort_order"],
        })
        floors_created += 1
        
        for room_data in floor_data["rooms"]:
            manager.db.create_room(floor_id, room_data)
            rooms_created += 1
    
    # ─────────────────────────────────────────────────────────────
    # Create sample devices
    # ─────────────────────────────────────────────────────────────
    
    def guess_room_for_device(device_id: str, group_addresses: dict) -> str | None:
        """Guess which room a device belongs to based on its ID or GA names."""
        # Check device ID
        device_id_lower = device_id.lower()
        for pattern, room_id in DEVICE_ROOM_MAP.items():
            if pattern in device_id_lower:
                return room_id
        
        # Check GA names (e.g., "channel_a_switch" might have comment "Living Room")
        ga_str = str(group_addresses).lower()
        for pattern, room_id in DEVICE_ROOM_MAP.items():
            if pattern in ga_str:
                return room_id
        
        # Default: unassigned
        return None
    
    devices_created = 0
    for dev_config in config.get("devices", []):
        device_data = {
            "id": dev_config["id"],
            "type": dev_config["type"],
            "individual_address": dev_config["individual_address"],
            "group_addresses": dev_config.get("group_addresses", {}),
            "initial_state": dev_config.get("initial", {}),
            "room_id": guess_room_for_device(
                dev_config["id"],
                dev_config.get("group_addresses", {}),
            ),
        }
        
        # Handle template devices
        if dev_config.get("template"):
            device_data["config"] = {"template": dev_config["template"]}
        
        try:
            manager.add_device(premise_id, device_data)
            devices_created += 1
        except Exception as e:
            print(f"Warning: Failed to create device {dev_config['id']}: {e}")
    
    # ─────────────────────────────────────────────────────────────
    # Reload scenarios
    # ─────────────────────────────────────────────────────────────
    
    scenarios_created = 0
    live_premise = manager.premises.get(premise_id)
    if live_premise and live_premise.scenario_runner:
        live_premise.scenario_runner.stop()
        live_premise.scenario_runner = None
    
    for scenario_config in config.get("scenarios", []):
        try:
            manager.db.create_scenario(premise_id, {
                "id": f"scenario-{scenario_config['device_id']}-{scenario_config['field']}",
                "device_id": scenario_config["device_id"],
                "field": scenario_config["field"],
                "type": scenario_config["type"],
                "params": scenario_config.get("params", {}),
            })
            scenarios_created += 1
        except Exception:
            pass  # May already exist
    
    # Reload premise to pick up new devices
    if live_premise:
        manager._load_premise_from_db(premise_id)
    
    # Mark setup as complete
    manager.db.mark_premise_setup_complete(premise_id)
    
    return {
        "status": "ok",
        "floors_created": floors_created,
        "rooms_created": rooms_created,
        "devices_deleted": len(existing_devices),
        "devices_created": devices_created,
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
    
    Clears all devices, floors, rooms, and sets setup_complete=false
    so the user sees the welcome modal to choose their setup.
    """
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")
    
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
        "devices_deleted": len(existing_devices),
        "floors_deleted": len(existing_floors),
    }
