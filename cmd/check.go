package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/shared"
	"github.com/takl/takl/internal/validation"
)

var (
	checkGlobal bool
	checkFix    bool
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate issue files",
	Long: `Validate all issue files in the current project or across all registered projects.

This command checks:
- YAML frontmatter format and syntax
- Required fields (id, type, title, status)
- File naming matches ID
- Valid issue types (bug, feature, task, epic)
- Valid priority values
- Proper timestamp formats

Examples:
  takl check                    # Check current project
  takl check --global          # Check all registered projects
  takl check --fix            # Auto-fix common issues (experimental)`,
	RunE: checkIssues,
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().BoolVar(&checkGlobal, "global", false, "Check all registered projects")
	checkCmd.Flags().BoolVar(&checkFix, "fix", false, "Auto-fix common issues (experimental)")
}

// ValidationError represents a validation issue found in a file
type ValidationError struct {
	File    string
	Line    int
	Field   string
	Message string
	Fixable bool
}

// ValidationResult holds all validation results for a project
type ValidationResult struct {
	ProjectID   string
	ProjectName string
	ProjectPath string // Added for proper relative path computation
	TotalFiles  int
	ValidFiles  int
	Errors      []ValidationError
	Warnings    []ValidationError
}

func checkIssues(cmd *cobra.Command, args []string) error {
	if checkGlobal {
		return checkAllProjects()
	}

	ctx, err := RequireProjectContext()
	if err != nil {
		return err
	}

	result := validateProject(ctx.GetProjectPath(), ctx.GetProjectID(), ctx.GetProjectInfo())
	printValidationResult(result)

	if len(result.Errors) > 0 {
		return fmt.Errorf("validation failed: %d error(s) found", len(result.Errors))
	}

	return nil
}

func checkAllProjects() error {
	client := sdkClient()

	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No registered projects found")
		return nil
	}

	totalErrors := 0
	for _, project := range projects {
		fmt.Printf("\n=== Checking project: %s (%s) ===\n", project.Name, project.ID)
		result := validateProject(project.Path, project.ID, project.Name)
		printValidationResult(result)
		totalErrors += len(result.Errors)
	}

	if totalErrors > 0 {
		return fmt.Errorf("validation failed: %d total error(s) across all projects", totalErrors)
	}

	fmt.Printf("\n✅ All projects validated successfully\n")
	return nil
}

func validateProject(projectPath, projectID, projectName string) ValidationResult {
	result := ValidationResult{
		ProjectID:   projectID,
		ProjectName: projectName,
		ProjectPath: projectPath, // Set project path for proper relative path computation
		Errors:      []ValidationError{},
		Warnings:    []ValidationError{},
	}

	// Determine issues directory based on mode
	issuesDir := filepath.Join(projectPath, ".takl", "issues")
	if _, err := os.Stat(issuesDir); os.IsNotExist(err) {
		// Try standalone mode
		issuesDir = filepath.Join(projectPath, ".issues")
		if _, err := os.Stat(issuesDir); os.IsNotExist(err) {
			result.Errors = append(result.Errors, ValidationError{
				File:    projectPath,
				Message: "No issues directory found (.takl/issues or .issues)",
			})
			return result
		}
	}

	// Walk through all markdown files
	err := filepath.WalkDir(issuesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		result.TotalFiles++
		validateFile(path, &result)

		return nil
	})

	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			File:    issuesDir,
			Message: fmt.Sprintf("Failed to scan directory: %v", err),
		})
	}

	return result
}

func validateFile(filePath string, result *ValidationResult) {
	relPath, _ := filepath.Rel(result.ProjectPath, filePath)
	fileHadError := false

	// Load and parse issue
	issue, err := shared.LoadIssueFromFile(filePath)
	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			File:    relPath,
			Message: fmt.Sprintf("Failed to load issue: %v", err),
		})
		return // File load error means it's invalid
	}

	// Use centralized validation
	validator := validation.NewValidator(nil) // No workflow for basic validation
	if err := validator.ValidateIssue(issue); err != nil {
		// Parse validation error and categorize
		errorMsg := err.Error()
		field := extractFieldFromError(errorMsg)

		// Determine if this is fixable
		fixable := strings.Contains(errorMsg, "status") && issue.Status == ""

		result.Errors = append(result.Errors, ValidationError{
			File:    relPath,
			Field:   field,
			Message: errorMsg,
			Fixable: fixable,
		})
		fileHadError = true
	}

	// Additional validation specific to file checking
	// Validate file naming convention
	actual := strings.ToLower(filepath.Base(filePath))
	expected := strings.ToLower(issue.ID) + "-" // enforce the hyphen after ID
	if issue.ID != "" && !strings.HasPrefix(actual, expected) {
		result.Warnings = append(result.Warnings, ValidationError{
			File:    relPath,
			Message: fmt.Sprintf("File name should start with %q (got %q)", expected, actual),
			Fixable: true,
		})
	}

	// Check email format using shared validation (from create.go)
	if issue.Assignee != "" && !isValidEmail(issue.Assignee) {
		result.Warnings = append(result.Warnings, ValidationError{
			File:    relPath,
			Field:   "assignee",
			Message: fmt.Sprintf("Invalid email format for assignee: %s", issue.Assignee),
			Fixable: false,
		})
	}

	// Track valid files per-file, not per-project
	if !fileHadError {
		result.ValidFiles++
	}

	// Apply fixes if requested
	if checkFix && hasFixableIssues(result, relPath) {
		applyFixes(filePath, issue)
	}
}

