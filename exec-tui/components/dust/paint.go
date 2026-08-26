package dust

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// brailleDots is the braille bit for each dot position in a cell:
// four dot rows down, two dot columns across. U+2800 plus the OR of
// the set dots is the glyph, so every swirl of specks has a symbol.
var brailleDots = [4][2]rune{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

// dotBit is the braille dot under a unit-space point: the cell splits
// into two dot columns across its width and four dot rows down its
// height, and the speck lights the dot it sits in.
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

// paint lays every engine's live specks onto one cols×rows stage.
// Concentration picks each cell's symbol: at HalfAt and past it wears
// the half shade in light gray, at QuarterAt the quarter shade in mid
// gray, and the thin fringe wears braille in deep gray — its dots
// merged from every speck in the cell. Untouched cells stay sky.
func paint(c PuffConfig, cols, rows int, engines ...*particle.Engine) sprite.Sprite {
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
		case n >= c.HalfAt:
			painted = sprite.Cell{Ch: '▒', FG: c.HalfFG, BG: -1}
		case n >= c.QuarterAt:
			painted = sprite.Cell{Ch: '░', FG: c.QuarterFG, BG: -1}
		default:
			painted = sprite.Cell{Ch: 0x2800 | dots[cell], FG: c.BrailleFG, BG: -1}
		}
		sp.Set(cell.Row, cell.Col, painted)
	}
	return sp
}
