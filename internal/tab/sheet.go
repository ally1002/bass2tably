package tab

import (
	"strconv"
	"strings"

	"bass2tably/internal/music"
)

type CellKind uint8

const (
	Empty CellKind = iota
	Onset
	Sustain
	Tied
)

type Cell struct {
	Kind CellKind
	Fret int
	Note int
}

type Column struct {
	Slot  int
	Onset int
	Len   int
	Cells []Cell
}

func (c Column) Silent() bool {
	for _, cell := range c.Cells {
		if cell.Kind != Empty {
			return false
		}
	}
	return true
}

type Measure struct {
	Index   int
	Columns []Column
}

type System struct {
	Measures []Measure
}

type Sheet struct {
	Systems         []System
	Names           []string
	NameW           int
	CellW           int
	SlotsPerMeasure int
	MeasuresPerLine int
	Rows            int
}

type Glyphs struct {
	Dash, Sustain, Bar, End string
	Ties                    bool
}

var (
	ASCII   = Glyphs{Dash: "-", Sustain: "-", Bar: "|", End: "|"}
	Unicode = Glyphs{Dash: "─", Sustain: "═", Bar: "│", End: "║", Ties: true}
)

func Build(notes []music.Note, frets []Fretting, tuning []int, gridDiv, perLine, minCellW int) Sheet {
	if perLine < 1 {
		perLine = 1
	}
	spm := gridDiv * 4
	total := spm
	for i, n := range notes {
		if f := fretAt(frets, i); f.String >= 0 && n.Slot+1 > total {
			total = n.Slot + 1
		}
	}
	measures := (total + spm - 1) / spm
	cols := make([]Column, measures*spm)
	for k := range cols {
		cols[k] = Column{Slot: k, Onset: -1, Cells: make([]Cell, len(tuning))}
	}
	cellW := max(minCellW, 1)
	for i, n := range notes {
		f := fretAt(frets, i)
		if f.String < 0 || f.String >= len(tuning) {
			continue
		}
		cols[n.Slot].Cells[f.String] = Cell{Kind: Onset, Fret: f.Fret, Note: i}
		cols[n.Slot].Onset, cols[n.Slot].Len = i, n.Len
		cellW = max(cellW, len(strconv.Itoa(f.Fret))+1)
	}
	for i, n := range notes {
		f := fretAt(frets, i)
		if f.String < 0 || f.String >= len(tuning) {
			continue
		}
		for k := n.Slot + 1; k < n.Slot+n.Len && k < len(cols); k++ {
			c := &cols[k].Cells[f.String]
			if c.Kind != Empty {
				break
			}
			*c = Cell{Kind: Sustain, Fret: f.Fret, Note: i}
			if k%spm == 0 {
				c.Kind = Tied
			}
		}
	}
	return group(cols, names(tuning), cellW, spm, perLine, len(tuning))
}

func fretAt(frets []Fretting, i int) Fretting {
	if i < len(frets) {
		return frets[i]
	}
	return Fretting{String: -1, Fret: -1}
}

func names(tuning []int) []string {
	out := make([]string, len(tuning))
	for s, open := range tuning {
		out[s] = strings.TrimRight(music.MidiName(open), "-0123456789")
	}
	return out
}

func group(cols []Column, names []string, cellW, spm, perLine, rows int) Sheet {
	nameW := 0
	for _, n := range names {
		nameW = max(nameW, len(n))
	}
	sh := Sheet{
		Names:           names,
		NameW:           nameW,
		CellW:           cellW,
		SlotsPerMeasure: spm,
		MeasuresPerLine: perLine,
		Rows:            rows,
	}
	measures := len(cols) / spm
	for first := 0; first < measures; first += perLine {
		var sys System
		for mi := first; mi < first+perLine && mi < measures; mi++ {
			sys.Measures = append(sys.Measures, Measure{
				Index:   mi,
				Columns: cols[mi*spm : (mi+1)*spm],
			})
		}
		sh.Systems = append(sh.Systems, sys)
	}
	return sh
}

func (sh Sheet) Measures() int {
	n := 0
	for _, sys := range sh.Systems {
		n += len(sys.Measures)
	}
	return n
}

func (sh Sheet) Layout() Layout {
	return Layout{
		NameW:           sh.NameW,
		CellW:           sh.CellW,
		SlotsPerMeasure: sh.SlotsPerMeasure,
		MeasuresPerLine: sh.MeasuresPerLine,
		Measures:        sh.Measures(),
		Rows:            sh.Rows,
	}
}

func (sh Sheet) Text(g Glyphs) string {
	var b strings.Builder
	for si, sys := range sh.Systems {
		if si > 0 {
			b.WriteString("\n")
		}
		for s := sh.Rows - 1; s >= 0; s-- {
			b.WriteString(sh.Names[s])
			b.WriteString(strings.Repeat(" ", sh.NameW-len(sh.Names[s])))
			b.WriteString(g.Bar)
			for mi, m := range sys.Measures {
				for _, col := range m.Columns {
					b.WriteString(sh.CellText(col, s, g))
				}
				if si == len(sh.Systems)-1 && mi == len(sys.Measures)-1 {
					b.WriteString(g.End)
				} else {
					b.WriteString(g.Bar)
				}
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (sh Sheet) CellText(col Column, s int, g Glyphs) string {
	c := col.Cells[s]
	switch c.Kind {
	case Onset:
		fs := strconv.Itoa(c.Fret)
		fill := g.Dash
		if col.Len > 1 {
			fill = g.Sustain
		}
		return fs + strings.Repeat(fill, sh.CellW-len(fs))
	case Sustain:
		return strings.Repeat(g.Sustain, sh.CellW)
	case Tied:
		fs := strconv.Itoa(c.Fret)
		if g.Ties && sh.CellW >= len(fs)+2 {
			return "(" + fs + ")" + strings.Repeat(g.Sustain, sh.CellW-len(fs)-2)
		}
		return strings.Repeat(g.Sustain, sh.CellW)
	default:
		return strings.Repeat(g.Dash, sh.CellW)
	}
}
