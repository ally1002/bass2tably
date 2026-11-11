package tui

import (
	"math"

	"github.com/ally1002/bass2tably/internal/app"
)

func cursorSlot(res *app.Result, posSec float64) int {
	if len(res.Notes) == 0 {
		return 0
	}
	slotSec := 60.0 / res.BPM / float64(res.GridDiv)
	t0 := res.Notes[0].StartSec
	slot := int(math.Round((posSec - t0) / slotSec))
	if slot < 0 {
		slot = 0
	}
	if max := res.Layout.Measures*res.Layout.SlotsPerMeasure - 1; slot > max {
		slot = max
	}
	return slot
}

func slotTime(res *app.Result, slot int) float64 {
	if len(res.Notes) == 0 {
		return 0
	}
	slotSec := 60.0 / res.BPM / float64(res.GridDiv)
	return res.Notes[0].StartSec + float64(slot)*slotSec
}
