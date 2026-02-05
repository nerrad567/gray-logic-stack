package tsdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// QueryRange executes a PromQL range query against VictoriaMetrics.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - query: PromQL query string
//   - start: Start time for the range
//   - end: End time for the range
//   - step: Query resolution step
//
// Returns:
//   - json.RawMessage: Raw Prometheus API JSON response
//   - error: nil on success, otherwise the query error
func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (json.RawMessage, error) {
	if c == nil || !c.IsConnected() {
		return nil, ErrNotConnected
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("tsdb query is required")
	}
	if step <= 0 {
		return nil, fmt.Errorf("step must be positive")
	}
	if end.Before(start) {
		return nil, fmt.Errorf("end must be after start")
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("start", formatUnixSeconds(start))
	params.Set("end", formatUnixSeconds(end))
	params.Set("step", formatStepSeconds(step))

	return c.doQuery(ctx, "/api/v1/query_range", params)
}

// QueryInstant executes a PromQL instant query against VictoriaMetrics.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - query: PromQL query string
//
// Returns:
//   - json.RawMessage: Raw Prometheus API JSON response
//   - error: nil on success, otherwise the query error
func (c *Client) QueryInstant(ctx context.Context, query string) (json.RawMessage, error) {
	if c == nil || !c.IsConnected() {
		return nil, ErrNotConnected
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("tsdb query is required")
	}

	params := url.Values{}
	params.Set("query", query)

	return c.doQuery(ctx, "/api/v1/query", params)
}

// doQuery executes a query request and returns the raw response body.
func (c *Client) doQuery(ctx context.Context, path string, params url.Values) (json.RawMessage, error) {
	endpoint := c.url + path
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	defer resp.Body.Close()

	const maxResponseSize = 10 << 20 // 10 MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query failed: HTTP %d", resp.StatusCode)
	}

	return json.RawMessage(body), nil
}

// formatUnixSeconds converts a timestamp to a seconds-since-epoch string.
func formatUnixSeconds(t time.Time) string {
	seconds := float64(t.UnixNano()) / float64(time.Second)
	return strconv.FormatFloat(seconds, 'f', -1, 64)
}

// formatStepSeconds converts a step duration to a Prometheus-compatible seconds string.
func formatStepSeconds(step time.Duration) string {
	return strconv.FormatFloat(step.Seconds(), 'f', -1, 64)
}
