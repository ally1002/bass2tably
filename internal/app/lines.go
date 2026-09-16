package app

import (
	"fmt"
	"strings"

	"bass2tably/internal/music"
	"bass2tably/internal/tab"
)

func (r *Result) StringName(s int) string {
	if s < 0 || s >= len(r.Tuning) {
		return "?"
	}
	return strings.TrimRight(music.MidiName(r.Tuning[s]), "-0123456789")
}

func (r *Result) NoteLines() []string {
	alts := make([]string, len(r.Notes))
	altW := 0
	for i, n := range r.Notes {
		var parts []string
		for _, p := range tab.Positions(n.Midi, r.Tuning) {
			if p == r.Frets[i] {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s fret %d", r.StringName(p.String), p.Fret))
		}
		if len(parts) > 0 {
			alts[i] = fmt.Sprintf("[also: %s]", strings.Join(parts, ", "))
		}
		if len(alts[i]) > altW {
			altW = len(alts[i])
		}
	}

	lines := make([]string, len(r.Notes))
	for i, n := range r.Notes {
		f := r.Frets[i]
		pos := "unplayable in this tuning"
		if f.String >= 0 {
			pos = fmt.Sprintf("string %s fret %d", r.StringName(f.String), f.Fret)
		}
		lines[i] = fmt.Sprintf("#%-2d  %5.2fs  %-3s  %-16s  %-*s  %s",
			i+1, n.StartSec, music.MidiName(n.Midi), pos, altW, alts[i], BeatsLabel(n.Len, r.GridDiv))
	}
	return lines
}

func BeatsLabel(slots, gridDiv int) string {
	g := gcd(slots, gridDiv)
	num, den := slots/g, gridDiv/g
	switch {
	case den == 1 && num == 1:
		return "1 beat"
	case den == 1:
		return fmt.Sprintf("%d beats", num)
	default:
		return fmt.Sprintf("%d/%d beat", num, den)
	}
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func NoteAt(r *Result, posSec float64) int {
	idx := 0
	for i, n := range r.Notes {
		if n.StartSec > posSec+1e-9 {
			break
		}
		idx = i
	}
	return idx
}
