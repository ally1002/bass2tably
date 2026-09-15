package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"bass2tably/internal/dsp"
	"bass2tably/internal/midiout"
	"bass2tably/internal/music"
	"bass2tably/internal/tab"
	"bass2tably/internal/wavio"
)

var tunings = map[string][]int{
	"BEADG": {23, 28, 33, 38, 43},
	"EADG":  {28, 33, 38, 43},
	"DADG":  {26, 33, 38, 43},
	"BEAD":  {23, 28, 33, 38},
}

func main() {
	bpm := flag.Float64("bpm", 0, "tempo of the recording in BPM (0 = estimate automatically)")
	tuning := flag.String("tuning", "BEADG", "tuning: BEADG, EADG, DADG or BEAD")
	grid := flag.Int("grid", 16, "quantization grid: 8 or 16")
	position := flag.Float64("position", 0, "bias fret assignment toward this neck position")
	out := flag.String("out", "", "MIDI output path (default: <input>.mid)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: bass2tably [flags] input.wav\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := run(*bpm, *tuning, *grid, *position, *out, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "bass2tably:", err)
		os.Exit(1)
	}
}

func run(bpm float64, tuning string, grid int, position float64, out string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected exactly one input WAV file, got %d", len(args))
	}
	if _, ok := tunings[tuning]; !ok {
		return fmt.Errorf("unknown tuning %q (valid: BEADG, EADG, DADG, BEAD)", tuning)
	}
	if grid != 8 && grid != 16 {
		return fmt.Errorf("--grid must be 8 or 16, got %d", grid)
	}

	input := args[0]
	if out == "" {
		out = strings.TrimSuffix(input, ".wav") + ".mid"
	}

	clip, err := wavio.Load(input)
	if err != nil {
		return err
	}

	fmt.Printf("input:       %s\n", input)
	fmt.Printf("sample rate: %d Hz\n", clip.SampleRate)
	fmt.Printf("duration:    %.3f s\n", clip.Duration())
	fmt.Printf("peak:        %.3f\n", clip.Peak())

	segs := dsp.Segments(clip.Samples, clip.SampleRate)
	fmt.Printf("notes:       %d\n", len(segs))

	if bpm <= 0 {
		onsets := make([]float64, len(segs))
		for i, s := range segs {
			onsets[i] = float64(s.Start) / float64(clip.SampleRate)
		}
		est, ok := music.EstimateBPM(onsets)
		if !ok {
			return fmt.Errorf("could not estimate tempo (too few notes); pass --bpm")
		}
		bpm = est
		fmt.Printf("\nestimated tempo: %.1f BPM   (override with --bpm if wrong)\n", bpm)
	}

	notes := make([]music.Note, 0, len(segs))
	for i, s := range segs {
		start := float64(s.Start) / float64(clip.SampleRate)
		dur := float64(s.End-s.Start) / float64(clip.SampleRate)
		freq, ok := dsp.NotePitch(clip.Samples, clip.SampleRate, s)
		if !ok {
			fmt.Printf("  #%-2d  %.3fs  dur %.3fs  (no pitch, skipped)\n", i+1, start, dur)
			continue
		}
		notes = append(notes, music.Note{
			Midi:     music.MidiFromFreq(freq),
			StartSec: start,
			DurSec:   dur,
		})
	}

	gridDiv := grid / 4
	music.Quantize(notes, bpm, gridDiv)

	open := tunings[tuning]
	frets := tab.Assign(notes, open, position)

	fmt.Println()
	fmt.Print(tab.Render(notes, frets, open, gridDiv))
	fmt.Println()

	stringName := func(s int) string {
		return strings.TrimRight(music.MidiName(open[s]), "-0123456789")
	}
	alts := make([]string, len(notes))
	altW := 0
	for i, n := range notes {
		var parts []string
		for _, p := range tab.Positions(n.Midi, open) {
			if p == frets[i] {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s fret %d", stringName(p.String), p.Fret))
		}
		if len(parts) > 0 {
			alts[i] = fmt.Sprintf("[also: %s]", strings.Join(parts, ", "))
		}
		if len(alts[i]) > altW {
			altW = len(alts[i])
		}
	}
	for i, n := range notes {
		f := frets[i]
		pos := "unplayable in this tuning"
		if f.String >= 0 {
			pos = fmt.Sprintf("string %s fret %d", stringName(f.String), f.Fret)
		}
		fmt.Printf("#%-2d  %5.2fs  %-3s  %-16s  %-*s  %s\n",
			i+1, n.StartSec, music.MidiName(n.Midi), pos, altW, alts[i], beatsLabel(n.Len, gridDiv))
	}

	if err := midiout.Write(out, notes, bpm, gridDiv); err != nil {
		return err
	}
	fmt.Printf("\nwrote %s\n", out)
	return nil
}

func beatsLabel(slots, gridDiv int) string {
	g := gcd(slots, gridDiv)
	num, den := slots/g, gridDiv/g
	switch {
	case den == 1 && num == 1:
		return "1 beat"
	case den == 1:
		return fmt.Sprintf("%d beats", num)
	default:
		return fmt.Sprintf("%d/%d beat", num, den)
	}
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