func hasFixableIssues(result *ValidationResult, relPath string) bool {
	for _, err := range append(result.Errors, result.Warnings...) {
		if err.File == relPath && err.Fixable {
			return true
		}
	}
	return false
}

func applyFixes(filePath string, issue *domain.Issue) {
	changed := false

	// 1) Default status
	if issue.Status == "" {
		issue.Status = "open"
		changed = true
		if verbose {
			fmt.Printf("Fixed: Set default status 'open' in %s\n", filePath)
		}
	}

	// 2) Rename file to match ID prefix (if needed)
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	expectedBase := strings.ToLower(issue.ID) + "-" + slugify(issue.Title) + ".md"
	if issue.ID != "" && base != expectedBase {
		newPath := filepath.Join(dir, expectedBase)

		// Handle collision by adding numeric suffix
		originalNewPath := newPath
		suffix := 1
		for fileExists(newPath) && newPath != filePath {
			ext := filepath.Ext(originalNewPath)
			nameWithoutExt := strings.TrimSuffix(originalNewPath, ext)
			suffix++
			newPath = fmt.Sprintf("%s-%d%s", nameWithoutExt, suffix, ext)
		}

		// Only rename if target is different from source
		if newPath != filePath {
			if err := os.Rename(filePath, newPath); err == nil {
				if verbose {
					fmt.Printf("Fixed: Renamed %s to %s\n", base, filepath.Base(newPath))
				}
				issue.FilePath = newPath // Update issue's file path for saving
			} else if verbose {
				fmt.Printf("Warning: Could not rename %s: %v\n", base, err)
			}
		}
	}

	// 3) Write back frontmatter if we changed anything
	if changed {
		if err := shared.SaveIssueToFile(issue); err != nil {
			if verbose {
				fmt.Printf("Warning: Could not save changes to %s: %v\n", filePath, err)
			}
		}
	}
}

// extractFieldFromError extracts the field name from a validation error message
func extractFieldFromError(errorMsg string) string {
	if strings.Contains(errorMsg, "ID") || strings.Contains(errorMsg, "id") {
		return "id"
	}
	if strings.Contains(errorMsg, "title") {
		return "title"
	}
	if strings.Contains(errorMsg, "type") {
		return "type"
	}
	if strings.Contains(errorMsg, "status") {
		return "status"
	}
	if strings.Contains(errorMsg, "priority") {
		return "priority"
	}
	if strings.Contains(errorMsg, "assignee") {
		return "assignee"
	}
	return ""
}

// isValidEmail is shared from create.go

func slugify(text string) string {
	// Convert to lowercase and replace spaces with hyphens
	slug := strings.ToLower(text)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	slug = reg.ReplaceAllString(slug, "")
	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
	}
	return slug
}

func printValidationResult(result ValidationResult) {
	fmt.Printf("\nValidation Results for %s:\n", result.ProjectName)
	fmt.Printf("  Total files: %d\n", result.TotalFiles)
	fmt.Printf("  Valid files: %d\n", result.ValidFiles)

	if len(result.Errors) > 0 {
		fmt.Printf("\n❌ Errors (%d):\n", len(result.Errors))
		for _, err := range result.Errors {
			if err.Field != "" {
				fmt.Printf("  - %s [%s]: %s\n", err.File, err.Field, err.Message)
			} else {
				fmt.Printf("  - %s: %s\n", err.File, err.Message)
			}
			if err.Fixable && checkFix {
				fmt.Printf("    → Will be fixed\n")
			} else if err.Fixable {
				fmt.Printf("    → Fixable with --fix\n")
			}
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Printf("\n⚠️  Warnings (%d):\n", len(result.Warnings))
		for _, warn := range result.Warnings {
			if warn.Field != "" {
				fmt.Printf("  - %s [%s]: %s\n", warn.File, warn.Field, warn.Message)
			} else {
				fmt.Printf("  - %s: %s\n", warn.File, warn.Message)
			}
			if warn.Fixable && checkFix {
				fmt.Printf("    → Will be fixed\n")
			} else if warn.Fixable {
				fmt.Printf("    → Fixable with --fix\n")
			}
		}
	}

	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		fmt.Printf("\n✅ All files validated successfully!\n")
	}
}

// fileExists checks if a file exists and is not a directory
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
