package cloud

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

var brailleDots = [4][2]rune{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

func dotBit(x, y float64) rune {
	col := int((x - math.Floor(x)) * 2)
	top := math.Floor(y/particle.CellHeightUnits) * particle.CellHeightUnits
	row := int((y - top) * 2)
	if col < 0 {
		col = 0
	}
	if col > 1 {
		col = 1
	}
	if row < 0 {
		row = 0
	}
	if row > 3 {
		row = 3
	}
	return brailleDots[row][col]
}

// paint lays every engine's parked specks onto one cols×rows stage.
// Concentration picks each cell's symbol: at ThickAt and past it wears
// the half shade in bright white, at ThinAt the quarter shade in mid
// white, and the thin fringe wears braille. Untouched cells stay sky.
func paint(c Config, cols, rows int, engines ...*particle.Engine) sprite.Sprite {
	sp := sprite.New(cols, rows)
	counts := map[particle.Cell]int{}
	dots := map[particle.Cell]rune{}
	for _, e := range engines {
		if e == nil {
			continue
		}
		for _, p := range e.Particles {
			if p.Life <= 0 {
				continue
			}
			cell := particle.CellOf(p.Pos.X, p.Pos.Y)
			counts[cell]++
			dots[cell] |= dotBit(p.Pos.X, p.Pos.Y)
		}
	}
	for cell, n := range counts {
		var painted sprite.Cell
		switch {
		case n >= c.ThickAt:
			painted = sprite.Cell{Ch: '▒', FG: c.ThickFG, BG: -1}
		case n >= c.ThinAt:
			painted = sprite.Cell{Ch: '░', FG: c.MidFG, BG: -1}
		default:
			painted = sprite.Cell{Ch: 0x2800 | dots[cell], FG: c.ThinFG, BG: -1}
		}
		sp.Set(cell.Row, cell.Col, painted)
	}
	return sp
}
