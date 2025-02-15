package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jkingsman/ROMCopyEngine/cli_parsing"
	"github.com/jkingsman/ROMCopyEngine/copy_funcs"
	"github.com/jkingsman/ROMCopyEngine/file_operations"
	"github.com/jkingsman/ROMCopyEngine/logging"
)

func summarizeWarnConfirm(config *cli_parsing.Config) {
	cli_parsing.PrintCLIOpts(config)
	fmt.Println()

	if !config.SkipConfirm && !config.DryRun {
		if config.CleanTarget {
			logging.LogWarning("You have chosen to run with the '--cleanTarget' option enabled. This will delete all contents from the following directories before copying:")
			for _, mapping := range config.Mappings {
				logging.Log(logging.Action, "", "• %s", filepath.Join(strings.TrimRight(config.TargetDir, "/\\"), strings.TrimLeft(mapping.Destination, "/\\")))
			}
			fmt.Println()
		}

		fmt.Println("[Hint: you can rerun this with '--dryRun' to see all operations that would be performed without performing them, or use '--skipConfirm' to skip this confirmation]")
		if cli_parsing.GetConfirmation("All files will be copied as summarized above. If file names conflict, they will be overwritten. Are you sure you want to proceed?") {
			logging.Log(logging.Base, "", "Beginning copy...")
		} else {
			logging.Log(logging.Base, "", "Copy cancelled. No operations performed.")
			os.Exit(1)
		}
	} else {
		logging.Log(logging.Base, "", "-y passed; skipping confirmation... Let's rock!")
		fmt.Println()
	}
}

func explodeDirs(config *cli_parsing.Config, destPath string) error {
	logging.Log(logging.Action, "", "Exploding directories...")
	for _, explodeDir := range config.ExplodeDirs {
		if config.DryRun {
			logging.LogDryRun(logging.Detail, logging.IconExplode, "If located, would have exploded %s into %s", explodeDir, destPath)
			continue
		}
		found, err := file_operations.ExplodeFolder(destPath, explodeDir)
		if !found {
			continue
		}

		if err != nil {
			return fmt.Errorf("error exploding directory: %w", err)
		}

		logging.Log(logging.Detail, logging.IconExplode, "Exploded %s into %s", explodeDir, destPath)
	}

	logging.LogComplete("Exploding")
	return nil
}

func processRenames(config *cli_parsing.Config, destPath string) error {
	logging.Log(logging.Action, "", "Processing renames...")
	for _, r := range config.Renames {
		if config.DryRun {
			logging.LogDryRun(logging.Detail, logging.IconRename, "If located in %s, would have renamed %s to %s", destPath, r.OldName, r.NewName)
			continue
		}

		oldPath := filepath.Join(destPath, r.OldName)
		newPath := filepath.Join(destPath, r.NewName)

		_, err := os.Stat(oldPath)
		if err != nil {
			if os.IsNotExist(err) {
				logging.Log(logging.Detail, logging.IconSkip, "Unable to locate %s in %s; skipping", r.OldName, destPath)
				continue
			}
			return fmt.Errorf("error renaming item: %w", err)
		}

		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("error renaming item: %w", err)
		}

		logging.Log(logging.Detail, logging.IconRename, "Renamed %s to %s", r.OldName, r.NewName)
	}

	logging.LogComplete("Renames")
	return nil
}

func processRewrites(config *cli_parsing.Config, destPath string) error {
	logging.Log(logging.Action, "", "Processing rewrites...")
	for _, r := range config.FileRewrites {
		if config.DryRun {
			rewriteType := "literal"
			if config.RewritesAreRegex {
				rewriteType = "regex"
			}
			logging.LogDryRun(logging.Detail, logging.IconRewrite, "If files found matching glob '%s' located in %s, would have rewritten %s to %s via %s search", r.FileGlob, destPath, r.SearchPattern, r.ReplacePattern, rewriteType)
			continue
		}

		found, err := file_operations.SearchAndReplace(destPath, r.FileGlob, r.SearchPattern, r.ReplacePattern, config.RewritesAreRegex)

		if !found {
			logging.Log(logging.Detail, logging.IconSkip, "No files matching glob '%s' in %s for rewrite of %s to %s; skipping...", r.FileGlob, destPath, r.SearchPattern, r.ReplacePattern)
			continue
		}

		if err != nil {
			return fmt.Errorf("error rewriting %s to %s for glob %s: %w", r.SearchPattern, r.ReplacePattern, r.FileGlob, err)
		}
	}
	logging.LogComplete("Rewrites")
	return nil
}

