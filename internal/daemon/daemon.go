//go:build unix

package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gurisko/takl/internal/paths"
	"github.com/gurisko/takl/internal/registry"
)

// ensureParentDir ensures the parent directory of the given path exists with secure permissions
func ensureParentDir(path string) error {
	dir := filepath.Dir(path)

	// Create directory with 0700 permissions (owner only)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Ensure directory has correct permissions (best effort)
	_ = os.Chmod(dir, 0o700)

	return nil
}

// removeSocketIfExists removes the socket file if it exists and is actually a socket
func removeSocketIfExists(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if fi.Mode()&os.ModeSocket != 0 {
		return os.Remove(path)
	}

	return fmt.Errorf("refusing to remove non-socket path: %s", path)
}

type Daemon struct {
	socketPath string
	pidFile    string
	listener   net.Listener
	server     *http.Server
	registry   *registry.Registry
	httpClient *http.Client

	// Stats
	startTime time.Time
}

type Config struct {
	SocketPath   string
	PIDFile      string
	RegistryPath string
}

func DefaultConfig() *Config {
	return &Config{
		SocketPath:   paths.DefaultSocketPath(),
		PIDFile:      paths.DefaultPIDPath(),
		RegistryPath: paths.DefaultRegistryPath(),
	}
}

func New(cfg *Config) (*Daemon, error) {
	// Apply defaults for any empty fields
	defaults := DefaultConfig()
	if cfg.SocketPath == "" {
		cfg.SocketPath = defaults.SocketPath
	}
	if cfg.PIDFile == "" {
		cfg.PIDFile = defaults.PIDFile
	}
	if cfg.RegistryPath == "" {
		cfg.RegistryPath = defaults.RegistryPath
	}

	// Initialize registry
	reg, err := registry.New(cfg.RegistryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Create HTTP client for Unix socket communication
	tr := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var nd net.Dialer
			return nd.DialContext(ctx, "unix", cfg.SocketPath)
		},
	}

	return &Daemon{
		socketPath: cfg.SocketPath,
		pidFile:    cfg.PIDFile,
		registry:   reg,
		httpClient: &http.Client{Transport: tr, Timeout: 2 * time.Second},
		startTime:  time.Now().UTC(),
	}, nil
}

func (d *Daemon) Start() error {
	// Check if already running
	if d.IsRunning() {
		pid, _ := d.readPIDFile()
		return fmt.Errorf("daemon already running (PID: %d)", pid)
	}

	return d.startForeground()
}

func (d *Daemon) startForeground() error {
	// Ensure parent directory exists with secure permissions
	if err := ensureParentDir(d.socketPath); err != nil {
		return fmt.Errorf("failed to prepare socket directory: %w", err)
	}

	// Remove any existing socket (but only if it's actually a socket)
	if err := removeSocketIfExists(d.socketPath); err != nil {
		return err
	}

	// Create Unix socket listener
	listener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}
	d.listener = listener

	// Set socket permissions (owner only)
	if err := os.Chmod(d.socketPath, 0600); err != nil {
		listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	// Write PID file
	if err := d.writePIDFile(); err != nil {
		listener.Close()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	d.setupRoutes(mux)

	d.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
		BaseContext:  func(net.Listener) context.Context { return context.Background() },
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigChan)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		fmt.Printf("TAKL daemon started (PID: %d)\n", os.Getpid())
		fmt.Printf("Socket: %s\n", d.socketPath)
		serverErr <- d.server.Serve(listener)
	}()

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal %v, shutting down...\n", sig)
	case err := <-serverErr:
		if err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}

	// Graceful shutdown
	d.shutdown()
	return nil
}

func (d *Daemon) Stop() error {
	pid, err := d.readPIDFile()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("daemon not running")
		}
		return fmt.Errorf("failed reading pidfile: %w", err)
	}

	// Send SIGTERM
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}

	// Wait for shutdown (max 5 seconds)
	for i := 0; i < 50; i++ {
		if !d.IsRunning() {
			fmt.Println("TAKL daemon stopped")
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("daemon did not stop gracefully")
}

func (d *Daemon) GetStatus() (*StatusInfo, error) {
	info := &StatusInfo{
		SocketPath: d.socketPath,
	}

	pid, err := d.readPIDFile()
	if err != nil {
		// No PID file
		return info, nil
	}

	info.PID = pid

	// Check if process is alive
	if !isProcessAlive(pid) {
		// Stale PID file
		return info, nil
	}

	// Try to get health from daemon to verify identity
	health, err := d.getHealth()
	if err != nil {
		// Process alive but not responding on socket
		info.ErrorMessage = err.Error()
		return info, nil
	}

	// Daemon is healthy and responding
	info.Running = true
	info.Uptime = time.Duration(health.Uptime * float64(time.Second))
	return info, nil
}

func (d *Daemon) IsRunning() bool {
	pid, err := d.readPIDFile()
	if err != nil {
		return false
	}

	// Check if process is alive
	if !isProcessAlive(pid) {
		return false
	}

	// Verify daemon identity by checking if it responds on socket
	// This protects against PID reuse
	if _, err := d.getHealth(); err != nil {
		return false
	}

	return true
}

func (d *Daemon) shutdown() {
	// Shutdown server
	if d.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := d.server.Shutdown(ctx); err != nil {
			fmt.Printf("Warning: server shutdown error: %v\n", err)
		}
	}

	// Close HTTP client connections
	if d.httpClient != nil {
		d.httpClient.CloseIdleConnections()
	}

	// Close listener
	if d.listener != nil {
		d.listener.Close()
	}

	// Clean up
	removeSocketIfExists(d.socketPath)
	os.Remove(d.pidFile)
}

func (d *Daemon) writePIDFile() error {
	pid := os.Getpid()

	// Ensure parent directory exists
	if err := ensureParentDir(d.pidFile); err != nil {
		return fmt.Errorf("failed to create PID file directory: %w", err)
	}

	// Try to create PID file atomically with O_EXCL
	for {
		f, err := os.OpenFile(d.pidFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			defer f.Close()
			_, err = f.WriteString(strconv.Itoa(pid))
			return err
		}
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create PID file: %w", err)
		}
		// File exists, check if process is still alive
		if oldPID, err2 := d.readPIDFile(); err2 == nil && isProcessAlive(oldPID) {
			return fmt.Errorf("daemon already running (PID: %d)", oldPID)
		}
		// Stale PID file; remove and retry
		if err := os.Remove(d.pidFile); err != nil {
			return fmt.Errorf("stale pidfile exists and cannot remove: %w", err)
		}
	}
}

// isProcessAlive checks if a process with the given PID is alive
func isProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Send signal 0 to check if process is alive
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func (d *Daemon) readPIDFile() (int, error) {
	data, err := os.ReadFile(d.pidFile)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(strings.TrimSpace(string(data)))
}

type HealthResponse struct {
	Status string  `json:"status"`
	Uptime float64 `json:"uptime"`
}

type StatusInfo struct {
	Running      bool
	PID          int
	SocketPath   string
	Uptime       time.Duration
	ErrorMessage string // For when process exists but not responding
}

func (d *Daemon) getHealth() (*HealthResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/health", nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health returned HTTP %d", resp.StatusCode)
	}

	lr := io.LimitReader(resp.Body, 1<<20) // 1MB safety cap

	var health HealthResponse
	if err := json.NewDecoder(lr).Decode(&health); err != nil {
		return nil, err
	}

	return &health, nil
}
