package wavio_test

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"

	"bass2tably/internal/dsp"
	"bass2tably/internal/music"
	"bass2tably/internal/synth"
	"bass2tably/internal/wavio"
)

func TestLoadAnyFormats(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}
	const sr = 44100
	src := filepath.Join(t.TempDir(), "riff.wav")
	writeWav(t, src, synth.Render(synth.TestRiff, 100, sr), sr)

	wantMidi := make([]int, len(synth.TestRiff))
	for i, e := range synth.TestRiff {
		wantMidi[i] = e.Midi
	}

	cases := []struct {
		name string
		ext  string
		args []string
	}{
		{"s16", ".wav", []string{"-c:a", "pcm_s16le"}},
		{"s24", ".wav", []string{"-c:a", "pcm_s24le"}},
		{"s32", ".wav", []string{"-c:a", "pcm_s32le"}},
		{"f32", ".wav", []string{"-c:a", "pcm_f32le"}},
		{"mp3", ".mp3", nil},
		{"m4a", ".m4a", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			enc := filepath.Join(t.TempDir(), tc.name+tc.ext)
			args := append([]string{"-y", "-loglevel", "error", "-i", src}, tc.args...)
			args = append(args, enc)
			if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
				t.Skipf("ffmpeg cannot encode %s: %v: %s", tc.name, err, out)
			}

			clip, err := wavio.LoadAny(enc)
			if err != nil {
				t.Fatal(err)
			}
			segs := dsp.Segments(clip.Samples, clip.SampleRate)
			if len(segs) != len(wantMidi) {
				t.Fatalf("got %d segments, want %d", len(segs), len(wantMidi))
			}
			for i, seg := range segs {
				freq, ok := dsp.NotePitch(clip.Samples, clip.SampleRate, seg)
				if !ok {
					t.Fatalf("note %d: no pitch detected", i)
				}
				if got := music.MidiFromFreq(freq); got != wantMidi[i] {
					t.Errorf("note %d: midi %d, want %d", i, got, wantMidi[i])
				}
			}
		})
	}
}

func writeWav(t *testing.T, path string, samples []float64, sr int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	data := make([]int, len(samples))
	for i, s := range samples {
		data[i] = int(math.Round(math.Max(-1, math.Min(1, s)) * 32767))
	}
	enc := wav.NewEncoder(f, sr, 16, 1, 1)
	buf := &audio.IntBuffer{
		Format:         &audio.Format{NumChannels: 1, SampleRate: sr},
		Data:           data,
		SourceBitDepth: 16,
	}
	if err := enc.Write(buf); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
}
