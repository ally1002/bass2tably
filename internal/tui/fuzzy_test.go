package tui

import (
	"sort"
	"testing"
)

func TestFuzzyRanks(t *testing.T) {
	names := []string{"s_h_h.wav", "ahh-shh.wav", "shh.wav", "shhh.wav"}
	type ranked struct {
		name  string
		score int
	}
	var got []ranked
	for _, n := range names {
		score, pos, ok := fuzzy("shh", n)
		if !ok {
			t.Fatalf("%q did not match shh", n)
		}
		if len(pos) != 3 {
			t.Errorf("%q: %d matched positions, want 3", n, len(pos))
		}
		got = append(got, ranked{n, score})
	}
	sort.SliceStable(got, func(i, j int) bool { return got[i].score > got[j].score })
	want := []string{"shh.wav", "ahh-shh.wav", "shhh.wav", "s_h_h.wav"}
	for i, w := range want {
		if got[i].name != w {
			t.Fatalf("rank %d = %q (%d), want %q; full order %v", i, got[i].name, got[i].score, w, got)
		}
	}
	if got[1].score != got[2].score {
		t.Errorf("ahh-shh.wav (%d) and shhh.wav (%d) should tie on score", got[1].score, got[2].score)
	}

	if _, _, ok := fuzzy("xyz", "shh.wav"); ok {
		t.Error("xyz matched shh.wav")
	}
	if _, _, ok := fuzzy("wavs", "shh.wav"); ok {
		t.Error("a pattern longer than its subsequence matched")
	}
	score, pos, ok := fuzzy("", "anything.wav")
	if !ok || score != 0 || pos != nil {
		t.Errorf("empty pattern: score %d pos %v ok %v, want 0 nil true", score, pos, ok)
	}
	_, pos, ok = fuzzy("SHH", "Shh.WAV")
	if !ok || len(pos) != 3 || pos[0] != 0 {
		t.Errorf("case-insensitive match failed: pos %v ok %v", pos, ok)
	}
	_, pos, _ = fuzzy("tw", "takes/two.wav")
	if len(pos) != 2 || pos[0] != 6 || pos[1] != 7 {
		t.Errorf("positions after a boundary = %v, want [6 7]", pos)
	}
}
