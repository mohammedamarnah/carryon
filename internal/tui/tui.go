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

// --- teal theme ---

var (
	colTeal       = lipgloss.Color("#14B8A6")
	colTealBright = lipgloss.Color("#5EEAD4")
	colTealDim    = lipgloss.Color("#0F766E")
	colMuted      = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#94A3B8"}
	colFg         = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#E5E7EB"}
	colSelFg      = lipgloss.Color("#042F2E") // dark text that reads on a teal background
	colWarn       = lipgloss.Color("#F59E0B")

	styTitle    = lipgloss.NewStyle().Foreground(colTealBright).Bold(true)
	styCount    = lipgloss.NewStyle().Foreground(colMuted)
	stySearch   = lipgloss.NewStyle().Foreground(colTealBright).Bold(true)
	styTool     = lipgloss.NewStyle().Foreground(colTeal).Bold(true)
	styProj     = lipgloss.NewStyle().Foreground(colFg)
	styAge      = lipgloss.NewStyle().Foreground(colMuted)
	styBranch   = lipgloss.NewStyle().Foreground(colTealDim)
	styRowTitle = lipgloss.NewStyle().Foreground(colFg)
	stySelected = lipgloss.NewStyle().Background(colTeal).Foreground(colSelFg).Bold(true)
	styMore     = lipgloss.NewStyle().Foreground(colTealDim).Italic(true)
	styPrevHdr  = lipgloss.NewStyle().Foreground(colTeal).Bold(true)
	styPrev     = lipgloss.NewStyle().Foreground(colFg)
	styWarn     = lipgloss.NewStyle().Foreground(colWarn).Bold(true)
	styFooter   = lipgloss.NewStyle().Foreground(colMuted)

	styFrame = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colTeal).
			Padding(0, 1)
	styRightPane = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(colTealDim).
			PaddingLeft(1)
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	home, _ := os.UserHomeDir()

	cwd := ""
	if m.cwdOnly {
		cwd = " · cwd-only"
	}
	header := styTitle.Render("carryon") + "  " +
		styCount.Render(fmt.Sprintf("%d conversations · tool: %s%s", len(m.filtered), m.toolLabel(), cwd))
	if m.searching {
		header += "\n" + stySearch.Render("search: "+m.search+"▏")
	}

	bodyH := m.bodyHeight()
	left := lipgloss.NewStyle().Width(m.leftWidth()).Height(bodyH).Render(m.listView(home))
	preview := clampLines(m.previewHeader(home)+"\n\n"+styPrev.Render(m.previewText), bodyH)
	right := styRightPane.Width(m.rightWidth()).Height(bodyH).Render(preview)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	footer := styFooter.Render("↑↓ move · ↵ resume · / search · t tool · . cwd-only · q quit")

	content := lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", footer)
	return styFrame.Width(m.innerWidth()).Render(content)
}

// listView renders only the rows visible in the current viewport window,
// with "more" indicators above/below when the list is taller than the screen.
func (m Model) listView(home string) string {
	if len(m.filtered) == 0 {
		return styMore.Render("\n  No matches.")
	}
	start, end := m.windowBounds()
	w := m.leftWidth()
	var b strings.Builder
	if start > 0 {
		b.WriteString(styMore.Render(fmt.Sprintf("  ↑ %d more", start)) + "\n")
	}
	for i := start; i < end; i++ {
		b.WriteString(m.renderRow(m.all[m.filtered[i]], home, i == m.cursor, w))
		b.WriteByte('\n')
	}
	if end < len(m.filtered) {
		b.WriteString(styMore.Render(fmt.Sprintf("  ↓ %d more", len(m.filtered)-end)))
	}
	return b.String()
}

// renderRow formats one conversation line, fit to width. The selected row gets a
// full-width teal highlight; others get per-column teal accents.
func (m Model) renderRow(c model.Conversation, home string, selected bool, width int) string {
	const (
		wTool   = 6
		wProj   = 16
		wAge    = 4
		wBranch = 8
		// marker(2) + 4 two-space gaps + the fixed columns
		fixed = 2 + wTool + 2 + wProj + 2 + wAge + 2 + wBranch + 2
	)
	titleW := width - fixed
	if titleW < 4 {
		titleW = 4
	}
	marker := "  "
	if selected {
		marker = "› "
	}
	tool := padTrunc(c.Tool.String(), wTool)
	proj := padTrunc(shortenPath(c.Cwd, home), wProj)
	age := padTrunc(humanizeAge(c.Modified, time.Now()), wAge)
	branch := padTrunc(c.Branch, wBranch)
	title := padTrunc(c.Title, titleW)

	if selected {
		plain := marker + tool + "  " + proj + "  " + age + "  " + branch + "  " + title
		return stySelected.Render(padTrunc(plain, width))
	}
	return marker +
		styTool.Render(tool) + "  " +
		styProj.Render(proj) + "  " +
		styAge.Render(age) + "  " +
		styBranch.Render(branch) + "  " +
		styRowTitle.Render(title)
}

func (m Model) previewHeader(home string) string {
	c := m.Current()
	if c == nil {
		return ""
	}
	hdr := styPrevHdr.Render(fmt.Sprintf("%s · %s · %s",
		c.Tool.String(), c.Cwd, humanizeAge(c.Modified, time.Now())))
	if _, err := os.Stat(c.Cwd); err != nil {
		hdr += styWarn.Render("  ⚠ path missing")
	}
	return hdr
}

// --- layout sizing (accounts for the rounded frame) ---

func (m Model) frameWidth() int {
	if m.width <= 0 {
		return 100
	}
	return m.width
}

// innerWidth is the content width inside the frame (border 2 + padding 2).
func (m Model) innerWidth() int {
	w := m.frameWidth() - 4
	if w < 24 {
		w = 24
	}
	return w
}

// leftWidth is the list pane width; the preview pane reserves 2 columns for its
// left border and padding.
func (m Model) leftWidth() int {
	w := (m.innerWidth() - 2) * 66 / 100
	if w < 12 {
		w = 12
	}
	return w
}

func (m Model) rightWidth() int {
	w := m.innerWidth() - 2 - m.leftWidth()
	if w < 10 {
		w = 10
	}
	return w
}

// bodyHeight is how many lines the list and preview panes occupy.
func (m Model) bodyHeight() int {
	overhead := 6 // frame(2) + header(1) + 2 blanks + footer(1)
	if m.searching {
		overhead++
	}
	h := m.height - overhead
	if h < 3 {
		h = 3
	}
	return h
}

// listHeight is how many conversation rows fit, reserving 2 lines for the
// scroll indicators.
func (m Model) listHeight() int {
	h := m.bodyHeight() - 2
	if h < 1 {
		h = 1
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

// padTrunc fits s to exactly w columns (rune-approximate), truncating with an
// ellipsis or right-padding with spaces.
func padTrunc(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) > w {
		if w == 1 {
			return "…"
		}
		return string(r[:w-1]) + "…"
	}
	return s + strings.Repeat(" ", w-len(r))
}

// clampLines keeps at most n lines of s.
func clampLines(s string, n int) string {
	if n < 1 {
		n = 1
	}
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
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
