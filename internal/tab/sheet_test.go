package tab_test

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"bass2tably/internal/music"
	"bass2tably/internal/tab"
)

// legacyRenderLayoutWidth is the body of tab.RenderLayoutWidth as it was at
// master 7ac76da, copied verbatim before it was replaced by Build + Text; the
// only edits are the `tab.` package qualifiers this external test needs (R14).
func legacyRenderLayoutWidth(notes []music.Note, frets []tab.Fretting, tuning []int, gridDiv, measuresPerLine int) (string, tab.Layout) {
	if measuresPerLine < 1 {
		measuresPerLine = 1
	}
	slotsPerMeasure := gridDiv * 4

	totalSlots := slotsPerMeasure
	for i, n := range notes {
		if i >= len(frets) || frets[i].String < 0 {
			continue
		}
		if n.Slot+1 > totalSlots {
			totalSlots = n.Slot + 1
		}
	}
	measures := (totalSlots + slotsPerMeasure - 1) / slotsPerMeasure

	cells := make([][]int, len(tuning))
	for s := range cells {
		cells[s] = make([]int, measures*slotsPerMeasure)
		for k := range cells[s] {
			cells[s][k] = -1
		}
	}
	cellW := 2
	for i, n := range notes {
		if i >= len(frets) {
			continue
		}
		f := frets[i]
		if f.String < 0 || f.String >= len(tuning) {
			continue
		}
		cells[f.String][n.Slot] = f.Fret
		if f.Fret > 9 {
			cellW = 3
		}
	}

	names := make([]string, len(tuning))
	nameW := 0
	for s, open := range tuning {
		names[s] = strings.TrimRight(music.MidiName(open), "-0123456789")
		if len(names[s]) > nameW {
			nameW = len(names[s])
		}
	}

	var b strings.Builder
	for line := 0; line < measures; line += measuresPerLine {
		lineEnd := line + measuresPerLine
		if lineEnd > measures {
			lineEnd = measures
		}
		for s := len(tuning) - 1; s >= 0; s-- {
			b.WriteString(names[s])
			b.WriteString(strings.Repeat(" ", nameW-len(names[s])))
			b.WriteString("|")
			for m := line; m < lineEnd; m++ {
				for k := m * slotsPerMeasure; k < (m+1)*slotsPerMeasure; k++ {
					if fret := cells[s][k]; fret >= 0 {
						fs := strconv.Itoa(fret)
						b.WriteString(fs)
						b.WriteString(strings.Repeat("-", cellW-len(fs)))
					} else {
						b.WriteString(strings.Repeat("-", cellW))
					}
				}
				b.WriteString("|")
			}
			b.WriteString("\n")
		}
		if lineEnd < measures {
			b.WriteString("\n")
		}
	}
	return b.String(), tab.Layout{
		NameW:           nameW,
		CellW:           cellW,
		SlotsPerMeasure: slotsPerMeasure,
		MeasuresPerLine: measuresPerLine,
		Measures:        measures,
		Rows:            len(tuning),
	}
}

func layoutNotesFrets() ([]music.Note, []tab.Fretting) {
	notes := []music.Note{
		{Midi: 33, Slot: 0, Len: 1},
		{Midi: 45, Slot: 17, Len: 1},
		{Midi: 40, Slot: 47, Len: 1},
		{Midi: 50, Slot: 70, Len: 1},
	}
	frets := []tab.Fretting{
		{String: 2, Fret: 0},
		{String: 2, Fret: 12},
		{String: 3, Fret: 2},
		{String: 3, Fret: 12},
	}
	return notes, frets
}

func assertMatchesLegacy(t *testing.T, name string, notes []music.Note, frets []tab.Fretting, gridDiv, perLine int) {
	t.Helper()
	wantText, wantLayout := legacyRenderLayoutWidth(notes, frets, beadg, gridDiv, perLine)
	sh := tab.Build(notes, frets, beadg, gridDiv, perLine, 2)
	if got := sh.Text(tab.ASCII); got != wantText {
		t.Errorf("%s: Text(ASCII) differs from legacy\ngot:\n%s\nwant:\n%s", name, got, wantText)
	}
	if got := sh.Layout(); got != wantLayout {
		t.Errorf("%s: Layout = %+v, want %+v", name, got, wantLayout)
	}
	gotText, gotLayout := tab.RenderLayoutWidth(notes, frets, beadg, gridDiv, perLine)
	if gotText != wantText || gotLayout != wantLayout {
		t.Errorf("%s: RenderLayoutWidth wrapper differs from legacy", name)
	}
}

