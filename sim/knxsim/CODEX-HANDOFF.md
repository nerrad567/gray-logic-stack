# KNXSim — Codex Handoff

> **Created:** 2026-02-04
> **Purpose:** Structured task list for Codex to close out KNXSim quality gaps.
> **Goal:** After these tasks, KNXSim should be production-grade and never need revisiting.

---

## Context

KNXSim is a Python-based KNX simulator with:
- **FastAPI** backend (73+ REST endpoints across 11 route files)
- **Alpine.js** frontend (single-page dashboard)
- **KNXnet/IP** tunnelling server (UDP 3671)
- **SQLite** persistence via `persistence/db.py`
- **57 YAML templates** for device types

### What's Already Tested (107 tests, all passing)
- `tests/test_dpt_codec.py` — DPT encode/decode roundtrips (73 parametrized cases)
- `tests/test_frames.py` — cEMI frame encode/decode, address parsing
- `tests/test_ga_normalization.py` — GA normalisation and deprecation warnings

### What's NOT Tested (the gap this handoff closes)
- 0 of 73+ API endpoints
- 0 of 9 device classes
- 0 of 6 core modules (Premise, PremiseManager, TelegramInspector, WebSocketHub, etc.)
- 0 integration/smoke tests
- No test fixtures beyond sys.path setup

---

## Task Execution Order

Work through these tasks **in order**. Each builds on the previous.

---

### Task 1: Clean Up Debug Code and Dead Code

**Priority:** Do first — quick win, removes noise before writing tests.

#### 1a. Remove debug print statements

Replace `print()` calls with proper `logger.warning()` or remove entirely:

| File | Line(s) | Action |
|------|---------|--------|
| `api/routes_premises.py` | ~206 | `print(f"Warning: Failed to create device...")` → `logger.warning(...)` |
| `api/routes_premises.py` | ~271 | `print(f"Warning: Failed to create load...")` → `logger.warning(...)` |
| `api/routes_export.py` | ~572 | `print(f"[EXPORT DEBUG] SKIP room...")` → **delete** |
| `api/routes_export.py` | ~578-579 | `print(f"[EXPORT DEBUG] load...")` → **delete** |
| `api/routes_export.py` | ~588 | `print(f"[EXPORT DEBUG] load...")` → **delete** |
| `api/routes_export.py` | ~929 | `print(f"[EXPORT DEBUG] loads=...")` → **delete** |

**Verify:** `grep -rn "print(" api/ core/ --include="*.py"` should return zero hits (device files may have legitimate print for KNX bus simulation — check context).

#### 1b. Remove dead code in premise_manager.py

In `core/premise_manager.py`, lines ~68-129: everything after `return` on line ~67 is dead code (commented note says "kept for reference"). **Delete the dead block** from the `return` statement through the end of that method's unreachable code. The code was for legacy config-file device import, now handled by the reset-sample endpoint.

#### 1c. Narrow broad exception handlers

These are lower priority but improve debuggability. Change `except Exception` to specific types:

| File | Line | Suggested Type |
|------|------|----------------|
| `knxip/server.py` | ~131 | `except OSError` (socket errors) |
| `api/routes_premises.py` | ~231 | `except sqlite3.IntegrityError` (duplicate key) |
| `api/routes_premises.py` | ~205 | `except (KeyError, ValueError, TypeError) as e` |
| `core/premise.py` | ~265 | `except (KeyError, ValueError)` |
| `api/websocket_hub.py` | ~62 | `except (ConnectionError, RuntimeError)` |
| `templates/loader.py` | ~140 | `except (yaml.YAMLError, OSError, KeyError)` |

**Run after:** `cd sim/knxsim && python -m pytest tests/ -v` — all 107 tests must still pass.

---

### Task 2: Build Test Infrastructure (conftest.py)

**Priority:** Required before any new tests can be written.

**File:** `tests/conftest.py`

Add these fixtures:

```python
"""Pytest configuration and shared fixtures for KNXSim tests."""

from __future__ import annotations

import os
import sys
import tempfile

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
    """Create a fresh SimDatabase with schema applied."""
    from persistence.db import SimDatabase
    database = SimDatabase(tmp_db_path)
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
    db.create_premise({
        "id": premise_id,
        "name": "Test Premise",
        "port": 3671,
    })
    # Add a light switch device
    db.create_device(premise_id, {
        "id": "light-1",
        "type": "light_switch",
        "individual_address": "1.1.1",
        "group_addresses": {"switch": {"ga": "0/0/1", "dpt": "1.001"}},
        "state": {"on": False},
        "initial_state": {"on": False},
    })
    # Add a dimmer device
    db.create_device(premise_id, {
        "id": "dimmer-1",
        "type": "light_dimmer",
        "individual_address": "1.1.2",
        "group_addresses": {
            "switch": {"ga": "0/0/2", "dpt": "1.001"},
            "brightness": {"ga": "0/0/3", "dpt": "5.001"},
        },
        "state": {"on": False, "brightness": 0},
        "initial_state": {"on": False, "brightness": 0},
    })
    return premise_id


@pytest.fixture
def telegram_inspector():
    """Create a TelegramInspector instance."""
    from core.telegram_inspector import TelegramInspector
    return TelegramInspector(max_history=100)


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
def client(app):
    """Create a FastAPI TestClient for HTTP testing."""
    from fastapi.testclient import TestClient
    return TestClient(app)
```

