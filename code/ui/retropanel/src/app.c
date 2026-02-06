/**
 * App initialisation — boot sequence for the retro panel.
 *
 * 1. Load config from environment
 * 2. If config valid: REST load hierarchy → devices → scenes
 * 3. If config invalid or REST fails: use hardcoded demo data
 * 4. Start MQTT client for live state updates
 * 5. Main tick drains MQTT updates and refreshes widgets
 */
#include "app.h"
#include "theme/retro_theme.h"
#include "data/data_model.h"
#include "data/data_store.h"
#include "net/panel_config.h"
#include "net/rest_client.h"
#include "net/mqtt_client.h"
#include "screens/scr_room_view.h"
#include <stdio.h>
#include <string.h>

static room_view_t *current_view = NULL;
static panel_config_t config;
static bool networking_active = false;

/* ── MQTT callbacks (called on LVGL thread via drain) ─────────────── */

static void on_state_update(const mqtt_state_update_t *update, void *user_data)
{
    (void)user_data;
    /* Update the data store */
    data_store_apply_update(update);

    /* Refresh the widget */
    if (current_view) {
        const room_data_t *data = data_store_get_room_data();
        for (int i = 0; i < data->device_count; i++) {
            if (strcmp(data->devices[i].id, update->device_id) == 0) {
                scr_room_view_update_device(current_view, update->device_id,
                                            &data->devices[i]);
                break;
            }
        }
    }
}

static void on_scene_event(const mqtt_scene_event_t *event, void *user_data)
{
    (void)user_data;
    data_store_set_active_scene(event->scene_id);

    if (current_view) {
        const room_data_t *data = data_store_get_room_data();
        scr_room_view_update_scene(current_view, event->scene_id,
                                   data->scenes, data->scene_count);
    }
}

/* ── Boot sequence ────────────────────────────────────────────────── */

static bool try_live_boot(room_data_t *data)
{
    if (!panel_config_is_valid(&config)) return false;

    rest_client_init(&config);

    /* Load rooms from hierarchy */
    room_t rooms[MAX_ROOMS];
    int room_count = rest_load_rooms(rooms, MAX_ROOMS);
    if (room_count == 0) {
        printf("[app] no rooms found — falling back to demo\n");
        rest_client_cleanup();
        return false;
    }

    /* Find the configured room */
    int room_idx = -1;
    for (int i = 0; i < room_count; i++) {
        if (strcmp(rooms[i].id, config.room_id) == 0) {
            room_idx = i;
            break;
        }
    }
    if (room_idx < 0) {
        printf("[app] room '%s' not found in hierarchy\n", config.room_id);
        /* Use the first room as fallback */
        room_idx = 0;
        printf("[app] using first room: %s (%s)\n", rooms[0].name, rooms[0].id);
    }

    memset(data, 0, sizeof(*data));
    data->room = rooms[room_idx];

    /* Load devices */
    data->device_count = rest_load_devices(data->room.id, data->devices,
                                           MAX_DEVICES_PER_ROOM);

    /* Load scenes */
    data->scene_count = rest_load_scenes(data->room.id, data->scenes,
                                         MAX_SCENES_PER_ROOM,
                                         data->active_scene_id,
                                         sizeof(data->active_scene_id));

    printf("[app] live boot: %s — %d devices, %d scenes\n",
           data->room.name, data->device_count, data->scene_count);

    /* Start MQTT for live updates */
    if (mqtt_client_init(&config)) {
        printf("[app] MQTT connected — live updates active\n");
    } else {
        printf("[app] MQTT failed — REST-only mode\n");
    }

    return true;
}

void app_init(void)
{
    /* Apply the retro theme to the default display */
    lv_display_t *disp = lv_display_get_default();
    retro_theme_init(disp);

    /* Try to load config and boot with live data */
    bool live = panel_config_load(&config);
    room_data_t room_data;

    if (live && try_live_boot(&room_data)) {
        data_store_init(&room_data);
        data_store_set_live(true);
        networking_active = true;
    } else {
        /* Fall back to demo data */
        demo_data_create(&room_data);
        data_store_init(&room_data);
        data_store_set_live(false);
        printf("[app] running in demo mode (set GRAYLOGIC_TOKEN and GRAYLOGIC_ROOM for live)\n");
    }

    /* Build and load the room view screen */
    current_view = scr_room_view_create(&room_data);
    if (current_view) {
        lv_screen_load(current_view->screen);
    }
}

void app_tick(void)
{
    if (networking_active) {
        mqtt_client_drain_updates(on_state_update, on_scene_event, NULL);
    }
}

void app_cleanup(void)
{
    if (networking_active) {
        mqtt_client_cleanup();
        rest_client_cleanup();
        networking_active = false;
    }
}
