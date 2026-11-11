package tui

import (
	"math"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ally1002/bass2tably/internal/tab"
)

func practiceModel(t *testing.T) (model, *stubPlayer) {
	t.Helper()
	notes, frets := cursorFixtureNotesFrets()
	res := newResult(t, notes, frets, 100, 4, 30)
	sp := &stubPlayer{speed: 1}
	m := newModel("x.wav", testOpts)
	m.newPlayer = func([]float64, int) (player, error) { return sp, nil }
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 120, Height: 30})
	m, _ = advance(t, m, analysisDoneMsg{res})
	return m, sp
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestLoopKeysSetMeasures(t *testing.T) {
	m, sp := practiceModel(t)
	defer m.closePlayer()
	spm := m.res.Layout.SlotsPerMeasure
	m, _ = advance(t, m, keyPress('a'))
	if sp.loopSet {
		t.Fatal("a alone set a loop on the player")
	}
	m, _ = advance(t, m, keyPress('.'))
	m, _ = advance(t, m, keyPress('.'))
	m, _ = advance(t, m, keyPress('b'))
	if !sp.loopSet || !near(sp.loopA, slotTime(m.res, 0)) || !near(sp.loopB, slotTime(m.res, 3*spm)) {
		t.Fatalf("loop = [%.3f, %.3f) set %v, want [%.3f, %.3f)", sp.loopA, sp.loopB, sp.loopSet, slotTime(m.res, 0), slotTime(m.res, 3*spm))
	}
	band := ansi.Strip(m.statusView())
	if !strings.Contains(band, "↻ 1–3") {
		t.Fatalf("band %q lacks the loop label", band)
	}
	la, lb := m.loopFractions()
	if !near(la, slotTime(m.res, 0)/30) || !near(lb, slotTime(m.res, 3*spm)/30) {
		t.Fatalf("loop fractions %.3f %.3f", la, lb)
	}
	m, _ = advance(t, m, keyPress('x'))
	if sp.loopSet || sp.loopClears < 1 || m.loopActive() {
		t.Fatal("x did not clear the loop")
	}
	if strings.Contains(ansi.Strip(m.statusView()), "↻") {
		t.Fatal("band still shows a loop after x")
	}
}

func TestLoopSwapsWhenEndBeforeStart(t *testing.T) {
	m, sp := practiceModel(t)
	defer m.closePlayer()
	spm := m.res.Layout.SlotsPerMeasure
	m, _ = advance(t, m, keyPress('.'))
	m, _ = advance(t, m, keyPress('.'))
	m, _ = advance(t, m, keyPress('a'))
	m, _ = advance(t, m, keyPress(','))
	m, _ = advance(t, m, keyPress(','))
	m, _ = advance(t, m, keyPress('b'))
	if m.loopA != 0 || m.loopB != 3 {
		t.Fatalf("loop measures [%d, %d), want [0, 3) after swapping", m.loopA, m.loopB)
	}
	if !near(sp.loopA, slotTime(m.res, 0)) || !near(sp.loopB, slotTime(m.res, 3*spm)) {
		t.Fatalf("player loop [%.3f, %.3f)", sp.loopA, sp.loopB)
	}
	m, _ = advance(t, m, keyPress('b'))
	if m.loopA != 0 || m.loopB != 1 {
		t.Fatalf("b on measure 1 with the loop at [0, 3): got [%d, %d), want [0, 1)", m.loopA, m.loopB)
	}
}

