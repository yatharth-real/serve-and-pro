package ui

import (
	"os"
	"regexp"
	"strings"
)

// ANSI formatting constants.
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"

	// Standard 16 colors
	FgBlack   = "\033[30m"
	FgRed     = "\033[31m"
	FgGreen   = "\033[32m"
	FgYellow  = "\033[33m"
	FgBlue    = "\033[34m"
	FgMagenta = "\033[35m"
	FgCyan    = "\033[36m"
	FgWhite   = "\033[37m"

	FgBrightBlack   = "\033[90m"
	FgBrightRed     = "\033[91m"
	FgBrightGreen   = "\033[92m"
	FgBrightYellow  = "\033[93m"
	FgBrightBlue    = "\033[94m"
	FgBrightMagenta = "\033[95m"
	FgBrightCyan    = "\033[96m"
	FgBrightWhite   = "\033[97m"

	// Extended 256-color codes
	FgAmber256  = "\033[38;5;214m"
	FgOrange256 = "\033[38;5;208m"
	FgViolet256 = "\033[38;5;141m"
	FgPurple256 = "\033[38;5;135m"
	FgOcean256  = "\033[38;5;45m"
	FgTeal256   = "\033[38;5;50m"
	FgForest256 = "\033[38;5;48m"
	FgMint256   = "\033[38;5;121m"
	FgSlate256  = "\033[38;5;245m"
	FgSmoke256  = "\033[38;5;240m"
	FgDark256   = "\033[38;5;236m"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes all ANSI escape sequences from a string.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// VisibleLen returns the printable length of a string without ANSI escapes.
func VisibleLen(s string) int {
	return len([]rune(StripANSI(s)))
}

// SupportsColor returns true if the current environment supports ANSI colors.
func SupportsColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	term := strings.ToLower(os.Getenv("TERM"))
	if term == "dumb" {
		return false
	}
	return true
}
