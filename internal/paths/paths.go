package paths

import (
	"os"
	"path/filepath"
)

func DefaultRuntimeDir() string {
	if x := os.Getenv("XDG_RUNTIME_DIR"); x != "" {
		return filepath.Join(x, "takl")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".takl")
}

func DefaultStateDir() string {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return filepath.Join(x, "takl")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "takl")
}

func DefaultSocketPath() string   { return filepath.Join(DefaultRuntimeDir(), "daemon.sock") }
func DefaultPIDPath() string      { return filepath.Join(DefaultRuntimeDir(), "daemon.pid") }
func DefaultRegistryPath() string { return filepath.Join(DefaultStateDir(), "projects.yaml") }
