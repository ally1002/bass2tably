package dsp

import (
	"math"
	"testing"

	"bass2tably/internal/music"
	"bass2tably/internal/wavio"
)

func TestRangeForTuning(t *testing.T) {
	cases := []struct {
		name       string
		open       []int
		minHz, max float64
	}{
		{"BEADG", []int{23, 28, 33, 38, 43}, 29.14, 415.30},
		{"EADG", []int{28, 33, 38, 43}, 38.89, 415.30},
		{"BEAD", []int{23, 28, 33, 38}, 29.14, 311.13},
	}
	for _, c := range cases {
		r := RangeForTuning(c.open)
		if math.Abs(r.MinHz-c.minHz) > 0.01 || math.Abs(r.MaxHz-c.max) > 0.01 {
			t.Errorf("%s: range %.2f-%.2f, want %.2f-%.2f", c.name, r.MinHz, r.MaxHz, c.minHz, c.max)
		}
	}
	if r := RangeForTuning(nil); r != DefaultRange {
		t.Errorf("empty tuning: %v, want DefaultRange", r)
	}
	if got := music.MidiFromFreq(freqFromMidi(23)); got != 23 {
		t.Errorf("freqFromMidi(23) round-trips to %d", got)
	}
}

func TestOpenBIsRefusedOnAFourString(t *testing.T) {
	clip, err := wavio.LoadAny("../../testdata/missing-fundamental.wav")
	if err != nil {
		t.Fatal(err)
	}
	x, sr := clip.Samples, clip.SampleRate
	seg := segmentNear(Segments(x, sr), sr, 0.726)

	f, ok := NotePitchIn(x, sr, seg, RangeForTuning([]int{23, 28, 33, 38, 43}))
	if !ok || music.MidiFromFreq(f) != 23 {
		t.Errorf("BEADG: %.2f Hz, %v; want the open B (midi 23)", f, ok)
	}
	f, ok = NotePitchIn(x, sr, seg, RangeForTuning([]int{28, 33, 38, 43}))
	if ok && f < RangeForTuning([]int{28, 33, 38, 43}).MinHz {
		t.Errorf("EADG: %.2f Hz is below the instrument's lowest string", f)
	}
	if !ok || music.MidiFromFreq(f) != 35 {
		t.Errorf("EADG: %.2f Hz, %v; measured behaviour is the octave B1 (midi 35)", f, ok)
	}
}
