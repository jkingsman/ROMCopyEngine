package copy_funcs

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDirectory(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "copy_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test directory structure
	dirs := []string{
		"folder1",
		"folder1/subfolder",
		"folder2",
		"folder2/deep/nested",
	}

	files := []string{
		"root.txt",
		"folder1/file1.txt",
		"folder1/file2.jpg",
		"folder1/subfolder/sub1.txt",
		"folder2/test.go",
		"folder2/deep/nested/deep.txt",
	}

	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	for _, file := range files {
		path := filepath.Join(tempDir, file)
		err := os.WriteFile(path, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func TestShouldInclude(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		include  []string
		exclude  []string
		expected bool
	}{
		{
			name:     "no patterns",
			path:     "test.txt",
			include:  []string{},
			exclude:  []string{},
			expected: true,
		},
		{
			name:     "simple include match",
			path:     "test.txt",
			include:  []string{"*.txt"},
			exclude:  []string{},
			expected: true,
		},
		{
			name:     "simple exclude match",
			path:     "test.txt",
			include:  []string{"*"},
			exclude:  []string{"*.txt"},
			expected: false,
		},
		{
			name:     "nested path include with explicit subfolder",
			path:     "folder/subfolder/test.txt",
			include:  []string{"*/*.txt", "*/subfolder/*.txt"},
			exclude:  []string{},
			expected: true,
		},
		{
			name:     "nested path include with doublestar glob",
			path:     "folder/subfolder/test.txt",
			include:  []string{"**/*.txt"},
			exclude:  []string{},
			expected: true,
		},
		{
			name:     "exclude overrides include",
			path:     "test.txt",
			include:  []string{"*.txt"},
			exclude:  []string{"test.txt"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldInclude(tt.path, tt.include, tt.exclude)
			if result != tt.expected {
				t.Errorf("shouldInclude(%q, %v, %v) = %v; want %v",
					tt.path, tt.include, tt.exclude, result, tt.expected)
			}
		})
	}
}

func TestCopyFiles(t *testing.T) {
	sourceDir, cleanupSource := setupTestDirectory(t)
	defer cleanupSource()

	tests := []struct {
		name     string
		include  []string
		exclude  []string
		dryRun   bool
		expected []string // Expected files in destination
	}{
		{
			name:    "copy all files",
			include: []string{},
			exclude: []string{},
			dryRun:  false,
			expected: []string{
				"root.txt",
				"folder1/file1.txt",
				"folder1/file2.jpg",
				"folder1/subfolder/sub1.txt",
				"folder2/test.go",
				"folder2/deep/nested/deep.txt",
			},
		},
		{
			name:    "copy only txt files",
			include: []string{"**/*.txt"},
			exclude: []string{},
			dryRun:  false,
			expected: []string{
				"root.txt",
				"folder1/file1.txt",
				"folder1/subfolder/sub1.txt",
				"folder2/deep/nested/deep.txt",
			},
		},
		{
			name:    "exclude nested files",
			include: []string{},
			exclude: []string{"**/nested/**"},
			dryRun:  false,
			expected: []string{
				"root.txt",
				"folder1/file1.txt",
				"folder1/file2.jpg",
				"folder1/subfolder/sub1.txt",
				"folder2/test.go",
			},
		},
		{
			name:     "dry run should not copy",
			include:  []string{"*.txt"},
			exclude:  []string{},
			dryRun:   true,
			expected: []string{
				// No files should be copied
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destDir, err := os.MkdirTemp("", "copy_test_dest_*")
			if err != nil {
				t.Fatalf("Failed to create destination directory: %v", err)
			}
			defer os.RemoveAll(destDir)

			err = CopyFiles(sourceDir, destDir, tt.include, tt.exclude, tt.dryRun)
			if err != nil {
				t.Fatalf("CopyFiles failed: %v", err)
			}

			// Verify copied files
			var copiedFiles []string
			err = filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					relPath, err := filepath.Rel(destDir, path)
					if err != nil {
						return err
					}
					copiedFiles = append(copiedFiles, filepath.ToSlash(relPath))
				}
				return nil
			})
			if err != nil {
				t.Fatalf("Failed to walk destination directory: %v", err)
			}

			// Convert expected paths to slashes for consistent comparison
			expectedFiles := make([]string, len(tt.expected))
			for i, path := range tt.expected {
				expectedFiles[i] = filepath.ToSlash(path)
			}

			// Check if copied files match expected files
			if len(copiedFiles) != len(expectedFiles) {
				t.Errorf("Got %d files, want %d", len(copiedFiles), len(expectedFiles))
				t.Errorf("Copied files: %v", copiedFiles)
				t.Errorf("Expected files: %v", expectedFiles)
			}

			// Check each expected file exists
			for _, expected := range expectedFiles {
				found := false
				for _, copied := range copiedFiles {
					if copied == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected file %s not found in destination", expected)
				}
			}
		})
	}
}
