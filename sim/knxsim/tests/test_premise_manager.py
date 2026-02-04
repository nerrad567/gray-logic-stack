"""Unit tests for PremiseManager."""

from __future__ import annotations

import uuid


def _premise_data(port: int) -> dict:
    pid = f"premise-{uuid.uuid4().hex[:8]}"
    return {
        "id": pid,
        "name": "Test Premise",
        "gateway_address": "1.0.0",
        "client_address": "1.0.255",
        "port": port,
    }


def test_create_list_get_delete_premise(premise_manager, free_udp_port):
    data = _premise_data(free_udp_port)
    premise_id = data["id"]
    premise_manager.create_premise(data)
    try:
        premise = premise_manager.get_premise(premise_id)
        assert premise is not None

        premises = premise_manager.list_premises()
        assert any(p["id"] == premise_id for p in premises)
    finally:
        premise_manager.delete_premise(premise_id)

    assert premise_manager.get_premise(premise_id) is None


def test_premise_not_found(premise_manager):
    assert premise_manager.get_premise("missing-premise") is None
