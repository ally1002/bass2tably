package tui

import (
	"errors"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ally1002/bass2tably/internal/app"
	"github.com/ally1002/bass2tably/internal/tab"
)

var testOpts = app.Options{Tuning: "BEADG", Grid: 16}

func keyPress(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func advance(t *testing.T, m tea.Model, msg tea.Msg) (model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	mm, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T, want model", next)
	}
	return mm, cmd
}

func TestAnalysisDoneShowsViewer(t *testing.T) {
	m := newModel("x.wav", testOpts)
	if m.screen != analyzingScreen {
		t.Fatalf("screen = %d, want analyzingScreen", m.screen)
	}
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	res := testResult(t)
	res.Samples = nil
	m, _ = advance(t, m, analysisDoneMsg{res})
	if m.screen != viewerScreen {
		t.Fatalf("screen = %d, want viewerScreen", m.screen)
	}
	if m.res != res {
		t.Fatalf("res not stored")
	}
	want := m.sheet.Text(tab.Unicode)
	if got := stripChrome(m.vp.GetContent(), m.res.Layout); got != want {
		t.Fatalf("viewport content = %q, want %q", got, want)
	}
	if m.res.Layout.Top != 1 || m.res.Layout.CellW != defaultCellW {
		t.Fatalf("Layout = %+v, want Top 1 and CellW %d", m.res.Layout, defaultCellW)
	}
	if strings.Contains(ansi.Strip(m.vp.GetContent()), "|") {
		t.Fatal("viewer drew ASCII bars with the Unicode glyph set")
	}
}

func TestASCIIConfigUsesASCIIGlyphs(t *testing.T) {
	m := newModel("x.wav", testOpts)
	m.glyphs = tab.ASCII
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	res := testResult(t)
	res.Samples = nil
	m, _ = advance(t, m, analysisDoneMsg{res})
	if got := stripChrome(m.vp.GetContent(), m.res.Layout); got != m.sheet.Text(tab.ASCII) {
		t.Fatalf("viewport content = %q, want the ASCII text", got)
	}
}

func TestAnalysisErrReturnsToPicker(t *testing.T) {
	m := newModel("x.wav", testOpts)
	wantErr := errors.New("boom")
	m, cmd := advance(t, m, analysisErrMsg{wantErr})
	if m.screen != pickerScreen {
		t.Fatalf("screen = %d, want pickerScreen", m.screen)
	}
	if m.err != wantErr {
		t.Fatalf("err = %v, want %v", m.err, wantErr)
	}
	if cmd == nil {
		t.Fatal("expected a command (picker init + error auto-clear), got nil")
	}
	m, _ = advance(t, m, clearErrMsg{})
	if m.err != nil {
		t.Fatalf("err = %v after clearErrMsg, want nil", m.err)
	}
}

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "bass2tably-tui-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_STATE_HOME", dir)
	os.Setenv("XDG_CACHE_HOME", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func TestQuitKey(t *testing.T) {
	m := newModel("", testOpts)
	_, cmd := advance(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("expected tea.Quit, got nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("cmd() = %T, want tea.QuitMsg", cmd())
	}
}

func TestEscReturnsToPicker(t *testing.T) {
	m := newModel("x.wav", testOpts)
	res := testResult(t)
	res.Samples = nil
	m, _ = advance(t, m, analysisDoneMsg{res})
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.screen != pickerScreen {
		t.Fatalf("screen = %d, want pickerScreen", m.screen)
	}
	if m.res != nil {
		t.Fatal("res should be cleared when leaving the viewer")
	}
}

func TestHelpToggle(t *testing.T) {
	m := newModel("x.wav", testOpts)
	res := testResult(t)
	res.Samples = nil
	m, _ = advance(t, m, analysisDoneMsg{res})
	m, _ = advance(t, m, keyPress('?'))
	if !m.help.ShowAll {
		t.Fatal("help.ShowAll = false after ?, want true")
	}
	m, _ = advance(t, m, keyPress('?'))
	if m.help.ShowAll {
		t.Fatal("help.ShowAll = true after second ?, want false")
	}
}

func TestLettersInFinderFilterInsteadOfQuitting(t *testing.T) {
	m := newModel("", testOpts)
	m, _ = advance(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, cmd := advance(t, m, keyPress('q'))
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("q quit from the finder instead of filtering")
		}
	}
	m, _ = advance(t, m, keyPress('t'))
	if m.finder.input.Value() != "qt" {
		t.Fatalf("finder input = %q, want qt", m.finder.input.Value())
	}
	if m.themeIdx != 0 {
		t.Fatal("t cycled the theme from inside the finder")
	}
	m, _ = advance(t, m, tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	if m.themeIdx != 1 {
		t.Fatal("ctrl+t did not cycle the theme")
	}
	if !strings.Contains(ansi.Strip(m.pickerView()), "qt") {
		t.Fatal("the finder view does not show the typed filter")
	}
}
