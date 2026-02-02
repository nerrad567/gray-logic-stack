/**
 * KNXSim Alpine.js Store
 *
 * Global reactive state for the application.
 */

import { API } from "./api.js";

// Device type to icon mapping
const DEVICE_ICONS = {
  light_switch: "💡",
  light_dimmer: "💡",
  light_rgb: "🎨",
  light_colour_temp: "💡",
  blind: "🪟",
  blind_position: "🪟",
  blind_position_slat: "🪟",
  sensor: "🌡️",
  temperature_sensor: "🌡️",
  humidity_sensor: "💧",
  co2_sensor: "🌬️",
  lux_sensor: "☀️",
  presence: "👤",
  presence_detector: "👤",
  thermostat: "🌡️",
  hvac_unit: "❄️",
  energy_meter: "⚡",
  solar_inverter: "☀️",
  push_button_2: "🔘",
  push_button_4: "🔘",
  scene_controller: "🎬",
  binary_input: "⏺️",
};

// Room type options
const ROOM_TYPES = [
  { value: "living", label: "Living Room" },
  { value: "bedroom", label: "Bedroom" },
  { value: "bathroom", label: "Bathroom" },
  { value: "kitchen", label: "Kitchen" },
  { value: "hallway", label: "Hallway" },
  { value: "office", label: "Office" },
  { value: "utility", label: "Utility" },
  { value: "outdoor", label: "Outdoor" },
  { value: "garage", label: "Garage" },
  { value: "other", label: "Other" },
];

/**
 * Initialise Alpine stores.
 */
