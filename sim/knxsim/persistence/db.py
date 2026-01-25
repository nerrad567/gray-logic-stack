"""SQLite persistence layer for KNX simulator management.

Stores premises, floors, rooms, devices, and scenarios.
Uses synchronous sqlite3 (run in thread pool from async context).
"""

import json
import logging
import os
import sqlite3
from datetime import datetime, timezone
from typing import Optional

logger = logging.getLogger("knxsim.persistence")

SCHEMA_SQL = """
CREATE TABLE IF NOT EXISTS premises (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    gateway_address TEXT NOT NULL,
    client_address TEXT NOT NULL,
    port INTEGER NOT NULL UNIQUE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS floors (
    id TEXT PRIMARY KEY,
    premise_id TEXT NOT NULL REFERENCES premises(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS rooms (
    id TEXT PRIMARY KEY,
    floor_id TEXT NOT NULL REFERENCES floors(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    grid_col INTEGER NOT NULL DEFAULT 0,
    grid_row INTEGER NOT NULL DEFAULT 0,
    grid_width INTEGER NOT NULL DEFAULT 1,
    grid_height INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    premise_id TEXT NOT NULL REFERENCES premises(id) ON DELETE CASCADE,
    room_id TEXT REFERENCES rooms(id) ON DELETE SET NULL,
    type TEXT NOT NULL,
    individual_address TEXT NOT NULL,
    group_addresses JSON NOT NULL,
    state JSON NOT NULL DEFAULT '{}',
    initial_state JSON NOT NULL DEFAULT '{}',
    config JSON,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS scenarios (
    id TEXT PRIMARY KEY,
    premise_id TEXT NOT NULL REFERENCES premises(id) ON DELETE CASCADE,
    device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    field TEXT NOT NULL,
    type TEXT NOT NULL,
    params JSON NOT NULL DEFAULT '{}',
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
"""


def _now() -> str:
    return datetime.now(timezone.utc).isoformat()


