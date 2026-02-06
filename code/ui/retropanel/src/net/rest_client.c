/**
 * REST client — libcurl-based HTTP for SDL simulator.
 *
 * All calls are blocking. Boot-time calls run before LVGL renders.
 * Command calls run in the main thread (fast — local network only).
 *
 * When PANEL_HAS_NETWORKING is not defined, all functions return failure
 * so the panel falls back to demo mode.
 */
#include "net/rest_client.h"
#include <string.h>
#include <stdio.h>

#ifdef PANEL_HAS_NETWORKING

#include <curl/curl.h>
#include <cjson/cJSON.h>
#include <stdlib.h>

static panel_config_t config;
static bool initialised = false;

/* ── Curl response buffer ─────────────────────────────────────────── */

typedef struct {
    char *data;
    size_t size;
} response_buf_t;

static size_t write_cb(void *ptr, size_t size, size_t nmemb, void *userdata)
{
    size_t total = size * nmemb;
    response_buf_t *buf = (response_buf_t *)userdata;
    char *tmp = realloc(buf->data, buf->size + total + 1);
    if (!tmp) return 0;
    buf->data = tmp;
    memcpy(buf->data + buf->size, ptr, total);
    buf->size += total;
    buf->data[buf->size] = '\0';
    return total;
}

/* ── HTTP helpers ─────────────────────────────────────────────────── */

static cJSON *do_get(const char *path)
{
    if (!initialised) return NULL;

    char url[512];
    snprintf(url, sizeof(url), "%s%s", config.server_url, path);

    CURL *curl = curl_easy_init();
    if (!curl) return NULL;

    response_buf_t resp = {NULL, 0};
    struct curl_slist *headers = NULL;
    char auth_hdr[256];
    snprintf(auth_hdr, sizeof(auth_hdr), "X-Panel-Token: %s", config.panel_token);
    headers = curl_slist_append(headers, auth_hdr);
    headers = curl_slist_append(headers, "Accept: application/json");

    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, write_cb);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &resp);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 5L);

    CURLcode res = curl_easy_perform(curl);
    long http_code = 0;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
    curl_slist_free_all(headers);
    curl_easy_cleanup(curl);

    if (res != CURLE_OK || http_code < 200 || http_code >= 300) {
        printf("[rest] GET %s → %s (HTTP %ld)\n", path,
               curl_easy_strerror(res), http_code);
        free(resp.data);
        return NULL;
    }

    cJSON *json = cJSON_Parse(resp.data);
    free(resp.data);
    return json;
}

static bool do_post(const char *path, const char *body)
{
    if (!initialised) return false;

    char url[512];
    snprintf(url, sizeof(url), "%s%s", config.server_url, path);

    CURL *curl = curl_easy_init();
    if (!curl) return false;

    response_buf_t resp = {NULL, 0};
    struct curl_slist *headers = NULL;
    char auth_hdr[256];
    snprintf(auth_hdr, sizeof(auth_hdr), "X-Panel-Token: %s", config.panel_token);
    headers = curl_slist_append(headers, auth_hdr);
    headers = curl_slist_append(headers, "Content-Type: application/json");

    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body ? body : "{}");
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, write_cb);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &resp);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);

    CURLcode res = curl_easy_perform(curl);
    long http_code = 0;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
    curl_slist_free_all(headers);
    curl_easy_cleanup(curl);
    free(resp.data);

    if (res != CURLE_OK) {
        printf("[rest] POST %s → %s\n", path, curl_easy_strerror(res));
        return false;
    }
    return http_code >= 200 && http_code < 300;
}

/* ── Public API ───────────────────────────────────────────────────── */

void rest_client_init(const panel_config_t *cfg)
{
    if (!cfg) return;
    memcpy(&config, cfg, sizeof(config));
    curl_global_init(CURL_GLOBAL_DEFAULT);
    initialised = true;
    printf("[rest] initialised → %s\n", config.server_url);
}

void rest_client_cleanup(void)
{
    if (initialised) {
        curl_global_cleanup();
        initialised = false;
    }
}

static void parse_device_state(cJSON *state, device_t *dev)
{
    cJSON *on = cJSON_GetObjectItem(state, "on");
    if (cJSON_IsBool(on)) dev->on = cJSON_IsTrue(on);

    cJSON *level = cJSON_GetObjectItem(state, "level");
    if (cJSON_IsNumber(level)) dev->level = (uint8_t)level->valueint;

    cJSON *pos = cJSON_GetObjectItem(state, "position");
    if (cJSON_IsNumber(pos)) dev->position = (uint8_t)pos->valueint;

    cJSON *tilt = cJSON_GetObjectItem(state, "tilt");
    if (cJSON_IsNumber(tilt)) dev->tilt = (uint8_t)tilt->valueint;

    cJSON *temp = cJSON_GetObjectItem(state, "temperature");
    if (cJSON_IsNumber(temp)) dev->temperature = (float)temp->valuedouble;

    cJSON *sp = cJSON_GetObjectItem(state, "setpoint");
    if (cJSON_IsNumber(sp)) dev->setpoint = (float)sp->valuedouble;
}

