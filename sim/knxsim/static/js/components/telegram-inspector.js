/**
 * Telegram Inspector Component
 *
 * Live feed of KNX bus traffic with filtering.
 * Persists history to localStorage across page refreshes.
 */

const MAX_TELEGRAMS = 200;
const STORAGE_KEY = "knxsim_telegram_history";

export class TelegramInspector {
  constructor(container) {
    this.container = container;
    this.telegrams = [];
    this.filter = "all";
    this.paused = false;
    this.placeholder = container.querySelector(".telegram-placeholder");

    // Load persisted telegrams from localStorage
    this._loadFromStorage();
  }

  /**
   * Load telegram history from localStorage.
   */
  _loadFromStorage() {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        this.telegrams = JSON.parse(stored);
        if (this.telegrams.length > 0) {
          this._renderAll();
        }
      }
    } catch (e) {
      console.warn("Failed to load telegram history:", e);
      this.telegrams = [];
    }
  }

  /**
   * Save telegram history to localStorage.
   */
  _saveToStorage() {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(this.telegrams));
    } catch (e) {
      console.warn("Failed to save telegram history:", e);
    }
  }

  /**
   * Add a telegram to the feed.
   */
  addTelegram(data) {
    if (this.paused) return;

    // Add to buffer
    this.telegrams.unshift(data);

    // Trim to max size
    if (this.telegrams.length > MAX_TELEGRAMS) {
      this.telegrams.pop();
    }

    // Persist to localStorage
    this._saveToStorage();

    // Render if matches filter
    if (this._matchesFilter(data)) {
      this._prependRow(data);
      this._trimRows();
    }

    // Remove placeholder
    if (this.placeholder) {
      this.placeholder.remove();
      this.placeholder = null;
    }
  }

  /**
   * Set filter (all, rx, tx).
   */
  setFilter(filter) {
    this.filter = filter;
    this._renderAll();
  }

  /**
   * Toggle pause state.
   */
  togglePause() {
    this.paused = !this.paused;
    return this.paused;
  }

  /**
   * Clear all telegrams (also clears localStorage).
   */
  clear() {
    this.telegrams = [];
    localStorage.removeItem(STORAGE_KEY);
    this.container.innerHTML = `
            <div class="telegram-placeholder">
                <p>Waiting for telegrams...</p>
            </div>
        `;
    this.placeholder = this.container.querySelector(".telegram-placeholder");
  }

  /**
   * Check if telegram matches current filter.
   */
  _matchesFilter(data) {
    if (this.filter === "all") return true;
    return data.direction === this.filter;
  }

  /**
   * Render a single telegram row.
   */
  _renderRow(data) {
    const time = this._formatTime(data.timestamp);
    const dir = data.direction || "rx";
    const dirLabel = data.direction_label || dir.toUpperCase();
    const dirDesc = data.direction_desc || "";
    const dirSymbol = dir === "rx" ? "←" : "→";
    const ga = data.destination || data.dst || "-";
    const gaFunc = data.ga_function || "";
    const device = data.device_id || "-";
    const rawPayload = data.payload || "-";
    const dpt = data.dpt || "-";
    const apci = data.apci || "GroupWrite";
    const decodedDisplay = this._formatDecodedDisplay(data);

    // Build tooltip for direction with APCI context
    const apciDesc = this._getApciDescription(apci, dir);
    const dirTitle = `title="${apciDesc}${dirDesc ? " — " + dirDesc : ""}"`;

    // Build GA display with function name if available
    const gaDisplay = gaFunc ? `${ga} (${gaFunc})` : ga;

    // Style class for special APCI types
    const apciClass = apci === "GroupRead" ? "apci-read" : "";

    return `
            <div class="telegram-row ${apciClass}">
                <span class="telegram-time">${time}</span>
                <span class="telegram-dir ${dir}" ${dirTitle}>${dirSymbol} ${dirLabel}</span>
                <span class="telegram-apci">${this._formatApci(apci)}</span>
                <span class="telegram-ga" title="${gaFunc}">${gaDisplay}</span>
                <span class="telegram-device">${device}</span>
                <span class="telegram-payload" title="Raw payload">${rawPayload}</span>
                <span class="telegram-decoded">${decodedDisplay}</span>
                <span class="telegram-dpt" title="Data Point Type">${dpt}</span>
            </div>
        `;
  }

  /**
   * Get human-readable APCI description.
   */
  _getApciDescription(apci, dir) {
    if (apci === "GroupRead") {
      return dir === "rx"
        ? "Request for current value"
        : "Requesting value from bus";
    }
    if (apci === "GroupResponse") {
      return "Response to a read request";
    }
    return dir === "rx" ? "Command received" : "Value update sent";
  }

  /**
   * Format APCI for compact display.
   */
  _formatApci(apci) {
    if (apci === "GroupWrite") return "W";
    if (apci === "GroupRead") return "R?";
    if (apci === "GroupResponse") return "R!";
    return apci.substring(0, 2);
  }

  /**
   * Format decoded value with unit for display.
   */
  _formatDecodedDisplay(data) {
    // Don't decode GroupRead - it's a request, not a value
    if (data.apci === "GroupRead") {
      return "(read request)";
    }

    if (data.decoded_value === undefined || data.decoded_value === null) {
      return "-";
    }

    const value = this._formatDecodedValue(data.decoded_value);
    const unit = data.unit || "";

    if (unit) {
      return `${value} ${unit}`;
    }
    return value;
  }

  /**
   * Prepend a new row to the feed.
   */
  _prependRow(data) {
    const html = this._renderRow(data);
    this.container.insertAdjacentHTML("afterbegin", html);
  }

  /**
   * Trim excess rows from the DOM.
   */
  _trimRows() {
    const rows = this.container.querySelectorAll(".telegram-row");
    if (rows.length > MAX_TELEGRAMS) {
      for (let i = MAX_TELEGRAMS; i < rows.length; i++) {
        rows[i].remove();
      }
    }
  }

  /**
   * Re-render all telegrams (after filter change).
   */
  _renderAll() {
    const filtered = this.telegrams.filter((t) => this._matchesFilter(t));

    if (filtered.length === 0) {
      this.container.innerHTML = `
                <div class="telegram-placeholder">
                    <p>No telegrams match filter</p>
                </div>
            `;
      return;
    }

    this.container.innerHTML = filtered
      .slice(0, MAX_TELEGRAMS)
      .map((t) => this._renderRow(t))
      .join("");
  }

  /**
   * Format timestamp for display.
   */
  _formatTime(timestamp) {
    if (!timestamp) return "--:--:--";

    let date;
    if (typeof timestamp === "number") {
      // Unix timestamp (seconds or milliseconds)
      date = new Date(timestamp > 1e12 ? timestamp : timestamp * 1000);
    } else {
      date = new Date(timestamp);
    }

    return date.toLocaleTimeString("en-GB", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  }

  /**
   * Format a decoded value for display.
   */
  _formatDecodedValue(value) {
    if (typeof value === "boolean") {
      return value ? "ON" : "OFF";
    }
    if (typeof value === "number") {
      // Format nicely
      if (Number.isInteger(value)) {
        return value.toString();
      }
      return value.toFixed(2);
    }
    if (typeof value === "object") {
      return JSON.stringify(value);
    }
    return String(value);
  }
}
