package editor

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func (m *Model) openColorPalette() {
	if m.PickerOpen || m.FilePickerOpen || m.GlyphGridOpen || m.Inserting {
		return
	}
	m.ColorPaletteOpen = true
	if m.PalIdx >= 0 && m.PalIdx < len(m.Atlas.Palette) {
		m.ColorPaletteIdx = m.PalIdx
	} else {
		m.ColorPaletteIdx = 0
	}
	m.status = "colors  jk move  enter pick  esc close"
}

func (m *Model) closeColorPalette(apply bool) {
	if apply {
		if m.ColorPaletteIdx >= 0 && m.ColorPaletteIdx < len(m.Atlas.Palette) {
			p := m.Atlas.Palette[m.ColorPaletteIdx]
			m.PalIdx = m.ColorPaletteIdx
			m.applyNamedColor(p)
		}
	} else {
		m.status = "color palette cancelled"
	}
	m.ColorPaletteOpen = false
}

func (m Model) handleColorPaletteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeColorPalette(false)
		return m, nil
	case "enter", "space":
		m.closeColorPalette(true)
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}
	r := runeFrom(msg)
	switch r {
	case 'k', 'h':
		if m.ColorPaletteIdx > 0 {
			m.ColorPaletteIdx--
		}
	case 'j', 'l':
		if n := len(m.Atlas.Palette); n > 0 && m.ColorPaletteIdx+1 < n {
			m.ColorPaletteIdx++
		}
	}
	return m, nil
}

func renderColorPalette(m Model) string {
	const w = 36
	lines := []string{
		padPlain("named  jk  enter pick", w),
		padPlain("recent colors", w),
		padPlain(renderClutch(m), w),
	}
	if len(m.Atlas.Palette) == 0 {
		lines = append(lines, padPlain("  (empty)", w))
	}
	for i, p := range m.Atlas.Palette {
		mark := "  "
		if i == m.ColorPaletteIdx {
			mark = "> "
		}
		swatch := " "
		if p.FG >= 0 || p.BG >= 0 {
			swatch = sprite.Render(oneCell(sprite.Cell{Ch: '█', FG: p.FG, BG: p.BG}))
		}
		bg := "-"
		if p.BG >= 0 {
			bg = fmt.Sprintf("%d", p.BG)
		}
		lines = append(lines, padPlain(fmt.Sprintf("%s%s %-7s fg %-3d bg %s", mark, swatch, p.Name, p.FG, bg), w))
	}
	return box(" colors ", lines, w)
}
