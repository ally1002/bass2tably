package dsp

import "math"

func Yin(frame []float64, sampleRate int, fmin, fmax, threshold float64) (float64, bool) {
	w := len(frame) / 2
	tauMin := int(float64(sampleRate) / fmax)
	tauMax := int(math.Ceil(float64(sampleRate) / fmin))
	if tauMax > w-1 {
		tauMax = w - 1
	}
	if tauMin < 2 {
		tauMin = 2
	}
	if tauMin >= tauMax {
		return 0, false
	}

	diff := make([]float64, tauMax+1)
	for tau := 1; tau <= tauMax; tau++ {
		var sum float64
		for i := 0; i < w; i++ {
			d := frame[i] - frame[i+tau]
			sum += d * d
		}
		diff[tau] = sum
	}

	cmnd := make([]float64, tauMax+1)
	cmnd[0] = 1
	var running float64
	for tau := 1; tau <= tauMax; tau++ {
		running += diff[tau]
		if running == 0 {
			cmnd[tau] = 1
		} else {
			cmnd[tau] = diff[tau] * float64(tau) / running
		}
	}

	for tau := tauMin; tau <= tauMax; tau++ {
		if cmnd[tau] < threshold {
			for tau+1 <= tauMax && cmnd[tau+1] < cmnd[tau] {
				tau++
			}
			return float64(sampleRate) / refineTau(cmnd, tau), true
		}
	}
	return 0, false
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
