"""Reference data routes — DPT catalog, GA structure guide, flags.

Provides technical documentation for the UI to help users configure
devices correctly without leaving the application.
"""

from fastapi import APIRouter

router = APIRouter(prefix="/api/v1/reference", tags=["reference"])


# ---------------------------------------------------------------------------
# Group Address Structure
# ---------------------------------------------------------------------------

# ---------------------------------------------------------------------------
# Individual Address Structure
# ---------------------------------------------------------------------------

INDIVIDUAL_ADDRESS_GUIDE = {
    "format": "area.line.device",
    "bits": "4.4.8 (area 0-15, line 0-15, device 0-255)",
    "total_addresses": 65536,
    "description": (
        "Individual addresses uniquely identify each device on the KNX bus. "
        "Think of it like a postal address: Area is the building/zone, Line is the "
        "floor/section, Device is the specific unit. Unlike group addresses (for "
        "communication), individual addresses are for device identification and "
        "point-to-point configuration."
    ),
    "structure": {
        "area": {
            "bits": "4 (bits 15-12)",
            "range": "0-15",
            "description": "Topological area or backbone segment. Typically represents a building or major zone.",
        },
        "line": {
            "bits": "4 (bits 11-8)", 
            "range": "0-15",
            "description": "Line within the area. Typically represents a floor, wing, or functional section.",
        },
        "device": {
            "bits": "8 (bits 7-0)",
            "range": "0-255",
            "description": "Device number on the line. 0 is reserved for line couplers.",
        },
    },
    "special_addresses": [
        {"address": "0.0.0", "meaning": "Reserved (invalid)", "note": "Never use"},
        {"address": "x.0.0", "meaning": "Backbone coupler for area x", "note": "Connects areas to backbone"},
        {"address": "x.y.0", "meaning": "Line coupler for line y in area x", "note": "Connects line to area"},
        {"address": "15.15.255", "meaning": "Broadcast/programming address", "note": "Used during commissioning"},
    ],
    "conventions": [
        {
            "name": "By Physical Location",
            "description": "Area = building/floor, Line = room/zone, Device = sequential",
            "examples": [
                {"address": "1.1.1", "meaning": "Area 1, Line 1, Device 1 (first device on first line)"},
                {"address": "1.1.2", "meaning": "Area 1, Line 1, Device 2 (second device)"},
                {"address": "1.2.1", "meaning": "Area 1, Line 2, Device 1 (first device on second line)"},
                {"address": "2.1.1", "meaning": "Area 2, Line 1, Device 1 (different area)"},
            ],
        },
        {
            "name": "By Device Type",
            "description": "Group similar devices on the same line for easier management",
            "examples": [
                {"address": "1.1.x", "meaning": "Actuators (dimmers, switch actuators)"},
                {"address": "1.2.x", "meaning": "Sensors (presence, temperature)"},
                {"address": "1.3.x", "meaning": "Wall switches and push buttons"},
                {"address": "1.4.x", "meaning": "HVAC controllers"},
            ],
        },
        {
            "name": "Simulator Convention",
            "description": "For KNXSim, use consistent addressing for easy identification",
            "examples": [
                {"address": "1.1.x", "meaning": "Actuators (lights, blinds)"},
                {"address": "1.2.x", "meaning": "Wall switches and controls"},
                {"address": "1.3.x", "meaning": "Sensors"},
                {"address": "1.4.x", "meaning": "System devices"},
            ],
        },
    ],
    "tips": [
        "Each device MUST have a unique individual address — no duplicates allowed",
        "Address 0 in any position has special meaning (couplers, broadcast)",
        "Plan your addressing scheme before commissioning — changing later is painful",
        "Document your addressing in a spreadsheet or ETS project",
        "In a simulator, addresses don't affect routing — but use realistic values for export compatibility",
        "KNXSim assigns addresses to tunneling clients (e.g., 1.0.255 for Core)",
    ],
    "vs_group_address": {
        "individual": "Identifies a specific physical device. Used for configuration, diagnostics, and point-to-point communication.",
        "group": "Logical function address. Multiple devices can listen to the same GA. Used for runtime control and status.",
        "analogy": "Individual = phone number (who you are), Group = radio channel (what you're talking about)",
    },
}


