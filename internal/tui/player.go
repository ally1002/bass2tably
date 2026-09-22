package tui

import "bass2tably/internal/audio"

type player interface {
	Play() error
	Pause() error
	Playing() bool
	PosSec() float64
	SeekSec(t float64)
	Done() bool
	Close()
	SetSpeed(f float64)
	SetLoop(startSec, endSec float64)
	ClearLoop()
	SetClick(bpm, offsetSec float64, accentEvery int, on bool)
}

func newPlayer(samples []float64, rate int) (player, error) {
	p, err := audio.New(samples, rate)
	if err != nil {
		return nil, err
	}
	return p, nil
}
