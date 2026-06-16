package tui

import "github.com/charmbracelet/lipgloss"

// palette (indigo/violet accents on a dark surface, opcode-like)
var (
	cAccent  = lipgloss.Color("141") // violet
	cProject = lipgloss.Color("111") // blue
	cModel   = lipgloss.Color("250") // gray
	cAge     = lipgloss.Color("108") // muted green
	cTitle   = lipgloss.Color("252") // near-white
	cDim     = lipgloss.Color("247") // readable secondary text
	cBorder  = lipgloss.Color("238") // subtle frame/divider
	cSelBg   = lipgloss.Color("237")
	cDanger  = lipgloss.Color("203")
	cGood    = lipgloss.Color("114")
	cAwait   = lipgloss.Color("214") // amber: waiting for user input
)

var (
	frameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cBorder).
			Padding(0, 1)

	brandStyle   = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
	countStyle   = lipgloss.NewStyle().Foreground(cDim)
	dividerSt    = lipgloss.NewStyle().Foreground(cBorder)
	colHeadStyle = lipgloss.NewStyle().Bold(true).Foreground(cDim)

	ageStyle      = lipgloss.NewStyle().Foreground(cAge)
	awaitStyle    = lipgloss.NewStyle().Bold(true).Foreground(cAwait)
	projectStyle  = lipgloss.NewStyle().Foreground(cProject)
	modelStyle    = lipgloss.NewStyle().Foreground(cModel)
	msgsStyle     = lipgloss.NewStyle().Foreground(cDim)
	titleColStyle = lipgloss.NewStyle().Foreground(cTitle)
	markerStyle   = lipgloss.NewStyle().Foreground(cAccent)

	rowSelStyle = lipgloss.NewStyle().Background(cSelBg)

	helpStyle    = lipgloss.NewStyle().Foreground(cDim)
	keyStyle     = lipgloss.NewStyle().Foreground(cAccent)
	errorStyle   = lipgloss.NewStyle().Foreground(cDanger)
	confirmStyle = lipgloss.NewStyle().Bold(true).Foreground(cDanger)
	statusStyle  = lipgloss.NewStyle().Foreground(cGood)
	filterStyle  = lipgloss.NewStyle().Foreground(cAccent)
)
