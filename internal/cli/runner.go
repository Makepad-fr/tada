package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/idilsaglam/todo/internal/model"
	"github.com/idilsaglam/todo/internal/store/jsonstore"
	"github.com/idilsaglam/todo/internal/ui"
)

// Options tune output behavior from root flags.
type Options struct {
	Group bool // list grouped by pending/done
}

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
			ui.Fail("usage: todo add <title...>")
			return 2
		}
		return doAdd(strings.Join(a, " "))

	case "done":
		if len(a) != 1 {
			ui.Fail("usage: todo done <index>")
			return 2
		}
		n, err := strconv.Atoi(a[0])
		if err != nil {
			ui.Fail("done: not a number: " + a[0])
			return 2
		}
		return doToggle(n)

	case "rm":
		if len(a) != 1 {
			ui.Fail("usage: todo rm <index>")
			return 2
		}
		n, err := strconv.Atoi(a[0])
		if err != nil {
			ui.Fail("rm: not a number: " + a[0])
			return 2
		}
		return doRemove(n)
	}

	ui.Fail("unknown subcommand: " + cmd)
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
		ui.Fail("load: " + err.Error())
		return 1
	}

	// Header + progress
	d, p := stats(items)
	header := fmt.Sprintf("%s  %s %d  %s %d  %s %d",
		ui.C(ui.Current().Title, "Todos"),
		ui.C(ui.Current().Success, "✔"), d,
		ui.C(ui.Current().Pending, "•"), p,
		ui.C(ui.Current().Accent, "Total"), len(items),
	)

	var lines []string
	lines = append(lines, header)
	lines = append(lines, ui.C(ui.Current().Muted, ui.ProgressBar(d, d+p, 28)))
	lines = append(lines, "")

	if opt.Group {
		lines = append(lines, groupLines(items)...)
	} else {
		lines = append(lines, flatLines(items)...)
	}
	lines = append(lines, "")
	lines = append(lines, ui.C(ui.Current().Muted, "Tip: add with `todo add \"Buy milk\"`"))
	ui.Panel(lines)
	return 0
}

func doAdd(title string) int {
	items, err := jsonstore.Load()
	if err != nil {
		ui.Fail("load: " + err.Error())
		return 1
	}
	title = strings.TrimSpace(title)
	if title == "" {
		ui.Fail("add: empty title")
		return 2
	}
	items = append(items, model.Item{Title: title})
	if err := jsonstore.Save(items); err != nil {
		ui.Fail("save: " + err.Error())
		return 1
	}
	ui.OK("added")
	return 0
}

func doToggle(userIndex int) int {
	items, err := jsonstore.Load()
	if err != nil {
		ui.Fail("load: " + err.Error())
		return 1
	}
	if userIndex < 1 || userIndex > len(items) {
		ui.Fail(fmt.Sprintf("index out of range: have %d, got %d", len(items), userIndex))
		fmt.Fprintln(os.Stderr, ui.C("\033[90m", "Hint: run `todo ls` to see valid indexes"))
		return 2
	}
	idx := userIndex - 1
	items[idx].Done = !items[idx].Done
	if err := jsonstore.Save(items); err != nil {
		ui.Fail("save: " + err.Error())
		return 1
	}
	ui.OK("toggled")
	return 0
}

func doRemove(userIndex int) int {
	items, err := jsonstore.Load()
	if err != nil {
		ui.Fail("load: " + err.Error())
		return 1
	}
	if userIndex < 1 || userIndex > len(items) {
		ui.Fail(fmt.Sprintf("index out of range: have %d, got %d", len(items), userIndex))
		fmt.Fprintln(os.Stderr, ui.C("\033[90m", "Hint: run `todo ls` to see valid indexes"))
		return 2
	}
	idx := userIndex - 1
	items = append(items[:idx], items[idx+1:]...)
	if err := jsonstore.Save(items); err != nil {
		ui.Fail("save: " + err.Error())
		return 1
	}
	ui.OK("removed")
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
		return []string{ui.C(ui.Current().Muted, "no items")}
	}
	out := make([]string, 0, len(items))
	for i, it := range items {
		idx := fmt.Sprintf("%2d.", i+1)
		box := ui.Current().BoxUnchecked
		color := ui.Current().Muted
		if it.Done {
			box, color = ui.Current().BoxChecked, ui.Current().Success
		}
		title := it.Title
		if len(title) > 80 {
			title = title[:77] + "..."
		}
		out = append(out, fmt.Sprintf("%s %s %s",
			ui.C("\033[2m", idx), ui.C(color, box), title))
	}
	return out
}

func groupLines(items []model.Item) []string {
	var pend, done []model.Item
	for _, it := range items {
		if it.Done {
			done = append(done, it)
		} else {
			pend = append(pend, it)
		}
	}
	var lines []string
	lines = append(lines, ui.C(ui.Current().Accent, "Pending"))
	if len(pend) == 0 {
		lines = append(lines, ui.C(ui.Current().Muted, "(none)"))
	} else {
		lines = append(lines, flatLines(pend)...)
	}
	lines = append(lines, "")
	lines = append(lines, ui.C(ui.Current().Accent, "Done"))
	if len(done) == 0 {
		lines = append(lines, ui.C(ui.Current().Muted, "(none)"))
	} else {
		lines = append(lines, flatLines(done)...)
	}
	return lines
}
