//go:build integration

package daemon_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/takl/takl/internal/daemon"
)

func TestDaemonIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"daemon lifecycle", testDaemonLifecycle},
		{"daemon API endpoints", testDaemonAPIEndpoints},
		{"daemon concurrent requests", testDaemonConcurrentRequests},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Don't run integration tests in parallel to avoid port conflicts
			tt.testFunc(t)
		})
	}
}

func testDaemonLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock home directory with cleanup
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })
	os.Setenv("HOME", tmpDir)

	// Create daemon with custom socket path
	socketPath := filepath.Join(tmpDir, "test-daemon.sock")
	pidPath := filepath.Join(tmpDir, "test-daemon.pid")

	d, err := daemon.New(&daemon.Config{
		SocketPath: socketPath,
		PIDFile:    pidPath,
	})
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Test that daemon is not initially running
	if d.IsRunning() {
		t.Error("Daemon should not be running initially")
	}

	// Start daemon in background
	doneCh := make(chan error, 1)
	go func() {
		doneCh <- d.Start()
	}()

	// Give daemon time to start
	time.Sleep(100 * time.Millisecond)

	// Test that daemon is now running
	if !d.IsRunning() {
		t.Error("Daemon should be running after start")
	}

	// Test daemon stop
	if err := d.Stop(); err != nil {
		t.Errorf("Failed to stop daemon: %v", err)
	}

	// Wait for daemon to finish
	select {
	case err := <-doneCh:
		if err != nil {
			t.Errorf("Daemon returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Daemon did not shut down within timeout")
	}

	// Verify daemon is no longer running
	if d.IsRunning() {
		t.Error("Daemon should not be running after stop")
	}
}

func testDaemonAPIEndpoints(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock home directory with cleanup
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })
	os.Setenv("HOME", tmpDir)

	// Create test project structure
	projectDir := filepath.Join(tmpDir, "integration-test-project")
	os.MkdirAll(projectDir, 0755)
	os.MkdirAll(filepath.Join(projectDir, ".takl", "issues"), 0755)

	// Create daemon with custom socket path
	socketPath := filepath.Join(tmpDir, "api-test-daemon.sock")
	pidPath := filepath.Join(tmpDir, "api-test-daemon.pid")

	d, err := daemon.New(&daemon.Config{
		SocketPath: socketPath,
		PIDFile:    pidPath,
	})
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Start daemon in background
	doneCh := make(chan error, 1)
	go func() {
		doneCh <- d.Start()
	}()

	// Give daemon time to start
	time.Sleep(100 * time.Millisecond)

	// Ensure cleanup
	t.Cleanup(func() {
		d.Stop()
		select {
		case <-doneCh:
		case <-time.After(2 * time.Second):
		}
	})

	// Create HTTP client that can talk to Unix socket
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
		Timeout: 5 * time.Second,
	}

	// Test health endpoint
	resp, err := client.Get("http://localhost/health")
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var healthResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if healthResp["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", healthResp["status"])
	}

	// Test stats endpoint
	resp, err = client.Get("http://localhost/stats")
	if err != nil {
		t.Fatalf("Failed to call stats endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var statsResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&statsResp); err != nil {
		t.Fatalf("Failed to decode stats response: %v", err)
	}

	if _, exists := statsResp["request_count"]; !exists {
		t.Error("Expected request_count in stats response")
	}
}

func testDaemonConcurrentRequests(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock home directory with cleanup
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })
	os.Setenv("HOME", tmpDir)

	// Create daemon with custom socket path
	socketPath := filepath.Join(tmpDir, "concurrent-test-daemon.sock")
	pidPath := filepath.Join(tmpDir, "concurrent-test-daemon.pid")

	d, err := daemon.New(&daemon.Config{
		SocketPath: socketPath,
		PIDFile:    pidPath,
	})
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Start daemon in background
	doneCh := make(chan error, 1)
	go func() {
		doneCh <- d.Start()
	}()

	// Give daemon time to start
	time.Sleep(100 * time.Millisecond)

	// Ensure cleanup
	t.Cleanup(func() {
		d.Stop()
		select {
		case <-doneCh:
		case <-time.After(2 * time.Second):
		}
	})

	// Create HTTP client
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
		Timeout: 5 * time.Second,
	}

	// Make concurrent requests
	const numRequests = 10
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := client.Get("http://localhost/health")
			if err != nil {
				results <- err
				return
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				results <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
				return
			}

			results <- nil
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Errorf("Concurrent request failed: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Error("Concurrent request timed out")
		}
	}
}
