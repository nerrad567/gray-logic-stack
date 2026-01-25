"""WebSocket routes for real-time telegram and state streams.

Endpoints:
  WS /ws/telegrams?premise={id}  — Live decoded telegram stream
  WS /ws/state?premise={id}      — Device state change notifications
"""

import logging

from fastapi import APIRouter, Query, WebSocket, WebSocketDisconnect

router = APIRouter(tags=["websocket"])
logger = logging.getLogger("knxsim.ws.routes")


@router.websocket("/ws/telegrams")
async def ws_telegrams(ws: WebSocket, premise: str = Query(default="default")):
    """Stream decoded telegrams in real-time for a premise."""
    hub = router.app.state.ws_hub
    await ws.accept()
    channel = f"telegrams:{premise}"
    await hub.subscribe(ws, channel)
    logger.info("Telegram stream connected: premise=%s", premise)

    try:
        # Keep connection alive — client can send pings or we just wait
        while True:
            # Wait for client messages (ping/pong or close)
            await ws.receive_text()
    except WebSocketDisconnect:
        pass
    finally:
        await hub.unsubscribe(ws, channel)
        logger.info("Telegram stream disconnected: premise=%s", premise)


@router.websocket("/ws/state")
async def ws_state(ws: WebSocket, premise: str = Query(default="default")):
    """Stream device state changes in real-time for a premise."""
    hub = router.app.state.ws_hub
    await ws.accept()
    channel = f"state:{premise}"
    await hub.subscribe(ws, channel)
    logger.info("State stream connected: premise=%s", premise)

    try:
        while True:
            await ws.receive_text()
    except WebSocketDisconnect:
        pass
    finally:
        await hub.unsubscribe(ws, channel)
        logger.info("State stream disconnected: premise=%s", premise)
