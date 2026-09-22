package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"bass2tably/internal/app"
	"bass2tably/internal/tab"
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

type Config struct {
	Theme string
	ASCII bool
}

type viewerKeys struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Top        key.Binding
	Bottom     key.Binding
	PlayPause  key.Binding
	PlayFrom   key.Binding
	SeekBack   key.Binding
	SeekFwd    key.Binding
	Notes      key.Binding
	PanelSide  key.Binding
	PanelGrow  key.Binding
	PanelShrnk key.Binding
	Fewer      key.Binding
	More       key.Binding
	AutoFit    key.Binding
	Zoom       key.Binding
	Rhythm     key.Binding
	ViewMode   key.Binding
	LoopA      key.Binding
	LoopB      key.Binding
	LoopClear  key.Binding
	Click      key.Binding
	Slower     key.Binding
	Faster     key.Binding
	Theme      key.Binding
	Back       key.Binding
	Help       key.Binding
	Quit       key.Binding
}

func (k viewerKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.PlayPause, k.Notes, k.Theme, k.Help, k.Quit}
}

func (k viewerKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.PlayPause, k.PlayFrom, k.SeekBack, k.SeekFwd},
		{k.Up, k.Down, k.Left, k.Right, k.Top, k.Bottom},
		{k.Notes, k.PanelSide, k.PanelGrow, k.PanelShrnk},
		{k.Fewer, k.More, k.AutoFit, k.Zoom, k.Rhythm, k.ViewMode},
		{k.LoopA, k.LoopB, k.LoopClear, k.Click, k.Slower, k.Faster},
		{k.Theme, k.Help, k.Back, k.Quit},
	}
}

func defaultViewerKeys() viewerKeys {
	return viewerKeys{
		PlayPause:  key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "play/pause")),
		PlayFrom:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "play from note")),
		SeekBack:   key.NewBinding(key.WithKeys("shift+left", ","), key.WithHelp("shift+←/,", "seek −1 measure")),
		SeekFwd:    key.NewBinding(key.WithKeys("shift+right", "."), key.WithHelp("shift+→/.", "seek +1 measure")),
		Notes:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "note list")),
		PanelSide:  key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "notes side")),
		PanelGrow:  key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "grow panel")),
		PanelShrnk: key.NewBinding(key.WithKeys("-", "_"), key.WithHelp("-", "shrink panel")),
		Fewer:      key.NewBinding(key.WithKeys("["), key.WithHelp("[", "fewer per line")),
		More:       key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "more per line")),
		AutoFit:    key.NewBinding(key.WithKeys("0"), key.WithHelp("0", "fit to width")),
		Zoom:       key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "zoom")),
		Rhythm:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rhythm rows")),
		ViewMode:   key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "page/scroll")),
		LoopA:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "loop start")),
		LoopB:      key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "loop end")),
		LoopClear:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "clear loop")),
		Click:      key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "metronome")),
		Slower:     key.NewBinding(key.WithKeys("<"), key.WithHelp("<", "slower")),
		Faster:     key.NewBinding(key.WithKeys(">"), key.WithHelp(">", "faster")),
		Theme:      key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "theme")),
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:       key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
		Right:      key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
		Top:        key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom:     key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
		Back:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "finder")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

type model struct {
	screen      screen
	finder      finder
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
	themeIdx    int
	dark        bool
	st          styles
	mpl         int
	panelSize   int
	sheet       tab.Sheet
	cellW       int
	glyphs      tab.Glyphs
	rhythm      bool
	wave        []float64
	mode        viewMode
	loopA       int
	loopB       int
	metronome   bool
	speed       float64
}

type viewMode int

const (
	pageMode viewMode = iota
	scrollMode
)

func (v viewMode) String() string {
	if v == scrollMode {
		return "scroll"
	}
	return "page"
}

const defaultCellW = 3

