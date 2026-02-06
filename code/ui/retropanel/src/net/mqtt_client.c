/**
 * MQTT client — libmosquitto for SDL simulator.
 *
 * Runs mosquitto's network loop on a background thread.
 * State updates are queued in a ring buffer and drained by the LVGL thread.
 *
 * When PANEL_HAS_NETWORKING is not defined, all functions are no-ops.
 */
#include "net/mqtt_client.h"
#include <string.h>
#include <stdio.h>

#ifdef PANEL_HAS_NETWORKING

#include <mosquitto.h>
#include <cjson/cJSON.h>
#include <pthread.h>

#define STATE_QUEUE_SIZE 64
#define SCENE_QUEUE_SIZE 16

/* ── Ring buffers ─────────────────────────────────────────────────── */

static mqtt_state_update_t state_queue[STATE_QUEUE_SIZE];
static int state_head = 0;
static int state_tail = 0;
static pthread_mutex_t state_mutex = PTHREAD_MUTEX_INITIALIZER;

static mqtt_scene_event_t scene_queue[SCENE_QUEUE_SIZE];
static int scene_head = 0;
static int scene_tail = 0;
static pthread_mutex_t scene_mutex = PTHREAD_MUTEX_INITIALIZER;

static struct mosquitto *mosq = NULL;
static bool connected = false;

/* ── Queue helpers ────────────────────────────────────────────────── */

static void enqueue_state(const mqtt_state_update_t *update)
{
    pthread_mutex_lock(&state_mutex);
    state_queue[state_head] = *update;
    state_head = (state_head + 1) % STATE_QUEUE_SIZE;
    if (state_head == state_tail) {
        state_tail = (state_tail + 1) % STATE_QUEUE_SIZE; /* drop oldest */
    }
    pthread_mutex_unlock(&state_mutex);
}

static bool dequeue_state(mqtt_state_update_t *update)
{
    pthread_mutex_lock(&state_mutex);
    if (state_head == state_tail) {
        pthread_mutex_unlock(&state_mutex);
        return false;
    }
    *update = state_queue[state_tail];
    state_tail = (state_tail + 1) % STATE_QUEUE_SIZE;
    pthread_mutex_unlock(&state_mutex);
    return true;
}

static void enqueue_scene(const mqtt_scene_event_t *event)
{
    pthread_mutex_lock(&scene_mutex);
    scene_queue[scene_head] = *event;
    scene_head = (scene_head + 1) % SCENE_QUEUE_SIZE;
    if (scene_head == scene_tail) {
        scene_tail = (scene_tail + 1) % SCENE_QUEUE_SIZE;
    }
    pthread_mutex_unlock(&scene_mutex);
}

static bool dequeue_scene(mqtt_scene_event_t *event)
{
    pthread_mutex_lock(&scene_mutex);
    if (scene_head == scene_tail) {
        pthread_mutex_unlock(&scene_mutex);
        return false;
    }
    *event = scene_queue[scene_tail];
    scene_tail = (scene_tail + 1) % SCENE_QUEUE_SIZE;
    pthread_mutex_unlock(&scene_mutex);
    return true;
}

/* ── Topic parsing ────────────────────────────────────────────────── */

/**
 * Extract device_id from topic: graylogic/core/device/{device_id}/state
 */
static bool parse_device_topic(const char *topic, char *device_id, int size)
{
    const char *prefix = "graylogic/core/device/";
    if (strncmp(topic, prefix, strlen(prefix)) != 0) return false;

    const char *start = topic + strlen(prefix);
    const char *end = strstr(start, "/state");
    if (!end) return false;

    int len = (int)(end - start);
    if (len <= 0 || len >= size) return false;
    memcpy(device_id, start, len);
    device_id[len] = '\0';
    return true;
}

/**
 * Extract scene_id from topic: graylogic/core/scene/{scene_id}/activated
 */
static bool parse_scene_topic(const char *topic, char *scene_id, int size)
{
    const char *prefix = "graylogic/core/scene/";
    if (strncmp(topic, prefix, strlen(prefix)) != 0) return false;

    const char *start = topic + strlen(prefix);
    const char *end = strstr(start, "/activated");
    if (!end) return false;

    int len = (int)(end - start);
    if (len <= 0 || len >= size) return false;
    memcpy(scene_id, start, len);
    scene_id[len] = '\0';
    return true;
}

/* ── Mosquitto callbacks ──────────────────────────────────────────── */

static void on_connect(struct mosquitto *m, void *userdata, int rc)
{
    (void)userdata;
    if (rc == 0) {
        connected = true;
        printf("[mqtt] connected\n");
        mosquitto_subscribe(m, NULL, "graylogic/core/device/+/state", 0);
        mosquitto_subscribe(m, NULL, "graylogic/core/scene/+/activated", 0);
    } else {
        printf("[mqtt] connection failed: %s\n", mosquitto_connack_string(rc));
    }
}

static void on_disconnect(struct mosquitto *m, void *userdata, int rc)
{
    (void)m; (void)userdata;
    connected = false;
    printf("[mqtt] disconnected (rc=%d)\n", rc);
}

