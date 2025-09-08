package web

import (
	"embed"
	"io/fs"
	"net/http"
)

// Embed the web assets from the dist directory
// This will be populated when we build the web app
// Using a build constraint to only embed when dist directory exists
//
//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded web assets filesystem
// If the embedded files don't exist (during development), returns nil
func DistFS() (fs.FS, error) {
	// Try to access the dist directory to see if it exists
	_, err := distFS.ReadDir("dist")
	if err != nil {
		return nil, err
	}

	return fs.Sub(distFS, "dist")
}

// HTTPHandler returns an HTTP handler for serving the embedded assets
func HTTPHandler() (http.Handler, error) {
	fsys, err := DistFS()
	if err != nil {
		return nil, err
	}

	return http.FileServer(http.FS(fsys)), nil
}

// HasEmbeddedAssets returns true if embedded assets are available
func HasEmbeddedAssets() bool {
	_, err := distFS.ReadDir("dist")
	return err == nil
}
