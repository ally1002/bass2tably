package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
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

func TestPickerListsOnlyDirsAndAudio(t *testing.T) {
	dir := setupDir(t)
	msg := readTestDir(t, dir)
	want := []pickEntry{
		{name: "sub", isDir: true},
		{name: "riff.wav"},
		{name: "take.M4A"},
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

func TestPickerSelectFileEmitsChosen(t *testing.T) {
	dir := setupDir(t)
	p := newPicker(wavExts)
	p, _ = p.Update(readTestDir(t, dir))

	p, cmd := p.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if cmd != nil {
		t.Fatal("moving the cursor should not produce a command")
	}
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
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

func TestPickerDescendsIntoDir(t *testing.T) {
	dir := setupDir(t)
	p := newPicker(wavExts)
	p, _ = p.Update(readTestDir(t, dir))

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on a directory should produce a command")
	}
	sub, ok := cmd().(dirReadMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want dirReadMsg", cmd())
	}
	if sub.err != nil {
		t.Fatal(sub.err)
	}
	p, _ = p.Update(sub)
	if len(p.entries) != 1 || p.entries[0].name != "nested.flac" {
		t.Fatalf("sub entries = %+v, want just nested.flac", p.entries)
	}
}

func TestPickerErrorSurfacesOnPickerScreen(t *testing.T) {
	p := newPicker(wavExts)
	msg := readDirCmd(filepath.Join(t.TempDir(), "missing"), wavExts)().(dirReadMsg)
	if msg.err == nil {
		t.Fatal("expected an error for a missing directory")
	}
	_, cmd := p.Update(msg)
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
		t.Fatal("model err not set from picker error")
	}
	if cmd2 == nil {
		t.Fatal("expected auto-clear command")
	}
}
