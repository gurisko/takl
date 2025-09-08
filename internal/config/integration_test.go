package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/takl/takl/internal/paradigm"
	_ "github.com/takl/takl/internal/paradigms/kanban" // Register Kanban
)

func TestConfigIntegrationWithKanban(t *testing.T) {
	// Create a temporary directory for test
	tmpDir, err := os.MkdirTemp("", "takl-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .takl directory
	taklDir := filepath.Join(tmpDir, ".takl")
	if err := os.MkdirAll(taklDir, 0755); err != nil {
		t.Fatalf("Failed to create .takl dir: %v", err)
	}

	// Create test config file
	configContent := `project:
  name: "test-project"

paradigm:
  id: "kanban"
  options:
    wip_limits:
      doing: 5
      review: 3
    block_on_downstream_full: true

ui:
  date_format: "2006-01-02"`

	configPath := filepath.Join(taklDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify config was loaded correctly
	if cfg.Project.Name != "test-project" {
		t.Errorf("Expected project name 'test-project', got '%s'", cfg.Project.Name)
	}

	if cfg.Paradigm.ID != "kanban" {
		t.Errorf("Expected paradigm 'kanban', got '%s'", cfg.Paradigm.ID)
	}

	// Verify paradigm options
	wipLimits, ok := cfg.Paradigm.Options["wip_limits"].(map[string]any)
	if !ok {
		t.Fatalf("Expected wip_limits to be a map")
	}

	if doingLimit, ok := wipLimits["doing"].(int); !ok || doingLimit != 5 {
		t.Errorf("Expected doing limit 5, got %v", wipLimits["doing"])
	}

	// Test paradigm initialization with config options
	kanbanParadigm, err := paradigm.Create("kanban")
	if err != nil {
		t.Fatalf("Failed to create kanban paradigm: %v", err)
	}

	// Initialize with loaded config options
	deps := paradigm.Deps{} // Empty deps for test
	err = kanbanParadigm.Init(context.Background(), deps, cfg.Paradigm.Options)
	if err != nil {
		t.Fatalf("Failed to initialize kanban with config options: %v", err)
	}

	t.Log("✅ Config integration with Kanban paradigm successful")
}

func TestConfigValidationWithInvalidKanban(t *testing.T) {
	// Create a temporary directory for test
	tmpDir, err := os.MkdirTemp("", "takl-config-invalid-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .takl directory
	taklDir := filepath.Join(tmpDir, ".takl")
	if err := os.MkdirAll(taklDir, 0755); err != nil {
		t.Fatalf("Failed to create .takl dir: %v", err)
	}

	// Create test config with invalid Kanban options
	configContent := `project:
  name: "test-project"

paradigm:
  id: "kanban"
  options:
    wip_limits:
      doing: -1  # Invalid negative WIP
    unknown_field: "should fail"`

	configPath := filepath.Join(taklDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config - this should succeed (config loading doesn't validate paradigm options)
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test paradigm initialization with invalid config options - this should fail
	kanbanParadigm, err := paradigm.Create("kanban")
	if err != nil {
		t.Fatalf("Failed to create kanban paradigm: %v", err)
	}

	// Initialize with invalid config options
	deps := paradigm.Deps{} // Empty deps for test
	err = kanbanParadigm.Init(context.Background(), deps, cfg.Paradigm.Options)
	if err == nil {
		t.Fatalf("Expected error when initializing kanban with invalid options, but got none")
	}

	t.Logf("✅ Validation correctly rejected invalid Kanban config: %v", err)
}
