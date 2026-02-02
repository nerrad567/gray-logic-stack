"""Group Address hierarchy API routes.

Provides CRUD for Main Groups and Middle Groups, plus GA suggestion helpers.
"""

from fastapi import APIRouter, HTTPException, Query
from pydantic import BaseModel

router = APIRouter(prefix="/api/v1", tags=["groups"])


class MainGroupCreate(BaseModel):
    group_number: int
    name: str
    description: str | None = None


class MainGroupUpdate(BaseModel):
    name: str | None = None
    description: str | None = None


class MiddleGroupCreate(BaseModel):
    group_number: int
    name: str
    description: str | None = None
    floor_id: str | None = None


class MiddleGroupUpdate(BaseModel):
    name: str | None = None
    description: str | None = None
    floor_id: str | None = None


# ==============================================================================
# Group Address Tree
# ==============================================================================


@router.get("/premises/{premise_id}/groups")
def get_group_tree(premise_id: str):
    """Get full GA hierarchy tree for a premise."""
    db = router.app.state.manager.db
    premise = db.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")

    tree = db.get_group_address_tree(premise_id)
    tree["used_addresses"] = db.get_used_group_addresses(premise_id)
    return tree


# ==============================================================================
# Main Groups (0-31, typically function-based: Lighting, Blinds, HVAC)
# ==============================================================================


@router.get("/premises/{premise_id}/main-groups")
def list_main_groups(premise_id: str):
    """List all main groups for a premise."""
    db = router.app.state.manager.db
    premise = db.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")

    return db.list_main_groups(premise_id)


@router.post("/premises/{premise_id}/main-groups", status_code=201)
def create_main_group(premise_id: str, data: MainGroupCreate):
    """Create a new main group."""
    db = router.app.state.manager.db
    premise = db.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")

    if not (0 <= data.group_number <= 31):
        raise HTTPException(status_code=400, detail="group_number must be 0-31")

    # Check for duplicate
    existing = db.get_main_group_by_number(premise_id, data.group_number)
    if existing:
        raise HTTPException(status_code=409, detail=f"Main group {data.group_number} already exists")

    return db.create_main_group(premise_id, data.model_dump())


@router.get("/main-groups/{main_group_id}")
def get_main_group(main_group_id: str):
    """Get a single main group."""
    db = router.app.state.manager.db
    group = db.get_main_group(main_group_id)
    if not group:
        raise HTTPException(status_code=404, detail="Main group not found")
    return group


@router.patch("/main-groups/{main_group_id}")
def update_main_group(main_group_id: str, data: MainGroupUpdate):
    """Update a main group."""
    db = router.app.state.manager.db
    group = db.get_main_group(main_group_id)
    if not group:
        raise HTTPException(status_code=404, detail="Main group not found")

    update_data = {k: v for k, v in data.model_dump().items() if v is not None}
    return db.update_main_group(main_group_id, update_data)


@router.delete("/main-groups/{main_group_id}", status_code=204)
def delete_main_group(main_group_id: str):
    """Delete a main group (cascades to middle groups)."""
    db = router.app.state.manager.db
    group = db.get_main_group(main_group_id)
    if not group:
        raise HTTPException(status_code=404, detail="Main group not found")

    db.delete_main_group(main_group_id)
    return None


# ==============================================================================
# Middle Groups (0-7, typically location-based: Ground Floor, First Floor)
# ==============================================================================


@router.get("/main-groups/{main_group_id}/middle-groups")
def list_middle_groups(main_group_id: str):
    """List all middle groups for a main group."""
    db = router.app.state.manager.db
    main_group = db.get_main_group(main_group_id)
    if not main_group:
        raise HTTPException(status_code=404, detail="Main group not found")

    return db.list_middle_groups(main_group_id)


@router.post("/main-groups/{main_group_id}/middle-groups", status_code=201)
def create_middle_group(main_group_id: str, data: MiddleGroupCreate):
    """Create a new middle group."""
    db = router.app.state.manager.db
    main_group = db.get_main_group(main_group_id)
    if not main_group:
        raise HTTPException(status_code=404, detail="Main group not found")

    if not (0 <= data.group_number <= 7):
        raise HTTPException(status_code=400, detail="group_number must be 0-7")

    # Check for duplicate
    existing = db.get_middle_group_by_number(main_group_id, data.group_number)
    if existing:
        raise HTTPException(status_code=409, detail=f"Middle group {data.group_number} already exists")

    return db.create_middle_group(main_group_id, data.model_dump())


