package visualizer

import (
	"fmt"
	"strings"
)

// ANSI color escape codes for rich terminal output
const (
	Reset       = "\033[0m"
	Bold        = "\033[1m"
	Dim         = "\033[2m"
	Italic      = "\033[3m"
	Underline   = "\033[4m"

	// Foreground colors
	FgBlack   = "\033[30m"
	FgRed     = "\033[31m"
	FgGreen   = "\033[32m"
	FgYellow  = "\033[33m"
	FgBlue    = "\033[34m"
	FgMagenta = "\033[35m"
	FgCyan    = "\033[36m"
	FgWhite   = "\033[37m"

	// Bright foreground
	FgHiBlack   = "\033[90m"
	FgHiRed     = "\033[91m"
	FgHiGreen   = "\033[92m"
	FgHiYellow  = "\033[93m"
	FgHiBlue    = "\033[94m"
	FgHiMagenta = "\033[95m"
	FgHiCyan    = "\033[96m"
	FgHiWhite   = "\033[97m"

	// Background colors
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// Color formatting functions
func Colorize(color, text string) string {
	return fmt.Sprintf("%s%s%s", color, text, Reset)
}

func Cyan(text string) string    { return Colorize(FgCyan, text) }
func Green(text string) string   { return Colorize(FgGreen, text) }
func Yellow(text string) string  { return Colorize(FgYellow, text) }
func Red(text string) string     { return Colorize(FgRed, text) }
func Blue(text string) string    { return Colorize(FgBlue, text) }
func Magenta(text string) string { return Colorize(FgMagenta, text) }
func Gray(text string) string    { return Colorize(FgHiBlack, text) }
func HiCyan(text string) string  { return Colorize(FgHiCyan, text) }
func HiGreen(text string) string { return Colorize(FgHiGreen, text) }

// SectionHeader formats a main section banner
func SectionHeader(title string) string {
	width := 70
	pad := width - len(title) - 4
	if pad < 2 {
		pad = 2
	}
	leftPad := pad / 2
	rightPad := pad - leftPad

	border := strings.Repeat("═", width)
	content := fmt.Sprintf("║ %s%s%s ║", strings.Repeat(" ", leftPad), Bold+FgHiCyan+title+Reset, strings.Repeat(" ", rightPad))

	return fmt.Sprintf("\n%s\n%s\n%s\n",
		FgCyan+border+Reset,
		content,
		FgCyan+border+Reset,
	)
}

// SubHeader formats a subsection header
func SubHeader(title string) string {
	pad := 70 - len(title)
	if pad < 4 {
		pad = 4
	}
	return fmt.Sprintf("\n%s── %s%s %s\n", FgHiYellow, Bold+FgHiWhite, title, FgHiYellow+strings.Repeat("─", pad)+Reset)
}

// Box creates an enclosed decorative box around content
func Box(title string, lines []string) string {
	maxLen := len(title)
	for _, l := range lines {
		clean := StripANSI(l)
		if len(clean) > maxLen {
			maxLen = len(clean)
		}
	}
	width := maxLen + 4
	if width < 40 {
		width = 40
	}

	var sb strings.Builder
	titlePad := width - len(title) - 4
	if titlePad < 0 {
		titlePad = 0
	}
	sb.WriteString(fmt.Sprintf("%s┌─ %s%s %s┐%s\n", FgCyan, Bold+FgHiWhite, title, FgCyan+strings.Repeat("─", titlePad), Reset))

	for _, l := range lines {
		cleanLen := len(StripANSI(l))
		pad := width - cleanLen - 2
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(fmt.Sprintf("%s│%s %s%s%s│%s\n", FgCyan, Reset, l, strings.Repeat(" ", pad), FgCyan, Reset))
	}

	sb.WriteString(fmt.Sprintf("%s└%s┘%s\n", FgCyan, strings.Repeat("─", width), Reset))
	return sb.String()
}

// Badge creates a colored badge e.g. [RUNNING] [READY]
func Badge(text, bgColor, fgColor string) string {
	return fmt.Sprintf("%s%s %s %s", bgColor, fgColor+Bold, text, Reset)
}

// StripANSI strips ANSI color and styling escape sequences
func StripANSI(str string) string {
	var sb strings.Builder
	inEscape := false
	for i := 0; i < len(str); i++ {
		if str[i] == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if str[i] == 'm' {
				inEscape = false
			}
			continue
		}
		sb.WriteByte(str[i])
	}
	return sb.String()
}
