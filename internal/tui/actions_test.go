package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestActionIDsUnique(t *testing.T) {
	m, _ := practiceModel(t)
	defer m.closePlayer()
	seen := map[string]bool{}
	for _, a := range m.actions() {
		if a.id == "" || seen[a.id] {
			t.Errorf("action id %q is empty or repeated", a.id)
		}
		seen[a.id] = true
	}
}

func TestEveryActionHasHelp(t *testing.T) {
	m, _ := practiceModel(t)
	defer m.closePlayer()
	for _, a := range m.actions() {
		if a.title == "" || a.enabled == nil || a.run == nil {
			t.Errorf("action %q is missing a title, an enabled predicate or a run function", a.id)
		}
		if len(a.binding.Keys()) == 0 || a.binding.Help().Key == "" || a.binding.Help().Desc == "" {
			t.Errorf("action %q has no key or no help text", a.id)
		}
	}
}

func TestHelpDerivedFromActions(t *testing.T) {
	m, _ := practiceModel(t)
	defer m.closePlayer()
	inTable := map[string]bool{}
	for _, a := range m.actions() {
		inTable[strings.Join(a.binding.Keys(), ",")] = true
	}
	h := m.helpMap()
	inHelp := map[string]bool{}
	for _, col := range h.full {
		if len(col) > helpColumnRows {
			t.Errorf("a help column has %d rows, want at most %d", len(col), helpColumnRows)
		}
		for _, b := range col {
			name := strings.Join(b.Keys(), ",")
			if !inTable[name] {
				t.Errorf("help shows %q, which is not in the action table", name)
			}
			inHelp[name] = true
		}
	}
	for name := range inTable {
		if !inHelp[name] {
			t.Errorf("action bound to %q is missing from the full help", name)
		}
	}
	if len(h.short) != 5 {
		t.Fatalf("short help has %d entries, want 5", len(h.short))
	}
	for i, want := range []string{"space", "n", "ctrl+p", "?", "q"} {
		if got := h.short[i].Help().Key; got != want {
			t.Errorf("short help %d = %q, want %q", i, got, want)
		}
	}
}

func TestKeysStillRunThroughTheTable(t *testing.T) {
	m, sp := practiceModel(t)
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress('m'))
	if !sp.clickOn {
		t.Fatal("m did not reach the metronome action")
	}
	m, _ = advance(t, m, keyPress('n'))
	if !m.notes.open {
		t.Fatal("n did not open the note list")
	}
	m.notes.setSel(2)
	m, _ = advance(t, m, keyPress('k'))
	if m.notes.sel != 1 {
		t.Fatalf("k with the panel open selected note %d, want 1", m.notes.sel)
	}
	m, _ = advance(t, m, keyPress('n'))
	m.vp.SetContent(strings.Repeat("x\n", 100))
	m, _ = advance(t, m, keyPress('j'))
	if m.vp.YOffset() != 1 {
		t.Fatalf("j with the panel closed scrolled to %d, want 1", m.vp.YOffset())
	}
	m, _ = advance(t, m, keyPress('G'))
	if m.vp.YOffset() == 1 {
		t.Fatal("G did not scroll to the bottom")
	}
	m, _ = advance(t, m, keyPress('l'))
	if m.vp.XOffset() != 0 {
		t.Fatal("l scrolled right past the content width")
	}
}

