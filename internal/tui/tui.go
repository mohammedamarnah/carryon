package tui

import (
	"carryon/internal/model"
	tea "github.com/charmbracelet/bubbletea"
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
	return m
}

func (m Model) Init() tea.Cmd { return nil }

// View satisfies tea.Model; the full implementation is added in Task 12.
func (m Model) View() string { return "" }

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
		if m.searching {
			return m.updateSearching(msg)
		}
		return m.updateBrowsing(msg)
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
