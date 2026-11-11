package wavio_test

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/ally1002/bass2tably/internal/wavio"
)

func TestReadInfoDescribesWithoutDecoding(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "take.wav")
	samples := make([]float64, 44100*3/2)
	for i := range samples {
		samples[i] = 0.5 * math.Sin(float64(i)/20)
	}
	writeWav(t, path, samples, 44100)

	info, err := wavio.ReadInfo(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.SampleRate != 44100 || info.Channels != 1 || info.BitDepth != 16 {
		t.Errorf("info = %+v, want 44100 Hz mono 16-bit", info)
	}
	if math.Abs(info.Duration-1.5) > 1e-3 {
		t.Errorf("Duration = %f, want 1.5", info.Duration)
	}

	bad := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(bad, []byte("not audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := wavio.ReadInfo(bad); err == nil {
		t.Error("ReadInfo accepted a text file")
	}
	if _, err := wavio.ReadInfo(filepath.Join(dir, "missing.wav")); err == nil {
		t.Error("ReadInfo accepted a missing file")
	}
}
