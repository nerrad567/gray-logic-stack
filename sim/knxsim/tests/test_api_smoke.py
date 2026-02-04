"""Happy-path smoke tests for KNXSim API routes."""

from __future__ import annotations

import uuid

import pytest

pytestmark = pytest.mark.anyio


def _premise_payload(port: int) -> dict:
    premise_id = f"premise-{uuid.uuid4().hex[:8]}"
    return {
        "id": premise_id,
        "name": "Test Premise",
        "area_number": 1,
        "line_number": 1,
        "port": port,
    }


async def _create_premise(client, port: int) -> str:
    payload = _premise_payload(port)
    resp = await client.post("/api/v1/premises", json=payload)
    assert resp.status_code == 201
    return resp.json()["id"]


async def test_premises_crud(client, free_udp_port):
    premise_id = await _create_premise(client, free_udp_port)
    try:
        resp = await client.get("/api/v1/premises")
        assert resp.status_code == 200
        assert any(p["id"] == premise_id for p in resp.json())

        resp = await client.get(f"/api/v1/premises/{premise_id}")
        assert resp.status_code == 200
        assert resp.json()["id"] == premise_id
    finally:
        resp = await client.delete(f"/api/v1/premises/{premise_id}")
        assert resp.status_code == 204


async def test_devices_crud(client, free_udp_port):
    premise_id = await _create_premise(client, free_udp_port)
    try:
        # Create a floor and room first (needed for device room assignment)
        floor_payload = {"id": "floor-1", "name": "Ground Floor", "sort_order": 0}
        resp = await client.post(
            f"/api/v1/premises/{premise_id}/floors", json=floor_payload
        )
        assert resp.status_code == 201
        floor_id = resp.json()["id"]

        room_payload = {"id": "room-1", "name": "Living Room", "room_type": "living"}
        resp = await client.post(
            f"/api/v1/premises/{premise_id}/floors/{floor_id}/rooms", json=room_payload
        )
        assert resp.status_code == 201

        device_payload = {
            "id": "device-1",
            "type": "light_switch",
            "individual_address": "1.1.10",
            "group_addresses": {
                "switch_cmd": {"ga": "0/0/1", "dpt": "1.001"},
                "switch_status": {"ga": "0/0/2", "dpt": "1.001"},
            },
            "initial_state": {"on": False},
        }
        resp = await client.post(
            f"/api/v1/premises/{premise_id}/devices", json=device_payload
        )
        assert resp.status_code == 201
        device_id = resp.json()["id"]

        resp = await client.get(f"/api/v1/premises/{premise_id}/devices")
        assert resp.status_code == 200
        assert any(d["id"] == device_id for d in resp.json())

        resp = await client.get(f"/api/v1/premises/{premise_id}/devices/{device_id}")
        assert resp.status_code == 200
        assert resp.json()["id"] == device_id

        resp = await client.patch(
            f"/api/v1/premises/{premise_id}/devices/{device_id}",
            json={"room_id": "room-1"},
        )
        assert resp.status_code == 200
        assert resp.json()["room_id"] == "room-1"

        resp = await client.delete(
            f"/api/v1/premises/{premise_id}/devices/{device_id}"
        )
        assert resp.status_code == 204
    finally:
        await client.delete(f"/api/v1/premises/{premise_id}")


async def test_floors_and_rooms_smoke(client, free_udp_port):
    premise_id = await _create_premise(client, free_udp_port)
    try:
        floor_payload = {"id": "floor-1", "name": "Ground Floor", "sort_order": 0}
        resp = await client.post(
            f"/api/v1/premises/{premise_id}/floors", json=floor_payload
        )
        assert resp.status_code == 201
        floor_id = resp.json()["id"]

        resp = await client.get(f"/api/v1/premises/{premise_id}/floors")
        assert resp.status_code == 200
        assert any(f["id"] == floor_id for f in resp.json())

        room_payload = {
            "id": "room-1",
            "name": "Living Room",
            "room_type": "living",
        }
        resp = await client.post(
            f"/api/v1/premises/{premise_id}/floors/{floor_id}/rooms",
            json=room_payload,
        )
        assert resp.status_code == 201

        resp = await client.get(
            f"/api/v1/premises/{premise_id}/floors/{floor_id}/rooms"
        )
        assert resp.status_code == 200
        assert any(r["id"] == "room-1" for r in resp.json())
    finally:
        await client.delete(f"/api/v1/premises/{premise_id}")


