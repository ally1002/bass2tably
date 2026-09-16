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

func TestHighlightColumnStripsBackToPlain(t *testing.T) {
	notes, frets := cursorFixtureNotesFrets()
	res := newResult(t, notes, frets, 100, 4, 0)
	for _, n := range res.Notes {
		block, col := res.Layout.SlotCell(n.Slot)
		got := ansi.Strip(highlightColumn(res.Tab, res.Layout, block, col, cursorStyle))
		if got != res.Tab {
			t.Fatalf("slot %d: stripped highlight differs from plain render", n.Slot)
		}
	}
}

func TestHighlightColumnStylesExactCell(t *testing.T) {
	notes, frets := cursorFixtureNotesFrets()
	res := newResult(t, notes, frets, 100, 4, 0)
	plain := strings.Split(res.Tab, "\n")
	for _, n := range res.Notes {
		block, col := res.Layout.SlotCell(n.Slot)
		out := strings.Split(highlightColumn(res.Tab, res.Layout, block, col, cursorStyle), "\n")
		if len(out) != len(plain) {
			t.Fatalf("slot %d: line count changed", n.Slot)
		}
		start := block * (res.Layout.Rows + 1)
		for r := range plain {
			if r >= start && r < start+res.Layout.Rows {
				want := plain[r][:col] + cursorStyle.Render(plain[r][col:col+res.Layout.CellW]) + plain[r][col+res.Layout.CellW:]
				if out[r] != want {
					t.Errorf("slot %d row %d: styled line mismatch", n.Slot, r)
				}
			} else if out[r] != plain[r] {
				t.Errorf("slot %d row %d: line outside block was modified", n.Slot, r)
			}
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
