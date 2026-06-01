package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mohammedamarnah/carryon/internal/model"
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

// --- theme: purple/pink app chrome, per-tool accents (Claude=orange, Codex=teal) ---

var (
	// app chrome
	colApp    = lipgloss.Color("#C084FC") // purple — title, frame, indicators
	colAppDim = lipgloss.Color("#7C3AED") // deep purple — separators, branch
	colPink   = lipgloss.Color("#F472B6") // pink — search, user turns
	colMuted  = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	colFg     = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#E5E7EB"}
	colWarn   = lipgloss.Color("#F87171") // red — path missing

	// per-tool accents
	colClaude     = lipgloss.Color("#FB923C") // orange — Claude Code
	colClaudeText = lipgloss.Color("#431407") // dark text on the orange highlight
	colCodex      = lipgloss.Color("#2DD4BF") // teal — Codex
	colCodexText  = lipgloss.Color("#042F2E") // dark text on the teal highlight

	styTitle    = lipgloss.NewStyle().Foreground(colApp).Bold(true)
	styCount    = lipgloss.NewStyle().Foreground(colMuted)
	stySearch   = lipgloss.NewStyle().Foreground(colPink).Bold(true)
	styProj     = lipgloss.NewStyle().Foreground(colFg)
	styAge      = lipgloss.NewStyle().Foreground(colMuted)
	styBranch   = lipgloss.NewStyle().Foreground(colAppDim)
	styRowTitle = lipgloss.NewStyle().Foreground(colFg)
	styMore     = lipgloss.NewStyle().Foreground(colApp).Italic(true)
	styPrev     = lipgloss.NewStyle().Foreground(colFg)
	styWarn     = lipgloss.NewStyle().Foreground(colWarn).Bold(true)
	styRoleUser = lipgloss.NewStyle().Foreground(colPink).Bold(true)
	styRoleAsst = lipgloss.NewStyle().Foreground(colApp).Bold(true)
	styFooter   = lipgloss.NewStyle().Foreground(colMuted)

	stySep    = lipgloss.NewStyle().Foreground(colAppDim)
	styBorder = lipgloss.NewStyle().Foreground(colApp)

	// per-tool tool-name accent (non-selected rows) and preview header
	styToolClaude = lipgloss.NewStyle().Foreground(colClaude).Bold(true)
	styToolCodex  = lipgloss.NewStyle().Foreground(colCodex).Bold(true)

	// per-tool selected-row highlight
	stySelClaude = lipgloss.NewStyle().Background(colClaude).Foreground(colClaudeText).Bold(true)
	stySelCodex  = lipgloss.NewStyle().Background(colCodex).Foreground(colCodexText).Bold(true)
)

// toolStyle returns the per-tool accent style (Claude=orange, Codex=teal).
func toolStyle(t model.Tool) lipgloss.Style {
	if t == model.Codex {
		return styToolCodex
	}
	return styToolClaude
}

// selStyle returns the per-tool selected-row highlight style.
func selStyle(t model.Tool) lipgloss.Style {
	if t == model.Codex {
		return stySelCodex
	}
	return stySelClaude
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	home, _ := os.UserHomeDir()

	inner := m.innerWidth()

	cwd := ""
	if m.cwdOnly {
		cwd = " · cwd-only"
	}
	count := trunc(fmt.Sprintf("%d conversations · tool: %s%s", len(m.filtered), m.toolLabel(), cwd), inner-9)
	header := styTitle.Render("carryon") + "  " + styCount.Render(count)
	if m.searching {
		header += "\n" + stySearch.Render(trunc("search: "+m.search+"▏", inner))
	}

	// Zip the two panes line-by-line at exact widths and height, with a manual
	// separator. This avoids lipgloss's border/padding width ambiguities,
	// guaranteeing every body line is exactly innerWidth and there are exactly
	// bodyH of them — so nothing overflows the frame or the terminal.
	bodyH := m.bodyHeight()
	leftLines := fitBlock(strings.Split(m.listView(home), "\n"), m.leftWidth(), bodyH)
	rightLines := m.previewLines(home, m.rightWidth(), bodyH)
	sep := stySep.Render("│") + " "

	footer := styFooter.Render(trunc("↑↓ move · ↵ resume · / search · t tool · . cwd-only · q quit", inner))

	var content []string
	content = append(content, strings.Split(header, "\n")...)
	content = append(content, "")
	for i := 0; i < bodyH; i++ {
		content = append(content, leftLines[i]+sep+rightLines[i])
	}
	content = append(content, "", footer)
	return drawFrame(content, inner)
}

