"""Pydantic models for the KNX Simulator Management API.

Defines request/response schemas for premises, devices, floors, and rooms.
"""

from typing import Any, Optional

from pydantic import BaseModel, Field

# ---------------------------------------------------------------------------
# Premises
# ---------------------------------------------------------------------------


class PremiseCreate(BaseModel):
    id: str = Field(..., min_length=1, max_length=64)
    name: str = Field(..., min_length=1, max_length=128)
    gateway_address: str = Field(default="1.0.0")
    client_address: str = Field(default="1.0.255")
    port: int = Field(..., ge=1024, le=65535)


class PremiseResponse(BaseModel):
    id: str
    name: str
    gateway_address: str
    client_address: str
    port: int
    running: bool = False
    device_count: int = 0
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


# ---------------------------------------------------------------------------
# Devices
# ---------------------------------------------------------------------------


class DeviceCreate(BaseModel):
    id: str = Field(..., min_length=1, max_length=64)
    type: str = Field(..., min_length=1)
    individual_address: str = Field(..., min_length=3)
    group_addresses: dict[str, str] = Field(default_factory=dict)
    initial_state: dict[str, Any] = Field(default_factory=dict)
    room_id: Optional[str] = None


class DeviceUpdate(BaseModel):
    room_id: Optional[str] = None
    individual_address: Optional[str] = None
    group_addresses: Optional[dict[str, str]] = None


class DevicePlacement(BaseModel):
    room_id: Optional[str] = None


class DeviceCommand(BaseModel):
    """Send a command to a device (triggers GroupWrite on appropriate GA)."""

    command: str = Field(..., min_length=1)  # e.g., "switch", "brightness", "position"
    value: Any = Field(...)  # The value to set (bool, int, float, etc.)


class DeviceResponse(BaseModel):
    id: str
    premise_id: str
    room_id: Optional[str] = None
    type: str
    individual_address: str
    group_addresses: dict[str, str] = Field(default_factory=dict)
    state: dict[str, Any] = Field(default_factory=dict)
    initial_state: dict[str, Any] = Field(default_factory=dict)
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


# ---------------------------------------------------------------------------
# Floors
# ---------------------------------------------------------------------------


class FloorCreate(BaseModel):
    id: str = Field(..., min_length=1, max_length=64)
    name: str = Field(..., min_length=1, max_length=128)
    sort_order: int = Field(default=0)


class FloorUpdate(BaseModel):
    name: Optional[str] = None
    sort_order: Optional[int] = None


class RoomCreate(BaseModel):
    id: str = Field(..., min_length=1, max_length=64)
    name: str = Field(..., min_length=1, max_length=128)
    grid_col: int = Field(default=0, ge=0)
    grid_row: int = Field(default=0, ge=0)
    grid_width: int = Field(default=1, ge=1, le=12)
    grid_height: int = Field(default=1, ge=1, le=12)


class RoomUpdate(BaseModel):
    name: Optional[str] = None
    grid_col: Optional[int] = Field(default=None, ge=0)
    grid_row: Optional[int] = Field(default=None, ge=0)
    grid_width: Optional[int] = Field(default=None, ge=1, le=12)
    grid_height: Optional[int] = Field(default=None, ge=1, le=12)


class RoomResponse(BaseModel):
    id: str
    floor_id: str
    name: str
    grid_col: int = 0
    grid_row: int = 0
    grid_width: int = 1
    grid_height: int = 1


class FloorResponse(BaseModel):
    id: str
    premise_id: str
    name: str
    sort_order: int = 0
    rooms: list[RoomResponse] = Field(default_factory=list)