**Notes for Codex:**
- The `create_app()` factory in `api/app.py` accepts `manager`, `ws_hub`, and `telegram_inspector`
- Routes access dependencies via `request.app.state.manager`, `request.app.state.ws_hub`, etc.
- The `TemplateLoader` reads YAML files from `sim/knxsim/templates/` — use real templates, not mocks
- `SimDatabase` auto-creates schema on init — no migration step needed for tests

**Verify:** Create a minimal test that uses the `client` fixture to hit `GET /api/v1/health` and assert `status == "ok"`.

---

### Task 3: Smoke Tests (API Happy Paths)

**Priority:** Highest-value tests — cover the most surface area quickly.

**File:** `tests/test_api_smoke.py`

Write one happy-path test per route group. Each test should:
1. Set up minimal required data (create premise, device, etc.)
2. Call the endpoint
3. Assert HTTP 200/201 and basic response shape

#### Endpoints to cover (minimum):

```
# Premises
POST   /api/v1/premises                    → 201
GET    /api/v1/premises                    → 200, list
GET    /api/v1/premises/{id}               → 200
DELETE /api/v1/premises/{id}               → 200

# Devices
POST   /api/v1/premises/{id}/devices       → 201
GET    /api/v1/premises/{id}/devices       → 200, list
GET    /api/v1/premises/{id}/devices/{did} → 200
PATCH  /api/v1/premises/{id}/devices/{did} → 200
DELETE /api/v1/premises/{id}/devices/{did} → 200

# Floors & Rooms
POST   /api/v1/premises/{id}/floors        → 201
GET    /api/v1/premises/{id}/floors        → 200, list
POST   /api/v1/premises/{id}/rooms         → 201
GET    /api/v1/premises/{id}/rooms         → 200, list

# Loads
POST   /api/v1/premises/{id}/loads         → 201
GET    /api/v1/premises/{id}/loads         → 200, list

# Topology (areas, lines, main groups, middle groups)
POST   /api/v1/premises/{id}/topology/areas        → 201
GET    /api/v1/premises/{id}/topology/areas         → 200, list
POST   /api/v1/premises/{id}/topology/lines         → 201
POST   /api/v1/premises/{id}/topology/main-groups   → 201
POST   /api/v1/premises/{id}/topology/middle-groups → 201

# Groups
POST   /api/v1/premises/{id}/groups/group-addresses → 201
GET    /api/v1/premises/{id}/groups/group-addresses  → 200, list

# Templates
GET    /api/v1/templates                   → 200, list
GET    /api/v1/templates/{id}              → 200

# Telegrams
GET    /api/v1/premises/{id}/telegrams/history → 200
GET    /api/v1/premises/{id}/telegrams/stats   → 200

# Reference data
GET    /api/v1/reference/dpt-types         → 200
GET    /api/v1/reference/device-types      → 200
GET    /api/v1/reference/manufacturers     → 200

# Health
GET    /api/v1/health                      → 200
```

**Pattern for each test:**
```python
def test_create_and_list_premises(client):
    # Create
    resp = client.post("/api/v1/premises", json={"name": "Test", "port": 3672})
    assert resp.status_code == 201
    premise_id = resp.json()["id"]

    # List
    resp = client.get("/api/v1/premises")
    assert resp.status_code == 200
    assert any(p["id"] == premise_id for p in resp.json())
```

**Target:** ~30 tests covering all route groups.

---

### Task 4: Core Module Unit Tests

**Priority:** High — these are the brains of the simulator.

#### 4a. TelegramInspector tests

**File:** `tests/test_telegram_inspector.py`

