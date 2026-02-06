/**
 * Retro panel colour palette — 1960s/70s instrument aesthetic.
 *
 * Amber-on-charcoal with warm browns and olive accents.
 * Designed for high contrast on small displays.
 */
#ifndef RETRO_COLORS_H
#define RETRO_COLORS_H

#include "lvgl.h"

/* Primary palette */
#define RETRO_AMBER         lv_color_hex(0xF5A623)
#define RETRO_AMBER_DIM     lv_color_hex(0xAA7418)
#define RETRO_AMBER_BRIGHT  lv_color_hex(0xFFBF47)
#define RETRO_CREAM         lv_color_hex(0xFFF8E7)
#define RETRO_OLIVE          lv_color_hex(0x6B7B3A)
#define RETRO_BURNT_ORANGE  lv_color_hex(0xCC5500)

/* Neutrals */
#define RETRO_DARK_BROWN    lv_color_hex(0x3E2723)
#define RETRO_MED_BROWN     lv_color_hex(0x5D4037)
#define RETRO_CHARCOAL      lv_color_hex(0x37474F)
#define RETRO_DARK_BG       lv_color_hex(0x263238)
#define RETRO_NEAR_BLACK    lv_color_hex(0x1A1A1A)

/* Special effects */
#define RETRO_NIXIE_GLOW    lv_color_hex(0xFF8C00)
#define RETRO_NIXIE_BG      lv_color_hex(0x1C1008)
#define RETRO_GLOW_SHADOW   lv_color_hex(0xFF6600)
#define RETRO_ERROR_RED     lv_color_hex(0xCC3333)

/* Status */
#define RETRO_ONLINE_GREEN  lv_color_hex(0x6B8E23)
#define RETRO_OFFLINE_RED   lv_color_hex(0x8B0000)

/* Opacity presets */
#define RETRO_OPA_SCANLINE  (LV_OPA_10 + 12)   /* ~15 — subtle CRT scan lines */
#define RETRO_OPA_INACTIVE  LV_OPA_50
#define RETRO_OPA_HOVER     LV_OPA_80

#endif /* RETRO_COLORS_H */
