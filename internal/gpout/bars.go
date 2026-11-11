package gpout

import (
	"math/bits"

	"github.com/ally1002/bass2tably/internal/music"
	"github.com/ally1002/bass2tably/internal/tab"
)

type Beat struct {
	Dur    int8
	Dotted bool
	Rest   bool
	String int
	Fret   int
	Tied   bool
}

type Report struct {
	Bars, Notes, Ties, Rests, Unplayable, Dropped int
}

type span struct {
	start, end int
	rest       bool
	str, fret  int
}

func spans(notes []music.Note, frets []tab.Fretting, nStrings, spb int) ([]span, Report) {
	var out []span
	var rep Report
	cursor := 0
	for i, n := range notes {
		if n.Slot < cursor {
			rep.Dropped++
			continue
		}
		if n.Slot > cursor {
			out = append(out, span{start: cursor, end: n.Slot, rest: true})
		}
		s := span{start: n.Slot, end: n.Slot + n.Len}
		if f := frets[i]; f.String < 0 {
			s.rest = true
			rep.Unplayable++
		} else {
			s.str, s.fret = nStrings-f.String, f.Fret
		}
		out = append(out, s)
		cursor = s.end
	}
	if pad := (spb - cursor%spb) % spb; pad > 0 || cursor == 0 {
		if cursor == 0 {
			pad = spb
		}
		out = append(out, span{start: cursor, end: cursor + pad, rest: true})
	}
	return out, rep
}

func tokens(p, l int) []int {
	var out []int
	for l > 0 {
		k := 1
		for k*2 <= l && p%(k*2) == 0 {
			k *= 2
		}
		out = append(out, k)
		p += k
		l -= k
	}
	return out
}

type token struct {
	base   int
	dotted bool
}

func (t token) slots() int {
	if t.dotted {
		return t.base + t.base/2
	}
	return t.base
}

func dot(ts []int) []token {
	var out []token
	for i := 0; i < len(ts); i++ {
		if i+1 < len(ts) && ts[i+1]*2 == ts[i] {
			out = append(out, token{base: ts[i], dotted: true})
			i++
			continue
		}
		out = append(out, token{base: ts[i]})
	}
	return out
}

func durByte(base, gridDiv int) int8 {
	return int8(bits.Len(uint(gridDiv)) - bits.Len(uint(base)))
}

func Bars(notes []music.Note, frets []tab.Fretting, nStrings, gridDiv int) ([][]Beat, Report) {
	spb := 4 * gridDiv
	sp, rep := spans(notes, frets, nStrings, spb)
	nBars := sp[len(sp)-1].end / spb
	rep.Bars = nBars
	bars := make([][]Beat, nBars)
	for _, s := range sp {
		tied := false
		for pos := s.start; pos < s.end; {
			bar := pos / spb
			p := pos % spb
			l := min(s.end, (bar+1)*spb) - pos
			for _, t := range dot(tokens(p, l)) {
				b := Beat{
					Dur:    durByte(t.base, gridDiv),
					Dotted: t.dotted,
					Rest:   s.rest,
					String: s.str,
					Fret:   s.fret,
					Tied:   tied && !s.rest,
				}
				switch {
				case b.Rest:
					rep.Rests++
				case b.Tied:
					rep.Ties++
				default:
					rep.Notes++
				}
				bars[bar] = append(bars[bar], b)
				tied = true
				pos += t.slots()
			}
		}
	}
	return bars, rep
}
