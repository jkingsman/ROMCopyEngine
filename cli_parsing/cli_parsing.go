package cli_parsing

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alecthomas/kong"
)

type CLI struct {
	SourceDir        string   `help:"the source directory containing platform folders ('snes', 'gba', etc.) to be copied from e.g. 'C:\\ROMS' or '/home/ROMS'" name:"sourceDir" type:"path" required:""`
	TargetDir        string   `help:"target directory (usually on device) containing platform folders ('snes', 'gba', etc.), e.g. 'J:\\' or '/media/usb-drive/'" name:"targetDir" type:"path" required:""`
	Mappings         []string `help:"a mapping of source platform folder to destination platform folder for the ROMs in the format 'source:destination'. For example, '--mapping snes:SFC --mapping gg:GameGear' would copy the contents of the sourceDir's 'snes' folder to the targetDir's 'SFC' folder and the contents of the sourceDir's 'gg' folder to the targetDir's 'GameGear' folder." name:"mapping" required:"" type:"string"`
	Renames          []string `help:"rename files or folders from a given name to a given name after copy. For example, '--rename gameslist.xml:miyoogameslist.xml' would rename all occurrences of 'gameslist.xml' in all folders to 'miyoogameslist.xml'; '--rename images:Imgs' could be used to rename image folders. Multiples of this flag are allowed." name:"rename" type:"string"`
	CopyInclude      []string `help:"copy only files and folders within each mapping which match the given glob (for example, '--copyInclude '*_favorite*'' would only copy files/folders from each source folder containing the string 'favorite'; '--copyInclude '*.xml' would only copy XML files found in each source folder. Remember to single quote your glob to prevent shell expansion. Multiples of this flag are allowed, and will be processed as an OR relation (files matching any --copyInclude will be included). This supports globstar (e.g. '--copyInclude **/*.png' copies PNGs from all child directories, whereas '--copyInclude *.png' only copies top-level PNGs in the platform root)." name:"copyInclude" type:"string"`
	CopyExclude      []string `help:"copy only files and folders within each mapping which do NOT match the given glob (for example, '--copyExclude '*.xml'' would copy all files and folders except those ending in '.xml'. Remember to single quote your glob to prevent shell expansion. Multiples of this flag are allowed, and will be processed as an AND relation (files matching any --copyExclude will be excluded). '--copyExclude' entries are processed after '--copyExclude' entries" name:"copyExclude" type:"string"`
	ExplodeDirs      []string `help:"provides a directory name contained in a ROM folder that should have its contents copied to the parent directory for that system, then delete the empty folder. For example, '--explodeDir images' would copy the contents of the image directory into its parent folder. Commonly used to bring boxart images out of an 'images' directory and onto the same level as ROMs. Multiples of this flag are allowed." name:"explodeDir" type:"string"`
	FileRewrites     []string `help:"for a given file glob, execute a find and replace on all matching files in the format <glob>:<search term>:<replace term>. Useful for fixing paths in XML files. Remember to single quote your globs to prevent shell expansion and don't glob '*' unless you want to rewrite binary ROMs. For example, '--rewrite '*.xml:../images:./images'' would replace all occurrences of the string '../images' to './images' in all XML files. Multiples of this flag are allowed." name:"rewrite" type:"string"`
	RewritesAreRegex bool     `help:"when set, the search term in any --rewrite flag is interpreted as a Golang regular expression" optional:"" name:"rewritesAreRegex"`
	CleanTarget      bool     `help:"delete all files in the destination platform folder before copying ROMs in" optional:"" name:"cleanTarget"`
	SkipConfirm      bool     `help:'skip all confirmations and execute the copy process' optional:"" name:"skipConfirm"`
	DryRun           bool     `help:"don't execute any file copies or operations; just print what would be done" optional:"" name:"dryRun"`
}

type Config struct {
	SourceDir        string
	TargetDir        string
	Mappings         []DirMapping
	Renames          []NameMapping
	CopyInclude      []string
	CopyExclude      []string
	ExplodeDirs      []string
	FileRewrites     []RewriteRule
	RewritesAreRegex bool
	CleanTarget      bool
	SkipConfirm      bool
	DryRun           bool
}

type DirMapping struct {
	Source      string
	Destination string
}

type NameMapping struct {
	OldName string
	NewName string
}

type RewriteRule struct {
	FileGlob       string
	SearchPattern  string
	ReplacePattern string
}

func (c *Config) Validate() error {
	if c.SourceDir == "" {
		return fmt.Errorf("source directory is required")
	}

	if c.TargetDir == "" {
		return fmt.Errorf("target directory is required")
	}

	// Validate mappings
	if len(c.Mappings) == 0 {
		return fmt.Errorf("at least one mapping is required")
	}

	return nil
}

