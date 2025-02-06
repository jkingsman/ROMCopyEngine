package file_operations

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// testSetup creates a temporary directory and returns cleanup function
func testSetup(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "fileops-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// createTestFile creates a file with given content
func createTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// createTestDir creates a directory with specified files
func createTestDir(baseDir string, files map[string]string) error {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	for name, content := range files {
		path := filepath.Join(baseDir, name)

		// Create parent directories if they don't exist
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		if err := createTestFile(path, content); err != nil {
			return err
		}
	}
	return nil
}

func TestMoveItem_File(t *testing.T) {
	tmpDir, cleanup := testSetup(t)
	defer cleanup()

	tests := []struct {
		name    string
		setup   func(dir string) (string, string, error)
		wantErr bool
	}{
		{
			name: "simple file move",
			setup: func(dir string) (string, string, error) {
				src := filepath.Join(dir, "source.txt")
				dst := filepath.Join(dir, "dest.txt")
				return src, dst, createTestFile(src, "test content")
			},
			wantErr: false,
		},
		{
			name: "source doesn't exist",
			setup: func(dir string) (string, string, error) {
				return filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dest.txt"), nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst, err := tt.setup(tmpDir)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			err = moveItem(src, dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("moveItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify source doesn't exist
				if _, err := os.Stat(src); !os.IsNotExist(err) {
					t.Error("Source file still exists after move")
				}
				// Verify destination exists
				if _, err := os.Stat(dst); os.IsNotExist(err) {
					t.Error("Destination file doesn't exist after move")
				}
			}
		})
	}
}

