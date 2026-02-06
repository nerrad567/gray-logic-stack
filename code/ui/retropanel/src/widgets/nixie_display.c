/**
 * Nixie display — glowing orange numerals on dark rounded-rect.
 *
 * The glow is achieved with a strong orange shadow behind the text.
 * The container has a dark brown border to simulate a nixie tube housing.
 */
#include "widgets/nixie_display.h"
#include "theme/retro_colors.h"
#include "theme/retro_theme.h"
#include <stdio.h>
#include <stdlib.h>

nixie_display_t *nixie_display_create(lv_obj_t *parent, const char *unit, float value)
{
    nixie_display_t *nd = lv_malloc(sizeof(nixie_display_t));
    if (!nd) return NULL;

    /* Dark tube housing */
    nd->container = lv_obj_create(parent);
    lv_obj_set_size(nd->container, LV_SIZE_CONTENT, LV_SIZE_CONTENT);
    lv_obj_set_style_bg_color(nd->container, RETRO_NIXIE_BG, 0);
    lv_obj_set_style_bg_opa(nd->container, LV_OPA_COVER, 0);
    lv_obj_set_style_radius(nd->container, 6, 0);
    lv_obj_set_style_border_color(nd->container, RETRO_DARK_BROWN, 0);
    lv_obj_set_style_border_width(nd->container, 2, 0);
    lv_obj_set_style_pad_hor(nd->container, 10, 0);
    lv_obj_set_style_pad_ver(nd->container, 4, 0);
    lv_obj_set_flex_flow(nd->container, LV_FLEX_FLOW_ROW);
    lv_obj_set_flex_align(nd->container, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_END, LV_FLEX_ALIGN_CENTER);
    lv_obj_remove_flag(nd->container, LV_OBJ_FLAG_SCROLLABLE);
    lv_obj_set_style_pad_column(nd->container, 2, 0);
    /* Inner glow on the container itself */
    lv_obj_set_style_shadow_color(nd->container, RETRO_GLOW_SHADOW, 0);
    lv_obj_set_style_shadow_width(nd->container, 20, 0);
    lv_obj_set_style_shadow_spread(nd->container, -4, 0);
    lv_obj_set_style_shadow_opa(nd->container, LV_OPA_20, 0);

    /* Value digits — large nixie font */
    nd->value_label = lv_label_create(nd->container);
    lv_obj_set_style_text_font(nd->value_label, retro_font_nixie_sm(), 0);
    lv_obj_set_style_text_color(nd->value_label, RETRO_NIXIE_GLOW, 0);

    /* Unit suffix */
    nd->unit_label = lv_label_create(nd->container);
    lv_label_set_text(nd->unit_label, unit);
    lv_obj_set_style_text_font(nd->unit_label, retro_font_body(), 0);
    lv_obj_set_style_text_color(nd->unit_label, RETRO_AMBER_DIM, 0);

    nixie_display_set_value(nd, value);
    return nd;
}

void nixie_display_set_value(nixie_display_t *nd, float value)
{
    if (!nd) return;
    char buf[16];
    snprintf(buf, sizeof(buf), "%.1f", (double)value);
    lv_label_set_text(nd->value_label, buf);
}

void nixie_display_set_int(nixie_display_t *nd, int value)
{
    if (!nd) return;
    char buf[16];
    snprintf(buf, sizeof(buf), "%d", value);
    lv_label_set_text(nd->value_label, buf);
}
