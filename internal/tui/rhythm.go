package tui

import (
	"strings"

	"github.com/ally1002/bass2tably/internal/tab"
)

type rhythmGlyphs struct{ stem, rest, open, join, close, line rune }

var (
	rhythmUnicode = rhythmGlyphs{stem: '│', rest: '·', open: '└', join: '┴', close: '┘', line: '─'}
	rhythmASCII   = rhythmGlyphs{stem: '|', rest: '.', open: '\\', join: '+', close: '/', line: '-'}
)

type rhythmOpts struct {
	gridDiv int
	levels  []int
	glyphs  rhythmGlyphs
	on      bool
}

func beamLevels(gridDiv int) []int {
	levels := []int{gridDiv}
	if gridDiv == 4 {
		levels = append(levels, gridDiv/2)
	}
	return levels
}

func anyShorterThanBeat(sh tab.Sheet, gridDiv int) bool {
	for _, sys := range sh.Systems {
		for _, m := range sys.Measures {
			for _, col := range m.Columns {
				if col.Onset >= 0 && col.Len < gridDiv {
					return true
				}
			}
		}
	}
	return false
}

func (r rhythmOpts) rows() int {
	if !r.on {
		return 0
	}
	return 1 + len(r.levels)
}

type beam struct{ first, last int }

func beams(cols []tab.Column, gridDiv, shorterThan int) []beam {
	var out []beam
	for b := 0; b < len(cols); b += gridDiv {
		g := beam{first: -1}
		for c := b; c < b+gridDiv && c < len(cols); c++ {
			if cols[c].Onset < 0 || cols[c].Len >= shorterThan {
				continue
			}
			if g.first < 0 {
				g.first = c
			}
			g.last = c
		}
		if g.first >= 0 {
			out = append(out, g)
		}
	}
	return out
}

func blank(w int) []rune {
	row := make([]rune, w)
	for i := range row {
		row[i] = ' '
	}
	return row
}

func rhythmRows(m tab.Measure, gridDiv, cellW int, levels []int, g rhythmGlyphs) []string {
	w := len(m.Columns) * cellW
	stems := blank(w)
	for c, col := range m.Columns {
		x := c * cellW
		switch {
		case col.Onset >= 0:
			stems[x] = g.stem
		case col.Silent() && (c == 0 || !m.Columns[c-1].Silent()):
			stems[x] = g.rest
		}
	}
	rows := []string{string(stems)}
	for _, shorter := range levels {
		row := blank(w)
		for _, b := range beams(m.Columns, gridDiv, shorter) {
			x0, x1 := b.first*cellW, b.last*cellW
			if x0 == x1 {
				row[x0], row[x0+1] = g.open, g.line
				continue
			}
			for x := x0 + 1; x < x1; x++ {
				row[x] = g.line
			}
			row[x0], row[x1] = g.open, g.close
			for c := b.first + 1; c < b.last; c++ {
				if m.Columns[c].Onset >= 0 && m.Columns[c].Len < shorter {
					row[c*cellW] = g.join
				}
			}
		}
		rows = append(rows, string(row))
	}
	return rows
}

func systemRhythmRows(sh tab.Sheet, sys tab.System, r rhythmOpts) []string {
	n := r.rows()
	if n == 0 {
		return nil
	}
	parts := make([][]string, n)
	gutter := strings.Repeat(" ", sh.NameW+1)
	for i := range parts {
		parts[i] = append(parts[i], gutter)
	}
	for _, m := range sys.Measures {
		for i, row := range rhythmRows(m, r.gridDiv, sh.CellW, r.levels, r.glyphs) {
			parts[i] = append(parts[i], row, " ")
		}
	}
	rows := make([]string, n)
	for i := range rows {
		rows[i] = strings.Join(parts[i], "")
	}
	return rows
}
