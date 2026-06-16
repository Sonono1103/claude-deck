package usecase

import (
	"testing"
	"time"

	"github.com/Sonono1103/claude-deck/internal/domain"
)

type fakeRepo struct {
	sessions []domain.Session
	err      error
}

func (f fakeRepo) List() ([]domain.Session, error) { return f.sessions, f.err }

type fakePins struct {
	pinned map[string]bool
}

func (f fakePins) IsPinned(id string) bool { return f.pinned[id] }
func (f *fakePins) SetPinned(id string, pinned bool) error {
	if f.pinned == nil {
		f.pinned = map[string]bool{}
	}
	f.pinned[id] = pinned
	return nil
}

type liveEntry struct {
	state domain.LiveState
	alive bool
	ok    bool
}

type fakeLive struct {
	entries map[string]liveEntry
}

func (f fakeLive) Lookup(id string) (domain.LiveState, bool, bool) {
	e := f.entries[id]
	return e.state, e.alive, e.ok
}

type fakeTrash struct{ trashed []domain.Session }

func (f *fakeTrash) Trash(s domain.Session) error {
	f.trashed = append(f.trashed, s)
	return nil
}

func newService(repo domain.SessionRepository, live LiveStateProvider, pins PinStore) *SessionService {
	if pins == nil {
		pins = &fakePins{}
	}
	if live == nil {
		live = fakeLive{}
	}
	return NewSessionService(repo, &fakeTrash{}, pins, live)
}

func TestIsWaiting(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-1 * time.Hour)
	stale := now.Add(-25 * time.Hour)

	cases := []struct {
		name     string
		live     liveEntry
		awaiting bool
		lastTS   time.Time
		want     bool
	}{
		// Live state authoritative (seen + alive + not unknown).
		{"live waiting", liveEntry{"waiting", true, true}, false, recent, true},
		{"live needs_input", liveEntry{"needs_input", true, true}, false, recent, true},
		{"live running", liveEntry{"running", true, true}, true, recent, false},
		// Live state present but not authoritative -> fall back to heuristic.
		{"live unknown falls back", liveEntry{"unknown", true, true}, true, recent, true},
		{"live dead falls back", liveEntry{"waiting", false, true}, true, recent, true},
		{"live not seen falls back", liveEntry{"waiting", true, false}, true, recent, true},
		// Heuristic: awaiting + recent within window.
		{"heuristic awaiting recent", liveEntry{}, true, recent, true},
		{"heuristic not awaiting", liveEntry{}, false, recent, false},
		{"heuristic stale beyond window", liveEntry{}, true, stale, false},
		{"heuristic zero ts", liveEntry{}, true, time.Time{}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			live := fakeLive{entries: map[string]liveEntry{"s": c.live}}
			s := newService(fakeRepo{}, live, nil)
			sess := domain.Session{ID: "s", Awaiting: c.awaiting, LastTS: c.lastTS}
			if got := s.isWaiting(sess, now); got != c.want {
				t.Errorf("isWaiting = %v, want %v", got, c.want)
			}
		})
	}
}

func TestIsWaitingWindowBoundary(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	s := newService(fakeRepo{}, nil, nil)

	exactly := domain.Session{ID: "s", Awaiting: true, LastTS: now.Add(-awaitWindow)}
	if !s.isWaiting(exactly, now) {
		t.Error("exactly at awaitWindow should still be waiting (<=)")
	}
	over := domain.Session{ID: "s", Awaiting: true, LastTS: now.Add(-awaitWindow - time.Nanosecond)}
	if s.isWaiting(over, now) {
		t.Error("just past awaitWindow should not be waiting")
	}
}

func TestSortSessions(t *testing.T) {
	t0 := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)
	in := []domain.Session{
		{ID: "old", LastTS: t0.Add(1 * time.Hour)},
		{ID: "pinned-old", LastTS: t0, Pinned: true},
		{ID: "new", LastTS: t0.Add(3 * time.Hour)},
		{ID: "pinned-new", LastTS: t0.Add(2 * time.Hour), Pinned: true},
	}
	SortSessions(in)

	want := []string{"pinned-new", "pinned-old", "new", "old"}
	for i, id := range want {
		if in[i].ID != id {
			t.Errorf("position %d = %q, want %q", i, in[i].ID, id)
		}
	}
}

func TestListExcludesSidechainAndAppliesState(t *testing.T) {
	now := time.Now()
	repo := fakeRepo{sessions: []domain.Session{
		{ID: "main", LastTS: now, Awaiting: true},
		{ID: "sub", LastTS: now, IsSidechain: true},
	}}
	pins := &fakePins{pinned: map[string]bool{"main": true}}
	s := newService(repo, nil, pins)

	out, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d sessions, want 1 (sidechain excluded)", len(out))
	}
	if out[0].ID != "main" {
		t.Fatalf("got %q, want main", out[0].ID)
	}
	if !out[0].Pinned {
		t.Error("pin state not applied")
	}
	if !out[0].Waiting {
		t.Error("waiting heuristic not applied")
	}
}

func TestTogglePin(t *testing.T) {
	pins := &fakePins{}
	s := newService(fakeRepo{}, nil, pins)

	on, err := s.TogglePin("x")
	if err != nil || !on {
		t.Fatalf("first toggle: got (%v, %v), want (true, nil)", on, err)
	}
	off, err := s.TogglePin("x")
	if err != nil || off {
		t.Fatalf("second toggle: got (%v, %v), want (false, nil)", off, err)
	}
}

func TestDelete(t *testing.T) {
	trash := &fakeTrash{}
	s := NewSessionService(fakeRepo{}, trash, &fakePins{}, fakeLive{})
	sess := domain.Session{ID: "gone"}
	if err := s.Delete(sess); err != nil {
		t.Fatal(err)
	}
	if len(trash.trashed) != 1 || trash.trashed[0].ID != "gone" {
		t.Errorf("session not trashed: %+v", trash.trashed)
	}
}
