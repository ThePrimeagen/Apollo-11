package editor

// Paint kit: ten clutch colors on 1-0, a named symbol list of full / half /
// quarter blocks, P pastes the selected symbol, and i enters one-shot insert
// so you can type any character.

import "fmt"

// GlyphKeys is shift-1 through shift-0 — bang, at, hash, and friends.
var GlyphKeys = []rune{'!', '@', '#', '$', '%', '^', '&', '*', '(', ')'}

// DefaultGlyphs maps GlyphKeys onto the first ten symbols: four densities,
// four halves, two quadrants.
var DefaultGlyphs = []rune{'░', '▒', '▓', '█', '▀', '▄', '▌', '▐', '▖', '▗'}

// ExtraGlyphs are the other block-elements on the symbol list.
var ExtraGlyphs = []rune{'▘', '▝', '▛', '▜', '▙', '▟', '▞', '▚'}

// ColorKeys is 1-9 then 0 for clutch slots 1-10.
var ColorKeys = []rune{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}

// Sym is one entry on the symbol selection list.
type Sym struct {
	Ch   rune
	Name string
	Kind string
}

// SymbolList is the selectable block alphabet: full, halves, shades, quarters,
// three-quarter blocks, diagonals, and lower eighths.
var SymbolList = []Sym{
	{'█', "full", "full"},
	{'▀', "up half", "half"},
	{'▄', "lo half", "half"},
	{'▌', "left half", "half"},
	{'▐', "right half", "half"},
	{'░', "light", "shade"},
	{'▒', "medium", "shade"},
	{'▓', "dark", "shade"},
	{'▘', "UL 1/4", "quarter"},
	{'▝', "UR 1/4", "quarter"},
	{'▖', "LL 1/4", "quarter"},
	{'▗', "LR 1/4", "quarter"},
	{'▛', "UL 3/4", "quarter"},
	{'▜', "UR 3/4", "quarter"},
	{'▙', "LL 3/4", "quarter"},
	{'▟', "LR 3/4", "quarter"},
	{'▞', "diag /", "diag"},
	{'▚', "diag \\", "diag"},
	{'▁', "1/8", "eighth"},
	{'▂', "2/8", "eighth"},
	{'▃', "3/8", "eighth"},
	{'▅', "5/8", "eighth"},
	{'▆', "6/8", "eighth"},
	{'▇', "7/8", "eighth"},
}

// Greys is xterm 232-255, dark to white. 250-255 are the near-whites.
var Greys = func() []int {
	out := make([]int, 24)
	for i := 0; i < 24; i++ {
		out[i] = 232 + i
	}
	return out
}()

// CubeReds is the 6 red-axis slices of the 6×6×6 color cube (indexes 16-231).
const cubeSide = 6

// Swatch is one fg/bg pair. BG -1 means transparent background.
type Swatch struct {
	FG, BG int
}

func glyphIndex(r rune) int {
	for i, k := range GlyphKeys {
		if k == r {
			return i
		}
	}
	return -1
}

func colorIndex(r rune) int {
	for i, k := range ColorKeys {
		if k == r {
			return i
		}
	}
	return -1
}

func rememberSwatch(list []Swatch, v Swatch, capN int) []Swatch {
	out := []Swatch{v}
	for _, x := range list {
		if x == v {
			continue
		}
		out = append(out, x)
		if len(out) == capN {
			break
		}
	}
	return out
}

func rememberGlyph(list []rune, v rune, capN int) []rune {
	if v == 0 || v == ' ' {
		return list
	}
	out := []rune{v}
	for _, x := range list {
		if x == v {
			continue
		}
		out = append(out, x)
		if len(out) == capN {
			break
		}
	}
	return out
}

func seedColors() []Swatch {
	// named LM materials, then extra whites/greys so the clutch is useful
	// before anyone opens the 8-bit picker.
	return []Swatch{
		{FG: 252, BG: -1}, // silver
		{FG: 178, BG: 94}, // gold
		{FG: 24, BG: 232}, // window
		{FG: 245, BG: -1}, // engine
		{FG: 208, BG: 52}, // plume
		{FG: 255, BG: -1}, // white
		{FG: 254, BG: -1},
		{FG: 253, BG: -1},
		{FG: 251, BG: -1},
		{FG: 250, BG: -1},
	}
}

