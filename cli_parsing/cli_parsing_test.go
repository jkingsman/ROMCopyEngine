package cli_parsing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAndValidate(t *testing.T) {
	// Create temporary test directories
	tmpSource := t.TempDir()
	tmpTarget := t.TempDir()

	// Create platform subdirectories
	sourceNes := filepath.Join(tmpSource, "nes")
	targetNES := filepath.Join(tmpTarget, "NES")
	sourceSnes := filepath.Join(tmpSource, "snes")
	targetSFC := filepath.Join(tmpTarget, "SFC")

	dirs := []string{sourceNes, targetNES, sourceSnes, targetSFC}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}

	tests := []struct {
		name      string
		args      []string
		wantError bool
		validate  func(*testing.T, *Config)
	}{
		{
			name: "basic valid config",
			args: []string{
				"--sourceDir", tmpSource,
				"--targetDir", tmpTarget,
				"--mapping", "nes:NES",
				"--mapping", "snes:SFC",
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if len(c.Mappings) != 2 {
					t.Errorf("Expected 2 mappings, got %d", len(c.Mappings))
				}
				if c.Mappings[0].Source != "nes" || c.Mappings[0].Destination != "NES" {
					t.Errorf("Incorrect mapping: %v", c.Mappings[0])
				}
			},
		},
		{
			name: "missing source dir",
			args: []string{
				"--sourceDir", "/nonexistent",
				"--targetDir", tmpTarget,
				"--mapping", "nes:NES",
			},
			wantError: true,
		},
		{
			name: "invalid mapping format",
			args: []string{
				"--sourceDir", tmpSource,
				"--targetDir", tmpTarget,
				"--mapping", "nes:NES:extra",
			},
			wantError: true,
		},
		{
			name: "valid rename",
			args: []string{
				"--sourceDir", tmpSource,
				"--targetDir", tmpTarget,
				"--mapping", "nes:NES",
				"--rename", "old.xml:new.xml",
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if len(c.Renames) != 1 {
					t.Errorf("Expected 1 rename, got %d", len(c.Renames))
				}
				if c.Renames[0].OldName != "old.xml" || c.Renames[0].NewName != "new.xml" {
					t.Errorf("Incorrect rename: %v", c.Renames[0])
				}
			},
		},
		{
			name: "valid rewrite",
			args: []string{
				"--sourceDir", tmpSource,
				"--targetDir", tmpTarget,
				"--mapping", "nes:NES",
				"--rewrite", "*.xml:../images:./images",
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if len(c.FileRewrites) != 1 {
					t.Errorf("Expected 1 rewrite, got %d", len(c.FileRewrites))
				}
				if c.FileRewrites[0].FileGlob != "*.xml" {
					t.Errorf("Expected file glob '*.xml', got %q", c.FileRewrites[0].FileGlob)
				}
				if c.FileRewrites[0].SearchPattern != "../images" {
					t.Errorf("Expected search pattern '../images', got %q", c.FileRewrites[0].SearchPattern)
				}
				if c.FileRewrites[0].ReplacePattern != "./images" {
					t.Errorf("Expected replace pattern './images', got %q", c.FileRewrites[0].ReplacePattern)
				}
			},
		},
		{
			name: "valid rewrite with regex",
			args: []string{
				"--sourceDir", tmpSource,
				"--targetDir", tmpTarget,
				"--mapping", "nes:NES",
				"--rewrite", "*.xml:../images:./images",
				"--rewritesAreRegex",
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if len(c.FileRewrites) != 1 {
					t.Errorf("Expected 1 rewrite, got %d", len(c.FileRewrites))
				}
				if !c.RewritesAreRegex {
					t.Error("Expected RewritesAreRegex to be true")
				}
			},
		},
		{
			name: "invalid rewrite format",
			args: []string{
				"--sourceDir", tmpSource,
				"--targetDir", tmpTarget,
				"--mapping", "nes:NES",
				"--rewrite", "*.xml:foo", // Missing replace pattern
			},
			wantError: true,
		},
		{
			name: "invalid regex pattern",
			args: []string{
				"--sourceDir", tmpSource,
				"--targetDir", tmpTarget,
				"--mapping", "nes:NES",
				"--rewrite", "*.xml:[invalid:replace",
				"--rewritesAreRegex",
			},
			wantError: true,
		},
		{
			name: "invalid rewrite format",
			args: []string{
				"--sourceDir", tmpSource,
				"--targetDir", tmpTarget,
				"--mapping", "nes:NES",
				"--rewrite", "*.xml:foo:bar:baz", // Invalid format with too many colons
				"--rewritesAreRegex",
			},
			wantError: true,
		},
		{
			name: "copy include and exclude",
			args: []string{
				"--sourceDir", tmpSource,
				"--targetDir", tmpTarget,
				"--mapping", "nes:NES",
				"--copyInclude", "*.rom",
				"--copyExclude", "*.bak",
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if len(c.CopyInclude) != 1 || c.CopyInclude[0] != "*.rom" {
					t.Errorf("Incorrect copyInclude: %v", c.CopyInclude)
				}
				if len(c.CopyExclude) != 1 || c.CopyExclude[0] != "*.bak" {
					t.Errorf("Incorrect copyExclude: %v", c.CopyExclude)
				}
			},
		},
		{
			name: "explode directories",
			args: []string{
				"--sourceDir", tmpSource,
				"--targetDir", tmpTarget,
				"--mapping", "nes:NES",
				"--explodeDir", "images",
				"--explodeDir", "manuals",
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if len(c.ExplodeDirs) != 2 {
					t.Errorf("Expected 2 explode dirs, got %d", len(c.ExplodeDirs))
				}
			},
		},
		{
			name: "clean target and dry run",
			args: []string{
				"--sourceDir", tmpSource,
				"--targetDir", tmpTarget,
				"--mapping", "nes:NES",
				"--cleanTarget",
				"--dryRun",
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if !c.CleanTarget {
					t.Error("CleanTarget should be true")
				}
				if !c.DryRun {
					t.Error("DryRun should be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set command line arguments
			os.Args = append([]string{"cmd"}, tt.args...)

			// Parse configuration
			config, err := ParseAndValidate()

			// Check error status
			if (err != nil) != tt.wantError {
				t.Errorf("ParseAndValidate() error = %v, wantError %v", err, tt.wantError)
				return
			}

			// If we don't expect an error and have a validation function, run it
			if !tt.wantError && tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestGetConfirmation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"yes", "y\n", true},
		{"no", "n\n", false},
		{"YES", "YES\n", true},
		{"NO", "NO\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a pipe to simulate stdin
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}

			// Save original stdin
			oldStdin := os.Stdin
			// Replace stdin with our pipe
			os.Stdin = r

			// Write test input
			go func() {
				defer w.Close()
				w.Write([]byte(tt.input))
			}()

			// Run the function
			result := GetConfirmation("test prompt")

			// Restore original stdin
			os.Stdin = oldStdin

			if result != tt.expected {
				t.Errorf("GetConfirmation() = %v, want %v", result, tt.expected)
			}
		})
	}
}
