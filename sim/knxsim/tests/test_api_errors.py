"""API error-path tests for KNXSim."""

from __future__ import annotations

import uuid


def _premise_payload(port: int) -> dict:
    premise_id = f"premise-{uuid.uuid4().hex[:8]}"
    return {
        "id": premise_id,
        "name": "Test Premise",
        "gateway_address": "1.0.0",
        "client_address": "1.0.255",
        "port": port,
    }


def test_premise_not_found(client):
    resp = client.get("/api/v1/premises/nonexistent")
    assert resp.status_code == 404

    resp = client.delete("/api/v1/premises/nonexistent")
    assert resp.status_code == 404


def test_device_errors(client, free_udp_port):
    payload = _premise_payload(free_udp_port)
    resp = client.post("/api/v1/premises", json=payload)
    assert resp.status_code == 201
    premise_id = resp.json()["id"]

    try:
        resp = client.get(f"/api/v1/premises/{premise_id}/devices/bad-id")
        assert resp.status_code == 404

        resp = client.post(f"/api/v1/premises/{premise_id}/devices", json={})
        assert resp.status_code == 422

        resp = client.patch(
            f"/api/v1/premises/{premise_id}/devices/bad-id",
            json={"room_id": "room-1"},
        )
        assert resp.status_code == 404
    finally:
        client.delete(f"/api/v1/premises/{premise_id}")


def test_create_premise_validation(client):
    resp = client.post("/api/v1/premises", json={"name": "Missing fields"})
    assert resp.status_code == 422
