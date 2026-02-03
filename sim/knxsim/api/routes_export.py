"""KNX Project Export Routes.

Exports KNXSim premise data as a .knxproj file (ZIP archive with XML)
compatible with ETS project imports.
"""

import io
import json as _json
import uuid
import zipfile
from collections import Counter
from datetime import UTC, datetime
from xml.etree import ElementTree as ET

from fastapi import APIRouter
from fastapi.responses import StreamingResponse

router = APIRouter(prefix="/api/v1/premises/{premise_id}/export", tags=["export"])

# KNX XML namespace (using schema version 21 for ETS6 compatibility)
KNX_NS = "http://knx.org/xml/project/21"
NS = {"knx": KNX_NS}


def _ga_to_int(ga_str: str) -> int:
    """Convert group address string '1/2/3' to integer."""
    parts = ga_str.split("/")
    if len(parts) == 3:
        main, middle, sub = int(parts[0]), int(parts[1]), int(parts[2])
        return (main << 11) | (middle << 8) | sub
    elif len(parts) == 2:
        main, sub = int(parts[0]), int(parts[1])
        return (main << 11) | sub
    else:
        return int(parts[0])


def _id_to_display_name(device_id: str) -> str:
    """Convert device ID slug to human-readable display name.

    Examples:
        'living-room-ceiling-light' -> 'Living Room Ceiling Light'
        'ch-3-blinds' -> 'Ch 3 Blinds'
    """
    # Replace hyphens and underscores with spaces
    name = device_id.replace("-", " ").replace("_", " ")
    # Title case each word
    return " ".join(word.capitalize() for word in name.split())


def _ga_name_to_function(ga_name: str) -> str:
    """Convert internal GA name to human-readable function name.

    Examples:
        'switch_cmd' -> 'Switch'
        'brightness_status' -> 'Brightness Status'
        'position_cmd' -> 'Position'
    """
    # Map common suffixes to cleaner names
    name = ga_name.replace("_", " ")
    # Remove 'cmd' suffix as it's implied for commands
    if name.endswith(" cmd"):
        name = name[:-4]
    # Title case
    return " ".join(word.capitalize() for word in name.split())


def _extract_channel_letter(ga_name: str) -> str | None:
    """Extract channel letter from a GA name like 'channel_a_switch' → 'a'."""
    lower = ga_name.lower()
    # "channel_" is 8 chars, so index 8 = letter, index 9 = "_"
    if lower.startswith("channel_") and len(lower) > 9 and lower[9] == "_":
        return lower[8]
    return None


def _normalise_channel_function(ga_name: str) -> str:
    """Strip channel prefix from a GA name and normalise to a standard function.

    Handles both conventions:
      - channel_a_switch / channel_a_switch_status  (config.yaml letter-based)
      - ch1_switch_cmd / ch1_switch_status           (template number-based)
      - button_1 / button_1_led                      (push buttons)

    Returns a normalised function name like 'switch', 'switch_status',
    'brightness', 'valve', 'position', etc.
    """
    lower = ga_name.lower()

    # Strip "channel_X_" prefix (letter-based)
    if lower.startswith("channel_") and len(lower) > 9 and lower[9] == "_":
        lower = lower[10:]  # everything after "channel_X_"

    # Strip "chN_" prefix (number-based, 1-99)
    elif lower.startswith("ch"):
        rest = lower[2:]
        # Find end of digits
        i = 0
        while i < len(rest) and rest[i].isdigit():
            i += 1
        if i > 0 and i < len(rest) and rest[i] == "_":
            lower = rest[i + 1 :]

    # Strip "button_N" → "switch" (push buttons control lights)
    elif lower.startswith("button_"):
        rest = lower[7:]
        # button_N (no suffix) → switch command
        if rest.isdigit():
            return "switch"
        # button_N_led → switch_status
        if "_led" in rest:
            return "switch_status"
        return rest

    # Strip "group_N_" prefix (DALI)
    elif lower.startswith("group_"):
        rest = lower[6:]
        i = 0
        while i < len(rest) and rest[i].isdigit():
            i += 1
        if i > 0 and i < len(rest) and rest[i] == "_":
            lower = rest[i + 1 :]

    # Normalise common suffixes: remove _cmd, keep _status
    # "switch_cmd" → "switch", "switch_status" stays
    if lower.endswith("_cmd"):
        lower = lower[:-4]

    return lower


