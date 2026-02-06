/**
 * Command abstraction — send device commands and scene activations.
 *
 * All commands go via REST (not MQTT publish). Core handles protocol routing.
 * Commands are fire-and-forget with optimistic UI — confirmation comes via MQTT.
 */
#ifndef COMMAND_H
#define COMMAND_H

#include <stdbool.h>

/** Toggle a device on/off. */
bool cmd_toggle(const char *device_id, bool current_state);

/** Set dimmer level (0-100). */
bool cmd_set_level(const char *device_id, int level);

/** Set blind position (0-100). */
bool cmd_set_position(const char *device_id, int position);

/** Set thermostat setpoint. */
bool cmd_set_setpoint(const char *device_id, float setpoint);

/** Activate a scene. */
bool cmd_activate_scene(const char *scene_id);

#endif
