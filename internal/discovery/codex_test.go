package discovery

import (
	"path/filepath"
	"testing"

	"carryon/internal/model"
)

func TestParseCodexFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout-2026-05-20T14-42-09-019e46e8.jsonl")
	content := `{"timestamp":"2026-05-20T19:43:08.423Z","type":"session_meta","payload":{"id":"019e46e8-e689-71b0-8ff1-c049f8d451c2","cwd":"/Users/me/Downloads","timestamp":"2026-05-20T19:42:09.033Z"}}
{"type":"response_item","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"<permissions instructions>"}]}}
{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"<environment_context>\n  <cwd>/Users/me/Downloads</cwd>\n</environment_context>"}]}}
{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"add retry logic to the webhook"}]}}
`
	writeFile(t, path, content)

	conv, err := parseCodexFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if conv.Tool != model.Codex {
		t.Errorf("Tool = %v, want Codex", conv.Tool)
	}
	if conv.SessionID != "019e46e8-e689-71b0-8ff1-c049f8d451c2" {
		t.Errorf("SessionID = %q", conv.SessionID)
	}
	if conv.Cwd != "/Users/me/Downloads" {
		t.Errorf("Cwd = %q", conv.Cwd)
	}
	if conv.Branch != "" {
		t.Errorf("Branch = %q, want empty", conv.Branch)
	}
	if conv.Title != "add retry logic to the webhook" {
		t.Errorf("Title = %q", conv.Title)
	}
}
