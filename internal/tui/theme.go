package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

type palette struct {
	accent   color.Color
	accentFg color.Color
	fg       color.Color
	muted    color.Color
	faint    color.Color
	frame    color.Color
	danger   color.Color
	fret     color.Color
	selBg    color.Color
	band     color.Color
	played   color.Color
	strings  []color.Color
	dim      []color.Color
}

type theme struct {
	name  string
	tag   string
	dark  palette
	light palette
}

func hex(s string) color.Color { return lipgloss.Color(s) }

var themes = []theme{
	{
		name: "amber",
		tag:  "tube glow",
		dark: palette{
			accent: hex("#F5A524"), accentFg: hex("#1A1206"),
			fg: hex("#EDE3D1"), muted: hex("#9C8B70"), faint: hex("#4A4033"),
			frame: hex("#6B5A3E"), danger: hex("#FF6B57"), fret: hex("#FFF7E6"), selBg: hex("#3A2E1A"), band: hex("#2A2216"), played: hex("#6E6049"),
			strings: []color.Color{hex("#D9560F"), hex("#EE8A1F"), hex("#F5A524"), hex("#FBCB5F"), hex("#FFE8A3")},
			dim:     []color.Color{hex("#8A3A0C"), hex("#9A5A14"), hex("#A06A18"), hex("#A38540"), hex("#A6976A")},
		},
		light: palette{
			accent: hex("#B45309"), accentFg: hex("#FFF7E6"),
			fg: hex("#2B2115"), muted: hex("#7A6A52"), faint: hex("#C9BCA4"),
			frame: hex("#A89474"), danger: hex("#B91C1C"), fret: hex("#1C1408"), selBg: hex("#F3E6CC"), band: hex("#EFE2C8"), played: hex("#A3927A"),
			strings: []color.Color{hex("#7C2D12"), hex("#B45309"), hex("#D97706"), hex("#A16207"), hex("#854D0E")},
			dim:     []color.Color{hex("#C48A72"), hex("#E0A46A"), hex("#EDBB7A"), hex("#D4B36A"), hex("#C9A26E")},
		},
	},
	{
		name: "neon",
		tag:  "coated strings",
		dark: palette{
			accent: hex("#FF2E88"), accentFg: hex("#12010A"),
			fg: hex("#E6E6F0"), muted: hex("#7C7C99"), faint: hex("#3A3A52"),
			frame: hex("#4B4B6B"), danger: hex("#FF5C5C"), fret: hex("#FFFFFF"), selBg: hex("#2A1A33"), band: hex("#1C1C2A"), played: hex("#5B5B76"),
			strings: []color.Color{hex("#B36BFF"), hex("#4FA3FF"), hex("#3DF29B"), hex("#FFD23F"), hex("#FF6B3D")},
			dim:     []color.Color{hex("#6B4099"), hex("#2F6299"), hex("#24915D"), hex("#997E26"), hex("#994025")},
		},
		light: palette{
			accent: hex("#C2185B"), accentFg: hex("#FFF5F9"),
			fg: hex("#1E1B2E"), muted: hex("#6B6785"), faint: hex("#C8C4DA"),
			frame: hex("#A9A5C2"), danger: hex("#B00020"), fret: hex("#0B0A14"), selBg: hex("#F1E6F5"), band: hex("#ECE8F4"), played: hex("#9A96B0"),
			strings: []color.Color{hex("#6D28D9"), hex("#1D4ED8"), hex("#047857"), hex("#B45309"), hex("#C2410C")},
			dim:     []color.Color{hex("#B79BE8"), hex("#93AEEC"), hex("#84C2AB"), hex("#E0AE7A"), hex("#E5A38A")},
		},
	},
	{
		name: "ink",
		tag:  "monochrome",
		dark: palette{
			accent: hex("#FFFFFF"), accentFg: hex("#000000"),
			fg: hex("#D4D4D4"), muted: hex("#8A8A8A"), faint: hex("#3F3F3F"),
			frame: hex("#5A5A5A"), danger: hex("#FF7B72"), fret: hex("#FFFFFF"), selBg: hex("#2E2E2E"), band: hex("#1F1F1F"), played: hex("#5E5E5E"),
			strings: []color.Color{hex("#D4D4D4")},
			dim:     []color.Color{hex("#6E6E6E")},
		},
		light: palette{
			accent: hex("#111111"), accentFg: hex("#FFFFFF"),
			fg: hex("#1A1A1A"), muted: hex("#6E6E6E"), faint: hex("#CFCFCF"),
			frame: hex("#9A9A9A"), danger: hex("#B00020"), fret: hex("#000000"), selBg: hex("#E8E8E8"), band: hex("#ECECEC"), played: hex("#9E9E9E"),
			strings: []color.Color{hex("#1A1A1A")},
			dim:     []color.Color{hex("#A8A8A8")},
		},
	},
}

