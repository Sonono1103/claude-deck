package domain

import (
	"path/filepath"
	"time"
)

// Source identifies which CLI produced a session.
type Source string

const (
	SourceClaude Source = "claude"
)

// LiveState is a session's live lifecycle, as last reported by a Claude Code hook.
type LiveState string

const (
	LiveRunning    LiveState = "running"
	LiveWaiting    LiveState = "waiting"
	LiveNeedsInput LiveState = "needs_input"
	// LiveUnknown means a hook fired but the lifecycle is genuinely ambiguous (a
	// resume/compact SessionStart): don't assert running or waiting.
	LiveUnknown LiveState = "unknown"
)

// Session is the aggregated view of a single Claude Code session log.
type Session struct {
	ID           string
	Source       Source
	FilePath     string
	Title        string
	Preview      string
	Model        string
	CWD          string
	GitBranch    string
	MessageCount int
	FirstTS      time.Time
	LastTS       time.Time
	IsSidechain  bool
	Pinned       bool
	// Awaiting is the raw signal from the transcript: Claude ended its turn.
	Awaiting bool
	// Waiting is the application's final decision (live hook state, else the
	// recency heuristic) that the session is left waiting for user input.
	Waiting bool
}

// Project is the basename of the session's working directory.
func (s Session) Project() string {
	if s.CWD == "" {
		return ""
	}
	return filepath.Base(s.CWD)
}

// SessionRepository reads sessions from an underlying store.
type SessionRepository interface {
	List() ([]Session, error)
}
