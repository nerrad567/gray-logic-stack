/**
 * Panel configuration — server URL, token, room ID.
 *
 * SDL simulator reads from environment variables.
 * ESP32 reads from NVS flash.
 */
#ifndef PANEL_CONFIG_H
#define PANEL_CONFIG_H

#include <stdbool.h>

typedef struct {
    char server_url[256];   /* e.g. "http://localhost:8090" */
    char panel_token[128];  /* X-Panel-Token value */
    char room_id[48];       /* Which room to display */
    char mqtt_host[128];    /* MQTT broker host */
    int  mqtt_port;         /* MQTT broker port */
} panel_config_t;

/**
 * Load configuration from environment variables:
 *   GRAYLOGIC_URL    — Core server URL (default: http://localhost:8090)
 *   GRAYLOGIC_TOKEN  — Panel auth token (required for networking)
 *   GRAYLOGIC_ROOM   — Room ID to display (required for networking)
 *   GRAYLOGIC_MQTT_HOST — MQTT broker (default: localhost)
 *   GRAYLOGIC_MQTT_PORT — MQTT port (default: 1883)
 *
 * Returns true if enough config is present for networking.
 */
bool panel_config_load(panel_config_t *cfg);

/** Returns true if the config has enough data to connect to Core. */
bool panel_config_is_valid(const panel_config_t *cfg);

#endif
