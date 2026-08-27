package gunfire

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// The Doom flame palette on the xterm cube. The flame is painted by
// density and age: a cell's glyph thickens with how many specks share
// it, and its color is the age of its freshest speck — bright yellow
// at birth, cooling through orange and red down to a maroon ember as
// the tongue dies. The core climbs the config's concentration ladder
// from an orange fringe dot to a white-hot block on a bright yellow
// floor, and it outshines the flame wherever they meet.
const (
	fringeGlyph, fringeFG = '·', 214
	edgeGlyph, edgeFG     = '*', 220
	midGlyph, midFG       = '▓', 226
	coreGlyph             = '█'
	coreFG, coreBG        = 231, 220

	flameYellowFG = 226 // birth
	flameOrangeFG = 208 // two tenths burnt
	flameRedFG    = 196 // four tenths
	flameEmberFG  = 160 // six tenths
	flameMaroonFG = 124 // eight tenths to the end

	// age fractions where a tongue cools into its next color
	orangeFrac = 0.2
	redFrac    = 0.4
	emberFrac  = 0.6
	maroonFrac = 0.8

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

// paint lays the burn onto one cols×rows stage, back to front: the
// flame first, then the white-hot core over it — the heart outshines
// the tongues.
func paint(c BlastConfig, cols, rows int, core, flame *particle.Engine) sprite.Sprite {
	sp := sprite.New(cols, rows)
	paintFlame(sp, flame)
	paintCore(sp, c, core)
	return sp
}

// paintFlame thickens each cell's glyph with density and colors it by
// the age of its freshest speck — the hottest wins.
func paintFlame(sp sprite.Sprite, flame *particle.Engine) {
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
			FG: flameColor(freshest[cell]),
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

// flameColor is the cooling ramp: yellow at birth, orange, red,
// ember, then maroon as the tongue burns down.
func flameColor(f float64) int {
	switch {
	case f < orangeFrac:
		return flameYellowFG
	case f < redFrac:
		return flameOrangeFG
	case f < emberFrac:
		return flameRedFG
	case f < maroonFrac:
		return flameEmberFG
	default:
		return flameMaroonFG
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
