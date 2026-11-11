package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ally1002/bass2tably/internal/app"
	"github.com/ally1002/bass2tably/internal/wavio"
)

const (
	scanMaxDepth   = 4
	scanMaxFiles   = 5000
	previewMaxSize = 64 << 20
	previewBins    = 256
	twoPaneWidth   = 90
)

type fileEntry struct {
	name   string
	path   string
	isDir  bool
	size   int64
	recent bool
}

type dirReadMsg struct {
	dir     string
	entries []fileEntry
	err     error
}

type scanDoneMsg struct {
	gen   int
	root  string
	files []fileEntry
}

type recentMsg struct{ files []fileEntry }

type preview struct {
	info wavio.Info
	err  error
	pk   []float64
	wav  bool
}

type previewMsg struct {
	path string
	pv   preview
}

type fileChosenMsg struct{ path string }

type hit struct {
	entry fileEntry
	score int
	pos   []int
}

type finder struct {
	dir      string
	start    string
	exts     []string
	input    queryInput
	browse   []fileEntry
	recent   []fileEntry
	scanned  []fileEntry
	scanRoot string
	scanGen  int
	scanning bool
	hits     []hit
	sel      int
	offset   int
	width    int
	height   int
	ascii    bool
	opts     app.Options
	preview  map[string]preview
	pending  map[string]bool
	badges   map[string]*app.Summary
}

func newFinder(exts []string, opts app.Options) finder {
	in := queryInput{prompt: "› ", placeholder: "type to find a take"}
	start, err := filepath.Abs(".")
	if err != nil {
		start = "."
	}
	return finder{
		dir:     start,
		start:   start,
		exts:    exts,
		input:   in,
		height:  10,
		width:   80,
		opts:    opts,
		preview: map[string]preview{},
		pending: map[string]bool{},
		badges:  map[string]*app.Summary{},
	}
}

func (f finder) Init() tea.Cmd {
	return tea.Batch(readDirCmd(f.dir, f.exts), recentCmd())
}

func recentCmd() tea.Cmd {
	return func() tea.Msg {
		var files []fileEntry
		home, _ := os.UserHomeDir()
		for _, p := range loadRecent() {
			fi, err := os.Stat(p)
			if err != nil {
				continue
			}
			name := p
			if home != "" && strings.HasPrefix(p, home) {
				name = "~" + strings.TrimPrefix(p, home)
			}
			files = append(files, fileEntry{name: name, path: p, size: fi.Size(), recent: true})
		}
		return recentMsg{files}
	}
}

func readDirCmd(dir string, exts []string) tea.Cmd {
	return func() tea.Msg {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return dirReadMsg{dir: dir, err: err}
		}
		des, err := os.ReadDir(abs)
		if err != nil {
			return dirReadMsg{dir: abs, err: err}
		}
		var entries []fileEntry
		for _, de := range des {
			name := de.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			isDir := de.IsDir()
			var size int64
			full := filepath.Join(abs, name)
			fi, statErr := os.Stat(full)
			if de.Type()&os.ModeSymlink != 0 && statErr == nil {
				isDir = fi.IsDir()
			}
			if !isDir && !extAllowed(name, exts) {
				continue
			}
			if !isDir && statErr == nil {
				size = fi.Size()
			}
			entries = append(entries, fileEntry{name: name, path: full, isDir: isDir, size: size})
		}
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].isDir != entries[j].isDir {
				return entries[i].isDir
			}
			return entries[i].name < entries[j].name
		})
		return dirReadMsg{dir: abs, entries: entries}
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

func depth(root, p string) int {
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}

func scanCmd(root string, exts []string, gen int) tea.Cmd {
	return func() tea.Msg {
		var files []fileEntry
		filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if p != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules" || depth(root, p) >= scanMaxDepth) {
					return filepath.SkipDir
				}
				return nil
			}
			if extAllowed(d.Name(), exts) {
				files = append(files, newFileEntry(root, p, d))
			}
			if len(files) >= scanMaxFiles {
				return filepath.SkipAll
			}
			return nil
		})
		return scanDoneMsg{gen, root, files}
	}
}

func newFileEntry(root, p string, d fs.DirEntry) fileEntry {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		rel = p
	}
	e := fileEntry{name: rel, path: p}
	if fi, err := d.Info(); err == nil {
		e.size = fi.Size()
	}
	return e
}