def _get_channel_gas(device: dict, channel_id: str) -> list[tuple[str, str, str]]:
    """Extract GAs for a specific channel from a multi-channel device.

    Tries both naming conventions:
      - channel_<letter>_* (config.yaml: channel_a_switch, channel_b_valve)
      - ch<number>_*       (templates: ch1_switch_cmd, ch2_valve_status)

    Args:
        device: Device dict with 'group_addresses' field.
        channel_id: Channel identifier, e.g. "A", "B", "1", "2".

    Returns:
        List of (normalised_function, ga_string, dpt) tuples.
    """
    gas = device.get("group_addresses", {})
    results = []

    # Normalise channel_id to both letter and number forms
    ch_id_upper = channel_id.upper()
    ch_letter = ch_id_upper.lower() if ch_id_upper.isalpha() else None
    ch_number = ch_id_upper if ch_id_upper.isdigit() else None

    # Map letter to number and vice versa: A=1, B=2, ...
    if ch_letter and not ch_number:
        ch_number = str(ord(ch_letter) - ord("a") + 1)
    elif ch_number and not ch_letter:
        n = int(ch_number)
        if 1 <= n <= 26:
            ch_letter = chr(ord("a") + n - 1)

    for ga_name, ga_data in gas.items():
        lower_name = ga_name.lower()
        matched = False

        # Try channel_<letter>_ pattern
        if ch_letter and lower_name.startswith(f"channel_{ch_letter}_"):
            matched = True
        # Try ch<number>_ pattern
        elif ch_number and lower_name.startswith(f"ch{ch_number}_"):
            matched = True

        if matched:
            if isinstance(ga_data, dict):
                ga_str = ga_data.get("ga", "")
                ga_dpt = ga_data.get("dpt", "")
            else:
                ga_str = ga_data
                ga_dpt = ""

            if ga_str:
                norm_fn = _normalise_channel_function(ga_name)
                results.append((norm_fn, ga_str, ga_dpt))

    return results


def _load_type_to_function_type(load_type: str) -> tuple[str, str]:
    """Map load type to ETS Function Type for the Trades section.

    Returns (function_type, comment) tuple.
    """
    mapping = {
        "light": ("SwitchableLight", ""),
        "dimmer": ("DimmableLight", ""),
        "blind": ("Sunblind", ""),
        "shutter": ("Sunblind", ""),
        "valve": ("HeatingFloor", ""),
        "motor": ("Sunblind", ""),
        "fan": ("Custom", "fan"),
        "speaker": ("Custom", "speaker"),
    }
    return mapping.get(load_type, ("Custom", load_type))


def _make_id(prefix: str) -> str:
    """Generate a unique ID in ETS format."""
    return f"{prefix}-{uuid.uuid4().hex[:8].upper()}"


def _dpt_to_size(dpt: str) -> str:
    """Convert DPT to approximate size description."""
    if not dpt:
        return "1 Bit"

    main_dpt = dpt.split(".")[0] if "." in dpt else dpt

    # Common DPT size mappings
    sizes = {
        "1": "1 Bit",
        "2": "2 Bit",
        "3": "4 Bit",
        "4": "1 Byte",
        "5": "1 Byte",
        "6": "1 Byte",
        "7": "2 Bytes",
        "8": "2 Bytes",
        "9": "2 Bytes",
        "10": "3 Bytes",
        "11": "3 Bytes",
        "12": "4 Bytes",
        "13": "4 Bytes",
        "14": "4 Bytes",
        "16": "14 Bytes",
        "17": "1 Byte",
        "18": "1 Byte",
        "232": "3 Bytes",
    }
    return sizes.get(main_dpt, "1 Byte")


