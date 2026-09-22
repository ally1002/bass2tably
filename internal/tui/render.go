package tui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"bass2tably/internal/tab"
)

type paint struct{ pre, post string }

const paintProbe = "\x1f"

func newPaint(st lipgloss.Style) paint {
	s := st.Render(paintProbe)
	i := strings.Index(s, paintProbe)
	if i < 0 {
		return paint{}
	}
	return paint{pre: s[:i], post: s[i+len(paintProbe):]}
}

type paints struct {
	name, fret, sustain                                    []paint
	bar, dash, cursor, cursorDash, played, ruler, rulerCur paint
}

func newPaints(st styles, rows int) paints {
	p := paints{
		bar:        newPaint(st.bar),
		dash:       newPaint(st.dash),
		cursor:     newPaint(st.cursor),
		cursorDash: newPaint(st.cursorDash),
		played:     newPaint(st.played),
		ruler:      newPaint(st.ruler),
		rulerCur:   newPaint(st.rulerCur),
	}
	for s := 0; s < rows; s++ {
		p.name = append(p.name, newPaint(st.name(s)))
		p.fret = append(p.fret, newPaint(st.fret(s)))
		p.sustain = append(p.sustain, newPaint(st.sustain(s)))
	}
	return p
}

type runWriter struct {
	b     strings.Builder
	paint paint
	buf   strings.Builder
}

func (w *runWriter) write(p paint, s string) {
	if p != w.paint {
		w.flush()
		w.paint = p
	}
	w.buf.WriteString(s)
}

func (w *runWriter) flush() {
	if w.buf.Len() == 0 {
		return
	}
	w.b.WriteString(w.paint.pre)
	w.b.WriteString(w.buf.String())
	w.b.WriteString(w.paint.post)
	w.buf.Reset()
}

func (w *runWriter) String() string {
	w.flush()
	return w.b.String()
}

func renderSheet(sh tab.Sheet, g tab.Glyphs, st styles, cur int, shading bool) string {
	var b strings.Builder
	p := newPaints(st, sh.Rows)
	last := len(sh.Systems) - 1
	for si, sys := range sh.Systems {
		if si > 0 {
			b.WriteString("\n")
		}
		b.WriteString(rulerRow(sh, sys, p, cur))
		b.WriteString("\n")
		for s := sh.Rows - 1; s >= 0; s-- {
			b.WriteString(stringRow(sh, sys, s, g, p, cur, shading, si == last))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func rulerRow(sh tab.Sheet, sys tab.System, p paints, cur int) string {
	var w runWriter
	w.write(p.ruler, strings.Repeat(" ", sh.NameW+1))
	fieldW := sh.SlotsPerMeasure*sh.CellW + 1
	for _, m := range sys.Measures {
		label := strconv.Itoa(m.Index + 1)
		style := p.ruler
		if cur >= 0 && cur/sh.SlotsPerMeasure == m.Index {
			style = p.rulerCur
		}
		w.write(style, label+strings.Repeat(" ", max(fieldW-len(label), 0)))
	}
	return w.String()
}

func stringRow(sh tab.Sheet, sys tab.System, s int, g tab.Glyphs, p paints, cur int, shading, last bool) string {
	var w runWriter
	w.write(p.name[s], sh.Names[s]+strings.Repeat(" ", sh.NameW-len(sh.Names[s])))
	w.write(p.bar, g.Bar)
	for mi, m := range sys.Measures {
		for _, col := range m.Columns {
			w.write(cellPaint(p, col, col.Cells[s], s, cur, shading), sh.CellText(col, s, g))
		}
		if last && mi == len(sys.Measures)-1 {
			w.write(p.bar, g.End)
		} else {
			w.write(p.bar, g.Bar)
		}
	}
	return w.String()
}

func cellPaint(p paints, col tab.Column, c tab.Cell, s, cur int, shading bool) paint {
	switch {
	case col.Slot == cur && c.Kind == tab.Onset:
		return p.cursor
	case col.Slot == cur:
		return p.cursorDash
	case c.Kind == tab.Onset && shading && col.Slot < cur:
		return p.played
	case c.Kind == tab.Onset:
		return p.fret[s]
	case c.Kind == tab.Sustain || c.Kind == tab.Tied:
		return p.sustain[s]
	default:
		return p.dash
	}
}

func fitMeasures(width, nameW, cellW, spm, measures int) int {
	mw := spm*cellW + 1
	n := (width - nameW - 1) / mw
	if n < 1 {
		n = 1
	}
	if measures > 0 && n > measures {
		n = measures
	}
	return n
}