// drawFrame draws a rounded border around the content lines, padding each
// line to innerWidth. The result is exactly innerWidth+4 columns wide and
// len(lines)+2 rows tall — no lipgloss border box-model guesswork.
func drawFrame(lines []string, innerWidth int) string {
	bar := strings.Repeat("─", innerWidth+2)
	side := styBorder.Render("│")
	var b strings.Builder
	b.WriteString(styBorder.Render("╭"+bar+"╮") + "\n")
	for _, ln := range lines {
		pad := innerWidth - lipgloss.Width(ln)
		if pad < 0 {
			pad = 0
		}
		b.WriteString(side + " " + ln + strings.Repeat(" ", pad) + " " + side + "\n")
	}
	b.WriteString(styBorder.Render("╰" + bar + "╯"))
	return b.String()
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

// renderRow formats one conversation line, fit to exactly `width` columns so it
// never wraps the pane. The title takes the remaining space; the project and
// branch columns drop out on narrow panes. The selected row gets a full-width
// per-tool highlight (Claude=orange, Codex=teal); others get a per-tool
// tool-name accent.
func (m Model) renderRow(c model.Conversation, home string, selected bool, width int) string {
	marker := "  "
	if selected {
		marker = "› "
	}

	const wTool, wAge = 6, 4
	wProj, wBranch := 16, 8
	switch {
	case width < 44:
		wProj, wBranch = 0, 0
	case width < 60:
		wProj, wBranch = 12, 0
	}

	used := 2 + wTool + 2 + wAge + 2 // marker + tool + gap + age + gap
	if wProj > 0 {
		used += wProj + 2
	}
	if wBranch > 0 {
		used += wBranch + 2
	}
	titleW := width - used
	if titleW < 1 {
		titleW = 1
	}

	tool := padTrunc(c.Tool.String(), wTool)
	age := padTrunc(humanizeAge(c.Modified, time.Now()), wAge)
	title := padTrunc(c.Title, titleW)

	if selected {
		var p strings.Builder
		p.WriteString(marker + tool + "  ")
		if wProj > 0 {
			p.WriteString(padTrunc(shortenPath(c.Cwd, home), wProj) + "  ")
		}
		p.WriteString(age + "  ")
		if wBranch > 0 {
			p.WriteString(padTrunc(c.Branch, wBranch) + "  ")
		}
		p.WriteString(title)
		return selStyle(c.Tool).Render(padTrunc(p.String(), width))
	}

	var b strings.Builder
	b.WriteString(marker + toolStyle(c.Tool).Render(tool) + "  ")
	if wProj > 0 {
		b.WriteString(styProj.Render(padTrunc(shortenPath(c.Cwd, home), wProj)) + "  ")
	}
	b.WriteString(styAge.Render(age) + "  ")
	if wBranch > 0 {
		b.WriteString(styBranch.Render(padTrunc(c.Branch, wBranch)) + "  ")
	}
	b.WriteString(styRowTitle.Render(title))
	row := b.String()
	if lipgloss.Width(row) > width { // last-resort guard against wrapping
		return styRowTitle.Render(padTrunc(marker+c.Title, width))
	}
	return row
}

// previewLines renders the highlighted conversation's transcript as exactly
// `height` lines, each no wider than `w` columns, so it can never overflow the
// frame or bleed into the list pane. Each piece is wrapped from PLAIN text in a
// single styled Render — wrapping already-styled (ANSI) text mis-measures width.
func (m Model) previewLines(home string, w, height int) []string {
	wrap := func(s string, st lipgloss.Style) []string {
		return strings.Split(st.Width(w).Render(s), "\n")
	}

	var lines []string
	if c := m.Current(); c != nil {
		lines = append(lines, wrap(fmt.Sprintf("%s · %s · %s",
			c.Tool.String(), c.Cwd, humanizeAge(c.Modified, time.Now())), toolStyle(c.Tool))...)
		if _, err := os.Stat(c.Cwd); err != nil {
			lines = append(lines, wrap("⚠ path missing", styWarn)...)
		}
		lines = append(lines, "")
	}
	for _, line := range strings.Split(m.previewText, "\n") {
		lines = append(lines, previewTurnLines(line, w)...)
	}

	return fitBlock(lines, w, height)
}

// previewTurnLines wraps one transcript line to w columns and colors a leading
// "user:" / "assistant:" role label. The text is wrapped while plain, then each
// final line is styled, so widths stay correct.
func previewTurnLines(s string, w int) []string {
	wrapped := strings.Split(lipgloss.NewStyle().Width(w).Render(s), "\n")

	var roleStyle lipgloss.Style
	var label string
	switch {
	case strings.HasPrefix(s, "user: "):
		roleStyle, label = styRoleUser, "user:"
	case strings.HasPrefix(s, "assistant: "):
		roleStyle, label = styRoleAsst, "assistant:"
	}

	for i, ln := range wrapped {
		if i == 0 && label != "" && len(ln) >= len(label) {
			wrapped[i] = roleStyle.Render(label) + styPrev.Render(ln[len(label):])
		} else {
			wrapped[i] = styPrev.Render(ln)
		}
	}
	return wrapped
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

// trunc shortens s to at most w columns (rune-approximate) with an ellipsis,
// without padding. It assumes s has no ANSI escapes.
func trunc(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
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

// fitBlock returns exactly height lines, each padded with spaces to width.
// Padding uses lipgloss.Width so ANSI-styled lines measure correctly; lines are
// assumed to already be no wider than width.
func fitBlock(lines []string, width, height int) []string {
	out := make([]string, height)
	for i := 0; i < height; i++ {
		s := ""
		if i < len(lines) {
			s = lines[i]
		}
		if pad := width - lipgloss.Width(s); pad > 0 {
			s += strings.Repeat(" ", pad)
		}
		out[i] = s
	}
	return out
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
	case tea.KeyRunes, tea.KeySpace:
		// Space arrives as KeySpace (Runes == [' ']), not KeyRunes.
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
