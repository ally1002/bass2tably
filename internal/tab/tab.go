package tab

import (
	"math"
	"strconv"
	"strings"

	"bass2tably/internal/music"
)

const maxFret = 24

type Fretting struct {
	String int
	Fret   int
}

func Assign(notes []music.Note, tuning []int) []Fretting {
	frets := make([]Fretting, len(notes))
	prevFret := -1
	for i, n := range notes {
		best := Fretting{String: -1, Fret: -1}
		bestCost := math.MaxFloat64
		for s, open := range tuning {
			fret := n.Midi - open
			if fret < 0 || fret > maxFret {
				continue
			}
			cost := 0.4 * float64(fret)
			if prevFret >= 0 {
				cost += math.Abs(float64(fret - prevFret))
			}
			if cost < bestCost {
				bestCost = cost
				best = Fretting{String: s, Fret: fret}
			}
		}
		frets[i] = best
		if best.String >= 0 {
			prevFret = best.Fret
		}
	}
	return frets
}

func Render(notes []music.Note, frets []Fretting, tuning []int, gridDiv int) string {
	slotsPerMeasure := gridDiv * 4
	const measuresPerLine = 4

	totalSlots := slotsPerMeasure
	for i, n := range notes {
		if frets[i].String < 0 {
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
		f := frets[i]
		if f.String < 0 {
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
	return b.String()
}
