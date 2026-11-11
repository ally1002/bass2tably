package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"

	"github.com/ally1002/bass2tably/internal/synth"
)

func main() {
	bpm := flag.Float64("bpm", 100, "tempo of the generated riff in BPM")
	sr := flag.Int("sr", 44100, "sample rate")
	lead := flag.Float64("lead", 0, "seconds of silence before the riff")
	out := flag.String("out", "testriff.wav", "output WAV path")
	flag.Parse()

	if err := run(*bpm, *sr, *lead, *out); err != nil {
		fmt.Fprintln(os.Stderr, "gentest:", err)
		os.Exit(1)
	}
}

func run(bpm float64, sr int, lead float64, out string) error {
	samples := synth.RenderLead(synth.TestRiff, bpm, sr, lead)
	if err := writeWav(out, samples, sr); err != nil {
		return err
	}
	var beats float64
	for _, e := range synth.TestRiff {
		beats += e.Beats
	}
	fmt.Printf("wrote %s: %d notes, %.3f s at %g BPM, %d Hz\n",
		out, len(synth.TestRiff), lead+beats*60/bpm, bpm, sr)
	return nil
}

func writeWav(path string, samples []float64, sr int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	data := make([]int, len(samples))
	for i, s := range samples {
		if s > 1 {
			s = 1
		} else if s < -1 {
			s = -1
		}
		data[i] = int(math.Round(s * 32767))
	}
	enc := wav.NewEncoder(f, sr, 16, 1, 1)
	buf := &audio.IntBuffer{
		Format:         &audio.Format{NumChannels: 1, SampleRate: sr},
		Data:           data,
		SourceBitDepth: 16,
	}
	if err := enc.Write(buf); err != nil {
		return err
	}
	return enc.Close()
}
