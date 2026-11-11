package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ally1002/bass2tably/internal/app"
)

type panelSide int

const (
	panelBottom panelSide = iota
	panelRight
)

type notePanel struct {
	open   bool
	side   panelSide
	rows   []app.NoteRow
	names  []string
	sel    int
	offset int
	height int
	width  int
}

func (p *notePanel) setResult(res *app.Result) {
	p.rows = res.NoteRows()
	p.names = make([]string, len(res.Tuning))
	for s := range res.Tuning {
		p.names[s] = res.StringName(s)
	}
	p.sel = 0
	p.offset = 0
}

func (p *notePanel) setSize(w, h int) {
	if h < 1 {
		h = 1
	}
	if w < 1 {
		w = 1
	}
	p.height = h
	p.width = w
	p.scrollIntoView()
}

func (p *notePanel) setSel(i int) {
	if len(p.rows) == 0 {
		p.sel = 0
		return
	}
	if i < 0 {
		i = 0
	}
	if i > len(p.rows)-1 {
		i = len(p.rows) - 1
	}
	p.sel = i
	p.scrollIntoView()
}

func (p *notePanel) scrollIntoView() {
	if p.height < 1 {
		return
	}
	if p.sel < p.offset {
		p.offset = p.sel
	}
	if p.sel >= p.offset+p.height {
		p.offset = p.sel - p.height + 1
	}
	if max := len(p.rows) - p.height; p.offset > max {
		p.offset = max
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

func (p notePanel) stringName(s int) string {
	if s < 0 || s >= len(p.names) {
		return "?"
	}
	return p.names[s]
}

func (p notePanel) header(st styles) string {
	if len(p.rows) == 0 {
		return st.muted.Render("no pitched notes")
	}
	return st.fg.Bold(true).Render(fmt.Sprintf("notes %d/%d", p.sel+1, len(p.rows)))
}

func (p notePanel) View(st styles) string {
	if len(p.rows) == 0 {
		return ""
	}
	idxW := len(fmt.Sprint(len(p.rows)))
	end := p.offset + p.height
	if end > len(p.rows) {
		end = len(p.rows)
	}
	lines := make([]string, 0, p.height)
	for i := p.offset; i < end; i++ {
		lines = append(lines, p.row(st, p.rows[i], idxW, i == p.sel))
	}
	return strings.Join(lines, "\n")
}

func (p notePanel) row(st styles, r app.NoteRow, idxW int, selected bool) string {
	bg := func(s lipgloss.Style) lipgloss.Style {
		if selected {
			return s.Background(st.pal.selBg)
		}
		return s
	}
	mark := "  "
	if selected {
		mark = st.selMark.Render("▶ ")
	}
	var b strings.Builder
	b.WriteString(mark)
	b.WriteString(bg(st.muted).Render(fmt.Sprintf("#%-*d", idxW, r.Index)))
	b.WriteString(bg(st.fg).Render(fmt.Sprintf(" %6.2fs  ", r.StartSec)))
	b.WriteString(bg(st.name(r.String)).Render(fmt.Sprintf("%-3s", r.Name)))
	if r.String >= 0 {
		b.WriteString(bg(st.muted).Render("  "))
		b.WriteString(bg(st.name(r.String)).Render(p.stringName(r.String)))
		b.WriteString(bg(st.fg).Render(fmt.Sprintf(" %-2d", r.Fret)))
	} else {
		b.WriteString(bg(st.danger).Render("  unplayable"))
	}
	b.WriteString(bg(st.muted).Render(fmt.Sprintf("  %-8s", r.Beats)))
	if len(r.Alts) > 0 {
		parts := make([]string, len(r.Alts))
		for i, a := range r.Alts {
			parts[i] = fmt.Sprintf("%s%d", p.stringName(a.String), a.Fret)
		}
		b.WriteString(bg(st.faint).Render("  also " + strings.Join(parts, " ")))
	}
	line := ansi.Truncate(b.String(), p.width, "…")
	if pad := p.width - ansi.StringWidth(line); pad > 0 {
		line += bg(lipgloss.NewStyle()).Render(strings.Repeat(" ", pad))
	}
	return line
}

func (p *notePanel) scroll(delta int) {
	p.offset = clamp(p.offset+delta, 0, max(len(p.rows)-p.height, 0))
}
