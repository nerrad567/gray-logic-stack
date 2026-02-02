"""Load CRUD routes for the KNX Simulator Management API.

Loads represent physical equipment (lights, motors, valves, etc.) that are
controlled by KNX actuator channels. They are NOT KNX devices themselves -
they have no individual address and no bus presence.

Their state is derived from the linked actuator channel.
"""

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

router = APIRouter(prefix="/api/v1/premises/{premise_id}/loads", tags=["loads"])


class LoadCreate(BaseModel):
    """Request body for creating a load."""
    id: str | None = None
    name: str
    type: str  # light, motor, valve, fan, heater, speaker, etc.
    room_id: str | None = None
    icon: str | None = None
    actuator_device_id: str | None = None
    actuator_channel_id: str | None = None


class LoadUpdate(BaseModel):
    """Request body for updating a load."""
    name: str | None = None
    type: str | None = None
    room_id: str | None = None
    icon: str | None = None
    actuator_device_id: str | None = None
    actuator_channel_id: str | None = None


class LoadResponse(BaseModel):
    """Response for a single load."""
    id: str
    premise_id: str
    room_id: str | None
    name: str
    type: str
    icon: str | None
    actuator_device_id: str | None
    actuator_channel_id: str | None
    actuator_type: str | None = None
    actuator_address: str | None = None
    created_at: str
    updated_at: str


class LoadWithState(LoadResponse):
    """Load response including current state from actuator."""
    state: dict | None = None


@router.get("", response_model=list[LoadResponse])
def list_loads(premise_id: str):
    """List all loads in a premise."""
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")
    
    return manager.db.list_loads(premise_id)


@router.get("/orphaned", response_model=list[LoadResponse])
def list_orphaned_loads(premise_id: str):
    """List loads whose actuator has been deleted or is missing."""
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")
    
    return manager.db.list_orphaned_loads(premise_id)


@router.get("/{load_id}", response_model=LoadWithState)
def get_load(premise_id: str, load_id: str):
    """Get a single load by ID, including its current state."""
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")
    
    load = manager.db.get_load(load_id)
    if not load:
        raise HTTPException(status_code=404, detail="Load not found")
    
    # Get state from linked actuator channel
    state = manager.db.get_load_state(load_id)
    load["state"] = state
    
    return load


@router.post("", response_model=LoadResponse, status_code=201)
def create_load(premise_id: str, body: LoadCreate):
    """Create a new load (physical equipment)."""
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")
    
    # Validate room exists if provided
    if body.room_id:
        room = manager.db.get_room(body.room_id)
        if not room:
            raise HTTPException(status_code=400, detail="Room not found")
    
    # Validate actuator and channel if provided
    if body.actuator_device_id:
        device = manager.db.get_device(body.actuator_device_id)
        if not device:
            raise HTTPException(status_code=400, detail="Actuator device not found")
        
        # Validate channel exists
        if body.actuator_channel_id:
            channels = device.get("channels", [])
            channel_ids = [c.get("id") for c in channels]
            if body.actuator_channel_id not in channel_ids:
                raise HTTPException(
                    status_code=400,
                    detail=f"Channel {body.actuator_channel_id} not found in device. "
                           f"Available: {channel_ids}"
                )
    
    return manager.db.create_load(premise_id, body.model_dump())


@router.patch("/{load_id}", response_model=LoadResponse)
def update_load(premise_id: str, load_id: str, body: LoadUpdate):
    """Update a load's properties."""
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")
    
    load = manager.db.get_load(load_id)
    if not load:
        raise HTTPException(status_code=404, detail="Load not found")
    
    update_data = body.model_dump(exclude_unset=True)
    
    # Validate room if changing
    if "room_id" in update_data and update_data["room_id"]:
        room = manager.db.get_room(update_data["room_id"])
        if not room:
            raise HTTPException(status_code=400, detail="Room not found")
    
    # Validate actuator and channel if changing
    if "actuator_device_id" in update_data and update_data["actuator_device_id"]:
        device = manager.db.get_device(update_data["actuator_device_id"])
        if not device:
            raise HTTPException(status_code=400, detail="Actuator device not found")
        
        channel_id = update_data.get("actuator_channel_id") or load.get("actuator_channel_id")
        if channel_id:
            channels = device.get("channels", [])
            channel_ids = [c.get("id") for c in channels]
            if channel_id not in channel_ids:
                raise HTTPException(
                    status_code=400,
                    detail=f"Channel {channel_id} not found in device"
                )
    
    return manager.db.update_load(load_id, update_data)


@router.delete("/{load_id}", status_code=204)
def delete_load(premise_id: str, load_id: str):
    """Delete a load."""
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")
    
    if not manager.db.delete_load(load_id):
        raise HTTPException(status_code=404, detail="Load not found")


# Additional endpoints for room-based queries

@router.get("/by-room/{room_id}", response_model=list[LoadWithState])
def list_loads_by_room(premise_id: str, room_id: str):
    """List all loads in a specific room with their states."""
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")
    
    loads = manager.db.list_loads_by_room(room_id)
    
    # Add state to each load
    for load in loads:
        load["state"] = manager.db.get_load_state(load["id"])
    
    return loads


@router.get("/by-actuator/{device_id}/{channel_id}", response_model=list[LoadResponse])
def list_loads_by_actuator_channel(premise_id: str, device_id: str, channel_id: str):
    """List all loads connected to a specific actuator channel."""
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")
    
    return manager.db.list_loads_by_actuator_channel(device_id, channel_id)
