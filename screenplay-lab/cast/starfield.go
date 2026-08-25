package cast

import (
	"github.com/theprimeagen/apollo-11/stars-lab/stars"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

// StarFPS converts the starfield's seconds into stars.Field ticks.
const StarFPS = 30

// Starfield is the sky. It sizes itself to whatever stage it paints and
// flies with the given strategy. Paint it first: everything else lands
// on top.
type Starfield struct {
	Strategy stars.Strategy
}

// NewStarfield opens a sky flying in the given style.
func NewStarfield(s stars.Strategy) *Starfield {
	return &Starfield{}
}

// Advance accumulates time. dt <= 0 holds the sky.
func (f *Starfield) Advance(dt float64) {
}

// Paint fills the stage with this instant's sky.
func (f *Starfield) Paint(st *screenplay.Stage) {
}
