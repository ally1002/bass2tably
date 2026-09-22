package gpout_test

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"bass2tably/internal/app"
	"bass2tably/internal/gpout"
)

func hx(s string) []byte {
	b, err := hex.DecodeString(strings.NewReplacer(" ", "", "\n", "").Replace(s))
	if err != nil {
		panic(err)
	}
	return b
}

func measure(beats ...string) []byte {
	out := []byte{byte(len(beats)), 0, 0, 0}
	for _, b := range beats {
		out = append(out, hx(b)...)
	}
	return append(out, hx("01 00 00 00 40 00 00 00 00 00 00")...)
}

const (
	restS = "40 02 02 00 00 00"
	restE = "40 02 01 00 00 00"
	restQ = "40 02 00 00 00 00"
	restH = "40 02 ff 00 00 00"
	restW = "40 02 fe 00 00 00"
)

func TestEncodeBars(t *testing.T) {
	cases := []struct {
		name    string
		gridDiv int
		ins     []in
		want    [][]byte
	}{
		{"sixteenth on lowest string", 4, []in{{0, 1, 0, 0}}, [][]byte{
			measure("00 02 04 30 01 06 00 00 00 00", restS, restE, restQ, restH)}},
		{"tie across the bar line", 4, []in{{14, 4, 3, 3}}, [][]byte{
			measure("41 02 ff 00 00 00", restE, "00 01 20 30 01 06 03 00 00 00"),
			measure("00 01 20 30 02 06 03 00 00 00", restE, restQ, restH)}},
		{"five sixteenths", 4, []in{{0, 5, 3, 3}}, [][]byte{
			measure("00 00 20 30 01 06 03 00 00 00", "00 02 20 30 02 06 03 00 00 00", restS, restE, restH)}},
		{"dotted quarter", 4, []in{{0, 6, 3, 3}}, [][]byte{
			measure("01 00 20 30 01 06 03 00 00 00", restE, restH)}},
		{"quarter plus tied half off the beat", 4, []in{{4, 12, 3, 3}}, [][]byte{
			measure(restQ, "00 00 20 30 01 06 03 00 00 00", "00 ff 20 30 02 06 03 00 00 00")}},
		{"eighth grid dotted quarter", 2, []in{{0, 3, 3, 3}}, [][]byte{
			measure("01 00 20 30 01 06 03 00 00 00", restE, restH)}},
		{"no notes", 4, nil, [][]byte{measure(restW)}},
		{"whole bar note", 4, []in{{0, 16, 3, 3}}, [][]byte{
			measure("00 fe 20 30 01 06 03 00 00 00")}},
		{"unplayable becomes a rest", 4, []in{{0, 4, -1, 0}}, [][]byte{
			measure(restQ, restQ, restH)}},
		{"second note in the same slot is dropped", 4, []in{{0, 1, 3, 3}, {0, 4, 2, 5}}, [][]byte{
			measure("00 02 20 30 01 06 03 00 00 00", restS, restE, restQ, restH)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			notes, frets := build(c.ins)
			bars, _ := gpout.Bars(notes, frets, nStrings, c.gridDiv)
			got := gpout.Encode(gpout.Score{Title: "t", Tempo: 90, Tuning: app.Tunings["BEADG"], Bars: bars})
			want := bytes.Join(c.want, nil)
			start := 1368 + 4*len(c.want)
			if len(got) != start+len(want) {
				t.Fatalf("length: got %d, want %d", len(got), start+len(want))
			}
			if region := got[start:]; !bytes.Equal(region, want) {
				i := 0
				for i < len(want) && region[i] == want[i] {
					i++
				}
				t.Errorf("measures differ at byte %d (file offset %d):\n got  % x\n want % x", i, start+i, region, want)
			}
		})
	}
}
