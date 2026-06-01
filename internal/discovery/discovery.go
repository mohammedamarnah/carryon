package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/mohammedamarnah/carryon/internal/model"
)

// Discover reads the user's default Claude and Codex stores under $HOME.
func Discover() ([]model.Conversation, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return DiscoverIn(
		filepath.Join(home, ".claude", "projects"),
		filepath.Join(home, ".codex", "sessions"),
	)
}

// DiscoverIn reads the two given store roots and returns all conversations,
// sorted most-recently-modified first. Unparseable files are skipped.
func DiscoverIn(claudeRoot, codexRoot string) ([]model.Conversation, error) {
	var convs []model.Conversation

	claudeFiles, _ := filepath.Glob(filepath.Join(claudeRoot, "*", "*.jsonl"))
	for _, p := range claudeFiles {
		conv, err := parseClaudeFile(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "carryon: skipping %s: %v\n", p, err)
			continue
		}
		convs = append(convs, conv)
	}

	codexFiles, _ := filepath.Glob(filepath.Join(codexRoot, "*", "*", "*", "rollout-*.jsonl"))
	for _, p := range codexFiles {
		conv, err := parseCodexFile(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "carryon: skipping %s: %v\n", p, err)
			continue
		}
		convs = append(convs, conv)
	}

	sort.SliceStable(convs, func(i, j int) bool {
		return convs[i].Modified.After(convs[j].Modified)
	})
	return convs, nil
}