def _build_knxproj_xml(
    premise: dict, floors: list, devices: list, loads: list | None = None
) -> str:
    """Build the 0.xml content for the .knxproj archive."""

    # Create root element with namespace
    root = ET.Element("KNX", xmlns=KNX_NS)
    root.set("CreatedBy", "KNXSim")
    root.set("ToolVersion", "1.0.0")

    # Project element
    project_id = _make_id("P")
    project = ET.SubElement(root, "Project", Id=project_id)
    project.set("Name", premise.get("name", "KNXSim Export"))

    # ProjectInformation
    info = ET.SubElement(project, "ProjectInformation")
    info.set("Name", premise.get("name", "KNXSim Export"))
    info.set("Comment", f"Exported from KNXSim on {datetime.now(UTC).isoformat()}")

    # Installations
    installations = ET.SubElement(project, "Installations")
    installation = ET.SubElement(installations, "Installation")
    installation.set("Name", premise.get("name", "Installation"))
    installation.set("InstallationId", "0")

    # === Locations (Building Structure) ===
    locations = ET.SubElement(installation, "Locations")
    building = ET.SubElement(locations, "Location")
    building.set("Id", _make_id("L"))
    building.set("Name", premise.get("name", "Building"))
    building.set("Type", "Building")

    # Add floors and rooms
    room_to_location_id = {}
    for floor in floors:
        floor_loc = ET.SubElement(building, "Location")
        floor_loc.set("Id", _make_id("L"))
        floor_loc.set("Name", floor.get("name", floor.get("id", "Floor")))
        floor_loc.set("Type", "Floor")

        for room in floor.get("rooms", []):
            room_loc = ET.SubElement(floor_loc, "Location")
            room_id = _make_id("L")
            room_loc.set("Id", room_id)
            room_loc.set("Name", room.get("name", room.get("id", "Room")))
            room_loc.set("Type", "Room")
            room_to_location_id[room.get("id")] = room_id

    # === Group Addresses ===
    # Build 3-level hierarchy: Domain > Floor > Room
    # This allows the parser to extract room info from the GroupRange path
    group_addresses = ET.SubElement(installation, "GroupAddresses")
    group_ranges = ET.SubElement(group_addresses, "GroupRanges")

    # Map main groups to domain names
    MAIN_GROUP_DOMAINS = {
        "1": "Lighting",
        "2": "Shutters",
        "3": "Climate",
        "4": "Sensors",
        "5": "Scenes",
        "6": "Status",
    }

    # Build room lookup from floors
    room_id_to_info = {}
    for floor in floors:
        floor_name = floor.get("name", floor.get("id", "Floor"))
        for room in floor.get("rooms", []):
            room_id_to_info[room.get("id")] = {
                "name": room.get("name", room.get("id", "Room")),
                "floor": floor_name,
            }

    # Build channel-to-room and channel-to-load-name mappings from loads.
    # Loads tell us which actuator channel controls which room, so we can
    # place each channel's GAs under the correct room in the GroupRange
    # hierarchy instead of lumping them all under the actuator's physical
    # location (e.g. Distribution Board).
    #
    # channel_room_map: {device_id: {channel_letter: room_id}}
    # channel_name_map: {device_id: {channel_letter: load_name}}
    channel_room_map: dict[str, dict[str, str]] = {}
    channel_name_map: dict[str, dict[str, str]] = {}
    if loads:
        for load in loads:
            act_id = load.get("actuator_device_id")
            ch_id = load.get("actuator_channel_id")
            load_room = load.get("room_id")
            load_name = load.get("name", "")
            if act_id and ch_id:
                ch_key = ch_id.lower()
                if load_room:
                    channel_room_map.setdefault(act_id, {})[ch_key] = load_room
                if load_name:
                    channel_name_map.setdefault(act_id, {})[ch_key] = load_name

    # Organize GAs by: main_group > floor > room
    # Structure: {main: {floor: {room: [ga_info, ...]}}}
    ga_hierarchy = {}
    ga_id_map = {}  # Maps GA string to XML ID

    for device in devices:
        device_id = device.get("id", "device")
        device_name = device.get("name") or _id_to_display_name(device_id)
        device_type = device.get("type", "")
        device_room_id = device.get("room_id")
        device_channels = channel_room_map.get(device_id, {})
        device_ch_names = channel_name_map.get(device_id, {})

        # Default floor/room from the device's own placement
        default_room_info = room_id_to_info.get(device_room_id, {})
        default_floor = default_room_info.get("floor", "Unassigned")
        default_room = default_room_info.get("name", "Unassigned")

        for ga_name, ga_data in device.get("group_addresses", {}).items():
            # Handle both old format (string) and new format (dict with ga, dpt, flags)
            if isinstance(ga_data, dict):
                ga_str = ga_data.get("ga", "")
                ga_dpt = ga_data.get("dpt", "")
            else:
                ga_str = ga_data
                ga_dpt = ""

            if ga_str and "/" in ga_str:
                main_group = ga_str.split("/")[0]

                # For multi-channel actuators, resolve channel letter from
                # GA name (e.g. "channel_a_switch" → "a") and look up the
                # room that channel's load belongs to.
                floor_name = default_floor
                room_name = default_room
                ga_display_name = device_name  # default: use device name
                if device_channels:
                    ch_letter = _extract_channel_letter(ga_name)
                    if ch_letter and ch_letter in device_channels:
                        ch_room_id = device_channels[ch_letter]
                        ch_room_info = room_id_to_info.get(ch_room_id, {})
                        if ch_room_info:
                            floor_name = ch_room_info.get("floor", default_floor)
                            room_name = ch_room_info.get("name", default_room)
                        # Use load name for GA display if available
                        if ch_letter in device_ch_names:
                            ga_display_name = device_ch_names[ch_letter]

                # Initialize hierarchy levels
                if main_group not in ga_hierarchy:
                    ga_hierarchy[main_group] = {}
                if floor_name not in ga_hierarchy[main_group]:
                    ga_hierarchy[main_group][floor_name] = {}
                if room_name not in ga_hierarchy[main_group][floor_name]:
                    ga_hierarchy[main_group][floor_name][room_name] = []

                # Use normalised function name for channels, raw for others
                if device_channels and _extract_channel_letter(ga_name):
                    function_name = _ga_name_to_function(_normalise_channel_function(ga_name))
                else:
                    function_name = _ga_name_to_function(ga_name)

                ga_hierarchy[main_group][floor_name][room_name].append(
                    {
                        "address": ga_str,
                        "name": f"{ga_display_name} : {function_name}",
                        "device_id": device_id,
                        "dpt": ga_dpt or _guess_dpt(device_type, ga_name),
                    }
                )

    # Create GroupRange elements with 3-level hierarchy
    for main_group, floors_data in sorted(ga_hierarchy.items()):
        domain_name = MAIN_GROUP_DOMAINS.get(main_group, f"Group {main_group}")
        main_range = ET.SubElement(group_ranges, "GroupRange")
        main_range.set("Id", _make_id("GR"))
        main_range.set("Name", domain_name)
        main_range.set("RangeStart", str(int(main_group) << 11))
        main_range.set("RangeEnd", str(((int(main_group) + 1) << 11) - 1))

        for floor_name, rooms_data in sorted(floors_data.items()):
            floor_range = ET.SubElement(main_range, "GroupRange")
            floor_range.set("Id", _make_id("GR"))
            floor_range.set("Name", floor_name)

            for room_name, gas in sorted(rooms_data.items()):
                room_range = ET.SubElement(floor_range, "GroupRange")
                room_range.set("Id", _make_id("GR"))
                room_range.set("Name", room_name)

                for ga_info in gas:
                    ga_elem = ET.SubElement(room_range, "GroupAddress")
                    ga_id = _make_id("GA")
                    ga_elem.set("Id", ga_id)
                    ga_elem.set("Address", ga_info["address"])
                    ga_elem.set("Name", ga_info["name"])
                    if ga_info.get("dpt"):
                        ga_elem.set("DatapointType", f"DPST-{ga_info['dpt'].replace('.', '-')}")

                    ga_id_map[ga_info["address"]] = ga_id

    # === Topology (Devices with manufacturer metadata) ===
    # Collect unique manufacturers and build device instances
    manufacturers = {}  # {mfr_id: {name, products: {hw_id: {name, app_id, app_name}}}}
    device_counter = 0

    topology = ET.SubElement(installation, "Topology")
    area = ET.SubElement(topology, "Area")
    area.set("Id", _make_id("A"))
    area.set("Address", "1")
    area.set("Name", "Backbone")
    line = ET.SubElement(area, "Line")
    line.set("Id", _make_id("L"))
    line.set("Address", "1")
    line.set("Name", "Main Line")

    for device in devices:
        config = device.get("config") or {}
        mfr_id = config.get("manufacturer_id", "")
        mfr_name = config.get("manufacturer_name", "")
        product_model = config.get("product_model", "")
        app_program = config.get("application_program", "")
        ind_addr = device.get("individual_address", "")

        if not mfr_id:
            # No manufacturer metadata — skip topology entry
            continue

        device_counter += 1
        dev_xml_id = f"D-{device_counter:04d}"
        hw_id = f"{mfr_id}_H-{device_counter:04d}"
        app_id = f"{mfr_id}_A-{device_counter:04d}"
        product_ref_id = f"{hw_id}-HP-0001"

        # Track manufacturer for ManufacturerData section
        if mfr_id not in manufacturers:
            manufacturers[mfr_id] = {
                "name": mfr_name,
                "products": {},
            }
        manufacturers[mfr_id]["products"][hw_id] = {
            "model": product_model,
            "product_ref_id": product_ref_id,
            "app_id": app_id,
            "app_name": app_program,
        }

        # DeviceInstance element
        dev_inst = ET.SubElement(line, "DeviceInstance")
        dev_inst.set("Id", dev_xml_id)
        dev_inst.set("Name", config.get("application_program", device.get("id", "")))
        if ind_addr:
            dev_inst.set("IndividualAddress", ind_addr)
        dev_inst.set("ProductRefId", product_ref_id)
        dev_inst.set("ApplicationProgramRef", app_id)

        # ComObjectInstanceRefs — link device to its group addresses
        com_refs = ET.SubElement(dev_inst, "ComObjectInstanceRefs")
        co_counter = 0
        for ga_name, ga_data in device.get("group_addresses", {}).items():
            ga_str = ga_data.get("ga", "") if isinstance(ga_data, dict) else ga_data
            ga_dpt = ga_data.get("dpt", "") if isinstance(ga_data, dict) else ""
            if ga_str and ga_str in ga_id_map:
                co_counter += 1
                co_ref = ET.SubElement(com_refs, "ComObjectInstanceRef")
                co_ref.set("RefId", f"{dev_xml_id}-CO-{co_counter:03d}")
                if ga_dpt:
                    co_ref.set("DatapointType", f"DPST-{ga_dpt.replace('.', '-')}")
                connectors = ET.SubElement(co_ref, "Connectors")
                send = ET.SubElement(connectors, "Send")
                send.set("GroupAddressRefId", ga_id_map[ga_str])

    # === ManufacturerData ===
    if manufacturers:
        mfr_data = ET.SubElement(root, "ManufacturerData")
        for mfr_id, mfr_info in sorted(manufacturers.items()):
            mfr_elem = ET.SubElement(mfr_data, "Manufacturer")
            mfr_elem.set("Id", mfr_id)
            mfr_elem.set("Name", mfr_info["name"])
            for hw_id, hw_info in mfr_info["products"].items():
                hw_elem = ET.SubElement(mfr_elem, "Hardware")
                hw_elem.set("Id", hw_id)
                hw_elem.set("Name", hw_info["model"])
                product_elem = ET.SubElement(hw_elem, "Product")
                product_elem.set("Id", hw_info["product_ref_id"])
                app_elem = ET.SubElement(mfr_elem, "ApplicationProgram")
                app_elem.set("Id", hw_info["app_id"])
                app_elem.set("Name", hw_info["app_name"])
                app_elem.set("ApplicationVersion", "1")

    # === Trades (Functions) - Maps GAs to rooms with ETS Function Types ===
    #
    # Strategy: Emit one Function per LOAD (not per device) for multi-channel
    # actuators. This makes GLCore create separate devices like "Bedroom Light"
    # instead of one blob "Light Actuator 6ch". Devices not covered by loads
    # (thermostats, sensors, etc.) still get per-device Functions.
    trades = ET.SubElement(installation, "Trades")

    # Build index of devices by ID for fast lookup
    device_by_id = {d.get("id"): d for d in devices}

    # Track which device IDs are covered by loads and which GAs are consumed
    covered_device_ids: set[str] = set()
    covered_gas: set[str] = set()

    if loads:
        # Disambiguate duplicate load names by prepending room name.
        # e.g. five loads all named "UFH Valve" become "Living Room UFH Valve",
        # "Kitchen UFH Valve", etc. — preventing suggested_id collisions in GLCore.
        _load_name_counts = Counter(ld.get("name", "Load") for ld in loads)
        _load_display_names: dict[str, str] = {}  # load_id → unique display name
        for ld in loads:
            lid = ld.get("id", "")
            raw_name = ld.get("name", "Load")
            if _load_name_counts[raw_name] > 1:
                room_info = room_id_to_info.get(ld.get("room_id", ""), {})
                room_label = room_info.get("name", "")
                _load_display_names[lid] = f"{room_label} {raw_name}" if room_label else raw_name
            else:
                _load_display_names[lid] = raw_name

        # First pass: emit per-load Functions and collect covered GAs
        for load in loads:
            act_id = load.get("actuator_device_id")
            ch_id = load.get("actuator_channel_id")
            load_room = load.get("room_id")
            load_name = _load_display_names.get(load.get("id", ""), load.get("name", "Load"))
            load_type = load.get("type", "")

            if not act_id or not ch_id or not load_room:
                continue
            if load_room not in room_to_location_id:
                print(
                    f"[EXPORT DEBUG] SKIP room {load_room} not in {list(room_to_location_id.keys())}"
                )
                continue

            actuator = device_by_id.get(act_id)
            print(
                f"[EXPORT DEBUG] load {load.get('id')}: act_id={act_id} found={actuator is not None}"
            )
            if not actuator:
                continue

            covered_device_ids.add(act_id)

            # Get the channel's GAs with normalised function names
            channel_gas = _get_channel_gas(actuator, ch_id)
            print(f"[EXPORT DEBUG] load {load.get('id')}: ch_id={ch_id}, channel_gas={channel_gas}")
            if not channel_gas:
                continue

            # Track all GAs consumed by loads
            for _, ga_str, _ in channel_gas:
                covered_gas.add(ga_str)

            # Emit Function for this load
            func_type, comment = _load_type_to_function_type(load_type)

            trade = ET.SubElement(trades, "Trade")
            trade.set("Id", _make_id("T"))
            trade.set("Name", load_name)

            func = ET.SubElement(trade, "Function")
            func.set("Id", _make_id("F"))
            func.set("Name", load_name)
            func.set("Type", func_type)
            if comment:
                func.set("Comment", comment)

            loc_ref = ET.SubElement(func, "LocationReference")
            loc_ref.set("RefId", room_to_location_id[load_room])

            ga_refs = ET.SubElement(func, "GroupAddressRefs")
            for norm_fn, ga_str, _ in channel_gas:
                if ga_str in ga_id_map:
                    ga_ref = ET.SubElement(ga_refs, "GroupAddressRef")
                    ga_ref.set("RefId", ga_id_map[ga_str])
                    ga_ref.set("Name", norm_fn)

    # Pass 1.5: emit infrastructure Functions for covered actuators.
    # These appear in the Distribution Board room and carry channel metadata
    # in the Comment field so GLCore can build the admin actuator display.
    if loads and covered_device_ids:
        # Build per-actuator channel info from loads
        actuator_channels: dict[str, dict[str, dict]] = {}  # {act_id: {ch: info}}
        for load in loads:
            act_id = load.get("actuator_device_id")
            ch_id = load.get("actuator_channel_id", "").upper()
            if act_id and ch_id:
                load_id = load.get("id", "")
                actuator_channels.setdefault(act_id, {})[ch_id] = {
                    "load_name": _load_display_names.get(load_id, load.get("name")),
                    "load_type": load.get("type"),
                    "room": load.get("room_id"),
                }

        for device_id in covered_device_ids:
            actuator = device_by_id.get(device_id)
            if not actuator:
                continue

            act_room = actuator.get("room_id", "distribution-board")
            if act_room not in room_to_location_id:
                continue

            act_name = actuator.get("name") or _id_to_display_name(device_id)
            ch_info = actuator_channels.get(device_id, {})

            # Build channel metadata with GAs for each channel
            channels_meta: dict[str, dict] = {}
            gas = actuator.get("group_addresses", {})
            # Discover all channels present in the GA names
            seen_channels: set[str] = set()
            for ga_name in gas:
                ch_letter = _extract_channel_letter(ga_name)
                if ch_letter:
                    seen_channels.add(ch_letter.upper())
            # Also include channels from loads
            for ch in ch_info:
                seen_channels.add(ch)

            for ch_letter in sorted(seen_channels):
                channel_gas = _get_channel_gas(actuator, ch_letter)
                ga_map = {}
                for norm_fn, ga_str, _ in channel_gas:
                    ga_map[norm_fn] = ga_str
                load_info = ch_info.get(ch_letter, {})
                channels_meta[ch_letter] = {
                    "load_name": load_info.get("load_name"),
                    "load_type": load_info.get("load_type"),
                    "room": load_info.get("room"),
                    "gas": ga_map,
                }

            comment_data = {
                "infrastructure": True,
                "channels": channels_meta,
            }

            trade = ET.SubElement(trades, "Trade")
            trade.set("Id", _make_id("T"))
            trade.set("Name", act_name)

            func = ET.SubElement(trade, "Function")
            func.set("Id", _make_id("F"))
            func.set("Name", act_name)
            func.set("Type", "Custom")
            func.set("Comment", _json.dumps(comment_data, separators=(",", ":")))

            loc_ref = ET.SubElement(func, "LocationReference")
            loc_ref.set("RefId", room_to_location_id[act_room])

            # Reference ALL the actuator's GAs
            ga_refs = ET.SubElement(func, "GroupAddressRefs")
            for ga_name, ga_data in gas.items():
                ga_str = ga_data.get("ga", "") if isinstance(ga_data, dict) else ga_data
                if ga_str in ga_id_map:
                    norm_fn = _normalise_channel_function(ga_name)
                    ch_letter = _extract_channel_letter(ga_name)
                    # Prefix with channel letter to avoid key collisions
                    ref_name = f"ch_{ch_letter}_{norm_fn}" if ch_letter else ga_name
                    ga_ref = ET.SubElement(ga_refs, "GroupAddressRef")
                    ga_ref.set("RefId", ga_id_map[ga_str])
                    ga_ref.set("Name", ref_name)

    # Second pass: emit per-device Functions for non-covered devices
    for device in devices:
        device_id = device.get("id", "")

        # Skip devices whose channels are represented by loads
        if device_id in covered_device_ids:
            continue

        room_id = device.get("room_id")
        if not room_id or room_id not in room_to_location_id:
            continue

        # Check if ALL this device's GAs are already covered by loads
        # (e.g., a wall switch that shares GAs with a light load)
        device_gas = set()
        for ga_data in device.get("group_addresses", {}).values():
            ga_str = ga_data.get("ga", "") if isinstance(ga_data, dict) else ga_data
            if ga_str:
                device_gas.add(ga_str)
        if device_gas and device_gas.issubset(covered_gas):
            continue  # All GAs covered — skip this device

        device_name = device.get("name") or _id_to_display_name(device_id)
        dev_config = device.get("config") or {}
        template_id = dev_config.get("template_id", device.get("type", ""))

        func_type, comment = _device_type_to_function_type(template_id)

        trade = ET.SubElement(trades, "Trade")
        trade.set("Id", _make_id("T"))
        trade.set("Name", device_name)

        func = ET.SubElement(trade, "Function")
        func.set("Id", _make_id("F"))
        func.set("Name", device_name)
        func.set("Type", func_type)
        if comment:
            func.set("Comment", comment)

        loc_ref = ET.SubElement(func, "LocationReference")
        loc_ref.set("RefId", room_to_location_id[room_id])

        ga_refs = ET.SubElement(func, "GroupAddressRefs")
        for ga_name, ga_data in device.get("group_addresses", {}).items():
            ga_str = ga_data.get("ga", "") if isinstance(ga_data, dict) else ga_data
            if ga_str in ga_id_map:
                ga_ref = ET.SubElement(ga_refs, "GroupAddressRef")
                ga_ref.set("RefId", ga_id_map[ga_str])
                ga_ref.set("Name", ga_name)

    # Convert to string with declaration
    ET.indent(root, space="  ")
    xml_str = ET.tostring(root, encoding="unicode", xml_declaration=False)
    return f'<?xml version="1.0" encoding="utf-8"?>\n{xml_str}'


