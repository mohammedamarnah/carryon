// Package tui implements carryon's interactive conversation picker.
package tui

import (
	"fmt"
	"strings"
	"time"
)

// humanizeAge renders the gap between t and now as a compact age string.
func humanizeAge(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dw", int(d.Hours()/24/7))
	}
}

// shortenPath replaces $HOME with ~ and keeps the last one or two segments.
func shortenPath(p, home string) string {
	if home != "" && p == home {
		return "~"
	}
	segs := strings.Split(strings.Trim(p, "/"), "/")
	if len(segs) == 0 {
		return p
	}
	if len(segs) <= 2 {
		return strings.Join(segs, "/")
	}
	return strings.Join(segs[len(segs)-2:], "/")
}

// fuzzyMatch reports whether pattern's runes appear in target in order
// (case-insensitive). An empty pattern always matches.
func fuzzyMatch(pattern, target string) bool {
	if pattern == "" {
		return true
	}
	p := []rune(strings.ToLower(pattern))
	t := []rune(strings.ToLower(target))
	i := 0
	for _, tc := range t {
		if tc == p[i] {
			i++
			if i == len(p) {
				return true
			}
		}
	}
	return false
}