func TestMoveItem_Directory(t *testing.T) {
	tmpDir, cleanup := testSetup(t)
	defer cleanup()

	tests := []struct {
		name    string
		setup   func(dir string) (string, string, error)
		verify  func(t *testing.T, dst string)
		wantErr bool
	}{
		{
			name: "simple directory move",
			setup: func(dir string) (string, string, error) {
				src := filepath.Join(dir, "srcdir")
				dst := filepath.Join(dir, "dstdir")
				files := map[string]string{
					"file1.txt": "content1",
					"file2.txt": "content2",
				}
				return src, dst, createTestDir(src, files)
			},
			verify: func(t *testing.T, dst string) {
				files := []string{"file1.txt", "file2.txt"}
				for _, f := range files {
					path := filepath.Join(dst, f)
					if _, err := os.Stat(path); os.IsNotExist(err) {
						t.Errorf("Expected file %s missing in destination", f)
					}
				}
			},
			wantErr: false,
		},
		{
			name: "nested directory move",
			setup: func(dir string) (string, string, error) {
				src := filepath.Join(dir, "srcdir")
				dst := filepath.Join(dir, "dstdir")
				if err := os.MkdirAll(filepath.Join(src, "subdir"), 0755); err != nil {
					return "", "", err
				}
				files := map[string]string{
					"file1.txt":        "content1",
					"subdir/file2.txt": "content2",
				}
				return src, dst, createTestDir(src, files)
			},
			verify: func(t *testing.T, dst string) {
				paths := []string{
					filepath.Join(dst, "file1.txt"),
					filepath.Join(dst, "subdir", "file2.txt"),
				}
				for _, p := range paths {
					if _, err := os.Stat(p); os.IsNotExist(err) {
						t.Errorf("Expected path %s missing in destination", p)
					}
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst, err := tt.setup(tmpDir)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			err = moveItem(src, dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("moveItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify source doesn't exist
				if _, err := os.Stat(src); !os.IsNotExist(err) {
					t.Error("Source directory still exists after move")
				}
				// Run custom verification
				tt.verify(t, dst)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir, cleanup := testSetup(t)
	defer cleanup()

	tests := []struct {
		name    string
		content string
		mode    os.FileMode
		setup   func(string, string, os.FileMode) (string, string, error)
		wantErr bool
	}{
		{
			name:    "normal file copy",
			content: "test content",
			mode:    0644,
			setup: func(dir, content string, mode os.FileMode) (string, string, error) {
				src := filepath.Join(dir, "source.txt")
				dst := filepath.Join(dir, "dest.txt")
				return src, dst, createTestFile(src, content)
			},
			wantErr: false,
		},
		{
			name:    "copy to existing file",
			content: "test content",
			mode:    0644,
			setup: func(dir, content string, mode os.FileMode) (string, string, error) {
				src := filepath.Join(dir, "source.txt")
				dst := filepath.Join(dir, "dest.txt")
				if err := createTestFile(src, content); err != nil {
					return "", "", err
				}
				return src, dst, createTestFile(dst, "existing content")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst, err := tt.setup(tmpDir, tt.content, tt.mode)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			err = CopyFile(src, dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify content
				gotContent, err := os.ReadFile(dst)
				if err != nil {
					t.Fatalf("Failed to read destination file: %v", err)
				}
				if !bytes.Equal(gotContent, []byte(tt.content)) {
					t.Errorf("Content mismatch. got = %s, want = %s", gotContent, tt.content)
				}

				// Verify permissions
				info, err := os.Stat(dst)
				if err != nil {
					t.Fatalf("Failed to stat destination file: %v", err)
				}
				if info.Mode().Perm() != tt.mode.Perm() {
					t.Errorf("Mode mismatch. got = %v, want = %v", info.Mode().Perm(), tt.mode.Perm())
				}
			}
		})
	}
}

func TestCopyDir(t *testing.T) {
	tmpDir, cleanup := testSetup(t)
	defer cleanup()

	tests := []struct {
		name    string
		files   map[string]string
		wantErr bool
	}{
		{
			name:    "empty directory",
			files:   map[string]string{},
			wantErr: false,
		},
		{
			name: "flat directory",
			files: map[string]string{
				"file1.txt": "content1",
				"file2.txt": "content2",
			},
			wantErr: false,
		},
		{
			name: "nested directory",
			files: map[string]string{
				"file1.txt":         "content1",
				"subdir/file2.txt":  "content2",
				"subdir/file3.txt":  "content3",
				"subdir2/file4.txt": "content4",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := filepath.Join(tmpDir, "source")
			dst := filepath.Join(tmpDir, "dest")

			if err := createTestDir(src, tt.files); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			err := copyDir(src, dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("copyDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify all files exist in destination
				for path, wantContent := range tt.files {
					dstPath := filepath.Join(dst, path)
					gotContent, err := os.ReadFile(dstPath)
					if err != nil {
						t.Errorf("Failed to read %s: %v", dstPath, err)
						continue
					}
					if string(gotContent) != wantContent {
						t.Errorf("Content mismatch for %s. got = %s, want = %s", path, gotContent, wantContent)
					}
				}
			}
		})
	}
}

func TestClearDirectory(t *testing.T) {
	tmpDir, cleanup := testSetup(t)
	defer cleanup()

	tests := []struct {
		name    string
		setup   func(dir string) error
		verify  func(t *testing.T, dir string)
		wantErr bool
	}{
		{
			name: "empty directory",
			setup: func(dir string) error {
				return os.MkdirAll(dir, 0755)
			},
			verify: func(t *testing.T, dir string) {
				entries, err := os.ReadDir(dir)
				if err != nil {
					t.Fatalf("Failed to read dir: %v", err)
				}
				if len(entries) != 0 {
					t.Errorf("Directory not empty, contains %d entries", len(entries))
				}
			},
			wantErr: false,
		},
		{
			name: "directory with files",
			setup: func(dir string) error {
				files := map[string]string{
					"file1.txt": "content1",
					"file2.txt": "content2",
				}
				return createTestDir(dir, files)
			},
			verify: func(t *testing.T, dir string) {
				entries, err := os.ReadDir(dir)
				if err != nil {
					t.Fatalf("Failed to read dir: %v", err)
				}
				if len(entries) != 0 {
					t.Errorf("Directory not empty, contains %d entries", len(entries))
				}
			},
			wantErr: false,
		},
		{
			name: "directory with subdirectories",
			setup: func(dir string) error {
				files := map[string]string{
					"file1.txt":        "content1",
					"subdir/file2.txt": "content2",
				}
				return createTestDir(dir, files)
			},
			verify: func(t *testing.T, dir string) {
				entries, err := os.ReadDir(dir)
				if err != nil {
					t.Fatalf("Failed to read dir: %v", err)
				}
				if len(entries) != 0 {
					t.Errorf("Directory not empty, contains %d entries", len(entries))
				}
			},
			wantErr: false,
		},
		{
			name: "non-existent directory",
			setup: func(dir string) error {
				return os.RemoveAll(dir)
			},
			verify:  func(t *testing.T, dir string) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, "test")
			if err := tt.setup(testDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			err := ClearDirectory(testDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ClearDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				tt.verify(t, testDir)
			}
		})
	}
}
