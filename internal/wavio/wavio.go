package wavio

import (
	"fmt"
	"os"

	"github.com/go-audio/wav"
)

type Clip struct {
	Samples    []float64
	SampleRate int
}

func Load(path string) (*Clip, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d := wav.NewDecoder(f)
	buf, err := d.FullPCMBuffer()
	if err != nil {
		return nil, err
	}
	if buf == nil || len(buf.Data) == 0 {
		return nil, fmt.Errorf("%s: no PCM data", path)
	}
	ch := buf.Format.NumChannels
	if ch < 1 {
		return nil, fmt.Errorf("%s: invalid channel count %d", path, ch)
	}
	scale := float64(int(1) << (int(d.BitDepth) - 1))
	n := len(buf.Data) / ch
	samples := make([]float64, n)
	for i := range samples {
		var s float64
		for c := 0; c < ch; c++ {
			s += float64(buf.Data[i*ch+c])
		}
		samples[i] = s / float64(ch) / scale
	}
	return &Clip{Samples: samples, SampleRate: buf.Format.SampleRate}, nil
}

func (c *Clip) Duration() float64 {
	return float64(len(c.Samples)) / float64(c.SampleRate)
}

func (c *Clip) Peak() float64 {
	var p float64
	for _, s := range c.Samples {
		if s < 0 {
			s = -s
		}
		if s > p {
			p = s
		}
	}
	return p
}