GA_STRUCTURE_GUIDE = {
    "format": "main/middle/sub",
    "bits": "5/3/8 (main 0-31, middle 0-7, sub 0-255)",
    "total_addresses": 65536,
    "description": (
        "KNX group addresses use a three-level hierarchy. The structure helps "
        "organize your installation logically. Common conventions vary by region "
        "and project size."
    ),
    "conventions": [
        {
            "name": "Function-based (recommended for small-medium)",
            "description": "Main = function type, Middle = floor/zone, Sub = device instance",
            "example": {
                "1/x/x": "Lighting",
                "2/x/x": "Blinds/Shutters",
                "3/x/x": "HVAC",
                "4/x/x": "Sensors",
                "5/x/x": "Scenes",
            },
            "example_addresses": [
                {"ga": "1/0/1", "meaning": "Lighting, Ground floor, Switch 1"},
                {"ga": "1/0/2", "meaning": "Lighting, Ground floor, Dimmer 1 brightness"},
                {"ga": "1/1/1", "meaning": "Lighting, First floor, Switch 1"},
                {"ga": "2/0/1", "meaning": "Blinds, Ground floor, Position 1"},
                {"ga": "4/1/0", "meaning": "Sensors, First floor, Temperature"},
            ],
        },
        {
            "name": "Room-based (intuitive for residential)",
            "description": "Main = floor, Middle = room, Sub = function",
            "example": {
                "0/x/x": "Ground floor",
                "1/x/x": "First floor",
                "2/x/x": "Second floor",
            },
            "example_addresses": [
                {"ga": "0/1/1", "meaning": "Ground floor, Living room, Light switch"},
                {"ga": "0/1/2", "meaning": "Ground floor, Living room, Light dim"},
                {"ga": "0/2/1", "meaning": "Ground floor, Kitchen, Light switch"},
                {"ga": "1/1/1", "meaning": "First floor, Bedroom 1, Light switch"},
            ],
        },
        {
            "name": "Building-based (large commercial)",
            "description": "Main = building/wing, Middle = floor, Sub = device/function",
            "example": {
                "0-9": "Building A",
                "10-19": "Building B",
            },
        },
    ],
    "tips": [
        "Keep status/feedback GAs separate from command GAs (e.g., 1/0/1 for switch, 1/0/101 for status)",
        "Reserve address ranges for future expansion",
        "Document your addressing scheme — future you will thank present you",
        "Consider ETS import/export compatibility if using real KNX hardware",
    ],
}


# ---------------------------------------------------------------------------
# Communication Flags
# ---------------------------------------------------------------------------

FLAGS_GUIDE = {
    "format": "CRWTUI",
    "description": (
        "Communication flags control how a group object behaves on the KNX bus. "
        "Each letter represents a capability that can be enabled (-) or disabled."
    ),
    "flags": [
        {
            "letter": "C",
            "name": "Communication",
            "description": "Object participates in bus communication. Almost always enabled.",
            "typical": True,
        },
        {
            "letter": "R",
            "name": "Read",
            "description": "Object responds to GroupRead requests. Enable for sensors/status objects.",
            "typical": False,
        },
        {
            "letter": "W",
            "name": "Write",
            "description": "Object accepts GroupWrite commands. Enable for actuator inputs.",
            "typical": True,
        },
        {
            "letter": "T",
            "name": "Transmit",
            "description": "Object sends GroupWrite when its value changes. Enable for sensors/buttons.",
            "typical": False,
        },
        {
            "letter": "U",
            "name": "Update",
            "description": "Object updates internal value from GroupResponse. For status synchronization.",
            "typical": True,
        },
        {
            "letter": "I",
            "name": "Read on Init",
            "description": "Object sends GroupRead at startup to sync state. Use sparingly to avoid bus floods.",
            "typical": False,
        },
    ],
    "common_patterns": [
        {"flags": "C-W-U-", "use_case": "Command input (switch, dimmer command)", "description": "Receives writes, updates from responses"},
        {"flags": "CR-T--", "use_case": "Status output (switch status, sensor value)", "description": "Can be read, transmits on change"},
        {"flags": "C--T--", "use_case": "Button/trigger (wall switch press)", "description": "Only transmits, doesn't receive"},
        {"flags": "CRW-U-", "use_case": "Bidirectional (setpoint, scene)", "description": "Can be read, written, and updated"},
        {"flags": "CRWTUI", "use_case": "Full (diagnostic/testing)", "description": "All capabilities enabled"},
    ],
}


# ---------------------------------------------------------------------------
# DPT Catalog (grouped by category)
# ---------------------------------------------------------------------------

