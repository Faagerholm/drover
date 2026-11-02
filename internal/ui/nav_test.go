package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func ctrl(c rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: c, Mod: tea.ModCtrl} }

// focusRunning puts the model in focus mode on a running agent, so keys that are
// not drover's are forwarded to it rather than dropping back to the list.
func focusRunning(m *Model) {
	m.rows[0].running = true
	m.mode = modeFocus
}

// TestNavKeys checks where each ctrl+hjkl goes: inside drover, to the user's
// tmux in a direction, or on to the agent. The outer tmux is a recorder,
// never a real one; noOuter is drover outside a tmux pane.
func TestNavKeys(t *testing.T) {
	for _, c := range []struct {
		name     string
		setup    func(*Model)
		noOuter  bool
		navOff   bool
		key      rune
		wantMode mode
		wantOut  string // the direction handed to the outer tmux, "" for none
	}{
		{name: "list ctrl+h leaves left", key: 'h', wantMode: modeList, wantOut: "L"},
		{name: "list ctrl+j leaves down", key: 'j', wantMode: modeList, wantOut: "D"},
		{name: "list ctrl+k leaves up", key: 'k', wantMode: modeList, wantOut: "U"},
		{name: "list ctrl+l focuses the pane", key: 'l', wantMode: modeNameAgent}, // no agent yet: asks for a name, like enter
		{name: "header ctrl+l leaves right", setup: func(m *Model) { m.hdr = orgID(m.rows[0]) }, key: 'l', wantMode: modeList, wantOut: "R"},
		{name: "list outside tmux does nothing", noOuter: true, key: 'j', wantMode: modeList},
		{name: "list ctrl+l outside tmux still focuses", noOuter: true, key: 'l', wantMode: modeNameAgent},
		{name: "focus ctrl+h back to the list", setup: focusRunning, key: 'h', wantMode: modeList},
		{name: "focus ctrl+l leaves right", setup: focusRunning, key: 'l', wantMode: modeFocus, wantOut: "R"},
		{name: "focus ctrl+j leaves down", setup: focusRunning, key: 'j', wantMode: modeFocus, wantOut: "D"},
		{name: "focus ctrl+k leaves up", setup: focusRunning, key: 'k', wantMode: modeFocus, wantOut: "U"},
		{name: "zoomed ctrl+h leaves left", setup: func(m *Model) { focusRunning(m); m.zoom = true }, key: 'h', wantMode: modeFocus, wantOut: "L"},
		{name: "focus outside tmux: ctrl+j is the agent's", setup: focusRunning, noOuter: true, key: 'j', wantMode: modeFocus},
		{name: "focus outside tmux: ctrl+h still goes back", setup: focusRunning, noOuter: true, key: 'h', wantMode: modeList},
		{name: "nav off: focus ctrl+h is the agent's", setup: focusRunning, navOff: true, key: 'h', wantMode: modeFocus},
		{name: "nav off: focus ctrl+l is the agent's", setup: focusRunning, navOff: true, key: 'l', wantMode: modeFocus},
		{name: "nav off: list ctrl+l does nothing", navOff: true, key: 'l', wantMode: modeList},
		{name: "help swallows ctrl+l", setup: func(m *Model) { m.help = true }, key: 'l', wantMode: modeList},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := mouseModel(t)
			manyRows(&m, 2)
			var out []string
			m.selectOuter = func(dir string) error { out = append(out, dir); return nil }
			if c.noOuter {
				m.selectOuter = nil
			}
			m.cfg.NavKeys = !c.navOff
			if c.setup != nil {
				c.setup(&m)
			}
			next, cmd := m.Update(ctrl(c.key))
			got := next.(Model)
			if cmd != nil { // a batch only returns its children, so nothing else runs
				cmd()
			}
			if got.mode != c.wantMode {
				t.Errorf("mode %d, want %d", got.mode, c.wantMode)
			}
			if got.cursor != 0 {
				t.Errorf("the key moved the selection to %d", got.cursor)
			}
			want := []string{}
			if c.wantOut != "" {
				want = []string{c.wantOut}
			}
			if len(out) != len(want) || (len(out) == 1 && out[0] != want[0]) {
				t.Errorf("outer tmux got %v, want %v", out, want)
			}
		})
	}
}
