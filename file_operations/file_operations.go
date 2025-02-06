package file_operations

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// copies all contents out of destPath/explodeDir into destPath, then removes destPath/explodeDir
// bool: whether the folder was found
func ExplodeFolder(destPath string, explodeDir string) (bool, error) {
	// Construct the full path to the folder we want to explode
	folderPath := filepath.Join(destPath, explodeDir)

	// Check if the folder exists
	info, err := os.Stat(folderPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // Folder doesn't exist, but that's not an error
		}
		return false, fmt.Errorf("error accessing folder: %v", err)
	}

	// Verify it's a directory
	if !info.IsDir() {
		return true, fmt.Errorf("'%s' exists but is not a directory", explodeDir)
	}

	// Read all items from the folder
	items, err := os.ReadDir(folderPath)
	if err != nil {
		return true, fmt.Errorf("error reading directory contents: %v", err)
	}

	// Move each item to the parent directory
	for _, item := range items {
		sourcePath := filepath.Join(folderPath, item.Name())
		destPath := filepath.Join(destPath, item.Name())

		// Check if destination already exists
		if _, err := os.Stat(destPath); err == nil {
			return true, fmt.Errorf("cannot move '%s': destination already exists", item.Name())
		}

		// Move the item
		if err := moveItem(sourcePath, destPath); err != nil {
			return true, fmt.Errorf("error moving '%s': %v", item.Name(), err)
		}
	}

	// Remove the now-empty directory
	if err := os.Remove(folderPath); err != nil {
		return true, fmt.Errorf("error removing empty directory '%s': %v", explodeDir, err)
	}

	return true, nil
}

// moveItem handles moving both files and directories
func moveItem(sourcePath string, destPath string) error {
	// First try a simple rename/move operation
	err := os.Rename(sourcePath, destPath)
	if err == nil {
		return nil
	}

	// If rename fails (e.g., across devices), fall back to copy and delete
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	if sourceInfo.IsDir() {
		// Handle directory copy
		if err := copyDir(sourcePath, destPath); err != nil {
			return err
		}
	} else {
		// Handle file copy
		if err := CopyFile(sourcePath, destPath); err != nil {
			return err
		}
	}

	// Remove the source after successful copy
	return os.RemoveAll(sourcePath)
}

func CopyFile(srcPath string, destPath string) error {
	source, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dest.Close()

	if _, err := io.Copy(dest, source); err != nil {
		return err
	}

	sourceInfo, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	return os.Chmod(destPath, sourceInfo.Mode())
}

func copyDir(sourcePath, destPath string) error {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	// Create the destination directory
	if err := os.MkdirAll(destPath, sourceInfo.Mode()); err != nil {
		return err
	}

	// Read directory contents
	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return err
	}

	// Copy each entry recursively
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
	// Read the directory
	dir, err := os.Open(dirPath)
	if err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}
	defer dir.Close()

	// Get all entries in the directory
	entries, err := dir.Readdir(-1)
	if err != nil {
		return fmt.Errorf("failed to read directory contents: %w", err)
	}

	// Iterate through entries and remove each one
	for _, entry := range entries {
		path := filepath.Join(dirPath, entry.Name())

		// If it's a directory, remove it and all its contents
		if entry.IsDir() {
			err = os.RemoveAll(path)
		} else {
			// If it's a file, just remove the file
			err = os.Remove(path)
		}

		if err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	return nil
}

func SearchAndReplace(path string, glob string, searchTerm string, replaceTerm string, isRegex bool) (bool, error) {
	// Get list of files matching the glob pattern
	matches, err := filepath.Glob(filepath.Join(path, glob))
	if err != nil {
		return false, fmt.Errorf("error finding files: %w", err)
	}

	if len(matches) == 0 {
		return false, fmt.Errorf("no files found matching pattern %s in path %s", glob, path)
	}

	// Process each matching file
	for _, file := range matches {
		// Read file content
		content, err := os.ReadFile(file)
		if err != nil {
			return true, fmt.Errorf("error reading file %s: %w", file, err)
		}

		var newContent string
		if isRegex {
			// Compile and execute regex replacement
			regex, err := regexp.Compile(searchTerm)
			if err != nil {
				return true, fmt.Errorf("invalid regex pattern %s: %w", searchTerm, err)
			}
			newContent = regex.ReplaceAllString(string(content), replaceTerm)
		} else {
			// Perform simple string replacement
			newContent = strings.ReplaceAll(string(content), searchTerm, replaceTerm)
		}

		// Write the modified content back to the file
		err = os.WriteFile(file, []byte(newContent), 0644)
		if err != nil {
			return true, fmt.Errorf("error writing to file %s: %w", file, err)
		}

		process_type := "literal"
		if isRegex {
			process_type = "regex"
		}

		fmt.Printf("    🔀 Processing '%s' to '%s' for glob '%s' as %s replace in %s...\n", searchTerm, replaceTerm, glob, process_type, file)

	}

	return true, nil
}
