package app

import (
	"fmt"
	"strings"

	"github.com/ally1002/bass2tably/internal/music"
	"github.com/ally1002/bass2tably/internal/tab"
)

type NoteRow struct {
	Index    int
	StartSec float64
	Name     string
	String   int
	Fret     int
	Alts     []tab.Fretting
	Beats    string
}

func (r *Result) StringName(s int) string {
	if s < 0 || s >= len(r.Tuning) {
		return "?"
	}
	return strings.TrimRight(music.MidiName(r.Tuning[s]), "-0123456789")
}

func (r *Result) fretAt(i int) tab.Fretting {
	if i < len(r.Frets) {
		return r.Frets[i]
	}
	return tab.Fretting{String: -1, Fret: -1}
}

func (r *Result) NoteRows() []NoteRow {
	rows := make([]NoteRow, len(r.Notes))
	for i, n := range r.Notes {
		f := r.fretAt(i)
		row := NoteRow{
			Index:    i + 1,
			StartSec: n.StartSec,
			Name:     music.MidiName(n.Midi),
			String:   f.String,
			Fret:     f.Fret,
			Beats:    BeatsLabel(n.Len, r.GridDiv),
		}
		for _, p := range tab.Positions(n.Midi, r.Tuning) {
			if p != f {
				row.Alts = append(row.Alts, p)
			}
		}
		rows[i] = row
	}
	return rows
}

func (r *Result) altLabel(row NoteRow) string {
	if len(row.Alts) == 0 {
		return ""
	}
	parts := make([]string, len(row.Alts))
	for i, p := range row.Alts {
		parts[i] = fmt.Sprintf("%s fret %d", r.StringName(p.String), p.Fret)
	}
	return fmt.Sprintf("[also: %s]", strings.Join(parts, ", "))
}

func (r *Result) NoteLines() []string {
	rows := r.NoteRows()
	alts := make([]string, len(rows))
	altW := 0
	for i, row := range rows {
		alts[i] = r.altLabel(row)
		if len(alts[i]) > altW {
			altW = len(alts[i])
		}
	}

	lines := make([]string, len(rows))
	for i, row := range rows {
		pos := "unplayable in this tuning"
		if row.String >= 0 {
			pos = fmt.Sprintf("string %s fret %d", r.StringName(row.String), row.Fret)
		}
		lines[i] = fmt.Sprintf("#%-2d  %5.2fs  %-3s  %-16s  %-*s  %s",
			row.Index, row.StartSec, row.Name, pos, altW, alts[i], row.Beats)
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