func seedGlyphs() []rune {
	out := append([]rune(nil), DefaultGlyphs...)
	return out
}

func cubeColor(red, idx int) int {
	// idx is 0-35 in a 6x6 green×blue slice; red is 0-5.
	if red < 0 {
		red = 0
	}
	if red > 5 {
		red = 5
	}
	if idx < 0 {
		idx = 0
	}
	if idx > 35 {
		idx = 35
	}
	green := idx / cubeSide
	blue := idx % cubeSide
	return 16 + red*36 + green*cubeSide + blue
}

func (m *Model) cyclePaint() {
	n := len(SymbolList)
	if n == 0 {
		return
	}
	idx := m.SymIdx
	for i, s := range SymbolList {
		if s.Ch == m.PaintCh {
			idx = i
			break
		}
	}
	m.SymIdx = (idx + 1) % n
	m.PaintCh = SymbolList[m.SymIdx].Ch
	m.status = fmt.Sprintf("paint %s %s", string(m.PaintCh), SymbolList[m.SymIdx].Name)
}

func (m *Model) syncSymIdx() {
	for i, s := range SymbolList {
		if s.Ch == m.PaintCh {
			m.SymIdx = i
			return
		}
	}
}

func (m *Model) applyGlyphKey(r rune) bool {
	i := glyphIndex(r)
	if i < 0 {
		return false
	}
	ch := DefaultGlyphs[i]
	m.PaintCh = ch
	m.syncSymIdx()
	m.status = fmt.Sprintf("paint %s", string(ch))
	return true
}

func (m *Model) applyColorKey(r rune) bool {
	i := colorIndex(r)
	if i < 0 {
		return false
	}
	if i >= len(m.RecentColors) {
		return true // consumed, nothing to load
	}
	m.Brush = m.RecentColors[i]
	m.PalIdx = -1
	m.status = fmt.Sprintf("color fg %d bg %d", m.Brush.FG, m.Brush.BG)
	return true
}

func (m *Model) openPicker() {
	m.PickerOpen = true
	m.PickerIdx = 23 // start on white 255
	m.status = "8-bit picker  hjkl  space pick  esc close  [ ] cube slice"
}

func (m *Model) closePicker(apply bool) {
	if apply {
		fg := Greys[0]
		if m.PickerIdx >= 0 && m.PickerIdx < len(Greys) {
			fg = Greys[m.PickerIdx]
		}
		if m.PickerCube {
			fg = cubeColor(m.CubeRed, m.PickerIdx)
		}
		m.Brush = Swatch{FG: fg, BG: m.Brush.BG}
		m.PalIdx = -1
		m.RecentColors = rememberSwatch(m.RecentColors, m.Brush, 10)
		m.status = fmt.Sprintf("color fg %d", fg)
	}
	m.PickerOpen = false
}

func (m *Model) movePicker(r rune) {
	max := len(Greys) - 1
	cols := 12
	if m.PickerCube {
		max = 35
		cols = 6
	}
	switch r {
	case 'h':
		if m.PickerIdx > 0 {
			m.PickerIdx--
		}
	case 'l':
		if m.PickerIdx < max {
			m.PickerIdx++
		}
	case 'k':
		m.PickerIdx -= cols
		if m.PickerIdx < 0 {
			m.PickerIdx = 0
		}
	case 'j':
		m.PickerIdx += cols
		if m.PickerIdx > max {
			m.PickerIdx = max
		}
	case '[':
		if m.CubeRed > 0 {
			m.CubeRed--
		}
		m.PickerCube = true
	case ']':
		if m.CubeRed < 5 {
			m.CubeRed++
		}
		m.PickerCube = true
	case 'g':
		m.PickerCube = false
		if m.PickerIdx > 23 {
			m.PickerIdx = 23
		}
	}
}