func TestBuildTextMatchesLegacy(t *testing.T) {
	riff := riffNotes()
	riffFrets := tab.Assign(riff, beadg, 0)
	for _, gridDiv := range []int{2, 4} {
		for _, perLine := range []int{1, 2, 4} {
			assertMatchesLegacy(t, fmt.Sprintf("riff grid %d perLine %d", gridDiv, perLine), riff, riffFrets, gridDiv, perLine)
		}
	}

	notes, frets := layoutNotesFrets()
	assertMatchesLegacy(t, "fret 12", notes, frets, 4, 4)
	if sh := tab.Build(notes, frets, beadg, 4, 4, 2); sh.CellW != 3 {
		t.Errorf("fret 12: CellW = %d, want 3", sh.CellW)
	}

	unplayable := []music.Note{{Midi: 33, Slot: 0, Len: 1}, {Midi: 10, Slot: 20, Len: 1}}
	assertMatchesLegacy(t, "unplayable", unplayable, []tab.Fretting{{String: 2, Fret: 0}, {String: -1, Fret: -1}}, 4, 4)
	assertMatchesLegacy(t, "short frets", unplayable, []tab.Fretting{{String: 2, Fret: 0}}, 4, 4)
	assertMatchesLegacy(t, "out-of-range string", unplayable, []tab.Fretting{{String: 2, Fret: 0}, {String: 7, Fret: 3}}, 4, 4)

	assertMatchesLegacy(t, "no notes", nil, nil, 4, 4)
	if sh := tab.Build(nil, nil, beadg, 4, 4, 2); sh.Measures() != 1 {
		t.Errorf("no notes: Measures = %d, want 1", sh.Measures())
	}
}

func TestBuildTextProperty(t *testing.T) {
	for seed := int64(0); seed < 300; seed++ {
		r := rand.New(rand.NewSource(seed))
		n := 1 + r.Intn(40)
		slots := r.Perm(200)[:n]
		sort.Ints(slots)
		notes := make([]music.Note, n)
		frets := make([]tab.Fretting, n)
		for i := range notes {
			midi := beadg[0] + r.Intn(beadg[len(beadg)-1]+24-beadg[0]+1)
			ps := tab.Positions(midi, beadg)
			notes[i] = music.Note{Midi: midi, Slot: slots[i], Len: 1 + r.Intn(8)}
			frets[i] = ps[r.Intn(len(ps))]
		}
		gridDiv := []int{2, 4}[r.Intn(2)]
		perLine := 1 + r.Intn(6)
		wantText, wantLayout := legacyRenderLayoutWidth(notes, frets, beadg, gridDiv, perLine)
		sh := tab.Build(notes, frets, beadg, gridDiv, perLine, 2)
		if got := sh.Text(tab.ASCII); got != wantText {
			t.Fatalf("seed %d: Text(ASCII) differs from legacy\ngot:\n%s\nwant:\n%s", seed, got, wantText)
		}
		if got := sh.Layout(); got != wantLayout {
			t.Fatalf("seed %d: Layout = %+v, want %+v", seed, got, wantLayout)
		}
	}
}

