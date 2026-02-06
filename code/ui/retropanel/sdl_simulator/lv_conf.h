/**
 * LVGL configuration for SDL simulator build.
 * 32-bit color depth, logging enabled, SDL display/input drivers.
 */
#ifndef LV_CONF_H
#define LV_CONF_H

/* Core */
#define LV_COLOR_DEPTH          32
#define LV_DPI_DEF              130

/* Memory — use stdlib malloc for simulator (no pool) */
#define LV_MEM_CUSTOM           1
#define LV_MEM_CUSTOM_INCLUDE   <stdlib.h>
#define LV_MEM_CUSTOM_ALLOC     malloc
#define LV_MEM_CUSTOM_FREE      free
#define LV_MEM_CUSTOM_REALLOC   realloc

/* Display */
#define LV_USE_SDL              1
#define LV_SDL_WINDOW_TITLE     "Gray Logic — Retro Panel"
#define LV_SDL_INCLUDE_PATH     <SDL2/SDL.h>

/* Rendering */
#define LV_USE_DRAW_SW          1
#define LV_DRAW_SW_SHADOW       1

/* Logging */
#define LV_USE_LOG              1
#define LV_LOG_LEVEL            LV_LOG_LEVEL_WARN
#define LV_LOG_PRINTF           1

/* Fonts — built-in defaults, we add custom fonts via C arrays */
#define LV_FONT_MONTSERRAT_14   1
#define LV_FONT_MONTSERRAT_18   1
#define LV_FONT_MONTSERRAT_24   1
#define LV_FONT_MONTSERRAT_48   0
#define LV_FONT_DEFAULT         &lv_font_montserrat_14

/* Widget features */
#define LV_USE_ARC              1
#define LV_USE_BAR              1
#define LV_USE_BTN              1
#define LV_USE_LABEL            1
#define LV_USE_SLIDER           1
#define LV_USE_SWITCH           1
#define LV_USE_TEXTAREA         0
#define LV_USE_TABLE            0
#define LV_USE_CHART            0
#define LV_USE_ROLLER           0
#define LV_USE_DROPDOWN         0
#define LV_USE_CHECKBOX         0
#define LV_USE_BTNMATRIX        0
#define LV_USE_KEYBOARD         0
#define LV_USE_LIST             0
#define LV_USE_MSGBOX           0
#define LV_USE_SPINBOX          0
#define LV_USE_SPINNER          0
#define LV_USE_TABVIEW          0
#define LV_USE_TILEVIEW         0
#define LV_USE_WIN              0
#define LV_USE_SPAN             0
#define LV_USE_IMGBTN           0
#define LV_USE_LED              0
#define LV_USE_ANIMIMG          0
#define LV_USE_CALENDAR         0
#define LV_USE_MENU             0
#define LV_USE_SCALE            0

/* Layouts */
#define LV_USE_FLEX             1
#define LV_USE_GRID             1

/* Animation */
#define LV_USE_ANIM             1

/* Misc */
#define LV_USE_MSG              1
#define LV_USE_OBSERVER         1
#define LV_USE_OBJ_ID           0
#define LV_USE_OBJ_PROPERTY     0

#endif /* LV_CONF_H */
