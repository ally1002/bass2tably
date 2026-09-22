package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func scrollModel(t *testing.T) (model, *stubPlayer) {
	t.Helper()
	notes, frets := cursorFixtureNotesFrets()
	res := newResult(t, notes, frets, 100, 4, 30)
	sp := &stubPlayer{}
	m := newModel("x.wav", testOpts)
	m.newPlayer = func([]float64, int) (player, error) { return sp, nil }
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = advance(t, m, analysisDoneMsg{res})
	return m, sp
}

func TestScrollModeOneSystem(t *testing.T) {
	m, _ := scrollModel(t)
	defer m.closePlayer()
	if len(m.sheet.Systems) < 2 {
		t.Fatalf("page mode at 80 columns should wrap 5 measures, got %d systems", len(m.sheet.Systems))
	}
	m, _ = advance(t, m, keyPress('v'))
	if m.mode != scrollMode {
		t.Fatal("v did not switch to scroll mode")
	}
	if len(m.sheet.Systems) != 1 || m.res.Layout.MeasuresPerLine != m.res.Layout.Measures {
		t.Fatalf("scroll mode: %d systems, %d per line, want one system of %d", len(m.sheet.Systems), m.res.Layout.MeasuresPerLine, m.res.Layout.Measures)
	}
	if m.vp.YOffset() != 0 {
		t.Fatalf("YOffset = %d in scroll mode, want 0", m.vp.YOffset())
	}
	header := ansi.Strip(m.headerView())
	if !strings.Contains(header, "scroll") || strings.Contains(header, "page") {
		t.Fatalf("header does not say scroll: %q", header)
	}
	band := ansi.Strip(m.statusView())
	if !strings.Contains(band, "scroll") || strings.Contains(band, "/line") {
		t.Fatalf("status band does not say scroll: %q", band)
	}
	m, _ = advance(t, m, keyPress('v'))
	if m.mode != pageMode || len(m.sheet.Systems) < 2 {
		t.Fatalf("v did not return to the wrapped page layout: mode %v, %d systems", m.mode, len(m.sheet.Systems))
	}
	if !strings.Contains(ansi.Strip(m.headerView()), "page") {
		t.Fatal("header does not say page after toggling back")
	}
}

func TestScrollModePinsThird(t *testing.T) {
	m, sp := scrollModel(t)
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress('v'))
	mid := m.res.Notes[1]
	sp.pos = slotTime(m.res, mid.Slot)
	sp.playing = true
	m.tickRunning = true
	m, _ = advance(t, m, tickMsg{gen: m.tickGen})
	_, col := m.res.Layout.SlotCell(mid.Slot)
	if want := col - m.vp.Width()/3; m.vp.XOffset() != want {
		t.Fatalf("XOffset = %d with the cursor at column %d, want %d (a third in)", m.vp.XOffset(), col, want)
	}
	if m.vp.YOffset() != 0 {
		t.Fatalf("YOffset = %d, want 0", m.vp.YOffset())
	}
	sp.pos = 0
	m, _ = advance(t, m, tickMsg{gen: m.tickGen})
	if m.vp.XOffset() != 0 {
		t.Fatalf("XOffset = %d with the cursor at the start, want 0", m.vp.XOffset())
	}
	last := m.res.Notes[len(m.res.Notes)-1]
	sp.pos = slotTime(m.res, last.Slot)
	m, _ = advance(t, m, tickMsg{gen: m.tickGen})
	_, lastCol := m.res.Layout.SlotCell(last.Slot)
	if m.vp.XOffset() > lastCol-m.vp.Width()/3 || lastCol-m.vp.XOffset() >= m.vp.Width() {
		t.Fatalf("XOffset = %d at the last note (column %d): the content end may clamp the pin, but the cursor must stay on screen", m.vp.XOffset(), lastCol)
	}
}

func TestScrollModeIgnoresPerLineKeys(t *testing.T) {
	m, _ := scrollModel(t)
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress('v'))
	want := m.res.Layout.MeasuresPerLine
	for _, k := range []rune{'[', ']', '0'} {
		m, _ = advance(t, m, keyPress(k))
		if m.res.Layout.MeasuresPerLine != want || len(m.sheet.Systems) != 1 {
			t.Fatalf("%q changed the layout in scroll mode: %d per line, %d systems", k, m.res.Layout.MeasuresPerLine, len(m.sheet.Systems))
		}
	}
	m, _ = advance(t, m, keyPress('v'))
	m, _ = advance(t, m, keyPress('['))
	if m.res.Layout.MeasuresPerLine >= want {
		t.Fatal("[ did not work again after returning to page mode")
	}
}

func TestPageModeFollowUnchanged(t *testing.T) {
	m, sp := scrollModel(t)
	defer m.closePlayer()
	if m.mode != pageMode {
		t.Fatal("the default mode is not page")
	}
	last := m.res.Notes[len(m.res.Notes)-1]
	sp.pos = slotTime(m.res, last.Slot)
	sp.playing = true
	m.tickRunning = true
	m, _ = advance(t, m, tickMsg{gen: m.tickGen})
	block, _ := m.res.Layout.SlotCell(last.Slot)
	top := block * m.res.Layout.Stride()
	if m.vp.YOffset() > top || m.vp.YOffset()+m.vp.Height() <= top+m.res.Layout.Top {
		t.Fatalf("page mode did not scroll the cursor's system into view: YOffset %d, system top %d", m.vp.YOffset(), top)
	}
	if m.vp.XOffset() != 0 {
		t.Fatalf("page mode moved horizontally to %d", m.vp.XOffset())
	}
}
