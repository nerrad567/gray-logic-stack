---
title: UI/UX Patterns
version: 1.0.0
status: active
last_updated: 2026-01-17
depends_on:
  - interfaces/api.md
---

# UI/UX Patterns

This document defines the standard interaction patterns for all Gray Logic user interfaces (Wall Panels, Mobile App, Web Admin).

To ensure a "rock solid" feel, the UI must handle network latency and state synchronization gracefully, avoiding the common "flicker" issues seen in other home automation systems.

---

## 1. Optimistic State Updates with Rollback

**The Problem:**
In a typical system, when a user toggles a switch:
1. UI sends command.
2. UI waits.
3. Switch eventually confirms.
4. UI updates state.
*Result:* A sluggish, unresponsive feel (200ms-500ms lag).

**The Solution:**
The UI must immediately assume success, display the new state, and only revert if the system reports failure or times out.

### State Machine

Each controllable UI element (switch, slider, button) maintains a local state machine:

| State | Visual Indicator | Interaction |
|-------|------------------|-------------|
| `Synced` | Normal (Solid Color) | Enabled |
| `Pending` | Optimistic (Slightly Dimmed or Spinner) | Disabled (debounce) |
| `Error` | Error Icon / Red Border | Enabled (Retry) |

### Interaction Flow

1.  **User Action:** User taps "Living Room Light" (currently OFF).
2.  **Immediate Update:** 
    - UI state changes to `Pending` (ON).
    - Visual: Switch slides to ON immediately. Opacity set to 50% or small loading spinner appears.
3.  **Command Sent:** API request `PUT /devices/{id}/state` is fired in background.
4.  **Scenario A: Success (Typical)**
    - WebSocket event `device.state_changed` arrives with `new_state: ON`.
    - UI state changes to `Synced` (ON).
    - Visual: Opacity returns to 100%, spinner disappears.
5.  **Scenario B: Failure (Error/Timeout)**
    - API returns non-200 error OR 3-second timeout occurs.
    - UI triggers **Rollback**.
    - UI state changes back to `Synced` (OFF).
    - Visual: Switch slides back to OFF.
    - **Feedback:** Toast notification appears: "Failed to turn on light: Device unreachable".

### Implementation Guidelines (Flutter/Svelte)

```dart
// Pseudo-code for Flutter Widget
void onToggle(bool newValue) {
  // 1. Optimistic Update
  setState(() {
    _currentValue = newValue;
    _status = DeviceStatus.pending;
  });

  // 2. API Call
  api.setDeviceState(deviceId, newValue).then((response) {
    // 3A. Success - Wait for WebSocket to confirm final state
    // (Actual state sync happens via StreamBuilder/Provider)
  }).catchError((error) {
    // 3B. Failure - Rollback
    setState(() {
      _currentValue = !newValue; // Revert
      _status = DeviceStatus.error;
    });
    showToast("Command failed: ${error.message}");
  });
}
```

---

## 2. Connection State & Global Health

The UI must transparently show the health of the connection to the Core.

### Connectivity Indicators

| Status | Indicator | Description |
|--------|-----------|-------------|
| **Connected** | None (Hidden) | Normal operation. |
| **Connecting** | Amber Dot / "Connecting..." | Establishing WebSocket connection. |
| **Offline** | Red Bar / "Offline" | No network connection to Core. |
| **Degraded** | Orange Warning Icon | Connected, but Core reports bridge failures. |

### Degraded Mode Handling

If the Core reports a bridge is down (via `system.status` or `bridge.status` event):
- **Do not disable** the entire UI.
- **Visual Cue:** Grey out only the devices belonging to the failed bridge.
- **Interaction:** Tapping a greyed-out device shows a specific error: "KNX Bridge is offline. Check physical connection."

---

## 3. Feedback & Notifications

### Toast Notifications (Snackbars)

Used for transient feedback.

- **Success:** Generally suppressed to reduce noise (the switch moving is feedback enough). Used only for invisible actions (e.g., "Scene Saved").
- **Error:** Always shown. Must be actionable or descriptive.
  - *Bad:* "Error 500"
  - *Good:* "Could not save scene: Name already exists."

### Blocking Dialogs

Use sparingly. Only for critical confirmations (e.g., "Delete Data", "Factory Reset") or creating complex configuration.

---

## 4. Voice Feedback

When triggering actions via Voice:
- **Visual:** If a screen is visible, show a "Listening" overlay.
- **Audio:**
  - **Wake Word:** Subtle chime (optional).
  - **Success:** No verbal response for simple actions ("Turn on lights" -> Lights turn on). Verbal response for queries ("What is the temperature?" -> "It is 21 degrees").
  - **Failure:** Verbal apology ("I couldn't reach the kitchen light").

---

## 5. Dashboard Organization

### Hierarchy
1.  **Home (Dashboard):** Favorites, Scenes, Global Status.
2.  **Area/Room List:** Navigation to specific spaces.
3.  **Room View:** All devices in a room, grouped by domain (Lights, Climate, Shades).
4.  **Device Detail:** Deep control (e.g., Color Picker, Schedule editing).

### Complexity Hiding
- **Default:** Show simple controls (On/Off, Dimmer Slider).
- **Expanded:** Tap icon or long-press to reveal advanced controls (Color Temp, RGB, Configuration).
