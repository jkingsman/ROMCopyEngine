package file_operations

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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

func copyDir(sourcePath, destPath string) error {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	// make destPath dirs
	if err := os.MkdirAll(destPath, sourceInfo.Mode()); err != nil {
		return err
	}

	// get stuff to be copied
	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return err
	}

	// loop copy
	for _, entry := range entries {
		sourceSub := filepath.Join(sourcePath, entry.Name())
		destSub := filepath.Join(destPath, entry.Name())

		if entry.IsDir() {
			if err := copyDir(sourceSub, destSub); err != nil {
				return err
			}
		} else {
			if err := CopyFile(sourceSub, destSub); err != nil {
				return err
			}
		}
	}

	return nil
}

func ClearDirectory(dirPath string) error {
	dir, err := os.Open(dirPath)
	if err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}
	defer dir.Close()

	// line em up....
	entries, err := dir.Readdir(-1)
	if err != nil {
		return fmt.Errorf("failed to read directory contents: %w", err)
	}

	// ...and knock em down
	for _, entry := range entries {
		path := filepath.Join(dirPath, entry.Name())

		if entry.IsDir() {
			err = os.RemoveAll(path)
		} else {
			err = os.Remove(path)
		}

		if err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	return nil
}

func SearchAndReplace(path string, glob string, searchTerm string, replaceTerm string, isRegex bool) (bool, error) {
	// get glob matches
	matches, err := filepath.Glob(filepath.Join(path, glob))
	if err != nil {
		return false, fmt.Errorf("error finding files: %w", err)
	}

	if len(matches) == 0 {
		return false, fmt.Errorf("no files found matching pattern %s in path %s", glob, path)
	}

	// loop over matches
	for _, file := range matches {
		content, err := os.ReadFile(file)
		if err != nil {
			return true, fmt.Errorf("error reading file %s: %w", file, err)
		}

		// run the replace
		var newContent string
		if isRegex {
			regex, err := regexp.Compile(searchTerm)
			if err != nil {
				return true, fmt.Errorf("invalid regex pattern %s: %w", searchTerm, err)
			}
			newContent = regex.ReplaceAllString(string(content), replaceTerm)
		} else {
			newContent = strings.ReplaceAll(string(content), searchTerm, replaceTerm)
		}

		// write content back to file
		err = os.WriteFile(file, []byte(newContent), 0644)
		if err != nil {
			return true, fmt.Errorf("error writing to file %s: %w", file, err)
		}

		process_type := "literal"
		if isRegex {
			process_type = "regex"
		}

		logging.Log(logging.Detail, logging.IconReplace, "Replaced '%s' with '%s' for glob '%s' as %s replace in %s", searchTerm, replaceTerm, glob, process_type, file)

	}

	return true, nil
}
