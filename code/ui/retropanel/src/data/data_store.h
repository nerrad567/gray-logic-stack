/**
 * Data store — thread-safe in-memory store for the current room.
 *
 * Holds the room data (devices, scenes) and provides update functions
 * that can be called from the MQTT thread. The LVGL thread reads via
 * data_store_get_room_data() which returns a snapshot.
 */
#ifndef DATA_STORE_H
#define DATA_STORE_H

#include "data/data_model.h"
#include "net/mqtt_client.h"
#include <stdbool.h>

/** Initialise the data store with room data (takes ownership). */
void data_store_init(const room_data_t *data);

/** Get a pointer to the current room data (read-only, LVGL thread only). */
const room_data_t *data_store_get_room_data(void);

/** Apply a device state update (called from MQTT drain on LVGL thread). */
void data_store_apply_update(const mqtt_state_update_t *update);

/** Set the active scene ID. */
void data_store_set_active_scene(const char *scene_id);

/** Check if networking is available (vs demo mode). */
bool data_store_is_live(void);

/** Mark the store as using live data. */
void data_store_set_live(bool live);

#endif
