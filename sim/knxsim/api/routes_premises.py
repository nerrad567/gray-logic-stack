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
    """Reset premise to sample devices from config.yaml.
    
    This will:
    1. Delete all existing devices in the premise
    2. Reload sample devices from the config file
    
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
    
    # Add sample devices from config
    devices_created = 0
    for dev_config in config.get("devices", []):
        device_data = {
            "id": dev_config["id"],
            "type": dev_config["type"],
            "individual_address": dev_config["individual_address"],
            "group_addresses": dev_config.get("group_addresses", {}),
            "initial_state": dev_config.get("initial", {}),
        }
        
        # Handle template devices
        if dev_config.get("template"):
            device_data["config"] = {"template": dev_config["template"]}
        
        try:
            manager.add_device(premise_id, device_data)
            devices_created += 1
        except Exception as e:
            # Log but continue
            print(f"Warning: Failed to create device {dev_config['id']}: {e}")
    
    # Reload scenarios
    scenarios_created = 0
    live_premise = manager.premises.get(premise_id)
    if live_premise and live_premise.scenario_runner:
        live_premise.scenario_runner.stop()
        live_premise.scenario_runner = None
    
    # Re-add scenarios from config
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
    
    # Restart scenarios
    # Note: ScenarioRunner is typically started by premise_manager
    # For now, just let the existing runner continue or reload the premise
    if live_premise:
        # Re-fetch devices and rebuild GA map
        manager._load_premise_from_db(premise_id)
    
    return {
        "status": "ok",
        "devices_deleted": len(existing_devices),
        "devices_created": devices_created,
        "scenarios_created": scenarios_created,
    }
