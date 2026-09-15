package wavio

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	isFloat := d.WavAudioFormat == 3
	scale := float64(int(1) << (int(d.BitDepth) - 1))
	n := len(buf.Data) / ch
	samples := make([]float64, n)
	for i := range samples {
		var s float64
		for c := 0; c < ch; c++ {
			v := buf.Data[i*ch+c]
			if isFloat {
				s += float64(math.Float32frombits(uint32(int32(v))))
			} else {
				s += float64(v) / scale
			}
		}
		samples[i] = s / float64(ch)
	}
	return &Clip{Samples: samples, SampleRate: buf.Format.SampleRate}, nil
}

func LoadAny(path string) (*Clip, error) {
	if strings.EqualFold(filepath.Ext(path), ".wav") {
		return Load(path)
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("bass2tably-%d.wav", os.Getpid()))
	defer os.Remove(tmp)
	cmd := exec.Command("ffmpeg", "-y", "-loglevel", "error",
		"-i", path, "-ac", "1", "-ar", "44100", "-c:a", "pcm_s16le", tmp)
	if outb, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg decode failed: %v: %s", err, outb)
	}
	return Load(tmp)
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