func TestPaletteFilters(t *testing.T) {
	m, _ := practiceModel(t)
	defer m.closePlayer()
	m, _ = advance(t, m, tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	if !m.palette.open {
		t.Fatal("ctrl+p did not open the palette")
	}
	if len(m.palette.hits) < 20 {
		t.Fatalf("an empty palette lists %d commands", len(m.palette.hits))
	}
	for _, a := range m.palette.hits {
		if a.id == "palette" {
			t.Fatal("the palette lists itself")
		}
	}
	for _, r := range "loop" {
		m, _ = advance(t, m, keyPress(r))
	}
	if len(m.palette.hits) < 3 {
		t.Fatalf("loop matched %d commands", len(m.palette.hits))
	}
	for i, want := range []string{"loop-a", "loop-b", "loop-clear"} {
		if m.palette.hits[i].id != want {
			t.Errorf("hit %d = %q, want %q", i, m.palette.hits[i].id, want)
		}
	}
	view := ansi.Strip(m.viewerView())
	if !strings.Contains(view, "Set loop start at this measure") || !strings.Contains(view, "› loop") {
		t.Fatalf("palette overlay missing from the view:\n%s", view)
	}
	if h := strings.Count(view, "\n") + 1; h != m.h {
		t.Fatalf("overlay changed the view height to %d", h)
	}
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.palette.open {
		t.Fatal("esc did not close the palette")
	}
	if m.screen != viewerScreen {
		t.Fatal("esc in the palette left the viewer")
	}
}

func TestPaletteEnterRuns(t *testing.T) {
	m, sp := practiceModel(t)
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress(':'))
	if !m.palette.open {
		t.Fatal(": did not open the palette")
	}
	for _, r := range "metro" {
		m, _ = advance(t, m, keyPress(r))
	}
	if m.palette.hits[0].id != "click" {
		t.Fatalf("first hit for metro is %q", m.palette.hits[0].id)
	}
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !sp.clickOn || m.palette.open {
		t.Fatalf("enter did not run the metronome and close: click %v open %v", sp.clickOn, m.palette.open)
	}
	m, _ = advance(t, m, keyPress(':'))
	for _, r := range "qt" {
		m, _ = advance(t, m, keyPress(r))
	}
	if m.themeIdx != 0 || m.palette.input.Value() != "qt" {
		t.Fatal("q or t acted as global keys inside the palette")
	}
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = advance(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.palette.sel != min(1, len(m.palette.hits)-1) {
		t.Fatalf("sel = %d after down down up", m.palette.sel)
	}
}

func TestPaletteDisabledActionsHidden(t *testing.T) {
	m, _ := practiceModel(t)
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress('v'))
	m, _ = advance(t, m, keyPress(':'))
	for _, a := range m.palette.hits {
		if a.id == "fewer" || a.id == "more" || a.id == "fit" || a.id == "panel-grow" {
			t.Errorf("%q is listed although disabled", a.id)
		}
	}
}

func keyTable(m *model) string {
	var b strings.Builder
	b.WriteString("| Key | Action |\n|---|---|\n")
	for _, a := range m.actions() {
		fmt.Fprintf(&b, "| `%s` | %s |\n", a.binding.Help().Key, a.title)
	}
	return b.String()
}

func TestKeyTableRenders(t *testing.T) {
	m, _ := practiceModel(t)
	defer m.closePlayer()
	table := keyTable(&m)
	if !strings.Contains(table, "| `space` | Play / pause |") || !strings.Contains(table, "| `ctrl+p` | Command palette |") {
		t.Fatalf("key table:\n%s", table)
	}
}

func TestPaletteCardIsPaddedTopAndBottom(t *testing.T) {
	m, _ := practiceModel(t)
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress(':'))
	rows := strings.Split(ansi.Strip(m.paletteView()), "\n")
	if strings.TrimSpace(rows[0]) != "" || strings.TrimSpace(rows[len(rows)-1]) != "" {
		t.Fatalf("card lacks a blank margin row at the top or the bottom:\n%q\n%q", rows[0], rows[len(rows)-1])
	}
	if !strings.Contains(rows[1], "type a command") || strings.TrimSpace(rows[2]) != "" {
		t.Fatalf("input row and the gap below it are misplaced:\n%q\n%q", rows[1], rows[2])
	}
	w := ansi.StringWidth(rows[0])
	for i, r := range rows {
		if ansi.StringWidth(r) != w {
			t.Errorf("card row %d is %d wide, want %d", i, ansi.StringWidth(r), w)
		}
	}
}

func TestPaletteCardCarriesBandEverywhere(t *testing.T) {
	m, _ := practiceModel(t)
	defer m.closePlayer()
	m, _ = advance(t, m, keyPress(':'))
	for _, r := range "sp" {
		m, _ = advance(t, m, keyPress(r))
	}
	card := m.paletteView()
	bg := bandSeq(m.st)
	for i, row := range strings.Split(card, "\n") {
		row = strings.ReplaceAll(row, m.st.cursor.Render(" "), "")
		if !bandCovered(row, bg) {
			t.Errorf("card row %d has a cell without the band background:\n%q", i, row)
		}
	}
	if !strings.Contains(card, m.st.onBand(m.st.fg).Bold(true).Render("sp")) {
		t.Error("typed text is not drawn bold on the band")
	}
}
