package watcher

import (
	"path/filepath"
	"strings"
)

// ignoreRules defines which files/directories to ignore in filesystem watching
type ignoreRules struct {
	// File suffixes to ignore
	ignoreSuffixes []string

	// Exact filenames to ignore
	ignoreFiles []string

	// Directory names to ignore
	ignoreDirs []string
}

// newIgnoreRules creates default ignore rules for common temp files, VCS, and atomic saves
func newIgnoreRules() *ignoreRules {
	return &ignoreRules{
		ignoreSuffixes: []string{
			// Vim family
			"~",    // Vim/Emacs backup files
			".swp", // Vim swap files
			".swo", // Vim swap files
			".swn", // Vim swap files
			".swx", // Vim additional swap

			// Editor temporary files
			".tmp",  // Generic temporary files
			".temp", // Alternative temporary extension
			".bak",  // Backup files
			".old",  // Old file backups
			".orig", // Git merge conflict backups
			".lock", // Lock files
			".part", // Partial downloads

			// Emacs family
			"#",  // Emacs auto-save files (file#)
			".#", // Emacs lock files prefix

			// IDE and editor specific
			".crdownload", // Chrome downloads
			".download",   // Firefox downloads
			".partial",    // Partial file transfers
		},
		ignoreFiles: []string{
			// System metadata
			".DS_Store",   // macOS metadata
			"desktop.ini", // Windows metadata
			"Thumbs.db",   // Windows thumbnails
			".directory",  // KDE directory metadata

			// Git and VCS
			".gitkeep",       // Git placeholder files
			".gitignore",     // Git ignore files (not issues)
			".gitattributes", // Git attributes

			// Editor specific files
			"4913", // Vim temporary file pattern
		},
		ignoreDirs: []string{
			// Version control
			".git",             // Git repository data
			".svn",             // Subversion data
			".hg",              // Mercurial data
			".bzr",             // Bazaar data
			".fossil-settings", // Fossil VCS

			// IDEs and editors
			".idea",     // JetBrains IDEs
			".vscode",   // Visual Studio Code
			".vs",       // Visual Studio
			".settings", // Eclipse settings
			".project",  // Project metadata
			".metadata", // IDE metadata

			// Build and package directories
			"node_modules",  // Node.js dependencies
			".npm",          // NPM cache
			"vendor",        // Go vendor directory
			"target",        // Maven/Rust target
			"build",         // Generic build directory
			"dist",          // Distribution directory
			"__pycache__",   // Python cache
			".pytest_cache", // Pytest cache

			// Temporary directories
			".tmp",   // Temporary directories
			"tmp",    // Alternative temp dir
			".cache", // Cache directories
		},
	}
}

// shouldIgnore returns true if the given path should be ignored
func (r *ignoreRules) shouldIgnore(path string) bool {
	// Get the base filename
	base := filepath.Base(path)

	// Check if file ends with ignored suffix
	for _, suffix := range r.ignoreSuffixes {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}

	// Check if it's an exact filename match
	for _, ignoreFile := range r.ignoreFiles {
		if base == ignoreFile {
			return true
		}
	}

	// Check if any directory in the path should be ignored
	pathParts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	for _, part := range pathParts {
		for _, ignoreDir := range r.ignoreDirs {
			if part == ignoreDir {
				return true
			}
		}
	}

	// Special case: files starting with # (Emacs auto-save)
	if strings.HasPrefix(base, "#") && strings.HasSuffix(base, "#") {
		return true
	}

	// Special case: files starting with .# (Emacs lock files)
	if strings.HasPrefix(base, ".#") {
		return true
	}

	// Special case: numbered temporary files (e.g., file.txt.1234, file.tmp.5678)
	if r.isNumberedTempFile(base) {
		return true
	}

	// Special case: atomic save patterns (many editors create temp files then rename)
	if r.isAtomicSaveTemp(base) {
		return true
	}

	return false
}

// isNumberedTempFile detects temporary files with numeric suffixes
func (r *ignoreRules) isNumberedTempFile(filename string) bool {
	// Pattern: filename.extension.numbers (e.g., file.md.1234)
	parts := strings.Split(filename, ".")
	if len(parts) >= 3 {
		lastPart := parts[len(parts)-1]
		if len(lastPart) >= 3 && r.isAllDigits(lastPart) {
			return true
		}
	}
	return false
}

// isAtomicSaveTemp detects atomic save temporary files used by editors
func (r *ignoreRules) isAtomicSaveTemp(filename string) bool {
	// Common atomic save patterns:
	patterns := []string{
		".tmp",    // file.md.tmp
		".temp",   // file.md.temp
		"~",       // file.md~
		".new",    // file.md.new
		".atomic", // file.md.atomic
	}

	for _, pattern := range patterns {
		if strings.Contains(filename, pattern) && !strings.HasSuffix(filename, ".md") {
			// It has temp pattern but isn't a final .md file
			return true
		}
	}

	// VSCode atomic save pattern: .file.md.hash
	if strings.HasPrefix(filename, ".") && strings.Count(filename, ".") >= 2 {
		parts := strings.Split(filename[1:], ".") // Remove leading dot
		if len(parts) >= 3 && parts[len(parts)-2] == "md" {
			// Last part might be a hash
			lastPart := parts[len(parts)-1]
			if len(lastPart) >= 6 && r.isHexString(lastPart) {
				return true
			}
		}
	}

	return false
}

// isAllDigits checks if a string contains only digits
func (r *ignoreRules) isAllDigits(s string) bool {
	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}
	return len(s) > 0
}

// isHexString checks if a string is a hexadecimal hash
func (r *ignoreRules) isHexString(s string) bool {
	if len(s) < 6 {
		return false
	}
	for _, char := range s {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}
