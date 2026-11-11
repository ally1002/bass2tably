package app

import (
	"testing"

	"github.com/ally1002/bass2tably/internal/music"
	"github.com/ally1002/bass2tably/internal/tab"
)

func TestResultValidOK(t *testing.T) {
	r := &Result{
		Tuning: []int{23, 28, 33, 38, 43},
		Notes:  []music.Note{{Midi: 33}, {Midi: 28}},
		Frets:  []tab.Fretting{{String: 2, Fret: 0}, {String: -1, Fret: -1}},
	}
	if err := r.Valid(); err != nil {
		t.Errorf("Valid() = %v, want nil", err)
	}
}

func TestResultValidShortFrets(t *testing.T) {
	r := &Result{
		Tuning: []int{23, 28, 33, 38, 43},
		Notes:  []music.Note{{Midi: 33}, {Midi: 28}},
		Frets:  []tab.Fretting{{String: 2, Fret: 0}},
	}
	if err := r.Valid(); err == nil {
		t.Error("Valid() = nil, want error for mismatched Notes/Frets lengths")
	}
}

func TestResultValidOutOfRangeString(t *testing.T) {
	r := &Result{
		Tuning: []int{23, 28, 33, 38, 43},
		Notes:  []music.Note{{Midi: 33}},
		Frets:  []tab.Fretting{{String: 5, Fret: 0}},
	}
	if err := r.Valid(); err == nil {
		t.Error("Valid() = nil, want error for out-of-range Fretting.String")
	}
}

func TestResultValidEmpty(t *testing.T) {
	r := &Result{Tuning: []int{23, 28, 33, 38, 43}}
	if err := r.Valid(); err != nil {
		t.Errorf("Valid() = %v, want nil for empty Notes/Frets", err)
	}
}
