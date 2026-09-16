package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"bass2tably/internal/app"
)

type screen int

const (
	pickerScreen screen = iota
	analyzingScreen
	viewerScreen
)

type analysisDoneMsg struct{ res *app.Result }
type analysisErrMsg struct{ err error }
type clearErrMsg struct{}
type tickMsg struct{ gen int }

type viewerKeys struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Top       key.Binding
	Bottom    key.Binding
	PlayPause key.Binding
	PlayFrom  key.Binding
	SeekBack  key.Binding
	SeekFwd   key.Binding
	Notes     key.Binding
	Back      key.Binding
	Help      key.Binding
	Quit      key.Binding
}

func (k viewerKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.PlayPause, k.Notes, k.Help, k.Back, k.Quit}
}

func (k viewerKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Top, k.Bottom, k.PlayPause},
		{k.SeekBack, k.SeekFwd, k.PlayFrom},
		{k.Notes, k.Back, k.Help, k.Quit},
	}
}

func defaultViewerKeys() viewerKeys {
	return viewerKeys{
		PlayPause: key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "play/pause")),
		PlayFrom:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "play from note")),
		SeekBack:  key.NewBinding(key.WithKeys("shift+left", ","), key.WithHelp("shift+←/,", "seek -1 measure")),
		SeekFwd:   key.NewBinding(key.WithKeys("shift+right", "."), key.WithHelp("shift+→/.", "seek +1 measure")),
		Notes:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "note list")),
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:      key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
		Right:     key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
		Top:       key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom:    key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
		Back:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back to picker")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type model struct {
	screen      screen
	picker      picker
	spin        spinner.Model
	vp          viewport.Model
	help        help.Model
	notes       notePanel
	keys        viewerKeys
	opts        app.Options
	path        string
	res         *app.Result
	err         error
	w, h        int
	player      player
	newPlayer   func(samples []float64, rate int) (player, error)
	audioErr    error
	tickGen     int
	tickRunning bool
	curSlot     int
	curPos      float64
}

func newModel(path string, opts app.Options) model {
	m := model{
		picker:    newPicker([]string{".wav", ".m4a", ".mp3", ".ogg", ".flac"}),
		spin:      spinner.New(spinner.WithSpinner(spinner.Dot)),
		vp:        viewport.New(),
		help:      help.New(),
		keys:      defaultViewerKeys(),
		opts:      opts,
		path:      path,
		newPlayer: newPlayer,
	}
	m.vp.KeyMap.PageDown.SetKeys("pgdown", "f")
	if path != "" {
		m.screen = analyzingScreen
	}
	return m
}

func Run(path string, opts app.Options) error {
	_, err := tea.NewProgram(newModel(path, opts)).Run()
	return err
}

func analyzeCmd(path string, opts app.Options) tea.Cmd {
	return func() tea.Msg {
		res, err := app.Analyze(path, opts)
		if err != nil {
			return analysisErrMsg{err}
		}
		return analysisDoneMsg{res}
	}
}

func clearErrCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearErrMsg{} })
}

func (m model) Init() tea.Cmd {
	if m.screen == analyzingScreen {
		return tea.Batch(m.spin.Tick, analyzeCmd(m.path, m.opts))
	}
	return m.picker.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.closePlayer()
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.resizeViewer()
	}
	switch m.screen {
	case pickerScreen:
		return m.updatePicker(msg)
	case analyzingScreen:
		return m.updateAnalyzing(msg)
	default:
		return m.updateViewer(msg)
	}
}

const (
	pickerChromeLines = 5
	viewerChromeLines = 3
)

func (m *model) resizeViewer() {
	w, h := m.w, m.h-viewerChromeLines
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if m.notes.open {
		ph := h / 3
		if ph > 12 {
			ph = 12
		}
		if ph > h-4 {
			ph = h - 4
		}
		if ph < 1 {
			ph = 1
		}
		m.notes.setHeight(ph)
		if h -= ph + 1; h < 1 {
			h = 1
		}
	}
	m.vp.SetWidth(w)
	m.vp.SetHeight(h)
	m.help.SetWidth(w)
	m.picker.setHeight(m.h - pickerChromeLines)
}

