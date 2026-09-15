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

type viewerKeys struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Top    key.Binding
	Bottom key.Binding
	Back   key.Binding
	Help   key.Binding
	Quit   key.Binding
}

func (k viewerKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Back, k.Quit}
}

func (k viewerKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Top, k.Bottom},
		{k.Back, k.Help, k.Quit},
	}
}

func defaultViewerKeys() viewerKeys {
	return viewerKeys{
		Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:   key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
		Right:  key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
		Top:    key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom: key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
		Back:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back to picker")),
		Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type model struct {
	screen screen
	picker picker
	spin   spinner.Model
	vp     viewport.Model
	help   help.Model
	keys   viewerKeys
	opts   app.Options
	path   string
	res    *app.Result
	err    error
	w, h   int
}

func newModel(path string, opts app.Options) model {
	m := model{
		picker: newPicker([]string{".wav", ".m4a", ".mp3", ".ogg", ".flac"}),
		spin:   spinner.New(spinner.WithSpinner(spinner.Dot)),
		vp:     viewport.New(),
		help:   help.New(),
		keys:   defaultViewerKeys(),
		opts:   opts,
		path:   path,
	}
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

func (m *model) resizeViewer() {
	w, h := m.w, m.h-viewerChromeLines
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	m.vp.SetWidth(w)
	m.vp.SetHeight(h)
	m.help.SetWidth(w)
	m.picker.setHeight(m.h - pickerChromeLines)
}

const pickerChromeLines = 5

const viewerChromeLines = 3

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
		m.resizeViewer()
		m.vp.SetContent(msg.res.Tab)
		m.vp.GotoTop()
		m.vp.SetXOffset(0)
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
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, m.keys.Top):
			m.vp.GotoTop()
			return m, nil
		case key.Matches(msg, m.keys.Bottom):
			m.vp.GotoBottom()
			return m, nil
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, m.keys.Back):
			m.screen = pickerScreen
			m.res = nil
			m.path = ""
			return m, m.picker.Init()
		}
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
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
	return header + "\n" + m.vp.View() + "\n" + m.help.View(m.keys)
}
