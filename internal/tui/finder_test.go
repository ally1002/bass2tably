package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

var wavExts = []string{".wav", ".m4a", ".mp3", ".ogg", ".flac"}

func readTestDir(t *testing.T, dir string) dirReadMsg {
	t.Helper()
	msg, ok := readDirCmd(dir, wavExts)().(dirReadMsg)
	if !ok {
		t.Fatal("readDirCmd did not return dirReadMsg")
	}
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	return msg
}

func setupDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range []string{"riff.wav", "take.M4A", "go.sum", "main.go", "README.md", ".hidden.wav"} {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "nested.flac"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func finderAt(t *testing.T, dir string) finder {
	t.Helper()
	f := newFinder(wavExts)
	f.start = dir
	f, _ = f.Update(readTestDir(t, dir))
	return f
}

func TestBrowseListsOnlyDirsAndAudio(t *testing.T) {
	dir := setupDir(t)
	msg := readTestDir(t, dir)
	want := []fileEntry{
		{name: "sub", path: filepath.Join(dir, "sub"), isDir: true},
		{name: "riff.wav", path: filepath.Join(dir, "riff.wav")},
		{name: "take.M4A", path: filepath.Join(dir, "take.M4A")},
	}
	if len(msg.entries) != len(want) {
		t.Fatalf("entries = %+v, want %+v", msg.entries, want)
	}
	for i := range want {
		if msg.entries[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, msg.entries[i], want[i])
		}
	}
}

func TestBrowseSelectFileEmitsChosen(t *testing.T) {
	dir := setupDir(t)
	f := finderAt(t, dir)
	f, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if cmd != nil {
		if _, ok := cmd().(previewMsg); !ok {
			t.Fatal("moving onto a wav file may only request its preview")
		}
	}
	f, cmd = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on a file should produce a command")
	}
	chosen, ok := cmd().(fileChosenMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want fileChosenMsg", cmd())
	}
	if want := filepath.Join(dir, "riff.wav"); chosen.path != want {
		t.Fatalf("chosen = %q, want %q", chosen.path, want)
	}
}

func TestBrowseDescendsAndGoesUp(t *testing.T) {
	dir := setupDir(t)
	f := finderAt(t, dir)
	f, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on a directory should produce a command")
	}
	sub, ok := cmd().(dirReadMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want dirReadMsg", cmd())
	}
	f, _ = f.Update(sub)
	if len(f.browse) != 1 || f.browse[0].name != "nested.flac" {
		t.Fatalf("sub entries = %+v, want just nested.flac", f.browse)
	}
	f, cmd = f.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if cmd == nil {
		t.Fatal("backspace with an empty filter should go up")
	}
	up := cmd().(dirReadMsg)
	if up.dir != dir {
		t.Fatalf("went up to %q, want %q", up.dir, dir)
	}
	f, cmd = f.Update(keyPress('h'))
	if cmd == nil {
		t.Fatal("h with an empty filter should go up")
	}
	if _, ok := cmd().(dirReadMsg); !ok {
		t.Fatal("h did not read a directory")
	}
}

func TestBrowseErrorSurfacesOnPickerScreen(t *testing.T) {
	f := newFinder(wavExts)
	msg := readDirCmd(filepath.Join(t.TempDir(), "missing"), wavExts)().(dirReadMsg)
	if msg.err == nil {
		t.Fatal("expected an error for a missing directory")
	}
	_, cmd := f.Update(msg)
	if cmd == nil {
		t.Fatal("expected an error command")
	}
	errMsg, ok := cmd().(analysisErrMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want analysisErrMsg", cmd())
	}
	m := newModel("", testOpts)
	m, cmd2 := advance(t, m, errMsg)
	if m.err == nil {
		t.Fatal("model err not set from finder error")
	}
	if cmd2 == nil {
		t.Fatal("expected auto-clear command")
	}
}

func setupTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mk := func(rel string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 6; i++ {
		mk(fmt.Sprintf("take%d.wav", i))
	}
	for i := 0; i < 4; i++ {
		mk(fmt.Sprintf("takes/sub/deep%d.wav", i))
	}
	mk("notes.txt")
	mk(".hidden/secret.wav")
	mk("node_modules/pkg/bundled.wav")
	mk("a/b/c/d/e/f/toodeep.wav")
	mk("a/b/c/edge.wav")
	mk("a/b/c/d/past.wav")
	return root
}

