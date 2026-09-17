package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"bass2tably/internal/app"
	"bass2tably/internal/dsp"
	"bass2tably/internal/music"
	"bass2tably/internal/wavio"
)

func TestRealRecordings(t *testing.T) {
	wavs, err := filepath.Glob(filepath.Join("testdata", "*.wav"))
	if err != nil {
		t.Fatal(err)
	}
	if len(wavs) == 0 {
		t.Skip("no real-recording fixtures in testdata/")
	}
	for _, wavPath := range wavs {
		t.Run(strings.TrimSuffix(filepath.Base(wavPath), ".wav"), func(t *testing.T) {
			data, err := os.ReadFile(strings.TrimSuffix(wavPath, ".wav") + ".expected")
			if err != nil {
				t.Fatal(err)
			}
			var want []int
			for _, line := range strings.Split(string(data), "\n") {
				if i := strings.IndexByte(line, '#'); i >= 0 {
					line = line[:i]
				}
				for _, f := range strings.Fields(line) {
					v, err := strconv.Atoi(f)
					if err != nil {
						t.Fatalf("bad .expected value %q: %v", f, err)
					}
					want = append(want, v)
				}
			}

			clip, err := wavio.LoadAny(wavPath)
			if err != nil {
				t.Fatal(err)
			}
			rng := dsp.RangeForTuning(app.Tunings["BEADG"])
			var got []int
			for _, seg := range dsp.Segments(clip.Samples, clip.SampleRate) {
				if freq, ok := dsp.NotePitchIn(clip.Samples, clip.SampleRate, seg, rng); ok {
					got = append(got, music.MidiFromFreq(freq))
				}
			}

			if len(got) != len(want) {
				t.Errorf("got %d pitched notes, want %d", len(got), len(want))
			}
			for _, o := range diffMidi(got, want) {
				switch o.kind {
				case 'x':
					t.Errorf("note %d: got %d (%s), want %d (%s)",
						o.at, o.got, music.MidiName(o.got), o.want, music.MidiName(o.want))
				case '-':
					t.Errorf("note %d: missing, want %d (%s)",
						o.at, o.want, music.MidiName(o.want))
				case '+':
					t.Errorf("note %d: extra %d (%s)",
						o.at, o.got, music.MidiName(o.got))
				}
			}
		})
	}
}

type op struct {
	kind      byte
	got, want int
	at        int
}

func diffMidi(got, want []int) []op {
	n, m := len(got), len(want)
	d := make([][]int, n+1)
	for i := range d {
		d[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		d[i][m] = n - i
	}
	for j := m - 1; j >= 0; j-- {
		d[n][j] = m - j
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			sub := d[i+1][j+1]
			if got[i] != want[j] {
				sub++
			}
			best := sub
			if v := d[i][j+1] + 1; v < best {
				best = v
			}
			if v := d[i+1][j] + 1; v < best {
				best = v
			}
			d[i][j] = best
		}
	}

	var ops []op
	i, j := 0, 0
	for i < n && j < m {
		sub := d[i+1][j+1]
		if got[i] != want[j] {
			sub++
		}
		switch {
		case got[i] == want[j] && sub == d[i][j]:
			ops = append(ops, op{'=', got[i], want[j], j})
			i, j = i+1, j+1
		case d[i][j+1]+1 == d[i][j]:
			ops = append(ops, op{'-', -1, want[j], j})
			j++
		case sub == d[i][j]:
			ops = append(ops, op{'x', got[i], want[j], j})
			i, j = i+1, j+1
		default:
			ops = append(ops, op{'+', got[i], -1, j})
			i++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, op{'+', got[i], -1, m})
	}
	for ; j < m; j++ {
		ops = append(ops, op{'-', -1, want[j], j})
	}
	return ops
}

func TestDiffMidi(t *testing.T) {
	fmtOps := func(ops []op) string {
		var b strings.Builder
		for _, o := range ops {
			if o.kind == '=' {
				continue
			}
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			switch o.kind {
			case 'x':
				b.WriteString(strconv.Itoa(o.at) + ":x" + strconv.Itoa(o.got) + "/" + strconv.Itoa(o.want))
			case '-':
				b.WriteString(strconv.Itoa(o.at) + ":-" + strconv.Itoa(o.want))
			case '+':
				b.WriteString(strconv.Itoa(o.at) + ":+" + strconv.Itoa(o.got))
			}
		}
		return b.String()
	}

	cases := []struct {
		name string
		got  []int
		want []int
		ops  string
	}{
		{"identical", []int{28, 31, 33}, []int{28, 31, 33}, ""},
		{"substitution", []int{28, 24, 33}, []int{28, 23, 33}, "1:x24/23"},
		{"missing", []int{28, 33}, []int{28, 31, 33}, "1:-31"},
		{"extra", []int{28, 31, 33}, []int{28, 33}, "1:+31"},
		{"empty got", []int{}, []int{28, 31}, "0:-28 1:-31"},
		{
			"substitution after a run of missing notes",
			[]int{31, 31, 24, 26},
			[]int{31, 31, 31, 31, 31, 23, 26},
			"2:-31 3:-31 4:-31 5:x24/23",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := fmtOps(diffMidi(c.got, c.want)); got != c.ops {
				t.Errorf("diffMidi = %q, want %q", got, c.ops)
			}
		})
	}
}

func renderResult(res *app.Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "notes: %d\n", len(res.Notes)+len(res.Skipped))
	fmt.Fprintf(&b, "bpm: %.1f\n", res.BPM)
	for _, s := range res.Skipped {
		fmt.Fprintf(&b, "skipped #%d %.3fs %.3fs\n", s.Index+1, s.StartSec, s.DurSec)
	}
	b.WriteString(res.Tab)
	for _, line := range res.NoteLines() {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func TestGoldenShh(t *testing.T) {
	res, err := app.Analyze(filepath.Join("testdata", "shh.wav"), app.Options{Tuning: "BEADG", Grid: 16})
	if err != nil {
		t.Fatal(err)
	}
	got := renderResult(res)
	want, err := os.ReadFile(filepath.Join("testdata", "shh.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("shh output differs from testdata/shh.golden:\n%s", unifiedDiff(string(want), got))
	}
}

func unifiedDiff(want, got string) string {
	ids := map[string]int{}
	var lines []string
	intern := func(s string) []int {
		var out []int
		for _, l := range strings.Split(strings.TrimSuffix(s, "\n"), "\n") {
			id, ok := ids[l]
			if !ok {
				id = len(lines)
				ids[l] = id
				lines = append(lines, l)
			}
			out = append(out, id)
		}
		return out
	}
	w, g := intern(want), intern(got)

	var b strings.Builder
	for _, o := range diffMidi(g, w) {
		switch o.kind {
		case '=':
			fmt.Fprintf(&b, "      %s\n", lines[o.want])
		case '-':
			fmt.Fprintf(&b, "-%-4d %s\n", o.at+1, lines[o.want])
		case '+':
			fmt.Fprintf(&b, "+%-4d %s\n", o.at+1, lines[o.got])
		case 'x':
			fmt.Fprintf(&b, "-%-4d %s\n", o.at+1, lines[o.want])
			fmt.Fprintf(&b, "+%-4d %s\n", o.at+1, lines[o.got])
		}
	}
	return b.String()
}

func TestUnifiedDiff(t *testing.T) {
	got := unifiedDiff("a\nb\nc\n", "a\nx\nc\nd\n")
	want := "      a\n-2    b\n+2    x\n      c\n+4    d\n"
	if got != want {
		t.Errorf("unifiedDiff =\n%s\nwant\n%s", got, want)
	}
}
