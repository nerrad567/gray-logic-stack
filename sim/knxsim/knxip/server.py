"""KNXnet/IP tunnelling server.

Listens on UDP 3671 and handles the tunnelling protocol:
  - CONNECT_REQUEST → assign channel, respond with individual address
  - CONNECTION_STATE_REQUEST → heartbeat ACK
  - DISCONNECT_REQUEST → tear down channel
  - TUNNELLING_REQUEST → parse cEMI, dispatch to device handler, send ACK + response
"""

import logging
import socket
import struct
import threading
from typing import Callable, Optional

from . import constants as C
from . import frames

logger = logging.getLogger("knxsim.server")


class TunnelChannel:
    """Represents an active tunnelling connection with a client (knxd)."""

    def __init__(self, channel_id: int, client_addr: tuple, individual_address: int):
        self.channel_id = channel_id
        self.client_addr = client_addr  # (ip, port) for sending responses
        self.individual_address = individual_address
        self.recv_seq = 0  # Next expected sequence from client
        self.send_seq = 0  # Our outgoing sequence counter


class KNXIPServer:
    """UDP server implementing KNXnet/IP tunnelling protocol.

    Args:
        host: Bind address (default 0.0.0.0 — all interfaces)
        port: UDP port (default 3671 — standard KNXnet/IP)
        client_address: Individual address assigned to connecting clients
        gateway_address: Our individual address as the "gateway"
        on_telegram: Callback for incoming telegrams: fn(channel, cemi_dict) → optional response cEMI bytes
    """

    def __init__(
        self,
        host: str = "0.0.0.0",
        port: int = 3671,
        client_address: int = 0x10FF,  # 1.0.255
        gateway_address: int = 0x1000,  # 1.0.0
        on_telegram: Optional[Callable] = None,
    ):
        self.host = host
        self.port = port
        self.client_address = client_address
        self.gateway_address = gateway_address
        self.on_telegram = on_telegram

        self._sock: Optional[socket.socket] = None
        self._running = False
        self._thread: Optional[threading.Thread] = None
        self._channels: dict[int, TunnelChannel] = {}
        self._next_channel_id = 1
        self._lock = threading.Lock()

    def start(self):
        """Start the UDP server in a background thread."""
        self._sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self._sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self._sock.bind((self.host, self.port))
        self._sock.settimeout(1.0)  # Allow periodic check of _running flag
        self._running = True
        self._thread = threading.Thread(
            target=self._recv_loop, name="knxip-server", daemon=True
        )
        self._thread.start()
        logger.info("KNXnet/IP server listening on %s:%d/udp", self.host, self.port)

    def stop(self):
        """Stop the server."""
        self._running = False
        if self._thread:
            self._thread.join(timeout=3.0)
        if self._sock:
            self._sock.close()
        logger.info("KNXnet/IP server stopped")

    def send_telegram(self, cemi: bytes):
        """Send a tunnelling request (e.g., sensor update) to all connected clients."""
        with self._lock:
            channels = list(self._channels.values())
        if not channels:
            return
        for ch in channels:
            frame = frames.encode_tunnelling_request(ch.channel_id, ch.send_seq, cemi)
            logger.debug(
                "→ TUNNEL_REQ ch=%d seq=%d to %s (%d bytes)",
                ch.channel_id,
                ch.send_seq,
                ch.client_addr,
                len(frame),
            )
            ch.send_seq = (ch.send_seq + 1) & 0xFF
            self._send(frame, ch.client_addr)

    def _recv_loop(self):
        """Main receive loop — reads UDP datagrams and dispatches by service type."""
        while self._running:
            try:
                data, addr = self._sock.recvfrom(1024)
            except socket.timeout:
                continue
            except OSError:
                if self._running:
                    logger.exception("Socket error")
                break

            if len(data) < C.HEADER_SIZE:
                continue

            logger.debug("← RAW UDP from %s: %s", addr, data.hex())

            try:
                service_type, total_len = frames.decode_header(data)
            except ValueError as e:
                logger.warning("Bad frame from %s: %s", addr, e)
                continue

            body = data[C.HEADER_SIZE : total_len]

            try:
                self._dispatch(service_type, body, addr)
            except Exception:
                logger.exception(
                    "Error handling service %#06x from %s", service_type, addr
                )

    def _dispatch(self, service_type: int, body: bytes, addr: tuple):
        """Route frame to appropriate handler."""
        if service_type == C.CONNECT_REQUEST:
            self._handle_connect(body, addr)
        elif service_type == C.CONNECTION_STATE_REQUEST:
            self._handle_connection_state(body, addr)
        elif service_type == C.DISCONNECT_REQUEST:
            self._handle_disconnect(body, addr)
        elif service_type == C.TUNNELLING_REQUEST:
            self._handle_tunnelling(body, addr)
        elif service_type == C.TUNNELLING_ACK:
            pass  # Client acknowledged our telegram — nothing to do
        else:
            logger.debug("Ignoring unknown service type %#06x", service_type)

    def _handle_connect(self, body: bytes, addr: tuple):
        """Handle CONNECT_REQUEST — assign a channel and respond."""
        ctrl_ip, ctrl_port, data_ip, data_port, conn_type = (
            frames.decode_connect_request(body)
        )

        if conn_type != C.TUNNEL_CONNECTION:
            resp = frames.encode_connect_response(
                0, C.E_CONNECTION_TYPE, "0.0.0.0", 0, 0
            )
            self._send(resp, addr)
            return

        with self._lock:
            if len(self._channels) >= 255:
                resp = frames.encode_connect_response(
                    0, C.E_NO_MORE_CONNECTIONS, "0.0.0.0", 0, 0
                )
                self._send(resp, addr)
                return

            channel_id = self._next_channel_id
            self._next_channel_id = (self._next_channel_id % 255) + 1

            # Use sender's actual address for responses (NAT-friendly)
            channel = TunnelChannel(channel_id, addr, self.client_address)
            self._channels[channel_id] = channel

        resp = frames.encode_connect_response(
            channel_id,
            C.E_NO_ERROR,
            "0.0.0.0",
            0,  # HPAI 0.0.0.0:0 = use sender address
            self.client_address,
        )
        self._send(resp, addr)

        logger.info(
            "CONNECT from %s:%d → channel %d, assigned %s",
            addr[0],
            addr[1],
            channel_id,
            frames.format_individual_address(self.client_address),
        )

    def _handle_connection_state(self, body: bytes, addr: tuple):
        """Handle CONNECTION_STATE_REQUEST — heartbeat response."""
        (channel_id,) = frames.decode_connection_state_request(body)

        with self._lock:
            if channel_id not in self._channels:
                status = C.E_DATA_CONNECTION
            else:
                status = C.E_NO_ERROR

        resp = frames.encode_connection_state_response(channel_id, status)
        self._send(resp, addr)

    def _handle_disconnect(self, body: bytes, addr: tuple):
        """Handle DISCONNECT_REQUEST — remove channel and confirm."""
        (channel_id,) = frames.decode_disconnect_request(body)

        with self._lock:
            channel = self._channels.pop(channel_id, None)

        resp = frames.encode_disconnect_response(channel_id, C.E_NO_ERROR)
        self._send(resp, addr)

        if channel:
            logger.info(
                "DISCONNECT channel %d from %s:%d", channel_id, addr[0], addr[1]
            )

    def _handle_tunnelling(self, body: bytes, addr: tuple):
        """Handle TUNNELLING_REQUEST — parse cEMI, dispatch, send ACK + response."""
        channel_id, seq, cemi_bytes = frames.decode_tunnelling_request(body)

        with self._lock:
            channel = self._channels.get(channel_id)

        if not channel:
            return

        # Send ACK immediately
        ack = frames.encode_tunnelling_ack(channel_id, seq)
        self._send(ack, addr)

        # Update expected sequence
        channel.recv_seq = (seq + 1) & 0xFF

        # Parse and dispatch cEMI to device handler
        logger.info(
            "← TUNNEL_REQ ch=%d seq=%d cemi_len=%d cemi=%s",
            channel_id,
            seq,
            len(cemi_bytes),
            cemi_bytes.hex(),
        )
        if self.on_telegram and len(cemi_bytes) > 7:
            cemi_dict = frames.decode_cemi(cemi_bytes)
            logger.info(
                "  cEMI parsed: msg_code=%#04x src=%s dst=%s apci=%#04x is_group=%s payload=%s",
                cemi_dict["msg_code"],
                frames.format_individual_address(cemi_dict["src"]),
                frames.format_group_address(cemi_dict["dst"])
                if cemi_dict["is_group"]
                else frames.format_individual_address(cemi_dict["dst"]),
                cemi_dict["apci"],
                cemi_dict["is_group"],
                cemi_dict["payload"].hex() if cemi_dict["payload"] else "(empty)",
            )
            # Only process data requests from client (L_DATA.req or L_DATA.ind with group addr)
            if cemi_dict["is_group"]:
                responses = self.on_telegram(channel, cemi_dict)
                if responses:
                    logger.info(
                        "  on_telegram returned: %d response(s)",
                        len(responses),
                    )
                    # Send each response as a separate TUNNELLING_REQUEST
                    for response_cemi in responses:
                        resp_frame = frames.encode_tunnelling_request(
                            channel_id, channel.send_seq, response_cemi
                        )
                        logger.info(
                            "→ TUNNEL_REQ ch=%d seq=%d response (%d bytes)",
                            channel_id,
                            channel.send_seq,
                            len(resp_frame),
                        )
                        channel.send_seq = (channel.send_seq + 1) & 0xFF
                        self._send(resp_frame, addr)

    def _send(self, data: bytes, addr: tuple):
        """Send a UDP datagram."""
        try:
            self._sock.sendto(data, addr)
        except OSError:
            logger.exception("Failed to send to %s", addr)
