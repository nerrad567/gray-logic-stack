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
