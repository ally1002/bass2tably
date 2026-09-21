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
		{k.Notes, k.PanelSide, k.PanelGrow, k.PanelShrnk, k.Fewer, k.More},
		{k.AutoFit, k.Theme, k.Help, k.Back, k.Quit},
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
		Theme:      key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "theme")),
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:       key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
		Right:      key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
		Top:        key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom:     key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
		Back:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "picker")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

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
	themeIdx    int
	dark        bool
	st          styles
	mpl         int
	panelSize   int
}

func newModel(path string, opts app.Options) model {
	m := model{
		picker:    newPicker([]string{".wav", ".m4a", ".mp3", ".ogg", ".flac"}),
		spin:      spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		vp:        viewport.New(),
		help:      help.New(),
		keys:      defaultViewerKeys(),
		opts:      opts,
		path:      path,
		newPlayer: newPlayer,
		dark:      true,
	}
	m.vp.KeyMap.PageDown.SetKeys("pgdown", "f")
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
	return tea.Batch(requestBackground, m.picker.Init())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.closePlayer()
			return m, tea.Quit
		case "t":
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
	pickerFixed = 6
)

func (m *model) resizeViewer() {
	w := m.w
	h := m.h - 2 - lipgloss.Height(m.help.View(m.keys))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
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
	m.picker.setSize(m.w, m.h-pickerFixed-pickerErr)
	m.refitTab()
}

func (m model) autoPanelWidth(w int) int {
	if m.res == nil {
		return w * 2 / 5
	}
	l := m.res.Layout
	n := fitMeasures(w-minPanelW-1, l)
	tabW := l.NameW + 1 + n*(l.SlotsPerMeasure*l.CellW+1)
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

func (m *model) refitTab() {
	if m.res == nil {
		return
	}
	mpl := m.mpl
	if mpl <= 0 {
		if m.vp.Width() <= 1 {
			return
		}
		mpl = fitMeasures(m.vp.Width(), m.res.Layout)
	}
	if mpl == m.res.Layout.MeasuresPerLine {
		return
	}
	m.res.Tab, m.res.Layout = tab.RenderLayoutWidth(m.res.Notes, m.res.Frets, m.res.Tuning, m.res.GridDiv, mpl)
	m.refreshCursor()
	m.ensureCursorVisible()
}

func (m *model) setMeasuresPerLine(delta int) {
	if m.res == nil {
		return
	}
	cur := m.res.Layout.MeasuresPerLine
	next := clamp(cur+delta, 1, max(m.res.Layout.Measures, 1))
	if next == cur {
		return
	}
	m.mpl = next
	m.refitTab()
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
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

func (m model) updateAnalyzing(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case analysisDoneMsg:
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
		m.vp.GotoTop()
		m.vp.SetXOffset(0)
		m.resizeViewer()
		m.refreshCursor()
		return m, nil
	case analysisErrMsg:
		m.err = msg.err
		m.screen = pickerScreen
		m.resizeViewer()
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
			m.mpl = 0
			m.refitTab()
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
	block, col := -1, -1
	if len(m.res.Notes) > 0 {
		block, col = m.res.Layout.SlotCell(m.curSlot)
	}
	m.vp.SetContent(renderTab(m.res.Tab, m.res.Layout, m.st, block, col))
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
	b.WriteString(" " + m.picker.breadcrumb(m.st))
	b.WriteString("\n")
	if m.err != nil {
		b.WriteString(" " + m.st.danger.Render("✗ "+m.err.Error()))
		b.WriteString("\n")
	}
	b.WriteString(m.rule(m.st.muted.Render(fmt.Sprintf("%d items", len(m.picker.entries))), w))
	b.WriteString("\n")
	b.WriteString(m.picker.View(m.st))
	b.WriteString("\n")
	b.WriteString(m.rule("", w))
	b.WriteString("\n")
	b.WriteString(m.hint([][2]string{
		{"↑↓", "move"}, {"enter", "open"}, {"h", "up"}, {"~", "home"}, {"t", "theme"}, {"q", "quit"},
	}))
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
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.st.pal.frame).
		Padding(1, 3)
	body := m.spin.View() + " " + m.st.title.Render("analyzing") + "\n" +
		m.st.accent.Render(filepath.Base(m.path)) + "\n\n" +
		m.st.muted.Render("onsets · tempo · pitch candidates")
	if m.w == 0 || m.h == 0 {
		return card.Render(body)
	}
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, card.Render(body))
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
	bpm := fmt.Sprintf("%.1f", m.res.BPM)
	if m.res.BPMEstimated {
		bpm = "~" + bpm
	}
	sep := m.st.faint.Render("  ·  ")
	stat := func(v, unit string) string {
		return m.st.fg.Render(v) + " " + m.st.muted.Render(unit)
	}
	name := m.brand() + " " + m.st.title.Render(" "+filepath.Base(m.res.Path))
	grid := fmt.Sprintf("1/%d", m.res.GridDiv*4)
	notes := fmt.Sprint(len(m.res.Notes))
	full := name + sep + stat(bpm, "bpm") + sep + stat(grid, "grid") +
		sep + stat(m.tuningName(), "tuning") + sep + stat(notes, "notes")
	compact := name + sep + stat(bpm, "bpm") + sep + m.st.fg.Render(grid) +
		sep + m.st.fg.Render(m.tuningName()) + sep + stat(notes, "notes")
	right := m.st.muted.Render(m.st.themeName) + " " + m.st.faint.Render(m.st.themeTag) + " "
	if lipgloss.Width(full)+lipgloss.Width(right)+1 <= w {
		return m.spread(full, right, w)
	}
	if lipgloss.Width(compact) <= w {
		return m.spread(compact, right, w)
	}
	return ansi.Truncate(name+sep+stat(bpm, "bpm")+sep+stat(notes, "notes"), w, "…")
}

