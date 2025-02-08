package copy_funcs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldInclude(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		includes []string
		excludes []string
		want     bool
	}{
		{
			name:     "no patterns",
			path:     "file.txt",
			includes: []string{},
			excludes: []string{},
			want:     true,
		},
		{
			name:     "simple include match",
			path:     "file.txt",
			includes: []string{"*.txt"},
			excludes: []string{},
			want:     true,
		},
		{
			name:     "simple include no match",
			path:     "file.txt",
			includes: []string{"*.jpg"},
			excludes: []string{},
			want:     false,
		},
		{
			name:     "simple exclude match",
			path:     "file.txt",
			includes: []string{},
			excludes: []string{"*.txt"},
			want:     false,
		},
		{
			name:     "include and exclude match",
			path:     "file.txt",
			includes: []string{"*.txt"},
			excludes: []string{"file.*"},
			want:     false,
		},
		{
			name:     "nested path match",
			path:     "dir/subdir/file.txt",
			includes: []string{"**/*.txt"},
			excludes: []string{},
			want:     true,
		},
		{
			name:     "directory pattern match",
			path:     "dir/subdir/file.txt",
			includes: []string{"dir/**"},
			excludes: []string{},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldInclude(tt.path, tt.includes, tt.excludes)
			if got != tt.want {
				t.Errorf("shouldInclude() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldIncludeDir(t *testing.T) {
	// Create temporary test directory structure
	tmpDir, err := os.MkdirTemp("", "test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test directory structure:
	// tmpDir/
	//   ├── empty/
	//   ├── with_matching_file/
	//   │   └── file.txt
	//   └── with_non_matching_file/
	//       └── file.jpg

	dirs := []string{
		filepath.Join(tmpDir, "empty"),
		filepath.Join(tmpDir, "with_matching_file"),
		filepath.Join(tmpDir, "with_non_matching_file"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	// Create test files
	if err := os.WriteFile(
		filepath.Join(tmpDir, "with_matching_file", "file.txt"),
		[]byte("test"),
		0644,
	); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(tmpDir, "with_non_matching_file", "file.jpg"),
		[]byte("test"),
		0644,
	); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name     string
		dirPath  string
		includes []string
		excludes []string
		want     bool
	}{
		{
			name:     "empty directory with no patterns",
			dirPath:  filepath.Join(tmpDir, "empty"),
			includes: []string{},
			excludes: []string{},
			want:     true,
		},
		{
			name:     "directory with matching file",
			dirPath:  filepath.Join(tmpDir, "with_matching_file"),
			includes: []string{"**/*.txt"},
			excludes: []string{},
			want:     true,
		},
		{
			name:     "directory with non-matching file",
			dirPath:  filepath.Join(tmpDir, "with_non_matching_file"),
			includes: []string{"**/*.txt"},
			excludes: []string{},
			want:     false,
		},
		{
			name:     "empty directory with matching pattern",
			dirPath:  filepath.Join(tmpDir, "empty"),
			includes: []string{"empty/**"},
			excludes: []string{},
			want:     true,
		},
		{
			name:     "empty directory with excluding pattern",
			dirPath:  filepath.Join(tmpDir, "empty"),
			includes: []string{},
			excludes: []string{"empty/**"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shouldIncludeDir(tt.dirPath, tmpDir, tt.includes, tt.excludes)
			if err != nil {
				t.Errorf("shouldIncludeDir() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("shouldIncludeDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyFiles(t *testing.T) {
	// Create temporary source and destination directories
	sourceDir, err := os.MkdirTemp("", "source-*")
	if err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	defer os.RemoveAll(sourceDir)

	destDir, err := os.MkdirTemp("", "dest-*")
	if err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Create test directory structure:
	// sourceDir/
	//   ├── file1.txt
	//   ├── file2.jpg
	//   ├── subdir1/
	//   │   ├── file3.txt
	//   │   └── file4.jpg
	//   ├── empty/
	//   └── nested/
	//       └── empty_nested/

	// Create directories
	dirs := []string{
		filepath.Join(sourceDir, "subdir1"),
		filepath.Join(sourceDir, "empty"),
		filepath.Join(sourceDir, "nested", "empty_nested"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	// Create test files
	files := map[string][]byte{
		filepath.Join(sourceDir, "file1.txt"):         []byte("file1"),
		filepath.Join(sourceDir, "file2.jpg"):         []byte("file2"),
		filepath.Join(sourceDir, "subdir1/file3.txt"): []byte("file3"),
		filepath.Join(sourceDir, "subdir1/file4.jpg"): []byte("file4"),
	}

	for path, content := range files {
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", path, err)
		}
	}

	tests := []struct {
		name            string
		includes        []string
		excludes        []string
		dryRun          bool
		wantFiles       []string
		wantDirs        []string
		wantMissing     []string
		wantMissingDirs []string
	}{
		{
			name:     "copy all files and directories",
			includes: []string{},
			excludes: []string{},
			dryRun:   false,
			wantFiles: []string{
				"file1.txt",
				"file2.jpg",
				"subdir1/file3.txt",
				"subdir1/file4.jpg",
			},
			wantDirs: []string{
				"empty",
				"nested",
				"nested/empty_nested",
				"subdir1",
			},
			wantMissing:     []string{},
			wantMissingDirs: []string{},
		},
		{
			name:     "copy only txt files, empty dirs not included with pattern",
			includes: []string{"**/*.txt"},
			excludes: []string{},
			dryRun:   false,
			wantFiles: []string{
				"file1.txt",
				"subdir1/file3.txt",
			},
			wantDirs: []string{
				"subdir1", // only included because it contains matching files
			},
			wantMissing: []string{
				"file2.jpg",
				"subdir1/file4.jpg",
			},
			wantMissingDirs: []string{
				"empty",
				"nested",
				"nested/empty_nested",
			},
		},
		{
			name:     "exclude subdir, empty dirs included",
			includes: []string{},
			excludes: []string{"subdir1/**"},
			dryRun:   false,
			wantFiles: []string{
				"file1.txt",
				"file2.jpg",
			},
			wantDirs: []string{
				"empty",
				"nested",
				"nested/empty_nested",
			},
			wantMissing: []string{
				"subdir1/file3.txt",
				"subdir1/file4.jpg",
			},
			wantMissingDirs: []string{
				"subdir1",
			},
		},
		{
			name:      "include directory pattern includes empty dirs",
			includes:  []string{"**/empty/**", "**/empty"},
			excludes:  []string{},
			dryRun:    false,
			wantFiles: []string{},
			wantDirs: []string{
				"empty",
			},
			wantMissing: []string{
				"file1.txt",
				"file2.jpg",
				"subdir1/file3.txt",
				"subdir1/file4.jpg",
			},
			wantMissingDirs: []string{
				"subdir1",
				"nested",
				"nested/empty_nested",
			},
		},
		{
			name:      "nested empty directory pattern",
			includes:  []string{"**/nested/**"},
			excludes:  []string{},
			dryRun:    false,
			wantFiles: []string{},
			wantDirs: []string{
				"nested",
				"nested/empty_nested",
			},
			wantMissing: []string{
				"file1.txt",
				"file2.jpg",
				"subdir1/file3.txt",
				"subdir1/file4.jpg",
			},
			wantMissingDirs: []string{
				"empty",
				"subdir1",
			},
		},
		{
			name:      "dry run",
			includes:  []string{},
			excludes:  []string{},
			dryRun:    true,
			wantFiles: []string{},
			wantDirs:  []string{},
			wantMissing: []string{
				"file1.txt",
				"file2.jpg",
				"subdir1/file3.txt",
				"subdir1/file4.jpg",
			},
			wantMissingDirs: []string{
				"empty",
				"nested",
				"nested/empty_nested",
				"subdir1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean destination directory
			os.RemoveAll(destDir)
			os.MkdirAll(destDir, 0755)

			_, err := CopyFiles(sourceDir, destDir, tt.includes, tt.excludes, tt.dryRun)
			if err != nil {
				t.Errorf("CopyFiles() error = %v", err)
				return
			}

			// Check expected files exist
			for _, file := range tt.wantFiles {
				path := filepath.Join(destDir, file)
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("expected file %s to exist", file)
				}
			}

			// Check expected directories exist
			for _, dir := range tt.wantDirs {
				path := filepath.Join(destDir, dir)
				if info, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("expected directory %s to exist", dir)
				} else if !info.IsDir() {
					t.Errorf("expected %s to be a directory", dir)
				}
			}

			// Check expected files don't exist
			for _, file := range tt.wantMissing {
				path := filepath.Join(destDir, file)
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Errorf("expected file %s to not exist", file)
				}
			}

			// Check expected directories don't exist
			for _, dir := range tt.wantMissingDirs {
				path := filepath.Join(destDir, dir)
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Errorf("expected directory %s to not exist", dir)
				}
			}
		})
	}
}
