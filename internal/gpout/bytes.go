package gpout

import (
	"bytes"
	"encoding/binary"
	"strings"
)

type enc struct{ bytes.Buffer }

func (e *enc) u8(v byte)   { e.WriteByte(v) }
func (e *enc) i8(v int8)   { e.WriteByte(byte(v)) }
func (e *enc) i16(v int16) { binary.Write(e, binary.LittleEndian, v) }
func (e *enc) i32(v int32) { binary.Write(e, binary.LittleEndian, v) }
func (e *enc) pad(n int)   { e.Write(make([]byte, n)) }

func (e *enc) bstr(s string, size int) {
	s = ascii(s, size)
	e.u8(byte(len(s)))
	e.WriteString(s)
	e.pad(size - len(s))
}

func (e *enc) ibstr(s string) {
	s = ascii(s, 255)
	e.i32(int32(len(s) + 1))
	e.u8(byte(len(s)))
	e.WriteString(s)
}

func (e *enc) istr(s string) {
	s = ascii(s, 1<<31-1)
	e.i32(int32(len(s)))
	e.WriteString(s)
}

func ascii(s string, max int) string {
	var b strings.Builder
	for _, r := range s {
		if b.Len() == max {
			break
		}
		if r > 127 {
			r = '?'
		}
		b.WriteRune(r)
	}
	return b.String()
}
