"""Template browsing and from-template device creation routes."""

from typing import Any

from fastapi import APIRouter, HTTPException, Query
from pydantic import BaseModel, Field, model_validator

from persistence.db import ConflictError

router = APIRouter(tags=["templates"])


# ---------------------------------------------------------------------------
# Models
# ---------------------------------------------------------------------------


class FromTemplateRequest(BaseModel):
    template_id: str = Field(..., description="Template type to instantiate")
    device_id: str = Field(..., min_length=1, max_length=64, description="Unique device ID")
    individual_address: str | None = Field(
        default=None,
        min_length=3,
        description="KNX individual address (e.g., 1.1.10)",
    )
    device_number: int | None = Field(
        default=None, ge=1, le=255, description="Device number on line"
    )
    group_addresses: dict[str, str] = Field(
        ..., description="Mapping of template GA slots to actual group addresses"
    )
    room_id: str | None = Field(default=None, description="Room to place device in")

    @model_validator(mode="after")
    def _validate_addressing(self):
        if not self.individual_address and self.device_number is None:
            raise ValueError("Provide individual_address or device_number")
        return self


def _merge_ga_with_template(
    user_gas: dict[str, str], template_gas: dict[str, Any]
) -> dict[str, dict[str, Any]]:
    """Merge user-provided GA strings with template metadata (DPT, flags, etc).

    Args:
        user_gas: User-provided mapping of slot names to GA strings
        template_gas: Template definition with DPT, flags, direction, description

    Returns:
        Merged dict with full GA info objects
    """
    result = {}
    for slot_name, ga_string in user_gas.items():
        template_info = template_gas.get(slot_name, {})
        result[slot_name] = {
            "ga": ga_string,
            "dpt": template_info.get("dpt", "1.001"),
            "flags": template_info.get("flags", "C-W-U-"),
            "description": template_info.get("description", ""),
        }
    return result


# ---------------------------------------------------------------------------
# Template browsing
# ---------------------------------------------------------------------------


@router.get("/api/v1/templates")
def list_templates(domain: str | None = Query(default=None)):
    """List all available device templates, optionally filtered by domain."""
    loader = router.app.state.template_loader
    return {
        "templates": loader.list_templates(domain=domain),
        "count": len(loader.list_templates(domain=domain)),
    }


@router.get("/api/v1/templates/domains")
def list_domains():
    """List all available template domains."""
    loader = router.app.state.template_loader
    return {"domains": loader.list_domains()}


@router.get("/api/v1/templates/{template_id}")
def get_template(template_id: str):
    """Get a single template by ID with full details."""
    loader = router.app.state.template_loader
    template = loader.get_template(template_id)
    if not template:
        raise HTTPException(status_code=404, detail="Template not found")
    return template.to_dict()


# ---------------------------------------------------------------------------
# Create device from template
# ---------------------------------------------------------------------------


@router.post("/api/v1/premises/{premise_id}/devices/from-template", status_code=201)
def create_device_from_template(premise_id: str, body: FromTemplateRequest):
    """Create a device from a template with assigned group addresses.

    The template defines the DPT types, initial state, and optional scenarios.
    The caller provides the actual group addresses and placement.
    """
    manager = router.app.state.manager
    loader = router.app.state.template_loader

    # Validate premise exists
    if not manager.get_premise(premise_id):
        raise HTTPException(status_code=404, detail="Premise not found")

    # Validate template exists
    template = loader.get_template(body.template_id)
    if not template:
        raise HTTPException(status_code=404, detail=f"Template not found: {body.template_id}")

    # Check for duplicate device ID
    if manager.get_device(body.device_id):
        raise HTTPException(status_code=409, detail="Device ID already exists")

    # Validate that required GA slots are provided
    required_slots = template.get_required_gas()
    provided_slots = set(body.group_addresses.keys())
    missing = set(required_slots) - provided_slots
    if missing:
        raise HTTPException(
            status_code=422,
            detail=f"Missing required group addresses: {sorted(missing)}",
        )

    # Merge user-provided GAs with template metadata (DPT, flags, etc)
    merged_gas = _merge_ga_with_template(body.group_addresses, template.group_addresses)

    # Create device using the template_device type
    config = {
        "template_id": template.id,
        "template_def": template.group_addresses,
    }

    # Inject manufacturer metadata for .knxproj export
    if template.manufacturer_name:
        config["manufacturer_id"] = template.manufacturer_id
        config["manufacturer_name"] = template.manufacturer_name
        config["product_model"] = template.product_model
        config["application_program"] = template.application_program
        config["hardware_type"] = template.hardware_type

    device_data = {
        "id": body.device_id,
        "type": "template_device",
        "individual_address": body.individual_address,
        "device_number": body.device_number,
        "group_addresses": merged_gas,
        "initial_state": dict(template.initial_state),
        "room_id": body.room_id,
        "config": config,
    }

    try:
        result = manager.add_device(premise_id, device_data)
    except ConflictError as exc:
        raise HTTPException(status_code=409, detail=str(exc)) from exc
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    if not result:
        raise HTTPException(status_code=500, detail="Failed to create device")

    # Add template-defined scenarios
    if template.scenarios:
        for i, sc in enumerate(template.scenarios):
            manager.db.create_scenario(
                premise_id,
                {
                    "id": f"{body.device_id}-scenario-{i}",
                    "device_id": body.device_id,
                    "field": sc["field"],
                    "type": sc["type"],
                    "params": sc.get("params", {}),
                    "enabled": True,
                },
            )

    result["template_id"] = template.id
    return result
