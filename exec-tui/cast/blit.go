package cast

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"

	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// styleFor converts the labs' xterm-256 indexes (-1 for "no color")
// into the lip gloss style a screen cell carries.
func styleFor(fg, bg int) uv.Style {
	var st uv.Style
	if fg >= 0 && fg < 256 {
		st.Fg = ansi.IndexedColor(uint8(fg))
	}
	if bg >= 0 && bg < 256 {
		st.Bg = ansi.IndexedColor(uint8(bg))
	}
	return st
}

// PutCell writes one cell in the labs' native color language: xterm-256
// indexes, -1 for "no color". Out of bounds is ignored.
func PutCell(scr *screenplay.Screen, x, y int, ch rune, fg, bg int) {
	scr.Put(x, y, ch, styleFor(fg, bg))
}

// BlitSprite lays a lander-lab sprite onto the screen with its top-left
// cell at (x, y). Transparent sprite cells do not overwrite what is
// already there; anything past an edge is clipped.
func BlitSprite(scr *screenplay.Screen, x, y int, sp sprite.Sprite) {
	if scr == nil {
		return
	}
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			cell := sp.At(r, c)
			if cell.Transparent() {
				continue
			}
			PutCell(scr, x+c, y+r, cell.Ch, cell.FG, cell.BG)
		}
	}
}
