/**
 * Scene bar — row of styled scene buttons at the bottom of the room view.
 *
 * Each button has a small colour dot (from the scene's colour field)
 * and the scene name. Active scene gets amber border highlight.
 */
#include "widgets/scene_bar.h"
#include "theme/retro_colors.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

static lv_style_t style_active;
static lv_style_t style_inactive;
static bool styles_init = false;

static void init_scene_styles(void)
{
    if (styles_init) return;
    styles_init = true;

    lv_style_init(&style_active);
    lv_style_set_border_color(&style_active, RETRO_AMBER_BRIGHT);
    lv_style_set_border_width(&style_active, 2);
    lv_style_set_text_color(&style_active, RETRO_AMBER_BRIGHT);
    lv_style_set_bg_color(&style_active, RETRO_DARK_BROWN);
    lv_style_set_shadow_color(&style_active, RETRO_GLOW_SHADOW);
    lv_style_set_shadow_width(&style_active, 8);
    lv_style_set_shadow_opa(&style_active, LV_OPA_30);

    lv_style_init(&style_inactive);
    lv_style_set_border_color(&style_inactive, RETRO_MED_BROWN);
    lv_style_set_border_width(&style_inactive, 1);
    lv_style_set_text_color(&style_inactive, RETRO_AMBER_DIM);
    lv_style_set_bg_color(&style_inactive, RETRO_NEAR_BLACK);
    lv_style_set_shadow_width(&style_inactive, 0);
}

static lv_color_t parse_hex_color(const char *hex)
{
    if (!hex || hex[0] != '#' || strlen(hex) < 7) return RETRO_AMBER;
    unsigned int r, g, b;
    if (sscanf(hex + 1, "%02x%02x%02x", &r, &g, &b) == 3) {
        return lv_color_make((uint8_t)r, (uint8_t)g, (uint8_t)b);
    }
    return RETRO_AMBER;
}

static void update_highlight(scene_bar_t *sb)
{
    for (int i = 0; i < sb->scene_count; i++) {
        lv_obj_remove_style(sb->buttons[i], &style_active, 0);
        lv_obj_remove_style(sb->buttons[i], &style_inactive, 0);
        if (i == sb->active_index) {
            lv_obj_add_style(sb->buttons[i], &style_active, 0);
        } else {
            lv_obj_add_style(sb->buttons[i], &style_inactive, 0);
        }
    }
}

static void scene_click_cb(lv_event_t *e)
{
    scene_bar_t *sb = (scene_bar_t *)lv_event_get_user_data(e);
    lv_obj_t *clicked = lv_event_get_target(e);

    /* Find which button was clicked */
    for (int i = 0; i < sb->scene_count; i++) {
        if (sb->buttons[i] == clicked ||
            lv_obj_get_parent(clicked) == sb->buttons[i]) {
            sb->active_index = i;
            update_highlight(sb);
            break;
        }
    }
}

scene_bar_t *scene_bar_create(lv_obj_t *parent, const scene_t *scenes, int count,
                              const char *active_scene_id)
{
    init_scene_styles();

    scene_bar_t *sb = lv_malloc(sizeof(scene_bar_t));
    if (!sb) return NULL;

    sb->scene_count = count > MAX_SCENES_PER_ROOM ? MAX_SCENES_PER_ROOM : count;
    sb->active_index = -1;

    /* Horizontal row container */
    sb->container = lv_obj_create(parent);
    lv_obj_set_size(sb->container, lv_pct(100), LV_SIZE_CONTENT);
    lv_obj_set_flex_flow(sb->container, LV_FLEX_FLOW_ROW);
    lv_obj_set_flex_align(sb->container, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_bg_opa(sb->container, LV_OPA_TRANSP, 0);
    lv_obj_set_style_border_width(sb->container, 0, 0);
    lv_obj_set_style_pad_all(sb->container, 2, 0);
    lv_obj_set_style_pad_column(sb->container, 6, 0);
    lv_obj_remove_flag(sb->container, LV_OBJ_FLAG_SCROLLABLE);

    for (int i = 0; i < sb->scene_count; i++) {
        const scene_t *sc = &scenes[i];

        lv_obj_t *btn = lv_button_create(sb->container);
        lv_obj_set_size(btn, LV_SIZE_CONTENT, LV_SIZE_CONTENT);
        lv_obj_set_style_pad_hor(btn, 8, 0);
        lv_obj_set_style_pad_ver(btn, 4, 0);
        lv_obj_set_style_radius(btn, 6, 0);
        lv_obj_set_flex_flow(btn, LV_FLEX_FLOW_ROW);
        lv_obj_set_flex_align(btn, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
        lv_obj_set_style_pad_column(btn, 6, 0);

        /* Colour dot */
        lv_obj_t *dot = lv_obj_create(btn);
        lv_obj_set_size(dot, 8, 8);
        lv_obj_set_style_radius(dot, LV_RADIUS_CIRCLE, 0);
        lv_obj_set_style_bg_color(dot, parse_hex_color(sc->colour), 0);
        lv_obj_set_style_bg_opa(dot, LV_OPA_COVER, 0);
        lv_obj_set_style_border_width(dot, 0, 0);
        lv_obj_remove_flag(dot, LV_OBJ_FLAG_SCROLLABLE | LV_OBJ_FLAG_CLICKABLE);

        /* Scene name label */
        lv_obj_t *lbl = lv_label_create(btn);
        lv_label_set_text(lbl, sc->name);
        lv_obj_set_style_text_font(lbl, &lv_font_montserrat_14, 0);

        sb->buttons[i] = btn;

        lv_obj_add_event_cb(btn, scene_click_cb, LV_EVENT_CLICKED, sb);

        /* Check if this is the active scene */
        if (active_scene_id && strcmp(sc->id, active_scene_id) == 0) {
            sb->active_index = i;
        }
    }

    update_highlight(sb);
    return sb;
}

void scene_bar_set_active(scene_bar_t *sb, int index)
{
    if (!sb) return;
    sb->active_index = index;
    update_highlight(sb);
}

void scene_bar_set_active_by_id(scene_bar_t *sb, const scene_t *scenes, int count,
                                const char *scene_id)
{
    if (!sb || !scene_id) {
        scene_bar_set_active(sb, -1);
        return;
    }
    for (int i = 0; i < count && i < sb->scene_count; i++) {
        if (strcmp(scenes[i].id, scene_id) == 0) {
            scene_bar_set_active(sb, i);
            return;
        }
    }
    scene_bar_set_active(sb, -1);
}
