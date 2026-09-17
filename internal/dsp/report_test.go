package dsp

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"bass2tably/internal/music"
	"bass2tably/internal/synth"
	"bass2tably/internal/wavio"
)

func TestCorpusReport(t *testing.T) {
	if os.Getenv("BASS2TABLY_REPORT") == "" {
		t.Skip("set BASS2TABLY_REPORT=1 to print the per-segment pitch table")
	}
	for _, f := range []string{"../../testdata/shh.wav", "../../testdata/ahhintro.wav", "../../testdata/ahhoutro.wav"} {
		clip, err := wavio.LoadAny(f)
		if err != nil {
			t.Fatal(err)
		}
		reportCorpus(t, filepath.Base(f), clip.Samples, clip.SampleRate)
	}
	for _, bpm := range []float64{80, 100, 140} {
		for _, lead := range []float64{0, 0.30, 1.37} {
			x := synth.RenderLead(synth.TestRiff, bpm, 44100, lead)
			reportCorpus(t, fmt.Sprintf("riff-%.0f-%.2f", bpm, lead), x, 44100)
		}
	}
}

func reportCorpus(t *testing.T, name string, x []float64, sr int) {
	t.Helper()
	t.Logf("== %s", name)
	t.Logf("%3s %8s %6s %5s  %8s %5s  %8s %2s  %6s  %s",
		"seg", "start", "len", "rung", "yin", "frm", "spec", "h", "cents", "final")

	var same, diff, unvoiced, noEst int
	var abs []float64
	for i, seg := range Segments(x, sr) {
		start := seg.Start + int(attackSkipSec*float64(sr))
		end := seg.End
		if end > len(x) {
			end = len(x)
		}
		rung := specRung(len(x), start, end-start)

		freqs, frames := yinFrames(x, sr, seg)
		yf, yok := yinMedian(x, sr, seg)
		ps := spectralPartials(x, sr, start, end-start)
		sf, sok := spectralF0(ps, yinFmin, yinFmax)
		_, matched, _ := harmonicMatch(ps, sf, 8)
		ff, fok := NotePitch(x, sr, seg)

		yin := "-"
		if yok {
			yin = fmt.Sprintf("%8.2f", yf)
		} else {
			unvoiced++
		}
		spec := "-"
		if sok {
			spec = fmt.Sprintf("%8.2f", sf)
		} else {
			noEst++
			matched = 0
		}
		cents := "-"
		if yok && sok {
			c := 1200 * math.Log2(sf/yf)
			cents = fmt.Sprintf("%+6.0f", c)
			abs = append(abs, math.Abs(c))
			if music.MidiFromFreq(yf) == music.MidiFromFreq(sf) {
				same++
			} else {
				diff++
			}
		}
		final := "-"
		if fok {
			final = fmt.Sprintf("%d %s %.2f", music.MidiFromFreq(ff), music.MidiName(music.MidiFromFreq(ff)), ff)
		}
		t.Logf("%3d %8.3f %6d %5d  %8s %2d/%-2d  %8s %2d  %6s  %s",
			i, float64(seg.Start)/float64(sr), seg.End-seg.Start, rung,
			yin, len(freqs), frames, spec, matched, cents, final)
	}
	t.Logf("-- %s: same %d, different %d, |cents| p50/p90/max %.0f/%.0f/%.0f, unvoiced %d, no-estimate %d",
		name, same, diff, percentile(abs, 0.5), percentile(abs, 0.9), percentile(abs, 1), unvoiced, noEst)
}

func percentile(v []float64, p float64) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	k := int(math.Ceil(p*float64(len(s)))) - 1
	if k < 0 {
		k = 0
	}
	return s[k]
}
