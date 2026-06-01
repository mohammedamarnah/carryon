// Package model defines the common conversation type shared across carryon.
package model

import "time"

// Tool identifies which CLI owns a conversation.
type Tool int

const (
	Claude Tool = iota
	Codex
)

func (t Tool) String() string {
	switch t {
	case Claude:
		return "claude"
	case Codex:
		return "codex"
	default:
		return "unknown"
	}
}

// ResumeCommand returns the constant shell command that resumes a session.
// The session ID is supplied separately as positional arg $1 — never
// interpolated into this string — so a malicious ID cannot inject shell code.
func (t Tool) ResumeCommand() string {
	switch t {
	case Claude:
		return `exec claude --resume "$1"`
	case Codex:
		return `exec codex resume "$1"`
	default:
		return ""
	}
}

// Conversation is a single resumable Claude Code or Codex session.
type Conversation struct {
	Tool      Tool
	SessionID string    // UUID passed to resume
	Cwd       string    // original project directory
	Branch    string    // git branch (Claude only; "" for Codex)
	Title     string    // first real user message, trimmed
	Modified  time.Time // file mtime; used for sort + age
	Path      string    // path to the .jsonl
}
