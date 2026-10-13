package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mohammedamarnah/carryon/internal/model"
)

func TestDeleteClaudeRemovesTranscriptAndSidecarDir(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "-Users-me-proj", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl")
	writeFile(t, path, "{}\n")
	sidecar := filepath.Join(root, "-Users-me-proj", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "subagents", "x.jsonl")
	writeFile(t, sidecar, "{}\n")
	other := filepath.Join(root, "-Users-me-proj", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb.jsonl")
	writeFile(t, other, "{}\n")

	if err := Delete(model.Conversation{Tool: model.Claude, Path: path}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("transcript still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(sidecar)); !os.IsNotExist(err) {
		t.Errorf("sidecar dir still exists: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("unrelated conversation was touched: %v", err)
	}
}

func TestDeleteCodexRemovesOnlyRollout(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "2026", "05", "20", "rollout-2026-05-20T14-42-09-bbbb.jsonl")
	writeFile(t, path, "{}\n")
	other := filepath.Join(root, "2026", "05", "20", "rollout-2026-05-20T15-00-00-cccc.jsonl")
	writeFile(t, other, "{}\n")

	if err := Delete(model.Conversation{Tool: model.Codex, Path: path}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("rollout still exists: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("unrelated rollout was touched: %v", err)
	}
}

func TestDeleteMissingFileErrors(t *testing.T) {
	if err := Delete(model.Conversation{Tool: model.Claude, Path: filepath.Join(t.TempDir(), "nope.jsonl")}); err == nil {
		t.Error("expected error deleting a missing file")
	}
}
