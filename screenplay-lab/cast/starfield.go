package cast

import (
	"github.com/theprimeagen/apollo-11/stars-lab/stars"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

// StarFPS converts the starfield's seconds into stars.Field ticks.
const StarFPS = 30

// Starfield is the sky. It sizes itself to whatever stage it paints and
// flies with the given strategy. Cast it first: everything else lands
// on top.
type Starfield struct {
	Strategy stars.Strategy
	clock    float64
}

// NewStarfield opens a sky flying in the given style.
func NewStarfield(s stars.Strategy) *Starfield {
	return &Starfield{Strategy: s}
}

// Update accumulates time. dt <= 0 holds the sky.
func (f *Starfield) Update(dt float64) {
	if f == nil || dt <= 0 {
		return
	}
	f.clock += dt
}

// Render fills the screen with this instant's sky.
func (f *Starfield) Render(scr *screenplay.Screen) {
}
