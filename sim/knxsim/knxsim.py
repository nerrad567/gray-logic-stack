"""KNX/IP Gateway Simulator — Main Entry Point.

Loads device configuration from config.yaml (on first run) or SQLite (subsequent),
creates virtual devices via PremiseManager, starts the KNXnet/IP tunnelling server
on UDP 3671, and runs the FastAPI management API on port 9090.

This script is the single process that handles everything:
  - KNXnet/IP protocol (UDP server per premise)
  - Virtual device state machines
  - Periodic sensor scenarios
  - FastAPI management API (HTTP + WebSocket on port 9090)
"""

import logging
import os
import signal
import threading

# Use PyYAML if available, otherwise fall back to a simple parser
try:
    import yaml
except ImportError:
    yaml = None

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------


def load_config(path: str) -> dict:
    """Load config.yaml using PyYAML or a basic fallback parser."""
    with open(path) as f:
        if yaml:
            return yaml.safe_load(f)
        else:
            return _parse_yaml_minimal(f.read())


def _parse_yaml_minimal(text: str) -> dict:
    """Very basic YAML parser — handles our specific config structure.

    Only supports: scalars, lists of dicts, nested dicts (2 levels).
    Good enough for config.yaml without requiring pip install pyyaml.
    """
    result = {"gateway": {}, "devices": [], "scenarios": []}
    current_section = None
    current_item = None
    current_sub = None

    for line in text.split("\n"):
        stripped = line.rstrip()
        if not stripped or stripped.startswith("#"):
            continue

        indent = len(line) - len(line.lstrip())

        # Top-level keys
        if indent == 0 and ":" in stripped:
            key = stripped.split(":")[0].strip()
            current_section = key
            current_item = None
            current_sub = None
            continue

        # List item start
        if stripped.lstrip().startswith("- "):
            if current_section in ("devices", "scenarios"):
                current_item = {}
                result[current_section].append(current_item)
                current_sub = None
                # Parse key on same line as -
                rest = stripped.lstrip()[2:]
                if ":" in rest:
                    k, v = rest.split(":", 1)
                    current_item[k.strip()] = _parse_value(v.strip())
            continue

        # Nested dict or value
        if ":" in stripped:
            k, v = stripped.split(":", 1)
            k = k.strip()
            v = v.strip()

            if current_item is not None:
                if v == "" or v == "{}":
                    current_sub = k
                    current_item[k] = {}
                elif current_sub and indent >= 8:
                    current_item[current_sub][k] = _parse_value(v)
                else:
                    if v:
                        current_item[k] = _parse_value(v)
                    else:
                        current_sub = k
                        current_item[k] = {}
            elif current_section == "gateway":
                result["gateway"][k] = _parse_value(v)

    return result


