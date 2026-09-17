package dsp

import "math"

type yinCandidate struct {
	Hz    float64
	Depth float64
}

const yinCeiling = 0.70

func Yin(frame []float64, sampleRate int, fmin, fmax, threshold float64) (float64, bool) {
	for _, c := range yinCandidates(frame, sampleRate, fmin, fmax) {
		if c.Depth < threshold {
			return c.Hz, true
		}
	}
	return 0, false
}

func yinCandidates(frame []float64, sampleRate int, fmin, fmax float64) []yinCandidate {
	cmnd, tauMin, tauMax := cmndf(frame, sampleRate, fmin, fmax)
	if cmnd == nil {
		return nil
	}
	var out []yinCandidate
	for tau := tauMin; tau <= tauMax; tau++ {
		if cmnd[tau] >= yinCeiling || cmnd[tau] >= cmnd[tau-1] || cmnd[tau] > cmnd[tau+1] {
			continue
		}
		out = append(out, yinCandidate{Hz: float64(sampleRate) / refineTau(cmnd, tau), Depth: cmnd[tau]})
	}
	return out
}

func cmndf(frame []float64, sampleRate int, fmin, fmax float64) (cmnd []float64, tauMin, tauMax int) {
	w := len(frame) / 2
	tauMin = int(float64(sampleRate) / fmax)
	tauMax = int(math.Ceil(float64(sampleRate) / fmin))
	if tauMax > w-1 {
		tauMax = w - 1
	}
	if tauMin < 2 {
		tauMin = 2
	}
	if tauMin >= tauMax {
		return nil, 0, 0
	}

	n := tauMax + 1
	diff := make([]float64, n+1)
	for tau := 1; tau <= n; tau++ {
		var sum float64
		for i := 0; i < w; i++ {
			d := frame[i] - frame[i+tau]
			sum += d * d
		}
		diff[tau] = sum
	}

	cmnd = make([]float64, n+1)
	cmnd[0] = 1
	var running float64
	for tau := 1; tau <= n; tau++ {
		running += diff[tau]
		if running == 0 {
			cmnd[tau] = 1
		} else {
			cmnd[tau] = diff[tau] * float64(tau) / running
		}
	}
	return cmnd, tauMin, tauMax
}

func refineTau(cmnd []float64, tau int) float64 {
	if tau < 1 || tau+1 >= len(cmnd) {
		return float64(tau)
	}
	a, b, c := cmnd[tau-1], cmnd[tau], cmnd[tau+1]
	den := a - 2*b + c
	if den == 0 {
		return float64(tau)
	}
	return float64(tau) + 0.5*(a-c)/den
}
