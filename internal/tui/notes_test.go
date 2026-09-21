package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func viewerWithNotes(t *testing.T) (model, *stubPlayer) {
	t.Helper()
	sp := &stubPlayer{}
	m := newModel("x.wav", testOpts)
	m.newPlayer = func(samples []float64, rate int) (player, error) { return sp, nil }
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = advance(t, m, analysisDoneMsg{testResult(t)})
	return m, sp
}

func TestNotePanelTogglesAndShrinksViewport(t *testing.T) {
	m, _ := viewerWithNotes(t)
	defer m.closePlayer()

	if m.notes.open {
		t.Fatal("note panel should start closed")
	}
	closedH := m.vp.Height()
	if got := len(m.notes.rows); got != 4 {
		t.Fatalf("panel has %d lines, want 4", got)
	}
	if ansi.Strip(m.viewerView()) == "" {
		t.Fatal("viewer rendered empty")
	}

	m, _ = advance(t, m, keyPress('n'))
	if !m.notes.open {
		t.Fatal("n did not open the note panel")
	}
	if m.vp.Height() >= closedH {
		t.Errorf("viewport height %d did not shrink below %d when the panel opened", m.vp.Height(), closedH)
	}
	view := ansi.Strip(m.viewerView())
	if !containsAll(view, "notes 1/4", "#1", "#4") {
		t.Errorf("panel view missing expected content:\n%s", view)
	}

	m, _ = advance(t, m, keyPress('n'))
	if m.notes.open {
		t.Fatal("second n did not close the note panel")
	}
	if m.vp.Height() != closedH {
		t.Errorf("viewport height %d not restored to %d after closing", m.vp.Height(), closedH)
	}
}

func TestNoteSelectionSeeksAndMovesCursor(t *testing.T) {
	m, sp := viewerWithNotes(t)
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress('n'))

	m, _ = advance(t, m, keyPress('j'))
	if m.notes.sel != 1 {
		t.Fatalf("selection = %d after j, want 1", m.notes.sel)
	}
	if m.curPos != 0.6 {
		t.Errorf("curPos = %.3f, want the selected note's start 0.600", m.curPos)
	}
	if m.curSlot != 4 {
		t.Errorf("curSlot = %d, want 4 (the selected note's slot)", m.curSlot)
	}
	if got := sp.PosSec(); got != 0.6 {
		t.Errorf("player PosSec = %.3f, want 0.600", got)
	}
	if sp.Playing() {
		t.Error("moving the selection must not start playback")
	}

	m, _ = advance(t, m, keyPress('k'))
	if m.notes.sel != 0 || m.curSlot != 0 {
		t.Errorf("after k: sel = %d, curSlot = %d, want 0 and 0", m.notes.sel, m.curSlot)
	}
	if want := []float64{0.6, 0}; !equalSeeks(sp.seeks, want) {
		t.Errorf("seeks = %v, want %v (each selection seeks to that note's start)", sp.seeks, want)
	}
}

func TestNoteSelectionClampsAtEnds(t *testing.T) {
	m, _ := viewerWithNotes(t)
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress('n'))

	m, _ = advance(t, m, keyPress('k'))
	if m.notes.sel != 0 {
		t.Errorf("selection = %d after k at the top, want 0", m.notes.sel)
	}
	for i := 0; i < 10; i++ {
		m, _ = advance(t, m, keyPress('j'))
	}
	if want := len(m.notes.rows) - 1; m.notes.sel != want {
		t.Errorf("selection = %d after running past the end, want %d", m.notes.sel, want)
	}
	if m.curPos != 1.8 {
		t.Errorf("curPos = %.3f, want the last note's start 1.800", m.curPos)
	}
}

func TestPanelKeysOnlyStealScrollWhileOpen(t *testing.T) {
	m, _ := viewerWithNotes(t)
	defer m.closePlayer()

	m.vp.SetContent("a\nb\nc\nd\ne\nf\ng\nh\ni\nj\nk\nl\nm\nn\no\np\nq\nr\ns\nt")
	m.vp.SetHeight(3)
	m.vp.GotoTop()
	m, _ = advance(t, m, keyPress('j'))
	if m.vp.YOffset() == 0 {
		t.Error("with the panel closed, j must scroll the tab viewport")
	}
	if m.notes.sel != 0 {
		t.Errorf("with the panel closed, j must not move the note selection (sel = %d)", m.notes.sel)
	}

	m, _ = advance(t, m, keyPress('n'))
	m, _ = advance(t, m, keyPress('j'))
	if m.notes.sel != 1 {
		t.Errorf("with the panel open, j must move the selection (sel = %d)", m.notes.sel)
	}
}

func TestPanelFollowsPlaybackPosition(t *testing.T) {
	m, _ := viewerWithNotes(t)
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress('n'))

	m.curPos = 1.25
	m.syncNoteSel()
	if m.notes.sel != 2 {
		t.Errorf("selection = %d at 1.25 s, want 2 (note starting at 1.2 s)", m.notes.sel)
	}
	m.curPos = 0.1
	m.syncNoteSel()
	if m.notes.sel != 0 {
		t.Errorf("selection = %d at 0.1 s, want 0", m.notes.sel)
	}
}

func TestPlayFromSelectionStartsPlayback(t *testing.T) {
	m, sp := viewerWithNotes(t)
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress('n'))

	m, _ = advance(t, m, keyPress('j'))
	m, cmd := advance(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter did not start a tick chain")
	}
	if !sp.Playing() {
		t.Error("enter did not start playback")
	}
	if !m.tickRunning {
		t.Error("tickRunning not set after enter")
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		found := false
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func equalSeeks(got, want []float64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
