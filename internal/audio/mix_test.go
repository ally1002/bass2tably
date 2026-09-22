package audio

import (
	"math"
	"testing"
)

func ramp(n int) []float64 {
	s := make([]float64, n)
	for i := range s {
		s[i] = float64(i) / float64(n)
	}
	return s
}

func TestMixLoopWrapsExactly(t *testing.T) {
	samples := ramp(1000)
	out := make([]float32, 200)
	p := &params{speed: 1, loopA: 100, loopB: 200}
	pos, frac := mix(out, samples, 150, 0, p)
	if pos != 150 || frac != 0 {
		t.Fatalf("pos %d frac %f after two laps, want 150 and 0", pos, frac)
	}
	if again, _ := mix(make([]float32, 250), samples, 150, 0, p); again != 100 {
		t.Fatalf("pos %d after 250 frames, want 100 (the wrap lands exactly on loopA)", again)
	}
	want := make([]int, 0, 200)
	for i := 150; i < 200; i++ {
		want = append(want, i)
	}
	for i := 100; i < 200; i++ {
		want = append(want, i)
	}
	for i := 100; i < 150; i++ {
		want = append(want, i)
	}
	for i, w := range want {
		if got := out[i]; got != float32(samples[w]) {
			t.Fatalf("frame %d = %f, want sample %d (%f)", i, got, w, samples[w])
		}
	}
}

func TestMixNoLoopClampsAtEnd(t *testing.T) {
	samples := ramp(100)
	out := make([]float32, 150)
	pos, _ := mix(out, samples, 80, 0, &params{speed: 1})
	if pos != 100 {
		t.Fatalf("pos = %d past the end, want clamped to 100", pos)
	}
	for i := 20; i < 150; i++ {
		if out[i] != 0 {
			t.Fatalf("frame %d = %f past the end, want silence", i, out[i])
		}
	}
	mix(out, samples, 100, 0, &params{speed: 1, loopA: 10, loopB: 100})
	if out[0] != float32(samples[10]) {
		t.Fatalf("starting on loopB did not wrap to loopA: %f", out[0])
	}
}

func TestMixHalfSpeedConsumesHalf(t *testing.T) {
	samples := ramp(1000)
	out := make([]float32, 256)
	pos, frac := mix(out, samples, 0, 0, &params{speed: 0.5})
	if pos != 128 || frac != 0 {
		t.Fatalf("256 frames at 0.5×: pos %d frac %f, want 128 and 0", pos, frac)
	}
	pos, frac = mix(out[:255], samples, 0, 0, &params{speed: 0.5})
	if pos != 127 || frac != 0.5 {
		t.Fatalf("255 frames at 0.5×: pos %d frac %f, want 127 and 0.5", pos, frac)
	}
	pos, frac = mix(out[:100], samples, pos, frac, &params{speed: 0.5})
	if pos != 177 || frac != 0.5 {
		t.Fatalf("carried frac: pos %d frac %f, want 177 and 0.5", pos, frac)
	}
	pos, _ = mix(out, samples, 0, 0, &params{speed: 0.75})
	if pos != 192 {
		t.Fatalf("256 frames at 0.75×: pos %d, want 192", pos)
	}
}

func TestMixLinearInterpolation(t *testing.T) {
	samples := ramp(1000)
	out := make([]float32, 8)
	mix(out, samples, 0, 0, &params{speed: 0.5})
	for i, got := range out {
		want := float32(float64(i) / 2 / 1000)
		if math.Abs(float64(got-want)) > 1e-6 {
			t.Fatalf("frame %d = %f, want the midpoint %f", i, got, want)
		}
	}
}

func TestClickOnBeatsOnly(t *testing.T) {
	const rate = 48000
	c := newClick(rate, 120, 0.25, 4)
	if c.period != rate/2 || c.phase != rate/4 {
		t.Fatalf("period %d phase %d, want %d and %d", c.period, c.phase, rate/2, rate/4)
	}
	if len(c.normal) != rate/100 || len(c.accent) != rate/100 {
		t.Fatalf("buffers are %d and %d frames, want 10 ms = %d", len(c.normal), len(c.accent), rate/100)
	}
	for pos := int64(0); pos < c.phase; pos += 97 {
		if c.at(pos) != 0 {
			t.Fatalf("click before the phase at %d", pos)
		}
	}
	for beat := int64(0); beat < 8; beat++ {
		start := c.phase + beat*c.period
		buf := c.normal
		if beat%4 == 0 {
			buf = c.accent
		}
		for off := int64(0); off < int64(len(buf)); off++ {
			if got := c.at(start + off); got != buf[off] {
				t.Fatalf("beat %d offset %d: %f, want %f", beat, off, got, buf[off])
			}
		}
		for off := int64(len(buf)); off < c.period; off += 331 {
			if c.at(start+off) != 0 {
				t.Fatalf("beat %d: click still sounding at offset %d", beat, off)
			}
		}
	}
	if c.normal[1] == c.accent[1] {
		t.Fatal("normal and accent clicks are identical")
	}
	var peak float32
	for _, v := range c.accent {
		peak = max(peak, float32(math.Abs(float64(v))))
	}
	if peak > clickAmplitude || peak < clickAmplitude/4 {
		t.Fatalf("accent peak %f, want within (%.2f, %.2f]", peak, clickAmplitude/4, clickAmplitude)
	}
	var nilClick *clickTrack
	if nilClick.at(123) != 0 {
		t.Fatal("a nil click made a sound")
	}
}

func TestClickAlignsWithFirstOnset(t *testing.T) {
	c := newClick(44100, 124.5, 1.2345, 4)
	if want := int64(math.Round(1.2345 * 44100)); c.phase != want {
		t.Fatalf("phase %d, want %d", c.phase, want)
	}
	if want := int64(math.Round(60 / 124.5 * 44100)); c.period != want {
		t.Fatalf("period %d, want %d", c.period, want)
	}
	out := make([]float32, 100)
	silence := make([]float64, 100000)
	mix(out, silence, c.phase, 0, &params{speed: 1, click: c})
	if out[1] != c.accent[1] {
		t.Fatalf("mix did not place the accent click at the phase: %f vs %f", out[1], c.accent[1])
	}
	if newClick(44100, 0, 0, 4) != nil || newClick(0, 120, 0, 4) != nil {
		t.Fatal("an impossible click was built")
	}
}
