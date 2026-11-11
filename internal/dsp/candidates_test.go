package dsp

import (
	"math"
	"testing"

	"github.com/ally1002/bass2tably/internal/wavio"
)

func stretchedPartials(sr, n int, f0, b float64, amps []float64) []float64 {
	x := make([]float64, n)
	for i := range x {
		t := float64(i) / float64(sr)
		for k, a := range amps {
			h := float64(k + 2)
			x[i] += a * math.Sin(2*math.Pi*h*f0*math.Sqrt(1+b*h*h)*t)
		}
	}
	return x
}

func hasCandidateNear(cs []yinCandidate, hz, cents float64) (yinCandidate, bool) {
	for _, c := range cs {
		if math.Abs(1200*math.Log2(c.Hz/hz)) <= cents {
			return c, true
		}
	}
	return yinCandidate{}, false
}

func segmentNear(segs []Segment, sr int, sec float64) Segment {
	best := segs[0]
	for _, s := range segs {
		if math.Abs(float64(s.Start)/float64(sr)-sec) < math.Abs(float64(best.Start)/float64(sr)-sec) {
			best = s
		}
	}
	return best
}

func TestYinCandidatesSawtooth(t *testing.T) {
	const sr = 44100
	win, _ := yinWindow(sr, yinFmin)
	x := make([]float64, win)
	for i := range x {
		for h := 1; h <= 20; h++ {
			x[i] += math.Sin(2*math.Pi*55*float64(h)*float64(i)/sr) / float64(h)
		}
	}
	cs := yinCandidates(x, sr, 20, yinFmax)
	c55, ok := hasCandidateNear(cs, 55, 10)
	if !ok || c55.Depth > 0.01 {
		t.Fatalf("55 Hz sawtooth: want a candidate at 55 with depth ~0, got %v", cs)
	}
	if _, ok := hasCandidateNear(cs, 27.5, 10); !ok {
		t.Errorf("55 Hz sawtooth: want the sub-octave 27.5 as a candidate, got %v", cs)
	}
	for _, c := range cs {
		if c.Depth < c55.Depth {
			t.Errorf("candidate %.2f Hz at depth %.3f is shallower than the fundamental's %.3f", c.Hz, c.Depth, c55.Depth)
		}
	}
}

func TestYinCandidatesKeepWhatTheThresholdScanSkips(t *testing.T) {
	const sr = 44100
	const f0 = 30.87
	win, _ := yinWindow(sr, yinFmin)
	x := stretchedPartials(sr, win, f0, 1.5e-4, []float64{1, .3, .8, .2, .7, .2, .5})

	got, ok := Yin(x, sr, yinFmin, yinFmax, yinThreshold)
	if !ok || math.Abs(1200*math.Log2(got/(2*f0))) > 50 {
		t.Fatalf("Yin = %.2f Hz, %v; this fixture is built so the threshold scan stops at the octave %.2f", got, ok, 2*f0)
	}
	cs := yinCandidates(x, sr, yinFmin, yinFmax)
	c, ok := hasCandidateNear(cs, f0, 20)
	if !ok {
		t.Fatalf("candidates %v do not contain the fundamental %.2f", cs, f0)
	}
	if c.Depth > 0.05 {
		t.Errorf("fundamental candidate depth %.3f, want the deepest dip in the frame", c.Depth)
	}
}

func TestYinCandidatesCarryTheOpenB(t *testing.T) {
	clip, err := wavio.LoadAny("../../testdata/four-bar-repeats.wav")
	if err != nil {
		t.Fatal(err)
	}
	const truth = 30.87
	x, sr := clip.Samples, clip.SampleRate
	seg := segmentNear(Segments(x, sr), sr, 12.434)
	frames := segFrames(x, sr, seg, DefaultRange)
	if len(frames) < 5 {
		t.Fatalf("segment at %.3f s has %d frames, want a sustained note", float64(seg.Start)/float64(sr), len(frames))
	}
	for i, fr := range frames {
		cs := yinCandidates(fr, sr, yinFmin, yinFmax)
		if _, ok := hasCandidateNear(cs, truth, 30); !ok {
			t.Errorf("frame %d: candidates %v do not contain the open B %.2f", i, cs, truth)
		}
		if f, ok := Yin(fr, sr, yinFmin, yinFmax, yinThreshold); ok && math.Abs(1200*math.Log2(f/truth)) < 50 {
			t.Errorf("frame %d: Yin returned %.2f; the threshold scan was measured never to find the open B on this note", i, f)
		}
	}
}

func TestYinCandidatesOnSilenceAndGaps(t *testing.T) {
	const sr = 44100
	win, _ := yinWindow(sr, yinFmin)
	if cs := yinCandidates(make([]float64, win), sr, yinFmin, yinFmax); len(cs) != 0 {
		t.Errorf("silence: got candidates %v, want none", cs)
	}

	clip, err := wavio.LoadAny("../../testdata/looped-phrase.wav")
	if err != nil {
		t.Fatal(err)
	}
	x := clip.Samples
	for _, sec := range []float64{3.030, 11.029, 19.029, 27.028} {
		seg := segmentNear(Segments(x, clip.SampleRate), clip.SampleRate, sec)
		for _, fr := range segFrames(x, clip.SampleRate, seg, DefaultRange) {
			for _, c := range yinCandidates(fr, clip.SampleRate, yinFmin, yinFmax) {
				if c.Depth < yinThreshold {
					t.Errorf("gap at %.3f s: candidate %.2f Hz at depth %.3f would voice a rest", sec, c.Hz, c.Depth)
				}
			}
		}
	}
}
