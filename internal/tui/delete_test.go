package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mohammedamarnah/carryon/internal/model"
)

func newDeleteModel(fn func(model.Conversation) error) Model {
	return New(sampleConvs(), "/Users/me/hootmail3", stubPreview).WithDelete(fn)
}

func TestDeleteConfirmYesRemovesHighlighted(t *testing.T) {
	var deleted []string
	m := newDeleteModel(func(c model.Conversation) error {
		deleted = append(deleted, c.SessionID)
		return nil
	})
	m = update(m, key("d"))
	if len(deleted) != 0 {
		t.Fatal("deleted before confirmation")
	}
	if !strings.Contains(m.View(), "y/N") {
		t.Error("confirmation prompt not shown")
	}
	m = update(m, key("y"))
	if len(deleted) != 1 || deleted[0] != "id-claude" {
		t.Fatalf("deleted = %v, want [id-claude]", deleted)
	}
	if m.VisibleCount() != 1 || m.Current().SessionID != "id-codex" {
		t.Fatalf("after delete: count=%d", m.VisibleCount())
	}
	if !strings.Contains(m.View(), "deleted claude conversation") {
		t.Error("success status not shown")
	}
}

func TestDeleteAnyOtherKeyCancels(t *testing.T) {
	called := false
	m := newDeleteModel(func(model.Conversation) error { called = true; return nil })
	m = update(m, key("d"))
	m = update(m, key("n"))
	m = update(m, key("y")) // no longer confirming; must not delete
	if called {
		t.Fatal("delete ran after cancel")
	}
	if m.VisibleCount() != 2 {
		t.Errorf("count = %d, want 2", m.VisibleCount())
	}
}

func TestDeleteEnterDoesNotConfirm(t *testing.T) {
	called := false
	m := newDeleteModel(func(model.Conversation) error { called = true; return nil })
	m = update(m, key("d"))
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if called || m.Selected() != nil {
		t.Fatal("enter during confirmation should only cancel")
	}
}

func TestDeleteFailureKeepsConversation(t *testing.T) {
	m := newDeleteModel(func(model.Conversation) error { return errors.New("permission denied") })
	m = update(m, key("d"))
	m = update(m, key("y"))
	if m.VisibleCount() != 2 {
		t.Errorf("count = %d, want 2", m.VisibleCount())
	}
	if !strings.Contains(m.View(), "delete failed: permission denied") {
		t.Error("error not shown")
	}
}

func TestDeleteDisabledWithoutFunc(t *testing.T) {
	m := newTestModel()
	m = update(m, key("d"))
	m = update(m, key("y"))
	if m.VisibleCount() != 2 {
		t.Errorf("count = %d, want 2", m.VisibleCount())
	}
}

func TestDeleteRespectsFilter(t *testing.T) {
	var deleted string
	m := newDeleteModel(func(c model.Conversation) error { deleted = c.SessionID; return nil })
	m = update(m, key("t")) // claude
	m = update(m, key("t")) // codex
	m = update(m, key("d"))
	m = update(m, key("y"))
	if deleted != "id-codex" {
		t.Fatalf("deleted %q, want id-codex", deleted)
	}
	m = update(m, key("t")) // all
	if m.VisibleCount() != 1 || m.Current().SessionID != "id-claude" {
		t.Errorf("remaining: count=%d", m.VisibleCount())
	}
}
