package dsp

import "sort"

const (
	yinWin        = 3072
	yinHop        = 1024
	yinFmin       = 28.0
	yinFmax       = 450.0
	yinThreshold  = 0.35
	attackSkipSec = 0.010
)

func NotePitch(samples []float64, sampleRate int, seg Segment) (float64, bool) {
	start := seg.Start + int(attackSkipSec*float64(sampleRate))
	end := seg.End
	if end > len(samples) {
		end = len(samples)
	}

	var freqs []float64
	for p := start; p+yinWin <= end; p += yinHop {
		if f, ok := Yin(samples[p:p+yinWin], sampleRate, yinFmin, yinFmax, yinThreshold); ok {
			freqs = append(freqs, f)
		}
	}
	if len(freqs) == 0 && start < len(samples) {
		stop := start + yinWin
		if stop > len(samples) {
			stop = len(samples)
		}
		if f, ok := Yin(samples[start:stop], sampleRate, yinFmin, yinFmax, yinThreshold); ok {
			freqs = append(freqs, f)
		}
	}
	if len(freqs) == 0 {
		return 0, false
	}

	sort.Float64s(freqs)
	mid := len(freqs) / 2
	if len(freqs)%2 == 0 {
		return (freqs[mid-1] + freqs[mid]) / 2, true
	}
	return freqs[mid], true
}
