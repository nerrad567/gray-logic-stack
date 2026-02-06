/**
 * App — top-level application state and lifecycle.
 */
#ifndef APP_H
#define APP_H

/** Initialise the retro panel application (call after LVGL + display init) */
void app_init(void);

/**
 * App tick — call from the main loop after lv_timer_handler().
 * Drains MQTT updates and refreshes widgets.
 */
void app_tick(void);

/** Clean up networking resources (call before exit, if reachable). */
void app_cleanup(void);

#endif /* APP_H */
