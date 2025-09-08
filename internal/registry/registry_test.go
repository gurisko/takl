package registry_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/takl/takl/internal/registry"
)

func TestRegistryCreation(t *testing.T) {
	t.Parallel()

	// Create temporary directory
	tmpDir := t.TempDir()

	// Mock home directory with cleanup
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })
	os.Setenv("HOME", tmpDir)

	reg, err := registry.New()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	if reg == nil {
		t.Fatal("Registry is nil")
	}

	// Check that .takl directory was created
	taklDir := filepath.Join(tmpDir, ".takl")
	if _, err := os.Stat(taklDir); os.IsNotExist(err) {
		t.Fatal("Expected .takl directory to be created")
	}

	// Check that projects directory was created
	projectsDir := filepath.Join(taklDir, "projects")
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		t.Fatal("Expected projects directory to be created")
	}
}

func TestProjectRegistration(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		description string
		wantMode    string
		wantErr     bool
	}{
		{
			name:        "valid project",
			projectName: "Test Project",
			description: "A test project",
			wantMode:    "embedded",
			wantErr:     false,
		},
		{
			name:        "empty description",
			projectName: "Simple Project",
			description: "",
			wantMode:    "embedded",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()

			// Mock home directory with cleanup
			oldHome := os.Getenv("HOME")
			t.Cleanup(func() { os.Setenv("HOME", oldHome) })
			os.Setenv("HOME", tmpDir)

			// Create test project structure
			projectDir := filepath.Join(tmpDir, "test-project")
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				t.Fatalf("Failed to create project dir: %v", err)
			}
			if err := os.MkdirAll(filepath.Join(projectDir, ".takl", "issues"), 0755); err != nil {
				t.Fatalf("Failed to create issues dir: %v", err)
			}

			reg, err := registry.New()
			if err != nil {
				t.Fatalf("Failed to create registry: %v", err)
			}

			// Test project registration
			project, err := reg.RegisterProject(projectDir, tt.projectName, tt.description)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RegisterProject() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if project == nil {
				t.Fatal("Project is nil")
			}

			if project.Name != tt.projectName {
				t.Errorf("Expected name '%s', got '%s'", tt.projectName, project.Name)
			}

			if project.Mode != tt.wantMode {
				t.Errorf("Expected mode '%s', got '%s'", tt.wantMode, project.Mode)
			}

			if project.Description != tt.description {
				t.Errorf("Expected description '%s', got '%s'", tt.description, project.Description)
			}
		})
	}
}

func TestProjectLookup(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create test project structure
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".takl", "issues"), 0755); err != nil {
		t.Fatalf("Failed to create issues dir: %v", err)
	}

	reg, err := registry.New()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register project
	project, err := reg.RegisterProject(projectDir, "Test Project", "")
	if err != nil {
		t.Fatalf("Failed to register project: %v", err)
	}

	// Test lookup by ID
	found, exists := reg.GetProject(project.ID)
	if !exists {
		t.Fatal("Project should exist when looked up by ID")
	}

	if found.ID != project.ID {
		t.Errorf("Expected ID '%s', got '%s'", project.ID, found.ID)
	}

	// Test lookup by path
	foundByPath, exists := reg.FindProjectByPath(projectDir)
	if !exists {
		t.Fatal("Project should exist when looked up by path")
	}

	if foundByPath.ID != project.ID {
		t.Errorf("Expected ID '%s', got '%s'", project.ID, foundByPath.ID)
	}
}

func TestProjectUnregistration(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create test project structure
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".takl", "issues"), 0755); err != nil {
		t.Fatalf("Failed to create issues dir: %v", err)
	}

	reg, err := registry.New()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register project
	project, err := reg.RegisterProject(projectDir, "Test Project", "")
	if err != nil {
		t.Fatalf("Failed to register project: %v", err)
	}

	// Unregister project
	err = reg.UnregisterProject(project.ID)
	if err != nil {
		t.Fatalf("Failed to unregister project: %v", err)
	}

	// Verify it's gone
	_, exists := reg.GetProject(project.ID)
	if exists {
		t.Fatal("Project should not exist after unregistration")
	}
}

func TestHealthCheck(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create healthy project
	healthyDir := filepath.Join(tmpDir, "healthy-project")
	if err := os.MkdirAll(healthyDir, 0755); err != nil {
		t.Fatalf("Failed to create healthy dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(healthyDir, ".takl", "issues"), 0755); err != nil {
		t.Fatalf("Failed to create issues dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(healthyDir, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create git dir: %v", err)
	}

	// Create unhealthy project (missing issues dir)
	unhealthyDir := filepath.Join(tmpDir, "unhealthy-project")
	if err := os.MkdirAll(unhealthyDir, 0755); err != nil {
		t.Fatalf("Failed to create unhealthy dir: %v", err)
	}

	reg, err := registry.New()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register healthy project
	healthy, err := reg.RegisterProject(healthyDir, "Healthy Project", "")
	if err != nil {
		t.Fatalf("Failed to register healthy project: %v", err)
	}

	// Now remove the issues directory to make healthy project unhealthy
	os.RemoveAll(filepath.Join(healthyDir, ".takl", "issues"))

	// Run health check
	_, unhealthyProjects, err := reg.HealthCheck()
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	// Since we removed issues dir, should be 0 healthy, 1 unhealthy
	if len(unhealthyProjects) == 0 {
		t.Error("Expected at least one unhealthy project")
	}

	// Find our project in the unhealthy list
	found := false
	for _, p := range unhealthyProjects {
		if p.ID == healthy.ID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find our project in unhealthy list")
	}
}

func TestCleanupStaleProjects(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create test project structure
	projectDir := filepath.Join(tmpDir, "stale-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".takl", "issues"), 0755); err != nil {
		t.Fatalf("Failed to create issues dir: %v", err)
	}

	reg, err := registry.New()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register project
	project, err := reg.RegisterProject(projectDir, "Stale Project", "")
	if err != nil {
		t.Fatalf("Failed to register project: %v", err)
	}

	// Manually set old timestamps to make it stale
	project.LastAccess = time.Now().Add(-35 * 24 * time.Hour) // 35 days ago
	project.LastSeen = time.Now().Add(-35 * 24 * time.Hour)
	project.Active = false

	// Run cleanup for projects older than 30 days
	removed, err := reg.CleanupStaleProjects(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	if len(removed) != 1 {
		t.Errorf("Expected 1 removed project, got %d", len(removed))
	}

	if len(removed) > 0 && removed[0].ID != project.ID {
		t.Errorf("Expected removed project ID '%s', got '%s'", project.ID, removed[0].ID)
	}

	// Verify project is gone
	_, exists := reg.GetProject(project.ID)
	if exists {
		t.Fatal("Project should not exist after cleanup")
	}
}
