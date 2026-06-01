package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Space arrives from Bubble Tea as KeySpace, not KeyRunes; the search box must
// still accept it.
func TestSearchAcceptsSpace(t *testing.T) {
	m := newTestModel()
	m = update(m, key("/"))
	m = update(m, key("fix"))
	m = update(m, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	m = update(m, key("bounce"))

	if m.search != "fix bounce" {
		t.Fatalf("search = %q, want %q", m.search, "fix bounce")
	}
	// "fix bounce" is a subsequence of the claude convo's "fix the bounce handler"
	if m.VisibleCount() != 1 || m.Current().SessionID != "id-claude" {
		t.Errorf("expected the claude convo to match, got count=%d", m.VisibleCount())
	}
}