def _guess_dpt(device_type: str, ga_name: str) -> str:
    """Guess DPT based on device type and GA function name."""
    ga_lower = ga_name.lower()

    # Switch/status commands
    if "switch" in ga_lower or "on_off" in ga_lower:
        return "1.001"
    if "status" in ga_lower and "switch" in ga_lower:
        return "1.001"

    # Dimming
    if "brightness" in ga_lower or "dimm" in ga_lower or "level" in ga_lower:
        return "5.001"
    if "dimming" in ga_lower:
        return "3.007"

    # Blinds/Shutters
    if "position" in ga_lower:
        return "5.001"
    if "slat" in ga_lower or "tilt" in ga_lower or "angle" in ga_lower:
        return "5.001"
    if "move" in ga_lower or "up_down" in ga_lower:
        return "1.008"
    if "stop" in ga_lower:
        return "1.010"

    # Sensors
    if "temperature" in ga_lower or "temp" in ga_lower:
        return "9.001"
    if "humidity" in ga_lower:
        return "9.007"
    if "lux" in ga_lower or "brightness" in ga_lower:
        return "9.004"
    if "co2" in ga_lower:
        return "9.008"
    if "presence" in ga_lower or "motion" in ga_lower:
        return "1.018"

    # Climate / Heating
    if "valve" in ga_lower or "heating" in ga_lower:
        return "5.001"
    if "setpoint" in ga_lower:
        return "9.001"
    if "mode" in ga_lower and "hvac" in ga_lower:
        return "20.102"

    # Default based on device type
    type_defaults = {
        "light_switch": "1.001",
        "light_dimmer": "5.001",
        "blind": "5.001",
        "sensor": "9.001",
        "presence": "1.018",
        "thermostat": "9.001",
    }
    return type_defaults.get(device_type, "1.001")


