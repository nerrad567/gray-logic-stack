/**
 * Room Grid Component
 *
 * Renders rooms with device tiles showing live status.
 */

// Device type to icon mapping
const DEVICE_ICONS = {
    light_switch: '💡',
    light_dimmer: '💡',
    light_rgb: '🎨',
    light_colour_temp: '💡',
    blind: '🪟',
    blind_position: '🪟',
    blind_position_slat: '🪟',
    sensor: '🌡️',
    temperature_sensor: '🌡️',
    humidity_sensor: '💧',
    co2_sensor: '🌬️',
    lux_sensor: '☀️',
    presence: '👤',
    presence_detector: '👤',
    thermostat: '🌡️',
    hvac_unit: '❄️',
    energy_meter: '⚡',
    solar_inverter: '☀️',
    push_button_2: '🔘',
    push_button_4: '🔘',
    scene_controller: '🎬',
    binary_input: '⏺️',
};

export class RoomGrid {
    constructor(container, onDeviceClick) {
        this.container = container;
        this.onDeviceClick = onDeviceClick;
    }

    /**
     * Render rooms with their devices.
     * @param {Array} rooms - Array of {id, name, devices: []}
     * @param {string} selectedDeviceId - Currently selected device ID
     */
    render(rooms, selectedDeviceId = null) {
        if (rooms.length === 0) {
            this.container.innerHTML = `
                <div class="loading-placeholder">
                    <p>No devices found</p>
                </div>
            `;
            return;
        }

        this.container.innerHTML = rooms.map(room => this._renderRoom(room, selectedDeviceId)).join('');

        // Bind click events
        this.container.querySelectorAll('.device-tile').forEach(tile => {
            tile.addEventListener('click', () => {
                const deviceId = tile.dataset.deviceId;
                const device = this._findDevice(rooms, deviceId);
                if (device) {
                    this.onDeviceClick(device);
                }
            });
        });
    }

    _renderRoom(room, selectedDeviceId) {
        const statusSummary = this._getRoomStatusSummary(room.devices);

        return `
            <div class="room-card" data-room-id="${room.id}">
                <div class="room-header">
                    <span class="room-name">${this._escapeHtml(room.name)}</span>
                    <span class="room-status">${statusSummary}</span>
                </div>
                <div class="room-devices">
                    ${room.devices.map(d => this._renderDevice(d, selectedDeviceId)).join('')}
                </div>
            </div>
        `;
    }

    _renderDevice(device, selectedDeviceId) {
        const icon = DEVICE_ICONS[device.type] || '📦';
        const stateDisplay = this._getStateDisplay(device);
        const isSelected = device.id === selectedDeviceId;

        return `
            <div class="device-tile ${isSelected ? 'selected' : ''}" data-device-id="${device.id}">
                <div class="device-info">
                    <span class="device-icon">${icon}</span>
                    <span class="device-name">${this._formatDeviceName(device.id)}</span>
                </div>
                <div class="device-state">
                    ${stateDisplay}
                </div>
            </div>
        `;
    }

    /**
     * Get display HTML for device state based on type.
     */
    _getStateDisplay(device) {
        const state = device.state || {};
        const type = device.type;

        // Light devices
        if (type.startsWith('light_')) {
            const isOn = state.on === true;
            const brightness = state.brightness;

            let html = `<span class="state-indicator ${isOn ? 'on' : ''}"></span>`;

            if (brightness !== undefined && isOn) {
                html += `<span class="state-value">${brightness}%</span>`;
            } else {
                html += `<span class="state-value">${isOn ? 'ON' : 'OFF'}</span>`;
            }

            return html;
        }

        // Blind devices
        if (type.startsWith('blind')) {
            const position = state.position ?? 0;
            return `<span class="state-value">${position}%</span>`;
        }

        // Presence
        if (type === 'presence' || type === 'presence_detector') {
            const present = state.presence === true;
            return `
                <span class="state-indicator ${present ? 'on' : ''}"></span>
                <span class="state-value">${present ? 'OCCUPIED' : 'EMPTY'}</span>
            `;
        }

        // Temperature sensors
        if (type === 'sensor' || type === 'temperature_sensor') {
            const temp = state.temperature;
            if (temp !== undefined) {
                return `<span class="state-value">${temp.toFixed(1)}°C</span>`;
            }
        }

        // Humidity
        if (type === 'humidity_sensor') {
            const humidity = state.humidity;
            if (humidity !== undefined) {
                return `<span class="state-value">${humidity.toFixed(0)}%</span>`;
            }
        }

        // Thermostat
        if (type === 'thermostat') {
            const current = state.current_temperature;
            const setpoint = state.setpoint;
            if (current !== undefined) {
                let html = `<span class="state-value">${current.toFixed(1)}°C</span>`;
                if (setpoint !== undefined) {
                    html += ` <span class="state-value" style="color: var(--accent-blue);">→${setpoint}°C</span>`;
                }
                return html;
            }
        }

        // Energy meter
        if (type === 'energy_meter' || type === 'solar_inverter') {
            const power = state.power;
            if (power !== undefined) {
                return `<span class="state-value">${power.toFixed(0)}W</span>`;
            }
        }

        // Default: show first meaningful state value
        for (const [key, value] of Object.entries(state)) {
            if (typeof value === 'boolean') {
                return `
                    <span class="state-indicator ${value ? 'on' : ''}"></span>
                    <span class="state-value">${value ? 'ON' : 'OFF'}</span>
                `;
            }
            if (typeof value === 'number') {
                return `<span class="state-value">${value}</span>`;
            }
        }

        return '<span class="state-value">-</span>';
    }

    /**
     * Get summary icons for room status.
     */
    _getRoomStatusSummary(devices) {
        let lightsOn = 0;
        let hasPresence = false;
        let temp = null;

        devices.forEach(d => {
            const state = d.state || {};

            if (d.type.startsWith('light_') && state.on) {
                lightsOn++;
            }
            if ((d.type === 'presence' || d.type === 'presence_detector') && state.presence) {
                hasPresence = true;
            }
            if ((d.type === 'sensor' || d.type === 'temperature_sensor') && state.temperature !== undefined) {
                temp = state.temperature;
            }
        });

        const parts = [];
        if (lightsOn > 0) parts.push(`💡${lightsOn}`);
        if (hasPresence) parts.push('👤');
        if (temp !== null) parts.push(`${temp.toFixed(1)}°`);

        return parts.join(' ');
    }

    /**
     * Format device ID into readable name.
     */
    _formatDeviceName(id) {
        return id
            .replace(/-/g, ' ')
            .replace(/_/g, ' ')
            .split(' ')
            .map(word => word.charAt(0).toUpperCase() + word.slice(1))
            .join(' ');
    }

    /**
     * Find device by ID across all rooms.
     */
    _findDevice(rooms, deviceId) {
        for (const room of rooms) {
            const device = room.devices.find(d => d.id === deviceId);
            if (device) return device;
        }
        return null;
    }

    /**
     * Escape HTML to prevent XSS.
     */
    _escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}
