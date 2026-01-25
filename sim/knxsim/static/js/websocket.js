/**
 * KNXSim WebSocket Manager
 *
 * Handles connection, reconnection, and message routing.
 */

export class WebSocketManager {
  constructor(options = {}) {
    this.options = options;
    this.ws = null;
    this.premiseId = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 10;
    this.reconnectDelay = 1000;
    this.reconnectTimer = null;
    this.intentionalClose = false;
  }

  /**
   * Connect to WebSocket for a specific premise.
   */
  connect(premiseId) {
    // Close existing connection if different premise
    if (this.ws && this.premiseId !== premiseId) {
      this.disconnect();
    }

    this.premiseId = premiseId;
    this.intentionalClose = false;
    this._connect();
  }

  /**
   * Disconnect and stop reconnection attempts.
   */
  disconnect() {
    this.intentionalClose = true;
    this._clearReconnectTimer();

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  /**
   * Internal connect method.
   */
  _connect() {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    // Build WebSocket URL
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = window.location.host;
    const url = `${protocol}//${host}/ws/telegrams?premise=${this.premiseId}`;

    this.options.onConnecting?.();

    try {
      this.ws = new WebSocket(url);
      this._bindEvents();
    } catch (err) {
      console.error("WebSocket creation failed:", err);
      this._scheduleReconnect();
    }
  }

  /**
   * Bind WebSocket event handlers.
   */
  _bindEvents() {
    this.ws.onopen = () => {
      console.log("WebSocket connected");
      this.reconnectAttempts = 0;
      this.reconnectDelay = 1000;
      this.options.onConnect?.();
    };

    this.ws.onclose = (event) => {
      console.log("WebSocket closed:", event.code, event.reason);
      this.options.onDisconnect?.();

      if (!this.intentionalClose) {
        this._scheduleReconnect();
      }
    };

    this.ws.onerror = (error) => {
      console.error("WebSocket error:", error);
    };

    this.ws.onmessage = (event) => {
      this._handleMessage(event.data);
    };
  }

  /**
   * Handle incoming WebSocket message.
   */
  _handleMessage(data) {
    try {
      const message = JSON.parse(data);

      // Route message based on type
      if (message.type === "telegram") {
        this.options.onTelegram?.(message);
      } else if (message.type === "state_change") {
        this.options.onStateChange?.(message);
      }
    } catch (err) {
      console.error("Failed to parse WebSocket message:", err, data);
    }
  }

  /**
   * Schedule a reconnection attempt with exponential backoff.
   */
  _scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error("Max reconnection attempts reached");
      return;
    }

    this._clearReconnectTimer();

    this.reconnectAttempts++;
    const delay = Math.min(
      this.reconnectDelay * Math.pow(1.5, this.reconnectAttempts - 1),
      30000,
    );

    console.log(
      `Reconnecting in ${Math.round(delay / 1000)}s (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`,
    );

    this.reconnectTimer = setTimeout(() => {
      this._connect();
    }, delay);
  }

  /**
   * Clear any pending reconnection timer.
   */
  _clearReconnectTimer() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  /**
   * Send a message over the WebSocket.
   */
  send(data) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    } else {
      console.warn("WebSocket not connected, cannot send:", data);
    }
  }

  /**
   * Get current connection state.
   */
  get isConnected() {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}
