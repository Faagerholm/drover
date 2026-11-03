package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/lnware/drover/internal/tmux"
)

// selectModel is a mouseModel whose first row has a running agent showing
// lines, with no tmux behind it: selecting only reads the capture.
func selectModel(t *testing.T, lines ...string) Model {
	t.Helper()
	m := mouseModel(t)
	manyRows(&m, 2)
	m.rows[0].running = true
	m.all = m.rows
	m.preview = preview{session: m.target(m.rows[0]), ok: true, cap: tmux.Capture{Lines: lines}}
	return m
}

// at is the screen cell of a cell in the pane's terminal area.
func at(m Model, col, row int) (int, int) {
	leftW, _, _ := m.geometry()
	return leftW + 1 + col, 1 + paneChrome + row
}

// dragAcross presses on from, moves through to and releases there.
func dragAcross(m Model, from, to [2]int) (Model, tea.Cmd) {
	x0, y0 := at(m, from[0], from[1])
	x1, y1 := at(m, to[0], to[1])
	m = step(m, click(x0, y0, tea.MouseLeft))
	m = step(m, motion(x1, y1))
	next, cmd := m.Update(release(x1, y1))
	return next.(Model), cmd
}

func TestDragInPaneCopiesSelection(t *testing.T) {
	lines := []string{"hello \x1b[1mworld\x1b[0m", "foo bar", "", "last line"}
	for _, c := range []struct {
		name     string
		from, to [2]int
		want     string
	}{
		{"part of a line", [2]int{6, 0}, [2]int{10, 0}, "world"},
		{"across lines", [2]int{6, 0}, [2]int{2, 1}, "world\nfoo"},
		{"backwards", [2]int{2, 1}, [2]int{6, 0}, "world\nfoo"},
		{"keeps blank lines, drops padding", [2]int{4, 1}, [2]int{3, 3}, "bar\n\nlast"},
		{"past the edge clamps", [2]int{5, 3}, [2]int{500, 3}, "line"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m, cmd := dragAcross(selectModel(t, lines...), c.from, c.to)
			if cmd == nil {
				t.Fatal("release after a drag copied nothing")
			}
			if got := fmt.Sprint(cmd()); got != c.want {
				t.Errorf("copied %q, want %q", got, c.want)
			}
			if !strings.Contains(m.render(), "\x1b[7m") {
				t.Error("the selection is not highlighted")
			}
			if !strings.HasPrefix(m.notice, "copied") {
				t.Errorf("notice %q, want one saying what was copied", m.notice)
			}
			if got := step(m, tea.KeyPressMsg{Code: 'j'}); got.selShown() {
				t.Error("a key left the selection drawn")
			}
		})
	}
}

func TestClickInPaneIsNotASelection(t *testing.T) {
	m := selectModel(t, "hello world")
	m.mode = modeFocus
	x, y := at(m, 3, 0)
	m = step(m, click(x, y, tea.MouseLeft))
	next, cmd := m.Update(release(x, y))
	m = next.(Model)
	if cmd != nil || m.notice != "" || m.selShown() || m.sel.pressed {
		t.Errorf("a click copied or left a selection: cmd %v notice %q sel %+v", cmd != nil, m.notice, m.sel)
	}
	// In the list a click on the pane still focuses it, on release.
	m = selectModel(t, "hello world")
	m = step(m, click(x, y, tea.MouseLeft))
	if m.mode != modeList {
		t.Error("the press focused the pane before the release")
	}
	if m = step(m, release(x, y)); m.mode != modeFocus {
		t.Errorf("click on the pane left mode %d, want focus", m.mode)
	}
}

func TestSelectionOnlyOverTheCapture(t *testing.T) {
	m := selectModel(t, "hello world")
	leftW, _, _ := m.geometry()
	// The pane's header is not the capture: a press there is a plain click.
	if got := step(m, click(leftW+3, 2, tea.MouseLeft)); got.sel.pressed {
		t.Error("a press on the pane header started a selection")
	}
	m.preview.ok = false
	x, y := at(m, 1, 0)
	if got := step(m, click(x, y, tea.MouseLeft)); got.sel.pressed {
		t.Error("a press on a pane with no capture started a selection")
	}
}

func TestSelectionFreezesThePaneAndFollowsWhatItShows(t *testing.T) {
	m := selectModel(t, "first", "second")
	x0, y0 := at(m, 0, 0)
	x1, y1 := at(m, 4, 0)
	m = step(m, click(x0, y0, tea.MouseLeft))
	m = step(m, motion(x1, y1))
	// Output arrives mid-drag: the pane keeps showing what was pressed on.
	m.preview.cap = tmux.Capture{Lines: []string{"second", "third"}}
	if got := paneText(m)[0]; got != "first" {
		t.Errorf("pane moved under the drag: row 0 is %q", got)
	}
	next, cmd := m.Update(release(x1, y1))
	m = next.(Model)
	if cmd == nil || fmt.Sprint(cmd()) != "first" {
		t.Fatal("the copy is not what was highlighted")
	}
	// Released, the pane goes live again and the stale highlight is gone.
	if got := paneText(m)[0]; got != "second" {
		t.Errorf("pane still frozen after the release: row 0 is %q", got)
	}
	if strings.Contains(m.render(), "\x1b[7m") {
		t.Error("highlight kept on text that moved")
	}
}

func TestDragOverBlanksCopiesNothing(t *testing.T) {
	m, cmd := dragAcross(selectModel(t, "x"), [2]int{0, 5}, [2]int{20, 6})
	if cmd != nil || m.notice != "" || m.selShown() {
		t.Errorf("a blank drag copied or stayed: cmd %v notice %q", cmd != nil, m.notice)
	}
}
