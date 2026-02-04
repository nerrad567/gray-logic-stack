"""Pytest configuration and shared fixtures for KNXSim tests."""

from __future__ import annotations

import itertools
import os
import sys

import pytest

_TESTS_DIR = os.path.dirname(__file__)
_PROJECT_ROOT = os.path.abspath(os.path.join(_TESTS_DIR, ".."))
if _PROJECT_ROOT not in sys.path:
    sys.path.insert(0, _PROJECT_ROOT)


@pytest.fixture
def tmp_db_path(tmp_path):
    """Provide a temporary SQLite database path."""
    return str(tmp_path / "test_knxsim.db")


@pytest.fixture
def db(tmp_db_path):
    """Create a fresh Database with schema applied."""
    from persistence.db import Database

    database = Database(db_path=tmp_db_path)
    database.connect()
    yield database
    database.close()


@pytest.fixture
def template_loader():
    """Create a TemplateLoader with real templates loaded."""
    from templates.loader import TemplateLoader

    loader = TemplateLoader()
    loader.load_all()
    return loader


@pytest.fixture
def premise_manager(db, template_loader):
    """Create a PremiseManager with a test database."""
    from core.premise_manager import PremiseManager

    manager = PremiseManager(db=db, template_loader=template_loader)
    return manager


@pytest.fixture
def sample_premise(premise_manager, db):
    """Create a premise with a few sample devices for testing."""
    premise_id = "test-premise"
    db.create_premise(
        {
            "id": premise_id,
            "name": "Test Premise",
            "area_number": 1,
            "line_number": 1,
            "port": 3671,
        }
    )
    # Add a light switch device
    db.create_device(
        premise_id,
        {
            "id": "light-1",
            "type": "light_switch",
            "individual_address": "1.1.1",
            "group_addresses": {"switch": {"ga": "0/0/1", "dpt": "1.001"}},
            "state": {"on": False},
            "initial_state": {"on": False},
        },
    )
    # Add a dimmer device
    db.create_device(
        premise_id,
        {
            "id": "dimmer-1",
            "type": "light_dimmer",
            "individual_address": "1.1.2",
            "group_addresses": {
                "switch": {"ga": "0/0/2", "dpt": "1.001"},
                "brightness": {"ga": "0/0/3", "dpt": "5.001"},
            },
            "state": {"on": False, "brightness": 0},
            "initial_state": {"on": False, "brightness": 0},
        },
    )
    return premise_id


@pytest.fixture
def telegram_inspector():
    """Create a TelegramInspector instance."""
    from core.telegram_inspector import TelegramInspector

    return TelegramInspector(max_size=100)


@pytest.fixture
def ws_hub():
    """Create a WebSocketHub instance (no event loop bound)."""
    from api.websocket_hub import WebSocketHub

    return WebSocketHub()


@pytest.fixture
def app(premise_manager, ws_hub, telegram_inspector):
    """Create a FastAPI test app with all dependencies injected."""
    from api.app import create_app

    return create_app(
        manager=premise_manager,
        ws_hub=ws_hub,
        telegram_inspector=telegram_inspector,
    )


@pytest.fixture
async def client(app):
    """Create an AsyncClient for HTTP testing against the ASGI app."""
    try:
        import httpx
    except ModuleNotFoundError:
        pytest.skip("httpx not installed in venv; skipping API client tests")

    transport = httpx.ASGITransport(app=app)
    async with httpx.AsyncClient(
        transport=transport, base_url="http://testserver"
    ) as client:
        yield client


_PORT_COUNTER = itertools.count(40000)


@pytest.fixture(autouse=True)
def _disable_knxip_server(monkeypatch):
    """Avoid binding UDP sockets in tests."""
    from knxip.server import KNXIPServer

    def _start(self):  # pragma: no cover - trivial override
        self._running = True

    def _stop(self):  # pragma: no cover - trivial override
        self._running = False

    monkeypatch.setattr(KNXIPServer, "start", _start)
    monkeypatch.setattr(KNXIPServer, "stop", _stop)


@pytest.fixture(autouse=True)
def _run_sync_endpoints_inline(monkeypatch):
    """Run sync FastAPI endpoints inline to avoid AnyIO threadpool hangs."""
    import fastapi.concurrency as fastapi_concurrency
    import fastapi.dependencies.utils as fastapi_dep_utils
    import fastapi.routing as fastapi_routing
    import starlette.concurrency as starlette_concurrency

    async def _run_inline(func, *args, **kwargs):
        return func(*args, **kwargs)

    monkeypatch.setattr(starlette_concurrency, "run_in_threadpool", _run_inline)
    monkeypatch.setattr(fastapi_concurrency, "run_in_threadpool", _run_inline)
    monkeypatch.setattr(fastapi_routing, "run_in_threadpool", _run_inline)
    monkeypatch.setattr(fastapi_dep_utils, "run_in_threadpool", _run_inline)


@pytest.fixture
def free_udp_port():
    """Return a deterministic UDP port (sockets disabled in tests)."""
    return next(_PORT_COUNTER)
