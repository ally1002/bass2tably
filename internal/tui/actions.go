package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type action struct {
	id      string
	title   string
	binding key.Binding
	enabled func(m *model) bool
	run     func(m *model) tea.Cmd
}

const (
	helpColumnRows = 7
	paletteWidth   = 60
	paletteRows    = 10
)

func always(*model) bool      { return true }
func hasPlayer(m *model) bool { return m.player != nil }
func notesOpen(m *model) bool { return m.notes.open }
func inPage(m *model) bool    { return m.mode == pageMode }

func (m *model) actions() []action {
	k := m.keys
	do := func(f func(m *model)) func(*model) tea.Cmd {
		return func(m *model) tea.Cmd { f(m); return nil }
	}
	return []action{
		{"play", "Play / pause", k.PlayPause, hasPlayer, func(m *model) tea.Cmd { next, cmd := m.togglePlayback(); *m = next.(model); return cmd }},
		{"play-from", "Play from the selected note", k.PlayFrom, notesOpen, func(m *model) tea.Cmd { next, cmd := m.playFromSelection(); *m = next.(model); return cmd }},
		{"seek-back", "Seek one measure back", k.SeekBack, always, do(func(m *model) { m.seekMeasures(-1) })},
		{"seek-fwd", "Seek one measure forward", k.SeekFwd, always, do(func(m *model) { m.seekMeasures(+1) })},
		{"note-prev", "Select the previous note", k.Up, notesOpen, do(func(m *model) { m.selectNote(m.notes.sel - 1) })},
		{"note-next", "Select the next note", k.Down, notesOpen, do(func(m *model) { m.selectNote(m.notes.sel + 1) })},
		{"note-first", "Select the first note", k.Top, notesOpen, do(func(m *model) { m.selectNote(0) })},
		{"note-last", "Select the last note", k.Bottom, notesOpen, do(func(m *model) { m.selectNote(len(m.notes.rows) - 1) })},
		{"scroll-up", "Scroll the tab up", k.Up, always, do(func(m *model) { m.vp.ScrollUp(1) })},
		{"scroll-down", "Scroll the tab down", k.Down, always, do(func(m *model) { m.vp.ScrollDown(1) })},
		{"scroll-left", "Scroll the tab left", k.Left, always, do(func(m *model) { m.vp.ScrollLeft(4) })},
		{"scroll-right", "Scroll the tab right", k.Right, always, do(func(m *model) { m.vp.ScrollRight(4) })},
		{"top", "Scroll to the top", k.Top, always, do(func(m *model) { m.vp.GotoTop() })},
		{"bottom", "Scroll to the bottom", k.Bottom, always, do(func(m *model) { m.vp.GotoBottom() })},
		{"notes", "Toggle the note list", k.Notes, always, do((*model).toggleNotes)},
		{"notes-side", "Dock the note list at the bottom or on the right", k.PanelSide, always, do((*model).togglePanelSide)},
		{"panel-grow", "Grow the note list", k.PanelGrow, notesOpen, do(func(m *model) { m.resizePanel(+1) })},
		{"panel-shrink", "Shrink the note list", k.PanelShrnk, notesOpen, do(func(m *model) { m.resizePanel(-1) })},
		{"fewer", "Fewer measures per line", k.Fewer, inPage, do(func(m *model) { m.setMeasuresPerLine(-1) })},
		{"more", "More measures per line", k.More, inPage, do(func(m *model) { m.setMeasuresPerLine(+1) })},
		{"fit", "Fit measures per line to the width", k.AutoFit, inPage, do(func(m *model) { m.mpl = 0; m.rebuild() })},
		{"zoom", "Cycle the cell width (zoom)", k.Zoom, always, do((*model).cycleZoom)},
		{"rhythm", "Toggle the rhythm rows", k.Rhythm, always, do(func(m *model) { m.rhythm = !m.rhythm; m.rebuild() })},
		{"view", "Switch between the page and scroll layouts", k.ViewMode, always, do((*model).toggleMode)},
		{"loop-a", "Set loop start at this measure", k.LoopA, hasPlayer, do((*model).setLoopA)},
		{"loop-b", "Set loop end after this measure", k.LoopB, hasPlayer, do((*model).setLoopB)},
		{"loop-clear", "Clear the loop", k.LoopClear, hasPlayer, do((*model).clearLoop)},
		{"click", "Toggle the metronome", k.Click, hasPlayer, do((*model).toggleClick)},
		{"slower", "Slower playback", k.Slower, hasPlayer, do(func(m *model) { m.stepSpeed(-1) })},
		{"faster", "Faster playback", k.Faster, hasPlayer, do(func(m *model) { m.stepSpeed(+1) })},
		{"palette", "Command palette", k.Palette, always, do((*model).openPalette)},
		{"theme", "Next theme", k.Theme, always, do((*model).cycleTheme)},
		{"help", "Toggle the full help", k.Help, always, do(func(m *model) { m.help.ShowAll = !m.help.ShowAll; m.resizeViewer() })},
		{"back", "Back to the finder", k.Back, always, (*model).backToFinder},
		{"quit", "Quit", k.Quit, always, func(m *model) tea.Cmd { m.closePlayer(); return tea.Quit }},
	}
}