func newModel(path string, opts app.Options) model {
	m := model{
		finder:    newFinder([]string{".wav", ".m4a", ".mp3", ".ogg", ".flac"}),
		spin:      spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		vp:        viewport.New(),
		help:      help.New(),
		keys:      defaultViewerKeys(),
		opts:      opts,
		path:      path,
		newPlayer: newPlayer,
		dark:      true,
		cellW:     defaultCellW,
		glyphs:    tab.Unicode,
		rhythm:    true,
		loopA:     -1,
		loopB:     -1,
		speed:     1,
	}
	m.vp.KeyMap.PageDown.SetKeys("pgdown", "f")
	m.vp.KeyMap.PageUp.SetKeys("pgup")
	m.vp.SetHorizontalStep(4)
	m.applyTheme()
	if path != "" {
		m.screen = analyzingScreen
	}
	return m
}

func Run(path string, opts app.Options, cfg Config) error {
	idx, err := themeIndex(cfg.Theme)
	if err != nil {
		return err
	}
	m := newModel(path, opts)
	m.themeIdx = idx
	if cfg.ASCII {
		m.glyphs = tab.ASCII
		m.finder.ascii = true
	}
	m.applyTheme()
	_, err = tea.NewProgram(m).Run()
	return err
}

func (m *model) applyTheme() {
	m.st = newStyles(themes[m.themeIdx], m.dark)
	m.help.Styles = m.st.help
	m.spin.Style = m.st.accent
	m.setTabContent()
}

func (m *model) cycleTheme() {
	m.themeIdx = (m.themeIdx + 1) % len(themes)
	m.applyTheme()
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

func requestBackground() tea.Msg { return tea.RequestBackgroundColor() }

func (m model) Init() tea.Cmd {
	if m.screen == analyzingScreen {
		return tea.Batch(requestBackground, m.spin.Tick, analyzeCmd(m.path, m.opts))
	}
	return tea.Batch(requestBackground, m.finder.Init())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		inFinder := m.screen == pickerScreen
		switch s := msg.String(); {
		case s == "ctrl+c", s == "q" && !inFinder:
			m.closePlayer()
			return m, tea.Quit
		case s == "ctrl+t", s == "t" && !inFinder:
			m.cycleTheme()
			return m, nil
		}
	case tea.BackgroundColorMsg:
		m.dark = msg.IsDark()
		m.applyTheme()
		return m, nil
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
	minTabRows  = 4
	minTabCols  = 24
	minPanelW   = 28
	pickerFixed = 7
)

func (m *model) resizeViewer() {
	w := m.w
	h := m.h - 3 - lipgloss.Height(m.help.View(m.keys))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if m.res != nil && len(m.res.Samples) > 0 && len(m.wave) != m.w {
		m.wave = peaks(m.res.Samples, m.w)
	}
	if m.notes.open {
		switch m.notes.side {
		case panelRight:
			pw := m.panelSize
			if pw <= 0 {
				pw = m.autoPanelWidth(w)
			}
			pw = clamp(pw, minPanelW, w-minTabCols-1)
			m.notes.setSize(pw, max(h-1, 1))
			w = max(w-pw-1, 1)
		default:
			ph := m.panelSize
			if ph <= 0 {
				ph = min(h/3, 12)
			}
			ph = clamp(ph, 1, h-minTabRows-1)
			m.notes.setSize(w, ph)
			h = max(h-ph-1, 1)
		}
	}
	m.vp.SetWidth(w)
	m.vp.SetHeight(h)
	m.help.SetWidth(m.w)
	pickerErr := 0
	if m.err != nil {
		pickerErr = 1
	}
	m.finder.setSize(m.w, m.h-pickerFixed-pickerErr)
	m.rebuild()
}

func (m model) cellWidth() int {
	return max(m.cellW, tab.MinCellW(m.res.Frets))
}

func (m model) autoPanelWidth(w int) int {
	if m.res == nil {
		return w * 2 / 5
	}
	l := m.res.Layout
	cellW := m.cellWidth()
	n := fitMeasures(w-minPanelW-1, l.NameW, cellW, l.SlotsPerMeasure, l.Measures)
	tabW := l.NameW + 1 + n*(l.SlotsPerMeasure*cellW+1)
	return w - tabW - 1
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (m *model) rebuild() {
	if m.res == nil {
		return
	}
	l := m.res.Layout
	mpl := m.mpl
	switch {
	case m.mode == scrollMode:
		mpl = max(l.Measures, 1)
	case mpl > 0:
	case m.vp.Width() <= 1:
		mpl = l.MeasuresPerLine
	default:
		mpl = fitMeasures(m.vp.Width(), l.NameW, m.cellWidth(), l.SlotsPerMeasure, l.Measures)
	}
	m.sheet = tab.Build(m.res.Notes, m.res.Frets, m.res.Tuning, m.res.GridDiv, mpl, m.cellW)
	l = m.sheet.Layout()
	l.Top = 1
	l.Bottom = m.rhythmOpts().rows()
	m.res.Layout = l
	m.refreshCursor()
	m.ensureCursorVisible()
}

func (m model) rhythmOpts() rhythmOpts {
	r := rhythmOpts{gridDiv: m.res.GridDiv, glyphs: rhythmUnicode, on: m.rhythm}
	if m.glyphs == tab.ASCII {
		r.glyphs = rhythmASCII
	}
	if anyShorterThanBeat(m.sheet, r.gridDiv) {
		r.levels = beamLevels(r.gridDiv)
	}
	return r
}

func (m *model) cycleZoom() {
	m.cellW++
	if m.cellW > 4 {
		m.cellW = 2
	}
	m.rebuild()
}

func (m *model) setMeasuresPerLine(delta int) {
	if m.res == nil || m.mode == scrollMode {
		return
	}
	cur := m.res.Layout.MeasuresPerLine
	next := clamp(cur+delta, 1, max(m.res.Layout.Measures, 1))
	if next == cur {
		return
	}
	m.mpl = next
	m.rebuild()
}

func (m *model) resizePanel(delta int) {
	if !m.notes.open {
		return
	}
	switch m.notes.side {
	case panelRight:
		m.panelSize = m.notes.width + delta*4
	default:
		m.panelSize = m.notes.height + delta
	}
	m.resizeViewer()
}

func (m model) updatePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		cmd := m.finder.click(msg.X, msg.Y-m.finderListTop())
		return m, cmd
	case tea.MouseWheelMsg:
		return m, m.finder.wheel(wheelDelta(msg))
	case clearErrMsg:
		m.err = nil
		m.resizeViewer()
		return m, nil
	case analysisErrMsg:
		m.err = msg.err
		m.resizeViewer()
		return m, clearErrCmd()
	case fileChosenMsg:
		m.screen = analyzingScreen
		m.path = msg.path
		return m, tea.Batch(m.spin.Tick, analyzeCmd(msg.path, m.opts))
	}
	var cmd tea.Cmd
	m.finder, cmd = m.finder.Update(msg)
	return m, cmd
}

