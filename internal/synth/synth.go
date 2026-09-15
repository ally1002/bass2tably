package synth

import "math"

type Event struct {
	Midi  int
	Beats float64
}

var TestRiff = []Event{
	{33, 0.5}, {33, 0.5}, {36, 1}, {38, 1},
	{40, 0.5}, {38, 0.5}, {33, 1}, {28, 1}, {23, 1},
}

func RenderLead(events []Event, bpm float64, sr int, leadSec float64) []float64 {
	lead := make([]float64, int(leadSec*float64(sr)))
	return append(lead, Render(events, bpm, sr)...)
}

func Render(events []Event, bpm float64, sr int) []float64 {
	var beats float64
	for _, e := range events {
		beats += e.Beats
	}
	total := int(math.Round(beats * 60 / bpm * float64(sr)))
	out := make([]float64, total)

	var pos float64
	for _, e := range events {
		start := int(math.Round(pos * float64(sr)))
		dur := e.Beats * 60 / bpm
		n := int(math.Round(dur * float64(sr)))
		if start+n > total {
			n = total - start
		}
		addNote(out, start, n, e.Midi, sr)
		pos += dur
	}
	return out
}

func addNote(out []float64, start, n, midiNote, sampleRate int) {
	freq := 440 * math.Pow(2, float64(midiNote-69)/12.0)
	for i := 0; i < n; i++ {
		t := float64(i) / float64(sampleRate)
		env := math.Exp(-3*t) * math.Min(1, t*200)
		var v float64
		for h := 1; h <= 5; h++ {
			v += math.Sin(2*math.Pi*freq*float64(h)*t) / float64(h)
		}
		out[start+i] += 0.4 * env * v
	}
}