DPT_CATALOG = {
    "description": (
        "Datapoint Types (DPTs) define how values are encoded on the KNX bus. "
        "Using the correct DPT ensures devices interpret values correctly. "
        "Based on KNX Standard v3.0.0 (03_07_02 Datapoint Types v02.02.01)."
    ),
    "categories": [
        {
            "name": "Boolean (1-bit) — DPT 1.xxx",
            "description": "Single bit values. Encoding: 0x00=0, 0x01=1",
            "dpts": [
                {"id": "1.001", "name": "DPT_Switch", "values": "Off / On", "use_case": "Light switching, relay control"},
                {"id": "1.002", "name": "DPT_Bool", "values": "False / True", "use_case": "Generic boolean"},
                {"id": "1.003", "name": "DPT_Enable", "values": "Disable / Enable", "use_case": "Function enable/disable"},
                {"id": "1.005", "name": "DPT_Alarm", "values": "No alarm / Alarm", "use_case": "Alarm status"},
                {"id": "1.006", "name": "DPT_BinaryValue", "values": "Low / High", "use_case": "Binary input"},
                {"id": "1.007", "name": "DPT_Step", "values": "Decrease / Increase", "use_case": "Step control"},
                {"id": "1.008", "name": "DPT_UpDown", "values": "Up / Down", "use_case": "Blind direction"},
                {"id": "1.009", "name": "DPT_OpenClose", "values": "Open / Close", "use_case": "Valve, contact, door"},
                {"id": "1.010", "name": "DPT_Start", "values": "Stop / Start", "use_case": "Motor control"},
                {"id": "1.011", "name": "DPT_State", "values": "Inactive / Active", "use_case": "Generic state"},
                {"id": "1.015", "name": "DPT_Reset", "values": "No action / Reset", "use_case": "Reset trigger"},
                {"id": "1.016", "name": "DPT_Ack", "values": "No action / Acknowledge", "use_case": "Acknowledgment"},
                {"id": "1.017", "name": "DPT_Trigger", "values": "Trigger / Trigger", "use_case": "Scene trigger, pulse (both values same effect)"},
                {"id": "1.018", "name": "DPT_Occupancy", "values": "Not occupied / Occupied", "use_case": "Presence detection"},
                {"id": "1.019", "name": "DPT_Window_Door", "values": "Closed / Open", "use_case": "Window/door contact"},
                {"id": "1.024", "name": "DPT_DayNight", "values": "Day / Night", "use_case": "Day/night mode"},
            ],
        },
        {
            "name": "Control (2-4 bit) — DPT 2.xxx, 3.xxx",
            "description": "Control with direction and step. DPT 2: c+v bits, DPT 3: c+stepcode",
            "dpts": [
                {"id": "2.001", "name": "DPT_Switch_Control", "values": "c=control bit, v=value", "use_case": "Priority switch (c=1 overrides)"},
                {"id": "2.002", "name": "DPT_Bool_Control", "values": "c=control bit, v=value", "use_case": "Priority boolean"},
                {"id": "3.007", "name": "DPT_Control_Dimming", "values": "c=0 decrease, c=1 increase", "use_case": "Relative dim up/down", "note": "stepcode 0=stop, 1=100%, 2=50%, 3=25%..."},
                {"id": "3.008", "name": "DPT_Control_Blinds", "values": "c=0 up, c=1 down", "use_case": "Relative blind control", "note": "stepcode 0=stop, 1-7=step size"},
            ],
        },
        {
            "name": "Unsigned 8-bit — DPT 5.xxx",
            "description": "8-bit unsigned (0-255). Note: 5.001 is scaled, 5.004 is linear",
            "dpts": [
                {"id": "5.001", "name": "DPT_Scaling", "unit": "%", "range": "0-100", "use_case": "Dimmer level, blind position", "note": "0x00=0%, 0x80=50%, 0xFF=100%"},
                {"id": "5.003", "name": "DPT_Angle", "unit": "°", "range": "0-360", "use_case": "Blind slat angle", "note": "0x00=0°, 0xFF=360°"},
                {"id": "5.004", "name": "DPT_Percent_U8", "unit": "%", "range": "0-255", "use_case": "Raw percentage (linear)", "note": "0x32=50%, 0x64=100%"},
                {"id": "5.005", "name": "DPT_DecimalFactor", "unit": "ratio", "range": "0-255", "use_case": "Multiplication factor"},
                {"id": "5.006", "name": "DPT_Tariff", "unit": "-", "range": "0-254", "use_case": "Tariff number (255=invalid)"},
                {"id": "5.010", "name": "DPT_Value_1_Ucount", "unit": "pulses", "range": "0-255", "use_case": "Pulse counter"},
            ],
        },
        {
            "name": "Signed 8-bit — DPT 6.xxx",
            "description": "8-bit signed two's complement (-128 to +127)",
            "dpts": [
                {"id": "6.001", "name": "DPT_Percent_V8", "unit": "%", "range": "-128 to 127", "use_case": "Relative adjustment"},
                {"id": "6.010", "name": "DPT_Value_1_Count", "unit": "pulses", "range": "-128 to 127", "use_case": "Signed counter"},
            ],
        },
        {
            "name": "Unsigned 16-bit — DPT 7.xxx",
            "description": "16-bit unsigned (0-65535), MSB first",
            "dpts": [
                {"id": "7.001", "name": "DPT_Value_2_Ucount", "unit": "pulses", "range": "0-65535", "use_case": "Counter value"},
                {"id": "7.002", "name": "DPT_TimePeriodMsec", "unit": "ms", "range": "0-65535", "use_case": "Millisecond timer"},
                {"id": "7.005", "name": "DPT_TimePeriodSec", "unit": "s", "range": "0-65535", "use_case": "Second timer"},
                {"id": "7.006", "name": "DPT_TimePeriodMin", "unit": "min", "range": "0-65535", "use_case": "Minute timer"},
                {"id": "7.007", "name": "DPT_TimePeriodHrs", "unit": "h", "range": "0-65535", "use_case": "Hour timer"},
                {"id": "7.011", "name": "DPT_Length_mm", "unit": "mm", "range": "0-65535", "use_case": "Length measurement"},
                {"id": "7.012", "name": "DPT_UElCurrentmA", "unit": "mA", "range": "0-65535", "use_case": "Current measurement"},
                {"id": "7.013", "name": "DPT_Brightness", "unit": "lux", "range": "0-65535", "use_case": "Brightness (integer lux)"},
                {"id": "7.600", "name": "DPT_Absolute_Colour_Temperature", "unit": "K", "range": "0-65535", "use_case": "Color temperature (Kelvin)"},
            ],
        },
        {
            "name": "Signed 16-bit — DPT 8.xxx",
            "description": "16-bit signed two's complement, MSB first",
            "dpts": [
                {"id": "8.001", "name": "DPT_Value_2_Count", "unit": "pulses", "range": "±32767", "use_case": "Signed counter"},
                {"id": "8.005", "name": "DPT_DeltaTimeSec", "unit": "s", "range": "±32767", "use_case": "Time difference"},
                {"id": "8.010", "name": "DPT_Percent_V16", "unit": "%", "range": "-327.68 to 327.67", "use_case": "Precise percentage", "note": "0x7FFF=invalid"},
                {"id": "8.011", "name": "DPT_Rotation_Angle", "unit": "°", "range": "±32767", "use_case": "Rotation angle"},
            ],
        },
        {
            "name": "2-byte Float — DPT 9.xxx",
            "description": "KNX 16-bit float: (0.01 × M) × 2^E. Resolution 0.01. 0x7FFF=invalid",
            "dpts": [
                {"id": "9.001", "name": "DPT_Value_Temp", "unit": "°C", "range": "-273 to 670760", "use_case": "Temperature"},
                {"id": "9.002", "name": "DPT_Value_Tempd", "unit": "K", "range": "±670760", "use_case": "Temperature difference"},
                {"id": "9.003", "name": "DPT_Value_Tempa", "unit": "K/h", "range": "±670760", "use_case": "Temperature change rate"},
                {"id": "9.004", "name": "DPT_Value_Lux", "unit": "lux", "range": "0-670760", "use_case": "Light level sensor"},
                {"id": "9.005", "name": "DPT_Value_Wsp", "unit": "m/s", "range": "0-670760", "use_case": "Wind speed"},
                {"id": "9.006", "name": "DPT_Value_Pres", "unit": "Pa", "range": "0-670760", "use_case": "Air pressure"},
                {"id": "9.007", "name": "DPT_Value_Humidity", "unit": "%", "range": "0-670760", "use_case": "Relative humidity"},
                {"id": "9.008", "name": "DPT_Value_AirQuality", "unit": "ppm", "range": "0-670760", "use_case": "CO2, VOC sensors"},
                {"id": "9.009", "name": "DPT_Value_AirFlow", "unit": "m³/h", "range": "±670760", "use_case": "Ventilation airflow"},
                {"id": "9.020", "name": "DPT_Value_Volt", "unit": "mV", "range": "±670760", "use_case": "Voltage measurement"},
                {"id": "9.021", "name": "DPT_Value_Curr", "unit": "mA", "range": "±670760", "use_case": "Current measurement"},
                {"id": "9.024", "name": "DPT_Power", "unit": "kW", "range": "±670760", "use_case": "Power measurement"},
                {"id": "9.027", "name": "DPT_Value_Temp_F", "unit": "°F", "range": "-459.6 to 670760", "use_case": "Temperature (Fahrenheit)"},
                {"id": "9.028", "name": "DPT_Value_Wsp_kmh", "unit": "km/h", "range": "0-670760", "use_case": "Wind speed (km/h)"},
            ],
        },
        {
            "name": "Time & Date — DPT 10, 11, 19",
            "description": "Time (3 bytes), Date (3 bytes), DateTime (8 bytes)",
            "dpts": [
                {"id": "10.001", "name": "DPT_TimeOfDay", "format": "DDD HHHHH : 00MMMMMM : 00SSSSSS", "use_case": "Time with weekday (0=no day, 1=Mon...7=Sun)"},
                {"id": "11.001", "name": "DPT_Date", "format": "000DDDDD : 0000MMMM : 0YYYYYYY", "use_case": "Date (year: ≥90=19xx, <90=20xx)"},
                {"id": "19.001", "name": "DPT_DateTime", "format": "8 bytes: Y,M,D,DH,M,S,flags", "use_case": "Full timestamp"},
            ],
        },
        {
            "name": "Scene — DPT 17, 18",
            "description": "Scene recall and control",
            "dpts": [
                {"id": "17.001", "name": "DPT_SceneNumber", "range": "0-63", "use_case": "Scene recall (displayed as 1-64)"},
                {"id": "18.001", "name": "DPT_SceneControl", "format": "L0SSSSSS", "use_case": "Scene control", "note": "L=0 activate, L=1 learn/store"},
            ],
        },
        {
            "name": "HVAC Modes — DPT 20.xxx",
            "description": "8-bit enumeration for HVAC control",
            "dpts": [
                {"id": "20.001", "name": "DPT_SCLOMode", "values": "0=Autonomous, 1=Slave", "use_case": "SCLO mode"},
                {"id": "20.002", "name": "DPT_BuildingMode", "values": "0=In use, 1=Not used, 2=Protection", "use_case": "Building occupancy mode"},
                {"id": "20.003", "name": "DPT_OccMode", "values": "0=Occupied, 1=Standby, 2=Not occupied", "use_case": "Room occupancy"},
                {"id": "20.102", "name": "DPT_HVACMode", "values": "0=Auto, 1=Comfort, 2=Standby, 3=Economy, 4=Protection", "use_case": "HVAC operating mode"},
                {"id": "20.105", "name": "DPT_HVACContrMode", "values": "0=Auto, 1=Heat, 2=Morning warmup, 3=Cool, 4=Night purge, 5=Precool, 6=Off...", "use_case": "HVAC control mode (20 values)"},
            ],
        },
        {
            "name": "Unsigned 32-bit — DPT 12.xxx",
            "description": "32-bit unsigned (0 to 4294967295), MSB first",
            "dpts": [
                {"id": "12.001", "name": "DPT_Value_4_Ucount", "unit": "pulses", "range": "0-4294967295", "use_case": "Large counter"},
                {"id": "12.100", "name": "DPT_LongTimePeriod_Sec", "unit": "s", "range": "0-4294967295", "use_case": "Long duration (seconds)"},
                {"id": "12.101", "name": "DPT_LongTimePeriod_Min", "unit": "min", "range": "0-4294967295", "use_case": "Long duration (minutes)"},
            ],
        },
        {
            "name": "Signed 32-bit — DPT 13.xxx",
            "description": "32-bit signed two's complement, MSB first",
            "dpts": [
                {"id": "13.001", "name": "DPT_Value_4_Count", "unit": "pulses", "range": "±2147483647", "use_case": "Signed counter"},
                {"id": "13.002", "name": "DPT_FlowRate_m3/h", "unit": "0.0001 m³/h", "range": "±2147483647", "use_case": "Flow rate"},
                {"id": "13.010", "name": "DPT_ActiveEnergy", "unit": "Wh", "range": "±2147483647", "use_case": "Energy meter (Wh)"},
                {"id": "13.011", "name": "DPT_ApparantEnergy", "unit": "VAh", "range": "±2147483647", "use_case": "Apparent energy"},
                {"id": "13.012", "name": "DPT_ReactiveEnergy", "unit": "VARh", "range": "±2147483647", "use_case": "Reactive energy"},
                {"id": "13.013", "name": "DPT_ActiveEnergy_kWh", "unit": "kWh", "range": "±2147483647", "use_case": "Energy meter (kWh)"},
            ],
        },
        {
            "name": "4-byte Float — DPT 14.xxx",
            "description": "IEEE 754 single precision float",
            "dpts": [
                {"id": "14.000", "name": "DPT_Value_Acceleration", "unit": "m/s²", "use_case": "Acceleration"},
                {"id": "14.007", "name": "DPT_Value_AngleDeg", "unit": "°", "use_case": "Angle (degrees)"},
                {"id": "14.019", "name": "DPT_Value_Electric_Current", "unit": "A", "use_case": "Current (high precision)"},
                {"id": "14.027", "name": "DPT_Value_Electric_Potential", "unit": "V", "use_case": "Voltage (high precision)"},
                {"id": "14.033", "name": "DPT_Value_Frequency", "unit": "Hz", "use_case": "Frequency"},
                {"id": "14.056", "name": "DPT_Value_Power", "unit": "W", "use_case": "Power (high precision)"},
                {"id": "14.057", "name": "DPT_Value_Power_Factor", "unit": "-", "use_case": "Power factor (cos φ)"},
                {"id": "14.065", "name": "DPT_Value_Speed", "unit": "m/s", "use_case": "Speed"},
                {"id": "14.068", "name": "DPT_Value_Common_Temperature", "unit": "°C", "use_case": "Temperature (high precision)"},
                {"id": "14.076", "name": "DPT_Value_Volume", "unit": "m³", "use_case": "Volume"},
                {"id": "14.077", "name": "DPT_Value_Volume_Flux", "unit": "m³/s", "use_case": "Volume flow"},
            ],
        },
        {
            "name": "Text — DPT 4, 16",
            "description": "Character and string values",
            "dpts": [
                {"id": "4.001", "name": "DPT_Char_ASCII", "size": "1 byte", "use_case": "Single ASCII character (0-127)"},
                {"id": "4.002", "name": "DPT_Char_8859_1", "size": "1 byte", "use_case": "Single Latin-1 character (0-255)"},
                {"id": "16.000", "name": "DPT_String_ASCII", "size": "14 bytes", "use_case": "Text display (ASCII, NUL-padded)"},
                {"id": "16.001", "name": "DPT_String_8859_1", "size": "14 bytes", "use_case": "Text display (Latin-1)"},
            ],
        },
        {
            "name": "Color — DPT 232, 251",
            "description": "RGB and RGBW color values",
            "dpts": [
                {"id": "232.600", "name": "DPT_Colour_RGB", "format": "R, G, B (0-255 each)", "use_case": "RGB LED control", "note": "3 bytes"},
                {"id": "251.600", "name": "DPT_Colour_RGBW", "format": "R, G, B, W + validity", "use_case": "RGBW LED control", "note": "6 bytes (4 color + validity + reserved)"},
            ],
        },
    ],
}