func (m model) updatePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearErrMsg:
		m.err = nil
		return m, nil
	case analysisErrMsg:
		m.err = msg.err
		return m, clearErrCmd()
	case fileChosenMsg:
		m.screen = analyzingScreen
		m.path = msg.path
		return m, tea.Batch(m.spin.Tick, analyzeCmd(msg.path, m.opts))
	}
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

func (m model) updateAnalyzing(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case analysisDoneMsg:
		m.res = msg.res
		m.screen = viewerScreen
		m.notes.setLines(msg.res.NoteLines())
		m.resizeViewer()
		m.vp.SetContent(msg.res.Tab)
		m.vp.GotoTop()
		m.vp.SetXOffset(0)
		m.closePlayer()
		if len(msg.res.Samples) > 0 {
			m.player, m.audioErr = m.newPlayer(msg.res.Samples, msg.res.SampleRate)
		}
		m.curSlot = 0
		m.curPos = 0
		m.refreshCursor()
		return m, nil
	case analysisErrMsg:
		m.err = msg.err
		m.screen = pickerScreen
		return m, tea.Batch(m.picker.Init(), clearErrCmd())
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) updateViewer(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if msg.gen != m.tickGen {
			return m, nil
		}
		if m.player == nil || !m.player.Playing() {
			m.tickRunning = false
			return m, nil
		}
		m.curPos = m.player.PosSec()
		if m.player.Done() {
			if err := m.player.Pause(); err != nil {
				m.audioErr = err
			}
			m.tickRunning = false
			m.syncNoteSel()
			m.refreshCursor()
			return m, nil
		}
		m.refreshCursor()
		return m, tickCmd(m.tickGen)
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.PlayPause):
			return m.togglePlayback()
		case key.Matches(msg, m.keys.SeekBack):
			m.seekMeasures(-1)
			return m, nil
		case key.Matches(msg, m.keys.SeekFwd):
			m.seekMeasures(+1)
			return m, nil
		case key.Matches(msg, m.keys.Notes):
			m.toggleNotes()
			return m, nil
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			m.resizeViewer()
			return m, nil
		case key.Matches(msg, m.keys.Back):
			m.closePlayer()
			m.screen = pickerScreen
			m.res = nil
			m.path = ""
			return m, m.picker.Init()
		}
		if m.notes.open {
			switch {
			case key.Matches(msg, m.keys.Up):
				m.selectNote(m.notes.sel - 1)
				return m, nil
			case key.Matches(msg, m.keys.Down):
				m.selectNote(m.notes.sel + 1)
				return m, nil
			case key.Matches(msg, m.keys.Top):
				m.selectNote(0)
				return m, nil
			case key.Matches(msg, m.keys.Bottom):
				m.selectNote(len(m.notes.lines) - 1)
				return m, nil
			case key.Matches(msg, m.keys.PlayFrom):
				return m.playFromSelection()
			}
		}
		switch {
		case key.Matches(msg, m.keys.Top):
			m.vp.GotoTop()
			return m, nil
		case key.Matches(msg, m.keys.Bottom):
			m.vp.GotoBottom()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m *model) toggleNotes() {
	m.notes.open = !m.notes.open
	m.resizeViewer()
	if m.notes.open {
		m.syncNoteSel()
	}
}

func (m model) togglePlayback() (tea.Model, tea.Cmd) {
	if m.player == nil {
		return m, nil
	}
	if m.player.Playing() {
		if err := m.player.Pause(); err != nil {
			m.audioErr = err
		}
		return m, nil
	}
	if m.player.Done() {
		m.setPos(0)
	}
	if err := m.player.Play(); err != nil {
		m.audioErr = err
		return m, nil
	}
	m.refreshCursor()
	return m, m.ensureTick()
}

func (m model) playFromSelection() (tea.Model, tea.Cmd) {
	m.selectNote(m.notes.sel)
	if m.player == nil || m.player.Playing() {
		return m, nil
	}
	if err := m.player.Play(); err != nil {
		m.audioErr = err
		return m, nil
	}
	return m, m.ensureTick()
}

