package tui

import (
	"strings"
	"testing"
)

func TestViewShowsConversations(t *testing.T) {
	m := newTestModel()
	m.width, m.height = 120, 30
	out := m.View()
	if !strings.Contains(out, "fix the bounce handler") {
		t.Errorf("view missing claude title:\n%s", out)
	}
	if !strings.Contains(out, "add retry logic") {
		t.Errorf("view missing codex title:\n%s", out)
	}
	if !strings.Contains(out, "PREVIEW:id-claude") {
		t.Errorf("view missing preview of highlighted row:\n%s", out)
	}
}

func TestViewNoMatches(t *testing.T) {
	m := newTestModel()
	m.width, m.height = 120, 30
	m = update(m, key("/"))
	m = update(m, key("zzzzz"))
	out := m.View()
	if !strings.Contains(out, "No matches") {
		t.Errorf("view should show no-matches state:\n%s", out)
	}
}
