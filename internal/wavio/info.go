package wavio

import (
	"fmt"
	"os"

	"github.com/go-audio/wav"
)

type Info struct {
	SampleRate, Channels, BitDepth int
	Duration                       float64
}

func ReadInfo(path string) (Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return Info{}, err
	}
	defer f.Close()
	d := wav.NewDecoder(f)
	d.ReadInfo()
	if !d.IsValidFile() {
		return Info{}, fmt.Errorf("%s: not a WAV file", path)
	}
	dur, err := d.Duration()
	if err != nil {
		return Info{}, err
	}
	return Info{int(d.SampleRate), int(d.NumChans), int(d.BitDepth), dur.Seconds()}, nil
}
