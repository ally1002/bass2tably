package dsp

import (
	"math"
	"math/cmplx"
	"sort"
)

const (
	specWin      = 16384
	specWinMin   = 4096
	specMaxHz    = 520.0
	specPeakRel  = 0.10
	specMaxPeaks = 64
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
	for n > specWinMin && (length < n || start+n > len(x)) {
		n >>= 1
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

const (
	specTolCents     = 40.0
	specSubRatio     = 0.60
	specStepCent     = 25.0
	specMinHarmonics = 3
)

func harmonicScore(ps []partial, f0 float64, harmonics int) float64 {
	s, _, _ := harmonicMatch(ps, f0, harmonics)
	return s
}

func harmonicMatch(ps []partial, f0 float64, harmonics int) (score float64, matched int, refined float64) {
	var wsum float64
	for h := 1; h <= harmonics; h++ {
		target := f0 * float64(h)
		if target > specMaxHz {
			break
		}
		for _, p := range ps {
			if math.Abs(1200*math.Log2(p.Hz/target)) <= specTolCents {
				score += p.Mag
				matched++
				refined += p.Hz / float64(h) * p.Mag
				wsum += p.Mag
				break
			}
		}
	}
	if wsum == 0 {
		return 0, 0, f0
	}
	return score, matched, refined / wsum
}

func oddHarmonicsPresent(ps []partial, sub float64, div int) bool {
	found := 0
	for h := 3; h <= 7; h += 2 {
		if h%div == 0 {
			continue
		}
		if harmonicScore(ps, sub*float64(h), 1) > 0 {
			found++
		}
	}
	return found >= 2
}

func spectralF0(ps []partial, fmin, fmax float64) (float64, bool) {
	if len(ps) < 3 {
		return 0, false
	}
	best, bestScore, bestMatched := 0.0, 0.0, 0
	for cents := 0.0; ; cents += specStepCent {
		f := fmin * math.Exp2(cents/1200)
		if f > fmax {
			break
		}
		s, matched, refined := harmonicMatch(ps, f, 8)
		if s > 0 && s >= bestScore {
			best, bestScore, bestMatched = refined, s, matched
		}
	}
	if bestMatched < specMinHarmonics {
		return 0, false
	}
	for div := 2; div <= 3; div++ {
		sub := best / float64(div)
		if sub < fmin {
			break
		}
		if !oddHarmonicsPresent(ps, sub, div) {
			continue
		}
		if s, _, refined := harmonicMatch(ps, sub, 8); s >= specSubRatio*bestScore {
			best = refined
		}
	}
	return best, true
}
