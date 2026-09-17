package dsp

import (
	"math"
	"math/cmplx"
	"sort"
)

const (
	specWin      = 16384
	specWinShort = 4096
	specMaxHz    = 520.0
	specPeakRel  = 0.10
	specMaxPeaks = 12
)

type partial struct {
	Hz  float64
	Mag float64
}

func spectralPartials(x []float64, sampleRate, start, length int) []partial {
	return spectralPartialsRel(x, sampleRate, start, length, specPeakRel)
}

func spectralPartialsRel(x []float64, sampleRate, start, length int, rel float64) []partial {
	if start < 0 || length <= 0 {
		return nil
	}
	n := specWin
	if length < n || start+n > len(x) {
		n = specWinShort
	}
	if length < n || start+n > len(x) {
		return nil
	}

	buf := make([]complex128, n)
	for i := 0; i < n; i++ {
		w := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(n))
		buf[i] = complex(x[start+i]*w, 0)
	}
	fft(buf)

	binHz := float64(sampleRate) / float64(n)
	last := int(specMaxHz/binHz) + 1
	if last > n/2-1 {
		last = n/2 - 1
	}
	if last < 2 {
		return nil
	}
	mag := make([]float64, last+2)
	for b := range mag {
		mag[b] = cmplx.Abs(buf[b])
	}

	var peak float64
	for b := 1; b <= last; b++ {
		if mag[b] > peak {
			peak = mag[b]
		}
	}
	if peak == 0 {
		return nil
	}

	var ps []partial
	for b := 1; b <= last; b++ {
		if mag[b] < rel*peak || mag[b] <= mag[b-1] || mag[b] <= mag[b+1] {
			continue
		}
		hz := refineTau(mag, b) * binHz
		if hz > specMaxHz {
			continue
		}
		ps = append(ps, partial{Hz: hz, Mag: mag[b]})
	}

	if len(ps) > specMaxPeaks {
		sort.Slice(ps, func(i, j int) bool { return ps[i].Mag > ps[j].Mag })
		ps = ps[:specMaxPeaks]
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].Hz < ps[j].Hz })
	return ps
}
