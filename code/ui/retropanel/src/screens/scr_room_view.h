/**
 * Room view screen — main control interface for a single room.
 *
 * Layout: header (room name + temp) → domain sections → scene bar.
 * Supports live updates via MQTT and command sending via REST.
 */
#ifndef SCR_ROOM_VIEW_H
#define SCR_ROOM_VIEW_H

#include "lvgl.h"
#include "data/data_model.h"
#include "widgets/vu_meter.h"
#include "widgets/nixie_display.h"
#include "widgets/bakelite_btn.h"
#include "widgets/scene_bar.h"
#include "widgets/blind_slider.h"

/** Widget slot for a lighting device */
typedef struct {
    char device_id[48];
    bakelite_btn_t *btn;
    vu_meter_t *vu;    /* NULL for switch-only devices */
} lighting_slot_t;

/** Widget slot for a blind device */
typedef struct {
    char device_id[48];
    blind_slider_t *slider;
} blind_slot_t;

/** Widget slot for a climate device */
typedef struct {
    char device_id[48];
    nixie_display_t *temp_display;
    nixie_display_t *setpoint_display;
    float setpoint;  /* local copy for stepper */
} climate_slot_t;

/** Room view — holds all widget references for live updates */
typedef struct {
    lv_obj_t *screen;

    lighting_slot_t lighting[MAX_DEVICES_PER_ROOM];
    int lighting_count;

    blind_slot_t blinds[MAX_DEVICES_PER_ROOM];
    int blind_count;

    climate_slot_t climate[MAX_DEVICES_PER_ROOM];
    int climate_count;

    scene_bar_t *scene_bar;
    nixie_display_t *header_temp;
    char header_temp_device_id[48]; /* device providing header temp */
} room_view_t;

/** Create the room view screen from room data. Returns a handle for live updates. */
room_view_t *scr_room_view_create(const room_data_t *data);

/** Update a device's widgets from an MQTT state change. */
void scr_room_view_update_device(room_view_t *rv, const char *device_id,
                                 const device_t *dev);

/** Update the active scene highlight. */
void scr_room_view_update_scene(room_view_t *rv, const char *scene_id,
                                const scene_t *scenes, int scene_count);

#endif /* SCR_ROOM_VIEW_H */
