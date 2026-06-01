package tui

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mohammedamarnah/carryon/internal/model"
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

func TestViewScrollsToCursor(t *testing.T) {
	convs := make([]model.Conversation, 40)
	for i := range convs {
		convs[i] = model.Conversation{
			Tool:      model.Claude,
			SessionID: "id-" + string(rune('a'+i%26)),
			Cwd:       "/p",
			Title:     "title-" + strconv.Itoa(i),
			Modified:  time.Now(),
		}
	}
	m := New(convs, "/p", stubPreview)
	m.width, m.height = 120, 20 // small screen -> must window

	// Move cursor to the last row.
	for i := 0; i < len(convs)-1; i++ {
		m = update(m, key("j"))
	}
	out := m.View()
	if !strings.Contains(out, "title-39") {
		t.Errorf("view should show the row under the cursor (title-39):\n%s", out)
	}
	if strings.Contains(out, "title-0\n") {
		t.Errorf("view should have scrolled title-0 out of the window:\n%s", out)
	}
	if !strings.Contains(out, "↑") {
		t.Errorf("view should show an up-scroll indicator when scrolled down:\n%s", out)
	}
}
