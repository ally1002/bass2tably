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
		t.Errorf("short segment: got %d partials, want 1 (%d-point floor): %v",
			len(ps), specWinMin, ps)
	}
	if ps := spectralPartials(x, sr, 0, 2000); ps != nil {
		t.Errorf("segment shorter than the fallback window: got %v, want nil", ps)
	}
	if ps := spectralPartials(x, sr, len(x)-2000, 8000); ps != nil {
		t.Errorf("window past the end of the signal: got %v, want nil", ps)
	}
}

func series(f0 float64, harmonics ...int) []partial {
	ps := make([]partial, 0, len(harmonics))
	for _, h := range harmonics {
		ps = append(ps, partial{Hz: f0 * float64(h), Mag: 1})
	}
	return ps
}

func TestNotePitchMissingFundamental(t *testing.T) {
	const sr = 44100
	const f0 = 30.87
	x := sines(sr, sr/2,
		[]float64{2 * f0, 3 * f0, 4 * f0, 5 * f0, 6 * f0, 7 * f0},
		[]float64{0.5, 0.8, 1.0, 0.6, 0.9, 0.4})
	seg := Segment{Start: 0, End: len(x)}
	_, voiced := yinMedian(x, sr, seg, DefaultRange)
	f, ok := NotePitch(x, sr, seg)
	if !ok {
		t.Fatalf("no pitch (yin voiced: %v)", voiced)
	}
	if cents := 1200 * math.Log2(f/f0); math.Abs(cents) > 40 {
		t.Errorf("NotePitch = %.2f Hz (%.0f cents off %.2f, yin voiced: %v)", f, cents, f0, voiced)
	}
}

func TestNotePitchSilenceStaysUnpitched(t *testing.T) {
	const sr = 44100
	x := make([]float64, 3584)
	if f, ok := NotePitch(x, sr, Segment{Start: 0, End: len(x)}); ok {
		t.Errorf("silence returned %.2f Hz", f)
	}
}

func TestTwmSupportTolerance(t *testing.T) {
	const f0 = 100.0
	off := func(cents float64) []partial {
		return []partial{{Hz: f0 * math.Exp2(cents/1200), Mag: 1}}
	}
	if s := twmSupport(off(35), f0, 0); s != 1 {
		t.Errorf("35 cents off: support %d, want 1", s)
	}
	if s := twmSupport(off(45), f0, 0); s != 0 {
		t.Errorf("45 cents off: support %d, want 0", s)
	}
	if s := twmSupport(series(500, 1, 2), 500, 0); s != 1 {
		t.Errorf("harmonic past specMaxHz: support %d, want 1", s)
	}
}

func decide(ps []partial) (float64, bool) {
	ranked := twmRank(ps, mergeCandidates(yinFmin, yinFmax, partialCandidates(ps, yinFmin, yinFmax)), inharmB)
	if len(ranked) == 0 {
		return 0, false
	}
	return ranked[0].Hz, true
}

func TestDecisionOnMissingFundamentalSignal(t *testing.T) {
	const sr = 44100
	const f0 = 30.87
	x := sines(sr, sr, []float64{2 * f0, 3 * f0, 4 * f0, 5 * f0}, []float64{1, 0.8, 0.6, 0.4})
	f, ok := decide(spectralPartials(x, sr, 0, len(x)))
	if !ok {
		t.Fatal("no decision from a four-partial series")
	}
	if cents := 1200 * math.Log2(f/f0); math.Abs(cents) > 20 {
		t.Errorf("decision = %.3f Hz (%.1f cents off %.2f), want the absent fundamental", f, cents, f0)
	}
}

func TestDecisionAgreesWithOddHarmonicOracle(t *testing.T) {
	cases := []struct {
		name string
		ps   []partial
		top  float64
	}{
		{"full series of 55", series(55, 1, 2, 3, 4, 5, 6, 7, 8), 55},
		{"full series of 110", series(110, 1, 2, 3, 4, 5, 6, 7, 8), 110},
		{"61.74 with no odd content at 30.87", series(61.74, 1, 2, 3, 4, 5, 6), 61.74},
		{"harmonics 2-7 of 30.87", series(30.87, 2, 3, 4, 5, 6, 7), 61.74},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, ok := decide(c.ps)
			if !ok {
				t.Fatal("no decision")
			}
			sub := c.top / 2
			descended := math.Abs(1200*math.Log2(f/sub)) < 50
			if oracle := oddHarmonicsPresent(c.ps, sub, 2); oracle != descended {
				t.Errorf("decision = %.2f Hz (descended: %v) but the odd-harmonic oracle says %v", f, descended, oracle)
			}
		})
	}
}

func TestDecisionRefusesTooFewPartials(t *testing.T) {
	if _, ok := decide(nil); ok {
		t.Error("empty partial list decided a pitch")
	}
	if _, ok := decide(series(100, 1, 2)); ok {
		t.Error("two partials decided a pitch")
	}
}
