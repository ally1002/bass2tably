package tab

import (
	"math"
	"testing"

	"github.com/ally1002/bass2tably/internal/music"
)

func assignGreedy(cands [][]Fretting, position float64) []int {
	choice := make([]int, len(cands))
	prev := Fretting{String: -1, Fret: -1}
	for k, ps := range cands {
		best := pathCost{move: math.Inf(1)}
		for ci, p := range ps {
			c := pathCost{move: placeCost(p, position)}
			if prev.String >= 0 {
				c = pathCost{move: placeCost(p, position)}.plus(prev, p)
			}
			if c.less(best) {
				best = c
				choice[k] = ci
			}
		}
		prev = ps[choice[k]]
	}
	return choice
}

func totalCost(cands [][]Fretting, choice []int, position float64) float64 {
	var total float64
	prev := Fretting{String: -1, Fret: -1}
	for k, ci := range choice {
		p := cands[k][ci]
		total += placeCost(p, position)
		if prev.String >= 0 {
			total += math.Abs(float64(p.Fret - prev.Fret))
		}
		prev = p
	}
	return total
}

func TestAssignBeatsGreedy(t *testing.T) {
	tuning := []int{23, 28, 33, 38, 43}
	midis := []int{40, 52, 40, 52}
	notes := make([]music.Note, len(midis))
	cands := make([][]Fretting, len(midis))
	for i, m := range midis {
		notes[i] = music.Note{Midi: m}
		cands[i] = Positions(m, tuning)
	}

	dp := Assign(notes, tuning, 0)
	want := []Fretting{{2, 7}, {4, 9}, {2, 7}, {4, 9}}
	for i, f := range dp {
		if f != want[i] {
			t.Errorf("note %d: got string %d fret %d, want string %d fret %d",
				i, f.String, f.Fret, want[i].String, want[i].Fret)
		}
	}

	dpChoice := dpAssign(cands, 0)
	gChoice := assignGreedy(cands, 0)
	dpCost := totalCost(cands, dpChoice, 0)
	gCost := totalCost(cands, gChoice, 0)
	if dpCost >= gCost {
		t.Errorf("dp cost %.2f not below greedy cost %.2f", dpCost, gCost)
	}
}
