/**
 * Scanline overlay — CRT-style horizontal scan lines.
 *
 * Creates a transparent overlay that draws semi-transparent lines
 * every 2 pixels to simulate a vintage CRT display.
 */
#ifndef SCANLINE_OVERLAY_H
#define SCANLINE_OVERLAY_H

#include "lvgl.h"

/** Create the scanline overlay on the given screen */
lv_obj_t *scanline_overlay_create(lv_obj_t *screen);

#endif /* SCANLINE_OVERLAY_H */