# ---------------------------------------------------------------------------
# Device Type GA Templates
# ---------------------------------------------------------------------------

# ---------------------------------------------------------------------------
# Multi-Channel Device Templates
# ---------------------------------------------------------------------------

MULTI_CHANNEL_TEMPLATES = {
    "switch_actuator_2fold": {
        "description": "2-channel switch actuator (e.g., ABB SA/S 2.16.1)",
        "channel_count": 2,
        "channel_template": {
            "group_objects": [
                {"name": "switch", "dpt": "1.001", "flags": "CWU", "description": "On/Off command"},
                {"name": "switch_status", "dpt": "1.001", "flags": "CRT", "description": "Current state"},
            ],
            "state_fields": {"on": False},
        },
        "device_parameters": ["power_on_state", "bus_voltage_failure"],
    },
    "switch_actuator_4fold": {
        "description": "4-channel switch actuator (e.g., ABB SA/S 4.16.1)",
        "channel_count": 4,
        "channel_template": {
            "group_objects": [
                {"name": "switch", "dpt": "1.001", "flags": "CWU", "description": "On/Off command"},
                {"name": "switch_status", "dpt": "1.001", "flags": "CRT", "description": "Current state"},
            ],
            "state_fields": {"on": False},
        },
        "device_parameters": ["power_on_state", "bus_voltage_failure"],
    },
    "switch_actuator_8fold": {
        "description": "8-channel switch actuator (e.g., ABB SA/S 8.16.1)",
        "channel_count": 8,
        "channel_template": {
            "group_objects": [
                {"name": "switch", "dpt": "1.001", "flags": "CWU", "description": "On/Off command"},
                {"name": "switch_status", "dpt": "1.001", "flags": "CRT", "description": "Current state"},
            ],
            "state_fields": {"on": False},
        },
        "device_parameters": ["power_on_state", "bus_voltage_failure"],
    },
    "dimmer_actuator_2fold": {
        "description": "2-channel dimmer actuator (e.g., ABB UD/S 2.300.2)",
        "channel_count": 2,
        "channel_template": {
            "group_objects": [
                {"name": "switch", "dpt": "1.001", "flags": "CWU", "description": "On/Off command"},
                {"name": "switch_status", "dpt": "1.001", "flags": "CRT", "description": "On/Off state"},
                {"name": "brightness", "dpt": "5.001", "flags": "CWU", "description": "Brightness (0-100%)"},
                {"name": "brightness_status", "dpt": "5.001", "flags": "CRT", "description": "Current brightness"},
                {"name": "dim", "dpt": "3.007", "flags": "CW", "description": "Relative dimming"},
            ],
            "state_fields": {"on": False, "brightness": 0},
        },
        "device_parameters": ["min_brightness", "max_brightness", "dim_speed", "power_on_state"],
    },
    "dimmer_actuator_4fold": {
        "description": "4-channel dimmer actuator (e.g., ABB UD/S 4.210.2)",
        "channel_count": 4,
        "channel_template": {
            "group_objects": [
                {"name": "switch", "dpt": "1.001", "flags": "CWU", "description": "On/Off command"},
                {"name": "switch_status", "dpt": "1.001", "flags": "CRT", "description": "On/Off state"},
                {"name": "brightness", "dpt": "5.001", "flags": "CWU", "description": "Brightness (0-100%)"},
                {"name": "brightness_status", "dpt": "5.001", "flags": "CRT", "description": "Current brightness"},
                {"name": "dim", "dpt": "3.007", "flags": "CW", "description": "Relative dimming"},
            ],
            "state_fields": {"on": False, "brightness": 0},
        },
        "device_parameters": ["min_brightness", "max_brightness", "dim_speed", "power_on_state"],
    },
    "blind_actuator_2fold": {
        "description": "2-channel blind/shutter actuator (e.g., ABB JRA/S 2.230.5.1)",
        "channel_count": 2,
        "channel_template": {
            "group_objects": [
                {"name": "move", "dpt": "1.008", "flags": "CW", "description": "Up/Down command"},
                {"name": "stop", "dpt": "1.017", "flags": "CW", "description": "Stop movement"},
                {"name": "position", "dpt": "5.001", "flags": "CWU", "description": "Target position (0-100%)"},
                {"name": "position_status", "dpt": "5.001", "flags": "CRT", "description": "Current position"},
            ],
            "state_fields": {"position": 0, "moving": False},
        },
        "device_parameters": ["travel_time_up", "travel_time_down", "reverse_direction"],
    },
    "blind_actuator_4fold": {
        "description": "4-channel blind/shutter actuator (e.g., ABB JRA/S 4.230.5.1)",
        "channel_count": 4,
        "channel_template": {
            "group_objects": [
                {"name": "move", "dpt": "1.008", "flags": "CW", "description": "Up/Down command"},
                {"name": "stop", "dpt": "1.017", "flags": "CW", "description": "Stop movement"},
                {"name": "position", "dpt": "5.001", "flags": "CWU", "description": "Target position (0-100%)"},
                {"name": "position_status", "dpt": "5.001", "flags": "CRT", "description": "Current position"},
            ],
            "state_fields": {"position": 0, "moving": False},
        },
        "device_parameters": ["travel_time_up", "travel_time_down", "reverse_direction"],
    },
    "push_button_2fold": {
        "description": "2-button push button interface (e.g., ABB US/U 2.2)",
        "channel_count": 2,
        "channel_template": {
            "group_objects": [
                {"name": "switch", "dpt": "1.001", "flags": "CRT", "description": "Button toggle output"},
                {"name": "long_press", "dpt": "1.001", "flags": "CRT", "description": "Long press output"},
            ],
            "state_fields": {"pressed": False},
        },
        "device_parameters": ["long_press_time", "led_feedback"],
    },
    "push_button_4fold": {
        "description": "4-button push button interface (e.g., ABB US/U 4.2)",
        "channel_count": 4,
        "channel_template": {
            "group_objects": [
                {"name": "switch", "dpt": "1.001", "flags": "CRT", "description": "Button toggle output"},
                {"name": "long_press", "dpt": "1.001", "flags": "CRT", "description": "Long press output"},
            ],
            "state_fields": {"pressed": False},
        },
        "device_parameters": ["long_press_time", "led_feedback"],
    },
    "push_button_6fold": {
        "description": "6-button push button interface (e.g., Gira push button sensor 3)",
        "channel_count": 6,
        "channel_template": {
            "group_objects": [
                {"name": "switch", "dpt": "1.001", "flags": "CRT", "description": "Button toggle output"},
                {"name": "long_press", "dpt": "1.001", "flags": "CRT", "description": "Long press output"},
            ],
            "state_fields": {"pressed": False},
        },
        "device_parameters": ["long_press_time", "led_feedback"],
    },
    "binary_input_4fold": {
        "description": "4-channel binary input (e.g., ABB US/U 4.2)",
        "channel_count": 4,
        "channel_template": {
            "group_objects": [
                {"name": "state", "dpt": "1.001", "flags": "CRT", "description": "Input state"},
                {"name": "counter", "dpt": "12.001", "flags": "CRT", "description": "Pulse counter (optional)"},
            ],
            "state_fields": {"active": False},
        },
        "device_parameters": ["debounce_time", "invert_input"],
    },
    "binary_input_8fold": {
        "description": "8-channel binary input (e.g., ABB US/U 8.2)",
        "channel_count": 8,
        "channel_template": {
            "group_objects": [
                {"name": "state", "dpt": "1.001", "flags": "CRT", "description": "Input state"},
                {"name": "counter", "dpt": "12.001", "flags": "CRT", "description": "Pulse counter (optional)"},
            ],
            "state_fields": {"active": False},
        },
        "device_parameters": ["debounce_time", "invert_input"],
    },
}

