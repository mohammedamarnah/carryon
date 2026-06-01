package model

import "testing"

func TestToolString(t *testing.T) {
	if Claude.String() != "claude" {
		t.Errorf("Claude.String() = %q, want %q", Claude.String(), "claude")
	}
	if Codex.String() != "codex" {
		t.Errorf("Codex.String() = %q, want %q", Codex.String(), "codex")
	}
}

func TestResumeCommand(t *testing.T) {
	if got := Claude.ResumeCommand(); got != `exec claude --resume "$1"` {
		t.Errorf("Claude.ResumeCommand() = %q", got)
	}
	if got := Codex.ResumeCommand(); got != `exec codex resume "$1"` {
		t.Errorf("Codex.ResumeCommand() = %q", got)
	}
}