Test:
- `record()` stores telegrams in ring buffer
- Ring buffer evicts oldest when full (max_history)
- `get_stats()` returns correct tx_count, rx_count, unique_gas, top_gas
- `get_history()` with direction/device/ga filters
- `clear()` resets all counters and history
- Per-premise isolation (premise_a telegrams don't appear in premise_b)

#### 4b. Premise tests

**File:** `tests/test_premise.py`

Test:
- Device creation and retrieval
- Device state updates
- Telegram dispatch (device receives telegram for its GA)
- Scenario start/stop lifecycle
- `get_device_states()` returns all device states

#### 4c. PremiseManager tests

**File:** `tests/test_premise_manager.py`

Test:
- Create premise via manager
- List premises
- Get premise by ID
- Delete premise
- Premise not found returns appropriate error

#### 4d. SimDatabase tests

**File:** `tests/test_database.py`

Test:
- Schema creation on fresh DB
- CRUD for premises, devices, floors, rooms, loads
- CRUD for topology (areas, lines, main groups, middle groups, group addresses)
- Foreign key constraints (delete premise cascades)
- Duplicate ID handling

---

### Task 5: Device Type Unit Tests

**Priority:** Medium — validates state machine logic for each device type.

**File:** `tests/test_devices.py`

For each device type, test the state machine:

| Device | Key Behaviours to Test |
|--------|----------------------|
| `LightSwitch` | on/off toggle, state persistence |
| `LightDimmer` | on/off, brightness 0-100, brightness implies on |
| `Blind` | position 0-100, up/down/stop commands, moving state |
| `Thermostat` | setpoint changes, current temp updates, mode switching |
| `Sensor` | value updates for temperature, humidity, lux |
| `PresenceSensor` | presence on/off, occupancy state |
| `ValveActuator` | valve position 0-100 |
| `TemplateDevice` | dynamic GA mapping from template definition |

**Pattern:**
```python
def test_light_switch_toggle():
    from devices.light_switch import LightSwitch
    dev = LightSwitch(device_id="ls-1", individual_address="1.1.1",
                      group_addresses={"switch": {"ga": "0/0/1", "dpt": "1.001"}},
                      initial_state={"on": False})
    assert dev.state["on"] is False
    dev.handle_group_write("0/0/1", b"\x01")
    assert dev.state["on"] is True
```

**Target:** ~25-30 tests across all device types.

---

### Task 6: Template Loader Tests

**Priority:** Medium — validates YAML template parsing.

**File:** `tests/test_template_loader.py`

Test:
- `load_all()` finds and parses all 57 YAML templates without error
- Each template has required fields: `id`, `name`, `type`, `group_addresses`
- `get_template(id)` returns correct template
- `get_template("nonexistent")` returns None
- Template group addresses use dict-form GAs (not legacy string-form)
- Template categories match directory names

---

### Task 7: Error Path Tests

**Priority:** Lower — but important for robustness.

**File:** `tests/test_api_errors.py`

Test that endpoints return proper error codes:

```
GET    /api/v1/premises/nonexistent           → 404
DELETE /api/v1/premises/nonexistent           → 404
GET    /api/v1/premises/{id}/devices/bad-id   → 404
POST   /api/v1/premises/{id}/devices          → 422 (invalid body)
POST   /api/v1/premises                       → 422 (missing required fields)
PATCH  /api/v1/premises/{id}/devices/{did}    → 404 (device not found)
```

**Target:** ~10-15 tests.

---

## Verification Checklist

After all tasks are complete, run:

```bash
cd sim/knxsim
python -m pytest tests/ -v --tb=short
```

**Expected outcome:**
- All original 107 tests still pass
- New tests bring total to ~200+
- Zero `print()` debug statements remain in `api/` and `core/`
- No dead code blocks remain
- All broad `except Exception` handlers narrowed

**Quick sanity checks:**
```bash
# No debug prints left
grep -rn "print(" api/ core/ --include="*.py" | grep -v "# noqa"

# No dead code after return
grep -n "^        return$" core/premise_manager.py
```

---

## What NOT To Do

- **Don't touch the frontend** (HTML/JS/CSS) — it's complete
- **Don't add new features** — this is consolidation only
- **Don't refactor the SQL in db.py** — the f-string UPDATE pattern is safe (field names come from validated Python code, values use `?` params). Refactoring to a query builder is out of scope
- **Don't add type annotations everywhere** — only add them where they help test clarity
- **Don't create mock KNXnet/IP servers** — UDP testing is out of scope; test at the API and unit level
- **Don't change any existing public API signatures** — tests should validate current behaviour

---

## Dependencies

Ensure these are available in the test environment:
- `pytest` (should already be installed)
- `httpx` (required by FastAPI TestClient)
- `fastapi[all]` (already installed)

If `httpx` is missing: `pip install httpx`

---

## File Summary

| New/Modified File | Purpose |
|-------------------|---------|
| `tests/conftest.py` | **Modify** — add fixtures |
| `tests/test_api_smoke.py` | **New** — API happy-path smoke tests |
| `tests/test_api_errors.py` | **New** — API error response tests |
| `tests/test_telegram_inspector.py` | **New** — TelegramInspector unit tests |
| `tests/test_premise.py` | **New** — Premise lifecycle tests |
| `tests/test_premise_manager.py` | **New** — PremiseManager tests |
| `tests/test_database.py` | **New** — SimDatabase CRUD tests |
| `tests/test_devices.py` | **New** — Device state machine tests |
| `tests/test_template_loader.py` | **New** — Template loading/validation tests |
| `api/routes_export.py` | **Modify** — remove debug prints |
| `api/routes_premises.py` | **Modify** — print→logger, narrow exceptions |
| `core/premise_manager.py` | **Modify** — remove dead code block |
| Various | **Modify** — narrow `except Exception` handlers |
