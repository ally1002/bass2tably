package music_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ally1002/bass2tably/internal/dsp"
	"github.com/ally1002/bass2tably/internal/midiout"
	"github.com/ally1002/bass2tably/internal/music"
	"github.com/ally1002/bass2tably/internal/synth"
	"github.com/ally1002/bass2tably/internal/tab"
)

func TestQuantizeTestRiff(t *testing.T) {
	const bpm = 100.0
	const sr = 44100
	const gridDiv = 4

	samples := synth.Render(synth.TestRiff, bpm, sr)
	segs := dsp.Segments(samples, sr)
	if len(segs) != len(synth.TestRiff) {
		t.Fatalf("got %d segments, want %d", len(segs), len(synth.TestRiff))
	}

	notes := make([]music.Note, len(segs))
	for i, seg := range segs {
		freq, ok := dsp.NotePitch(samples, sr, seg)
		if !ok {
			t.Fatalf("note %d: no pitch detected", i)
		}
		notes[i] = music.Note{
			Midi:     music.MidiFromFreq(freq),
			StartSec: float64(seg.Start) / sr,
			DurSec:   float64(seg.End-seg.Start) / sr,
		}
	}
	music.Quantize(notes, bpm, gridDiv)

	wantSlot := []int{0, 2, 4, 8, 12, 14, 16, 20, 24}
	wantLen := []int{2, 2, 4, 4, 2, 2, 4, 4, 4}
	for i, n := range notes {
		if n.Slot != wantSlot[i] {
			t.Errorf("note %d: slot %d, want %d", i, n.Slot, wantSlot[i])
		}
		if n.Len != wantLen[i] {
			t.Errorf("note %d: len %d, want %d", i, n.Len, wantLen[i])
		}
	}
	for i := 0; i+1 < len(notes); i++ {
		if notes[i].Slot+notes[i].Len > notes[i+1].Slot {
			t.Errorf("note %d overlaps note %d", i, i+1)
		}
	}
}

func TestQuantizeLeadInvariance(t *testing.T) {
	const bpm = 100.0
	const sr = 44100
	const gridDiv = 4
	open := []int{23, 28, 33, 38, 43}

	pipeline := func(name string, samples []float64) (string, []byte) {
		segs := dsp.Segments(samples, sr)
		if len(segs) != len(synth.TestRiff) {
			t.Fatalf("%s: got %d segments, want %d", name, len(segs), len(synth.TestRiff))
		}
		notes := make([]music.Note, len(segs))
		for i, seg := range segs {
			freq, ok := dsp.NotePitch(samples, sr, seg)
			if !ok {
				t.Fatalf("%s: note %d: no pitch detected", name, i)
			}
			notes[i] = music.Note{
				Midi:     music.MidiFromFreq(freq),
				StartSec: float64(seg.Start) / sr,
				DurSec:   float64(seg.End-seg.Start) / sr,
			}
		}
		music.Quantize(notes, bpm, gridDiv)
		frets := tab.Assign(notes, open, 0)
		ascii := tab.Render(notes, frets, open, gridDiv)
		path := filepath.Join(t.TempDir(), name+".mid")
		if err := midiout.Write(path, notes, bpm, gridDiv); err != nil {
			t.Fatalf("%s: midi write: %v", name, err)
		}
		mid, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: midi read: %v", name, err)
		}
		return ascii, mid
	}

	baseTab, baseMid := pipeline("base", synth.Render(synth.TestRiff, bpm, sr))
	leadTab, leadMid := pipeline("lead", synth.RenderLead(synth.TestRiff, bpm, sr, 1.37))

	if leadTab != baseTab {
		t.Errorf("tab differs with 1.37s lead:\nbase:\n%s\nlead:\n%s", baseTab, leadTab)
	}
	if !bytes.Equal(leadMid, baseMid) {
		t.Errorf("MIDI differs with 1.37s lead: %d vs %d bytes", len(baseMid), len(leadMid))
	}
}

func TestQuantizeClampsOverlap(t *testing.T) {
	notes := []music.Note{
		{Midi: 33, StartSec: 0, DurSec: 1.0},
		{Midi: 36, StartSec: 0.45, DurSec: 0.3},
	}
	music.Quantize(notes, 100, 4)
	if notes[0].Slot != 0 || notes[1].Slot != 3 {
		t.Fatalf("slots %d,%d, want 0,3", notes[0].Slot, notes[1].Slot)
	}
	if notes[0].Len != 3 {
		t.Errorf("note 0 len %d, want 3 after clamp", notes[0].Len)
	}
}
