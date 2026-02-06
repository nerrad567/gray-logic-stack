/**
 * Data model utilities and demo data for development.
 */
#include "data/data_model.h"
#include <string.h>

bool device_has_cap(const device_t *dev, device_capability_t cap)
{
    for (int i = 0; i < dev->cap_count; i++) {
        if (dev->capabilities[i] == cap) return true;
    }
    return false;
}

/* Hardcoded demo room for Phase 1 visual development */
void demo_data_create(room_data_t *data)
{
    memset(data, 0, sizeof(*data));

    /* Room */
    strncpy(data->room.id, "room-living-1", sizeof(data->room.id) - 1);
    strncpy(data->room.name, "Living Room", sizeof(data->room.name) - 1);
    data->room.device_count = 5;
    data->room.scene_count = 4;
    data->room.sort_order = 1;

    /* --- Devices --- */

    /* Ceiling light — dimmable */
    device_t *d = &data->devices[0];
    strncpy(d->id, "light-living-ceiling", sizeof(d->id) - 1);
    strncpy(d->name, "Ceiling", sizeof(d->name) - 1);
    strncpy(d->room_id, "room-living-1", sizeof(d->room_id) - 1);
    d->domain = DOMAIN_LIGHTING;
    d->capabilities[0] = CAP_ON_OFF;
    d->capabilities[1] = CAP_DIM;
    d->cap_count = 2;
    d->health = HEALTH_ONLINE;
    d->on = true;
    d->level = 75;

    /* Floor lamp — dimmable */
    d = &data->devices[1];
    strncpy(d->id, "light-living-floor", sizeof(d->id) - 1);
    strncpy(d->name, "Floor Lamp", sizeof(d->name) - 1);
    strncpy(d->room_id, "room-living-1", sizeof(d->room_id) - 1);
    d->domain = DOMAIN_LIGHTING;
    d->capabilities[0] = CAP_ON_OFF;
    d->capabilities[1] = CAP_DIM;
    d->cap_count = 2;
    d->health = HEALTH_ONLINE;
    d->on = true;
    d->level = 40;

    /* Reading light — switch only */
    d = &data->devices[2];
    strncpy(d->id, "light-living-reading", sizeof(d->id) - 1);
    strncpy(d->name, "Reading", sizeof(d->name) - 1);
    strncpy(d->room_id, "room-living-1", sizeof(d->room_id) - 1);
    d->domain = DOMAIN_LIGHTING;
    d->capabilities[0] = CAP_ON_OFF;
    d->cap_count = 1;
    d->health = HEALTH_ONLINE;
    d->on = true;
    d->level = 100;

    /* Blind — position control */
    d = &data->devices[3];
    strncpy(d->id, "blind-living-main", sizeof(d->id) - 1);
    strncpy(d->name, "Blinds", sizeof(d->name) - 1);
    strncpy(d->room_id, "room-living-1", sizeof(d->room_id) - 1);
    d->domain = DOMAIN_BLINDS;
    d->capabilities[0] = CAP_POSITION;
    d->cap_count = 1;
    d->health = HEALTH_ONLINE;
    d->on = true;
    d->position = 50;

    /* Thermostat — temperature read + setpoint */
    d = &data->devices[4];
    strncpy(d->id, "climate-living-thermo", sizeof(d->id) - 1);
    strncpy(d->name, "Thermostat", sizeof(d->name) - 1);
    strncpy(d->room_id, "room-living-1", sizeof(d->room_id) - 1);
    d->domain = DOMAIN_CLIMATE;
    d->capabilities[0] = CAP_TEMPERATURE_READ;
    d->capabilities[1] = CAP_TEMPERATURE_SET;
    d->cap_count = 2;
    d->health = HEALTH_ONLINE;
    d->on = true;
    d->temperature = 22.5f;
    d->setpoint = 22.0f;

    data->device_count = 5;

    /* --- Scenes --- */

    scene_t *s = &data->scenes[0];
    strncpy(s->id, "scene-evening", sizeof(s->id) - 1);
    strncpy(s->name, "Evening", sizeof(s->name) - 1);
    strncpy(s->room_id, "room-living-1", sizeof(s->room_id) - 1);
    strncpy(s->colour, "#F5A623", sizeof(s->colour) - 1);
    strncpy(s->icon, "evening", sizeof(s->icon) - 1);
    s->enabled = true;
    s->sort_order = 1;

    s = &data->scenes[1];
    strncpy(s->id, "scene-movie", sizeof(s->id) - 1);
    strncpy(s->name, "Movie", sizeof(s->name) - 1);
    strncpy(s->room_id, "room-living-1", sizeof(s->room_id) - 1);
    strncpy(s->colour, "#CC5500", sizeof(s->colour) - 1);
    strncpy(s->icon, "movie", sizeof(s->icon) - 1);
    s->enabled = true;
    s->sort_order = 2;

    s = &data->scenes[2];
    strncpy(s->id, "scene-morning", sizeof(s->id) - 1);
    strncpy(s->name, "Morning", sizeof(s->name) - 1);
    strncpy(s->room_id, "room-living-1", sizeof(s->room_id) - 1);
    strncpy(s->colour, "#FFF8E7", sizeof(s->colour) - 1);
    strncpy(s->icon, "morning", sizeof(s->icon) - 1);
    s->enabled = true;
    s->sort_order = 3;

    s = &data->scenes[3];
    strncpy(s->id, "scene-all-off", sizeof(s->id) - 1);
    strncpy(s->name, "All Off", sizeof(s->name) - 1);
    strncpy(s->room_id, "room-living-1", sizeof(s->room_id) - 1);
    strncpy(s->colour, "#6B7B3A", sizeof(s->colour) - 1);
    strncpy(s->icon, "off", sizeof(s->icon) - 1);
    s->enabled = true;
    s->sort_order = 4;

    data->scene_count = 4;

    /* Evening is the active scene */
    strncpy(data->active_scene_id, "scene-evening", sizeof(data->active_scene_id) - 1);
}
