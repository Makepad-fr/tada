package internal

//runner.go orchestrates subcommands (add, ls, done, rm). For ls, it launches the Bubble Tea TUI.
import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	selectedStyle = lipgloss.NewStyle().Bold(true).Reverse(true)
	doneStyle     = lipgloss.NewStyle().Faint(true).Strikethrough(true)
	helpStyle     = lipgloss.NewStyle().Faint(true)

	boxChecked   = "☑"
	boxUnchecked = "☐"
)

var kb = struct {
	Toggle key.Binding
	Delete key.Binding
	Quit   key.Binding
}{
	Toggle: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle done")),
	Delete: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Quit:   key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q/esc", "quit & save")),
}

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
	items, err := Load()
	if err != nil {
		fail("load: " + err.Error())
		return 1
	}

	// Launch interactive TUI list (Bubble Tea). Saves on exit if changed.
	if err := runInteractiveList(items, opt); err != nil {
		fail("tui: " + err.Error())
		return 1
	}
	return 0
}

func doAdd(title string) int {
	items, err := Load()
	if err != nil {
		fail("load: " + err.Error())
		return 1
	}
	title = strings.TrimSpace(title)
	if title == "" {
		fail("add: empty title")
		return 2
	}
	items = append(items, Item{Title: title})
	if err := Save(items); err != nil {
		fail("save: " + err.Error())
		return 1
	}
	ok("added")
	return 0
}

func doToggle(userIndex int) int {
	items, err := Load()
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
	if err := Save(items); err != nil {
		fail("save: " + err.Error())
		return 1
	}
	ok("toggled")
	return 0
}

func doRemove(userIndex int) int {
	items, err := Load()
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
	if err := Save(items); err != nil {
		fail("save: " + err.Error())
		return 1
	}
	ok("removed")
	return 0
}

// -------------- rendering helpers --------------

func stats(items []Item) (done, pending int) {
	for _, it := range items {
		if it.Done {
			done++
		} else {
			pending++
		}
	}
	return
}

func flatLines(items []Item) []string {
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

func groupLines(items []Item) []string {
	var pend, doneItems []Item
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

// ---------------- Bubble Tea interactive list ----------------

// listItem adapts our model.Item to bubbles/list.Item
type listItem struct {
	Text string
	Done bool
}

func (i listItem) TitleText() string {
	box := boxUnchecked
	if i.Done {
		box = boxChecked
	}
	return fmt.Sprintf("%s %s", box, i.Text)
}

// Implement list.Item interface
func (i listItem) Title() string       { return i.TitleText() }
func (i listItem) Description() string { return "" }
func (i listItem) FilterValue() string { return i.Text }

type modelTUI struct {
	list     list.Model
	changed  bool
	itemsRef *[]Item // pointer to original slice to write back updates

	// Inline add
	adding bool            // true when inline add is active
	ti     textinput.Model // shared text input model (used for add & edit)
	addErr string          // last add validation error (shown briefly)

	// Inline edit
	editing   bool // true when inline edit is active
	editIndex int  // index of item being edited
	editErr   string

	// Undo support (single-level)
	canUndo   bool
	undoIndex int
	undoItem  *listItem
}

// Custom delegate to control how items render (single line)
type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, _ := item.(listItem)
	raw := it.TitleText() // e.g. "☐ Buy milk"
	// split into box and text so we style them separately
	space := strings.Index(raw, " ")
	if space < 0 {
		space = len(raw)
	}
	box, text := raw[:space], strings.TrimSpace(raw[space:])

	boxStyled := mutedStyle.Render(box)
	textStyled := text
	if it.Done {
		boxStyled = successStyle.Render(boxChecked)
		textStyled = doneStyle.Render(text)
	}

	line := fmt.Sprintf("%s %s", boxStyled, textStyled)
	prefix := "  "
	if index == m.Index() {
		prefix = selectedStyle.Render("> ")
	}
	fmt.Fprintln(w, prefix+line)
}

// runInteractiveList starts the Bubble Tea list and persists changes when quitting.
func runInteractiveList(items []Item, opt Options) error {
	// Build items for the list
	li := make([]list.Item, 0, len(items))
	for _, it := range items {
		li = append(li, listItem{Text: it.Title, Done: it.Done})
	}

	l := list.New(li, itemDelegate{}, 0, 0)

	// Header title with live counts
	dn, pn := stats(items)
	ltitle := fmt.Sprintf("%s   %s %d  %s %d  %s %d",
		titleStyle.Render("Todos"),
		successStyle.Render("✔"), dn,
		pendingStyle.Render("•"), pn,
		accentStyle.Render("Total"), len(items),
	)

	l.Title = ltitle
	l.SetShowHelp(true)
	l.SetShowPagination(true)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.HelpStyle = helpStyle
	l.Styles.PaginationStyle = helpStyle
	l.FilterInput.Prompt = "/ "
	l.SetStatusBarItemName("item", "items")

	// Extend help with Add / Edit / Undo bindings
	addBind := key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add"))
	editBind := key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit"))
	undoBind := key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "undo"))
	l.AdditionalShortHelpKeys = func() []key.Binding { return []key.Binding{addBind, editBind, undoBind, kb.Toggle, kb.Delete, kb.Quit} }
	l.AdditionalFullHelpKeys = func() []key.Binding { return []key.Binding{addBind, editBind, undoBind, kb.Toggle, kb.Delete, kb.Quit} }

	m := modelTUI{
		list:     l,
		itemsRef: &items,
	}
	// set up text input for inline add
	m.ti = textinput.New()
	m.ti.Prompt = "> "
	m.ti.Placeholder = "New item title..."
	m.ti.CharLimit = 200
	m.ti.Focus() // not adding yet, but ensures cursor works when activated

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	fm, okModel := finalModel.(modelTUI)
	if !okModel {
		return nil
	}

	// Write back list state to items and persist if changed
	if fm.changed {
		out := make([]Item, 0, len(fm.list.Items()))
		for _, it := range fm.list.Items() {
			if li, ok := it.(listItem); ok {
				out = append(out, Item{Title: li.Text, Done: li.Done})
			}
		}
		if err := Save(out); err != nil {
			return err
		}
		ok("saved")
	}
	return nil
}

