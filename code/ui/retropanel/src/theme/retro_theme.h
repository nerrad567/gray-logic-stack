/**
 * Retro theme — LVGL v9 theme with 60s/70s instrument styling.
 */
#ifndef RETRO_THEME_H
#define RETRO_THEME_H

#include "lvgl.h"

/* Apply the retro theme to the given display */
void retro_theme_init(lv_display_t *disp);

/* Get the larger font for section headers */
const lv_font_t *retro_font_heading(void);

/* Get the monospace body font */
const lv_font_t *retro_font_body(void);

/* Get the large nixie-style font for numeric displays (48px) */
const lv_font_t *retro_font_nixie(void);

/* Get the smaller nixie font for secondary displays (28px) */
const lv_font_t *retro_font_nixie_sm(void);

#endif /* RETRO_THEME_H */
