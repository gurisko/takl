package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takl/takl/internal/config"
)

func TestInitRoundTrip(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "init-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Save and restore original repoPath
	oldRepoPath := repoPath
	defer func() { repoPath = oldRepoPath }()
	repoPath = tempDir

	// Run init command
	cmd := initCmd
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("Init command failed: %v", err)
	}

	// Determine expected paths based on detected mode
	detectedMode, err := config.DetectMode(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	var configPath string
	var issuesDir string

	if detectedMode == config.ModeEmbedded {
		taklDir := filepath.Join(tempDir, ".takl")
		if _, err := os.Stat(taklDir); os.IsNotExist(err) {
			t.Error(".takl directory was not created")
		}
		issuesDir = filepath.Join(taklDir, "issues")
		configPath = filepath.Join(taklDir, "config.yaml")
	} else {
		// Standalone mode
		issuesDir = filepath.Join(tempDir, ".issues")
		configPath = filepath.Join(tempDir, "config.yaml")
	}

	if _, err := os.Stat(issuesDir); os.IsNotExist(err) {
		t.Errorf("Issues directory was not created at %s", issuesDir)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file was not created at %s", configPath)
	}

	// Load the config using the strict loader and verify fields
	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("Failed to load generated config: %v", err)
	}

	// Debug output
	t.Logf("Loaded config: %+v", cfg)
	t.Logf("Paradigm options: %+v", cfg.Paradigm.Options)

	// Verify modern config structure
	if cfg.Project.Name == "" {
		t.Error("Project name should be set")
	}

	if cfg.Paradigm.ID != "kanban" {
		t.Errorf("Expected paradigm to be kanban, got %s", cfg.Paradigm.ID)
	}

	// Verify WIP limits are set
	if wipLimits, ok := cfg.Paradigm.Options["wip_limits"].(map[string]interface{}); ok {
		if doing, ok := wipLimits["doing"].(int); !ok || doing != 3 {
			t.Errorf("Expected WIP limit for doing to be 3, got %v", wipLimits["doing"])
		}
		if review, ok := wipLimits["review"].(int); !ok || review != 2 {
			t.Errorf("Expected WIP limit for review to be 2, got %v", wipLimits["review"])
		}
	} else {
		t.Error("WIP limits not configured correctly")
	}

	if cfg.UI.DateFormat != "2006-01-02 15:04" {
		t.Errorf("Expected date format to be '2006-01-02 15:04', got '%s'", cfg.UI.DateFormat)
	}

	if !cfg.Git.AutoCommit {
		t.Error("Git auto-commit should be enabled by default")
	}

	// Verify legacy fields are still present for compatibility
	if cfg.Mode != detectedMode {
		t.Errorf("Expected mode to be %s, got %s", detectedMode, cfg.Mode)
	}

	if cfg.WebPort != 3000 {
		t.Errorf("Expected web port to be 3000, got %d", cfg.WebPort)
	}

	// Read the actual file content to verify structure
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)

	// Verify the modern structure is present
	requiredSections := []string{
		"project:",
		"paradigm:",
		"wip_limits:",
		"ui:",
		"date_format:",
		"git:",
		"auto_commit:",
	}

	for _, section := range requiredSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("Config file missing required section: %s", section)
		}
	}

	// Verify comments are included
	if !strings.Contains(contentStr, "# TAKL Configuration") {
		t.Error("Config file missing header comment")
	}

	if !strings.Contains(contentStr, "# Legacy fields") {
		t.Error("Config file missing legacy fields comment")
	}

	t.Log("✅ Init round-trip test successful")
	t.Logf("Generated project name: %s", cfg.Project.Name)
	t.Logf("Generated config file:\n%s", contentStr)
}

func TestInitDoesNotOverwriteExistingConfig(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "init-no-overwrite-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create .takl directory
	taklDir := filepath.Join(tempDir, ".takl")
	if err := os.MkdirAll(taklDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create existing config file
	configPath := filepath.Join(taklDir, "config.yaml")
	existingContent := "# Existing config\nproject:\n  name: Existing Project\n"
	if err := os.WriteFile(configPath, []byte(existingContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Save and restore original repoPath
	oldRepoPath := repoPath
	defer func() { repoPath = oldRepoPath }()
	repoPath = tempDir

	// Run init command
	cmd := initCmd
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("Init command failed: %v", err)
	}

	// Verify existing config was not overwritten
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != existingContent {
		t.Error("Init command overwrote existing config file")
	}

	t.Log("✅ Init correctly preserves existing config")
}