static device_domain_t parse_domain(const char *s)
{
    if (!s) return DOMAIN_OTHER;
    if (strcmp(s, "lighting") == 0) return DOMAIN_LIGHTING;
    if (strcmp(s, "climate") == 0)  return DOMAIN_CLIMATE;
    if (strcmp(s, "blinds") == 0)   return DOMAIN_BLINDS;
    if (strcmp(s, "audio") == 0)    return DOMAIN_AUDIO;
    return DOMAIN_OTHER;
}

static device_capability_t parse_capability(const char *s)
{
    if (!s) return CAP_ON_OFF;
    if (strcmp(s, "on_off") == 0)           return CAP_ON_OFF;
    if (strcmp(s, "dim") == 0)              return CAP_DIM;
    if (strcmp(s, "position") == 0)         return CAP_POSITION;
    if (strcmp(s, "tilt") == 0)             return CAP_TILT;
    if (strcmp(s, "temperature_set") == 0)  return CAP_TEMPERATURE_SET;
    if (strcmp(s, "temperature_read") == 0) return CAP_TEMPERATURE_READ;
    return CAP_ON_OFF;
}

static health_status_t parse_health(const char *s)
{
    if (!s) return HEALTH_UNKNOWN;
    if (strcmp(s, "online") == 0)   return HEALTH_ONLINE;
    if (strcmp(s, "offline") == 0)  return HEALTH_OFFLINE;
    if (strcmp(s, "degraded") == 0) return HEALTH_DEGRADED;
    return HEALTH_UNKNOWN;
}

int rest_load_rooms(room_t *rooms, int max_rooms)
{
    cJSON *json = do_get("/api/v1/hierarchy");
    if (!json) return 0;

    int count = 0;
    cJSON *site = cJSON_GetObjectItem(json, "site");
    cJSON *areas = site ? cJSON_GetObjectItem(site, "areas") : NULL;

    cJSON *area;
    cJSON_ArrayForEach(area, areas) {
        cJSON *area_rooms = cJSON_GetObjectItem(area, "rooms");
        cJSON *room;
        cJSON_ArrayForEach(room, area_rooms) {
            if (count >= max_rooms) break;
            room_t *r = &rooms[count];
            memset(r, 0, sizeof(*r));

            cJSON *id = cJSON_GetObjectItem(room, "id");
            cJSON *name = cJSON_GetObjectItem(room, "name");
            cJSON *sort = cJSON_GetObjectItem(room, "sort_order");
            cJSON *dc = cJSON_GetObjectItem(room, "device_count");
            cJSON *sc = cJSON_GetObjectItem(room, "scene_count");

            if (cJSON_IsString(id))   snprintf(r->id, sizeof(r->id), "%s", id->valuestring);
            if (cJSON_IsString(name)) snprintf(r->name, sizeof(r->name), "%s", name->valuestring);
            if (cJSON_IsNumber(sort)) r->sort_order = sort->valueint;
            if (cJSON_IsNumber(dc))   r->device_count = dc->valueint;
            if (cJSON_IsNumber(sc))   r->scene_count = sc->valueint;
            count++;
        }
    }

    cJSON_Delete(json);
    printf("[rest] loaded %d rooms from hierarchy\n", count);
    return count;
}

int rest_load_devices(const char *room_id, device_t *devices, int max_devices)
{
    char path[256];
    snprintf(path, sizeof(path), "/api/v1/devices?room_id=%s", room_id);

    cJSON *json = do_get(path);
    if (!json) return 0;

    int count = 0;
    cJSON *data = cJSON_GetObjectItem(json, "data");

    cJSON *item;
    cJSON_ArrayForEach(item, data) {
        if (count >= max_devices) break;
        device_t *d = &devices[count];
        memset(d, 0, sizeof(*d));

        cJSON *id = cJSON_GetObjectItem(item, "id");
        cJSON *name = cJSON_GetObjectItem(item, "name");
        cJSON *rid = cJSON_GetObjectItem(item, "room_id");
        cJSON *domain = cJSON_GetObjectItem(item, "domain");
        cJSON *health = cJSON_GetObjectItem(item, "health_status");

        if (cJSON_IsString(id))     snprintf(d->id, sizeof(d->id), "%s", id->valuestring);
        if (cJSON_IsString(name))   snprintf(d->name, sizeof(d->name), "%s", name->valuestring);
        if (cJSON_IsString(rid))    snprintf(d->room_id, sizeof(d->room_id), "%s", rid->valuestring);
        if (cJSON_IsString(domain)) d->domain = parse_domain(domain->valuestring);
        if (cJSON_IsString(health)) d->health = parse_health(health->valuestring);

        /* Capabilities */
        cJSON *caps = cJSON_GetObjectItem(item, "capabilities");
        cJSON *cap;
        d->cap_count = 0;
        cJSON_ArrayForEach(cap, caps) {
            if (d->cap_count >= MAX_CAPABILITIES) break;
            if (cJSON_IsString(cap)) {
                d->capabilities[d->cap_count++] = parse_capability(cap->valuestring);
            }
        }

        /* State */
        cJSON *state = cJSON_GetObjectItem(item, "state");
        if (state) parse_device_state(state, d);

        count++;
    }

    cJSON_Delete(json);
    printf("[rest] loaded %d devices for room %s\n", count, room_id);
    return count;
}