@router.get("/middle-groups/{middle_group_id}")
def get_middle_group(middle_group_id: str):
    """Get a single middle group."""
    db = router.app.state.manager.db
    group = db.get_middle_group(middle_group_id)
    if not group:
        raise HTTPException(status_code=404, detail="Middle group not found")
    return group


@router.patch("/middle-groups/{middle_group_id}")
def update_middle_group(middle_group_id: str, data: MiddleGroupUpdate):
    """Update a middle group."""
    db = router.app.state.manager.db
    group = db.get_middle_group(middle_group_id)
    if not group:
        raise HTTPException(status_code=404, detail="Middle group not found")

    update_data = {k: v for k, v in data.model_dump().items() if v is not None}
    return db.update_middle_group(middle_group_id, update_data)


@router.delete("/middle-groups/{middle_group_id}", status_code=204)
def delete_middle_group(middle_group_id: str):
    """Delete a middle group."""
    db = router.app.state.manager.db
    group = db.get_middle_group(middle_group_id)
    if not group:
        raise HTTPException(status_code=404, detail="Middle group not found")

    db.delete_middle_group(middle_group_id)
    return None


# ==============================================================================
# GA Suggestion Helpers
# ==============================================================================


@router.get("/premises/{premise_id}/groups/suggest")
def suggest_group_addresses(
    premise_id: str,
    device_type: str = Query("", description="Device type (light_switch, blind, etc.)"),
    room_id: str | None = Query(None, description="Room ID for floor-based suggestions"),
    main_group: int | None = Query(None, description="Override main group number"),
):
    """Suggest group addresses for a device based on type and room."""
    db = router.app.state.manager.db
    premise = db.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")

    # Determine main group based on device type (or use override)
    if main_group is not None:
        main_group_num = main_group
    else:
        main_group_num = _get_main_group_for_device_type(device_type)

    # Determine middle group based on room → floor
    middle_group_num = 0
    floor_name = None
    if room_id:
        room = db.get_room(room_id)
        if room:
            floor = db.get_floor(room["floor_id"])
            if floor:
                floor_name = floor["name"]
                # Try to find a middle group linked to this floor
                mg = db.get_main_group_by_number(premise_id, main_group_num)
                if mg:
                    middle = db.get_middle_group_by_floor(mg["id"], floor["id"])
                    if middle:
                        middle_group_num = middle["group_number"]
                    else:
                        # Use floor sort_order as middle group if no explicit link
                        middle_group_num = min(floor.get("sort_order", 0), 7)

    # Get suggested GAs based on device type
    suggestions = _get_ga_suggestions_for_device(
        db, premise_id, device_type, main_group_num, middle_group_num
    )

    return {
        "main_group": main_group_num,
        "middle_group": middle_group_num,
        "floor_name": floor_name,
        "suggestions": suggestions,
    }


@router.get("/premises/{premise_id}/groups/next-sub")
def get_next_sub_address(
    premise_id: str,
    main: int = Query(..., description="Main group number"),
    middle: int = Query(..., description="Middle group number"),
):
    """Get the next available sub-group address for a main/middle group."""
    db = router.app.state.manager.db
    premise = db.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")

    next_ga = db.suggest_next_ga(premise_id, main, middle)
    return {"next_address": next_ga}


# ==============================================================================
# Default GA Structure
# ==============================================================================

# Standard 3-Level GA Structure (Industry Standard)
# Main Groups: Function type (what it does)
# Middle Groups: Location (where it is)
# Sub Groups: Specific object

STANDARD_MAIN_GROUPS = [
    {"group_number": 1, "name": "Lighting", "description": "Switch and dimming control"},
    {"group_number": 2, "name": "Blinds", "description": "Position and slat control"},
    {"group_number": 3, "name": "HVAC", "description": "Heating, ventilation, cooling"},
    {"group_number": 4, "name": "Sensors", "description": "Temperature, presence, brightness"},
    {"group_number": 5, "name": "Scenes", "description": "Scene recall and storage"},
    {"group_number": 6, "name": "Audio/Video", "description": "Media and entertainment"},
    {"group_number": 7, "name": "Security", "description": "Alarms and access control"},
]


@router.delete("/premises/{premise_id}/groups/all", status_code=204)
def delete_all_groups(premise_id: str):
    """Delete all main groups (and their middle groups) for a premise.
    
    Use this to reset the GA structure and start fresh.
    Note: This does NOT affect device group_addresses - those remain unchanged.
    """
    db = router.app.state.manager.db
    premise = db.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")

    # Get all main groups and delete them (cascades to middle groups)
    main_groups = db.list_main_groups(premise_id)
    for mg in main_groups:
        db.delete_main_group(mg["id"])

    return None


