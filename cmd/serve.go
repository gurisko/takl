package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/takl/takl/internal/daemon"
	"github.com/takl/takl/internal/web"
)

var (
	servePort   int
	serveHost   string
	serveDir    string
	serveSocket string
	serveOpen   bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve TAKL web UI and proxy API",
	Long: `Serves the TAKL web interface and reverse-proxies API calls to the daemon.

The serve command starts a local HTTP server that:
- Serves the TAKL web application from embedded assets or a directory
- Reverse-proxies all /api/* requests to the TAKL daemon Unix socket
- Provides a complete web interface for issue management

Examples:
  takl serve                    # Start on default port (5173) with embedded assets
  takl serve --port 3000        # Start on port 3000
  takl serve --host 0.0.0.0     # Allow external connections (default: localhost only)
  takl serve --dir web/dist     # Serve from local build directory (for development)

The web interface will be available at http://host:port/`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVar(&servePort, "port", 5173, "HTTP port to listen on")
	serveCmd.Flags().StringVar(&serveHost, "host", "127.0.0.1", "HTTP host to bind to")
	serveCmd.Flags().StringVar(&serveDir, "dir", "", "Directory of built web assets (defaults to embedded)")
	serveCmd.Flags().StringVar(&serveSocket, "socket", "", "Daemon socket path (defaults to TAKL_SOCKET or ~/.takl/daemon.sock)")
	serveCmd.Flags().BoolVar(&serveOpen, "open", false, "Open browser after starting server")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Ensure daemon is running
	client := sdkClient()
	if err := client.EnsureDaemonRunning(); err != nil {
		return fmt.Errorf("failed to ensure daemon is running: %w", err)
	}

	// Determine socket path
	sock := serveSocket
	if sock == "" {
		cfg := daemon.DefaultConfig()
		sock = cfg.SocketPath
	}

	// Verify socket exists and is accessible
	if err := testSocketConnection(sock); err != nil {
		return fmt.Errorf("daemon socket not accessible: %w", err)
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// Setup API reverse proxy
	apiProxy, err := createUnixSocketReverseProxy(sock)
	if err != nil {
		return fmt.Errorf("failed to create API proxy: %w", err)
	}
	mux.Handle("/api/", apiProxy)

	// Setup static file serving
	staticHandler, err := createStaticFileHandler(serveDir)
	if err != nil {
		return fmt.Errorf("failed to create static file handler: %w", err)
	}

	// SPA handler - all routes serve index.html except API calls and static assets
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Check if this looks like a static asset first
		if isStaticAsset(path) {
			staticHandler.ServeHTTP(w, r)
			return
		}

		// For all other paths (including root), serve index.html for SPA routing
		// The custom file handler will automatically serve index.html for "/"
		staticHandler.ServeHTTP(w, r)
	})

	// Add a healthz endpoint for the proxy itself
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		// Check daemon connectivity
		if err := testSocketConnection(sock); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","error":"daemon unreachable: %s"}`, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"healthy","daemon_socket":"%s"}`, sock)
	})

	// Start server
	addr := fmt.Sprintf("%s:%d", serveHost, servePort)

	log.Printf("TAKL web UI starting on http://%s", addr)
	log.Printf("Proxying /api to daemon socket: %s", sock)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Open browser if requested
	if serveOpen {
		go func() {
			time.Sleep(500 * time.Millisecond) // Give server time to start
			if err := openBrowser(fmt.Sprintf("http://%s", addr)); err != nil {
				log.Printf("Failed to open browser: %v", err)
			}
		}()
	}

	log.Printf("TAKL web UI ready at http://%s", addr)
	return srv.ListenAndServe()
}

func createUnixSocketReverseProxy(socketPath string) (http.Handler, error) {
	target, _ := url.Parse("http://unix")
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Custom dialer for Unix socket
	proxy.Transport = &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
		MaxIdleConns:      10,
		IdleConnTimeout:   30 * time.Second,
		DisableKeepAlives: false,
	}

	// Custom director to preserve original paths
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = "unix"
		// Keep original path (already starts with /api/)
		req.Header.Set("X-Forwarded-For", req.RemoteAddr)
		req.Header.Set("X-Forwarded-Proto", "http")
		req.Header.Set("X-Backend", "unix-socket")
	}

	return proxy, nil
}

func createStaticFileHandler(dir string) (http.Handler, error) {
	if dir != "" {
		// Serve from filesystem directory with custom handler to avoid redirect issues
		if !dirExists(dir) {
			return nil, fmt.Errorf("directory does not exist: %s", dir)
		}
		return &customFileHandler{dir: dir}, nil
	}

	// Try to serve from embedded assets first
	if web.HasEmbeddedAssets() {
		fsys, err := web.DistFS()
		if err == nil {
			return &customFileHandler{fsys: fsys}, nil
		}
		log.Printf("Warning: embedded assets found but failed to create handler: %v", err)
	}

	// Fallback to basic placeholder HTML
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, basicIndexHTML())
		} else {
			http.NotFound(w, r)
		}
	}), nil
}

