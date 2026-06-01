package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"carryon/internal/model"
)

const previewTextRunes = 200

type turn struct {
	role string
	text string
}

// LoadPreview returns the last maxTurns user/assistant turns of a conversation,
// formatted one block per turn. It reads the whole file lazily (called only when
// a row is highlighted). On error it returns a short message instead of failing.
func LoadPreview(conv model.Conversation, maxTurns int) string {
	turns, err := extractTurns(conv)
	if err != nil {
		return fmt.Sprintf("(could not read transcript: %v)", err)
	}
	if len(turns) == 0 {
		return "(no messages)"
	}
	if len(turns) > maxTurns {
		turns = turns[len(turns)-maxTurns:]
	}
	var b strings.Builder
	for _, t := range turns {
		fmt.Fprintf(&b, "%s: %s\n\n", t.role, firstLineTrunc(t.text, previewTextRunes))
	}
	return strings.TrimRight(b.String(), "\n")
}

func extractTurns(conv model.Conversation) ([]turn, error) {
	f, err := os.Open(conv.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var turns []turn
	sc := newScanner(f)
	for sc.Scan() {
		if conv.Tool == model.Claude {
			turns = appendClaudeTurn(turns, sc.Bytes())
		} else {
			turns = appendCodexTurn(turns, sc.Bytes())
		}
	}
	return turns, nil
}

func appendClaudeTurn(turns []turn, line []byte) []turn {
	var ln claudeLine
	if err := json.Unmarshal(line, &ln); err != nil || ln.IsMeta || ln.Message == nil {
		return turns
	}
	role := ln.Message.Role
	if role != "user" && role != "assistant" {
		return turns
	}
	text := strings.TrimSpace(claudeText(ln.Message.Content))
	if text == "" || strings.HasPrefix(text, "<") {
		return turns
	}
	return append(turns, turn{role: role, text: text})
}

func appendCodexTurn(turns []turn, line []byte) []turn {
	var ln codexLine
	if err := json.Unmarshal(line, &ln); err != nil || ln.Type != "response_item" {
		return turns
	}
	var item codexItem
	if err := json.Unmarshal(ln.Payload, &item); err != nil || item.Type != "message" {
		return turns
	}
	if item.Role != "user" && item.Role != "assistant" {
		return turns
	}
	for _, b := range item.Content {
		text := strings.TrimSpace(b.Text)
		if text != "" && !strings.HasPrefix(text, "<") {
			return append(turns, turn{role: item.Role, text: text})
		}
	}
	return turns
}
