package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"bass2tably/internal/tab"
)

type runWriter struct {
	b     strings.Builder
	style lipgloss.Style
	key   string
	buf   strings.Builder
}

func (w *runWriter) write(key string, st lipgloss.Style, s string) {
	if key != w.key {
		w.flush()
		w.key = key
		w.style = st
	}
	w.buf.WriteString(s)
}

func (w *runWriter) flush() {
	if w.buf.Len() == 0 {
		return
	}
	w.b.WriteString(w.style.Render(w.buf.String()))
	w.buf.Reset()
}

func (w *runWriter) String() string {
	w.flush()
	return w.b.String()
}

func renderTab(plain string, l tab.Layout, st styles, curBlock, curCol int) string {
	lines := strings.Split(plain, "\n")
	stride := l.Stride()
	if l.Rows < 1 {
		return plain
	}
	for r, line := range lines {
		row := r % stride
		if row < l.Top || row >= l.Top+l.Rows || len(line) < l.NameW+1 {
			continue
		}
		block := r / stride
		s := l.Top + l.Rows - 1 - row
		lines[r] = renderTabRow(line, l, st, s, block == curBlock, curCol)
	}
	return strings.Join(lines, "\n")
}

func renderTabRow(line string, l tab.Layout, st styles, s int, cursorBlock bool, curCol int) string {
	var w runWriter
	w.write("name", st.name(s), line[:l.NameW])
	col := l.NameW
	for col < len(line) {
		if line[col] == '|' {
			w.write("bar", st.bar, "|")
			col++
			continue
		}
		end := col + l.CellW
		if end > len(line) {
			w.write("tail", st.dash, line[col:])
			break
		}
		cell := line[col:end]
		isFret := cell[0] >= '0' && cell[0] <= '9'
		switch {
		case cursorBlock && col == curCol && isFret:
			w.write("cursor", st.cursor, cell)
		case cursorBlock && col == curCol:
			w.write("cursorDash", st.cursorDash, cell)
		case isFret:
			w.write("fret", st.fret(s), cell)
		default:
			w.write("dash", st.dash, cell)
		}
		col = end
	}
	return w.String()
}

func fitMeasures(width int, l tab.Layout) int {
	mw := l.SlotsPerMeasure*l.CellW + 1
	n := (width - l.NameW - 1) / mw
	if n < 1 {
		n = 1
	}
	if l.Measures > 0 && n > l.Measures {
		n = l.Measures
	}
	return n
}
