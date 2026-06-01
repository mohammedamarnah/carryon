package tui

import (
	"testing"
	"time"
)

func TestHumanizeAge(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		ago  time.Duration
		want string
	}{
		{30 * time.Second, "just now"},
		{5 * time.Minute, "5m"},
		{3 * time.Hour, "3h"},
		{2 * 24 * time.Hour, "2d"},
		{10 * 24 * time.Hour, "1w"},
	}
	for _, c := range cases {
		got := humanizeAge(now.Add(-c.ago), now)
		if got != c.want {
			t.Errorf("humanizeAge(-%v) = %q, want %q", c.ago, got, c.want)
		}
	}
}

func TestShortenPath(t *testing.T) {
	home := "/Users/me"
	if got := shortenPath("/Users/me", home); got != "~" {
		t.Errorf("shortenPath(home) = %q, want ~", got)
	}
	if got := shortenPath("/Users/me/workspace/growth/apps/hootmail3", home); got != "apps/hootmail3" {
		t.Errorf("shortenPath = %q, want apps/hootmail3", got)
	}
	if got := shortenPath("/single", home); got != "single" {
		t.Errorf("shortenPath = %q, want single", got)
	}
}

func TestFuzzyMatch(t *testing.T) {
	if !fuzzyMatch("", "anything") {
		t.Error("empty pattern should match")
	}
	if !fuzzyMatch("hoot", "claude hootmail3 fix bounce") {
		t.Error("substring should match")
	}
	if !fuzzyMatch("hml", "claude hootmail3") {
		t.Error("subsequence should match")
	}
	if fuzzyMatch("zzz", "claude hootmail3") {
		t.Error("absent chars should not match")
	}
}
