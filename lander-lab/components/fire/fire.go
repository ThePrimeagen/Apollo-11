// Package fire is a thin color layer on particle. The default trail is
// 45°, 100 particles, and four terminal rows. Occupancy becomes a flame:
// solid yellow where the particles stack, orange in the middle, small red
// at the tips. The package does not draw a lander.
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
	ViewCols  = 28
	ViewRows  = 12
	viewOx    = 7
	viewOy    = 4
	fps       = 20
)

// DefaultConfig is the 45° four-row trail: 100 particles, down-right,
// dying at the far edge of the box.
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

// Flame is one running trail.
type Flame struct {
	Eng      *particle.Engine
	YellowAt int
	OrangeAt int
}

// New starts a trail that has not yet emitted.
func New(seed int64) *Flame {
	return &Flame{
		Eng:      particle.New(seed, DefaultConfig()),
		YellowAt: 8,
		OrangeAt: 3,
	}
}

// Update advances the particle engine.
func (f *Flame) Update(dt float64) {
	if f == nil || f.Eng == nil {
		return
	}
	f.Eng.Update(dt)
}

// Color maps an occupancy count onto a flame cell using the default
// yellow/orange thresholds (8 and 3).
func Color(n int) sprite.Cell {
	return shade(n, 8, 3)
}

func shade(n, yellowAt, orangeAt int) sprite.Cell {
	if yellowAt < 1 {
		yellowAt = 8
	}
	if orangeAt < 1 || orangeAt > yellowAt {
		orangeAt = 3
	}
	switch {
	case n >= yellowAt:
		return sprite.Cell{Ch: '█', FG: 226, BG: 220}
	case n >= orangeAt:
		return sprite.Cell{Ch: '▓', FG: 208, BG: 166}
	case n >= 1:
		return sprite.Cell{Ch: '·', FG: 196, BG: -1}
	default:
		return sprite.Cell{Ch: ' ', FG: -1, BG: -1}
	}
}

// Sprite is the fixed 14×4 fire box. Empty cells stay transparent.
func (f *Flame) Sprite() sprite.Sprite {
	sp := sprite.New(Cols, Rows)
	if f == nil || f.Eng == nil {
		return sp
	}
	for cell, n := range f.Eng.Occupancy() {
		if cell.Col < 0 || cell.Row < 0 || cell.Col >= Cols || cell.Row >= Rows {
			continue
		}
		sp.Set(cell.Row, cell.Col, shade(f.heat(cell, n), f.YellowAt, f.OrangeAt))
	}
	return sp
}

// heat fades occupancy along the trail so the far end reads as tips
// even when a late burst is still dense.
func (f *Flame) heat(cell particle.Cell, n int) int {
	o := f.Eng.Cfg.Origin
	dx := float64(cell.Col) + 0.5 - o.X
	dy := float64(cell.Row)*particle.CellHeightUnits + 1 - o.Y
	dist := math.Hypot(dx, dy)
	fade := 1 - dist/9
	if fade < 0.12 {
		fade = 0.12
	}
	h := int(math.Ceil(float64(n) * fade))
	if h < 1 && n > 0 {
		return 1
	}
	return h
}

// Render is the ANSI view of the 4-row fire box.
func (f *Flame) Render() string { return sprite.Render(f.Sprite()) }

// View is a fixed padded canvas so a tape does not crop or jump.
func (f *Flame) View() sprite.Sprite {
	board := sprite.New(ViewCols, ViewRows)
	if f == nil || f.Eng == nil {
		return board
	}
	flame := f.Sprite()
	for r := 0; r < flame.Height; r++ {
		for c := 0; c < flame.Width; c++ {
			cell := flame.At(r, c)
			if cell.Transparent() {
				continue
			}
			board.Set(viewOy+r, viewOx+c, cell)
		}
	}
	origin := particle.CellOf(f.Eng.Cfg.Origin.X, f.Eng.Cfg.Origin.Y)
	board.Set(viewOy+origin.Row, viewOx+origin.Col-1, sprite.Cell{Ch: '▄', FG: 245, BG: 238})
	return board
}
