package copy_funcs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/jkingsman/ROMCopyEngine/file_operations"
	"github.com/jkingsman/ROMCopyEngine/logging"
)

// shouldIncludeDir determines if a directory should be included based on:
// 1. If it's empty and matches the include/exclude rules
// 2. If it contains any files that match the include/exclude rules
func shouldIncludeDir(dirPath string, absSource string, includes []string, excludes []string) (bool, error) {
	// First check if the directory itself matches the rules (for empty directories)
	relPath, err := filepath.Rel(absSource, dirPath)
	if err != nil {
		return false, fmt.Errorf("failed to get relative path for %s: %w", dirPath, err)
	}

	if relPath == "." {
		return true, nil
	}

	dirShouldBeIncluded := shouldInclude(relPath, includes, excludes)

	// Check if the directory has any matching files
	hasMatchingFiles := false
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == dirPath {
			return nil
		}

		relPath, err := filepath.Rel(absSource, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		// If we find a matching file, mark it and stop walking
		if !info.IsDir() && shouldInclude(relPath, includes, excludes) {
			hasMatchingFiles = true
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return false, err
	}

	// Check if directory is empty
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}
	isEmpty := len(entries) == 0

	// Include the directory if:
	// 1. It's empty and matches the include/exclude rules, or
	// 2. It contains matching files
	return (isEmpty && dirShouldBeIncluded) || hasMatchingFiles, nil
}

func CopyFiles(sourcePath string, destPath string, copyInclude []string, copyExclude []string, dryRun bool) error {
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute source path: %w", err)
	}

	absDest, err := filepath.Abs(destPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute destination path: %w", err)
	}

	// First pass: collect all directories that should be created
	dirsToCreate := make(map[string]os.FileMode)
	err = filepath.Walk(absSource, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		if !info.IsDir() {
			return nil
		}

		shouldInclude, err := shouldIncludeDir(path, absSource, copyInclude, copyExclude)
		if err != nil {
			return err
		}

		if shouldInclude {
			relPath, err := filepath.Rel(absSource, path)
			if err != nil {
				return fmt.Errorf("failed to get relative path for %s: %w", path, err)
			}

			if relPath != "." {
				destDir := filepath.Join(absDest, relPath)
				dirsToCreate[destDir] = info.Mode()
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Second pass: copy files and create necessary directories
	return filepath.Walk(absSource, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		relPath, err := filepath.Rel(absSource, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		if relPath == "." {
			return nil
		}

		destFile := filepath.Join(absDest, relPath)

		if info.IsDir() {
			if mode, exists := dirsToCreate[destFile]; exists {
				if dryRun {
					logging.LogDryRun(logging.Detail, logging.IconFolder, "Creating dir: %s", destFile)
				} else {
					logging.Log(logging.Detail, logging.IconFolder, "Creating dir: %s", destFile)
					if err := os.MkdirAll(destFile, mode); err != nil {
						return fmt.Errorf("failed to create directory %s: %w", destFile, err)
					}
				}
			}
			return nil
		}

		if !shouldInclude(relPath, copyInclude, copyExclude) {
			logging.Log(logging.Detail, logging.IconSkip, "Skipping file: %s", relPath)
			return nil
		}

		if dryRun {
			logging.LogDryRun(logging.Detail, logging.IconCopy, "Copying file: %s -> %s",
				filepath.Join(filepath.Base(absSource), relPath),
				filepath.Join(filepath.Base(absDest), relPath))
		} else {
			logging.Log(logging.Detail, logging.IconCopy, "Copying file: %s -> %s",
				filepath.Join(filepath.Base(absSource), relPath),
				filepath.Join(filepath.Base(absDest), relPath))

			// Create parent directory if it's in our list of directories to create
			parentDir := filepath.Dir(destFile)
			if mode, exists := dirsToCreate[parentDir]; exists {
				if err := os.MkdirAll(parentDir, mode); err != nil {
					return fmt.Errorf("failed to create directories for %s: %w", destFile, err)
				}
			}
			return file_operations.CopyFile(path, destFile)
		}

		return nil
	})
}

func shouldInclude(path string, includes []string, excludes []string) bool {
	path = filepath.ToSlash(path)
	included := len(includes) == 0

	for _, pattern := range includes {
		pattern = filepath.ToSlash(pattern)
		if matched, _ := doublestar.Match(pattern, path); matched {
			included = true
			break
		}
	}

	if !included {
		return false
	}

	for _, pattern := range excludes {
		pattern = filepath.ToSlash(pattern)
		if matched, _ := doublestar.Match(pattern, path); matched {
			return false
		}
	}

	return true
}