func TestScanBounds(t *testing.T) {
	root := setupTree(t)
	msg := scanCmd(root, wavExts, 7)().(scanDoneMsg)
	if msg.gen != 7 || msg.root != root {
		t.Fatalf("msg = gen %d root %q", msg.gen, msg.root)
	}
	var names []string
	for _, e := range msg.files {
		names = append(names, e.name)
		if e.size != 1 {
			t.Errorf("%s: size %d, want 1", e.name, e.size)
		}
	}
	got := strings.Join(names, " ")
	for _, banned := range []string{"secret", "bundled", "toodeep", "past.wav", "notes.txt"} {
		if strings.Contains(got, banned) {
			t.Errorf("scan listed %s: %s", banned, got)
		}
	}
	if !strings.Contains(got, filepath.Join("a", "b", "c", "edge.wav")) {
		t.Errorf("a file at depth 4 was skipped: %s", got)
	}
	if len(msg.files) != 11 {
		t.Errorf("scan listed %d files, want 11: %s", len(msg.files), got)
	}
}

func TestScanStaleGenerationDropped(t *testing.T) {
	f := newFinder(wavExts)
	f.scanGen = 3
	f, _ = f.Update(scanDoneMsg{gen: 2, root: "/x", files: []fileEntry{{name: "stale.wav"}}})
	if f.scanned != nil {
		t.Fatal("a stale scan result was accepted")
	}
	f, _ = f.Update(scanDoneMsg{gen: 3, root: "/x", files: []fileEntry{{name: "fresh.wav"}}})
	if len(f.scanned) != 1 || f.scanned[0].name != "fresh.wav" {
		t.Fatalf("scanned = %+v, want fresh.wav", f.scanned)
	}
}

func typeString(f finder, s string) (finder, []tea.Cmd) {
	var cmds []tea.Cmd
	for _, r := range s {
		var cmd tea.Cmd
		f, cmd = f.Update(keyPress(r))
		cmds = append(cmds, cmd)
	}
	return f, cmds
}

func TestTypingFiltersScannedFiles(t *testing.T) {
	root := setupTree(t)
	f := finderAt(t, root)
	f, cmds := typeString(f, "de")
	if !f.filtering() || f.input.Value() != "de" {
		t.Fatalf("input = %q after typing de", f.input.Value())
	}
	if !f.scanning || f.scanGen != 1 {
		t.Fatalf("typing did not start exactly one scan: scanning %v gen %d", f.scanning, f.scanGen)
	}
	var scan scanDoneMsg
	found := false
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		msg := cmd()
		if b, ok := msg.(tea.BatchMsg); ok {
			for _, c := range b {
				if c == nil {
					continue
				}
				if s, ok := c().(scanDoneMsg); ok {
					scan, found = s, true
				}
			}
		} else if s, ok := msg.(scanDoneMsg); ok {
			scan, found = s, true
		}
	}
	if !found {
		t.Fatal("no scan command was issued")
	}
	f, _ = f.Update(scan)
	if f.scanning {
		t.Fatal("still scanning after the result arrived")
	}
	if len(f.hits) != 5 {
		t.Fatalf("%d hits for de, want the four deep takes and edge.wav: %+v", len(f.hits), f.hits)
	}
	for _, h := range f.hits[:4] {
		if !strings.Contains(h.entry.name, "deep") {
			t.Errorf("hit %q ranked above the deep takes", h.entry.name)
		}
	}
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if f.filtering() || len(f.entries()) != len(f.browse) {
		t.Fatal("esc did not clear the filter back to browse mode")
	}
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if f.dir != root {
		t.Fatal("a second esc changed the directory")
	}
}

func TestTypingHDoesNotGoUp(t *testing.T) {
	root := setupTree(t)
	f := finderAt(t, root)
	f, _ = typeString(f, "s")
	f, cmd := f.Update(keyPress('h'))
	if f.input.Value() != "sh" {
		t.Fatalf("input = %q, want sh", f.input.Value())
	}
	if cmd != nil {
		if _, ok := cmd().(dirReadMsg); ok {
			t.Fatal("h inside a filter read a directory")
		}
	}
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if f.input.Value() != "s" {
		t.Fatalf("backspace inside a filter edited to %q, want s", f.input.Value())
	}
}

