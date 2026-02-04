"""API error-path tests for KNXSim."""

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


async def test_premise_not_found(client):
    resp = await client.get("/api/v1/premises/nonexistent")
    assert resp.status_code == 404

    resp = await client.delete("/api/v1/premises/nonexistent")
    assert resp.status_code == 404


async def test_device_errors(client, free_udp_port):
    payload = _premise_payload(free_udp_port)
    resp = await client.post("/api/v1/premises", json=payload)
    assert resp.status_code == 201
    premise_id = resp.json()["id"]

    try:
        resp = await client.get(f"/api/v1/premises/{premise_id}/devices/bad-id")
        assert resp.status_code == 404

        resp = await client.post(f"/api/v1/premises/{premise_id}/devices", json={})
        assert resp.status_code == 422

        resp = await client.patch(
            f"/api/v1/premises/{premise_id}/devices/bad-id",
            json={"room_id": "room-1"},
        )
        assert resp.status_code == 404
    finally:
        await client.delete(f"/api/v1/premises/{premise_id}")


async def test_create_premise_validation(client):
    resp = await client.post("/api/v1/premises", json={"name": "Missing fields"})
    assert resp.status_code == 422
