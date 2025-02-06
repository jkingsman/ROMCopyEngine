package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jkingsman/ROMCopyEngine/cli_parsing"
	"github.com/jkingsman/ROMCopyEngine/copy_funcs"
	"github.com/jkingsman/ROMCopyEngine/file_operations"
)

func summarizeWarnConfirm(config *cli_parsing.Config) {
	cli_parsing.PrintCLIOpts(config)
	fmt.Println()

	if !config.SkipConfirm && !config.DryRun {
		if config.CleanTarget {
			fmt.Println("⚠️ WARNING ⚠️")
			fmt.Println("You have chosen to run with the '--cleanTarget' option enabled. This will delete all contents from the following directories before copying:")
			for _, mapping := range config.Mappings {
				fmt.Println(" •", filepath.Join(strings.TrimRight(config.TargetDir, "/\\"), strings.TrimLeft(mapping.Destination, "/\\")))
			}
			fmt.Println()
		}

		fmt.Println("[Hint: you can rerun this with '--dryRun' to see all operations that would be performed without performing them, or use '--skipConfirm' to skip this confirmation]")
		if cli_parsing.GetConfirmation("Are you sure you want to proceed?") {
			fmt.Println("Beginning copy...")
		} else {
			fmt.Println("Copy cancelled. No operations performed.")
			os.Exit(1)
		}
	} else {
		fmt.Println("-y passed; skipping confirmation... Let's rock!")
		fmt.Println()
	}
}

func explodeDirs(config *cli_parsing.Config, destPath string) {
	fmt.Println("  Exploding directories...")
	for _, explodeDir := range config.ExplodeDirs {
		if config.DryRun {
			fmt.Printf("   💥 [DRY RUN] If located, would have exploded %s into %s\n", explodeDir, destPath)
			continue
		}
		found, err := file_operations.ExplodeFolder(destPath, explodeDir)
		if !found {
			fmt.Printf("    ⏭️ Unable to locate %s folder to explode; skipping\n", explodeDir)
			continue
		}

		if err != nil {
			fmt.Printf("    Error exploding directory: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("    💥 Exploded %s into %s\n", explodeDir, destPath)
	}

	fmt.Println("  Exploding complete!")
}

func processRenames(config *cli_parsing.Config, destPath string) {
	fmt.Println("  Processing renames...")
	for _, r := range config.Renames {
		if config.DryRun {
			fmt.Printf("   🏷️ [DRY RUN] If located in %s, would have renamed %s to %s\n", destPath, r.OldName, r.NewName)
			continue
		}

		// Construct full paths
		oldPath := filepath.Join(destPath, r.OldName)
		newPath := filepath.Join(destPath, r.NewName)

		// Check if old file/folder exists
		_, err := os.Stat(oldPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("    ⏭️ Unable to locate %s in %s; skipping\n", r.OldName, destPath)
				continue
			}
			fmt.Printf("    Error renaming item: %s\n", err)
			os.Exit(1)
		}

		// Attempt to rename
		err = os.Rename(oldPath, newPath)
		if err != nil {
			fmt.Printf("    Error renaming item: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("    🏷️ Renamed %s to %s\n", r.OldName, r.NewName)

	}

	fmt.Println("  Renames complete!")
}

func processRewrites(config *cli_parsing.Config, destPath string) {
	fmt.Println("  Processing rewrites...")
	for _, r := range config.FileRewrites {
		if config.DryRun {
			rewriteType := "literal"
			if config.RewritesAreRegex {
				rewriteType = "regex"
			}
			fmt.Printf("   🔀 [DRY RUN] If files found matching glob '%s' located in %s, would have rewritten %s to %s via %s search\n", r.FileGlob, destPath, r.SearchPattern, r.ReplacePattern, rewriteType)
			continue
		}

		found, err := file_operations.SearchAndReplace(destPath, r.FileGlob, r.SearchPattern, r.ReplacePattern, config.RewritesAreRegex)

		if !found {
			fmt.Printf("   ⏭️ No files matching glob '%s' in %s for rewrite of %s to %s; skipping...\n", r.FileGlob, destPath, r.SearchPattern, r.ReplacePattern)
			continue
		}

		if err != nil {
			fmt.Printf("   Error rewriting %s to %s for glob %s...: %s", r.SearchPattern, r.ReplacePattern, r.FileGlob, err)
			os.Exit(1)
		}
	}
	fmt.Println("  Rewrites complete!")
}

func main() {
	intro := `

   ___  ____  __  ________               ____          _
  / _ \/ __ \/  |/  / ___/__  ___  __ __/ __/__  ___ _(_)__  ___
 / , _/ /_/ / /|_/ / /__/ _ \/ _ \/ // / _// _ \/ _ '/ / _ \/ -_)
/_/|_|\____/_/  /_/\___/\___/ .__/\_, /___/_//_/\_, /_/_//_/\__/
                           /_/   /___/         /___/

          ┌┐ ┬ ┬   ┬┌─┐┌─┐┬┌─  ┬┌─┬┌┐┌┌─┐┌─┐┌┬┐┌─┐┌┐┌
          ├┴┐└┬┘   │├─┤│  ├┴┐  ├┴┐│││││ ┬└─┐│││├─┤│││
          └─┘ ┴   └┘┴ ┴└─┘┴ ┴  ┴ ┴┴┘└┘└─┘└─┘┴ ┴┴ ┴┘└┘
`
	fmt.Println(intro)

	config, err := cli_parsing.ParseAndValidate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	summarizeWarnConfirm(config)

	for _, mapping := range config.Mappings {
		fmt.Printf("Beginning operations for \033[1;34m%s -> %s\033[0m\n", mapping.Source, mapping.Destination)

		// bake full path and normalize slashes
		sourcePath := filepath.Join(strings.TrimRight(config.SourceDir, "/\\"), strings.TrimLeft(mapping.Source, "/\\"))
		destPath := filepath.Join(strings.TrimRight(config.TargetDir, "/\\"), strings.TrimLeft(mapping.Destination, "/\\"))

		// clean if desired
		if config.CleanTarget {
			if config.DryRun {
				fmt.Println(" 🧹 [DRY RUN] Cleaning target directory...")
			} else {
				fmt.Println("  🧹 Cleaning target directory...")
				err := file_operations.ClearDirectory(destPath)
				if err != nil {
					fmt.Printf("  Error cleaning target directory: %v\n", err)
					os.Exit(1)
				}
			}
		}

		// copy files
		fmt.Println("  Beginning copy...")
		err := copy_funcs.CopyFiles(
			sourcePath,
			destPath,
			config.CopyInclude,
			config.CopyExclude,
			config.DryRun,
		)
		fmt.Println("  Copy complete!")

		if err != nil {
			fmt.Printf("  Error copying files: %v\n", err)
			os.Exit(1)
		}

		// explode dirs
		if len(config.ExplodeDirs) > 0 {
			explodeDirs(config, destPath)
		}

		// process renames
		if len(config.Renames) > 0 {
			processRenames(config, destPath)
		}

		// process rewrites
		if len(config.FileRewrites) > 0 {
			processRewrites(config, destPath)
		}

		fmt.Printf("Operations for %s -> %s complete!\n", mapping.Source, mapping.Destination)
	}

	fmt.Println("All transfers & processing completed successfully!")
}
