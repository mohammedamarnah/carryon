package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mohammedamarnah/carryon/internal/discovery"
	"github.com/mohammedamarnah/carryon/internal/launch"
	"github.com/mohammedamarnah/carryon/internal/model"
	"github.com/mohammedamarnah/carryon/internal/tui"
)

const previewTurns = 12

func main() {
	convs, err := discovery.Discover()
	if err != nil {
		fmt.Fprintln(os.Stderr, "carryon:", err)
		os.Exit(1)
	}
	if len(convs) == 0 {
		fmt.Println("No Claude Code or Codex conversations found.")
		return
	}

	currentDir, _ := os.Getwd()
	loadPreview := func(c model.Conversation) string {
		return discovery.LoadPreview(c, previewTurns)
	}

	p := tea.NewProgram(
		tui.New(convs, currentDir, loadPreview),
		tea.WithAltScreen(),
	)
	final, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "carryon:", err)
		os.Exit(1)
	}

	selected := final.(tui.Model).Selected()
	if selected == nil {
		return // user quit without choosing
	}

	spec, warnings, err := launch.Build(*selected, launch.DefaultEnv())
	if err != nil {
		fmt.Fprintln(os.Stderr, "carryon:", err)
		os.Exit(1)
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "carryon:", w)
	}

	// Replaces this process; does not return on success.
	if err := launch.Exec(spec); err != nil {
		fmt.Fprintln(os.Stderr, "carryon: launch failed:", err)
		os.Exit(1)
	}
}