type customFileHandler struct {
	dir  string
	fsys fs.FS
}

func (h *customFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	var content []byte
	var err error

	if h.fsys != nil {
		// Serve from embedded filesystem
		content, err = fs.ReadFile(h.fsys, path)
		// If file doesn't exist and it's not a static asset, serve index.html for SPA routing
		if err != nil && !isStaticAsset(r.URL.Path) {
			content, err = fs.ReadFile(h.fsys, "index.html")
		}
	} else {
		// Serve from directory
		fullPath := filepath.Join(h.dir, path)
		content, err = os.ReadFile(fullPath)
		// If file doesn't exist and it's not a static asset, serve index.html for SPA routing
		if err != nil && !isStaticAsset(r.URL.Path) {
			content, err = os.ReadFile(filepath.Join(h.dir, "index.html"))
			path = "index.html" // Update path for correct content-type
		}
	}

	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Set appropriate content type
	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(content); err != nil {
		// If we can't write the response, there's not much we can do
		// The client likely disconnected
		_ = err // Silence linter
	}
}

func isStaticAsset(path string) bool {
	// SvelteKit assets are in /_app/ directory
	if strings.HasPrefix(path, "/_app/") {
		return true
	}

	// Common static files at root level
	staticFiles := map[string]bool{
		"/favicon.ico": true, "/favicon.png": true, "/robots.txt": true,
		"/manifest.json": true, "/service-worker.js": true,
		"/app.webmanifest": true,
	}

	if staticFiles[path] {
		return true
	}

	// Files with static extensions at root or shallow paths
	if ext := filepath.Ext(path); ext != "" {
		staticExts := map[string]bool{
			".css": true, ".js": true, ".json": true, ".map": true,
			".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true, ".ico": true,
			".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
			".txt": true, ".xml": true,
		}

		if staticExts[strings.ToLower(ext)] {
			return true
		}
	}

	return false
}

func testSocketConnection(socketPath string) error {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func dirExists(path string) bool {
	if stat, err := os.Stat(path); err == nil {
		return stat.IsDir()
	}
	return false
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch {
	case commandExists("xdg-open"):
		cmd = "xdg-open"
		args = []string{url}
	case commandExists("open"):
		cmd = "open"
		args = []string{url}
	case commandExists("cmd"):
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return fmt.Errorf("no suitable browser launcher found")
	}

	return exec.Command(cmd, args...).Start()
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func basicIndexHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>TAKL Web Interface</title>
	<style>
		body { font-family: system-ui, sans-serif; margin: 2rem; background: #f5f5f5; }
		.container { max-width: 800px; margin: 0 auto; background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
		h1 { color: #333; margin-bottom: 1rem; }
		.status { padding: 1rem; border-radius: 4px; margin: 1rem 0; }
		.success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
		.info { background: #d1ecf1; color: #0c5460; border: 1px solid #bee5eb; }
		.api-link { color: #007bff; text-decoration: none; }
		.api-link:hover { text-decoration: underline; }
		ul { padding-left: 1.5rem; }
		li { margin: 0.5rem 0; }
	</style>
</head>
<body>
	<div class="container">
		<h1>🎯 TAKL Web Interface</h1>
		
		<div class="status success">
			✅ Web server is running and connected to TAKL daemon
		</div>
		
		<div class="status info">
			<strong>Note:</strong> This is a placeholder interface. The full Svelte web application will be embedded here in the production build.
		</div>
		
		<h2>Available API Endpoints</h2>
		<p>The daemon API is available and proxied through this server:</p>
		<ul>
			<li><a href="/api/registry/projects" class="api-link">/api/registry/projects</a> - List registered projects</li>
			<li><a href="/api/registry/health" class="api-link">/api/registry/health</a> - Registry health check</li>
			<li><a href="/health" class="api-link">/health</a> - Daemon health</li>
			<li><a href="/stats" class="api-link">/stats</a> - Daemon statistics</li>
			<li><a href="/healthz" class="api-link">/healthz</a> - Proxy health check</li>
		</ul>
		
		<h2>Development</h2>
		<p>To develop with the full Svelte interface:</p>
		<ol>
			<li>Build the web assets: <code>cd web && npm run build</code></li>
			<li>Serve with local assets: <code>takl serve --dir web/dist</code></li>
		</ol>
	</div>
</body>
</html>`
}
