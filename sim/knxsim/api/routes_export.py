"""KNX Project Export Routes.

Exports KNXSim premise data as a .knxproj file (ZIP archive with XML)
compatible with ETS project imports.
"""

import io
import uuid
import zipfile
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


def _build_knxproj_xml(premise: dict, floors: list, devices: list) -> str:
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

    # Organize GAs by: main_group > floor > room
    # Structure: {main: {floor: {room: [ga_info, ...]}}}
    ga_hierarchy = {}
    ga_id_map = {}  # Maps GA string to XML ID

    for device in devices:
        device_id = device.get("id", "device")
        device_name = device.get("name") or _id_to_display_name(device_id)
        device_type = device.get("type", "")
        device_room_id = device.get("room_id")

        # Get floor and room info
        room_info = room_id_to_info.get(device_room_id, {})
        floor_name = room_info.get("floor", "Unassigned")
        room_name = room_info.get("name", "Unassigned")

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

                # Initialize hierarchy levels
                if main_group not in ga_hierarchy:
                    ga_hierarchy[main_group] = {}
                if floor_name not in ga_hierarchy[main_group]:
                    ga_hierarchy[main_group][floor_name] = {}
                if room_name not in ga_hierarchy[main_group][floor_name]:
                    ga_hierarchy[main_group][floor_name][room_name] = []

                function_name = _ga_name_to_function(ga_name)
                ga_hierarchy[main_group][floor_name][room_name].append({
                    "address": ga_str,
                    "name": f"{device_name} : {function_name}",
                    "device_id": device_id,
                    "dpt": ga_dpt or _guess_dpt(device_type, ga_name),
                })

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

    # === Trades (Functions) - Maps GAs to rooms ===
    trades = ET.SubElement(installation, "Trades")

    for device in devices:
        room_id = device.get("room_id")
        if not room_id or room_id not in room_to_location_id:
            continue

        # Create a Function for each device
        trade = ET.SubElement(trades, "Trade")
        trade.set("Id", _make_id("T"))
        trade.set("Name", device.get("id", "Device"))

        # Function type based on device type
        func_type = _device_type_to_function_type(device.get("type", ""))

        func = ET.SubElement(trade, "Function")
        func.set("Id", _make_id("F"))
        func.set("Name", device.get("id", "Device"))
        func.set("Type", func_type)

        # Link to location
        loc_ref = ET.SubElement(func, "LocationReference")
        loc_ref.set("RefId", room_to_location_id[room_id])

        # Link group addresses
        ga_refs = ET.SubElement(func, "GroupAddressRefs")
        for ga_name, ga_data in device.get("group_addresses", {}).items():
            # Handle both old format (string) and new format (dict)
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

    # Climate
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


def _device_type_to_function_type(device_type: str) -> str:
    """Map device type to ETS Function type."""
    mapping = {
        "light_switch": "FT-1",  # Switchable Light
        "light_dimmer": "FT-2",  # Dimmable Light
        "light_rgb": "FT-3",  # RGB Light
        "blind": "FT-4",  # Sun Protection
        "blind_position": "FT-4",
        "sensor": "FT-6",  # Sensor
        "presence": "FT-6",
        "thermostat": "FT-5",  # HVAC
    }
    return mapping.get(device_type, "FT-0")  # FT-0 = Custom


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

    # Build XML content
    xml_content = _build_knxproj_xml(premise, floors, devices)

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
