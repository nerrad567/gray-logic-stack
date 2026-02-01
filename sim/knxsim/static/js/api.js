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

    // ─────────────────────────────────────────────────────────────
    // Reference Data
    // ─────────────────────────────────────────────────────────────

    async getReference() {
        return request('/reference');
    },

    async getGaStructure() {
        return request('/reference/ga-structure');
    },

    async getFlags() {
        return request('/reference/flags');
    },

    async getDpts() {
        return request('/reference/dpts');
    },

    async getDeviceTemplates() {
        return request('/reference/device-templates');
    },

    async getDeviceTemplate(deviceType) {
        return request(`/reference/device-templates/${deviceType}`);
    },

    // ─────────────────────────────────────────────────────────────
    // Topology (Areas & Lines)
    // ─────────────────────────────────────────────────────────────

    async getTopology(premiseId) {
        return request(`/premises/${premiseId}/topology`);
    },

    async getAreas(premiseId) {
        return request(`/premises/${premiseId}/areas`);
    },

    async createArea(premiseId, data) {
        return request(`/premises/${premiseId}/areas`, {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    async updateArea(premiseId, areaId, data) {
        return request(`/premises/${premiseId}/areas/${areaId}`, {
            method: 'PATCH',
            body: JSON.stringify(data),
        });
    },

    async deleteArea(premiseId, areaId) {
        return request(`/premises/${premiseId}/areas/${areaId}`, {
            method: 'DELETE',
        });
    },

    async getLines(premiseId, areaId) {
        return request(`/premises/${premiseId}/areas/${areaId}/lines`);
    },

    async createLine(premiseId, areaId, data) {
        return request(`/premises/${premiseId}/areas/${areaId}/lines`, {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    async updateLine(premiseId, areaId, lineId, data) {
        return request(`/premises/${premiseId}/areas/${areaId}/lines/${lineId}`, {
            method: 'PATCH',
            body: JSON.stringify(data),
        });
    },

    async deleteLine(premiseId, areaId, lineId) {
        return request(`/premises/${premiseId}/areas/${areaId}/lines/${lineId}`, {
            method: 'DELETE',
        });
    },

    // ─────────────────────────────────────────────────────────────
    // Group Address Hierarchy
    // ─────────────────────────────────────────────────────────────

    async getGroupTree(premiseId) {
        return request(`/premises/${premiseId}/groups`);
    },

    async getMainGroups(premiseId) {
        return request(`/premises/${premiseId}/main-groups`);
    },

    async createMainGroup(premiseId, data) {
        return request(`/premises/${premiseId}/main-groups`, {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    async updateMainGroup(mainGroupId, data) {
        return request(`/main-groups/${mainGroupId}`, {
            method: 'PATCH',
            body: JSON.stringify(data),
        });
    },

    async deleteMainGroup(mainGroupId) {
        return request(`/main-groups/${mainGroupId}`, {
            method: 'DELETE',
        });
    },

    async getMiddleGroups(mainGroupId) {
        return request(`/main-groups/${mainGroupId}/middle-groups`);
    },

    async createMiddleGroup(mainGroupId, data) {
        return request(`/main-groups/${mainGroupId}/middle-groups`, {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    async updateMiddleGroup(middleGroupId, data) {
        return request(`/middle-groups/${middleGroupId}`, {
            method: 'PATCH',
            body: JSON.stringify(data),
        });
    },

    async deleteMiddleGroup(middleGroupId) {
        return request(`/middle-groups/${middleGroupId}`, {
            method: 'DELETE',
        });
    },

    async createDefaultGroups(premiseId) {
        return request(`/premises/${premiseId}/groups/create-defaults`, {
            method: 'POST',
        });
    },

    async suggestGroupAddresses(premiseId, deviceType, roomId = null, mainGroup = null) {
        const params = new URLSearchParams({ device_type: deviceType });
        if (roomId) params.append('room_id', roomId);
        if (mainGroup !== null) params.append('main_group', mainGroup);
        return request(`/premises/${premiseId}/groups/suggest?${params}`);
    },

    async getNextSubAddress(premiseId, main, middle) {
        return request(`/premises/${premiseId}/groups/next-sub?main=${main}&middle=${middle}`);
    },
};