func ParseAndValidate() (*Config, error) {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("ROMCopyEngine"),
		kong.Description("A tool for copying and transforming game ROM directories. See more at https://github.com/jkingsman/ROMCopyEngine."),
		kong.UsageOnError(),
	)

	if err := ctx.Validate(); err != nil {
		return nil, fmt.Errorf("invalid command line arguments: %w", err)
	}

	config := &Config{
		SourceDir:        filepath.Clean(cli.SourceDir),
		TargetDir:        filepath.Clean(cli.TargetDir),
		CopyInclude:      cli.CopyInclude,
		CopyExclude:      cli.CopyExclude,
		ExplodeDirs:      cli.ExplodeDirs,
		RewritesAreRegex: cli.RewritesAreRegex,
		CleanTarget:      cli.CleanTarget,
		SkipConfirm:      cli.SkipConfirm,
		DryRun:           cli.DryRun,
	}

	// Validate source directory exists
	if !isDirExists(config.SourceDir) {
		return nil, fmt.Errorf("source directory does not exist: %s", config.SourceDir)
	}

	// Parse mappings
	config.Mappings = make([]DirMapping, 0, len(cli.Mappings))
	for _, mapping := range cli.Mappings {
		parts := strings.Split(mapping, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid mapping format '%s': must be in format 'source:destination'", mapping)
		}

		sourcePath := filepath.Join(config.SourceDir, parts[0])
		if !isDirExists(sourcePath) {
			return nil, fmt.Errorf("source mapping directory does not exist: %s", sourcePath)
		}

		config.Mappings = append(config.Mappings, DirMapping{
			Source:      parts[0],
			Destination: parts[1],
		})
	}

	// Parse renames
	config.Renames = make([]NameMapping, 0, len(cli.Renames))
	for _, rename := range cli.Renames {
		parts := strings.Split(rename, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid rename format '%s': must be in format 'old:new'", rename)
		}

		config.Renames = append(config.Renames, NameMapping{
			OldName: parts[0],
			NewName: parts[1],
		})
	}

	// Parse file rewrites
	config.FileRewrites = make([]RewriteRule, 0, len(cli.FileRewrites))
	for _, rewrite := range cli.FileRewrites {
		parts := strings.Split(rewrite, ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid rewrite format '%s': must be in format 'glob:search:replace'", rewrite)
		}

		// If using regex, validate the pattern
		if cli.RewritesAreRegex {
			if _, err := regexp.Compile(parts[1]); err != nil {
				return nil, fmt.Errorf("invalid regex pattern '%s': %w", parts[1], err)
			}
		}

		config.FileRewrites = append(config.FileRewrites, RewriteRule{
			FileGlob:       parts[0],
			SearchPattern:  parts[1],
			ReplacePattern: parts[2],
		})
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func PrintCLIOpts(config *Config) {
	fmt.Println()
	fmt.Println("==== Configuration ====")
	fmt.Println()

	fmt.Printf("Copy sources and destinations:\n")
	for _, m := range config.Mappings {
		fmt.Printf("  %s -> %s\n", filepath.Join(config.SourceDir, m.Source), filepath.Join(config.TargetDir, m.Destination))
	}

	if len(config.Renames) > 0 {
		fmt.Printf("Renames:\n")
		for _, r := range config.Renames {
			fmt.Printf("  • All files named %s will be renamed to %s\n", r.OldName, r.NewName)
		}
	}

	if len(config.ExplodeDirs) > 0 {
		fmt.Printf("Exploded directories:\n")
		for _, e := range config.ExplodeDirs {
			fmt.Printf("  • All directories named %s will have their contents copied to the parent platform folder\n", e)
		}
	}

	if len(config.FileRewrites) > 0 {
		if config.RewritesAreRegex {
			fmt.Println("Regex file rewrites:")
		} else {
			fmt.Println("Literal file rewrites:")
		}

		fmt.Printf("Rewrites:\n")
		for _, r := range config.FileRewrites {
			fmt.Printf("  • All files matching glob '%s' will have %s replaced with %s\n", r.FileGlob, r.SearchPattern, r.ReplacePattern)
		}
	}

	if len(config.CopyInclude) > 0 || len(config.CopyExclude) > 0 {
		fmt.Println("Copies:")
	}
	if len(config.CopyInclude) > 0 {
		fmt.Println("• Copy will include files/folders matching any of:")
		for _, c := range config.CopyInclude {
			fmt.Printf("  • %s\n", c)
		}
	}

	if len(config.CopyExclude) > 0 {
		fmt.Println("• Copy will exclude files/folders matching any of:")
		for _, c := range config.CopyExclude {
			fmt.Printf("  • %s\n", c)
		}
	}

	if config.CleanTarget {
		fmt.Println("Target directory will be cleaned before copying")
	}

	if config.DryRun {
		fmt.Println("Dry run mode enabled; no files will be copied or modified")
	}

	if config.SkipConfirm {
		fmt.Println("Skip-confirm enabled; no warnings given before proceeding")
	}

	fmt.Println()

	fmt.Printf("==== End Configuration ====\n")
}

func GetConfirmation(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", prompt)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Please enter 'y' or 'n'")
		}
	}
}

func isDirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
