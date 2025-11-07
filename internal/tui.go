package internal

import (
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// listItem adapts our Item to bubbles/list.Item
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
	l.AdditionalShortHelpKeys = func() []key.Binding { return []key.Binding{addBind, editBind, undoBind} }
	l.AdditionalFullHelpKeys = func() []key.Binding { return []key.Binding{addBind, editBind, undoBind} }

	m := modelTUI{
		list:     l,
		itemsRef: &items,
	}
	// set up text input for inline add/edit
	m.ti = textinput.New()
	m.ti.Prompt = "> "
	m.ti.Placeholder = "New item title..."
	m.ti.CharLimit = 200

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
	// add mode
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
				m.list.InsertItem(m.list.Index()+1, listItem{Text: title, Done: false})
				m.changed = true
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

	// edit mode
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
	w, h := widthHeight()
	listHeight := h - 4
	if m.adding || m.editing {
		listHeight = h - 6
	}
	m.list.SetSize(w-2, listHeight)

	content := m.list.View()
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

// small list stats used for the header
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
