package ui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Selecting text in the right pane. Capturing the mouse takes the terminal's
// own selection away, so drover does it instead: a press in the pane's
// terminal area anchors a selection, motion with the button held extends it
// cell by cell (partial lines included, like a terminal), and the release
// copies it to the clipboard with OSC 52, which tmux passes on when
// set-clipboard is on. A press and release on one cell is still a click.
//
// The press freezes the pane: while the button is held the pane shows the
// lines it showed then, so output arriving meanwhile cannot move the text
// under the pointer, and the copy is exactly what was highlighted. After the
// release the highlight stays until the next press, key or wheel, or until
// the pane shows anything else (new output, a resize).
//
// The copy is what the pane shows: a line the program wrapped comes out as
// separate lines, and the selection stops at the pane's edges (scroll first).

// cell is a position in the pane's terminal area, 0-based.
type cell struct{ col, row int }

type selection struct {
	key     string // the pane it was made in (scrollKey); "" when none
	pressed bool   // the left button went down in the pane and is held
	dragged bool   // the pointer left the anchor cell: not a click
	anchor  cell
	head    cell
	click   hit      // the press, replayed as a click when nothing was dragged
	lines   []string // the pane as the press found it, without the highlight
}

// selectable says whether the pane shows a capture that can be selected.
func (m Model) selectable(h hit) (string, bool) {
	key, ok := m.scrollKey()
	if !ok || h.zone != zonePane || m.paneRowAt(h.line) < 0 {
		return "", false
	}
	if _, _, shown := m.shownCapture(); !shown && (!m.scrolled() || !m.scroll.capOK) {
		return "", false
	}
	return key, true
}

// paneCell maps a screen cell to the pane's terminal area, clamped to it, so
// a drag past an edge selects up to that edge.
func (m Model) paneCell(x, y int) cell {
	line, chrome := y-1, paneChrome // the title bar is line -1
	if m.zoomed() {
		line, chrome = y, zoomChrome
	}
	col, r := m.panePoint(x, line-chrome)
	return cell{col - 1, r - 1}
}

func (m Model) pressSelection(key string, h hit, x, y int) (tea.Model, tea.Cmd) {
	p := m.paneCell(x, y)
	cols, rows := m.paneSize()
	m.sel = selection{key: key, pressed: true, anchor: p, head: p, click: h, lines: m.capturedLines(cols, rows)}
	return m, nil
}

func (m Model) dragSelection(x, y int) (tea.Model, tea.Cmd) {
	m.sel.head = m.paneCell(x, y)
	m.sel.dragged = m.sel.dragged || m.sel.head != m.sel.anchor
	return m, nil
}

// releaseSelection copies what was dragged over, or clicks where the press
// was when nothing was.
func (m Model) releaseSelection() (tea.Model, tea.Cmd) {
	sel := m.sel
	m.sel.pressed = false
	if !sel.dragged {
		m.sel = selection{}
		return m.handleClick(sel.click)
	}
	if !m.selShown() {
		m.sel = selection{}
		return m, nil
	}
	text := m.selectedText()
	if text == "" {
		m.sel = selection{} // only blanks: nothing to copy, nothing to show
		return m, nil
	}
	m.clearPrompts()
	n := strings.Count(text, "\n") + 1
	m.notice = fmt.Sprintf("copied %d characters", len([]rune(text)))
	if n > 1 {
		m.notice = fmt.Sprintf("copied %d lines", n)
	}
	return m, tea.SetClipboard(text)
}

// selShown says whether a selection is to be drawn on the pane shown now.
func (m Model) selShown() bool {
	if !m.sel.dragged {
		return false
	}
	key, ok := m.scrollKey()
	return ok && key == m.sel.key
}

// frozen says whether the pane is to show the lines the press found.
func (m Model) frozen() bool {
	key, ok := m.scrollKey()
	return m.sel.pressed && ok && key == m.sel.key
}

// selRange is the selection in reading order, end inclusive.
func (m Model) selRange() (from, to cell) {
	a, b := m.sel.anchor, m.sel.head
	if b.row < a.row || (b.row == a.row && b.col < a.col) {
		a, b = b, a
	}
	return a, b
}

// selCols is the half-open column span selected on row r of a pane w wide.
func (m Model) selCols(r, w int) (int, int, bool) {
	from, to := m.selRange()
	if r < from.row || r > to.row {
		return 0, 0, false
	}
	c0, c1 := 0, w
	if r == from.row {
		c0 = from.col
	}
	if r == to.row {
		c1 = min(to.col+1, w)
	}
	return c0, c1, c0 < c1
}

// highlight draws the selection over the pane's lines in reverse video, the
// way withCursor draws the cursor.
func (m Model) highlight(lines []string, w int) []string {
	if m.frozen() {
		lines = slices.Clone(m.sel.lines)
	}
	if !m.selShown() || !slices.Equal(lines, m.sel.lines) {
		return lines
	}
	for r, l := range lines {
		if c0, c1, ok := m.selCols(r, w); ok {
			lines[r] = ansi.Cut(l, 0, c0) + "\x1b[7m" + ansi.Strip(ansi.Cut(l, c0, c1)) + "\x1b[27m" + ansi.Cut(l, c1, w)
		}
	}
	return lines
}

// selectedText is the selected cells as plain text, each line's trailing
// blanks (the padding of the screen) dropped.
func (m Model) selectedText() string {
	cols, _ := m.paneSize()
	var out []string
	for r, l := range m.sel.lines {
		if c0, c1, ok := m.selCols(r, cols); ok {
			out = append(out, strings.TrimRight(ansi.Strip(ansi.Cut(l, c0, c1)), " "))
		}
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}
