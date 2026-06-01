package tui

import (
	"testing"
	"time"

	"carryon/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func sampleConvs() []model.Conversation {
	now := time.Now()
	return []model.Conversation{
		{Tool: model.Claude, SessionID: "id-claude", Cwd: "/Users/me/hootmail3", Title: "fix the bounce handler", Modified: now},
		{Tool: model.Codex, SessionID: "id-codex", Cwd: "/Users/me/dl", Title: "add retry logic", Modified: now.Add(-time.Hour)},
	}
}

func stubPreview(c model.Conversation) string { return "PREVIEW:" + c.SessionID }

func newTestModel() Model {
	return New(sampleConvs(), "/Users/me/hootmail3", stubPreview)
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func update(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	return next.(Model)
}

func TestSearchFilters(t *testing.T) {
	m := newTestModel()
	m = update(m, key("/"))     // enter search mode
	m = update(m, key("retry")) // type query
	if n := m.VisibleCount(); n != 1 {
		t.Fatalf("VisibleCount = %d, want 1", n)
	}
	if m.Current().SessionID != "id-codex" {
		t.Errorf("Current = %q, want id-codex", m.Current().SessionID)
	}
}

func TestNavigation(t *testing.T) {
	m := newTestModel()
	if m.Current().SessionID != "id-claude" {
		t.Fatalf("initial Current = %q", m.Current().SessionID)
	}
	m = update(m, key("j")) // down
	if m.Current().SessionID != "id-codex" {
		t.Errorf("after j, Current = %q, want id-codex", m.Current().SessionID)
	}
	m = update(m, key("j")) // clamp at bottom
	if m.Current().SessionID != "id-codex" {
		t.Errorf("cursor should clamp at bottom, got %q", m.Current().SessionID)
	}
	m = update(m, key("k")) // up
	if m.Current().SessionID != "id-claude" {
		t.Errorf("after k, Current = %q, want id-claude", m.Current().SessionID)
	}
}
