// Package hooksetup installs and removes claude-deck's Claude Code hooks in the
// user's ~/.claude/settings.json without disturbing other settings.
package hooksetup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Sonono1103/claude-deck/internal/hook"
	"github.com/Sonono1103/claude-deck/internal/infra/atomicfile"
)

type hookCmd struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type matcherGroup struct {
	Matcher string    `json:"matcher,omitempty"`
	Hooks   []hookCmd `json:"hooks"`
}

const commandSuffix = "hook" // our command is "<exe> hook"

func command() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return exe + " " + commandSuffix, nil
}

func isOurs(c hookCmd) bool {
	return strings.HasSuffix(strings.TrimSpace(c.Command), " "+commandSuffix)
}

func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// load returns the settings as a key-preserving map plus its parsed hooks map.
func load(path string) (map[string]json.RawMessage, map[string][]matcherGroup, error) {
	top := map[string]json.RawMessage{}
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &top); err != nil {
			return nil, nil, fmt.Errorf("parse %s: %w", path, err)
		}
	}
	hooks := map[string][]matcherGroup{}
	if rawHooks, ok := top["hooks"]; ok {
		_ = json.Unmarshal(rawHooks, &hooks)
	}
	return top, hooks, nil
}

func save(path string, top map[string]json.RawMessage, hooks map[string][]matcherGroup) error {
	if len(hooks) == 0 {
		delete(top, "hooks")
	} else {
		rawHooks, err := json.Marshal(hooks)
		if err != nil {
			return err
		}
		top["hooks"] = rawHooks
	}
	raw, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.Write(path, raw, 0o644)
}

// Install adds our hook to each tracked event, leaving existing hooks intact.
func Install() error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	cmd, err := command()
	if err != nil {
		return err
	}
	top, hooks, err := load(path)
	if err != nil {
		return err
	}
	for _, ev := range hook.Events {
		if hasOurHook(hooks[ev]) {
			continue
		}
		hooks[ev] = append(hooks[ev], matcherGroup{
			Hooks: []hookCmd{{Type: "command", Command: cmd}},
		})
	}
	return save(path, top, hooks)
}

// Uninstall removes only claude-deck's hook entries.
func Uninstall() error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	top, hooks, err := load(path)
	if err != nil {
		return err
	}
	for ev, groups := range hooks {
		kept := groups[:0]
		for _, g := range groups {
			g.Hooks = filterOutOurs(g.Hooks)
			if len(g.Hooks) > 0 {
				kept = append(kept, g)
			}
		}
		if len(kept) == 0 {
			delete(hooks, ev)
		} else {
			hooks[ev] = kept
		}
	}
	return save(path, top, hooks)
}

// Status reports which events currently have our hook installed.
func Status() (installed []string, missing []string, err error) {
	path, perr := settingsPath()
	if perr != nil {
		return nil, nil, perr
	}
	_, hooks, lerr := load(path)
	if lerr != nil {
		return nil, nil, lerr
	}
	for _, ev := range hook.Events {
		if hasOurHook(hooks[ev]) {
			installed = append(installed, ev)
		} else {
			missing = append(missing, ev)
		}
	}
	return installed, missing, nil
}

func hasOurHook(groups []matcherGroup) bool {
	return slices.ContainsFunc(groups, func(g matcherGroup) bool {
		return slices.ContainsFunc(g.Hooks, isOurs)
	})
}

func filterOutOurs(cmds []hookCmd) []hookCmd {
	kept := cmds[:0]
	for _, c := range cmds {
		if !isOurs(c) {
			kept = append(kept, c)
		}
	}
	return kept
}
