package dsp

import (
	"math"
	"testing"
)

func sines(sampleRate, n int, hz, amp []float64) []float64 {
	x := make([]float64, n)
	for i := range x {
		t := float64(i) / float64(sampleRate)
		for k, f := range hz {
			x[i] += amp[k] * math.Sin(2*math.Pi*f*t)
		}
	}
	return x
}

func nearest(ps []partial, hz float64) (partial, bool) {
	best, ok := partial{}, false
	for _, p := range ps {
		if !ok || math.Abs(p.Hz-hz) < math.Abs(best.Hz-hz) {
			best, ok = p, true
		}
	}
	return best, ok
}

func TestSpectralPartialsPureSine(t *testing.T) {
	const sr = 44100
	x := sines(sr, sr, []float64{100}, []float64{1})
	ps := spectralPartials(x, sr, 0, len(x))
	if len(ps) != 1 {
		t.Fatalf("got %d partials, want 1: %v", len(ps), ps)
	}
	if math.Abs(ps[0].Hz-100) > 0.5 {
		t.Errorf("partial at %.3f Hz, want 100 ± 0.5", ps[0].Hz)
	}
}

func TestSpectralPartialsMissingFundamental(t *testing.T) {
	const sr = 44100
	const f0 = 30.87
	hz := []float64{2 * f0, 3 * f0, 4 * f0, 5 * f0}
	x := sines(sr, sr, hz, []float64{1, 0.8, 0.6, 0.4})
	ps := spectralPartials(x, sr, 0, len(x))
	if len(ps) != len(hz) {
		t.Fatalf("got %d partials, want %d: %v", len(ps), len(hz), ps)
	}
	for i, want := range hz {
		if math.Abs(ps[i].Hz-want) > 0.5 {
			t.Errorf("partial %d at %.3f Hz, want %.3f ± 0.5", i, ps[i].Hz, want)
		}
	}
	if p, _ := nearest(ps, f0); math.Abs(1200*math.Log2(p.Hz/f0)) < 600 {
		t.Errorf("partial at %.3f Hz claims the absent fundamental %.3f", p.Hz, f0)
	}
}

func TestSpectralPartialsRelativeGate(t *testing.T) {
	const sr = 44100
	x := sines(sr, sr, []float64{100, 300}, []float64{1, 0.05})
	if ps := spectralPartials(x, sr, 0, len(x)); len(ps) != 1 {
		t.Fatalf("at specPeakRel = %.2f got %d partials, want 1: %v", specPeakRel, len(ps), ps)
	}
	ps := spectralPartialsRel(x, sr, 0, len(x), 0.01)
	if len(ps) != 2 {
		t.Fatalf("at rel = 0.01 got %d partials, want 2: %v", len(ps), ps)
	}
	if math.Abs(ps[1].Hz-300) > 0.5 {
		t.Errorf("weak partial at %.3f Hz, want 300 ± 0.5", ps[1].Hz)
	}
}

func TestSpectralPartialsAboveBand(t *testing.T) {
	const sr = 44100
	x := sines(sr, sr, []float64{3000}, []float64{1})
	if ps := spectralPartials(x, sr, 0, len(x)); len(ps) != 0 {
		t.Errorf("got %d partials for a 3000 Hz sine, want none: %v", len(ps), ps)
	}
}

func TestSpectralPartialsShortInput(t *testing.T) {
	const sr = 44100
	x := sines(sr, sr, []float64{100}, []float64{1})
	if ps := spectralPartials(x, sr, 0, 8000); len(ps) != 1 {
		t.Errorf("short segment: got %d partials, want 1 (%d-point fallback): %v",
			len(ps), specWinShort, ps)
	}
	if ps := spectralPartials(x, sr, 0, 2000); ps != nil {
		t.Errorf("segment shorter than the fallback window: got %v, want nil", ps)
	}
	if ps := spectralPartials(x, sr, len(x)-2000, 8000); ps != nil {
		t.Errorf("window past the end of the signal: got %v, want nil", ps)
	}
}
