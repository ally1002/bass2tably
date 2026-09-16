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
}

func newPlayer(samples []float64, rate int) (player, error) {
	p, err := audio.New(samples, rate)
	if err != nil {
		return nil, err
	}
	return p, nil
}
