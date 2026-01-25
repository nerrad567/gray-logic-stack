/**
 * Device Panel Component
 *
 * Engineer mode detail view with GA info and controls.
 */

export class DevicePanel {
  constructor(container, onCommand) {
    this.container = container;
    this.onCommand = onCommand;
    this.device = null;

    // Cache DOM elements
    this.els = {
      name: document.getElementById("panel-device-name"),
      type: document.getElementById("panel-device-type"),
      individualAddr: document.getElementById("panel-individual-addr"),
      gaTable: document.getElementById("panel-ga-table").querySelector("tbody"),
      state: document.getElementById("panel-state"),
      controls: document.getElementById("panel-controls"),
    };
  }

  /**
   * Show the panel with device details.
   */
  show(device) {
    this.device = device;
    this.container.classList.remove("hidden");
    this._render();
  }

  /**
   * Hide the panel.
   */
  hide() {
    this.device = null;
    this.container.classList.add("hidden");
  }

  /**
   * Update state display and controls when state changes.
   */
  updateState(device) {
    this.device = device;
    this._renderState();
    this._renderControls(); // Re-render controls to reflect new state
  }

  /**
   * Render all panel content.
   */
  _render() {
    if (!this.device) return;

    const d = this.device;

    // Header info
    this.els.name.textContent = this._formatDeviceName(d.id);
    this.els.type.textContent = d.type;
    this.els.individualAddr.textContent = d.individual_address || "-";

    // Group addresses table
    this._renderGATable();

    // State
    this._renderState();

    // Controls
    this._renderControls();
  }

  /**
   * Render group addresses table.
   */
  _renderGATable() {
    const gas = this.device.group_addresses || {};

    if (Object.keys(gas).length === 0) {
      this.els.gaTable.innerHTML = `
                <tr><td colspan="3" style="color: var(--text-muted);">No group addresses</td></tr>
            `;
      return;
    }

    this.els.gaTable.innerHTML = Object.entries(gas)
      .map(
        ([name, ga]) => `
                <tr>
                    <td>${name}</td>
                    <td>${ga}</td>
                    <td>${this._guessDPT(name, this.device.type)}</td>
                </tr>
            `,
      )
      .join("");
  }

  /**
   * Render current state.
   */
  _renderState() {
    const state = this.device?.state || {};
    this.els.state.textContent = JSON.stringify(state, null, 2);
  }

  /**
   * Render device controls based on type.
   */
  _renderControls() {
    const type = this.device.type;
    const state = this.device.state || {};

    let html = "";

    // Light controls
    if (type.startsWith("light_")) {
      const isOn = state.on === true;
      html += `
                <div class="control-row">
                    <span class="control-label">Power</span>
                    <button class="toggle-btn ${isOn ? "on" : ""}" data-command="switch" data-value="${!isOn}">
                        ${isOn ? "ON" : "OFF"}
                    </button>
                </div>
            `;

      // Dimmer brightness
      if (
        type === "light_dimmer" ||
        type === "light_rgb" ||
        type === "light_colour_temp"
      ) {
        const brightness = state.brightness ?? 0;
        html += `
                    <div class="control-row">
                        <span class="control-label">Brightness</span>
                        <div class="slider-control">
                            <input type="range" class="slider" min="0" max="100" value="${brightness}"
                                   data-command="brightness">
                            <span class="slider-value">${brightness}%</span>
                        </div>
                    </div>
                `;
      }
    }

    // Blind controls
    if (type.startsWith("blind")) {
      const position = state.position ?? 0;
      html += `
                <div class="control-row">
                    <span class="control-label">Position</span>
                    <div class="slider-control">
                        <input type="range" class="slider" min="0" max="100" value="${position}"
                               data-command="position">
                        <span class="slider-value">${position}%</span>
                    </div>
                </div>
            `;

      // Slat/tilt if available
      if (state.tilt !== undefined || type === "blind_position_slat") {
        const tilt = state.tilt ?? 0;
        html += `
                    <div class="control-row">
                        <span class="control-label">Slat Angle</span>
                        <div class="slider-control">
                            <input type="range" class="slider" min="0" max="100" value="${tilt}"
                                   data-command="slat">
                            <span class="slider-value">${tilt}%</span>
                        </div>
                    </div>
                `;
      }
    }

    // Thermostat setpoint
    if (type === "thermostat") {
      const setpoint = state.setpoint ?? 21;
      html += `
                <div class="control-row">
                    <span class="control-label">Setpoint</span>
                    <div class="slider-control">
                        <input type="range" class="slider" min="15" max="30" step="0.5" value="${setpoint}"
                               data-command="setpoint">
                        <span class="slider-value">${setpoint}°C</span>
                    </div>
                </div>
            `;
    }

    // Scene controller
    if (type === "scene_controller") {
      html += `
                <div class="control-row">
                    <span class="control-label">Scene</span>
                    <select class="scene-select" data-command="scene">
                        ${Array.from(
                          { length: 8 },
                          (_, i) =>
                            `<option value="${i}">Scene ${i + 1}</option>`,
                        ).join("")}
                    </select>
                    <button class="btn btn-small" data-action="recall-scene">Recall</button>
                </div>
            `;
    }

    // Presence sensor controls
    if (type === "presence_sensor" || type === "presence") {
      const isPresent = state.presence === true;
      const lux = state.lux ?? 300;

      html += `
                <div class="control-row">
                    <span class="control-label">Motion</span>
                    <button class="toggle-btn ${isPresent ? "on" : ""}" data-command="presence" data-value="${!isPresent}">
                        ${isPresent ? "DETECTED" : "CLEAR"}
                    </button>
                </div>
                <div class="control-row">
                    <span class="control-label">Lux Level</span>
                    <div class="slider-control">
                        <input type="range" class="slider" min="0" max="2000" step="10" value="${lux}"
                               data-command="lux">
                        <span class="slider-value">${lux} lx</span>
                    </div>
                </div>
            `;
    }

    // Default: no controls
    if (!html) {
      html =
        '<p style="color: var(--text-muted); font-size: 12px;">No controls available for this device type</p>';
    }

    this.els.controls.innerHTML = html;

    // Bind control events
    this._bindControlEvents();
  }

