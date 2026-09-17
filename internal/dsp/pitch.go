package dsp

import (
	"math"
	"sort"
)

const (
	yinPeriods = 3.0
	yinHopFrac = 0.25
)

func yinWindow(sampleRate int, fmin float64) (win, hop int) {
	win = int(math.Ceil(yinPeriods * float64(sampleRate) / fmin))
	win += win % 2
	hop = int(float64(win) * yinHopFrac)
	return win, hop
}

const (
	yinFmin        = 28.0
	yinFmax        = 450.0
	yinThreshold   = 0.35
	attackSkipSec  = 0.010
	arbitrateCents = 50.0
)

func NotePitch(samples []float64, sampleRate int, seg Segment) (float64, bool) {
	return notePitch(samples, sampleRate, seg, arbitrateCents, specSubRatio)
}

func notePitch(samples []float64, sampleRate int, seg Segment, cents, subRatio float64) (float64, bool) {
	f, yok := yinMedian(samples, sampleRate, seg)
	start := seg.Start + int(attackSkipSec*float64(sampleRate))
	end := seg.End
	if end > len(samples) {
		end = len(samples)
	}
	ps := spectralPartials(samples, sampleRate, start, end-start)
	sf, sok := spectralF0Ratio(ps, yinFmin, yinFmax, subRatio)
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

func segFrames(samples []float64, sampleRate int, seg Segment) [][]float64 {
	start := seg.Start + int(attackSkipSec*float64(sampleRate))
	end := seg.End
	if end > len(samples) {
		end = len(samples)
	}
	win, hop := yinWindow(sampleRate, yinFmin)

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

func yinFrames(samples []float64, sampleRate int, seg Segment) (freqs []float64, frames int) {
	fs := segFrames(samples, sampleRate, seg)
	for _, fr := range fs {
		if f, ok := Yin(fr, sampleRate, yinFmin, yinFmax, yinThreshold); ok {
			freqs = append(freqs, f)
		}
	}
	return freqs, len(fs)
}

func yinMedian(samples []float64, sampleRate int, seg Segment) (float64, bool) {
	freqs, _ := yinFrames(samples, sampleRate, seg)
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
