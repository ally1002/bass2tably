package dsp

import "math"

const (
	envWin      = 512
	envHop      = 256
	envSmooth   = 2
	fluxStdMult = 2.5
	gateRatio   = 0.05
	minGapSec   = 0.060
	endRatio    = 0.1
)

type Segment struct {
	Start int
	End   int
}

func Segments(samples []float64, sampleRate int) []Segment {
	env := smooth(envelope(samples), envSmooth)
	if len(env) < 3 {
		return nil
	}
	onsets := onsetFrames(env, sampleRate)
	segs := make([]Segment, 0, len(onsets))
	for i, k := range onsets {
		limit := len(env)
		if i+1 < len(onsets) {
			limit = onsets[i+1]
		}
		peak := k
		for j := k; j < limit; j++ {
			if env[j] > env[peak] {
				peak = j
			}
		}
		end := limit
		for j := peak; j < limit; j++ {
			if env[j] < endRatio*env[peak] {
				end = j
				break
			}
		}
		segs = append(segs, Segment{Start: k * envHop, End: end * envHop})
	}
	return segs
}

func envelope(x []float64) []float64 {
	if len(x) < envWin {
		return nil
	}
	n := (len(x)-envWin)/envHop + 1
	env := make([]float64, n)
	for k := 0; k < n; k++ {
		var s float64
		for i := k * envHop; i < k*envHop+envWin; i++ {
			s += x[i] * x[i]
		}
		env[k] = math.Sqrt(s / envWin)
	}
	return env
}

func smooth(x []float64, radius int) []float64 {
	out := make([]float64, len(x))
	for i := range x {
		lo, hi := i-radius, i+radius
		if lo < 0 {
			lo = 0
		}
		if hi > len(x)-1 {
			hi = len(x) - 1
		}
		var s float64
		for j := lo; j <= hi; j++ {
			s += x[j]
		}
		out[i] = s / float64(hi-lo+1)
	}
	return out
}

func onsetFrames(env []float64, sampleRate int) []int {
	var envMax float64
	for _, e := range env {
		if e > envMax {
			envMax = e
		}
	}
	if envMax == 0 {
		return nil
	}
	gate := gateRatio * envMax

	flux := make([]float64, len(env))
	for k := 1; k < len(env); k++ {
		if d := env[k] - env[k-1]; d > 0 {
			flux[k] = d
		}
	}
	mean, std := meanStd(flux)
	thresh := mean + fluxStdMult*std
	minGap := int(math.Ceil(minGapSec * float64(sampleRate) / envHop))

	var onsets []int
	for k := 1; k < len(flux)-1; k++ {
		if flux[k] < flux[k-1] || flux[k] <= flux[k+1] {
			continue
		}
		if flux[k] <= thresh || env[k] <= gate {
			continue
		}
		if len(onsets) > 0 && k-onsets[len(onsets)-1] < minGap {
			if flux[k] > flux[onsets[len(onsets)-1]] {
				onsets[len(onsets)-1] = k
			}
			continue
		}
		onsets = append(onsets, k)
	}
	return onsets
}

func meanStd(x []float64) (float64, float64) {
	if len(x) == 0 {
		return 0, 0
	}
	var sum float64
	for _, v := range x {
		sum += v
	}
	mean := sum / float64(len(x))
	var sq float64
	for _, v := range x {
		d := v - mean
		sq += d * d
	}
	return mean, math.Sqrt(sq / float64(len(x)))
}
