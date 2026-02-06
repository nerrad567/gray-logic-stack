/**
 * Command sending — all commands go via REST to Core.
 *
 * Fire-and-forget with logging. Confirmation comes via MQTT state updates.
 */
#include "net/command.h"
#include "net/rest_client.h"
#include "data/data_store.h"
#include <stdio.h>

bool cmd_toggle(const char *device_id, bool current_state)
{
    if (!data_store_is_live()) return false;
    printf("[cmd] toggle %s (currently %s)\n", device_id, current_state ? "on" : "off");
    return rest_send_command(device_id, "toggle", "{}");
}

bool cmd_set_level(const char *device_id, int level)
{
    if (!data_store_is_live()) return false;
    char params[64];
    snprintf(params, sizeof(params), "{\"level\":%d}", level);
    printf("[cmd] set_level %s → %d\n", device_id, level);
    return rest_send_command(device_id, "set_level", params);
}

bool cmd_set_position(const char *device_id, int position)
{
    if (!data_store_is_live()) return false;
    char params[64];
    snprintf(params, sizeof(params), "{\"position\":%d}", position);
    printf("[cmd] set_position %s → %d\n", device_id, position);
    return rest_send_command(device_id, "set_position", params);
}

bool cmd_set_setpoint(const char *device_id, float setpoint)
{
    if (!data_store_is_live()) return false;
    char params[64];
    snprintf(params, sizeof(params), "{\"setpoint\":%.1f}", setpoint);
    printf("[cmd] set_setpoint %s → %.1f\n", device_id, setpoint);
    return rest_send_command(device_id, "set_setpoint", params);
}

bool cmd_activate_scene(const char *scene_id)
{
    if (!data_store_is_live()) return false;
    printf("[cmd] activate_scene %s\n", scene_id);
    return rest_activate_scene(scene_id);
}
