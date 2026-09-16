package dsp

import (
	"math"
	"math/cmplx"
	"slices"
)

const (
	envWin      = 512
	envHop      = 256
	envSmooth   = 2
	fluxStdMult = 2.2
	fluxWin     = 1024
	fluxHop     = 256
	fluxMult    = 2.0
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
	rms := framesToSamples(pickOnsets(flux(env), sampleRate, envHop, fluxStdMult, env, gateRatio), envHop, 0)
	hfc := framesToSamples(pickOnsets(hfcFlux(samples), sampleRate, fluxHop, fluxMult, nil, 0), fluxHop, fluxWin/2)
	minGap := int(math.Ceil(minGapSec * float64(sampleRate)))

	var onsets []int
	for _, s := range mergeOnsets(rms, hfc, minGap) {
		if k := s / envHop; k < len(env) {
			onsets = append(onsets, k)
		}
	}
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

func flux(x []float64) []float64 {
	out := make([]float64, len(x))
	for k := 1; k < len(x); k++ {
		if d := x[k] - x[k-1]; d > 0 {
			out[k] = d
		}
	}
	return out
}

func hfcFlux(x []float64) []float64 {
	if len(x) < fluxWin {
		return nil
	}
	win := make([]float64, fluxWin)
	for i := range win {
		win[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(fluxWin))
	}

	n := (len(x)-fluxWin)/fluxHop + 1
	out := make([]float64, n)
	buf := make([]complex128, fluxWin)
	prev := make([]float64, fluxWin/2+1)
	cur := make([]float64, fluxWin/2+1)
	for k := 0; k < n; k++ {
		base := k * fluxHop
		for i := 0; i < fluxWin; i++ {
			buf[i] = complex(x[base+i]*win[i], 0)
		}
		fft(buf)
		for b := range cur {
			cur[b] = math.Log(1 + 1000*cmplx.Abs(buf[b]))
		}
		if k > 0 {
			var s float64
			for b := 1; b < len(cur); b++ {
				if d := cur[b] - prev[b]; d > 0 {
					s += d * float64(b)
				}
			}
			out[k] = s
		}
		prev, cur = cur, prev
	}
	return out
}

func framesToSamples(frames []int, hop, offset int) []int {
	out := make([]int, len(frames))
	for i, k := range frames {
		out[i] = k*hop + offset
	}
	return out
}

func mergeOnsets(primary, extra []int, minGap int) []int {
	out := append([]int(nil), primary...)
	slices.Sort(out)
	for _, s := range extra {
		i, _ := slices.BinarySearch(out, s)
		if i > 0 && s-out[i-1] < minGap {
			continue
		}
		if i < len(out) && out[i]-s < minGap {
			continue
		}
		out = slices.Insert(out, i, s)
	}
	return out
}

func pickOnsets(nov []float64, sampleRate, hop int, mult float64, gate []float64, gateRatio float64) []int {
	if len(nov) < 3 {
		return nil
	}

	var level float64
	if gate != nil {
		var max float64
		for _, g := range gate {
			if g > max {
				max = g
			}
		}
		if max == 0 {
			return nil
		}
		level = gateRatio * max
	}

	mean, std := meanStd(nov)
	thresh := mean + mult*std
	minGap := int(math.Ceil(minGapSec * float64(sampleRate) / float64(hop)))

	var onsets []int
	for k := 1; k < len(nov)-1; k++ {
		if nov[k] < nov[k-1] || nov[k] <= nov[k+1] {
			continue
		}
		if nov[k] <= thresh {
			continue
		}
		if gate != nil && (k >= len(gate) || gate[k] <= level) {
			continue
		}
		if len(onsets) > 0 && k-onsets[len(onsets)-1] < minGap {
			if nov[k] > nov[onsets[len(onsets)-1]] {
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
