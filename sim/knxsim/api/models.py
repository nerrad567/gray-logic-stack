"""Pydantic models for the KNX Simulator Management API.

Defines request/response schemas for premises, devices, floors, and rooms.
"""

from typing import Any

from pydantic import BaseModel, Field

# ---------------------------------------------------------------------------
# Group Address with metadata
# ---------------------------------------------------------------------------


class GroupAddressInfo(BaseModel):
    """Extended group address info with DPT and flags."""

    ga: str | None = Field(default=None, description="Group address (e.g., '1/2/3')")
    dpt: str = Field(default="1.001", description="Datapoint type")
    flags: str = Field(default="C-W-U-", description="KNX flags (CRWTUI)")
    description: str = Field(default="", description="Human-readable description")


# Type alias: GA can be string (legacy) or object (new)
GroupAddressValue = str | GroupAddressInfo | dict[str, Any]


# ---------------------------------------------------------------------------
# Channels (multi-channel device support)
# ---------------------------------------------------------------------------


class ChannelGroupObject(BaseModel):
    """Group object within a channel."""

    ga: str | None = Field(default=None, description="Assigned group address")
    dpt: str = Field(default="1.001", description="Datapoint type")
    flags: str = Field(default="CW", description="KNX flags")
    description: str = Field(default="", description="Human-readable description")


class Channel(BaseModel):
    """A single channel within a multi-channel device."""

    id: str = Field(..., description="Channel identifier (A, B, C, etc.)")
    name: str = Field(..., description="Channel name (e.g., 'Kitchen Light')")
    group_objects: dict[str, ChannelGroupObject | dict[str, Any]] = Field(
        default_factory=dict, description="Named group objects with GA assignments"
    )
    state: dict[str, Any] = Field(default_factory=dict, description="Current channel state")
    initial_state: dict[str, Any] = Field(default_factory=dict, description="Initial state on startup")
    parameters: dict[str, Any] = Field(default_factory=dict, description="Channel-specific parameters")

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
    setup_complete: bool = False
    running: bool = False
    device_count: int = 0
    created_at: str | None = None
    updated_at: str | None = None


# ---------------------------------------------------------------------------
# Devices
# ---------------------------------------------------------------------------


class DeviceCreate(BaseModel):
    id: str = Field(..., min_length=1, max_length=64)
    type: str = Field(..., min_length=1)
    individual_address: str = Field(..., min_length=3)
    group_addresses: dict[str, GroupAddressValue] = Field(default_factory=dict)
    initial_state: dict[str, Any] = Field(default_factory=dict)
    channels: list[Channel | dict[str, Any]] | None = Field(
        default=None, description="Channel configuration for multi-channel devices"
    )
    room_id: str | None = None


class DeviceUpdate(BaseModel):
    room_id: str | None = None
    individual_address: str | None = None
    group_addresses: dict[str, GroupAddressValue] | None = None
    channels: list[Channel | dict[str, Any]] | None = Field(
        default=None, description="Channel configuration update"
    )


class DevicePlacement(BaseModel):
    room_id: str | None = None


class DeviceCommand(BaseModel):
    """Send a command to a device (triggers GroupWrite on appropriate GA)."""

    command: str = Field(..., min_length=1)  # e.g., "switch", "brightness", "position"
    value: Any = Field(...)  # The value to set (bool, int, float, etc.)


class DeviceResponse(BaseModel):
    id: str
    premise_id: str
    room_id: str | None = None
    type: str
    individual_address: str
    group_addresses: dict[str, GroupAddressValue] = Field(default_factory=dict)
    state: dict[str, Any] = Field(default_factory=dict)
    initial_state: dict[str, Any] = Field(default_factory=dict)
    channels: list[Channel | dict[str, Any]] | None = Field(
        default=None, description="Channel configuration for multi-channel devices"
    )
    line_id: str | None = Field(default=None, description="Topology line reference")
    device_number: int | None = Field(default=None, description="Device number on line")
    created_at: str | None = None
    updated_at: str | None = None


# ---------------------------------------------------------------------------
# Floors
# ---------------------------------------------------------------------------


class FloorCreate(BaseModel):
    id: str = Field(..., min_length=1, max_length=64)
    name: str = Field(..., min_length=1, max_length=128)
    sort_order: int = Field(default=0)


class FloorUpdate(BaseModel):
    name: str | None = None
    sort_order: int | None = None


class RoomCreate(BaseModel):
    id: str = Field(..., min_length=1, max_length=64)
    name: str = Field(..., min_length=1, max_length=128)
    room_type: str = Field(default="other")
    grid_col: int = Field(default=0, ge=0)
    grid_row: int = Field(default=0, ge=0)
    grid_width: int = Field(default=1, ge=1, le=12)
    grid_height: int = Field(default=1, ge=1, le=12)


class RoomUpdate(BaseModel):
    name: str | None = None
    room_type: str | None = None
    grid_col: int | None = Field(default=None, ge=0)
    grid_row: int | None = Field(default=None, ge=0)
    grid_width: int | None = Field(default=None, ge=1, le=12)
    grid_height: int | None = Field(default=None, ge=1, le=12)


class RoomResponse(BaseModel):
    id: str
    floor_id: str
    name: str
    room_type: str = "other"
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
