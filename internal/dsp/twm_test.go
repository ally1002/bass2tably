package dsp

import (
	"math"
	"testing"

	"bass2tably/internal/wavio"
)

func stretchedList(f0, b float64, first int, amps []float64) []partial {
	var ps []partial
	for k, a := range amps {
		ps = append(ps, partial{Hz: partialFreq(f0, first+k, b), Mag: a})
	}
	return ps
}

func assertCheaper(t *testing.T, ps []partial, cheaper, dearer float64) {
	t.Helper()
	a, b := twmCost(ps, cheaper, inharmB), twmCost(ps, dearer, inharmB)
	if !(a < b) {
		t.Errorf("twmCost(%.2f) = %.3f, want below twmCost(%.2f) = %.3f", cheaper, a, dearer, b)
	}
}

func TestTwmCostMissingFundamental(t *testing.T) {
	ps := stretchedList(30.87, inharmB, 2, []float64{0.5, 0.8, 1.0, 0.6, 0.9, 0.4})
	assertCheaper(t, ps, 30.87, 61.74)
	assertCheaper(t, ps, 30.87, 41.2)
	assertCheaper(t, ps, 30.87, 20.58)
	if s := twmSupport(ps, 30.87, inharmB); s != 6 {
		t.Errorf("support for 30.87 = %d, want 6 matched harmonics", s)
	}
}

func TestTwmCostFullSeriesStaysPut(t *testing.T) {
	ps := series(110, 1, 2, 3, 4)
	assertCheaper(t, ps, 110, 55)
	assertCheaper(t, ps, 110, 220)
}

func TestTwmCostPrefersSupportOverLoudness(t *testing.T) {
	ps := []partial{
		{93.27, 1.00}, {143.81, 0.91}, {187.66, 1.59}, {234.29, 1.28},
		{280.58, 0.60}, {328.97, 1.48}, {376.48, 3.79}, {424.52, 4.91},
		{470.17, 2.33}, {494.30, 0.78},
	}
	assertCheaper(t, ps, 46.6, 61.72)
	assertCheaper(t, ps, 46.6, 93.83)
	ranked := twmRank(ps, mergeCandidates(yinFmin, yinFmax, partialCandidates(ps, yinFmin, yinFmax)), inharmB)
	if len(ranked) == 0 || math.Abs(1200*math.Log2(ranked[0].Hz/46.6)) > 40 {
		t.Errorf("ranked winner %v, want within 40 cents of 46.6", ranked[:min(3, len(ranked))])
	}
}

func TestTwmStretchWidensTheOpenBMargin(t *testing.T) {
	clip, err := wavio.LoadAny("../../testdata/ahhintro.wav")
	if err != nil {
		t.Fatal(err)
	}
	x, sr := clip.Samples, clip.SampleRate
	seg := segmentNear(Segments(x, sr), sr, 12.434)
	start := seg.Start + int(attackSkipSec*float64(sr))
	ps := spectralPartials(x, sr, start, seg.End-start)
	cands := noteCandidates(ps, segFrames(x, sr, seg, DefaultRange), sr, yinFmin, yinFmax)

	var margins [2]float64
	for i, b := range []float64{0, inharmB} {
		ranked := twmRank(ps, cands, b)
		if len(ranked) == 0 || math.Abs(1200*math.Log2(ranked[0].Hz/30.87)) > 50 {
			t.Fatalf("b=%.1e: winner %v, want the open B", b, ranked[:min(3, len(ranked))])
		}
		margins[i] = twmMargin(ranked)
	}
	if !(margins[1] > margins[0]) {
		t.Errorf("margin with stretch %.3f, without %.3f; the inharmonicity term should widen it", margins[1], margins[0])
	}
}

func TestTwmRankRefusesUnsupportedNoise(t *testing.T) {
	ps := []partial{{500, 1}, {503, 1}, {507, 1}}
	if r := twmRank(ps, mergeCandidates(yinFmin, yinFmax, partialCandidates(ps, yinFmin, yinFmax)), inharmB); len(r) != 0 {
		t.Errorf("three peaks at 500-507 Hz ranked %v, want no supported candidate", r)
	}
	if r := twmRank(nil, []float64{55}, inharmB); len(r) != 0 {
		t.Errorf("empty partial list ranked %v", r)
	}
}

func TestEstimateB(t *testing.T) {
	ps := stretchedList(30.87, 1.5e-4, 2, []float64{1, 1, 1, 1, 1, 1})
	b, ok := estimateB(ps, 30.87)
	if !ok || math.Abs(b-1.5e-4) > 1e-6 {
		t.Errorf("estimateB = %.3e, %v; want 1.5e-4", b, ok)
	}
	if _, ok := estimateB(series(100, 1, 2), 100); ok {
		t.Error("two partials gave an estimate")
	}
	if b, ok := estimateB(stretchedList(30.87, 0.05, 2, []float64{1, 1, 1, 1}), 30.87); ok && b > inharmBMax {
		t.Errorf("estimateB = %.3e exceeds the clamp %.3e", b, inharmBMax)
	}
}

func TestMergeCandidates(t *testing.T) {
	got := mergeCandidates(28, 450, []float64{55, 55.2, 110}, []float64{27, 55.1, 500, 220})
	want := []float64{55, 110, 220}
	if len(got) != len(want) {
		t.Fatalf("mergeCandidates = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("mergeCandidates[%d] = %.2f, want %.2f", i, got[i], want[i])
		}
	}
}
