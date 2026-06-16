package tui

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/Sonono1103/claude-deck/internal/domain"
)

func sampleSessions() []domain.Session {
	now := time.Now()
	return []domain.Session{
		{ID: "1", CWD: "/x/example-app", Model: "claude-opus-4-8", MessageCount: 12, Title: "全角タイトルの折り返し幅テスト", LastTS: now, Pinned: true},
		{ID: "2", CWD: "/x/demo-frontend", Model: "claude-sonnet-4-6", MessageCount: 8, Title: "add login form and wire up validation", LastTS: now.Add(-26 * time.Hour)},
		{ID: "3", CWD: "/x/sandbox", Model: "claude-opus-4-8", MessageCount: 3, Title: "investigate flaky test", LastTS: now.Add(-3 * time.Hour), Waiting: true},
		{ID: "4", CWD: "/x/demo-backend-work", Model: "claude-opus-4-8", MessageCount: 75, Title: "refactor cache layer", LastTS: now.Add(-72 * time.Hour)},
	}
}

func TestViewRenders(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)

	m := Model{width: 100, height: 16}
	m.sessions = sampleSessions()
	m.visible = m.sessions
	m.cursor = 1

	out := m.View()
	if out == "" {
		t.Fatal("empty view")
	}
	t.Logf("\n%s", out)
}
