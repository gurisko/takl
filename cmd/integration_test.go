package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/shared"
)

func TestListSearchIntegration(t *testing.T) {
	// Create temporary project for testing
	tempDir, err := os.MkdirTemp("", "list-search-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo (required for some operations)
	if err := initRealGitRepo(tempDir); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Save and restore original repoPath
	oldRepoPath := repoPath
	defer func() { repoPath = oldRepoPath }()
	repoPath = tempDir

	// Initialize TAKL project
	if err := initCmd.RunE(initCmd, []string{}); err != nil {
		t.Fatalf("Failed to init TAKL project: %v", err)
	}

	// Create test issues with correct embedded mode paths
	testIssues := []*domain.Issue{
		{
			ID:       "ISS-001",
			Type:     "bug",
			Title:    "Login button not working",
			Status:   "open",
			Priority: "high",
			Content:  "The login button doesn't respond to clicks on mobile",
			FilePath: filepath.Join(tempDir, ".takl", "issues", "bug", "iss-001-login-button-not-working.md"),
		},
		{
			ID:       "ISS-002",
			Type:     "feature",
			Title:    "Add dark mode",
			Status:   "in_progress",
			Priority: "medium",
			Content:  "Users want a dark mode theme option",
			FilePath: filepath.Join(tempDir, ".takl", "issues", "feature", "iss-002-add-dark-mode.md"),
		},
		{
			ID:       "ISS-003",
			Type:     "task",
			Title:    "Update API documentation",
			Status:   "done",
			Priority: "low",
			Content:  "API docs need to be updated for the new endpoints",
			FilePath: filepath.Join(tempDir, ".takl", "issues", "task", "iss-003-update-api-documentation.md"),
		},
	}

	// Create issue files
	for _, issue := range testIssues {
		if err := os.MkdirAll(filepath.Dir(issue.FilePath), 0755); err != nil {
			t.Fatal(err)
		}

		// Normalize and save the issue
		issue.Created = time.Now()
		issue.Updated = time.Now()
		issue.Normalize()
		if err := shared.SaveIssueToFile(issue); err != nil {
			t.Fatalf("Failed to save test issue %s: %v", issue.ID, err)
		}
		t.Logf("Created issue file: %s", issue.FilePath)
	}

	// Debug: list what was actually created
	if files, err := os.ReadDir(tempDir); err == nil {
		t.Logf("Files in tempDir: %v", files)
		for _, file := range files {
			if file.IsDir() {
				if subfiles, err := os.ReadDir(filepath.Join(tempDir, file.Name())); err == nil {
					t.Logf("Files in %s: %v", file.Name(), subfiles)
					// Look deeper into issues directory
					if file.Name() == ".takl" {
						for _, subfile := range subfiles {
							if subfile.Name() == "issues" {
								issuesPath := filepath.Join(tempDir, ".takl", "issues")
								if typesDirs, err := os.ReadDir(issuesPath); err == nil {
									t.Logf("Issue types: %v", typesDirs)
									for _, typeDir := range typesDirs {
										if typeDir.IsDir() {
											typePath := filepath.Join(issuesPath, typeDir.Name())
											if issueFiles, err := os.ReadDir(typePath); err == nil {
												t.Logf("Issues in %s: %v", typeDir.Name(), issueFiles)
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Build takl binary for testing
	buildCmd := exec.Command("go", "build", "-o", "takl-test", ".")
	buildCmd.Dir = "/home/ubuntu/takl"
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build takl: %v", err)
	}
	defer os.Remove("/home/ubuntu/takl/takl-test")

	taklPath := "/home/ubuntu/takl/takl-test"

	t.Run("list_all_issues", func(t *testing.T) {
		// Register project first
		registerCmd := exec.Command(taklPath, "register", tempDir, "Test Project")
		registerCmd.Dir = tempDir
		registerOutput, err := registerCmd.CombinedOutput()
		if err != nil {
			t.Logf("Register failed: %v, output: %s", err, string(registerOutput))
		} else {
			t.Logf("Register output: %s", string(registerOutput))
		}

		// Test list all issues
		cmd := exec.Command(taklPath, "list")
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("List command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		t.Logf("List all output: %s", outputStr)

		// Verify all issues are listed
		expectedIssues := []string{"ISS-001", "ISS-002", "ISS-003"}
		for _, issueID := range expectedIssues {
			if !strings.Contains(outputStr, issueID) {
				t.Errorf("Issue %s not found in list output", issueID)
			}
		}

		// Verify count is shown
		if !strings.Contains(outputStr, "Found 3 issue(s)") {
			t.Error("Expected to find 3 issues")
		}
	})

	t.Run("list_filtered_by_type", func(t *testing.T) {
		// Test filtering by type
		cmd := exec.Command(taklPath, "list", "--type=bug")
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("List filtered command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		t.Logf("List bug output: %s", outputStr)

		// Should include bug issue
		if !strings.Contains(outputStr, "ISS-001") {
			t.Error("Bug issue ISS-001 not found in filtered output")
		}

		// Should not include feature or task issues
		if strings.Contains(outputStr, "ISS-002") || strings.Contains(outputStr, "ISS-003") {
			t.Error("Non-bug issues found in bug-filtered output")
		}
	})

	t.Run("list_filtered_by_status", func(t *testing.T) {
		// Test filtering by status (test the normalized in_progress)
		cmd := exec.Command(taklPath, "list", "--status=in_progress")
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("List status filtered command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		t.Logf("List in-progress output: %s", outputStr)

		// Should include in_progress issue
		if !strings.Contains(outputStr, "ISS-002") {
			t.Error("In-progress issue ISS-002 not found in status-filtered output")
		}
	})

	t.Run("search_issues", func(t *testing.T) {
		// Test search functionality
		cmd := exec.Command(taklPath, "search", "login")
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Search command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		t.Logf("Search login output: %s", outputStr)

		// Should find the login-related issue
		if !strings.Contains(outputStr, "ISS-001") {
			t.Error("Login-related issue ISS-001 not found in search results")
		}

		if !strings.Contains(outputStr, "Login button") {
			t.Error("Issue title not found in search results")
		}
	})

	t.Run("search_no_results", func(t *testing.T) {
		// Test search with no results
		cmd := exec.Command(taklPath, "search", "nonexistent")
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Search no results command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		t.Logf("Search no results output: %s", outputStr)

		// Should show no results message
		if !strings.Contains(outputStr, "No issues found") {
			t.Error("Expected 'No issues found' message")
		}
	})
}

func initRealGitRepo(dir string) error {
	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}

	// Set git config to avoid errors
	configCmds := [][]string{
		{"git", "config", "user.name", "Test User"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "init.defaultBranch", "main"},
	}

	for _, args := range configCmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func TestServeIntegration(t *testing.T) {
	// Create temporary project for testing
	tempDir, err := os.MkdirTemp("", "serve-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	if err := initRealGitRepo(tempDir); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Save and restore original repoPath
	oldRepoPath := repoPath
	defer func() { repoPath = oldRepoPath }()
	repoPath = tempDir

	// Initialize TAKL project
	if err := initCmd.RunE(initCmd, []string{}); err != nil {
		t.Fatalf("Failed to init TAKL project: %v", err)
	}

	// Create a test issue to ensure we have data
	testIssue := &domain.Issue{
		ID:       "ISS-001",
		Type:     "bug",
		Title:    "Test serve functionality",
		Status:   "open",
		Priority: "medium",
		Content:  "Test issue for serve integration test",
		Created:  time.Now(),
		Updated:  time.Now(),
		FilePath: filepath.Join(tempDir, ".takl", "issues", "bug", "iss-001-test-serve-functionality.md"),
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(testIssue.FilePath), 0755); err != nil {
		t.Fatalf("Failed to create issues directory: %v", err)
	}

	if err := shared.SaveIssueToFile(testIssue); err != nil {
		t.Fatalf("Failed to save test issue: %v", err)
	}

	// Stop any existing daemon first
	stopCmd := exec.Command(getTaklPath(), "daemon", "stop")
	stopCmd.Dir = tempDir
	_ = stopCmd.Run() // Ignore error - might not be running

	// Start daemon
	daemonCmd := exec.Command(getTaklPath(), "daemon", "start")
	daemonCmd.Dir = tempDir
	if err := daemonCmd.Start(); err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}

	// Ensure daemon is cleaned up
	defer func() {
		stopCmd := exec.Command(getTaklPath(), "daemon", "stop")
		stopCmd.Dir = tempDir
		_ = stopCmd.Run() // Ignore error
		if daemonCmd.Process != nil {
			_ = daemonCmd.Process.Kill()
		}
	}()

	// Give daemon time to start
	time.Sleep(100 * time.Millisecond)

	// Find available port
	port, err := findAvailablePort()
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}

	// Start serve command in background
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serveCmd := exec.CommandContext(ctx, getTaklPath(), "serve", "--port", strconv.Itoa(port))
	serveCmd.Dir = tempDir

	if err := serveCmd.Start(); err != nil {
		t.Fatalf("Failed to start serve command: %v", err)
	}

	// Ensure serve is cleaned up
	defer func() {
		if serveCmd.Process != nil {
			_ = serveCmd.Process.Kill()
		}
	}()

	// Wait for server to start
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	if err := waitForServer(serverURL, 5*time.Second); err != nil {
		t.Fatalf("Server did not start: %v", err)
	}

	t.Run("health_endpoint", func(t *testing.T) {
		resp, err := http.Get(serverURL + "/api/health")
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Health check returned status %d, expected %d", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("proxy_health", func(t *testing.T) {
		resp, err := http.Get(serverURL + "/healthz")
		if err != nil {
			t.Fatalf("Proxy health check failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Proxy health check returned status %d, expected %d", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("spa_fallback", func(t *testing.T) {
		// Test that non-existent routes serve index.html (SPA fallback)
		resp, err := http.Get(serverURL + "/issues")
		if err != nil {
			t.Fatalf("SPA fallback test failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("SPA fallback returned status %d, expected %d", resp.StatusCode, http.StatusOK)
		}

		// Check that it's serving HTML content
		if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "text/html") {
			t.Errorf("SPA fallback returned content-type %s, expected text/html", contentType)
		}
	})

	t.Run("api_proxy", func(t *testing.T) {
		// Test that API requests are properly proxied to daemon
		resp, err := http.Get(serverURL + "/api/registry/projects")
		if err != nil {
			t.Fatalf("API proxy test failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("API proxy returned status %d, expected %d", resp.StatusCode, http.StatusOK)
		}

		// Check that it's serving JSON content
		if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
			t.Errorf("API proxy returned content-type %s, expected application/json", contentType)
		}
	})
}

// Helper functions for serve test

func getTaklPath() string {
	// Get absolute path to takl binary
	taklPath, err := filepath.Abs("../takl")
	if err != nil {
		panic(fmt.Sprintf("Failed to get absolute path: %v", err))
	}

	// Build the takl binary if it doesn't exist
	if _, err := os.Stat(taklPath); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "takl", ".")
		cmd.Dir = ".."
		if err := cmd.Run(); err != nil {
			panic(fmt.Sprintf("Failed to build takl binary: %v", err))
		}
		// Update path after building
		taklPath, _ = filepath.Abs("../takl")
	}

	return taklPath
}

func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

func waitForServer(url string, timeout time.Duration) error {
	client := &http.Client{Timeout: 1 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url + "/healthz")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("server did not become healthy within %v", timeout)
}
