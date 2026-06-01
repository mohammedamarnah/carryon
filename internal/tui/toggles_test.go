package tui

import (
	"testing"

	"carryon/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestToolFilterCycle(t *testing.T) {
	m := newTestModel()
	m = update(m, key("t")) // all -> claude
	if m.VisibleCount() != 1 || m.Current().Tool != model.Claude {
		t.Fatalf("after first t: count=%d", m.VisibleCount())
	}
	m = update(m, key("t")) // claude -> codex
	if m.VisibleCount() != 1 || m.Current().Tool != model.Codex {
		t.Fatalf("after second t: count=%d", m.VisibleCount())
	}
	m = update(m, key("t")) // codex -> all
	if m.VisibleCount() != 2 {
		t.Fatalf("after third t: count=%d, want 2", m.VisibleCount())
	}
}

func TestCwdOnlyToggle(t *testing.T) {
	m := newTestModel() // currentDir = /Users/me/hootmail3
	m = update(m, key("."))
	if m.VisibleCount() != 1 {
		t.Fatalf("cwd-only count = %d, want 1", m.VisibleCount())
	}
	if m.Current().Cwd != "/Users/me/hootmail3" {
		t.Errorf("Current cwd = %q", m.Current().Cwd)
	}
	m = update(m, key(".")) // toggle off
	if m.VisibleCount() != 2 {
		t.Errorf("after toggle off, count = %d, want 2", m.VisibleCount())
	}
}

func TestEnterSelects(t *testing.T) {
	m := newTestModel()
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Selected() == nil {
		t.Fatal("Selected() is nil after Enter")
	}
	if m.Selected().SessionID != "id-claude" {
		t.Errorf("Selected = %q, want id-claude", m.Selected().SessionID)
	}
}

func TestQuitNoSelection(t *testing.T) {
	m := newTestModel()
	m = update(m, key("q"))
	if m.Selected() != nil {
		t.Error("Selected() should be nil after quit")
	}
}