@router.post("/premises/{premise_id}/groups/create-defaults")
def create_default_structure(
    premise_id: str,
    layout: str = Query("standard", description="Layout template (ignored, standard only)"),
):
    """Create a standard 3-level GA structure.
    
    Structure follows the industry standard:
    - Main Groups (1-7): Function types (Lighting, Blinds, HVAC, etc.)
    - Middle Groups (0-7): Location/floors (auto-created from premise floors)
    - Sub Groups: Individual addresses assigned per device
    
    Main Group 0 is reserved and not created.
    """
    db = router.app.state.manager.db
    premise = db.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")

    floors = db.list_floors(premise_id)
    created_main_count = 0
    created_middle_count = 0

    # Create standard main groups (by function)
    main_groups = []
    for mg_data in STANDARD_MAIN_GROUPS:
        existing = db.get_main_group_by_number(premise_id, mg_data["group_number"])
        if not existing:
            mg = db.create_main_group(premise_id, mg_data)
            main_groups.append(mg)
            created_main_count += 1
        else:
            main_groups.append(existing)

    # Create middle groups from floors (by location)
    for mg in main_groups:
        if floors:
            for i, floor in enumerate(floors):
                if i > 7:  # Max 8 middle groups (0-7)
                    break
                existing = db.get_middle_group_by_number(mg["id"], i)
                if not existing:
                    db.create_middle_group(mg["id"], {
                        "group_number": i,
                        "name": floor["name"],
                        "floor_id": floor["id"],
                    })
                    created_middle_count += 1
        else:
            # No floors exist - create a default "Common" middle group
            existing = db.get_middle_group_by_number(mg["id"], 0)
            if not existing:
                db.create_middle_group(mg["id"], {
                    "group_number": 0,
                    "name": "Common",
                    "description": "Default location group",
                })
                created_middle_count += 1

    return {
        "layout": "standard",
        "layout_name": "Standard 3-Level (Function/Location/Object)",
        "main_groups_created": created_main_count,
        "middle_groups_created": created_middle_count,
        "structure": db.get_group_address_tree(premise_id),
    }


# ==============================================================================
# Helper Functions
# ==============================================================================


def _get_main_group_for_device_type(device_type: str) -> int:
    """Map device type to default main group number.
    
    Uses standard KNX layout (avoiding reserved group 0):
    - 1: Lighting
    - 2: Blinds
    - 3: HVAC
    - 4: Sensors
    - 5: Scenes
    """
    mapping = {
        # Lighting (1)
        "light_switch": 1,
        "light_dimmer": 1,
        # Blinds (2)
        "blind": 2,
        "shutter": 2,
        # HVAC (3)
        "thermostat": 3,
        "hvac": 3,
        "valve": 3,
        # Sensors (4)
        "temperature_sensor": 4,
        "presence_sensor": 4,
        "presence": 4,
        "brightness_sensor": 4,
        "sensor": 4,
        # Wall controls - usually lighting (1)
        "push_button": 1,
        "push_button_2": 1,
        "push_button_4": 1,
        "wall_switch": 1,
        "template_device": 1,
    }
    return mapping.get(device_type, 1)


def _get_ga_suggestions_for_device(
    db,
    premise_id: str,
    device_type: str,
    main_group: int,
    middle_group: int,
) -> dict:
    """Get GA suggestions for a device type.
    
    Returns dict mapping function names to suggested GAs.
    """
    # Templates for different device types
    templates = {
        "light_switch": ["switch", "switch_status"],
        "light_dimmer": ["switch", "switch_status", "brightness", "brightness_status"],
        "blind": ["move", "stop", "position", "position_status", "slat", "slat_status"],
        "shutter": ["move", "stop", "position", "position_status"],
        "presence_sensor": ["presence", "brightness"],
        "presence": ["presence", "brightness"],
        "temperature_sensor": ["temperature"],
        "push_button": ["switch"],
        "push_button_2": ["switch_1", "switch_2"],
        "push_button_4": ["switch_1", "switch_2", "switch_3", "switch_4"],
        "wall_switch": ["switch"],
        "template_device": ["switch"],
        "thermostat": ["setpoint", "temperature", "mode"],
    }

    functions = templates.get(device_type, ["switch"])
    
    # Get used GAs to avoid conflicts
    used = set(db.get_used_group_addresses(premise_id))
    prefix = f"{main_group}/{middle_group}/"
    
    suggestions = {}
    sub = 0
    for func in functions:
        while f"{prefix}{sub}" in used or f"{prefix}{sub}" in suggestions.values():
            sub += 1
            if sub > 255:
                break
        suggestions[func] = f"{prefix}{sub}"
        sub += 1

    return suggestions
