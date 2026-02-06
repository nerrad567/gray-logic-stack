/**
 * Blind slider — compact horizontal slider with name and value.
 */
#include "widgets/blind_slider.h"
#include "theme/retro_colors.h"
#include "theme/retro_theme.h"
#include <stdio.h>
#include <stdlib.h>

static void update_label(blind_slider_t *bs, int value)
{
    char buf[8];
    snprintf(buf, sizeof(buf), "%d%%", value);
    lv_label_set_text(bs->value_label, buf);
}

static void slider_event_cb(lv_event_t *e)
{
    lv_obj_t *slider = lv_event_get_target(e);
    blind_slider_t *bs = (blind_slider_t *)lv_event_get_user_data(e);
    update_label(bs, lv_slider_get_value(slider));
}

blind_slider_t *blind_slider_create(lv_obj_t *parent, const char *name, int value)
{
    blind_slider_t *bs = lv_malloc(sizeof(blind_slider_t));
    if (!bs) return NULL;

    bs->container = lv_obj_create(parent);
    lv_obj_set_size(bs->container, lv_pct(100), LV_SIZE_CONTENT);
    lv_obj_set_flex_flow(bs->container, LV_FLEX_FLOW_ROW);
    lv_obj_set_flex_align(bs->container, LV_FLEX_ALIGN_START, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_bg_opa(bs->container, LV_OPA_TRANSP, 0);
    lv_obj_set_style_border_width(bs->container, 0, 0);
    lv_obj_set_style_pad_hor(bs->container, 6, 0);
    lv_obj_set_style_pad_ver(bs->container, 2, 0);
    lv_obj_set_style_pad_column(bs->container, 8, 0);
    lv_obj_remove_flag(bs->container, LV_OBJ_FLAG_SCROLLABLE);

    bs->name_label = lv_label_create(bs->container);
    lv_label_set_text(bs->name_label, name);
    lv_obj_set_style_text_color(bs->name_label, RETRO_AMBER, 0);
    lv_obj_set_style_text_font(bs->name_label, retro_font_body(), 0);
    lv_obj_set_width(bs->name_label, 90);
    lv_label_set_long_mode(bs->name_label, LV_LABEL_LONG_CLIP);

    bs->slider = lv_slider_create(bs->container);
    lv_slider_set_range(bs->slider, 0, 100);
    lv_slider_set_value(bs->slider, value, LV_ANIM_OFF);
    lv_obj_set_height(bs->slider, 10);
    lv_obj_set_flex_grow(bs->slider, 1);
    lv_obj_add_event_cb(bs->slider, slider_event_cb, LV_EVENT_VALUE_CHANGED, bs);

    bs->value_label = lv_label_create(bs->container);
    lv_obj_set_style_text_font(bs->value_label, retro_font_body(), 0);
    lv_obj_set_style_text_color(bs->value_label, RETRO_NIXIE_GLOW, 0);
    lv_obj_set_width(bs->value_label, 40);
    lv_obj_set_style_text_align(bs->value_label, LV_TEXT_ALIGN_RIGHT, 0);
    update_label(bs, value);

    return bs;
}

void blind_slider_set_value(blind_slider_t *bs, int value)
{
    if (!bs) return;
    if (value < 0) value = 0;
    if (value > 100) value = 100;
    lv_slider_set_value(bs->slider, value, LV_ANIM_ON);
    update_label(bs, value);
}

int blind_slider_get_value(const blind_slider_t *bs)
{
    if (!bs) return 0;
    return lv_slider_get_value(bs->slider);
}