// Update and View implement Bubble Tea's Model on modelTUI
func (m modelTUI) Init() tea.Cmd { return nil }

func (m modelTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// When in add mode, most keys go to the text input first
	if m.adding {
		var cmd tea.Cmd
		switch x := msg.(type) {
		case tea.KeyMsg:
			switch x.String() {
			case "enter":
				title := strings.TrimSpace(m.ti.Value())
				if title == "" {
					m.addErr = "Title cannot be empty"
					return m, nil
				}
				// append to list
				m.list.InsertItem(m.list.Index()+1, listItem{Text: title, Done: false})
				m.changed = true
				// reset input and exit add mode
				m.ti.SetValue("")
				m.ti.Blur()
				m.adding = false
				return m, nil
			case "esc":
				m.adding = false
				m.ti.SetValue("")
				m.ti.Blur()
				return m, nil
			}
		}
		m.ti, cmd = m.ti.Update(msg)
		return m, cmd
	}

	// When in edit mode, route keys to the text input
	if m.editing {
		var cmd tea.Cmd
		switch x := msg.(type) {
		case tea.KeyMsg:
			switch x.String() {
			case "enter":
				title := strings.TrimSpace(m.ti.Value())
				if title == "" {
					m.editErr = "Title cannot be empty"
					return m, nil
				}
				// apply edit to item at editIndex
				if m.editIndex >= 0 && m.editIndex < len(m.list.Items()) {
					if li, ok := m.list.Items()[m.editIndex].(listItem); ok {
						li.Text = title
						m.list.SetItem(m.editIndex, li)
						m.changed = true
					}
				}
				m.ti.SetValue("")
				m.ti.Blur()
				m.editing = false
				return m, nil
			case "esc":
				m.editing = false
				m.ti.SetValue("")
				m.ti.Blur()
				return m, nil
			}
		}
		m.ti, cmd = m.ti.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		case " ":
			// toggle done on selected
			i := m.list.Index()
			if i >= 0 && i < len(m.list.Items()) {
				if li, ok := m.list.Items()[i].(listItem); ok {
					li.Done = !li.Done
					m.list.SetItem(i, li)
					m.changed = true
				}
			}
			return m, nil
		case "d":
			i := m.list.Index()
			if i >= 0 && i < len(m.list.Items()) {
				// stash item for undo before removal
				if li, ok := m.list.Items()[i].(listItem); ok {
					tmp := li
					m.undoItem = &tmp
					m.undoIndex = i
					m.canUndo = true
				}
				m.list.RemoveItem(i)
				m.changed = true
			}
			return m, nil
		case "a":
			m.adding = true
			m.ti.SetValue("")
			m.ti.Placeholder = "New item title..."
			m.ti.Focus()
			return m, nil
		case "e":
			i := m.list.Index()
			if i >= 0 && i < len(m.list.Items()) {
				if li, ok := m.list.Items()[i].(listItem); ok {
					m.editing = true
					m.editIndex = i
					m.ti.SetValue(li.Text)
					m.ti.CursorEnd()
					m.ti.Placeholder = "Edit item title..."
					m.ti.Focus()
					return m, nil
				}
			}
			return m, nil
		case "u":
			if m.canUndo && m.undoItem != nil {
				// clamp index in case list got shorter
				idx := m.undoIndex
				if idx < 0 {
					idx = 0
				}
				if idx > len(m.list.Items()) {
					idx = len(m.list.Items())
				}
				m.list.InsertItem(idx, *m.undoItem)
				m.changed = true
				m.canUndo = false
				m.undoItem = nil
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m modelTUI) View() string {
	// Resize to terminal size every render
	w, h := widthHeight()
	// Leave space for the inline bar when active
	listHeight := h - 4
	if m.adding || m.editing {
		listHeight = h - 6
	}
	m.list.SetSize(w-2, listHeight)

	// Base view
	content := m.list.View()

	// Inline bars
	if m.adding || m.editing {
		bar := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")).Padding(0, 1)
		title := "Add new item"
		if m.editing {
			title = "Edit item"
		}
		if m.addErr != "" && m.adding {
			title += " — " + errorStyle.Render(m.addErr)
		}
		if m.editErr != "" && m.editing {
			title += " — " + errorStyle.Render(m.editErr)
		}
		inputLine := title + "\n" + m.ti.View()
		content = content + "\n" + bar.Render(inputLine)
	}

	return panelString(content)
}

// helpers for View
func panelString(inner string) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1)
	return border.Render(inner)
}

func widthHeight() (int, int) {
	w, h := 80, 24
	if tw, th, err := termSize(); err == nil {
		w, h = tw, th
	}
	return w, h
}

// portable terminal size
func termSize() (int, int, error) {
	fd := int(os.Stdout.Fd())
	type winsize struct {
		Row, Col, Xpixel, Ypixel uint16
	}
	ws := &winsize{}
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(ws)))
	if err != 0 {
		return 0, 0, fmt.Errorf("ioctl: %v", err)
	}
	return int(ws.Col), int(ws.Row), nil
}
