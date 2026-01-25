/**
 * KNXSim Dashboard — Main Application
 *
 * Vanilla JS with ES6 modules. No build step required.
 */

import { API } from './api.js';
import { WebSocketManager } from './websocket.js';
import { RoomGrid } from './components/room-grid.js';
import { TelegramInspector } from './components/telegram-inspector.js';
import { DevicePanel } from './components/device-panel.js';

class KNXSimApp {
    constructor() {
        // State
        this.premises = [];
        this.currentPremise = null;
        this.floors = [];
        this.currentFloor = null;
        this.devices = [];
        this.engineerMode = false;
        this.selectedDevice = null;

        // Stats
        this.telegramCount = 0;
        this.lastTelegramTime = Date.now();
        this.tps = 0;

        // DOM Elements
        this.els = {
            premiseSelector: document.getElementById('premise-selector'),
            connectionStatus: document.getElementById('connection-status'),
            engineerModeBtn: document.getElementById('engineer-mode-btn'),
            floorTabs: document.querySelector('.floor-tabs'),
            roomGrid: document.getElementById('room-grid'),
            devicePanel: document.getElementById('device-panel'),
            telegramFeed: document.getElementById('telegram-feed'),
            telegramFilter: document.getElementById('telegram-filter'),
            telegramClear: document.getElementById('telegram-clear'),
            telegramPause: document.getElementById('telegram-pause'),
            statDevices: document.getElementById('stat-devices'),
            statTps: document.getElementById('stat-tps'),
            statUptime: document.getElementById('stat-uptime'),
        };

        // Components
        this.roomGrid = new RoomGrid(this.els.roomGrid, this.onDeviceClick.bind(this));
        this.telegramInspector = new TelegramInspector(this.els.telegramFeed);
        this.devicePanel = new DevicePanel(this.els.devicePanel, this.onDeviceCommand.bind(this));

        // WebSocket
        this.ws = new WebSocketManager({
            onConnect: () => this.updateConnectionStatus('connected'),
            onDisconnect: () => this.updateConnectionStatus('disconnected'),
            onConnecting: () => this.updateConnectionStatus('connecting'),
            onTelegram: (data) => this.handleTelegram(data),
            onStateChange: (data) => this.handleStateChange(data),
        });

        // Bind events
        this.bindEvents();

        // Start
        this.init();
    }

    async init() {
        try {
            // Load premises
            this.premises = await API.getPremises();
            this.renderPremiseSelector();

            // Auto-select first premise
            if (this.premises.length > 0) {
                await this.selectPremise(this.premises[0].id);
            }

            // Start stats updater
            this.startStatsUpdater();

        } catch (err) {
            console.error('Init failed:', err);
            this.showError('Failed to connect to KNXSim API');
        }
    }

    bindEvents() {
        // Premise selector
        this.els.premiseSelector.addEventListener('change', (e) => {
            this.selectPremise(e.target.value);
        });

        // Engineer mode toggle
        this.els.engineerModeBtn.addEventListener('click', () => {
            this.toggleEngineerMode();
        });

        // Telegram controls
        this.els.telegramFilter.addEventListener('change', (e) => {
            this.telegramInspector.setFilter(e.target.value);
        });

        this.els.telegramClear.addEventListener('click', () => {
            this.telegramInspector.clear();
        });

        this.els.telegramPause.addEventListener('click', () => {
            const paused = this.telegramInspector.togglePause();
            this.els.telegramPause.textContent = paused ? 'Resume' : 'Pause';
        });

        // Close device panel
        document.getElementById('panel-close').addEventListener('click', () => {
            this.closeDevicePanel();
        });
    }

    // ─────────────────────────────────────────────────────────────
    // Premise Management
    // ─────────────────────────────────────────────────────────────

    renderPremiseSelector() {
        this.els.premiseSelector.innerHTML = this.premises
            .map(p => `<option value="${p.id}">${p.name}</option>`)
            .join('');
    }

    async selectPremise(premiseId) {
        if (!premiseId) return;

        this.currentPremise = this.premises.find(p => p.id === premiseId);

        // Load floors and devices
        const [floors, devices] = await Promise.all([
            API.getFloors(premiseId),
            API.getDevices(premiseId),
        ]);

        this.floors = floors;
        this.devices = devices;

        // Render floor tabs
        this.renderFloorTabs();

        // Select first floor or show all
        if (this.floors.length > 0) {
            this.selectFloor(this.floors[0].id);
        } else {
            this.currentFloor = null;
            this.renderDevices();
        }

        // Connect WebSocket
        this.ws.connect(premiseId);

        // Update stats
        this.els.statDevices.textContent = this.devices.length;
    }

    // ─────────────────────────────────────────────────────────────
    // Floor Management
    // ─────────────────────────────────────────────────────────────

    renderFloorTabs() {
        if (this.floors.length === 0) {
            this.els.floorTabs.innerHTML = '<button class="floor-tab active">All Devices</button>';
            return;
        }

        this.els.floorTabs.innerHTML = this.floors
            .map(f => `<button class="floor-tab" data-floor-id="${f.id}">${f.name}</button>`)
            .join('');

        // Bind click events
        this.els.floorTabs.querySelectorAll('.floor-tab').forEach(tab => {
            tab.addEventListener('click', () => {
                this.selectFloor(tab.dataset.floorId);
            });
        });
    }

