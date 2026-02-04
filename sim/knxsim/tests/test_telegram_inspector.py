"""Unit tests for TelegramInspector."""

from __future__ import annotations

from core.telegram_inspector import TelegramInspector
from knxip import constants as C
from knxip import frames


def _cemi(dst: str, src: str = "1.1.1", payload: bytes = b"\x01") -> dict:
    return {
        "src": frames.parse_individual_address(src),
        "dst": frames.parse_group_address(dst),
        "apci": C.APCI_GROUP_WRITE,
        "payload": payload,
    }


def test_record_and_history_order():
    inspector = TelegramInspector(max_size=5)
    inspector.record("p1", _cemi("1/1/1"), direction="rx", device_id="dev-1")
    inspector.record("p1", _cemi("1/1/2"), direction="tx", device_id="dev-2")

    history = inspector.get_history("p1")
    assert len(history) == 2
    assert history[0]["destination"] == "1/1/2"
    assert history[1]["destination"] == "1/1/1"


def test_ring_buffer_eviction():
    inspector = TelegramInspector(max_size=2)
    inspector.record("p1", _cemi("1/1/1"))
    inspector.record("p1", _cemi("1/1/2"))
    inspector.record("p1", _cemi("1/1/3"))

    history = inspector.get_history("p1")
    assert len(history) == 2
    assert history[0]["destination"] == "1/1/3"
    assert history[1]["destination"] == "1/1/2"


def test_stats_and_filters():
    inspector = TelegramInspector(max_size=10)
    inspector.record("p1", _cemi("1/1/1"), direction="rx", device_id="dev-1")
    inspector.record("p1", _cemi("1/1/2"), direction="tx", device_id="dev-2")
    inspector.record("p1", _cemi("1/1/2"), direction="tx", device_id="dev-2")

    stats = inspector.get_stats("p1")
    assert stats["rx_count"] == 1
    assert stats["tx_count"] == 2
    assert stats["unique_gas"] == 2
    assert stats["top_gas"][0]["ga"] == "1/1/2"
    assert stats["top_gas"][0]["count"] == 2

    assert len(inspector.get_history("p1", direction="rx")) == 1
    assert len(inspector.get_history("p1", device="dev-2")) == 2
    assert len(inspector.get_history("p1", ga="1/1/2")) == 2


def test_clear_and_isolation():
    inspector = TelegramInspector(max_size=5)
    inspector.record("p1", _cemi("1/1/1"))
    inspector.record("p2", _cemi("2/2/2"))

    assert len(inspector.get_history("p1")) == 1
    assert len(inspector.get_history("p2")) == 1

    inspector.clear("p1")
    assert inspector.get_history("p1") == []
    assert len(inspector.get_history("p2")) == 1
