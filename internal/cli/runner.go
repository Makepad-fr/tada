package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idilsaglam/todo/internal/model"
	"github.com/idilsaglam/todo/internal/store/jsonstore"
)

// Options tune output behavior from root flags.
type Options struct {
	Group bool // list grouped by pending/done
}

// ------- minimal styling helpers (Lip Gloss) -------
var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	pendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	accentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	mutedStyle   = lipgloss.NewStyle().Faint(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
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

// ---------------------------------------------------

// Run dispatches subcommands and returns an exit code (0 ok, 1 error, 2 usage).
func Run(args []string, opt Options) int {
	if len(args) == 0 {
		PrintHelp()
		return 2
	}
	cmd, a := args[0], args[1:]

	switch cmd {
	case "help", "-h", "--help":
		PrintHelp()
		return 0

	case "ls":
		return doList(opt)

	case "add":
		if len(a) == 0 {
			fail("usage: todo add <title...>")
			return 2
		}
		return doAdd(strings.Join(a, " "))

	case "done":
		if len(a) != 1 {
			fail("usage: todo done <index>")
			return 2
		}
		n, err := strconv.Atoi(a[0])
		if err != nil {
			fail("done: not a number: " + a[0])
			return 2
		}
		return doToggle(n)

	case "rm":
		if len(a) != 1 {
			fail("usage: todo rm <index>")
			return 2
		}
		n, err := strconv.Atoi(a[0])
		if err != nil {
			fail("rm: not a number: " + a[0])
			return 2
		}
		return doRemove(n)
	}

	fail("unknown subcommand: " + cmd)
	fmt.Fprintln(os.Stderr)
	PrintHelp()
	return 2
}

func PrintHelp() {
	fmt.Printf(`todo - a tiny CLI

Usage:
  todo <subcommand> [args]

Subcommands:
  add <title...>     Add a new item (title can be multiple words)
  ls                 List items
  done <index>       Toggle done for item at 1-based index
  rm <index>         Remove item at 1-based index

Examples:
  todo add "Buy milk"
  todo ls
  todo done 2
  todo rm 3
`)
}

// -------------- subcommand impls ----------------

func doList(opt Options) int {
	items, err := jsonstore.Load()
	if err != nil {
		fail("load: " + err.Error())
		return 1
	}

	// Header + progress
	d, p := stats(items)
	header := fmt.Sprintf("%s  %s %d  %s %d  %s %d",
		titleStyle.Render("Todos"),
		successStyle.Render("✔"), d,
		pendingStyle.Render("•"), p,
		accentStyle.Render("Total"), len(items),
	)

	var lines []string
	lines = append(lines, header)
	lines = append(lines, mutedStyle.Render(progressBar(d, d+p, 28)))
	lines = append(lines, "")

	if opt.Group {
		lines = append(lines, groupLines(items)...)
	} else {
		lines = append(lines, flatLines(items)...)
	}
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render("Tip: add with `todo add \"Buy milk\"`"))
	panel(lines)
	return 0
}

func doAdd(title string) int {
	items, err := jsonstore.Load()
	if err != nil {
		fail("load: " + err.Error())
		return 1
	}
	title = strings.TrimSpace(title)
	if title == "" {
		fail("add: empty title")
		return 2
	}
	items = append(items, model.Item{Title: title})
	if err := jsonstore.Save(items); err != nil {
		fail("save: " + err.Error())
		return 1
	}
	ok("added")
	return 0
}

func doToggle(userIndex int) int {
	items, err := jsonstore.Load()
	if err != nil {
		fail("load: " + err.Error())
		return 1
	}
	if userIndex < 1 || userIndex > len(items) {
		fail(fmt.Sprintf("index out of range: have %d, got %d", len(items), userIndex))
		fmt.Fprintln(os.Stderr, mutedStyle.Render("Hint: run `todo ls` to see valid indexes"))
		return 2
	}
	idx := userIndex - 1
	items[idx].Done = !items[idx].Done
	if err := jsonstore.Save(items); err != nil {
		fail("save: " + err.Error())
		return 1
	}
	ok("toggled")
	return 0
}

func doRemove(userIndex int) int {
	items, err := jsonstore.Load()
	if err != nil {
		fail("load: " + err.Error())
		return 1
	}
	if userIndex < 1 || userIndex > len(items) {
		fail(fmt.Sprintf("index out of range: have %d, got %d", len(items), userIndex))
		fmt.Fprintln(os.Stderr, mutedStyle.Render("Hint: run `todo ls` to see valid indexes"))
		return 2
	}
	idx := userIndex - 1
	items = append(items[:idx], items[idx+1:]...)
	if err := jsonstore.Save(items); err != nil {
		fail("save: " + err.Error())
		return 1
	}
	ok("removed")
	return 0
}

// -------------- rendering helpers --------------

func stats(items []model.Item) (done, pending int) {
	for _, it := range items {
		if it.Done {
			done++
		} else {
			pending++
		}
	}
	return
}

func flatLines(items []model.Item) []string {
	if len(items) == 0 {
		return []string{mutedStyle.Render("no items")}
	}
	out := make([]string, 0, len(items))
	for i, it := range items {
		idx := fmt.Sprintf("%2d.", i+1)
		box := boxUnchecked
		style := mutedStyle
		if it.Done {
			box, style = boxChecked, successStyle
		}
		title := it.Title
		if len(title) > 80 {
			title = title[:77] + "..."
		}
		out = append(out, fmt.Sprintf("%s %s %s",
			mutedStyle.Render(idx), style.Render(box), title))
	}
	return out
}

func groupLines(items []model.Item) []string {
	var pend, doneItems []model.Item
	for _, it := range items {
		if it.Done {
			doneItems = append(doneItems, it)
		} else {
			pend = append(pend, it)
		}
	}
	var lines []string
	lines = append(lines, accentStyle.Render("Pending"))
	if len(pend) == 0 {
		lines = append(lines, mutedStyle.Render("(none)"))
	} else {
		lines = append(lines, flatLines(pend)...)
	}
	lines = append(lines, "")
	lines = append(lines, accentStyle.Render("Done"))
	if len(doneItems) == 0 {
		lines = append(lines, mutedStyle.Render("(none)"))
	} else {
		lines = append(lines, flatLines(doneItems)...)
	}
	return lines
}
