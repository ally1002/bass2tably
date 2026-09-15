package music

import "math"

const (
	tempoMin    = 55.0
	tempoMax    = 200.0
	tempoStep   = 0.5
	tempoCenter = 110.0
	tempoSigma  = 0.5
)

func EstimateBPM(onsetsSec []float64) (float64, bool) {
	var iois []float64
	for i := 1; i < len(onsetsSec); i++ {
		if d := onsetsSec[i] - onsetsSec[i-1]; d > 1e-3 {
			iois = append(iois, d)
		}
	}
	if len(iois) < 3 {
		return 0, false
	}

	best, bestScore := 0.0, 0.0
	for bpm := tempoMin; bpm <= tempoMax; bpm += tempoStep {
		if s := tempoScore(iois, bpm); s > bestScore {
			best, bestScore = bpm, s
		}
	}
	if bestScore <= 0 {
		return 0, false
	}
	return refine(iois, best), true
}

func tempoScore(iois []float64, bpm float64) float64 {
	return gridScore(iois, bpm) * tempoPrior(bpm)
}

func gridScore(iois []float64, bpm float64) float64 {
	slot := 60 / bpm / 4
	var score float64
	for _, d := range iois {
		m := math.Round(d / slot)
		if m < 1 || m > 16 {
			continue
		}
		e := math.Abs(d-m*slot) / slot
		w := 1 / math.Sqrt(m)
		if !isPowerOfTwo(int(m)) {
			w *= 0.7
		}
		score += (1 - 2*e) * w
	}
	return score
}

func isPowerOfTwo(n int) bool {
	return n > 0 && n&(n-1) == 0
}

func tempoPrior(bpm float64) float64 {
	oct := math.Log2(bpm / tempoCenter)
	return math.Exp(-oct * oct / (2 * tempoSigma * tempoSigma))
}

func refine(iois []float64, bpm float64) float64 {
	best, bestScore := bpm, tempoScore(iois, bpm)
	for b := bpm - 2; b <= bpm+2; b += 0.05 {
		if s := tempoScore(iois, b); s > bestScore {
			best, bestScore = b, s
		}
	}
	return best
}
