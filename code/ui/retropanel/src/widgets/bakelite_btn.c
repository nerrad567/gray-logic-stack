/**
 * Bakelite button — raised brown toggle with 3D shadow.
 *
 * When "on", the button lights up with amber text and a brighter border.
 * When "off", it dims to a muted brown. Clicking toggles the state
 * with a shadow-shrink press animation (handled by theme pressed style).
 */
#include "widgets/bakelite_btn.h"
#include "theme/retro_colors.h"
#include <stdlib.h>

static lv_style_t style_on;
static lv_style_t style_off;
static bool styles_init = false;

static void init_btn_styles(void)
{
    if (styles_init) return;
    styles_init = true;

    lv_style_init(&style_on);
    lv_style_set_text_color(&style_on, RETRO_AMBER_BRIGHT);
    lv_style_set_border_color(&style_on, RETRO_AMBER);
    lv_style_set_bg_color(&style_on, RETRO_DARK_BROWN);
    lv_style_set_shadow_color(&style_on, RETRO_GLOW_SHADOW);
    lv_style_set_shadow_width(&style_on, 6);
    lv_style_set_shadow_opa(&style_on, LV_OPA_30);

    lv_style_init(&style_off);
    lv_style_set_text_color(&style_off, RETRO_AMBER_DIM);
    lv_style_set_border_color(&style_off, RETRO_MED_BROWN);
    lv_style_set_bg_color(&style_off, RETRO_NEAR_BLACK);
    lv_style_set_shadow_opa(&style_off, LV_OPA_20);
}

static void apply_visual_state(bakelite_btn_t *bb)
{
    if (bb->toggled) {
        lv_obj_remove_style(bb->btn, &style_off, 0);
        lv_obj_add_style(bb->btn, &style_on, 0);
    } else {
        lv_obj_remove_style(bb->btn, &style_on, 0);
        lv_obj_add_style(bb->btn, &style_off, 0);
    }
}

static void click_cb(lv_event_t *e)
{
    bakelite_btn_t *bb = (bakelite_btn_t *)lv_event_get_user_data(e);
    bb->toggled = !bb->toggled;
    apply_visual_state(bb);
}

bakelite_btn_t *bakelite_btn_create(lv_obj_t *parent, const char *text, bool toggled)
{
    init_btn_styles();

    bakelite_btn_t *bb = lv_malloc(sizeof(bakelite_btn_t));
    if (!bb) return NULL;

    bb->toggled = toggled;

    bb->btn = lv_button_create(parent);
    lv_obj_set_size(bb->btn, LV_SIZE_CONTENT, LV_SIZE_CONTENT);

    bb->label = lv_label_create(bb->btn);
    lv_label_set_text(bb->label, text);
    lv_obj_center(bb->label);

    apply_visual_state(bb);
    lv_obj_add_event_cb(bb->btn, click_cb, LV_EVENT_CLICKED, bb);

    return bb;
}

void bakelite_btn_set_state(bakelite_btn_t *bb, bool toggled)
{
    if (!bb) return;
    bb->toggled = toggled;
    apply_visual_state(bb);
}

bool bakelite_btn_get_state(const bakelite_btn_t *bb)
{
    if (!bb) return false;
    return bb->toggled;
}

lv_obj_t *bakelite_btn_get_obj(const bakelite_btn_t *bb)
{
    if (!bb) return NULL;
    return bb->btn;
}
