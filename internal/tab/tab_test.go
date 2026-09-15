package tab_test

import (
	"strings"
	"testing"

	"bass2tably/internal/music"
	"bass2tably/internal/tab"
)

var beadg = []int{23, 28, 33, 38, 43}

func riffNotes() []music.Note {
	midis := []int{33, 33, 36, 38, 40, 38, 33, 28, 23}
	slots := []int{0, 2, 4, 8, 12, 14, 16, 20, 24}
	lens := []int{2, 2, 4, 4, 2, 2, 4, 4, 4}
	notes := make([]music.Note, len(midis))
	for i := range midis {
		notes[i] = music.Note{Midi: midis[i], Slot: slots[i], Len: lens[i]}
	}
	return notes
}

func TestPositions(t *testing.T) {
	got := tab.Positions(33, beadg)
	want := []tab.Fretting{{2, 0}, {1, 5}, {0, 10}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("position %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestAssignPositionBias(t *testing.T) {
	notes := []music.Note{{Midi: 28}, {Midi: 30}, {Midi: 33}, {Midi: 35}}
	frets := tab.Assign(notes, beadg, 5)
	want := []tab.Fretting{{0, 5}, {0, 7}, {1, 5}, {1, 7}}
	for i, f := range frets {
		if f != want[i] {
			t.Errorf("note %d: got string %d fret %d, want string %d fret %d",
				i, f.String, f.Fret, want[i].String, want[i].Fret)
		}
	}
}

func TestAssignTestRiff(t *testing.T) {
	frets := tab.Assign(riffNotes(), beadg, 0)
	want := []tab.Fretting{
		{2, 0}, {2, 0}, {2, 3}, {3, 0}, {3, 2}, {3, 0}, {2, 0}, {1, 0}, {0, 0},
	}
	for i, f := range frets {
		if f != want[i] {
			t.Errorf("note %d: got string %d fret %d, want string %d fret %d",
				i, f.String, f.Fret, want[i].String, want[i].Fret)
		}
	}
}

func TestRenderTestRiff(t *testing.T) {
	notes := riffNotes()
	out := tab.Render(notes, tab.Assign(notes, beadg, 0), beadg, 4)
	want := strings.Join([]string{
		"G|--------------------------------|--------------------------------|",
		"D|----------------0-------2---0---|--------------------------------|",
		"A|0---0---3-----------------------|0-------------------------------|",
		"E|--------------------------------|--------0-----------------------|",
		"B|--------------------------------|----------------0---------------|",
		"",
	}, "\n")
	if out != want {
		t.Errorf("render mismatch:\ngot:\n%s\nwant:\n%s", out, want)
	}
}