def _device_type_to_function_type(device_type: str) -> tuple[str, str]:
    """Map device type to standard ETS Function Type and Comment.

    Returns (function_type, comment) tuple. Standard ETS types include
    SwitchableLight, DimmableLight, Sunblind, HeatingRadiator, etc.
    Non-standard types use 'Custom' with a Comment carrying the template ID.
    """
    standard = {
        # Lighting
        "light_switch": "SwitchableLight",
        "light": "SwitchableLight",
        "switch_actuator_4ch": "SwitchableLight",
        "switch_actuator_6fold": "SwitchableLight",
        "switch_actuator_8ch": "SwitchableLight",
        "switch_actuator_12ch": "SwitchableLight",
        "light_dimmer": "DimmableLight",
        "dimmer_actuator_4ch": "DimmableLight",
        "light_rgb": "DimmableLight",
        "light_colour_temp": "DimmableLight",
        "dali_gateway": "DimmableLight",
        # Blinds
        "blind_position": "Sunblind",
        "blind_position_slat": "Sunblind",
        "shutter_actuator_4ch": "Sunblind",
        "shutter_actuator_8ch": "Sunblind",
        "awning_controller": "Sunblind",
        # Climate
        "thermostat": "HeatingRadiator",
        "heating_actuator_6ch": "HeatingFloor",
        "heating_actuator_6fold": "HeatingFloor",
        "fan_coil_controller": "HeatingContinuousVariable",
        "hvac_unit": "HeatingContinuousVariable",
        "air_handling_unit": "HeatingContinuousVariable",
    }
    if device_type in standard:
        return standard[device_type], ""

    # Everything else → Custom with template ID as comment hint
    return "Custom", device_type


