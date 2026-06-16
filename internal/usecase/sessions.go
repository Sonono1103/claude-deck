package usecase

import (
	"sort"
	"time"

	"github.com/Sonono1103/claude-deck/internal/domain"
)

// awaitWindow bounds how recently an awaiting session must have been active to
// count as "left waiting for input" when no live hook state is available.
const awaitWindow = 24 * time.Hour

// Trasher soft-deletes a session's underlying storage.
type Trasher interface {
	Trash(domain.Session) error
}

// PinStore persists which sessions the user has pinned.
type PinStore interface {
	IsPinned(id string) bool
	SetPinned(id string, pinned bool) error
}

// LiveStateProvider reports live, hook-driven session state. ok is false when no
// hook ever reported the session (fall back to the heuristic then).
type LiveStateProvider interface {
	Lookup(id string) (state domain.LiveState, alive bool, ok bool)
}

// SessionService is the application layer over a SessionRepository.
type SessionService struct {
	repo  domain.SessionRepository
	trash Trasher
	pins  PinStore
	live  LiveStateProvider
}

func NewSessionService(repo domain.SessionRepository, trash Trasher, pins PinStore, live LiveStateProvider) *SessionService {
	return &SessionService{repo: repo, trash: trash, pins: pins, live: live}
}

// isWaiting decides whether to highlight a session as left waiting for input.
// Live hook state (with a process-liveness check) is authoritative; sessions
// never seen by a hook fall back to the jsonl end-of-turn heuristic.
func (s *SessionService) isWaiting(sess domain.Session, now time.Time) bool {
	// Trust live hook state only when it is authoritative: the session was
	// seen, its process is alive, and the state isn't the ambiguous "unknown"
	// (a resume/compact). Otherwise — never seen, dead process, or unknown —
	// fall back to the jsonl end-of-turn heuristic rather than asserting
	// not-waiting, which would hide a session that really is awaiting input.
	if state, alive, ok := s.live.Lookup(sess.ID); ok && alive && state != domain.LiveUnknown {
		return state == domain.LiveWaiting || state == domain.LiveNeedsInput
	}
	return sess.Awaiting && !sess.LastTS.IsZero() && now.Sub(sess.LastTS) <= awaitWindow
}

// Delete soft-deletes a session (moves it to trash).
func (s *SessionService) Delete(sess domain.Session) error {
	return s.trash.Trash(sess)
}

// TogglePin flips the pinned state of a session and reports the new state.
func (s *SessionService) TogglePin(id string) (bool, error) {
	pinned := !s.pins.IsPinned(id)
	if err := s.pins.SetPinned(id, pinned); err != nil {
		return false, err
	}
	return pinned, nil
}

// SortSessions orders sessions pinned-first, then most-recent-first.
func SortSessions(sessions []domain.Session) {
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].Pinned != sessions[j].Pinned {
			return sessions[i].Pinned
		}
		return sessions[i].LastTS.After(sessions[j].LastTS)
	})
}

// List returns resumable sessions (subagent sidechains excluded), sorted by
// most recent activity first.
func (s *SessionService) List() ([]domain.Session, error) {
	all, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	out := make([]domain.Session, 0, len(all))
	for _, sess := range all {
		if sess.IsSidechain {
			continue
		}
		sess.Pinned = s.pins.IsPinned(sess.ID)
		sess.Waiting = s.isWaiting(sess, now)
		out = append(out, sess)
	}

	SortSessions(out)
	return out, nil
}
