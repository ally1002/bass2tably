package music

import "math"

type Note struct {
	Midi     int
	StartSec float64
	DurSec   float64
	Slot     int
	Len      int
}

func Quantize(notes []Note, bpm float64, gridDiv int) {
	if len(notes) == 0 {
		return
	}
	slotSec := 60.0 / bpm / float64(gridDiv)
	t0 := notes[0].StartSec
	for i := range notes {
		notes[i].Slot = int(math.Round((notes[i].StartSec - t0) / slotSec))
		l := int(math.Round(notes[i].DurSec / slotSec))
		if l < 1 {
			l = 1
		}
		notes[i].Len = l
	}
	for i := 0; i+1 < len(notes); i++ {
		gap := notes[i+1].Slot - notes[i].Slot
		if notes[i].Len > gap {
			notes[i].Len = gap
			if notes[i].Len < 1 {
				notes[i].Len = 1
			}
		}
	}
}
