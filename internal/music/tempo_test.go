package music_test

import (
	"math"
	"testing"

	"github.com/ally1002/bass2tably/internal/dsp"
	"github.com/ally1002/bass2tably/internal/music"
	"github.com/ally1002/bass2tably/internal/synth"
)

func TestEstimateBPMSyntheticRiff(t *testing.T) {
	const sr = 44100
	const gridDiv = 4

	for _, bpm := range []float64{80, 100, 140} {
		samples := synth.Render(synth.TestRiff, bpm, sr)
		segs := dsp.Segments(samples, sr)
		if len(segs) != len(synth.TestRiff) {
			t.Fatalf("bpm %g: got %d segments, want %d", bpm, len(segs), len(synth.TestRiff))
		}

		onsets := make([]float64, len(segs))
		ref := make([]music.Note, len(segs))
		est := make([]music.Note, len(segs))
		for i, seg := range segs {
			onsets[i] = float64(seg.Start) / sr
			n := music.Note{
				StartSec: float64(seg.Start) / sr,
				DurSec:   float64(seg.End-seg.Start) / sr,
			}
			ref[i], est[i] = n, n
		}

		got, ok := music.EstimateBPM(onsets)
		if !ok {
			t.Fatalf("bpm %g: estimation failed", bpm)
		}
		if math.Abs(got-bpm) > 1 {
			t.Errorf("bpm %g: estimated %.2f, want within ±1", bpm, got)
			continue
		}

		music.Quantize(ref, bpm, gridDiv)
		music.Quantize(est, got, gridDiv)
		for i := range ref {
			if est[i].Slot != ref[i].Slot || est[i].Len != ref[i].Len {
				t.Errorf("bpm %g: note %d slot/len %d/%d with estimate %.2f, want %d/%d",
					bpm, i, est[i].Slot, est[i].Len, got, ref[i].Slot, ref[i].Len)
			}
		}
	}
}