# Channel ID labels (A-H for up to 8 channels)
CHANNEL_LABELS = ["A", "B", "C", "D", "E", "F", "G", "H"]

DEVICE_GA_TEMPLATES = {
    "light_switch": {
        "description": "Simple on/off light",
        "channel_count": 1,
        "recommended_gas": [
            {"name": "switch", "dpt": "1.001", "flags": "C-W-U-", "description": "On/Off command input"},
            {"name": "switch_status", "dpt": "1.001", "flags": "CR-T--", "description": "Current on/off state"},
        ],
    },
    "light_dimmer": {
        "description": "Dimmable light",
        "channel_count": 1,
        "recommended_gas": [
            {"name": "switch", "dpt": "1.001", "flags": "C-W-U-", "description": "On/Off command"},
            {"name": "switch_status", "dpt": "1.001", "flags": "CR-T--", "description": "On/Off state"},
            {"name": "brightness", "dpt": "5.001", "flags": "C-W-U-", "description": "Brightness command (0-100%)"},
            {"name": "brightness_status", "dpt": "5.001", "flags": "CR-T--", "description": "Current brightness"},
            {"name": "dim", "dpt": "3.007", "flags": "C-W---", "description": "Relative dimming (optional)"},
        ],
    },
    "blind": {
        "description": "Blind/shutter",
        "channel_count": 1,
        "recommended_gas": [
            {"name": "move", "dpt": "1.008", "flags": "C-W---", "description": "Up/Down command"},
            {"name": "stop", "dpt": "1.017", "flags": "C-W---", "description": "Stop movement"},
            {"name": "position", "dpt": "5.001", "flags": "C-W-U-", "description": "Target position (0=open, 100=closed)"},
            {"name": "position_status", "dpt": "5.001", "flags": "CR-T--", "description": "Current position"},
        ],
    },
    "blind_position_slat": {
        "description": "Blind with slat/tilt control",
        "channel_count": 1,
        "recommended_gas": [
            {"name": "move", "dpt": "1.008", "flags": "C-W---", "description": "Up/Down command"},
            {"name": "stop", "dpt": "1.017", "flags": "C-W---", "description": "Stop movement"},
            {"name": "position", "dpt": "5.001", "flags": "C-W-U-", "description": "Target position"},
            {"name": "position_status", "dpt": "5.001", "flags": "CR-T--", "description": "Current position"},
            {"name": "slat", "dpt": "5.001", "flags": "C-W-U-", "description": "Slat angle command"},
            {"name": "slat_status", "dpt": "5.001", "flags": "CR-T--", "description": "Current slat angle"},
        ],
    },
    "sensor": {
        "description": "Temperature/environmental sensor",
        "channel_count": 1,
        "recommended_gas": [
            {"name": "temperature", "dpt": "9.001", "flags": "CR-T--", "description": "Temperature reading"},
            {"name": "humidity", "dpt": "9.007", "flags": "CR-T--", "description": "Humidity reading (optional)"},
        ],
    },
    "presence": {
        "description": "Presence/motion detector",
        "channel_count": 1,
        "recommended_gas": [
            {"name": "presence", "dpt": "1.018", "flags": "CR-T--", "description": "Occupancy state"},
            {"name": "lux", "dpt": "9.004", "flags": "CR-T--", "description": "Light level (if equipped)"},
        ],
    },
    "thermostat": {
        "description": "Room thermostat",
        "channel_count": 1,
        "recommended_gas": [
            {"name": "setpoint", "dpt": "9.001", "flags": "CRW-U-", "description": "Target temperature"},
            {"name": "setpoint_status", "dpt": "9.001", "flags": "CR-T--", "description": "Current setpoint"},
            {"name": "temperature", "dpt": "9.001", "flags": "CR-T--", "description": "Measured temperature"},
            {"name": "mode", "dpt": "20.102", "flags": "CRW-U-", "description": "HVAC mode"},
            {"name": "mode_status", "dpt": "20.102", "flags": "CR-T--", "description": "Current mode"},
            {"name": "heating", "dpt": "1.001", "flags": "CR-T--", "description": "Heating active status"},
        ],
    },
}

