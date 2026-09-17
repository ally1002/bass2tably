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
	rangeMarginCents = 100.0
	maxFret          = 24
	preciseCents     = 30.0
	refineTop        = 5
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
	fc := frameCandidates(segFrames(samples, sampleRate, seg, r), sampleRate, r)
	start, length := analysisSpan(samples, sampleRate, seg)
	ps := spectralPartials(samples, sampleRate, start, length)
	ranked := twmRank(ps, noteCandidates(ps, fc, r.MinHz, r.MaxHz), inharmB)
	if len(ranked) == 0 {
		return yinOnly(fc)
	}
	best, b := ranked[0].Hz, inharmB
	if eb, ok := estimateB(ps, best); ok {
		best, b = refineWithB(ps, ranked, eb), eb
	}
	return precise(fc, ps, best, b), true
}

func analysisSpan(samples []float64, sampleRate int, seg Segment) (start, length int) {
	start = seg.Start + int(attackSkipSec*float64(sampleRate))
	end := seg.End
	if end > len(samples) {
		end = len(samples)
	}
	return start, end - start
}

func segFrames(samples []float64, sampleRate int, seg Segment, r PitchRange) [][]float64 {
	start, length := analysisSpan(samples, sampleRate, seg)
	end := start + length
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

func frameCandidates(frames [][]float64, sampleRate int, r PitchRange) [][]yinCandidate {
	out := make([][]yinCandidate, len(frames))
	for i, fr := range frames {
		out[i] = yinCandidates(fr, sampleRate, r.MinHz, r.MaxHz)
	}
	return out
}

func yinOnly(fc [][]yinCandidate) (float64, bool) {
	var freqs []float64
	for _, cs := range fc {
		for _, c := range cs {
			if c.Depth < yinThreshold {
				freqs = append(freqs, c.Hz)
				break
			}
		}
	}
	if len(freqs) == 0 {
		return 0, false
	}
	return median(freqs), true
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
	return yinOnly(frameCandidates(segFrames(samples, sampleRate, seg, r), sampleRate, r))
}

func refineWithB(ps []partial, ranked []scored, b float64) float64 {
	best, bestCost := ranked[0].Hz, math.Inf(1)
	for i, s := range ranked {
		if i >= refineTop {
			break
		}
		if c := twmCost(ps, s.Hz, b); c < bestCost {
			best, bestCost = s.Hz, c
		}
	}
	return best
}

func precise(fc [][]yinCandidate, ps []partial, best, b float64) float64 {
	var voiced, any []float64
	for _, cs := range fc {
		for _, c := range cs {
			if math.Abs(1200*math.Log2(c.Hz/best)) > preciseCents {
				continue
			}
			any = append(any, c.Hz)
			if c.Depth < yinThreshold {
				voiced = append(voiced, c.Hz)
			}
		}
	}
	if len(voiced) > 0 {
		return median(voiced)
	}
	if len(any) > 0 {
		return median(any)
	}
	return partialPitch(ps, best, b)
}

func partialPitch(ps []partial, f0, b float64) float64 {
	var sum, wsum float64
	for h := 1; h <= twmHarmonics; h++ {
		fh := partialFreq(f0, h, b)
		if fh > specMaxHz || len(ps) == 0 {
			break
		}
		p := nearestPartial(ps, fh)
		if math.Abs(1200*math.Log2(p.Hz/fh)) > specTolCents {
			continue
		}
		sum += p.Hz / (fh / f0) * p.Mag
		wsum += p.Mag
	}
	if wsum == 0 {
		return f0
	}
	return sum / wsum
}

func median(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	mid := len(s) / 2
	if len(s)%2 == 0 {
		return (s[mid-1] + s[mid]) / 2
	}
	return s[mid]
}
