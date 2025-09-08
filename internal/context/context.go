package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/takl/takl/internal/registry"
)

type ProjectContext struct {
	Project     *registry.Project
	Registry    *registry.Registry
	WorkingDir  string
	IsInProject bool
}

func DetectContext(workingDir string) (*ProjectContext, error) {
	if workingDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		workingDir = wd
	}

	registry, err := registry.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize registry: %w", err)
	}

	ctx := &ProjectContext{
		Registry:    registry,
		WorkingDir:  workingDir,
		IsInProject: false,
	}

	// Try to find project by exact path match first
	if project, found := registry.FindProjectByPath(workingDir); found {
		ctx.Project = project
		ctx.IsInProject = true
		return ctx, nil
	}

	// If not exact match, check if we're inside a registered project
	project, found := ctx.findContainingProject(workingDir)
	if found {
		ctx.Project = project
		ctx.IsInProject = true
	}

	return ctx, nil
}

// findContainingProject checks if the current directory is within any registered project
func (ctx *ProjectContext) findContainingProject(dir string) (*registry.Project, bool) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, false
	}

	projects := ctx.Registry.ListProjects()

	// Find the deepest (most specific) project that contains this directory
	var bestMatch *registry.Project
	var bestMatchDepth int

	for _, project := range projects {
		projectPath, err := filepath.Abs(project.Path)
		if err != nil {
			continue
		}

		// Check if absDir is within projectPath
		if strings.HasPrefix(absDir, projectPath) {
			// Ensure it's actually a subdirectory, not just a prefix match
			if absDir == projectPath || strings.HasPrefix(absDir, projectPath+string(filepath.Separator)) {
				depth := strings.Count(projectPath, string(filepath.Separator))
				if bestMatch == nil || depth > bestMatchDepth {
					bestMatch = project
					bestMatchDepth = depth
				}
			}
		}
	}

	return bestMatch, bestMatch != nil
}

// GetProjectPath returns the root path of the current project
func (ctx *ProjectContext) GetProjectPath() string {
	if ctx.Project == nil {
		return ctx.WorkingDir
	}
	return ctx.Project.Path
}

// GetIssuesDir returns the issues directory for the current project
func (ctx *ProjectContext) GetIssuesDir() string {
	if ctx.Project == nil {
		// Fallback to legacy behavior
		if _, err := os.Stat(filepath.Join(ctx.WorkingDir, ".takl", "issues")); err == nil {
			return filepath.Join(ctx.WorkingDir, ".takl", "issues")
		}
		if _, err := os.Stat(filepath.Join(ctx.WorkingDir, ".issues")); err == nil {
			return filepath.Join(ctx.WorkingDir, ".issues")
		}
		return ""
	}
	return ctx.Project.IssuesDir
}

// GetProjectMode returns the project mode (embedded/standalone)
func (ctx *ProjectContext) GetProjectMode() string {
	if ctx.Project == nil {
		return "unknown"
	}
	return ctx.Project.Mode
}

// UpdateLastAccess updates the last access time for the current project
func (ctx *ProjectContext) UpdateLastAccess() error {
	if ctx.Project == nil {
		return nil // No project to update
	}
	return ctx.Registry.UpdateLastAccess(ctx.Project.ID)
}

// RequiresProject returns an error if not in a project context
func (ctx *ProjectContext) RequiresProject() error {
	if !ctx.IsInProject {
		return fmt.Errorf("not in a TAKL project. Run 'takl register' to register this directory or navigate to a registered project")
	}
	return nil
}

// GetProjectID returns the project ID
func (ctx *ProjectContext) GetProjectID() string {
	if ctx.Project == nil {
		return ""
	}
	return ctx.Project.ID
}

// GetProjectInfo returns formatted project information
func (ctx *ProjectContext) GetProjectInfo() string {
	if !ctx.IsInProject {
		return "No project context"
	}

	return fmt.Sprintf("Project: %s (%s) - %s mode",
		ctx.Project.Name,
		ctx.Project.ID,
		ctx.Project.Mode)
}
