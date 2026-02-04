"""Unit tests for Database CRUD operations."""

from __future__ import annotations

import sqlite3

import pytest


def _create_premise(db, premise_id: str = "premise-1") -> dict:
    return db.create_premise(
        {
            "id": premise_id,
            "name": "Test Premise",
            "area_number": 1,
            "line_number": 1,
            "port": 3671,
        }
    )


def test_schema_created(db):
    row = db.conn.execute(
        "SELECT name FROM sqlite_master WHERE type='table' AND name='premises'"
    ).fetchone()
    assert row is not None


def test_premise_device_floor_room_load_crud(db):
    premise = _create_premise(db)
    premise_id = premise["id"]

    floor = db.create_floor(premise_id, {"id": "floor-1", "name": "Ground Floor", "sort_order": 0})
    room = db.create_room(
        floor["id"], {"id": "room-1", "name": "Living Room", "room_type": "living"}
    )

    device = db.create_device(
        premise_id,
        {
            "id": "device-1",
            "type": "light_switch",
            "individual_address": "1.1.10",
            "group_addresses": {"switch_cmd": {"ga": "0/0/1", "dpt": "1.001"}},
            "initial_state": {"on": False},
            "room_id": room["id"],
        },
    )
    assert db.get_device(device["id"]) is not None

    load = db.create_load(
        premise_id,
        {
            "name": "Test Load",
            "type": "light",
            "room_id": room["id"],
            "actuator_device_id": device["id"],
        },
    )
    assert db.get_load(load["id"]) is not None

    updated_room = db.update_room(room["id"], {"name": "Living Room Updated"})
    assert updated_room["name"] == "Living Room Updated"

    updated_device = db.update_device(device["id"], {"room_id": "room-1"})
    assert updated_device["room_id"] == "room-1"

    assert db.delete_load(load["id"]) is True
    assert db.get_load(load["id"]) is None


def test_topology_and_groups_crud(db):
    premise = _create_premise(db, premise_id="premise-2")
    premise_id = premise["id"]

    # Topology is now derived from premise area/line numbers
    topo = db.get_topology(premise_id)
    assert topo["area_number"] == 1
    assert topo["line_number"] == 1
    assert topo["gateway"] == "1.1.0"
    assert isinstance(topo["devices"], list)

    main_group = db.create_main_group(premise_id, {"group_number": 1, "name": "Lighting"})
    middle_group = db.create_middle_group(
        main_group["id"], {"group_number": 1, "name": "Ground Floor"}
    )

    tree = db.get_group_address_tree(premise_id)
    assert len(tree["main_groups"]) == 1
    assert tree["main_groups"][0]["id"] == main_group["id"]
    assert tree["main_groups"][0]["middle_groups"][0]["id"] == middle_group["id"]


def test_delete_premise_cascades(db):
    premise = _create_premise(db, premise_id="premise-3")
    premise_id = premise["id"]

    floor = db.create_floor(premise_id, {"id": "floor-3", "name": "Floor", "sort_order": 0})
    room = db.create_room(floor["id"], {"id": "room-3", "name": "Room", "room_type": "other"})
    device = db.create_device(
        premise_id,
        {
            "id": "device-3",
            "type": "light_switch",
            "individual_address": "1.1.11",
            "group_addresses": {"switch_cmd": {"ga": "0/0/2", "dpt": "1.001"}},
            "initial_state": {"on": False},
        },
    )

    assert db.delete_premise(premise_id) is True
    assert db.get_premise(premise_id) is None
    assert db.get_floor(floor["id"]) is None
    assert db.get_room(room["id"]) is None
    assert db.get_device(device["id"]) is None


def test_duplicate_premise_id(db):
    _create_premise(db, premise_id="premise-dup")
    with pytest.raises(sqlite3.IntegrityError):
        _create_premise(db, premise_id="premise-dup")