class Database:
    """Synchronous SQLite database for simulator persistence."""

    def __init__(self, db_path: str = "/app/data/knxsim.db"):
        os.makedirs(os.path.dirname(db_path), exist_ok=True)
        self.db_path = db_path
        self._conn: Optional[sqlite3.Connection] = None

    def connect(self):
        """Open the database and apply schema."""
        self._conn = sqlite3.connect(self.db_path, check_same_thread=False)
        self._conn.row_factory = sqlite3.Row
        self._conn.execute("PRAGMA journal_mode=WAL")
        self._conn.execute("PRAGMA foreign_keys=ON")
        self._conn.executescript(SCHEMA_SQL)
        self._conn.commit()
        logger.info("Database opened: %s", self.db_path)

    def close(self):
        if self._conn:
            self._conn.close()
            self._conn = None

    @property
    def conn(self) -> sqlite3.Connection:
        if not self._conn:
            raise RuntimeError("Database not connected")
        return self._conn

    # ------------------------------------------------------------------
    # Premises
    # ------------------------------------------------------------------

    def list_premises(self) -> list[dict]:
        rows = self.conn.execute("SELECT * FROM premises ORDER BY name").fetchall()
        return [dict(r) for r in rows]

    def get_premise(self, premise_id: str) -> Optional[dict]:
        row = self.conn.execute(
            "SELECT * FROM premises WHERE id = ?", (premise_id,)
        ).fetchone()
        return dict(row) if row else None

    def create_premise(self, data: dict) -> dict:
        now = _now()
        self.conn.execute(
            """INSERT INTO premises (id, name, gateway_address, client_address, port, created_at, updated_at)
               VALUES (?, ?, ?, ?, ?, ?, ?)""",
            (
                data["id"],
                data["name"],
                data["gateway_address"],
                data["client_address"],
                data["port"],
                now,
                now,
            ),
        )
        self.conn.commit()
        return self.get_premise(data["id"])

    def delete_premise(self, premise_id: str) -> bool:
        cur = self.conn.execute("DELETE FROM premises WHERE id = ?", (premise_id,))
        self.conn.commit()
        return cur.rowcount > 0

    # ------------------------------------------------------------------
    # Floors
    # ------------------------------------------------------------------

    def list_floors(self, premise_id: str) -> list[dict]:
        rows = self.conn.execute(
            "SELECT * FROM floors WHERE premise_id = ? ORDER BY sort_order",
            (premise_id,),
        ).fetchall()
        return [dict(r) for r in rows]

    def get_floor(self, floor_id: str) -> Optional[dict]:
        row = self.conn.execute(
            "SELECT * FROM floors WHERE id = ?", (floor_id,)
        ).fetchone()
        return dict(row) if row else None

    def create_floor(self, premise_id: str, data: dict) -> dict:
        self.conn.execute(
            """INSERT INTO floors (id, premise_id, name, sort_order)
               VALUES (?, ?, ?, ?)""",
            (data["id"], premise_id, data["name"], data.get("sort_order", 0)),
        )
        self.conn.commit()
        return self.get_floor(data["id"])

    def update_floor(self, floor_id: str, data: dict) -> Optional[dict]:
        sets = []
        vals = []
        for key in ("name", "sort_order"):
            if key in data:
                sets.append(f"{key} = ?")
                vals.append(data[key])
        if not sets:
            return self.get_floor(floor_id)
        vals.append(floor_id)
        self.conn.execute(f"UPDATE floors SET {', '.join(sets)} WHERE id = ?", vals)
        self.conn.commit()
        return self.get_floor(floor_id)

    def delete_floor(self, floor_id: str) -> bool:
        cur = self.conn.execute("DELETE FROM floors WHERE id = ?", (floor_id,))
        self.conn.commit()
        return cur.rowcount > 0

    # ------------------------------------------------------------------
    # Rooms
    # ------------------------------------------------------------------

    def list_rooms(self, floor_id: str) -> list[dict]:
        rows = self.conn.execute(
            "SELECT * FROM rooms WHERE floor_id = ? ORDER BY grid_row, grid_col",
            (floor_id,),
        ).fetchall()
        return [dict(r) for r in rows]

    def list_rooms_by_premise(self, premise_id: str) -> list[dict]:
        rows = self.conn.execute(
            """SELECT r.* FROM rooms r
               JOIN floors f ON r.floor_id = f.id
               WHERE f.premise_id = ?
               ORDER BY f.sort_order, r.grid_row, r.grid_col""",
            (premise_id,),
        ).fetchall()
        return [dict(r) for r in rows]

    def get_room(self, room_id: str) -> Optional[dict]:
        row = self.conn.execute(
            "SELECT * FROM rooms WHERE id = ?", (room_id,)
        ).fetchone()
        return dict(row) if row else None

    def create_room(self, floor_id: str, data: dict) -> dict:
        self.conn.execute(
            """INSERT INTO rooms (id, floor_id, name, grid_col, grid_row, grid_width, grid_height)
               VALUES (?, ?, ?, ?, ?, ?, ?)""",
            (
                data["id"],
                floor_id,
                data["name"],
                data.get("grid_col", 0),
                data.get("grid_row", 0),
                data.get("grid_width", 1),
                data.get("grid_height", 1),
            ),
        )
        self.conn.commit()
        return self.get_room(data["id"])

    def update_room(self, room_id: str, data: dict) -> Optional[dict]:
        sets = []
        vals = []
        for key in ("name", "grid_col", "grid_row", "grid_width", "grid_height"):
            if key in data:
                sets.append(f"{key} = ?")
                vals.append(data[key])
        if not sets:
            return self.get_room(room_id)
        vals.append(room_id)
        self.conn.execute(f"UPDATE rooms SET {', '.join(sets)} WHERE id = ?", vals)
        self.conn.commit()
        return self.get_room(room_id)

    def delete_room(self, room_id: str) -> bool:
        cur = self.conn.execute("DELETE FROM rooms WHERE id = ?", (room_id,))
        self.conn.commit()
        return cur.rowcount > 0

    # ------------------------------------------------------------------
    # Devices
    # ------------------------------------------------------------------

    def list_devices(self, premise_id: str) -> list[dict]:
        rows = self.conn.execute(
            "SELECT * FROM devices WHERE premise_id = ? ORDER BY id",
            (premise_id,),
        ).fetchall()
        return [_parse_device_row(r) for r in rows]

    def get_device(self, device_id: str) -> Optional[dict]:
        row = self.conn.execute(
            "SELECT * FROM devices WHERE id = ?", (device_id,)
        ).fetchone()
        return _parse_device_row(row) if row else None

    def create_device(self, premise_id: str, data: dict) -> dict:
        now = _now()
        self.conn.execute(
            """INSERT INTO devices (id, premise_id, room_id, type, individual_address,
               group_addresses, state, initial_state, config, created_at, updated_at)
               VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)""",
            (
                data["id"],
                premise_id,
                data.get("room_id"),
                data["type"],
                data["individual_address"],
                json.dumps(data.get("group_addresses", {})),
                json.dumps(data.get("state", data.get("initial_state", {}))),
                json.dumps(data.get("initial_state", {})),
                json.dumps(data.get("config")) if data.get("config") else None,
                now,
                now,
            ),
        )
        self.conn.commit()
        return self.get_device(data["id"])

    def update_device(self, device_id: str, data: dict) -> Optional[dict]:
        sets = []
        vals = []
        now = _now()
        for key in ("room_id", "type", "individual_address"):
            if key in data:
                sets.append(f"{key} = ?")
                vals.append(data[key])
        for json_key in ("group_addresses", "state", "initial_state", "config"):
            if json_key in data:
                sets.append(f"{json_key} = ?")
                vals.append(json.dumps(data[json_key]))
        if not sets:
            return self.get_device(device_id)
        sets.append("updated_at = ?")
        vals.append(now)
        vals.append(device_id)
        self.conn.execute(f"UPDATE devices SET {', '.join(sets)} WHERE id = ?", vals)
        self.conn.commit()
        return self.get_device(device_id)

    def update_device_state(self, device_id: str, state: dict):
        """Update just the state field (called frequently by scenarios/telegrams)."""
        self.conn.execute(
            "UPDATE devices SET state = ?, updated_at = ? WHERE id = ?",
            (json.dumps(state), _now(), device_id),
        )
        self.conn.commit()

    def delete_device(self, device_id: str) -> bool:
        cur = self.conn.execute("DELETE FROM devices WHERE id = ?", (device_id,))
        self.conn.commit()
        return cur.rowcount > 0

    # ------------------------------------------------------------------
    # Scenarios
    # ------------------------------------------------------------------

    def list_scenarios(self, premise_id: str) -> list[dict]:
        rows = self.conn.execute(
            "SELECT * FROM scenarios WHERE premise_id = ? ORDER BY device_id, field",
            (premise_id,),
        ).fetchall()
        return [_parse_scenario_row(r) for r in rows]

    def create_scenario(self, premise_id: str, data: dict) -> dict:
        now = _now()
        self.conn.execute(
            """INSERT INTO scenarios (id, premise_id, device_id, field, type, params, enabled, created_at, updated_at)
               VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)""",
            (
                data["id"],
                premise_id,
                data["device_id"],
                data["field"],
                data["type"],
                json.dumps(data.get("params", {})),
                1 if data.get("enabled", True) else 0,
                now,
                now,
            ),
        )
        self.conn.commit()
        row = self.conn.execute(
            "SELECT * FROM scenarios WHERE id = ?", (data["id"],)
        ).fetchone()
        return _parse_scenario_row(row)

    def delete_scenario(self, scenario_id: str) -> bool:
        cur = self.conn.execute("DELETE FROM scenarios WHERE id = ?", (scenario_id,))
        self.conn.commit()
        return cur.rowcount > 0


def _parse_device_row(row) -> dict:
    """Parse a device row, deserializing JSON fields."""
    if row is None:
        return None
    d = dict(row)
    for key in ("group_addresses", "state", "initial_state", "config"):
        if d.get(key) and isinstance(d[key], str):
            d[key] = json.loads(d[key])
    return d


def _parse_scenario_row(row) -> dict:
    """Parse a scenario row, deserializing JSON fields."""
    if row is None:
        return None
    d = dict(row)
    if d.get("params") and isinstance(d["params"], str):
        d["params"] = json.loads(d["params"])
    d["enabled"] = bool(d.get("enabled", 1))
    return d
