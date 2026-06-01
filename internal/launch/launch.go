// Package launch builds and performs the handoff to the native CLI.
package launch

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"carryon/internal/model"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Env holds injectable dependencies so Build stays pure and testable.
type Env struct {
	Shell string
	Stat  func(string) (os.FileInfo, error)
	Getwd func() (string, error)
}

// DefaultEnv reads from the real process environment.
func DefaultEnv() Env {
	return Env{Shell: os.Getenv("SHELL"), Stat: os.Stat, Getwd: os.Getwd}
}

// Spec is a fully-resolved launch command.
type Spec struct {
	Shell string
	Argv  []string
	Dir   string
}

// Build validates inputs and constructs the launch Spec. The session ID is placed
// as the final positional arg ($1), never interpolated into the shell command.
func Build(conv model.Conversation, env Env) (Spec, []string, error) {
	var warnings []string

	if !uuidRe.MatchString(conv.SessionID) {
		return Spec{}, nil, fmt.Errorf("invalid session id %q", conv.SessionID)
	}

	shell := resolveShell(env, &warnings)
	dir := resolveDir(conv.Cwd, env, &warnings)

	argv := []string{shell, "-ic", conv.Tool.ResumeCommand(), "carryon", conv.SessionID}
	return Spec{Shell: shell, Argv: argv, Dir: dir}, warnings, nil
}

func resolveShell(env Env, warnings *[]string) string {
	const fallback = "/bin/sh"
	s := env.Shell
	if s == "" || !filepath.IsAbs(s) {
		*warnings = append(*warnings, fmt.Sprintf("$SHELL unusable (%q); using %s", s, fallback))
		return fallback
	}
	info, err := env.Stat(s)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		*warnings = append(*warnings, fmt.Sprintf("$SHELL not executable (%q); using %s", s, fallback))
		return fallback
	}
	return s
}

func resolveDir(cwd string, env Env, warnings *[]string) string {
	if cwd != "" {
		if info, err := env.Stat(cwd); err == nil && info.IsDir() {
			return cwd
		}
	}
	wd, err := env.Getwd()
	if err != nil {
		wd = "."
	}
	*warnings = append(*warnings, fmt.Sprintf("project dir %q missing; launching from %s", cwd, wd))
	return wd
}
