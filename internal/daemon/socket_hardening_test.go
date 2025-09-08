package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureTaklDir(t *testing.T) {
	tests := []struct {
		name       string
		socketPath string
	}{
		{
			name:       "basic_path",
			socketPath: "/tmp/test-takl/daemon.sock",
		},
		{
			name:       "nested_path",
			socketPath: "/tmp/test-takl/sub/daemon.sock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before test
			dir := filepath.Dir(tt.socketPath)
			os.RemoveAll(dir)

			// Test directory creation with secure permissions
			err := ensureTaklDir(tt.socketPath)
			if err != nil {
				t.Fatalf("ensureTaklDir() failed: %v", err)
			}

			// Verify directory exists
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				t.Errorf("Directory %s was not created", dir)
			}

			// Verify permissions are 0700
			info, err := os.Stat(dir)
			if err != nil {
				t.Fatalf("Failed to stat directory: %v", err)
			}

			mode := info.Mode()
			expectedMode := os.FileMode(0o700)
			if mode.Perm() != expectedMode {
				t.Errorf("Directory permissions = %o, expected %o", mode.Perm(), expectedMode)
			}

			// Clean up after test
			os.RemoveAll(dir)
		})
	}
}

func TestEnsureTaklDir_ExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "existing")
	socketPath := filepath.Join(testDir, "daemon.sock")

	// Create directory with wrong permissions initially
	err := os.MkdirAll(testDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Verify it has wrong permissions
	info, err := os.Stat(testDir)
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}
	if info.Mode().Perm() == 0o700 {
		t.Fatal("Test setup failed: directory already has correct permissions")
	}

	// Run ensureTaklDir to fix permissions
	err = ensureTaklDir(socketPath)
	if err != nil {
		t.Fatalf("ensureTaklDir() failed: %v", err)
	}

	// Verify permissions are now fixed
	info, err = os.Stat(testDir)
	if err != nil {
		t.Fatalf("Failed to stat directory after fix: %v", err)
	}

	expectedMode := os.FileMode(0o700)
	if info.Mode().Perm() != expectedMode {
		t.Errorf("Directory permissions = %o, expected %o", info.Mode().Perm(), expectedMode)
	}
}
