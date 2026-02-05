# Codex Task: Phase 4 — State History REST Endpoints

## Summary

Add REST endpoints for querying device state history from SQLite (Phase 3 audit trail) and time-series metrics from VictoriaMetrics (PromQL proxy).

## Prerequisites

- Phase 3 (SQLite State Audit Trail) must be complete
- VictoriaMetrics running on port 8428 (optional — endpoints degrade gracefully)

## Context

The state pipeline writes to three stores:
1. **Device Registry** (in-memory + SQLite) — current state
2. **VictoriaMetrics** — numeric time-series for analytics
3. **State History** (SQLite) — full state snapshots for audit trail

This task exposes #2 and #3 via REST endpoints.

## Requirements

### 1. State History Endpoint (SQLite)

**`GET /api/v1/devices/{id}/history`**

Returns recent state change history for a device from the SQLite audit trail.

Query parameters:
- `limit` — max entries to return (default: 50, max: 200)
- `since` — ISO 8601 timestamp, only return entries after this time

Response:
```json
{
  "device_id": "light-living-01",
  "history": [
    {
      "id": 1234,
      "state": {"on": true, "level": 75},
      "source": "mqtt",
      "created_at": "2026-02-05T10:30:45Z"
    },
    {
      "id": 1233,
      "state": {"on": false, "level": 0},
      "source": "command",
      "created_at": "2026-02-05T10:29:12Z"
    }
  ],
  "count": 2
}
```

Error cases:
- 404 if device_id not found in registry
- 400 if limit > 200 or invalid `since` format

### 2. Time-Series Metrics Endpoint (VictoriaMetrics proxy)

**`GET /api/v1/devices/{id}/metrics`**

Proxies a PromQL range query to VictoriaMetrics for the given device.

Query parameters:
- `field` — metric field name (required, e.g., "level", "on", "temperature")
- `start` — ISO 8601 or Unix timestamp (default: 1 hour ago)
- `end` — ISO 8601 or Unix timestamp (default: now)
- `step` — query resolution (default: "1m", format: Prometheus duration)

The server builds a PromQL query internally:
```
device_metrics{device_id="light-living-01", measurement="level"}
```

Then proxies to VictoriaMetrics:
```
GET http://localhost:8428/api/v1/query_range?query=...&start=...&end=...&step=...
```

Response: Pass through VictoriaMetrics JSON response directly (Prometheus API format):
```json
{
  "status": "success",
  "data": {
    "resultType": "matrix",
    "result": [
      {
        "metric": {"device_id": "light-living-01", "measurement": "level"},
        "values": [
          [1738750245, "75"],
          [1738750305, "50"],
          [1738750365, "75"]
        ]
      }
    ]
  }
}
```

Error cases:
- 400 if `field` is missing
- 503 if VictoriaMetrics is not available (TSDB disabled or unreachable)
- 404 if device_id not found in registry

### 3. Device Metrics Summary Endpoint

**`GET /api/v1/devices/{id}/metrics/summary`**

Returns available metric fields and their latest values for a device. This helps the UI know which fields can be graphed.

Queries VictoriaMetrics:
```
GET http://localhost:8428/api/v1/query?query=device_metrics{device_id="light-living-01"}
```

Response:
```json
{
  "device_id": "light-living-01",
  "fields": ["on", "level"],
  "latest": {
    "on": {"value": 1, "timestamp": "2026-02-05T10:30:45Z"},
    "level": {"value": 75, "timestamp": "2026-02-05T10:30:45Z"}
  }
}
```

Error cases:
- 503 if TSDB not available
- 404 if device not found

## Implementation

### File: `internal/api/state_history.go` (NEW, ~120 lines)

Handler functions:
- `handleGetDeviceHistory(w, r)` — parse params, call `stateHistory.GetHistory()`, return JSON
- `handleGetDeviceMetrics(w, r)` — build PromQL, proxy to VictoriaMetrics, return response
- `handleGetDeviceMetricsSummary(w, r)` — instant query to VictoriaMetrics

### File: `internal/api/router.go` (MODIFY)

Add routes under the existing device routes:
```go
r.Get("/devices/{id}/history", s.handleGetDeviceHistory)
r.Get("/devices/{id}/metrics", s.handleGetDeviceMetrics)
r.Get("/devices/{id}/metrics/summary", s.handleGetDeviceMetricsSummary)
```

### File: `internal/infrastructure/tsdb/query.go` (NEW, ~80 lines)

Add query methods to the TSDB client:

```go
// QueryRange executes a PromQL range query.
func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (json.RawMessage, error)

// QueryInstant executes a PromQL instant query.
func (c *Client) QueryInstant(ctx context.Context, query string) (json.RawMessage, error)
```

These are thin HTTP wrappers:
- `QueryRange`: `GET {url}/api/v1/query_range?query={q}&start={unix}&end={unix}&step={dur}`
- `QueryInstant`: `GET {url}/api/v1/query?query={q}`

Return `json.RawMessage` to avoid deserializing/re-serializing the Prometheus response format.

### File: `internal/infrastructure/tsdb/query_test.go` (NEW)

- `TestQueryRange` — mock HTTP server returning Prometheus-format JSON
- `TestQueryInstant` — mock HTTP server
- Test error cases (timeout, 503, invalid query)

### File: `internal/api/state_history_test.go` (NEW, ~200 lines)

- `TestHandleGetDeviceHistory` — mock state history repo, verify JSON response
- `TestHandleGetDeviceMetrics` — mock TSDB client, verify PromQL query construction
- `TestHandleGetDeviceMetrics_TSDBDisabled` — verify 503 when tsdb is nil
- `TestHandleGetDeviceHistory_InvalidParams` — verify 400 on bad input

## Files to Create/Modify

| File | Action |
|------|--------|
| `internal/api/state_history.go` | CREATE (handlers) |
| `internal/api/state_history_test.go` | CREATE (tests) |
| `internal/api/router.go` | MODIFY (add 3 routes) |
| `internal/infrastructure/tsdb/query.go` | CREATE (PromQL query methods) |
| `internal/infrastructure/tsdb/query_test.go` | CREATE (tests) |

## Patterns to Follow

- Look at `internal/api/devices.go` for handler patterns (Chi router, URL params, JSON responses)
- Look at `internal/api/scenes.go` for list endpoints with query params
- Look at `internal/infrastructure/tsdb/client.go` for HTTP client patterns
- Use `writeJSON()`, `writeBadRequest()`, `writeNotFound()` helpers from `internal/api/helpers.go`
- Extract `{id}` from URL with `chi.URLParam(r, "id")`
- Validate device exists via `s.registry.GetDevice(ctx, id)` before querying

## Verification

```bash
cd code/core

# Build
go build ./...

# Tests
make test-race

# Manual verification (with dev services running):

# 1. Start the stack
make dev-services && make dev-run

# 2. Trigger some state changes via KNXSim

# 3. Query state history
curl http://localhost:8090/api/v1/devices/{id}/history

# 4. Query metrics (if TSDB enabled in config)
curl "http://localhost:8090/api/v1/devices/{id}/metrics?field=level&start=$(date -d '1 hour ago' +%s)&end=$(date +%s)"

# 5. Query metrics summary
curl http://localhost:8090/api/v1/devices/{id}/metrics/summary
```

## Constraints

- No external dependencies for query.go — use `net/http` + `encoding/json`
- PromQL queries must be safely escaped (no injection via device_id or field params)
- Metrics endpoints return 503 (not 500) when TSDB is nil/disabled
- State history endpoint works independently of TSDB
- All handlers are safe for concurrent use
- Follow existing code style (check `golangci-lint run`)
