package tui

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"bass2tably/internal/tab"
)

func TestDeviceInitFailureKeepsViewerUsable(t *testing.T) {
	res := testResult(t)
	wantErr := errors.New("no audio device")
	m := newModel(res.Path, testOpts)
	m.newPlayer = func(samples []float64, rate int) (player, error) { return nil, wantErr }
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = advance(t, m, analysisDoneMsg{res})

	if m.player != nil {
		t.Fatal("player should be nil after a device init failure")
	}
	if m.audioErr != wantErr {
		t.Fatalf("audioErr = %v, want %v", m.audioErr, wantErr)
	}
	if got, want := stripRulers(m.vp.GetContent(), m.res.Layout), m.sheet.Text(tab.Unicode); got != want {
		t.Fatalf("tab content = %q, want %q", got, want)
	}
	if view := ansi.Strip(m.viewerView()); !containsAll(view, "playback unavailable", wantErr.Error()) {
		t.Errorf("view missing playback-unavailable notice:\n%s", view)
	}

	m, cmd := advance(t, m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if cmd != nil {
		t.Error("space with a failed player produced a command")
	}
	if m.tickRunning {
		t.Error("space with a failed player must not start a tick chain")
	}

	m, _ = advance(t, m, keyPress('n'))
	m, cmd = advance(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter with a failed player produced a command")
	}
	if m.tickRunning {
		t.Error("enter with a failed player must not start a tick chain")
	}
}

func TestTickAutoPausesOnDone(t *testing.T) {
	res := testResult(t)
	sp := &stubPlayer{}
	m := newModel(res.Path, testOpts)
	m.newPlayer = func(samples []float64, rate int) (player, error) { return sp, nil }
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = advance(t, m, analysisDoneMsg{res})
	defer m.closePlayer()

	sp.playing = true
	sp.done = true
	sp.pos = res.DurationSec
	m.tickGen = 3
	m.tickRunning = true

	m, cmd := advance(t, m, tickMsg{gen: 3})
	if cmd != nil {
		t.Fatal("tick after Done produced another tick command")
	}
	if m.tickRunning {
		t.Fatal("tickRunning should reset once playback reaches the end")
	}
	if sp.Playing() {
		t.Fatal("player should be paused once playback reaches the end")
	}
	if sp.pauseCalls != 1 {
		t.Fatalf("Pause called %d times, want 1", sp.pauseCalls)
	}
	if m.curPos != res.DurationSec {
		t.Fatalf("curPos = %.3f, want %.3f", m.curPos, res.DurationSec)
	}
}

func TestTickPauseErrorIsRecorded(t *testing.T) {
	res := testResult(t)
	sp := &stubPlayer{}
	pauseErr := errors.New("device gone")
	m := newModel(res.Path, testOpts)
	m.newPlayer = func(samples []float64, rate int) (player, error) { return sp, nil }
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = advance(t, m, analysisDoneMsg{res})
	defer m.closePlayer()

	sp.playing = true
	sp.done = true
	sp.pauseErr = pauseErr
	m.tickGen = 1
	m.tickRunning = true

	m, _ = advance(t, m, tickMsg{gen: 1})
	if m.audioErr != pauseErr {
		t.Fatalf("audioErr = %v, want %v", m.audioErr, pauseErr)
	}
}

func TestPlaybackPositionDrivesCursorAndNoteSelection(t *testing.T) {
	res := testResult(t)
	sp := &stubPlayer{}
	m := newModel(res.Path, testOpts)
	m.newPlayer = func(samples []float64, rate int) (player, error) { return sp, nil }
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = advance(t, m, analysisDoneMsg{res})
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress('n'))

	sp.playing = true
	sp.pos = res.Notes[2].StartSec
	m.tickGen = 1
	m.tickRunning = true

	m, cmd := advance(t, m, tickMsg{gen: 1})
	if cmd == nil {
		t.Fatal("tick while still playing should reschedule itself")
	}
	if m.curPos != sp.pos {
		t.Fatalf("curPos = %.3f, want %.3f", m.curPos, sp.pos)
	}
	if want := cursorSlot(res, sp.pos); m.curSlot != want {
		t.Fatalf("curSlot = %d, want %d", m.curSlot, want)
	}
	if m.notes.sel != 2 {
		t.Fatalf("note selection = %d, want 2", m.notes.sel)
	}
}

func TestLeavingViewerReleasesDevice(t *testing.T) {
	res := testResult(t)
	sp := &stubPlayer{}
	m := newModel(res.Path, testOpts)
	m.newPlayer = func(samples []float64, rate int) (player, error) { return sp, nil }
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = advance(t, m, analysisDoneMsg{res})

	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if !sp.closed {
		t.Error("esc back to the picker must release the audio device")
	}
	if m.player != nil {
		t.Error("player not cleared after leaving the viewer")
	}
}

func TestQuitReleasesDevice(t *testing.T) {
	res := testResult(t)
	sp := &stubPlayer{}
	m := newModel(res.Path, testOpts)
	m.newPlayer = func(samples []float64, rate int) (player, error) { return sp, nil }
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = advance(t, m, analysisDoneMsg{res})

	if _, cmd := advance(t, m, keyPress('q')); cmd == nil {
		t.Fatal("q did not quit")
	}
	if !sp.closed {
		t.Error("quitting must release the audio device")
	}
}

func TestPlayErrorIsRecorded(t *testing.T) {
	res := testResult(t)
	playErr := errors.New("device busy")
	sp := &stubPlayer{playErr: playErr}
	m := newModel(res.Path, testOpts)
	m.newPlayer = func(samples []float64, rate int) (player, error) { return sp, nil }
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = advance(t, m, analysisDoneMsg{res})
	defer m.closePlayer()

	m, cmd := advance(t, m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if cmd != nil {
		t.Error("a failed Play must not start a tick chain")
	}
	if sp.playCalls != 1 {
		t.Errorf("Play called %d times, want 1", sp.playCalls)
	}
	if m.audioErr != playErr {
		t.Errorf("audioErr = %v, want %v", m.audioErr, playErr)
	}
	if m.tickRunning {
		t.Error("tickRunning set despite a failed Play")
	}
}
