package copy_funcs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/jkingsman/ROMCopyEngine/file_operations"
)

func CopyFiles(sourcePath string, destPath string, copyInclude []string, copyExclude []string, dryRun bool) error {
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute source path: %w", err)
	}

	absDest, err := filepath.Abs(destPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute destination path: %w", err)
	}

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
		prefix := ""
		if dryRun {
			prefix = "[DRY RUN] "
		}

		if info.IsDir() {
			if !dryRun {
				fmt.Printf("    📁 %sCreating dir: %s\n", prefix, destFile)
				if err := os.MkdirAll(destFile, info.Mode()); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", destFile, err)
				}
			}
			return nil
		}

		if !shouldInclude(relPath, copyInclude, copyExclude) {
			fmt.Printf("    ⏭️ Skipping file: %s\n", relPath)
			return nil
		}

		fmt.Printf("    📋 %sCopying file: %s -> %s\n", prefix, filepath.Join(filepath.Base(absSource), relPath), filepath.Join(filepath.Base((absDest)), relPath))
		if !dryRun {
			if err := os.MkdirAll(filepath.Dir(destFile), 0755); err != nil {
				return fmt.Errorf("failed to create directories for %s: %w", destFile, err)
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
