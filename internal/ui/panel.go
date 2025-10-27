package ui

import (
	"fmt"
	"regexp"
	"strings"
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRegexp.ReplaceAllString(s, "") }

// ProgressBar renders a Unicode progress bar with percentage.
func ProgressBar(done, total, width int) string {
	if total <= 0 {
		total = 1
	}
	if width < 5 {
		width = 5
	}
	filled := int(float64(done) / float64(total) * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	pct := int(float64(done) / float64(total) * 100)
	return fmt.Sprintf("%s %3d%%", bar, pct)
}

// Panel draws a framed box using the current theme.
func Panel(lines []string) {
	t := Current()
	// compute visible width
	maxw := 0
	for _, ln := range lines {
		w := len(stripANSI(ln))
		if w > maxw {
			maxw = w
		}
	}
	pad := func(s string) string {
		vis := len(stripANSI(s))
		if vis < maxw {
			s = s + strings.Repeat(" ", maxw-vis)
		}
		return s
	}
	leftPad := " "
	fmt.Println(t.CornerTL + strings.Repeat(t.H, maxw+2) + t.CornerTR)
	for _, ln := range lines {
		fmt.Println(t.V + leftPad + pad(ln) + " " + t.V)
	}
	fmt.Println(t.CornerBL + strings.Repeat(t.H, maxw+2) + t.CornerBR)
}