func (m *model) selectNote(i int) {
	if m.res == nil || len(m.res.Notes) == 0 {
		return
	}
	m.notes.setSel(i)
	m.setPos(m.res.Notes[m.notes.sel].StartSec)
	m.ensureCursorVisible()
}

func (m *model) seekMeasures(delta float64) {
	if m.res == nil {
		return
	}
	m.setPos(m.curPos + delta*240.0/m.res.BPM)
	m.syncNoteSel()
	m.ensureCursorVisible()
}

func (m *model) setPos(t float64) {
	if m.res == nil {
		return
	}
	if t < 0 {
		t = 0
	}
	if d := m.res.DurationSec; d > 0 && t > d {
		t = d
	}
	m.curPos = t
	if m.player != nil {
		m.player.SeekSec(t)
	}
	m.refreshCursor()
}

func (m *model) syncNoteSel() {
	if m.res == nil || len(m.res.Notes) == 0 {
		return
	}
	m.notes.setSel(app.NoteAt(m.res, m.curPos))
}

func (m *model) ensureTick() tea.Cmd {
	if m.tickRunning {
		return nil
	}
	m.tickRunning = true
	m.tickGen++
	return tickCmd(m.tickGen)
}

func tickCmd(gen int) tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{gen} })
}

func (m *model) closePlayer() {
	if m.player != nil {
		m.player.Close()
		m.player = nil
	}
	m.audioErr = nil
	m.tickGen++
	m.tickRunning = false
}

func (m model) playing() bool {
	return m.player != nil && m.player.Playing()
}

func (m *model) refreshCursor() {
	if m.res == nil || len(m.res.Notes) == 0 {
		return
	}
	m.curSlot = cursorSlot(m.res, m.curPos)
	block, col := m.res.Layout.SlotCell(m.curSlot)
	m.vp.SetContent(highlightColumn(m.res.Tab, m.res.Layout, block, col, cursorStyle))
	if m.playing() {
		m.syncNoteSel()
		m.followCursor(block, col)
	}
}

func (m *model) ensureCursorVisible() {
	if m.res == nil {
		return
	}
	block, col := m.res.Layout.SlotCell(m.curSlot)
	m.followCursor(block, col)
}

func (m *model) followCursor(block, col int) {
	l := m.res.Layout
	top := block * (l.Rows + 1)
	bottom := top + l.Rows - 1
	switch {
	case top < m.vp.YOffset():
		m.vp.SetYOffset(top)
	case bottom >= m.vp.YOffset()+m.vp.Height():
		m.vp.SetYOffset(bottom - m.vp.Height() + 1)
	}
	switch {
	case col < m.vp.XOffset():
		m.vp.SetXOffset(col)
	case col+l.CellW > m.vp.XOffset()+m.vp.Width():
		m.vp.SetXOffset(col + l.CellW - m.vp.Width())
	}
}

func (m model) View() tea.View {
	var s string
	switch m.screen {
	case pickerScreen:
		s = m.pickerView()
	case analyzingScreen:
		s = m.analyzingView()
	default:
		s = m.viewerView()
	}
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

func (m model) pickerView() string {
	head := titleStyle.Render("bass2tably — pick an audio file")
	if m.err != nil {
		head += "\n" + errStyle.Render("error: "+m.err.Error())
	}
	return head + "\n\n" + m.picker.View()
}

func (m model) analyzingView() string {
	return fmt.Sprintf("\n %s analyzing %s…\n", m.spin.View(), m.path)
}

func (m model) viewerView() string {
	if m.res == nil {
		return ""
	}
	bpm := fmt.Sprintf("%.1f BPM", m.res.BPM)
	if m.res.BPMEstimated {
		bpm += " (estimated)"
	}
	header := titleStyle.Render(m.res.Path) + dimStyle.Render(
		fmt.Sprintf("  %s · 1/%d grid · %d notes", bpm, m.res.GridDiv*4, len(m.res.Notes)))
	if m.audioErr != nil {
		header += "\n" + errStyle.Render("playback unavailable: "+m.audioErr.Error())
	}
	s := header + "\n" + m.vp.View()
	if m.notes.open {
		s += "\n" + m.notes.View(m.w)
	}
	return s + "\n" + m.help.View(m.keys)
}
