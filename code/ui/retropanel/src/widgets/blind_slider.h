/**
 * Blind slider — vertical slider for blind/shade position control.
 *
 * Retro-styled vertical slider with brown track, amber fill,
 * and a bakelite-style knob.
 */
#ifndef BLIND_SLIDER_H
#define BLIND_SLIDER_H

#include "lvgl.h"

typedef struct {
    lv_obj_t *container;
    lv_obj_t *slider;
    lv_obj_t *value_label;
    lv_obj_t *name_label;
} blind_slider_t;

/**
 * Create a blind slider.
 * @param parent  Parent LVGL object
 * @param name    Device name
 * @param value   Initial position 0-100
 */
blind_slider_t *blind_slider_create(lv_obj_t *parent, const char *name, int value);

/** Set position (0-100) */
void blind_slider_set_value(blind_slider_t *bs, int value);

/** Get position */
int blind_slider_get_value(const blind_slider_t *bs);

#endif /* BLIND_SLIDER_H */
