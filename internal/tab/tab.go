package tab

import (
	"math"
	"sort"

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
	Top, Bottom     int
}

func (l Layout) Stride() int { return l.Top + l.Rows + l.Bottom + 1 }

func (l Layout) StringRow(block, s int) int {
	return block*l.Stride() + l.Top + l.Rows - 1 - s
}

// Block = index of the line group; Col = byte column inside each row.
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
	sh := Build(notes, frets, tuning, gridDiv, measuresPerLine, 2)
	return sh.Text(ASCII), sh.Layout()
}
