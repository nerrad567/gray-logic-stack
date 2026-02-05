package tsdb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestClient creates a TSDB client bound to the test server.
func newTestClient(server *httptest.Server) *Client {
	return &Client{
		url:        server.URL,
		httpClient: server.Client(),
		connected:  true,
	}
}

// TestQueryRange verifies query parameter handling and response passthrough.
func TestQueryRange(t *testing.T) {
	var gotQuery string
	var gotParams = make(map[string]string)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query_range" {
			t.Fatalf("path = %q, want /api/v1/query_range", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		for key, values := range r.URL.Query() {
			if len(values) > 0 {
				gotParams[key] = values[0]
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":{"result":[]}}`))
	}))
	defer server.Close()

	client := newTestClient(server)

	start := time.Date(2026, 2, 5, 8, 0, 0, 0, time.UTC)
	end := start.Add(10 * time.Minute)
	step := time.Minute
	query := `device_metrics{device_id="dev-1",measurement="level"}`

	resp, err := client.QueryRange(context.Background(), query, start, end, step)
	if err != nil {
		t.Fatalf("QueryRange() error = %v", err)
	}
	if gotQuery == "" {
		t.Fatal("query parameters were not captured")
	}
	if gotParams["query"] != query {
		t.Errorf("query param = %q, want %q", gotParams["query"], query)
	}
	if gotParams["start"] != formatUnixSeconds(start) {
		t.Errorf("start param = %q, want %q", gotParams["start"], formatUnixSeconds(start))
	}
	if gotParams["end"] != formatUnixSeconds(end) {
		t.Errorf("end param = %q, want %q", gotParams["end"], formatUnixSeconds(end))
	}
	if gotParams["step"] != formatStepSeconds(step) {
		t.Errorf("step param = %q, want %q", gotParams["step"], formatStepSeconds(step))
	}

	var payload map[string]any
	if err := json.Unmarshal(resp, &payload); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if status, ok := payload["status"].(string); !ok || status != "success" {
		t.Fatalf("status = %v, want success", payload["status"])
	}
}

// TestQueryInstant verifies instant query handling and response passthrough.
func TestQueryInstant(t *testing.T) {
	var gotParam string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query" {
			t.Fatalf("path = %q, want /api/v1/query", r.URL.Path)
		}
		gotParam = r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":{"result":[]}}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	query := `device_metrics{device_id="dev-1"}`

	resp, err := client.QueryInstant(context.Background(), query)
	if err != nil {
		t.Fatalf("QueryInstant() error = %v", err)
	}
	if gotParam != query {
		t.Errorf("query param = %q, want %q", gotParam, query)
	}

	var payload map[string]any
	if err := json.Unmarshal(resp, &payload); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
}

// TestQueryRange_InvalidQuery ensures empty queries are rejected.
func TestQueryRange_InvalidQuery(t *testing.T) {
	client := &Client{connected: true}

	_, err := client.QueryRange(context.Background(), "", time.Now().UTC(), time.Now().UTC(), time.Minute)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

// TestQueryRange_ServerError verifies non-200 responses are surfaced as errors.
func TestQueryRange_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := newTestClient(server)

	_, err := client.QueryRange(context.Background(), "up", time.Now().UTC().Add(-time.Minute), time.Now().UTC(), time.Second)
	if err == nil {
		t.Fatal("expected error for 503 response")
	}
}

// TestQueryInstant_ServerError verifies instant query errors are surfaced.
func TestQueryInstant_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := newTestClient(server)

	_, err := client.QueryInstant(context.Background(), "up")
	if err == nil {
		t.Fatal("expected error for 503 response")
	}
}

// TestQueryRange_Timeout verifies context cancellation propagates.
func TestQueryRange_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	client := newTestClient(server)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.QueryRange(ctx, "up", time.Now().UTC().Add(-time.Minute), time.Now().UTC(), time.Second)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want deadline exceeded", err)
	}
}