export function initStores() {
  // ─────────────────────────────────────────────────────────────
  // Main Application Store
  // ─────────────────────────────────────────────────────────────
  Alpine.store("app", {
    // Connection state
    connected: false,
    connecting: false,

    // Data
    premises: [],
    currentPremiseId: null,
    floors: [],
    currentFloorId: null,
    devices: [],
    templates: [],
    templateDomains: [],

    // UI state
    engineerMode: false,
    helpOpen: false,
    selectedDeviceId: null,
    floorPlanMode: false,
    viewMode: "building", // 'building', 'topology', or 'groups'

    // Topology data
    topology: null,
    expandedAreas: new Set(),
    expandedLines: new Set(),

    // Drag-and-drop state
    draggingDeviceId: null,
    dragOverRoomId: null,
    dragOverLineId: null,

    // Stats
    telegramCount: 0,
    tps: 0,

    // Constants
    DEVICE_ICONS,
    ROOM_TYPES,

    // ─────────────────────────────────────────────────────────
    // Getters
    // ─────────────────────────────────────────────────────────

    get currentPremise() {
      return this.premises.find((p) => p.id === this.currentPremiseId) || null;
    },

    get currentFloor() {
      return this.floors.find((f) => f.id === this.currentFloorId) || null;
    },

    get selectedDevice() {
      return this.devices.find((d) => d.id === this.selectedDeviceId) || null;
    },

    get filteredDevices() {
      if (!this.currentFloor || !this.currentFloor.rooms) {
        return this.devices;
      }
      const roomIds = this.currentFloor.rooms.map((r) => r.id);
      // Include devices assigned to rooms on this floor OR unassigned devices
      return this.devices.filter(
        (d) => !d.room_id || roomIds.includes(d.room_id),
      );
    },

    get roomsWithDevices() {
      const devices = this.filteredDevices;
      const roomMap = new Map();
      const unassigned = [];

      // First, add ALL rooms from the current floor (even empty ones)
      if (this.currentFloor && this.currentFloor.rooms) {
        for (const room of this.currentFloor.rooms) {
          roomMap.set(room.id, {
            id: room.id,
            name: room.name,
            room_type: room.room_type || "other",
            devices: [],
          });
        }
      }

      // Then assign devices to their rooms
      devices.forEach((device) => {
        if (device.room_id) {
          if (roomMap.has(device.room_id)) {
            roomMap.get(device.room_id).devices.push(device);
          } else {
            // Device assigned to room not on this floor - still show it
            roomMap.set(device.room_id, {
              id: device.room_id,
              name: device.room_id,
              room_type: "other",
              devices: [device],
            });
          }
        } else {
          unassigned.push(device);
        }
      });

      const rooms = Array.from(roomMap.values());

      // Add unassigned devices as pseudo-room
      if (unassigned.length > 0) {
        rooms.push({
          id: "_unassigned",
          name: "Unassigned",
          room_type: "other",
          devices: unassigned,
        });
      }

      return rooms;
    },

    // ─────────────────────────────────────────────────────────
    // Actions — Data Loading
    // ─────────────────────────────────────────────────────────

    async loadPremises() {
      try {
        this.premises = await API.getPremises();
        if (this.premises.length > 0 && !this.currentPremiseId) {
          await this.selectPremise(this.premises[0].id);
        }
      } catch (err) {
        console.error("Failed to load premises:", err);
      }
    },

    async selectPremise(premiseId) {
      if (!premiseId) return;
      this.currentPremiseId = premiseId;

      try {
        const [floors, devices] = await Promise.all([
          API.getFloors(premiseId),
          API.getDevices(premiseId),
        ]);

        this.floors = floors;
        this.devices = devices;

        // Select first floor or clear
        if (this.floors.length > 0) {
          this.selectFloor(this.floors[0].id);
        } else {
          this.currentFloorId = null;
        }

        // Load telegram history for this premise
        Alpine.store("telegrams").loadHistory(premiseId);

        // Load group address structure
        Alpine.store("groups").load(premiseId);
      } catch (err) {
        console.error("Failed to load premise data:", err);
      }
    },

    selectFloor(floorId) {
      this.currentFloorId = floorId;
      this.selectedDeviceId = null;
    },

    selectDevice(deviceId) {
      this.selectedDeviceId = deviceId;
    },

    clearSelection() {
      this.selectedDeviceId = null;
    },

    toggleEngineerMode() {
      this.engineerMode = !this.engineerMode;
      if (!this.engineerMode) {
        this.selectedDeviceId = null;
        Alpine.store("modal").close();
      }
    },

    toggleHelp() {
      this.helpOpen = !this.helpOpen;
    },

    // ─────────────────────────────────────────────────────────
    // Actions — View Mode & Topology
    // ─────────────────────────────────────────────────────────

    async switchView(mode) {
      this.viewMode = mode;
      this.selectedDeviceId = null;
      if (mode === "topology" && !this.topology) {
        await this.loadTopology();
      }
      if (mode === "groups" && !Alpine.store("groups").loaded) {
        await Alpine.store("groups").load(this.currentPremiseId);
      }
    },

    async loadTopology() {
      if (!this.currentPremiseId) return;
      try {
        this.topology = await API.getTopology(this.currentPremiseId);
        // Expand all areas by default
        this.expandedAreas = new Set(
          this.topology.areas.map((a) => a.id)
        );
        // Expand all lines by default
        this.expandedLines = new Set(
          this.topology.areas.flatMap((a) => a.lines.map((l) => l.id))
        );
      } catch (err) {
        console.error("Failed to load topology:", err);
      }
    },

    toggleArea(areaId) {
      if (this.expandedAreas.has(areaId)) {
        this.expandedAreas.delete(areaId);
      } else {
        this.expandedAreas.add(areaId);
      }
      // Trigger reactivity
      this.expandedAreas = new Set(this.expandedAreas);
    },

    toggleLine(lineId) {
      if (this.expandedLines.has(lineId)) {
        this.expandedLines.delete(lineId);
      } else {
        this.expandedLines.add(lineId);
      }
      // Trigger reactivity
      this.expandedLines = new Set(this.expandedLines);
    },

    isAreaExpanded(areaId) {
      return this.expandedAreas.has(areaId);
    },

    isLineExpanded(lineId) {
      return this.expandedLines.has(lineId);
    },

    async createArea(data) {
      if (!this.currentPremiseId) return;
      try {
        await API.createArea(this.currentPremiseId, data);
        await this.loadTopology();
        Alpine.store("modal").close();
      } catch (err) {
        console.error("Failed to create area:", err);
        alert("Failed to create area: " + err.message);
      }
    },

    async updateArea(areaId, data) {
      if (!this.currentPremiseId) return;
      try {
        await API.updateArea(this.currentPremiseId, areaId, data);
        await this.loadTopology();
        Alpine.store("modal").close();
      } catch (err) {
        console.error("Failed to update area:", err);
        alert("Failed to update area: " + err.message);
      }
    },

    async deleteArea(areaId) {
      if (!this.currentPremiseId) return;
      try {
        await API.deleteArea(this.currentPremiseId, areaId);
        await this.loadTopology();
      } catch (err) {
        console.error("Failed to delete area:", err);
        alert("Failed to delete area: " + err.message);
      }
    },

    async createLine(areaId, data) {
      if (!this.currentPremiseId) return;
      try {
        await API.createLine(this.currentPremiseId, areaId, data);
        await this.loadTopology();
        Alpine.store("modal").close();
      } catch (err) {
        console.error("Failed to create line:", err);
        alert("Failed to create line: " + err.message);
      }
    },

    async updateLine(areaId, lineId, data) {
      if (!this.currentPremiseId) return;
      try {
        await API.updateLine(this.currentPremiseId, areaId, lineId, data);
        await this.loadTopology();
        Alpine.store("modal").close();
      } catch (err) {
        console.error("Failed to update line:", err);
        alert("Failed to update line: " + err.message);
      }
    },

    async deleteLine(areaId, lineId) {
      if (!this.currentPremiseId) return;
      try {
        await API.deleteLine(this.currentPremiseId, areaId, lineId);
        await this.loadTopology();
      } catch (err) {
        console.error("Failed to delete line:", err);
        alert("Failed to delete line: " + err.message);
      }
    },

    // Move device to a different line (topology)
    async moveDeviceToLine(deviceId, lineId, deviceNumber) {
      if (!this.currentPremiseId) return;
      try {
        await API.updateDevice(this.currentPremiseId, deviceId, {
          line_id: lineId,
          device_number: deviceNumber,
        });
        // Reload both topology and devices
        await Promise.all([
          this.loadTopology(),
          API.getDevices(this.currentPremiseId).then((d) => (this.devices = d)),
        ]);
      } catch (err) {
        console.error("Failed to move device:", err);
        alert("Failed to move device: " + err.message);
      }
    },

    // Get next available device number on a line
    getNextDeviceNumber(lineId) {
      if (!this.topology) return 1;
      for (const area of this.topology.areas) {
        const line = area.lines.find((l) => l.id === lineId);
        if (line) {
          const usedNumbers = line.devices.map((d) => d.device_number || 0);
          for (let i = 1; i <= 255; i++) {
            if (!usedNumbers.includes(i)) return i;
          }
        }
      }
      return 1;
    },

    // ─────────────────────────────────────────────────────────
    // Actions — Drag and Drop
    // ─────────────────────────────────────────────────────────

    startDragDevice(event, deviceId) {
      this.draggingDeviceId = deviceId;
      event.dataTransfer.setData("text/plain", deviceId);
      event.dataTransfer.effectAllowed = "move";
    },

    endDragDevice() {
      this.draggingDeviceId = null;
      this.dragOverRoomId = null;
    },

    async dropDeviceOnRoom(event, roomId) {
      event.preventDefault();
      const deviceId = event.dataTransfer.getData("text/plain");
      this.dragOverRoomId = null;
      this.draggingDeviceId = null;

      if (!deviceId) return;

      // Find the device
      const device = this.devices.find((d) => d.id === deviceId);
      if (!device) return;

      // Skip if dropping on same room
      if (device.room_id === roomId) return;

      try {
        // Update via API
        await API.updateDevice(this.currentPremiseId, deviceId, {
          room_id: roomId,
        });

        // Update local state
        device.room_id = roomId;
        console.log(
          `Moved device ${deviceId} to room ${roomId || "unassigned"}`,
        );
      } catch (err) {
        console.error("Failed to move device:", err);
      }
    },

    // ─────────────────────────────────────────────────────────
    // Actions — Device Commands
    // ─────────────────────────────────────────────────────────

    async sendCommand(deviceId, command, value) {
      if (!this.currentPremiseId) return;
      try {
        await API.sendCommand(this.currentPremiseId, deviceId, command, value);
      } catch (err) {
        console.error("Command failed:", err);
      }
    },

    /**
     * Send command to a specific channel of a multi-channel device
     */
    async sendChannelCommand(deviceId, channelId, command, value) {
      if (!this.currentPremiseId) return;
      try {
        // For now, we send to the device and the backend will route to the channel
        // In future, we could add a channel-specific endpoint
        await API.sendChannelCommand(this.currentPremiseId, deviceId, channelId, command, value);
      } catch (err) {
        console.error("Channel command failed:", err);
      }
    },

    /**
     * Toggle a device on/off (inverts current state)
     */
    async toggleDevice(deviceId, command, currentState) {
      await this.sendCommand(deviceId, command, !currentState);
    },

    /**
     * Press/release a push button (for wall switches) - momentary action
     */
    async pressButton(deviceId, buttonName, pressed) {
      await this.sendCommand(deviceId, buttonName, pressed);
    },

    /**
     * Toggle a push button on/off (latching action for simulation)
     */
    async toggleButton(deviceId, buttonName, currentState) {
      await this.sendCommand(deviceId, buttonName, !currentState);
    },

    /**
     * Check if a device is currently "on" (for visual feedback)
     */
    isDeviceOn(device) {
      const state = device.state || {};
      const type = device.type;

      if (type.startsWith("light_")) {
        return state.on === true;
      }
      if (
        type === "presence" ||
        type === "presence_detector" ||
        type === "presence_sensor"
      ) {
        return state.presence === true;
      }
      return false;
    },

    /**
     * Get button group addresses from a template device (for wall switches)
     */
    getButtonGAs(device) {
      const gas = device.group_addresses || {};
      const buttons = {};
      for (const [name, ga] of Object.entries(gas)) {
        if (name.startsWith("button_")) {
          buttons[name] = ga;
        }
      }
      return buttons;
    },

    /**
     * Get sensor display value
     */
    getSensorDisplay(device) {
      const state = device.state || {};
      if (state.temperature !== undefined) {
        return `${state.temperature.toFixed(1)}°C`;
      }
      if (state.humidity !== undefined) {
        return `${state.humidity.toFixed(0)}%`;
      }
      if (state.lux !== undefined) {
        return `${Math.round(state.lux)} lx`;
      }
      return "-";
    },

    // ─────────────────────────────────────────────────────────
    // Actions — Floor CRUD
    // ─────────────────────────────────────────────────────────

    async createFloor(data) {
      if (!this.currentPremiseId) return;
      try {
        const floor = await API.createFloor(this.currentPremiseId, data);
        floor.rooms = [];
        this.floors.push(floor);
        return floor;
      } catch (err) {
        console.error("Failed to create floor:", err);
        throw err;
      }
    },

    async updateFloor(floorId, data) {
      if (!this.currentPremiseId) return;
      try {
        const updated = await API.updateFloor(
          this.currentPremiseId,
          floorId,
          data,
        );
        const idx = this.floors.findIndex((f) => f.id === floorId);
        if (idx !== -1) {
          this.floors[idx] = { ...this.floors[idx], ...updated };
        }
        return updated;
      } catch (err) {
        console.error("Failed to update floor:", err);
        throw err;
      }
    },

    async deleteFloor(floorId) {
      if (!this.currentPremiseId) return;
      try {
        await API.deleteFloor(this.currentPremiseId, floorId);
        this.floors = this.floors.filter((f) => f.id !== floorId);
        if (this.currentFloorId === floorId) {
          this.currentFloorId = this.floors[0]?.id || null;
        }
        // Reload devices (room assignments may have changed)
        this.devices = await API.getDevices(this.currentPremiseId);
      } catch (err) {
        console.error("Failed to delete floor:", err);
        throw err;
      }
    },

    // ─────────────────────────────────────────────────────────
    // Actions — Room CRUD
    // ─────────────────────────────────────────────────────────

    async createRoom(floorId, data) {
      if (!this.currentPremiseId) return;
      try {
        const room = await API.createRoom(this.currentPremiseId, floorId, data);
        const floor = this.floors.find((f) => f.id === floorId);
        if (floor) {
          if (!floor.rooms) floor.rooms = [];
          floor.rooms.push(room);
        }
        return room;
      } catch (err) {
        console.error("Failed to create room:", err);
        throw err;
      }
    },

    async updateRoom(floorId, roomId, data) {
      if (!this.currentPremiseId) return;
      try {
        const updated = await API.updateRoom(
          this.currentPremiseId,
          floorId,
          roomId,
          data,
        );
        const floor = this.floors.find((f) => f.id === floorId);
        if (floor && floor.rooms) {
          const idx = floor.rooms.findIndex((r) => r.id === roomId);
          if (idx !== -1) {
            floor.rooms[idx] = { ...floor.rooms[idx], ...updated };
          }
        }
        return updated;
      } catch (err) {
        console.error("Failed to update room:", err);
        throw err;
      }
    },

    async deleteRoom(floorId, roomId) {
      if (!this.currentPremiseId) return;
      try {
        await API.deleteRoom(this.currentPremiseId, floorId, roomId);
        const floor = this.floors.find((f) => f.id === floorId);
        if (floor && floor.rooms) {
          floor.rooms = floor.rooms.filter((r) => r.id !== roomId);
        }
        // Reload devices (assignments cleared)
        this.devices = await API.getDevices(this.currentPremiseId);
      } catch (err) {
        console.error("Failed to delete room:", err);
        throw err;
      }
    },

    // ─────────────────────────────────────────────────────────
    // Actions — Device CRUD
    // ─────────────────────────────────────────────────────────

    async loadTemplates() {
      try {
        const templatesResponse = await API.getTemplates();
        const domainsResponse = await API.getDomains();
        // API returns { templates: [...] } and { domains: [...] }
        this.templates = templatesResponse.templates || [];
        this.templateDomains = domainsResponse.domains || [];
      } catch (err) {
        console.error("Failed to load templates:", err);
      }
    },

    async createDevice(data) {
      if (!this.currentPremiseId) return;
      try {
        const device = await API.createDevice(this.currentPremiseId, data);
        this.devices.push(device);
        return device;
      } catch (err) {
        console.error("Failed to create device:", err);
        throw err;
      }
    },

    async createDeviceFromTemplate(templateId, data) {
      if (!this.currentPremiseId) return;
      try {
        const device = await API.createDeviceFromTemplate(
          this.currentPremiseId,
          templateId,
          data,
        );
        this.devices.push(device);
        return device;
      } catch (err) {
        console.error("Failed to create device from template:", err);
        throw err;
      }
    },

    async updateDevice(deviceId, data) {
      if (!this.currentPremiseId) return;
      try {
        const updated = await API.updateDevice(
          this.currentPremiseId,
          deviceId,
          data,
        );
        const idx = this.devices.findIndex((d) => d.id === deviceId);
        if (idx !== -1) {
          this.devices[idx] = { ...this.devices[idx], ...updated };
        }
        return updated;
      } catch (err) {
        console.error("Failed to update device:", err);
        throw err;
      }
    },

    async deleteDevice(deviceId) {
      if (!this.currentPremiseId) return;
      try {
        await API.deleteDevice(this.currentPremiseId, deviceId);
        this.devices = this.devices.filter((d) => d.id !== deviceId);
        if (this.selectedDeviceId === deviceId) {
          this.selectedDeviceId = null;
        }
      } catch (err) {
        console.error("Failed to delete device:", err);
        throw err;
      }
    },

    async assignDeviceToRoom(deviceId, roomId) {
      if (!this.currentPremiseId) return;
      try {
        await API.setDevicePlacement(this.currentPremiseId, deviceId, roomId);
        const device = this.devices.find((d) => d.id === deviceId);
        if (device) {
          device.room_id = roomId;
        }
      } catch (err) {
        console.error("Failed to assign device:", err);
        throw err;
      }
    },

    // ─────────────────────────────────────────────────────────
    // Actions — State Updates (from WebSocket)
    // ─────────────────────────────────────────────────────────

    updateDeviceState(deviceId, state) {
      const device = this.devices.find((d) => d.id === deviceId);
      if (device) {
        device.state = state;
      }
    },

    setConnected(connected) {
      this.connected = connected;
      this.connecting = false;
    },

    setConnecting() {
      this.connecting = true;
      this.connected = false;
    },

    incrementTelegramCount() {
      this.telegramCount++;
    },

    // ─────────────────────────────────────────────────────────
    // Helpers
    // ─────────────────────────────────────────────────────────

    getDeviceIcon(type) {
      return DEVICE_ICONS[type] || "📦";
    },

    formatDeviceName(id) {
      return id
        .replace(/-/g, " ")
        .replace(/_/g, " ")
        .split(" ")
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
        .join(" ");
    },

    getStateDisplay(device) {
      const state = device.state || {};
      const type = device.type;

      // Light devices
      if (type.startsWith("light_")) {
        const isOn = state.on === true;
        const brightness = state.brightness;
        if (brightness !== undefined && isOn) {
          return { indicator: isOn, value: `${brightness}%` };
        }
        return { indicator: isOn, value: isOn ? "ON" : "OFF" };
      }

      // Blind devices
      if (type.startsWith("blind")) {
        const position = state.position ?? 0;
        return { indicator: null, value: `${position}%` };
      }

      // Presence
      if (type === "presence" || type === "presence_detector") {
        const present = state.presence === true;
        return { indicator: present, value: present ? "OCCUPIED" : "EMPTY" };
      }

      // Temperature sensors
      if (type === "sensor" || type === "temperature_sensor") {
        const temp = state.temperature;
        if (temp !== undefined) {
          return { indicator: null, value: `${temp.toFixed(1)}°C` };
        }
      }

      // Thermostat
      if (type === "thermostat") {
        const current = state.current_temperature;
        if (current !== undefined) {
          return { indicator: null, value: `${current.toFixed(1)}°C` };
        }
      }

      // Default
      for (const [key, value] of Object.entries(state)) {
        if (typeof value === "boolean") {
          return { indicator: value, value: value ? "ON" : "OFF" };
        }
        if (typeof value === "number") {
          return { indicator: null, value: String(value) };
        }
      }

      return { indicator: null, value: "-" };
    },

    getRoomStatusSummary(devices) {
      let lightsOn = 0;
      let hasPresence = false;
      let temp = null;

      devices.forEach((d) => {
        const state = d.state || {};
        if (d.type.startsWith("light_") && state.on) lightsOn++;
        if (
          (d.type === "presence" || d.type === "presence_detector") &&
          state.presence
        )
          hasPresence = true;
        if (
          (d.type === "sensor" || d.type === "temperature_sensor") &&
          state.temperature !== undefined
        ) {
          temp = state.temperature;
        }
      });

      const parts = [];
      if (lightsOn > 0) parts.push(`💡${lightsOn}`);
      if (hasPresence) parts.push("👤");
      if (temp !== null) parts.push(`${temp.toFixed(1)}°`);

      return parts.join(" ");
    },

    guessDPT(gaName, deviceType) {
      const name = gaName.toLowerCase();
      // Switch/toggle types (DPT 1.001)
      if (name.includes("switch") || name.includes("on_off")) return "1.001";
      // Push buttons and LEDs (DPT 1.001 - boolean)
      if (name.includes("button") || name.includes("led")) return "1.001";
      // Percentage values (DPT 5.001)
      if (
        name.includes("brightness") ||
        name.includes("position") ||
        name.includes("slat")
      )
        return "5.001";
      // Temperature (DPT 9.001)
      if (name.includes("temperature") || name.includes("temp")) return "9.001";
      // Humidity (DPT 9.007)
      if (name.includes("humidity")) return "9.007";
      // Lux (DPT 9.004)
      if (name.includes("lux")) return "9.004";
      // Presence (DPT 1.018)
      if (name.includes("presence") || name.includes("occupancy"))
        return "1.018";
      // Scene (DPT 17.001)
      if (name.includes("scene")) return "17.001";
      // HVAC mode (DPT 20.102)
      if (name.includes("hvac") || name.includes("mode")) return "20.102";
      // Power (DPT 14.056)
      if (name.includes("power")) return "14.056";
      // Energy (DPT 13.010)
      if (name.includes("energy")) return "13.010";
      return "-";
    },

    // Guess KNX communication flags based on GA name
    guessFlags(gaName) {
      const name = gaName.toLowerCase();
      // Status/feedback objects - read + transmit
      if (name.includes("status") || name.includes("feedback")) {
        return "CR-T--";
      }
      // Buttons/switches that transmit - transmit only
      if (name.includes("button") || name.includes("press")) {
        return "C--T--";
      }
      // Command objects - write + update
      if (name.includes("cmd") || name.includes("command")) {
        return "C-W-U-";
      }
      // Sensor readings - read + transmit
      if (
        name.includes("temperature") ||
        name.includes("humidity") ||
        name.includes("lux") ||
        name.includes("sensor")
      ) {
        return "CR-T--";
      }
      // LED feedback - write + update (receives status)
      if (name.includes("led")) {
        return "C-W-U-";
      }
      // Default: write + update (typical for command inputs)
      return "C-W-U-";
    },

    // Get tooltip explaining what each flag means
    getFlagsTooltip(flags) {
      if (!flags || flags === "-") return "No flags set";
      const explanations = [];
      if (flags[0] === "C") explanations.push("C: Communication enabled");
      if (flags[1] === "R") explanations.push("R: Can be read");
      if (flags[2] === "W") explanations.push("W: Can be written");
      if (flags[3] === "T") explanations.push("T: Transmits on change");
      if (flags[4] === "U") explanations.push("U: Updates from bus");
      if (flags[5] === "I") explanations.push("I: Reads at startup");
      return explanations.join("\n") || "No flags set";
    },
  });

  // ─────────────────────────────────────────────────────────────
  // Modal Store
  // ─────────────────────────────────────────────────────────────
  Alpine.store("modal", {
    open: false,
    type: null, // 'floor', 'room', 'device', 'template-browser', 'confirm'
    mode: null, // 'create', 'edit'
    data: {},
    onConfirm: null,

    show(type, mode = "create", data = {}) {
      this.type = type;
      this.mode = mode;
      this.data = data;
      this.open = true;
    },

    close() {
      this.open = false;
      this.type = null;
      this.mode = null;
      this.data = {};
      this.onConfirm = null;
    },

    confirm(title, message, onConfirm) {
      this.type = "confirm";
      this.data = { title, message };
      this.onConfirm = onConfirm;
      this.open = true;
    },

    async doConfirm() {
      if (this.onConfirm) {
        await this.onConfirm();
      }
      this.close();
    },
  });

  // ─────────────────────────────────────────────────────────────
  // Telegram Store
  // ─────────────────────────────────────────────────────────────
  Alpine.store("telegrams", {
    items: [],
    buffer: [], // Buffer for telegrams while paused
    filter: "all", // 'all', 'rx', 'tx'
    paused: false,
    maxItems: 200,
    maxBuffer: 500, // Max buffered while paused
    loading: false,
    hasMore: false,
    totalBuffered: 0, // Total in backend ring buffer

    _formatTelegram(telegram) {
      // Format timestamp as HH:MM:SS.mmm
      let timeFormatted = "-";
      if (telegram.timestamp) {
        const date = new Date(telegram.timestamp * 1000);
        const hh = String(date.getHours()).padStart(2, "0");
        const mm = String(date.getMinutes()).padStart(2, "0");
        const ss = String(date.getSeconds()).padStart(2, "0");
        const ms = String(date.getMilliseconds()).padStart(3, "0");
        timeFormatted = `${hh}:${mm}:${ss}.${ms}`;
      }
      return {
        ...telegram,
        id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
        time_formatted: timeFormatted,
      };
    },

    add(telegram) {
      const formatted = this._formatTelegram(telegram);

      if (this.paused) {
        // Buffer while paused (newest at start)
        this.buffer.unshift(formatted);
        if (this.buffer.length > this.maxBuffer) {
          this.buffer = this.buffer.slice(0, this.maxBuffer);
        }
        return;
      }

      this.items.unshift(formatted);

      // Trim to max
      if (this.items.length > this.maxItems) {
        this.items = this.items.slice(0, this.maxItems);
      }

      Alpine.store("app").incrementTelegramCount();
    },

    get filtered() {
      if (this.filter === "all") return this.items;
      return this.items.filter((t) => t.direction === this.filter);
    },

    async loadHistory(premiseId) {
      this.loading = true;
      try {
        const response = await API.getTelegrams(premiseId, this.maxItems);
        this.totalBuffered = response.total_buffered || 0;
        if (response.telegrams && response.telegrams.length > 0) {
          // Format and load history (already newest-first from API)
          this.items = response.telegrams.map((t) => this._formatTelegram(t));
          // Check if there's more to load
          this.hasMore = this.items.length < this.totalBuffered;
          console.log(
            `Loaded ${this.items.length}/${this.totalBuffered} telegram(s) from history`,
          );
        } else {
          this.hasMore = false;
        }
      } catch (err) {
        console.error("Failed to load telegram history:", err);
      } finally {
        this.loading = false;
      }
    },

    async loadMore(premiseId) {
      if (this.loading || !this.hasMore) return;

      this.loading = true;
      try {
        // Load next batch, offset by current count
        const response = await API.getTelegrams(
          premiseId,
          this.maxItems,
          this.items.length,
        );
        this.totalBuffered = response.total_buffered || 0;

        if (response.telegrams && response.telegrams.length > 0) {
          // Append older telegrams to end
          const older = response.telegrams.map((t) => this._formatTelegram(t));
          this.items = [...this.items, ...older];
          this.hasMore = this.items.length < this.totalBuffered;
          console.log(
            `Loaded ${older.length} more, total: ${this.items.length}/${this.totalBuffered}`,
          );
        } else {
          this.hasMore = false;
        }
      } catch (err) {
        console.error("Failed to load more telegrams:", err);
      } finally {
        this.loading = false;
      }
    },

    get bufferedCount() {
      return this.buffer.length;
    },

    setFilter(filter) {
      this.filter = filter;
    },

    togglePause() {
      this.paused = !this.paused;

      // When resuming, flush buffer to items
      if (!this.paused && this.buffer.length > 0) {
        // Prepend buffered items (they're already newest-first)
        this.items = [...this.buffer, ...this.items].slice(0, this.maxItems);
        this.buffer = [];
        console.log(
          `Resumed: flushed ${this.buffer.length} buffered telegram(s)`,
        );
      }

      return this.paused;
    },

    clear() {
      this.items = [];
    },

    copyToClipboard() {
      const telegrams = this.filtered;
      if (telegrams.length === 0) {
        return;
      }

      // Format as TSV (tab-separated) for easy pasting into spreadsheets
      const header = [
        "Time",
        "Dir",
        "APCI",
        "Group Address",
        "Device",
        "Payload",
        "Decoded",
        "DPT",
      ].join("\t");
      const rows = telegrams.map((t) =>
        [
          t.time_formatted || "",
          t.direction?.toUpperCase() || "",
          t.apci || "",
          t.destination + (t.ga_function ? ` (${t.ga_function})` : ""),
          t.device_id || "",
          t.payload || "",
          t.decoded_value !== undefined && t.decoded_value !== null
            ? t.decoded_value + (t.unit ? " " + t.unit : "")
            : "",
          t.dpt || "",
        ].join("\t"),
      );

      const text = [header, ...rows].join("\n");

      // Use execCommand fallback - works on HTTP (Clipboard API needs HTTPS)
      const textarea = document.createElement("textarea");
      textarea.value = text;
      textarea.style.position = "fixed";
      textarea.style.left = "-9999px";
      document.body.appendChild(textarea);
      textarea.select();

      try {
        document.execCommand("copy");
        console.log(`Copied ${telegrams.length} telegram(s) to clipboard`);
      } catch (err) {
        console.error("Failed to copy to clipboard:", err);
      } finally {
        document.body.removeChild(textarea);
      }
    },
  });

  // ─────────────────────────────────────────────────────────────
  // Groups Store (GA Hierarchy)
  // ─────────────────────────────────────────────────────────────
  Alpine.store("groups", {
    loaded: false,
    loading: false,
    mainGroups: [],
    usedAddresses: [],

    // Expanded state for tree view
    expandedMainGroups: new Set(),
    expandedMiddleGroups: new Set(),

    isMainGroupExpanded(id) {
      return this.expandedMainGroups.has(id);
    },

    toggleMainGroup(id) {
      if (this.expandedMainGroups.has(id)) {
        this.expandedMainGroups.delete(id);
      } else {
        this.expandedMainGroups.add(id);
      }
    },

    isMiddleGroupExpanded(id) {
      return this.expandedMiddleGroups.has(id);
    },

    toggleMiddleGroup(id) {
      if (this.expandedMiddleGroups.has(id)) {
        this.expandedMiddleGroups.delete(id);
      } else {
        this.expandedMiddleGroups.add(id);
      }
    },

    // Get main group by number
    getMainGroupByNumber(num) {
      return this.mainGroups.find(g => g.group_number === num);
    },

    // Get middle group by main group ID and number
    getMiddleGroupByNumber(mainGroupId, num) {
      const main = this.mainGroups.find(g => g.id === mainGroupId);
      if (!main || !main.middle_groups) return null;
      return main.middle_groups.find(g => g.group_number === num);
    },

    // Check if a GA is used
    isGaUsed(ga) {
      return this.usedAddresses.includes(ga);
    },

    // Get devices using a specific GA
    getDevicesUsingGa(ga) {
      const devices = Alpine.store("app").devices || [];
      const result = [];
      for (const dev of devices) {
        for (const [name, gaData] of Object.entries(dev.group_addresses || {})) {
          const gaStr = typeof gaData === "object" ? gaData.ga : gaData;
          if (gaStr === ga) {
            result.push({ device: dev, function: name });
          }
        }
      }
      return result;
    },

    // Get all GAs in a main/middle group with their device mappings
    getGAsInGroup(mainNum, middleNum) {
      const prefix = `${mainNum}/${middleNum}/`;
      const devices = Alpine.store("app").devices || [];
      const gas = [];

      for (const dev of devices) {
        for (const [name, gaData] of Object.entries(dev.group_addresses || {})) {
          const gaStr = typeof gaData === "object" ? gaData.ga : gaData;
          if (gaStr && gaStr.startsWith(prefix)) {
            gas.push({
              ga: gaStr,
              sub: parseInt(gaStr.split("/")[2], 10),
              device: dev,
              function: name,
              dpt: typeof gaData === "object" ? gaData.dpt : null,
            });
          }
        }
      }

      // Sort by sub-address
      return gas.sort((a, b) => a.sub - b.sub);
    },

    async load(premiseId) {
      if (this.loading) return;
      this.loading = true;

      try {
        const data = await API.getGroupTree(premiseId);
        this.mainGroups = data.main_groups || [];
        this.usedAddresses = data.used_addresses || [];
        this.loaded = true;
        console.log("Groups loaded:", this.mainGroups.length, "main groups");
      } catch (err) {
        console.error("Failed to load groups:", err);
      } finally {
        this.loading = false;
      }
    },

    async createDefaults(premiseId) {
      try {
        await API.createDefaultGroups(premiseId);
        await this.load(premiseId);
      } catch (err) {
        console.error("Failed to create default groups:", err);
        throw err;
      }
    },

    async deleteAll(premiseId) {
      try {
        await API.deleteAllGroups(premiseId);
        this.mainGroups = [];
        this.loaded = false;
      } catch (err) {
        console.error("Failed to delete all groups:", err);
        throw err;
      }
    },

    async createMainGroup(premiseId, data) {
      try {
        await API.createMainGroup(premiseId, data);
        await this.load(premiseId);
      } catch (err) {
        console.error("Failed to create main group:", err);
        throw err;
      }
    },

    async updateMainGroup(mainGroupId, data) {
      try {
        await API.updateMainGroup(mainGroupId, data);
        const premiseId = Alpine.store("app").currentPremiseId;
        if (premiseId) await this.load(premiseId);
      } catch (err) {
        console.error("Failed to update main group:", err);
        throw err;
      }
    },

    async deleteMainGroup(mainGroupId) {
      try {
        await API.deleteMainGroup(mainGroupId);
        const premiseId = Alpine.store("app").currentPremiseId;
        if (premiseId) await this.load(premiseId);
      } catch (err) {
        console.error("Failed to delete main group:", err);
        throw err;
      }
    },

    async createMiddleGroup(mainGroupId, data) {
      try {
        await API.createMiddleGroup(mainGroupId, data);
        const premiseId = Alpine.store("app").currentPremiseId;
        if (premiseId) await this.load(premiseId);
      } catch (err) {
        console.error("Failed to create middle group:", err);
        throw err;
      }
    },

    async updateMiddleGroup(middleGroupId, data) {
      try {
        await API.updateMiddleGroup(middleGroupId, data);
        const premiseId = Alpine.store("app").currentPremiseId;
        if (premiseId) await this.load(premiseId);
      } catch (err) {
        console.error("Failed to update middle group:", err);
        throw err;
      }
    },

    async deleteMiddleGroup(middleGroupId) {
      try {
        await API.deleteMiddleGroup(middleGroupId);
        const premiseId = Alpine.store("app").currentPremiseId;
        if (premiseId) await this.load(premiseId);
      } catch (err) {
        console.error("Failed to delete middle group:", err);
        throw err;
      }
    },

    async suggestGAs(premiseId, deviceType, roomId = null) {
      try {
        return await API.suggestGroupAddresses(premiseId, deviceType, roomId);
      } catch (err) {
        console.error("Failed to suggest GAs:", err);
        return null;
      }
    },
  });

  // ─────────────────────────────────────────────────────────────
  // Reference Data Store
  // ─────────────────────────────────────────────────────────────
  Alpine.store("reference", {
    loaded: false,
    loading: false,
    individualAddress: null,
    gaStructure: null,
    flags: null,
    dpts: null,
    deviceTemplates: null,

    // Flattened DPT list for autocomplete
    get dptList() {
      if (!this.dpts?.categories) return [];
      const list = [];
      for (const cat of this.dpts.categories) {
        for (const dpt of cat.dpts) {
          list.push({
            id: dpt.id,
            name: dpt.name,
            category: cat.name,
            unit: dpt.unit || "",
            range: dpt.range || dpt.values || "",
            use_case: dpt.use_case || "",
          });
        }
      }
      return list;
    },

    // Search DPTs by id or name
    searchDpts(query) {
      if (!query) return this.dptList.slice(0, 20);
      const q = query.toLowerCase();
      return this.dptList.filter(
        (d) =>
          d.id.toLowerCase().includes(q) ||
          d.name.toLowerCase().includes(q) ||
          d.use_case.toLowerCase().includes(q)
      );
    },

    // Get recommended GAs for a device type
    getRecommendedGAs(deviceType) {
      if (!this.deviceTemplates) return null;
      return this.deviceTemplates[deviceType];
    },

    // Get existing devices of the same type (for reference)
    getSimilarDevices(deviceType) {
      const devices = Alpine.store("app").devices || [];
      return devices.filter((d) => d.type === deviceType);
    },

    // Get ALL devices for reference
    getAllDevices() {
      return Alpine.store("app").devices || [];
    },

    // Parse individual address string to components
    parseIA(addr) {
      if (!addr) return null;
      const parts = addr.split(".");
      if (parts.length !== 3) return null;
      return {
        area: parseInt(parts[0], 10),
        line: parseInt(parts[1], 10),
        device: parseInt(parts[2], 10),
      };
    },

    // Format IA components to string
    formatIA(area, line, device) {
      return `${area}.${line}.${device}`;
    },

    // Get all IAs in use
    getAllUsedIAs() {
      const devices = this.getAllDevices();
      return devices
        .map((d) => d.individual_address)
        .filter((ia) => ia)
        .sort((a, b) => {
          const pa = this.parseIA(a);
          const pb = this.parseIA(b);
          if (!pa || !pb) return 0;
          if (pa.area !== pb.area) return pa.area - pb.area;
          if (pa.line !== pb.line) return pa.line - pb.line;
          return pa.device - pb.device;
        });
    },

    // Get devices grouped by area.line
    getDevicesByLine() {
      const devices = this.getAllDevices();
      const byLine = {};
      for (const dev of devices) {
        const parsed = this.parseIA(dev.individual_address);
        if (!parsed) continue;
        const key = `${parsed.area}.${parsed.line}`;
        if (!byLine[key]) {
          byLine[key] = { area: parsed.area, line: parsed.line, devices: [] };
        }
        byLine[key].devices.push({
          id: dev.id,
          device: parsed.device,
          type: dev.type,
          address: dev.individual_address,
        });
      }
      // Sort devices within each line
      for (const key of Object.keys(byLine)) {
        byLine[key].devices.sort((a, b) => a.device - b.device);
      }
      return byLine;
    },

    // Suggest next available IA on a given line
    suggestNextIA(areaLine = "1.1") {
      const [area, line] = areaLine.split(".").map((x) => parseInt(x, 10) || 1);
      const devices = this.getAllDevices();

      // Find max device number on this line
      let maxDevice = 0;
      for (const dev of devices) {
        const parsed = this.parseIA(dev.individual_address);
        if (parsed && parsed.area === area && parsed.line === line) {
          maxDevice = Math.max(maxDevice, parsed.device);
        }
      }

      // Suggest next (skip 0 which is for couplers)
      const next = maxDevice < 255 ? maxDevice + 1 : maxDevice;
      return this.formatIA(area, line, Math.max(1, next));
    },

    // Get all GAs in use across all devices
    getAllUsedGAs() {
      const devices = this.getAllDevices();
      const gas = new Map(); // ga string -> { devices: [], names: [] }

      for (const dev of devices) {
        for (const [name, gaData] of Object.entries(dev.group_addresses || {})) {
          const gaStr = typeof gaData === "object" ? gaData.ga : gaData;
          if (!gaStr) continue;

          if (!gas.has(gaStr)) {
            gas.set(gaStr, { ga: gaStr, devices: [], usages: [] });
          }
          const entry = gas.get(gaStr);
          if (!entry.devices.includes(dev.id)) {
            entry.devices.push(dev.id);
          }
          entry.usages.push({ device: dev.id, name, dpt: gaData?.dpt, flags: gaData?.flags });
        }
      }

      // Convert to sorted array
      return Array.from(gas.values()).sort((a, b) => {
        // Sort by GA numerically
        const parseGA = (s) => {
          const p = s.split("/").map((x) => parseInt(x, 10));
          return (p[0] || 0) * 65536 + (p[1] || 0) * 256 + (p[2] || 0);
        };
        return parseGA(a.ga) - parseGA(b.ga);
      });
    },

    // Suggest next available GA in a given main/middle group
    suggestNextGA(mainMiddle = "1/1") {
      const [main, middle] = mainMiddle.split("/").map((x) => parseInt(x, 10) || 0);
      const usedGAs = this.getAllUsedGAs();

      // Find max sub address in this main/middle group
      let maxSub = 0;
      for (const entry of usedGAs) {
        const parts = entry.ga.split("/").map((x) => parseInt(x, 10));
        if (parts.length >= 3 && parts[0] === main && parts[1] === middle) {
          maxSub = Math.max(maxSub, parts[2]);
        }
      }

      const next = maxSub < 255 ? maxSub + 1 : maxSub;
      return `${main}/${middle}/${next}`;
    },

    async load() {
      if (this.loaded || this.loading) return;
      this.loading = true;
      try {
        const data = await API.getReference();
        this.individualAddress = data.individual_address;
        this.gaStructure = data.ga_structure;
        this.flags = data.flags;
        this.dpts = data.dpts;
        this.deviceTemplates = data.device_templates;
        this.loaded = true;
        console.log("Reference data loaded");
      } catch (err) {
        console.error("Failed to load reference data:", err);
      } finally {
        this.loading = false;
      }
    },
  });
}
