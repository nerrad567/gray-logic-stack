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
    selectedDeviceId: null,
    floorPlanMode: false,

    // Drag-and-drop state
    draggingDeviceId: null,
    dragOverRoomId: null,

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
      if (name.includes("presence") || name.includes("occupancy"))
        return "1.018";
      if (name.includes("scene")) return "17.001";
      if (name.includes("hvac") || name.includes("mode")) return "20.102";
      if (name.includes("power")) return "14.056";
      if (name.includes("energy")) return "13.010";
      return "-";
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
        id: Date.now() + Math.random(),
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
}