func ThemeNames() []string {
	names := make([]string, len(themes))
	for i, t := range themes {
		names[i] = t.name
	}
	return names
}

func themeIndex(name string) (int, error) {
	if name == "" {
		return 0, nil
	}
	for i, t := range themes {
		if strings.EqualFold(t.name, name) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unknown theme %q (valid: %s)", name, strings.Join(ThemeNames(), ", "))
}

type styles struct {
	pal        palette
	themeName  string
	themeTag   string
	title      lipgloss.Style
	badge      lipgloss.Style
	fg         lipgloss.Style
	muted      lipgloss.Style
	faint      lipgloss.Style
	frame      lipgloss.Style
	danger     lipgloss.Style
	accent     lipgloss.Style
	names      []lipgloss.Style
	frets      []lipgloss.Style
	sustains   []lipgloss.Style
	played     lipgloss.Style
	ruler      lipgloss.Style
	rulerCur   lipgloss.Style
	band       lipgloss.Style
	dash       lipgloss.Style
	bar        lipgloss.Style
	cursor     lipgloss.Style
	cursorDash lipgloss.Style
	sel        lipgloss.Style
	selMark    lipgloss.Style
	pickDir    lipgloss.Style
	pickFile   lipgloss.Style
	pickSel    lipgloss.Style
	help       help.Styles
}

func newStyles(t theme, dark bool) styles {
	p := t.dark
	if !dark {
		p = t.light
	}
	base := lipgloss.NewStyle()
	st := styles{
		pal:        p,
		themeName:  t.name,
		themeTag:   t.tag,
		title:      base.Bold(true).Foreground(p.fg),
		badge:      base.Bold(true).Foreground(p.accentFg).Background(p.accent).Padding(0, 1),
		fg:         base.Foreground(p.fg),
		muted:      base.Foreground(p.muted),
		faint:      base.Foreground(p.faint),
		frame:      base.Foreground(p.frame),
		danger:     base.Foreground(p.danger),
		accent:     base.Foreground(p.accent),
		dash:       base.Foreground(p.faint),
		bar:        base.Foreground(p.frame),
		cursor:     base.Bold(true).Foreground(p.accentFg).Background(p.accent),
		cursorDash: base.Foreground(p.muted).Background(p.selBg),
		played:     base.Foreground(p.played),
		ruler:      base.Foreground(p.muted),
		rulerCur:   base.Bold(true).Foreground(p.accent),
		band:       base.Background(p.band).Foreground(p.fg),
		sel:        base.Background(p.selBg),
		selMark:    base.Bold(true).Foreground(p.accent).Background(p.selBg),
		pickDir:    base.Bold(true).Foreground(p.accent),
		pickFile:   base.Foreground(p.fg),
		pickSel:    base.Background(p.selBg),
	}
	for _, c := range p.strings {
		st.names = append(st.names, base.Bold(true).Foreground(c))
		st.frets = append(st.frets, base.Bold(true).Foreground(c))
	}
	for _, c := range p.dim {
		st.sustains = append(st.sustains, base.Foreground(c))
	}
	key := base.Foreground(p.accent)
	desc := base.Foreground(p.muted)
	sep := base.Foreground(p.faint)
	st.help = help.Styles{
		Ellipsis: sep, ShortKey: key, ShortDesc: desc, ShortSeparator: sep,
		FullKey: key, FullDesc: desc, FullSeparator: sep,
	}
	return st
}

func (st styles) name(s int) lipgloss.Style {
	if len(st.names) == 0 || s < 0 {
		return st.fg
	}
	return st.names[s%len(st.names)]
}

func (st styles) fret(s int) lipgloss.Style {
	if len(st.frets) == 0 || s < 0 {
		return st.fg
	}
	return st.frets[s%len(st.frets)]
}

func (st styles) sustain(s int) lipgloss.Style {
	if len(st.sustains) == 0 || s < 0 {
		return st.dash
	}
	return st.sustains[s%len(st.sustains)]
}
