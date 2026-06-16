package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Sonono1103/claude-deck/internal/hook"
	"github.com/Sonono1103/claude-deck/internal/hooksetup"
	"github.com/Sonono1103/claude-deck/internal/infra/claudecli"
	"github.com/Sonono1103/claude-deck/internal/infra/claudestore"
	"github.com/Sonono1103/claude-deck/internal/infra/livestate"
	"github.com/Sonono1103/claude-deck/internal/infra/meta"
	"github.com/Sonono1103/claude-deck/internal/infra/trash"
	"github.com/Sonono1103/claude-deck/internal/tui"
	"github.com/Sonono1103/claude-deck/internal/usecase"
)

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "hook":
		runHook()
	case "hooks":
		runHooks(os.Args[2:])
	default:
		runTUI()
	}
}

func runHook() {
	// Stay silent and exit 0 even on error so a hook never disrupts Claude.
	_ = hook.Run(os.Stdin)
}

func runHooks(args []string) {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "install":
		fail(hooksetup.Install(), "hooks install")
		fmt.Println("claude-deck hooks installed in ~/.claude/settings.json")
	case "uninstall":
		fail(hooksetup.Uninstall(), "hooks uninstall")
		fmt.Println("claude-deck hooks removed")
	case "status":
		installed, missing, err := hooksetup.Status()
		fail(err, "hooks status")
		fmt.Printf("installed: %v\nmissing:   %v\n", installed, missing)
	default:
		fmt.Fprintln(os.Stderr, "usage: claude-deck hooks <install|uninstall|status>")
		os.Exit(2)
	}
}

func runTUI() {
	repo, err := claudestore.New()
	fail(err, "init")
	tr, err := trash.New()
	fail(err, "init")
	pins, err := meta.New()
	fail(err, "init")
	live, err := livestate.NewProvider()
	fail(err, "init")

	svc := usecase.NewSessionService(repo, tr, pins, live)
	p := tea.NewProgram(tui.New(svc), tea.WithAltScreen())
	final, err := p.Run()
	fail(err, "run")

	// Hand the terminal over to `claude --resume` if a session was chosen.
	if m, ok := final.(tui.Model); ok {
		if s, ok := m.ResumeTarget(); ok {
			if err := claudecli.ResumeCommand(s).Run(); err != nil {
				fail(err, "resume")
			}
		}
	}
}

func fail(err error, ctx string) {
	if err != nil {
		fmt.Fprintln(os.Stderr, ctx+":", err)
		os.Exit(1)
	}
}
