"""Device Template Loader — discovers and validates YAML templates.

Templates are YAML files in domain subdirectories (lighting/, climate/, etc.).
Each template defines a device type with its group address slots, DPT types,
initial state, and optional scenario configuration.

Template YAML format:
    id: light_dimmer
    name: Dimmable Light
    domain: lighting
    description: Standard KNX dimmable light actuator
    group_addresses:
      switch_cmd:
        dpt: "1.001"
        direction: write
        description: On/Off command
      switch_status:
        dpt: "1.001"
        direction: status
        description: On/Off feedback
      brightness_cmd:
        dpt: "5.001"
        direction: write
        description: Brightness command (0-100%)
      brightness_status:
        dpt: "5.001"
        direction: status
        description: Brightness feedback
    initial_state:
      on: false
      brightness: 0
    scenarios:
      - field: brightness
        type: sine_wave
        params:
          center: 50
          amplitude: 30
          period: 120
"""

import logging
import os
from typing import Optional

try:
    import yaml
except ImportError:
    yaml = None

logger = logging.getLogger("knxsim.templates")


class DeviceTemplate:
    """A validated device template."""

    def __init__(self, data: dict, file_path: str):
        self.id = data["id"]
        self.name = data["name"]
        self.domain = data["domain"]
        self.description = data.get("description", "")
        self.group_addresses = data.get("group_addresses", {})
        self.initial_state = data.get("initial_state", {})
        self.scenarios = data.get("scenarios", [])
        self.file_path = file_path

    def to_dict(self) -> dict:
        return {
            "id": self.id,
            "name": self.name,
            "domain": self.domain,
            "description": self.description,
            "group_addresses": self.group_addresses,
            "initial_state": self.initial_state,
            "scenarios": self.scenarios,
        }

    def get_required_gas(self) -> list[str]:
        """Return list of GA slot names that must be provided."""
        return list(self.group_addresses.keys())


class TemplateLoader:
    """Discovers and loads device templates from the filesystem."""

    def __init__(self, templates_dir: Optional[str] = None):
        if templates_dir is None:
            templates_dir = os.path.dirname(__file__)
        self.templates_dir = templates_dir
        self._templates: dict[str, DeviceTemplate] = {}
        self._loaded = False

    def load_all(self):
        """Scan domain directories and load all YAML templates."""
        if not yaml:
            logger.warning("PyYAML not installed — templates unavailable")
            return

        count = 0
        for domain_dir in sorted(os.listdir(self.templates_dir)):
            domain_path = os.path.join(self.templates_dir, domain_dir)
            if not os.path.isdir(domain_path):
                continue
            if domain_dir.startswith("_") or domain_dir == "__pycache__":
                continue

            for filename in sorted(os.listdir(domain_path)):
                if not filename.endswith((".yaml", ".yml")):
                    continue
                file_path = os.path.join(domain_path, filename)
                try:
                    template = self._load_template(file_path)
                    if template:
                        self._templates[template.id] = template
                        count += 1
                except Exception as e:
                    logger.warning("Failed to load template %s: %s", file_path, e)

        self._loaded = True
        logger.info("Loaded %d device templates from %s", count, self.templates_dir)

    def _load_template(self, file_path: str) -> Optional[DeviceTemplate]:
        """Load and validate a single template file."""
        with open(file_path, "r") as f:
            data = yaml.safe_load(f)

        if not data or not isinstance(data, dict):
            return None

        # Validate required fields
        for field in ("id", "name", "domain"):
            if field not in data:
                logger.warning("Template %s missing field: %s", file_path, field)
                return None

        return DeviceTemplate(data, file_path)

    def get_template(self, template_id: str) -> Optional[DeviceTemplate]:
        """Get a template by ID."""
        if not self._loaded:
            self.load_all()
        return self._templates.get(template_id)

    def list_templates(self, domain: Optional[str] = None) -> list[dict]:
        """List all templates, optionally filtered by domain."""
        if not self._loaded:
            self.load_all()
        templates = self._templates.values()
        if domain:
            templates = [t for t in templates if t.domain == domain]
        return [
            t.to_dict() for t in sorted(templates, key=lambda t: (t.domain, t.name))
        ]

    def list_domains(self) -> list[str]:
        """List all available domains."""
        if not self._loaded:
            self.load_all()
        return sorted(set(t.domain for t in self._templates.values()))