func (m model) updateAnalyzing(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case analysisDoneMsg:
		pushRecent(msg.res.Path)
		m.res = msg.res
		m.screen = viewerScreen
		m.notes.setResult(msg.res)
		m.closePlayer()
		if len(msg.res.Samples) > 0 {
			m.player, m.audioErr = m.newPlayer(msg.res.Samples, msg.res.SampleRate)
		}
		m.curSlot = 0
		m.curPos = 0
		m.mpl = 0
		m.wave = nil
		m.loopA, m.loopB = -1, -1
		m.metronome = false
		m.speed = 1
		m.vp.GotoTop()
		m.vp.SetXOffset(0)
		m.resizeViewer()
		m.refreshCursor()
		return m, nil
	case analysisErrMsg:
		m.err = msg.err
		m.screen = pickerScreen
		m.resizeViewer()
		return m, tea.Batch(m.finder.Init(), clearErrCmd())
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) updateViewer(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			m.click(msg.X, msg.Y)
		}
		return m, nil
	case tea.MouseWheelMsg:
		if _, inside := m.notesRowAt(msg.X, msg.Y); inside {
			m.notes.scroll(wheelDelta(msg))
			return m, nil
		}
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
		case key.Matches(msg, m.keys.PanelSide):
			m.togglePanelSide()
			return m, nil
		case key.Matches(msg, m.keys.PanelGrow):
			m.resizePanel(+1)
			return m, nil
		case key.Matches(msg, m.keys.PanelShrnk):
			m.resizePanel(-1)
			return m, nil
		case key.Matches(msg, m.keys.Fewer):
			m.setMeasuresPerLine(-1)
			return m, nil
		case key.Matches(msg, m.keys.More):
			m.setMeasuresPerLine(+1)
			return m, nil
		case key.Matches(msg, m.keys.AutoFit):
			if m.mode == pageMode {
				m.mpl = 0
				m.rebuild()
			}
			return m, nil
		case key.Matches(msg, m.keys.ViewMode):
			m.toggleMode()
			return m, nil
		case key.Matches(msg, m.keys.LoopA):
			m.setLoopA()
			return m, nil
		case key.Matches(msg, m.keys.LoopB):
			m.setLoopB()
			return m, nil
		case key.Matches(msg, m.keys.LoopClear):
			m.clearLoop()
			return m, nil
		case key.Matches(msg, m.keys.Click):
			m.toggleClick()
			return m, nil
		case key.Matches(msg, m.keys.Slower):
			m.stepSpeed(-1)
			return m, nil
		case key.Matches(msg, m.keys.Faster):
			m.stepSpeed(1)
			return m, nil
		case key.Matches(msg, m.keys.Zoom):
			m.cycleZoom()
			return m, nil
		case key.Matches(msg, m.keys.Rhythm):
			m.rhythm = !m.rhythm
			m.rebuild()
			return m, nil
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			m.resizeViewer()
			return m, nil
		case key.Matches(msg, m.keys.Back):
			m.closePlayer()
			m.screen = pickerScreen
			m.res = nil
			m.wave = nil
			m.path = ""
			return m, m.finder.Init()
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
				m.selectNote(len(m.notes.rows) - 1)
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

