package ui

import "strings"

// Theme bundles palette + symbols + box borders.
// All UI helpers pull from `current`.
type Theme struct {
	Title, Muted, Accent, Success, Error, Pending string
	BoxUnchecked, BoxChecked                      string
	CornerTL, CornerTR, CornerBL, CornerBR        string
	H, V                                          string
	SymDone, SymUnchecked                         string
}

var current Theme

func SetTheme(name string) {
	switch strings.ToLower(name) {
	case "neon":
		current = Theme{
			Title: "\033[95m", // bright magenta
			Muted: fgGray, Accent: "\033[96m",
			Success: fgGreen, Error: fgRed, Pending: "\033[93m",
			BoxUnchecked: "◻", BoxChecked: "◼",
			CornerTL: "╭", CornerTR: "╮", CornerBL: "╰", CornerBR: "╯",
			H: "─", V: "│",
			SymDone: "✔", SymUnchecked: "•",
		}
	case "mono":
		disableColor = true
		current = Theme{
			Title: "", Muted: "", Accent: "", Success: "", Error: "", Pending: "",
			BoxUnchecked: "[ ]", BoxChecked: "[x]",
			CornerTL: "+", CornerTR: "+", CornerBL: "+", CornerBR: "+",
			H: "-", V: "|",
			SymDone: "x", SymUnchecked: "-",
		}
	default: // classic
		current = Theme{
			Title: bold, Muted: fgGray, Accent: fgBlue,
			Success: fgGreen, Error: fgRed, Pending: fgYellow,
			BoxUnchecked: "☐", BoxChecked: "☑",
			CornerTL: "┌", CornerTR: "┐", CornerBL: "└", CornerBR: "┘",
			H: "─", V: "│",
			SymDone: "✔", SymUnchecked: "•",
		}
	}
}

// Expose what renderers need
func Current() Theme { return current }
