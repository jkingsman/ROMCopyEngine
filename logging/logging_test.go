package logging

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// captureOutput captures stdout during the execution of f and returns it as a string
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	w.Close()
	os.Stdout = old
	return <-outC
}

func TestGetIndentation(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{Base, ""},
		{Action, "  "},
		{Detail, "    "},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("LogLevel_%d", tt.level), func(t *testing.T) {
			if got := getIndentation(tt.level); got != tt.expected {
				t.Errorf("getIndentation(%d) = %q, want %q", tt.level, got, tt.expected)
			}
		})
	}
}

func TestLog(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		icon     string
		message  string
		args     []interface{}
		expected string
	}{
		{
			name:     "Base level without icon",
			level:    Base,
			icon:     "",
			message:  "Test message",
			args:     nil,
			expected: "Test message\n",
		},
		{
			name:     "Action level with icon",
			level:    Action,
			icon:     IconCopy,
			message:  "Copying %s",
			args:     []interface{}{"test.txt"},
			expected: "  📋 Copying test.txt\n",
		},
		{
			name:     "Detail level with formatting",
			level:    Detail,
			icon:     IconFolder,
			message:  "Created dir: %s/%s",
			args:     []interface{}{"path", "subdir"},
			expected: "    📁 Created dir: path/subdir\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				Log(tt.level, tt.icon, tt.message, tt.args...)
			})
			if output != tt.expected {
				t.Errorf("Log() output = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestLogDryRun(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		icon     string
		message  string
		args     []interface{}
		expected string
	}{
		{
			name:     "Dry run with icon",
			level:    Detail,
			icon:     IconCopy,
			message:  "Copying %s to %s",
			args:     []interface{}{"source.txt", "dest.txt"},
			expected: "    📋 [DRY RUN] Copying source.txt to dest.txt\n",
		},
		{
			name:     "Dry run without icon",
			level:    Action,
			icon:     "",
			message:  "Processing directory %s",
			args:     []interface{}{"test"},
			expected: "  [DRY RUN] Processing directory test\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				LogDryRun(tt.level, tt.icon, tt.message, tt.args...)
			})
			if output != tt.expected {
				t.Errorf("LogDryRun() output = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestLogWarning(t *testing.T) {
	output := captureOutput(func() {
		LogWarning("Test warning: %s", "caution")
	})
	expected := "⚠️ WARNING Test warning: caution\n"
	if output != expected {
		t.Errorf("LogWarning() output = %q, want %q", output, expected)
	}
}

func TestLogComplete(t *testing.T) {
	output := captureOutput(func() {
		LogComplete("Test operation")
	})
	expected := "  Test operation complete!\n"
	if output != expected {
		t.Errorf("LogComplete() output = %q, want %q", output, expected)
	}
}

func TestLogError(t *testing.T) {
	output := captureOutput(func() {
		LogError("Error occurred: %s", "test error")
	})
	expected := "❌ Error occurred: test error\n"
	if output != expected {
		t.Errorf("LogError() output = %q, want %q", output, expected)
	}
}

func TestIconConstants(t *testing.T) {
	// Test that all icon constants are non-empty and unique
	icons := map[string]string{
		"IconCopy":     IconCopy,
		"IconSkip":     IconSkip,
		"IconFolder":   IconFolder,
		"IconExplode":  IconExplode,
		"IconWarning":  IconWarning,
		"IconRename":   IconRename,
		"IconComplete": IconComplete,
		"IconReplace":  IconReplace,
		"IconRewrite":  IconRewrite,
		"IconClean":    IconClean,
		"IconError":    IconError,
	}

	// Check for empty icons
	for name, icon := range icons {
		if strings.TrimSpace(icon) == "" {
			t.Errorf("Icon %s is empty", name)
		}
	}

	// Check for duplicate icons (except IconReplace and IconRewrite which are intentionally the same)
	seen := make(map[string]string)
	for name, icon := range icons {
		if prev, exists := seen[icon]; exists && name != "IconRewrite" && prev != "IconReplace" {
			t.Errorf("Duplicate icon %s found for %s and %s", icon, prev, name)
		}
		seen[icon] = name
	}
}