func (m *model) togglePanelSide() {
	if m.notes.side == panelBottom {
		m.notes.side = panelRight
	} else {
		m.notes.side = panelBottom
	}
	m.panelSize = 0
	if !m.notes.open {
		m.notes.open = true
		m.syncNoteSel()
	}
	m.resizeViewer()
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

func (m *model) setTabContent() {
	if m.res == nil {
		return
	}
	cur := -1
	if len(m.res.Notes) > 0 {
		cur = m.curSlot
	}
	m.vp.SetContent(renderSheet(m.sheet, m.glyphs, m.st, cur, m.shading(), m.rhythmOpts(), m.loopRange()))
}

func (m model) shading() bool {
	return m.curPos > 0
}

func (m *model) refreshCursor() {
	if m.res == nil {
		return
	}
	if len(m.res.Notes) == 0 {
		m.setTabContent()
		return
	}
	m.curSlot = cursorSlot(m.res, m.curPos)
	m.setTabContent()
	if m.playing() {
		m.syncNoteSel()
		block, col := m.res.Layout.SlotCell(m.curSlot)
		m.followCursor(block, col)
	}
}

func (m *model) ensureCursorVisible() {
	if m.res == nil || len(m.res.Notes) == 0 {
		return
	}
	block, col := m.res.Layout.SlotCell(m.curSlot)
	m.followCursor(block, col)
}

func (m *model) toggleMode() {
	if m.mode == pageMode {
		m.mode = scrollMode
	} else {
		m.mode = pageMode
	}
	m.vp.GotoTop()
	m.vp.SetXOffset(0)
	m.rebuild()
}

func (m *model) followCursor(block, col int) {
	l := m.res.Layout
	if m.mode == scrollMode {
		m.vp.SetXOffset(max(0, col-m.vp.Width()/3))
		return
	}
	top := block * l.Stride()
	bottom := top + l.Stride() - 2
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
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m model) brand() string {
	return m.st.badge.Render("♪ bass2tably")
}

func (m model) rule(label string, width int) string {
	if width < 1 {
		return ""
	}
	if label == "" {
		return m.st.frame.Render(strings.Repeat("─", width))
	}
	lead := m.st.frame.Render("── ")
	rest := width - 3 - lipgloss.Width(label)
	if rest < 0 {
		return ansi.Truncate(lead+label, width, "")
	}
	return lead + label + " " + m.st.frame.Render(strings.Repeat("─", max(rest-1, 0)))
}

func (m model) pickerView() string {
	w := max(m.w, 1)
	var b strings.Builder
	b.WriteString(m.brand())
	b.WriteString(m.st.muted.Render("  turn a bass recording into tab"))
	b.WriteString("\n\n")
	b.WriteString(" " + ansi.Truncate(m.finder.input.View(m.st), max(w-1, 1), ""))
	b.WriteString("\n")
	b.WriteString(" " + m.finder.breadcrumb(m.st))
	b.WriteString("\n")
	if m.err != nil {
		b.WriteString(" " + m.st.danger.Render("✗ "+m.err.Error()))
		b.WriteString("\n")
	}
	b.WriteString(m.rule(m.st.muted.Render(m.finder.status()), w))
	b.WriteString("\n")
	b.WriteString(m.finder.View(m.st))
	b.WriteString("\n")
	b.WriteString(m.rule("", w))
	b.WriteString("\n")
	b.WriteString(ansi.Truncate(m.hint([][2]string{
		{"type", "find"}, {"↑↓", "move"}, {"enter", "open"}, {"⌫", "up"}, {"~", "home"}, {"esc", "clear"}, {"ctrl+t", "theme"}, {"ctrl+c", "quit"},
	}), w, ""))
	return b.String()
}

func (m model) hint(pairs [][2]string) string {
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = m.st.help.ShortKey.Render(p[0]) + " " + m.st.help.ShortDesc.Render(p[1])
	}
	return " " + strings.Join(parts, m.st.help.ShortSeparator.Render(" • "))
}

