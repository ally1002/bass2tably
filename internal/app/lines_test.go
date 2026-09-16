package app

import (
	"strings"
	"testing"

	"bass2tably/internal/music"
	"bass2tably/internal/tab"
)

func fixture() *Result {
	return &Result{
		GridDiv: 4,
		Tuning:  []int{23, 28, 33, 38, 43},
		Notes: []music.Note{
			{Midi: 33, StartSec: 0.02, Len: 2},
			{Midi: 28, StartSec: 0.60, Len: 4},
			{Midi: 99, StartSec: 1.20, Len: 3},
		},
		Frets: []tab.Fretting{{String: 2, Fret: 0}, {String: 1, Fret: 0}, {String: -1, Fret: -1}},
	}
}

func TestNoteLines(t *testing.T) {
	want := []string{
		"#1    0.02s  A1   string A fret 0   [also: E fret 5, B fret 10]  1/2 beat",
		"#2    0.60s  E1   string E fret 0   [also: B fret 5]             1 beat",
		"#3    1.20s  D#7  unplayable in this tuning                               3/4 beat",
	}
	got := fixture().NoteLines()
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d:\n got %q\nwant %q", i, got[i], want[i])
		}
	}
}

func TestBeatsLabel(t *testing.T) {
	cases := []struct {
		slots, gridDiv int
		want           string
	}{
		{4, 4, "1 beat"},
		{8, 4, "2 beats"},
		{2, 4, "1/2 beat"},
		{3, 4, "3/4 beat"},
		{7, 4, "7/4 beat"},
		{1, 2, "1/2 beat"},
	}
	for _, c := range cases {
		if got := BeatsLabel(c.slots, c.gridDiv); got != c.want {
			t.Errorf("BeatsLabel(%d, %d) = %q, want %q", c.slots, c.gridDiv, got, c.want)
		}
	}
}

func TestNoteAt(t *testing.T) {
	r := fixture()
	cases := []struct {
		pos  float64
		want int
	}{
		{-1, 0},
		{0, 0},
		{0.02, 0},
		{0.59, 0},
		{0.60, 1},
		{1.19, 1},
		{1.20, 2},
		{99, 2},
	}
	for _, c := range cases {
		if got := NoteAt(r, c.pos); got != c.want {
			t.Errorf("NoteAt(%.2f) = %d, want %d", c.pos, got, c.want)
		}
	}
}

func TestNoteLinesEmpty(t *testing.T) {
	r := &Result{GridDiv: 4, Tuning: []int{23, 28, 33, 38, 43}}
	if got := r.NoteLines(); len(got) != 0 {
		t.Fatalf("got %d lines for a result with no notes, want 0", len(got))
	}
}

func TestNoteLinesShortFrets(t *testing.T) {
	r := &Result{
		GridDiv: 4,
		Tuning:  []int{23, 28, 33, 38, 43},
		Notes: []music.Note{
			{Midi: 33, StartSec: 0.02, Len: 2},
			{Midi: 28, StartSec: 0.60, Len: 4},
		},
		Frets: []tab.Fretting{{String: 2, Fret: 0}},
	}
	got := r.NoteLines()
	if len(got) != 2 {
		t.Fatalf("got %d lines, want 2", len(got))
	}
	if !strings.Contains(got[1], "unplayable in this tuning") {
		t.Errorf("line for note with no fret = %q, want it marked unplayable", got[1])
	}
}

func TestStringNameOutOfRange(t *testing.T) {
	r := &Result{Tuning: []int{23, 28, 33, 38, 43}}
	if got := r.StringName(5); got != "?" {
		t.Errorf("StringName(5) = %q, want %q", got, "?")
	}
	if got := r.StringName(-1); got != "?" {
		t.Errorf("StringName(-1) = %q, want %q", got, "?")
	}
}
