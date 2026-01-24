"""KNXnet/IP frame encoding and decoding.

All frames share a common 6-byte header:
  [header_len=0x06] [protocol=0x10] [service_type: 2B] [total_len: 2B]

After the header, the body varies by service type. This module provides
encode/decode functions for each frame type the simulator needs.
"""

import struct

from . import constants as C

# ---------------------------------------------------------------------------
# Address helpers
# ---------------------------------------------------------------------------


def parse_individual_address(text: str) -> int:
    """Parse "1.0.0" → 0x1000 (area.line.device as 4.4.8 bits)."""
    parts = text.split(".")
    area, line, device = int(parts[0]), int(parts[1]), int(parts[2])
    return (area << 12) | (line << 8) | device


def format_individual_address(addr: int) -> str:
    """Format 0x1000 → "1.0.0"."""
    return f"{(addr >> 12) & 0x0F}.{(addr >> 8) & 0x0F}.{addr & 0xFF}"


def parse_group_address(text: str) -> int:
    """Parse "1/1/0" → 0x0900 (main/middle/sub as 5.3.8 bits)."""
    parts = text.split("/")
    main, middle, sub = int(parts[0]), int(parts[1]), int(parts[2])
    return (main << 11) | (middle << 8) | sub


def format_group_address(addr: int) -> str:
    """Format 0x0900 → "1/1/0"."""
    return f"{(addr >> 11) & 0x1F}/{(addr >> 8) & 0x07}/{addr & 0xFF}"


# ---------------------------------------------------------------------------
# Header
# ---------------------------------------------------------------------------


def encode_header(service_type: int, body_len: int) -> bytes:
    """Encode the 6-byte KNXnet/IP header."""
    total = C.HEADER_SIZE + body_len
    return struct.pack("!BBHH", C.HEADER_SIZE, C.PROTOCOL_VERSION, service_type, total)


def decode_header(data: bytes):
    """Decode header → (service_type, total_length). Raises ValueError on bad header."""
    if len(data) < C.HEADER_SIZE:
        raise ValueError(f"Frame too short: {len(data)} bytes")
    hlen, version, service_type, total_len = struct.unpack("!BBHH", data[:6])
    if hlen != C.HEADER_SIZE:
        raise ValueError(f"Bad header length: {hlen}")
    if version != C.PROTOCOL_VERSION:
        raise ValueError(f"Bad protocol version: {version:#x}")
    return service_type, total_len


# ---------------------------------------------------------------------------
# HPAI (Host Protocol Address Information) — 8 bytes
# ---------------------------------------------------------------------------


def encode_hpai(ip: str = "0.0.0.0", port: int = 0) -> bytes:
    """Encode HPAI. Default 0.0.0.0:0 means 'use sender address' (NAT mode)."""
    octets = [int(x) for x in ip.split(".")]
    return struct.pack(
        "!BBBBBBH",
        C.HPAI_SIZE,
        C.IPV4_UDP,
        octets[0],
        octets[1],
        octets[2],
        octets[3],
        port,
    )


def decode_hpai(data: bytes, offset: int = 0):
    """Decode HPAI → (ip_str, port, bytes_consumed)."""
    hlen, protocol = struct.unpack_from("!BB", data, offset)
    ip = f"{data[offset + 2]}.{data[offset + 3]}.{data[offset + 4]}.{data[offset + 5]}"
    port = struct.unpack_from("!H", data, offset + 6)[0]
    return ip, port, hlen


# ---------------------------------------------------------------------------
# CONNECT_REQUEST / CONNECT_RESPONSE
# ---------------------------------------------------------------------------


