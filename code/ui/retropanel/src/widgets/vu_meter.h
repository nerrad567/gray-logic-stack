/**
 * VU meter — arc gauge widget for dimmer level control.
 *
 * Renders as a semi-circular arc with tick marks and a value label
 * in nixie font. Wraps lv_arc with retro styling.
 */
#ifndef VU_METER_H
#define VU_METER_H

#include "lvgl.h"

typedef struct {
    lv_obj_t *container;
    lv_obj_t *arc;
    lv_obj_t *value_label;
    lv_obj_t *name_label;
} vu_meter_t;

/**
 * Create a VU meter control.
 * @param parent     Parent LVGL object
 * @param name       Device name shown below the arc
 * @param value      Initial value 0-100
 * @return           VU meter handle
 */
vu_meter_t *vu_meter_create(lv_obj_t *parent, const char *name, int value);

/** Set the value (0-100), updates arc and label */
void vu_meter_set_value(vu_meter_t *vm, int value);

/** Get the current value */
int vu_meter_get_value(const vu_meter_t *vm);

#endif /* VU_METER_H */