@router.get("/knxproj")
async def export_knxproj(premise_id: str):
    """Export premise as .knxproj file (ETS-compatible ZIP archive)."""

    manager = router.app.state.manager

    # Get premise info
    premise = manager.get_premise(premise_id)
    if not premise:
        return {"error": "Premise not found"}

    # Get floors with rooms
    floors = manager.list_floors(premise_id)

    # Get all devices
    devices = manager.list_devices(premise_id)

    # Get loads (for channel-to-room mapping of multi-channel actuators)
    loads = manager.db.list_loads(premise_id)
    print(f"[EXPORT DEBUG] loads={len(loads)}, first={loads[0] if loads else None}")

    # Build XML content
    xml_content = _build_knxproj_xml(premise, floors, devices, loads)

    # Create ZIP archive in memory
    zip_buffer = io.BytesIO()
    project_folder = f"P-{uuid.uuid4().hex[:4].upper()}"

    with zipfile.ZipFile(zip_buffer, "w", zipfile.ZIP_DEFLATED) as zf:
        # Main project XML
        zf.writestr(f"{project_folder}/0.xml", xml_content)

        # Minimal project info file (some parsers expect this)
        project_info = f"""<?xml version="1.0" encoding="utf-8"?>
<KNX xmlns="{KNX_NS}">
  <Project Id="{project_folder}">
    <ProjectInformation Name="{premise.get("name", "KNXSim Export")}" />
  </Project>
</KNX>"""
        zf.writestr(f"{project_folder}/project.xml", project_info)

    zip_buffer.seek(0)

    # Generate filename
    safe_name = "".join(
        c if c.isalnum() or c in "-_" else "_" for c in premise.get("name", "export")
    )
    filename = f"{safe_name}_{datetime.now().strftime('%Y%m%d_%H%M%S')}.knxproj"

    return StreamingResponse(
        zip_buffer,
        media_type="application/zip",
        headers={"Content-Disposition": f'attachment; filename="{filename}"'},
    )


