/**
 * Scene bar — horizontal row of scene activation buttons.
 *
 * Each button shows a colour dot + scene name. The active scene
 * gets a highlighted border and brighter text.
 */
#ifndef SCENE_BAR_H
#define SCENE_BAR_H

#include "lvgl.h"
#include "data/data_model.h"

typedef struct {
    lv_obj_t *container;
    lv_obj_t *buttons[MAX_SCENES_PER_ROOM];
    int scene_count;
    int active_index;   /* -1 = none active */
} scene_bar_t;

/**
 * Create a scene bar.
 * @param parent         Parent LVGL object
 * @param scenes         Array of scenes
 * @param count          Number of scenes
 * @param active_scene_id  ID of the currently active scene (NULL if none)
 */
scene_bar_t *scene_bar_create(lv_obj_t *parent, const scene_t *scenes, int count,
                              const char *active_scene_id);

/** Set the active scene by index (-1 for none) */
void scene_bar_set_active(scene_bar_t *sb, int index);

/** Set the active scene by ID */
void scene_bar_set_active_by_id(scene_bar_t *sb, const scene_t *scenes, int count,
                                const char *scene_id);

#endif /* SCENE_BAR_H */
