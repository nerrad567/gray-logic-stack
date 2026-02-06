/**
 * Panel configuration — loads from environment variables (SDL simulator).
 */
#include "net/panel_config.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

bool panel_config_load(panel_config_t *cfg)
{
    if (!cfg) return false;
    memset(cfg, 0, sizeof(*cfg));

    const char *url = getenv("GRAYLOGIC_URL");
    const char *token = getenv("GRAYLOGIC_TOKEN");
    const char *room = getenv("GRAYLOGIC_ROOM");
    const char *mqtt_host = getenv("GRAYLOGIC_MQTT_HOST");
    const char *mqtt_port = getenv("GRAYLOGIC_MQTT_PORT");

    snprintf(cfg->server_url, sizeof(cfg->server_url), "%s",
             url ? url : "http://localhost:8090");
    snprintf(cfg->mqtt_host, sizeof(cfg->mqtt_host), "%s",
             mqtt_host ? mqtt_host : "localhost");
    cfg->mqtt_port = mqtt_port ? atoi(mqtt_port) : 1883;

    if (token) snprintf(cfg->panel_token, sizeof(cfg->panel_token), "%s", token);
    if (room)  snprintf(cfg->room_id, sizeof(cfg->room_id), "%s", room);

    bool valid = panel_config_is_valid(cfg);
    if (valid) {
        printf("[config] server=%s room=%s mqtt=%s:%d\n",
               cfg->server_url, cfg->room_id, cfg->mqtt_host, cfg->mqtt_port);
    } else {
        printf("[config] incomplete — running in demo mode\n");
    }
    return valid;
}

bool panel_config_is_valid(const panel_config_t *cfg)
{
    return cfg && cfg->panel_token[0] != '\0' && cfg->room_id[0] != '\0';
}