func TestLoopHighlightsRulerAndWave(t *testing.T) {
	m, _ := practiceModel(t)
	defer m.closePlayer()
	m.loopA, m.loopB = 1, 3
	m.applyLoop()
	sh, st := m.sheet, m.st
	fieldW := sh.SlotsPerMeasure*sh.CellW + 1
	out := m.vp.GetContent()
	loopPaint := st.ruler.Background(st.pal.selBg)
	for _, mi := range []int{2, 3} {
		label := strings.Repeat(" ", 0) + string(rune('0'+mi))
		if !strings.Contains(out, loopPaint.Render(label+strings.Repeat(" ", fieldW-1))) {
			t.Errorf("measure %d is not drawn in the loop shade", mi)
		}
	}
	if strings.Contains(out, loopPaint.Render("4"+strings.Repeat(" ", fieldW-1))) {
		t.Error("measure 4 outside the loop is drawn in the loop shade")
	}
	if !strings.Contains(out, st.rulerCur.Background(st.pal.selBg).Render("1"+strings.Repeat(" ", fieldW-1))) {
		m.loopA = 0
		m.applyLoop()
		if !strings.Contains(m.vp.GetContent(), st.rulerCur.Background(st.pal.selBg).Render("1"+strings.Repeat(" ", fieldW-1))) {
			t.Error("the current measure inside the loop is not drawn in the loop-cursor shade")
		}
	}

	pk := make([]float64, 40)
	for i := range pk {
		pk[i] = 1
	}
	row := waveRow(pk, 0.25, 0.5, 0.75, st, false)
	plain := ansi.Strip(row)
	runes := []rune(plain)
	if len(runes) != 40 {
		t.Fatalf("wave row is %d cells wide, want 40", len(runes))
	}
	for i, r := range runes {
		switch {
		case i == 10:
			if string(r) != playheadBar {
				t.Errorf("cell %d = %q, want the playhead", i, r)
			}
		case i >= 20 && i < 30:
			if r != '▂' {
				t.Errorf("cell %d = %q inside the loop, want the capped ▂", i, r)
			}
		default:
			if r != '█' {
				t.Errorf("cell %d = %q, want the full block", i, r)
			}
		}
	}
	if !strings.Contains(row, st.accent.Render(strings.Repeat("▂", 10))) {
		t.Error("the loop region is not one accent run")
	}
	_ = tab.Unicode
}

func TestMetronomeKey(t *testing.T) {
	m, sp := practiceModel(t)
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress('m'))
	if !sp.clickOn || sp.clickBPM != m.res.BPM || sp.clickPhase != m.res.Notes[0].StartSec || sp.clickEvery != 4 {
		t.Fatalf("click on %v bpm %.1f phase %.3f every %d", sp.clickOn, sp.clickBPM, sp.clickPhase, sp.clickEvery)
	}
	if !strings.Contains(ansi.Strip(m.statusView()), "♪ click") {
		t.Fatal("band does not show the click")
	}
	m, _ = advance(t, m, keyPress('m'))
	if sp.clickOn || strings.Contains(ansi.Strip(m.statusView()), "click") {
		t.Fatal("second m did not turn the click off")
	}
}

func TestSpeedKeys(t *testing.T) {
	m, sp := practiceModel(t)
	defer m.closePlayer()
	band := func() string { return ansi.Strip(m.statusView()) }
	if strings.Contains(band(), "×") {
		t.Fatal("band shows a speed at 1×")
	}
	m, _ = advance(t, m, keyPress('<'))
	if sp.speed != 0.75 || !strings.Contains(band(), "0.75×") {
		t.Fatalf("after <: speed %g, band %q", sp.speed, band())
	}
	m, _ = advance(t, m, keyPress('<'))
	if sp.speed != 0.5 || !strings.Contains(band(), "0.5× (−1 oct)") {
		t.Fatalf("after < <: speed %g, band %q", sp.speed, band())
	}
	m, _ = advance(t, m, keyPress('<'))
	if sp.speed != 0.5 {
		t.Fatalf("speed went below 0.5: %g", sp.speed)
	}
	m, _ = advance(t, m, keyPress('>'))
	m, _ = advance(t, m, keyPress('>'))
	m, _ = advance(t, m, keyPress('>'))
	if sp.speed != 1 || strings.Contains(band(), "×") {
		t.Fatalf("after > > >: speed %g, band %q", sp.speed, band())
	}
	sp.pos = slotTime(m.res, 17)
	m.curPos = sp.PosSec()
	m.refreshCursor()
	if m.curSlot != 17 {
		t.Fatalf("curSlot = %d from PosSec at slot 17; speed must not change source time", m.curSlot)
	}
}

func TestPracticeStateResetsWithNewResult(t *testing.T) {
	m, _ := practiceModel(t)
	m, _ = advance(t, m, keyPress('a'))
	m, _ = advance(t, m, keyPress('b'))
	m, _ = advance(t, m, keyPress('m'))
	m, _ = advance(t, m, keyPress('<'))
	m.closePlayer()
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	m.screen = analyzingScreen
	m.newPlayer = func([]float64, int) (player, error) { return &stubPlayer{}, nil }
	m, _ = advance(t, m, analysisDoneMsg{testResult(t)})
	defer m.closePlayer()
	if m.loopActive() || m.metronome || m.speed != 1 {
		t.Fatalf("loop %v click %v speed %g after a new result", m.loopActive(), m.metronome, m.speed)
	}
}