    selectFloor(floorId) {
        this.currentFloor = this.floors.find(f => f.id === floorId);

        // Update active tab
        this.els.floorTabs.querySelectorAll('.floor-tab').forEach(tab => {
            tab.classList.toggle('active', tab.dataset.floorId === floorId);
        });

        this.renderDevices();
    }

    // ─────────────────────────────────────────────────────────────
    // Device Rendering
    // ─────────────────────────────────────────────────────────────

    renderDevices() {
        // Filter devices by current floor if applicable
        let devices = this.devices;

        if (this.currentFloor && this.currentFloor.rooms) {
            const roomIds = this.currentFloor.rooms.map(r => r.id);
            devices = this.devices.filter(d => roomIds.includes(d.room_id));
        }

        // Group by room
        const rooms = this.groupDevicesByRoom(devices);

        this.roomGrid.render(rooms, this.selectedDevice?.id);
    }

    groupDevicesByRoom(devices) {
        const roomMap = new Map();
        const unassigned = [];

        devices.forEach(device => {
            if (device.room_id) {
                if (!roomMap.has(device.room_id)) {
                    // Find room info
                    let roomInfo = null;
                    for (const floor of this.floors) {
                        if (floor.rooms) {
                            roomInfo = floor.rooms.find(r => r.id === device.room_id);
                            if (roomInfo) break;
                        }
                    }
                    roomMap.set(device.room_id, {
                        id: device.room_id,
                        name: roomInfo?.name || device.room_id,
                        devices: []
                    });
                }
                roomMap.get(device.room_id).devices.push(device);
            } else {
                unassigned.push(device);
            }
        });

        const rooms = Array.from(roomMap.values());

        // Add unassigned devices as a pseudo-room
        if (unassigned.length > 0) {
            rooms.push({
                id: '_unassigned',
                name: 'Unassigned',
                devices: unassigned
            });
        }

        return rooms;
    }

    // ─────────────────────────────────────────────────────────────
    // Device Interaction
    // ─────────────────────────────────────────────────────────────

    onDeviceClick(device) {
        this.selectedDevice = device;

        if (this.engineerMode) {
            this.devicePanel.show(device);
        }

        this.renderDevices();
    }

    async onDeviceCommand(deviceId, command, value) {
        try {
            await API.sendCommand(this.currentPremise.id, deviceId, command, value);
        } catch (err) {
            console.error('Command failed:', err);
        }
    }

    closeDevicePanel() {
        this.selectedDevice = null;
        this.devicePanel.hide();
        this.renderDevices();
    }

    // ─────────────────────────────────────────────────────────────
    // Engineer Mode
    // ─────────────────────────────────────────────────────────────

    toggleEngineerMode() {
        this.engineerMode = !this.engineerMode;
        this.els.engineerModeBtn.classList.toggle('active', this.engineerMode);

        if (!this.engineerMode) {
            this.closeDevicePanel();
        } else if (this.selectedDevice) {
            this.devicePanel.show(this.selectedDevice);
        }
    }

    // ─────────────────────────────────────────────────────────────
    // WebSocket Handlers
    // ─────────────────────────────────────────────────────────────

    handleTelegram(data) {
        this.telegramInspector.addTelegram(data);
        this.telegramCount++;
    }

    handleStateChange(data) {
        // Update device in local state
        const device = this.devices.find(d => d.id === data.device_id);
        if (device) {
            device.state = data.state;
            this.renderDevices();

            // Update panel if this device is selected
            if (this.selectedDevice?.id === data.device_id) {
                this.selectedDevice = device;
                this.devicePanel.updateState(device);
            }
        }
    }

    updateConnectionStatus(status) {
        const el = this.els.connectionStatus;
        el.className = `status ${status}`;
        el.querySelector('.status-text').textContent =
            status === 'connected' ? 'Connected' :
            status === 'connecting' ? 'Connecting...' :
            'Disconnected';
    }

    // ─────────────────────────────────────────────────────────────
    // Stats
    // ─────────────────────────────────────────────────────────────

    startStatsUpdater() {
        setInterval(() => {
            // Calculate TPS
            const now = Date.now();
            const elapsed = (now - this.lastTelegramTime) / 1000;
            if (elapsed >= 1) {
                this.tps = Math.round(this.telegramCount / elapsed);
                this.telegramCount = 0;
                this.lastTelegramTime = now;
                this.els.statTps.textContent = this.tps;
            }
        }, 1000);
    }

    // ─────────────────────────────────────────────────────────────
    // Error Handling
    // ─────────────────────────────────────────────────────────────

    showError(message) {
        // Simple error display - could be enhanced
        this.els.roomGrid.innerHTML = `
            <div class="loading-placeholder">
                <p style="color: var(--status-error);">${message}</p>
            </div>
        `;
    }
}

// Start app when DOM ready
document.addEventListener('DOMContentLoaded', () => {
    window.app = new KNXSimApp();
});
