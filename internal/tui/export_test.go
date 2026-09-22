package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"bass2tably/internal/app"
)

func exportModel(t *testing.T, res *app.Result, dir string) model {
	t.Helper()
	res.Path = filepath.Join(dir, "shh.wav")
	res.Samples = nil
	m := newModel(res.Path, testOpts)
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 120, Height: 30})
	m, _ = advance(t, m, analysisDoneMsg{res})
	return m
}

func runExport(t *testing.T, m model) (model, flashMsg) {
	t.Helper()
	m, cmd := advance(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.exp.open {
		t.Fatal("enter did not close the export dialog")
	}
	if cmd == nil {
		t.Fatal("enter produced no export command")
	}
	msg, ok := cmd().(flashMsg)
	if !ok {
		t.Fatalf("export returned %#v", cmd())
	}
	m, tick := advance(t, m, msg)
	if tick == nil {
		t.Fatal("a flash did not schedule its own clearing")
	}
	return m, msg
}

func TestExportDialogWritesBothNextToInput(t *testing.T) {
	res, err := app.Analyze(filepath.Join("..", "..", "testdata", "shh.wav"), app.Options{Tuning: "BEADG", Grid: 16})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	m := exportModel(t, res, dir)
	m, _ = advance(t, m, keyPress('e'))
	if !m.exp.open || m.exp.gp5 || m.exp.mid || m.exp.sel != 0 {
		t.Fatalf("dialog after e: %+v, want nothing ticked and the first row highlighted", m.exp)
	}
	if want := filepath.Join(dir, "shh"); m.exp.path.Value() != want {
		t.Fatalf("default path %q, want %q", m.exp.path.Value(), want)
	}
	view := ansi.Strip(m.viewerView())
	for _, want := range []string{"Export", "[ ] Guitar Pro 5", "[ ] MIDI", "shh.gp5", "shh.mid", "to ›"} {
		if !strings.Contains(view, want) {
			t.Errorf("dialog lacks %q:\n%s", want, view)
		}
	}
	if h := strings.Count(view, "\n") + 1; h != m.h {
		t.Fatalf("overlay changed the view height to %d", h)
	}
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !m.exp.gp5 || !m.exp.mid {
		t.Fatalf("both formats should be ticked: %+v", m.exp)
	}
	if !strings.Contains(ansi.Strip(m.exportView()), "[x] Guitar Pro 5") {
		t.Fatal("dialog does not show Guitar Pro 5 ticked")
	}

	m, msg := runExport(t, m)
	if msg.err {
		t.Fatalf("export failed: %s", msg.text)
	}
	band := ansi.Strip(m.statusView())
	if !strings.Contains(band, "wrote shh.gp5") || !strings.Contains(band, "shh.mid") {
		t.Fatalf("band %q lacks the export report", band)
	}
	got, err := os.ReadFile(filepath.Join(dir, "shh.gp5"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("..", "..", "testdata", "shh.gp5"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("shh.gp5 from the viewer (%d bytes) differs from testdata/shh.gp5 (%d bytes)", len(got), len(want))
	}
	if fi, err := os.Stat(filepath.Join(dir, "shh.mid")); err != nil || fi.Size() == 0 {
		t.Fatalf("shh.mid missing or empty: %v", err)
	}
	m, _ = advance(t, m, clearFlashMsg{})
	if strings.Contains(ansi.Strip(m.statusView()), "wrote") {
		t.Fatal("the flash did not clear")
	}
}

func TestExportDialogChoosesFormatAndPath(t *testing.T) {
	dir := t.TempDir()
	m := exportModel(t, testResult(t), dir)
	m, _ = advance(t, m, keyPress('e'))
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !m.exp.gp5 || m.exp.mid {
		t.Fatalf("space on the first row: %+v, want only Guitar Pro 5 ticked", m.exp)
	}
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.exp.sel != 2 {
		t.Fatalf("sel = %d after tab, want 2 (path)", m.exp.sel)
	}
	for range "shh" {
		m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	for _, r := range "two takes" {
		m, _ = advance(t, m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if want := filepath.Join(dir, "two takes"); m.exp.path.Value() != want {
		t.Fatalf("path %q after editing, want %q", m.exp.path.Value(), want)
	}
	m, msg := runExport(t, m)
	if msg.err || !strings.Contains(msg.text, "two takes.gp5") || strings.Contains(msg.text, ".mid") {
		t.Fatalf("flash %q", msg.text)
	}
	if _, err := os.Stat(filepath.Join(dir, "two takes.gp5")); err != nil {
		t.Fatal("the renamed .gp5 was not written")
	}
	if _, err := os.Stat(filepath.Join(dir, "two takes.mid")); err == nil {
		t.Fatal("a .mid was written although unticked")
	}
	if _, err := os.Stat(filepath.Join(dir, "shh.gp5")); err == nil {
		t.Fatal("the default path was written as well")
	}
}

func TestExportDialogNeedsAFormatAndCancels(t *testing.T) {
	dir := t.TempDir()
	m := exportModel(t, testResult(t), dir)
	m, _ = advance(t, m, keyPress('e'))
	m, cmd := advance(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.exp.open || cmd != nil {
		t.Fatal("enter with no format ticked closed the dialog or exported")
	}
	if !strings.Contains(ansi.Strip(m.exportView()), "pick at least one format") {
		t.Fatal("dialog does not ask for a format")
	}
	m, _ = advance(t, m, keyPress('q'))
	if m.exp.open != true || m.screen != viewerScreen {
		t.Fatal("q inside the dialog quit or left the viewer")
	}
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.exp.open || m.screen != viewerScreen {
		t.Fatal("esc did not close the dialog in place")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("cancelled export wrote %d files", len(entries))
	}
	if got := exportBase("  ~/takes/riff.GP5 "); !strings.HasSuffix(got, filepath.Join("takes", "riff")) || strings.HasPrefix(got, "~") {
		t.Fatalf("exportBase = %q", got)
	}
}

func TestExportErrorShowsInBand(t *testing.T) {
	m := exportModel(t, testResult(t), filepath.Join(t.TempDir(), "missing"))
	m, _ = advance(t, m, keyPress('e'))
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	m, msg := runExport(t, m)
	if !msg.err || !m.flashErr {
		t.Fatalf("export into a missing directory: %+v", msg)
	}
	if !strings.Contains(ansi.Strip(m.statusView()), "missing") {
		t.Fatalf("band does not show the error: %q", ansi.Strip(m.statusView()))
	}
}
