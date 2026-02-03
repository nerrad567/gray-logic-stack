package panel

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandlerServesRoot(t *testing.T) {
	handler := Handler("")

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
	handler := Handler("")

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
	handler := Handler("")

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
	handler := Handler("")

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
	handler := Handler("")

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

func TestHandlerFilesystemMode(t *testing.T) {
	// Create a temp directory with a minimal index.html
	dir := t.TempDir()
	indexContent := `<!DOCTYPE html><html><body>filesystem panel</body></html>`
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(indexContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test.js"), []byte("console.log('test')"), 0644); err != nil {
		t.Fatal(err)
	}

	handler := Handler(dir)

	// Root should serve filesystem index.html
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("filesystem GET /: got status %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "filesystem panel") {
		t.Errorf("filesystem GET /: expected filesystem content, got %q", w.Body.String())
	}

	// Static asset should be served from filesystem
	req = httptest.NewRequest(http.MethodGet, "/test.js", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("filesystem GET /test.js: got status %d, want 200", w.Code)
	}

	// SPA fallback should work with filesystem too
	req = httptest.NewRequest(http.MethodGet, "/deep/route", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("filesystem SPA fallback: got status %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "filesystem panel") {
		t.Error("filesystem SPA fallback didn't serve filesystem index.html")
	}
}

func TestHandlerInvalidDirFallsBackToEmbed(t *testing.T) {
	// Non-existent dir should fall back to embedded assets
	handler := Handler("/nonexistent/dir/that/does/not/exist")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("invalid dir GET /: got status %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<!DOCTYPE html>") {
		t.Error("invalid dir: didn't fall back to embedded index.html")
	}
}
