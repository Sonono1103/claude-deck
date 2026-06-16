package tui

import (
	"fmt"
	"strings"
	"time"
)

func ago(t time.Time) string {
	if t.IsZero() {
		return "?"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// shortModel renders a model id compactly: "claude-opus-4-8" -> "opus4.8",
// "claude-haiku-4-5-20251001" -> "haiku4.5". Unrecognized shapes fall back to
// the id with the "claude-" prefix stripped.
func shortModel(m string) string {
	m = strings.TrimPrefix(m, "claude-")
	parts := strings.Split(m, "-")
	if len(parts) >= 3 && isDigits(parts[1]) && isDigits(parts[2]) {
		return parts[0] + parts[1] + "." + parts[2]
	}
	return m
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}