int rest_load_scenes(const char *room_id, scene_t *scenes, int max_scenes,
                     char *active_scene_id, int active_id_size)
{
    char path[256];
    snprintf(path, sizeof(path), "/api/v1/scenes?room_id=%s", room_id);

    cJSON *json = do_get(path);
    if (!json) return 0;

    int count = 0;
    cJSON *scene_arr = cJSON_GetObjectItem(json, "scenes");

    cJSON *item;
    cJSON_ArrayForEach(item, scene_arr) {
        if (count >= max_scenes) break;
        scene_t *s = &scenes[count];
        memset(s, 0, sizeof(*s));

        cJSON *id = cJSON_GetObjectItem(item, "id");
        cJSON *name = cJSON_GetObjectItem(item, "name");
        cJSON *rid = cJSON_GetObjectItem(item, "room_id");
        cJSON *colour = cJSON_GetObjectItem(item, "colour");
        cJSON *icon = cJSON_GetObjectItem(item, "icon");
        cJSON *enabled = cJSON_GetObjectItem(item, "enabled");
        cJSON *sort = cJSON_GetObjectItem(item, "sort_order");

        if (cJSON_IsString(id))     snprintf(s->id, sizeof(s->id), "%s", id->valuestring);
        if (cJSON_IsString(name))   snprintf(s->name, sizeof(s->name), "%s", name->valuestring);
        if (cJSON_IsString(rid))    snprintf(s->room_id, sizeof(s->room_id), "%s", rid->valuestring);
        if (cJSON_IsString(colour)) snprintf(s->colour, sizeof(s->colour), "%s", colour->valuestring);
        if (cJSON_IsString(icon))   snprintf(s->icon, sizeof(s->icon), "%s", icon->valuestring);
        if (cJSON_IsBool(enabled))  s->enabled = cJSON_IsTrue(enabled);
        if (cJSON_IsNumber(sort))   s->sort_order = sort->valueint;
        count++;
    }

    /* Extract active scene for this room */
    if (active_scene_id) {
        active_scene_id[0] = '\0';
        cJSON *active_map = cJSON_GetObjectItem(json, "active_scenes");
        if (active_map) {
            cJSON *active = cJSON_GetObjectItem(active_map, room_id);
            if (cJSON_IsString(active)) {
                snprintf(active_scene_id, active_id_size, "%s", active->valuestring);
            }
        }
    }

    cJSON_Delete(json);
    printf("[rest] loaded %d scenes for room %s (active: %s)\n",
           count, room_id, active_scene_id && active_scene_id[0] ? active_scene_id : "none");
    return count;
}

bool rest_send_command(const char *device_id, const char *command,
                       const char *param_json)
{
    char path[256];
    snprintf(path, sizeof(path), "/api/v1/devices/%s/state", device_id);

    char body[512];
    snprintf(body, sizeof(body), "{\"command\":\"%s\",\"parameters\":%s}",
             command, param_json ? param_json : "{}");

    return do_post(path, body);
}

bool rest_activate_scene(const char *scene_id)
{
    char path[256];
    snprintf(path, sizeof(path), "/api/v1/scenes/%s/activate", scene_id);
    return do_post(path, "{\"trigger_type\":\"manual\",\"trigger_source\":\"panel\"}");
}

#else /* !PANEL_HAS_NETWORKING */

void rest_client_init(const panel_config_t *cfg) { (void)cfg; }
void rest_client_cleanup(void) {}
int rest_load_rooms(room_t *r, int m) { (void)r; (void)m; return 0; }
int rest_load_devices(const char *rid, device_t *d, int m) { (void)rid; (void)d; (void)m; return 0; }
int rest_load_scenes(const char *rid, scene_t *s, int m, char *a, int as)
    { (void)rid; (void)s; (void)m; (void)a; (void)as; return 0; }
bool rest_send_command(const char *d, const char *c, const char *p)
    { (void)d; (void)c; (void)p; return false; }
bool rest_activate_scene(const char *s) { (void)s; return false; }

#endif /* PANEL_HAS_NETWORKING */
