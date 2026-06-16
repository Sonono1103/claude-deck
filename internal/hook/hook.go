// Package hook handles a single Claude Code hook invocation: it reads the event
// payload from stdin and records the session's lifecycle state.
package hook

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/Sonono1103/claude-deck/internal/domain"
	"github.com/Sonono1103/claude-deck/internal/infra/livestate"
)

// Claude Code hook event names claude-deck subscribes to.
const (
	EventSessionStart     = "SessionStart"
	EventUserPromptSubmit = "UserPromptSubmit"
	EventPreToolUse       = "PreToolUse"
	EventStop             = "Stop"
	EventNotification     = "Notification"
	EventSessionEnd       = "SessionEnd"
)

// Events lists every event claude-deck must be hooked into; installing them all
// is what keeps the live-state lifecycle in stateFor complete.
var Events = []string{
	EventSessionStart,
	EventUserPromptSubmit,
	EventPreToolUse,
	EventStop,
	EventNotification,
	EventSessionEnd,
}

type payload struct {
	SessionID     string `json:"session_id"`
	CWD           string `json:"cwd"`
	HookEventName string `json:"hook_event_name"`
	Source        string `json:"source"` // SessionStart: startup|clear|resume|compact
	Message       struct {
		Type string `json:"type"`
	} `json:"message"`
}

// Run reads the hook payload from r and updates the live-state store. It is
// deliberately tolerant: any malformed input or unknown event is a no-op so the
// hook never disrupts Claude.
func Run(r io.Reader) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	var p payload
	if json.Unmarshal(raw, &p) != nil || p.SessionID == "" {
		return nil
	}

	if p.HookEventName == EventSessionEnd {
		return livestate.Upsert(func(m map[string]livestate.Entry) bool {
			if _, ok := m[p.SessionID]; !ok {
				return false
			}
			delete(m, p.SessionID)
			return true
		})
	}

	state, ok := stateFor(p)
	if !ok {
		return nil
	}
	pid := os.Getppid() // the claude process that spawned this hook
	return livestate.Upsert(func(m map[string]livestate.Entry) bool {
		// Skip the rewrite when nothing material changed (ignoring UpdatedAt),
		// so a PreToolUse stream on an already running session is a no-op.
		if cur, ok := m[p.SessionID]; ok && cur.State == state && cur.CWD == p.CWD && cur.PID == pid {
			return false
		}
		m[p.SessionID] = livestate.Entry{
			State:     state,
			CWD:       p.CWD,
			PID:       pid,
			UpdatedAt: time.Now().Unix(),
		}
		return true
	})
}

func stateFor(p payload) (domain.LiveState, bool) {
	switch p.HookEventName {
	case EventSessionStart:
		// A resume or compact reattaches to an existing session whose real
		// state we can't infer, so stay ambiguous instead of asserting running.
		switch p.Source {
		case "resume", "compact":
			return domain.LiveUnknown, true
		default: // startup, clear, or unset: a fresh turn is beginning
			return domain.LiveRunning, true
		}
	case EventUserPromptSubmit, EventPreToolUse:
		// PreToolUse also clears a stale needs_input: once a tool runs, the
		// user has answered the permission/elicitation prompt.
		return domain.LiveRunning, true
	case EventStop:
		return domain.LiveWaiting, true
	case EventNotification:
		switch p.Message.Type {
		case "idle_prompt":
			return domain.LiveWaiting, true
		case "permission_prompt", "elicitation_dialog":
			return domain.LiveNeedsInput, true
		}
	}
	return "", false
}
