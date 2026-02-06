/**
 * Data store — holds the current room data.
 *
 * Since LVGL is single-threaded and we drain MQTT updates on the LVGL thread,
 * the store doesn't need a mutex for SDL. The MQTT drain function is called
 * from the main loop, ensuring all access is single-threaded.
 */
#include "data/data_store.h"
#include <string.h>
#include <stdio.h>

static room_data_t store_data;
static bool store_live = false;

void data_store_init(const room_data_t *data)
{
    if (data) {
        memcpy(&store_data, data, sizeof(store_data));
    }
}

const room_data_t *data_store_get_room_data(void)
{
    return &store_data;
}

void data_store_apply_update(const mqtt_state_update_t *update)
{
    if (!update) return;

    for (int i = 0; i < store_data.device_count; i++) {
        device_t *dev = &store_data.devices[i];
        if (strcmp(dev->id, update->device_id) != 0) continue;

        if (update->has_on)    dev->on = update->on;
        if (update->has_level) dev->level = (uint8_t)update->level;
        if (update->has_pos)   dev->position = (uint8_t)update->position;
        if (update->has_temp)  dev->temperature = update->temperature;
        if (update->has_sp)    dev->setpoint = update->setpoint;
        if (update->has_health) dev->health = (health_status_t)update->health;
        return;
    }
}

void data_store_set_active_scene(const char *scene_id)
{
    if (scene_id) {
        snprintf(store_data.active_scene_id, sizeof(store_data.active_scene_id),
                 "%s", scene_id);
    } else {
        store_data.active_scene_id[0] = '\0';
    }
}

bool data_store_is_live(void) { return store_live; }
void data_store_set_live(bool live) { store_live = live; }
