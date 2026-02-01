/**
 * KNXSim Dashboard — Alpine.js Application
 *
 * Initialises Alpine stores and WebSocket connection.
 */

// Re-export initStores for use in inline script
export { initStores } from "./store.js";

// ─────────────────────────────────────────────────────────────────────────────
// WebSocket Manager
// ─────────────────────────────────────────────────────────────────────────────

class WebSocketManager {
  constructor() {
    this.ws = null;
    this.premiseId = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 10;
    this.reconnectDelay = 1000;
    this.reconnectTimer = null;
  }

  connect(premiseId) {
    if (this.ws) {
      this.disconnect();
    }

    this.premiseId = premiseId;
    this.reconnectAttempts = 0;
    this._connect();
  }

  _connect() {
    if (!this.premiseId) return;

    Alpine.store("app").setConnecting();

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/ws/state?premise=${this.premiseId}`;

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log("WebSocket connected");
        Alpine.store("app").setConnected(true);
        this.reconnectAttempts = 0;
        this.reconnectDelay = 1000;

        // Also connect to telegram stream
        this._connectTelegramStream();
      };

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          this._handleMessage(data);
        } catch (err) {
          console.error("Failed to parse WebSocket message:", err);
        }
      };

      this.ws.onclose = () => {
        console.log("WebSocket closed");
        Alpine.store("app").setConnected(false);
        this._scheduleReconnect();
      };

      this.ws.onerror = (err) => {
        console.error("WebSocket error:", err);
      };
    } catch (err) {
      console.error("Failed to create WebSocket:", err);
      this._scheduleReconnect();
    }
  }

  _connectTelegramStream() {
    if (this.telegramWs) {
      this.telegramWs.close();
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/ws/telegrams?premise=${this.premiseId}`;

    this.telegramWs = new WebSocket(wsUrl);

    this.telegramWs.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === "telegram") {
          Alpine.store("telegrams").add(data.payload);
        }
      } catch (err) {
        console.error("Failed to parse telegram message:", err);
      }
    };
  }

  _handleMessage(data) {
    if (data.type === "state_change") {
      Alpine.store("app").updateDeviceState(data.device_id, data.state);
    }
  }

  _scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log("Max reconnection attempts reached");
      return;
    }

    this.reconnectTimer = setTimeout(() => {
      this.reconnectAttempts++;
      console.log(`Reconnecting... (attempt ${this.reconnectAttempts})`);
      this._connect();
    }, this.reconnectDelay);

    // Exponential backoff
    this.reconnectDelay = Math.min(this.reconnectDelay * 1.5, 30000);
  }

  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    if (this.telegramWs) {
      this.telegramWs.close();
      this.telegramWs = null;
    }
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// TPS Calculator
// ─────────────────────────────────────────────────────────────────────────────

class TPSCalculator {
  constructor() {
    this.lastCount = 0;
    this.lastTime = Date.now();
  }

  start() {
    setInterval(() => {
      const app = Alpine.store("app");
      const now = Date.now();
      const elapsed = (now - this.lastTime) / 1000;

      if (elapsed >= 1) {
        const count = app.telegramCount - this.lastCount;
        app.tps = Math.round(count / elapsed);
        this.lastCount = app.telegramCount;
        this.lastTime = now;
      }
    }, 1000);
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Initialisation
// ─────────────────────────────────────────────────────────────────────────────

// Global instances
let wsManager = null;
let tpsCalculator = null;

/**
 * Start the application after Alpine is initialized.
 * Called from inline script in index.html after alpine:initialized.
 */
export async function startApp() {
  // Create WebSocket manager
  wsManager = new WebSocketManager();

  // Start TPS calculator
  tpsCalculator = new TPSCalculator();
  tpsCalculator.start();

  // Watch for premise changes to reconnect WebSocket
  Alpine.effect(() => {
    const premiseId = Alpine.store("app").currentPremiseId;
    if (premiseId) {
      wsManager.connect(premiseId);
    }
  });

  // Global keyboard shortcuts
  document.addEventListener("keydown", (e) => {
    // Escape closes modals
    if (e.key === "Escape") {
      const modal = Alpine.store("modal");
      if (modal.open) {
        modal.close();
      }
    }
  });

  // Load initial data
  const app = Alpine.store("app");
  await Promise.all([
    app.loadPremises(),
    app.loadTemplates(),
    Alpine.store("reference").load(),
  ]);
}

// Export for potential external use
window.KNXSim = {
  get wsManager() {
    return wsManager;
  },
  get app() {
    return Alpine.store("app");
  },
  get modal() {
    return Alpine.store("modal");
  },
  get telegrams() {
    return Alpine.store("telegrams");
  },
};
