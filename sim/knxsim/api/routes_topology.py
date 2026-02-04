"""Topology routes — flat device list for a single KNX TP line.

Each premise represents one physical KNX TP line with its own KNXnet/IP
interface. The topology view shows all devices on that line as a flat list.
"""

from fastapi import APIRouter, HTTPException

router = APIRouter(prefix="/api/v1/premises/{premise_id}", tags=["topology"])


@router.get("/topology")
def get_topology(premise_id: str):
    """Get flat topology: premise area/line info + device list.

    Each premise is a single TP line, so the topology is a flat list
    of devices with the premise's area and line numbers.
    """
    manager = router.app.state.manager
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")
    return manager.db.get_topology(premise_id)


@router.get("/next-device-number")
def get_next_device_number(premise_id: str):
    """Get next available device number on this premise's line (1-255)."""
    manager = router.app.state.manager
    premise = manager.get_premise(premise_id)
    if not premise:
        raise HTTPException(status_code=404, detail="Premise not found")
    next_number = manager.db.get_next_device_number(premise_id)
    if next_number is None:
        raise HTTPException(status_code=409, detail="No available device numbers on this line")
    individual_address = manager.db.compute_individual_address(premise_id, next_number)
    return {
        "premise_id": premise_id,
        "next_device_number": next_number,
        "individual_address": individual_address,
    }
