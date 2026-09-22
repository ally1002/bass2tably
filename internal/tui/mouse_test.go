package tui

import (
	"math"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func mouseModel(t *testing.T, w, h int) (model, *stubPlayer) {
	t.Helper()
	res := testResult(t)
	sp := &stubPlayer{}
	m := newModel("x.wav", testOpts)
	m.newPlayer = func([]float64, int) (player, error) { return sp, nil }
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: w, Height: h})
	m, _ = advance(t, m, analysisDoneMsg{res})
	return m, sp
}

func click(x, y int) tea.MouseClickMsg {
	return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft}
}

func TestViewEnablesMouse(t *testing.T) {
	m := newModel("", testOpts)
	if got := m.View().MouseMode; got != tea.MouseModeCellMotion {
		t.Fatalf("MouseMode = %v, want cell motion", got)
	}
}

func TestClickWaveSeeks(t *testing.T) {
	m, sp := mouseModel(t, 80, 24)
	defer m.closePlayer()
	m, _ = advance(t, m, click(40, m.waveRowY()))
	if len(sp.seeks) != 1 || math.Abs(sp.seeks[0]-m.res.DurationSec/2) > 1e-9 {
		t.Fatalf("seeks = %v, want one seek to %.2f", sp.seeks, m.res.DurationSec/2)
	}
	if sp.Playing() {
		t.Fatal("a wave click started playback")
	}
	m, _ = advance(t, m, click(40, m.waveRowY()-1))
	if len(sp.seeks) != 1 {
		t.Fatalf("a click above the wave row seeked: %v", sp.seeks)
	}
}

func TestClickNoteSeeks(t *testing.T) {
	m, sp := mouseModel(t, 120, 30)
	defer m.closePlayer()
	l := m.res.Layout
	for i, n := range m.res.Notes {
		block, col := l.SlotCell(n.Slot)
		y := tabTop + l.StringRow(block, m.res.Frets[i].String) - m.vp.YOffset()
		x := col + l.CellW - 1 - m.vp.XOffset()
		m, _ = advance(t, m, click(x, y))
		want := slotTime(m.res, n.Slot)
		if got := sp.seeks[len(sp.seeks)-1]; math.Abs(got-want) > 1e-9 {
			t.Errorf("note %d: seek to %.3f, want %.3f", i, got, want)
		}
		if m.curSlot != n.Slot {
			t.Errorf("note %d: curSlot %d, want %d", i, m.curSlot, n.Slot)
		}
		if m.notes.sel != i {
			t.Errorf("note %d: note panel selection %d", i, m.notes.sel)
		}
	}
	before := len(sp.seeks)
	m, _ = advance(t, m, click(0, tabTop))
	m, _ = advance(t, m, click(l.NameW+1, 0))
	if len(sp.seeks) != before {
		t.Fatalf("the ruler or the gutter seeked: %v", sp.seeks[before:])
	}
	m, _ = advance(t, m, tea.MouseClickMsg{X: l.NameW + 1, Y: tabTop + 1, Button: tea.MouseRight})
	if len(sp.seeks) != before {
		t.Fatal("a right click seeked")
	}
}

func TestClickNotePanelSelects(t *testing.T) {
	m, sp := mouseModel(t, 100, 30)
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress('n'))
	y0 := tabTop + m.vp.Height() + 1
	m, _ = advance(t, m, click(5, y0+2))
	if m.notes.sel != 2 {
		t.Fatalf("bottom panel: sel = %d after clicking row 2, want 2", m.notes.sel)
	}
	if got := sp.seeks[len(sp.seeks)-1]; got != m.res.Notes[2].StartSec {
		t.Fatalf("bottom panel: seek to %.2f, want the note's start %.2f", got, m.res.Notes[2].StartSec)
	}
	m, _ = advance(t, m, click(5, y0+m.notes.height+3))
	if m.notes.sel != 2 {
		t.Fatal("a click below the panel changed the selection")
	}

	m, _ = advance(t, m, keyPress('N'))
	x0 := m.vp.Width() + 1
	m, _ = advance(t, m, click(x0+3, tabTop+1+1))
	if m.notes.sel != 1 {
		t.Fatalf("right panel: sel = %d after clicking row 1, want 1", m.notes.sel)
	}
	m, _ = advance(t, m, click(x0+3, tabTop))
	if m.notes.sel != 1 {
		t.Fatal("a click on the panel header changed the selection")
	}
}