# Merge multi-channel templates into device templates for unified access
ALL_DEVICE_TEMPLATES = {**DEVICE_GA_TEMPLATES, **MULTI_CHANNEL_TEMPLATES}


# ---------------------------------------------------------------------------
# API Endpoints
# ---------------------------------------------------------------------------


@router.get("/individual-address")
def get_individual_address_guide():
    """Get individual address structure guide and conventions."""
    return INDIVIDUAL_ADDRESS_GUIDE


@router.get("/ga-structure")
def get_ga_structure():
    """Get group address structure guide and conventions."""
    return GA_STRUCTURE_GUIDE


@router.get("/flags")
def get_flags_guide():
    """Get communication flags reference."""
    return FLAGS_GUIDE


@router.get("/dpts")
def get_dpt_catalog():
    """Get complete DPT catalog with descriptions."""
    return DPT_CATALOG


@router.get("/device-templates")
def get_device_templates():
    """Get all device templates (single and multi-channel)."""
    return ALL_DEVICE_TEMPLATES


@router.get("/device-templates/{device_type}")
def get_device_template(device_type: str):
    """Get template for a specific device type."""
    template = ALL_DEVICE_TEMPLATES.get(device_type)
    if not template:
        return {"error": f"Unknown device type: {device_type}", "available": list(ALL_DEVICE_TEMPLATES.keys())}
    return template


