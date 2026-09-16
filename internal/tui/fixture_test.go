package tui

import (
	"testing"

	"bass2tably/internal/app"
	"bass2tably/internal/music"
	"bass2tably/internal/tab"
)

var fixtureTuning = []int{23, 28, 33, 38, 43}

func newResult(t *testing.T, notes []music.Note, frets []tab.Fretting, bpm float64, gridDiv int, durationSec float64) *app.Result {
	t.Helper()
	s, l := tab.RenderLayout(notes, frets, fixtureTuning, gridDiv)
	return &app.Result{
		Path:        "x.wav",
		SampleRate:  44100,
		DurationSec: durationSec,
		Samples:     make([]float64, int(durationSec*44100)),
		BPM:         bpm,
		GridDiv:     gridDiv,
		Tuning:      fixtureTuning,
		Notes:       notes,
		Frets:       frets,
		Tab:         s,
		Layout:      l,
	}
}

func testResult(t *testing.T) *app.Result {
	t.Helper()
	notes := []music.Note{
		{Midi: 33, StartSec: 0.0, Slot: 0, Len: 4},
		{Midi: 38, StartSec: 0.6, Slot: 4, Len: 4},
		{Midi: 40, StartSec: 1.2, Slot: 8, Len: 4},
		{Midi: 43, StartSec: 1.8, Slot: 12, Len: 4},
	}
	frets := []tab.Fretting{
		{String: 2, Fret: 0},
		{String: 3, Fret: 0},
		{String: 3, Fret: 2},
		{String: 4, Fret: 0},
	}
	return newResult(t, notes, frets, 100, 4, 2.4)
}
