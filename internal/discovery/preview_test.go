package discovery

import (
	"path/filepath"
	"strings"
	"testing"

	"carryon/internal/model"
)

func TestLoadPreviewClaude(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cccccccc-cccc-cccc-cccc-cccccccccccc.jsonl")
	writeFile(t, path, `{"type":"user","cwd":"/p","message":{"role":"user","content":"first question"}}`+"\n"+
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"here is an answer"}]}}`+"\n")

	conv := model.Conversation{Tool: model.Claude, Path: path}
	out := LoadPreview(conv, 10)
	if !strings.Contains(out, "first question") {
		t.Errorf("preview missing user turn:\n%s", out)
	}
	if !strings.Contains(out, "here is an answer") {
		t.Errorf("preview missing assistant turn:\n%s", out)
	}
}

func TestLoadPreviewLastN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dddddddd-dddd-dddd-dddd-dddddddddddd.jsonl")
	var b strings.Builder
	for i := 0; i < 5; i++ {
		b.WriteString(`{"type":"user","message":{"role":"user","content":"msg`)
		b.WriteString(strings.Repeat("x", 0))
		b.WriteString(string(rune('0' + i)))
		b.WriteString(`"}}` + "\n")
	}
	writeFile(t, path, b.String())

	conv := model.Conversation{Tool: model.Claude, Path: path}
	out := LoadPreview(conv, 2)
	if strings.Contains(out, "msg0") {
		t.Errorf("preview should only keep last 2 turns, but contains msg0:\n%s", out)
	}
	if !strings.Contains(out, "msg4") {
		t.Errorf("preview should contain newest turn msg4:\n%s", out)
	}
}