@router.get("/esf")
async def export_esf(premise_id: str):
    """Export premise group addresses as ESF (ETS Symbol File) format."""

    manager = router.app.state.manager

    # Get premise info
    premise = manager.get_premise(premise_id)
    if not premise:
        return {"error": "Premise not found"}

    # Get all devices
    devices = manager.list_devices(premise_id)

    # Build ESF content (tab-separated)
    lines = []
    lines.append(
        '"Group Name"\t"Address"\t"Central"\t"Unfiltered"\t"Description"\t"DatapointType"\t"Security"'
    )

    for device in devices:
        for ga_name, ga_str in device.get("group_addresses", {}).items():
            if ga_str:
                name = f"{device.get('id', 'Device')}/{ga_name}"
                dpt = _guess_dpt(device.get("type"), ga_name)
                # ESF format: "Name" "Address" "Central" "Unfiltered" "Description" "DPT" "Security"
                lines.append(
                    f'"{name}"\t"{ga_str}"\t""\t""\t""\t"DPST-{dpt.replace(".", "-")}"\t""'
                )

    content = "\n".join(lines)

    safe_name = "".join(
        c if c.isalnum() or c in "-_" else "_" for c in premise.get("name", "export")
    )
    filename = f"{safe_name}_{datetime.now().strftime('%Y%m%d_%H%M%S')}.esf"

    return StreamingResponse(
        io.BytesIO(content.encode("utf-8")),
        media_type="text/plain",
        headers={"Content-Disposition": f'attachment; filename="{filename}"'},
    )
