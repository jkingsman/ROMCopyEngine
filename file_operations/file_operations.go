package file_operations

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/jkingsman/ROMCopyEngine/logging"
)

// copies all contents out of destPath/explodeDir into destPath, then removes destPath/explodeDir
// bool: whether the folder was found
func ExplodeFolder(destPath string, explodeDir string) (bool, error) {
	folderPath := filepath.Join(destPath, explodeDir)

	// Check if the folder exists and is a directory
	info, err := os.Stat(folderPath)
	if err != nil {
		if os.IsNotExist(err) {
			logging.Log(logging.Detail, logging.IconSkip, "Unable to locate %s folder to explode; skipping", explodeDir)
			return false, nil // Not an error if folder doesn't exist
		}
		return false, fmt.Errorf("failed to access folder %s: %w", folderPath, err)
	}

	if !info.IsDir() {
		return true, fmt.Errorf("path %s exists but is not a directory", folderPath)
	}

	// Read directory contents
	items, err := os.ReadDir(folderPath)
	if err != nil {
		return true, fmt.Errorf("failed to read contents of directory %s: %w", folderPath, err)
	}

	// Move each item up one level
	for _, item := range items {
		sourcePath := filepath.Join(folderPath, item.Name())
		destPath := filepath.Join(destPath, item.Name())

		// Check for naming conflicts
		if _, err := os.Stat(destPath); err == nil {
			return true, fmt.Errorf("cannot move %s: destination %s already exists", sourcePath, destPath)
		}

		if err := moveItem(sourcePath, destPath); err != nil {
			return true, fmt.Errorf("failed to move %s to %s: %w", sourcePath, destPath, err)
		}
		logging.Log(logging.Detail, logging.IconExplode, "Moved %s to %s", item.Name(), destPath)
	}

	// Remove the now-empty source directory
	if err := os.Remove(folderPath); err != nil {
		return true, fmt.Errorf("failed to remove empty directory %s: %w", folderPath, err)
	}

	return true, nil
}

func moveItem(sourcePath string, destPath string) error {
	// Try a direct move first
	if err := os.Rename(sourcePath, destPath); err == nil {
		return nil
	}

	// If direct move fails, try copy and delete approach
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get source info for %s: %w", sourcePath, err)
	}

	if sourceInfo.IsDir() {
		if err := copyDir(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to copy directory from %s to %s: %w", sourcePath, destPath, err)
		}
	} else {
		if err := CopyFile(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to copy file from %s to %s: %w", sourcePath, destPath, err)
		}
	}

	// delete copied file
	if err := os.RemoveAll(sourcePath); err != nil {
		return fmt.Errorf("failed to remove source after copy %s: %w", sourcePath, err)
	}

	return nil
}

// File operations
func CopyFile(srcPath string, destPath string) error {
	source, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", srcPath, err)
	}
	defer source.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", destPath, err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, source); err != nil {
		return fmt.Errorf("failed to copy file contents from %s to %s: %w", srcPath, destPath, err)
	}

	sourceInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("failed to get source file info for %s: %w", srcPath, err)
	}

	return os.Chmod(destPath, sourceInfo.Mode())
}

func copyDir(sourcePath string, destPath string) error {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get source directory info for %s: %w", sourcePath, err)
	}

	if err := os.MkdirAll(destPath, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destPath, err)
	}

	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source directory %s: %w", sourcePath, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(sourcePath, entry.Name())
		dstPath := filepath.Join(destPath, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy directory from %s to %s: %w", srcPath, dstPath, err)
			}
		} else {
			if err := CopyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy file from %s to %s: %w", srcPath, dstPath, err)
			}
		}
	}

	return nil
}

// Directory operations
func ClearDirectory(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	for _, entry := range entries {
		path := filepath.Join(dirPath, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	return nil
}

// Content operations
func SearchAndReplace(path string, glob string, searchTerm string, replaceTerm string, isRegex bool) (bool, error) {
	pattern := filepath.Join(path, glob)
	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return false, fmt.Errorf("failed to process glob pattern %s: %w", pattern, err)
	}

	if len(matches) == 0 {
		return false, nil
	}

	var searchRegex *regexp.Regexp
	if isRegex {
		searchRegex, err = regexp.Compile(searchTerm)
		if err != nil {
			return true, fmt.Errorf("invalid regex pattern %s: %w", searchTerm, err)
		}
	}

	for _, file := range matches {
		content, err := os.ReadFile(file)
		if err != nil {
			return true, fmt.Errorf("failed to read file %s: %w", file, err)
		}

		var newContent []byte
		if isRegex {
			newContent = searchRegex.ReplaceAll(content, []byte(replaceTerm))
		} else {
			newContent = []byte(strings.ReplaceAll(string(content), searchTerm, replaceTerm))
		}

		if err := os.WriteFile(file, newContent, 0644); err != nil {
			return true, fmt.Errorf("failed to write to file %s: %w", file, err)
		}

		logging.Log(logging.Detail, logging.IconRewrite, "Rewrote %s", file)
	}

	return true, nil
}
