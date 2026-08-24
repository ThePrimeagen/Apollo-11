// Package fire is a thin color layer on particle. Occupancy paints the
// flame: one particle is invisible, two show a little, and more particles
// make more fire — orange, then solid yellow. Particles stack; they do
// not fan. The package does not draw a lander.
package fire

import (
	"math"

	"github.com/theprimeagen/apollo-11/lander-lab/particle"
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

const (
	Rows      = 4
	Cols      = 14
	Particles = 100
	ViewCols  = 16
	ViewRows  = 8
	fps       = 20
)

// DefaultConfig is the 45° four-row trail.
func DefaultConfig() particle.Config {
	return particle.Config{
		Width:     float64(Cols) - 0.01,
		Height:    float64(Rows)*particle.CellHeightUnits - 0.01,
		Origin:    particle.Vec2{X: 1.5, Y: 1.0},
		Direction: particle.Vec2{X: 1, Y: 1},
		Count:     Particles,
		Period:    0.08,
		MinLife:   0.42,
		MaxLife:   0.58,
		MinSpeed:  17,
		MaxSpeed:  23,
		Spread:    0.14,
	}
}

// BoosterConfig is the size-4 sideways plume: left-to-right, four units
// wide, two units tall (one terminal row). Spread is zero so particles
// stack on the axis instead of fanning out.
func BoosterConfig() particle.Config {
	return particle.Config{
		Width:     4 - 0.01,
		Height:    2 - 0.01,
		Origin:    particle.Vec2{X: 0.4, Y: 1.0},
		Direction: particle.Vec2{X: 1, Y: 0},
		Count:     Particles,
		Period:    0.07,
		MinLife:   0.16,
		MaxLife:   0.28,
		MinSpeed:  12,
		MaxSpeed:  20,
		Spread:    0,
	}
}

// Flame is one running trail.
type Flame struct {
	Eng      *particle.Engine
	YellowAt int
	OrangeAt int
}

// New starts the 45° trail. It has not yet emitted.
func New(seed int64) *Flame {
	return newFlame(seed, DefaultConfig())
}

// Booster starts the left-to-right four-by-two plume. Thresholds sit
// higher than the 45° demo so a hundred stacked particles still show
// a ramp instead of a solid yellow bar.
func Booster(seed int64) *Flame {
	f := newFlame(seed, BoosterConfig())
	f.YellowAt = 80
	f.OrangeAt = 24
	return f
}

func newFlame(seed int64, cfg particle.Config) *Flame {
	return &Flame{
		Eng:      particle.New(seed, cfg),
		YellowAt: 8,
		OrangeAt: 5,
	}
}

// Update advances the particle engine.
func (f *Flame) Update(dt float64) {
	if f == nil || f.Eng == nil {
		return
	}
	f.Eng.Update(dt)
}

// Color maps an occupancy count onto a flame cell. One particle is
// invisible; two show a little; more occupancy paints more flame.
func Color(n int) sprite.Cell {
	return shade(n, 8, 5)
}

func shade(n, yellowAt, orangeAt int) sprite.Cell {
	if yellowAt < 1 {
		yellowAt = 8
	}
	if orangeAt < 1 || orangeAt > yellowAt {
		orangeAt = 5
	}
	switch {
	case n >= yellowAt:
		return sprite.Cell{Ch: '█', FG: 226, BG: 220}
	case n >= orangeAt:
		return sprite.Cell{Ch: '▓', FG: 208, BG: 166}
	case n >= 3:
		return sprite.Cell{Ch: '░', FG: 196, BG: -1}
	case n >= 2:
		return sprite.Cell{Ch: '·', FG: 160, BG: -1}
	default:
		return sprite.Cell{Ch: ' ', FG: -1, BG: -1}
	}
}

func (f *Flame) box() (cols, rows int) {
	if f == nil || f.Eng == nil {
		return Cols, Rows
	}
	cols = int(math.Ceil(f.Eng.Cfg.Width - 1e-9))
	rows = int(math.Ceil((f.Eng.Cfg.Height - 1e-9) / particle.CellHeightUnits))
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cols, rows
}

// Sprite is the fire box sized from the engine. Empty cells stay transparent.
func (f *Flame) Sprite() sprite.Sprite {
	cols, rows := f.box()
	sp := sprite.New(cols, rows)
	if f == nil || f.Eng == nil {
		return sp
	}
	for cell, n := range f.Eng.Occupancy() {
		if cell.Col < 0 || cell.Row < 0 || cell.Col >= cols || cell.Row >= rows {
			continue
		}
		sp.Set(cell.Row, cell.Col, shade(n, f.YellowAt, f.OrangeAt))
	}
	return sp
}

// Render is the ANSI view of the fire box.
func (f *Flame) Render() string { return sprite.Render(f.Sprite()) }

// View is a fixed padded canvas so a tape does not crop or jump. The
// flame sits in the middle of a wide void.
func (f *Flame) View() sprite.Sprite {
	board := sprite.New(ViewCols, ViewRows)
	if f == nil || f.Eng == nil {
		return board
	}
	flame := f.Sprite()
	ox := (ViewCols - flame.Width) / 2
	oy := (ViewRows - flame.Height) / 2
	for r := 0; r < flame.Height; r++ {
		for c := 0; c < flame.Width; c++ {
			cell := flame.At(r, c)
			if cell.Transparent() {
				continue
			}
			board.Set(oy+r, ox+c, cell)
		}
	}
	origin := particle.CellOf(f.Eng.Cfg.Origin.X, f.Eng.Cfg.Origin.Y)
	board.Set(oy+origin.Row, ox+origin.Col-1, sprite.Cell{Ch: '▄', FG: 245, BG: 238})
	return board
}