func (m model) analyzingView() string {
	body := m.spin.View() + " " + m.st.title.Render("analyzing ") + m.st.accent.Render(filepath.Base(m.path)) + "\n" +
		m.st.muted.Render("onsets · tempo · pitch candidates")
	if m.w == 0 || m.h == 0 {
		return body
	}
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, body)
}

func fmtTime(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	mm := int(sec) / 60
	ss := sec - float64(mm*60)
	return fmt.Sprintf("%02d:%04.1f", mm, ss)
}

func (m model) tuningName() string {
	var b strings.Builder
	for s := range m.res.Tuning {
		b.WriteString(m.res.StringName(s))
	}
	return b.String()
}

func (m model) headerView() string {
	w := max(m.w, 1)
	name := m.brand() + " " + m.st.title.Render(" "+filepath.Base(m.res.Path))
	sep := m.st.faint.Render(" · ")
	rhythm := "rhythm off"
	if m.rhythm {
		rhythm = "rhythm"
	}
	right := m.st.muted.Render(fmt.Sprintf("×%d", m.res.Layout.CellW)) + sep +
		m.st.muted.Render(rhythm) + sep + m.st.muted.Render(m.mode.String()) + " "
	return m.spread(name, right, w)
}

func (m model) spread(left, right string, w int) string {
	lw, rw := lipgloss.Width(left), lipgloss.Width(right)
	if lw+rw+1 > w {
		return ansi.Truncate(left, w, "…")
	}
	return left + strings.Repeat(" ", w-lw-rw) + right
}

func (m model) waveView() string {
	w := max(m.w, 1)
	if len(m.wave) == 0 {
		return blankRow(w)
	}
	frac := 0.0
	if d := m.res.DurationSec; d > 0 {
		frac = m.curPos / d
	}
	la, lb := m.loopFractions()
	return waveRow(m.wave, frac, la, lb, m.st, m.glyphs == tab.ASCII)
}