func TestRecentSectionOnTop(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := setupTree(t)
	pushRecent(filepath.Join(root, "take3.wav"))
	f := finderAt(t, root)
	f, _ = f.Update(recentCmd()())
	es := f.entries()
	if len(es) != len(f.browse)+1 || !es[0].recent || filepath.Base(es[0].path) != "take3.wav" {
		t.Fatalf("entries = %+v, want the recent take first", es)
	}
	f.setSize(80, 20)
	view := ansi.Strip(f.View(testStyles()))
	if !strings.Contains(view, "recent") || strings.Index(view, "recent") > strings.Index(view, "take0.wav") {
		t.Fatalf("recent section missing or below the files:\n%s", view)
	}
	sub, _ := f.Update(readTestDir(t, filepath.Join(root, "takes")))
	if sub.showRecent() {
		t.Fatal("recent section shown outside the start directory")
	}
}

func TestFinderTwoPanesAt90Columns(t *testing.T) {
	root := setupTree(t)
	f := finderAt(t, root)
	st := testStyles()
	f.setSize(89, 12)
	one := ansi.Strip(f.View(st))
	if strings.Contains(one, "│") {
		t.Fatalf("89 columns drew a pane separator:\n%s", one)
	}
	f.setSize(90, 12)
	two := f.View(st)
	if lipgloss_width := ansi.StringWidth(strings.Split(two, "\n")[0]); lipgloss_width != 90 {
		t.Fatalf("two-pane first row is %d wide, want 90", lipgloss_width)
	}
	plain := ansi.Strip(two)
	if !strings.Contains(plain, "│") {
		t.Fatalf("90 columns drew no pane separator:\n%s", plain)
	}
	if !strings.Contains(plain, "directory") {
		t.Fatalf("preview pane lacks the selected directory's card:\n%s", plain)
	}
	f.move(3)
	plain = ansi.Strip(f.View(st))
	if !strings.Contains(plain, "take0.wav") || !strings.Contains(plain, "reading…") {
		t.Fatalf("preview pane lacks the selected file's card:\n%s", plain)
	}
	rows := strings.Split(two, "\n")
	if len(rows) != 12 {
		t.Fatalf("two-pane view is %d rows tall, want 12", len(rows))
	}
}

func TestPreviewCardShowsWavFacts(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "take.wav")
	writeTestWav(t, path, 44100, 22050)
	f := newFinder(wavExts)
	f.start = root
	f, cmd := f.Update(readTestDir(t, root))
	if cmd == nil {
		t.Fatal("selecting a wav did not request a preview")
	}
	if f.requestPreview() != nil {
		t.Fatal("a second request was issued while the first is pending")
	}
	msg := cmd().(previewMsg)
	f, _ = f.Update(msg)
	f.setSize(100, 12)
	plain := ansi.Strip(f.View(testStyles()))
	for _, want := range []string{"take.wav", "44.1 kHz", "mono", "16-bit", "0:00.5"} {
		if !strings.Contains(plain, want) {
			t.Errorf("preview lacks %q:\n%s", want, plain)
		}
	}
	if !strings.ContainsAny(plain, blocks) {
		t.Errorf("preview has no mini waveform:\n%s", plain)
	}
	if f.requestPreview() != nil {
		t.Fatal("a cached preview was requested again")
	}
}

func TestPreviewNonWavSaysFfmpeg(t *testing.T) {
	root := setupDir(t)
	f := finderAt(t, root)
	f.setSize(100, 12)
	f.move(2)
	if e, _ := f.selected(); e.name != "take.M4A" {
		t.Fatalf("selected %q, want take.M4A", e.name)
	}
	if f.requestPreview() != nil {
		t.Fatal("a preview was requested for a non-wav file")
	}
	plain := ansi.Strip(f.View(testStyles()))
	if !strings.Contains(plain, "M4A") || !strings.Contains(plain, "ffmpeg") {
		t.Fatalf("non-wav preview:\n%s", plain)
	}
}

func TestFinderFillsItsHeight(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := setupTree(t)
	pushRecent(filepath.Join(root, "take3.wav"))
	st := testStyles()
	for _, size := range [][2]int{{70, 18}, {70, 6}, {100, 15}, {100, 4}} {
		for _, withRecent := range []bool{false, true} {
			f := finderAt(t, root)
			if withRecent {
				f, _ = f.Update(recentCmd()())
			}
			f.setSize(size[0], size[1])
			view := f.View(st)
			rows := strings.Split(view, "\n")
			if len(rows) != size[1] {
				t.Errorf("%dx%d recent %v: finder view is %d rows tall:\n%s", size[0], size[1], withRecent, len(rows), ansi.Strip(view))
			}
			for i, row := range rows {
				if w := ansi.StringWidth(row); w > size[0] {
					t.Errorf("%dx%d recent %v: row %d is %d wide", size[0], size[1], withRecent, i, w)
				}
			}
		}
	}
}
