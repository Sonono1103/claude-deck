package tui

import (
	"testing"
	"time"
)

func TestShortModel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"claude-opus-4-8", "opus4.8"},
		{"claude-sonnet-4-6", "sonnet4.6"},
		{"claude-haiku-4-5-20251001", "haiku4.5"},
		{"claude-opus-4", "opus-4"},     // too few parts: fallback
		{"claude-opus-x-8", "opus-x-8"}, // non-digit minor: fallback
		{"claude-opus-4-x", "opus-4-x"}, // non-digit patch: fallback
		{"gpt-4", "gpt-4"},              // no claude- prefix, fallback
		{"", ""},                        // empty
		{"claude-", ""},                 // prefix only
	}
	for _, c := range cases {
		if got := shortModel(c.in); got != c.want {
			t.Errorf("shortModel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want string
	}{
		{"hello", 10, "hello"}, // fits
		{"hello", 5, "hello"},  // exact fit
		{"hello", 4, "hel…"},   // truncated, ellipsis replaces last kept rune
		{"hello", 1, "…"},      // n==1 special case
		{"hello", 0, ""},       // n<=0
		{"hello", -3, ""},      // negative
		{"全角文字列", 3, "全角…"},    // multibyte counted by rune
		{"全角文字列", 5, "全角文字列"},  // exact rune fit
		{"全角文字列", 2, "全…"},     // multibyte truncated
	}
	for _, c := range cases {
		if got := truncate(c.s, c.n); got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.s, c.n, got, c.want)
		}
	}
}

func TestAgo(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		t    time.Time
		want string
	}{
		{"zero", time.Time{}, "?"},
		{"now", now.Add(-30 * time.Second), "now"},
		{"minutes", now.Add(-5 * time.Minute), "5m"},
		{"hours", now.Add(-3 * time.Hour), "3h"},
		{"days", now.Add(-50 * time.Hour), "2d"},
		{"minute boundary", now.Add(-90 * time.Second), "1m"},
		{"hour boundary", now.Add(-90 * time.Minute), "1h"},
	}
	for _, c := range cases {
		if got := ago(c.t); got != c.want {
			t.Errorf("%s: ago() = %q, want %q", c.name, got, c.want)
		}
	}
}