func previewCmd(e fileEntry) tea.Cmd {
	return func() tea.Msg {
		pv := preview{wav: true}
		pv.info, pv.err = wavio.ReadInfo(e.path)
		if pv.err == nil && e.size <= previewMaxSize {
			if clip, err := wavio.Load(e.path); err == nil {
				pv.pk = peaks(clip.Samples, previewBins)
			}
		}
		return previewMsg{e.path, pv}
	}
}

func isWav(name string) bool { return strings.EqualFold(filepath.Ext(name), ".wav") }

func (f finder) filtering() bool { return f.input.Value() != "" }

func (f finder) showRecent() bool { return !f.filtering() && f.dir == f.start && len(f.recent) > 0 }

func (f finder) entries() []fileEntry {
	if f.filtering() {
		out := make([]fileEntry, len(f.hits))
		for i, h := range f.hits {
			out[i] = h.entry
		}
		return out
	}
	if f.showRecent() {
		return append(append([]fileEntry{}, f.recent...), f.browse...)
	}
	return f.browse
}

func (f finder) selected() (fileEntry, bool) {
	es := f.entries()
	if f.sel < 0 || f.sel >= len(es) {
		return fileEntry{}, false
	}
	return es[f.sel], true
}

func (f *finder) rank() {
	f.hits = f.hits[:0]
	pattern := f.input.Value()
	recent := map[string]bool{}
	for _, r := range f.recent {
		recent[r.path] = true
	}
	for _, e := range f.scanned {
		score, pos, ok := fuzzy(pattern, e.name)
		if !ok {
			continue
		}
		if recent[e.path] {
			score += 4
			e.recent = true
		}
		f.hits = append(f.hits, hit{e, score, pos})
	}
	sort.SliceStable(f.hits, func(i, j int) bool {
		a, b := f.hits[i], f.hits[j]
		if a.score != b.score {
			return a.score > b.score
		}
		if len(a.entry.name) != len(b.entry.name) {
			return len(a.entry.name) < len(b.entry.name)
		}
		return a.entry.name < b.entry.name
	})
	f.clampSel()
}

func (f *finder) clampSel() {
	n := len(f.entries())
	if f.sel > n-1 {
		f.sel = n - 1
	}
	if f.sel < 0 {
		f.sel = 0
	}
	f.scrollIntoView()
}

func (f *finder) startScan() tea.Cmd {
	f.scanGen++
	f.scanning = true
	f.scanRoot = f.dir
	return scanCmd(f.dir, f.exts, f.scanGen)
}

func (f *finder) afterQuery() tea.Cmd {
	if !f.filtering() {
		f.hits = f.hits[:0]
		f.clampSel()
		return f.requestPreview()
	}
	var cmd tea.Cmd
	if f.scanRoot != f.dir || (f.scanned == nil && !f.scanning) {
		cmd = f.startScan()
	}
	f.rank()
	return tea.Batch(cmd, f.requestPreview())
}

func (f *finder) requestPreview() tea.Cmd {
	e, ok := f.selected()
	if !ok || e.isDir || !isWav(e.name) || f.pending[e.path] {
		return nil
	}
	if _, done := f.preview[e.path]; done {
		return nil
	}
	f.pending[e.path] = true
	return previewCmd(e)
}

func (f *finder) move(delta int) {
	f.sel += delta
	f.clampSel()
}

func (f *finder) goTo(dir string) tea.Cmd {
	return readDirCmd(dir, f.exts)
}

func (f *finder) open() tea.Cmd {
	e, ok := f.selected()
	if !ok {
		return nil
	}
	if e.isDir {
		f.input.SetValue("")
		return f.goTo(e.path)
	}
	return func() tea.Msg { return fileChosenMsg{e.path} }
}

func (f finder) Update(msg tea.Msg) (finder, tea.Cmd) {
	switch msg := msg.(type) {
	case dirReadMsg:
		if msg.err != nil {
			return f, func() tea.Msg { return analysisErrMsg{msg.err} }
		}
		f.dir = msg.dir
		f.browse = msg.entries
		clear(f.badges)
		f.scanned = nil
		f.scanRoot = ""
		f.scanning = false
		f.sel = 0
		f.offset = 0
		if f.filtering() {
			cmd := f.startScan()
			f.rank()
			return f, cmd
		}
		return f, f.requestPreview()
	case recentMsg:
		f.recent = msg.files
		f.clampSel()
		return f, f.requestPreview()
	case scanDoneMsg:
		if msg.gen != f.scanGen {
			return f, nil
		}
		f.scanning = false
		f.scanned = msg.files
		f.scanRoot = msg.root
		f.rank()
		return f, f.requestPreview()
	case previewMsg:
		delete(f.pending, msg.path)
		f.preview[msg.path] = msg.pv
		return f, nil
	case tea.KeyPressMsg:
		return f.key(msg)
	}
	return f, nil
}

