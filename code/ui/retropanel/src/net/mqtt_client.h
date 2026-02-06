/**
 * MQTT client — subscribe to device state and scene events.
 *
 * Uses libmosquitto for SDL simulator, esp_mqtt for ESP32.
 * Runs on a background thread; state updates are queued for the LVGL thread.
 */
#ifndef MQTT_CLIENT_H
#define MQTT_CLIENT_H

#include "net/panel_config.h"
#include <stdbool.h>

/** State update from MQTT, queued for the LVGL thread. */
typedef struct {
    char device_id[48];
    bool has_on;      bool on;
    bool has_level;   int  level;
    bool has_pos;     int  position;
    bool has_temp;    float temperature;
    bool has_sp;      float setpoint;
    bool has_health;  int  health;
} mqtt_state_update_t;

/** Scene activation event from MQTT. */
typedef struct {
    char scene_id[48];
    char room_id[48];
} mqtt_scene_event_t;

/**
 * Initialise and connect the MQTT client.
 * Subscribes to device state and scene activation topics.
 * Returns true on successful connection.
 */
bool mqtt_client_init(const panel_config_t *cfg);

/** Disconnect and clean up. */
void mqtt_client_cleanup(void);

/**
 * Drain pending state updates (call from LVGL main loop).
 * Returns the number of updates drained.
 * Each update is passed to the callback.
 */
typedef void (*mqtt_state_cb_t)(const mqtt_state_update_t *update, void *user_data);
typedef void (*mqtt_scene_cb_t)(const mqtt_scene_event_t *event, void *user_data);

int mqtt_client_drain_updates(mqtt_state_cb_t state_cb, mqtt_scene_cb_t scene_cb,
                              void *user_data);

/** Returns true if currently connected to the broker. */
bool mqtt_client_is_connected(void);

#endif
