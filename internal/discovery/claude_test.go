package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mohammedamarnah/carryon/internal/model"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseClaudeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a82756b2-a42e-4d8b-ab86-a2cc889e890d.jsonl")
	content := `{"type":"last-prompt","leafUuid":"x","sessionId":"a82756b2"}
{"type":"user","isMeta":true,"message":{"role":"user","content":"<local-command-caveat>ignore me</local-command-caveat>"}}
{"type":"user","cwd":"/Users/me/proj","gitBranch":"master","message":{"role":"user","content":"fix the bounce handler please"}}
`
	writeFile(t, path, content)

	conv, err := parseClaudeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if conv.Tool != model.Claude {
		t.Errorf("Tool = %v, want Claude", conv.Tool)
	}
	if conv.SessionID != "a82756b2-a42e-4d8b-ab86-a2cc889e890d" {
		t.Errorf("SessionID = %q", conv.SessionID)
	}
	if conv.Cwd != "/Users/me/proj" {
		t.Errorf("Cwd = %q", conv.Cwd)
	}
	if conv.Branch != "master" {
		t.Errorf("Branch = %q", conv.Branch)
	}
	if conv.Title != "fix the bounce handler please" {
		t.Errorf("Title = %q", conv.Title)
	}
}

func TestParseClaudeFileContentBlocks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "11111111-1111-1111-1111-111111111111.jsonl")
	content := `{"type":"user","cwd":"/p","message":{"role":"user","content":[{"type":"text","text":"why is the webhook failing"}]}}
`
	writeFile(t, path, content)
	conv, err := parseClaudeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if conv.Title != "why is the webhook failing" {
		t.Errorf("Title = %q", conv.Title)
	}
}

func TestParseClaudeFileNoMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "22222222-2222-2222-2222-222222222222.jsonl")
	content := `{"type":"summary","cwd":"/p"}
not valid json at all
`
	writeFile(t, path, content)
	conv, err := parseClaudeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if conv.Title != "(no message)" {
		t.Errorf("Title = %q, want fallback", conv.Title)
	}
}