func processMapping(config *cli_parsing.Config, mapping cli_parsing.DirMapping) error {
	sourcePath := filepath.Join(strings.TrimRight(config.SourceDir, "/\\"), strings.TrimLeft(mapping.Source, "/\\"))
	destPath := filepath.Join(strings.TrimRight(config.TargetDir, "/\\"), strings.TrimLeft(mapping.Destination, "/\\"))

	logging.Log(logging.Base, "", "Beginning operations for \033[1;34m%s -> %s\033[0m (%s -> %s)",
		mapping.Source, mapping.Destination, sourcePath, destPath)

	// Clean target directory if requested
	if config.CleanTarget {
		if err := cleanTargetDir(config, destPath); err != nil {
			return err
		}
	}

	// Copy files
	logging.Log(logging.Action, "", "Beginning copy...")
	filesCopied, err := copy_funcs.CopyFiles(sourcePath, destPath, config.CopyInclude, config.CopyExclude, config.DryRun)
	if err != nil {
		return fmt.Errorf("error copying files: %w", err)
	}
	logging.LogComplete("Copy")

	logging.Log(logging.Action, "", "Beginning re-glob-and-copy-matches [ignoring excludes!!!]...")
	if config.LoopbackCopy && len(filesCopied) > 0 {
		globifiedFileList := copy_funcs.GlobifyFilenameOfPathList(filesCopied)

		logging.Log(logging.Detail, logging.IconCopy, "Beginning loopback from %d glob(s): [%s]", len(filesCopied), strings.Join(globifiedFileList, ", "))
		_, err := copy_funcs.CopyFiles(sourcePath, destPath, globifiedFileList, nil, config.DryRun)
		if err != nil {
			return fmt.Errorf("error copying files: %w", err)
		}
	}
	logging.LogComplete("Re-glob-and-copy-matches")

	// Post-copy operations
	if err := runPostCopyOperations(config, destPath); err != nil {
		return err
	}

	logging.Log(logging.Base, "", "Operations for %s -> %s complete!", mapping.Source, mapping.Destination)
	return nil
}

func cleanTargetDir(config *cli_parsing.Config, destPath string) error {
	if config.DryRun {
		logging.LogDryRun(logging.Action, logging.IconClean, "Cleaning target directory...")
		return nil
	}

	logging.Log(logging.Action, logging.IconClean, "Cleaning target directory...")
	if err := file_operations.ClearDirectory(destPath); err != nil {
		return fmt.Errorf("error cleaning target directory: %w", err)
	}
	return nil
}

func runPostCopyOperations(config *cli_parsing.Config, destPath string) error {
	// Explode directories if configured
	if len(config.ExplodeDirs) > 0 {
		if err := explodeDirs(config, destPath); err != nil {
			return err
		}
	}

	// Process renames if configured
	if len(config.Renames) > 0 {
		if err := processRenames(config, destPath); err != nil {
			return err
		}
	}

	// Process rewrites if configured
	if len(config.FileRewrites) > 0 {
		if err := processRewrites(config, destPath); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	intro := `   ___  ____  __  ________               ____          _
  / _ \/ __ \/  |/  / ___/__  ___  __ __/ __/__  ___ _(_)__  ___
 / , _/ /_/ / /|_/ / /__/ _ \/ _ \/ // / _// _ \/ _ '/ / _ \/ -_)
/_/|_|\____/_/  /_/\___/\___/ .__/\_, /___/_//_/\_, /_/_//_/\__/
                           /_/   /___/         /___/`
	fmt.Println(intro)

	config, err := cli_parsing.ParseAndValidate()
	if err != nil {
		logging.LogError("Error: %v", err)
		os.Exit(1)
	}

	summarizeWarnConfirm(config)

	for _, mapping := range config.Mappings {
		if err := processMapping(config, mapping); err != nil {
			logging.LogError("Error: %v", err)
			os.Exit(1)
		}
	}

	logging.Log(logging.Base, "", "All transfers & processing completed successfully!")
}