def decode_connect_request(body: bytes):
    """Decode CONNECT_REQUEST body → (ctrl_ip, ctrl_port, data_ip, data_port, conn_type).

    Body layout:
      [HPAI control endpoint: 8B] [HPAI data endpoint: 8B] [CRI: 4B]
    CRI (Connection Request Information):
      [length=0x04] [conn_type] [tunnel_layer] [reserved=0x00]
    """
    ctrl_ip, ctrl_port, ctrl_len = decode_hpai(body, 0)
    data_ip, data_port, data_len = decode_hpai(body, ctrl_len)
    cri_offset = ctrl_len + data_len
    cri_len = body[cri_offset]
    conn_type = body[cri_offset + 1]
    return ctrl_ip, ctrl_port, data_ip, data_port, conn_type


def encode_connect_response(
    channel_id: int, status: int, data_ip: str, data_port: int, individual_address: int
) -> bytes:
    """Encode CONNECT_RESPONSE.

    Body layout:
      [channel_id: 1B] [status: 1B] [HPAI data endpoint: 8B] [CRD: 4B]
    CRD (Connection Response Data):
      [length=0x04] [conn_type=0x04] [individual_address: 2B]
    """
    body = struct.pack("!BB", channel_id, status)
    body += encode_hpai(data_ip, data_port)
    body += struct.pack("!BBH", 0x04, C.TUNNEL_CONNECTION, individual_address)
    return encode_header(C.CONNECT_RESPONSE, len(body)) + body


# ---------------------------------------------------------------------------
# CONNECTION_STATE_REQUEST / CONNECTION_STATE_RESPONSE
# ---------------------------------------------------------------------------


def decode_connection_state_request(body: bytes):
    """Decode → (channel_id,)."""
    return (body[0],)


def encode_connection_state_response(
    channel_id: int, status: int = C.E_NO_ERROR
) -> bytes:
    """Encode CONNECTION_STATE_RESPONSE."""
    body = struct.pack("!BB", channel_id, status)
    return encode_header(C.CONNECTION_STATE_RESPONSE, len(body)) + body


# ---------------------------------------------------------------------------
# DISCONNECT_REQUEST / DISCONNECT_RESPONSE
# ---------------------------------------------------------------------------


def decode_disconnect_request(body: bytes):
    """Decode → (channel_id,)."""
    return (body[0],)


def encode_disconnect_response(channel_id: int, status: int = C.E_NO_ERROR) -> bytes:
    """Encode DISCONNECT_RESPONSE."""
    body = struct.pack("!BB", channel_id, status)
    return encode_header(C.DISCONNECT_RESPONSE, len(body)) + body


# ---------------------------------------------------------------------------
# TUNNELLING_REQUEST / TUNNELLING_ACK
# ---------------------------------------------------------------------------


def decode_tunnelling_request(body: bytes):
    """Decode TUNNELLING_REQUEST body → (channel_id, seq, cemi_bytes).

    Body layout:
      [conn_header: 4B] [cEMI frame: variable]
    Connection header:
      [length=0x04] [channel_id] [sequence_counter] [reserved=0x00]
    """
    # Connection header
    ch_len = body[0]  # Always 0x04
    channel_id = body[1]
    seq = body[2]
    # cEMI starts after connection header
    cemi = body[ch_len:]
    return channel_id, seq, cemi


def encode_tunnelling_ack(
    channel_id: int, seq: int, status: int = C.E_NO_ERROR
) -> bytes:
    """Encode TUNNELLING_ACK."""
    body = struct.pack("!BBBB", 0x04, channel_id, seq, status)
    return encode_header(C.TUNNELLING_ACK, len(body)) + body


def encode_tunnelling_request(channel_id: int, seq: int, cemi: bytes) -> bytes:
    """Encode TUNNELLING_REQUEST with given cEMI payload."""
    conn_header = struct.pack("!BBBB", 0x04, channel_id, seq, 0x00)
    body = conn_header + cemi
    return encode_header(C.TUNNELLING_REQUEST, len(body)) + body


# ---------------------------------------------------------------------------
# cEMI frame helpers
# ---------------------------------------------------------------------------


