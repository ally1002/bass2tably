package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type pickEntry struct {
	name  string
	isDir bool
}

type dirReadMsg struct {
	dir     string
	entries []pickEntry
	err     error
}

type fileChosenMsg struct{ path string }

type picker struct {
	dir     string
	exts    []string
	entries []pickEntry
	sel     int
	offset  int
	height  int
}

var (
	pickDirStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	pickCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
)

func newPicker(exts []string) picker {
	return picker{dir: ".", exts: exts, height: 10}
}

func (p picker) Init() tea.Cmd {
	return readDirCmd(p.dir, p.exts)
}

func readDirCmd(dir string, exts []string) tea.Cmd {
	return func() tea.Msg {
		des, err := os.ReadDir(dir)
		if err != nil {
			return dirReadMsg{dir: dir, err: err}
		}
		var entries []pickEntry
		for _, de := range des {
			name := de.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			isDir := de.IsDir()
			if de.Type()&os.ModeSymlink != 0 {
				if fi, err := os.Stat(filepath.Join(dir, name)); err == nil {
					isDir = fi.IsDir()
				}
			}
			if !isDir && !extAllowed(name, exts) {
				continue
			}
			entries = append(entries, pickEntry{name: name, isDir: isDir})
		}
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].isDir != entries[j].isDir {
				return entries[i].isDir
			}
			return entries[i].name < entries[j].name
		})
		return dirReadMsg{dir: dir, entries: entries}
	}
}

func extAllowed(name string, exts []string) bool {
	for _, e := range exts {
		if strings.EqualFold(filepath.Ext(name), e) {
			return true
		}
	}
	return false
}

func (p picker) Update(msg tea.Msg) (picker, tea.Cmd) {
	switch msg := msg.(type) {
	case dirReadMsg:
		if msg.err != nil {
			return p, func() tea.Msg { return analysisErrMsg{msg.err} }
		}
		p.dir = msg.dir
		p.entries = msg.entries
		p.sel = 0
		p.offset = 0
		return p, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if p.sel > 0 {
				p.sel--
			}
		case "down", "j":
			if p.sel < len(p.entries)-1 {
				p.sel++
			}
		case "g", "home":
			p.sel = 0
		case "G", "end":
			p.sel = len(p.entries) - 1
			if p.sel < 0 {
				p.sel = 0
			}
		case "left", "h", "backspace", "esc":
			return p, readDirCmd(filepath.Join(p.dir, ".."), p.exts)
		case "right", "l", "enter":
			if p.sel >= len(p.entries) {
				return p, nil
			}
			e := p.entries[p.sel]
			path := filepath.Join(p.dir, e.name)
			if e.isDir {
				return p, readDirCmd(path, p.exts)
			}
			return p, func() tea.Msg { return fileChosenMsg{path} }
		}
		p.scrollIntoView()
	}
	return p, nil
}

func (p *picker) scrollIntoView() {
	if p.sel < p.offset {
		p.offset = p.sel
	}
	if p.sel >= p.offset+p.height {
		p.offset = p.sel - p.height + 1
	}
}

func (p *picker) setHeight(h int) {
	if h < 1 {
		h = 1
	}
	p.height = h
	p.scrollIntoView()
}

func (p picker) View() string {
	var b strings.Builder
	b.WriteString(dimStyle.Render(p.dir) + "\n")
	if len(p.entries) == 0 {
		b.WriteString(dimStyle.Render("  (no audio files here — h to go up)"))
		return b.String()
	}
	end := p.offset + p.height
	if end > len(p.entries) {
		end = len(p.entries)
	}
	for i := p.offset; i < end; i++ {
		e := p.entries[i]
		cursor := "  "
		if i == p.sel {
			cursor = pickCursorStyle.Render("> ")
		}
		name := e.name
		if e.isDir {
			name = pickDirStyle.Render(name + "/")
		}
		fmt.Fprintf(&b, "%s%s\n", cursor, name)
	}
	return strings.TrimRight(b.String(), "\n")
}
