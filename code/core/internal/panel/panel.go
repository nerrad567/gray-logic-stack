package panel

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

//go:embed web/*
var content embed.FS

// Handler returns an http.Handler that serves the Flutter web UI.
//
// When dir is non-empty and the directory exists, assets are served from the
// filesystem (dev mode — no recompile needed after Flutter rebuild).
// When dir is empty, assets are served from the embedded go:embed FS (production).
//
// Both modes implement SPA fallback: if a requested file doesn't exist,
// index.html is served so client-side routing works correctly.
// Panics if the embedded web assets cannot be loaded (build error).
func Handler(dir string) http.Handler {
	var fileSystem http.FileSystem

	if dir != "" {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			fileSystem = http.Dir(dir)
		}
	}

	// Fall back to embedded assets if dir was empty or didn't exist
	if fileSystem == nil {
		webFS, err := fs.Sub(content, "web")
		if err != nil {
			panic(fmt.Sprintf("panel: failed to load embedded web assets: %v", err))
		}
		fileSystem = http.FS(webFS)
	}

	fileServer := http.FileServer(fileSystem)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the path
		upath := path.Clean(r.URL.Path)
		if upath == "." {
			upath = "/"
		}

		// Set cache headers based on asset mutability.
		// Flutter content-hashed chunks (e.g. main.dart.js.sha256.js) are immutable —
		// the filename changes when content changes — so they get long-lived cache.
		// Mutable files (index.html, manifest.json) get no-cache.
		setCacheHeaders(w, upath)

		// For root, let FileServer handle it (serves index.html automatically)
		if upath == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Try to open the requested file
		filePath := upath[1:] // strip leading /
		f, err := fileSystem.Open(filePath)
		if err != nil {
			// File not found — SPA fallback: serve index.html with 200
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close()

		// File exists — serve it directly
		fileServer.ServeHTTP(w, r)
	})
}

// setCacheHeaders sets appropriate Cache-Control headers based on the asset path.
// Content-hashed assets (Flutter build chunks) get long-lived immutable cache.
// Mutable assets (index.html, manifest, service worker) get no-cache.
func setCacheHeaders(w http.ResponseWriter, upath string) {
	// Mutable files that must always be revalidated
	if upath == "/" || strings.HasSuffix(upath, ".html") ||
		strings.HasSuffix(upath, "manifest.json") ||
		strings.HasSuffix(upath, "flutter_service_worker.js") {
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		return
	}

	// Assets under /assets/ are Flutter content-hashed chunks — immutable
	if strings.HasPrefix(upath, "/assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}

	// Default: short cache for other static files (icons, fonts, etc.)
	w.Header().Set("Cache-Control", "public, max-age=3600")
}
