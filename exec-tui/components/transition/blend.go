package transition

import "github.com/theprimeagen/apollo-11/exec-tui/components/sprite"

// Blend is one cell of a background crossfade. t=0 is from, t=1 is
// to, and in between the floor walks through RGB. A cell that has a
// background on either side keeps one, so the result stays a floor
// a later layer can sit on. Two empty cells stay empty.
func Blend(from, to sprite.Cell, t float64) sprite.Cell {
	if t <= 0 {
		return from
	}
	if t >= 1 {
		return to
	}
	out := sprite.Cell{Ch: ' ', FG: -1, BG: -1}
	switch {
	case from.BG >= 0 && to.BG >= 0:
		out.BG = LerpInk(from.BG, to.BG, t)
	case from.BG >= 0:
		out.BG = from.BG
	case to.BG >= 0:
		out.BG = to.BG
	}
	switch {
	case hasGlyph(to):
		out.Ch = to.Ch
		out.FG = lerpChannel(fgOf(from), fgOf(to), t)
	case hasGlyph(from):
		out.Ch = from.Ch
		out.FG = lerpChannel(fgOf(from), fgOf(to), t)
	}
	return out
}

func hasGlyph(c sprite.Cell) bool {
	return c.Ch != 0 && c.Ch != ' '
}

func fgOf(c sprite.Cell) int {
	if c.FG >= 0 {
		return c.FG
	}
	return c.BG
}

func lerpChannel(a, b int, t float64) int {
	if a < 0 && b < 0 {
		return -1
	}
	if a < 0 {
		return b
	}
	if b < 0 {
		return a
	}
	return LerpInk(a, b, t)
}
