package tui

import (
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Sonono1103/claude-deck/internal/domain"
	"github.com/Sonono1103/claude-deck/internal/usecase"
)

// Model is the root Bubble Tea model for the session list.
type Model struct {
	svc *usecase.SessionService

	filter   textinput.Model
	sessions []domain.Session // all loaded, sorted
	visible  []domain.Session // currently shown (after filter)

	cursor int
	offset int

	query      string
	filtering  bool
	confirming bool
	pending    *domain.Session
	resume     *domain.Session
	status     string

	loading bool
	err     error
	width   int
	height  int
}

func New(svc *usecase.SessionService) Model {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "filter by title or project…"
	return Model{svc: svc, filter: ti, loading: true}
}

// ResumeTarget reports the session the user chose to resume, if any.
func (m Model) ResumeTarget() (domain.Session, bool) {
	if m.resume == nil {
		return domain.Session{}, false
	}
	return *m.resume, true
}

// messages

type sessionsMsg []domain.Session
type deletedMsg struct{ id string }
type pinToggledMsg struct {
	id     string
	pinned bool
}
type errMsg struct{ err error }

func (m Model) loadSessions() tea.Msg {
	sessions, err := m.svc.List()
	if err != nil {
		return errMsg{err}
	}
	return sessionsMsg(sessions)
}

func (m Model) deleteCmd(s domain.Session) tea.Cmd {
	return func() tea.Msg {
		if err := m.svc.Delete(s); err != nil {
			return errMsg{err}
		}
		return deletedMsg{id: s.ID}
	}
}

func (m Model) togglePinCmd(id string) tea.Cmd {
	return func() tea.Msg {
		pinned, err := m.svc.TogglePin(id)
		if err != nil {
			return errMsg{err}
		}
		return pinToggledMsg{id: id, pinned: pinned}
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadSessions
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clampView()
		return m, nil

	case sessionsMsg:
		m.loading = false
		m.sessions = []domain.Session(msg)
		m.applyFilter()
		return m, nil

	case deletedMsg:
		m.sessions = slices.DeleteFunc(m.sessions, func(s domain.Session) bool {
			return s.ID == msg.id
		})
		m.applyFilter()
		m.status = "deleted ✓"
		return m, nil

	case pinToggledMsg:
		for i := range m.sessions {
			if m.sessions[i].ID == msg.id {
				m.sessions[i].Pinned = msg.pinned
				break
			}
		}
		usecase.SortSessions(m.sessions)
		m.applyFilter()
		m.focusByID(msg.id)
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	m.status = ""

	if m.confirming {
		switch msg.String() {
		case "y", "enter":
			s := *m.pending
			m.confirming, m.pending = false, nil
			return m, m.deleteCmd(s)
		case "n", "esc":
			m.confirming, m.pending = false, nil
		}
		return m, nil
	}

	if m.filtering {
		switch msg.String() {
		case "esc":
			m.filtering = false
			m.filter.Blur()
			m.filter.SetValue("")
			m.query = ""
			m.applyFilter()
			return m, nil
		case "enter":
			m.filtering = false
			m.filter.Blur()
			return m, nil
		case "up":
			m.move(-1)
			return m, nil
		case "down":
			m.move(1)
			return m, nil
		}
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.query = m.filter.Value()
		m.applyFilter()
		return m, cmd
	}

	switch msg.String() {
	case "q", "esc":
		return m, tea.Quit
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "pgup", "ctrl+u":
		m.move(-m.listHeight())
	case "pgdown", "ctrl+d":
		m.move(m.listHeight())
	case "home", "g":
		m.move(-len(m.visible))
	case "end", "G":
		m.move(len(m.visible))
	case "/":
		m.filtering = true
		m.clampView()
		return m, m.filter.Focus()
	case "enter":
		if s, ok := m.selected(); ok {
			m.resume = &s
			return m, tea.Quit
		}
	case "d":
		if s, ok := m.selected(); ok {
			m.confirming = true
			m.pending = &s
		}
	case "p":
		if s, ok := m.selected(); ok {
			return m, m.togglePinCmd(s.ID)
		}
	}
	return m, nil
}

func (m *Model) move(delta int) {
	n := len(m.visible)
	if n == 0 {
		m.cursor, m.offset = 0, 0
		return
	}
	m.cursor = clamp(m.cursor+delta, 0, n-1)
	m.ensureVisible()
}

func (m *Model) ensureVisible() {
	h := m.listHeight()
	if h <= 0 {
		return
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	m.offset = clamp(m.offset, 0, max(len(m.visible)-h, 0))
}

func (m *Model) clampView() {
	m.cursor = clamp(m.cursor, 0, max(len(m.visible)-1, 0))
	m.ensureVisible()
}

func (m *Model) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.query))
	if q == "" {
		m.visible = m.sessions
	} else {
		out := make([]domain.Session, 0, len(m.sessions))
		for _, s := range m.sessions {
			if strings.Contains(strings.ToLower(s.Title), q) ||
				strings.Contains(strings.ToLower(s.Project()), q) {
				out = append(out, s)
			}
		}
		m.visible = out
	}
	m.cursor, m.offset = 0, 0
}

func (m *Model) focusByID(id string) {
	for i, s := range m.visible {
		if s.ID == id {
			m.cursor = i
			m.ensureVisible()
			return
		}
	}
}

func (m Model) selected() (domain.Session, bool) {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return domain.Session{}, false
	}
	return m.visible[m.cursor], true
}

func clamp(v, lo, hi int) int {
	return min(max(v, lo), hi)
}
