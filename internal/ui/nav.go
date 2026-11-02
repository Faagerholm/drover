package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/lnware/drover/internal/tmux"
)

// Seamless pane navigation, like vim-tmux-navigator: ctrl+h and ctrl+l move
// between the list and the pane, and at drover's own edges the key goes to
// the user's tmux, which moves to the neighbouring pane there. The user's
// tmux must pass the keys to drover first, as it does for vim (README).
//
// In focus mode the keys are taken from the agent, so they never reach it
// (Claude Code's ctrl+j newline included). Outside a tmux pane there is no
// edge to hand off to, so only ctrl+h is taken there and ctrl+j/k/l still
// reach the agent. nav_keys = false gives them all back.

// navDir is the select-pane direction of a navigation key.
func navDir(key string) (string, bool) {
	switch key {
	case "ctrl+h":
		return "L", true
	case "ctrl+j":
		return "D", true
	case "ctrl+k":
		return "U", true
	case "ctrl+l":
		return "R", true
	}
	return "", false
}

// outerSelector is the outer tmux's select-pane, or nil outside a tmux pane.
func outerSelector() func(dir string) error {
	if o, ok := tmux.OuterFromEnv(); ok {
		return o.SelectPane
	}
	return nil
}

// navListKey handles a navigation key in the list: ctrl+l focuses the pane
// (like enter on a worktree or companion row); everything else, and ctrl+l on
// a header, leaves drover.
func (m Model) navListKey(key string) (tea.Model, tea.Cmd, bool) {
	dir, ok := navDir(key)
	if !ok || !m.cfg.NavKeys {
		return m, nil, false
	}
	if dir == "R" && m.hdr == "" {
		if _, ok := m.selected(); ok {
			next, cmd := m.focus()
			return next, cmd, true
		}
	}
	return m, m.navOut(dir), true
}

// navFocusKey handles a navigation key in focus mode: ctrl+h goes back to the
// list, everything else leaves drover. Zoomed, the pane is all there is, so
// every key leaves. A key with nowhere to go is not taken: it is the agent's.
func (m Model) navFocusKey(key string) (tea.Model, tea.Cmd, bool) {
	dir, ok := navDir(key)
	if !ok || !m.cfg.NavKeys {
		return m, nil, false
	}
	if dir == "L" && !m.zoomed() {
		next, cmd := m.leaveFocus()
		return next, cmd, true
	}
	if m.selectOuter == nil {
		return m, nil, false
	}
	return m, m.navOut(dir), true
}

// navOut asks the user's tmux to move to the pane in dir. Outside tmux, or
// with no pane there, nothing happens.
func (m Model) navOut(dir string) tea.Cmd {
	nav := m.selectOuter
	if nav == nil {
		return nil
	}
	return func() tea.Msg {
		_ = nav(dir)
		return nil
	}
}
