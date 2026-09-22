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

func TestSlotAtInvertsSlotCell(t *testing.T) {
	riff := riffNotes()
	layoutNotes, layoutFrets := layoutNotesFrets()
	fixtures := []struct {
		name    string
		notes   []music.Note
		frets   []tab.Fretting
		gridDiv int
		perLine int
		cellW   int
	}{
		{"riff", riff, tab.Assign(riff, beadg, 0), 4, 4, 2},
		{"riff wide", riff, tab.Assign(riff, beadg, 0), 4, 2, 3},
		{"fret 12", layoutNotes, layoutFrets, 4, 3, 3},
		{"grid 8", riff, tab.Assign(riff, beadg, 0), 2, 1, 4},
	}
	for _, fx := range fixtures {
		for _, chrome := range [][2]int{{0, 0}, {1, 0}, {1, 3}} {
			l := tab.Build(fx.notes, fx.frets, beadg, fx.gridDiv, fx.perLine, fx.cellW).Layout()
			l.Top, l.Bottom = chrome[0], chrome[1]
			total := l.Measures * l.SlotsPerMeasure
			for slot := 0; slot < total; slot++ {
				block, col := l.SlotCell(slot)
				for s := 0; s < l.Rows; s++ {
					for dx := 0; dx < l.CellW; dx++ {
						got, ok := l.SlotAt(l.StringRow(block, s), col+dx)
						if !ok || got != slot {
							t.Fatalf("%s top %d bottom %d: SlotAt(StringRow(%d, %d), %d) = %d %v, want slot %d", fx.name, l.Top, l.Bottom, block, s, col+dx, got, ok, slot)
						}
					}
				}
				if _, ok := l.SlotAt(l.StringRow(block, 0), l.NameW); ok {
					t.Errorf("%s: the gutter column mapped to a slot", fx.name)
				}
				if l.Top > 0 {
					if _, ok := l.SlotAt(block*l.Stride(), col); ok {
						t.Errorf("%s: the ruler row mapped to a slot", fx.name)
					}
				}
				for b := 0; b < l.Bottom; b++ {
					if _, ok := l.SlotAt(block*l.Stride()+l.Top+l.Rows+b, col); ok {
						t.Errorf("%s: rhythm row %d mapped to a slot", fx.name, b)
					}
				}
				if _, ok := l.SlotAt(block*l.Stride()+l.Stride()-1, col); ok {
					t.Errorf("%s: the blank separator mapped to a slot", fx.name)
				}
			}
			for mi := 0; mi < l.MeasuresPerLine; mi++ {
				bar := l.NameW + 1 + (mi+1)*(l.SlotsPerMeasure*l.CellW+1) - 1
				if _, ok := l.SlotAt(l.StringRow(0, 0), bar); ok {
					t.Errorf("%s: bar column %d mapped to a slot", fx.name, bar)
				}
			}
			lastBlock, lastCol := l.SlotCell(total - 1)
			if _, ok := l.SlotAt(l.StringRow(lastBlock+1, 0), lastCol); ok {
				t.Errorf("%s: a row past the last system mapped to a slot", fx.name)
			}
			if _, ok := l.SlotAt(-1, lastCol); ok {
				t.Errorf("%s: a negative line mapped to a slot", fx.name)
			}
		}
	}
}
