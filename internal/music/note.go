package music

import (
	"fmt"
	"math"
)

var noteNames = [12]string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

func MidiFromFreq(freq float64) int {
	return int(math.Round(69 + 12*math.Log2(freq/440)))
}

func MidiName(midi int) string {
	return fmt.Sprintf("%s%d", noteNames[((midi%12)+12)%12], midi/12-1)
}
