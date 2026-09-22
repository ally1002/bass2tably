package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"bass2tably/internal/gpout"
	"bass2tably/internal/midiout"
)

const exportWidth = 80

type exportDialog struct {
	open bool
	gp5  bool
	mid  bool
	path queryInput
	sel  int
}

func (m *model) openExport() {
	if m.res == nil {
		return
	}
	base := strings.TrimSuffix(m.res.Path, filepath.Ext(m.res.Path))
	if abs, err := filepath.Abs(base); err == nil {
		base = abs
	}
	m.exp = exportDialog{open: true, path: queryInput{prompt: "› ", value: base}}
}

func (m *model) updateExport(msg tea.KeyPressMsg) tea.Cmd {
	d := &m.exp
	switch msg.String() {
	case "esc":
		d.open = false
	case "up", "shift+tab":
		d.sel = max(d.sel-1, 0)
	case "down", "tab":
		d.sel = min(d.sel+1, 2)
	case "enter":
		if !d.gp5 && !d.mid {
			return nil
		}
		d.open = false
		return m.export(*d)
	case "space":
		switch d.sel {
		case 0:
			d.gp5 = !d.gp5
		case 1:
			d.mid = !d.mid
		default:
			d.path.key(msg)
		}
	default:
		if d.sel == 2 {
			d.path.key(msg)
		}
	}
	return nil
}

func exportBase(path string) string {
	p := strings.TrimSpace(path)
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			p = home + strings.TrimPrefix(p, "~")
		}
	}
	for _, ext := range []string{".gp5", ".mid"} {
		if strings.EqualFold(filepath.Ext(p), ext) {
			p = strings.TrimSuffix(p, filepath.Ext(p))
		}
	}
	return p
}

func (m *model) export(d exportDialog) tea.Cmd {
	res := m.res
	base := exportBase(d.path.Value())
	title := filepath.Base(base)
	return func() tea.Msg {
		var wrote []string
		if d.gp5 {
			score, rep := gpout.Build(res.Notes, res.Frets, res.Tuning, res.GridDiv, res.BPM, title)
			if err := gpout.Write(base+".gp5", score); err != nil {
				return flashMsg{err.Error(), true}
			}
			wrote = append(wrote, fmt.Sprintf("%s.gp5 (%d bars, %d notes)", title, rep.Bars, rep.Notes))
		}
		if d.mid {
			if err := midiout.Write(base+".mid", res.Notes, res.BPM, res.GridDiv); err != nil {
				return flashMsg{err.Error(), true}
			}
			wrote = append(wrote, title+".mid")
		}
		return flashMsg{"wrote " + strings.Join(wrote, " and "), false}
	}
}

func (m model) exportView() string {
	c := m.newCard(exportWidth)
	on := m.st.onBand
	d := m.exp
	base := exportBase(d.path.Value())
	title := filepath.Base(base)
	row := func(i int, checked bool, label, file string) string {
		box := "[ ]"
		if checked {
			box = "[x]"
		}
		mark, style := c.fill(2), on(m.st.fg)
		if d.sel == i {
			mark, style = on(m.st.accent).Render("▌ "), on(m.st.accent).Bold(true)
		}
		return c.line(mark+style.Render(box+" "+label), on(m.st.muted).Render(file))
	}
	var path string
	if d.sel == 2 {
		path = d.path.view(on(m.st.accent), on(m.st.fg).Bold(true), on(m.st.muted), m.st.cursor)
	} else {
		path = on(m.st.accent).Render(d.path.prompt) + on(m.st.fg).Render(d.path.Value())
	}
	if over := lipglossWidth(path) - (c.w - 6); over > 0 {
		path = ansi.TruncateLeft(path, over, "…")
	}
	pathMark := c.fill(2)
	if d.sel == 2 {
		pathMark = on(m.st.accent).Render("▌ ")
	}
	hint := "↑↓ move · space toggle · enter write · esc cancel"
	if !d.gp5 && !d.mid {
		hint = "pick at least one format"
	}
	rows := []string{
		c.blank(),
		c.line(on(m.st.fg).Bold(true).Render("Export"), on(m.st.faint).Render("esc")),
		c.blank(),
		row(0, d.gp5, "Guitar Pro 5", title+".gp5"),
		row(1, d.mid, "MIDI", title+".mid"),
		c.blank(),
		c.line(pathMark+on(m.st.muted).Render("to ")+path, ""),
		c.blank(),
		c.line(on(m.st.faint).Render(hint), ""),
		c.blank(),
	}
	return strings.Join(rows, "\n")
}

func lipglossWidth(s string) int { return ansi.StringWidth(s) }
