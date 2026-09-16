package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

var noteSelStyle = lipgloss.NewStyle().Reverse(true)

type notePanel struct {
	open   bool
	lines  []string
	sel    int
	offset int
	height int
}

func (p *notePanel) setLines(lines []string) {
	p.lines = lines
	p.sel = 0
	p.offset = 0
}

func (p *notePanel) setHeight(h int) {
	if h < 1 {
		h = 1
	}
	p.height = h
	p.scrollIntoView()
}

func (p *notePanel) setSel(i int) {
	if len(p.lines) == 0 {
		p.sel = 0
		return
	}
	if i < 0 {
		i = 0
	}
	if i > len(p.lines)-1 {
		i = len(p.lines) - 1
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
	if max := len(p.lines) - p.height; p.offset > max {
		p.offset = max
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

func (p notePanel) View(width int) string {
	var b strings.Builder
	b.WriteString(dimStyle.Render(fmt.Sprintf("notes %d/%d", p.sel+1, len(p.lines))))
	if len(p.lines) == 0 {
		return dimStyle.Render("no pitched notes")
	}
	end := p.offset + p.height
	if end > len(p.lines) {
		end = len(p.lines)
	}
	for i := p.offset; i < end; i++ {
		line := p.lines[i]
		if width > 0 && len(line) > width {
			line = line[:width]
		}
		b.WriteString("\n")
		if i == p.sel {
			b.WriteString(noteSelStyle.Render(line))
		} else {
			b.WriteString(line)
		}
	}
	return b.String()
}
