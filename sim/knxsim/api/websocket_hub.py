"""WebSocket Hub — manages connections and broadcasts events.

Supports two channel types:
  - "telegrams:{premise_id}" — raw decoded telegram stream
  - "state:{premise_id}" — device state change notifications

Thread-safe: synchronous code (KNX handlers) can push events via
the `push_telegram()` and `push_state_change()` methods which use
`asyncio.run_coroutine_threadsafe()` to bridge into the async loop.
"""

import asyncio
import json
import logging

from fastapi import WebSocket

logger = logging.getLogger("knxsim.ws")


class WebSocketHub:
    """Manages WebSocket connections and broadcasts events to subscribers."""

    def __init__(self):
        # channel -> set of WebSocket connections
        self._channels: dict[str, set[WebSocket]] = {}
        self._loop: asyncio.AbstractEventLoop | None = None

    def set_loop(self, loop: asyncio.AbstractEventLoop):
        """Set the asyncio event loop (called from uvicorn startup)."""
        self._loop = loop

    async def subscribe(self, ws: WebSocket, channel: str):
        """Add a WebSocket to a channel."""
        if channel not in self._channels:
            self._channels[channel] = set()
        self._channels[channel].add(ws)
        logger.info(
            "WS subscribed: %s (total: %d)", channel, len(self._channels[channel])
        )

    async def unsubscribe(self, ws: WebSocket, channel: str):
        """Remove a WebSocket from a channel."""
        if channel in self._channels:
            self._channels[channel].discard(ws)
            if not self._channels[channel]:
                del self._channels[channel]
            logger.info("WS unsubscribed: %s", channel)

    async def broadcast(self, channel: str, data: dict):
        """Send a JSON message to all subscribers of a channel."""
        subscribers = self._channels.get(channel)
        if not subscribers:
            return

        message = json.dumps(data)
        dead = []
        for ws in subscribers:
            try:
                await ws.send_text(message)
                logger.info("broadcast: sent to client on %s", channel)
            except (ConnectionError, RuntimeError) as e:
                logger.warning("broadcast: send failed: %s", e)
                dead.append(ws)

        # Clean up disconnected clients
        for ws in dead:
            subscribers.discard(ws)

    def push_telegram(self, premise_id: str, telegram_data: dict):
        """Thread-safe: push a telegram event from the KNX handler thread."""
        if not self._loop:
            logger.warning("push_telegram: no event loop set")
            return
        if not self._loop.is_running():
            logger.warning("push_telegram: event loop not running")
            return
        channel = f"telegrams:{premise_id}"
        subscribers = self._channels.get(channel)
        sub_count = len(subscribers) if subscribers else 0

        # Wrap in payload field for JS message routing
        data = {"type": "telegram", "payload": telegram_data}
        logger.info(
            "push_telegram: %s -> %d subscriber(s), loop_running=%s",
            channel,
            sub_count,
            self._loop.is_running(),
        )
        try:
            asyncio.run_coroutine_threadsafe(
                self.broadcast(channel, data),
                self._loop,
            )
            # Don't block, but log if there's an error
            self._loop.call_soon_threadsafe(
                lambda: logger.info("push_telegram: coroutine scheduled")
            )
        except RuntimeError as e:
            logger.error("push_telegram: RuntimeError: %s", e)

    def push_state_change(self, premise_id: str, device_id: str, state: dict, channels: list = None):
        """Thread-safe: push a state change event from the KNX handler thread."""
        if not self._loop:
            return
        if not self._loop.is_running():
            return
        data = {
            "type": "state_change",
            "premise_id": premise_id,
            "device_id": device_id,
            "state": state,
        }
        if channels is not None:
            data["channels"] = channels
        try:
            # Broadcast to state channel (for dedicated state subscribers)
            asyncio.run_coroutine_threadsafe(
                self.broadcast(f"state:{premise_id}", data),
                self._loop,
            )
            # Also broadcast to telegrams channel (for engineer UI)
            asyncio.run_coroutine_threadsafe(
                self.broadcast(f"telegrams:{premise_id}", data),
                self._loop,
            )
        except RuntimeError:
            pass  # Loop closed during shutdown

    @property
    def connection_count(self) -> int:
        """Total active WebSocket connections across all channels."""
        return sum(len(s) for s in self._channels.values())
