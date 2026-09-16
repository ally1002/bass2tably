package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"bass2tably/internal/app"
	"bass2tably/internal/music"
	"bass2tably/internal/tab"
)

func renderFixture(t *testing.T) (string, tab.Layout, []music.Note, []tab.Fretting) {
	t.Helper()
	tuning := []int{23, 28, 33, 38, 43}
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
	s, l := tab.RenderLayout(notes, frets, tuning, 4)
	return s, l, notes, frets
}

func TestHighlightColumnStripsBackToPlain(t *testing.T) {
	s, l, notes, _ := renderFixture(t)
	for _, n := range notes {
		block, col := l.SlotCell(n.Slot)
		got := ansi.Strip(highlightColumn(s, l, block, col, cursorStyle))
		if got != s {
			t.Fatalf("slot %d: stripped highlight differs from plain render", n.Slot)
		}
	}
}

func TestHighlightColumnStylesExactCell(t *testing.T) {
	s, l, notes, _ := renderFixture(t)
	plain := strings.Split(s, "\n")
	for _, n := range notes {
		block, col := l.SlotCell(n.Slot)
		out := strings.Split(highlightColumn(s, l, block, col, cursorStyle), "\n")
		if len(out) != len(plain) {
			t.Fatalf("slot %d: line count changed", n.Slot)
		}
		start := block * (l.Rows + 1)
		for r := range plain {
			if r >= start && r < start+l.Rows {
				want := plain[r][:col] + cursorStyle.Render(plain[r][col:col+l.CellW]) + plain[r][col+l.CellW:]
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
	_, l, notes, frets := renderFixture(t)
	res := &app.Result{
		BPM:     100,
		GridDiv: 4,
		Notes:   notes,
		Frets:   frets,
		Layout:  l,
	}
	if got := cursorSlot(res, -10); got != 0 {
		t.Errorf("cursorSlot(-10) = %d, want 0", got)
	}
	maxSlot := l.Measures*l.SlotsPerMeasure - 1
	if got := cursorSlot(res, 9999); got != maxSlot {
		t.Errorf("cursorSlot(9999) = %d, want %d", got, maxSlot)
	}
	slotSec := 60.0 / res.BPM / float64(res.GridDiv)
	if got := cursorSlot(res, notes[0].StartSec+17*slotSec); got != 17 {
		t.Errorf("cursorSlot at slot 17 time = %d, want 17", got)
	}
}

func TestStaleTickDies(t *testing.T) {
	m := newModel("x.wav", testOpts)
	m, _ = advance(t, m, analysisDoneMsg{&app.Result{Tab: "G|---|\n"}})
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
	m, _ = advance(t, m, analysisDoneMsg{&app.Result{Tab: "G|---|\n"}})
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
	s, l, notes, frets := renderFixture(t)
	res := &app.Result{
		Path:        "x.wav",
		SampleRate:  44100,
		DurationSec: 1.0,
		Tuning:      []int{23, 28, 33, 38, 43},
		Samples:     make([]float64, 44100),
		BPM:         100,
		GridDiv:     4,
		Notes:       notes,
		Frets:       frets,
		Tab:         s,
		Layout:      l,
	}
	m := newModel("x.wav", testOpts)
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = advance(t, m, analysisDoneMsg{res})
	if m.player == nil {
		t.Skipf("no audio device available: %v", m.audioErr)
	}
	defer m.closePlayer()

	m, cmd := advance(t, m, keyPress('.'))
	if cmd != nil {
		t.Error("seek produced a command")
	}
	if pos := m.player.PosSec(); pos != 1.0 {
		t.Errorf("PosSec after +1 measure on a 1 s clip = %.4f, want clamped to 1.0", pos)
	}
	if m.player.Playing() {
		t.Error("seek while paused must not start playback")
	}
	if m.curSlot == 0 {
		t.Error("cursor did not move on seek")
	}

	m, _ = advance(t, m, keyPress(','))
	m, _ = advance(t, m, keyPress(','))
	if pos := m.player.PosSec(); pos != 0 {
		t.Errorf("PosSec after seeking past the start = %.4f, want clamped to 0", pos)
	}
	if m.curSlot != 0 {
		t.Errorf("curSlot = %d after seeking to start, want 0", m.curSlot)
	}
	if m.player.Playing() {
		t.Error("seek while paused must not start playback")
	}
}

func TestSpaceWithoutPlayerIsIgnored(t *testing.T) {
	m := newModel("x.wav", testOpts)
	m, _ = advance(t, m, analysisDoneMsg{&app.Result{Tab: "G|---|\n"}})
	m2, cmd := advance(t, m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if cmd != nil {
		t.Fatal("space with no player produced a command")
	}
	if m2.tickRunning {
		t.Fatal("space with no player must not start a tick chain")
	}
}