func (m *model) backToFinder() tea.Cmd {
	m.closePlayer()
	m.screen = pickerScreen
	m.res = nil
	m.wave = nil
	m.path = ""
	return m.finder.Init()
}

func (m *model) runKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	for _, a := range m.actions() {
		if key.Matches(msg, a.binding) && a.enabled(m) {
			return a.run(m), true
		}
	}
	return nil, false
}

type helpMap struct {
	short []key.Binding
	full  [][]key.Binding
}

func (h helpMap) ShortHelp() []key.Binding  { return h.short }
func (h helpMap) FullHelp() [][]key.Binding { return h.full }

func (m *model) helpMap() helpMap {
	var h helpMap
	seen := map[string]bool{}
	var col []key.Binding
	for _, a := range m.actions() {
		name := strings.Join(a.binding.Keys(), ",")
		if seen[name] {
			continue
		}
		seen[name] = true
		switch a.id {
		case "play", "notes", "palette", "help", "quit":
			h.short = append(h.short, a.binding)
		}
		col = append(col, a.binding)
		if len(col) == helpColumnRows {
			h.full = append(h.full, col)
			col = nil
		}
	}
	if len(col) > 0 {
		h.full = append(h.full, col)
	}
	return h
}

type cmdPalette struct {
	open  bool
	input queryInput
	hits  []action
	sel   int
}

func (m *model) openPalette() {
	m.palette = cmdPalette{open: true, input: queryInput{prompt: "› ", placeholder: "type a command"}}
	m.filterPalette()
}

func (m *model) filterPalette() {
	type scored struct {
		a     action
		score int
	}
	var out []scored
	pattern := m.palette.input.Value()
	for _, a := range m.actions() {
		if a.id == "palette" || !a.enabled(m) {
			continue
		}
		score, _, ok := fuzzyScore(pattern, a.title)
		if !ok {
			continue
		}
		out = append(out, scored{a, score})
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].score > out[j-1].score; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	m.palette.hits = m.palette.hits[:0]
	for _, s := range out {
		m.palette.hits = append(m.palette.hits, s.a)
	}
	m.palette.sel = clamp(m.palette.sel, 0, max(len(m.palette.hits)-1, 0))
}

func (m *model) updatePalette(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "ctrl+p":
		m.palette.open = false
	case "up":
		m.palette.sel = max(m.palette.sel-1, 0)
	case "down":
		m.palette.sel = min(m.palette.sel+1, max(len(m.palette.hits)-1, 0))
	case "enter":
		if m.palette.sel < len(m.palette.hits) {
			a := m.palette.hits[m.palette.sel]
			m.palette.open = false
			return a.run(m)
		}
		m.palette.open = false
	default:
		if m.palette.input.key(msg) {
			m.palette.sel = 0
			m.filterPalette()
		}
	}
	return nil
}

func (m model) paletteView() string {
	w := min(paletteWidth, max(m.w-4, 20))
	on := m.st.onBand
	fill := func(n int) string { return on(m.st.fg).Render(strings.Repeat(" ", max(n, 0))) }
	line := func(left, right string) string {
		left = ansi.Truncate(left, max(w-2-lipgloss.Width(right)-1, 0), "…")
		return fill(1) + left + fill(w-2-lipgloss.Width(left)-lipgloss.Width(right)) + right + fill(1)
	}
	p := m.palette
	input := p.input.view(on(m.st.accent), on(m.st.fg).Bold(true), on(m.st.muted), m.st.cursor)
	rows := []string{fill(w), line(input, on(m.st.faint).Render("esc")), fill(w)}
	if len(p.hits) == 0 {
		rows = append(rows, line(on(m.st.muted).Render("no command matches"), ""))
	}
	start := 0
	if p.sel >= paletteRows {
		start = p.sel - paletteRows + 1
	}
	for i := start; i < len(p.hits) && i < start+paletteRows; i++ {
		a := p.hits[i]
		title, keyStyle := on(m.st.fg), on(m.st.help.ShortKey)
		mark := fill(2)
		if i == p.sel {
			title = on(m.st.accent).Bold(true)
			mark = on(m.st.accent).Render("▌ ")
		}
		rows = append(rows, line(mark+title.Render(a.title), keyStyle.Render(a.binding.Help().Key)))
	}
	rows = append(rows, fill(w))
	return strings.Join(rows, "\n")
}

func (m model) overlay(body, card string) string {
	x := max((m.w-lipgloss.Width(card))/2, 0)
	root := lipgloss.NewLayer(body)
	root.AddLayers(lipgloss.NewLayer(card).X(x).Y(2).Z(1))
	return lipgloss.NewCompositor(root).Render()
}
