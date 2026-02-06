/**
 * Bakelite button — vintage tactile button widget.
 *
 * A styled button that looks like a brown Bakelite toggle switch
 * with 3D shadow effect and press-in animation.
 */
#ifndef BAKELITE_BTN_H
#define BAKELITE_BTN_H

#include "lvgl.h"
#include <stdbool.h>

typedef struct {
    lv_obj_t *btn;
    lv_obj_t *label;
    bool toggled;
} bakelite_btn_t;

/**
 * Create a bakelite toggle button.
 * @param parent   Parent LVGL object
 * @param text     Button label text
 * @param toggled  Initial toggle state
 */
bakelite_btn_t *bakelite_btn_create(lv_obj_t *parent, const char *text, bool toggled);

/** Set toggle state visually */
void bakelite_btn_set_state(bakelite_btn_t *bb, bool toggled);

/** Get current toggle state */
bool bakelite_btn_get_state(const bakelite_btn_t *bb);

/** Get the underlying LVGL button for event registration */
lv_obj_t *bakelite_btn_get_obj(const bakelite_btn_t *bb);

#endif /* BAKELITE_BTN_H */