func (f finder) key(msg tea.KeyPressMsg) (finder, tea.Cmd) {
	empty := !f.filtering()
	s := msg.String()
	switch s {
	case "up":
		f.move(-1)
	case "down":
		f.move(1)
	case "pgup", "ctrl+u":
		f.move(-f.height)
	case "pgdown", "ctrl+d":
		f.move(f.height)
	case "home":
		f.move(-len(f.entries()))
	case "end":
		f.move(len(f.entries()))
	case "enter", "right":
		return f, f.open()
	case "esc":
		if empty {
			return f, nil
		}
		f.input.SetValue("")
		f.sel = 0
		return f, f.afterQuery()
	case "~":
		if home, err := os.UserHomeDir(); err == nil {
			return f, f.goTo(home)
		}
		return f, nil
	case "left":
		return f, f.goTo(filepath.Dir(f.dir))
	default:
		if empty && (s == "backspace" || s == "h") {
			return f, f.goTo(filepath.Dir(f.dir))
		}
		if !f.input.key(msg) {
			return f, nil
		}
		f.sel = 0
		return f, f.afterQuery()
	}
	return f, f.requestPreview()
}

type queryInput struct {
	value       string
	prompt      string
	placeholder string
}

func (q queryInput) Value() string { return q.value }

func (q *queryInput) SetValue(s string) { q.value = s }

func (q *queryInput) key(msg tea.KeyPressMsg) bool {
	if msg.String() == "backspace" {
		if q.value == "" {
			return false
		}
		r := []rune(q.value)
		q.value = string(r[:len(r)-1])
		return true
	}
	if msg.Text == "" || msg.Mod&(tea.ModCtrl|tea.ModAlt|tea.ModSuper) != 0 {
		return false
	}
	q.value += msg.Text
	return true
}

func (q queryInput) View(st styles) string {
	return q.view(st.accent, st.fg, st.faint, st.cursor)
}

func (q queryInput) view(prompt, text, placeholder, cursor lipgloss.Style) string {
	block := cursor.Render(" ")
	if q.value == "" {
		return prompt.Render(q.prompt) + block + placeholder.Render(q.placeholder)
	}
	return prompt.Render(q.prompt) + text.Render(q.value) + block
}

func (f *finder) scrollIntoView() {
	if f.sel < f.offset {
		f.offset = f.sel
	}
	if f.sel >= f.offset+f.listHeight() {
		f.offset = f.sel - f.listHeight() + 1
	}
	if f.offset < 0 {
		f.offset = 0
	}
}

func (f *finder) setSize(w, h int) {
	if h < 1 {
		h = 1
	}
	if w < 1 {
		w = 1
	}
	f.height = h
	f.width = w
	f.scrollIntoView()
}

func (f finder) twoPanes() bool { return f.width >= twoPaneWidth }

func (f finder) listWidth() int {
	if f.twoPanes() {
		return f.width * 3 / 5
	}
	return f.width
}

func (f finder) previewWidth() int {
	if f.twoPanes() {
		return f.width - f.listWidth() - 1
	}
	return f.width
}

func (f finder) stackedPreviewRows() int {
	if f.twoPanes() || f.height < 8 {
		return 0
	}
	return 3
}

func (f finder) listHeight() int {
	h := f.height - f.stackedPreviewRows()
	if f.showRecent() {
		h -= 2
	}
	return max(h, 1)
}

func (f finder) breadcrumb(st styles) string {
	abs := f.dir
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(abs, home) {
		abs = "~" + strings.TrimPrefix(abs, home)
	}
	parent, last := filepath.Split(abs)
	if last == "" {
		return st.accent.Bold(true).Render(abs)
	}
	sep := st.frame.Render(" ▸ ")
	crumbs := strings.Split(strings.Trim(parent, string(filepath.Separator)), string(filepath.Separator))
	var parts []string
	if strings.HasPrefix(parent, string(filepath.Separator)) {
		parts = append(parts, st.muted.Render("/"))
	}
	for _, c := range crumbs {
		if c != "" {
			parts = append(parts, st.muted.Render(c))
		}
	}
	parts = append(parts, st.accent.Bold(true).Render(last))
	return ansi.TruncateLeft(strings.Join(parts, sep), max(0, lipgloss.Width(strings.Join(parts, sep))-f.width+1), "…")
}

