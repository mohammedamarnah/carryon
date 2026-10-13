package discovery

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mohammedamarnah/carryon/internal/model"
)

// Delete permanently removes a conversation's transcript. For Claude, the
// sibling "<session-id>/" directory (subagent transcripts, tool results) is
// removed too, so nothing orphaned is left behind.
func Delete(c model.Conversation) error {
	if err := os.Remove(c.Path); err != nil {
		return err
	}
	if c.Tool == model.Claude {
		dir := strings.TrimSuffix(c.Path, filepath.Ext(c.Path))
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return os.RemoveAll(dir)
		}
	}
	return nil
}
