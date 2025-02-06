package file_operations

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestFolder(t *testing.T, structure map[string]string) (string, func()) {
	tempDir, err := os.MkdirTemp("", "explode_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	for path, content := range structure {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		if content == "DIR" {
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				t.Fatalf("Failed to create directory %s: %v", fullPath, err)
			}
		} else {
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create file %s: %v", fullPath, err)
			}
		}
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func verifyFileContent(t *testing.T, path string, expectedContent string) {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to read file %s: %v", path, err)
		return
	}
	if string(content) != expectedContent {
		t.Errorf("File %s content mismatch. Got %s, want %s", path, content, expectedContent)
	}
}

func verifyFileExists(t *testing.T, path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestExplodeFolder(t *testing.T) {
	tests := []struct {
		name          string
		structure     map[string]string
		explodeDir    string
		expectSuccess bool
		expectError   bool
		verifyFunc    func(*testing.T, string)
	}{
		{
			name: "Happy path - simple folder explosion",
			structure: map[string]string{
				"target/file1.txt": "content1",
				"target/file2.txt": "content2",
			},
			explodeDir:    "target",
			expectSuccess: true,
			expectError:   false,
			verifyFunc: func(t *testing.T, baseDir string) {
				if verifyFileExists(t, filepath.Join(baseDir, "target")) {
					t.Error("Target directory should not exist")
				}
				verifyFileContent(t, filepath.Join(baseDir, "file1.txt"), "content1")
				verifyFileContent(t, filepath.Join(baseDir, "file2.txt"), "content2")
			},
		},
		{
			name: "Non-existent folder",
			structure: map[string]string{
				"other/file.txt": "content",
			},
			explodeDir:    "target",
			expectSuccess: false,
			expectError:   false,
			verifyFunc: func(t *testing.T, baseDir string) {
				if !verifyFileExists(t, filepath.Join(baseDir, "other/file.txt")) {
					t.Error("Original file should remain untouched")
				}
			},
		},
		{
			name: "Target is a file",
			structure: map[string]string{
				"target": "file content",
			},
			explodeDir:    "target",
			expectSuccess: true,
			expectError:   true,
			verifyFunc: func(t *testing.T, baseDir string) {
				verifyFileContent(t, filepath.Join(baseDir, "target"), "file content")
			},
		},
		{
			name: "Destination file exists",
			structure: map[string]string{
				"target/file1.txt": "content1",
				"file1.txt":        "existing",
			},
			explodeDir:    "target",
			expectSuccess: true,
			expectError:   true,
			verifyFunc: func(t *testing.T, baseDir string) {
				verifyFileContent(t, filepath.Join(baseDir, "file1.txt"), "existing")
				verifyFileContent(t, filepath.Join(baseDir, "target/file1.txt"), "content1")
			},
		},
		{
			name: "Empty folder",
			structure: map[string]string{
				"target": "DIR",
			},
			explodeDir:    "target",
			expectSuccess: true,
			expectError:   false,
			verifyFunc: func(t *testing.T, baseDir string) {
				if verifyFileExists(t, filepath.Join(baseDir, "target")) {
					t.Error("Empty target directory should be removed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir, cleanup := setupTestFolder(t, tt.structure)
			defer cleanup()

			success, err := ExplodeFolder(baseDir, tt.explodeDir)

			if success != tt.expectSuccess {
				t.Errorf("Expected success=%v, got %v (%v)", tt.expectSuccess, success, err)
			}

			if (err != nil) != tt.expectError {
				t.Errorf("Expected error=%v, got %v", tt.expectError, err)
			}

			tt.verifyFunc(t, baseDir)
		})
	}
}
