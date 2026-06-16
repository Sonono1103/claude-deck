package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Sonono1103/claude-deck/internal/domain"
)

// fixed column widths; Title takes the remaining space
const (
	wMarker  = 2
	wAge     = 5
	wProj    = 24
	wModel   = 13
	wMsgs    = 4
	colGap   = 1
	framePad = 4 // border (2) + horizontal padding (2)
)

// cols holds resolved column widths for a given inner width.
type cols struct{ age, proj, model, msgs, title int }

func columns(inner int) cols {
	return cols{
		age:   wAge,
		proj:  wProj,
		model: wModel,
		msgs:  wMsgs,
		title: max(inner-wMarker-wAge-wProj-wModel-wMsgs-4*colGap, 10),
	}
}

// cell returns a single-line style clamped to exactly width w.
func cell(w int) lipgloss.Style {
	return lipgloss.NewStyle().Inline(true).Width(w).MaxWidth(w)
}

// fill pads lines up to height h with blank full-width rows.
func fill(lines []string, h, width int) []string {
	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return lines
}

func (m Model) innerWidth() int {
	w := m.width
	if w == 0 {
		w = 80
	}
	return max(w-framePad, 40)
}

func (m Model) listHeight() int {
	h := m.height
	if h == 0 {
		h = 24
	}
	chrome := 6 // border×2 + brand + col-header + divider + footer
	if m.filtering {
		chrome++
	}
	return max(h-chrome, 1)
}

func (m Model) View() string {
	if m.err != nil {
		return errorStyle.Render("error: "+m.err.Error()) + "\n"
	}
	if m.loading {
		return "  reading sessions…\n"
	}

	inner := m.innerWidth()
	lines := []string{
		m.headerLine(inner),
		m.colHeaderLine(inner),
		dividerSt.Render(strings.Repeat("─", inner)),
	}
	if m.filtering {
		lines = append(lines, pad(filterStyle.Render(m.filter.View()), inner))
	}
	lines = append(lines, m.listLines(inner)...)
	lines = append(lines, m.footerLine(inner))

	return frameStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) headerLine(inner int) string {
	left := brandStyle.Render("claude-deck")
	right := countStyle.Render(m.positionText())
	gap := max(inner-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) positionText() string {
	n := len(m.visible)
	if n == 0 {
		return "no sessions"
	}
	if m.query != "" {
		return fmt.Sprintf("%d/%d  ·  %s", m.cursor+1, n, pluralizeSessions(len(m.sessions)))
	}
	return fmt.Sprintf("%d/%d", m.cursor+1, n)
}

func (m Model) colHeaderLine(inner int) string {
	c := columns(inner)
	gap := " "
	cells := cell(c.age).Render("Age") + gap +
		cell(c.proj).Render("Project") + gap +
		cell(c.model).Render("Model") + gap +
		cell(c.msgs).Align(lipgloss.Right).Render("Msgs") + gap +
		cell(c.title).Render("Title")
	return "  " + colHeadStyle.Render(cells)
}

func (m Model) listLines(inner int) []string {
	h := m.listHeight()
	if len(m.visible) == 0 {
		empty := helpStyle.Render("  no matching sessions")
		return fill([]string{pad(empty, inner)}, h, inner)
	}

	end := min(m.offset+h, len(m.visible))
	out := make([]string, 0, h)
	for i := m.offset; i < end; i++ {
		out = append(out, m.rowLine(m.visible[i], i == m.cursor, inner))
	}
	return fill(out, h, inner)
}

func (m Model) rowLine(s domain.Session, selected bool, inner int) string {
	title := s.Title
	if title == "" {
		title = "(untitled)"
	}
	c := columns(inner)
	modelText := shortModel(s.Model)
	if modelText == "" {
		modelText = "-"
	}
	projText := s.Project()
	if projText == "" {
		projText = "-"
	}
	age := cell(c.age).Render(ago(s.LastTS))
	proj := cell(c.proj).Render(projText)
	model := cell(c.model).Render(modelText)
	msgs := cell(c.msgs).Align(lipgloss.Right).Render(strconv.Itoa(s.MessageCount))
	ttl := cell(c.title).Render(title)

	gap := " "
	body := age + gap + proj + gap + model + gap + msgs + gap + ttl

	star := " "
	if s.Pinned {
		star = "★"
	}

	if selected {
		marker := markerStyle.Background(cSelBg).Render("▌" + star)
		return marker + rowSelStyle.Foreground(cTitle).Width(inner-wMarker).Render(body)
	}

	ageCol := ageStyle.Render(age)
	if s.Waiting {
		ageCol = awaitStyle.Render(age)
	}
	colored := ageCol + gap +
		projectStyle.Render(proj) + gap +
		modelStyle.Render(model) + gap +
		msgsStyle.Render(msgs) + gap +
		titleColStyle.Render(ttl)

	lead := " "
	if s.Waiting {
		lead = awaitStyle.Render("●")
	}
	return lead + markerStyle.Render(star) + colored
}

func (m Model) footerLine(inner int) string {
	var content string
	switch {
	case m.confirming:
		t := "(untitled)"
		if m.pending != nil && m.pending.Title != "" {
			t = m.pending.Title
		}
		content = confirmStyle.Render(fmt.Sprintf("delete “%s” ?  y/n", truncate(t, 36)))
	case m.status != "":
		content = statusStyle.Render(m.status)
	default:
		content = hint("↑↓", "move") + "  " + hint("↵", "resume") + "  " +
			hint("/", "filter") + "  " + hint("p", "pin") + "  " + hint("d", "delete") + "  " + hint("q", "quit")
	}
	return pad(content, inner)
}

func hint(key, label string) string {
	return keyStyle.Render(key) + " " + helpStyle.Render(label)
}

// pad right-pads a (possibly styled) string to display width w.
func pad(s string, w int) string {
	gap := w - lipgloss.Width(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

func pluralizeSessions(n int) string {
	if n == 1 {
		return "1 session"
	}
	return strconv.Itoa(n) + " sessions"
}
