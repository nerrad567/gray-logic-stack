package panel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesRoot(t *testing.T) {
	handler := Handler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /: got status %d, want 200", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("GET /: empty response body")
	}
	if !strings.Contains(w.Body.String(), "<!DOCTYPE html>") {
		t.Error("GET /: response doesn't contain HTML doctype")
	}
}

func TestHandlerServesStaticAsset(t *testing.T) {
	handler := Handler()

	req := httptest.NewRequest(http.MethodGet, "/flutter.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /flutter.js: got status %d, want 200", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("GET /flutter.js: empty response body")
	}
}

func TestHandlerServesManifest(t *testing.T) {
	handler := Handler()

	req := httptest.NewRequest(http.MethodGet, "/manifest.json", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /manifest.json: got status %d, want 200", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("GET /manifest.json: empty response body")
	}
}

func TestHandlerSPAFallback(t *testing.T) {
	handler := Handler()

	// Non-existent path should return index.html content (SPA routing)
	req := httptest.NewRequest(http.MethodGet, "/some/deep/route", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /some/deep/route: got status %d, want 200 (SPA fallback)", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("GET /some/deep/route: empty body (expected index.html)")
	}
	if !strings.Contains(w.Body.String(), "<!DOCTYPE html>") {
		t.Error("GET /some/deep/route: SPA fallback didn't serve index.html")
	}
}

func TestHandlerSPAFallbackSingleSegment(t *testing.T) {
	handler := Handler()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /nonexistent: got status %d, want 200 (SPA fallback)", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<!DOCTYPE html>") {
		t.Error("GET /nonexistent: SPA fallback didn't serve index.html")
	}
}
