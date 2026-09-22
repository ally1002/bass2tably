package tab_test

import (
	"strconv"
	"strings"
	"testing"

	"bass2tably/internal/music"
	"bass2tably/internal/tab"
)

func TestSlotCellMatchesRenderLayout(t *testing.T) {
	notes := []music.Note{
		{Midi: 33, Slot: 0, Len: 1},
		{Midi: 45, Slot: 17, Len: 1},
		{Midi: 40, Slot: 47, Len: 1},
		{Midi: 50, Slot: 70, Len: 1},
	}
	frets := []tab.Fretting{
		{String: 2, Fret: 0},
		{String: 2, Fret: 12},
		{String: 3, Fret: 2},
		{String: 3, Fret: 12},
	}

	s, l := tab.RenderLayout(notes, frets, beadg, 4)

	if l.Rows != len(beadg) {
		t.Fatalf("Rows = %d, want %d", l.Rows, len(beadg))
	}
	if l.CellW != 3 {
		t.Fatalf("CellW = %d, want 3", l.CellW)
	}
	if l.Measures != 5 {
		t.Fatalf("Measures = %d, want 5", l.Measures)
	}

	lines := strings.Split(s, "\n")
	for i, n := range notes {
		block, col := l.SlotCell(n.Slot)
		row := l.StringRow(block, frets[i].String)
		fs := strconv.Itoa(frets[i].Fret)
		want := fs + strings.Repeat("-", l.CellW-len(fs))
		got := lines[row][col : col+l.CellW]
		if got != want {
			t.Errorf("slot %d: cell at block %d row %d col %d = %q, want %q",
				n.Slot, block, row, col, got, want)
		}
	}
}
