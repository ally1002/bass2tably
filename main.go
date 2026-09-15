package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bass2tably/internal/app"
	"bass2tably/internal/midiout"
	"bass2tably/internal/music"
	"bass2tably/internal/tab"
)

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

	input := args[0]
	if out == "" {
		out = strings.TrimSuffix(input, filepath.Ext(input)) + ".mid"
	}

	res, err := app.Analyze(input, app.Options{
		BPM:      bpm,
		Tuning:   tuning,
		Grid:     grid,
		Position: position,
	})
	if err != nil {
		return err
	}

	printResult(res)

	if err := midiout.Write(out, res.Notes, res.BPM, res.GridDiv); err != nil {
		return err
	}
	fmt.Printf("\nwrote %s\n", out)
	return nil
}

func printResult(res *app.Result) {
	fmt.Printf("input:       %s\n", res.Path)
	fmt.Printf("sample rate: %d Hz\n", res.SampleRate)
	fmt.Printf("duration:    %.3f s\n", res.DurationSec)
	fmt.Printf("peak:        %.3f\n", res.Peak)
	fmt.Printf("notes:       %d\n", len(res.Notes)+len(res.Skipped))

	if res.BPMEstimated {
		fmt.Printf("\nestimated tempo: %.1f BPM   (override with --bpm if wrong)\n", res.BPM)
	}

	for _, s := range res.Skipped {
		fmt.Printf("  #%-2d  %.3fs  dur %.3fs  (no pitch, skipped)\n", s.Index+1, s.StartSec, s.DurSec)
	}

	fmt.Println()
	fmt.Print(res.Tab)
	fmt.Println()

	stringName := func(s int) string {
		return strings.TrimRight(music.MidiName(res.Tuning[s]), "-0123456789")
	}
	alts := make([]string, len(res.Notes))
	altW := 0
	for i, n := range res.Notes {
		var parts []string
		for _, p := range tab.Positions(n.Midi, res.Tuning) {
			if p == res.Frets[i] {
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
	for i, n := range res.Notes {
		f := res.Frets[i]
		pos := "unplayable in this tuning"
		if f.String >= 0 {
			pos = fmt.Sprintf("string %s fret %d", stringName(f.String), f.Fret)
		}
		fmt.Printf("#%-2d  %5.2fs  %-3s  %-16s  %-*s  %s\n",
			i+1, n.StartSec, music.MidiName(n.Midi), pos, altW, alts[i], beatsLabel(n.Len, res.GridDiv))
	}
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