func (m model) statusView() string {
	w := max(m.w, 1)
	on := m.st.onBand
	sp := func(n int) string { return on(m.st.fg).Render(strings.Repeat(" ", n)) }
	sep := on(m.st.faint).Render(" · ")
	var left string
	if m.audioErr != nil {
		left = sp(1) + on(m.st.danger).Render("⊘ playback unavailable: "+m.audioErr.Error())
	} else {
		icon := on(m.st.muted).Render("■")
		switch {
		case m.playing():
			icon = on(m.st.accent).Render("▶")
		case m.player != nil:
			icon = on(m.st.fg).Render("❚❚")
		}
		clock := on(m.st.fg.Bold(true)).Render(fmtTime(m.curPos)) + on(m.st.faint).Render(" / ") +
			on(m.st.muted).Render(fmtTime(m.res.DurationSec))
		measure := m.curSlot/m.res.Layout.SlotsPerMeasure + 1
		beat := (m.curSlot%m.res.Layout.SlotsPerMeasure)/m.res.GridDiv + 1
		pos := on(m.st.muted).Render("m ") + on(m.st.fg).Render(fmt.Sprint(measure)) +
			on(m.st.faint).Render(fmt.Sprintf("/%d", m.res.Layout.Measures)) +
			on(m.st.muted).Render("  beat ") + on(m.st.fg).Render(fmt.Sprint(beat))
		left = sp(1) + icon + sp(1) + clock + sp(3) + pos + m.practiceLabels(sp)
	}
	bpm := fmt.Sprintf("%.1f", m.res.BPM)
	if m.res.BPMEstimated {
		bpm = "~" + bpm
	}
	tempo := on(m.st.fg).Render(bpm+" ") + on(m.st.muted).Render("bpm")
	theme := on(m.st.muted).Render(m.st.themeName + " ")
	full := tempo + sep + on(m.st.fg).Render(m.tuningName()) + sep +
		on(m.st.fg).Render(fmt.Sprintf("1/%d", m.res.GridDiv*4)) + sep +
		on(m.st.fg).Render(m.perLineLabel()) + sep + theme
	for _, right := range []string{full, tempo + sep + theme, ""} {
		if lipgloss.Width(left)+lipgloss.Width(right)+1 <= w {
			return m.bandLine(left, right, w)
		}
	}
	return m.bandLine(left, "", w)
}

func (m model) bandLine(left, right string, w int) string {
	fill := m.st.onBand(m.st.fg)
	lw, rw := lipgloss.Width(left), lipgloss.Width(right)
	if lw+rw+1 > w {
		left = ansi.Truncate(left, max(w-1, 0), "") + fill.Render("…")
		lw, right, rw = lipgloss.Width(left), "", 0
	}
	return left + fill.Render(strings.Repeat(" ", max(w-lw-rw, 0))) + right
}

func (m model) viewerView() string {
	if m.res == nil {
		return ""
	}
	tabView := lipgloss.NewStyle().Width(m.vp.Width()).Height(m.vp.Height()).Render(m.vp.View())
	body := tabView
	if m.notes.open {
		switch m.notes.side {
		case panelRight:
			border := strings.TrimRight(strings.Repeat(m.st.frame.Render("│")+"\n", m.vp.Height()), "\n")
			panel := m.rule(m.notes.header(m.st), m.notes.width) + "\n" + m.notes.View(m.st)
			panel = lipgloss.NewStyle().Width(m.notes.width).Height(m.vp.Height()).MaxHeight(m.vp.Height()).Render(panel)
			body = lipgloss.JoinHorizontal(lipgloss.Top, tabView, border, panel)
		default:
			body += "\n" + m.rule(m.notes.header(m.st), m.w) + "\n" +
				lipgloss.NewStyle().Height(m.notes.height).MaxHeight(m.notes.height).Render(m.notes.View(m.st))
		}
	}
	return m.headerView() + "\n" + body + "\n" + m.waveView() + "\n" + m.statusView() + "\n" + m.helpView()
}

func (m model) helpView() string {
	lines := strings.Split(m.help.View(m.keys), "\n")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, max(m.w, 1), "")
	}
	return strings.Join(lines, "\n")
}

const (
	tabTop        = 1
	finderListTop = 5
	wheelStep     = 3
)

func wheelDelta(msg tea.MouseWheelMsg) int {
	if msg.Button == tea.MouseWheelUp {
		return -wheelStep
	}
	return wheelStep
}

func (m model) finderListTop() int {
	if m.err != nil {
		return finderListTop + 1
	}
	return finderListTop
}

func (m model) waveRowY() int {
	y := tabTop + m.vp.Height()
	if m.notes.open && m.notes.side == panelBottom {
		y += 1 + m.notes.height
	}
	return y
}

func (m model) notesRowAt(x, y int) (int, bool) {
	if !m.notes.open {
		return 0, false
	}
	var y0 int
	switch m.notes.side {
	case panelRight:
		if x < m.vp.Width()+1 {
			return 0, false
		}
		y0 = tabTop + 1
	default:
		y0 = tabTop + m.vp.Height() + 1
	}
	if y < y0 || y >= y0+m.notes.height {
		return 0, false
	}
	return y - y0 + m.notes.offset, true
}

