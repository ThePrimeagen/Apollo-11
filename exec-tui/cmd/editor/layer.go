package editor

import (
	"fmt"
	"strings"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// EditLayer is one of the three canvas images. Ctrl-H / Ctrl-L cycle them.
type EditLayer int

const (
	LayerOutline EditLayer = iota
	LayerFG
	LayerBG
)

const (
	layerCount     = 3
	outlineWhite   = 231 // xterm white, so the glyphs mask reads as shape
	bgLayerGlyphFG = 16  // dark glyph on a visible fill
	asciiMagenta   = 201 // leftover ASCII with no color on this layer
)

func (l EditLayer) String() string {
	switch l {
	case LayerFG:
		return "fg"
	case LayerBG:
		return "bg"
	default:
		return "outline"
	}
}

func (l EditLayer) DisplayName() string {
	switch l {
	case LayerFG:
		return "foreground"
	case LayerBG:
		return "background"
	default:
		return "ascii"
	}
}

func renderLayerTabs(current EditLayer) string {
	names := []EditLayer{LayerOutline, LayerFG, LayerBG}
	parts := make([]string, 0, len(names))
	for _, layer := range names {
		label := layer.DisplayName()
		if layer == current {
			label = "\x1b[7m" + label + "\x1b[0m"
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, " · ")
}

func (m *Model) stepLayer(delta int) {
	if m.Win != WinCanvas {
		return
	}
	n := int(m.Layer) + delta
	n %= layerCount
	if n < 0 {
		n += layerCount
	}
	m.Layer = EditLayer(n)
	m.status = fmt.Sprintf("layer %s  ^H/^L layers  hjkl move  ^E glyphs  ^K colors", m.Layer)
}

func namedBG(p sprite.PaletteEntry) int {
	if p.BG >= 0 {
		return p.BG
	}
	return p.FG
}

func (m *Model) applyNamedColor(p sprite.PaletteEntry) {
	switch m.Layer {
	case LayerFG:
		m.Brush.FG = p.FG
		m.status = fmt.Sprintf("fg %s %d", p.Name, p.FG)
	case LayerBG:
		m.Brush.BG = namedBG(p)
		m.status = fmt.Sprintf("bg %s %d", p.Name, m.Brush.BG)
	default:
		m.Brush = Swatch{FG: p.FG, BG: p.BG}
		m.status = fmt.Sprintf("color %s fg %d bg %d", p.Name, p.FG, p.BG)
	}
	m.RecentColors = rememberSwatch(m.RecentColors, m.Brush, 10)
}

func (m *Model) applyPickedColor(n int) {
	switch m.Layer {
	case LayerBG:
		m.Brush.BG = n
		m.status = fmt.Sprintf("color bg %d", n)
	default:
		m.Brush.FG = n
		m.status = fmt.Sprintf("color fg %d", n)
	}
	m.PalIdx = -1
	m.RecentColors = rememberSwatch(m.RecentColors, m.Brush, 10)
}

func (m *Model) pasteOnLayer() {
	switch m.Layer {
	case LayerFG:
		m.paint('f')
	case LayerBG:
		m.paint('b')
	default:
		m.pasteSymbol()
	}
}

// layerCell is the on-screen cell for the active layer. The atlas is
// unchanged — this is only how the canvas is shown.
func layerCell(layer EditLayer, c sprite.Cell) sprite.Cell {
	switch layer {
	case LayerFG:
		if c.Ch == 0 || c.Ch == ' ' {
			return sprite.Cell{Ch: ' ', FG: -1, BG: -1}
		}
		fg := c.FG
		if fg < 0 {
			fg = asciiMagenta
		}
		return sprite.Cell{Ch: c.Ch, FG: fg, BG: -1}
	case LayerBG:
		if c.BG >= 0 {
			ch := c.Ch
			if ch == 0 || ch == ' ' {
				ch = '█'
			}
			return sprite.Cell{Ch: ch, FG: bgLayerGlyphFG, BG: c.BG}
		}
		if c.Ch == 0 || c.Ch == ' ' {
			return sprite.Cell{Ch: ' ', FG: -1, BG: -1}
		}
		return sprite.Cell{Ch: c.Ch, FG: asciiMagenta, BG: -1}
	default:
		if c.Ch == 0 || c.Ch == ' ' {
			return sprite.Cell{Ch: ' ', FG: -1, BG: -1}
		}
		return sprite.Cell{Ch: c.Ch, FG: outlineWhite, BG: -1}
	}
}

// renderComposite is the real ship: glyphs + fg + bg together.
func renderComposite(sp sprite.Sprite) string {
	return sprite.Render(sp)
}