func (f finder) status() string {
	switch {
	case f.filtering() && f.scanning && len(f.hits) == 0:
		return "scanning…"
	case f.filtering():
		return fmt.Sprintf("%d matches", len(f.hits))
	default:
		return fmt.Sprintf("%d items", len(f.browse))
	}
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	units := "KMGT"
	v := float64(n)
	i := -1
	for v >= unit && i < len(units)-1 {
		v /= unit
		i++
	}
	return fmt.Sprintf("%.1f %cB", v, units[i])
}

func (f finder) View(st styles) string {
	list := f.listView(st)
	if f.twoPanes() {
		border := strings.TrimRight(strings.Repeat(st.frame.Render("│")+"\n", f.height), "\n")
		pv := lipgloss.NewStyle().Width(f.previewWidth()).Height(f.height).Render(f.previewView(st, f.previewWidth()))
		return lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(f.listWidth()).Render(list), border, pv)
	}
	if rows := f.stackedPreviewRows(); rows > 0 {
		lines := strings.Split(f.previewView(st, f.width), "\n")
		if len(lines) > rows {
			lines = lines[:rows]
		}
		return list + "\n" + lipgloss.NewStyle().Height(rows).Render(strings.Join(lines, "\n"))
	}
	return list
}

func (f finder) listView(st styles) string {
	w := f.listWidth()
	var lines []string
	es := f.entries()
	if len(es) == 0 {
		switch {
		case f.filtering() && f.scanning:
			lines = append(lines, st.muted.Render("  scanning…"))
		case f.filtering():
			lines = append(lines, st.muted.Render("  no match for "+f.input.Value()))
			lines = append(lines, st.faint.Render("  esc clears the filter"))
		default:
			lines = append(lines, st.muted.Render("  no audio files here"))
			lines = append(lines, st.faint.Render("  backspace goes up a level · ~ jumps home · type to search below"))
		}
		return lipgloss.NewStyle().Height(f.height - f.stackedPreviewRows()).Render(strings.Join(lines, "\n"))
	}
	for _, ln := range f.visibleLines() {
		if ln.idx < 0 {
			lines = append(lines, f.sectionRow(st, ln.header, w))
			continue
		}
		var pos []int
		if f.filtering() && ln.idx < len(f.hits) {
			pos = f.hits[ln.idx].pos
		}
		lines = append(lines, f.entryRow(st, es[ln.idx], pos, ln.idx == f.sel, w))
	}
	return lipgloss.NewStyle().Height(f.height - f.stackedPreviewRows()).Render(strings.Join(lines, "\n"))
}

func (f finder) sectionRow(st styles, label string, w int) string {
	lead := st.frame.Render("── ") + st.muted.Render(label) + " "
	rest := w - lipgloss.Width(lead)
	return ansi.Truncate(lead+st.frame.Render(strings.Repeat("─", max(rest, 0))), w, "")
}

func (f finder) entryRow(st styles, e fileEntry, pos []int, selected bool, w int) string {
	bg := func(s lipgloss.Style) lipgloss.Style {
		if selected {
			return s.Background(st.pal.selBg)
		}
		return s
	}
	mark := "  "
	if selected {
		mark = st.selMark.Render("▌ ")
	}
	var name, meta string
	switch {
	case e.isDir:
		name = bg(st.pickDir).Render("▸ " + e.name + "/")
	default:
		name = bg(st.pickFile).Render("♪ ") + highlight(e.name, pos, bg(st.pickFile), bg(st.pickDir))
		meta = bg(st.muted).Render(humanSize(e.size))
		if s, ok := f.badge(e); ok {
			meta = bg(st.accent).Render("✓ "+badgeText(s)) + bg(lipgloss.NewStyle()).Render("  ") + meta
		}
		if e.recent {
			meta = bg(st.faint).Render("recent  ") + meta
		}
	}
	metaW := ansi.StringWidth(meta)
	nameMax := max(w-2-metaW-3, 4)
	name = ansi.Truncate(name, nameMax, "…")
	gap := max(w-2-ansi.StringWidth(name)-metaW-1, 1)
	return mark + name + bg(lipgloss.NewStyle()).Render(strings.Repeat(" ", gap)) + meta + bg(lipgloss.NewStyle()).Render(" ")
}

