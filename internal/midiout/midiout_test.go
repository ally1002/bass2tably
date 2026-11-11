package midiout_test

import (
	"path/filepath"
	"testing"

	"gitlab.com/gomidi/midi/v2/smf"

	"github.com/ally1002/bass2tably/internal/midiout"
	"github.com/ally1002/bass2tably/internal/music"
)

func TestWriteRoundtrip(t *testing.T) {
	notes := []music.Note{
		{Midi: 33, Slot: 0, Len: 2},
		{Midi: 33, Slot: 2, Len: 2},
		{Midi: 36, Slot: 4, Len: 4},
	}
	path := filepath.Join(t.TempDir(), "riff.mid")
	if err := midiout.Write(path, notes, 100, 4); err != nil {
		t.Fatal(err)
	}

	s, err := smf.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Tracks) != 1 {
		t.Fatalf("got %d tracks, want 1", len(s.Tracks))
	}

	type ev struct {
		tick int
		on   bool
		key  uint8
	}
	var got []ev
	tick := 0
	var bpm float64
	for _, e := range s.Tracks[0] {
		tick += int(e.Delta)
		var ch, key, vel uint8
		if e.Message.GetMetaTempo(&bpm) {
			continue
		}
		if e.Message.GetNoteStart(&ch, &key, &vel) {
			got = append(got, ev{tick, true, key})
		} else if e.Message.GetNoteEnd(&ch, &key) {
			got = append(got, ev{tick, false, key})
		}
	}

	want := []ev{
		{0, true, 33},
		{480, false, 33}, {480, true, 33},
		{960, false, 33}, {960, true, 36},
		{1920, false, 36},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d note events, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
	if bpm != 100 {
		t.Errorf("tempo %.1f, want 100", bpm)
	}
}
