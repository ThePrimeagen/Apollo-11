// Package fire is a thin color layer on particle. Each cell's heat is
// its occupancy plus every neighbour except the incoming side. Heat
// picks a glyph: braille dots, shades, half squares, then bright yellow.
// The package does not draw a lander.
package fire

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	Rows      = 4
	Cols      = 14
	Particles = 2500
	ViewCols  = 20
	ViewRows  = 10
	fps       = 20
)

// DefaultConfig is the 45° trail: one particle every millisecond.
func DefaultConfig() particle.Config {
	return particle.Config{
		Width:     float64(Cols) - 0.01,
		Height:    float64(Rows)*particle.CellHeightUnits - 0.01,
		Origin:    particle.Vec2{X: 1.5, Y: 1.0},
		Direction: particle.Vec2{X: 1, Y: 1},
		Count:     5,
		Period:    0.001,
		MinLife:   0.45,
		MaxLife:   0.55,
		MinSpeed:  14,
		MaxSpeed:  22,
		Spread:    0.28,
		Nozzle:    1.2,
	}
}

// BoosterConfig is left-to-right fire: five particles every millisecond,
// a normal around the axis, about 2500 live. The box is wide enough for
// the cone and the red fringe to show.
func BoosterConfig() particle.Config {
	return particle.Config{
		Width:     12 - 0.01,
		Height:    12 - 0.01,
		Origin:    particle.Vec2{X: 1.0, Y: 6.0},
		Direction: particle.Vec2{X: 1, Y: 0},
		Count:     5,
		Period:    0.001,
		MinLife:   0.45,
		MaxLife:   0.55,
		MinSpeed:  10,
		MaxSpeed:  18,
		Spread:    0.30,
		Nozzle:    3.2,
	}
}

// Flame is one running trail.
type Flame struct {
	Eng *particle.Engine
}

// New starts the 45° trail. It has not yet emitted.
func New(seed int64) *Flame {
	return &Flame{Eng: particle.New(seed, DefaultConfig())}
}

// Booster starts the left-to-right plume.
func Booster(seed int64) *Flame {
	return Toward(seed, particle.Vec2{X: 1, Y: 0})
}

// Update advances the particle engine.
func (f *Flame) Update(dt float64) {
	if f == nil || f.Eng == nil {
		return
	}
	f.Eng.Update(dt)
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

// Sprite paints every cell from neighborhood heat so a hole next to
// fire still lights instead of staying blank.
func (f *Flame) Sprite() sprite.Sprite {
	cols, rows := f.box()
	sp := sprite.New(cols, rows)
	if f == nil || f.Eng == nil {
		return sp
	}
	occ := f.Eng.Occupancy()
	dir := f.Eng.Cfg.Direction
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			h := Heat(occ, particle.Cell{Col: c, Row: r}, dir)
			sp.Set(r, c, Style(h))
		}
	}
	return sp
}

// Render is the ANSI view of the fire box.
func (f *Flame) Render() string { return sprite.Render(f.Sprite()) }

// View is a fixed padded canvas so a tape does not crop or jump.
func (f *Flame) View() sprite.Sprite {
	board := sprite.New(ViewCols, ViewRows)
	if f == nil || f.Eng == nil {
		return board
	}
	flame := f.Sprite()
	ox := (ViewCols - flame.Width) / 2
	oy := (ViewRows - flame.Height) / 2
	if ox < 1 {
		ox = 1
	}
	for r := 0; r < flame.Height; r++ {
		for c := 0; c < flame.Width; c++ {
			cell := flame.At(r, c)
			if cell.Transparent() {
				continue
			}
			board.Set(oy+r, ox+c, cell)
		}
	}
	oc := particle.CellOf(f.Eng.Cfg.Origin.X, f.Eng.Cfg.Origin.Y)
	board.Set(oy+oc.Row, ox+oc.Col, sprite.Cell{Ch: '█', FG: 245, BG: 238})
	return board
}
