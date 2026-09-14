package dsp_test

import (
	"math"
	"testing"

	"bass2tably/internal/dsp"
	"bass2tably/internal/synth"
)

func TestSegmentsFindTestRiffOnsets(t *testing.T) {
	const bpm = 100.0
	const sr = 44100

	samples := synth.Render(synth.TestRiff, bpm, sr)
	segs := dsp.Segments(samples, sr)

	if len(segs) != len(synth.TestRiff) {
		t.Fatalf("got %d segments, want %d: %+v", len(segs), len(synth.TestRiff), segs)
	}

	var beat float64
	for i, e := range synth.TestRiff {
		want := beat * 60 / bpm
		got := float64(segs[i].Start) / sr
		if math.Abs(got-want) > 0.020 {
			t.Errorf("segment %d starts at %.4fs, want %.4fs ±20ms", i, got, want)
		}
		beat += e.Beats
	}

	for i, s := range segs {
		if s.End <= s.Start {
			t.Errorf("segment %d: End %d <= Start %d", i, s.End, s.Start)
		}
		if i+1 < len(segs) && s.End > segs[i+1].Start {
			t.Errorf("segment %d: End %d overlaps next Start %d", i, s.End, segs[i+1].Start)
		}
	}
}
