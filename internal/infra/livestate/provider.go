package livestate

import "github.com/Sonono1103/claude-deck/internal/domain"

// Provider exposes a snapshot of live session state to the application layer.
// It is loaded once (at TUI startup) from the hook-written state file.
type Provider struct {
	sessions map[string]Entry
}

func NewProvider() (*Provider, error) {
	m, err := Load()
	if err != nil {
		return nil, err
	}
	return &Provider{sessions: m}, nil
}

// Lookup returns a session's last-known state and whether its process is alive.
// ok is false when no hook ever reported this session.
func (p *Provider) Lookup(id string) (state domain.LiveState, alive bool, ok bool) {
	e, found := p.sessions[id]
	if !found {
		return "", false, false
	}
	return e.State, Alive(e.PID), true
}
