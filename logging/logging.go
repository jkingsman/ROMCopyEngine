package logging

import "fmt"

// log level == indentation
type LogLevel int

const (
	// 0 space
	Base LogLevel = iota
	// 2 space
	Action
	// 4 space
	Detail
)

// Icons for different types of log messages
const (
	IconCopy     = "📋"
	IconSkip     = "⏭️"
	IconFolder   = "📁"
	IconExplode  = "💥"
	IconWarning  = "⚠️"
	IconRename   = "🏷️"
	IconComplete = "✅"
	IconReplace  = "🔀"
	IconRewrite  = "🔀"
	IconClean    = "🧹"
	IconError    = "❌"
)

func getIndentation(level LogLevel) string {
	switch level {
	case Action:
		return "  "
	case Detail:
		return "    "
	default:
		return ""
	}
}

// log message with icon and level
func Log(level LogLevel, icon, message string, args ...interface{}) {
	indent := getIndentation(level)
	if icon != "" {
		fmt.Printf("%s%s %s\n", indent, icon, fmt.Sprintf(message, args...))
	} else {
		fmt.Printf("%s%s\n", indent, fmt.Sprintf(message, args...))
	}
}

// same as Log but with [DRY RUN] prefix
func LogDryRun(level LogLevel, icon, message string, args ...interface{}) {
	indent := getIndentation(level)
	if icon != "" {
		fmt.Printf("%s%s [DRY RUN] %s\n", indent, icon, fmt.Sprintf(message, args...))
	} else {
		fmt.Printf("%s[DRY RUN] %s\n", indent, fmt.Sprintf(message, args...))
	}
}

func LogWarning(message string, args ...interface{}) {
	fmt.Printf("%s WARNING %s\n", IconWarning, fmt.Sprintf(message, args...))
}

func LogComplete(message string) {
	fmt.Printf("%s%s complete!\n", getIndentation(Action), message)
}

func LogError(message string, args ...interface{}) {
	fmt.Printf("%s %s\n", IconError, fmt.Sprintf(message, args...))
}
