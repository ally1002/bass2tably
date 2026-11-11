package app

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"

	"github.com/ally1002/bass2tably/internal/synth"
)

var cacheOpts = Options{Tuning: "BEADG", Grid: 16, BPM: 100}

func writeRiff(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "riff.wav")
	samples := synth.RenderLead(synth.TestRiff, 100, 44100, 0)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	data := make([]int, len(samples))
	for i, s := range samples {
		data[i] = int(math.Round(math.Max(-1, math.Min(1, s)) * 32767))
	}
	enc := wav.NewEncoder(f, 44100, 16, 1, 1)
	if err := enc.Write(&audio.IntBuffer{Format: &audio.Format{NumChannels: 1, SampleRate: 44100}, Data: data, SourceBitDepth: 16}); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func stripSamples(r *Result) Result {
	c := *r
	c.Samples = nil
	return c
}

func TestCacheRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	path := writeRiff(t)
	if _, ok := Peek(path, cacheOpts); ok {
		t.Fatal("Peek found a summary before any analysis")
	}
	first, err := AnalyzeCached(path, cacheOpts)
	if err != nil {
		t.Fatal(err)
	}
	k, err := keyFor(path, cacheOpts)
	if err != nil {
		t.Fatal(err)
	}
	file, _ := cacheFile(k)
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("no cache file after the first analysis: %v", err)
	}
	if _, err := os.Stat(summaryFile(file)); err != nil {
		t.Fatalf("no summary file after the first analysis: %v", err)
	}
	c, ok := load(path, cacheOpts)
	if !ok || len(c.Notes) != len(first.Notes) {
		t.Fatalf("load after store: ok %v, %d notes", ok, len(c.Notes))
	}
	second, err := AnalyzeCached(path, cacheOpts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stripSamples(first), stripSamples(second)) {
		t.Fatalf("cached result differs from the analysis:\n%+v\n%+v", stripSamples(first), stripSamples(second))
	}
	if len(second.Samples) != len(first.Samples) {
		t.Fatalf("cached result reloaded %d samples, want %d", len(second.Samples), len(first.Samples))
	}
	s, ok := Peek(path, cacheOpts)
	if !ok || s.Notes != len(first.Notes) || s.BPM != first.BPM || s.Estimated != first.BPMEstimated {
		t.Fatalf("Peek = %+v %v, want %d notes at %.1f bpm", s, ok, len(first.Notes), first.BPM)
	}
}

func TestCacheMissOnIdentityOrOptions(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	path := writeRiff(t)
	res, err := AnalyzeCached(path, cacheOpts)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := Peek(path, cacheOpts); !ok {
		t.Fatal("no summary after analysis")
	}
	other := cacheOpts
	other.Grid = 8
	if _, ok := Peek(path, other); ok {
		t.Error("a different grid found the summary")
	}
	other = cacheOpts
	other.BPM = 0
	if _, ok := Peek(path, other); ok {
		t.Error("a different tempo option found the summary")
	}

	k, _ := keyFor(path, cacheOpts)
	k.Version++
	bumped, _ := cacheFile(k)
	if _, err := os.Stat(bumped); err == nil {
		t.Error("a bumped cacheVersion still maps to the stored file")
	}

	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
	if _, ok := Peek(path, cacheOpts); ok {
		t.Error("a touched file still found the summary")
	}
	Store(path, cacheOpts, res)
	if _, ok := Peek(path, cacheOpts); !ok {
		t.Error("Store with the new mtime did not refresh the summary")
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := Peek(path, cacheOpts); ok {
		t.Error("a file of a different size found the summary")
	}
}

func TestPeekReadsSummaryOnly(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	path := writeRiff(t)
	first, err := AnalyzeCached(path, cacheOpts)
	if err != nil {
		t.Fatal(err)
	}
	k, _ := keyFor(path, cacheOpts)
	file, _ := cacheFile(k)
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if s, ok := Peek(path, cacheOpts); !ok || s.Notes != len(first.Notes) {
		t.Fatalf("Peek without the .gob = %+v %v", s, ok)
	}
	again, err := AnalyzeCached(path, cacheOpts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stripSamples(first), stripSamples(again)) {
		t.Fatal("re-analysis after a missing .gob differs")
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatal("the .gob was not rewritten after the miss")
	}
}

func TestCacheUnwritableIsSilent(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "nope", "deeper"))
	path := writeRiff(t)
	if _, err := AnalyzeCached(path, cacheOpts); err != nil {
		t.Fatalf("analysis failed because the cache dir is unwritable: %v", err)
	}
	if _, ok := Peek(filepath.Join(t.TempDir(), "missing.wav"), cacheOpts); ok {
		t.Fatal("Peek found a missing file")
	}
}
