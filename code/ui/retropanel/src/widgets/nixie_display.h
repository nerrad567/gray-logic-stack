/**
 * Nixie display — glowing numeric readout widget.
 *
 * Shows a value (integer or float) in large nixie-style digits
 * with an orange glow shadow on a dark background.
 */
#ifndef NIXIE_DISPLAY_H
#define NIXIE_DISPLAY_H

#include "lvgl.h"

typedef struct {
    lv_obj_t *container;
    lv_obj_t *value_label;
    lv_obj_t *unit_label;
} nixie_display_t;

/**
 * Create a nixie display.
 * @param parent  Parent LVGL object
 * @param unit    Unit suffix (e.g., "°C", "%")
 * @param value   Initial value
 */
nixie_display_t *nixie_display_create(lv_obj_t *parent, const char *unit, float value);

/** Update the displayed value */
void nixie_display_set_value(nixie_display_t *nd, float value);

/** Update with integer value (no decimal) */
void nixie_display_set_int(nixie_display_t *nd, int value);

#endif /* NIXIE_DISPLAY_H */