func highlight(s string, pos []int, plain, hit lipgloss.Style) string {
	if len(pos) == 0 {
		return plain.Render(s)
	}
	var w runWriter
	p, h := newPaint(plain), newPaint(hit)
	next := 0
	for i, r := range []rune(s) {
		if next < len(pos) && pos[next] == i {
			w.write(h, string(r))
			next++
			continue
		}
		w.write(p, string(r))
	}
	return w.String()
}

func fmtClock(sec float64) string {
	mm := int(sec) / 60
	return fmt.Sprintf("%d:%04.1f", mm, sec-float64(mm*60))
}

func (f finder) previewView(st styles, w int) string {
	e, ok := f.selected()
	if !ok {
		return st.faint.Render(" nothing selected")
	}
	pad := func(s string) string { return " " + ansi.Truncate(s, max(w-2, 1), "…") }
	lines := []string{pad(st.title.Render(filepath.Base(e.path)))}
	switch {
	case e.isDir:
		lines = append(lines, pad(st.muted.Render("directory")))
	case !isWav(e.name):
		lines = append(lines, pad(st.fg.Render(strings.ToUpper(strings.TrimPrefix(filepath.Ext(e.name), ".")))+st.faint.Render(" · ")+st.muted.Render(humanSize(e.size))))
		lines = append(lines, pad(st.faint.Render("decoded with ffmpeg on open")))
	default:
		pv, done := f.preview[e.path]
		switch {
		case !done:
			lines = append(lines, pad(st.muted.Render(humanSize(e.size))), pad(st.faint.Render("reading…")))
		case pv.err != nil:
			lines = append(lines, pad(st.muted.Render(humanSize(e.size))), pad(st.danger.Render(pv.err.Error())))
		default:
			ch := "mono"
			if pv.info.Channels == 2 {
				ch = "stereo"
			} else if pv.info.Channels > 2 {
				ch = fmt.Sprintf("%d ch", pv.info.Channels)
			}
			facts := fmt.Sprintf("%.1f kHz · %s · %d-bit · %s", float64(pv.info.SampleRate)/1000, ch, pv.info.BitDepth, fmtClock(pv.info.Duration))
			lines = append(lines, pad(st.fg.Render(facts)), pad(st.muted.Render(humanSize(e.size))))
			if s, ok := f.badge(e); ok {
				lines = append(lines, pad(st.accent.Render("✓ analysed · "+badgeText(s))))
			}
			if len(pv.pk) > 0 && w >= 12 {
				lines = append(lines, "", " "+waveRow(peaks(pv.pk, w-2), 0, 0, 0, st, f.ascii))
			}
		}
	}
	return strings.Join(lines, "\n")
}

type listLine struct {
	idx    int
	header string
}

func (f finder) visibleLines() []listLine {
	es := f.entries()
	end := min(f.offset+f.listHeight(), len(es))
	recentHeader := f.showRecent()
	var out []listLine
	for i := f.offset; i < end; i++ {
		if recentHeader && i == 0 {
			out = append(out, listLine{idx: -1, header: "recent"})
		}
		if recentHeader && i == len(f.recent) {
			out = append(out, listLine{idx: -1, header: filepath.Base(f.dir)})
		}
		out = append(out, listLine{idx: i})
	}
	return out
}

func (f *finder) click(x, y int) tea.Cmd {
	if x < 0 || x >= f.listWidth() || y < 0 {
		return nil
	}
	lines := f.visibleLines()
	if y >= len(lines) || lines[y].idx < 0 {
		return nil
	}
	if lines[y].idx == f.sel {
		return f.open()
	}
	f.sel = lines[y].idx
	f.scrollIntoView()
	return f.requestPreview()
}

func (f *finder) wheel(delta int) tea.Cmd {
	f.move(delta)
	return f.requestPreview()
}

func (f finder) badge(e fileEntry) (app.Summary, bool) {
	if e.isDir || f.badges == nil {
		return app.Summary{}, false
	}
	if s, seen := f.badges[e.path]; seen {
		if s == nil {
			return app.Summary{}, false
		}
		return *s, true
	}
	s, ok := app.Peek(e.path, f.opts)
	if !ok {
		f.badges[e.path] = nil
		return app.Summary{}, false
	}
	f.badges[e.path] = &s
	return s, true
}

func badgeText(s app.Summary) string {
	bpm := fmt.Sprintf("%.1f", s.BPM)
	if s.Estimated {
		bpm = "~" + bpm
	}
	return fmt.Sprintf("%d notes · %s bpm", s.Notes, bpm)
}