func (m model) spread(left, right string, w int) string {
	lw, rw := lipgloss.Width(left), lipgloss.Width(right)
	if lw+rw+1 > w {
		return ansi.Truncate(left, w, "…")
	}
	return left + strings.Repeat(" ", w-lw-rw) + right
}

func (m model) transportView() string {
	w := max(m.w, 1)
	if m.audioErr != nil {
		return ansi.Truncate(" "+m.st.danger.Render("⊘ playback unavailable: "+m.audioErr.Error()), w, "…")
	}
	icon := m.st.muted.Render("■")
	switch {
	case m.playing():
		icon = m.st.accent.Render("▶")
	case m.player != nil:
		icon = m.st.fg.Render("❚❚")
	}
	dur := m.res.DurationSec
	clock := m.st.fg.Bold(true).Render(fmtTime(m.curPos)) + m.st.faint.Render(" / ") + m.st.muted.Render(fmtTime(dur))
	measure := m.curSlot/m.res.Layout.SlotsPerMeasure + 1
	beat := (m.curSlot%m.res.Layout.SlotsPerMeasure)/m.res.GridDiv + 1
	pos := m.st.muted.Render("m ") + m.st.fg.Render(fmt.Sprintf("%d", measure)) +
		m.st.faint.Render(fmt.Sprintf("/%d", m.res.Layout.Measures)) +
		m.st.muted.Render("  beat ") + m.st.fg.Render(fmt.Sprint(beat)) +
		m.st.faint.Render(fmt.Sprintf("   %d/line", m.res.Layout.MeasuresPerLine))
	fixed := lipgloss.Width(icon) + lipgloss.Width(clock) + lipgloss.Width(pos) + 7
	barW := w - fixed
	bar := ""
	if barW >= 8 {
		frac := 0.0
		if dur > 0 {
			frac = m.curPos / dur
		}
		on := int(frac*float64(barW) + 0.5)
		on = clamp(on, 0, barW)
		bar = m.st.accent.Render(strings.Repeat("━", on)) + m.st.faint.Render(strings.Repeat("─", barW-on))
		return " " + icon + " " + clock + "  " + bar + "  " + pos
	}
	return ansi.Truncate(" "+icon+" "+clock+"  "+pos, w, "…")
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
			panel = lipgloss.NewStyle().Width(m.notes.width).Height(m.vp.Height()).Render(panel)
			body = lipgloss.JoinHorizontal(lipgloss.Top, tabView, border, panel)
		default:
			body += "\n" + m.rule(m.notes.header(m.st), m.w) + "\n" + m.notes.View(m.st)
		}
	}
	return m.headerView() + "\n" + m.transportView() + "\n" + body + "\n" + m.help.View(m.keys)
}
