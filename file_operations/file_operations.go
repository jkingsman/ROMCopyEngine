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
	folderPath := filepath.Join(destPath, explodeDir)

	// Check if the folder exists
	info, err := os.Stat(folderPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // if it doesn't exist who cares
		}
		return false, fmt.Errorf("error accessing folder: %v", err)
	}

	if !info.IsDir() {
		return true, fmt.Errorf("'%s' exists but is not a directory", explodeDir)
	}

	// get contents
	items, err := os.ReadDir(folderPath)
	if err != nil {
		return true, fmt.Errorf("error reading directory contents: %v", err)
	}

	// copy up a level
	for _, item := range items {
		sourcePath := filepath.Join(folderPath, item.Name())
		destPath := filepath.Join(destPath, item.Name())

		if _, err := os.Stat(destPath); err == nil {
			return true, fmt.Errorf("cannot move '%s': destination already exists", item.Name())
		}

		if err := moveItem(sourcePath, destPath); err != nil {
			return true, fmt.Errorf("error moving '%s': %v", item.Name(), err)
		}
	}

	// kill empty folder
	if err := os.Remove(folderPath); err != nil {
		return true, fmt.Errorf("error removing empty directory '%s': %v", explodeDir, err)
	}

	return true, nil
}

func moveItem(sourcePath string, destPath string) error {
	// do a mv
	err := os.Rename(sourcePath, destPath)
	if err == nil {
		return nil
	}

	// else try copy and delete, for device safety
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	if sourceInfo.IsDir() {
		if err := copyDir(sourcePath, destPath); err != nil {
			return err
		}
	} else {
		if err := CopyFile(sourcePath, destPath); err != nil {
			return err
		}
	}

	// kill copied file
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

		fmt.Printf("    🔀 Processing '%s' to '%s' for glob '%s' as %s replace in %s...\n", searchTerm, replaceTerm, glob, process_type, file)

	}

	return true, nil
}
