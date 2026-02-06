/**
 * SDL simulator entry point for Retro Panel.
 *
 * Initialises LVGL with SDL2 display (480x320) and input drivers,
 * then hands off to app_init() which builds the UI.
 */
#include "lvgl.h"
#include "app.h"

#define WINDOW_WIDTH  480
#define WINDOW_HEIGHT 320

int main(void)
{
    lv_init();

    /* Create an SDL display — LVGL v9 handles SDL window creation internally */
    lv_display_t *disp = lv_sdl_window_create(WINDOW_WIDTH, WINDOW_HEIGHT);
    lv_sdl_window_set_title(disp, "Gray Logic - Retro Panel");

    /* Create an SDL mouse input device */
    lv_indev_t *mouse = lv_sdl_mouse_create();
    (void)mouse;

    /* Initialise the retro panel application */
    app_init();

    /* Main loop — LVGL handles timing, app_tick drains MQTT updates */
    while (1) {
        uint32_t idle_ms = lv_timer_handler();
        app_tick();
        if (idle_ms > 0) {
            lv_delay_ms(idle_ms < 5 ? idle_ms : 5);
        }
    }

    return 0;
}
