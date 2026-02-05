package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/nerrad567/gray-logic-core/internal/device"
)

const (
	defaultHistoryLimit   = 50
	maxHistoryLimit       = 200
	defaultMetricsRange   = time.Hour
	defaultMetricsStep    = time.Minute
	serviceUnavailableKey = "service_unavailable"
)

// metricsLatestEntry represents the latest value for a metrics field.
type metricsLatestEntry struct {
	Value     float64 `json:"value"`
	Timestamp string  `json:"timestamp"`
}

// promVectorResponse models the Prometheus instant query response payload.
type promVectorResponse struct {
	Status string          `json:"status"`
	Data   promVectorData  `json:"data"`
	Error  string          `json:"error,omitempty"`
	Type   json.RawMessage `json:"errorType,omitempty"`
}

type promVectorData struct {
	ResultType string             `json:"resultType"`
	Result     []promVectorResult `json:"result"`
}

type promVectorResult struct {
	Metric map[string]string `json:"metric"`
	Value  []any             `json:"value"`
}

// handleGetDeviceHistory returns state history entries for a device.
func (s *Server) handleGetDeviceHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" || len(deviceID) > maxQueryParamLen {
		writeBadRequest(w, "invalid device ID")
		return
	}

	limit, err := parseHistoryLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	since, err := parseSinceParam(r.URL.Query().Get("since"))
	if err != nil {
		writeBadRequest(w, "invalid since timestamp")
		return
	}

	if _, err := s.registry.GetDevice(ctx, deviceID); err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			writeNotFound(w, "device not found")
			return
		}
		writeInternalError(w, "failed to get device")
		return
	}

	if s.stateHistory == nil {
		writeError(w, http.StatusServiceUnavailable, serviceUnavailableKey, "state history unavailable")
		return
	}

	entries, err := s.stateHistory.GetHistory(ctx, deviceID, limit)
	if err != nil {
		writeInternalError(w, "failed to load device history")
		return
	}

	if !since.IsZero() {
		filtered := entries[:0]
		for _, entry := range entries {
			if entry.CreatedAt.After(since) {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"device_id": deviceID,
		"history":   entries,
		"count":     len(entries),
	})
}

// handleGetDeviceMetrics proxies a PromQL range query for a device field.
func (s *Server) handleGetDeviceMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" || len(deviceID) > maxQueryParamLen {
		writeBadRequest(w, "invalid device ID")
		return
	}

	field := strings.TrimSpace(r.URL.Query().Get("field"))
	if field == "" {
		writeBadRequest(w, "field is required")
		return
	}
	if len(field) > maxQueryParamLen {
		writeBadRequest(w, "field exceeds maximum length")
		return
	}

	start, end, step, err := parseMetricsRangeParams(r)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	if _, err := s.registry.GetDevice(ctx, deviceID); err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			writeNotFound(w, "device not found")
			return
		}
		writeInternalError(w, "failed to get device")
		return
	}

	if s.tsdb == nil || !s.tsdb.IsConnected() {
		writeError(w, http.StatusServiceUnavailable, serviceUnavailableKey, "time-series database unavailable")
		return
	}

	query, err := buildMetricsQuery(deviceID, field)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	resp, err := s.tsdb.QueryRange(ctx, query, start, end, step)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, serviceUnavailableKey, "time-series database unavailable")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleGetDeviceMetricsSummary returns available metric fields and latest values.
func (s *Server) handleGetDeviceMetricsSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" || len(deviceID) > maxQueryParamLen {
		writeBadRequest(w, "invalid device ID")
		return
	}

	if _, err := s.registry.GetDevice(ctx, deviceID); err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			writeNotFound(w, "device not found")
			return
		}
		writeInternalError(w, "failed to get device")
		return
	}

	if s.tsdb == nil || !s.tsdb.IsConnected() {
		writeError(w, http.StatusServiceUnavailable, serviceUnavailableKey, "time-series database unavailable")
		return
	}

	query, err := buildMetricsQuery(deviceID, "")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	resp, err := s.tsdb.QueryInstant(ctx, query)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, serviceUnavailableKey, "time-series database unavailable")
		return
	}

	latest, fields, err := parseMetricsSummary(resp)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, serviceUnavailableKey, "time-series database unavailable")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"device_id": deviceID,
		"fields":    fields,
		"latest":    latest,
	})
}

// parseHistoryLimit parses the limit query parameter with bounds enforcement.
func parseHistoryLimit(raw string) (int, error) {
	if raw == "" {
		return defaultHistoryLimit, nil
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("invalid limit")
	}
	if limit > maxHistoryLimit {
		return 0, fmt.Errorf("limit exceeds maximum")
	}

	return limit, nil
}

// parseSinceParam parses the since parameter as RFC3339/RFC3339Nano.
func parseSinceParam(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}

	return parseRFC3339(raw)
}

// parseMetricsRangeParams parses start, end, and step parameters with defaults.
func parseMetricsRangeParams(r *http.Request) (time.Time, time.Time, time.Duration, error) {
	now := time.Now().UTC()
	start, err := parseTimeParam(r.URL.Query().Get("start"), now.Add(-defaultMetricsRange))
	if err != nil {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid start timestamp")
	}

	end, err := parseTimeParam(r.URL.Query().Get("end"), now)
	if err != nil {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid end timestamp")
	}

	step, err := parseStepParam(r.URL.Query().Get("step"))
	if err != nil {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid step")
	}
	if step <= 0 {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid step")
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("end must be after start")
	}

	return start, end, step, nil
}

