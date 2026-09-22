package tui

import (
	"math"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestPeaksNormalizedAndSized(t *testing.T) {
	if got := peaks(nil, 10); got != nil {
		t.Errorf("peaks(nil) = %v, want nil", got)
	}
	if got := peaks([]float64{1}, 0); got != nil {
		t.Errorf("peaks(n=0) = %v, want nil", got)
	}
	samples := make([]float64, 1000)
	for i := range samples {
		samples[i] = 0.25 * math.Sin(float64(i)/7)
	}
	samples[500] = -0.5
	pk := peaks(samples, 40)
	if len(pk) != 40 {
		t.Fatalf("len = %d, want 40", len(pk))
	}
	top := 0.0
	for i, p := range pk {
		if p < 0 || p > 1 {
			t.Errorf("peak %d = %f out of [0, 1]", i, p)
		}
		top = max(top, p)
	}
	if top != 1 {
		t.Errorf("max peak = %f, want 1", top)
	}
	if pk[20] != 1 {
		t.Errorf("bin 20 holding the loudest sample = %f, want 1", pk[20])
	}
}

func TestWaveRowWidth(t *testing.T) {
	st := testStyles()
	samples := make([]float64, 5000)
	for i := range samples {
		samples[i] = math.Sin(float64(i) / 3)
	}
	for _, n := range []int{8, 33, 80, 200} {
		pk := peaks(samples, n)
		for _, frac := range []float64{0, 0.5, 1} {
			for _, ascii := range []bool{false, true} {
				if w := ansi.StringWidth(waveRow(pk, frac, st, ascii)); w != n {
					t.Errorf("n %d frac %.1f ascii %v: width %d", n, frac, ascii, w)
				}
			}
		}
	}
	if got := ansi.Strip(waveRow(nil, 0, st, false)); got != "" {
		t.Errorf("wave of no peaks = %q, want empty", got)
	}
}

func TestWaveGlyphsWidthOne(t *testing.T) {
	for _, r := range []rune(blocks + blocksASCII + playheadBar + playheadASC) {
		if w := ansi.StringWidth(string(r)); w != 1 {
			t.Errorf("%q has width %d, want 1", r, w)
		}
	}
}

func bandCovered(s string, bg string) bool {
	on := false
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], "\x1b[") {
			end := strings.IndexByte(s[i:], 'm')
			if end < 0 {
				return false
			}
			params := s[i+2 : i+end]
			switch {
			case strings.Contains(params, bg):
				on = true
			case params == "" || params == "0" || strings.Contains(";"+params+";", ";49;") || strings.HasPrefix(params, "0;"):
				on = false
			}
			i += end + 1
			continue
		}
		if !on {
			return false
		}
		i++
	}
	return true
}

func bandSeq(st styles) string {
	s := st.band.Render("x")
	i := strings.Index(s, "48;")
	if i < 0 {
		return "48;"
	}
	end := strings.IndexByte(s[i:], 'm')
	return s[i : i+end]
}

func TestStatusBandSpansWidth(t *testing.T) {
	for _, w := range []int{60, 80, 160} {
		m := newModel("x.wav", testOpts)
		m, _ = advance(t, m, tea.WindowSizeMsg{Width: w, Height: 24})
		res := testResult(t)
		m.newPlayer = func([]float64, int) (player, error) { return &stubPlayer{}, nil }
		m, _ = advance(t, m, analysisDoneMsg{res})
		band := m.statusView()
		if got := lipgloss.Width(band); got != w {
			t.Errorf("width %d: band is %d wide", w, got)
		}
		if !bandCovered(band, bandSeq(m.st)) {
			t.Errorf("width %d: a cell of the band is drawn without the band background:\n%q", w, band)
		}
		plain := ansi.Strip(band)
		wants := []string{"00:00.0", "m 1", "bpm", m.st.themeName}
		if w >= 80 {
			wants = append(wants, "BEADG", "1/16", "/line")
		}
		for _, want := range wants {
			if !strings.Contains(plain, want) {
				t.Errorf("width %d: band %q lacks %q", w, plain, want)
			}
		}
		m.closePlayer()
	}
}

func TestViewerFillsTheScreen(t *testing.T) {
	for _, size := range [][2]int{{60, 20}, {80, 24}, {160, 40}, {40, 12}} {
		m := newModel("x.wav", testOpts)
		m, _ = advance(t, m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		res := testResult(t)
		res.Samples = nil
		m, _ = advance(t, m, analysisDoneMsg{res})
		check := func(label string) {
			t.Helper()
			view := m.viewerView()
			if got := lipgloss.Height(view); got != size[1] {
				t.Errorf("%dx%d %s: view is %d rows tall", size[0], size[1], label, got)
			}
			for i, line := range strings.Split(view, "\n") {
				if lw := ansi.StringWidth(line); lw > size[0] {
					t.Errorf("%dx%d %s: row %d is %d wide", size[0], size[1], label, i, lw)
				}
			}
		}
		check("closed")
		m, _ = advance(t, m, keyPress('n'))
		check("panel bottom")
		m, _ = advance(t, m, keyPress('N'))
		check("panel right")
		m, _ = advance(t, m, keyPress('?'))
		check("full help")
	}
}

func TestWaveFollowsPosition(t *testing.T) {
	m := newModel("x.wav", testOpts)
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	res := testResult(t)
	for i := range res.Samples {
		res.Samples[i] = math.Sin(float64(i) / 5)
	}
	sp := &stubPlayer{}
	m.newPlayer = func([]float64, int) (player, error) { return sp, nil }
	m, _ = advance(t, m, analysisDoneMsg{res})
	defer m.closePlayer()
	if len(m.wave) != 80 {
		t.Fatalf("wave has %d peaks at 80 columns, want 80", len(m.wave))
	}
	head := m.st.accent.Bold(true).Render(playheadBar)
	at := func() int {
		row := m.waveView()
		i := strings.Index(row, head)
		if i < 0 {
			t.Fatalf("no playhead in %q", row)
		}
		return ansi.StringWidth(ansi.Strip(row[:i]))
	}
	if got := at(); got != 0 {
		t.Errorf("playhead at column %d at the start, want 0", got)
	}
	m.setPos(res.DurationSec / 2)
	if got := at(); got != 40 {
		t.Errorf("playhead at column %d at half time, want 40", got)
	}
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 120, Height: 24})
	if len(m.wave) != 120 {
		t.Errorf("wave has %d peaks after resize to 120, want 120", len(m.wave))
	}
}