static void on_message(struct mosquitto *m, void *userdata,
                       const struct mosquitto_message *msg)
{
    (void)m; (void)userdata;
    if (!msg->payload || msg->payloadlen == 0) return;

    char *payload = (char *)msg->payload;
    char device_id[48];
    char scene_id[48];

    /* Device state update */
    if (parse_device_topic(msg->topic, device_id, sizeof(device_id))) {
        cJSON *json = cJSON_ParseWithLength(payload, msg->payloadlen);
        if (!json) return;

        /* The payload might be the state map directly, or wrapped in {"state":{...}} */
        cJSON *state = cJSON_GetObjectItem(json, "state");
        if (!state) state = json;

        mqtt_state_update_t update;
        memset(&update, 0, sizeof(update));
        snprintf(update.device_id, sizeof(update.device_id), "%s", device_id);

        cJSON *on = cJSON_GetObjectItem(state, "on");
        if (cJSON_IsBool(on)) { update.has_on = true; update.on = cJSON_IsTrue(on); }

        cJSON *level = cJSON_GetObjectItem(state, "level");
        if (cJSON_IsNumber(level)) { update.has_level = true; update.level = level->valueint; }

        cJSON *pos = cJSON_GetObjectItem(state, "position");
        if (cJSON_IsNumber(pos)) { update.has_pos = true; update.position = pos->valueint; }

        cJSON *temp = cJSON_GetObjectItem(state, "temperature");
        if (cJSON_IsNumber(temp)) { update.has_temp = true; update.temperature = (float)temp->valuedouble; }

        cJSON *sp = cJSON_GetObjectItem(state, "setpoint");
        if (cJSON_IsNumber(sp)) { update.has_sp = true; update.setpoint = (float)sp->valuedouble; }

        enqueue_state(&update);
        cJSON_Delete(json);
    }

    /* Scene activation */
    else if (parse_scene_topic(msg->topic, scene_id, sizeof(scene_id))) {
        cJSON *json = cJSON_ParseWithLength(payload, msg->payloadlen);
        mqtt_scene_event_t event;
        memset(&event, 0, sizeof(event));
        snprintf(event.scene_id, sizeof(event.scene_id), "%s", scene_id);

        if (json) {
            cJSON *rid = cJSON_GetObjectItem(json, "room_id");
            if (cJSON_IsString(rid))
                snprintf(event.room_id, sizeof(event.room_id), "%s", rid->valuestring);
            cJSON_Delete(json);
        }
        enqueue_scene(&event);
    }
}

/* ── Public API ───────────────────────────────────────────────────── */

bool mqtt_client_init(const panel_config_t *cfg)
{
    if (!cfg) return false;

    mosquitto_lib_init();
    mosq = mosquitto_new("retro-panel", true, NULL);
    if (!mosq) {
        printf("[mqtt] failed to create client\n");
        return false;
    }

    mosquitto_connect_callback_set(mosq, on_connect);
    mosquitto_disconnect_callback_set(mosq, on_disconnect);
    mosquitto_message_callback_set(mosq, on_message);

    int rc = mosquitto_connect(mosq, cfg->mqtt_host, cfg->mqtt_port, 60);
    if (rc != MOSQ_ERR_SUCCESS) {
        printf("[mqtt] connect failed: %s\n", mosquitto_strerror(rc));
        mosquitto_destroy(mosq);
        mosq = NULL;
        return false;
    }

    /* Start the network loop on a background thread */
    rc = mosquitto_loop_start(mosq);
    if (rc != MOSQ_ERR_SUCCESS) {
        printf("[mqtt] loop_start failed: %s\n", mosquitto_strerror(rc));
        mosquitto_disconnect(mosq);
        mosquitto_destroy(mosq);
        mosq = NULL;
        return false;
    }

    printf("[mqtt] client started → %s:%d\n", cfg->mqtt_host, cfg->mqtt_port);
    return true;
}

void mqtt_client_cleanup(void)
{
    if (mosq) {
        mosquitto_loop_stop(mosq, true);
        mosquitto_disconnect(mosq);
        mosquitto_destroy(mosq);
        mosq = NULL;
    }
    mosquitto_lib_cleanup();
    connected = false;
}

int mqtt_client_drain_updates(mqtt_state_cb_t state_cb, mqtt_scene_cb_t scene_cb,
                              void *user_data)
{
    int count = 0;
    mqtt_state_update_t state_update;
    while (dequeue_state(&state_update)) {
        if (state_cb) state_cb(&state_update, user_data);
        count++;
    }

    mqtt_scene_event_t scene_event;
    while (dequeue_scene(&scene_event)) {
        if (scene_cb) scene_cb(&scene_event, user_data);
        count++;
    }
    return count;
}

bool mqtt_client_is_connected(void)
{
    return connected;
}

#else /* !PANEL_HAS_NETWORKING */

bool mqtt_client_init(const panel_config_t *cfg) { (void)cfg; return false; }
void mqtt_client_cleanup(void) {}
int mqtt_client_drain_updates(mqtt_state_cb_t sc, mqtt_scene_cb_t ec, void *ud)
    { (void)sc; (void)ec; (void)ud; return 0; }
bool mqtt_client_is_connected(void) { return false; }

#endif /* PANEL_HAS_NETWORKING */