  /**
   * Bind event handlers to controls.
   */
  _bindControlEvents() {
    // Toggle buttons
    this.els.controls.querySelectorAll(".toggle-btn").forEach((btn) => {
      btn.addEventListener("click", () => {
        const command = btn.dataset.command;
        const value = btn.dataset.value === "true";
        this.onCommand(this.device.id, command, value);
      });
    });

    // Sliders
    this.els.controls.querySelectorAll(".slider").forEach((slider) => {
      const valueDisplay = slider.parentElement.querySelector(".slider-value");

      // Update display while dragging
      slider.addEventListener("input", () => {
        const val = slider.value;
        const cmd = slider.dataset.command;
        let suffix = "%";
        if (cmd === "setpoint") suffix = "°C";
        else if (cmd === "lux") suffix = " lx";
        valueDisplay.textContent = `${val}${suffix}`;
      });

      // Send command on release
      slider.addEventListener("change", () => {
        const command = slider.dataset.command;
        const value = parseFloat(slider.value);
        this.onCommand(this.device.id, command, value);
      });
    });

    // Scene recall
    const recallBtn = this.els.controls.querySelector(
      '[data-action="recall-scene"]',
    );
    if (recallBtn) {
      recallBtn.addEventListener("click", () => {
        const select = this.els.controls.querySelector(".scene-select");
        const scene = parseInt(select.value, 10);
        this.onCommand(this.device.id, "scene", scene);
      });
    }
  }

  /**
   * Guess DPT based on GA name and device type.
   */
  _guessDPT(gaName, deviceType) {
    const name = gaName.toLowerCase();

    if (name.includes("switch") || name.includes("on_off")) return "1.001";
    if (
      name.includes("brightness") ||
      name.includes("position") ||
      name.includes("slat")
    )
      return "5.001";
    if (name.includes("temperature") || name.includes("temp")) return "9.001";
    if (name.includes("humidity")) return "9.007";
    if (name.includes("lux")) return "9.004";
    if (name.includes("presence") || name.includes("occupancy")) return "1.018";
    if (name.includes("scene")) return "17.001";
    if (name.includes("hvac") || name.includes("mode")) return "20.102";
    if (name.includes("power")) return "14.056";
    if (name.includes("energy")) return "13.010";

    return "-";
  }

  /**
   * Format device ID into readable name.
   */
  _formatDeviceName(id) {
    return id
      .replace(/-/g, " ")
      .replace(/_/g, " ")
      .split(" ")
      .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
      .join(" ");
  }
}
