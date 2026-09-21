package tab

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"bass2tably/internal/music"
)

const maxFret = 24

type Fretting struct {
	String int
	Fret   int
}

func Positions(midi int, tuning []int) []Fretting {
	var out []Fretting
	for s, open := range tuning {
		if f := midi - open; f >= 0 && f <= maxFret {
			out = append(out, Fretting{String: s, Fret: f})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Fret < out[j].Fret })
	return out
}

func Assign(notes []music.Note, tuning []int, position float64) []Fretting {
	frets := make([]Fretting, len(notes))
	for i := range frets {
		frets[i] = Fretting{String: -1, Fret: -1}
	}
	var idx []int
	var cands [][]Fretting
	for i, n := range notes {
		if ps := Positions(n.Midi, tuning); len(ps) > 0 {
			idx = append(idx, i)
			cands = append(cands, ps)
		}
	}
	if len(idx) == 0 {
		return frets
	}
	choice := dpAssign(cands, position)
	for k, i := range idx {
		frets[i] = cands[k][choice[k]]
	}
	return frets
}

func placeCost(p Fretting, position float64) float64 {
	return 0.4 * math.Abs(float64(p.Fret)-position)
}

type pathCost struct {
	move float64
	hops float64
}

func (a pathCost) less(b pathCost) bool {
	if math.Abs(a.move-b.move) > 1e-9 {
		return a.move < b.move
	}
	return a.hops < b.hops
}

func (a pathCost) plus(from, to Fretting) pathCost {
	return pathCost{
		move: a.move + math.Abs(float64(to.Fret-from.Fret)),
		hops: a.hops + math.Abs(float64(to.String-from.String)),
	}
}

func dpAssign(cands [][]Fretting, position float64) []int {
	n := len(cands)
	cost := make([][]pathCost, n)
	par := make([][]int, n)
	for k := range cands {
		cost[k] = make([]pathCost, len(cands[k]))
		par[k] = make([]int, len(cands[k]))
	}
	for ci, p := range cands[0] {
		cost[0][ci] = pathCost{move: placeCost(p, position)}
	}
	for k := 1; k < n; k++ {
		for ci, p := range cands[k] {
			best := pathCost{move: math.Inf(1)}
			for cj, q := range cands[k-1] {
				if c := cost[k-1][cj].plus(q, p); c.less(best) {
					best = c
					par[k][ci] = cj
				}
			}
			best.move += placeCost(p, position)
			cost[k][ci] = best
		}
	}

	bestCi := 0
	for ci := 1; ci < len(cands[n-1]); ci++ {
		if cost[n-1][ci].less(cost[n-1][bestCi]) {
			bestCi = ci
		}
	}
	choice := make([]int, n)
	choice[n-1] = bestCi
	for k := n - 1; k > 0; k-- {
		choice[k-1] = par[k][choice[k]]
	}
	return choice
}

type Layout struct {
	NameW, CellW    int
	SlotsPerMeasure int
	MeasuresPerLine int
	Measures        int
	Rows            int
}

// Block = index of the 5-row line group; Col = byte column inside each row.
func (l Layout) SlotCell(slot int) (block, col int) {
	m := slot / l.SlotsPerMeasure
	block = m / l.MeasuresPerLine
	mInLine := m % l.MeasuresPerLine
	col = l.NameW + 1 + mInLine*(l.SlotsPerMeasure*l.CellW+1) +
		(slot%l.SlotsPerMeasure)*l.CellW
	return
}

func Render(notes []music.Note, frets []Fretting, tuning []int, gridDiv int) string {
	s, _ := RenderLayout(notes, frets, tuning, gridDiv)
	return s
}

const DefaultMeasuresPerLine = 4

func RenderLayout(notes []music.Note, frets []Fretting, tuning []int, gridDiv int) (string, Layout) {
	return RenderLayoutWidth(notes, frets, tuning, gridDiv, DefaultMeasuresPerLine)
}

func RenderLayoutWidth(notes []music.Note, frets []Fretting, tuning []int, gridDiv, measuresPerLine int) (string, Layout) {
	if measuresPerLine < 1 {
		measuresPerLine = 1
	}
	slotsPerMeasure := gridDiv * 4

	totalSlots := slotsPerMeasure
	for i, n := range notes {
		if i >= len(frets) || frets[i].String < 0 {
			continue
		}
		if n.Slot+1 > totalSlots {
			totalSlots = n.Slot + 1
		}
	}
	measures := (totalSlots + slotsPerMeasure - 1) / slotsPerMeasure

	cells := make([][]int, len(tuning))
	for s := range cells {
		cells[s] = make([]int, measures*slotsPerMeasure)
		for k := range cells[s] {
			cells[s][k] = -1
		}
	}
	cellW := 2
	for i, n := range notes {
		if i >= len(frets) {
			continue
		}
		f := frets[i]
		if f.String < 0 || f.String >= len(tuning) {
			continue
		}
		cells[f.String][n.Slot] = f.Fret
		if f.Fret > 9 {
			cellW = 3
		}
	}

	names := make([]string, len(tuning))
	nameW := 0
	for s, open := range tuning {
		names[s] = strings.TrimRight(music.MidiName(open), "-0123456789")
		if len(names[s]) > nameW {
			nameW = len(names[s])
		}
	}

	var b strings.Builder
	for line := 0; line < measures; line += measuresPerLine {
		lineEnd := line + measuresPerLine
		if lineEnd > measures {
			lineEnd = measures
		}
		for s := len(tuning) - 1; s >= 0; s-- {
			b.WriteString(names[s])
			b.WriteString(strings.Repeat(" ", nameW-len(names[s])))
			b.WriteString("|")
			for m := line; m < lineEnd; m++ {
				for k := m * slotsPerMeasure; k < (m+1)*slotsPerMeasure; k++ {
					if fret := cells[s][k]; fret >= 0 {
						fs := strconv.Itoa(fret)
						b.WriteString(fs)
						b.WriteString(strings.Repeat("-", cellW-len(fs)))
					} else {
						b.WriteString(strings.Repeat("-", cellW))
					}
				}
				b.WriteString("|")
			}
			b.WriteString("\n")
		}
		if lineEnd < measures {
			b.WriteString("\n")
		}
	}
	return b.String(), Layout{
		NameW:           nameW,
		CellW:           cellW,
		SlotsPerMeasure: slotsPerMeasure,
		MeasuresPerLine: measuresPerLine,
		Measures:        measures,
		Rows:            len(tuning),
	}
}