func TestSustainAndTie(t *testing.T) {
	notes := []music.Note{
		{Midi: 33, Slot: 14, Len: 6},
		{Midi: 38, Slot: 24, Len: 1},
	}
	frets := []tab.Fretting{{String: 2, Fret: 0}, {String: 3, Fret: 0}}

	sh := tab.Build(notes, frets, beadg, 4, 4, 3)
	if sh.Measures() != 2 {
		t.Fatalf("Measures = %d, want 2", sh.Measures())
	}
	cols := append(sh.Systems[0].Measures[0].Columns, sh.Systems[0].Measures[1].Columns...)
	want := map[int]tab.CellKind{14: tab.Onset, 15: tab.Sustain, 16: tab.Tied, 17: tab.Sustain, 18: tab.Sustain, 19: tab.Sustain, 20: tab.Empty}
	for slot, kind := range want {
		if got := cols[slot].Cells[2].Kind; got != kind {
			t.Errorf("slot %d: kind %d, want %d", slot, got, kind)
		}
		if kind != tab.Empty && cols[slot].Cells[2].Note != 0 {
			t.Errorf("slot %d: Note = %d, want 0", slot, cols[slot].Cells[2].Note)
		}
	}
	if cols[14].Onset != 0 || cols[14].Len != 6 {
		t.Errorf("column 14: Onset %d Len %d, want 0 and 6", cols[14].Onset, cols[14].Len)
	}
	if cols[15].Onset != -1 {
		t.Errorf("column 15: Onset = %d, want -1", cols[15].Onset)
	}

	wide := sh.Text(tab.Unicode)
	if !strings.Contains(wide, "│(0)═══") {
		t.Errorf("CellW 3: tie not drawn as (0) at the bar line:\n%s", wide)
	}
	if !strings.Contains(wide, "0═════") {
		t.Errorf("CellW 3: onset fill is not the sustain glyph:\n%s", wide)
	}

	narrow := tab.Build(notes, frets, beadg, 4, 4, 2).Text(tab.Unicode)
	if !strings.Contains(narrow, "│════") {
		t.Errorf("CellW 2: tie not drawn as sustain glyphs:\n%s", narrow)
	}
	if strings.ContainsAny(narrow, "()") {
		t.Errorf("CellW 2: parentheses must be dropped, not clipped:\n%s", narrow)
	}
	if strings.Contains(sh.Text(tab.ASCII), "(") {
		t.Error("ASCII glyphs must never draw a tie")
	}
}

func TestSustainStopsAtNextOnset(t *testing.T) {
	notes := []music.Note{
		{Midi: 33, Slot: 0, Len: 8},
		{Midi: 35, Slot: 2, Len: 1},
	}
	frets := []tab.Fretting{{String: 2, Fret: 0}, {String: 2, Fret: 2}}
	sh := tab.Build(notes, frets, beadg, 4, 4, 2)
	cols := sh.Systems[0].Measures[0].Columns
	if cols[1].Cells[2].Kind != tab.Sustain || cols[1].Cells[2].Note != 0 {
		t.Errorf("slot 1: %+v, want a sustain of note 0", cols[1].Cells[2])
	}
	if cols[2].Cells[2].Kind != tab.Onset || cols[2].Cells[2].Note != 1 {
		t.Errorf("slot 2: %+v, want the onset of note 1", cols[2].Cells[2])
	}
	for _, col := range cols[3:] {
		if c := col.Cells[2]; c.Kind != tab.Empty {
			t.Errorf("slot %d: %+v, want empty after the second onset", col.Slot, c)
		}
	}
}

func TestGlyphsAreWidthOne(t *testing.T) {
	for name, g := range map[string]tab.Glyphs{"ASCII": tab.ASCII, "Unicode": tab.Unicode} {
		for field, s := range map[string]string{"Dash": g.Dash, "Sustain": g.Sustain, "Bar": g.Bar, "End": g.End} {
			if w := ansi.StringWidth(s); w != 1 {
				t.Errorf("%s.%s = %q has width %d, want 1", name, field, s, w)
			}
		}
	}
}

func TestStrideZeroMatchesLegacy(t *testing.T) {
	l := tab.Layout{Rows: 5}
	if got := l.Stride(); got != l.Rows+1 {
		t.Errorf("Stride() = %d, want %d", got, l.Rows+1)
	}
	for block := 0; block < 3; block++ {
		for s := 0; s < l.Rows; s++ {
			want := block*(l.Rows+1) + l.Rows - 1 - s
			if got := l.StringRow(block, s); got != want {
				t.Errorf("StringRow(%d, %d) = %d, want %d", block, s, got, want)
			}
		}
	}
	padded := tab.Layout{Rows: 5, Top: 1, Bottom: 2}
	if got := padded.Stride(); got != 9 {
		t.Errorf("padded Stride() = %d, want 9", got)
	}
	if got := padded.StringRow(1, 4); got != 10 {
		t.Errorf("padded StringRow(1, 4) = %d, want 10", got)
	}
}
