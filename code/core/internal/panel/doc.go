// Package panel serves the Flutter wall panel web UI as an embedded asset.
//
// The compiled Flutter web build is embedded into the Go binary using
// the go:embed directive, eliminating any runtime dependency on external files.
// The Handler function returns an http.Handler that serves these assets
// with SPA (Single Page Application) fallback routing: if a requested
// file does not exist, index.html is served so that client-side routing
// works correctly.
//
// Cache-control headers are set to no-cache for mutable assets (index.html,
// JS bootstrap). Flutter's content-hashed chunk files ensure proper
// cache-busting for immutable assets.
package panel
