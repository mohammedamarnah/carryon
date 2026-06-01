package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"carryon/internal/model"
)

func TestDiscoverIn(t *testing.T) {
	claudeRoot := t.TempDir()
	codexRoot := t.TempDir()

	claudePath := filepath.Join(claudeRoot, "-Users-me-proj", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl")
	writeFile(t, claudePath, `{"type":"user","cwd":"/Users/me/proj","gitBranch":"main","message":{"role":"user","content":"older claude convo"}}`+"\n")

	codexPath := filepath.Join(codexRoot, "2026", "05", "20", "rollout-2026-05-20T14-42-09-bbbb.jsonl")
	writeFile(t, codexPath, `{"type":"session_meta","payload":{"id":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","cwd":"/Users/me/dl"}}`+"\n"+
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"newer codex convo"}]}}`+"\n")

	// Make the codex file newer so it sorts first.
	older := time.Now().Add(-2 * time.Hour)
	newer := time.Now()
	if err := os.Chtimes(claudePath, older, older); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(codexPath, newer, newer); err != nil {
		t.Fatal(err)
	}

	convs, err := DiscoverIn(claudeRoot, codexRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(convs) != 2 {
		t.Fatalf("got %d conversations, want 2", len(convs))
	}
	if convs[0].Tool != model.Codex {
		t.Errorf("first conv tool = %v, want Codex (most recent first)", convs[0].Tool)
	}
	if convs[1].Tool != model.Claude {
		t.Errorf("second conv tool = %v, want Claude", convs[1].Tool)
	}
}
