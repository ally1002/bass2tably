package gpout

import (
	"bytes"
	"testing"
)

func TestPrimitives(t *testing.T) {
	var e enc
	e.ibstr("Bass")
	if want := []byte{5, 0, 0, 0, 4, 'B', 'a', 's', 's'}; !bytes.Equal(e.Bytes(), want) {
		t.Errorf("ibstr: got % x, want % x", e.Bytes(), want)
	}

	e.Reset()
	e.bstr(version, 30)
	got := e.Bytes()
	if len(got) != 31 || got[0] != 0x18 || !bytes.Equal(got[25:], make([]byte, 6)) {
		t.Errorf("bstr: got % x", got)
	}

	e.Reset()
	e.i32(-1)
	e.i16(-1)
	if want := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}; !bytes.Equal(e.Bytes(), want) {
		t.Errorf("i32/i16: got % x", e.Bytes())
	}

	e.Reset()
	e.istr("ab")
	if want := []byte{2, 0, 0, 0, 'a', 'b'}; !bytes.Equal(e.Bytes(), want) {
		t.Errorf("istr: got % x", e.Bytes())
	}

	e.Reset()
	e.ibstr("Blue Öyster")
	if want := []byte{12, 0, 0, 0, 11, 'B', 'l', 'u', 'e', ' ', '?', 'y', 's', 't', 'e', 'r'}; !bytes.Equal(e.Bytes(), want) {
		t.Errorf("ibstr fold: got % x", e.Bytes())
	}
}
