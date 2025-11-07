package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ------- minimal styling helpers (Lip Gloss) -------
var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	pendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	accentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	mutedStyle   = lipgloss.NewStyle().Faint(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)

	selectedStyle = lipgloss.NewStyle().Bold(true).Reverse(true)
	doneStyle     = lipgloss.NewStyle().Faint(true).Strikethrough(true)
	helpStyle     = lipgloss.NewStyle().Faint(true)

	boxChecked   = "☑"
	boxUnchecked = "☐"
)

func ok(msg string) {
	fmt.Println(successStyle.Render("✔ " + msg))
}
func fail(msg string) {
	fmt.Fprintln(os.Stderr, errorStyle.Render("✖ "+msg))
}

func panel(lines []string) {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1)
	fmt.Println(border.Render(strings.Join(lines, "\n")))
}

func progressBar(done, total, width int) string {
	if total == 0 {
		total = 1
	}
	if width <= 0 {
		width = 28
	}
	filled := int(float64(done) / float64(total) * float64(width))
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + fmt.Sprintf("] %d/%d", done, total)
}
