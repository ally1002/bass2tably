package dsp

import (
	"math"
	"sort"
)

type PitchRange struct {
	MinHz, MaxHz float64
}

const (
	yinPeriods = 3.0
	yinHopFrac = 0.25
)

const (
	yinFmin          = 28.0
	yinFmax          = 450.0
	yinThreshold     = 0.35
	attackSkipSec    = 0.010
	arbitrateCents   = 50.0
	rangeMarginCents = 100.0
	maxFret          = 24
)

var DefaultRange = PitchRange{MinHz: yinFmin, MaxHz: yinFmax}

func RangeForTuning(open []int) PitchRange {
	if len(open) == 0 {
		return DefaultRange
	}
	lo, hi := open[0], open[0]
	for _, m := range open {
		lo = min(lo, m)
		hi = max(hi, m)
	}
	return PitchRange{
		MinHz: freqFromMidi(lo) * math.Exp2(-rangeMarginCents/1200),
		MaxHz: freqFromMidi(hi+maxFret) * math.Exp2(rangeMarginCents/1200),
	}
}

func freqFromMidi(midi int) float64 {
	return 440 * math.Exp2(float64(midi-69)/12)
}

func yinWindow(sampleRate int, fmin float64) (win, hop int) {
	win = int(math.Ceil(yinPeriods * float64(sampleRate) / fmin))
	win += win % 2
	hop = int(float64(win) * yinHopFrac)
	return win, hop
}

func NotePitch(samples []float64, sampleRate int, seg Segment) (float64, bool) {
	return NotePitchIn(samples, sampleRate, seg, DefaultRange)
}

func NotePitchIn(samples []float64, sampleRate int, seg Segment, r PitchRange) (float64, bool) {
	return notePitch(samples, sampleRate, seg, r, arbitrateCents, specSubRatio)
}

func notePitch(samples []float64, sampleRate int, seg Segment, r PitchRange, cents, subRatio float64) (float64, bool) {
	f, yok := yinMedian(samples, sampleRate, seg, r)
	start := seg.Start + int(attackSkipSec*float64(sampleRate))
	end := seg.End
	if end > len(samples) {
		end = len(samples)
	}
	ps := spectralPartials(samples, sampleRate, start, end-start)
	sf, sok := spectralF0Ratio(ps, r.MinHz, r.MaxHz, subRatio)
	switch {
	case !yok && !sok:
		return 0, false
	case !yok:
		return sf, true
	case !sok:
		return f, true
	}
	if math.Abs(1200*math.Log2(sf/f)) > cents {
		return sf, true
	}
	return f, true
}

func segFrames(samples []float64, sampleRate int, seg Segment, r PitchRange) [][]float64 {
	start := seg.Start + int(attackSkipSec*float64(sampleRate))
	end := seg.End
	if end > len(samples) {
		end = len(samples)
	}
	win, hop := yinWindow(sampleRate, r.MinHz)

	var frames [][]float64
	for p := start; p+win <= end; p += hop {
		frames = append(frames, samples[p:p+win])
	}
	if len(frames) == 0 && start < len(samples) {
		stop := start + win
		if stop > len(samples) {
			stop = len(samples)
		}
		frames = append(frames, samples[start:stop])
	}
	return frames
}

func yinFrames(samples []float64, sampleRate int, seg Segment, r PitchRange) (freqs []float64, frames int) {
	fs := segFrames(samples, sampleRate, seg, r)
	for _, fr := range fs {
		if f, ok := Yin(fr, sampleRate, r.MinHz, r.MaxHz, yinThreshold); ok {
			freqs = append(freqs, f)
		}
	}
	return freqs, len(fs)
}

func yinMedian(samples []float64, sampleRate int, seg Segment, r PitchRange) (float64, bool) {
	freqs, _ := yinFrames(samples, sampleRate, seg, r)
	if len(freqs) == 0 {
		return 0, false
	}

	sort.Float64s(freqs)
	mid := len(freqs) / 2
	if len(freqs)%2 == 0 {
		return (freqs[mid-1] + freqs[mid]) / 2, true
	}
	return freqs[mid], true
}
