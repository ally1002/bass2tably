package tui

import (
	"math"
	"strings"
)

const (
	blocks      = "▁▂▃▄▅▆▇█"
	blocksASCII = " .:-=+*#"
	playheadBar = "▏"
	playheadASC = "|"
)

func peaks(samples []float64, n int) []float64 {
	if n <= 0 || len(samples) == 0 {
		return nil
	}
	out := make([]float64, n)
	var top float64
	for i := range out {
		lo, hi := i*len(samples)/n, (i+1)*len(samples)/n
		for _, s := range samples[lo:hi] {
			out[i] = max(out[i], math.Abs(s))
		}
		top = max(top, out[i])
	}
	if top > 0 {
		for i := range out {
			out[i] /= top
		}
	}
	return out
}

func waveRow(pk []float64, frac float64, st styles, ascii bool) string {
	var w runWriter
	shapes, head := []rune(blocks), playheadBar
	if ascii {
		shapes, head = []rune(blocksASCII), playheadASC
	}
	done, todo, cur := newPaint(st.accent), newPaint(st.faint), newPaint(st.accent.Bold(true))
	played := int(frac*float64(len(pk)) + 0.5)
	for i, p := range pk {
		r := string(shapes[min(int(p*7+0.5), 7)])
		switch {
		case i == played:
			w.write(cur, head)
		case i < played:
			w.write(done, r)
		default:
			w.write(todo, r)
		}
	}
	return w.String()
}

func blankRow(w int) string {
	return strings.Repeat(" ", max(w, 0))
}