@router.get("/multi-channel-templates")
def get_multi_channel_templates():
    """Get templates for multi-channel devices only."""
    return MULTI_CHANNEL_TEMPLATES


@router.get("/channel-labels")
def get_channel_labels():
    """Get standard channel labels (A-H)."""
    return {"labels": CHANNEL_LABELS}


@router.get("/device-templates/{device_type}/channels")
def get_default_channels(device_type: str):
    """Generate default channel structure for a device type.
    
    Returns ready-to-use channels array that can be assigned to a device.
    """
    # Check multi-channel templates first
    template = MULTI_CHANNEL_TEMPLATES.get(device_type)
    if template:
        channel_count = template["channel_count"]
        channel_template = template["channel_template"]
        
        channels = []
        for i in range(channel_count):
            label = CHANNEL_LABELS[i] if i < len(CHANNEL_LABELS) else str(i + 1)
            
            # Convert group_objects list to dict format
            group_objects = {}
            for go in channel_template["group_objects"]:
                group_objects[go["name"]] = {
                    "ga": None,  # To be assigned
                    "dpt": go["dpt"],
                    "flags": go["flags"],
                    "description": go.get("description", ""),
                }
            
            channels.append({
                "id": label,
                "name": f"Channel {label}",
                "group_objects": group_objects,
                "state": dict(channel_template.get("state_fields", {})),
                "initial_state": dict(channel_template.get("state_fields", {})),
                "parameters": {},
            })
        
        return {
            "device_type": device_type,
            "description": template["description"],
            "channel_count": channel_count,
            "channels": channels,
            "device_parameters": template.get("device_parameters", []),
        }
    
    # Check single-channel templates
    template = DEVICE_GA_TEMPLATES.get(device_type)
    if template:
        # Convert recommended_gas to group_objects format
        group_objects = {}
        for ga in template.get("recommended_gas", []):
            group_objects[ga["name"]] = {
                "ga": None,
                "dpt": ga["dpt"],
                "flags": ga["flags"],
                "description": ga.get("description", ""),
            }
        
        channels = [{
            "id": "A",
            "name": template["description"],
            "group_objects": group_objects,
            "state": {},
            "initial_state": {},
            "parameters": {},
        }]
        
        return {
            "device_type": device_type,
            "description": template["description"],
            "channel_count": 1,
            "channels": channels,
            "device_parameters": [],
        }
    
    return {"error": f"Unknown device type: {device_type}", "available": list(ALL_DEVICE_TEMPLATES.keys())}


@router.get("")
def get_all_reference():
    """Get all reference data in one request (for initial load)."""
    return {
        "individual_address": INDIVIDUAL_ADDRESS_GUIDE,
        "ga_structure": GA_STRUCTURE_GUIDE,
        "flags": FLAGS_GUIDE,
        "dpts": DPT_CATALOG,
        "device_templates": ALL_DEVICE_TEMPLATES,
        "multi_channel_templates": MULTI_CHANNEL_TEMPLATES,
        "channel_labels": CHANNEL_LABELS,
    }
