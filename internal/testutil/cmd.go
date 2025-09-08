package testutil

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// CmdExecutor helps test Cobra commands programmatically
type CmdExecutor struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	stdin  *bytes.Buffer
}

// NewCmdExecutor creates a new command executor for testing
func NewCmdExecutor() *CmdExecutor {
	return &CmdExecutor{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		stdin:  &bytes.Buffer{},
	}
}

// SetStdin sets the stdin content for the command
func (e *CmdExecutor) SetStdin(input string) {
	e.stdin.Reset()
	e.stdin.WriteString(input)
}

// ExecuteCommand executes a Cobra command with the given arguments
func (e *CmdExecutor) ExecuteCommand(root *cobra.Command, args ...string) error {
	// Reset buffers
	e.stdout.Reset()
	e.stderr.Reset()

	// Configure command I/O
	root.SetOut(e.stdout)
	root.SetErr(e.stderr)
	root.SetIn(e.stdin)
	root.SetArgs(args)

	// Execute command
	return root.Execute()
}

// Stdout returns the captured stdout
func (e *CmdExecutor) Stdout() string {
	return e.stdout.String()
}

// Stderr returns the captured stderr
func (e *CmdExecutor) Stderr() string {
	return e.stderr.String()
}

// StdoutLines returns stdout split by lines
func (e *CmdExecutor) StdoutLines() []string {
	output := strings.TrimSpace(e.stdout.String())
	if output == "" {
		return []string{}
	}
	return strings.Split(output, "\n")
}

// GoldenTest helps with golden file testing
type GoldenTest struct {
	t       *testing.T
	update  bool
	dataDir string
}

// NewGoldenTest creates a new golden test helper
func NewGoldenTest(t *testing.T, dataDir string) *GoldenTest {
	update := flag.Lookup("test.update") != nil || os.Getenv("UPDATE_GOLDEN") == "true"
	return &GoldenTest{
		t:       t,
		update:  update,
		dataDir: dataDir,
	}
}

// Check compares content with golden file
func (g *GoldenTest) Check(name, content string) {
	goldenPath := g.dataDir + "/" + name + ".golden"

	if g.update {
		if err := os.WriteFile(goldenPath, []byte(content), 0644); err != nil {
			g.t.Fatalf("Failed to update golden file %s: %v", goldenPath, err)
		}
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		g.t.Fatalf("Failed to read golden file %s: %v", goldenPath, err)
	}

	if string(expected) != content {
		g.t.Errorf("Content mismatch for %s:\n--- Expected ---\n%s\n--- Got ---\n%s", name, string(expected), content)
	}
}

// TempRepo creates a temporary git repository for testing
func TempRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Initialize git repo
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to cd to temp dir: %v", err)
	}

	// Create a minimal git repo structure without requiring git binary
	if err := os.MkdirAll(".git", 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	return tmpDir
}

// TempTAKLProject creates a temporary TAKL project for testing
func TempTAKLProject(t *testing.T) string {
	t.Helper()

	tmpDir := TempRepo(t)

	// Create TAKL structure
	if err := os.MkdirAll(".takl/issues/bug", 0755); err != nil {
		t.Fatalf("Failed to create bug issues dir: %v", err)
	}
	if err := os.MkdirAll(".takl/issues/feature", 0755); err != nil {
		t.Fatalf("Failed to create feature issues dir: %v", err)
	}
	if err := os.MkdirAll(".takl/issues/task", 0755); err != nil {
		t.Fatalf("Failed to create task issues dir: %v", err)
	}
	if err := os.MkdirAll(".takl/issues/epic", 0755); err != nil {
		t.Fatalf("Failed to create epic issues dir: %v", err)
	}

	// Create basic config
	configContent := `mode: embedded
issues_dir: .takl/issues
`
	if err := os.WriteFile(".takl/config.yaml", []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	return tmpDir
}
