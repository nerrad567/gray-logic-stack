"""SQLite persistence layer for KNX simulator management.

Stores premises, floors, rooms, devices, and scenarios.
Uses synchronous sqlite3 (run in thread pool from async context).
"""

import json
import logging
import os
import sqlite3
from datetime import UTC, datetime

logger = logging.getLogger("knxsim.persistence")

SCHEMA_SQL = """
CREATE TABLE IF NOT EXISTS premises (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    gateway_address TEXT NOT NULL,
    client_address TEXT NOT NULL,
    port INTEGER NOT NULL UNIQUE,
    setup_complete INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Topology: Areas (0-15 per premise)
CREATE TABLE IF NOT EXISTS areas (
    id TEXT PRIMARY KEY,
    premise_id TEXT NOT NULL REFERENCES premises(id) ON DELETE CASCADE,
    area_number INTEGER NOT NULL CHECK (area_number >= 0 AND area_number <= 15),
    name TEXT NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(premise_id, area_number)
);

-- Topology: Lines (0-15 per area)
CREATE TABLE IF NOT EXISTS lines (
    id TEXT PRIMARY KEY,
    area_id TEXT NOT NULL REFERENCES areas(id) ON DELETE CASCADE,
    line_number INTEGER NOT NULL CHECK (line_number >= 0 AND line_number <= 15),
    name TEXT NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(area_id, line_number)
);

-- Building view: Floors
CREATE TABLE IF NOT EXISTS floors (
    id TEXT PRIMARY KEY,
    premise_id TEXT NOT NULL REFERENCES premises(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);

-- Building view: Rooms
CREATE TABLE IF NOT EXISTS rooms (
    id TEXT PRIMARY KEY,
    floor_id TEXT NOT NULL REFERENCES floors(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    room_type TEXT NOT NULL DEFAULT 'other',
    grid_col INTEGER NOT NULL DEFAULT 0,
    grid_row INTEGER NOT NULL DEFAULT 0,
    grid_width INTEGER NOT NULL DEFAULT 1,
    grid_height INTEGER NOT NULL DEFAULT 1
);

-- Devices with optional topology and building placement
-- Note: line_id, device_number, channels added via migration for existing DBs
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
    channels JSON,
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

-- Group Address hierarchy: Main Groups (0-31, typically by function)
CREATE TABLE IF NOT EXISTS main_groups (
    id TEXT PRIMARY KEY,
    premise_id TEXT NOT NULL REFERENCES premises(id) ON DELETE CASCADE,
    group_number INTEGER NOT NULL CHECK (group_number >= 0 AND group_number <= 31),
    name TEXT NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(premise_id, group_number)
);

-- Group Address hierarchy: Middle Groups (0-7, typically by location/floor)
CREATE TABLE IF NOT EXISTS middle_groups (
    id TEXT PRIMARY KEY,
    main_group_id TEXT NOT NULL REFERENCES main_groups(id) ON DELETE CASCADE,
    group_number INTEGER NOT NULL CHECK (group_number >= 0 AND group_number <= 7),
    name TEXT NOT NULL,
    description TEXT,
    floor_id TEXT REFERENCES floors(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(main_group_id, group_number)
);

-- Index for topology queries (line_id index created in migration after column exists)
CREATE INDEX IF NOT EXISTS idx_lines_area ON lines(area_id);
CREATE INDEX IF NOT EXISTS idx_areas_premise ON areas(premise_id);
CREATE INDEX IF NOT EXISTS idx_main_groups_premise ON main_groups(premise_id);
CREATE INDEX IF NOT EXISTS idx_middle_groups_main ON middle_groups(main_group_id);
CREATE INDEX IF NOT EXISTS idx_middle_groups_floor ON middle_groups(floor_id);
"""

# Migrations for existing databases
MIGRATIONS_SQL = """
-- Add room_type column if it doesn't exist (v2026.01.25)
ALTER TABLE rooms ADD COLUMN room_type TEXT NOT NULL DEFAULT 'other';
"""


def _now() -> str:
    return datetime.now(UTC).isoformat()


def _guess_dpt_for_ga(device_type: str, ga_name: str) -> str:
    """Guess DPT based on device type and GA function name."""
    ga_lower = ga_name.lower()

    # Switch/status commands
    if "switch" in ga_lower or "on_off" in ga_lower:
        return "1.001"
    if "move" in ga_lower:
        return "1.008"
    if "stop" in ga_lower:
        return "1.017"
    if "presence" in ga_lower or "occupancy" in ga_lower:
        return "1.018"

    # Percentage values
    if "brightness" in ga_lower or "position" in ga_lower or "slat" in ga_lower:
        return "5.001"

    # Dimming
    if "dim" in ga_lower:
        return "3.007"

    # Temperature
    if "temp" in ga_lower or "setpoint" in ga_lower:
        return "9.001"

    # Humidity
    if "humidity" in ga_lower:
        return "9.007"

    # Lux
    if "lux" in ga_lower or "light_level" in ga_lower:
        return "9.004"

    # HVAC mode
    if "mode" in ga_lower and ("hvac" in device_type.lower() or "thermostat" in device_type.lower()):
        return "20.102"

    # Default to switch for unknown
    return "1.001"


