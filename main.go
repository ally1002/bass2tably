package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"bass2tably/internal/app"
	"bass2tably/internal/midiout"
	"bass2tably/internal/tui"
)

func main() {
	bpm := flag.Float64("bpm", 0, "tempo of the recording in BPM (0 = estimate automatically)")
	tuning := flag.String("tuning", "BEADG", "tuning: BEADG, EADG, DADG or BEAD")
	grid := flag.Int("grid", 16, "quantization grid: 8 or 16")
	position := flag.Float64("position", 0, "bias fret assignment toward this neck position")
	out := flag.String("out", "", "write a MIDI file to this path (omit to skip MIDI output)")
	tuiFlag := flag.Bool("tui", false, "open the TUI viewer for the given file")
	theme := flag.String("theme", "amber", "TUI colour theme: "+strings.Join(tui.ThemeNames(), ", "))
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: bass2tably [flags] [input.wav]\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	var err error
	if *tuiFlag || flag.NArg() == 0 {
		err = runTUI(*bpm, *tuning, *grid, *position, *theme, flag.Args())
	} else {
		err = run(*bpm, *tuning, *grid, *position, *out, flag.Args())
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bass2tably:", err)
		os.Exit(1)
	}
}

func runTUI(bpm float64, tuning string, grid int, position float64, theme string, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("expected at most one input file, got %d", len(args))
	}
	path := ""
	if len(args) == 1 {
		path = args[0]
	}
	return tui.Run(path, app.Options{
		BPM:      bpm,
		Tuning:   tuning,
		Grid:     grid,
		Position: position,
	}, tui.Config{Theme: theme})
}

func run(bpm float64, tuning string, grid int, position float64, out string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected exactly one input WAV file, got %d", len(args))
	}

	input := args[0]

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

	if out != "" {
		if err := midiout.Write(out, res.Notes, res.BPM, res.GridDiv); err != nil {
			return err
		}
		fmt.Printf("\nwrote %s\n", out)
	}
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

	for _, line := range res.NoteLines() {
		fmt.Println(line)
	}
}
