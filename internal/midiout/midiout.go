package midiout

import (
	"sort"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/smf"

	"github.com/ally1002/bass2tably/internal/music"
)

const (
	ticksPerQuarter = 960
	program         = 33
	velocity        = 90
)

type event struct {
	tick int
	off  bool
	key  uint8
}

func Write(path string, notes []music.Note, bpm float64, gridDiv int) error {
	ticksPerSlot := ticksPerQuarter / gridDiv

	events := make([]event, 0, 2*len(notes))
	for _, n := range notes {
		key := uint8(n.Midi)
		events = append(events,
			event{tick: n.Slot * ticksPerSlot, key: key},
			event{tick: (n.Slot + n.Len) * ticksPerSlot, off: true, key: key},
		)
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].tick != events[j].tick {
			return events[i].tick < events[j].tick
		}
		return events[i].off && !events[j].off
	})

	s := smf.New()
	s.TimeFormat = smf.MetricTicks(ticksPerQuarter)

	var tr smf.Track
	tr.Add(0, smf.MetaTempo(bpm))
	tr.Add(0, midi.ProgramChange(0, program))
	prev := 0
	for _, e := range events {
		delta := uint32(e.tick - prev)
		prev = e.tick
		if e.off {
			tr.Add(delta, midi.NoteOff(0, e.key))
		} else {
			tr.Add(delta, midi.NoteOn(0, e.key, velocity))
		}
	}
	tr.Close(0)
	if err := s.Add(tr); err != nil {
		return err
	}
	return s.WriteFile(path)
}