// parseTimeParam parses an ISO8601 or Unix timestamp, with a fallback default.
func parseTimeParam(raw string, fallback time.Time) (time.Time, error) {
	if raw == "" {
		return fallback, nil
	}

	if parsed, err := parseRFC3339(raw); err == nil {
		return parsed, nil
	}

	parsed, err := parseUnixTimestamp(raw)
	if err != nil {
		return time.Time{}, err
	}

	return parsed, nil
}

// parseRFC3339 parses a timestamp in RFC3339 or RFC3339Nano format.
func parseRFC3339(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, raw)
	if err == nil {
		return parsed.UTC(), nil
	}

	parsed, err = time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, err
	}

	return parsed.UTC(), nil
}

// parseUnixTimestamp parses a Unix timestamp string into time.Time.
func parseUnixTimestamp(raw string) (time.Time, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return time.Time{}, err
	}

	seconds, fraction := math.Modf(value)
	return time.Unix(int64(seconds), int64(fraction*float64(time.Second))).UTC(), nil
}

// parseStepParam parses a Prometheus duration string into time.Duration.
func parseStepParam(raw string) (time.Duration, error) {
	if raw == "" {
		return defaultMetricsStep, nil
	}

	if parsed, err := time.ParseDuration(raw); err == nil {
		return parsed, nil
	}

	return parseExtendedDuration(raw)
}

// parseExtendedDuration handles day/week/year suffixes not supported by time.ParseDuration.
func parseExtendedDuration(raw string) (time.Duration, error) {
	if len(raw) < 2 {
		return 0, fmt.Errorf("invalid duration")
	}

	number := raw[:len(raw)-1]
	unit := raw[len(raw)-1]

	multiplier, ok := map[byte]time.Duration{
		'd': 24 * time.Hour,
		'w': 7 * 24 * time.Hour,
		'y': 365 * 24 * time.Hour,
	}[unit]
	if !ok {
		return 0, fmt.Errorf("invalid duration")
	}

	value, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration")
	}
	if value <= 0 {
		return 0, fmt.Errorf("invalid duration")
	}

	return time.Duration(value * float64(multiplier)), nil
}

// buildMetricsQuery builds a PromQL selector for the device metrics series.
func buildMetricsQuery(deviceID, measurement string) (string, error) {
	quotedDeviceID, err := quotePromQLLabelValue(deviceID)
	if err != nil {
		return "", err
	}

	if measurement == "" {
		return fmt.Sprintf("device_metrics{device_id=%s}", quotedDeviceID), nil
	}

	quotedMeasurement, err := quotePromQLLabelValue(measurement)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("device_metrics{device_id=%s,measurement=%s}", quotedDeviceID, quotedMeasurement), nil
}

// quotePromQLLabelValue safely quotes a label value for PromQL.
func quotePromQLLabelValue(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("value is required")
	}
	if len(value) > maxQueryParamLen {
		return "", fmt.Errorf("value exceeds maximum length")
	}

	return strconv.Quote(value), nil
}

// parseMetricsSummary converts a Prometheus instant query response into summary data.
func parseMetricsSummary(raw json.RawMessage) (map[string]metricsLatestEntry, []string, error) {
	var response promVectorResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, nil, err
	}
	if response.Status != "success" {
		return nil, nil, fmt.Errorf("query status %q", response.Status)
	}

	latest := make(map[string]metricsLatestEntry)
	for _, result := range response.Data.Result {
		measurement := strings.TrimSpace(result.Metric["measurement"])
		if measurement == "" {
			continue
		}

		timestamp, value, err := parsePrometheusValue(result.Value)
		if err != nil {
			return nil, nil, err
		}

		latest[measurement] = metricsLatestEntry{
			Value:     value,
			Timestamp: timestamp.UTC().Format(time.RFC3339),
		}
	}

	fields := make([]string, 0, len(latest))
	for field := range latest {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	return latest, fields, nil
}

// parsePrometheusValue extracts timestamp and value from a Prometheus sample.
func parsePrometheusValue(raw []any) (time.Time, float64, error) {
	if len(raw) != 2 {
		return time.Time{}, 0, fmt.Errorf("invalid sample length")
	}

	timestamp, err := parsePrometheusTimestamp(raw[0])
	if err != nil {
		return time.Time{}, 0, err
	}

	value, err := parsePrometheusSampleValue(raw[1])
	if err != nil {
		return time.Time{}, 0, err
	}

	return timestamp, value, nil
}

// parsePrometheusTimestamp parses the Prometheus sample timestamp value.
func parsePrometheusTimestamp(raw any) (time.Time, error) {
	switch value := raw.(type) {
	case float64:
		seconds, fraction := math.Modf(value)
		return time.Unix(int64(seconds), int64(fraction*float64(time.Second))).UTC(), nil
	case string:
		return parseUnixTimestamp(value)
	default:
		return time.Time{}, fmt.Errorf("invalid timestamp")
	}
}

// parsePrometheusSampleValue parses a Prometheus sample value.
func parsePrometheusSampleValue(raw any) (float64, error) {
	switch value := raw.(type) {
	case string:
		return strconv.ParseFloat(value, 64)
	case float64:
		return value, nil
	default:
		return 0, fmt.Errorf("invalid sample value")
	}
}
