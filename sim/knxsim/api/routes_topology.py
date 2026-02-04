"""Topology routes — Areas and Lines for KNX bus structure.

The topology view represents the physical KNX bus structure:
  Premise → Area (0-15) → Line (0-15) → Device (0-255)

This is separate from the building view (Floor/Room) which is for
visualization purposes.
"""

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field

router = APIRouter(prefix="/api/v1/premises/{premise_id}", tags=["topology"])


# ---------------------------------------------------------------------------
# Models
# ---------------------------------------------------------------------------


class AreaCreate(BaseModel):
    id: str | None = None
    area_number: int = Field(..., ge=0, le=15)
    name: str = Field(..., min_length=1, max_length=128)
    description: str | None = None


class AreaUpdate(BaseModel):
    name: str | None = None
    description: str | None = None


class AreaResponse(BaseModel):
    id: str
    premise_id: str
    area_number: int
    name: str
    description: str | None = None
    created_at: str | None = None
    updated_at: str | None = None


class LineCreate(BaseModel):
    id: str | None = None
    line_number: int = Field(..., ge=0, le=15)
    name: str = Field(..., min_length=1, max_length=128)
    description: str | None = None


class LineUpdate(BaseModel):
    name: str | None = None
    description: str | None = None


class LineResponse(BaseModel):
    id: str
    area_id: str
    line_number: int
    name: str
    description: str | None = None
    created_at: str | None = None
    updated_at: str | None = None


# ---------------------------------------------------------------------------
# Area Routes
# ---------------------------------------------------------------------------


@router.get("/areas", response_model=list[AreaResponse])
def list_areas(premise_id: str):
    """List all areas in a premise."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    return manager.db.list_areas(premise_id)


@router.get("/areas/{area_id}", response_model=AreaResponse)
def get_area(premise_id: str, area_id: str):
    """Get a specific area."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    area = manager.db.get_area(area_id)
    if not area or area.get("premise_id") != premise_id:
        raise HTTPException(status_code=404, detail="Area not found")
    return area


@router.post("/areas", response_model=AreaResponse, status_code=201)
def create_area(premise_id: str, body: AreaCreate):
    """Create a new area in the premise."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")

    # Check for duplicate area number
    existing = manager.db.get_area_by_number(premise_id, body.area_number)
    if existing:
        raise HTTPException(
            status_code=409,
            detail=f"Area {body.area_number} already exists in this premise",
        )

    return manager.db.create_area(premise_id, body.model_dump())


@router.patch("/areas/{area_id}", response_model=AreaResponse)
def update_area(premise_id: str, area_id: str, body: AreaUpdate):
    """Update an area."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    area = manager.db.get_area(area_id)
    if not area or area.get("premise_id") != premise_id:
        raise HTTPException(status_code=404, detail="Area not found")

    updates = body.model_dump(exclude_none=True)
    if not updates:
        return area
    return manager.db.update_area(area_id, updates)


@router.delete("/areas/{area_id}", status_code=204)
def delete_area(premise_id: str, area_id: str):
    """Delete an area and all its lines."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    area = manager.db.get_area(area_id)
    if not area or area.get("premise_id") != premise_id:
        raise HTTPException(status_code=404, detail="Area not found")
    manager.db.delete_area(area_id)


# ---------------------------------------------------------------------------
# Line Routes
# ---------------------------------------------------------------------------


@router.get("/areas/{area_id}/lines", response_model=list[LineResponse])
def list_lines(premise_id: str, area_id: str):
    """List all lines in an area."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    area = manager.db.get_area(area_id)
    if not area or area.get("premise_id") != premise_id:
        raise HTTPException(status_code=404, detail="Area not found")
    return manager.db.list_lines(area_id)


@router.get("/lines", response_model=list[LineResponse])
def list_all_lines(premise_id: str):
    """List all lines in a premise (across all areas)."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    return manager.db.list_lines_by_premise(premise_id)


@router.get("/areas/{area_id}/lines/{line_id}", response_model=LineResponse)
def get_line(premise_id: str, area_id: str, line_id: str):
    """Get a specific line."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    line = manager.db.get_line(line_id)
    if not line or line.get("area_id") != area_id:
        raise HTTPException(status_code=404, detail="Line not found")
    return line


@router.post("/areas/{area_id}/lines", response_model=LineResponse, status_code=201)
def create_line(premise_id: str, area_id: str, body: LineCreate):
    """Create a new line in an area."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    area = manager.db.get_area(area_id)
    if not area or area.get("premise_id") != premise_id:
        raise HTTPException(status_code=404, detail="Area not found")

    # Check for duplicate line number
    existing = manager.db.get_line_by_number(area_id, body.line_number)
    if existing:
        raise HTTPException(
            status_code=409,
            detail=f"Line {body.line_number} already exists in this area",
        )

    return manager.db.create_line(area_id, body.model_dump())


@router.patch("/areas/{area_id}/lines/{line_id}", response_model=LineResponse)
def update_line(premise_id: str, area_id: str, line_id: str, body: LineUpdate):
    """Update a line."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    line = manager.db.get_line(line_id)
    if not line or line.get("area_id") != area_id:
        raise HTTPException(status_code=404, detail="Line not found")

    updates = body.model_dump(exclude_none=True)
    if not updates:
        return line
    return manager.db.update_line(line_id, updates)


@router.delete("/areas/{area_id}/lines/{line_id}", status_code=204)
def delete_line(premise_id: str, area_id: str, line_id: str):
    """Delete a line (devices become unlinked from topology)."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    line = manager.db.get_line(line_id)
    if not line or line.get("area_id") != area_id:
        raise HTTPException(status_code=404, detail="Line not found")
    manager.db.delete_line(line_id)


# ---------------------------------------------------------------------------
# Topology View
# ---------------------------------------------------------------------------


@router.get("/topology")
def get_topology(premise_id: str):
    """Get full topology tree: Areas → Lines → Devices.

    Returns the physical KNX bus structure with all devices organized
    by their individual addresses.
    """
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    return manager.db.get_topology(premise_id)


@router.get("/lines/{line_id}/next-device-number")
def get_next_device_number(premise_id: str, line_id: str):
    """Get next available device number on a line (1-255)."""
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    line = manager.db.get_line_with_area(line_id)
    if not line or line.get("premise_id") != premise_id:
        raise HTTPException(status_code=404, detail="Line not found")
    next_number = manager.db.get_next_device_number(line_id)
    if next_number is None:
        raise HTTPException(status_code=409, detail="No available device numbers on this line")
    individual_address = manager.db.compute_individual_address(line_id, next_number)
    return {
        "line_id": line_id,
        "next_device_number": next_number,
        "individual_address": individual_address,
    }
