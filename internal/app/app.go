package app

import (
	"fmt"

	"bass2tably/internal/dsp"
	"bass2tably/internal/music"
	"bass2tably/internal/tab"
	"bass2tably/internal/wavio"
)

var Tunings = map[string][]int{
	"BEADG": {23, 28, 33, 38, 43},
	"EADG":  {28, 33, 38, 43},
	"DADG":  {26, 33, 38, 43},
	"BEAD":  {23, 28, 33, 38},
}

type Options struct {
	BPM      float64
	Tuning   string
	Grid     int
	Position float64
}

type Skipped struct {
	Index    int
	StartSec float64
	DurSec   float64
}

type Result struct {
	Path         string
	SampleRate   int
	DurationSec  float64
	Peak         float64
	Samples      []float64
	BPM          float64
	BPMEstimated bool
	GridDiv      int
	Tuning       []int
	Notes        []music.Note
	Frets        []tab.Fretting
	Skipped      []Skipped
	Tab          string
	Layout       tab.Layout
}

func Analyze(path string, opts Options) (*Result, error) {
	open, ok := Tunings[opts.Tuning]
	if !ok {
		return nil, fmt.Errorf("unknown tuning %q (valid: BEADG, EADG, DADG, BEAD)", opts.Tuning)
	}
	if opts.Grid != 8 && opts.Grid != 16 {
		return nil, fmt.Errorf("--grid must be 8 or 16, got %d", opts.Grid)
	}

	clip, err := wavio.LoadAny(path)
	if err != nil {
		return nil, err
	}

	segs := dsp.Segments(clip.Samples, clip.SampleRate)

	bpm := opts.BPM
	estimated := false
	if bpm <= 0 {
		onsets := make([]float64, len(segs))
		for i, s := range segs {
			onsets[i] = float64(s.Start) / float64(clip.SampleRate)
		}
		est, ok := music.EstimateBPM(onsets)
		if !ok {
			return nil, fmt.Errorf("could not estimate tempo (too few notes); pass --bpm")
		}
		bpm = est
		estimated = true
	}

	notes := make([]music.Note, 0, len(segs))
	var skipped []Skipped
	for i, s := range segs {
		start := float64(s.Start) / float64(clip.SampleRate)
		dur := float64(s.End-s.Start) / float64(clip.SampleRate)
		freq, ok := dsp.NotePitch(clip.Samples, clip.SampleRate, s)
		if !ok {
			skipped = append(skipped, Skipped{Index: i, StartSec: start, DurSec: dur})
			continue
		}
		notes = append(notes, music.Note{
			Midi:     music.MidiFromFreq(freq),
			StartSec: start,
			DurSec:   dur,
		})
	}

	gridDiv := opts.Grid / 4
	music.Quantize(notes, bpm, gridDiv)

	frets := tab.Assign(notes, open, opts.Position)
	tabStr, layout := tab.RenderLayout(notes, frets, open, gridDiv)

	return &Result{
		Path:         path,
		SampleRate:   clip.SampleRate,
		DurationSec:  clip.Duration(),
		Peak:         clip.Peak(),
		Samples:      clip.Samples,
		BPM:          bpm,
		BPMEstimated: estimated,
		GridDiv:      gridDiv,
		Tuning:       open,
		Notes:        notes,
		Frets:        frets,
		Skipped:      skipped,
		Tab:          tabStr,
		Layout:       layout,
	}, nil
}
