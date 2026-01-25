"""FastAPI application factory for the KNX Simulator Management API.

Creates the app with all routers mounted and the PremiseManager
injected via app.state. Serves static files for the web UI.
"""

import asyncio
import logging
import os

from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles

from .routes_devices import router as devices_router
from .routes_floors import router as floors_router
from .routes_premises import router as premises_router
from .routes_telegrams import router as telegrams_router
from .routes_templates import router as templates_router
from .routes_ws import router as ws_router
from .websocket_hub import WebSocketHub

logger = logging.getLogger("knxsim.api")


def create_app(
    manager, ws_hub: WebSocketHub = None, telegram_inspector=None
) -> FastAPI:
    """Create the FastAPI application with all routes.

    Args:
        manager: PremiseManager instance (injected into app.state)
        ws_hub: WebSocketHub for real-time event broadcast
        telegram_inspector: TelegramInspector for history access
    """
    app = FastAPI(
        title="KNX Simulator Manager",
        description="Management API for the KNX/IP Gateway Simulator",
        version="1.0.0",
    )

    # Initialize template loader
    from templates.loader import TemplateLoader

    template_loader = TemplateLoader()
    template_loader.load_all()

    # Store references for route handlers
    app.state.manager = manager
    app.state.ws_hub = ws_hub or WebSocketHub()
    app.state.telegram_inspector = telegram_inspector
    app.state.template_loader = template_loader

    # Mount routers
    app.include_router(premises_router)
    app.include_router(devices_router)
    app.include_router(floors_router)
    app.include_router(telegrams_router)
    app.include_router(templates_router)
    app.include_router(ws_router)

    # Set app reference on routers (needed for app.state access)
    premises_router.app = app
    devices_router.app = app
    floors_router.app = app
    telegrams_router.app = app
    templates_router.app = app
    ws_router.app = app

    # Capture the event loop on startup for thread-safe WebSocket pushes
    @app.on_event("startup")
    async def on_startup():
        loop = asyncio.get_running_loop()
        app.state.ws_hub.set_loop(loop)
        logger.info("WebSocket hub bound to event loop")

    # Health endpoint
    @app.get("/api/v1/health", tags=["system"])
    def health():
        """Health check — returns running premise count and WS connections."""
        running = sum(1 for p in manager.premises.values() if p.is_running)
        return {
            "status": "ok",
            "premises_total": len(manager.premises),
            "premises_running": running,
            "ws_connections": app.state.ws_hub.connection_count,
        }

    # Mount static files for web UI (if directory exists)
    static_dir = os.path.join(os.path.dirname(os.path.dirname(__file__)), "static")
    if os.path.isdir(static_dir):
        app.mount("/ui", StaticFiles(directory=static_dir, html=True), name="ui")
        logger.info("Static UI mounted at /ui from %s", static_dir)

    logger.info("FastAPI app created with %d routers", 6)
    return app
