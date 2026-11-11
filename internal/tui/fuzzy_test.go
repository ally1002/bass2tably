package tui

import (
	"sort"
	"testing"
)

func TestFuzzyRanks(t *testing.T) {
	names := []string{"d_u_b.wav", "low-dub.wav", "dub.wav", "dubb.wav"}
	type ranked struct {
		name  string
		score int
	}
	var got []ranked
	for _, n := range names {
		score, pos, ok := fuzzy("dub", n)
		if !ok {
			t.Fatalf("%q did not match dub", n)
		}
		if len(pos) != 3 {
			t.Errorf("%q: %d matched positions, want 3", n, len(pos))
		}
		got = append(got, ranked{n, score})
	}
	sort.SliceStable(got, func(i, j int) bool { return got[i].score > got[j].score })
	want := []string{"dub.wav", "low-dub.wav", "dubb.wav", "d_u_b.wav"}
	for i, w := range want {
		if got[i].name != w {
			t.Fatalf("rank %d = %q (%d), want %q; full order %v", i, got[i].name, got[i].score, w, got)
		}
	}
	if got[1].score != got[2].score {
		t.Errorf("low-dub.wav (%d) and dubb.wav (%d) should tie on score", got[1].score, got[2].score)
	}

	if _, _, ok := fuzzy("xyz", "dub.wav"); ok {
		t.Error("xyz matched dub.wav")
	}
	if _, _, ok := fuzzy("wavs", "dub.wav"); ok {
		t.Error("a pattern longer than its subsequence matched")
	}
	score, pos, ok := fuzzy("", "anything.wav")
	if !ok || score != 0 || pos != nil {
		t.Errorf("empty pattern: score %d pos %v ok %v, want 0 nil true", score, pos, ok)
	}
	_, pos, ok = fuzzy("DUB", "Dub.WAV")
	if !ok || len(pos) != 3 || pos[0] != 0 {
		t.Errorf("case-insensitive match failed: pos %v ok %v", pos, ok)
	}
	_, pos, _ = fuzzy("tw", "takes/two.wav")
	if len(pos) != 2 || pos[0] != 6 || pos[1] != 7 {
		t.Errorf("positions after a boundary = %v, want [6 7]", pos)
	}
}
