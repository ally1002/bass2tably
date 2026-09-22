package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"bass2tably/internal/music"
	"bass2tably/internal/tab"
)

func riffMeasure(t *testing.T) tab.Measure {
	t.Helper()
	midis := []int{33, 33, 36, 38, 40, 38, 33, 28, 23}
	slots := []int{0, 2, 4, 8, 12, 14, 16, 20, 24}
	lens := []int{2, 2, 4, 4, 2, 2, 4, 4, 4}
	notes := make([]music.Note, len(midis))
	for i := range midis {
		notes[i] = music.Note{Midi: midis[i], Slot: slots[i], Len: lens[i]}
	}
	sh := tab.Build(notes, tab.Assign(notes, fixtureTuning, 0), fixtureTuning, 4, 4, 2)
	return sh.Systems[0].Measures[0]
}

func TestRhythmRowsRiffMeasureOne(t *testing.T) {
	rows := rhythmRows(riffMeasure(t), 4, 2, beamLevels(4), rhythmUnicode)
	want := []string{
		"│   │   │       │       │   │   ",
		"└───┘                   └───┘   ",
		"                                ",
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i := range want {
		if rows[i] != want[i] {
			t.Errorf("row %d:\ngot  %q\nwant %q", i, rows[i], want[i])
		}
	}
}

func TestRhythmRowsRestDot(t *testing.T) {
	notes := []music.Note{{Midi: 33, Slot: 0, Len: 4}}
	frets := []tab.Fretting{{String: 2, Fret: 0}}
	sh := tab.Build(notes, frets, fixtureTuning, 4, 4, 2)
	rows := rhythmRows(sh.Systems[0].Measures[0], 4, 2, nil, rhythmUnicode)
	want := "│       ·                       "
	if len(rows) != 1 || rows[0] != want {
		t.Fatalf("stems:\ngot  %q\nwant %q", rows, want)
	}
}

func TestRhythmRowsSixteenthPairAndQuarter(t *testing.T) {
	notes := []music.Note{
		{Midi: 33, Slot: 0, Len: 1},
		{Midi: 33, Slot: 1, Len: 1},
		{Midi: 33, Slot: 4, Len: 4},
		{Midi: 33, Slot: 8, Len: 1},
	}
	frets := []tab.Fretting{{String: 2, Fret: 0}, {String: 2, Fret: 0}, {String: 2, Fret: 0}, {String: 2, Fret: 0}}
	sh := tab.Build(notes, frets, fixtureTuning, 4, 4, 2)
	m := sh.Systems[0].Measures[0]
	rows := rhythmRows(m, 4, 2, beamLevels(4), rhythmUnicode)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	for i := 1; i <= 2; i++ {
		if !strings.HasPrefix(rows[i], "└─┘ ") {
			t.Errorf("beam row %d = %q, want a sixteenth pair beamed as └─┘", i, rows[i])
		}
		if got := string([]rune(rows[i])[8:10]); got != "  " {
			t.Errorf("beam row %d beams the quarter at slot 4: %q", i, rows[i])
		}
		if got := string([]rune(rows[i])[16:18]); got != "└─" {
			t.Errorf("beam row %d = %q, want a flag └─ on the lone sixteenth at slot 8", i, got)
		}
	}
	for _, row := range rows {
		if w := ansi.StringWidth(row); w != len(m.Columns)*2 {
			t.Errorf("row %q is %d wide, want %d", row, w, len(m.Columns)*2)
		}
	}
	if got := rhythmRows(m, 4, 3, beamLevels(4), rhythmASCII); !strings.HasPrefix(got[1], "\\--/ ") {
		t.Errorf("ASCII beam at cellW 3 = %q, want \\--/", got[1])
	}
}

func TestRhythmRowsJoinInsideBeam(t *testing.T) {
	notes := []music.Note{
		{Midi: 33, Slot: 0, Len: 1},
		{Midi: 33, Slot: 1, Len: 1},
		{Midi: 33, Slot: 2, Len: 1},
		{Midi: 33, Slot: 3, Len: 1},
	}
	frets := make([]tab.Fretting, len(notes))
	for i := range frets {
		frets[i] = tab.Fretting{String: 2, Fret: 0}
	}
	sh := tab.Build(notes, frets, fixtureTuning, 4, 4, 2)
	rows := rhythmRows(sh.Systems[0].Measures[0], 4, 2, beamLevels(4), rhythmUnicode)
	if got := string([]rune(rows[1])[:8]); got != "└─┴─┴─┘ " {
		t.Errorf("beam row = %q, want └─┴─┴─┘", got)
	}
}

func TestBeamLevelsPerGrid(t *testing.T) {
	if got := beamLevels(2); len(got) != 1 || got[0] != 2 {
		t.Errorf("beamLevels(2) = %v, want [2]", got)
	}
	if got := beamLevels(4); len(got) != 2 || got[0] != 4 || got[1] != 2 {
		t.Errorf("beamLevels(4) = %v, want [4 2]", got)
	}
}

func TestRhythmGlyphsWidthOne(t *testing.T) {
	for _, g := range []rhythmGlyphs{rhythmUnicode, rhythmASCII} {
		for _, r := range []rune{g.stem, g.rest, g.open, g.join, g.close, g.line} {
			if w := ansi.StringWidth(string(r)); w != 1 {
				t.Errorf("%q has width %d, want 1", r, w)
			}
		}
	}
}

func TestRhythmLayoutBottom(t *testing.T) {
	m := newModel("x.wav", testOpts)
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	res := testResult(t)
	res.Samples = nil
	m, _ = advance(t, m, analysisDoneMsg{res})
	if got := m.res.Layout.Bottom; got != 1 {
		t.Fatalf("Bottom = %d with only quarter notes, want 1 (stems only)", got)
	}
	m, _ = advance(t, m, keyPress('r'))
	if got := m.res.Layout.Bottom; got != 0 {
		t.Fatalf("Bottom = %d after r, want 0", got)
	}
	if got := m.res.Layout.Stride(); got != 1+m.res.Layout.Rows+1 {
		t.Fatalf("Stride = %d with rhythm off, want %d", got, 1+m.res.Layout.Rows+1)
	}

	notes, frets := cursorFixtureNotesFrets()
	short := newResult(t, notes, frets, 100, 4, 0)
	m = newModel("x.wav", testOpts)
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = advance(t, m, analysisDoneMsg{short})
	if got := m.res.Layout.Bottom; got != 3 {
		t.Fatalf("Bottom = %d with sixteenths on a 1/16 grid, want 3", got)
	}
	if got := m.res.Layout.Stride(); got != 1+m.res.Layout.Rows+3+1 {
		t.Fatalf("Stride = %d, want Top + Rows + Bottom + 1", got)
	}
	lines := strings.Split(ansi.Strip(m.vp.GetContent()), "\n")
	l := m.res.Layout
	stemRow := lines[l.Top+l.Rows]
	if !strings.Contains(stemRow, string(rhythmUnicode.stem)) {
		t.Errorf("row after the strings is not the stem row: %q", stemRow)
	}
	if ansi.StringWidth(stemRow) != ansi.StringWidth(lines[l.Top]) {
		t.Errorf("stem row width %d differs from string row width %d", ansi.StringWidth(stemRow), ansi.StringWidth(lines[l.Top]))
	}
	if lines[l.Stride()-1] != "" {
		t.Errorf("row %d should be the blank separator, got %q", l.Stride()-1, lines[l.Stride()-1])
	}
}

func TestCursorRowWithRhythmOn(t *testing.T) {
	notes, frets := cursorFixtureNotesFrets()
	st := testStyles()
	sh := tab.Build(notes, frets, fixtureTuning, 4, 4, 3)
	rh := rhythmOpts{gridDiv: 4, levels: beamLevels(4), glyphs: rhythmUnicode, on: true}
	l := tuiLayout(sh)
	l.Bottom = rh.rows()
	if l.Bottom != 3 {
		t.Fatalf("Bottom = %d, want 3", l.Bottom)
	}
	for i, n := range notes {
		block, _ := l.SlotCell(n.Slot)
		fretRow := l.StringRow(block, frets[i].String)
		lines := strings.Split(renderSheet(sh, tab.Unicode, st, n.Slot, false, rh), "\n")
		cursorCell := st.cursor.Render(cellAt(sh, n.Slot, frets[i].String))
		for r, line := range lines {
			if strings.Contains(line, cursorCell) != (r == fretRow) {
				t.Errorf("slot %d: cursor style on row %d, want only row %d", n.Slot, r, fretRow)
			}
		}
		stripped := ansi.Strip(lines[block*l.Stride()+l.Top+l.Rows])
		if !strings.Contains(stripped, string(rhythmUnicode.stem)) {
			t.Errorf("slot %d: row under the strings is not the stem row: %q", n.Slot, stripped)
		}
	}
	if got := stripChrome(renderSheet(sh, tab.Unicode, st, -1, false, rh), l); got != sh.Text(tab.Unicode) {
		t.Fatalf("stripping rulers and rhythm rows does not give back the text:\n%s", got)
	}
}