def _guess_flags_for_ga(ga_name: str) -> str:
    """Guess communication flags based on GA function name."""
    ga_lower = ga_name.lower()

    # Status objects are typically read + transmit
    if "status" in ga_lower:
        return "CRT"

    # Command objects are typically write + update
    if any(cmd in ga_lower for cmd in ["switch", "brightness", "position", "move", "stop", "setpoint", "mode"]):
        return "CWU"

    # Sensor outputs are read + transmit
    if any(sensor in ga_lower for sensor in ["temperature", "humidity", "lux", "presence", "co2"]):
        return "CRT"

    # Default to write
    return "CW"


def _default_channel_name(device_type: str) -> str:
    """Generate default channel name based on device type."""
    type_lower = device_type.lower()

    if "light" in type_lower or "dimmer" in type_lower:
        return "Light Output"
    if "blind" in type_lower or "shutter" in type_lower:
        return "Blind Output"
    if "switch" in type_lower and "actuator" in type_lower:
        return "Switch Output"
    if "sensor" in type_lower or "presence" in type_lower:
        return "Sensor"
    if "thermostat" in type_lower:
        return "Thermostat"
    if "button" in type_lower:
        return "Button"
    if "input" in type_lower:
        return "Input"

    return "Channel A"


# Multi-channel device templates
# Defines how many channels each multi-channel device type has and their group objects
MULTI_CHANNEL_TEMPLATES = {
    "switch_actuator_2fold": {
        "channel_count": 2,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CWU"},
            {"name": "switch_status", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"on": False},
    },
    "switch_actuator_4fold": {
        "channel_count": 4,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CWU"},
            {"name": "switch_status", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"on": False},
    },
    "switch_actuator_8fold": {
        "channel_count": 8,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CWU"},
            {"name": "switch_status", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"on": False},
    },
    "dimmer_actuator_2fold": {
        "channel_count": 2,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CWU"},
            {"name": "switch_status", "dpt": "1.001", "flags": "CRT"},
            {"name": "brightness", "dpt": "5.001", "flags": "CWU"},
            {"name": "brightness_status", "dpt": "5.001", "flags": "CRT"},
        ],
        "state_fields": {"on": False, "brightness": 0},
    },
    "dimmer_actuator_4fold": {
        "channel_count": 4,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CWU"},
            {"name": "switch_status", "dpt": "1.001", "flags": "CRT"},
            {"name": "brightness", "dpt": "5.001", "flags": "CWU"},
            {"name": "brightness_status", "dpt": "5.001", "flags": "CRT"},
        ],
        "state_fields": {"on": False, "brightness": 0},
    },
    "blind_actuator_2fold": {
        "channel_count": 2,
        "group_objects": [
            {"name": "move", "dpt": "1.008", "flags": "CW"},
            {"name": "stop", "dpt": "1.017", "flags": "CW"},
            {"name": "position", "dpt": "5.001", "flags": "CWU"},
            {"name": "position_status", "dpt": "5.001", "flags": "CRT"},
        ],
        "state_fields": {"position": 0, "moving": False},
    },
    "blind_actuator_4fold": {
        "channel_count": 4,
        "group_objects": [
            {"name": "move", "dpt": "1.008", "flags": "CW"},
            {"name": "stop", "dpt": "1.017", "flags": "CW"},
            {"name": "position", "dpt": "5.001", "flags": "CWU"},
            {"name": "position_status", "dpt": "5.001", "flags": "CRT"},
        ],
        "state_fields": {"position": 0, "moving": False},
    },
    "push_button_2fold": {
        "channel_count": 2,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"pressed": False},
    },
    "push_button_4fold": {
        "channel_count": 4,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"pressed": False},
    },
    "push_button_6fold": {
        "channel_count": 6,
        "group_objects": [
            {"name": "switch", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"pressed": False},
    },
    "binary_input_4fold": {
        "channel_count": 4,
        "group_objects": [
            {"name": "state", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"active": False},
    },
    "binary_input_8fold": {
        "channel_count": 8,
        "group_objects": [
            {"name": "state", "dpt": "1.001", "flags": "CRT"},
        ],
        "state_fields": {"active": False},
    },
}

CHANNEL_LABELS = ["A", "B", "C", "D", "E", "F", "G", "H"]


def _generate_channels_from_template(device_type: str, group_addresses: dict = None) -> list[dict]:
    """Generate channel structure from multi-channel template.
    
    If group_addresses is provided (from config.yaml), distributes them across channels.
    Otherwise creates channels with empty GA assignments.
    """
    template = MULTI_CHANNEL_TEMPLATES.get(device_type)
    if not template:
        return None  # Not a multi-channel type
    
    channel_count = template["channel_count"]
    channels = []
    
    for i in range(channel_count):
        label = CHANNEL_LABELS[i] if i < len(CHANNEL_LABELS) else str(i + 1)
        
        # Build group_objects for this channel
        group_objects = {}
        for go in template["group_objects"]:
            # Check if we have a GA for this channel's group object
            # Convention: channel_A_switch, channel_B_switch OR button_1, button_2
            ga = None
            if group_addresses:
                # Try various naming conventions
                for key_pattern in [
                    f"channel_{label.lower()}_{go['name']}",
                    f"{label.lower()}_{go['name']}",
                    f"{go['name']}_{label.lower()}",
                    f"{go['name']}_{i+1}",
                    f"button_{i+1}" if go['name'] == 'switch' else None,
                    f"led_{i+1}" if go['name'] == 'switch_status' else None,
                ]:
                    if key_pattern and key_pattern in group_addresses:
                        ga = group_addresses[key_pattern]
                        break
            
            group_objects[go["name"]] = {
                "ga": ga,
                "dpt": go["dpt"],
                "flags": go["flags"],
            }
        
        channels.append({
            "id": label,
            "name": f"Channel {label}",
            "group_objects": group_objects,
            "state": dict(template.get("state_fields", {})),
            "initial_state": dict(template.get("state_fields", {})),
            "parameters": {},
        })
    
    return channels


class Database:
    """Synchronous SQLite database for simulator persistence."""

    def __init__(self, db_path: str = "/app/data/knxsim.db"):
        os.makedirs(os.path.dirname(db_path), exist_ok=True)
        self.db_path = db_path
        self._conn: sqlite3.Connection | None = None

    def connect(self):
        """Open the database and apply schema."""
        self._conn = sqlite3.connect(self.db_path, check_same_thread=False)
        self._conn.row_factory = sqlite3.Row
        self._conn.execute("PRAGMA journal_mode=WAL")
        self._conn.execute("PRAGMA foreign_keys=ON")
        self._conn.executescript(SCHEMA_SQL)
        self._conn.commit()
        self._apply_migrations()
        logger.info("Database opened: %s", self.db_path)

    def _apply_migrations(self):
        """Apply schema migrations for existing databases."""
        # Check if room_type column exists
        cursor = self.conn.execute("PRAGMA table_info(rooms)")
        columns = [row[1] for row in cursor.fetchall()]

        if "room_type" not in columns:
            logger.info("Applying migration: adding room_type column to rooms")
            self.conn.execute(
                "ALTER TABLE rooms ADD COLUMN room_type TEXT NOT NULL DEFAULT 'other'"
            )
            self.conn.commit()

        # Check if line_id column exists in devices (topology migration)
        cursor = self.conn.execute("PRAGMA table_info(devices)")
        device_columns = [row[1] for row in cursor.fetchall()]

        if "line_id" not in device_columns:
            logger.info("Applying migration: adding topology columns to devices")
            self.conn.execute("ALTER TABLE devices ADD COLUMN line_id TEXT")
            self.conn.execute("ALTER TABLE devices ADD COLUMN device_number INTEGER")
            # Create index for topology queries
            self.conn.execute("CREATE INDEX IF NOT EXISTS idx_devices_line ON devices(line_id)")
            self.conn.commit()

            # Migrate existing devices: parse their individual_address to create topology
            self._migrate_devices_to_topology()

        # Check if channels column exists in devices (channel support migration)
        cursor = self.conn.execute("PRAGMA table_info(devices)")
        device_columns = [row[1] for row in cursor.fetchall()]

        if "channels" not in device_columns:
            logger.info("Applying migration: adding channels column to devices")
            self.conn.execute("ALTER TABLE devices ADD COLUMN channels JSON")
            self.conn.commit()
            # Migrate existing devices to single-channel format
            self._migrate_devices_to_channels()

        # Check if setup_complete column exists in premises
        cursor = self.conn.execute("PRAGMA table_info(premises)")
        premise_columns = [row[1] for row in cursor.fetchall()]

        if "setup_complete" not in premise_columns:
            logger.info("Applying migration: adding setup_complete column to premises")
            # Existing premises are considered setup complete (they already have devices)
            self.conn.execute(
                "ALTER TABLE premises ADD COLUMN setup_complete INTEGER NOT NULL DEFAULT 1"
            )
            self.conn.commit()

    def close(self):
        if self._conn:
            self._conn.close()
            self._conn = None

    @property
    def conn(self) -> sqlite3.Connection:
        if not self._conn:
            raise RuntimeError("Database not connected")
        return self._conn

    def _migrate_devices_to_topology(self):
        """Migrate existing devices to topology structure.

        Parses individual_address, creates Areas/Lines as needed,
        and links devices to their topology location.
        """
        # Get all premises
        premises = self.list_premises()
        for premise in premises:
            premise_id = premise["id"]
            devices = self.conn.execute(
                "SELECT id, individual_address FROM devices WHERE premise_id = ?",
                (premise_id,),
            ).fetchall()

            for device in devices:
                device_id = device["id"]
                ia = device["individual_address"]
                if not ia:
                    continue

                parts = ia.split(".")
                if len(parts) != 3:
                    continue

                try:
                    area_num = int(parts[0])
                    line_num = int(parts[1])
                    device_num = int(parts[2])
                except ValueError:
                    continue

                # Find or create area
                area = self.conn.execute(
                    "SELECT id FROM areas WHERE premise_id = ? AND area_number = ?",
                    (premise_id, area_num),
                ).fetchone()

                if not area:
                    area_id = f"area-{premise_id}-{area_num}"
                    now = _now()
                    self.conn.execute(
                        """INSERT INTO areas (id, premise_id, area_number, name, created_at, updated_at)
                           VALUES (?, ?, ?, ?, ?, ?)""",
                        (area_id, premise_id, area_num, f"Area {area_num}", now, now),
                    )
                    logger.info("Migration: created Area %d for premise %s", area_num, premise_id)
                else:
                    area_id = area["id"]

                # Find or create line
                line = self.conn.execute(
                    "SELECT id FROM lines WHERE area_id = ? AND line_number = ?",
                    (area_id, line_num),
                ).fetchone()

                if not line:
                    line_id = f"line-{area_id}-{line_num}"
                    now = _now()
                    self.conn.execute(
                        """INSERT INTO lines (id, area_id, line_number, name, created_at, updated_at)
                           VALUES (?, ?, ?, ?, ?, ?)""",
                        (line_id, area_id, line_num, f"Line {area_num}.{line_num}", now, now),
                    )
                    logger.info("Migration: created Line %d.%d", area_num, line_num)
                else:
                    line_id = line["id"]

                # Update device with topology reference
                self.conn.execute(
                    "UPDATE devices SET line_id = ?, device_number = ? WHERE id = ?",
                    (line_id, device_num, device_id),
                )

            self.conn.commit()
            logger.info("Migration: linked %d devices to topology in premise %s", len(devices), premise_id)

    def _migrate_devices_to_channels(self):
        """Migrate existing devices to channel-based structure.

        Converts flat group_addresses and state into a single channel.
        This preserves backward compatibility while enabling multi-channel devices.
        """
        # Get all devices that don't have channels set
        devices = self.conn.execute(
            "SELECT id, type, group_addresses, state, initial_state FROM devices WHERE channels IS NULL"
        ).fetchall()

        migrated = 0
        for device in devices:
            device_id = device["id"]
            device_type = device["type"]

            # Parse existing JSON fields
            group_addresses = device["group_addresses"]
            if isinstance(group_addresses, str):
                group_addresses = json.loads(group_addresses)

            state = device["state"]
            if isinstance(state, str):
                state = json.loads(state)

            initial_state = device["initial_state"]
            if isinstance(initial_state, str):
                initial_state = json.loads(initial_state)

            # Convert flat group_addresses to group_objects format
            # Old format: {"switch": "1/0/1", "brightness": "1/0/2"}
            # New format: {"switch": {"ga": "1/0/1", "dpt": "1.001", "flags": "CW"}}
            group_objects = {}
            for ga_name, ga_value in (group_addresses or {}).items():
                if isinstance(ga_value, str):
                    # Simple string GA - convert to object
                    group_objects[ga_name] = {
                        "ga": ga_value,
                        "dpt": _guess_dpt_for_ga(device_type, ga_name),
                        "flags": _guess_flags_for_ga(ga_name),
                    }
                elif isinstance(ga_value, dict):
                    # Already in object format (might have been manually set)
                    group_objects[ga_name] = ga_value
                # Skip None/invalid values

            # Create single channel
            channel = {
                "id": "A",
                "name": _default_channel_name(device_type),
                "group_objects": group_objects,
                "state": state or {},
                "initial_state": initial_state or {},
                "parameters": {},
            }

            channels = [channel]

            # Update device
            self.conn.execute(
                "UPDATE devices SET channels = ? WHERE id = ?",
                (json.dumps(channels), device_id),
            )
            migrated += 1

        if migrated > 0:
            self.conn.commit()
            logger.info("Migration: converted %d devices to channel format", migrated)

    # ------------------------------------------------------------------
    # Premises
    # ------------------------------------------------------------------

    def list_premises(self) -> list[dict]:
        rows = self.conn.execute("SELECT * FROM premises ORDER BY name").fetchall()
        return [dict(r) for r in rows]

    def get_premise(self, premise_id: str) -> dict | None:
        row = self.conn.execute(
            "SELECT * FROM premises WHERE id = ?", (premise_id,)
        ).fetchone()
        return dict(row) if row else None

    def create_premise(self, data: dict) -> dict:
        now = _now()
        self.conn.execute(
            """INSERT INTO premises (id, name, gateway_address, client_address, port, setup_complete, created_at, updated_at)
               VALUES (?, ?, ?, ?, ?, ?, ?, ?)""",
            (
                data["id"],
                data["name"],
                data["gateway_address"],
                data["client_address"],
                data["port"],
                1 if data.get("setup_complete") else 0,
                now,
                now,
            ),
        )
        self.conn.commit()
        return self.get_premise(data["id"])

    def mark_premise_setup_complete(self, premise_id: str) -> bool:
        """Mark a premise as setup complete (user made a choice)."""
        now = _now()
        cur = self.conn.execute(
            "UPDATE premises SET setup_complete = 1, updated_at = ? WHERE id = ?",
            (now, premise_id),
        )
        self.conn.commit()
        return cur.rowcount > 0

    def delete_premise(self, premise_id: str) -> bool:
        cur = self.conn.execute("DELETE FROM premises WHERE id = ?", (premise_id,))
        self.conn.commit()
        return cur.rowcount > 0

    # ------------------------------------------------------------------
    # Areas (Topology)
    # ------------------------------------------------------------------

    def list_areas(self, premise_id: str) -> list[dict]:
        rows = self.conn.execute(
            "SELECT * FROM areas WHERE premise_id = ? ORDER BY area_number",
            (premise_id,),
        ).fetchall()
        return [dict(r) for r in rows]

    def get_area(self, area_id: str) -> dict | None:
        row = self.conn.execute(
            "SELECT * FROM areas WHERE id = ?", (area_id,)
        ).fetchone()
        return dict(row) if row else None

    def get_area_by_number(self, premise_id: str, area_number: int) -> dict | None:
        row = self.conn.execute(
            "SELECT * FROM areas WHERE premise_id = ? AND area_number = ?",
            (premise_id, area_number),
        ).fetchone()
        return dict(row) if row else None

    def create_area(self, premise_id: str, data: dict) -> dict:
        now = _now()
        area_id = data.get("id") or f"area-{premise_id}-{data['area_number']}"
        self.conn.execute(
            """INSERT INTO areas (id, premise_id, area_number, name, description, created_at, updated_at)
               VALUES (?, ?, ?, ?, ?, ?, ?)""",
            (
                area_id,
                premise_id,
                data["area_number"],
                data["name"],
                data.get("description"),
                now,
                now,
            ),
        )
        self.conn.commit()
        return self.get_area(area_id)

    def update_area(self, area_id: str, data: dict) -> dict | None:
        sets = []
        vals = []
        now = _now()
        for key in ("name", "description"):
            if key in data:
                sets.append(f"{key} = ?")
                vals.append(data[key])
        if not sets:
            return self.get_area(area_id)
        sets.append("updated_at = ?")
        vals.append(now)
        vals.append(area_id)
        self.conn.execute(f"UPDATE areas SET {', '.join(sets)} WHERE id = ?", vals)
        self.conn.commit()
        return self.get_area(area_id)

    def delete_area(self, area_id: str) -> bool:
        cur = self.conn.execute("DELETE FROM areas WHERE id = ?", (area_id,))
        self.conn.commit()
        return cur.rowcount > 0

    def get_or_create_area(self, premise_id: str, area_number: int, name: str = None) -> dict:
        """Get area by number, creating it if it doesn't exist."""
        area = self.get_area_by_number(premise_id, area_number)
        if area:
            return area
        return self.create_area(premise_id, {
            "area_number": area_number,
            "name": name or f"Area {area_number}",
        })

    # ------------------------------------------------------------------
    # Lines (Topology)
    # ------------------------------------------------------------------

    def list_lines(self, area_id: str) -> list[dict]:
        rows = self.conn.execute(
            "SELECT * FROM lines WHERE area_id = ? ORDER BY line_number",
            (area_id,),
        ).fetchall()
        return [dict(r) for r in rows]

    def list_lines_by_premise(self, premise_id: str) -> list[dict]:
        """List all lines in a premise, across all areas."""
        rows = self.conn.execute(
            """SELECT l.*, a.area_number, a.name as area_name
               FROM lines l
               JOIN areas a ON l.area_id = a.id
               WHERE a.premise_id = ?
               ORDER BY a.area_number, l.line_number""",
            (premise_id,),
        ).fetchall()
        return [dict(r) for r in rows]

    def get_line(self, line_id: str) -> dict | None:
        row = self.conn.execute(
            "SELECT * FROM lines WHERE id = ?", (line_id,)
        ).fetchone()
        return dict(row) if row else None

    def get_line_by_number(self, area_id: str, line_number: int) -> dict | None:
        row = self.conn.execute(
            "SELECT * FROM lines WHERE area_id = ? AND line_number = ?",
            (area_id, line_number),
        ).fetchone()
        return dict(row) if row else None

    def create_line(self, area_id: str, data: dict) -> dict:
        now = _now()
        line_id = data.get("id") or f"line-{area_id}-{data['line_number']}"
        self.conn.execute(
            """INSERT INTO lines (id, area_id, line_number, name, description, created_at, updated_at)
               VALUES (?, ?, ?, ?, ?, ?, ?)""",
            (
                line_id,
                area_id,
                data["line_number"],
                data["name"],
                data.get("description"),
                now,
                now,
            ),
        )
        self.conn.commit()
        return self.get_line(line_id)

    def update_line(self, line_id: str, data: dict) -> dict | None:
        sets = []
        vals = []
        now = _now()
        for key in ("name", "description"):
            if key in data:
                sets.append(f"{key} = ?")
                vals.append(data[key])
        if not sets:
            return self.get_line(line_id)
        sets.append("updated_at = ?")
        vals.append(now)
        vals.append(line_id)
        self.conn.execute(f"UPDATE lines SET {', '.join(sets)} WHERE id = ?", vals)
        self.conn.commit()
        return self.get_line(line_id)

    def delete_line(self, line_id: str) -> bool:
        cur = self.conn.execute("DELETE FROM lines WHERE id = ?", (line_id,))
        self.conn.commit()
        return cur.rowcount > 0

    def get_or_create_line(self, area_id: str, line_number: int, name: str = None) -> dict:
        """Get line by number, creating it if it doesn't exist."""
        line = self.get_line_by_number(area_id, line_number)
        if line:
            return line
        # Get area info for naming
        area = self.get_area(area_id)
        area_num = area["area_number"] if area else 0
        return self.create_line(area_id, {
            "line_number": line_number,
            "name": name or f"Line {area_num}.{line_number}",
        })

    # ------------------------------------------------------------------
    # Topology helpers
    # ------------------------------------------------------------------

    def get_topology(self, premise_id: str) -> dict:
        """Get full topology tree for a premise."""
        areas = self.list_areas(premise_id)
        for area in areas:
            area["lines"] = self.list_lines(area["id"])
            for line in area["lines"]:
                # Get devices on this line
                devices = self.conn.execute(
                    "SELECT * FROM devices WHERE line_id = ? ORDER BY device_number",
                    (line["id"],),
                ).fetchall()
                line["devices"] = [_parse_device_row(d) for d in devices]
        return {"areas": areas}

    def ensure_topology_for_ia(self, premise_id: str, individual_address: str) -> tuple[str, int] | None:
        """Ensure Area/Line exist for an individual address, return (line_id, device_number).

        Creates Area and Line if they don't exist. Returns None if IA is invalid.
        """
        if not individual_address:
            return None

        parts = individual_address.split(".")
        if len(parts) != 3:
            return None

        try:
            area_num = int(parts[0])
            line_num = int(parts[1])
            device_num = int(parts[2])
        except ValueError:
            return None

        if not (0 <= area_num <= 15 and 0 <= line_num <= 15 and 0 <= device_num <= 255):
            return None

        # Get or create area
        area = self.get_or_create_area(premise_id, area_num)
        # Get or create line
        line = self.get_or_create_line(area["id"], line_num)

        return (line["id"], device_num)

    # ------------------------------------------------------------------
    # Main Groups (Group Address hierarchy)
    # ------------------------------------------------------------------

    def list_main_groups(self, premise_id: str) -> list[dict]:
        rows = self.conn.execute(
            "SELECT * FROM main_groups WHERE premise_id = ? ORDER BY group_number",
            (premise_id,),
        ).fetchall()
        return [dict(r) for r in rows]

    def get_main_group(self, main_group_id: str) -> dict | None:
        row = self.conn.execute(
            "SELECT * FROM main_groups WHERE id = ?", (main_group_id,)
        ).fetchone()
        return dict(row) if row else None

    def get_main_group_by_number(self, premise_id: str, group_number: int) -> dict | None:
        row = self.conn.execute(
            "SELECT * FROM main_groups WHERE premise_id = ? AND group_number = ?",
            (premise_id, group_number),
        ).fetchone()
        return dict(row) if row else None

    def create_main_group(self, premise_id: str, data: dict) -> dict:
        now = _now()
        group_id = data.get("id") or f"main-group-{premise_id}-{data['group_number']}"
        self.conn.execute(
            """INSERT INTO main_groups (id, premise_id, group_number, name, description, created_at, updated_at)
               VALUES (?, ?, ?, ?, ?, ?, ?)""",
            (
                group_id,
                premise_id,
                data["group_number"],
                data["name"],
                data.get("description"),
                now,
                now,
            ),
        )
        self.conn.commit()
        return self.get_main_group(group_id)

    def update_main_group(self, main_group_id: str, data: dict) -> dict | None:
        sets = []
        vals = []
        now = _now()
        for key in ("name", "description"):
            if key in data:
                sets.append(f"{key} = ?")
                vals.append(data[key])
        if not sets:
            return self.get_main_group(main_group_id)
        sets.append("updated_at = ?")
        vals.append(now)
        vals.append(main_group_id)
        self.conn.execute(f"UPDATE main_groups SET {', '.join(sets)} WHERE id = ?", vals)
        self.conn.commit()
        return self.get_main_group(main_group_id)

    def delete_main_group(self, main_group_id: str) -> bool:
        cur = self.conn.execute("DELETE FROM main_groups WHERE id = ?", (main_group_id,))
        self.conn.commit()
        return cur.rowcount > 0

    def get_or_create_main_group(self, premise_id: str, group_number: int, name: str = None) -> dict:
        """Get main group by number, creating it if it doesn't exist."""
        group = self.get_main_group_by_number(premise_id, group_number)
        if group:
            return group
        return self.create_main_group(premise_id, {
            "group_number": group_number,
            "name": name or f"Main Group {group_number}",
        })

    # ------------------------------------------------------------------
    # Middle Groups (Group Address hierarchy)
    # ------------------------------------------------------------------

    def list_middle_groups(self, main_group_id: str) -> list[dict]:
        rows = self.conn.execute(
            "SELECT * FROM middle_groups WHERE main_group_id = ? ORDER BY group_number",
            (main_group_id,),
        ).fetchall()
        return [dict(r) for r in rows]

    def list_middle_groups_by_premise(self, premise_id: str) -> list[dict]:
        """List all middle groups in a premise, across all main groups."""
        rows = self.conn.execute(
            """SELECT mg.*, m.group_number as main_group_number, m.name as main_group_name
               FROM middle_groups mg
               JOIN main_groups m ON mg.main_group_id = m.id
               WHERE m.premise_id = ?
               ORDER BY m.group_number, mg.group_number""",
            (premise_id,),
        ).fetchall()
        return [dict(r) for r in rows]

    def get_middle_group(self, middle_group_id: str) -> dict | None:
        row = self.conn.execute(
            "SELECT * FROM middle_groups WHERE id = ?", (middle_group_id,)
        ).fetchone()
        return dict(row) if row else None

    def get_middle_group_by_number(self, main_group_id: str, group_number: int) -> dict | None:
        row = self.conn.execute(
            "SELECT * FROM middle_groups WHERE main_group_id = ? AND group_number = ?",
            (main_group_id, group_number),
        ).fetchone()
        return dict(row) if row else None

    def get_middle_group_by_floor(self, main_group_id: str, floor_id: str) -> dict | None:
        """Find a middle group linked to a specific floor."""
        row = self.conn.execute(
            "SELECT * FROM middle_groups WHERE main_group_id = ? AND floor_id = ?",
            (main_group_id, floor_id),
        ).fetchone()
        return dict(row) if row else None

    def create_middle_group(self, main_group_id: str, data: dict) -> dict:
        now = _now()
        group_id = data.get("id") or f"middle-group-{main_group_id}-{data['group_number']}"
        self.conn.execute(
            """INSERT INTO middle_groups (id, main_group_id, group_number, name, description, floor_id, created_at, updated_at)
               VALUES (?, ?, ?, ?, ?, ?, ?, ?)""",
            (
                group_id,
                main_group_id,
                data["group_number"],
                data["name"],
                data.get("description"),
                data.get("floor_id"),
                now,
                now,
            ),
        )
        self.conn.commit()
        return self.get_middle_group(group_id)

    def update_middle_group(self, middle_group_id: str, data: dict) -> dict | None:
        sets = []
        vals = []
        now = _now()
        for key in ("name", "description", "floor_id"):
            if key in data:
                sets.append(f"{key} = ?")
                vals.append(data[key])
        if not sets:
            return self.get_middle_group(middle_group_id)
        sets.append("updated_at = ?")
        vals.append(now)
        vals.append(middle_group_id)
        self.conn.execute(f"UPDATE middle_groups SET {', '.join(sets)} WHERE id = ?", vals)
        self.conn.commit()
        return self.get_middle_group(middle_group_id)

    def delete_middle_group(self, middle_group_id: str) -> bool:
        cur = self.conn.execute("DELETE FROM middle_groups WHERE id = ?", (middle_group_id,))
        self.conn.commit()
        return cur.rowcount > 0

    def get_or_create_middle_group(self, main_group_id: str, group_number: int, name: str = None, floor_id: str = None) -> dict:
        """Get middle group by number, creating it if it doesn't exist."""
        group = self.get_middle_group_by_number(main_group_id, group_number)
        if group:
            return group
        return self.create_middle_group(main_group_id, {
            "group_number": group_number,
            "name": name or f"Middle Group {group_number}",
            "floor_id": floor_id,
        })

    # ------------------------------------------------------------------
    # Group Address helpers
    # ------------------------------------------------------------------

    def get_group_address_tree(self, premise_id: str) -> dict:
        """Get full GA hierarchy tree for a premise."""
        main_groups = self.list_main_groups(premise_id)
        for mg in main_groups:
            mg["middle_groups"] = self.list_middle_groups(mg["id"])
        return {"main_groups": main_groups}

    def get_used_group_addresses(self, premise_id: str) -> list[str]:
        """Get all group addresses currently in use by devices."""
        devices = self.list_devices(premise_id)
        used_gas = set()
        for device in devices:
            gas = device.get("group_addresses", {})
            for ga in gas.values():
                if isinstance(ga, str):
                    used_gas.add(ga)
                elif isinstance(ga, list):
                    used_gas.update(ga)
        return sorted(used_gas)

    def suggest_next_ga(self, premise_id: str, main_group: int, middle_group: int) -> str:
        """Suggest the next available sub-group address for a main/middle group."""
        used_gas = self.get_used_group_addresses(premise_id)
        prefix = f"{main_group}/{middle_group}/"

        # Find used sub-groups in this main/middle
        used_subs = set()
        for ga in used_gas:
            if ga.startswith(prefix):
                try:
                    sub = int(ga.split("/")[2])
                    used_subs.add(sub)
                except (IndexError, ValueError):
                    pass

        # Find next available (start from 0)
        for sub in range(256):
            if sub not in used_subs:
                return f"{main_group}/{middle_group}/{sub}"

        return f"{main_group}/{middle_group}/0"  # Fallback

    # ------------------------------------------------------------------
    # Floors
    # ------------------------------------------------------------------

    def list_floors(self, premise_id: str) -> list[dict]:
        rows = self.conn.execute(
            "SELECT * FROM floors WHERE premise_id = ? ORDER BY sort_order",
            (premise_id,),
        ).fetchall()
        return [dict(r) for r in rows]

    def get_floor(self, floor_id: str) -> dict | None:
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

    def update_floor(self, floor_id: str, data: dict) -> dict | None:
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

    def get_room(self, room_id: str) -> dict | None:
        row = self.conn.execute(
            "SELECT * FROM rooms WHERE id = ?", (room_id,)
        ).fetchone()
        return dict(row) if row else None

    def create_room(self, floor_id: str, data: dict) -> dict:
        self.conn.execute(
            """INSERT INTO rooms (id, floor_id, name, room_type, grid_col, grid_row, grid_width, grid_height)
               VALUES (?, ?, ?, ?, ?, ?, ?, ?)""",
            (
                data["id"],
                floor_id,
                data["name"],
                data.get("room_type", "other"),
                data.get("grid_col", 0),
                data.get("grid_row", 0),
                data.get("grid_width", 1),
                data.get("grid_height", 1),
            ),
        )
        self.conn.commit()
        return self.get_room(data["id"])

    def update_room(self, room_id: str, data: dict) -> dict | None:
        sets = []
        vals = []
        for key in (
            "name",
            "room_type",
            "grid_col",
            "grid_row",
            "grid_width",
            "grid_height",
        ):
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

    def get_device(self, device_id: str) -> dict | None:
        row = self.conn.execute(
            "SELECT * FROM devices WHERE id = ?", (device_id,)
        ).fetchone()
        return _parse_device_row(row) if row else None

    def create_device(self, premise_id: str, data: dict) -> dict:
        now = _now()

        # Handle topology: either line_id+device_number provided, or parse from individual_address
        line_id = data.get("line_id")
        device_number = data.get("device_number")
        individual_address = data.get("individual_address", "")

        if not line_id and individual_address:
            # Auto-create topology from individual address
            topology = self.ensure_topology_for_ia(premise_id, individual_address)
            if topology:
                line_id, device_number = topology

        # Handle channels: if provided, use them; otherwise auto-generate
        channels = data.get("channels")
        if channels is None:
            device_type = data["type"]
            group_addresses = data.get("group_addresses", {})
            state = data.get("state", data.get("initial_state", {}))
            initial_state = data.get("initial_state", {})

            # Check if this is a multi-channel device type
            multi_channels = _generate_channels_from_template(device_type, group_addresses)
            
            if multi_channels:
                # Use auto-generated multi-channel structure
                channels = multi_channels
            else:
                # Single-channel device - create default channel from legacy fields
                # Convert group_addresses to group_objects format
                group_objects = {}
                for ga_name, ga_value in group_addresses.items():
                    if isinstance(ga_value, str):
                        group_objects[ga_name] = {
                            "ga": ga_value,
                            "dpt": _guess_dpt_for_ga(device_type, ga_name),
                            "flags": _guess_flags_for_ga(ga_name),
                        }
                    elif isinstance(ga_value, dict):
                        group_objects[ga_name] = ga_value

                channels = [{
                    "id": "A",
                    "name": _default_channel_name(device_type),
                    "group_objects": group_objects,
                    "state": state,
                    "initial_state": initial_state,
                    "parameters": {},
                }]

        self.conn.execute(
            """INSERT INTO devices (id, premise_id, line_id, device_number, room_id, type, individual_address,
               group_addresses, state, initial_state, config, channels, created_at, updated_at)
               VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)""",
            (
                data["id"],
                premise_id,
                line_id,
                device_number,
                data.get("room_id"),
                data["type"],
                individual_address,
                json.dumps(data.get("group_addresses", {})),
                json.dumps(data.get("state", data.get("initial_state", {}))),
                json.dumps(data.get("initial_state", {})),
                json.dumps(data.get("config")) if data.get("config") else None,
                json.dumps(channels),
                now,
                now,
            ),
        )
        self.conn.commit()
        return self.get_device(data["id"])

    def update_device(self, device_id: str, data: dict) -> dict | None:
        sets = []
        vals = []
        now = _now()

        # Handle topology fields
        for key in ("room_id", "type", "individual_address", "line_id", "device_number"):
            if key in data:
                sets.append(f"{key} = ?")
                vals.append(data[key])

        # If individual_address changed, update topology
        if "individual_address" in data and "line_id" not in data:
            device = self.get_device(device_id)
            if device:
                topology = self.ensure_topology_for_ia(device["premise_id"], data["individual_address"])
                if topology:
                    sets.append("line_id = ?")
                    vals.append(topology[0])
                    sets.append("device_number = ?")
                    vals.append(topology[1])

        for json_key in ("group_addresses", "state", "initial_state", "config", "channels"):
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
        """Update device state and sync to channels.
        
        For single-channel devices, updates channels[0].state.
        For multi-channel devices, updates all channels with matching state keys.
        Called frequently by scenarios/telegrams.
        """
        now = _now()
        
        # Get current device to access channels
        device = self.get_device(device_id)
        if device and device.get("channels"):
            channels = device["channels"]
            if len(channels) == 1:
                # Single channel - sync all state
                channels[0]["state"] = state
            else:
                # Multi-channel - sync matching state keys to each channel
                # This is a basic approach; full implementation would route by GA
                for channel in channels:
                    channel_state = channel.get("state", {})
                    for key, value in state.items():
                        if key in channel_state:
                            channel_state[key] = value
                    channel["state"] = channel_state
            
            # Update both state and channels
            self.conn.execute(
                "UPDATE devices SET state = ?, channels = ?, updated_at = ? WHERE id = ?",
                (json.dumps(state), json.dumps(channels), now, device_id),
            )
        else:
            # No channels - just update state
            self.conn.execute(
                "UPDATE devices SET state = ?, updated_at = ? WHERE id = ?",
                (json.dumps(state), now, device_id),
            )
        self.conn.commit()

    def find_device_channel_by_ga(self, premise_id: str, ga: str) -> tuple[dict, dict, str] | None:
        """Find device and channel that owns a group address.
        
        Returns (device, channel, group_object_name) or None if not found.
        Searches through all devices' channels' group_objects.
        """
        devices = self.list_devices(premise_id)
        for device in devices:
            channels = device.get("channels", [])
            for channel in channels:
                group_objects = channel.get("group_objects", {})
                for go_name, go_data in group_objects.items():
                    if isinstance(go_data, dict) and go_data.get("ga") == ga:
                        return (device, channel, go_name)
        return None

    def update_channel_state(self, device_id: str, channel_id: str, state_updates: dict):
        """Update a specific channel's state within a device.
        
        Args:
            device_id: Device ID
            channel_id: Channel ID (e.g., "A", "B")
            state_updates: Dict of state keys to update
        """
        device = self.get_device(device_id)
        if not device or not device.get("channels"):
            return
        
        channels = device["channels"]
        updated = False
        
        for channel in channels:
            if channel.get("id") == channel_id:
                channel_state = channel.get("state", {})
                channel_state.update(state_updates)
                channel["state"] = channel_state
                updated = True
                break
        
        if updated:
            # Also update top-level state for compatibility
            device_state = device.get("state", {})
            device_state.update(state_updates)
            
            self.conn.execute(
                "UPDATE devices SET state = ?, channels = ?, updated_at = ? WHERE id = ?",
                (json.dumps(device_state), json.dumps(channels), _now(), device_id),
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
    for key in ("group_addresses", "state", "initial_state", "config", "channels"):
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
