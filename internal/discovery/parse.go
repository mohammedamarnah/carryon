// Package discovery finds and parses Claude Code and Codex session files.
package discovery

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"carryon/internal/model"
)

const (
	maxScanBuffer = 8 * 1024 * 1024 // session lines can be large
	maxTitleRunes = 80
	noMessage     = "(no message)"
)

// --- Claude JSON shapes ---

type claudeLine struct {
	Type      string         `json:"type"`
	Cwd       string         `json:"cwd"`
	GitBranch string         `json:"gitBranch"`
	IsMeta    bool           `json:"isMeta"`
	Message   *claudeMessage `json:"message"`
}

type claudeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type claudeBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// claudeText returns human text from a message's content (string or block array),
// or "" if there is no usable text.
func claudeText(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []claudeBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return b.Text
			}
		}
	}
	return ""
}

func newScanner(f *os.File) *bufio.Scanner {
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), maxScanBuffer)
	return sc
}

func parseClaudeFile(path string) (model.Conversation, error) {
	f, err := os.Open(path)
	if err != nil {
		return model.Conversation{}, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return model.Conversation{}, err
	}

	conv := model.Conversation{
		Tool:      model.Claude,
		SessionID: strings.TrimSuffix(filepath.Base(path), ".jsonl"),
		Modified:  info.ModTime(),
		Path:      path,
	}

	sc := newScanner(f)
	for sc.Scan() {
		var ln claudeLine
		if err := json.Unmarshal(sc.Bytes(), &ln); err != nil {
			continue // skip corrupt lines
		}
		if conv.Cwd == "" && ln.Cwd != "" {
			conv.Cwd = ln.Cwd
		}
		if conv.Branch == "" && ln.GitBranch != "" {
			conv.Branch = ln.GitBranch
		}
		if conv.Title == "" && !ln.IsMeta && ln.Message != nil && ln.Message.Role == "user" {
			text := strings.TrimSpace(claudeText(ln.Message.Content))
			if text != "" && !strings.HasPrefix(text, "<") {
				conv.Title = firstLineTrunc(text, maxTitleRunes)
			}
		}
		if conv.Cwd != "" && conv.Title != "" {
			break // bounded read: we have what we need
		}
	}
	if conv.Title == "" {
		conv.Title = noMessage
	}
	return conv, nil
}

// --- Codex JSON shapes ---

type codexLine struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type codexMeta struct {
	ID  string `json:"id"`
	Cwd string `json:"cwd"`
}

type codexItem struct {
	Type    string       `json:"type"`
	Role    string       `json:"role"`
	Content []codexBlock `json:"content"`
}

type codexBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func parseCodexFile(path string) (model.Conversation, error) {
	f, err := os.Open(path)
	if err != nil {
		return model.Conversation{}, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return model.Conversation{}, err
	}

	conv := model.Conversation{
		Tool:     model.Codex,
		Modified: info.ModTime(),
		Path:     path,
	}

	sc := newScanner(f)
	for sc.Scan() {
		var ln codexLine
		if err := json.Unmarshal(sc.Bytes(), &ln); err != nil {
			continue
		}
		switch ln.Type {
		case "session_meta":
			var meta codexMeta
			if err := json.Unmarshal(ln.Payload, &meta); err == nil {
				conv.SessionID = meta.ID
				conv.Cwd = meta.Cwd
			}
		case "response_item":
			if conv.Title != "" {
				continue
			}
			var item codexItem
			if err := json.Unmarshal(ln.Payload, &item); err != nil {
				continue
			}
			if item.Type != "message" || item.Role != "user" {
				continue
			}
			for _, b := range item.Content {
				text := strings.TrimSpace(b.Text)
				if text != "" && !strings.HasPrefix(text, "<") {
					conv.Title = firstLineTrunc(text, maxTitleRunes)
					break
				}
			}
		}
		if conv.Cwd != "" && conv.Title != "" {
			break
		}
	}
	if conv.Title == "" {
		conv.Title = noMessage
	}
	return conv, nil
}

// firstLineTrunc collapses to the first line and truncates to n runes.
func firstLineTrunc(s string, n int) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}