func TestWheelScrollsViewportAndNotes(t *testing.T) {
	notes, frets := cursorFixtureNotesFrets()
	res := newResult(t, notes, frets, 100, 4, 0)
	m := newModel("x.wav", testOpts)
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 60, Height: 16})
	m, _ = advance(t, m, analysisDoneMsg{res})
	start := m.vp.YOffset()
	if m.vp.TotalLineCount() <= m.vp.Height() {
		t.Fatalf("fixture does not overflow the viewport: %d lines in %d rows", m.vp.TotalLineCount(), m.vp.Height())
	}
	wheel := func(b tea.MouseButton, x, y int) tea.MouseWheelMsg {
		return tea.MouseWheelMsg{X: x, Y: y, Button: b}
	}
	m, _ = advance(t, m, wheel(tea.MouseWheelDown, 10, tabTop+1))
	if m.vp.YOffset() != start+m.vp.MouseWheelDelta {
		t.Fatalf("YOffset = %d after one wheel notch from %d, want the viewport's delta %d", m.vp.YOffset(), start, m.vp.MouseWheelDelta)
	}
	m, _ = advance(t, m, wheel(tea.MouseWheelUp, 10, tabTop+1))
	if m.vp.YOffset() != start {
		t.Fatalf("YOffset = %d after wheeling back, want %d", m.vp.YOffset(), start)
	}

	m, _ = advance(t, m, keyPress('n'))
	start = m.vp.YOffset()
	m.notes.rows = append(m.notes.rows, m.notes.rows...)
	m.notes.rows = append(m.notes.rows, m.notes.rows...)
	y := tabTop + m.vp.Height() + 1
	m, _ = advance(t, m, wheel(tea.MouseWheelDown, 5, y))
	if m.notes.offset != min(wheelStep, len(m.notes.rows)-m.notes.height) {
		t.Fatalf("notes offset = %d after a wheel over the panel", m.notes.offset)
	}
	if m.vp.YOffset() != start {
		t.Fatal("a wheel over the note panel scrolled the tab")
	}
}

func TestFinderClickSelectsThenOpens(t *testing.T) {
	root := setupTree(t)
	m := newModel("", testOpts)
	m.finder.start = root
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 100, Height: 24})
	m, _ = advance(t, m, readTestDir(t, root))
	top := m.finderListTop()
	m, cmd := advance(t, m, click(10, top+3))
	if m.finder.sel != 3 {
		t.Fatalf("sel = %d after clicking row 3, want 3", m.finder.sel)
	}
	if cmd != nil {
		if _, ok := cmd().(fileChosenMsg); ok {
			t.Fatal("the first click opened the file")
		}
	}
	m, cmd = advance(t, m, click(10, top+3))
	if cmd == nil {
		t.Fatal("clicking the selected row did not open it")
	}
	chosen, ok := cmd().(fileChosenMsg)
	if !ok || chosen.path != filepath.Join(root, "take0.wav") {
		t.Fatalf("cmd() = %#v, want take0.wav chosen", cmd())
	}
	m, _ = advance(t, m, click(m.finder.listWidth()+2, top+1))
	if m.finder.sel != 3 {
		t.Fatal("a click on the preview pane changed the selection")
	}
	m, _ = advance(t, m, click(10, top-1))
	if m.finder.sel != 3 {
		t.Fatal("a click on the rule above the list changed the selection")
	}
	m, _ = advance(t, m, tea.MouseWheelMsg{X: 10, Y: top + 1, Button: tea.MouseWheelDown})
	if m.finder.sel != 6 {
		t.Fatalf("sel = %d after a wheel notch, want 6", m.finder.sel)
	}
}
