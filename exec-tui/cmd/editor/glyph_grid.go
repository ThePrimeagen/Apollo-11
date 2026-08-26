package editor

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	GlyphGridRows = 26
	GlyphGridCols = 10
)

var glyphGridColKeys = []rune{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}

var glyphGrid = buildGlyphGrid()

func buildGlyphGrid() [GlyphGridRows][GlyphGridCols]rune {
	chars := make([]rune, 0, 260)
	for r := rune(0x2580); r <= 0x259F; r++ { // block elements
		chars = append(chars, r)
	}
	for r := rune(0x2500); r <= 0x257F; r++ { // box drawing
		chars = append(chars, r)
	}
	for r := rune(0x25A0); r <= 0x25FF; r++ { // geometric shapes
		if r == '\u25FD' || r == '\u25FE' {
			continue // ◽ ◾ are two cells wide and shove the row
		}
		chars = append(chars, r)
	}
	chars = append(chars, '←', '↑', '→', '↓', '·', '•')
	var grid [GlyphGridRows][GlyphGridCols]rune
	i := 0
	for col := 0; col < GlyphGridCols; col++ {
		for row := 0; row < GlyphGridRows; row++ {
			grid[row][col] = chars[i]
			i++
		}
	}
	return grid
}

func gridColIndex(r rune) int {
	switch {
	case r >= '1' && r <= '9':
		return int(r - '1')
	case r == '0':
		return 9
	default:
		return -1
	}
}

func gridRowIndex(r rune) int {
	if r < 'a' || r > 'z' {
		return -1
	}
	return int(r - 'a')
}

// GlyphAt is the character at a number+letter address (1a … 0z).
// Numbers are columns, letters are rows, so a group stacks down a column.
func GlyphAt(colKey, rowKey rune) (rune, bool) {
	col := gridColIndex(colKey)
	row := gridRowIndex(rowKey)
	if col < 0 || row < 0 {
		return 0, false
	}
	return glyphGrid[row][col], true
}

func (m *Model) openGlyphGrid() {
	if m.PickerOpen || m.ShipPickerOpen || m.ColorPaletteOpen || m.Inserting {
		return
	}
	m.GlyphGridOpen = true
	m.GlyphGridDigit = 0
	m.GlyphGridRow = 0
	m.GlyphGridCol = 0
	m.status = "glyphs  hjkl move  enter pick  1a–0z jump  esc close"
}

func (m *Model) closeGlyphGrid() {
	m.GlyphGridOpen = false
	m.GlyphGridDigit = 0
}

func (m *Model) pickGlyph(ch rune) {
	m.PaintCh = ch
	m.syncSymIdx()
	m.RecentGlyphs = rememberGlyph(m.RecentGlyphs, ch, 10)
	m.status = "paint " + string(ch)
	m.closeGlyphGrid()
}

func (m *Model) moveGlyphCursor(r rune) {
	switch r {
	case 'h':
		m.GlyphGridCol = (m.GlyphGridCol - 1 + GlyphGridCols) % GlyphGridCols
	case 'l':
		m.GlyphGridCol = (m.GlyphGridCol + 1) % GlyphGridCols
	case 'k':
		m.GlyphGridRow = (m.GlyphGridRow - 1 + GlyphGridRows) % GlyphGridRows
	case 'j':
		m.GlyphGridRow = (m.GlyphGridRow + 1) % GlyphGridRows
	}
}

func (m Model) handleGlyphGridKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeGlyphGrid()
		m.status = "glyph grid cancelled"
		return m, nil
	case "enter":
		if m.GlyphGridRow >= 0 && m.GlyphGridRow < GlyphGridRows &&
			m.GlyphGridCol >= 0 && m.GlyphGridCol < GlyphGridCols {
			m.pickGlyph(glyphGrid[m.GlyphGridRow][m.GlyphGridCol])
		}
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}
	r := runeFrom(msg)
	if r == 0 {
		return m, nil
	}
	if col := gridColIndex(r); col >= 0 {
		m.GlyphGridDigit = r
		m.GlyphGridCol = col
		return m, nil
	}
	if row := gridRowIndex(r); row >= 0 && m.GlyphGridDigit != 0 {
		ch, ok := GlyphAt(m.GlyphGridDigit, r)
		if !ok {
			return m, nil
		}
		m.pickGlyph(ch)
		return m, nil
	}
	switch r {
	case 'h', 'j', 'k', 'l':
		m.moveGlyphCursor(r)
	}
	return m, nil
}

func renderGlyphGrid(m Model) string {
	const inner = 2 + GlyphGridCols*2
	var lines []string
	var head strings.Builder
	head.WriteString("  ")
	for _, col := range glyphGridColKeys {
		if m.GlyphGridDigit == col {
			head.WriteString("\x1b[7m")
			head.WriteRune(col)
			head.WriteString("\x1b[0m")
		} else {
			head.WriteRune(col)
		}
		head.WriteByte(' ')
	}
	lines = append(lines, padPlain(strings.TrimRight(head.String(), " "), inner))

	for i := 0; i < GlyphGridRows; i++ {
		rowKey := rune('a' + i)
		var b strings.Builder
		if i == m.GlyphGridRow {
			b.WriteByte('>')
		} else {
			b.WriteByte(' ')
		}
		b.WriteRune(rowKey)
		for col := 0; col < GlyphGridCols; col++ {
			ch := glyphGrid[i][col]
			if i == m.GlyphGridRow && col == m.GlyphGridCol {
				b.WriteString("\x1b[7m")
				b.WriteRune(ch)
				b.WriteString("\x1b[0m")
			} else {
				b.WriteRune(ch)
			}
			b.WriteByte(' ')
		}
		lines = append(lines, padPlain(b.String(), inner))
	}
	lines = append(lines, padPlain(renderRecentGlyphs(m), inner))
	return box(" glyphs  1a–0z ", lines, inner)
}
