package dsp

import "math"

const (
	oracleSubRatio     = 0.60
	oracleStepCent     = 25.0
	oracleMinHarmonics = 3
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
	for cents := 0.0; ; cents += oracleStepCent {
		f := fmin * math.Exp2(cents/1200)
		if f > fmax {
			break
		}
		s, matched, refined := harmonicMatch(ps, f, 8)
		if s <= 0 {
			continue
		}
		if matched > bestMatched || (matched == bestMatched && s >= bestScore) {
			best, bestScore, bestMatched = refined, s, matched
		}
	}
	if bestMatched < oracleMinHarmonics {
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
		if s, _, refined := harmonicMatch(ps, sub, 8); s >= oracleSubRatio*bestScore {
			best = refined
		}
	}
	return best, true
}
