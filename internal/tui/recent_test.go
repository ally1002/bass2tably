package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRecentPushDedupesAndCaps(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	files := t.TempDir()
	mk := func(name string) string {
		p := filepath.Join(files, name)
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if got := loadRecent(); got != nil {
		t.Fatalf("loadRecent before any push = %v, want nil", got)
	}
	a, b := mk("a.wav"), mk("b.wav")
	pushRecent(a)
	pushRecent(b)
	pushRecent(a)
	got := loadRecent()
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Fatalf("after a, b, a: %v, want [a b]", got)
	}

	for i := 0; i < recentMax+5; i++ {
		pushRecent(mk(fmt.Sprintf("take%02d.wav", i)))
	}
	got = loadRecent()
	if len(got) != recentMax {
		t.Fatalf("len = %d, want %d", len(got), recentMax)
	}
	if filepath.Base(got[0]) != fmt.Sprintf("take%02d.wav", recentMax+4) {
		t.Errorf("newest first: got[0] = %s", got[0])
	}

	os.Remove(got[0])
	if again := loadRecent(); len(again) != recentMax-1 || again[0] == got[0] {
		t.Errorf("a deleted file was still listed: %v", again[:2])
	}
}

func TestRecentUnwritableIsSilent(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "missing", "deeper"))
	pushRecent("whatever.wav")
	if got := loadRecent(); len(got) != 0 {
		t.Fatalf("got %v from an unwritable state dir", got)
	}
}
