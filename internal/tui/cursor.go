package tui

import (
	"math"
	"strings"

	"charm.land/lipgloss/v2"

	"bass2tably/internal/app"
	"bass2tably/internal/tab"
)

var cursorStyle = lipgloss.NewStyle().Reverse(true)

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

func highlightColumn(tabStr string, l tab.Layout, block, col int, st lipgloss.Style) string {
	lines := strings.Split(tabStr, "\n")
	start := block * (l.Rows + 1)
	for r := start; r < start+l.Rows && r < len(lines); r++ {
		line := lines[r]
		if col < 0 || col+l.CellW > len(line) {
			continue
		}
		lines[r] = line[:col] + st.Render(line[col:col+l.CellW]) + line[col+l.CellW:]
	}
	return strings.Join(lines, "\n")
}