async def test_loads_smoke(client, free_udp_port):
    premise_id = await _create_premise(client, free_udp_port)
    try:
        load_payload = {"name": "Test Load", "type": "light"}
        resp = await client.post(
            f"/api/v1/premises/{premise_id}/loads", json=load_payload
        )
        assert resp.status_code == 201
        load_id = resp.json()["id"]

        resp = await client.get(f"/api/v1/premises/{premise_id}/loads")
        assert resp.status_code == 200
        assert any(l["id"] == load_id for l in resp.json())

        resp = await client.get(f"/api/v1/premises/{premise_id}/loads/{load_id}")
        assert resp.status_code == 200
        assert resp.json()["id"] == load_id
    finally:
        await client.delete(f"/api/v1/premises/{premise_id}")


async def test_topology_smoke(client, free_udp_port):
    premise_id = await _create_premise(client, free_udp_port)
    try:
        resp = await client.get(f"/api/v1/premises/{premise_id}/topology")
        assert resp.status_code == 200
        data = resp.json()
        assert data["area_number"] == 1
        assert data["line_number"] == 1
        assert data["gateway"] == "1.1.0"
        assert isinstance(data["devices"], list)

        resp = await client.get(
            f"/api/v1/premises/{premise_id}/next-device-number"
        )
        assert resp.status_code == 200
        assert resp.json()["next_device_number"] == 1
    finally:
        await client.delete(f"/api/v1/premises/{premise_id}")


async def test_groups_smoke(client, free_udp_port):
    premise_id = await _create_premise(client, free_udp_port)
    try:
        mg_payload = {"group_number": 1, "name": "Lighting"}
        resp = await client.post(
            f"/api/v1/premises/{premise_id}/main-groups", json=mg_payload
        )
        assert resp.status_code == 201
        main_group_id = resp.json()["id"]

        resp = await client.get(f"/api/v1/premises/{premise_id}/main-groups")
        assert resp.status_code == 200
        assert any(mg["id"] == main_group_id for mg in resp.json())

        middle_payload = {"group_number": 1, "name": "Ground Floor"}
        resp = await client.post(
            f"/api/v1/main-groups/{main_group_id}/middle-groups",
            json=middle_payload,
        )
        assert resp.status_code == 201

        resp = await client.get(
            f"/api/v1/main-groups/{main_group_id}/middle-groups"
        )
        assert resp.status_code == 200
        assert len(resp.json()) == 1

        resp = await client.get(f"/api/v1/premises/{premise_id}/groups")
        assert resp.status_code == 200
        assert "main_groups" in resp.json()
    finally:
        await client.delete(f"/api/v1/premises/{premise_id}")


async def test_templates_smoke(client):
    resp = await client.get("/api/v1/templates")
    assert resp.status_code == 200
    data = resp.json()
    assert data["count"] > 0
    template_id = data["templates"][0]["id"]

    resp = await client.get(f"/api/v1/templates/{template_id}")
    assert resp.status_code == 200
    assert resp.json()["id"] == template_id


async def test_telegrams_smoke(client, free_udp_port):
    premise_id = await _create_premise(client, free_udp_port)
    try:
        resp = await client.get(f"/api/v1/premises/{premise_id}/telegrams")
        assert resp.status_code == 200
        assert resp.json()["count"] == 0

        resp = await client.get(
            f"/api/v1/premises/{premise_id}/telegrams/stats"
        )
        assert resp.status_code == 200
        assert resp.json()["premise_id"] == premise_id
    finally:
        await client.delete(f"/api/v1/premises/{premise_id}")


async def test_reference_smoke(client):
    resp = await client.get("/api/v1/reference/dpts")
    assert resp.status_code == 200

    resp = await client.get("/api/v1/reference/device-templates")
    assert resp.status_code == 200

    resp = await client.get("/api/v1/reference/individual-address")
    assert resp.status_code == 200


async def test_health_smoke(client):
    resp = await client.get("/api/v1/health")
    assert resp.status_code == 200
    assert resp.json()["status"] == "ok"