def decode_cemi(data: bytes):
    """Decode cEMI frame → dict with msg_code, src, dst, apci, payload.

    cEMI layout:
      [msg_code: 1B] [add_info_len: 1B] [add_info: variable]
      [ctrl1: 1B] [ctrl2: 1B] [src: 2B] [dst: 2B]
      [data_len: 1B] [TPCI+APCI+data: variable]

    For GroupWrite with 1 data byte (6-bit APCI encoding):
      data_len = 1, single byte = APCI | value (for small values ≤ 0x3F)

    For GroupWrite with >1 data byte:
      data_len = N+1, first byte = APCI (high nibble), rest = data
    """
    msg_code = data[0]
    add_info_len = data[1]
    offset = 2 + add_info_len

    ctrl1 = data[offset]
    ctrl2 = data[offset + 1]
    src = struct.unpack_from("!H", data, offset + 2)[0]
    dst = struct.unpack_from("!H", data, offset + 4)[0]
    data_len = data[offset + 6]

    # In cEMI, data_len counts bytes AFTER the first TPCI/APCI byte.
    # The APDU on wire is always data_len + 1 bytes:
    #   [TPCI_APCI_byte] [remaining data_len bytes]
    apdu = data[offset + 7 : offset + 7 + data_len + 1]

    # APCI decoding:
    # The APDU is always at least 2 bytes: [TPCI(4bits)|APCI_high(2bits)|data(2bits)] [APCI_low(2bits)|data(6bits)]
    # For short frames (data_len=1): APCI in bits 9-8 of second byte, data in lower 6 bits
    # For long frames (data_len>1): APCI in bits 9-8 of second byte, data in bytes 2+
    if len(apdu) >= 2:
        # APCI is always in the upper 2 bits of byte[1]
        apci = apdu[1] & 0xC0  # 0x00=Read, 0x40=Response, 0x80=Write
        if data_len == 1:
            # Short frame: data value in lower 6 bits of byte[1]
            payload = bytes([apdu[1] & 0x3F])
        else:
            # Long frame: data bytes follow after the 2-byte TPCI+APCI header
            payload = apdu[2:]
    elif len(apdu) == 1:
        # Minimal frame (GroupRead with no data): just TPCI byte
        apci = 0x00  # Read
        payload = b""
    else:
        apci = 0
        payload = b""

    return {
        "msg_code": msg_code,
        "ctrl1": ctrl1,
        "ctrl2": ctrl2,
        "src": src,
        "dst": dst,
        "apci": apci,
        "payload": payload,
        "is_group": bool(ctrl2 & 0x80),
    }


def encode_cemi(
    msg_code: int,
    src: int,
    dst: int,
    apci: int,
    payload: bytes = b"",
    is_group: bool = True,
) -> bytes:
    """Encode a cEMI frame for group communication.

    For short payloads (1 byte, value ≤ 0x3F): uses compact 6-bit encoding.
    For longer payloads: uses extended format.
    """
    ctrl1 = C.CTRL1_STANDARD
    ctrl2 = C.CTRL2_GROUP_HOP6 if is_group else 0x60

    if len(payload) <= 1 and (not payload or payload[0] <= 0x3F):
        # Short frame: APCI + data in one byte (6-bit compact encoding)
        # data_len = 1, APDU = 1 byte: [APCI | value]
        val = payload[0] if payload else 0
        apdu = bytes([0x00, apci | val])
        data_len = 1
    else:
        # Long frame: 2-byte APCI header + data bytes
        # Byte 0: TPCI (0x00) | APCI high bits
        # Byte 1: APCI low bits (0x00=Read, 0x40=Response, 0x80=Write)
        # Bytes 2+: data
        apdu = bytes([0x00, apci]) + payload
        data_len = 1 + len(
            payload
        )  # APCI_low + data bytes (first TPCI byte not counted)

    # No additional info
    frame = struct.pack("!BB", msg_code, 0x00)
    frame += struct.pack("!BBHHB", ctrl1, ctrl2, src, dst, data_len)
    frame += apdu
    return frame
