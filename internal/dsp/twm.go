package dsp

import (
	"math"
	"sort"
)

const (
	twmP          = 0.5
	twmQ          = 1.4
	twmR          = 0.5
	twmRho        = 1.625
	twmHarmonics  = 10
	twmMinSupport = 3
	inharmB       = 1.5e-4
	inharmBMax    = 0.01
	mergeCents    = 10.0
)

func partialFreq(f0 float64, h int, b float64) float64 {
	k := float64(h)
	return k * f0 * math.Sqrt(1+b*k*k)
}

func nearestPartial(ps []partial, hz float64) partial {
	i := sort.Search(len(ps), func(i int) bool { return ps[i].Hz >= hz })
	if i == 0 {
		return ps[0]
	}
	if i == len(ps) {
		return ps[len(ps)-1]
	}
	if hz-ps[i-1].Hz <= ps[i].Hz-hz {
		return ps[i-1]
	}
	return ps[i]
}

func twmCost(ps []partial, f0, b float64) float64 {
	if len(ps) == 0 || f0 <= 0 {
		return math.Inf(1)
	}
	var amax float64
	for _, p := range ps {
		amax = math.Max(amax, p.Mag)
	}
	var pm float64
	npm := 0
	for h := 1; h <= twmHarmonics; h++ {
		fh := partialFreq(f0, h, b)
		if fh > specMaxHz {
			break
		}
		p := nearestPartial(ps, fh)
		df := math.Abs(p.Hz-fh) * math.Pow(fh, -twmP)
		a := math.Sqrt(p.Mag / amax)
		pm += df + a*(twmQ*df-twmR)
		npm++
	}
	if npm == 0 {
		return math.Inf(1)
	}
	var mp float64
	for _, p := range ps {
		h := int(math.Max(1, math.Round(p.Hz/f0)))
		fh := partialFreq(f0, h, b)
		df := math.Abs(p.Hz-fh) * math.Pow(p.Hz, -twmP)
		a := math.Sqrt(p.Mag / amax)
		mp += df + a*(twmQ*df-twmR)
	}
	return pm/float64(npm) + twmRho*mp/float64(len(ps))
}

func twmSupport(ps []partial, f0, b float64) int {
	if len(ps) == 0 || f0 <= 0 {
		return 0
	}
	n := 0
	for h := 1; h <= twmHarmonics; h++ {
		fh := partialFreq(f0, h, b)
		if fh > specMaxHz {
			break
		}
		p := nearestPartial(ps, fh)
		if math.Abs(1200*math.Log2(p.Hz/fh)) <= specTolCents {
			n++
		}
	}
	return n
}

type scored struct {
	Hz      float64
	Cost    float64
	Support int
}

func twmRank(ps []partial, cands []float64, b float64) []scored {
	var out []scored
	for _, c := range cands {
		s := twmSupport(ps, c, b)
		if s < twmMinSupport {
			continue
		}
		out = append(out, scored{Hz: c, Cost: twmCost(ps, c, b), Support: s})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Cost != out[j].Cost {
			return out[i].Cost < out[j].Cost
		}
		return out[i].Hz > out[j].Hz
	})
	return out
}

func estimateB(ps []partial, f0 float64) (float64, bool) {
	var sxy, sxx float64
	n := 0
	for _, p := range ps {
		k := math.Round(p.Hz / f0)
		if k < 2 || k > float64(twmHarmonics) {
			continue
		}
		if math.Abs(1200*math.Log2(p.Hz/(k*f0))) > specTolCents {
			continue
		}
		y := (p.Hz/f0)*(p.Hz/f0) - k*k
		x := k * k * k * k
		sxy += x * y
		sxx += x * x
		n++
	}
	if n < 3 || sxx == 0 {
		return 0, false
	}
	return math.Max(0, math.Min(sxy/sxx, inharmBMax)), true
}

func partialCandidates(ps []partial, fmin, fmax float64) []float64 {
	var out []float64
	for _, p := range ps {
		for h := 1; h <= twmHarmonics; h++ {
			f := p.Hz / float64(h)
			if f < fmin {
				break
			}
			if f <= fmax {
				out = append(out, f)
			}
		}
	}
	return out
}

func mergeCandidates(fmin, fmax float64, lists ...[]float64) []float64 {
	var all []float64
	for _, l := range lists {
		for _, f := range l {
			if f >= fmin && f <= fmax {
				all = append(all, f)
			}
		}
	}
	sort.Float64s(all)
	var out []float64
	for _, f := range all {
		if len(out) == 0 || 1200*math.Log2(f/out[len(out)-1]) > mergeCents {
			out = append(out, f)
		}
	}
	return out
}

func noteCandidates(ps []partial, frames [][]float64, sampleRate int, fmin, fmax float64) []float64 {
	var yc []float64
	for _, fr := range frames {
		for _, c := range yinCandidates(fr, sampleRate, fmin, fmax) {
			yc = append(yc, c.Hz)
		}
	}
	return mergeCandidates(fmin, fmax, yc, partialCandidates(ps, fmin, fmax))
}

func twmMargin(ranked []scored) float64 {
	if len(ranked) == 0 {
		return math.NaN()
	}
	for _, s := range ranked[1:] {
		if math.Abs(1200*math.Log2(s.Hz/ranked[0].Hz)) >= 50 {
			return s.Cost - ranked[0].Cost
		}
	}
	return math.Inf(1)
}
