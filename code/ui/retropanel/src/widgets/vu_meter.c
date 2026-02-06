/**
 * VU meter — compact arc gauge with tick marks and glowing value.
 *
 * Sized to fit in a single row with a toggle button.
 * 48px arc with 16px body font for the value.
 */
#include "widgets/vu_meter.h"
#include "theme/retro_colors.h"
#include "theme/retro_theme.h"
#include <stdio.h>
#include <stdlib.h>
#include <math.h>

#define VU_ARC_SIZE     48
#define VU_ARC_WIDTH    8

#ifndef M_PI
#define M_PI 3.14159265358979323846
#endif

static void update_label(vu_meter_t *vm, int value)
{
    char buf[8];
    snprintf(buf, sizeof(buf), "%d", value);
    lv_label_set_text(vm->value_label, buf);
}

static void draw_ticks(lv_event_t *e)
{
    if (lv_event_get_code(e) != LV_EVENT_DRAW_MAIN_END) return;

    lv_obj_t *arc = lv_event_get_target(e);
    lv_layer_t *layer = lv_event_get_layer(e);
    lv_area_t coords;
    lv_obj_get_coords(arc, &coords);

    int cx = (coords.x1 + coords.x2) / 2;
    int cy = (coords.y1 + coords.y2) / 2;
    int radius = (coords.x2 - coords.x1) / 2 - 1;

    for (int i = 0; i <= 10; i++) {
        float angle_deg = 135.0f + (float)i * 27.0f;
        float angle_rad = angle_deg * (float)M_PI / 180.0f;
        float cos_a = cosf(angle_rad);
        float sin_a = sinf(angle_rad);

        bool major = (i == 0 || i == 5 || i == 10);
        int tick_inner = major ? radius - 10 : radius - 6;
        int tick_outer = radius - 2;

        lv_draw_line_dsc_t line_dsc;
        lv_draw_line_dsc_init(&line_dsc);
        line_dsc.color = major ? RETRO_AMBER : RETRO_AMBER_DIM;
        line_dsc.width = major ? 2 : 1;
        line_dsc.opa = LV_OPA_COVER;
        line_dsc.p1.x = cx + (int)(cos_a * tick_inner);
        line_dsc.p1.y = cy + (int)(sin_a * tick_inner);
        line_dsc.p2.x = cx + (int)(cos_a * tick_outer);
        line_dsc.p2.y = cy + (int)(sin_a * tick_outer);

        lv_draw_line(layer, &line_dsc);
    }
}

static void arc_event_cb(lv_event_t *e)
{
    lv_obj_t *arc = lv_event_get_target(e);
    vu_meter_t *vm = (vu_meter_t *)lv_event_get_user_data(e);
    update_label(vm, lv_arc_get_value(arc));
}

vu_meter_t *vu_meter_create(lv_obj_t *parent, const char *name, int value)
{
    vu_meter_t *vm = lv_malloc(sizeof(vu_meter_t));
    if (!vm) return NULL;

    /* No outer container — just the arc directly in parent */
    vm->container = NULL;

    vm->arc = lv_arc_create(parent);
    lv_obj_set_size(vm->arc, VU_ARC_SIZE, VU_ARC_SIZE);
    lv_arc_set_rotation(vm->arc, 135);
    lv_arc_set_bg_angles(vm->arc, 0, 270);
    lv_arc_set_range(vm->arc, 0, 100);
    lv_arc_set_value(vm->arc, value);

    lv_obj_set_style_arc_width(vm->arc, VU_ARC_WIDTH, LV_PART_MAIN);
    lv_obj_set_style_arc_width(vm->arc, VU_ARC_WIDTH, LV_PART_INDICATOR);
    lv_obj_set_style_arc_color(vm->arc, lv_color_hex(0x1A1F22), LV_PART_MAIN);
    lv_obj_set_style_arc_color(vm->arc, RETRO_AMBER, LV_PART_INDICATOR);

    /* Small glowing knob */
    lv_obj_set_style_bg_color(vm->arc, RETRO_AMBER_BRIGHT, LV_PART_KNOB);
    lv_obj_set_style_pad_all(vm->arc, 2, LV_PART_KNOB);
    lv_obj_set_style_border_width(vm->arc, 0, LV_PART_KNOB);
    lv_obj_set_style_shadow_color(vm->arc, RETRO_GLOW_SHADOW, LV_PART_KNOB);
    lv_obj_set_style_shadow_width(vm->arc, 8, LV_PART_KNOB);
    lv_obj_set_style_shadow_opa(vm->arc, LV_OPA_50, LV_PART_KNOB);

    /* Tick marks */
    lv_obj_add_event_cb(vm->arc, draw_ticks, LV_EVENT_DRAW_MAIN_END, NULL);

    /* Value label — use body font (16px), fits cleanly in 48px arc */
    vm->value_label = lv_label_create(vm->arc);
    lv_obj_center(vm->value_label);
    lv_obj_set_style_text_font(vm->value_label, retro_font_body(), 0);
    lv_obj_set_style_text_color(vm->value_label, RETRO_NIXIE_GLOW, 0);
    update_label(vm, value);

    lv_obj_add_event_cb(vm->arc, arc_event_cb, LV_EVENT_VALUE_CHANGED, vm);

    /* Name label (often empty in row layout) */
    vm->name_label = NULL;
    if (name && name[0]) {
        vm->name_label = lv_label_create(parent);
        lv_label_set_text(vm->name_label, name);
        lv_obj_set_style_text_font(vm->name_label, retro_font_body(), 0);
        lv_obj_set_style_text_color(vm->name_label, RETRO_AMBER_DIM, 0);
    }

    return vm;
}

void vu_meter_set_value(vu_meter_t *vm, int value)
{
    if (!vm) return;
    if (value < 0) value = 0;
    if (value > 100) value = 100;
    lv_arc_set_value(vm->arc, value);
    update_label(vm, value);
}

int vu_meter_get_value(const vu_meter_t *vm)
{
    if (!vm) return 0;
    return lv_arc_get_value(vm->arc);
}
