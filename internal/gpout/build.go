package gpout

import (
	"math"
	"os"

	"github.com/ally1002/bass2tably/internal/music"
	"github.com/ally1002/bass2tably/internal/tab"
)

func Build(notes []music.Note, frets []tab.Fretting, tuning []int, gridDiv int, bpm float64, title string) (Score, Report) {
	bars, rep := Bars(notes, frets, len(tuning), gridDiv)
	return Score{Title: title, Tempo: int(math.Round(bpm)), Tuning: tuning, Bars: bars}, rep
}

func Write(path string, s Score) error {
	return os.WriteFile(path, Encode(s), 0o644)
}
