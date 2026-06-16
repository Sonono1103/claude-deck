# claude-deck — a Claude Code session manager for the terminal

**`claude-deck` is a fast, keyboard-driven terminal UI (TUI) for managing
[Claude Code](https://claude.com/claude-code) sessions** — browse, search,
resume, pin, and delete your Claude Code conversations under
`~/.claude/projects` without leaving the terminal.

Think of it as a session browser / session switcher for the `claude` CLI: pick
up any past Claude Code conversation and resume it in its original working
directory.

Resuming a session launches the `claude` CLI directly, so it reuses your
existing Claude subscription auth rather than separate Anthropic API billing.

![claude-deck demo](docs/demo.gif)

## Features

- Session list across all projects, most recent first
- Live filter by title or project (`/`)
- Pin important sessions to the top (`p`), persisted across runs
- Resume a session in its original working directory (`enter`)
- Soft-delete to a trash directory, recoverable later (`d`)
- Append-aware metadata cache for fast startup

## Install

```
go install github.com/Sonono1103/claude-deck/cmd/ccdeck@latest
```

This installs a `ccdeck` binary into `$(go env GOPATH)/bin`; make sure that
directory is on your `PATH`, then run:

```
ccdeck
```

## Keys

| Key | Action |
|-----|--------|
| `↑`/`k`, `↓`/`j` | Move cursor |
| `ctrl+u`/`ctrl+d`, `pgup`/`pgdn` | Page up / down |
| `g`/`G` | Jump to top / bottom |
| `/` | Filter (`esc` to cancel, `enter` to apply) |
| `enter` | Resume selected session |
| `p` | Pin / unpin selected session |
| `d` | Delete selected session (confirm with `y`) |
| `q`/`esc` | Quit |

## Storage

`ccdeck` reads sessions from `~/.claude/projects` and never modifies them. Its
own state lives under your XDG data and cache directories:

- Pins and other metadata, deleted sessions, and live state — XDG data dir
  (`~/.local/share/claude-deck/` on Linux/macOS)
- Session metadata cache — XDG cache dir (`~/Library/Caches/claude-deck/` on
  macOS)

## Status

Early MVP. Chat/dev separation is planned.

## License

MIT — see [LICENSE](LICENSE).
