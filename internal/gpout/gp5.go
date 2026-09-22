package gpout

type Score struct {
	Title  string
	Tempo  int
	Tuning []int
	Bars   [][]Beat
}

const (
	version   = "FICHIER GUITAR PRO v5.00"
	program   = 33
	fretCount = 24
	trackName = "Bass"
	bassClef  = 35
)

var pageTemplates = []string{
	"%TITLE%",
	"%SUBTITLE%",
	"%ARTIST%",
	"%ALBUM%",
	"Words by %WORDS%",
	"Music by %MUSIC%",
	"Words & Music by %WORDSMUSIC%",
	"Copyright %COPYRIGHT%",
	"All Rights Reserved - International Copyright Secured",
	"Page %N%/%P%",
}

var trackTail = []byte{1, 2, 3, 4, 5, 6, 10, 7, 8, 9, 0xdf, 3}

func Encode(s Score) []byte {
	var e enc
	e.bstr(version, 30)
	writeInfo(&e, s.Title)
	writeLyrics(&e)
	writePageSetup(&e)
	e.ibstr("")
	e.i32(int32(s.Tempo))
	e.i8(0)
	e.i32(0)
	writeChannels(&e)
	writeDirections(&e)
	e.i32(0)
	e.i32(int32(len(s.Bars)))
	e.i32(1)
	writeMeasureHeaders(&e, len(s.Bars))
	writeTrack(&e, s.Tuning)
	e.pad(2)
	for _, bar := range s.Bars {
		writeMeasure(&e, bar)
	}
	return e.Bytes()
}

func writeInfo(e *enc, title string) {
	e.ibstr(title)
	for i := 0; i < 8; i++ {
		e.ibstr("")
	}
	e.i32(0)
}

func writeLyrics(e *enc) {
	e.i32(0)
	for i := 0; i < 5; i++ {
		e.i32(1)
		e.istr("")
	}
}

func writePageSetup(e *enc) {
	for _, v := range []int32{210, 297, 10, 10, 15, 10, 100} {
		e.i32(v)
	}
	e.i16(0x01ff)
	for _, t := range pageTemplates {
		e.ibstr(t)
	}
}

func writeChannels(e *enc) {
	for i := 0; i < 64; i++ {
		var instrument int32 = 25
		switch i {
		case 0:
			instrument = program
		case 9, 25, 41, 57:
			instrument = 0
		}
		e.i32(instrument)
		e.u8(13)
		e.u8(8)
		e.pad(6)
	}
}

func writeDirections(e *enc) {
	for i := 0; i < 19; i++ {
		e.i16(-1)
	}
}

func writeMeasureHeaders(e *enc, n int) {
	e.Write([]byte{0x43, 4, 4, 0, 0, 2, 2, 2, 2, 0, 0})
	for i := 1; i < n; i++ {
		e.pad(4)
	}
}

func writeTrack(e *enc, tuning []int) {
	e.u8(0)
	e.u8(0x08)
	e.bstr(trackName, 40)
	e.i32(int32(len(tuning)))
	for i := 0; i < 7; i++ {
		var v int32
		if i < len(tuning) {
			v = int32(tuning[len(tuning)-1-i])
		}
		e.i32(v)
	}
	e.i32(1)
	e.i32(1)
	e.i32(2)
	e.i32(fretCount)
	e.i32(0)
	e.Write([]byte{0xff, 0, 0, 0})
	e.i16(0x0143)
	e.pad(3)
	var clef int32
	if len(tuning) > 0 && tuning[0] < bassClef {
		clef = 12
	}
	e.i32(clef)
	e.i32(clef)
	e.i32(100)
	e.Write(trackTail)
	for i := 0; i < 15; i++ {
		e.u8(0xff)
	}
}

func writeMeasure(e *enc, bar []Beat) {
	e.i32(int32(len(bar)))
	for _, b := range bar {
		writeBeat(e, b)
	}
	e.i32(1)
	e.Write([]byte{0x40, 0, 0, 0, 0, 0})
	e.u8(0)
}

func writeBeat(e *enc, b Beat) {
	flags := byte(0)
	if b.Dotted {
		flags |= 0x01
	}
	if b.Rest {
		flags |= 0x40
	}
	e.u8(flags)
	if b.Rest {
		e.u8(2)
	}
	e.i8(b.Dur)
	if b.Rest {
		e.u8(0)
	} else {
		e.u8(1 << (7 - b.String))
		writeNote(e, b)
	}
	e.i16(0)
}

func writeNote(e *enc, b Beat) {
	e.u8(0x30)
	if b.Tied {
		e.u8(2)
	} else {
		e.u8(1)
	}
	e.i8(6)
	e.u8(byte(b.Fret))
	e.u8(0)
}
