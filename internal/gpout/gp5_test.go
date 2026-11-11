package gpout_test

import (
	"bytes"
	"testing"

	"github.com/ally1002/bass2tably/internal/app"
	"github.com/ally1002/bass2tably/internal/gpout"
)

func emptyBars(n int) [][]gpout.Beat {
	bars := make([][]gpout.Beat, n)
	for i := range bars {
		bars[i] = []gpout.Beat{{Dur: -2, Rest: true}}
	}
	return bars
}

const (
	offChannels = 31 + 54 + 44 + 259 + 14
	offHeaders  = offChannels + 768 + 42 + 8
	offTrack    = offHeaders + 11 + 3*4
	offMeasures = offTrack + 143 + 2
)

var emptyMeasure = []byte{
	1, 0, 0, 0, 0x40, 0x02, 0xfe, 0, 0, 0,
	1, 0, 0, 0, 0x40, 0, 0, 0, 0, 0,
	0,
}

func TestEncodeEmptyScore(t *testing.T) {
	got := gpout.Encode(gpout.Score{Title: "empty", Tempo: 100, Tuning: app.Tunings["BEADG"], Bars: emptyBars(4)})

	if len(got) != offMeasures+4*21 {
		t.Fatalf("length: got %d, want %d", len(got), offMeasures+4*21)
	}
	check := func(name string, off int, want []byte) {
		t.Helper()
		if !bytes.Equal(got[off:off+len(want)], want) {
			t.Errorf("%s at %d:\n got  % x\n want % x", name, off, got[off:off+len(want)], want)
		}
	}
	check("version", 0, append([]byte{0x18}, []byte("FICHIER GUITAR PRO v5.00")...))
	check("title", 31, []byte{6, 0, 0, 0, 5, 'e', 'm', 'p', 't', 'y'})
	check("tempo and key", 31+54+44+259, []byte{1, 0, 0, 0, 0, 100, 0, 0, 0, 0, 0, 0, 0, 0})
	check("channel 0", offChannels, []byte{33, 0, 0, 0, 13, 8, 0, 0, 0, 0, 0, 0})
	check("channel 1", offChannels+12, []byte{25, 0, 0, 0, 13, 8, 0, 0, 0, 0, 0, 0})
	check("channel 9", offChannels+9*12, []byte{0, 0, 0, 0, 13, 8, 0, 0, 0, 0, 0, 0})
	check("directions and counts", offChannels+768, append(bytes.Repeat([]byte{0xff}, 38), 0, 0, 0, 0, 4, 0, 0, 0, 1, 0, 0, 0))
	check("header 1", offHeaders, []byte{0x43, 4, 4, 0, 0, 2, 2, 2, 2, 0, 0})
	check("headers 2-4", offHeaders+11, make([]byte, 12))
	check("track head", offTrack, append([]byte{0, 0x08, 4, 'B', 'a', 's', 's'}, make([]byte, 36)...))
	check("track body", offTrack+43, []byte{
		5, 0, 0, 0,
		43, 0, 0, 0, 38, 0, 0, 0, 33, 0, 0, 0, 28, 0, 0, 0, 23, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		1, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 24, 0, 0, 0, 0, 0, 0, 0,
		0xff, 0, 0, 0,
		0x43, 0x01, 0, 0, 0,
		12, 0, 0, 0, 12, 0, 0, 0, 100, 0, 0, 0,
		1, 2, 3, 4, 5, 6, 10, 7, 8, 9, 0xdf, 3,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	})
	check("padding", offTrack+143, []byte{0, 0})
	for i := 0; i < 4; i++ {
		check("measure", offMeasures+i*21, emptyMeasure)
	}
}

func TestEncodeTunings(t *testing.T) {
	cases := []struct {
		name string
		want []byte
	}{
		{"EADG", []byte{4, 0, 0, 0, 43, 0, 0, 0, 38, 0, 0, 0, 33, 0, 0, 0, 28, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"BEAD", []byte{4, 0, 0, 0, 38, 0, 0, 0, 33, 0, 0, 0, 28, 0, 0, 0, 23, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"DADG", []byte{4, 0, 0, 0, 43, 0, 0, 0, 38, 0, 0, 0, 33, 0, 0, 0, 26, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
	}
	for _, c := range cases {
		got := gpout.Encode(gpout.Score{Title: "t", Tempo: 90, Tuning: app.Tunings[c.name], Bars: emptyBars(1)})
		if len(got) != offMeasures-16+21 {
			t.Errorf("%s: length %d, want %d", c.name, len(got), offMeasures-16+21)
		}
		off := offTrack - 16 + 43
		if !bytes.Equal(got[off:off+32], c.want) {
			t.Errorf("%s strings:\n got  % x\n want % x", c.name, got[off:off+32], c.want)
		}
		clef := got[off+61 : off+65]
		if !bytes.Equal(clef, []byte{12, 0, 0, 0}) {
			t.Errorf("%s clef: got % x", c.name, clef)
		}
	}
}
