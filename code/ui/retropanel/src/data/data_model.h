/**
 * Panel data model — lightweight C structs matching Gray Logic Core entities.
 * Only the fields the panel actually needs for display and control.
 */
#ifndef DATA_MODEL_H
#define DATA_MODEL_H

#include <stdint.h>
#include <stdbool.h>

#define MAX_DEVICES_PER_ROOM  32
#define MAX_SCENES_PER_ROOM   16
#define MAX_ROOMS             16
#define MAX_CAPABILITIES       8

/* Device domains — matches Core's domain constants */
typedef enum {
    DOMAIN_LIGHTING = 0,
    DOMAIN_CLIMATE,
    DOMAIN_BLINDS,
    DOMAIN_AUDIO,
    DOMAIN_OTHER,
    DOMAIN_COUNT
} device_domain_t;

/* Device capabilities — matches Core's capability constants */
typedef enum {
    CAP_ON_OFF = 0,
    CAP_DIM,
    CAP_POSITION,
    CAP_TILT,
    CAP_TEMPERATURE_READ,
    CAP_TEMPERATURE_SET,
    CAP_COLOR_TEMP,
    CAP_SPEED,
    CAP_COUNT
} device_capability_t;

/* Health status — matches Core's health_status field */
typedef enum {
    HEALTH_ONLINE = 0,
    HEALTH_OFFLINE,
    HEALTH_DEGRADED,
    HEALTH_UNKNOWN
} health_status_t;

/* Device — the panel's view of a Gray Logic device */
typedef struct {
    char id[48];
    char name[64];
    char room_id[48];
    device_domain_t domain;
    device_capability_t capabilities[MAX_CAPABILITIES];
    uint8_t cap_count;
    health_status_t health;

    /* State fields — updated via MQTT */
    bool on;
    uint8_t level;       /* 0-100 */
    uint8_t position;    /* 0-100 (blinds) */
    uint8_t tilt;        /* 0-100 (blinds) */
    float temperature;   /* current reading */
    float setpoint;      /* target temperature */
} device_t;

/* Scene — matches Core's scene model */
typedef struct {
    char id[48];
    char name[64];
    char room_id[48];
    char colour[8];      /* hex "#RRGGBB" */
    char icon[32];
    bool enabled;
    int sort_order;
} scene_t;

/* Room — from hierarchy response */
typedef struct {
    char id[48];
    char name[64];
    int device_count;
    int scene_count;
    int sort_order;
} room_t;

/* Room data bundle — everything the panel needs for one room */
typedef struct {
    room_t room;
    device_t devices[MAX_DEVICES_PER_ROOM];
    int device_count;
    scene_t scenes[MAX_SCENES_PER_ROOM];
    int scene_count;
    char active_scene_id[48];  /* currently active scene, empty if none */
} room_data_t;

/* Check if a device has a specific capability */
bool device_has_cap(const device_t *dev, device_capability_t cap);

/* Create demo data for development (hardcoded "Living Room") */
void demo_data_create(room_data_t *data);

#endif /* DATA_MODEL_H */
