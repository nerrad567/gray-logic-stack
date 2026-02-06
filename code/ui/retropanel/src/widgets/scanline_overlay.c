/**
 * CRT scanline overlay — draws subtle horizontal lines across the screen.
 *
 * Every 3rd pixel row gets a semi-transparent dark line.
 * The overlay sits on top of everything but passes through all input.
 */
#include "widgets/scanline_overlay.h"
#include "theme/retro_colors.h"

#define SCANLINE_SPACING  3
#define SCANLINE_ALPHA    25   /* more visible than before */

static void draw_event_cb(lv_event_t *e)
{
    lv_event_code_t code = lv_event_get_code(e);
    if (code != LV_EVENT_DRAW_MAIN) return;

    lv_layer_t *layer = lv_event_get_layer(e);
    lv_obj_t *obj = lv_event_get_target(e);
    lv_area_t coords;
    lv_obj_get_coords(obj, &coords);

    lv_draw_rect_dsc_t rect_dsc;
    lv_draw_rect_dsc_init(&rect_dsc);
    rect_dsc.bg_color = lv_color_black();
    rect_dsc.bg_opa = SCANLINE_ALPHA;
    rect_dsc.border_width = 0;
    rect_dsc.radius = 0;

    for (int y = coords.y1; y <= coords.y2; y += SCANLINE_SPACING) {
        lv_area_t line_area = {
            .x1 = coords.x1,
            .y1 = y,
            .x2 = coords.x2,
            .y2 = y
        };
        lv_draw_rect(layer, &rect_dsc, &line_area);
    }
}

lv_obj_t *scanline_overlay_create(lv_obj_t *screen)
{
    lv_obj_t *overlay = lv_obj_create(screen);
    lv_obj_set_size(overlay, lv_pct(100), lv_pct(100));
    lv_obj_set_pos(overlay, 0, 0);
    lv_obj_set_style_bg_opa(overlay, LV_OPA_TRANSP, 0);
    lv_obj_set_style_border_width(overlay, 0, 0);
    lv_obj_set_style_pad_all(overlay, 0, 0);
    lv_obj_remove_flag(overlay, LV_OBJ_FLAG_CLICKABLE | LV_OBJ_FLAG_SCROLLABLE);
    lv_obj_add_flag(overlay, LV_OBJ_FLAG_IGNORE_LAYOUT);
    lv_obj_add_event_cb(overlay, draw_event_cb, LV_EVENT_DRAW_MAIN, NULL);
    return overlay;
}
