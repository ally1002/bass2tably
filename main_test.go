package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"bass2tably/internal/dsp"
	"bass2tably/internal/music"
	"bass2tably/internal/wavio"
)

func TestRealRecordings(t *testing.T) {
	wavs, err := filepath.Glob(filepath.Join("testdata", "*.wav"))
	if err != nil {
		t.Fatal(err)
	}
	if len(wavs) == 0 {
		t.Skip("no real-recording fixtures in testdata/")
	}
	for _, wavPath := range wavs {
		t.Run(strings.TrimSuffix(filepath.Base(wavPath), ".wav"), func(t *testing.T) {
			data, err := os.ReadFile(strings.TrimSuffix(wavPath, ".wav") + ".expected")
			if err != nil {
				t.Fatal(err)
			}
			var want []int
			for _, line := range strings.Split(string(data), "\n") {
				if i := strings.IndexByte(line, '#'); i >= 0 {
					line = line[:i]
				}
				for _, f := range strings.Fields(line) {
					v, err := strconv.Atoi(f)
					if err != nil {
						t.Fatalf("bad .expected value %q: %v", f, err)
					}
					want = append(want, v)
				}
			}

			clip, err := wavio.LoadAny(wavPath)
			if err != nil {
				t.Fatal(err)
			}
			var got []int
			for _, seg := range dsp.Segments(clip.Samples, clip.SampleRate) {
				if freq, ok := dsp.NotePitch(clip.Samples, clip.SampleRate, seg); ok {
					got = append(got, music.MidiFromFreq(freq))
				}
			}

			if len(got) != len(want) {
				t.Fatalf("got %d pitched notes, want %d\ngot: %v", len(got), len(want), got)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("note %d: midi %d (%s), want %d (%s)",
						i, got[i], music.MidiName(got[i]), want[i], music.MidiName(want[i]))
				}
			}
		})
	}
}
