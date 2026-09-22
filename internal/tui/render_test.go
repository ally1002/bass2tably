package tui

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"bass2tably/internal/music"
	"bass2tably/internal/tab"
)

func TestShadingDimsPlayedOnly(t *testing.T) {
	res := testResult(t)
	st := testStyles()
	sh := tab.Build(res.Notes, res.Frets, fixtureTuning, res.GridDiv, 4, 3)
	l := tuiLayout(sh)
	cur := res.Notes[2].Slot
	shadedRows := strings.Split(renderSheet(sh, tab.Unicode, st, cur, true, noRhythm), "\n")
	unshadedRows := strings.Split(renderSheet(sh, tab.Unicode, st, cur, false, noRhythm), "\n")
	for i, n := range res.Notes {
		s := res.Frets[i].String
		block, _ := l.SlotCell(n.Slot)
		shaded, unshaded := shadedRows[l.StringRow(block, s)], unshadedRows[l.StringRow(block, s)]
		c := cellAt(sh, n.Slot, s)
		played, live := st.played.Render(c), st.fret(s).Render(c)
		switch {
		case n.Slot < cur:
			if !strings.Contains(shaded, played) {
				t.Errorf("note %d before the cursor is not shaded as played", i)
			}
			if strings.Contains(unshaded, played) {
				t.Errorf("note %d shaded as played with shading off", i)
			}
		case n.Slot > cur:
			if !strings.Contains(shaded, live) {
				t.Errorf("note %d after the cursor lost its string colour", i)
			}
			if strings.Contains(shaded, played) {
				t.Errorf("note %d after the cursor is shaded as played", i)
			}
		}
	}
}

func TestSustainsUseStringShade(t *testing.T) {
	res := testResult(t)
	st := testStyles()
	sh := tab.Build(res.Notes, res.Frets, fixtureTuning, res.GridDiv, 4, 3)
	out := renderSheet(sh, tab.Unicode, st, -1, false, noRhythm)
	s := res.Frets[0].String
	if !strings.Contains(out, st.sustain(s).Render(strings.Repeat("═", 9))) {
		t.Fatal("a three-slot sustain is not one run in the string's sustain shade")
	}
}

func TestZoomMovesSlotCell(t *testing.T) {
	m := newModel("x.wav", testOpts)
	m.cellW = 2
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 200, Height: 40})
	res := testResult(t)
	res.Samples = nil
	m, _ = advance(t, m, analysisDoneMsg{res})
	_, col2 := m.res.Layout.SlotCell(5)
	off2 := col2 - m.res.Layout.NameW - 1

	m, _ = advance(t, m, keyPress('z'))
	if m.cellW != 3 || m.res.Layout.CellW != 3 {
		t.Fatalf("after z: cellW %d, Layout.CellW %d, want 3 and 3", m.cellW, m.res.Layout.CellW)
	}
	_, col3 := m.res.Layout.SlotCell(5)
	if off3 := col3 - m.res.Layout.NameW - 1; off3 != off2*3/2 {
		t.Fatalf("in-measure offset %d at cellW 3, want %d", off3, off2*3/2)
	}
	m, _ = advance(t, m, keyPress('z'))
	m, _ = advance(t, m, keyPress('z'))
	if m.cellW != 2 {
		t.Fatalf("zoom did not cycle back to 2, got %d", m.cellW)
	}
}

func TestZoomNeverBelowWidestFret(t *testing.T) {
	notes, frets := cursorFixtureNotesFrets()
	m := newModel("x.wav", testOpts)
	m.cellW = 2
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 200, Height: 40})
	res := newResult(t, notes, frets, 100, 4, 0)
	m, _ = advance(t, m, analysisDoneMsg{res})
	if m.res.Layout.CellW != 3 {
		t.Fatalf("CellW = %d with fret 12 at zoom 2, want 3", m.res.Layout.CellW)
	}
}

func TestRulerNumbersMeasures(t *testing.T) {
	notes := []music.Note{{Midi: 33, Slot: 0, Len: 1}, {Midi: 33, Slot: 16 * 5, Len: 1}}
	frets := []tab.Fretting{{String: 2, Fret: 0}, {String: 2, Fret: 0}}
	sh := tab.Build(notes, frets, fixtureTuning, 4, 4, 3)
	l := tuiLayout(sh)
	if l.Measures != 6 || len(sh.Systems) != 2 {
		t.Fatalf("Measures %d, systems %d, want 6 and 2", l.Measures, len(sh.Systems))
	}
	lines := strings.Split(ansi.Strip(renderSheet(sh, tab.Unicode, testStyles(), 0, false, noRhythm)), "\n")
	fieldW := l.SlotsPerMeasure*l.CellW + 1
	check := func(row int, first, count int) {
		t.Helper()
		line := lines[row]
		if w := ansi.StringWidth(line); w != l.NameW+1+count*fieldW {
			t.Errorf("ruler row %d is %d wide, want %d", row, w, l.NameW+1+count*fieldW)
		}
		for i := 0; i < count; i++ {
			at := l.NameW + 1 + i*fieldW
			label := strconv.Itoa(first + i)
			if got := line[at : at+len(label)]; got != label {
				t.Errorf("ruler row %d col %d = %q, want %q", row, at, got, label)
			}
		}
		if strings.TrimSpace(strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return ' '
			}
			return r
		}, line)) != "" {
			t.Errorf("ruler row %d has more than digits and spaces: %q", row, line)
		}
	}
	check(0, 1, 4)
	check(l.Stride(), 5, 2)
	if ansi.StringWidth(lines[0]) != ansi.StringWidth(lines[1]) {
		t.Errorf("ruler width %d differs from string row width %d", ansi.StringWidth(lines[0]), ansi.StringWidth(lines[1]))
	}
}

func TestRulerHighlightsCurrentMeasure(t *testing.T) {
	res := testResult(t)
	st := testStyles()
	sh := tab.Build(res.Notes, res.Frets, fixtureTuning, res.GridDiv, 4, 3)
	fieldW := sh.SlotsPerMeasure*sh.CellW + 1
	out := renderSheet(sh, tab.Unicode, st, 0, false, noRhythm)
	if !strings.Contains(out, st.rulerCur.Render("1"+strings.Repeat(" ", fieldW-1))) {
		t.Fatal("measure 1 is not highlighted while the cursor is inside it")
	}
	if strings.Contains(renderSheet(sh, tab.Unicode, st, -1, false, noRhythm), st.rulerCur.Render("1"+strings.Repeat(" ", fieldW-1))) {
		t.Fatal("a measure is highlighted with no cursor")
	}
}
