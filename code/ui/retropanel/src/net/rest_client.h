/**
 * REST client — HTTP GET/PUT/POST to Gray Logic Core.
 *
 * Uses libcurl for SDL simulator, esp_http_client for ESP32.
 * All functions are blocking — call from boot sequence or background thread.
 */
#ifndef REST_CLIENT_H
#define REST_CLIENT_H

#include "net/panel_config.h"
#include "data/data_model.h"
#include <stdbool.h>

/** Initialise the REST client with panel config. Call once at startup. */
void rest_client_init(const panel_config_t *cfg);

/** Clean up REST client resources. */
void rest_client_cleanup(void);

/**
 * Load the site hierarchy and extract rooms.
 * Populates rooms array, returns count (0 on error).
 */
int rest_load_rooms(room_t *rooms, int max_rooms);

/**
 * Load devices for a specific room.
 * Populates devices array, returns count (0 on error).
 */
int rest_load_devices(const char *room_id, device_t *devices, int max_devices);

/**
 * Load scenes for a specific room.
 * Populates scenes array and active_scene_id, returns count (0 on error).
 */
int rest_load_scenes(const char *room_id, scene_t *scenes, int max_scenes,
                     char *active_scene_id, int active_id_size);

/**
 * Send a device command.
 * command: "toggle", "set_level", "set_position", "set_setpoint", etc.
 * param_json: JSON parameters string, e.g. "{\"level\":75}"
 * Returns true on success (202 Accepted).
 */
bool rest_send_command(const char *device_id, const char *command,
                       const char *param_json);

/**
 * Activate a scene.
 * Returns true on success (202 Accepted).
 */
bool rest_activate_scene(const char *scene_id);

#endif
