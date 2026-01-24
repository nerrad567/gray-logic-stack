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

---

## 6. Role-Based Interface Modes

The Flutter app (wall panel, mobile, web) is a **single codebase** that adapts its UI based on the authenticated user's role. This eliminates the need for separate installer/dealer software.

### Interface Modes

| Mode | Entry Method | Roles | Purpose |
|------|-------------|-------|---------|
| **Normal** | Default on launch | `user`, `guest` | Day-to-day control: activate scenes, control devices |
| **Admin** | Settings → Admin Login (PIN/password) | `admin`, `facility_manager`, `installer` | Configuration: create scenes, manage devices, assign rooms |

### Normal Mode (Default)

Available to all authenticated users:
- Room navigation with device tiles
- Scene activation (tap to trigger)
- Basic device control (on/off, dim, position)
- View sensor readings
- Personal preferences (favourites, display settings)

### Admin Mode (Authenticated)

Available to `admin`, `facility_manager`, and `installer` roles. Accessed via long-press on settings icon or dedicated menu entry. Requires elevated authentication (PIN re-entry or password).

| Feature | admin | facility_manager | installer |
|---------|-------|-------------------|-----------|
| **Scene Management** (create/edit/delete) | Yes | Yes | Yes |
| **Device Configuration** (rename, assign room) | Yes | Yes | Yes |
| **Room/Area Setup** (create, reorder, assign) | Yes | Yes | Yes |
| **Schedule Editor** (time-based automations) | Yes | Yes | No |
| **User Management** (create accounts, assign roles) | Yes | No | No |
| **System Diagnostics** (bridge status, logs) | Yes | Yes | Yes |
| **Mode Configuration** (Away, Night, Holiday) | Yes | Yes | No |
| **Network/Integration Settings** | Yes | No | Yes |

### Admin Mode UI Patterns

- **Visual Indicator:** Persistent banner or accent colour change indicating admin mode is active (prevents accidental configuration changes).
- **Auto-Timeout:** Admin mode automatically exits after 15 minutes of inactivity, reverting to normal mode.
- **Confirmation Dialogs:** Destructive actions (delete scene, remove device) require explicit confirmation.
- **Audit Trail:** All admin actions are logged with timestamp, user, and change details.

### Scene Management (Admin Mode)

When in admin mode, the scenes view extends from activation-only to full CRUD:

| Action | UI Element | API |
|--------|-----------|-----|
| **Create Scene** | FAB (+) button → scene builder form | `POST /api/v1/scenes` |
| **Edit Scene** | Long-press scene tile → edit form | `PUT /api/v1/scenes/{id}` |
| **Delete Scene** | Swipe-to-delete or edit → delete button | `DELETE /api/v1/scenes/{id}` |
| **Reorder Actions** | Drag-and-drop action list | `PUT /api/v1/scenes/{id}` |
| **Test Scene** | "Test" button in editor (activates without saving) | `POST /api/v1/scenes/{id}/activate` |

**Scene Builder Form Fields:**
- Name, category, icon
- Room/area assignment
- Actions list: device picker → command → parameters (level, position, etc.)
- Per-action: parallel flag, delay, fade duration
- Triggers: manual, schedule, voice phrase

### Panel-Specific Considerations

Wall panels may have admin mode restricted based on deployment context:

```yaml
# Panel configuration (per-device)
panel:
  id: "panel-living-room"
  admin_mode: enabled        # enabled | disabled | pin_required
  admin_timeout_minutes: 15
  allowed_roles: ["admin", "installer"]  # Subset of admin-capable roles
```

- **Public areas** (hotel lobby, office reception): `admin_mode: disabled`
- **Private residence**: `admin_mode: pin_required`
- **During commissioning**: `admin_mode: enabled` (installer convenience)
