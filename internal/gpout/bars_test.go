package gpout_test

import (
	"fmt"
	"math/bits"
	"math/rand"
	"strings"
	"testing"

	"github.com/ally1002/bass2tably/internal/gpout"
	"github.com/ally1002/bass2tably/internal/music"
	"github.com/ally1002/bass2tably/internal/tab"
)

const nStrings = 5

var durLetter = map[int8]string{-2: "w", -1: "h", 0: "q", 1: "e", 2: "s"}

func render(bars [][]gpout.Beat) string {
	var parts []string
	for _, bar := range bars {
		var bs []string
		for _, b := range bar {
			dur := durLetter[b.Dur]
			if b.Dotted {
				dur += "."
			}
			if b.Rest {
				bs = append(bs, "r:"+dur)
				continue
			}
			head := fmt.Sprintf("%d/%d:%s", b.Fret, b.String, dur)
			if b.Tied {
				head = "~" + head
			}
			bs = append(bs, head)
		}
		parts = append(parts, strings.Join(bs, " "))
	}
	return strings.Join(parts, " | ")
}

func beatSlots(b gpout.Beat, gridDiv int) int {
	n := 1 << (bits.Len(uint(gridDiv)) - 1 - int(b.Dur))
	if b.Dotted {
		n += n / 2
	}
	return n
}

type in struct {
	slot, len, str, fret int
}

func build(ins []in) ([]music.Note, []tab.Fretting) {
	notes := make([]music.Note, len(ins))
	frets := make([]tab.Fretting, len(ins))
	for i, x := range ins {
		notes[i] = music.Note{Slot: x.slot, Len: x.len}
		frets[i] = tab.Fretting{String: x.str, Fret: x.fret}
	}
	return notes, frets
}

func TestBars(t *testing.T) {
	cases := []struct {
		name    string
		gridDiv int
		ins     []in
		want    string
		rep     gpout.Report
	}{
		{"sixteenth on lowest string", 4, []in{{0, 1, 0, 0}},
			"0/5:s r:s r:e r:q r:h", gpout.Report{Bars: 1, Notes: 1, Rests: 4}},
		{"tie across the bar line", 4, []in{{14, 4, 3, 3}},
			"r:h. r:e 3/2:e | ~3/2:e r:e r:q r:h", gpout.Report{Bars: 2, Notes: 1, Ties: 1, Rests: 5}},
		{"five sixteenths", 4, []in{{0, 5, 3, 3}},
			"3/2:q ~3/2:s r:s r:e r:h", gpout.Report{Bars: 1, Notes: 1, Ties: 1, Rests: 3}},
		{"dotted quarter", 4, []in{{0, 6, 3, 3}},
			"3/2:q. r:e r:h", gpout.Report{Bars: 1, Notes: 1, Rests: 2}},
		{"quarter plus tied half off the beat", 4, []in{{4, 12, 3, 3}},
			"r:q 3/2:q ~3/2:h", gpout.Report{Bars: 1, Notes: 1, Ties: 1, Rests: 1}},
		{"eighth grid dotted quarter", 2, []in{{0, 3, 3, 3}},
			"3/2:q. r:e r:h", gpout.Report{Bars: 1, Notes: 1, Rests: 2}},
		{"no notes", 4, nil,
			"r:w", gpout.Report{Bars: 1, Rests: 1}},
		{"whole bar note", 4, []in{{0, 16, 3, 3}},
			"3/2:w", gpout.Report{Bars: 1, Notes: 1}},
		{"unplayable becomes a rest", 4, []in{{0, 4, -1, 0}},
			"r:q r:q r:h", gpout.Report{Bars: 1, Rests: 3, Unplayable: 1}},
		{"second note in the same slot is dropped", 4, []in{{0, 1, 3, 3}, {0, 4, 2, 5}},
			"3/2:s r:s r:e r:q r:h", gpout.Report{Bars: 1, Notes: 1, Rests: 4, Dropped: 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			notes, frets := build(c.ins)
			bars, rep := gpout.Bars(notes, frets, nStrings, c.gridDiv)
			if got := render(bars); got != c.want {
				t.Errorf("got  %q\nwant %q", got, c.want)
			}
			if rep != c.rep {
				t.Errorf("report: got %+v, want %+v", rep, c.rep)
			}
		})
	}
}

func TestBarsProperties(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for iter := 0; iter < 200; iter++ {
		gridDiv := 2 << rng.Intn(2)
		spb := 4 * gridDiv
		var ins []in
		slot := rng.Intn(spb)
		for i, n := 0, rng.Intn(12); i < n; i++ {
			slot += rng.Intn(6)
			str := rng.Intn(nStrings+1) - 1
			ins = append(ins, in{slot, 1 + rng.Intn(2*spb), str, rng.Intn(13)})
			if rng.Intn(4) > 0 {
				slot += rng.Intn(spb)
			}
		}
		notes, frets := build(ins)
		bars, rep := gpout.Bars(notes, frets, nStrings, gridDiv)
		if len(bars) != rep.Bars || len(bars) == 0 {
			t.Fatalf("iter %d: %d bars, report %+v", iter, len(bars), rep)
		}
		var prev *gpout.Beat
		for bi, bar := range bars {
			sum := 0
			for i := range bar {
				b := &bar[i]
				sum += beatSlots(*b, gridDiv)
				if b.Tied {
					if prev == nil || prev.Rest || prev.String != b.String || prev.Fret != b.Fret {
						t.Fatalf("iter %d bar %d beat %d: tie without a matching note before it\n%s", iter, bi, i, render(bars))
					}
				}
				prev = b
			}
			if sum != spb {
				t.Fatalf("iter %d bar %d: %d slots, want %d\n%s", iter, bi, sum, spb, render(bars))
			}
		}
		if rep.Notes+rep.Dropped+rep.Unplayable != len(ins) {
			t.Fatalf("iter %d: notes %d + dropped %d + unplayable %d != %d", iter, rep.Notes, rep.Dropped, rep.Unplayable, len(ins))
		}
	}
}
