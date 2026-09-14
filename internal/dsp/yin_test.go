package dsp_test

import (
	"math"
	"testing"

	"bass2tably/internal/dsp"
	"bass2tably/internal/music"
	"bass2tably/internal/synth"
)

func TestYinOnPureSines(t *testing.T) {
	const sr = 44100
	for _, want := range []float64{30.9, 41.2, 55, 110, 196} {
		frame := make([]float64, 4096)
		for i := range frame {
			frame[i] = 0.8 * math.Sin(2*math.Pi*want*float64(i)/sr)
		}
		got, ok := dsp.Yin(frame, sr, 28, 450, 0.15)
		if !ok {
			t.Errorf("%.1f Hz: no pitch detected", want)
			continue
		}
		if math.Abs(got-want)/want > 0.03 {
			t.Errorf("%.1f Hz: got %.2f Hz, off by %.1f%%", want, got, 100*math.Abs(got-want)/want)
		}
	}
}

func TestNotePitchOnTestRiff(t *testing.T) {
	const bpm = 100.0
	const sr = 44100

	samples := synth.Render(synth.TestRiff, bpm, sr)
	segs := dsp.Segments(samples, sr)
	if len(segs) != len(synth.TestRiff) {
		t.Fatalf("got %d segments, want %d", len(segs), len(synth.TestRiff))
	}
	for i, seg := range segs {
		freq, ok := dsp.NotePitch(samples, sr, seg)
		if !ok {
			t.Errorf("note %d: no pitch detected", i)
			continue
		}
		got := music.MidiFromFreq(freq)
		if got != synth.TestRiff[i].Midi {
			t.Errorf("note %d: got MIDI %d (%.2f Hz), want %d", i, got, freq, synth.TestRiff[i].Midi)
		}
	}
}
