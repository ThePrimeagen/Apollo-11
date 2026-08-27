package gunfire

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Every compass direction burns its own colors. A flame cell's glyph
// thickens with how many specks share it, and its color walks that
// DIRECTION's five-stop ramp by the age of the cell's freshest
// speck — stop one at birth, stop five as the tongue dies. The shared
// core climbs the config's concentration ladder from an orange fringe
// dot to a white-hot block on a bright yellow floor, and it outshines
// any flame it touches.
const (
	fringeGlyph, fringeFG = '·', 214
	edgeGlyph, edgeFG     = '*', 220
	midGlyph, midFG       = '▓', 226
	coreGlyph             = '█'
	coreFG, coreBG        = 231, 220

	// age fractions where a tongue cools into its next color stop
	stop2Frac = 0.2
	stop3Frac = 0.4
	stop4Frac = 0.6
	stop5Frac = 0.8

	// how many specks a flame cell needs to thicken its glyph
	shadeMediumAt = 2 // ▒
	shadeDarkAt   = 4 // ▓
	shadeFullAt   = 6 // █
)

// consumed is how much of a speck's life is behind it, 0 fresh to 1
// spent. Age counts up while Life counts down, so together they are
// the whole story.
func consumed(p particle.Particle) float64 {
	total := p.Age + p.Life
	if total <= 0 {
		return 1
	}
	return p.Age / total
}

// eachLive visits every live speck of an engine. Nil engines are quiet.
func eachLive(e *particle.Engine, visit func(particle.Particle)) {
	if e == nil {
		return
	}
	for _, p := range e.Particles {
		if p.Life <= 0 {
			continue
		}
		visit(p)
	}
}

// paint lays the burn onto one cols×rows stage, back to front: every
// compass direction's flame in its own colors, then the white-hot
// core over them all — the heart outshines the tongues. flames align
// with sprite.Headings; nil slots are quiet.
func paint(c BlastConfig, cols, rows int, core *particle.Engine, flames []*particle.Engine) sprite.Sprite {
	sp := sprite.New(cols, rows)
	for i, flame := range flames {
		if i >= len(sprite.Headings) {
			break
		}
		paintFlame(sp, flame, c.ShotAt(sprite.Headings[i]).Colors)
	}
	paintCore(sp, c, core)
	return sp
}

// paintFlame thickens each cell's glyph with density and colors it
// from the shot's ramp by the age of its freshest speck — the hottest
// wins.
func paintFlame(sp sprite.Sprite, flame *particle.Engine, ramp [5]int) {
	counts := map[particle.Cell]int{}
	freshest := map[particle.Cell]float64{}
	eachLive(flame, func(p particle.Particle) {
		cell := particle.CellOf(p.Pos.X, p.Pos.Y)
		counts[cell]++
		f := consumed(p)
		if have, ok := freshest[cell]; !ok || f < have {
			freshest[cell] = f
		}
	})
	for cell, n := range counts {
		sp.Set(cell.Row, cell.Col, sprite.Cell{
			Ch: flameGlyph(n),
			FG: ramp[stopAt(freshest[cell])],
			BG: -1,
		})
	}
}

// flameGlyph is the density ramp: a lone speck is a light shade, a
// crowd is a full block.
func flameGlyph(n int) rune {
	switch {
	case n >= shadeFullAt:
		return '█'
	case n >= shadeDarkAt:
		return '▓'
	case n >= shadeMediumAt:
		return '▒'
	default:
		return '░'
	}
}

// stopAt is which of the five ramp stops a tongue this far burnt
// wears: stop one at birth, stop five nearly out.
func stopAt(f float64) int {
	switch {
	case f < stop2Frac:
		return 0
	case f < stop3Frac:
		return 1
	case f < stop4Frac:
		return 2
	case f < stop5Frac:
		return 3
	default:
		return 4
	}
}

// paintCore climbs the config's concentration ladder: the fringe is
// an orange dot, EdgeAt earns the star, MidAt the bright yellow shade
// block, and CoreAt the white-hot core on a bright yellow floor.
func paintCore(sp sprite.Sprite, c BlastConfig, core *particle.Engine) {
	if core == nil {
		return
	}
	for cell, n := range core.Occupancy() {
		var painted sprite.Cell
		switch {
		case n >= c.CoreAt:
			painted = sprite.Cell{Ch: coreGlyph, FG: coreFG, BG: coreBG}
		case n >= c.MidAt:
			painted = sprite.Cell{Ch: midGlyph, FG: midFG, BG: -1}
		case n >= c.EdgeAt:
			painted = sprite.Cell{Ch: edgeGlyph, FG: edgeFG, BG: -1}
		default:
			painted = sprite.Cell{Ch: fringeGlyph, FG: fringeFG, BG: -1}
		}
		sp.Set(cell.Row, cell.Col, painted)
	}
}
