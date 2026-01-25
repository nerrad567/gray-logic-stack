/**
 * KNXSim API Client
 *
 * REST API wrapper with fetch.
 */

const BASE_URL = '/api/v1';

async function request(path, options = {}) {
    const url = `${BASE_URL}${path}`;

    const response = await fetch(url, {
        headers: {
            'Content-Type': 'application/json',
            ...options.headers,
        },
        ...options,
    });

    if (!response.ok) {
        const error = await response.json().catch(() => ({}));
        throw new Error(error.detail || `HTTP ${response.status}`);
    }

    // Handle empty responses
    const text = await response.text();
    return text ? JSON.parse(text) : null;
}

export const API = {
    // ─────────────────────────────────────────────────────────────
    // Health
    // ─────────────────────────────────────────────────────────────

    async getHealth() {
        return request('/health');
    },

    // ─────────────────────────────────────────────────────────────
    // Premises
    // ─────────────────────────────────────────────────────────────

    async getPremises() {
        return request('/premises');
    },

    async getPremise(premiseId) {
        return request(`/premises/${premiseId}`);
    },

    async createPremise(data) {
        return request('/premises', {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    async deletePremise(premiseId) {
        return request(`/premises/${premiseId}`, {
            method: 'DELETE',
        });
    },

    // ─────────────────────────────────────────────────────────────
    // Devices
    // ─────────────────────────────────────────────────────────────

    async getDevices(premiseId) {
        return request(`/premises/${premiseId}/devices`);
    },

    async getDevice(premiseId, deviceId) {
        return request(`/premises/${premiseId}/devices/${deviceId}`);
    },

    async createDevice(premiseId, data) {
        return request(`/premises/${premiseId}/devices`, {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    async updateDevice(premiseId, deviceId, data) {
        return request(`/premises/${premiseId}/devices/${deviceId}`, {
            method: 'PATCH',
            body: JSON.stringify(data),
        });
    },

    async deleteDevice(premiseId, deviceId) {
        return request(`/premises/${premiseId}/devices/${deviceId}`, {
            method: 'DELETE',
        });
    },

    async setDevicePlacement(premiseId, deviceId, roomId) {
        return request(`/premises/${premiseId}/devices/${deviceId}/placement`, {
            method: 'PATCH',
            body: JSON.stringify({ room_id: roomId }),
        });
    },

    /**
     * Send a command to a device.
     * This triggers a GroupWrite on the appropriate GA.
     */
    async sendCommand(premiseId, deviceId, command, value) {
        return request(`/premises/${premiseId}/devices/${deviceId}/command`, {
            method: 'POST',
            body: JSON.stringify({ command, value }),
        });
    },

    // ─────────────────────────────────────────────────────────────
    // Floors & Rooms
    // ─────────────────────────────────────────────────────────────

    async getFloors(premiseId) {
        return request(`/premises/${premiseId}/floors`);
    },

    async createFloor(premiseId, data) {
        return request(`/premises/${premiseId}/floors`, {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    async updateFloor(premiseId, floorId, data) {
        return request(`/premises/${premiseId}/floors/${floorId}`, {
            method: 'PATCH',
            body: JSON.stringify(data),
        });
    },

    async deleteFloor(premiseId, floorId) {
        return request(`/premises/${premiseId}/floors/${floorId}`, {
            method: 'DELETE',
        });
    },

    async getRooms(premiseId, floorId) {
        return request(`/premises/${premiseId}/floors/${floorId}/rooms`);
    },

    async createRoom(premiseId, floorId, data) {
        return request(`/premises/${premiseId}/floors/${floorId}/rooms`, {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    async updateRoom(premiseId, floorId, roomId, data) {
        return request(`/premises/${premiseId}/floors/${floorId}/rooms/${roomId}`, {
            method: 'PATCH',
            body: JSON.stringify(data),
        });
    },

    async deleteRoom(premiseId, floorId, roomId) {
        return request(`/premises/${premiseId}/floors/${floorId}/rooms/${roomId}`, {
            method: 'DELETE',
        });
    },

    // ─────────────────────────────────────────────────────────────
    // Templates
    // ─────────────────────────────────────────────────────────────

    async getTemplates(domain = null) {
        const params = domain ? `?domain=${domain}` : '';
        return request(`/templates${params}`);
    },

    async getTemplate(templateId) {
        return request(`/templates/${templateId}`);
    },

    async getDomains() {
        return request('/templates/domains');
    },

    async createDeviceFromTemplate(premiseId, templateId, data) {
        return request(`/premises/${premiseId}/devices/from-template`, {
            method: 'POST',
            body: JSON.stringify({ template_id: templateId, ...data }),
        });
    },

    // ─────────────────────────────────────────────────────────────
    // Telegrams
    // ─────────────────────────────────────────────────────────────

    async getTelegrams(premiseId, limit = 100, offset = 0) {
        return request(`/premises/${premiseId}/telegrams?limit=${limit}&offset=${offset}`);
    },

    async getTelegramStats(premiseId) {
        return request(`/premises/${premiseId}/telegrams/stats`);
    },

    async clearTelegrams(premiseId) {
        return request(`/premises/${premiseId}/telegrams`, {
            method: 'DELETE',
        });
    },
};
