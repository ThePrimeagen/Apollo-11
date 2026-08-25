package cast

import (
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

// PutCell writes one cell in the labs' native color language: xterm-256
// indexes, -1 for "no color". It converts to the lip gloss style the
// screen's cells carry.
func PutCell(scr *screenplay.Screen, x, y int, ch rune, fg, bg int) {
}

// BlitSprite lays a lander-lab sprite onto the screen with its top-left
// cell at (x, y). Transparent sprite cells do not overwrite what is
// already there; anything past an edge is clipped.
func BlitSprite(scr *screenplay.Screen, x, y int, sp sprite.Sprite) {
}