func (m *model) click(x, y int) {
	if m.res == nil {
		return
	}
	switch {
	case y == m.waveRowY():
		if d := m.res.DurationSec; d > 0 && m.w > 0 {
			m.setPos(d * float64(x) / float64(m.w))
			m.syncNoteSel()
			m.ensureCursorVisible()
		}
	case y >= tabTop && y < tabTop+m.vp.Height() && x < m.vp.Width():
		if len(m.res.Notes) == 0 {
			return
		}
		if slot, ok := m.res.Layout.SlotAt(y-tabTop+m.vp.YOffset(), x+m.vp.XOffset()); ok {
			m.setPos(slotTime(m.res, slot))
			m.syncNoteSel()
		}
	default:
		if row, inside := m.notesRowAt(x, y); inside && row < len(m.notes.rows) {
			m.selectNote(row)
		}
	}
}

func (m model) perLineLabel() string {
	if m.mode == scrollMode {
		return "scroll"
	}
	return fmt.Sprintf("%d/line", m.res.Layout.MeasuresPerLine)
}

var speedSteps = []float64{0.5, 0.75, 1}

func (m model) measureOf(slot int) int {
	return slot / m.res.Layout.SlotsPerMeasure
}

func (m model) loopRange() [2]int { return [2]int{m.loopA, m.loopB} }

func (m model) loopActive() bool { return m.loopA >= 0 && m.loopB > m.loopA }

func (m model) loopSeconds() (float64, float64) {
	spm := m.res.Layout.SlotsPerMeasure
	return slotTime(m.res, m.loopA*spm), slotTime(m.res, m.loopB*spm)
}

func (m model) loopFractions() (float64, float64) {
	if m.res == nil || !m.loopActive() || m.res.DurationSec <= 0 {
		return 0, 0
	}
	a, b := m.loopSeconds()
	return a / m.res.DurationSec, b / m.res.DurationSec
}

func (m *model) setLoopA() {
	if m.res == nil {
		return
	}
	m.loopA = m.measureOf(m.curSlot)
	if m.loopB >= 0 && m.loopB <= m.loopA {
		m.loopA, m.loopB = m.loopB-1, m.loopA+1
	}
	m.applyLoop()
}

func (m *model) setLoopB() {
	if m.res == nil {
		return
	}
	m.loopB = m.measureOf(m.curSlot) + 1
	if m.loopA < 0 {
		m.loopA = m.loopB - 1
	}
	if m.loopB <= m.loopA {
		m.loopA, m.loopB = m.loopB-1, m.loopA+1
	}
	m.applyLoop()
}

func (m *model) clearLoop() {
	m.loopA, m.loopB = -1, -1
	m.applyLoop()
}

func (m *model) applyLoop() {
	if m.player != nil {
		if m.loopActive() {
			a, b := m.loopSeconds()
			m.player.SetLoop(a, b)
		} else {
			m.player.ClearLoop()
		}
	}
	m.setTabContent()
}

func (m *model) toggleClick() {
	if m.res == nil || len(m.res.Notes) == 0 {
		return
	}
	m.metronome = !m.metronome
	if m.player != nil {
		m.player.SetClick(m.res.BPM, m.res.Notes[0].StartSec, 4, m.metronome)
	}
}

func (m *model) stepSpeed(dir int) {
	i := 0
	for k, s := range speedSteps {
		if s == m.speed {
			i = k
		}
	}
	i = clamp(i+dir, 0, len(speedSteps)-1)
	m.speed = speedSteps[i]
	if m.player != nil {
		m.player.SetSpeed(m.speed)
	}
}

func (m model) practiceLabels(sp func(int) string) string {
	on := m.st.onBand
	var b strings.Builder
	if m.loopActive() {
		b.WriteString(sp(3) + on(m.st.accent).Render(fmt.Sprintf("↻ %d–%d", m.loopA+1, m.loopB)))
	}
	if m.metronome {
		b.WriteString(sp(3) + on(m.st.accent).Render("♪ click"))
	}
	if m.speed != 1 {
		label := fmt.Sprintf("%g×", m.speed)
		if m.speed <= 0.5 {
			label += " (−1 oct)"
		}
		b.WriteString(sp(3) + on(m.st.accent).Render(label))
	}
	return b.String()
}
