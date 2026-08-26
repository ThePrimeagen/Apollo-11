package gunfire

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// The Doom muzzle palette on the xterm cube. The flash climbs the
// config's concentration ladder from an orange fringe dot to a
// white-hot core block sitting on bright yellow. Pellets are pale
// tracer heads dragging a dim straw dot one unit behind. Sparks cool
// through the fire ramp — yellow, orange, ember red — as their life
// burns down. Smoke is gray and dims with age.
const (
	fringeGlyph, fringeFG = '·', 214
	edgeGlyph, edgeFG     = '*', 220
	midGlyph, midFG       = '▓', 226
	coreGlyph             = '█'
	coreFG, coreBG        = 231, 220

	pelletGlyph, pelletFG = '•', 230
	trailGlyph, trailFG   = '·', 178
	trailGap              = 1.2  // units behind the head
	trailAfter            = 0.03 // seconds old before a trail shows

	sparkYoungGlyph, sparkYoungFG = '*', 226
	sparkMidGlyph, sparkMidFG     = '+', 208
	sparkOldGlyph, sparkOldFG     = '·', 160

	smokeYoungFG, smokeMidFG, smokeOldFG = 250, 245, 240
	smokeThickGlyph, smokeThickFG        = '░', 245
	smokeThickAt                         = 3

	// age fractions where a speck stops being young / starts dying
	midFrac = 0.35
	oldFrac = 0.70
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

// brailleDots is the braille bit for each dot position in a cell:
// four dot rows down, two dot columns across. U+2800 plus the OR of
// the set dots is the glyph, so every curl of smoke has a symbol.
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

// paint lays the whole shot onto one cols×rows stage, back to front:
// smoke, then sparks, then pellet trails and heads, then the flash on
// top — the bang outshines everything it touches.
func paint(c BlastConfig, cols, rows int, flash, pellets, sparks, smoke *particle.Engine) sprite.Sprite {
	sp := sprite.New(cols, rows)
	paintSmoke(sp, smoke)
	paintSparks(sp, sparks)
	paintPellets(sp, pellets)
	paintFlash(sp, c, flash)
	return sp
}

// paintSmoke merges every speck's braille dot per cell and wears the
// gray of the cell's freshest speck; a pile of smokeThickAt or more
// thickens into the shade block.
func paintSmoke(sp sprite.Sprite, smoke *particle.Engine) {
	counts := map[particle.Cell]int{}
	dots := map[particle.Cell]rune{}
	freshest := map[particle.Cell]float64{}
	eachLive(smoke, func(p particle.Particle) {
		cell := particle.CellOf(p.Pos.X, p.Pos.Y)
		counts[cell]++
		dots[cell] |= dotBit(p.Pos.X, p.Pos.Y)
		f := consumed(p)
		if have, ok := freshest[cell]; !ok || f < have {
			freshest[cell] = f
		}
	})
	for cell, n := range counts {
		painted := sprite.Cell{Ch: 0x2800 | dots[cell], FG: smokeGray(freshest[cell]), BG: -1}
		if n >= smokeThickAt {
			painted = sprite.Cell{Ch: smokeThickGlyph, FG: smokeThickFG, BG: -1}
		}
		sp.Set(cell.Row, cell.Col, painted)
	}
}

func smokeGray(f float64) int {
	switch {
	case f < midFrac:
		return smokeYoungFG
	case f < oldFrac:
		return smokeMidFG
	default:
		return smokeOldFG
	}
}

// paintSparks wears each spark by how much of its life has burnt:
// yellow, then orange, then ember red.
func paintSparks(sp sprite.Sprite, sparks *particle.Engine) {
	eachLive(sparks, func(p particle.Particle) {
		cell := particle.CellOf(p.Pos.X, p.Pos.Y)
		painted := sprite.Cell{Ch: sparkYoungGlyph, FG: sparkYoungFG, BG: -1}
		switch f := consumed(p); {
		case f >= oldFrac:
			painted = sprite.Cell{Ch: sparkOldGlyph, FG: sparkOldFG, BG: -1}
		case f >= midFrac:
			painted = sprite.Cell{Ch: sparkMidGlyph, FG: sparkMidFG, BG: -1}
		}
		sp.Set(cell.Row, cell.Col, painted)
	})
}

// paintPellets drags a dim trail one unit behind every pellet old
// enough to have cleared the muzzle, then paints the pale heads over
// everything the trails touched.
func paintPellets(sp sprite.Sprite, pellets *particle.Engine) {
	eachLive(pellets, func(p particle.Particle) {
		if p.Age < trailAfter {
			return
		}
		back := p.Vel.Normalize().Scale(-trailGap)
		cell := particle.CellOf(p.Pos.X+back.X, p.Pos.Y+back.Y)
		sp.Set(cell.Row, cell.Col, sprite.Cell{Ch: trailGlyph, FG: trailFG, BG: -1})
	})
	eachLive(pellets, func(p particle.Particle) {
		cell := particle.CellOf(p.Pos.X, p.Pos.Y)
		sp.Set(cell.Row, cell.Col, sprite.Cell{Ch: pelletGlyph, FG: pelletFG, BG: -1})
	})
}

// paintFlash climbs the config's concentration ladder: the fringe is
// an orange dot, EdgeAt earns the star, MidAt the bright yellow shade
// block, and CoreAt the white-hot core on a bright yellow floor.
func paintFlash(sp sprite.Sprite, c BlastConfig, flash *particle.Engine) {
	if flash == nil {
		return
	}
	for cell, n := range flash.Occupancy() {
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
