package panel

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
)

//go:embed web/*
var content embed.FS

// Handler returns an http.Handler that serves the Flutter web UI.
// It implements SPA fallback: if a requested file doesn't exist,
// it serves index.html so client-side routing works correctly.
func Handler() http.Handler {
	webFS, _ := fs.Sub(content, "web")
	fileServer := http.FileServer(http.FS(webFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent aggressive caching of mutable assets (index.html, JS).
		// Flutter hashes its chunk files, so this is safe for cache-busting.
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")

		// Clean the path
		upath := path.Clean(r.URL.Path)
		if upath == "." {
			upath = "/"
		}

		// For root, let FileServer handle it (serves index.html automatically)
		if upath == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Try to open the requested file
		filePath := upath[1:] // strip leading /
		f, err := webFS.Open(filePath)
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
