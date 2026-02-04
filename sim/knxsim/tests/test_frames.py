"""Tests for KNXnet/IP cEMI framing helpers."""

from __future__ import annotations

import pytest

from knxip import constants as C
from knxip.frames import (
    decode_cemi,
    encode_cemi,
    format_group_address,
    format_individual_address,
    parse_group_address,
    parse_individual_address,
)


def test_cemi_roundtrip_group_write_short_payload():
    """GroupWrite with 1-byte payload should roundtrip encode/decode."""
    src = parse_individual_address("1.1.10")
    dst = parse_group_address("1/2/3")
    payload = bytes([0x01])

    cemi = encode_cemi(
        msg_code=C.L_DATA_IND,
        src=src,
        dst=dst,
        apci=C.APCI_GROUP_WRITE,
        payload=payload,
    )
    decoded = decode_cemi(cemi)

    assert decoded["msg_code"] == C.L_DATA_IND
    assert decoded["src"] == src
    assert decoded["dst"] == dst
    assert decoded["apci"] == C.APCI_GROUP_WRITE
    assert decoded["payload"] == payload


def test_cemi_roundtrip_group_read():
    """GroupRead encodes as a short frame and decodes back cleanly."""
    src = parse_individual_address("1.1.10")
    dst = parse_group_address("1/2/3")

    cemi = encode_cemi(
        msg_code=C.L_DATA_IND,
        src=src,
        dst=dst,
        apci=C.APCI_GROUP_READ,
        payload=b"",
    )
    decoded = decode_cemi(cemi)

    assert decoded["msg_code"] == C.L_DATA_IND
    assert decoded["apci"] == C.APCI_GROUP_READ
    # Short frame GroupRead yields a zero data byte
    assert decoded["payload"] == bytes([0x00])


def test_cemi_roundtrip_group_response_long_payload():
    """GroupResponse with multi-byte payload should roundtrip encode/decode."""
    src = parse_individual_address("1.1.10")
    dst = parse_group_address("1/2/3")
    payload = bytes([0x12, 0x34])

    cemi = encode_cemi(
        msg_code=C.L_DATA_IND,
        src=src,
        dst=dst,
        apci=C.APCI_GROUP_RESPONSE,
        payload=payload,
    )
    decoded = decode_cemi(cemi)

    assert decoded["apci"] == C.APCI_GROUP_RESPONSE
    assert decoded["payload"] == payload


def test_cemi_golden_frame_decode():
    """Decode a known cEMI frame to guard against silent regressions."""
    # msg_code=0x29, src=1.1.10, dst=1/2/3, APCI=GroupWrite, payload=0x01
    golden = bytes([0x29, 0x00, 0xBC, 0xE0, 0x11, 0x0A, 0x0A, 0x03, 0x01, 0x00, 0x81])
    decoded = decode_cemi(golden)

    assert decoded["msg_code"] == C.L_DATA_IND
    assert decoded["src"] == parse_individual_address("1.1.10")
    assert decoded["dst"] == parse_group_address("1/2/3")
    assert decoded["apci"] == C.APCI_GROUP_WRITE
    assert decoded["payload"] == bytes([0x01])


def test_decode_cemi_short_buffer_raises():
    """Very short buffers should raise when decoding cEMI."""
    with pytest.raises(IndexError):
        decode_cemi(bytes([0x29, 0x00, 0xBC]))


def test_decode_cemi_bad_message_code_is_passed_through():
    """Unknown message codes should decode without crashing."""
    src = parse_individual_address("1.1.10")
    dst = parse_group_address("1/2/3")
    cemi = encode_cemi(
        msg_code=0xFF,
        src=src,
        dst=dst,
        apci=C.APCI_GROUP_WRITE,
        payload=bytes([0x01]),
    )
    decoded = decode_cemi(cemi)
    assert decoded["msg_code"] == 0xFF


def test_decode_cemi_truncated_payload_defaults():
    """Truncated payloads should not crash and return empty payloads."""
    truncated = bytes([0x29, 0x00, 0xBC, 0xE0, 0x11, 0x0A, 0x0A, 0x03, 0x02, 0x00])
    decoded = decode_cemi(truncated)
    assert decoded["payload"] == b""
    assert decoded["apci"] == 0x00


@pytest.mark.parametrize(
    "address",
    ["0.0.0", "1.0.0", "15.15.255"],
)
def test_individual_address_roundtrip(address: str):
    """Individual address parse/format should roundtrip."""
    encoded = parse_individual_address(address)
    assert format_individual_address(encoded) == address


@pytest.mark.parametrize(
    "address",
    ["0/0/0", "1/2/3", "31/7/255"],
)
def test_group_address_roundtrip(address: str):
    """Group address parse/format should roundtrip for 3-level addresses."""
    encoded = parse_group_address(address)
    assert encoded is not None
    assert format_group_address(encoded) == address
