package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"bass2tably/internal/music"
	"bass2tably/internal/tab"
)

func cursorFixtureNotesFrets() ([]music.Note, []tab.Fretting) {
	notes := []music.Note{
		{Midi: 33, Slot: 0, Len: 1},
		{Midi: 45, Slot: 17, Len: 1},
		{Midi: 50, Slot: 70, Len: 1},
	}
	frets := []tab.Fretting{
		{String: 2, Fret: 0},
		{String: 2, Fret: 12},
		{String: 3, Fret: 12},
	}
	return notes, frets
}

func testStyles() styles { return newStyles(themes[0], true) }

func tuiLayout(sh tab.Sheet) tab.Layout {
	l := sh.Layout()
	l.Top = 1
	return l
}

var noRhythm rhythmOpts

func stripChrome(rendered string, l tab.Layout) string {
	lines := strings.Split(ansi.Strip(rendered), "\n")
	kept := lines[:0]
	for r, line := range lines {
		rr := r % l.Stride()
		if (rr >= l.Top && rr < l.Top+l.Rows) || rr == l.Stride()-1 {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func TestRenderSheetStripsToText(t *testing.T) {
	notes, frets := cursorFixtureNotesFrets()
	st := testStyles()
	for _, cellW := range []int{2, 3, 4} {
		sh := tab.Build(notes, frets, fixtureTuning, 4, 4, cellW)
		l := tuiLayout(sh)
		want := sh.Text(tab.Unicode)
		for _, n := range notes {
			for _, shading := range []bool{false, true} {
				if got := stripChrome(renderSheet(sh, tab.Unicode, st, n.Slot, shading, noRhythm), l); got != want {
					t.Fatalf("cellW %d slot %d shading %v: stripped render differs from Text(Unicode)\ngot:\n%s\nwant:\n%s", cellW, n.Slot, shading, got, want)
				}
			}
		}
		if got := stripChrome(renderSheet(sh, tab.Unicode, st, -1, false, noRhythm), l); got != want {
			t.Fatalf("cellW %d: stripped render without cursor differs from Text(Unicode)", cellW)
		}
		if got := stripChrome(renderSheet(sh, tab.ASCII, st, -1, false, noRhythm), l); got != sh.Text(tab.ASCII) {
			t.Fatalf("cellW %d: stripped ASCII render differs from Text(ASCII)", cellW)
		}
	}
}

func TestRenderSheetCursorOnce(t *testing.T) {
	notes, frets := cursorFixtureNotesFrets()
	st := testStyles()
	sh := tab.Build(notes, frets, fixtureTuning, 4, 4, 3)
	l := tuiLayout(sh)
	out := func(cur int) []string {
		return strings.Split(renderSheet(sh, tab.Unicode, st, cur, false, noRhythm), "\n")
	}
	for i, n := range notes {
		block, _ := l.SlotCell(n.Slot)
		fretRow := l.StringRow(block, frets[i].String)
		lines := out(n.Slot)
		cursorCell := st.cursor.Render(cellAt(sh, n.Slot, frets[i].String))
		hits := 0
		for r, line := range lines {
			if !strings.Contains(line, cursorCell) {
				continue
			}
			hits++
			if r != fretRow {
				t.Errorf("slot %d: cursor style on row %d, want row %d", n.Slot, r, fretRow)
			}
		}
		if hits != 1 {
			t.Errorf("slot %d: fret cell styled as cursor %d times, want 1", n.Slot, hits)
		}
	}
	for _, line := range out(-1) {
		if strings.Contains(line, st.cursor.Render(cellAt(sh, 0, frets[0].String))) {
			t.Fatal("cursor style drawn with no cursor")
		}
	}
}

func cellAt(sh tab.Sheet, slot, s int) string {
	spm := sh.SlotsPerMeasure
	col := sh.Systems[slot/spm/sh.MeasuresPerLine].Measures[slot/spm%sh.MeasuresPerLine].Columns[slot%spm]
	return sh.CellText(col, s, tab.Unicode)
}

func TestFitMeasures(t *testing.T) {
	notes, frets := cursorFixtureNotesFrets()
	sh := tab.Build(notes, frets, fixtureTuning, 4, 4, 3)
	l := sh.Layout()
	mw := l.SlotsPerMeasure*l.CellW + 1
	cases := map[int]int{
		0:                      1,
		l.NameW + 1 + mw:       1,
		l.NameW + 1 + 2*mw:     2,
		l.NameW + 1 + 2*mw + 5: 2,
		l.NameW + 1 + 100*mw:   l.Measures,
	}
	for w, want := range cases {
		if got := fitMeasures(w, l.NameW, l.CellW, l.SlotsPerMeasure, l.Measures); got != want {
			t.Errorf("fitMeasures(%d) = %d, want %d", w, got, want)
		}
	}
}

func TestCursorSlotClamps(t *testing.T) {
	notes, frets := cursorFixtureNotesFrets()
	res := newResult(t, notes, frets, 100, 4, 0)
	if got := cursorSlot(res, -10); got != 0 {
		t.Errorf("cursorSlot(-10) = %d, want 0", got)
	}
	maxSlot := res.Layout.Measures*res.Layout.SlotsPerMeasure - 1
	if got := cursorSlot(res, 9999); got != maxSlot {
		t.Errorf("cursorSlot(9999) = %d, want %d", got, maxSlot)
	}
	slotSec := 60.0 / res.BPM / float64(res.GridDiv)
	if got := cursorSlot(res, res.Notes[0].StartSec+17*slotSec); got != 17 {
		t.Errorf("cursorSlot at slot 17 time = %d, want 17", got)
	}
}

func TestCursorSlotNoNotesDoesNotPanic(t *testing.T) {
	res := newResult(t, nil, nil, 100, 4, 0)
	if got := cursorSlot(res, 5); got != 0 {
		t.Errorf("cursorSlot with no notes = %d, want 0", got)
	}
}

func TestStaleTickDies(t *testing.T) {
	m := newModel("x.wav", testOpts)
	res := testResult(t)
	res.Samples = nil
	m, _ = advance(t, m, analysisDoneMsg{res})
	m.tickGen = 5
	m.tickRunning = true
	m2, cmd := advance(t, m, tickMsg{gen: 4})
	if cmd != nil {
		t.Fatal("stale tick produced a command; the old chain must die")
	}
	if !m2.tickRunning {
		t.Fatal("stale tick must not touch tickRunning")
	}
}

func TestTickWithoutPlayerStops(t *testing.T) {
	m := newModel("x.wav", testOpts)
	res := testResult(t)
	res.Samples = nil
	m, _ = advance(t, m, analysisDoneMsg{res})
	m.tickGen = 5
	m.tickRunning = true
	m2, cmd := advance(t, m, tickMsg{gen: 5})
	if cmd != nil {
		t.Fatal("tick with no player produced a command")
	}
	if m2.tickRunning {
		t.Fatal("tickRunning should reset when playback is impossible")
	}
}

func TestSeekClampsAndPausedSeekMovesCursorOnly(t *testing.T) {
	notes, frets := cursorFixtureNotesFrets()
	res := newResult(t, notes, frets, 100, 4, 1.0)
	sp := &stubPlayer{}
	m := newModel("x.wav", testOpts)
	m.newPlayer = func(samples []float64, rate int) (player, error) { return sp, nil }
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = advance(t, m, analysisDoneMsg{res})
	defer m.closePlayer()

	m, cmd := advance(t, m, keyPress('.'))
	if cmd != nil {
		t.Error("seek produced a command")
	}
	if pos := sp.PosSec(); pos != 1.0 {
		t.Errorf("PosSec after +1 measure on a 1 s clip = %.4f, want clamped to 1.0", pos)
	}
	if sp.Playing() {
		t.Error("seek while paused must not start playback")
	}
	if m.curSlot == 0 {
		t.Error("cursor did not move on seek")
	}

	m, _ = advance(t, m, keyPress(','))
	m, _ = advance(t, m, keyPress(','))
	if pos := sp.PosSec(); pos != 0 {
		t.Errorf("PosSec after seeking past the start = %.4f, want clamped to 0", pos)
	}
	if m.curSlot != 0 {
		t.Errorf("curSlot = %d after seeking to start, want 0", m.curSlot)
	}
	if sp.Playing() {
		t.Error("seek while paused must not start playback")
	}
}

func TestSpaceWithoutPlayerIsIgnored(t *testing.T) {
	m := newModel("x.wav", testOpts)
	res := testResult(t)
	res.Samples = nil
	m, _ = advance(t, m, analysisDoneMsg{res})
	m2, cmd := advance(t, m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if cmd != nil {
		t.Fatal("space with no player produced a command")
	}
	if m2.tickRunning {
		t.Fatal("space with no player must not start a tick chain")
	}
}
