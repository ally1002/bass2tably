package audio

import "math"

type params struct {
	speed        float64
	loopA, loopB int64
	click        *clickTrack
}

type clickTrack struct {
	period, phase  int64
	accentEvery    int64
	normal, accent []float32
}

const (
	clickSec       = 0.010
	clickDecaySec  = 0.003
	clickNormalHz  = 1000.0
	clickAccentHz  = 1500.0
	clickAmplitude = 0.5
)

func newClick(rate int, bpm, offsetSec float64, accentEvery int) *clickTrack {
	if rate <= 0 || bpm <= 0 {
		return nil
	}
	if accentEvery < 1 {
		accentEvery = 1
	}
	c := &clickTrack{
		period:      int64(math.Round(60 / bpm * float64(rate))),
		phase:       int64(math.Round(offsetSec * float64(rate))),
		accentEvery: int64(accentEvery),
	}
	if c.period < 1 {
		return nil
	}
	c.normal = clickBuffer(rate, clickNormalHz)
	c.accent = clickBuffer(rate, clickAccentHz)
	return c
}

func clickBuffer(rate int, hz float64) []float32 {
	n := int(clickSec * float64(rate))
	buf := make([]float32, n)
	for i := range buf {
		t := float64(i) / float64(rate)
		buf[i] = float32(clickAmplitude * math.Sin(2*math.Pi*hz*t) * math.Exp(-t/clickDecaySec))
	}
	return buf
}

func (c *clickTrack) at(pos int64) float32 {
	if c == nil || pos < c.phase {
		return 0
	}
	k := pos - c.phase
	beat, off := k/c.period, k%c.period
	buf := c.normal
	if beat%c.accentEvery == 0 {
		buf = c.accent
	}
	if off < int64(len(buf)) {
		return buf[off]
	}
	return 0
}

func mix(out []float32, samples []float64, pos int64, frac float64, p *params) (int64, float64) {
	total := int64(len(samples))
	looping := p.loopB > p.loopA
	for i := range out {
		if looping && pos >= p.loopB {
			pos = p.loopA + (pos - p.loopB)
		}
		var v float64
		if pos >= 0 && pos < total {
			v = samples[pos]
			if frac > 0 && pos+1 < total {
				v += (samples[pos+1] - v) * frac
			}
		}
		o := float32(v) + p.click.at(pos)
		if o > 1 {
			o = 1
		} else if o < -1 {
			o = -1
		}
		out[i] = o
		frac += p.speed
		step := int64(frac)
		frac -= float64(step)
		pos += step
		if looping && pos >= p.loopB {
			pos = p.loopA + (pos - p.loopB)
		}
	}
	return min(pos, total), frac
}
