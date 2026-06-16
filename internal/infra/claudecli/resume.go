package claudecli

import (
	"os"
	"os/exec"

	"github.com/Sonono1103/claude-deck/internal/domain"
)

// ResumeCommand builds a command that resumes the given session via the Claude
// Code CLI, wired to the current terminal. It runs in the session's working
// directory so relative context resolves correctly.
//
// Using the `claude` CLI keeps usage on the user's existing subscription auth
// rather than incurring separate API billing.
func ResumeCommand(s domain.Session) *exec.Cmd {
	cmd := exec.Command("claude", "--resume", s.ID)
	if s.CWD != "" {
		cmd.Dir = s.CWD
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}
