package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"carryon/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type toolFilter int

const (
	filterAll toolFilter = iota
	filterClaude
	filterCodex
)

// Model is the Bubble Tea model for the conversation picker.
type Model struct {
	all         []model.Conversation
	filtered    []int // indices into all
	cursor      int
	search      string
	searching   bool
	tool        toolFilter
	cwdOnly     bool
	currentDir  string
	width       int
	height      int
	loadPreview func(model.Conversation) string
	previewIdx  int
	previewText string
	selected    *model.Conversation
	quitting    bool
}

// New builds the initial model. loadPreview is injected for testability.
func New(convs []model.Conversation, currentDir string, loadPreview func(model.Conversation) string) Model {
	m := Model{
		all:         convs,
		currentDir:  currentDir,
		loadPreview: loadPreview,
		previewIdx:  -1,
	}
	m.recompute()
	m.refreshPreview()
	return m
}

func (m Model) Init() tea.Cmd { return nil }

// refreshPreview loads the highlighted conversation's preview into the cache if
// the cursor moved. Called from Update/New (value-receiver paths that persist),
// never from View, so the cache actually survives across renders.
func (m *Model) refreshPreview() {
	if len(m.filtered) == 0 {
		m.previewText = ""
		m.previewIdx = -1
		return
	}
	if m.previewIdx != m.cursor {
		m.previewText = m.loadPreview(m.all[m.filtered[m.cursor]])
		m.previewIdx = m.cursor
	}
}

func (m Model) toolLabel() string {
	switch m.tool {
	case filterClaude:
		return "claude"
	case filterCodex:
		return "codex"
	default:
		return "all"
	}
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	home, _ := os.UserHomeDir()

	header := fmt.Sprintf("carryon — %d conversations  [tool:%s%s]",
		len(m.filtered), m.toolLabel(), map[bool]string{true: " cwd-only", false: ""}[m.cwdOnly])
	if m.searching {
		header += "\nsearch: " + m.search
	}

	list := m.listView(home)

	footer := "↑↓ move  ↵ resume  / search  t tool  . cwd-only  q quit"

	left := lipgloss.NewStyle().Width(m.leftWidth()).Render(list)
	right := lipgloss.NewStyle().Width(m.rightWidth()).Render(m.previewHeader(home) + "\n\n" + m.previewText)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return header + "\n\n" + body + "\n\n" + footer
}

// listView renders only the rows visible in the current viewport window,
// with "more" indicators above/below when the list is taller than the screen.
func (m Model) listView(home string) string {
	if len(m.filtered) == 0 {
		return "\n  No matches.\n"
	}
	start, end := m.windowBounds()
	var b strings.Builder
	if start > 0 {
		fmt.Fprintf(&b, "  ↑ %d more\n", start)
	}
	for i := start; i < end; i++ {
		c := m.all[m.filtered[i]]
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		fmt.Fprintf(&b, "%s%-6s  %-16s  %-5s  %-8s  %s\n",
			cursor, c.Tool.String(), shortenPath(c.Cwd, home),
			humanizeAge(c.Modified, time.Now()), c.Branch, c.Title)
	}
	if end < len(m.filtered) {
		fmt.Fprintf(&b, "  ↓ %d more\n", len(m.filtered)-end)
	}
	return b.String()
}

// listHeight is how many conversation rows fit, leaving room for the header,
// footer, and scroll indicators. Returns the full count when height is unknown.
func (m Model) listHeight() int {
	h := m.height - 8
	if h < 1 {
		return len(m.filtered)
	}
	return h
}

// windowBounds returns the [start, end) slice of m.filtered to render, keeping
// the cursor in view.
func (m Model) windowBounds() (int, int) {
	lh := m.listHeight()
	if lh >= len(m.filtered) {
		return 0, len(m.filtered)
	}
	start := 0
	if m.cursor >= lh {
		start = m.cursor - lh + 1
	}
	end := start + lh
	if end > len(m.filtered) {
		end = len(m.filtered)
		start = end - lh
	}
	if start < 0 {
		start = 0
	}
	return start, end
}

func (m Model) previewHeader(home string) string {
	c := m.Current()
	if c == nil {
		return ""
	}
	warn := ""
	if _, err := os.Stat(c.Cwd); err != nil {
		warn = "  ⚠ path missing"
	}
	return fmt.Sprintf("%s · %s · %s%s",
		c.Tool.String(), c.Cwd, humanizeAge(c.Modified, time.Now()), warn)
}

func (m Model) leftWidth() int {
	if m.width <= 0 {
		return 60
	}
	return m.width * 6 / 10
}

func (m Model) rightWidth() int {
	if m.width <= 0 {
		return 40
	}
	return m.width - m.leftWidth() - 2
}

// Selected returns the chosen conversation, or nil if the user quit.
func (m Model) Selected() *model.Conversation { return m.selected }

// VisibleCount is the number of conversations after filtering (test helper).
func (m Model) VisibleCount() int { return len(m.filtered) }

// Current returns the highlighted conversation, or nil when the list is empty.
func (m Model) Current() *model.Conversation {
	if len(m.filtered) == 0 {
		return nil
	}
	c := m.all[m.filtered[m.cursor]]
	return &c
}

func (m *Model) recompute() {
	m.filtered = m.filtered[:0]
	for i, c := range m.all {
		if m.tool == filterClaude && c.Tool != model.Claude {
			continue
		}
		if m.tool == filterCodex && c.Tool != model.Codex {
			continue
		}
		if m.cwdOnly && c.Cwd != m.currentDir {
			continue
		}
		haystack := c.Tool.String() + " " + c.Cwd + " " + c.Title
		if !fuzzyMatch(m.search, haystack) {
			continue
		}
		m.filtered = append(m.filtered, i)
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.previewIdx = -1 // invalidate cached preview
}

func (m *Model) move(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
}

func (m *Model) cycleTool() {
	m.tool = (m.tool + 1) % 3
}

func (m Model) confirm() (tea.Model, tea.Cmd) {
	if c := m.Current(); c != nil {
		m.selected = c
	}
	m.quitting = true
	return m, tea.Quit
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		var next tea.Model
		var cmd tea.Cmd
		if m.searching {
			next, cmd = m.updateSearching(msg)
		} else {
			next, cmd = m.updateBrowsing(msg)
		}
		nm := next.(Model)
		nm.refreshPreview()
		return nm, cmd
	}
	return m, nil
}

func (m Model) updateSearching(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.search = ""
		m.searching = false
		m.recompute()
	case tea.KeyEnter:
		return m.confirm()
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyBackspace:
		if r := []rune(m.search); len(r) > 0 {
			m.search = string(r[:len(r)-1])
			m.recompute()
		}
	case tea.KeyUp:
		m.move(-1)
	case tea.KeyDown:
		m.move(1)
	case tea.KeyRunes:
		m.search += string(msg.Runes)
		m.recompute()
	}
	return m, nil
}

func (m Model) updateBrowsing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quitting = true
		return m, tea.Quit
	case "/":
		m.searching = true
		return m, nil
	case "t":
		m.cycleTool()
		m.recompute()
		return m, nil
	case ".":
		m.cwdOnly = !m.cwdOnly
		m.recompute()
		return m, nil
	case "j":
		m.move(1)
		return m, nil
	case "k":
		m.move(-1)
		return m, nil
	}
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyDown:
		m.move(1)
	case tea.KeyUp:
		m.move(-1)
	case tea.KeyEnter:
		return m.confirm()
	}
	return m, nil
}
