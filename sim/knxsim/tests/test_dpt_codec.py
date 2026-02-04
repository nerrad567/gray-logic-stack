"""Tests for KNX DPT codec encode/decode behavior."""

from __future__ import annotations

import pytest

from dpt.codec import _REGISTRY, DPTCodec


def _sample_value_for(dpt_id: str):
    """Return a representative value for the given DPT id."""
    main = dpt_id.split(".")[0]

    if dpt_id == "5.001":
        return 50.0
    if dpt_id == "5.003":
        return 90.0
    if dpt_id == "16.001":
        return "Hello KNXSim"
    if dpt_id == "17.001":
        return 12

    if main == "1":
        return True
    if main == "2":
        return {"control": True, "value": False}
    if main == "3":
        return {"control": True, "step": 5}
    if main == "4":
        return "A"
    if main == "5":
        return 42
    if main == "6":
        return -12
    if main == "7":
        return 1234
    if main == "8":
        return -1234
    if main == "9":
        return 21.5
    if main == "10":
        return {"day": 2, "hour": 14, "minute": 30, "second": 15}
    if main == "11":
        return {"day": 3, "month": 2, "year": 24}
    if main == "12":
        return 123456
    if main == "13":
        return -123456
    if main == "14":
        return 12.3456
    if main == "16":
        return "Hello KNXSim"
    if main == "17":
        return 12
    if main == "18":
        return {"learn": True, "number": 12}
    if main == "20":
        return 3

    raise AssertionError(f"No sample value configured for DPT {dpt_id}")


def _param_id(dpt_id: str) -> str:
    """Return a readable pytest id including DPT family grouping."""
    main = dpt_id.split(".")[0]
    return f"{dpt_id} ({main}.x)"


@pytest.mark.parametrize("dpt_id", sorted(_REGISTRY.keys()), ids=_param_id)
def test_roundtrip_all_registered_dpts(dpt_id: str):
    """All registered DPTs should roundtrip encode -> decode."""
    value = _sample_value_for(dpt_id)
    encoded = DPTCodec.encode(dpt_id, value)
    decoded = DPTCodec.decode(dpt_id, encoded)

    main = dpt_id.split(".")[0]
    if dpt_id in {"5.001"}:
        assert decoded == pytest.approx(value, abs=0.5)
        return
    if dpt_id in {"5.003"}:
        assert decoded == pytest.approx(value, abs=1.0)
        return
    if main == "9":
        assert decoded == pytest.approx(value, abs=0.05)
        return
    if main == "14":
        assert decoded == pytest.approx(value, abs=1e-4)
        return
    if main == "11":
        expected_year = value["year"]
        if expected_year < 90:
            expected_year = 2000 + expected_year
        else:
            expected_year = 1900 + expected_year
        assert decoded["day"] == value["day"]
        assert decoded["month"] == value["month"]
        assert decoded["year"] == expected_year
        return

    assert decoded == value


def test_dpt_registry_entries_have_encoders_and_decoders():
    """Ensure all registry entries expose encoder/decoder callables."""
    for dpt_id, entry in _REGISTRY.items():
        assert len(entry) == 3
        encoder, decoder, info = entry
        assert callable(encoder)
        assert callable(decoder)
        assert info is not None


@pytest.mark.parametrize("value", [0.0, 100.0])
def test_dpt5_001_boundaries(value: float):
    """DPT 5.001 should clamp to 0..100 and roundtrip at boundaries."""
    encoded = DPTCodec.encode("5.001", value)
    decoded = DPTCodec.decode("5.001", encoded)
    assert decoded == pytest.approx(value, abs=0.1)


def test_dpt9_invalid_reserved_value():
    """DPT 9 invalid reserved value 0x7FFF decodes to max range value."""
    decoded = DPTCodec.decode("9.001", bytes([0x7F, 0xFF]))
    assert decoded == pytest.approx(670760.96, abs=0.01)


@pytest.mark.parametrize(
    "text,expected",
    [
        ("", ""),
        ("12345678901234", "12345678901234"),
        ("12345678901234567890", "12345678901234"),
    ],
)
def test_dpt16_001_string_lengths(text: str, expected: str):
    """DPT 16.001 should truncate/pad to 14 ASCII characters."""
    encoded = DPTCodec.encode("16.001", text)
    decoded = DPTCodec.decode("16.001", encoded)
    assert decoded == expected


@pytest.mark.parametrize("value", [0, 63])
def test_dpt17_001_boundaries(value: int):
    """DPT 17.001 should preserve scene number boundaries 0..63."""
    encoded = DPTCodec.encode("17.001", value)
    decoded = DPTCodec.decode("17.001", encoded)
    assert decoded == value


def test_unknown_dpt_id_raises():
    """Unknown DPT IDs should raise ValueError on encode/decode."""
    with pytest.raises(ValueError):
        DPTCodec.encode("99.999", 1)
    with pytest.raises(ValueError):
        DPTCodec.decode("99.999", b"\x00")


def test_decode_none_payload_returns_default():
    """Decoding with None payload should return a safe default (not crash)."""
    result = DPTCodec.decode("1.001", None)
    assert result is False


@pytest.mark.parametrize(
    "dpt_id,payload,expected",
    [
        ("7.001", b"\x01", 0),
        ("8.001", b"\x01", 0),
        ("9.001", b"\x01", 0.0),
        ("10.001", b"\x01\x02", {"day": 0, "hour": 0, "minute": 0, "second": 0}),
        ("11.001", b"\x01\x02", {"day": 1, "month": 1, "year": 0}),
        ("12.001", b"\x01\x02\x03", 0),
        ("13.001", b"\x01\x02\x03", 0),
        ("14.019", b"\x01\x02\x03", 0.0),
    ],
)
def test_decode_payload_too_short_defaults(dpt_id: str, payload: bytes, expected):
    """Payloads shorter than the DPT size should return safe defaults."""
    decoded = DPTCodec.decode(dpt_id, payload)
    assert decoded == expected
