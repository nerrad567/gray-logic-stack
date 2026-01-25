"""Floor and Room CRUD routes for the KNX Simulator Management API.

Floors and rooms define the building layout for the schematic view.
Rooms use CSS Grid positioning (col, row, width, height) for the UI.
"""

from fastapi import APIRouter, HTTPException

from .models import (
    FloorCreate,
    FloorResponse,
    FloorUpdate,
    RoomCreate,
    RoomResponse,
    RoomUpdate,
)

router = APIRouter(prefix="/api/v1/premises/{premise_id}/floors", tags=["floors"])


# ---------------------------------------------------------------------------
# Floors
# ---------------------------------------------------------------------------


@router.get("", response_model=list[FloorResponse])
def list_floors(premise_id: str):
    """List all floors for a premise (with rooms attached)."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    return manager.list_floors(premise_id)


@router.post("", response_model=FloorResponse, status_code=201)
def create_floor(premise_id: str, body: FloorCreate):
    """Create a new floor in a premise."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    result = manager.create_floor(premise_id, body.model_dump())
    result["rooms"] = []
    return result


@router.patch("/{floor_id}", response_model=FloorResponse)
def update_floor(premise_id: str, floor_id: str, body: FloorUpdate):
    """Update a floor's name or sort order."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    updates = body.model_dump(exclude_none=True)
    if not updates:
        floor = manager.db.get_floor(floor_id)
        if not floor:
            raise HTTPException(status_code=404, detail="Floor not found")
        floor["rooms"] = manager.db.list_rooms(floor_id)
        return floor
    result = manager.update_floor(floor_id, updates)
    if not result:
        raise HTTPException(status_code=404, detail="Floor not found")
    result["rooms"] = manager.db.list_rooms(floor_id)
    return result


@router.delete("/{floor_id}", status_code=204)
def delete_floor(premise_id: str, floor_id: str):
    """Delete a floor and all its rooms."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    if not manager.delete_floor(floor_id):
        raise HTTPException(status_code=404, detail="Floor not found")


# ---------------------------------------------------------------------------
# Rooms (nested under floors)
# ---------------------------------------------------------------------------


@router.get("/{floor_id}/rooms", response_model=list[RoomResponse])
def list_rooms(premise_id: str, floor_id: str):
    """List all rooms on a floor."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    return manager.db.list_rooms(floor_id)


@router.post("/{floor_id}/rooms", response_model=RoomResponse, status_code=201)
def create_room(premise_id: str, floor_id: str, body: RoomCreate):
    """Create a room on a floor with grid positioning."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    if not manager.db.get_floor(floor_id):
        raise HTTPException(status_code=404, detail="Floor not found")
    return manager.create_room(floor_id, body.model_dump())


@router.patch("/{floor_id}/rooms/{room_id}", response_model=RoomResponse)
def update_room(premise_id: str, floor_id: str, room_id: str, body: RoomUpdate):
    """Update a room's name or grid positioning."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    updates = body.model_dump(exclude_none=True)
    if not updates:
        room = manager.db.get_room(room_id)
        if not room:
            raise HTTPException(status_code=404, detail="Room not found")
        return room
    result = manager.update_room(room_id, updates)
    if not result:
        raise HTTPException(status_code=404, detail="Room not found")
    return result


@router.delete("/{floor_id}/rooms/{room_id}", status_code=204)
def delete_room(premise_id: str, floor_id: str, room_id: str):
    """Delete a room (devices in it get room_id set to NULL)."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    if not manager.delete_room(room_id):
        raise HTTPException(status_code=404, detail="Room not found")