def _parse_value(v: str):
    """Parse a YAML scalar value."""
    if (v.startswith('"') and v.endswith('"')) or (v.startswith("'") and v.endswith("'")):
        return v[1:-1]
    if v.lower() in ("true", "yes"):
        return True
    if v.lower() in ("false", "no"):
        return False
    try:
        if "." in v:
            return float(v)
        return int(v)
    except ValueError:
        return v


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main():
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(name)s] %(levelname)s %(message)s",
        datefmt="%H:%M:%S",
    )
    logger = logging.getLogger("knxsim")

    # Late imports to avoid circular dependencies at module level
    from api.app import create_app
    from api.websocket_hub import WebSocketHub
    from core.premise_manager import PremiseManager
    from core.telegram_inspector import TelegramInspector
    from persistence.db import Database

    # Load configuration
    config_path = os.environ.get("KNXSIM_CONFIG", "/app/config.yaml")
    if not os.path.exists(config_path):
        config_path = os.path.join(os.path.dirname(__file__), "config.yaml")

    logger.info("Loading config from %s", config_path)
    config = load_config(config_path)

    # Initialize database
    db_path = os.environ.get("KNXSIM_DB", "/app/data/knxsim.db")
    db = Database(db_path=db_path)
    db.connect()
    logger.info("Database initialized: %s", db_path)

    # Create real-time infrastructure
    ws_hub = WebSocketHub()
    telegram_inspector = TelegramInspector(max_size=1000)

    # Import DPT codec for payload decoding
    from dpt.codec import DPTCodec

    # Create premise manager with event callbacks
    def on_telegram(premise_id, cemi_dict):
        """Called when any premise receives/sends a telegram."""
        logger.info("on_telegram called: premise=%s dst=%s", premise_id, cemi_dict.get("dst"))
        # Determine direction (rx = from client, tx = from scenario/simulator)
        direction = cemi_dict.pop("_direction", "rx")

        # Record in ring buffer — resolve device by GA or source address
        device_id = None
        device = None
        dst_ga = cemi_dict.get("dst", 0)
        premise = manager.premises.get(premise_id)
        if premise:
            if direction == "tx":
                # Outgoing: find device by source individual address
                src = cemi_dict.get("src", 0)
                for dev in premise.devices.values():
                    if dev.individual_address == src:
                        device_id = dev.device_id
                        device = dev
                        break
            else:
                # Incoming: find device by destination GA
                # _ga_map stores a list of devices per GA (multiple devices can share the same GA)
                devices = premise._ga_map.get(dst_ga)
                if devices and len(devices) > 0:
                    device = devices[0]  # Use first device for recording purposes
                    device_id = device.device_id

        # Decode payload using DPT if device and GA are known
        dpt = None
        decoded_value = None
        unit = None
        ga_function = None

        if device:
            ga_function = device.get_ga_name(dst_ga)
            dpt = device.get_dpt_for_ga(dst_ga)
            if dpt:
                payload = cemi_dict.get("payload", b"")
                if payload:
                    try:
                        decoded_value = DPTCodec.decode(dpt, payload)
                        dpt_info = DPTCodec.get_info(dpt)
                        if dpt_info:
                            unit = dpt_info.unit
                    except (ValueError, IndexError):
                        # Decoding failed — leave as None
                        pass

        telegram_inspector.record(
            premise_id,
            cemi_dict,
            direction=direction,
            device_id=device_id,
            dpt=dpt,
            decoded_value=decoded_value,
            unit=unit,
            ga_function=ga_function,
        )

        # Broadcast to WebSocket subscribers
        entry = telegram_inspector.get_history(premise_id, limit=1)
        if entry:
            ws_hub.push_telegram(premise_id, entry[0])

    def on_state_change(premise_id, device_id, state, ga=None):
        """Called when a device state changes.
        
        Args:
            premise_id: The premise ID
            device_id: The device ID
            state: The device state dict
            ga: Optional GA that triggered the change (for channel state updates)
        """
        # Persist device-level state to DB
        db.update_device_state(device_id, state)
        
        # If GA is provided, try to update channel state for multi-channel devices
        if ga is not None:
            db.update_channel_state_by_ga(device_id, ga, state)
        
        # Get updated device with channels for broadcast
        device = db.get_device(device_id)
        channels = device.get("channels") if device else None
        
        # Broadcast to WebSocket subscribers (include channels for UI)
        ws_hub.push_state_change(premise_id, device_id, state, channels=channels)

    # Load device templates for template_device support in config
    from templates.loader import TemplateLoader

    template_loader = TemplateLoader()
    template_loader.load_all()

    manager = PremiseManager(
        db=db,
        on_telegram=on_telegram,
        on_state_change=on_state_change,
        template_loader=template_loader,
    )

    # Bootstrap from config.yaml (first run) or load from DB (subsequent)
    manager.bootstrap_from_config(config)
    manager.load_all_premises()

    premise_count = len(manager.premises)
    device_count = sum(len(p.devices) for p in manager.premises.values())
    logger.info(
        "PremiseManager ready: %d premise(s), %d total device(s)",
        premise_count,
        device_count,
    )

    # Create FastAPI app
    app = create_app(manager, ws_hub=ws_hub, telegram_inspector=telegram_inspector)

    # Start uvicorn in a background thread
    api_port = int(os.environ.get("KNXSIM_API_PORT", "9090"))
    _start_api_server(app, api_port, logger)

    logger.info("KNX Simulator fully started")
    logger.info("  KNX server(s): %d premise(s) running", premise_count)
    logger.info("  Management API: http://0.0.0.0:%d", api_port)
    logger.info("  Health: http://0.0.0.0:%d/api/v1/health", api_port)

    # Wait for shutdown signal
    shutdown = threading.Event()

    def handle_signal(signum, frame):
        logger.info("Received signal %d — shutting down", signum)
        shutdown.set()

    signal.signal(signal.SIGTERM, handle_signal)
    signal.signal(signal.SIGINT, handle_signal)

    shutdown.wait()

    # Cleanup
    logger.info("Stopping all premises...")
    manager.stop_all()
    db.close()
    logger.info("Shutdown complete")


def _start_api_server(app, port: int, logger):
    """Start uvicorn in a daemon thread."""
    import uvicorn

    config = uvicorn.Config(
        app=app,
        host="0.0.0.0",
        port=port,
        log_level="info",
        access_log=False,
    )
    server = uvicorn.Server(config)

    thread = threading.Thread(target=server.run, name="uvicorn", daemon=True)
    thread.start()
    logger.info("Uvicorn started on port %d (daemon thread)", port)
    return thread


if __name__ == "__main__":
    main()
