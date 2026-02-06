/**
 * Room view screen — dense instrument panel layout for 480x320.
 *
 * Every pixel counts. Layout:
 *   Header 32px | Sections ~200px | Scene bar 36px = 268px (fits 320px)
 *
 * Phase 2/3: Supports live MQTT updates and REST command sending.
 * Widget event callbacks fire commands when data_store_is_live().
 */
#include "screens/scr_room_view.h"
#include "theme/retro_colors.h"
#include "theme/retro_theme.h"
#include "net/command.h"
#include "data/data_store.h"
#include "widgets/scanline_overlay.h"
#include <stdio.h>
#include <string.h>

/* ── Command callback context ────────────────────────────────────── */

typedef struct {
    char device_id[48];
} cmd_ctx_t;

typedef struct {
    char device_id[48];
    nixie_display_t *setpoint_display;
    float setpoint;
} climate_ctx_t;

/* ── Header ───────────────────────────────────────────────────────── */

static nixie_display_t *create_header(lv_obj_t *parent, const room_data_t *data,
                                      char *temp_device_id, int id_size)
{
    nixie_display_t *header_temp = NULL;

    lv_obj_t *hdr = lv_obj_create(parent);
    lv_obj_set_size(hdr, lv_pct(100), LV_SIZE_CONTENT);
    lv_obj_set_flex_flow(hdr, LV_FLEX_FLOW_ROW);
    lv_obj_set_flex_align(hdr, LV_FLEX_ALIGN_SPACE_BETWEEN, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_bg_color(hdr, RETRO_DARK_BROWN, 0);
    lv_obj_set_style_bg_grad_color(hdr, RETRO_NEAR_BLACK, 0);
    lv_obj_set_style_bg_grad_dir(hdr, LV_GRAD_DIR_VER, 0);
    lv_obj_set_style_bg_opa(hdr, LV_OPA_COVER, 0);
    lv_obj_set_style_border_color(hdr, RETRO_AMBER_DIM, 0);
    lv_obj_set_style_border_width(hdr, 1, 0);
    lv_obj_set_style_border_side(hdr, LV_BORDER_SIDE_BOTTOM, 0);
    lv_obj_set_style_pad_hor(hdr, 8, 0);
    lv_obj_set_style_pad_ver(hdr, 4, 0);
    lv_obj_set_style_radius(hdr, 0, 0);
    lv_obj_remove_flag(hdr, LV_OBJ_FLAG_SCROLLABLE);

    lv_obj_t *name = lv_label_create(hdr);
    lv_label_set_text(name, data->room.name);
    lv_obj_set_style_text_font(name, retro_font_heading(), 0);
    lv_obj_set_style_text_color(name, RETRO_CREAM, 0);

    for (int i = 0; i < data->device_count; i++) {
        if (data->devices[i].domain == DOMAIN_CLIMATE &&
            device_has_cap(&data->devices[i], CAP_TEMPERATURE_READ)) {
            header_temp = nixie_display_create(hdr, "\xC2\xB0" "C",
                                               data->devices[i].temperature);
            if (temp_device_id) {
                snprintf(temp_device_id, id_size, "%s", data->devices[i].id);
            }
            break;
        }
    }
    return header_temp;
}

/* ── Section divider ──────────────────────────────────────────────── */

static void create_section_label(lv_obj_t *parent, const char *title)
{
    lv_obj_t *row = lv_obj_create(parent);
    lv_obj_set_size(row, lv_pct(100), LV_SIZE_CONTENT);
    lv_obj_set_flex_flow(row, LV_FLEX_FLOW_ROW);
    lv_obj_set_flex_align(row, LV_FLEX_ALIGN_START, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_bg_opa(row, LV_OPA_TRANSP, 0);
    lv_obj_set_style_border_width(row, 0, 0);
    lv_obj_set_style_pad_left(row, 6, 0);
    lv_obj_set_style_pad_ver(row, 0, 0);
    lv_obj_set_style_pad_column(row, 5, 0);
    lv_obj_remove_flag(row, LV_OBJ_FLAG_SCROLLABLE);

    lv_obj_t *dot = lv_obj_create(row);
    lv_obj_set_size(dot, 4, 4);
    lv_obj_set_style_radius(dot, LV_RADIUS_CIRCLE, 0);
    lv_obj_set_style_bg_color(dot, RETRO_AMBER, 0);
    lv_obj_set_style_bg_opa(dot, LV_OPA_COVER, 0);
    lv_obj_set_style_border_width(dot, 0, 0);
    lv_obj_set_style_shadow_color(dot, RETRO_GLOW_SHADOW, 0);
    lv_obj_set_style_shadow_width(dot, 4, 0);
    lv_obj_set_style_shadow_opa(dot, LV_OPA_40, 0);
    lv_obj_remove_flag(dot, LV_OBJ_FLAG_SCROLLABLE | LV_OBJ_FLAG_CLICKABLE);

    lv_obj_t *lbl = lv_label_create(row);
    lv_label_set_text(lbl, title);
    lv_obj_set_style_text_font(lbl, retro_font_body(), 0);
    lv_obj_set_style_text_color(lbl, RETRO_OLIVE, 0);
}

/* ── Lighting command callbacks ───────────────────────────────────── */

static void on_toggle_click(lv_event_t *e)
{
    cmd_ctx_t *ctx = (cmd_ctx_t *)lv_event_get_user_data(e);
    bakelite_btn_t *btn = NULL;

    /* Find the btn from the event target */
    lv_obj_t *target = lv_event_get_target(e);
    /* The btn struct is stored as user data on the button object */
    btn = (bakelite_btn_t *)lv_obj_get_user_data(target);

    if (btn) {
        /* Toggle locally for instant feedback */
        bakelite_btn_set_state(btn, !btn->toggled);
    }

    /* Send command if live */
    if (ctx && ctx->device_id[0]) {
        cmd_toggle(ctx->device_id, btn ? !btn->toggled : false);
    }
}

static void on_level_changed(lv_event_t *e)
{
    cmd_ctx_t *ctx = (cmd_ctx_t *)lv_event_get_user_data(e);
    lv_obj_t *arc = lv_event_get_target(e);
    int value = lv_arc_get_value(arc);

    if (ctx && ctx->device_id[0]) {
        cmd_set_level(ctx->device_id, value);
    }
}

static void on_position_changed(lv_event_t *e)
{
    cmd_ctx_t *ctx = (cmd_ctx_t *)lv_event_get_user_data(e);
    lv_obj_t *slider = lv_event_get_target(e);
    int value = lv_slider_get_value(slider);

    if (ctx && ctx->device_id[0]) {
        cmd_set_position(ctx->device_id, value);
    }
}

/* ── Climate stepper callbacks ────────────────────────────────────── */

static void sp_minus_cb(lv_event_t *e)
{
    climate_ctx_t *ctx = (climate_ctx_t *)lv_event_get_user_data(e);
    ctx->setpoint -= 0.5f;
    if (ctx->setpoint < 5.0f) ctx->setpoint = 5.0f;
    nixie_display_set_value(ctx->setpoint_display, ctx->setpoint);
    cmd_set_setpoint(ctx->device_id, ctx->setpoint);
}

static void sp_plus_cb(lv_event_t *e)
{
    climate_ctx_t *ctx = (climate_ctx_t *)lv_event_get_user_data(e);
    ctx->setpoint += 0.5f;
    if (ctx->setpoint > 35.0f) ctx->setpoint = 35.0f;
    nixie_display_set_value(ctx->setpoint_display, ctx->setpoint);
    cmd_set_setpoint(ctx->device_id, ctx->setpoint);
}

/* ── Scene command callback ───────────────────────────────────────── */

typedef struct {
    room_view_t *rv;
    const scene_t *scenes;
    int scene_count;
} scene_cmd_ctx_t;

static scene_cmd_ctx_t scene_ctx;  /* single instance — one scene bar */

static void on_scene_click(lv_event_t *e)
{
    scene_bar_t *sb = (scene_bar_t *)lv_event_get_user_data(e);
    lv_obj_t *clicked = lv_event_get_target(e);

    for (int i = 0; i < sb->scene_count; i++) {
        if (sb->buttons[i] == clicked ||
            lv_obj_get_parent(clicked) == sb->buttons[i]) {
            sb->active_index = i;
            /* Activate scene via REST */
            if (i < scene_ctx.scene_count) {
                cmd_activate_scene(scene_ctx.scenes[i].id);
            }
            break;
        }
    }
}

/* ── Lighting row ─────────────────────────────────────────────────── */

static void create_lighting_row(lv_obj_t *parent, const device_t *dev,
                                lighting_slot_t *slot)
{
    snprintf(slot->device_id, sizeof(slot->device_id), "%s", dev->id);

    lv_obj_t *row = lv_obj_create(parent);
    lv_obj_set_size(row, lv_pct(100), LV_SIZE_CONTENT);
    lv_obj_set_flex_flow(row, LV_FLEX_FLOW_ROW);
    lv_obj_set_flex_align(row, LV_FLEX_ALIGN_START, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_bg_opa(row, LV_OPA_TRANSP, 0);
    lv_obj_set_style_border_width(row, 0, 0);
    lv_obj_set_style_pad_all(row, 0, 0);
    lv_obj_set_style_pad_left(row, 6, 0);
    lv_obj_set_style_pad_column(row, 6, 0);
    lv_obj_remove_flag(row, LV_OBJ_FLAG_SCROLLABLE);

    /* Toggle button */
    slot->btn = bakelite_btn_create(row, dev->name, dev->on);
    lv_obj_set_size(bakelite_btn_get_obj(slot->btn), LV_SIZE_CONTENT, 30);
    lv_obj_set_style_min_width(bakelite_btn_get_obj(slot->btn), 70, 0);
    lv_obj_set_style_max_width(bakelite_btn_get_obj(slot->btn), 100, 0);

    /* Store btn struct as user data for the toggle callback */
    lv_obj_set_user_data(bakelite_btn_get_obj(slot->btn), slot->btn);

    /* Toggle command callback */
    cmd_ctx_t *toggle_ctx = lv_malloc(sizeof(cmd_ctx_t));
    snprintf(toggle_ctx->device_id, sizeof(toggle_ctx->device_id), "%s", dev->id);
    lv_obj_add_event_cb(bakelite_btn_get_obj(slot->btn), on_toggle_click,
                        LV_EVENT_CLICKED, toggle_ctx);

    /* VU meter for dimmable lights */
    slot->vu = NULL;
    if (device_has_cap(dev, CAP_DIM)) {
        slot->vu = vu_meter_create(row, "", dev->level);

        /* Level command callback */
        cmd_ctx_t *level_ctx = lv_malloc(sizeof(cmd_ctx_t));
        snprintf(level_ctx->device_id, sizeof(level_ctx->device_id), "%s", dev->id);
        lv_obj_add_event_cb(slot->vu->arc, on_level_changed,
                            LV_EVENT_VALUE_CHANGED, level_ctx);
    }
}

/* ── Blind row ────────────────────────────────────────────────────── */

static void create_blind_row(lv_obj_t *parent, const device_t *dev,
                             blind_slot_t *slot)
{
    snprintf(slot->device_id, sizeof(slot->device_id), "%s", dev->id);
    slot->slider = blind_slider_create(parent, dev->name, dev->position);

    /* Position command callback */
    cmd_ctx_t *pos_ctx = lv_malloc(sizeof(cmd_ctx_t));
    snprintf(pos_ctx->device_id, sizeof(pos_ctx->device_id), "%s", dev->id);
    lv_obj_add_event_cb(slot->slider->slider, on_position_changed,
                        LV_EVENT_VALUE_CHANGED, pos_ctx);
}

/* ── Climate row ──────────────────────────────────────────────────── */

static void create_climate_row(lv_obj_t *parent, const device_t *dev,
                               climate_slot_t *slot)
{
    snprintf(slot->device_id, sizeof(slot->device_id), "%s", dev->id);
    slot->setpoint = dev->setpoint;

    lv_obj_t *row = lv_obj_create(parent);
    lv_obj_set_size(row, lv_pct(100), LV_SIZE_CONTENT);
    lv_obj_set_flex_flow(row, LV_FLEX_FLOW_ROW);
    lv_obj_set_flex_align(row, LV_FLEX_ALIGN_SPACE_EVENLY, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_bg_opa(row, LV_OPA_TRANSP, 0);
    lv_obj_set_style_border_width(row, 0, 0);
    lv_obj_set_style_pad_all(row, 1, 0);
    lv_obj_set_style_pad_column(row, 6, 0);
    lv_obj_remove_flag(row, LV_OBJ_FLAG_SCROLLABLE);

    slot->temp_display = nixie_display_create(row, "\xC2\xB0" "C", dev->temperature);

    /* Stepper */
    lv_obj_t *stepper = lv_obj_create(row);
    lv_obj_set_size(stepper, LV_SIZE_CONTENT, LV_SIZE_CONTENT);
    lv_obj_set_flex_flow(stepper, LV_FLEX_FLOW_ROW);
    lv_obj_set_flex_align(stepper, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_bg_opa(stepper, LV_OPA_TRANSP, 0);
    lv_obj_set_style_border_width(stepper, 0, 0);
    lv_obj_set_style_pad_all(stepper, 0, 0);
    lv_obj_set_style_pad_column(stepper, 4, 0);
    lv_obj_remove_flag(stepper, LV_OBJ_FLAG_SCROLLABLE);

    /* Allocate climate context (shared by +/- and stored in slot) */
    climate_ctx_t *ctx = lv_malloc(sizeof(climate_ctx_t));
    snprintf(ctx->device_id, sizeof(ctx->device_id), "%s", dev->id);
    ctx->setpoint = dev->setpoint;

    lv_obj_t *minus_btn = lv_button_create(stepper);
    lv_obj_set_size(minus_btn, 28, 28);
    lv_obj_t *ml = lv_label_create(minus_btn);
    lv_label_set_text(ml, "-");
    lv_obj_center(ml);
    lv_obj_add_event_cb(minus_btn, sp_minus_cb, LV_EVENT_CLICKED, ctx);

    ctx->setpoint_display = nixie_display_create(stepper, "\xC2\xB0", dev->setpoint);
    slot->setpoint_display = ctx->setpoint_display;

    lv_obj_t *plus_btn = lv_button_create(stepper);
    lv_obj_set_size(plus_btn, 28, 28);
    lv_obj_t *pl = lv_label_create(plus_btn);
    lv_label_set_text(pl, "+");
    lv_obj_center(pl);
    lv_obj_add_event_cb(plus_btn, sp_plus_cb, LV_EVENT_CLICKED, ctx);
}

/* ── Screen assembly ──────────────────────────────────────────────── */

room_view_t *scr_room_view_create(const room_data_t *data)
{
    room_view_t *rv = lv_malloc(sizeof(room_view_t));
    if (!rv) return NULL;
    memset(rv, 0, sizeof(*rv));

    rv->screen = lv_obj_create(NULL);
    lv_obj_set_style_bg_color(rv->screen, RETRO_NEAR_BLACK, 0);
    lv_obj_set_style_bg_opa(rv->screen, LV_OPA_COVER, 0);

    lv_obj_t *content = lv_obj_create(rv->screen);
    lv_obj_set_size(content, lv_pct(100), lv_pct(100));
    lv_obj_set_flex_flow(content, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_flex_align(content, LV_FLEX_ALIGN_START, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_bg_opa(content, LV_OPA_TRANSP, 0);
    lv_obj_set_style_border_width(content, 0, 0);
    lv_obj_set_style_pad_all(content, 0, 0);
    lv_obj_set_style_pad_row(content, 0, 0);
    lv_obj_add_flag(content, LV_OBJ_FLAG_SCROLLABLE);
    lv_obj_set_scrollbar_mode(content, LV_SCROLLBAR_MODE_AUTO);

    rv->header_temp = create_header(content, data, rv->header_temp_device_id,
                                    sizeof(rv->header_temp_device_id));

    /* Scan domains present */
    bool has[DOMAIN_COUNT] = {false};
    for (int i = 0; i < data->device_count; i++) {
        if (data->devices[i].domain < DOMAIN_COUNT)
            has[data->devices[i].domain] = true;
    }

    /* Lighting */
    if (has[DOMAIN_LIGHTING]) {
        create_section_label(content, "LIGHTING");
        for (int i = 0; i < data->device_count; i++) {
            if (data->devices[i].domain == DOMAIN_LIGHTING &&
                rv->lighting_count < MAX_DEVICES_PER_ROOM) {
                create_lighting_row(content, &data->devices[i],
                                    &rv->lighting[rv->lighting_count]);
                rv->lighting_count++;
            }
        }
    }

    /* Blinds */
    if (has[DOMAIN_BLINDS]) {
        create_section_label(content, "BLINDS");
        for (int i = 0; i < data->device_count; i++) {
            if (data->devices[i].domain == DOMAIN_BLINDS &&
                rv->blind_count < MAX_DEVICES_PER_ROOM) {
                create_blind_row(content, &data->devices[i],
                                 &rv->blinds[rv->blind_count]);
                rv->blind_count++;
            }
        }
    }

    /* Climate */
    if (has[DOMAIN_CLIMATE]) {
        create_section_label(content, "CLIMATE");
        for (int i = 0; i < data->device_count; i++) {
            if (data->devices[i].domain == DOMAIN_CLIMATE &&
                device_has_cap(&data->devices[i], CAP_TEMPERATURE_READ) &&
                rv->climate_count < MAX_DEVICES_PER_ROOM) {
                create_climate_row(content, &data->devices[i],
                                   &rv->climate[rv->climate_count]);
                rv->climate_count++;
            }
        }
    }

    /* Scene bar */
    rv->scene_bar = NULL;
    if (data->scene_count > 0) {
        rv->scene_bar = scene_bar_create(content, data->scenes, data->scene_count,
                                         data->active_scene_id);

        /* Override scene click to also send commands */
        scene_ctx.rv = rv;
        scene_ctx.scenes = data->scenes;
        scene_ctx.scene_count = data->scene_count;

        /* Register our command callback on each button */
        for (int i = 0; i < rv->scene_bar->scene_count; i++) {
            lv_obj_add_event_cb(rv->scene_bar->buttons[i], on_scene_click,
                                LV_EVENT_CLICKED, rv->scene_bar);
        }
    }

    scanline_overlay_create(rv->screen);
    return rv;
}

/* ── Live update functions ────────────────────────────────────────── */

void scr_room_view_update_device(room_view_t *rv, const char *device_id,
                                 const device_t *dev)
{
    if (!rv || !device_id || !dev) return;

    /* Update header temperature */
    if (rv->header_temp &&
        strcmp(rv->header_temp_device_id, device_id) == 0) {
        nixie_display_set_value(rv->header_temp, dev->temperature);
    }

    /* Update lighting widgets */
    for (int i = 0; i < rv->lighting_count; i++) {
        if (strcmp(rv->lighting[i].device_id, device_id) != 0) continue;
        bakelite_btn_set_state(rv->lighting[i].btn, dev->on);
        if (rv->lighting[i].vu) {
            vu_meter_set_value(rv->lighting[i].vu, dev->level);
        }
        return;
    }

    /* Update blind widgets */
    for (int i = 0; i < rv->blind_count; i++) {
        if (strcmp(rv->blinds[i].device_id, device_id) != 0) continue;
        blind_slider_set_value(rv->blinds[i].slider, dev->position);
        return;
    }

    /* Update climate widgets */
    for (int i = 0; i < rv->climate_count; i++) {
        if (strcmp(rv->climate[i].device_id, device_id) != 0) continue;
        if (rv->climate[i].temp_display) {
            nixie_display_set_value(rv->climate[i].temp_display, dev->temperature);
        }
        if (rv->climate[i].setpoint_display) {
            nixie_display_set_value(rv->climate[i].setpoint_display, dev->setpoint);
        }
        return;
    }
}

void scr_room_view_update_scene(room_view_t *rv, const char *scene_id,
                                const scene_t *scenes, int scene_count)
{
    if (!rv || !rv->scene_bar) return;
    scene_bar_set_active_by_id(rv->scene_bar, scenes, scene_count, scene_id);
}
